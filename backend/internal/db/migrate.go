// Package db aplica migrations SQL no banco.
//
// Tracking via tabela `schema_migrations`. Cada migration é aplicada uma
// única vez (mesmo conteúdo embedado). Para criar nova migration, basta
// adicionar `migrations/NNN_nome.sql` no diretório.
//
// Usa BEGIN IMMEDIATE (lock write no SQLite) pra evitar race entre
// múltiplas instâncias (ex: API + worker rodando ao mesmo tempo).
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationCount retorna o número de migrations no embed FS.
// Helper para testes (não-exportar migrationsFS deixa API limpa).
// Sprint 30 (v3.33.0): adicionado quando test de migrate hardcodou
// "want 13" — adicionar migration 014 quebrou test. Dinamizou.
func MigrationCount() int {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}

// isPostgres detecta o driver. Para simplificar, usamos o nome do driver
// exposto pelo sql.Driver. Migrate() também checa isso — single source
// of truth em IsPostgresDSN, mas ele só roda no open. Aqui usamos um
// scanner raso que checa PRAGMA (SQLite-only) — se rodar, é SQLite.
func isPostgresDB(d *sql.DB) bool {
	var v string
	if err := d.QueryRow("SELECT sqlite_version()").Scan(&v); err == nil {
		return false
	}
	return true // provavelmente Postgres
}

// slogDefault retorna logger default (helper pra evitar import cíclico).
func slogDefault() *slog.Logger {
	return slog.Default()
}

// Migrate aplica migrations pendentes em ordem. Idempotente e safe
// contra execução concorrente (BEGIN IMMEDIATE).
func Migrate(d *sql.DB) error {
	isPostgres := isPostgresDB(d)
	_ = isPostgres // usado dentro do loop via closure abaixo
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Tabela de controle (criada uma vez, idempotente)
	if _, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name        TEXT PRIMARY KEY,
			applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		// 2. Check se já aplicada (rápido, sem lock)
		var appliedAt string
		err := d.QueryRowContext(ctx,
			"SELECT applied_at FROM schema_migrations WHERE name = ?", name,
		).Scan(&appliedAt)
		if err == nil {
			continue // já aplicada
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		// 3. BEGIN IMMEDIATE (lock write — impede 2 instâncias aplicarem ao mesmo tempo)
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx %s: %w", name, err)
		}
		// Rollback se algo falhar (no-op se Commit rodar)
		defer func() { _ = tx.Rollback() }()

		// 4. Re-check dentro da transação (race-free)
		err = tx.QueryRowContext(ctx,
			"SELECT applied_at FROM schema_migrations WHERE name = ?", name,
		).Scan(&appliedAt)
		if err == nil {
			_ = tx.Commit()
			continue // outra instância aplicou no meio tempo
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("re-check migration %s: %w", name, err)
		}

		// 5. Lê e aplica SQL
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		// Sprint 13 [S14.3]: skip Postgres-only migrations em SQLite.
		// RLS policies (migrations/012) usam `ALTER TABLE ... ENABLE ROW
		// LEVEL SECURITY` que é syntax Postgres-only. Marcador
		// `-- @postgres-only` no header do arquivo indica skip em SQLite.
		sqlStr := string(content)
		if !isPostgres && strings.Contains(sqlStr, "-- @postgres-only") {
			slogDefault().Info("skipping postgres-only migration",
				"name", name,
				"hint", "apply manually on Postgres: psql -f migrations/"+name)
			// Marca como aplicada pra não tentar de novo.
			if _, err := tx.ExecContext(ctx,
				"INSERT INTO schema_migrations (name) VALUES (?)", name,
			); err != nil {
				return fmt.Errorf("mark %s applied (skipped): %w", name, err)
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("commit %s: %w", name, err)
			}
			continue
		}

		if _, err := tx.ExecContext(ctx, sqlStr); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		// 6. Marca como aplicada (mesma tx)
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (name) VALUES (?)", name,
		); err != nil {
			return fmt.Errorf("mark %s applied: %w", name, err)
		}

		// 7. Commit
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}

	return nil
}
