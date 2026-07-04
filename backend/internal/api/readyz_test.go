// Tests para readyz (F23.1) e authMiddleware validation (F23.2).
package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/api"
)

// TestReadyz_OK: readyz retorna 200 quando DB está OK.
func TestReadyz_OK(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("readyz com DB OK = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"db":"ok"`) {
		t.Errorf("body missing db:ok: %s", w.Body.String())
	}
}

// TestReadyz_NoDB: readyz retorna 503 quando DB é nil.
//
// Validação 23 (F23.1): K8s readiness probe deve tirar pod do LB se DB
// estiver indisponível. Sem este check, pod ficaria no LB com DB quebrado.
func TestReadyz_NoDB(t *testing.T) {
	srv := api.NewServer(nil, nil, nil, nil, nil, nil)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/readyz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz com DB=nil = %d, want 503", w.Code)
	}
}

// TestAuthMiddleware_IFIDTooLong: X-IF-ID > 64 chars retorna 400.
//
// Validação 23 (F23.2): sem limite, atacante envia X-IF-ID de 10KB
// → logado no audit_log → incha disco.
func TestAuthMiddleware_IFIDTooLong(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	tooLong := strings.Repeat("a", 65)
	req := httptest.NewRequest("GET", "/v1/schemas", nil)
	req.Header.Set("X-IF-ID", tooLong)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("X-IF-ID 65 chars = %d, want 400", w.Code)
	}
}

// TestAuthMiddleware_IFIDInvalidCharset: X-IF-ID com char especial retorna 400.
func TestAuthMiddleware_IFIDInvalidCharset(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/schemas", nil)
	req.Header.Set("X-IF-ID", "demo;DROP TABLE ifs")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("X-IF-ID com ; = %d, want 400", w.Code)
	}
}

// TestAuthMiddleware_IFIDAllowed: X-IF-ID alfanumérico-dash OK.
func TestAuthMiddleware_IFIDAllowed(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/schemas", nil)
	req.Header.Set("X-IF-ID", "demo-bank_2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusUnauthorized {
		t.Errorf("X-IF-ID válido = %d, want != 400/401", w.Code)
	}
}

// TestAuthMiddleware_IFIDMissing: X-IF-ID ausente retorna 401.
func TestAuthMiddleware_IFIDMissing(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/schemas", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("X-IF-ID ausente = %d, want 401", w.Code)
	}
}
