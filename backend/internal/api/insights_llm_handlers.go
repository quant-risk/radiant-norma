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
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
)

// streamAskLLM handles GET /v1/insights/ask/stream — SSE streaming for insights chat.
//
// Streams the LLM response in real-time as SSE events:
//
//	event: chunk
//	data: {"text":"partial answer...","model":"MiniMax-Text-01","done":false}
//
//	event: done
//	data: {"text":"","model":"MiniMax-Text-01","done":true}
//
//	event: error
//	data: {"error":"rate limit exceeded"}
//
// Feature flag: ifs.llm_insights_enabled (opt-in).
// Auth: JWT standard (mesma dos outros endpoints).
// Rate limit: 5 req/min/tenant (aplicado pelo LLMService).
func (s *Server) streamAskLLM(w http.ResponseWriter, r *http.Request) {
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

	enabled, err := s.isInsightsEnabled(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "check_insights_enabled")
		return
	}
	if !enabled {
		http.Error(w, "insights feature not enabled for this tenant", http.StatusForbidden)
		return
	}

	question := r.URL.Query().Get("question")
	if question == "" {
		http.Error(w, "question query param is required", http.StatusBadRequest)
		return
	}
	if len(question) > 2000 {
		http.Error(w, "question too long (max 2000 chars)", http.StatusBadRequest)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream chunks as they arrive.
	ch, err := s.InsightsLLM.StreamAsk(r.Context(), ifID, question)
	if err != nil {
		slog.Warn("StreamAsk init failed", "err", err)
		return
	}

	// Send audit log after streaming (non-blocking).
	// The audit will be logged after full response is received.
	defer func() {
		_, _ = s.AuditLog.Log(ifID, claims.Sub, "insights.asked.stream", "", nil, map[string]any{
			"question_len": len(question),
		})
	}()

	for chunk := range ch {
		if chunk.Error != nil {
			fmt.Fprintf(w, "event: error\ndata: {\"error\":%q}\n\n",
				chunk.Error.Error())
			flusher.Flush()
			return
		}
		done := chunk.Text == "" && chunk.Done
		fmt.Fprintf(w, "event: chunk\ndata: {\"text\":%q,\"model\":%q,\"done\":%v}\n\n",
			chunk.Text, chunk.Model, done)
		flusher.Flush()
		if done {
			return
		}
	}
}

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
		s.userError(w, http.StatusBadRequest, "AskLLM.json", err)
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
		"cached": answer.Cached,
	})
}

// getInsightsHistory handles GET /v1/insights/history.
func (s *Server) getInsightsHistory(w http.ResponseWriter, r *http.Request) {
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

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	msgs, err := s.InsightsLLM.GetHistory(r.Context(), ifID, limit)
	if err != nil {
		s.internalServerError(w, err, "getInsightsHistory")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": msgs,
		"count":    len(msgs),
	})
}

// deleteInsightsHistory handles DELETE /v1/insights/history.
func (s *Server) deleteInsightsHistory(w http.ResponseWriter, r *http.Request) {
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

	if err := s.InsightsLLM.ClearHistory(r.Context(), ifID); err != nil {
		s.internalServerError(w, err, "deleteInsightsHistory")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "cleared",
	})
}

// isInsightsEnabled checks if the tenant has opted in to LLM insights.
func (s *Server) isInsightsEnabled(ctx context.Context, ifID string) (bool, error) {
	var enabled int
	err := s.DB.QueryRowContext(ctx,
		"SELECT llm_insights_enabled FROM ifs WHERE id = ? AND deleted_at IS NULL",
		ifID,
	).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled == 1, nil
}
