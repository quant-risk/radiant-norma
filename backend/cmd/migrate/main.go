// Command migrate applies pending database migrations and exits.
// Supports both SQLite and PostgreSQL backends.
//
// Usage:
//   ./migrate                    # uses DATABASE_URL or radiant.db
//   ./migrate -db postgres://... # explicit DSN
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

func main() {
	var dbPath = flag.String("db", "", "DSN or file path")
	flag.Parse()

	resolvedDB := *dbPath
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		resolvedDB = envDB
	}
	if resolvedDB == "" {
		resolvedDB = "radiant.db"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	d, err := db.Open(resolvedDB)
	if err != nil {
		logger.Error("db open failed", "err", err, "backend", db.Backend(resolvedDB))
		os.Exit(1)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	count := db.MigrationCount()
	logger.Info("migrations applied", "backend", db.Backend(resolvedDB), "total_migrations", count)
	fmt.Println("OK —", count, "migrations applied")
}
