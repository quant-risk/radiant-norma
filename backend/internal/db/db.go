// Package db gerencia a conexão com o banco de dados.
//
// Sprint 6 v1.5.0: dual driver — SQLite (modernc puro Go, sem CGo) ou
// Postgres (pgx/v5 puro Go, sem CGo). Detecção automática via prefixo
// da DSN:
//
//	SQLite:  file:radiant.db?_pragma=...
//	Postgres: postgres://user:pass@host/db?sslmode=disable
//
// Uso:
//
//	db, err := db.Open("radiant.db")              // SQLite
//	db, err := db.Open("postgres://localhost/db") // Postgres
package db

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver (database/sql)
	_ "modernc.org/sqlite"             // SQLite driver (pure Go, no CGo)
)

// Open abre uma conexão SQLite OU Postgres baseado no DSN.
//
// Sprint 6 v1.5.0: dual driver detection via prefix.
//   - "postgres://" ou "postgresql://" → Postgres (pgx/v5)
//   - "file:" ou path cru → SQLite (modernc.org/sqlite)
//
// Para trocar pra Postgres:
//
//	dsn := "postgres://user:pass@localhost:5432/radiant?sslmode=disable"
//	db, err = sql.Open("pgx", dsn)
func Open(dsn string) (*sql.DB, error) {
	if IsPostgresDSN(dsn) {
		return openPostgres(dsn)
	}
	return openSQLite(dsn)
}

// IsPostgresDSN retorna true se DSN começa com prefixo Postgres.
//
// Aceita "postgres://" e "postgresql://".
//
// Exportado (Sprint 6 v1.5.0) para que testes possam validar detecção
// sem precisar de conexão real.
func IsPostgresDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") ||
		strings.HasPrefix(dsn, "postgresql://")
}

// openSQLite abre SQLite puro-Go com DSN moderna.
//
// Configuração (preservada de v1.4.x):
//   - _txlock=immediate: BEGIN IMMEDIATE em toda transação
//     (sem ele, múltiplos goroutines pegam o mesmo prev_hash no SELECT
//     auditlog, depois colidem no INSERT — SQLITE_BUSY ou PrevHash dup).
//   - journal_mode=WAL: melhor concorrência (readers não bloqueiam writer).
//   - foreign_keys=ON: integridade referencial.
//
// Trade-off: contenção extra em leituras. Em produção, usar Postgres.
func openSQLite(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_txlock=immediate", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open sqlite: %w", err)
	}
	d.SetMaxOpenConns(8)
	d.SetMaxIdleConns(2)
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return d, nil
}

// openPostgres abre Postgres via pgx/v5 (database/sql bridge).
//
// Configuração:
//   - pool: 25 conexões (Postgres aguenta mais que SQLite)
//   - sslmode=require em produção (DSN controla)
//
// Em produção:
//   - Migrations: precisam ser Postgres-flavor (sem features SQLite-only
//     como ALTER TABLE sem IF EXISTS que alguns migration scripts usam)
//   - Sprint 6 v1.5.0: migrations existentes são compatíveis (uso
//     mínimo de features SQLite-specific, testadas manualmente)
func openPostgres(dsn string) (*sql.DB, error) {
	d, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open pgx: %w", err)
	}
	d.SetMaxOpenConns(25)
	d.SetMaxIdleConns(5)
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return d, nil
}

// Backend retorna o nome do backend ("sqlite" ou "postgres") detectado
// a partir da DSN. Útil para logging / debug.
func Backend(dsn string) string {
	if IsPostgresDSN(dsn) {
		return "postgres"
	}
	return "sqlite"
}
