// Package api — tests para Sprint 62/63 v3.34.44 (Marketplace de regras customizadas).
//
// Cobertura:
//   - GET    /v1/marketplace               — listMarketplaceRules
//   - POST   /v1/marketplace               — publishRule
//   - POST   /v1/marketplace/{id}/install  — installRule
//   - POST   /v1/marketplace/{id}/rate     — rateRule
//   - GET    /v1/marketplace/installed      — listInstalledRules
//
// Auth: RADIANT_DEV_AUTH=1 → X-IF-ID header fallback.

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/marketplace"
	_ "modernc.org/sqlite"
)

// setupMarketplaceTest creates an in-memory DB with marketplace schema and a Server
// with the marketplace service wired in.
func setupMarketplaceTest(t *testing.T) (*Server, *sql.DB, func()) {
	t.Helper()
	t.Setenv("RADIANT_DEV_AUTH", "1")

	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS marketplace_rules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			code TEXT NOT NULL,
			cadoc TEXT NOT NULL,
			rule_type TEXT NOT NULL,
			config TEXT,
			author_if_id TEXT NOT NULL,
			author_name TEXT,
			rating REAL NOT NULL DEFAULT 0,
			rating_count INTEGER NOT NULL DEFAULT 0,
			install_count INTEGER NOT NULL DEFAULT 0,
			tags TEXT,
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS marketplace_installs (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL,
			if_id TEXT NOT NULL,
			installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(rule_id, if_id)
		)`,
		`CREATE TABLE IF NOT EXISTS marketplace_ratings (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL,
			if_id TEXT NOT NULL,
			stars INTEGER NOT NULL CHECK(stars >= 1 AND stars <= 5),
			UNIQUE(rule_id, if_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_marketplace_cadoc ON marketplace_rules(cadoc, active)`,
		`CREATE INDEX IF NOT EXISTS idx_installs_if ON marketplace_installs(if_id)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	mpSvc := marketplace.NewService(d)
	srv := &Server{DB: d, Marketplace: mpSvc, RateLimiter: newMemoryRateLimiter()}

	return srv, d, func() { _ = d.Close() }
}

// ============================================================================
// listMarketplaceRules
// ============================================================================

func TestListMarketplaceRules_Empty(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/marketplace", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if out["total"] != float64(0) {
		t.Errorf("expected total 0, got %v", out["total"])
	}
}

func TestListMarketplaceRules_WithRules(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-1", "Regra IPOC", "CUSTOM_001", "3040", "semantic", "author-if")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/marketplace", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string][]map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out["rules"]) != 1 {
		t.Errorf("expected 1 rule, got %d", len(out["rules"]))
	}
}

func TestListMarketplaceRules_Unauthorized(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/marketplace", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListMarketplaceRules_FilterByCadoc(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-3040", "Regra 3040", "CUSTOM_001", "3040", "semantic", "author-if")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-3050", "Regra 3050", "CUSTOM_002", "3050", "semantic", "author-if")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/marketplace?cadoc=3040", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var out map[string][]map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out["rules"]) != 1 {
		t.Errorf("expected 1 rule for cadoc=3040, got %d", len(out["rules"]))
	}
}

// ============================================================================
// publishRule
// ============================================================================

func TestPublishRule_OK(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	body := `{"name":"Regra IPOC","code":"CUSTOM_IPOC_01","cadoc":"3040","rule_type":"semantic"}`
	req := httptest.NewRequest("POST", "/v1/marketplace", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "author-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if out["code"] != "CUSTOM_IPOC_01" {
		t.Errorf("expected code CUSTOM_IPOC_01, got %v", out["code"])
	}
}

func TestPublishRule_InvalidRuleType(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	body := `{"name":"Regra","code":"CUSTOM_X","cadoc":"3040","rule_type":"invalid"}`
	req := httptest.NewRequest("POST", "/v1/marketplace", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublishRule_InvalidCode(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	body := `{"name":"Regra","code":"BAD_CODE","cadoc":"3040","rule_type":"semantic"}`
	req := httptest.NewRequest("POST", "/v1/marketplace", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// installRule
// ============================================================================

func TestInstallRule_OK(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-to-install", "Regra Teste", "CUSTOM_TEST", "3040", "semantic", "author-if")

	handler := srv.Router()
	req := httptest.NewRequest("POST", "/v1/marketplace/rule-to-install/install", nil)
	authRequest(req, "tenant-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify install was recorded.
	var count int
	_ = d.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM marketplace_installs WHERE rule_id=? AND if_id=?`,
		"rule-to-install", "tenant-bank").Scan(&count)
	if count != 1 {
		t.Errorf("expected install recorded (count=1), got %d", count)
	}
}

func TestInstallRule_NotFound(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("POST", "/v1/marketplace/nonexistent/install", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (not found), got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// rateRule
// ============================================================================

func TestRateRule_OK(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-to-rate", "Regra Teste", "CUSTOM_TEST", "3040", "semantic", "author-if")

	handler := srv.Router()
	body := `{"stars": 5}`
	req := httptest.NewRequest("POST", "/v1/marketplace/rule-to-rate/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "tenant-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify rating was recorded.
	var stars int
	_ = d.QueryRowContext(ctx,
		`SELECT stars FROM marketplace_ratings WHERE rule_id=? AND if_id=?`,
		"rule-to-rate", "tenant-bank").Scan(&stars)
	if stars != 5 {
		t.Errorf("expected stars=5, got %d", stars)
	}
}

func TestRateRule_InvalidStars(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-stars", "Regra", "CUSTOM_STARS", "3040", "semantic", "author-if")

	handler := srv.Router()
	body := `{"stars": 6}`
	req := httptest.NewRequest("POST", "/v1/marketplace/rule-stars/rate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ============================================================================
// listInstalledRules
// ============================================================================

func TestListInstalledRules_Empty(t *testing.T) {
	srv, _, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/marketplace/installed", nil)
	authRequest(req, "tenant-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Handler returns bare array.
	var out []any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 0 {
		t.Errorf("expected 0 installed rules, got %d", len(out))
	}
}

func TestListInstalledRules_WithInstalled(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-installed", "Regra Instalada", "CUSTOM_INST", "3040", "semantic", "author-if")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_installs (id, rule_id, if_id) VALUES (?, ?, ?)`,
		"inst-1", "rule-installed", "tenant-bank")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/marketplace/installed", nil)
	authRequest(req, "tenant-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 1 {
		t.Errorf("expected 1 installed rule, got %d", len(out))
	}
}

func TestListInstalledRules_TenantIsolation(t *testing.T) {
	srv, d, cleanup := setupMarketplaceTest(t)
	defer cleanup()
	ctx := context.Background()

	// Rule installed for "other" tenant.
	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_rules (id, name, code, cadoc, rule_type, author_if_id, active)
		 VALUES (?, ?, ?, ?, ?, ?, 1)`,
		"rule-other", "Regra Outro", "CUSTOM_OTHER", "3040", "semantic", "author-if")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO marketplace_installs (id, rule_id, if_id) VALUES (?, ?, ?)`,
		"inst-other", "rule-other", "other-tenant")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/marketplace/installed", nil)
	authRequest(req, "tenant-bank")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 0 {
		t.Errorf("tenant-bank should see 0 rules (rule belongs to other-tenant), got %d", len(out))
	}
}
