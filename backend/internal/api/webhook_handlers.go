package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/webhook"
	"github.com/go-chi/chi/v5"
)

// listWebhooks GET /v1/webhooks
func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request) {
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

	webhooks, err := s.Webhook.List(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "listWebhooks")
		return
	}

	writeJSON(w, http.StatusOK, webhooks)
}

// registerWebhook POST /v1/webhooks
func (s *Server) registerWebhook(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		URL         string `json:"url"`
		Events      string `json:"events"`
		Description string `json:"description,omitempty"`
		Secret      string `json:"secret,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.userError(w, http.StatusBadRequest, "registerWebhook.json", err)
		return
	}

	wk, err := s.Webhook.Register(r.Context(), ifID, req.URL, req.Events, req.Description, req.Secret)
	if err != nil {
		s.userError(w, http.StatusBadRequest, "registerWebhook", err)
		return
	}

	writeJSON(w, http.StatusCreated, wk)
}

// deleteWebhook DELETE /v1/webhooks/{id}
func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request) {
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
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		http.Error(w, "webhook id required", http.StatusBadRequest)
		return
	}

	err = s.Webhook.Delete(r.Context(), ifID, webhookID)
	if err != nil {
		s.userError(w, http.StatusNotFound, "deleteWebhook", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// listDeliveries GET /v1/webhooks/{id}/deliveries
func (s *Server) listDeliveries(w http.ResponseWriter, r *http.Request) {
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
	webhookID := chi.URLParam(r, "id")
	if webhookID == "" {
		http.Error(w, "webhook id required", http.StatusBadRequest)
		return
	}

	deliveries, err := s.Webhook.ListDeliveries(r.Context(), ifID, webhookID, 50)
	if err != nil {
		s.internalServerError(w, err, "listDeliveries")
		return
	}

	writeJSON(w, http.StatusOK, deliveries)
}

// DispatchValidationCompleted fires a validation.completed webhook event.
func DispatchValidationCompleted(wb *webhook.Service, ifID, cadoc, dataBase, xmlHash string, valid bool, errorCount, warnCount int) {
	if wb == nil {
		return
	}
	evt := webhook.EventValidationCompleted{}
	evt.WebhookBase = webhook.WebhookBase{
		Event:     "validation.completed",
		Timestamp: webhook.Now(),
		IFID:      ifID,
	}
	evt.Cadoc = cadoc
	evt.DataBase = dataBase
	evt.XMLHash = xmlHash
	evt.Valid = valid
	evt.ErrorCount = errorCount
	evt.WarnCount = warnCount

	wb.Dispatch(context.Background(), ifID, "validation.completed", evt)
}
