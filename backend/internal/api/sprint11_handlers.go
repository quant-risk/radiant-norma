// Sprint 11 — drill-down handlers: rule enable/disable.
//
// GET  /v1/rules/disabled       — lista regras desabilitadas por IF
// POST /v1/rules/{code}/toggle  — alterna (enable ↔ disable)
//
// Ambos autenticados (JWT middleware). Actor no audit event vem do
// Claims.Sub (user/email/IDP subject).
//
// Sprint 12 (v3.5.0) — hardening:
//   - C32.4 + C32.19: validação de formato de rule_code via regex
//     (^[A-Z][0-9]{1,3}$). Bloqueia input malicioso/typo antes de
//     chegar no DB.
//   - C32.10: ErrRuleNotDisabled agora é mapeado pra 200 idempotente
//     ao invés de 500 confuso.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	"github.com/fortvna/radiant-norma/backend/internal/ruleprefs"
	"github.com/go-chi/chi/v5"
)

// validRuleCodePattern valida formato de código de regra.
//
// Códigos BACEN seguem padrão [A-Z][0-9]{1,3} (B12, F23, S05, C001).
// Bloqueio de input malicioso via regex no handler — defense in depth
// (backend usa parameterized queries, mas regex protege contra typos
// extremos e payloads que passam mas não fazem sentido).
var validRuleCodePattern = regexp.MustCompile(`^[A-Z][0-9]{1,3}$`)

// isValidRuleCode retorna true se code é formato válido.
func isValidRuleCode(code string) bool {
	return validRuleCodePattern.MatchString(code)
}

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
//
// Sprint 12 (v3.5.0) — C32.22: rate limit por IF (10/min default).
// 429 com Retry-After header.
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

	// C32.22: rate limit antes de qualquer processamento
	if s.ToggleLimiter != nil {
		ok, retryAfter := s.ToggleLimiter.Allow(ifID)
		if !ok {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":       "rate limit exceeded",
				"retry_after": int(retryAfter.Seconds()) + 1,
				"limit":       "10 toggles per minute per IF",
			})
			return
		}
	}
	ruleCode := chi.URLParam(r, "code")
	if ruleCode == "" {
		http.Error(w, "rule code required", http.StatusBadRequest)
		return
	}
	// C32.4 + C32.19: valida formato (defense in depth)
	if !isValidRuleCode(ruleCode) {
		http.Error(w, "invalid rule code format (expected [A-Z][0-9]{1,3})", http.StatusBadRequest)
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
		// C32.10: ErrRuleNotDisabled não é erro real — é estado já
		// mudou por outro request. É idempotente do ponto de vista do
		// estado final. Logamos + retornamos o estado conhecido.
		//
		// Sem esse fix, race conditions entre requests concorrentes
		// (Sprint 12 multi-pod) resultariam em 500 confuso.
		if errors.Is(err, ruleprefs.ErrRuleNotDisabled) {
			// Estado já é "enabled" (Enable() falhou porque não estava
			// disabled). Confirm via IsDisabled pra reportar o estado
			// real.
			isDisabled, _ := s.RulePrefs.IsDisabled(r.Context(), ifID, ruleCode)
			actualState := "enabled"
			if isDisabled {
				actualState = "disabled"
			}
			slog.Default().Info("toggle race resolved idempotently",
				"if_id", ifID,
				"rule_code", ruleCode,
				"reported_state", actualState)
			writeJSON(w, http.StatusOK, map[string]any{
				"if_id":     ifID,
				"rule_code": ruleCode,
				"new_state": actualState,
			})
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
