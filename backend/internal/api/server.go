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
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/realtime"
	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/version"
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
	DB        *sql.DB
	Schema    *schema.Registry
	Audit     *audit.Service
	AuditLog  auditLogAPI
	STAClient sta.Client
	Radar     *radar.Service
	CrossDoc  *crossdoc.Engine // Sprint 6 v1.5.0 — Cross-Doc L3
	RulePrefs *ruleprefs.Preferences // Sprint 11 v3.4.0 — disable/enable por IF
	ToggleLimiter *ruleprefs.ToggleLimiter // Sprint 12 v3.5.0 — C32.22 rate limit toggle
	Insights  *insights.Acknowledgments // Sprint 12 v3.5.0 — recommendation ack

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

	// Sprint 10 — SSE hub para real-time push (alertas/audit/envios).
	// Se nil, /v1/events/stream retorna 503.
	EventsHub *realtime.Hub

	// Sprint 13 — v3.5.2 [S15.1] rate limiter global.
	RateLimiter *apiRateLimiter
}

// auditLogAPI é interface mínima que *auditlog.Logger e *realtime.HubAwareLogger
// ambos satisfazem. Permite wrap sem mudar assinatura do Server.
type auditLogAPI interface {
	Log(ifID, actor, action, target string, payload []byte, metadata any) (*auditlog.Entry, error)
	Verify() (bool, int, error)
}

// NewServer cria um Server.
func NewServer(d *sql.DB, sch *schema.Registry, aud *audit.Service, al auditLogAPI, staClient sta.Client, rad *radar.Service, rp *ruleprefs.Preferences, tl *ruleprefs.ToggleLimiter, ack *insights.Acknowledgments) *Server {
	return &Server{
		DB: d, Schema: sch, Audit: aud, AuditLog: al, STAClient: staClient,
		Radar: rad, RulePrefs: rp, ToggleLimiter: tl, Insights: ack,
		startedAt:   time.Now(),
		RateLimiter: newAPIRateLimiter(),
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
	r.Use(rateLimitMiddleware(s.RateLimiter))

	// Sprint 6 v1.5.0 (F12.8 fix): Recoverer ANTES de Logger para que
	// panics que viram 500 sejam loggados (não engolidos pelo Logger
	// que está acima na pilha).
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	// Health
	r.Get("/healthz", s.healthz)
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

		// Rules (críticas)
		r.Get("/rules", s.listRules)
		r.Get("/rules/{cadoc}", s.listRulesByCadoc)

		// Sprint 11 (v3.4.0) — drill-down: enable/disable regras por IF.
		// Persistência no backend (era localStorage no frontend).
		r.Get("/rules/disabled", s.listDisabledRules)
		r.Post("/rules/{code}/toggle", s.toggleRule)

		// Validation
		r.Post("/validate", s.validate)

		// STA submission (stub)
		r.Post("/sta/submit", s.staSubmit)

		// Radar regulatório (Sprint 4)
		r.Get("/radar/alerts", s.listRadarAlerts)
		r.Get("/radar/alerts/{id}", s.getRadarAlert)
		r.Post("/radar/alerts/{id}/resolve", s.resolveRadarAlert)
		r.Post("/radar/scan", s.triggerRadarScan)

		// Cross-Doc L3 (Sprint 6 v1.5.0) — diferencial proprietário.
		// Valida ecossistema inteiro (3040 ↔ 4111 ↔ DRSAC) em paralelo.
		r.Post("/crossdoc/validate", s.crossdocValidate)

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
		})

		// Sprint 10 — SSE real-time stream.
		// Auth vem do middleware JWT global (acima).
		// Stream filtra por IF automaticamente (atrás de IF=auth).
		r.Get("/events/stream", s.eventsStreamHandler)
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
			s.userError(w, http.StatusForbidden, "crossTenant.mismatch",
				fmt.Errorf("payload.if_id=%q != claims.if_id=%q", providedIFID, claims.IFID))
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
