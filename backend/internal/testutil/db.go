// Package testutil provides helpers for testing database-using code.
//
// Padrão: cada teste que precisa de DB usa NewTestDB(t) que retorna um
// *sql.DB conectado a SQLite in-memory com migrations aplicadas.
//
// Vantagens:
//   - Isolamento total: cada teste tem seu próprio banco
//   - Velocidade: in-memory é ~10x mais rápido que arquivo
//   - Cleanup automático: t.Cleanup fecha DB
//
// Uso:
//
//	func TestSomething(t *testing.T) {
//	    d := testutil.NewTestDB(t)
//	    // ... usa d
//	}
package testutil

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

// NewTestDB cria um SQLite in-memory com migrations aplicadas.
// O DB é automaticamente fechado em t.Cleanup.
//
// Aplica o mesmo DSN que produção (com _txlock=immediate) para que testes
// reproduzam comportamento real de concorrência.
//
// Sprint 13 [S14.1]: pre-seeda IFs comuns para satisfazer FK
// audit_log.if_id → ifs(id). Testes que emitem audit (validate,
// radar resolve, ack) sem pre-seed IF explícito agora funcionam.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// file::memory:?cache=shared permite múltiplas conexões na mesma
	// in-memory DB (necessário para testar concorrência real com
	// sql.DB.SetMaxOpenConns > 1).
	//
	// Adicionamos um path único por teste para isolar completamente
	// (cada teste tem sua própria in-memory DB).
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if err := db.Migrate(d); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}

	// Sprint 13 [S14.1]: pre-seeds para satisfazer FK audit_log/audit_events.
	testIFs := []string{
		"if-test", "if-concurrent", "if-demo", "if-1", "if-2", "if-b", "if-c",
		"if-x", "if-y", "if-d", "if-cache-audit",
		"audit-if", "demo", "demo-bank", "canary-if",
		"bank-1", "other", "system", "stress",
	}
	for i, ifID := range testIFs {
		_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, plano)
			VALUES (?, ?, ?, 'SCD', 'pro')`,
			ifID, fmt.Sprintf("%08d", i+100), "Test "+ifID)
	}

	t.Cleanup(func() {
		_ = d.Close()
	})

	return d
}

// NewInMemoryDB cria um SQLite puramente in-memory (sem arquivo).
// Mais rápido que NewTestDB mas não suporta múltiplas conexões compartilhadas.
//
// Use NewTestDB a menos que precise de velocidade máxima em testes single-goroutine.
func NewInMemoryDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use a unique DSN per test para isolamento.
	// :memory: é compartilhado entre todas as conexões, então usamos
	// um arquivo temporário com nome único.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "mem.db")

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}

	t.Cleanup(func() {
		_ = d.Close()
	})

	return d
}
