// Package api — tests do Metrics + endpoint /metrics.
//
// Sprint 17 — v3.7.0 [S17.5]: valida renderização Prometheus,
// contadores allowed/dropped/failOpen, e bypass de rate limit no
// endpoint /metrics (scraper não deve ser limitado).

package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/api"
)

// =============================================================================
// Unit tests — Metrics rendering
// =============================================================================

func TestMetrics_RenderEmpty(t *testing.T) {
	m := api.NewMetrics()
	out := m.Render()
	if !strings.Contains(out, "# HELP radiant_rate_limit_allowed_total") {
		t.Errorf("output deveria ter HELP para allowed_total:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE radiant_rate_limit_allowed_total counter") {
		t.Errorf("output deveria ter TYPE counter:\n%s", out)
	}
	if !strings.Contains(out, "radiant_rate_limit_backend_up 1") {
		t.Errorf("backend deveria estar UP por default:\n%s", out)
	}
}

func TestMetrics_IncDropped(t *testing.T) {
	m := api.NewMetrics()
	m.IncDropped("heavy", "memory")
	m.IncDropped("heavy", "memory")
	m.IncDropped("mutate", "memory")

	out := m.Render()
	if !strings.Contains(out, `radiant_rate_limit_dropped_total{bucket="heavy",backend="memory"} 2`) {
		t.Errorf("contador heavy/memory deveria ser 2:\n%s", out)
	}
	if !strings.Contains(out, `radiant_rate_limit_dropped_total{bucket="mutate",backend="memory"} 1`) {
		t.Errorf("contador mutate/memory deveria ser 1:\n%s", out)
	}
}

func TestMetrics_IncAllowed(t *testing.T) {
	m := api.NewMetrics()
	m.IncAllowed("heavy", "redis")
	m.IncAllowed("heavy", "redis")
	m.IncAllowed("heavy", "redis")

	out := m.Render()
	if !strings.Contains(out, `radiant_rate_limit_allowed_total{bucket="heavy",backend="redis"} 3`) {
		t.Errorf("contador allowed deveria ser 3:\n%s", out)
	}
}

func TestMetrics_IncFailOpen(t *testing.T) {
	m := api.NewMetrics()
	m.IncFailOpen("redis")
	m.IncFailOpen("redis")
	m.IncFailOpen("redis")

	out := m.Render()
	if !strings.Contains(out, "radiant_rate_limit_fail_open_total 3") {
		t.Errorf("fail_open deveria ser 3:\n%s", out)
	}
}

func TestMetrics_SetBackendUp(t *testing.T) {
	m := api.NewMetrics()
	m.SetBackendUp(false)
	if !strings.Contains(m.Render(), "radiant_rate_limit_backend_up 0") {
		t.Error("backend deveria estar DOWN")
	}
	m.SetBackendUp(true)
	if !strings.Contains(m.Render(), "radiant_rate_limit_backend_up 1") {
		t.Error("backend deveria estar UP")
	}
}

func TestMetrics_ConcurrentInc(t *testing.T) {
	m := api.NewMetrics()
	const goroutines = 50
	const incsPerG = 100

	done := make(chan struct{})
	for g := 0; g < goroutines; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < incsPerG; i++ {
				m.IncDropped("heavy", "memory")
			}
		}()
	}
	for g := 0; g < goroutines; g++ {
		<-done
	}

	out := m.Render()
	expected := goroutines * incsPerG
	if !strings.Contains(out, `radiant_rate_limit_dropped_total{bucket="heavy",backend="memory"} `+
		itoa(expected)) {
		t.Errorf("contador devia ser %d:\n%s", expected, out)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return strings.TrimSpace(formatInt(n))
}

func formatInt(n int) string {
	// Quick int→string sem importar strconv só pra evitar mais import
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// =============================================================================
// HTTP integration test — /metrics endpoint
// =============================================================================

func TestMetrics_EndpointExposed(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.Metrics = api.NewMetrics()
	handler := srv.Router()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/plain") {
		t.Errorf("Content-Type deveria ser text/plain, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "radiant_rate_limit_backend_up") {
		t.Errorf("body deveria ter backend_up metric:\n%s", w.Body.String())
	}
}

func TestMetrics_EndpointBypassesRateLimit(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.Metrics = api.NewMetrics()
	handler := srv.Router()

	// 20 requests ao /metrics — todas devem passar (sem 429).
	// Compare: validate tem bucket heavy=10/min. /metrics não conta.
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("request #%d a /metrics não deveria estar rate-limited", i+1)
		}
	}

	// Validate end-to-end: 11 reqs a /v1/validate (heavy bucket = 10)
	// devem gerar 10 allowed + 1 dropped no Metrics.
	for i := 0; i < 11; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/validate",
			strings.NewReader(`{"cadoc_code":"3040","xml":"<doc/>"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-IF-ID", "demo")
		req.Header.Set("Origin", "http://example.com")
		handler.ServeHTTP(w, req)
	}

	body := srv.Metrics.Render()
	if !strings.Contains(body, `radiant_rate_limit_allowed_total{bucket="heavy",backend="memory"} 10`) {
		t.Errorf("esperado 10 allowed em heavy/memory:\n%s", body)
	}
	if !strings.Contains(body, `radiant_rate_limit_dropped_total{bucket="heavy",backend="memory"} 1`) {
		t.Errorf("esperado 1 dropped em heavy/memory:\n%s", body)
	}
}