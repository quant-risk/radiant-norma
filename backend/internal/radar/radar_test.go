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

// Helper: criar Service com logger silencioso
func newTestService(t *testing.T, d interface{ Close() error }) *radar.Service {
	t.Helper()
	// (não usado aqui; Service criado direto nos tests)
	return nil
}

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hello")
	}))
	defer srv.Close()

	d := testutil.NewTestDB(t)
	svc := radar.New(d, 1*time.Hour)

	// Calcula hash esperado
	expectedSum := sha256.Sum256([]byte("hello"))
	expectedHash := hex.EncodeToString(expectedSum[:])

	// Acessa fetchHash indiretamente via ScanOnce
	content := "hello"
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv2.Close()

	src := sourceWithServer("2030", "Test", srv2.URL)
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
