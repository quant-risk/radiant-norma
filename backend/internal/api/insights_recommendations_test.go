// Package api — tests para Sprint 12 v3.5.1 (C34.16 integration).
//
// Validação 34 — C34.16: insightsRecommendations deve incluir status
// `acknowledged` em cada recommendation baseada no Acknowledgments service.

package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	_ "modernc.org/sqlite"
)

// setupRecommendationsTest cria Server com ruleprefs + insights em SQLite temp.
// Schema mínimo: criticas + rule_failures + envios (necessários pra heurística).
func setupRecommendationsTest(t *testing.T) (*Server, func()) {
	t.Helper()
	d, err := sqlOpen()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS criticas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cadoc_code TEXT NOT NULL,
			sheet TEXT, codigo TEXT NOT NULL,
			regra TEXT, descricao TEXT, gravidade TEXT,
			data_base_inicio DATETIME, mensagem_erro TEXT,
			enabled INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS rule_failures (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			if_id TEXT NOT NULL, cadoc_code TEXT NOT NULL,
			rule_code TEXT NOT NULL, rule_severity TEXT,
			envio_id TEXT, failed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS envios (
			id TEXT PRIMARY KEY, if_id TEXT NOT NULL, cadoc_code TEXT NOT NULL,
			period TEXT, status TEXT, rules_passed INT, rules_failed INT,
			duration_ms INT, protocol_sta TEXT, error_code TEXT, error_message TEXT,
			sent_at DATETIME, confirmed_at DATETIME, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS acknowledged_recommendations (
			if_id TEXT NOT NULL, rec_id TEXT NOT NULL,
			acknowledged_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			acknowledged_by TEXT NOT NULL,
			PRIMARY KEY (if_id, rec_id)
		)`,
	} {
		if _, err := d.Exec(stmt); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	srv := &Server{
		DB:            d,
		RulePrefs:     ruleprefs.NewPreferences(d),
		ToggleLimiter: ruleprefs.NewToggleLimiter(100, time.Minute),
		Insights:      insights.NewAcknowledgments(d),
	}
	return srv, func() { _ = d.Close() }
}

// sqlOpen é helper local que abre SQLite in-memory.
func sqlOpen() (*sql.DB, error) {
	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	return d, nil
}

// _ = silencia warning de imports não usados (slog, io).
var _ = slog.Default
var _ = io.Discard

func TestRecommendations_IncludesAcknowledgedStatus(t *testing.T) {
	srv, cleanup := setupRecommendationsTest(t)
	defer cleanup()
	ctx := context.Background()

	// Setup: insert 50 failures for "demo" IF (B12 = top, > 25% threshold)
	for i := 0; i < 50; i++ {
		_, _ = srv.DB.ExecContext(ctx, `INSERT INTO rule_failures (if_id, cadoc_code, rule_code, rule_severity) VALUES (?, ?, ?, ?)`,
			"demo", "4060", "B12", "E")
	}
	_, _ = srv.DB.ExecContext(ctx, `INSERT INTO rule_failures (if_id, cadoc_code, rule_code, rule_severity) VALUES (?, ?, ?, ?)`,
		"demo", "4060", "F99", "A")

	// Marca a recommendation esperada como acknowledged
	// (rec_id é SHA256 hash de (ifID, kind, top.Code) — não do headline)
	expectedID := recID("demo", "concentration", "B12")
	_, err := srv.Insights.Acknowledge(ctx, "demo", expectedID, "user-1")
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	t.Logf("expectedID: %s", expectedID)

	// Request
	req := httptest.NewRequest("GET", "/v1/insights/recommendations", nil)
	req = ctxWithClaims(req, "demo", "user-1")
	w := httptest.NewRecorder()
	srv.insightsRecommendations(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Recommendations []struct {
			ID             string     `json:"id"`
			Acknowledged   bool       `json:"acknowledged"`
			AcknowledgedAt *time.Time `json:"acknowledged_at"`
		} `json:"recommendations"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Recommendations) == 0 {
		t.Fatal("expected at least 1 recommendation")
	}

	// A rec ackada deve ter acknowledged=true
	found := false
	for _, r := range resp.Recommendations {
		t.Logf("rec in response: id=%s acked=%v", r.ID, r.Acknowledged)
		if r.ID == expectedID {
			found = true
			if !r.Acknowledged {
				t.Errorf("expected acknowledged=true for %s, got false", r.ID)
			}
			if r.AcknowledgedAt == nil {
				t.Error("expected acknowledged_at to be set")
			}
		} else {
			if r.Acknowledged {
				t.Errorf("expected acknowledged=false for non-acked %s, got true", r.ID)
			}
		}
	}
	if !found {
		t.Errorf("expected to find recommendation %s, but it wasn't in the list. total recs=%d", expectedID, len(resp.Recommendations))
	}
}

func TestRecommendations_NoInsightsService_StillWorks(t *testing.T) {
	d, err := sqlOpen()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer d.Close()

	for _, stmt := range []string{
		`CREATE TABLE criticas (id INTEGER PRIMARY KEY AUTOINCREMENT, cadoc_code TEXT, sheet TEXT, codigo TEXT, regra TEXT, descricao TEXT, gravidade TEXT, data_base_inicio DATETIME, mensagem_erro TEXT, enabled INTEGER DEFAULT 1)`,
		`CREATE TABLE rule_failures (id INTEGER PRIMARY KEY AUTOINCREMENT, if_id TEXT, cadoc_code TEXT, rule_code TEXT, rule_severity TEXT, envio_id TEXT, failed_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE envios (id TEXT PRIMARY KEY, if_id TEXT, cadoc_code TEXT, period TEXT, status TEXT, rules_passed INT, rules_failed INT, duration_ms INT, protocol_sta TEXT, error_code TEXT, error_message TEXT, sent_at DATETIME, confirmed_at DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
	} {
		_, _ = d.Exec(stmt)
	}

	srv := &Server{
		DB:            d,
		RulePrefs:     ruleprefs.NewPreferences(d),
		ToggleLimiter: ruleprefs.NewToggleLimiter(100, time.Minute),
		Insights:      nil, // Não injetado — legacy mode
	}

	req := httptest.NewRequest("GET", "/v1/insights/recommendations", nil)
	req = ctxWithClaims(req, "demo", "user-1")
	w := httptest.NewRecorder()
	srv.insightsRecommendations(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200 even without Insights service, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Recommendations []map[string]any `json:"recommendations"`
	}
	_ = json.NewDecoder(w.Body).Decode(&resp)
	for _, r := range resp.Recommendations {
		if r["acknowledged"] != false {
			t.Errorf("expected acknowledged=false without service, got %v", r["acknowledged"])
		}
	}
}

func TestToggleLimiter_Stats(t *testing.T) {
	l := ruleprefs.NewToggleLimiter(5, time.Minute)
	ctx := context.Background()

	if s := l.Stats(); s != 0 {
		t.Errorf("expected 0 keys initially, got %d", s)
	}

	for i := 0; i < 3; i++ {
		ok, _ := l.Allow("demo")
		if !ok {
			t.Fatalf("call %d should be allowed", i)
		}
	}

	if s := l.Stats(); s != 1 {
		t.Errorf("expected 1 key (demo), got %d", s)
	}

	// Different key
	ok, _ := l.Allow("other")
	if !ok {
		t.Fatal("first call to other should be allowed")
	}

	if s := l.Stats(); s != 2 {
		t.Errorf("expected 2 keys, got %d", s)
	}

	_ = ctx
}
