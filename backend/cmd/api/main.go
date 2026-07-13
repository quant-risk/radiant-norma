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
	"github.com/fortvna/radiant-norma/backend/internal/branding"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	crossrules "github.com/fortvna/radiant-norma/backend/internal/crossdoc/rules"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	gen2030pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2030"
	gen2060pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2060"
	gen2061pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2061"
	gen2062pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2062"
	gen2070pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2070"
	gen2160pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2160"
	gen2170pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen2170"
	gen3040pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	gen3050pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen3050"
	gen4111pkg "github.com/fortvna/radiant-norma/backend/internal/generator/gen4111"
	"github.com/fortvna/radiant-norma/backend/internal/generator/wizard"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/marketplace"
	"github.com/fortvna/radiant-norma/backend/internal/observability"
	"github.com/fortvna/radiant-norma/backend/internal/pilot"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/realtime"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Sprint 36 — v3.36.0: Observability foundation.
	// OTel tracer: exports spans via OTLP (OTEL_EXPORTER_OTLP_ENDPOINT)
	// or console (dev). W3C trace context propagated via HTTP headers.
	otelShutdown, err := observability.InitTracer(context.Background(), observability.NewConfig())
	if err != nil {
		logger.Error("otel tracer init", "err", err)
		os.Exit(1)
	}
	defer otelShutdown()

	// Sentry: error tracking + release + breadcrumbs.
	// Disabled if SENTRY_DSN is empty (safe for dev).
	sentryShutdown, err := observability.InitSentry(observability.NewSentryConfig())
	if err != nil {
		logger.Error("sentry init", "err", err)
		os.Exit(1)
	}
	defer sentryShutdown()

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
	// Validação 56 (v3.33.2) [F-56-C]: limpa driverCache no shutdown
	// para não deixar entries orfãs (DSN já cached, conn fechada).
	defer db.ClearDriverCache(d)
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
	// Sprint 12 (v3.5.0): C32.23 — wire ruleprefs no audit.Service
	// pra que regras desabilitadas em /v1/rules/{code}/toggle afetem
	// validação real (antes era cosmético).
	audSvc.SetRulePrefs(ruleprefs.NewPreferences(d))
	audLog := auditlog.New(d)
	// Sprint 18 (v3.8.0): STA client factory. Default stub preserva
	// compatibilidade; WSClient ativa quando RADIANT_STA_BACKEND=ws.
	staClient, err := sta.NewClientFromEnv(logger)
	if err != nil {
		logger.Error("STA client init failed", "err", loggerutil.SafeError(err))
		os.Exit(1)
	}
	logger.Info("STA client inicializado", "backend", sta.BackendName(staClient))
	radarSvc := radar.New(d, 6*time.Hour)
	brandingSvc := branding.NewBrandingService(d)
	marketplaceSvc := marketplace.NewService(d)
	pilotSvc := pilot.NewService(d)

	// Sprint 53 — v3.34.35: AI Insights via LLM (opt-in).
	// Configurado via env vars: LLM_PROVIDER, LLM_API_KEY, LLM_MODEL.
	var insightsLLM *insights.LLMService
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		var llmClient insights.LLMClient
		switch os.Getenv("LLM_PROVIDER") {
		case "openai":
			llmClient = insights.NewOpenAIChat(apiKey,
				envOr("LLM_MODEL", "gpt-4o-mini"),
				envOr("LLM_BASE_URL", "https://api.openai.com/v1"))
		default: // minimax (default)
			llmClient = insights.NewMiniMaxChat(insights.MiniMaxConfig{
				APIKey:  apiKey,
				Model:   envOr("LLM_MODEL", "MiniMax-Text-01"),
				BaseURL: envOr("LLM_BASE_URL", "https://api.minimax.chat/v1"),
			})
		}
		insightsLLM = insights.NewLLMService(insights.LLMConfig{
			LLMClient: llmClient,
			DB:        d,
			Logger:    logger,
			ConvStore: insights.NewConversationStore(d),
			RespCache: insights.NewResponseCache(5*time.Minute, 1000),
		})
		logger.Info("insights LLM initialized", "provider", os.Getenv("LLM_PROVIDER"))
	}

	// Cross-Doc L3 — endpoint /v1/crossdoc/validate and /v1/generate/batch.
	crossDocEngine := crossdoc.NewEngine(crossrules.BuiltinRegistry())

	// Sprint 57 — v3.34.37: Wizard store de sessões de geração.
	wizardStore := wizard.NewStore(d)

	// Sprint 57 — v3.36.3: Generator registry com todos os 10 CADOCs.
	// O registry não pode auto-registrar dentro do pacote generator
	// (import cycle: cada gen* importa generator, e generator não pode
	// importar os gen*). O glue é feito aqui em main.go que importa ambos.
	genReg := generator.NewRegistry()
	generator.RegisterDefaults(genReg, []generator.CADOCGenerator{
		gen2030pkg.New(),
		gen2060pkg.New(),
		gen2061pkg.New(),
		gen2062pkg.New(),
		gen2070pkg.New(),
		gen2160pkg.New(),
		gen2170pkg.New(),
		gen3040pkg.New(),
		gen3050pkg.New(),
		gen4111pkg.New(),
	})
	logger.Info("generator registry inicializado", "cadocs", len(genReg.List()))

	srv := api.NewServer(d, schReg, audSvc, audLog, staClient, radarSvc, ruleprefs.NewPreferences(d), ruleprefs.NewToggleLimiter(10, time.Minute), insights.NewAcknowledgments(d), brandingSvc, insightsLLM, marketplaceSvc, pilotSvc, crossDocEngine, wizardStore)
	srv.GeneratorRegistry = genReg

	// Sprint 10 — Hub SSE + wrap audit logger pra publicar eventos em
	// real-time. Em produção, hub pode ser substituído por Kafka/Redis
	// pub/sub sem mudar a API.
	eventsHub := realtime.NewHub(logger)
	srv.EventsHub = eventsHub
	hubAwareLog := realtime.WrapAuditLog(audLog, eventsHub)
	srv.AuditLog = hubAwareLog
	logger.Info("SSE hub inicializado", "subs", 0)

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
			// Mesmo default do dev-signer (linha 140/146) — "k1".
			// Antes (Sprint 8a shipped): usava os.Getenv puro → "" quando
			// env não setada, enquanto signer usava envOr(..., "k1").
			// Resultado: tokens mintados tinham kid="k1" no header, mas
			// keyring do verifier tinha kid="" → 401 invalid token.
			// Smoke test local (2026-07-04) pegou o mismatch end-to-end.
			Kid:       envOr("RADIANT_JWT_KID", "k1"),
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

	// Sprint 13 — v3.5.2 [S13.1] FAIL-CLOSED env gates.
	//
	// Quando RADIANT_ENV=production, recusar a iniciar se alguma flag de
	// dev-mode estiver ativa. Sem isso, deploy que esquece de unsetar
	// RADIANT_DEV_TOKEN / RADIANT_DEV_AUTH resulta em endpoint de
	// emissão de JWT arbitrário (CRITICAL F-API-2 do audit S-A).
	//
	// Comportamento:
	//   RADIANT_ENV=production + RADIANT_DEV_TOKEN=1        → panic
	//   RADIANT_ENV=production + RADIANT_DEV_AUTH=1         → panic
	//   RADIANT_ENV=production + RADIANT_JWT_PUBLIC_KEY=""  → panic (até aqui era warning silencioso)
	//   RADIANT_ENV != production                            → warning apenas (back-compat)
	if isProduction := os.Getenv("RADIANT_ENV") == "production"; isProduction {
		var fatal bool
		if os.Getenv("RADIANT_DEV_TOKEN") == "1" {
			logger.Error("FATAL: RADIANT_ENV=production mas RADIANT_DEV_TOKEN=1 — dev-token emitiria JWT arbitrário sem auth")
			fatal = true
		}
		if os.Getenv("RADIANT_DEV_AUTH") == "1" {
			logger.Error("FATAL: RADIANT_ENV=production mas RADIANT_DEV_AUTH=1 — X-IF-ID fallback aceitaria qualquer tenant")
			fatal = true
		}
		if os.Getenv("RADIANT_JWT_PUBLIC_KEY") == "" {
			logger.Error("FATAL: RADIANT_ENV=production mas RADIANT_JWT_PUBLIC_KEY não configurada — /v1/* retornaria 401 silencioso")
			fatal = true
		}
		if os.Getenv("RADIANT_NORMA_ADMIN_TOKEN") == "" {
			logger.Error("FATAL: RADIANT_ENV=production mas RADIANT_NORMA_ADMIN_TOKEN não configurada — /v1/radar/scan retornaria 401 silencioso")
			fatal = true
		}
		if fatal {
			os.Exit(1)
		}
	}

	// Sprint 6 v1.5.0 — hardening components wiring.
	// ANTES (v1.5.0 shipped): esses 4 componentes ficavam nil → endpoints
	// /v1/crossdoc/validate e /v1/radar/scan retornavam 503/401.
	// Validação 12 (F12.2): fix que ativa hardening em produção.

	// W4 — cadoc list com cache 5min (sempre ativo; sem env var).
	srv.CadocListCache = schema.NewCadocListCache(5 * time.Minute)

	// Sprint 17 — v3.7.0 [S17.5]: métricas Prometheus.
	srv.Metrics = api.NewMetrics()

	// Sprint 16 — v3.6.0 [S16.1]: rate limiter plugável.
	// Default memory (single-replica). Setar RADIANT_RATE_LIMIT_BACKEND=redis
	// + RADIANT_REDIS_URL=... pra produção multi-replica.
	rateLimiter, err := api.NewRateLimiterFromEnv()
	if err != nil {
		logger.Error("rate limiter init failed", "err", loggerutil.SafeError(err))
		os.Exit(1)
	}
	srv.RateLimiter = rateLimiter
	logger.Info("rate limiter ativo", "backend", rateLimiter.Backend())
	if rl, ok := rateLimiter.(*api.RedisRateLimiter); ok {
		// Sprint 17 [S17.5]: wire metrics no Redis limiter (IncFailOpen + SetBackendUp)
		rl.Metrics = srv.Metrics
		// Close Redis no shutdown
		defer func() { _ = rl.Close() }()
	}

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

	// Sprint 8a (v2.1.0) — dev-token endpoint /v1/auth/dev-token.
	//
	// Ativado por env RADIANT_DEV_TOKEN=1. Requer chave privada RSA
	// configurada (RADIANT_DEV_JWT_PRIVATE_KEY=path ou
	// RADIANT_DEV_JWT_PRIVATE_KEY_PEM=conteúdo PEM inline).
	//
	// Em produção: dev-token endpoint fica disabled (404) E bloqueado
	// pelo fail-closed gate acima. Tokens devem ser emitidos por IdP
	// externo (Keycloak/Okta/etc) — Sprint 9+.
	if os.Getenv("RADIANT_DEV_TOKEN") == "1" {
		var signer *auth.Signer
		var err error

		if pemPath := os.Getenv("RADIANT_DEV_JWT_PRIVATE_KEY"); pemPath != "" {
			signer, err = auth.NewSignerFromFile(
				pemPath,
				envOr("RADIANT_JWT_KID", "k1"),
				envOr("RADIANT_JWT_ISSUER", "radiant-norma"),
			)
		} else if pemInline := os.Getenv("RADIANT_DEV_JWT_PRIVATE_KEY_PEM"); pemInline != "" {
			signer, err = auth.NewSigner(auth.SignerConfig{
				PrivateKeyPEM: pemInline,
				Kid:           envOr("RADIANT_JWT_KID", "k1"),
				Issuer:        envOr("RADIANT_JWT_ISSUER", "radiant-norma"),
			})
		}

		if err != nil {
			logger.Error("dev-token signer setup", "err", loggerutil.SafeError(err))
			os.Exit(1)
		}
		if signer != nil {
			srv.DevSigner = signer
			// Sprint 13: warning reforçado se prod-environment-style RADIANT_ENV
			// já passou (não chegou aqui porque gate acima matou). Mesmo assim,
			// reforçamos em warn.
			logger.Warn("RADIANT_DEV_TOKEN=1 — /v1/auth/dev-token ATIVO. NÃO USE EM PRODUÇÃO.", "RADIANT_ENV", os.Getenv("RADIANT_ENV"))
		}
	}

	handler := srv.Router()

	// Sprint 36 — OTel + Sentry middleware como outermost layer.
	// Cria span por request + propaga W3C trace context + reporta panics.
	handler = observability.SentryMiddleware()(handler)

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
