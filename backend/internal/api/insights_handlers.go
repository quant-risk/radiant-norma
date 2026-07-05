// Sprint 12 (opcional) — drill-down handler: acknowledge recommendation.
//
// POST /v1/insights/recommendations/{id}/acknowledge  — marca como vista
//
// Auth: mesma do resto (JWT middleware). IFID vem do claims.
//
// Audit: rule.disabled-style event com action "recommendation.acknowledged"
// é emitido pelo HubAwareLogger.

package api

import (
	"errors"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/go-chi/chi/v5"
)

// acknowledgeRecommendation marca 1 recommendation como acknowledged.
//
// ID do recommendation é opaco (vem do backend que computa). Apenas
// armazenamos (if_id, rec_id) — sem validar que rec_id existe no
// domain. Validação E2E é feita via GET /v1/insights/recommendations
// que lista os válidos.
func (s *Server) acknowledgeRecommendation(w http.ResponseWriter, r *http.Request) {
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
	recID := chi.URLParam(r, "id")
	if recID == "" {
		http.Error(w, "recommendation id required", http.StatusBadRequest)
		return
	}

	ack, err := s.Insights.Acknowledge(r.Context(), ifID, recID, claims.Sub)
	if err != nil {
		s.internalServerError(w, err, "acknowledge_recommendation")
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, claims.Sub, "recommendation.acknowledged", recID, nil, map[string]any{
		"actor": claims.Sub,
		"role":  string(claims.Role),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"if_id":           ack.IFID,
		"rec_id":          ack.RecID,
		"acknowledged_at": ack.AcknowledgedAt,
		"acknowledged_by": ack.AcknowledgedBy,
	})
}

// unacknowledgeRecommendation remove acknowledgment (desfaz).
func (s *Server) unacknowledgeRecommendation(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	recID := chi.URLParam(r, "id")
	if recID == "" {
		http.Error(w, "recommendation id required", http.StatusBadRequest)
		return
	}

	err = s.Insights.Unacknowledge(r.Context(), ifID, recID)
	if err != nil {
		// C34.9: errors.Is ao invés de == pra forward compat.
		if errors.Is(err, insights.ErrRecommendationNotAcknowledged) {
			http.Error(w, "recommendation not acknowledged", http.StatusNotFound)
			return
		}
		s.internalServerError(w, err, "unacknowledge_recommendation")
		return
	}

	// Audit log
	_, _ = s.AuditLog.Log(ifID, claims.Sub, "recommendation.unacknowledged", recID, nil, map[string]any{
		"actor": claims.Sub,
		"role":  string(claims.Role),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"if_id":  ifID,
		"rec_id": recID,
		"status": "unacknowledged",
	})
}