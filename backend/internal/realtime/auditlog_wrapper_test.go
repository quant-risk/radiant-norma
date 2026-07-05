// Package realtime — tests do HubAwareLogger decorator.
package realtime

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	_ "modernc.org/sqlite"
)

// testAuditLog cria auditlog.Logger + DB SQLite temporário pra tests.
func testAuditLog(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "auditlog-test-*")
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Cria tabela audit_log mínima pro test (matches migration 001_initial).
	_, err = d.Exec(`CREATE TABLE IF NOT EXISTS audit_log (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		if_id           TEXT,
		actor           TEXT NOT NULL,
		action          TEXT NOT NULL,
		target          TEXT,
		payload_hash    TEXT NOT NULL,
		prev_hash       TEXT NOT NULL,
		entry_hash      TEXT NOT NULL,
		metadata        TEXT,
		created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	cleanup := func() {
		_ = d.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return d, cleanup
}

func TestHubAwareLogger_PublishesAfterLog(t *testing.T) {
	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	d, cleanup := testAuditLog(t)
	defer cleanup()

	// Cria auditlog.Logger real
	al := auditlog.New(d)

	wrap := WrapAuditLog(al, hub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, unreg := hub.Subscribe(ctx, "demo")
	defer unreg()

	// Log via wrapper
	entry, err := wrap.Log("demo", "user-1", "test.action", "target-1", []byte(`{"k":"v"}`), nil)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if entry == nil {
		t.Fatal("entry nil")
	}

	// Verifica evento publicado
	select {
	case evt := <-events:
		if evt.Kind != "test.action" {
			t.Errorf("expected kind=test.action, got %s", evt.Kind)
		}
		if evt.IFID != "demo" {
			t.Errorf("expected if_id=demo, got %s", evt.IFID)
		}
		if evt.Payload["action"] != "test.action" {
			t.Errorf("payload missing action, got %v", evt.Payload)
		}
		if evt.Payload["actor"] != "user-1" {
			t.Errorf("payload missing actor, got %v", evt.Payload)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("event not published")
	}
}

func TestHubAwareLogger_VerifyChainUnchanged(t *testing.T) {
	hub := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	d, cleanup := testAuditLog(t)
	defer cleanup()

	al := auditlog.New(d)
	wrap := WrapAuditLog(al, hub)

	// Adiciona 3 entries
	for i := 0; i < 3; i++ {
		_, err := wrap.Log("demo", "user-1", "test.action", "target-1", nil, nil)
		if err != nil {
			t.Fatalf("Log %d: %v", i, err)
		}
	}

	// Chain deve estar intacto (HubAwareLogger não altera hash chain)
	ok, count, err := wrap.Verify()
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Errorf("expected chain valid, got invalid")
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

func TestHubAwareLogger_NoHubIsNoOp(t *testing.T) {
	d, cleanup := testAuditLog(t)
	defer cleanup()

	al := auditlog.New(d)

	// Sem hub — não deve panic
	wrap := WrapAuditLog(al, nil)

	entry, err := wrap.Log("demo", "user-1", "test.action", "target-1", nil, nil)
	if err != nil {
		t.Fatalf("Log without hub: %v", err)
	}
	if entry == nil {
		t.Fatal("entry nil without hub")
	}
}