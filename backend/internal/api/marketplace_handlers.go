package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/marketplace"
	"github.com/go-chi/chi/v5"
)

// listMarketplaceRules GET /v1/marketplace
func (s *Server) listMarketplaceRules(w http.ResponseWriter, r *http.Request) {
	cadoc := r.URL.Query().Get("cadoc")
	tag := r.URL.Query().Get("tag")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	rules, total, err := s.Marketplace.List(r.Context(), cadoc, tag, limit, offset)
	if err != nil {
		s.internalServerError(w, err, "listMarketplaceRules")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"rules":  rules,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// publishRule POST /v1/marketplace
func (s *Server) publishRule(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	if ifID == "" {
		http.Error(w, "if_id required", http.StatusBadRequest)
		return
	}

	if r.ContentLength == 0 || r.Body == nil {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	var req marketplace.PublishRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.userError(w, http.StatusBadRequest, "publishRule.json", err)
		return
	}
	req.AuthorIFID = ifID

	rule, err := s.Marketplace.Publish(r.Context(), req)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "publishRule", err)
		return
	}

	writeJSON(w, http.StatusCreated, rule)
}

// installRule POST /v1/marketplace/{id}/install
func (s *Server) installRule(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	if ifID == "" {
		http.Error(w, "if_id required", http.StatusBadRequest)
		return
	}

	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		http.Error(w, "rule id required", http.StatusBadRequest)
		return
	}

	err = s.Marketplace.Install(r.Context(), ruleID, ifID)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "installRule", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "installed"})
}

// rateRule POST /v1/marketplace/{id}/rate
func (s *Server) rateRule(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	if ifID == "" {
		http.Error(w, "if_id required", http.StatusBadRequest)
		return
	}

	ruleID := chi.URLParam(r, "id")
	if ruleID == "" {
		http.Error(w, "rule id required", http.StatusBadRequest)
		return
	}

	var req struct {
		Stars int `json:"stars"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.userError(w, http.StatusBadRequest, "rateRule.json", err)
		return
	}

	err = s.Marketplace.Rate(r.Context(), ruleID, ifID, req.Stars)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "rateRule", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "rated"})
}

// listInstalledRules GET /v1/marketplace/installed
func (s *Server) listInstalledRules(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	ifID := claims.IFID
	if ifID == "" {
		http.Error(w, "if_id required", http.StatusBadRequest)
		return
	}

	rules, err := s.Marketplace.GetInstalled(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "listInstalledRules")
		return
	}

	writeJSON(w, http.StatusOK, rules)
}
