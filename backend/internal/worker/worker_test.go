// Tests for worker package — ProcessBatch, RunLeaseSweeper, ComputeBackoff.
//
// Cobertura (W1 + W2 do Sprint 6):
//   - ComputeBackoff: 5 tentativas (1m, 5m, 30m, 2h, 12h)
//   - ProcessBatch com STA failure: retry com backoff, depois dead letter
//   - ProcessBatch com STA success: status accepted/rejected
//   - RunLeaseSweeper: resseta stuck processing
//   - Retry timing: filter por next_retry_at
package worker_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	workpkg "github.com/fortvna/radiant-norma/backend/internal/worker"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// ============================================================
// ComputeBackoff — função pura
// ============================================================

func TestComputeBackoff_Table(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{0, 1 * time.Minute},   // 1ª falha → retry em 1min
		{1, 5 * time.Minute},   // 2ª falha → 5min
		{2, 30 * time.Minute},  // 3ª falha → 30min
		{3, 2 * time.Hour},     // 4ª falha → 2h
		{4, 12 * time.Hour},    // 5ª falha → 12h
		{5, 0},                 // ≥ MaxAttempts → dead letter (sem retry)
		{99, 0},                // muito longe → 0
		{-1, 1 * time.Minute},  // negativo → trata como 0
	}
	for _, c := range cases {
		c := c
		t.Run("", func(t *testing.T) {
			got := workpkg.ComputeBackoff(c.attempts)
			if got != c.want {
				t.Errorf("ComputeBackoff(%d) = %v, want %v", c.attempts, got, c.want)
			}
		})
	}
}

// ============================================================
// Helpers — STA stub fail/success
// ============================================================

// failingSTAClient retorna erro em Submit (simula BACEN offline).
type failingSTAClient struct {
	calls int
}

func (f *failingSTAClient) Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error) {
	f.calls++
	return nil, errors.New("simulated STA failure")
}

// alwaysAcceptSTAClient aceita tudo (success path).
type alwaysAcceptSTAClient struct {
	calls int
}

func (a *alwaysAcceptSTAClient) Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error) {
	a.calls++
	return &sta.Result{Accepted: true, ProtocolSTA: "PROTO-" + sub.CadocCode}, nil
}

// helperLogger cria logger silencioso pra tests.
func helperLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// insertTestIF cria uma IF (multi-tenant) para satisfazer FK em envios.
// INSERT OR IGNORE: se a IF já existir (de outro teste que usou mesmo id),
// ignora em vez de falhar com UNIQUE constraint (cnpj).
func insertTestIF(t *testing.T, d *sql.DB, id string) {
	t.Helper()
	cnpj := id
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	for len(cnpj) < 8 {
		cnpj = cnpj + "0"
	}
	_, err := d.Exec(`
		INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
		VALUES (?, ?, ?, 'SCD', 'lite')
	`, id, cnpj, "IF Test "+id)
	if err != nil {
		t.Fatalf("insert IF: %v", err)
	}
}

// insertPendingEnvio insere um envio pronto pra worker pickar.
// Cria IF automaticamente se não existir.
func insertPendingEnvio(t *testing.T, d *sql.DB, id, ifID, cadoc string) {
	t.Helper()
	insertTestIF(t, d, ifID)
	_, err := d.Exec(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    status, attempts)
		VALUES (?, ?, ?, '2024-01-01', 1, 'x', 'x', 'pending', 0)
	`, id, ifID, cadoc)
	if err != nil {
		t.Fatalf("insert envio: %v", err)
	}
}

// insertPendingEnvioWithAttempts variante que aceita attempts customizado.
func insertPendingEnvioWithAttempts(t *testing.T, d *sql.DB, id, ifID, cadoc string, attempts int) {
	t.Helper()
	insertTestIF(t, d, ifID)
	_, err := d.Exec(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    status, attempts)
		VALUES (?, ?, ?, '2024-01-01', 1, 'x', 'x', 'pending', ?)
	`, id, ifID, cadoc, attempts)
	if err != nil {
		t.Fatalf("insert envio: %v", err)
	}
}

// insertStuckEnvio insere envio em processing há X segundos (para lease sweeper).
func insertStuckEnvio(t *testing.T, d *sql.DB, id, ifID, cadoc string, secondsAgo int) {
	t.Helper()
	insertTestIF(t, d, ifID)
	_, err := d.Exec(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    status, processing_started_at)
		VALUES (?, ?, ?, '2024-01-01', 1, 'x', 'x', 'processing',
		        DATETIME(CURRENT_TIMESTAMP, ?))
	`, id, ifID, cadoc, fmt.Sprintf("-%d seconds", secondsAgo))
	if err != nil {
		t.Fatalf("insert stuck: %v", err)
	}
}

// insertPendingWithNextRetry insere envio pending com next_retry_at customizado.
func insertPendingWithNextRetry(t *testing.T, d *sql.DB, id, ifID, cadoc string, attempts int, retryAt string) {
	t.Helper()
	insertTestIF(t, d, ifID)
	_, err := d.Exec(`
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    status, attempts, next_retry_at)
		VALUES (?, ?, ?, '2024-01-01', 1, 'x', 'x', 'pending', ?, ?)
	`, id, ifID, cadoc, attempts, retryAt)
	if err != nil {
		t.Fatalf("insert with retry: %v", err)
	}
}

// ===========================
// ProcessBatch — failure path com W1 (W1 = retry/backoff)
// ===========================

// TestProcessBatch_FailureSchedulesRetry valida que falha de STA agenda
// retry com backoff (W1). Após 1 falha: status=pending, attempts=1,
// next_retry_at ~ NOW()+1min.
func TestProcessBatch_FailureSchedulesRetry(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Seed: 1 envio pendente
	insertPendingEnvio(t, d, "env-test-1", "if-1", "3040")

	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staFail := &failingSTAClient{}
	logger := helperLogger()

	n, err := workpkg.ProcessBatch(context.Background(), d, auditSvc, auditLog, staFail, 10, logger)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if n != 0 {
		t.Errorf("Esperado n=0 (Falhou, não processou), got %d", n)
	}

	// Verifica: status=pending, attempts=1, next_retry_at preenchido
	var status string
	var attempts int
	var nextRetryAt sql.NullTime
	err = d.QueryRow(`SELECT status, attempts, next_retry_at FROM envios WHERE id = 'env-test-1'`).Scan(&status, &attempts, &nextRetryAt)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want pending (retry)", status)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if !nextRetryAt.Valid {
		t.Errorf("next_retry_at deveria estar preenchido")
	}
}

// TestProcessBatch_FailureExhaustsRetries valida dead-letter após MaxAttempts.
//
// attempts inicial = 4 → +1 = 5 (= MaxAttempts) → dead_letter.
func TestProcessBatch_FailureExhaustsRetries(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Seed: 1 envio já com attempts=4 (última antes de dead letter)
	insertPendingEnvioWithAttempts(t, d, "env-exhaust", "if-1", "3040", 4)

	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staFail := &failingSTAClient{}
	logger := helperLogger()

	_, _ = workpkg.ProcessBatch(context.Background(), d, auditSvc, auditLog, staFail, 10, logger)

	var status string
	var attempts int
	err := d.QueryRow(`SELECT status, attempts FROM envios WHERE id = 'env-exhaust'`).Scan(&status, &attempts)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "dead_letter" {
		t.Errorf("status = %q, want dead_letter", status)
	}
	if attempts != 5 {
		t.Errorf("attempts = %d, want 5 (= MaxAttempts)", attempts)
	}
}

// TestProcessBatch_SuccessAccepted valida success path: STA aceita,
// status = accepted, protocol_sta preenchido.
func TestProcessBatch_SuccessAccepted(t *testing.T) {
	d := testutil.NewTestDB(t)
	insertPendingEnvio(t, d, "env-ok", "if-1", "3040")

	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staOK := &alwaysAcceptSTAClient{}
	logger := helperLogger()

	n, err := workpkg.ProcessBatch(context.Background(), d, auditSvc, auditLog, staOK, 10, logger)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if n != 1 {
		t.Errorf("Esperado n=1, got %d", n)
	}

	var status string
	var protocol sql.NullString
	err = d.QueryRow(`SELECT status, protocol_sta FROM envios WHERE id = 'env-ok'`).Scan(&status, &protocol)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "accepted" {
		t.Errorf("status = %q, want accepted", status)
	}
	if !protocol.Valid || !strings.HasPrefix(protocol.String, "PROTO-") {
		t.Errorf("protocol_sta = %v, want PROTO-...", protocol)
	}
}

// TestProcessBatch_NoEnvios retorna 0 sem erro quando DB vazio.
func TestProcessBatch_NoEnvios(t *testing.T) {
	d := testutil.NewTestDB(t)

	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staOK := &alwaysAcceptSTAClient{}
	logger := helperLogger()

	n, err := workpkg.ProcessBatch(context.Background(), d, auditSvc, auditLog, staOK, 10, logger)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if n != 0 {
		t.Errorf("Esperado n=0, got %d", n)
	}
}

// TestProcessBatch_RespectsNextRetryAt valida que envios com retry
// futuro NÃO são pegos pelo worker (filtra por next_retry_at <= NOW).
func TestProcessBatch_RespectsNextRetryAt(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Envio 1: retry futuro (não pode ser pego ainda)
	insertPendingWithNextRetry(t, d, "env-future", "if-1", "3040", 1,
		"DATETIME(CURRENT_TIMESTAMP, '+1 hour')")
	// Envio 2: retry no passado (DEVE ser pego)
	insertPendingEnvio(t, d, "env-past", "if-1", "3050")

	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staOK := &alwaysAcceptSTAClient{}
	logger := helperLogger()

	n, _ := workpkg.ProcessBatch(context.Background(), d, auditSvc, auditLog, staOK, 10, logger)
	if n != 1 {
		t.Errorf("Esperado n=1 (apenas env-past), got %d", n)
	}

	// env-future ainda pending, env-past accepted
	var futureStatus, pastStatus string
	_ = d.QueryRow(`SELECT status FROM envios WHERE id = 'env-future'`).Scan(&futureStatus)
	_ = d.QueryRow(`SELECT status FROM envios WHERE id = 'env-past'`).Scan(&pastStatus)
	if futureStatus != "pending" {
		t.Errorf("env-future deveria continuar pending, got %q", futureStatus)
	}
	if pastStatus != "accepted" {
		t.Errorf("env-past deveria ser accepted, got %q", pastStatus)
	}
}

// ============================================================
// RunLeaseSweeper — W2
// ============================================================

// TestRunLeaseSweeper_ResetsStuckProcessing valida que envios em
// 'processing' há > LeaseTimeout são resetados para 'pending'.
//
// Também valida que envios em 'processing' recentes NÃO são tocados.
func TestRunLeaseSweeper_ResetsStuckProcessing(t *testing.T) {
	d := testutil.NewTestDB(t)
	logger := helperLogger()

	// Envio 1: stuck há 10min (LeaseTimeout = 5min)
	insertStuckEnvio(t, d, "env-stuck", "if-1", "3040", 600)
	// Envio 2: processing há 30s (muito recente, não toca)
	insertStuckEnvio(t, d, "env-recent", "if-1", "3040", 30)
	// Envio 3: pending (nunca foi claimed — sweeper não toca)
	insertPendingEnvio(t, d, "env-pending", "if-1", "3050")

	n, err := workpkg.RunLeaseSweeper(context.Background(), d, logger)
	if err != nil {
		t.Fatalf("RunLeaseSweeper: %v", err)
	}
	if n != 1 {
		t.Errorf("Esperado n=1 (apenas env-stuck), got %d", n)
	}

	var stuckStatus, recentStatus, pendingStatus string
	_ = d.QueryRow(`SELECT status FROM envios WHERE id = 'env-stuck'`).Scan(&stuckStatus)
	_ = d.QueryRow(`SELECT status FROM envios WHERE id = 'env-recent'`).Scan(&recentStatus)
	_ = d.QueryRow(`SELECT status FROM envios WHERE id = 'env-pending'`).Scan(&pendingStatus)

	if stuckStatus != "pending" {
		t.Errorf("env-stuck deveria ser resetado para pending, got %q", stuckStatus)
	}
	if recentStatus != "processing" {
		t.Errorf("env-recent deveria continuar processing, got %q", recentStatus)
	}
	if pendingStatus != "pending" {
		t.Errorf("env-pending deveria continuar pending, got %q", pendingStatus)
	}
}

// TestRunLeaseSweeper_Idempotent garante que rodar sweeper 2x é safe.
func TestRunLeaseSweeper_Idempotent(t *testing.T) {
	d := testutil.NewTestDB(t)
	logger := helperLogger()

	insertStuckEnvio(t, d, "env-stuck-idem", "if-1", "3040", 600)

	// 1ª passada
	n1, _ := workpkg.RunLeaseSweeper(context.Background(), d, logger)
	if n1 != 1 {
		t.Errorf("1ª passada: esperado n=1, got %d", n1)
	}
	// 2ª passada — não há mais nada stuck
	n2, _ := workpkg.RunLeaseSweeper(context.Background(), d, logger)
	if n2 != 0 {
		t.Errorf("2ª passada: esperado n=0 (já resetado), got %d", n2)
	}
}

// TestRunLeaseSweeper_NoStuck não faz nada sem stuck envios.
func TestRunLeaseSweeper_NoStuck(t *testing.T) {
	d := testutil.NewTestDB(t)
	logger := helperLogger()

	insertPendingEnvio(t, d, "env-only-pending", "if-1", "3040")

	n, _ := workpkg.RunLeaseSweeper(context.Background(), d, logger)
	if n != 0 {
		t.Errorf("Esperado n=0 (sem stuck), got %d", n)
	}
}

// ============================================================
// Progress: ensure package compiles, no leaked goroutines
// ============================================================

// TestWorker_PackageSmoke garante que o pacote importa sem panic.
func TestWorker_PackageSmoke(t *testing.T) {
	if workpkg.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", workpkg.MaxAttempts)
	}
	if workpkg.LeaseTimeout != 5*time.Minute {
		t.Errorf("LeaseTimeout = %v, want 5m", workpkg.LeaseTimeout)
	}
	if workpkg.LeaseSweepInterval != 1*time.Minute {
		t.Errorf("LeaseSweepInterval = %v, want 1m", workpkg.LeaseSweepInterval)
	}
}

// Compilação: forçar uso de helperLogger pra evitar linter warning.
var _ = helperLogger
var _ = os.Stderr
