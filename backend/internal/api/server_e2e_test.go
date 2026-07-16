// E2E tests para endpoints HTTP restantes (F8 — Sprint 6 v1.5.0).
//
// Cobertura complementa server_test.go (que foca Healthz + ResolveRadarAlert)
// e server_admin_test.go (que foca R1 = auth admin + rate limit + cache).
//
// Aqui: Validate, STASubmit, ListSchemas, ListRules, ListVersions,
// GetSchema, AuthMiddleware rejects.
//
// Foco em regressões: gaps da validação 7 (v1.4.0).
package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// reqJSON faz request JSON para o handler.
func reqJSON(handler http.Handler, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// ============================================================
// AuthMiddleware
// ============================================================

func TestAuthMiddleware_RejectsNoIFIDHeader(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	// Sem X-IF-ID em qualquer endpoint v1
	endpoints := []struct {
		method, path string
	}{
		{"GET", "/v1/schemas"},
		{"GET", "/v1/rules"},
		{"GET", "/v1/radar/alerts"},
		{"POST", "/v1/validate"},
	}

	for _, e := range endpoints {
		w := reqJSON(handler, e.method, e.path, nil, nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status = %d, want 401 sem X-IF-ID",
				e.method, e.path, w.Code)
		}
	}
}

// TestAuthMiddleware_HealthzNoAuthRequired smoke: /healthz é público.
func TestAuthMiddleware_HealthzNoAuthRequired(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "GET", "/healthz", nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("healthz deveria ser público, got %d", w.Code)
	}
}

// ============================================================
// /v1/validate
// ============================================================

func TestValidate_InvalidJSONReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("POST", "/v1/validate", strings.NewReader(`{`))
	req.Header.Set("X-IF-ID", "demo")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (JSON inválido)", w.Code)
	}
}

func TestValidate_MissingCadocReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "POST", "/v1/validate",
		map[string]any{"xml": "<root/>"}, map[string]string{"X-IF-ID": "demo"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (sem cadoc_code)", w.Code)
	}
}

func TestValidate_MissingXMLReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "POST", "/v1/validate",
		map[string]any{"cadoc": "3040"}, map[string]string{"X-IF-ID": "demo"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (sem xml)", w.Code)
	}
}

func TestValidate_ValidCadocEmitsAudit(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Cadastro: crítica + schema para 3040
	_, _ = d.Exec(`INSERT INTO criticas (cadoc_code, sheet, codigo, regra, gravidade, enabled) VALUES ('3040', 'Básicas', 'B01', 'test', 'A', 1)`)
	_, _ = d.Exec(`INSERT INTO schema_versions (cadoc_code, effective_from, source_uri, fields_json) VALUES ('3040', '2024-01-01', 'http://example.com/v1', '[]')`)

	w := reqJSON(handler, "POST", "/v1/validate",
		map[string]any{
			"cadoc":    "3040",
			"data_base": "2026-06",
			"xml":      `<?xml version="1.0"?><Doc3040></Doc3040>`,
		},
		map[string]string{"X-IF-ID": "audit-if"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	// Audit emission
	var count int
	if err := d.QueryRow(`
		SELECT COUNT(*) FROM audit_log WHERE action = 'cadoc.validated' AND if_id = 'audit-if'
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("audit_log cadoc.validated = %d, want 1", count)
	}
}

// ============================================================
// /v1/sta/submit
// ============================================================

func TestSTASubmit_Basic(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Sprint 6 v1.5.0 (F8 fix): FK envios.if_id → ifs.id precisa da IF.
	_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano) VALUES ('if-sta-1', '99999991', 'IF STA Test', 'SCD', 'lite')`)

	w := reqJSON(handler, "POST", "/v1/sta/submit",
		map[string]any{
			"cadoc_code": "3040",
			"data_base":  "2024-12-01",
			"xml":        `<?xml version="1.0"?><root/>`,
		},
		map[string]string{"X-IF-ID": "if-sta-1"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	// Verifica que envio foi persistido
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM envios WHERE if_id = 'if-sta-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("envios = %d, want 1", count)
	}

	// Audit
	var auditCount int
	_ = d.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action = 'sta.submit'`).Scan(&auditCount)
	if auditCount != 1 {
		t.Errorf("audit sta.submit = %d, want 1", auditCount)
	}
}

func TestSTASubmit_MissingCadocReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "POST", "/v1/sta/submit",
		map[string]any{
			"data_base": "2024-12-01",
			"xml":       "<root/>",
		},
		map[string]string{"X-IF-ID": "if-sta-err"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// ============================================================
// /v1/schemas e /v1/rules — listagem dinâmica (W4)
// ============================================================

func TestListSchemas_EmptyDBReturnsEmpty(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "GET", "/v1/schemas", nil,
		map[string]string{"X-IF-ID": "if-ls"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	cadocs, _ := body["cadocs"].([]any)
	if len(cadocs) != 0 {
		t.Errorf("Esperado 0 cadocs em DB vazio, got %d", len(cadocs))
	}
}

func TestListSchemas_FromDB(t *testing.T) {
	srv, _ := newTestServer(t)
	d := srv.DB
	handler := srv.Router()

	// Cadastra 3 CADOCs via schema_versions
	day := "2024-01-01"
	for _, c := range []string{"3040", "3050", "2030"} {
		_, _ = d.Exec(`
			INSERT INTO schema_versions (cadoc_code, effective_from, source_uri, fields_json)
			VALUES (?, ?, 'http://example.com/'||?, '[]')
		`, c, day, c)
	}

	w := reqJSON(handler, "GET", "/v1/schemas", nil,
		map[string]string{"X-IF-ID": "if-ls"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	cadocs, _ := body["cadocs"].([]any)
	if len(cadocs) != 3 {
		t.Errorf("Esperado 3 cadocs, got %d", len(cadocs))
	}
}

func TestListRules_FromDB(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Cadastra críticas para 2 CADOCs
	for _, c := range []string{"2060", "2070"} {
		_, _ = d.Exec(`
			INSERT INTO criticas (cadoc_code, sheet, codigo, regra, enabled)
			VALUES (?, 'Básicas', 'B01', 'test', 1)
		`, c)
	}

	w := reqJSON(handler, "GET", "/v1/rules", nil,
		map[string]string{"X-IF-ID": "if-lr"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	cadocs, _ := body["cadocs"].([]any)
	if len(cadocs) != 2 {
		t.Errorf("Esperado 2 cadocs (de criticas), got %d", len(cadocs))
	}
}

func TestListRules_ByCadoc_WithEnabledFilter(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Cadastra 3 críticas: 2 enabled + 1 disabled
	for i, code := range []string{"B01", "B02", "B03"} {
		enabled := i < 2 // B01, B02 enabled
		_, _ = d.Exec(`
			INSERT INTO criticas (cadoc_code, sheet, codigo, regra, gravidade, enabled)
			VALUES ('3040', 'Test', ?, 'rule '||?, 'A', ?)
		`, code, code, enabled)
	}

	// ?enabled=true → 2
	w := reqJSON(handler, "GET", "/v1/rules/3040?enabled=true", nil,
		map[string]string{"X-IF-ID": "if-filt"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	rules, _ := body["rules"].([]any)
	if len(rules) != 2 {
		t.Errorf("Esperado 2 regras enabled, got %d", len(rules))
	}

	// ?enabled=false → 1
	w = reqJSON(handler, "GET", "/v1/rules/3040?enabled=false", nil,
		map[string]string{"X-IF-ID": "if-filt"})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	rules, _ = body["rules"].([]any)
	if len(rules) != 1 {
		t.Errorf("Esperado 1 regra disabled, got %d", len(rules))
	}
}

// ============================================================
// /v1/schemas/{cadoc} — getSchema
// ============================================================

func TestGetSchema_NotFoundReturns404(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "GET", "/v1/schemas/9999", nil,
		map[string]string{"X-IF-ID": "if-gs"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (CADOC inexistente)", w.Code)
	}
}

func TestGetSchema_ValidReturnsVersion(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	_, _ = d.Exec(`
		INSERT INTO schema_versions (cadoc_code, effective_from, source_uri, fields_json)
		VALUES ('3040', '2024-01-01', 'http://example.com/v1', '[{"tag":"testField","type":"A8","required":true}]')
	`)

	w := reqJSON(handler, "GET", "/v1/schemas/3040", nil,
		map[string]string{"X-IF-ID": "if-gs"})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var body map[string]any
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["cadoc_code"] != "3040" {
		t.Errorf("cadoc_code = %v, want 3040", body["cadoc_code"])
	}
}

// ============================================================
// /v1/radar/alerts/{id} — getRadarAlert
// ============================================================

func TestGetRadarAlert_NotFound(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "GET", "/v1/radar/alerts/9999", nil,
		map[string]string{"X-IF-ID": "if-ra"})

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (alert inexistente)", w.Code)
	}
}

func TestGetRadarAlert_InvalidIDReturns400(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := reqJSON(handler, "GET", "/v1/radar/alerts/notanumber", nil,
		map[string]string{"X-IF-ID": "if-ra"})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (id inválido)", w.Code)
	}
}
