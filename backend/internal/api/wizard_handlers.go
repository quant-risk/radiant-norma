// Package api — wizard HTTP handlers for the generation wizard (Sprint 57).
//
// Endpoints:
//
//	POST /v1/generate/wizard/start    — cria nova sessão wizard
//	GET  /v1/generate/wizard/{id}     — retorna sessão atual
//	PUT  /v1/generate/wizard/{id}    — avança para próximo step
//	GET  /v1/generate/wizard/{id}/xml — retorna XML gerado
//
// Auth: JWT via middleware (X-IF-ID or Bearer).
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/generator/wizard"
	"github.com/go-chi/chi/v5"
)

// startWizard handles POST /v1/generate/wizard/start.
func (s *Server) startWizard(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, `{"error":"X-IF-ID required"}`, http.StatusUnauthorized)
		return
	}

	session, err := s.WizardStore.Create(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "startWizard.create")
		return
	}

	writeJSON(w, http.StatusCreated, session)
}

// getWizard handles GET /v1/generate/wizard/{id}.
func (s *Server) getWizard(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, `{"error":"X-IF-ID required"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		s.userError(w, http.StatusBadRequest, "getWizard.id", errors.New("id requerido"))
		return
	}

	session, err := s.WizardStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, wizard.ErrSessionNotFound) {
			http.Error(w, `{"error":"sessão não encontrada"}`, http.StatusNotFound)
			return
		}
		s.internalServerError(w, err, "getWizard.get")
		return
	}

	// Tenant isolation.
	if session.IfID != ifID {
		http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// advanceWizard handles PUT /v1/generate/wizard/{id}.
func (s *Server) advanceWizard(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, `{"error":"X-IF-ID required"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		s.userError(w, http.StatusBadRequest, "advanceWizard.id", errors.New("id requerido"))
		return
	}

	session, err := s.WizardStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, wizard.ErrSessionNotFound) {
			http.Error(w, `{"error":"sessão não encontrada"}`, http.StatusNotFound)
			return
		}
		s.internalServerError(w, err, "advanceWizard.get")
		return
	}
	if session.IfID != ifID {
		http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
		return
	}

	// Only allow advance from non-terminal steps.
	if session.Step == wizard.StepGenerate {
		s.userError(w, http.StatusConflict, "advanceWizard.step",
			errors.New("wizard já está completo — use POST /generate para gerar novamente"))
		return
	}

	var body wizardAdvanceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.userError(w, http.StatusBadRequest, "advanceWizard.json", err)
		return
	}

	// Validate CADOC code if provided.
	if body.CadocCode != "" {
		if err := ValidateCadocCode(body.CadocCode); err != nil {
			s.userError(w, http.StatusBadRequest, "advanceWizard.cadocCode", err)
			return
		}
	}

	data := map[string]any{
		"cadoc_code":  body.CadocCode,
		"source_type": body.SourceType,
	}
	if body.FieldMapping != nil {
		data["field_mapping"] = body.FieldMapping
	}

	// Sprint 57 v3.36.3: backward navigation via ?direction=prev.
	// Usado pelo wizard UI para revisar/corrigir seleção anterior.
	direction := r.URL.Query().Get("direction")
	var updated *wizard.Session
	if direction == "prev" {
		updated, err = s.WizardStore.Revindicate(r.Context(), id, data)
		if errors.Is(err, wizard.ErrInvalidTransition) {
			s.userError(w, http.StatusConflict, "advanceWizard.prev",
				errors.New("não é possível voltar do step atual"))
			return
		}
	} else {
		updated, err = s.WizardStore.Advance(r.Context(), id, data)
	}
	if err != nil {
		s.internalServerError(w, err, "advanceWizard.advance")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// getWizardXML handles GET /v1/generate/wizard/{id}/xml.
func (s *Server) getWizardXML(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)
	if ifID == "" {
		http.Error(w, `{"error":"X-IF-ID required"}`, http.StatusUnauthorized)
		return
	}

	id := chi.URLParam(r, "id")
	session, err := s.WizardStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, wizard.ErrSessionNotFound) {
			http.Error(w, `{"error":"sessão não encontrada"}`, http.StatusNotFound)
			return
		}
		s.internalServerError(w, err, "getWizardXML.get")
		return
	}
	if session.IfID != ifID {
		http.Error(w, `{"error":"acesso negado"}`, http.StatusForbidden)
		return
	}
	if session.GeneratedXML == "" {
		http.Error(w, `{"error":"XML não gerado ainda"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	if _, err := w.Write([]byte(session.GeneratedXML)); err != nil {
		slog.Warn("wizard xml write", "err", err)
	}
}

// wizardAdvanceRequest is the body for PUT /v1/generate/wizard/{id}.
type wizardAdvanceRequest struct {
	CadocCode    string          `json:"cadoc_code,omitempty"`
	SourceType   string          `json:"source_type,omitempty"`
	FieldMapping json.RawMessage `json:"field_mapping,omitempty"`
}
