// Command radar é o worker que detecta mudanças regulatórias.
//
// Por design: roda como daemon, faz scan periódico das URLs BACEN
// conhecidas, persiste hashes baseline e insere alertas quando muda.
//
// Diferencial de marketing: "first-mover" — Radiant Norma detecta
// mudanças de leiaute ANTES do BACEN publicar oficialmente.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
)

func main() {
	var (
		dbPath   = flag.String("db", "", "DSN — DB path (SQLite) ou postgres://... (Postgres). DATABASE_URL env sobrescreve.")
		interval = flag.Duration("interval", 6*time.Hour, "interval between scans (default 6h)")
		once     = flag.Bool("once", false, "scan once and exit")
		verbose  = flag.Bool("v", false, "verbose (debug) logging")
	)
	flag.Parse()

	// Sprint 6 v1.5.0 (F12.2 follow-up): DATABASE_URL > -db flag.
	resolvedDB := *dbPath
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		resolvedDB = envDB
	}
	if resolvedDB == "" {
		resolvedDB = "radiant.db" // dev default
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	d, err := db.Open(resolvedDB)
	if err != nil {
		// Validação 15 (F15.1): sanitizar err.
		logger.Error("db open failed", "err", loggerutil.SafeError(err), "backend", db.Backend(resolvedDB))
		os.Exit(1)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		logger.Error("migrate failed", "err", loggerutil.SafeError(err))
		os.Exit(1)
	}

	svc := radar.New(d, *interval)
	svc.SetLogger(logger)

	logger.Info("radar worker started",
		// Validação 14 (F14.1): NÃO logar DSN (pode conter password Postgres).
		// Logger apenas backend name (sqlite/postgres) — não loga path/host.
		"backend", db.Backend(resolvedDB),
		"interval", interval.String(),
		"once", *once,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	scan := func() {
		// Validação 13 (F13.6): panic recover. Sem isso, panic no
		// ScanOnce mataria o goroutine principal sem log de erro.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("radar scan panic recovered",
					"panic", r,
					"stack_hint", "scanner continua na próxima tick")
			}
		}()
		alerts, err := svc.ScanOnce(ctx, nil) // nil = DefaultSources
		if err != nil {
			logger.Error("scan failed", "err", err)
			return
		}
		for _, a := range alerts {
			logger.Info("new alert",
				"id", a.ID,
				"cadoc", a.CadocCode,
				"severity", a.Severity,
				"title", a.Title,
			)
		}
	}

	scan() // immediate

	if *once {
		svc.Close()
		logger.Info("once mode: exiting")
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	defer svc.Close()

	for {
		select {
		case <-ctx.Done():
			logger.Info("radar worker stopped", "reason", ctx.Err())
			return
		case <-ticker.C:
			scan()
		}
	}
}
