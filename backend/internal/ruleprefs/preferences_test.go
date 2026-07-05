// Package ruleprefs — tests do Preferences service.
//
// Usa SQLite in-memory pra tests rápidos. Migration 007 roda no setup.

package ruleprefs

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "ruleprefs-test-*")
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	_, err = d.Exec(`CREATE TABLE IF NOT EXISTS disabled_rules (
		if_id        TEXT NOT NULL,
		rule_code    TEXT NOT NULL,
		disabled_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		disabled_by  TEXT NOT NULL,
		PRIMARY KEY (if_id, rule_code)
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	return d, func() {
		_ = d.Close()
		_ = os.RemoveAll(tmpDir)
	}
}

func TestPreferences_Disable(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	r, err := p.Disable(ctx, "demo", "B12", "user-1")
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if r.IFID != "demo" || r.RuleCode != "B12" || r.DisabledBy != "user-1" {
		t.Errorf("Disable returned wrong record: %+v", r)
	}
	if r.DisabledAt.IsZero() {
		t.Error("DisabledAt should be set")
	}
}

func TestPreferences_Disable_Idempotent(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	r1, _ := p.Disable(ctx, "demo", "B12", "user-1")
	time.Sleep(10 * time.Millisecond)
	r2, _ := p.Disable(ctx, "demo", "B12", "user-2")

	if !r2.DisabledAt.After(r1.DisabledAt) {
		t.Errorf("second Disable should update timestamp: r1=%v, r2=%v", r1.DisabledAt, r2.DisabledAt)
	}
	if r2.DisabledBy != "user-2" {
		t.Errorf("expected disabled_by=user-2, got %s", r2.DisabledBy)
	}
}

func TestPreferences_Enable(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	_, _ = p.Disable(ctx, "demo", "B12", "user-1")
	if err := p.Enable(ctx, "demo", "B12"); err != nil {
		t.Fatalf("Enable: %v", err)
	}

	disabled, _ := p.IsDisabled(ctx, "demo", "B12")
	if disabled {
		t.Error("rule should be enabled after Enable")
	}
}

func TestPreferences_Enable_NotDisabled(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	err := p.Enable(ctx, "demo", "B12")
	if err != ErrRuleNotDisabled {
		t.Errorf("expected ErrRuleNotDisabled, got %v", err)
	}
}

func TestPreferences_IsDisabled(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	// Empty
	disabled, _ := p.IsDisabled(ctx, "demo", "B12")
	if disabled {
		t.Error("should not be disabled initially")
	}

	// After disable
	_, _ = p.Disable(ctx, "demo", "B12", "user-1")
	disabled, _ = p.IsDisabled(ctx, "demo", "B12")
	if !disabled {
		t.Error("should be disabled after Disable")
	}

	// Different IF (isolation test)
	disabled, _ = p.IsDisabled(ctx, "other", "B12")
	if disabled {
		t.Error("isolation broken: 'other' should NOT have B12 disabled")
	}
}

func TestPreferences_ListDisabled(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	// Empty
	rules, _ := p.ListDisabled(ctx, "demo")
	if len(rules) != 0 {
		t.Errorf("expected empty list, got %d", len(rules))
	}

	// Add 3 rules for demo
	_, _ = p.Disable(ctx, "demo", "B12", "user-1")
	_, _ = p.Disable(ctx, "demo", "F23", "user-1")
	_, _ = p.Disable(ctx, "demo", "S05", "user-2")

	// Add 1 rule for other (should NOT appear in demo list)
	_, _ = p.Disable(ctx, "other", "X99", "user-1")

	rules, _ = p.ListDisabled(ctx, "demo")
	if len(rules) != 3 {
		t.Errorf("expected 3 rules for demo, got %d", len(rules))
	}
	for _, r := range rules {
		if r.IFID != "demo" {
			t.Errorf("isolation broken: got IFID=%s in demo list", r.IFID)
		}
	}
}

func TestPreferences_Toggle(t *testing.T) {
	d, cleanup := setupTestDB(t)
	defer cleanup()
	p := NewPreferences(d)
	ctx := context.Background()

	// First toggle: enabled → disabled
	state, _ := p.Toggle(ctx, "demo", "B12", "user-1")
	if state != "disabled" {
		t.Errorf("expected 'disabled', got %q", state)
	}

	// Second toggle: disabled → enabled
	state, _ = p.Toggle(ctx, "demo", "B12", "user-1")
	if state != "enabled" {
		t.Errorf("expected 'enabled', got %q", state)
	}

	// Third toggle: enabled → disabled again
	state, _ = p.Toggle(ctx, "demo", "B12", "user-1")
	if state != "disabled" {
		t.Errorf("expected 'disabled' on 3rd toggle, got %q", state)
	}
}