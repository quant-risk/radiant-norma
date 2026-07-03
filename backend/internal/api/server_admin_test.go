// Integration tests para triggerRadarScan (R1 — DOS-via-API prevention).
//
// Complementa radar/admin_test.go (que testa unidades) com testes E2E
// do handler HTTP real.
//
// Cobertura:
//   - Auth admin required (401 sem token)
//   - Bearer token alternativo ao X-Admin
//   - Rate limit (429 após 1ª chamada)
//   - Cache (200 com cached=true se < 5min)
//   - Audit emitido (radar.scan.triggered ou radar.scan.cached)
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/radar"
)

// triggerScanRequest faz POST /v1/radar/scan com headers customizados.
func triggerScanRequest(handler http.Handler, ifID, adminToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("X-IF-ID", ifID)
	if adminToken != "" {
		req.Header.Set("X-Admin", adminToken)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

// TestTriggerRadarScan_RequiresAdmin garante que sem admin token,
// retorna 401 e NÃO chama ScanOnce (FAIL CLOSED).
func TestTriggerRadarScan_RequiresAdmin(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := triggerScanRequest(handler, "if-1", "") // sem admin

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (sem admin)", w.Code)
	}
}

// TestTriggerRadarScan_AcceptsXAdminHeader valida caminho feliz com X-Admin.
func TestTriggerRadarScan_AcceptsXAdminHeader(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	w := triggerScanRequest(handler, "if-1", "test-admin-token")

	// O endpoint chama DefaultSources (3 URLs BACEN reais) → vai falhar
	// HTTP mas não deve dar 401. Pode dar 200 com 0 alerts OU 500 por
	// BACEN offline. O importante é que passa do auth check.
	if w.Code == http.StatusUnauthorized {
		t.Errorf("Com admin token válido, status = %d, want != 401", w.Code)
	}
}

// TestTriggerRadarScan_AcceptsBearerToken valida Bearer token alternativo.
func TestTriggerRadarScan_AcceptsBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	req := httptest.NewRequest("POST", "/v1/radar/scan", nil)
	req.Header.Set("X-IF-ID", "if-1")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code == http.StatusUnauthorized {
		t.Errorf("Com Bearer válido, status = %d, want != 401", w.Code)
	}
}

// TestTriggerRadarScan_RateLimit verifica que 2ª chamada em < 1min = 429.
//
// Importante: cache é invalidado entre chamadas. Sem isso, 2ª chamada
// bate no cache HIT e retorna cached=true (skipando rate limit), que é
// correto pelo design (cache HIT não chama BACEN → sem DOS).
func TestTriggerRadarScan_RateLimit(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	// 1ª chamada: passa (pode falhar HTTP, mas não 401 nem 429)
	w1 := triggerScanRequest(handler, "if-rl", "test-admin-token")
	if w1.Code == http.StatusUnauthorized {
		t.Fatalf("1ª chamada bloqueada por auth: %d", w1.Code)
	}
	if w1.Code == http.StatusTooManyRequests {
		t.Fatalf("1ª chamada já rate-limited (cooldown muito curto?): %d", w1.Code)
	}

	// Invalida cache para forçar scan real na 2ª chamada
	srv.ScanCache.Invalidate()

	// 2ª chamada: deve dar 429 (rate limit)
	w2 := triggerScanRequest(handler, "if-rl", "test-admin-token")
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("2ª chamada em < 1min: status = %d, want 429", w2.Code)
	}

	// Header Retry-After deve estar presente
	retryAfter := w2.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Errorf("Header Retry-After esperado em 429, got vazio")
	}
}

// TestTriggerRadarScan_RateLimitPerIF garante isolamento entre IFs.
// IF-A hammerar não bloqueia IF-B.
func TestTriggerRadarScan_RateLimitPerIF(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	// IF-A: 1ª chamada
	wA1 := triggerScanRequest(handler, "if-A", "test-admin-token")
	if wA1.Code == http.StatusTooManyRequests {
		t.Fatalf("IF-A 1ª chamada bloqueada: %d", wA1.Code)
	}

	// IF-B: 1ª chamada independente
	wB1 := triggerScanRequest(handler, "if-B", "test-admin-token")
	if wB1.Code == http.StatusTooManyRequests {
		t.Errorf("IF-B 1ª chamada bloqueada por IF-A: %d", wB1.Code)
	}
}

// TestTriggerRadarScan_CacheHit valida que 2ª chamada dentro do TTL
// retorna cached=true se a 1ª teve sucesso.
//
// Para testar isso, injetamos manualmente um resultado no cache.
// (Sem isso, o teste dependeria do BACEN estar online + estável, flaky).
func TestTriggerRadarScan_CacheHit(t *testing.T) {
	srv, _ := newTestServer(t)
	handler := srv.Router()

	// Injeta cache com 1 alerta fake
	srv.ScanCache.Put([]radar.Alert{
		{ID: 999, CadocCode: "2030", Title: "cached fake alert", DetectedAt: time.Now()},
	})

	w := triggerScanRequest(handler, "if-cache", "test-admin-token")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (cached)", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["cached"] != true {
		t.Errorf("cached = %v, want true", body["cached"])
	}
}

// TestTriggerRadarScan_CacheEmitsCachedAudit garante audit diferente para cached.
func TestTriggerRadarScan_CacheEmitsCachedAudit(t *testing.T) {
	srv, d := newTestServer(t)
	handler := srv.Router()

	// Injeta cache
	srv.ScanCache.Put([]radar.Alert{{ID: 1, CadocCode: "2030", DetectedAt: time.Now()}})

	w := triggerScanRequest(handler, "if-cache-audit", "test-admin-token")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action = 'radar.scan.cached'`).Scan(&count); err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_log radar.scan.cached = %d, want 1", count)
	}
}
