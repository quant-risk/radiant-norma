// Sprint 11 — drill-down handlers: rule enable/disable.
//
// GET  /v1/rules/disabled       — lista regras desabilitadas por IF
// POST /v1/rules/{code}/toggle  — alterna (enable ↔ disable)
//
// Ambos autenticados (JWT middleware). Actor no audit event vem do
// Claims.Sub (user/email/IDP subject).

package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/go-chi/chi/v5"
)

// listDisabledRules retorna todas as regras desabilitadas por 1 IF.
func (s *Server) listDisabledRules(w http.ResponseWriter, r *http.Request) {
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

	rules, err := s.RulePrefs.ListDisabled(r.Context(), ifID)
	if err != nil {
		s.internalServerError(w, err, "list_disabled")
		return
	}
	if rules == nil {
		rules = []ruleprefs.DisabledRule{} // nunca null no JSON
	}

	// Resposta inclui codes como Set pra facilitar lookup client-side.
	codes := make([]string, 0, len(rules))
	for _, r := range rules {
		codes = append(codes, r.RuleCode)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"if_id": ifID,
		"rules": rules,
		"codes": codes,
		"total": len(rules),
	})
}

// toggleRule alterna enable/disable de 1 regra por IF.
//
// Body: { "expected_state"?: "enabled" | "disabled" } (opcional —
// validação otimista, retorna 409 se estado atual difere).
//
// Audit event emitido: rule.disabled ou rule.enabled.
func (s *Server) toggleRule(w http.ResponseWriter, r *http.Request) {
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
	ruleCode := chi.URLParam(r, "code")
	if ruleCode == "" {
		http.Error(w, "rule code required", http.StatusBadRequest)
		return
	}

	// Parse body opcional (expected_state)
	var body struct {
		ExpectedState string `json:"expected_state"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
	}

	// Optimistic concurrency check
	if body.ExpectedState != "" {
		isDisabled, err := s.RulePrefs.IsDisabled(r.Context(), ifID, ruleCode)
		if err != nil {
			s.internalServerError(w, err, "toggle_check")
			return
		}
		currentState := "enabled"
		if isDisabled {
			currentState = "disabled"
		}
		if currentState != body.ExpectedState {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":         "state mismatch",
				"current_state": currentState,
				"expected":      body.ExpectedState,
			})
			return
		}
	}

	// Toggle
	newState, err := s.RulePrefs.Toggle(r.Context(), ifID, ruleCode, claims.Sub)
	if err != nil {
		if errors.Is(err, ruleprefs.ErrRuleNotDisabled) {
			http.Error(w, "internal state error", http.StatusInternalServerError)
			return
		}
		s.internalServerError(w, err, "toggle")
		return
	}

	// Audit log
	action := "rule.disabled"
	if newState == "enabled" {
		action = "rule.enabled"
	}
	auditBody, _ := json.Marshal(map[string]any{
		"rule_code": ruleCode,
		"new_state": newState,
	})
	_, _ = s.AuditLog.Log(ifID, claims.Sub, action, ruleCode, auditBody, map[string]any{
		"actor": claims.Sub,
		"role":  string(claims.Role),
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"if_id":     ifID,
		"rule_code": ruleCode,
		"new_state": newState,
	})
}