// Command worker é o processador assíncrono de envios STA.
//
// Por design: cmd/api recebe requests HTTP e enfileira envios. cmd/worker
// processa a fila: valida → submete STA → atualiza status. Isso desacopla
// latência HTTP da latência do BACEN (que pode levar minutos).
//
// Lógica de processamento está em internal/worker (testável). Aqui só
// temos flag parsing, signal handling, e loop principal.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	workpkg "github.com/fortvna/radiant-norma/backend/internal/worker"
)

func main() {
	var (
		dbPath   = flag.String("db", "", "DSN — DB path (SQLite) ou postgres://... (Postgres). DATABASE_URL env sobrescreve.")
		interval = flag.Duration("interval", 30*time.Second, "tick interval")
		batch    = flag.Int("batch", 10, "max envios per tick")
		once     = flag.Bool("once", false, "processa 1 batch e sai (útil pra teste)")
	)
	flag.Parse()

	// Sprint 6 v1.5.0 (F12.2 follow-up): DATABASE_URL > -db flag.
	// Mantém -db pra retrocompat; prefira env var em produção.
	resolvedDB := *dbPath
	if envDB := os.Getenv("DATABASE_URL"); envDB != "" {
		resolvedDB = envDB
	}
	if resolvedDB == "" {
		resolvedDB = "radiant.db" // dev default
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Open DB
	d, err := db.Open(resolvedDB)
	if err != nil {
		logger.Error("db open failed", "err", err, "backend", db.Backend(resolvedDB))
		os.Exit(1)
	}
	defer d.Close()

	// Migrations (worker pode rodar standalone antes da API ter criado schema)
	if err := db.Migrate(d); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	// Init services
	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staClient := sta.NewStubClient()

	logger.Info("worker started",
		// Validação 14 (F14.1): NÃO logar DSN (pode conter password Postgres).
		// Logger apenas backend name (sqlite/postgres) — não loga path/host.
		"backend", db.Backend(resolvedDB),
		"interval", interval.String(),
		"batch", *batch,
	)

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Sweeper rodando em paralelo (W2)
	go runLeaseSweeperLoop(ctx, d, logger)

	// Processa imediatamente no boot
	n, _ := workpkg.ProcessBatch(ctx, d, auditSvc, auditLog, staClient, *batch, logger)
	logger.Info("initial batch done", "processed", n)

	if *once {
		logger.Info("once mode: exiting")
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped", "reason", ctx.Err())
			return
		case <-ticker.C:
			n, err := workpkg.ProcessBatch(ctx, d, auditSvc, auditLog, staClient, *batch, logger)
			if err != nil {
				logger.Error("batch failed", "err", err)
				continue
			}
			if n > 0 {
				logger.Info("batch processed", "count", n)
			}
		}
	}
}

// runLeaseSweeperLoop executa o lease sweeper periodicamente até ctx.Done.
//
// W2 (Sprint 6 v1.5.0): detecta worker crashes ressetando envios stuck em
// 'processing' por mais de workpkg.LeaseTimeout (5min).
//
// Validação 13 (F13.6 follow-up): panic recover — sem isso, panic no
// sweeper mataria essa goroutine (silenciosamente — lease sweep para
// sem aviso).
func runLeaseSweeperLoop(ctx context.Context, d *sql.DB, logger *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("lease sweeper panic recovered (goroutine ending)",
				"panic", r)
		}
	}()
	ticker := time.NewTicker(workpkg.LeaseSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := workpkg.RunLeaseSweeper(ctx, d, logger); err != nil {
				logger.Error("lease sweeper failed", "err", err)
			}
		}
	}
}
