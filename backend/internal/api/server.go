// Package api implementa os handlers HTTP da API REST do Radiant Sentinel.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/fortvna/radiant-sentinel/backend/internal/audit"
	"github.com/fortvna/radiant-sentinel/backend/internal/auditlog"
	"github.com/fortvna/radiant-sentinel/backend/internal/schema"
	"github.com/fortvna/radiant-sentinel/backend/internal/sta"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server agrega todos os serviços.
type Server struct {
	Schema    *schema.Registry
	Audit     *audit.Service
	AuditLog  *auditlog.Logger
	STAClient sta.Client
}

// NewServer cria um Server.
func NewServer(sch *schema.Registry, aud *audit.Service, al *auditlog.Logger, staClient sta.Client) *Server {
	return &Server{Schema: sch, Audit: aud, AuditLog: al, STAClient: staClient}
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
	})

	return r
}

// --- Handlers ---

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"time":    time.Now().UTC().Format(time.RFC3339),
		"version": "1.2.0",
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
	writeJSON(w, http.StatusOK, map[string]any{
		"cadoc":    cadoc,
		"rules":    criticas,
		"total":    len(criticas),
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
		http.Error(w, "cadoc_code required", http.StatusBadRequest)
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
		"passed":     resp.Passed,
		"errors":     len(resp.Errors),
		"warnings":   len(resp.Warnings),
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

	// Lê XML e gera ZIP stub (em Sprint 4: ZIP real)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sub := &sta.Submission{
		CadocCode: r.URL.Query().Get("cadoc"),
		DataBase:  r.URL.Query().Get("data_base"),
		XML:       body,
		Zip:       body, // stub: mesmo conteúdo
		CNPJ:      ifID,
	}

	result, err := s.STAClient.Submit(r.Context(), sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, r.RemoteAddr, "sta.submit", sub.CadocCode, body, map[string]any{
		"protocol": result.ProtocolSTA,
		"accepted": result.Accepted,
	})

	writeJSON(w, http.StatusOK, result)
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