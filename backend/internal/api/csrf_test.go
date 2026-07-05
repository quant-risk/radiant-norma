// Package api — tests do CSRF middleware.

package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRF_AllowsSameOrigin(t *testing.T) {
	cfg := CSRFConfig{
		EnforceProduction: true,
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/v1/rules/B12/toggle", nil)
	req.Host = "localhost:8421"
	req.Header.Set("Origin", "http://localhost:8421")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for same-origin, got %d", w.Code)
	}
}

func TestCSRF_BlocksCrossOriginInProduction(t *testing.T) {
	cfg := CSRFConfig{
		EnforceProduction: true,
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/v1/rules/B12/toggle", nil)
	req.Host = "localhost:8421"
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-origin in prod, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "CSRF") {
		t.Errorf("expected CSRF error message, got %s", w.Body.String())
	}
}

func TestCSRF_AllowsCrossOriginInDev(t *testing.T) {
	cfg := CSRFConfig{
		EnforceProduction: false, // dev mode
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/v1/rules/B12/toggle", nil)
	req.Host = "localhost:8421"
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	// Dev: warning logged, request allowed
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 in dev mode, got %d", w.Code)
	}
}

func TestCSRF_AllowsWhitelistedRoute(t *testing.T) {
	cfg := CSRFConfig{
		WhitelistRoutes:   []string{"/api/login"},
		EnforceProduction: true,
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/login", nil)
	req.Host = "localhost:8421"
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for whitelisted route even cross-origin, got %d", w.Code)
	}
}

func TestCSRF_AllowsGETRequests(t *testing.T) {
	cfg := CSRFConfig{
		EnforceProduction: true,
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/rules/disabled", nil)
	req.Host = "localhost:8421"
	// No Origin set
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GET (no CSRF), got %d", w.Code)
	}
}

func TestCSRF_AllowsNoOriginLegacyClient(t *testing.T) {
	cfg := CSRFConfig{
		EnforceProduction: true,
	}
	mw := CSRF(cfg)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Postman/curl: sem Origin, sem Referer
	req := httptest.NewRequest("POST", "/v1/rules/B12/toggle", nil)
	req.Host = "localhost:8421"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for non-browser client (no Origin/Referer), got %d", w.Code)
	}
}

func TestIsSameOrigin(t *testing.T) {
	tests := []struct {
		origin, host string
		want         bool
	}{
		{"http://localhost:4180", "localhost:4180", true},
		{"https://app.example.com", "app.example.com", true},
		{"http://example.com", "localhost:4180", false},
		{"http://localhost:4180/path", "localhost:4180", true}, // path ignored
		{"ftp://localhost:4180", "localhost:4180", false},      // wrong scheme
		{"", "localhost:4180", false},
		{"http://evil.com", "localhost:4180", false},
	}

	for _, tt := range tests {
		got := isSameOrigin(tt.origin, tt.host)
		if got != tt.want {
			t.Errorf("isSameOrigin(%q, %q) = %v, want %v", tt.origin, tt.host, got, tt.want)
		}
	}
}
