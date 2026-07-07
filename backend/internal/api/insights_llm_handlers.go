// Sprint 53 v3.34.35: AI Insights — POST /v1/insights/ask.
//
// LLM interpreta o estado do ambiente CADOC/SCR/RADAR do tenant em
// linguagem natural, fundando respostas nos dados reais.
//
// Feature flag: ifs.llm_insights_enabled (opt-in).
// Rate limit: 5 req/min/tenant (aplicado pelo LLMService).
// Auth: JWT standard (mesma dos outros endpoints).
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
)

// AskLLM handles POST /v1/insights/ask.
func (s *Server) AskLLM(w http.ResponseWriter, r *http.Request) {
	if s.InsightsLLM == nil {
		http.Error(w, "insights not configured", http.StatusServiceUnavailable)
		return
	}

	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	if ifID == "" {
		http.Error(w, "if_id required in claims", http.StatusBadRequest)
		return
	}

	// Check feature flag
	enabled, err := s.isInsightsEnabled(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "check_insights_enabled")
		return
	}
	if !enabled {
		http.Error(w, "insights feature not enabled for this tenant", http.StatusForbidden)
		return
	}

	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.Question) == 0 {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}
	if len(body.Question) > 2000 {
		http.Error(w, "question too long (max 2000 chars)", http.StatusBadRequest)
		return
	}

	answer, err := s.InsightsLLM.Ask(r.Context(), ifID, body.Question)
	if err != nil {
		if errors.Is(err, insights.ErrRateLimited) {
			http.Error(w, "rate limit exceeded (5 req/min)", http.StatusTooManyRequests)
			return
		}
		s.internalServerError(w, err, "llm_ask")
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, claims.Sub, "insights.asked", "", nil, map[string]any{
		"question_len": len(body.Question),
		"model":        answer.Model,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"answer": answer.Answer,
		"model":  answer.Model,
	})
}

// isInsightsEnabled checks if the tenant has opted in to LLM insights.
func (s *Server) isInsightsEnabled(ctx context.Context, ifID string) (bool, error) {
	var enabled int
	err := s.DB.QueryRowContext(ctx,
		"SELECT llm_insights_enabled FROM ifs WHERE id = $1 AND deleted_at IS NULL",
		ifID,
	).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled == 1, nil
}
