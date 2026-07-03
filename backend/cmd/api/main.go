// cmd/api: entrypoint da API REST do Radiant Norma
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr := envOr("RADIANT_ADDR", ":8080")
	dbPath := envOr("RADIANT_DB", "radiant.db")

	// DB
	d, err := db.Open(dbPath)
	if err != nil {
		logger.Error("open db", "err", err)
		os.Exit(1)
	}
	defer d.Close()
	logger.Info("db connected", "path", dbPath)

	// Migrations
	if err := db.Migrate(d); err != nil {
		logger.Error("migrations", "err", err)
		os.Exit(1)
	}
	logger.Info("migrations applied")

	// Services
	schReg := schema.New(d)
	audSvc := audit.New(d)
	audLog := auditlog.New(d)
	staClient := sta.NewStubClient()
	radarSvc := radar.New(d, 6*time.Hour)

	srv := api.NewServer(d, schReg, audSvc, audLog, staClient, radarSvc)
	handler := srv.Router()

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second, // Slowloris protection
		WriteTimeout:      60 * time.Second, // validações podem demorar
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown — escuta erros em goroutine sem chamar os.Exit
	// (que sairia sem dar chance pro Shutdown rodar).
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("api listening", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Aguarda sinal OU erro fatal do server
	select {
	case err := <-serverErr:
		logger.Error("server fatal", "err", err)
		os.Exit(1)
	case sig := <-stop:
		logger.Info("shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "err", err)
		os.Exit(1)
	}
	logger.Info("bye")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}