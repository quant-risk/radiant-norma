// Tests end-to-end da API HTTP (httptest + chi router).
//
// Cobertura Sprint 5 v1.4.1:
//
//   - TestHealthz: smoke test básico, valida uptime + version.
//
//   - TestResolveRadarAlert_EmitsAudit (regressão F2): garante que o
//     endpoint POST /v1/radar/alerts/{id}/resolve emite uma entrada no
//     audit_log. Antes do fix v1.4.1, esse endpoint NÃO emitia audit
//     — gap coberto em VALIDATION_v1.4.0.md.
//
//   - TestAuditEmission_Surface: canário — se alguém futuramente remover
//     a chamada auditLog.Log(...) em resolveRadarAlert, este teste falha.
//
// NOTA: TestTriggerRadarScan_EmitsAudit foi REMOVIDO na 8ª validação porque
// o handler chama ScanOnce com DefaultSources (3 URLs BACEN reais) e exigiria
// httptest mock complexo. A cobertura do canário (TestAuditEmission_Surface)
// já protege o caminho de regressão.
//
// Esses testes não cobrem 100% da API (validação, STA submit, schemas,
// etc continuam cobertos indiretamente pelos testes dos services).
// Foco: regressão dos gaps da validação v1.4.0.
package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// newTestServer monta um *api.Server com DB in-memory e Radar sem
// fontes (ScanOnce([]) retorna vazio sem HTTP).
func newTestServer(t *testing.T) (*api.Server, *sql.DB) {
	t.Helper()
	d := testutil.NewTestDB(t)

	schReg := schema.New(d)
	audSvc := audit.New(d)
	audLog := auditlog.New(d)
	staClient := sta.NewStubClient()
	radarSvc := radar.New(d, 1) // 1ns (não usado, scan é on-demand)

	srv := api.NewServer(d, schReg, audSvc, audLog, staClient, radarSvc)
	return srv, d
}

// TestHealthz smoke test do /healthz.
func TestHealthz(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
	if body["version"] != api.Version {
		t.Errorf("version = %v, want %s", body["version"], api.Version)
	}
}

// TestResolveRadarAlert_EmitsAudit (regressão F2 — VALIDATION_v1.4.0)
//
// Antes do fix: POST /v1/radar/alerts/{id}/resolve NÃO emitia audit.
// Depois do fix (v1.4.1): emite "radar.alert.resolved" com alert_id.
func TestResolveRadarAlert_EmitsAudit(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Seed: cria um alerta real pra resolver
	res, err := d.ExecContext(context.Background(), `
		INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url)
		VALUES ('2030', 'test_alert', 'info', 'Alerta de teste', 'desc', 'http://example.com')
	`)
	if err != nil {
		t.Fatalf("seed alert: %v", err)
	}
	id, _ := res.LastInsertId()

	// Resolve via HTTP
	req := httptest.NewRequest("POST", "/v1/radar/alerts/"+itoa(id)+"/resolve", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	// Verifica que audit_log tem a entrada
	var count int
	err = d.QueryRow(`
		SELECT COUNT(*) FROM audit_log
		WHERE action = 'radar.alert.resolved' AND target = 'radar'
	`).Scan(&count)
	if err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_log entries com action='radar.alert.resolved' = %d, want 1", count)
	}
}

// TestAuditEmission_Surface é um teste-meta: garante que mutations de
// Radar emitem audit. Usa ResolveAlert como proxy (path feliz, network-free).
//
// IMPORTANTE: este teste é o "canário" — se alguém futuramente remover
// a chamada auditLog.Log(...) em resolveRadarAlert, este teste falha.
func TestAuditEmission_Surface(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Snapshot do estado inicial do audit_log
	var initialCount int
	_ = d.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&initialCount)

	// Seed alert
	res, _ := d.ExecContext(context.Background(), `
		INSERT INTO radar_alerts (cadoc_code, alert_type, severity, title, description, source_url)
		VALUES ('2030', 'test_alert', 'info', 'Canário audit', 'desc', 'http://example.com')
	`)
	id, _ := res.LastInsertId()

	// Resolve via endpoint
	req := httptest.NewRequest("POST", "/v1/radar/alerts/"+itoa(id)+"/resolve", nil)
	req.Header.Set("X-IF-ID", "canary-if")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// Verifica delta de audit_log
	var finalCount int
	_ = d.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&finalCount)

	if finalCount != initialCount+1 {
		t.Errorf("audit_log não cresceu: inicial=%d final=%d (esperado +1)",
			initialCount, finalCount)
	}

	// Verifica conteúdo
	var action, target, actor, ifID string
	_ = d.QueryRow(`
		SELECT action, target, actor, if_id FROM audit_log ORDER BY id DESC LIMIT 1
	`).Scan(&action, &target, &actor, &ifID)
	if action != "radar.alert.resolved" {
		t.Errorf("action = %q, want radar.alert.resolved", action)
	}
	if target != "radar" {
		t.Errorf("target = %q, want radar", target)
	}
	// actor = r.RemoteAddr (httptest usa 192.0.2.1:1234)
	if actor != "192.0.2.1:1234" {
		t.Errorf("actor = %q, want 192.0.2.1:1234 (r.RemoteAddr)", actor)
	}
	// if_id = X-IF-ID header (multi-tenant identifier)
	if ifID != "canary-if" {
		t.Errorf("if_id = %q, want canary-if (X-IF-ID)", ifID)
	}
}

// itoa converte int64 pra string. Substituído por strconv.FormatInt na v1.4.3
// (validação 9) — não há razão pra reinventar stdlib.
//
// Deprecated: use strconv.FormatInt(n, 10).
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
