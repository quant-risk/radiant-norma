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
	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	crossrules "github.com/fortvna/radiant-norma/backend/internal/crossdoc/rules"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	addr := envOr("RADIANT_ADDR", ":8080")
	// Sprint 6 v1.5.0 (F12.2 fix): DATABASE_URL tem prioridade — Postgres
	// quando detecta prefixo postgres://. Fallback SQLite pra dev.
	dbPath := envOr("DATABASE_URL", envOr("RADIANT_DB", "radiant.db"))

	// DB
	d, err := db.Open(dbPath)
	if err != nil {
		// Validação 15 (F15.1): sanitizar err para não vazar DSN.
		// pgx error messages incluem user+database (testado).
		logger.Error("open db", "err", loggerutil.SafeError(err), "backend", db.Backend(dbPath))
		os.Exit(1)
	}
	defer d.Close()
	logger.Info("db connected", "backend", db.Backend(dbPath))

	// Migrations
	if err := db.Migrate(d); err != nil {
		logger.Error("migrations", "err", loggerutil.SafeError(err))
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

	// Sprint 7a (v1.6.0): JWT verifier setup.
	//
	// Lê chave pública de env var (PEM-encoded). Default: dev fallback
	// X-IF-ID via RADIANT_DEV_AUTH=1. Prod: setar RADIANT_JWT_PUBLIC_KEY.
	//
	// Em produção, recomenda-se AWS KMS / Vault para rotação da chave
	// sem downtime (Sprint 8 follow-up).
	if pubKeyPEM := os.Getenv("RADIANT_JWT_PUBLIC_KEY"); pubKeyPEM != "" {
		pub, err := auth.ParsePublicKeyPEM([]byte(pubKeyPEM))
		if err != nil {
			logger.Error("parse JWT public key", "err", loggerutil.SafeError(err))
			os.Exit(1)
		}
		keyring := auth.NewKeyring()
		keyring.Add(&auth.Key{
			Kid:       os.Getenv("RADIANT_JWT_KID"),
			PublicKey: pub,
			Active:    true,
			CreatedAt: time.Now(),
		})
		srv.Auth = auth.NewVerifier(auth.Config{
			Issuer: os.Getenv("RADIANT_JWT_ISSUER"),
			Leeway: 30 * time.Second,
		}, keyring)
		logger.Info("JWT verifier ativo", "issuer", os.Getenv("RADIANT_JWT_ISSUER"))
	} else if os.Getenv("RADIANT_DEV_AUTH") == "1" {
		logger.Warn("RADIANT_DEV_AUTH=1 — X-IF-ID aceito. NÃO USE EM PRODUÇÃO.")
	} else {
		logger.Error("RADIANT_JWT_PUBLIC_KEY não configurada — /v1/* retorna 401. " +
			"Para dev: set RADIANT_DEV_AUTH=1. Para prod: configure chave pública.")
		// Não os.Exit: middlewares podem ser configurados via /readyz
		// sem auth, então healthcheck ainda funciona.
	}

	// Sprint 6 v1.5.0 — hardening components wiring.
	// ANTES (v1.5.0 shipped): esses 4 componentes ficavam nil → endpoints
	// /v1/crossdoc/validate e /v1/radar/scan retornavam 503/401.
	// Validação 12 (F12.2): fix que ativa hardening em produção.

	// W4 — cadoc list com cache 5min (sempre ativo; sem env var).
	srv.CadocListCache = schema.NewCadocListCache(5 * time.Minute)

	// R1 — DOS-via-API prevention. AdminAuth é FAIL CLOSED: sem token
	// configurado, /v1/radar/scan retorna 401.
	adminToken := os.Getenv("RADIANT_NORMA_ADMIN_TOKEN")
	srv.AdminAuth = &radar.AdminAuth{Token: adminToken}
	if adminToken == "" {
		logger.Warn("RADIANT_NORMA_ADMIN_TOKEN não configurado — /v1/radar/scan retorna 401 (admin auth FAIL CLOSED)")
	} else {
		// Validação 13 (F13.8): NÃO logar prefix do token em logs.
		// Senão, logs viram vetor de secret disclosure (logs são
		// tipicamente persistidos + agregados + podem vazar).
		// Em vez disso, log apenas que está configurado, com hash SHA-256 truncated.
		// (implementação deliberadamente simples — token config é nota binária)
		logger.Info("admin auth configurado", "token_length", len(adminToken))
	}
	srv.ScanLimiter = radar.NewScanLimiter(1 * time.Minute)
	srv.ScanCache = radar.NewScanCache(5 * time.Minute)

	// Cross-Doc L3 — endpoint /v1/crossdoc/validate.
	srv.CrossDoc = crossdoc.NewEngine(crossrules.BuiltinRegistry())

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
		// Validação 14 (F14.2): panic recover (defense in depth).
		// ListenAndServe raramente panica, mas se panic o servidor
		// morre sem avisar. Recover + log + exit.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("server goroutine panic", "panic", r)
				os.Exit(1)
			}
		}()
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
		// Validação 16 (F16.1): sanitizar err para não vazar DSN.
		logger.Error("server fatal", "err", loggerutil.SafeError(err))
		os.Exit(1)
	case sig := <-stop:
		logger.Info("shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		// Validação 16 (F16.1): sanitizar err para não vazar DSN.
		logger.Error("shutdown", "err", loggerutil.SafeError(err))
		os.Exit(1)
	}
	// Libera HTTP connections idle do Radar
	radarSvc.Close()
	logger.Info("bye")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// NOTA: usamos Go stdlib `min` (built-in desde 1.21). Removido wrapper
// customizado na validação 13 (F13.1) — memory pattern "reinventar
// stdlib".
