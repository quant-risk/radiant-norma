// Package db aplica migrations SQL no banco.
//
// Tracking via tabela `schema_migrations`. Cada migration é aplicada uma
// única vez (mesmo conteúdo embedado). Para criar nova migration, basta
// adicionar `migrations/NNN_nome.sql` no diretório.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate aplica migrations pendentes em ordem. Idempotente.
func Migrate(d *sql.DB) error {
	// Tabela de controle
	if _, err := d.Exec(`
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
		// Verifica se já aplicada
		var appliedAt string
		err := d.QueryRow("SELECT applied_at FROM schema_migrations WHERE name = ?", name).Scan(&appliedAt)
		if err == nil {
			// Já aplicada — skip
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", name, err)
		}

		// Aplica
		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := d.Exec(string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		// Marca como aplicada
		if _, err := d.Exec(
			"INSERT INTO schema_migrations (name) VALUES (?)", name,
		); err != nil {
			return fmt.Errorf("mark %s applied: %w", name, err)
		}
	}

	return nil
}