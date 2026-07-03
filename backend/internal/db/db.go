// Package db gerencia a conexão com o banco de dados (SQLite no spike, Postgres em prod).
//
// Uso:
//
//	db, err := db.Open("radiant.db")
//	if err != nil { log.Fatal(err) }
//	defer db.Close()
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open abre uma conexão SQLite (puro Go via modernc.org/sqlite).
// Para trocar pra Postgres, basta usar o driver lib/pq ou pgx:
//   import _ "github.com/lib/pq"
//   dsn := "postgres://user:pass@localhost/radiant?sslmode=disable"
//   db, err = sql.Open("postgres", dsn)
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	// Pool
	d.SetMaxOpenConns(8)
	d.SetMaxIdleConns(2)
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return d, nil
}