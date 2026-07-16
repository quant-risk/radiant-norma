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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
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

	// Phase 2: quando avança para StepGenerate, executa a geração.
	if updated.Step == wizard.StepGenerate && session.Step != wizard.StepGenerate {
		if err := s.executeWizardGeneration(r.Context(), ifID, updated); err != nil {
			// Registra erro mas não bloqueia — usuário pode ver erro.
			slog.Error("wizard generation failed", "session", id, "err", err)
			// Salva erro na sessão.
			s.WizardStore.SetError(r.Context(), id, []string{err.Error()})
		}
		// Recarrega sessão para obter XML gerado.
		updated, _ = s.WizardStore.Get(r.Context(), id)
	}

	writeJSON(w, http.StatusOK, updated)
}

// executeWizardGeneration executa a geração de XML para a sessão do wizard.
// Phase 2: integra o generator com o wizard step final.
func (s *Server) executeWizardGeneration(ctx context.Context, ifID string, session *wizard.Session) error {
	// Se não há CADOC, não pode gerar.
	if session.CadocCode == "" {
		return errors.New(" CADOC não selecionado")
	}

	g := s.resolveGenerator(session.CadocCode)
	if g == nil {
		return errors.New("generator não encontrado para " + session.CadocCode)
	}

	// Reconstrói o CanonicalDocument do JSON salvo.
	var doc canonical.CanonicalDocument
	if session.CanonicalJSON != "" {
		if err := json.Unmarshal([]byte(session.CanonicalJSON), &doc); err != nil {
			return fmt.Errorf("falha ao reconstruir canonical: %w", err)
		}
	}

	// Usa data_base atual ou padrão.
	dataBase := time.Now()
	if !time.Time(doc.DataBase).IsZero() {
		dataBase = time.Time(doc.DataBase)
	}

	// Gera o XML.
	generated, err := g.Generate(ctx, &doc, dataBase)
	if err != nil {
		return fmt.Errorf("falha na geração: %w", err)
	}

	// Salva XML gerado na sessão.
	if err := s.WizardStore.SetGeneratedXML(ctx, session.ID, string(generated.XML)); err != nil {
		return fmt.Errorf("falha ao salvar XML: %w", err)
	}

	return nil
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
