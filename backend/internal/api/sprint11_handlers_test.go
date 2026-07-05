// Package api — tests dos handlers Sprint 11 (rule toggle).

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/realtime"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

// ctxWithClaims injeta Claims JWT no context pra handlers que requerem
// auth.ClaimsFromContext.
func ctxWithClaims(req *http.Request, ifID, sub string) *http.Request {
	c := &auth.Claims{
		Sub:  sub,
		IFID: ifID,
		Role: auth.RoleIF,
		Iss:  "test",
	}
	return req.WithContext(auth.WithClaims(req.Context(), c))
}

// setupRuleToggleTest cria Server com ruleprefs + auditlog em SQLite temp.
func setupRuleToggleTest(t *testing.T) (*Server, *sql.DB, *realtime.Hub, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "sprint11-test-*")
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Setup schema mínimo
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS disabled_rules (
			if_id TEXT NOT NULL,
			rule_code TEXT NOT NULL,
			disabled_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			disabled_by TEXT NOT NULL,
			PRIMARY KEY (if_id, rule_code)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			if_id TEXT, actor TEXT NOT NULL, action TEXT NOT NULL,
			target TEXT, payload_hash TEXT NOT NULL,
			prev_hash TEXT NOT NULL, entry_hash TEXT NOT NULL,
			metadata TEXT, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	hub := realtime.NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	al := auditlog.New(d)
	_ = logger

	srv := &Server{
		DB:        d,
		RulePrefs: ruleprefs.NewPreferences(d),
		AuditLog:  realtime.WrapAuditLog(al, hub),
		EventsHub: hub,
	}

	cleanup := func() {
		_ = d.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return srv, d, hub, cleanup
}

func TestListDisabledRules_Empty(t *testing.T) {
	srv, _, _, cleanup := setupRuleToggleTest(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/v1/rules/disabled", nil)
	req = ctxWithClaims(req, "demo", "user-1")
	w := httptest.NewRecorder()
	srv.listDisabledRules(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp struct {
		IFID  string   `json:"if_id"`
		Codes []string `json:"codes"`
		Total int      `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.IFID != "demo" {
		t.Errorf("expected if_id=demo, got %s", resp.IFID)
	}
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
}

func TestToggleRule_DisableThenEnable(t *testing.T) {
	srv, _, _, cleanup := setupRuleToggleTest(t)
	defer cleanup()

	// Step 1: disable
	r := chi.NewRouter()
	r.Post("/rules/{code}/toggle", srv.toggleRule)
	r.Get("/rules/disabled", srv.listDisabledRules)

	req1 := httptest.NewRequest("POST", "/rules/B12/toggle", nil)
	req1 = ctxWithClaims(req1, "demo", "user-1")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != 200 {
		t.Errorf("expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var resp1 struct {
		NewState string `json:"new_state"`
	}
	_ = json.NewDecoder(w1.Body).Decode(&resp1)
	if resp1.NewState != "disabled" {
		t.Errorf("expected 'disabled', got %q", resp1.NewState)
	}

	// Step 2: list — should contain B12
	req2 := httptest.NewRequest("GET", "/rules/disabled", nil)
	req2 = ctxWithClaims(req2, "demo", "user-1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if !bytes.Contains(w2.Body.Bytes(), []byte("B12")) {
		t.Errorf("expected list to contain B12, got: %s", w2.Body.String())
	}

	// Step 3: toggle again → enabled
	req3 := httptest.NewRequest("POST", "/rules/B12/toggle", nil)
	req3 = ctxWithClaims(req3, "demo", "user-1")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	var resp3 struct {
		NewState string `json:"new_state"`
	}
	_ = json.NewDecoder(w3.Body).Decode(&resp3)
	if resp3.NewState != "enabled" {
		t.Errorf("expected 'enabled' on 2nd toggle, got %q", resp3.NewState)
	}
}

func TestToggleRule_OptimisticConcurrency(t *testing.T) {
	srv, _, _, cleanup := setupRuleToggleTest(t)
	defer cleanup()

	r := chi.NewRouter()
	r.Post("/rules/{code}/toggle", srv.toggleRule)

	// Step 1: disable B12 (no expected_state)
	req0 := httptest.NewRequest("POST", "/rules/B12/toggle", nil)
	req0 = ctxWithClaims(req0, "demo", "user-1")
	r.ServeHTTP(httptest.NewRecorder(), req0)

	// Step 2: try to toggle with expected_state="enabled" (mismatch — current is "disabled")
	body1 := bytes.NewBufferString(`{"expected_state":"enabled"}`)
	req1 := httptest.NewRequest("POST", "/rules/B12/toggle", body1)
	req1 = ctxWithClaims(req1, "demo", "user-1")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusConflict {
		t.Errorf("expected 409 (state mismatch), got %d: %s", w1.Code, w1.Body.String())
	}

	// Step 3: try with expected_state="disabled" (matches) → should succeed
	body2 := bytes.NewBufferString(`{"expected_state":"disabled"}`)
	req2 := httptest.NewRequest("POST", "/rules/B12/toggle", body2)
	req2 = ctxWithClaims(req2, "demo", "user-1")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Errorf("expected 200 (state matches), got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestToggleRule_RequiresClaims(t *testing.T) {
	srv, _, _, cleanup := setupRuleToggleTest(t)
	defer cleanup()

	r := chi.NewRouter()
	r.Post("/rules/{code}/toggle", srv.toggleRule)

	req := httptest.NewRequest("POST", "/rules/B12/toggle", nil)
	// No claims in context
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestToggleRule_EmitsAuditEvent(t *testing.T) {
	srv, d, hub, cleanup := setupRuleToggleTest(t)
	defer cleanup()

	// Subscribe ao hub pra capturar evento SSE
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, unreg := hub.Subscribe(ctx, "demo")
	defer unreg()

	r := chi.NewRouter()
	r.Post("/rules/{code}/toggle", srv.toggleRule)

	req := httptest.NewRequest("POST", "/rules/F23/toggle", nil)
	req = ctxWithClaims(req, "demo", "user-1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("toggle: %d %s", w.Code, w.Body.String())
	}

	// Verifica audit log no DB
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE action='rule.disabled' AND target='F23'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 rule.disabled audit entry, got %d", count)
	}

	// Verifica evento SSE publicado
	select {
	case evt := <-events:
		if evt.Kind != "rule.disabled" {
			t.Errorf("expected SSE event rule.disabled, got %s", evt.Kind)
		}
		// entryToPayload expõe target (que é o rule_code nesse handler).
		if evt.Payload["target"] != "F23" {
			t.Errorf("expected payload.target=F23, got %v", evt.Payload["target"])
		}
		// Metadata tem actor + role (passado pelo handler).
		metaRaw, ok := evt.Payload["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected metadata in payload, got %T (%v)", evt.Payload["metadata"], evt.Payload["metadata"])
		}
		if metaRaw["actor"] != "user-1" {
			t.Errorf("expected metadata.actor=user-1, got %v", metaRaw["actor"])
		}
		if metaRaw["role"] != "if" {
			t.Errorf("expected metadata.role=if, got %v", metaRaw["role"])
		}
	case <-ctx.Done():
		t.Fatal("context cancelled before event")
	}
}