// Tests para tenant.go (WithTenantTx + helpers).
//
// Sprint 30 (v3.33.0): helper é crítico para FORCE RLS funcionar.
// Sem testes unitários, regressões passariam despercebidas.
package db_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// TestWithTenantTx_CommitCallbackError valida que erro em callback é
// retornado wrapped (fail-loud).
func TestWithTenantTx_CommitCallbackError(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	wantErr := errors.New("callback fail")
	err := db.WithTenantTx(context.Background(), d, "if-test",
		func(tx *sql.Tx) error { return wantErr })
	if err == nil {
		t.Fatal("expected error from callback, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error chain should wrap wantErr, got: %v", err)
	}
}

// TestWithTenantTx_EmptyIFID valida que ifID vazio é aceito em SQLite
// (helper é no-op). Em Postgres, validateIFID rejeita — mas como não temos
// Postgres em test, validamos só o caminho SQLite.
//
// Importante: caller pode passar "" intencionalmente (admin/system actions).
func TestWithTenantTx_EmptyIFID(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	err := db.WithTenantTx(context.Background(), d, "",
		func(tx *sql.Tx) error { return nil })
	if err != nil {
		t.Errorf("WithTenantTx com ifID vazio em SQLite deve ser no-op, got: %v", err)
	}
}

// TestWithTenantTx_ValidIFID valida happy path: ifID alfanumérico, callback
// roda, tx commit.
func TestWithTenantTx_ValidIFID(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	called := false
	err := db.WithTenantTx(context.Background(), d, "if-bank-1",
		func(tx *sql.Tx) error {
			called = true
			return nil
		})
	if err != nil {
		t.Errorf("happy path should not error, got: %v", err)
	}
	if !called {
		t.Error("callback should have been invoked")
	}
}

// TestWithTenantTx_RollbackOnError valida que tx é rolled back quando
// callback retorna erro. Estratégia: tentar INSERT dentro do callback que
// falha (constraint violation), depois INSERT fora do tx que deveria ver
// estado consistente (sem a row).
func TestWithTenantTx_RollbackOnError(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	// Setup tabela para validar rollback.
	if _, err := d.Exec(`CREATE TABLE t_rollback_test (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Callback que faz INSERT e depois retorna erro.
	err := db.WithTenantTx(context.Background(), d, "if-test",
		func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(context.Background(),
				`INSERT INTO t_rollback_test (id) VALUES (1)`); err != nil {
				return err
			}
			return errors.New("force rollback")
		})
	if err == nil {
		t.Fatal("expected error")
	}

	// Validar que INSERT foi rolled back — tabela deve estar vazia.
	var count int
	if err := d.QueryRow(`SELECT COUNT(*) FROM t_rollback_test`).Scan(&count); err != nil {
		t.Fatalf("count after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after rollback, got %d", count)
	}
}

// TestWithTenantTx_NilDB valida que *sql.DB nil causa erro graceful
// (não panic).
func TestWithTenantTx_NilDB(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WithTenantTx com nil db não deve panic, got: %v", r)
		}
	}()

	err := db.WithTenantTx(context.Background(), nil, "if-test",
		func(tx *sql.Tx) error { return nil })
	if err == nil {
		t.Error("expected error com nil db")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("erro deveria mencionar nil, got: %v", err)
	}
}

// TestWithTenantTx_ContextCancel valida que context cancelado propaga erro.
func TestWithTenantTx_ContextCancel(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancela antes de chamar

	err := db.WithTenantTx(ctx, d, "if-test",
		func(tx *sql.Tx) error { return nil })
	if err == nil {
		t.Error("expected error com ctx cancelado")
	}
}
