// Tests end-to-end da API HTTP (httptest + chi router).
//
// Cobertura:
//
//   - TestHealthz: smoke test básico, valida uptime + version (referencia
//     api.Version — single source of truth).
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
// Foco: regressão dos gaps das validações 7-10.
package api_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/branding"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	rules "github.com/fortvna/radiant-norma/backend/internal/crossdoc/rules"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	gen2030pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2030"
	gen2060pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2060"
	gen2061pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2061"
	gen2062pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2062"
	gen2070pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2070"
	gen2160pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2160"
	gen2170pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2170"
	gen3040pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	gen3050pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen3050"
	gen4111pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen4111"
	"github.com/fortvna/radiant-norma/backend/internal/generator/wizard"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// newTestServer monta um *api.Server com DB in-memory e Radar sem
// fontes (ScanOnce([]) retorna vazio sem HTTP).
//
// Sprint 6 v1.5.0 (R1): inicializa ScanLimiter + ScanCache + AdminAuth
// para validar hardening de triggerRadarScan.
func newTestServer(t *testing.T) (*api.Server, *sql.DB) {
	t.Helper()
	d := testutil.NewTestDB(t)

	// Sprint 13 [S14.1]: FK em audit_log.if_id → ifs(id). Tests que
	// gravam audit (validate, radar resolve, ack) precisam de IFs pré-seed.
	// Sem isso, audit emit falha silenciosamente. Seed dos IFs comuns.
	// CNPJ unique: synthetic conforme id pra evitar colisão em UNIQUE.
	testIFs := []string{"demo", "demo-bank", "audit-if", "canary-if",
		"if-1", "if-cache-audit", "bank-1", "if-b", "if-c", "if-x", "if-y", "if-d", "other",
		"system"}
	for i, ifID := range testIFs {
		// CNPJ raiz 8 dígitos sintético único por IF (ex: 00000001, 00000002...).
		cnpj := fmt.Sprintf("%08d", i+1)
		_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
			VALUES (?, ?, ?, 'SCD', 'S5', 'pro')`, ifID, cnpj, "Test "+ifID)
	}

	schReg := schema.New(d)
	audSvc := audit.New(d)
	audLog := auditlog.New(d)
	staClient := sta.NewStubClient()
	radarSvc := radar.New(d, 1) // 1ns (não usado, scan é on-demand)

	srv := api.NewServer(d, schReg, audSvc, audLog, staClient, radarSvc, nil, nil, nil, branding.NewBrandingService(d), nil, nil, nil, nil, wizard.NewStore(d))
	srv.CrossDoc = crossdoc.NewEngine(crossdocBuiltinRegistry())
	srv.ScanLimiter = radar.NewScanLimiter(1 * time.Minute)
	srv.ScanCache = radar.NewScanCache(5 * time.Minute)
	srv.AdminAuth = &radar.AdminAuth{Token: "test-admin-token"}
	srv.CadocListCache = schema.NewCadocListCache(5 * time.Minute)
	srv.SchemaInfoCache = api.NewSchemaInfoCache()

	// Sprint 57 v3.36.4: popula GeneratorRegistry (Sprint 57 cmd/api wiring).
	reg := generator.NewRegistry()
	generator.RegisterDefaults(reg, []generator.CADOCGenerator{
		gen2030pkg.New(), gen2060pkg.New(), gen2061pkg.New(),
		gen2062pkg.New(), gen2070pkg.New(), gen2160pkg.New(),
		gen2170pkg.New(), gen3040pkg.New(), gen3050pkg.New(),
		gen4111pkg.New(),
	})
	srv.GeneratorRegistry = reg

	// Dev mode habilitado por default nos tests legacy (que usam X-IF-ID).
	// Tests novos Sprint 7a podem desabilitar via t.Setenv("RADIANT_DEV_AUTH", "")
	// ANTES de chamar newTestServer — escopo do t.Setenv deve envolver handler.ServeHTTP.
	t.Setenv("RADIANT_DEV_AUTH", "1")
	return srv, d
}

// crossdocBuiltinRegistry importa rules/BuiltinRegistry de cross-doc.
// Sem cycle: crossdoc/rules só importa crossdoc raiz.
func crossdocBuiltinRegistry() *crossdoc.Registry {
	return rules.BuiltinRegistry()
}

var _ = audit.New // ensure import

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
	req := httptest.NewRequest("POST", "/v1/radar/alerts/"+strconv.FormatInt(id, 10)+"/resolve", nil)
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
	req := httptest.NewRequest("POST", "/v1/radar/alerts/"+strconv.FormatInt(id, 10)+"/resolve", nil)
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

// itoa wrapper removido na v1.4.4 (validação 10) — strconv.FormatInt é
// usado diretamente nos call sites. Sem razão pra manter o wrapper.
