// Tests para o helper getIfID (Sprint 7c post / Validação 27 / F27.1).
//
// Cobertura:
//   - Prioriza auth.Claims.IFID (JWT, source-of-truth) quando context tem Claims
//   - Fallback para X-IF-ID header quando context não tem Claims (dev mode)
//   - Retorna string vazia quando nem Claims nem header (caller faz 401)
//
// Estes testes rodam no package `api` (não `api_test`) porque getIfID é
// helper não-exportado. Validação 27 fechou F27.1 — vetor onde handlers
// liam X-IF-ID cru, ignorando Claims JWT, criando vetor de cross-tenant
// injection se cliente enviasse X-IF-ID malicioso com JWT válido.
package api

import (
	"net/http/httptest"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
)

// TestGetIfID_PriorizaClaims: getIfID retorna Claims.IFID (do JWT) ao invés
// de header X-IF-ID. Defesa contra cross-tenant injection (F27.1).
func TestGetIfID_PriorizaClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("X-IF-ID", "header-attacker-tries-this")

	claims := &auth.Claims{IFID: "jwt-claims-authoritative"}
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	got := getIfID(req)
	if got != "jwt-claims-authoritative" {
		t.Errorf("getIfID com Claims+header malicioso = %q, want %q", got, "jwt-claims-authoritative")
	}
}

// TestGetIfID_FallbackHeader: sem Claims no context, getIfID retorna header
// (dev mode X-IF-ID).
func TestGetIfID_FallbackHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("X-IF-ID", "demo-bank")

	got := getIfID(req)
	if got != "demo-bank" {
		t.Errorf("getIfID sem Claims com X-IF-ID = %q, want %q", got, "demo-bank")
	}
}

// TestGetIfID_VazioSemCreds: sem Claims e sem header → string vazia → caller
// retorna 401.
func TestGetIfID_VazioSemCreds(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/test", nil)

	got := getIfID(req)
	if got != "" {
		t.Errorf("getIfID sem nada = %q, want empty", got)
	}
}

// TestGetIfID_ClaimsVazioFallbackHeader: edge — middleware deveria garantir
// IFID não-vazio em prod, mas se por bug vier vazio, getIfID cai pra header.
// Em produção isso não acontece porque middleware bloqueia antes.
func TestGetIfID_ClaimsVazioFallbackHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("X-IF-ID", "fallback-value")

	claims := &auth.Claims{IFID: ""}
	req = req.WithContext(auth.WithClaims(req.Context(), claims))

	got := getIfID(req)
	if got != "fallback-value" {
		t.Errorf("getIfID com Claims.IFID vazio = %q, want %q (fallback header)", got, "fallback-value")
	}
}
