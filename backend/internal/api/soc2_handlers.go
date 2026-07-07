// Sprint 56 v3.34.38: SOC 2 Type I — readiness + evidence handlers.
//
// GET /v1/admin/soc2/readiness  — relatório de readiness SOC 2
// GET /v1/admin/soc2/controls   — lista de controles e status
// GET /v1/admin/soc2/controls/{id}/evidence — coleta evidências para 1 controle
//
// Auth: admin only.
package api

import (
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/soc2"
	"github.com/go-chi/chi/v5"
)

// getSOC2Readiness handles GET /v1/admin/soc2/readiness.
func (s *Server) getSOC2Readiness(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if !claims.HasRole(auth.RoleAdmin) {
		http.Error(w, "admin role required", http.StatusForbidden)
		return
	}

	svc := soc2.NewService(s.DB)
	report, err := svc.GenerateReadinessReport(r.Context())
	if err != nil {
		s.internalServerError(w, err, "soc2_readiness")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

// listSOC2Controls handles GET /v1/admin/soc2/controls.
func (s *Server) listSOC2Controls(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if !claims.HasRole(auth.RoleAdmin) {
		http.Error(w, "admin role required", http.StatusForbidden)
		return
	}

	controls := soc2.DefaultControls()
	writeJSON(w, http.StatusOK, map[string]any{
		"controls": controls,
		"total":    len(controls),
	})
}

// getSOC2ControlEvidence handles GET /v1/admin/soc2/controls/{id}/evidence.
func (s *Server) getSOC2ControlEvidence(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if !claims.HasRole(auth.RoleAdmin) {
		http.Error(w, "admin role required", http.StatusForbidden)
		return
	}

	controlID := chi.URLParam(r, "id")
	if controlID == "" {
		http.Error(w, "control id required", http.StatusBadRequest)
		return
	}

	collector := soc2.NewEvidenceCollector(s.DB, nil)
	evidence, err := collector.CollectEvidenceForControl(r.Context(), controlID)
	if err != nil {
		s.internalServerError(w, err, "soc2_evidence")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"control_id": controlID,
		"evidence":   evidence,
		"total":      len(evidence),
	})
}
