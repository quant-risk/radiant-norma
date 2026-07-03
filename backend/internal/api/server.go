// Package api implementa os handlers HTTP da API REST do Radiant Norma.
package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server agrega todos os serviços.
type Server struct {
	DB        *sql.DB
	Schema    *schema.Registry
	Audit     *audit.Service
	AuditLog  *auditlog.Logger
	STAClient sta.Client
	Radar     *radar.Service
	startedAt time.Time
}

// NewServer cria um Server.
func NewServer(d *sql.DB, sch *schema.Registry, aud *audit.Service, al *auditlog.Logger, staClient sta.Client, rad *radar.Service) *Server {
	return &Server{DB: d, Schema: sch, Audit: aud, AuditLog: al, STAClient: staClient, Radar: rad, startedAt: time.Now()}
}

// Router retorna o chi router configurado.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware padrão
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health
	r.Get("/healthz", s.healthz)
	r.Get("/readyz", s.healthz)

	// API v1
	r.Route("/v1", func(r chi.Router) {
		r.Use(s.authMiddleware) // X-IF-ID obrigatório

		// Schemas
		r.Get("/schemas", s.listSchemas)
		r.Get("/schemas/{cadoc}", s.getSchema)
		r.Get("/schemas/{cadoc}/versions", s.listVersions)

		// Rules (críticas)
		r.Get("/rules", s.listRules)
		r.Get("/rules/{cadoc}", s.listRulesByCadoc)

		// Validation
		r.Post("/validate", s.validate)

		// STA submission (stub)
		r.Post("/sta/submit", s.staSubmit)

		// Radar regulatório (Sprint 4)
		r.Get("/radar/alerts", s.listRadarAlerts)
		r.Get("/radar/alerts/{id}", s.getRadarAlert)
		r.Post("/radar/alerts/{id}/resolve", s.resolveRadarAlert)
		r.Post("/radar/scan", s.triggerRadarScan)
	})

	return r
}

// --- Handlers ---

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"time":           time.Now().UTC().Format(time.RFC3339),
		"version":        "1.4.0",
		"uptime_seconds": int(time.Since(s.startedAt).Seconds()),
	})
}

func (s *Server) listSchemas(w http.ResponseWriter, r *http.Request) {
	cadocs := []string{"3040", "3042", "3044", "3050", "2030", "2060", "2061", "2062", "2070", "2160", "2170"}
	writeJSON(w, http.StatusOK, map[string]any{
		"cadocs": cadocs,
		"total":  len(cadocs),
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) listVersions(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	versions, err := s.Schema.List(cadoc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cadoc":    cadoc,
		"versions": versions,
		"total":    len(versions),
	})
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"cadocs": []string{"3040", "3042", "3044", "3050", "2030", "2060", "2061", "2062", "2070", "2160", "2170"},
	})
}

func (s *Server) listRulesByCadoc(w http.ResponseWriter, r *http.Request) {
	cadoc := chi.URLParam(r, "cadoc")
	criticas, err := s.Audit.LoadCriticas(r.Context(), cadoc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	ifID := r.Header.Get("X-IF-ID")
	if ifID == "" {
		http.Error(w, "X-IF-ID required", http.StatusUnauthorized)
		return
	}

	// Lê body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.CadocCode == "" {
		http.Error(w, "cadoc (or cadoc_code) required", http.StatusBadRequest)
		return
	}
	if len(req.XML) == 0 {
		http.Error(w, "xml required", http.StatusBadRequest)
		return
	}
	if req.ContentType == "" {
		req.ContentType = "application/xml"
	}

	// Executa validação
	resp, err := s.Audit.Validate(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	ifID := r.Header.Get("X-IF-ID")
	if ifID == "" {
		http.Error(w, "X-IF-ID required", http.StatusUnauthorized)
		return
	}

	// Aceita body JSON (preferencial, contrato documentado) OU query params (retrocompat).
	var sub sta.Submission
	body, _ := io.ReadAll(r.Body)

	if r.Header.Get("Content-Type") == "application/json" && len(body) > 0 {
		if err := json.Unmarshal(body, &sub); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.submit.persist_failed",
			sub.CadocCode, body, map[string]any{"err": dbErr.Error()})
		writeJSON(w, http.StatusOK, map[string]any{
			"protocol_sta": result.ProtocolSTA,
			"accepted":     result.Accepted,
			"rejection":    result.Rejection,
			"envio_id":     envioID,
			"warning":      "envio não persistido: " + dbErr.Error(),
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if a == nil {
		http.Error(w, "alert não encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) resolveRadarAlert(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resolved": true, "id": id})
}

func (s *Server) triggerRadarScan(w http.ResponseWriter, r *http.Request) {
	if s.Radar == nil {
		http.Error(w, "radar não inicializado", http.StatusServiceUnavailable)
		return
	}
	alerts, err := s.Radar.ScanOnce(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"new_alerts": alerts,
		"count":      len(alerts),
	})
}

// --- Middleware ---

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Em Sprint 4: validar JWT/OAuth
		// Por enquanto, X-IF-ID obrigatório (multi-tenant simples)
		if r.Header.Get("X-IF-ID") == "" {
			http.Error(w, `{"error":"X-IF-ID header required"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
