// Package api implementa os handlers HTTP da API REST do Radiant Norma.
package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/branding"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/generator/wizard"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/marketplace"
	"github.com/fortvna/radiant-norma/backend/internal/pilot"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/realtime"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/version"
	"github.com/fortvna/radiant-norma/backend/internal/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Version é a versão da API reportada no /healthz e em metadata de log.
//
// Validação 9 (v1.4.3): single source of truth dentro do package.
// Validação 18 (v1.5.0): propagou para package compartilhado
// `internal/version` (GAP-7.4 / F10.10). Esta constante é mantida
// como re-export para callers existentes (`api.Version` permanece).
//
// Single source of truth: `internal/version/version.go`.
//
// Bump de versão: alterar `internal/version/version.go::Version` +
// CHANGELOG.md + tag git. NÃO tocar este arquivo (re-exporta
// version.Version automaticamente).
const Version = version.Version

// Server agrega todos os serviços.
type Server struct {
	DB            *sql.DB
	Schema        *schema.Registry
	Audit         *audit.Service
	AuditLog      auditLogAPI
	STAClient     sta.Client
	Radar         *radar.Service
	CrossDoc      *crossdoc.Engine          // Sprint 6 v1.5.0 — Cross-Doc L3
	RulePrefs     *ruleprefs.Preferences    // Sprint 11 v3.4.0 — disable/enable por IF
	ToggleLimiter *ruleprefs.ToggleLimiter  // Sprint 12 v3.5.0 — C32.22 rate limit toggle
	Insights      *insights.Acknowledgments // Sprint 12 v3.5.0 — recommendation ack

	// Sprint 7a (v1.6.0): JWT verifier. Se nil, X-IF-ID fallback
	// (dev mode via RADIANT_DEV_AUTH=1) ainda funciona.
	Auth *auth.Verifier

	// Sprint 8a (v2.1.0): dev-token signer. Se setado, /v1/auth/dev-token
	// emite JWT in-process (requer RADIANT_DEV_TOKEN=1 para ativar).
	// Em prod, IdP externo emite tokens — DevSigner fica nil.
	DevSigner *auth.Signer

	startedAt time.Time

	// Sprint 6 v1.5.0 (R1) — DOS-via-API prevention.
	// TriggerRadarScan agora exige admin role + tem rate limit + cache.
	ScanLimiter *radar.ScanLimiter
	ScanCache   *radar.ScanCache
	AdminAuth   *radar.AdminAuth

	// Sprint 6 v1.5.0 (W4) — cadoc list cache.
	// listSchemas / listRules consultam DB mas com cache 5min.
	CadocListCache *schema.CadocListCache

	// Sprint 75: schema info cache (listSchemasV2).
	SchemaInfoCache *SchemaInfoCache

	// Sprint 10 — SSE hub para real-time push (alertas/audit/envios).
	// Se nil, /v1/events/stream retorna 503.
	EventsHub *realtime.Hub

	// Sprint 13 — v3.5.2 [S15.1] rate limiter global.
	RateLimiter RateLimiter

	// Sprint 17 — v3.7.0 [S17.5]: métricas Prometheus.
	// Wire via Server.Metrics = api.NewMetrics() antes de Router().
	// Endpoint /metrics (top-level, sem auth) consome via Render().
	Metrics *Metrics

	// Sprint 46 — v3.34.27: WhiteLabel branding por tenant.
	Branding *branding.BrandingService

	// Sprint 53 — v3.34.35: AI Insights via LLM (opt-in).
	InsightsLLM *insights.LLMService

	// Sprint 61 — v3.34.43: Webhooks outbound.
	Webhook *webhook.Service

	// Sprint 62 — v3.34.44: Marketplace de regras customizadas.
	Marketplace *marketplace.Service

	// Sprint 55 — Pilot3 ESG-first.
	Pilot *pilot.Service

	// Sprint 57 — v3.34.37: Wizard de geração de CADOCs.
	WizardStore *wizard.Store
}

// auditLogAPI é interface mínima que *auditlog.Logger e *realtime.HubAwareLogger
// ambos satisfazem. Permite wrap sem mudar assinatura do Server.
type auditLogAPI interface {
	Log(ifID, actor, action, target string, payload []byte, metadata any) (*auditlog.Entry, error)
	Verify() (bool, int, error)
}

// NewServer cria um Server.
func NewServer(d *sql.DB, sch *schema.Registry, aud *audit.Service, al auditLogAPI, staClient sta.Client, rad *radar.Service, rp *ruleprefs.Preferences, tl *ruleprefs.ToggleLimiter, ack *insights.Acknowledgments, br *branding.BrandingService, insightsLLM *insights.LLMService, mp *marketplace.Service, pilotSvc *pilot.Service, crossDoc *crossdoc.Engine, wizardStore *wizard.Store) *Server {
	return &Server{
		DB: d, Schema: sch, Audit: aud, AuditLog: al, STAClient: staClient,
		Radar: rad, RulePrefs: rp, ToggleLimiter: tl, Insights: ack,
		Branding:    br,
		InsightsLLM: insightsLLM,
		Marketplace: mp,
		Pilot:       pilotSvc,
		CrossDoc:    crossDoc,
		WizardStore: wizardStore,
		startedAt:   time.Now(),
		RateLimiter: newMemoryRateLimiter(),
	}
}

// Router retorna o chi router configurado.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware padrão
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// Sprint 12 (v3.5.0) — C32.21: CSRF protection.
	// Whitelist: /api/login (dev), /v1/auth/* (dev-token).
	// Em dev: warning + allow. Em prod (RADIANT_ENV=production): 403.
	r.Use(CSRF(DefaultCSRFConfig()))

	// Sprint 13 — v3.5.2 [S15.1]: rate limit global por (bucket, IFID).
	// Mitiga DoS-via-API authenticated. Aplicado DEPOIS de CSRF pra
	// que rate limit não conte requests bloqueadas.
	//
	// Sprint 17 — v3.7.0 [S17.5]: passa Metrics para middleware incrementar
	// counters allowed/dropped por bucket+backend.
	r.Use(rateLimitMiddleware(s.RateLimiter, s.Metrics))

	// Sprint 6 v1.5.0 (F12.8 fix): Recoverer ANTES de Logger para que
	// panics que viram 500 sejam loggados (não engolidos pelo Logger
	// que está acima na pilha).
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// Sprint 36 — v3.36.0: record request duration + count per endpoint.
	// Applied after Logger so context (IFID, trace_id) is available.
	r.Use(observabilityMiddleware(s.Metrics))

	// Health
	r.Get("/healthz", s.healthz)

	// Sprint 17 — v3.7.0 [S17.5]: Prometheus /metrics endpoint.
	// Top-level (não /v1/metrics) pra seguir convenção k8s/Prometheus.
	// Sem auth: scraper tipicamente roda na mesma rede privada.
	// Bypass rate limit + CSRF (whitelist via middleware skip acima).
	if s.Metrics != nil {
		r.Get("/metrics", s.metricsHandler)
	}
	r.Get("/readyz", s.readyz) // Validação 23 (F23.1): separado de /healthz

	// API v1
	r.Route("/v1", func(r chi.Router) {
		// Validação 20 (F20.3): MaxBytesReader middleware — limita payload
		// de entrada a 10MB. Sem isso, atacante autenticado (X-IF-ID válido)
		// pode enviar body gigante (1GB) e causar OOM via io.ReadAll.
		//
		// 10MB é suficiente para XMLs de CADOC (geralmente 50KB-5MB),
		// com folga para casos extremos (3040 com 50k linhas).
		r.Use(maxBodyBytesMiddleware(10 << 20)) // 10 MiB

		// Sprint 7a (v1.6.0): JWT middleware substitui X-IF-ID.
		// Fallback para X-IF-ID apenas em dev mode.
		r.Use(auth.Middleware(s.Auth))

		// Schemas
		r.Get("/schemas", s.listSchemas)
		r.Get("/schemas/{cadoc}", s.getSchema)
		r.Get("/schemas/{cadoc}/versions", s.listVersions)
		// Sprint 54 v3.34.37: specific version by ID.
		r.Get("/schemas/{cadoc}/versions/{versionId}", s.getSchemaVersion)
		// Sprint 54 v3.34.37: public changelog timeline.
		r.Get("/schemas/{cadoc}/changelog", s.listSchemaChangelog)
		// Sprint 73: schema listing with generation metadata.
		r.Get("/schema", s.listSchemasV2)

		// Rules (críticas)
		r.Get("/rules", s.listRules)
		r.Get("/rules/{cadoc}", s.listRulesByCadoc)

		// Sprint 11 (v3.4.0) — drill-down: enable/disable regras por IF.
		// Persistência no backend (era localStorage no frontend).
		r.Get("/rules/disabled", s.listDisabledRules)
		r.Post("/rules/{code}/toggle", s.toggleRule)

		// Validation
		r.Post("/validate", s.validate)

		// Geração de CADOCs (Sprint 57).
		r.Post("/generate/{cadoc}", s.generateCadoc)
		r.Get("/generate/{cadoc}/fields", s.listGenerateFields)
		r.Post("/generate/{cadoc}/sources", s.ingestSources)
		r.Get("/generate/adapters", s.listSourceAdapters)
		// Sprint 60: Wizard UI — parse uploaded CSV/XLSX to CanonicalDocument.
		r.Post("/generate/file/parse", s.parseUploadedFile)
		// Sprint 64: Batch generation with optional cross-doc validation.
		r.Post("/generate/batch", s.generateBatch)
		// Sprint 73: generation history.
		r.Get("/generate/history", s.listGenerateHistory)

		// Sprint 57 — v3.34.37: Wizard de geração.
		r.Post("/generate/wizard/start", s.startWizard)
		r.Get("/generate/wizard/{id}", s.getWizard)
		r.Put("/generate/wizard/{id}", s.advanceWizard)
		r.Get("/generate/wizard/{id}/xml", s.getWizardXML)

		// Cross-Doc L3 (Sprint 6 v1.5.0) — diferencial proprietário.
		r.Post("/crossdoc/validate", s.crossdocValidate)
		// Sprint 73: list all cross-doc rules.
		r.Get("/crossdoc/rules", s.listCrossDocRules)

		// STA submission (stub)
		r.Post("/sta/submit", s.staSubmit)

		// Sprint 20 (v3.10.0): read side REST endpoints. Requer
		// RADIANT_STA_BACKEND=ws; retorna 503 se backend=stub.
		r.Get("/sta/disponiveis", s.staDisponiveisHandler)
		r.Post("/sta/situacao", s.staSituacaoHandler)

		// Sprint 31 v3.34.31: RangeUploadAPI — chunked upload via STA §5.6.
		r.Post("/sta/range/init", s.staRangeInit)
		r.Put("/sta/range/{protocolo}", s.staRangeUpload)
		r.Get("/sta/range/{protocolo}", s.staRangeStatus)

		// Radar regulatório (Sprint 4)
		r.Get("/radar/alerts", s.listRadarAlerts)
		r.Get("/radar/alerts/{id}", s.getRadarAlert)
		r.Post("/radar/alerts/{id}/resolve", s.resolveRadarAlert)
		r.Post("/radar/scan", s.triggerRadarScan)

		// L4 Histórico (Sprint 55) — diff vs envio anterior.
		r.Get("/l4/compare", s.l4Compare)

		// Sprint 8c (v3.1.0) — endpoints de inteligência que destravam
		// o frontend v3.0.0 (estava em empty states por falta de dados).
		r.Get("/envios", s.listEnvios)
		r.Get("/envios/stats", s.enviosStats)
		r.Get("/audit_log", s.listAuditLog)
		r.Route("/insights", func(r chi.Router) {
			r.Get("/kpis", s.insightsKPIs)
			r.Get("/heatmap", s.insightsHeatmap)
			r.Get("/rules/top-failing", s.insightsTopFailingRules)
			r.Get("/recommendations", s.insightsRecommendations)
			// Sprint 12 (v3.5.0) — drill-down de recommendation.
			r.Post("/recommendations/{id}/acknowledge", s.acknowledgeRecommendation)
			r.Delete("/recommendations/{id}/acknowledge", s.unacknowledgeRecommendation)
			// Sprint 53 (v3.34.35) — AI Insights via LLM.
			r.Post("/ask", s.AskLLM)
			r.Get("/ask/stream", s.streamAskLLM) // SSE streaming
			r.Get("/history", s.getInsightsHistory)
			r.Delete("/history", s.deleteInsightsHistory)
		})

		// Sprint 10 — SSE real-time stream.
		// Auth vem do middleware JWT global (acima).
		// Stream filtra por IF automaticamente (atrás de IF=auth).
		r.Get("/events/stream", s.eventsStreamHandler)

		// Sprint 46 (v3.34.27): WhiteLabel branding.
		// GET /v1/tenant/branding: branding do tenant autenticado.
		// PUT /v1/tenant/branding: atualiza branding do tenant autenticado.
		// GET /v1/tenant/branding/public/{slug}: público por tenant_slug.
		// PUT /v1/admin/tenant/{id}/branding: admin atualiza branding de qualquer tenant.
		r.Route("/tenant/branding", func(r chi.Router) {
			r.Get("/", s.getBranding)
			r.Put("/", s.updateBranding)
			r.Get("/public/{slug}", s.getBrandingBySlug)
		})
		r.Put("/admin/tenant/{id}/branding", s.adminUpdateBranding)

		// Sprint 61 v3.34.43: Webhooks outbound.
		r.Route("/webhooks", func(r chi.Router) {
			r.Get("/", s.listWebhooks)
			r.Post("/", s.registerWebhook)
			r.Delete("/{id}", s.deleteWebhook)
			r.Get("/{id}/deliveries", s.listDeliveries)
			r.Get("/{id}/deliveries/{delivery_id}", s.getDelivery)
			r.Post("/{id}/deliveries/{delivery_id}/retry", s.retryDelivery)
		})

		// Sprint 62 v3.34.44: Marketplace de regras customizadas.
		r.Route("/marketplace", func(r chi.Router) {
			r.Get("/", s.listMarketplaceRules)
			r.Post("/", s.publishRule)
			r.Post("/{id}/install", s.installRule)
			r.Post("/{id}/rate", s.rateRule)
			r.Get("/installed", s.listInstalledRules)
		})

		// Sprint 55 — v3.34.49: Pilot3 ESG-first.
		r.Route("/pilot", func(r chi.Router) {
			r.Get("/programs", s.listPilotPrograms)
			r.Post("/programs", s.createPilotProgram)
			r.Get("/programs/{programId}/participants", s.listPilotParticipants)
			r.Post("/programs/{programId}/enroll", s.enrollPilotParticipant)
			r.Get("/participants/{ifID}/steps", s.getPilotSteps)
			r.Post("/participants/{ifID}/steps/{stepKey}/complete", s.completePilotStep)
			r.Get("/participants/{ifID}/esg/progress", s.getESGonboardingProgress)
			r.Post("/participants/{ifID}/esg/enroll", s.enrollESGParticipant)
		})

		// Sprint 56 v3.34.38: SOC 2 Type I readiness + evidence.
		r.Route("/admin/soc2", func(r chi.Router) {
			r.Get("/readiness", s.getSOC2Readiness)
			r.Get("/controls", s.listSOC2Controls)
			r.Get("/controls/{id}/evidence", s.getSOC2ControlEvidence)
		})

		// Sprint 54 v3.34.37: Admin schema management (insert new version).
		r.Post("/admin/schemas", s.publishSchema)
	})

	// Sprint 8a (v2.1.0): dev-token endpoint (FRENTE do middleware JWT).
	// Não exige auth (é o que GERA auth tokens para o cliente). Defense:
	// retorna 404 se RADIANT_DEV_TOKEN != "1" — esconde endpoint em prod.
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/dev-token", s.devTokenHandler)
	})

	return r
}

// --- Middleware ---

// maxBodyBytesMiddleware limita o tamanho do body de leitura
// para evitar DOS-via-large-body (F20.3).
//
// Validação 20: r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
// wraps the body, returning an error on Read when maxBytes é excedido.
// Sem isso, atacante autenticado pode enviar body 1GB e causar OOM.
//
// maxBytes: tamanho máximo em bytes (ex: 10 MiB = 10<<20).
//
// Em caso de overflow, o handler vai receber um erro que não é nil;
// via json.Decode/sql/etc vai propagar para UserError com 400/413.
func maxBodyBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength > maxBytes {
				// Rejeitar ANTES de ler — rápido, sem alocar.
				w.Header().Set("Connection", "close")
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodyBytesMiddlewareForTest expõe o middleware para tests externos.
// Misma implementação, apenas renomeada para evitar nome uppercase
// chochar com chi.Router.
func MaxBodyBytesMiddlewareForTest(maxBytes int64, next http.Handler) http.Handler {
	return maxBodyBytesMiddleware(maxBytes)(next)
}

// observabilityMiddleware records request duration and count for Prometheus.
// Sprint 36 — v3.36.0: enriched metrics for all endpoints.
func observabilityMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if metrics == nil {
				next.ServeHTTP(w, r)
				return
			}
			// Skip noisy paths that distort latency metrics.
			path := r.URL.Path
			if path == "/healthz" || path == "/readyz" || path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rw := &statusCapturer{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			durationMs := time.Since(start).Milliseconds()
			endpoint := normalizeEndpoint(path)
			metrics.ObserveRequest(endpoint, r.Method, rw.statusCode, durationMs)
		})
	}
}

// statusCapturer wraps http.ResponseWriter to capture the status code.
type statusCapturer struct {
	http.ResponseWriter
	statusCode int
}

func (sc *statusCapturer) WriteHeader(code int) {
	sc.statusCode = code
	sc.ResponseWriter.WriteHeader(code)
}

// normalizeEndpoint reduces variable path segments to wildcards for
// stable metric labels (e.g., /v1/envios/abc123 → /v1/envios/{id}).
// This prevents Prometheus high-cardinality label explosion.
func normalizeEndpoint(path string) string {
	// Cadoc IDs: alphanumeric IDs (uploaded file keys, envio IDs).
	// e.g. /v1/envios/abc123 → /v1/envios/{id}
	if clean := collapseRe(path, `/envios/[^/]+`, `/envios/{id}`); clean != path {
		return clean
	}
	// Schema versions: numeric IDs.
	// e.g. /v1/schemas/3040/versions/5 → /v1/schemas/{cadoc}/versions/{id}
	if clean := collapseRe(path, `/versions/[0-9]+`, `/versions/{id}`); clean != path {
		return clean
	}
	// Wizard sessions: alphanumeric session IDs.
	// e.g. /v1/generate/wizard/abc123xyz → /v1/generate/wizard/{sessionId}
	if clean := collapseRe(path, `/wizard/[^/]+`, `/wizard/{sessionId}`); clean != path {
		return clean
	}
	// Radar scan IDs: alphanumeric.
	if clean := collapseRe(path, `/scan/[^/]+`, `/scan/{scanId}`); clean != path {
		return clean
	}
	// Generic numeric ID: catches most remaining patterns.
	// Apply last to avoid double-collapsing already-normalized segments.
	if clean := collapseRe(path, `/[0-9]+`, `/{id}`); clean != path {
		return clean
	}
	return path
}

// collapseRe applies a single regex replacement, returning the original
// string if no match was made (avoids allocation in the common miss case).
func collapseRe(path, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	idx := re.FindStringIndex(path)
	if idx == nil {
		return path
	}
	return re.ReplaceAllString(path, replacement)
}

// --- Handlers ---

// UserError registra o erro nos logs e retorna um código HTTP com mensagem
// genérica. Não vaza err.Error() no body da response (vetor de information
// disclosure: SQL fragments, JSON offsets, table names, etc.).
//
// Validação 18 (F18.1): estende F15.3 para TODAS as respostas — antes
// havia `http.Error(w, err.Error(), 4xx)` que vazava metadata. Agora:
//
//   - Log do err com SafeError (camada logger)
//   - Response: <label do erro> (genérico) com status apropriado
//   - Caller não vê SQL/JSON/SQL driver detalhes
//
// Use para 4xx E 5xx. Para erros realmente corruptos (200 — warning
// payload), use writeJSON diretamente com metadata explícita.
//
// Exportada para testes externos (ver internal/api/user_error_test.go).
func (s *Server) UserError(w http.ResponseWriter, status int, ctx string, err error) {
	logger := slog.Default()
	logger.Error("server error",
		"context", ctx,
		"status", status,
		"err", loggerutil.SafeError(err))
	// Mensagem pública compacta, não inclui err.Error() cru.
	publicMsg := "erro"
	switch status {
	case http.StatusBadRequest:
		publicMsg = "requisição inválida"
	case http.StatusUnauthorized:
		publicMsg = "não autorizado"
	case http.StatusForbidden:
		publicMsg = "forbidden"
	case http.StatusNotFound:
		publicMsg = "não encontrado"
	case http.StatusConflict:
		publicMsg = "conflito"
	case http.StatusUnprocessableEntity:
		publicMsg = "entidade não processável"
	case http.StatusTooManyRequests:
		publicMsg = "rate limit excedido"
	case http.StatusInternalServerError:
		publicMsg = "erro interno (ver logs)"
	case http.StatusServiceUnavailable:
		publicMsg = "serviço indisponível"
	}
	http.Error(w, publicMsg, status)
}

// userError é o alias interno para UserError, mantido para código
// pré-validação 18 que ainda chama s.userError(...).
// Deprecated: use UserError.
func (s *Server) userError(w http.ResponseWriter, status int, ctx string, err error) {
	s.UserError(w, status, ctx, err)
}

// internalServerError retorna 500 com mensagem genérica + loga erro
// sanitizado internamente.
//
// Validação 15 (F15.3): evita information disclosure via err.Error()
// na response HTTP. err.Error() pode incluir SQL fragments, table
// names, ou em casos extremos (pgx): user+database da DSN.
//
// Resposta: "erro interno (correlation: <trace>)" — caller precisa de
// logs correlacionados pra debug. Não vaza internals.
//
// DEPRECATED em validação 18: prefira UserError(w, 500, ctx, err) que
// é a forma unificada.
func (s *Server) internalServerError(w http.ResponseWriter, err error, ctx string) {
	s.UserError(w, http.StatusInternalServerError, ctx, err)
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	// Liveness check — sempre OK enquanto processo está vivo.
	// Não checa dependências externas (DB, network). Sem isso, restart
	// loop em K8s se DB falhar temporariamente.
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"time":           time.Now().UTC().Format(time.RFC3339),
		"version":        Version,
		"uptime_seconds": int(time.Since(s.startedAt).Seconds()),
	})
}

// metricsHandler — Sprint 17 — v3.7.0 [S17.5]: Prometheus exposition format.
//
// Content-Type: text/plain; version=0.0.4 (formato oficial Prometheus).
// Scraper (Prometheus server, Grafana Agent, etc) parseia esse endpoint
// a cada scrape_interval (default 15s).
//
// Bypass auth + rate limit via whitelist em rateLimitMiddleware.
func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = s.Metrics.WriteTo(w)
}

// readyz — Validação 23 (F23.1): separado de healthz.
//
// Readiness check — verifica dependências críticas (DB). Usado por K8s
// readiness probe para tirar pod do load balancer se DB estiver
// indisponível. Sem isso, requests entram em pod zumbi.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil {
		http.Error(w, "db not configured", http.StatusServiceUnavailable)
		return
	}
	// Ping com context do request (cancela se cliente desconecta).
	if err := s.DB.PingContext(r.Context()); err != nil {
		http.Error(w, "db unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ready",
		"db":      "ok",
		"version": Version,
	})
}

func (s *Server) listSchemas(w http.ResponseWriter, r *http.Request) {
	cadocs, err := s.cadocsWithCache(r.Context())
	if err != nil {
		s.internalServerError(w, err, "listSchemas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cadocs": cadocs,
		"total":  len(cadocs),
	})
}

// cadocsWithCache retorna lista de CADOCs via DB + cache 5min.
func (s *Server) cadocsWithCache(ctx context.Context) ([]string, error) {
	if s.Schema == nil {
		return []string{}, nil
	}
	if s.CadocListCache == nil {
		return s.Schema.ListCadocs(ctx)
	}
	return s.CadocListCache.GetOrFetch(func() ([]string, error) {
		return s.Schema.ListCadocs(ctx)
	})
}

func (s *Server) getSchema(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	if cadoc == "" {
		http.Error(w, "cadoc required", http.StatusBadRequest)
		return
	}
	v, err := s.Schema.GetEffective(cadoc, time.Now())
	if err != nil {
		// Validação 18 (F18.1): err.Error() pode vazar SQL fragments. Use
		// userError para sanitizar a resposta.
		s.userError(w, http.StatusNotFound, "getSchema", err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	// Sprint 13 — v3.5.2 [S15.3]: format validation.
	if err := ValidateCadocCode(cadoc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	versions, err := s.Schema.List(cadoc)
	if err != nil {
		s.internalServerError(w, err, "listVersions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cadoc":    cadoc,
		"versions": versions,
		"total":    len(versions),
	})
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	cadocs, err := s.cadocsWithCache(r.Context())
	if err != nil {
		s.internalServerError(w, err, "listRules")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cadocs": cadocs,
	})
}

func (s *Server) listRulesByCadoc(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	// Sprint 13 — v3.5.2 [S15.3]: format validation.
	if err := ValidateCadocCode(cadoc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	criticas, err := s.Audit.LoadCriticas(r.Context(), cadoc)
	if err != nil {
		s.internalServerError(w, err, "listRulesByCadoc")
		return
	}

	// Filtra por ?enabled=true|false|all (case-insensitive, default: all).
	// Normaliza para lowercase antes do switch — aceita TRUE, True, true.
	enabledFilter := strings.ToLower(r.URL.Query().Get("enabled"))
	filtered := make([]audit.Critica, 0, len(criticas))
	for _, c := range criticas {
		switch enabledFilter {
		case "true":
			if c.Enabled {
				filtered = append(filtered, c)
			}
		case "false":
			if !c.Enabled {
				filtered = append(filtered, c)
			}
		default:
			// "all" ou "" → todas
			filtered = append(filtered, c)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"cadoc":     cadoc,
		"rules":     filtered,
		"total":     len(filtered),
		"total_all": len(criticas),
	})
}

func (s *Server) validate(w http.ResponseWriter, r *http.Request) {
	// Validação 27 (F27.1): getIfID prioriza auth.Claims (JWT) > header X-IF-ID.
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Lê body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Validação 18 (F18.1): err.Error() não vaza. Use userError.
		s.userError(w, http.StatusBadRequest, "validate.readBody", err)
		return
	}

	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Validação 18 (F18.1): err.Error() do json.Unmarshal pode incluir
		// offsets e field names — vetor de information disclosure.
		s.userError(w, http.StatusBadRequest, "validate.jsonUnmarshal", err)
		return
	}

	if req.CadocCode == "" {
		http.Error(w, "cadoc (or cadoc_code) required", http.StatusBadRequest)
		return
	}
	// Sprint 13 — v3.5.2 [S15.3]: cadoc_code format validation.
	// Sem isso, validate aceitar "DROP TABLE x" e bypass pra vários
	// paths (audit log, queries, etc). Pattern BACEN: [0-9]{4}.
	if err := ValidateCadocCode(req.CadocCode); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.XML) == 0 {
		http.Error(w, "xml required", http.StatusBadRequest)
		return
	}
	if req.ContentType == "" {
		req.ContentType = "application/xml"
	}

	// Sprint 12 (v3.5.0): C32.23 — popula IfID a partir do JWT claims
	// pra que audit.Service possa filtrar regras desabilitadas por IF
	// (toggle em /v1/rules/{code}/toggle).
	req.IfID = ifID

	// Executa validação
	resp, err := s.Audit.Validate(r.Context(), &req)
	if err != nil {
		s.internalServerError(w, err, "validate")
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "cadoc.validated", req.CadocCode, body, map[string]any{
		"passed":      resp.Passed,
		"errors":      len(resp.Errors),
		"warnings":    len(resp.Warnings),
		"duration_ms": resp.DurationMs,
	})

	// Sprint 61 v3.34.43: dispara webhook validation.completed.
	// Fire-and-forget — não bloqueia resposta nem afecta a validação.
	if s.Webhook != nil {
		DispatchValidationCompleted(s.Webhook, ifID, req.CadocCode, req.DataBase, resp.XMLHash,
			resp.Passed, len(resp.Errors), len(resp.Warnings))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) staSubmit(w http.ResponseWriter, r *http.Request) {
	// Validação 27 (F27.1): getIfID prioriza auth.Claims (JWT) > header X-IF-ID.
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	// Aceita body JSON (preferencial, contrato documentado) OU query params (retrocompat).
	var sub sta.Submission
	body, _ := io.ReadAll(r.Body)

	if r.Header.Get("Content-Type") == "application/json" && len(body) > 0 {
		if err := json.Unmarshal(body, &sub); err != nil {
			// Validação 18 (F18.1): err.Error() não vaza JSON parser detail.
			s.userError(w, http.StatusBadRequest, "staSubmit.jsonUnmarshal", err)
			return
		}
	} else {
		// Fallback retrocompat: query params
		sub.CadocCode = r.URL.Query().Get("cadoc")
		if sub.CadocCode == "" {
			sub.CadocCode = r.URL.Query().Get("cadoc_code")
		}
		sub.DataBase = r.URL.Query().Get("data_base")
	}

	// Fallback do CNPJ é o X-IF-ID (multi-tenant identifier)
	if sub.CNPJ == "" {
		sub.CNPJ = ifID
	}

	// Sprint 13 — v3.5.2 [S13.2 / C-API-4]: enforce tenant match.
	// Se cliente mandou CNPJ explícito diferente do tenant autenticado,
	// rejeitar antes de chegar no STA. Sem isso, IF_A poderia submeter
	// XML com CNPJ de IF_B (audit log cruzado, poluição).
	if !s.enforceSameIF(w, r, sub.CNPJ) {
		return
	}

	if sub.CadocCode == "" {
		http.Error(w, "cadoc_code required (in body JSON or ?cadoc= query param)", http.StatusBadRequest)
		return
	}

	// XML/ZIP: se não vier no body JSON, usa o body cru (XML direto)
	if sub.XML == "" {
		sub.XML = string(body)
	}
	// ZIP: stub usa o XML puro (não o body, que pode ser JSON inteiro).
	// Em produção: ZIP real viria do campo `zip` do JSON ou seria gerado aqui.
	if len(sub.Zip) == 0 {
		sub.Zip = []byte(sub.XML)
	}

	result, err := s.STAClient.Submit(r.Context(), &sub)
	if err != nil {
		s.internalServerError(w, err, "staSubmit")
		return
	}

	// Persiste envio no DB (para o cmd/worker reenviar em caso de falha)
	envioID := generateEnvioID()
	xmlHash := sha256.Sum256([]byte(sub.XML))
	zipHash := sha256.Sum256(sub.Zip)
	if len(sub.Zip) == 0 {
		zipHash = xmlHash
	}
	_, dbErr := s.DB.ExecContext(r.Context(), `
		INSERT INTO envios (id, if_id, cadoc_code, data_base, remessa, xml_hash, zip_hash,
		                    xml_content, zip_content, status, protocol_sta, sent_at, confirmed_at)
		VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, envioID, ifID, sub.CadocCode, sub.DataBase,
		hex.EncodeToString(xmlHash[:]), hex.EncodeToString(zipHash[:]),
		sub.XML, sub.Zip,
		statusFromResult(result), result.ProtocolSTA)
	if dbErr != nil {
		// Falha em persistir não bloqueia a response (STA já aceitou), mas avisamos
		// Validação 18 (F18.14): sanitizar err.Error() antes do AuditLog.
		// AuditLog persiste em disco — vetor de disclosure persistente se raw.
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.submit.persist_failed",
			sub.CadocCode, body, map[string]any{"err": loggerutil.SafeError(dbErr)})
		writeJSON(w, http.StatusOK, map[string]any{
			"protocol_sta": result.ProtocolSTA,
			"accepted":     result.Accepted,
			"rejection":    result.Rejection,
			"envio_id":     envioID,
			"warning":      "envio não persistido (ver logs)",
		})
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.submit", sub.CadocCode, body, map[string]any{
		"envio_id": envioID,
		"protocol": result.ProtocolSTA,
		"accepted": result.Accepted,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"protocol_sta": result.ProtocolSTA,
		"accepted":     result.Accepted,
		"rejection":    result.Rejection,
		"envio_id":     envioID,
	})
}

// generateEnvioID gera UUID v4-like para um envio.
// Não usa crypto/rand por simplicidade — em produção usar uuid.New().
func generateEnvioID() string {
	now := time.Now().UnixNano()
	return "env-" + hex.EncodeToString([]byte{
		byte(now >> 56), byte(now >> 48), byte(now >> 40), byte(now >> 32),
		byte(now >> 24), byte(now >> 16), byte(now >> 8), byte(now),
	})
}

func statusFromResult(r *sta.Result) string {
	if r.Accepted {
		return "accepted"
	}
	return "rejected"
}

// --- Radar handlers ---

func (s *Server) listRadarAlerts(w http.ResponseWriter, r *http.Request) {
	if s.Radar == nil {
		http.Error(w, "radar não inicializado", http.StatusServiceUnavailable)
		return
	}
	unresolvedOnly := r.URL.Query().Get("unresolved") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	alerts, err := s.Radar.ListAlerts(r.Context(), unresolvedOnly, limit)
	if err != nil {
		s.internalServerError(w, err, "listRadarAlerts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

func (s *Server) getRadarAlert(w http.ResponseWriter, r *http.Request) {
	if s.Radar == nil {
		http.Error(w, "radar não inicializado", http.StatusServiceUnavailable)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	a, err := s.Radar.GetAlertByID(r.Context(), id)
	if err != nil {
		s.internalServerError(w, err, "getRadarAlert")
		return
	}
	if a == nil {
		http.Error(w, "alert não encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) resolveRadarAlert(w http.ResponseWriter, r *http.Request) {
	// Validação 27 (F27.1): Claims (JWT) > X-IF-ID header.
	ifID := getIfID(r)
	if s.Radar == nil {
		http.Error(w, "radar não inicializado", http.StatusServiceUnavailable)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	if err := s.Radar.ResolveAlert(r.Context(), id); err != nil {
		// Validação 18 (F18.1): err.Error() de UPDATE pode vazar SQL fragments.
		s.userError(w, http.StatusNotFound, "resolveRadarAlert", err)
		return
	}

	// Audit log (Sprint 5 v1.4.1: era gap — mutações de Radar não emitiam).
	// Mutações de Radar precisam ser auditadas pra SOC 2 / LGPD.
	//
	// NOTA Sprint 13 [S13.2]: radar_alerts é tabela global (regulatory
	// changes — não há if_id em radar_alerts). Cross-tenant não se aplica —
	// qualquer IF pode resolver alertas. Audit log usa ifID do claims
	// para forensic trail.
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "radar.alert.resolved", "radar",
		[]byte(idStr), map[string]any{"alert_id": id})

	writeJSON(w, http.StatusOK, map[string]any{"resolved": true, "id": id})
}

func (s *Server) triggerRadarScan(w http.ResponseWriter, r *http.Request) {
	// Validação 27 (F27.1): Claims (JWT) > X-IF-ID header.
	ifID := getIfID(r)
	if s.Radar == nil {
		http.Error(w, "radar não inicializado", http.StatusServiceUnavailable)
		return
	}

	// R1 — Auth admin (Sprint 6 v1.5.0): bloqueia vetor de DOS-via-API.
	// Sem ADMIN_TOKEN configurada, retorna 401 (fail closed).
	//
	// Sprint 6 v1.5.0 (F12.19 fix): defesa em profundidade — se AdminAuth
	// é nil (misconfiguração), response 503 ao invés de nil deref panic.
	if s.AdminAuth == nil {
		logger := slog.Default()
		logger.Error("AdminAuth não inicializado — Server mal configurado")
		http.Error(w, "admin auth não configurado (Server misconfigured)", http.StatusServiceUnavailable)
		return
	}
	if !s.AdminAuth.IsAdmin(r) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, s.AdminAuth.Challenge(), http.StatusUnauthorized)
		return
	}

	// R1 — Cache check (Sprint 6 v1.5.0): se tem resultado < 5min, retorna cached.
	// Justificativa: evita refazer 3 HTTP requests pra BACEN em chamadas próximas.
	if cached, ok := s.ScanCache.Get(); ok {
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "radar.scan.cached", "radar",
			nil, map[string]any{"cached_count": len(cached)})
		writeJSON(w, http.StatusOK, map[string]any{
			"new_alerts": cached,
			"count":      len(cached),
			"cached":     true,
			"cached_at":  s.ScanCache.ScannedAt(),
		})
		return
	}

	// R1 — Rate limit (Sprint 6 v1.5.0): 1 scan/min por IF.
	// Ataque: atacante autenticado hammerar → DOS contra BACEN.
	if s.ScanLimiter != nil {
		allowed, retryAfter := s.ScanLimiter.Allow(ifID)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			http.Error(w,
				fmt.Sprintf(`{"error":"rate limit exceeded","retry_after_seconds":%d}`,
					int(retryAfter.Seconds())),
				http.StatusTooManyRequests)
			return
		}
	}

	// Real scan
	alerts, err := s.Radar.ScanOnce(r.Context(), nil)
	if err != nil {
		s.internalServerError(w, err, "triggerRadarScan")
		return
	}

	// Audit log (Sprint 5 v1.4.1: era gap — scan manual é uma mutação que
	// dispara HTTP requests pra BACEN e persiste alerts novos).
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "radar.scan.triggered", "radar",
		nil, map[string]any{"new_alerts": len(alerts)})

	// Cache para próximas chamadas (5min TTL).
	if s.ScanCache != nil {
		s.ScanCache.Put(alerts)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"new_alerts": alerts,
		"count":      len(alerts),
		"cached":     false,
	})
}

// --- Cross-Doc handlers ---

// crossdocValidate recebe múltiplos CADOCs e executa regras cross-document.
//
// Payload:
//
//	{
//	  "cadocs": {
//	    "3040": "<xml>...</xml>",
//	    "4111": "<xml>...</xml>",
//	    "2030": "<xml>...</xml>"
//	  }
//	}
//
// Resposta: ValidationResponse com passed, errors[], warnings[], rules_run[].
func (s *Server) crossdocValidate(w http.ResponseWriter, r *http.Request) {
	// Validação 27 (F27.1): Claims (JWT) > X-IF-ID header.
	ifID := getIfID(r)

	if s.CrossDoc == nil {
		http.Error(w, "crossdoc engine não inicializado", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Validação 18 (F18.1): err.Error() não vaza.
		s.userError(w, http.StatusBadRequest, "crossdocValidate.readBody", err)
		return
	}

	var req crossdoc.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Validação 18 (F18.1): err.Error() não vaza JSON parse detail.
		s.userError(w, http.StatusBadRequest, "crossdocValidate.jsonUnmarshal", err)
		return
	}

	if len(req.Cadocs) == 0 {
		http.Error(w, `{"error":"cadocs object required with at least one CADOC"}`,
			http.StatusBadRequest)
		return
	}

	// Sprint 13 — v3.5.2 [S13.2 / C-API-3]: cross-tenant guard.
	// Sem isso, IF_A mandava cadocs de IF_B e audit log poluía. Agora
	// se req.IfID for fornecido no payload, precisa bater com claims.
	if req.IfID != "" && !s.enforceSameIF(w, r, req.IfID) {
		return
	}

	resp := s.CrossDoc.Validate(r.Context(), &req)

	// Audit (Sprint 6 v1.5.0)
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "crossdoc.validated", "crossdoc",
		body, map[string]any{
			"passed":     resp.Passed,
			"errors":     len(resp.Errors),
			"warnings":   len(resp.Warnings),
			"rules_run":  len(resp.RulesRun),
			"rules_skip": len(resp.RulesSkip),
			"cadocs":     keysOf(req.Cadocs),
		})

	writeJSON(w, http.StatusOK, resp)
}

// keysOf retorna as keys de um map (helper para audit).
func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// l4Compare compara um envio com seu anteior (L4 Histórico).
//
// GET /v1/l4/compare?envio_id=UUID
//
// Resposta:
//
//	{
//	  "current": { "envio_id": "...", "cadoc_code": "2061", "data_base": "2024-03" },
//	  "previous": { "envio_id": "...", "cadoc_code": "2061", "data_base": "2024-02" },
//	  "new_failures": [...],
//	  "fixed_rules": [...],
//	  "changed_fields": [...],
//	  "alerts": [...]
//	}
func (s *Server) l4Compare(w http.ResponseWriter, r *http.Request) {
	envioID := r.URL.Query().Get("envio_id")
	if envioID == "" {
		http.Error(w, `{"error":"envio_id query parameter required"}`, http.StatusBadRequest)
		return
	}

	comp, err := s.Audit.CompareWithPrevious(r.Context(), envioID)
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "l4Compare", err)
		return
	}

	writeJSON(w, http.StatusOK, comp)
}

// --- Middleware ---

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifID := r.Header.Get("X-IF-ID")
		// Validação 23 (F23.2): validar formato do X-IF-ID. Sem isso,
		// atacante envia X-IF-ID de 10KB → logado no audit_log → incha
		// disco. Limites:
		//   - max 64 chars (CNPJ raiz tem 8; reservei margem)
		//   - charset alfanumérico + dash + underscore
		if ifID == "" {
			http.Error(w, `{"error":"X-IF-ID header required"}`, http.StatusUnauthorized)
			return
		}
		if len(ifID) > 64 {
			http.Error(w, `{"error":"X-IF-ID too long (max 64)"}`, http.StatusBadRequest)
			return
		}
		for _, c := range ifID {
			ok := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') ||
				c == '-' || c == '_'
			if !ok {
				http.Error(w, `{"error":"X-IF-ID contains invalid character"}`, http.StatusBadRequest)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// --- Helpers ---

// getIfID retorna o tenant identifier priorizando auth.Claims (JWT, source-of-truth)
// populado pelo auth.Middleware (Sprint 7a v1.6.0+). Fallback para X-IF-ID header
// apenas em dev mode (`RADIANT_DEV_AUTH=1`) onde o middleware aceita X-IF-ID direto.
//
// Por que: handlers lendo `r.Header.Get("X-IF-ID")` direto são vetor de
// cross-tenant injection — cliente com JWT válido para tenant A injeta
// X-IF-ID header de tenant B para escrever audit_log como B. Helper centraliza
// a regra: Claims (JWT validated) > header.
//
// Em prod (sem RADIANT_DEV_AUTH=1): middleware bloqueia clientes sem JWT antes
// de chegar aqui, então helper sempre retorna Claims.IFID.
//
// Validação 27 (F27.1): fechar o gap deixado por Sprint 7a.
func getIfID(r *http.Request) string {
	if claims, err := auth.ClaimsFromContext(r.Context()); err == nil && claims != nil && claims.IFID != "" {
		return claims.IFID
	}
	return r.Header.Get("X-IF-ID")
}

// enforceSameIF valida que providedIFID é consistente com o tenant
// autenticado via JWT (claims.IFID) ou, em dev mode, com X-IF-ID header.
//
// Sprint 13 — v3.5.2 [S13.2 / C-API-3]: previne cross-tenant write/read
// em handlers que recebem if_id do payload (crossdoc, sta/submit, etc).
// Handler cross-tenant era gap identificado no audit S-A: payload
// informava if_id=X, claims.IFID=Y → backend processava e audit log
// virava fonte de poluição.
//
// Comportamento:
//   - providedIFID == "" → passa (handler assume default = getIfID(r))
//   - claims.IFID != "" e providedIFID != claims.IFID → 403
//   - dev mode (claims nil) e providedIFID != headerIFID → 403
//   - match → passa
//
// Retorna true se passou, false se respondeu 403 (handler deve retornar).
func (s *Server) enforceSameIF(w http.ResponseWriter, r *http.Request, providedIFID string) bool {
	if providedIFID == "" {
		return true
	}
	claims, err := auth.ClaimsFromContext(r.Context())
	if err == nil && claims != nil && claims.IFID != "" {
		if providedIFID != claims.IFID {
			// Validação 37 (F-1): não loga if_id values específicas no
			// erro. Err genérico mantém audit trail (Sprint 13 depura
			// cross-tenant por timestamps no log estruturado), mas evita
			// ruído e ambiguidade em agregadores de log. Cliente vê
			// 403 sem detail; log interno tem crossTenant.mismatch.
			s.userError(w, http.StatusForbidden, "crossTenant.mismatch",
				fmt.Errorf("payload.if_id != claims.if_id"))
			return false
		}
		return true
	}
	// Dev mode sem claims: alinha com header X-IF-ID (que é o que o
	// middleware aceitou). Sem isso, atacante poderia definir if_id
	// arbitrário no payload e bypass.
	if headerIFID := r.Header.Get("X-IF-ID"); headerIFID != "" && providedIFID != headerIFID {
		http.Error(w, `{"error":"if_id mismatch with X-IF-ID"}`, http.StatusForbidden)
		return false
	}
	return true
}

// Sprint 46 (v3.34.27): WhiteLabel branding handlers.

func (s *Server) getBranding(w http.ResponseWriter, r *http.Request) {
	tenantID := getIfID(r)
	if tenantID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	b, err := s.Branding.GetBranding(r.Context(), tenantID)
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "branding.get", err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) updateBranding(w http.ResponseWriter, r *http.Request) {
	tenantID := getIfID(r)
	if tenantID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req branding.UpdateBrandingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	b, err := s.Branding.UpdateBranding(r.Context(), tenantID, req)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "branding.update", err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) getBrandingBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		http.Error(w, `{"error":"slug requerido"}`, http.StatusBadRequest)
		return
	}
	b, err := s.Branding.GetBrandingBySlug(r.Context(), slug)
	if err != nil {
		s.userError(w, http.StatusNotFound, "branding.slug", err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) adminUpdateBranding(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		http.Error(w, `{"error":"tenant id requerido"}`, http.StatusBadRequest)
		return
	}
	// Admin role check via claims.
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil || claims == nil || !claims.HasRole(auth.RoleAdmin) {
		http.Error(w, `{"error":"admin required"}`, http.StatusForbidden)
		return
	}
	var req branding.UpdateBrandingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	b, err := s.Branding.UpdateBranding(r.Context(), tenantID, req)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "branding.admin.update", err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// ============================================================
// Sprint 55: Pilot3 ESG-first handlers
// ============================================================

func (s *Server) listPilotPrograms(w http.ResponseWriter, r *http.Request) {
	programs, err := s.Pilot.ListPrograms(r.Context())
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "pilot.listPrograms", err)
		return
	}
	writeJSON(w, http.StatusOK, programs)
}

func (s *Server) createPilotProgram(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil || claims == nil || !claims.HasRole(auth.RoleAdmin) {
		http.Error(w, `{"error":"admin required"}`, http.StatusForbidden)
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name requerido"}`, http.StatusBadRequest)
		return
	}
	prog, err := s.Pilot.CreateProgram(r.Context(), req.Name, req.Description, nil, nil)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "pilot.createProgram", err)
		return
	}
	writeJSON(w, http.StatusCreated, prog)
}

func (s *Server) listPilotParticipants(w http.ResponseWriter, r *http.Request) {
	programID := chi.URLParam(r, "programId")
	if programID == "" {
		http.Error(w, `{"error":"program_id requerido"}`, http.StatusBadRequest)
		return
	}
	participants, err := s.Pilot.ListParticipants(r.Context(), programID)
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "pilot.listParticipants", err)
		return
	}
	writeJSON(w, http.StatusOK, participants)
}

func (s *Server) enrollPilotParticipant(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	programID := chi.URLParam(r, "programId")
	if programID == "" {
		http.Error(w, `{"error":"program_id requerido"}`, http.StatusBadRequest)
		return
	}
	if err := s.Pilot.Enroll(r.Context(), programID, ifID); err != nil {
		s.userError(w, http.StatusBadRequest, "pilot.enroll", err)
		return
	}
	p, err := s.Pilot.GetParticipant(r.Context(), programID, ifID)
	if err != nil {
		// Enroll succeeded but GetParticipant failed — return minimal response.
		writeJSON(w, http.StatusCreated, map[string]string{
			"program_id": programID, "if_id": ifID, "status": "onboarding"})
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getPilotSteps(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	steps, err := s.Pilot.GetOnboardingProgress(r.Context(), ifID)
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "pilot.getSteps", err)
		return
	}
	writeJSON(w, http.StatusOK, steps)
}

func (s *Server) completePilotStep(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	stepKey := chi.URLParam(r, "stepKey")
	if ifID == "" || stepKey == "" {
		http.Error(w, `{"error":"if_id e step_key requeridos"}`, http.StatusBadRequest)
		return
	}
	if err := s.Pilot.CompleteStep(r.Context(), ifID, stepKey); err != nil {
		s.userError(w, http.StatusBadRequest, "pilot.completeStep", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (s *Server) getESGonboardingProgress(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	progress, steps, err := s.Pilot.GetESGProgress(r.Context(), ifID)
	if err != nil {
		s.userError(w, http.StatusInternalServerError, "pilot.getESGProgress", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"progress": progress, "steps": steps})
}

func (s *Server) enrollESGParticipant(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if err := s.Pilot.EnrollESG(r.Context(), ifID); err != nil {
		s.userError(w, http.StatusBadRequest, "pilot.enrollESG", err)
		return
	}
	progress, steps, _ := s.Pilot.GetESGProgress(r.Context(), ifID)
	writeJSON(w, http.StatusCreated, map[string]any{"progress": progress, "steps": steps})
}

// ============================================================
// writeJSON
// ============================================================

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
