// Package api — tests para Sprint 79 v3.34.48 (Webhooks outbound).
//
// Cobertura:
//   - GET  /v1/webhooks                                  — listWebhooks
//   - POST /v1/webhooks                                   — registerWebhook
//   - DELETE /v1/webhooks/{id}                           — deleteWebhook
//   - GET  /v1/webhooks/{id}/deliveries                  — listDeliveries
//   - GET  /v1/webhooks/{id}/deliveries/{delivery_id}   — getDelivery
//   - POST /v1/webhooks/{id}/deliveries/{delivery_id}/retry — retryDelivery
//
// Auth: RADIANT_DEV_AUTH=1 → X-IF-ID header fallback (same pattern as
// server_test.go).

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/webhook"
	_ "modernc.org/sqlite"
)

// setupWebhookTest creates an in-memory DB with webhooks schema and a Server
// with the webhook service wired in. RADIANT_DEV_AUTH=1 is set so X-IF-ID
// header is accepted as auth.
func setupWebhookTest(t *testing.T) (*Server, *sql.DB, func()) {
	t.Helper()
	t.Setenv("RADIANT_DEV_AUTH", "1")

	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS webhooks (
			id TEXT PRIMARY KEY,
			if_id TEXT NOT NULL,
			url TEXT NOT NULL,
			secret TEXT,
			events TEXT NOT NULL,
			description TEXT,
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_if ON webhooks(if_id, active)`,
		`CREATE TABLE IF NOT EXISTS webhook_deliveries (
			id TEXT PRIMARY KEY,
			webhook_id TEXT NOT NULL,
			event TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			http_status INTEGER,
			response_body TEXT,
			attempt INTEGER NOT NULL DEFAULT 0,
			next_retry_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_delivery_webhook ON webhook_deliveries(webhook_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_delivery_status ON webhook_deliveries(status, next_retry_at) WHERE status IN ('pending', 'retrying')`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	wbSvc := webhook.NewService(d)
	srv := &Server{DB: d, Webhook: wbSvc, RateLimiter: newMemoryRateLimiter()}

	return srv, d, func() { _ = d.Close() }
}

// authRequest attaches X-IF-ID header for dev auth (RADIANT_DEV_AUTH=1).
func authRequest(r *http.Request, ifID string) {
	r.Header.Set("X-IF-ID", ifID)
}

// ============================================================================
// listWebhooks
// ============================================================================

func TestListWebhooks_Empty(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/webhooks", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string][]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
}

func TestListWebhooks_WithWebhooks(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-a", "demo", "https://example.com/a", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-b", "demo", "https://example.com/b", "schema.changed,radar.change_detected")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/webhooks", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Handler returns bare array, not wrapped in {"webhooks": ...}.
	var out []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(out))
	}
}

func TestListWebhooks_OtherTenantSeesOwnOnly(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-demo", "demo", "https://example.com/demo", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-other", "other", "https://example.com/other", "validation.completed")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/webhooks", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 1 {
		t.Errorf("demo should see only 1 webhook, got %d", len(out))
	}
}

func TestListWebhooks_Unauthorized(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/webhooks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ============================================================================
// registerWebhook
// ============================================================================

func TestRegisterWebhook_OK(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	body := `{"url":"https://example.com/hook","events":["validation.completed","schema.changed"]}`
	req := httptest.NewRequest("POST", "/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if out["url"] != "https://example.com/hook" {
		t.Errorf("unexpected url: %v", out["url"])
	}
}

func TestRegisterWebhook_InvalidURL(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	body := `{"url":"not-a-valid-url","events":["validation.completed"]}`
	req := httptest.NewRequest("POST", "/v1/webhooks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterWebhook_MissingBody(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("POST", "/v1/webhooks", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ============================================================================
// deleteWebhook
// ============================================================================

func TestDeleteWebhook_OK(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-del", "demo", "https://example.com/hook", "validation.completed")

	handler := srv.Router()
	req := httptest.NewRequest("DELETE", "/v1/webhooks/wh-del", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Soft delete: active=0 but row still exists.
	var count int
	_ = d.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhooks WHERE id=? AND active=1`, "wh-del").Scan(&count)
	if count != 0 {
		t.Errorf("webhook should be soft-deleted (active=0), but active count=%d", count)
	}
}

func TestDeleteWebhook_NotFound(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("DELETE", "/v1/webhooks/nonexistent", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteWebhook_OtherTenant(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-other", "other", "https://example.com/hook", "validation.completed")

	handler := srv.Router()
	req := httptest.NewRequest("DELETE", "/v1/webhooks/wh-other", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (cross-tenant), got %d", w.Code)
	}
}

// ============================================================================
// listDeliveries
// ============================================================================

func TestListDeliveries_OK(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-1", "demo", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status) VALUES (?, ?, ?, ?, ?)`,
		"del-1", "wh-1", "validation.completed", `{}`, "success")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status) VALUES (?, ?, ?, ?, ?)`,
		"del-2", "wh-1", "validation.completed", `{}`, "failed")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/webhooks/wh-1/deliveries", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Handler returns bare array.
	var out []map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out) != 2 {
		t.Errorf("expected 2 deliveries, got %d", len(out))
	}
}

func TestListDeliveries_NotFound(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/webhooks/nonexistent/deliveries", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ============================================================================
// getDelivery
// ============================================================================

func TestGetDelivery_OK(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-1", "demo", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempt) VALUES (?, ?, ?, ?, ?, ?)`,
		"del-1", "wh-1", "validation.completed", `{"cadoc":"3040"}`, "success", 1)

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/webhooks/wh-1/deliveries/del-1", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if out["id"] != "del-1" {
		t.Errorf("expected id del-1, got %v", out["id"])
	}
	if out["status"] != "success" {
		t.Errorf("expected status success, got %v", out["status"])
	}
}

func TestGetDelivery_NotFound(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("GET", "/v1/webhooks/wh-1/deliveries/del-nonexistent", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetDelivery_CrossTenant(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-other", "other", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status) VALUES (?, ?, ?, ?, ?)`,
		"del-other", "wh-other", "validation.completed", `{}`, "success")

	handler := srv.Router()
	req := httptest.NewRequest("GET", "/v1/webhooks/wh-other/deliveries/del-other", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 (cross-tenant), got %d", w.Code)
	}
}

// ============================================================================
// retryDelivery
// ============================================================================

func TestRetryDelivery_OK(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-1", "demo", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempt) VALUES (?, ?, ?, ?, ?, ?)`,
		"del-failed", "wh-1", "validation.completed", `{}`, "failed", 3)

	handler := srv.Router()
	req := httptest.NewRequest("POST", "/v1/webhooks/wh-1/deliveries/del-failed/retry", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	json.Unmarshal(w.Body.Bytes(), &out)
	if out["status"] != "pending" {
		t.Errorf("expected status pending, got %v", out["status"])
	}

	var status string
	_ = d.QueryRowContext(ctx, `SELECT status FROM webhook_deliveries WHERE id=?`, "del-failed").Scan(&status)
	if status != "pending" {
		t.Errorf("expected DB status pending, got %s", status)
	}
}

func TestRetryDelivery_NotFailed(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-1", "demo", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status) VALUES (?, ?, ?, ?, ?)`,
		"del-success", "wh-1", "validation.completed", `{}`, "success")

	handler := srv.Router()
	req := httptest.NewRequest("POST", "/v1/webhooks/wh-1/deliveries/del-success/retry", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRetryDelivery_NotFound(t *testing.T) {
	srv, _, cleanup := setupWebhookTest(t)
	defer cleanup()
	handler := srv.Router()

	req := httptest.NewRequest("POST", "/v1/webhooks/wh-1/deliveries/del-nonexistent/retry", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (not found), got %d: %s", w.Code, w.Body.String())
	}
}

func TestRetryDelivery_CrossTenant(t *testing.T) {
	srv, d, cleanup := setupWebhookTest(t)
	defer cleanup()
	ctx := context.Background()

	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhooks (id, if_id, url, events, active) VALUES (?, ?, ?, ?, 1)`,
		"wh-other", "other", "https://example.com/hook", "validation.completed")
	_, _ = d.ExecContext(ctx,
		`INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status) VALUES (?, ?, ?, ?, ?)`,
		"del-other", "wh-other", "validation.completed", `{}`, "failed")

	handler := srv.Router()
	req := httptest.NewRequest("POST", "/v1/webhooks/wh-other/deliveries/del-other/retry", nil)
	authRequest(req, "demo")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (cross-tenant), got %d", w.Code)
	}
}
