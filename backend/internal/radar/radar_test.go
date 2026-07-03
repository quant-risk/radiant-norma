// Tests for radar.Service — scanSource, recordBaseline, ResolveAlert.
//
// Cobertura:
//   - first scan: registra baseline
//   - hash unchanged: retorna nil (sem alerta)
//   - hash changed: cria alerta + atualiza baseline
//   - recordBaseline idempotente (UPDATE vs INSERT)
//   - ResolveAlert marca alerta como resolvido
//   - ListAlerts filtra por unresolved
package radar_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// Helper: criar Source com URL de servidor HTTP fake
func sourceWithServer(cadoc, label, url string) radar.Source {
	return radar.Source{
		CadocCode: cadoc,
		Label:     label,
		URL:       url,
		AlertType: "test_alert",
		Severity:  "info",
	}
}

// ============================================================
// First scan: registra baseline, não cria alerta
// ============================================================

func TestScanSource_FirstScan(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Servidor HTTP fake que retorna conteúdo estável
	content := "DRSAC FAQ content version 1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)
	alert, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	if len(alert) != 0 {
		t.Errorf("primeiro scan não deveria criar alerta, got %d", len(alert))
	}

	// Verifica que baseline foi gravada
	var baselineCount int
	err = d.QueryRow(`SELECT COUNT(*) FROM radar_alerts WHERE alert_type = '_baseline_drsac_faq' AND cadoc_code = '2030'`).Scan(&baselineCount)
	if err != nil {
		t.Fatalf("query baseline: %v", err)
	}
	if baselineCount != 1 {
		t.Errorf("esperado 1 baseline, got %d", baselineCount)
	}
}

// ============================================================
// Hash unchanged: retorna nil
// ============================================================

func TestScanSource_HashUnchanged(t *testing.T) {
	d := testutil.NewTestDB(t)

	content := "stable content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)

	// Primeiro scan: baseline
	_, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}

	// Segundo scan: hash não mudou → sem alerta
	alerts, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("scan sem mudança não deveria criar alerta, got %d", len(alerts))
	}
}

// ============================================================
// Hash changed: cria alerta + atualiza baseline (REGRESSÃO v1.3.4)
// ============================================================

func TestScanSource_HashChanged_CreatesAlert(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Servidor com conteúdo mutável via atomic
	var counter int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&counter, 1)
		_, _ = io.WriteString(w, "content version "+string(rune('A'+n-1)))
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)

	// Primeiro scan: baseline (counter=1)
	if _, err := svc.ScanOnce(context.Background(), []radar.Source{src}); err != nil {
		t.Fatalf("first: %v", err)
	}

	// Segundo scan: counter=2, hash diferente → alerta
	alerts, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if len(alerts) != 1 {
		t.Errorf("esperado 1 alerta após mudança de hash, got %d", len(alerts))
	}
	if len(alerts) > 0 {
		alert := alerts[0]
		if alert.CadocCode != "2030" {
			t.Errorf("alert.CadocCode = %q, want 2030", alert.CadocCode)
		}
		if !strings.Contains(alert.Title, "mudou") {
			t.Errorf("title deveria conter 'mudou', got: %s", alert.Title)
		}
	}

	// Terceiro scan: counter=3, hash mudou de novo → MAIS UM alerta (não duplica)
	alerts2, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("third: %v", err)
	}
	if len(alerts2) != 1 {
		t.Errorf("esperado 1 alerta na 3ª scan, got %d", len(alerts2))
	}
}

// ============================================================
// recordBaseline idempotente
// ============================================================

func TestScanSource_BaselineIdempotent(t *testing.T) {
	d := testutil.NewTestDB(t)

	content := "stable"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)

	// 3 scans: deve ter 1 baseline (não 3)
	for i := 0; i < 3; i++ {
		if _, err := svc.ScanOnce(context.Background(), []radar.Source{src}); err != nil {
			t.Fatalf("scan %d: %v", i, err)
		}
	}

	var baselineCount int
	err := d.QueryRow(`SELECT COUNT(*) FROM radar_alerts WHERE alert_type = '_baseline_drsac_faq' AND cadoc_code = '2030'`).Scan(&baselineCount)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if baselineCount != 1 {
		t.Errorf("recordBaseline deveria ser idempotente, got %d baselines", baselineCount)
	}
}

// ============================================================
// Hash computation
// ============================================================

func TestFetchHash_Stable(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := radar.New(d, 1*time.Hour)

	// Calcula hash esperado
	expectedSum := sha256.Sum256([]byte("hello"))
	expectedHash := hex.EncodeToString(expectedSum[:])

	// Servidor HTTP fake que retorna "hello"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hello")
	}))
	defer srv.Close()

	src := sourceWithServer("2030", "Test", srv.URL)
	if _, err := svc.ScanOnce(context.Background(), []radar.Source{src}); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Verifica que baseline gravada tem o hash esperado
	var storedHash string
	err := d.QueryRow(`SELECT description FROM radar_alerts WHERE alert_type = '_baseline_test' AND cadoc_code = '2030'`).Scan(&storedHash)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if storedHash != expectedHash {
		t.Errorf("baseline hash = %q, want %q", storedHash, expectedHash)
	}
}

// ============================================================
// ResolveAlert
// ============================================================

func TestResolveAlert(t *testing.T) {
	d := testutil.NewTestDB(t)

	var counter int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt64(&counter, 1)
		_, _ = io.WriteString(w, "v"+string(rune('A'+n-1)))
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)

	// First scan (baseline), second scan (alert)
	_, _ = svc.ScanOnce(context.Background(), []radar.Source{src})
	alerts, _ := svc.ScanOnce(context.Background(), []radar.Source{src})
	if len(alerts) == 0 {
		t.Fatal("esperado alerta")
	}

	alertID := alerts[0].ID

	// Resolve
	if err := svc.ResolveAlert(context.Background(), alertID); err != nil {
		t.Fatalf("ResolveAlert: %v", err)
	}

	// ResolveAlert novamente deve dar erro (already resolved)
	err := svc.ResolveAlert(context.Background(), alertID)
	if err == nil {
		t.Error("segundo ResolveAlert deveria falhar (já resolvido)")
	}

	// ResolveAlert com ID inexistente
	err = svc.ResolveAlert(context.Background(), 99999)
	if err == nil {
		t.Error("ResolveAlert com ID inexistente deveria falhar")
	}
}

// ============================================================
// ListAlerts
// ============================================================

func TestListAlerts(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Insere alertas manualmente
	_, err := d.Exec(`
		INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url)
		VALUES
		  ('2030', 'test_alert', 'info', 'Alerta 1', 'desc 1', 'http://example.com/1'),
		  ('2030', 'test_alert', 'info', 'Alerta 2', 'desc 2', 'http://example.com/2'),
		  ('2030', 'test_alert', 'info', 'Alerta 3 resolvido', 'desc 3', 'http://example.com/3')
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Resolve o terceiro
	_, _ = d.Exec(`UPDATE radar_alerts SET resolved_at = CURRENT_TIMESTAMP WHERE title = 'Alerta 3 resolvido'`)

	svc := radar.New(d, 1*time.Hour)

	// ListAlerts(unresolvedOnly=true) → 2
	unresolved, err := svc.ListAlerts(context.Background(), true, 100)
	if err != nil {
		t.Fatalf("ListAlerts unresolved: %v", err)
	}
	if len(unresolved) != 2 {
		t.Errorf("esperado 2 não resolvidos, got %d", len(unresolved))
	}

	// ListAlerts(unresolvedOnly=false) → 3
	all, err := svc.ListAlerts(context.Background(), false, 100)
	if err != nil {
		t.Fatalf("ListAlerts all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("esperado 3 totais, got %d", len(all))
	}
}

// ============================================================
// shortHash helper (Sprint 5 v1.4.1 — regressão F1)
// ============================================================

// TestShortHash_Normal valida comportamento normal (len >= 12).
//
// Função pura — não precisa de DB.
func TestShortHash_Normal(t *testing.T) {
	// SHA-256 hex tem 64 chars; [:12] deve retornar primeiros 12.
	full := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	got := radar.ShortHash(full)
	want := "abcdef012345"
	if got != want {
		t.Errorf("ShortHash(64 chars) = %q, want %q", got, want)
	}
}

// TestShortHash_Short valida o fix: hash com < 12 chars não panica.
// Validação v1.4.0: mesmo padrão do auditlog.Verify v1.4.0 bug #1.
//
// Função pura — não precisa de DB.
func TestShortHash_Short(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"1 char", "a", "a"},
		{"5 chars", "abcde", "abcde"},
		{"11 chars (just under)", "abcdefghijk", "abcdefghijk"},
		{"12 chars (exactly)", "abcdefghijkl", "abcdefghijkl"},
		{"13 chars (just over)", "abcdefghijklm", "abcdefghijkl"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := radar.ShortHash(c.in)
			if got != c.want {
				t.Errorf("ShortHash(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestShortHash_NeverPanics garante que ShortHash NUNCA panica,
// independente do input. Smoke test defensivo.
//
// Função pura — não precisa de DB.
func TestShortHash_NeverPanics(t *testing.T) {
	inputs := []string{"", "x", "abc", strings.Repeat("y", 100), "\x00\x00\x00"}
	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ShortHash(%q) panicked: %v", in, r)
				}
			}()
			_ = radar.ShortHash(in)
		}()
	}
}

// TestRecordBaseline_ShortHashInDB regressão do F1: se o DB tem um
// hash com < 12 chars (cenário de corrupção ou inserção manual errada),
// o radar NÃO pode panicar. Chamamos ScanOnce 2x pra forçar o caminho
// de "lastHash do DB + recordBaseline com hash novo".
func TestRecordBaseline_ShortHashInDB(t *testing.T) {
	d := testutil.NewTestDB(t)

	// Insere manualmente uma baseline com hash curto (corrompido).
	_, err := d.Exec(`
		INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url)
		VALUES ('2030', '_baseline_corrupted', 'info', 'baseline corrupted', 'shorty', 'http://example.com')
	`)
	if err != nil {
		t.Fatalf("seed corrupted baseline: %v", err)
	}

	// Servidor com conteúdo novo
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "new content")
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := radar.Source{
		CadocCode: "2030",
		Label:     "corrupted",
		URL:       srv.URL,
		AlertType: "test_alert",
		Severity:  "info",
	}

	// Antes do fix: panica em shortHash("shorty") (5 chars).
	// Depois do fix: retorna "shorty" sem panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ScanOnce paniced com hash curto no DB: %v", r)
		}
	}()

	alerts, err := svc.ScanOnce(context.Background(), []radar.Source{src})
	if err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}

	// Hash 'shorty' (5 chars) != SHA-256 hex do novo conteúdo → alerta criado.
	if len(alerts) != 1 {
		t.Errorf("esperado 1 alerta (corrupted baseline → mudança), got %d", len(alerts))
	}
	if len(alerts) > 0 {
		// Description deve usar shortHash defensivamente.
		if !strings.Contains(alerts[0].Description, "shorty") {
			t.Errorf("description deveria conter 'shorty' (hash curto), got: %s", alerts[0].Description)
		}
	}
}

// ============================================================
// FetchHash falha com 404
// ============================================================

func TestScanSource_FetchError(t *testing.T) {
	d := testutil.NewTestDB(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := radar.New(d, 1*time.Hour)
	svc.SetLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	src := sourceWithServer("2030", "DRSAC FAQ", srv.URL)

	// Não deve criar alerta, deve logar warning
	alerts, _ := svc.ScanOnce(context.Background(), []radar.Source{src})
	if len(alerts) != 0 {
		t.Errorf("HTTP 404 não deveria criar alerta, got %d", len(alerts))
	}

	// Verifica que não há baseline
	var baselineCount int
	err := d.QueryRow(`SELECT COUNT(*) FROM radar_alerts WHERE alert_type = '_baseline_drsac_faq' AND cadoc_code = '2030'`).Scan(&baselineCount)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if baselineCount != 0 {
		t.Errorf("fetch falhou, não deveria ter baseline, got %d", baselineCount)
	}
}
