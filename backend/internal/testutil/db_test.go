package testutil_test

import (
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

func TestNewTestDB(t *testing.T) {
	d := testutil.NewTestDB(t)
	if d == nil {
		t.Fatal("expected db, got nil")
	}
	// Verifica que migrations aplicaram
	var name string
	err := d.QueryRow("SELECT name FROM schema_migrations LIMIT 1").Scan(&name)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if name == "" {
		t.Fatal("expected migration name")
	}
	t.Logf("first migration: %s", name)
}
