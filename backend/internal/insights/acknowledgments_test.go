// Package insights — tests do Acknowledgments service.

package insights

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir, _ := os.MkdirTemp("", "insights-test-*")
	defer t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	d, err := sql.Open("sqlite", filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, err = d.Exec(`CREATE TABLE acknowledged_recommendations (
		if_id TEXT NOT NULL,
		rec_id TEXT NOT NULL,
		acknowledged_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		acknowledged_by TEXT NOT NULL,
		PRIMARY KEY (if_id, rec_id)
	)`)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return d, func() { _ = d.Close() }
}

func TestAcknowledgments_Acknowledge(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)

	r, err := a.Acknowledge(context.Background(), "demo", "rec-1", "user-1")
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	if r.IFID != "demo" || r.RecID != "rec-1" || r.AcknowledgedBy != "user-1" {
		t.Errorf("got %+v", r)
	}
}

func TestAcknowledgments_IsAcknowledged(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)
	ctx := context.Background()

	ok, _ := a.IsAcknowledged(ctx, "demo", "rec-1")
	if ok {
		t.Error("should not be acknowledged initially")
	}

	_, _ = a.Acknowledge(ctx, "demo", "rec-1", "user-1")
	ok, _ = a.IsAcknowledged(ctx, "demo", "rec-1")
	if !ok {
		t.Error("should be acknowledged after Acknowledge")
	}
}

func TestAcknowledgments_AcknowledgeIdempotent(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)
	ctx := context.Background()

	_, _ = a.Acknowledge(ctx, "demo", "rec-1", "user-1")
	_, _ = a.Acknowledge(ctx, "demo", "rec-1", "user-2") // mesmo ID, actor diferente

	ok, _ := a.IsAcknowledged(ctx, "demo", "rec-1")
	if !ok {
		t.Error("should still be acknowledged")
	}
}

func TestAcknowledgments_Unacknowledge(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)
	ctx := context.Background()

	_, _ = a.Acknowledge(ctx, "demo", "rec-1", "user-1")

	err := a.Unacknowledge(ctx, "demo", "rec-1")
	if err != nil {
		t.Fatalf("Unacknowledge: %v", err)
	}

	ok, _ := a.IsAcknowledged(ctx, "demo", "rec-1")
	if ok {
		t.Error("should not be acknowledged after Unacknowledge")
	}
}

func TestAcknowledgments_Unacknowledge_NotAcknowledged(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)

	err := a.Unacknowledge(context.Background(), "demo", "rec-missing")
	if err != ErrRecommendationNotAcknowledged {
		t.Errorf("expected ErrRecommendationNotAcknowledged, got %v", err)
	}
}

func TestAcknowledgments_ListAcknowledged(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	a := NewAcknowledgments(d)
	ctx := context.Background()

	_, _ = a.Acknowledge(ctx, "demo", "rec-1", "user-1")
	_, _ = a.Acknowledge(ctx, "demo", "rec-2", "user-1")
	_, _ = a.Acknowledge(ctx, "other", "rec-3", "user-1")

	out, _ := a.ListAcknowledged(ctx, "demo")
	if len(out) != 2 {
		t.Errorf("expected 2 acknowledged for demo, got %d", len(out))
	}
	if _, ok := out["rec-1"]; !ok {
		t.Error("expected rec-1")
	}
	if _, ok := out["rec-2"]; !ok {
		t.Error("expected rec-2")
	}
}