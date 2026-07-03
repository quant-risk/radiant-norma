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
	"github.com/fortvna/radiant-norma/backend/internal/radar"
)

func main() {
	var (
		dbPath   = flag.String("db", "radiant.db", "path to SQLite database")
		interval = flag.Duration("interval", 6*time.Hour, "interval between scans (default 6h)")
		once     = flag.Bool("once", false, "scan once and exit")
		verbose  = flag.Bool("v", false, "verbose (debug) logging")
	)
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	d, err := db.Open(*dbPath)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	svc := radar.New(d, *interval)
	svc.SetLogger(logger)

	logger.Info("radar worker started",
		"db", *dbPath,
		"interval", interval.String(),
		"once", *once,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	scan := func() {
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
