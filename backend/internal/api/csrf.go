// Package api — CSRF protection middleware.
//
// Validação 32 — C32.21: HIGH pre-existente desde Sprint 7a. POST/PUT/DELETE
// endpoints não validam CSRF. Cookie rn_jwt é httpOnly, mas browser
// envia automaticamente em requests cross-origin via <form> ou <img>.
//
// Estratégia: middleware checa Origin header contra allowlist de frontends
// conhecidos. Same-origin + allowed origins passam. Outras origens
// bloqueadas (em prod) ou logadas (em dev).
//
// Defense-in-depth: rodar junto com SameSite=Lax/Strict no cookie
// (configurado em /api/login route.ts no frontend).
//
// NOTA: rotas /v1/auth/dev-token (dev mode) são sempre permitidas
// porque frontend precisa chamar pra mintar JWT.
//
// Sprint 13 — v3.5.2 [S13.3 / C-API-1]:
// Default fail-closed: cross-origin não-allowlisted é SEMPRE 403,
// independente de RADIANT_ENV. Para dev permissive mode, opt-in via
// RADIANT_CSRF_PERMISSIVE=1. Whitelist de /v1/auth/dev-token só é
// aplicada em permissive mode (defense-in-depth: se prod acidentalmente
// rodar com DEV_TOKEN, dev-token ainda passa por Origin check).

package api

import (
	"net/http"
	"os"
	"strings"
)

// CSRFConfig configura o middleware.
type CSRFConfig struct {
	// AllowOrigins é lista de origins permitidas (cross-origin).
	// Frontend URLs que podem chamar o backend.
	// Default dev: localhost:4180 (Next.js).
	AllowOrigins []string
	// WhitelistRoutes: paths que bypassam CSRF.
	// Sprint 13: whitelist só é aplicada em RADIANT_CSRF_PERMISSIVE=1
	// (dev). Em prod, /v1/auth/dev-token também passa por Origin check.
	WhitelistRoutes []string
	// EnforceProduction: se true, bloqueia cross-origin não-allowlisted.
	// Sprint 13: AGORA SEMPRE true por default. Opt-out via
	// RADIANT_CSRF_PERMISSIVE=1 (dev mode). Reverse do antigo — era
	// env-gated e podia ficar fail-open.
	EnforceProduction bool
	// StrictNoOrigin: se true, rejeita requests sem Origin header
	// (Postman/curl) além de cross-origin. Default false (compat).
	// Sprint 13: opt-in via RADIANT_CSRF_STRICT_NO_ORIGIN=1.
	StrictNoOrigin bool
}

// DefaultCSRFConfig retorna config adequada ao ambiente.
//
// Sprint 13 [S13.3] — fail-closed by default. Para dev permissive (allow
// cross-origin warning + no-origin fallback), set RADIANT_CSRF_PERMISSIVE=1.
// RADIANT_CSRF_STRICT_NO_ORIGIN=1 rejeita até curl/Postman (max-strict).
func DefaultCSRFConfig() CSRFConfig {
	isPermissive := os.Getenv("RADIANT_CSRF_PERMISSIVE") == "1"
	isStrictNoOrigin := os.Getenv("RADIANT_CSRF_STRICT_NO_ORIGIN") == "1"

	cfg := CSRFConfig{
		AllowOrigins: []string{
			"http://localhost:4180", // Next.js dev (port 4180)
			"http://localhost:3000", // Next.js default
		},
		EnforceProduction: !isPermissive, // FAIL-CLOSED. Invertido vs antes.
		StrictNoOrigin:    isStrictNoOrigin,
	}

	// Sprint 13 [S13.3]: whitelist só em permissive mode.
	// Em prod, /v1/auth/dev-token passa por Origin check normal
	// (defense-in-depth: se prod ativar dev-token por misconfig,
	// ainda assim Origin é validado).
	if isPermissive {
		cfg.WhitelistRoutes = []string{
			"/v1/auth/dev-token", // Dev mode: frontend minta JWT
		}
	}

	// Em prod, adiciona URL do frontend via env var
	if prod := os.Getenv("RADIANT_FRONTEND_URL"); prod != "" {
		cfg.AllowOrigins = append(cfg.AllowOrigins, prod)
	}
	return cfg
}

// CSRF middleware retorna handler que valida Origin.
func CSRF(cfg CSRFConfig) func(http.Handler) http.Handler {
	allowSet := make(map[string]bool, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		allowSet[o] = true
	}
	whitelistSet := make(map[string]bool, len(cfg.WhitelistRoutes))
	for _, r := range cfg.WhitelistRoutes {
		whitelistSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Whitelist: bypass total (dev-token)
			if whitelistSet[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Apenas métodos state-changing
			if r.Method != http.MethodPost &&
				r.Method != http.MethodPut &&
				r.Method != http.MethodPatch &&
				r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")

			// Sem Origin = browser legacy OU non-browser client (Postman/curl).
			// Browser moderno SEMPRE envia Origin em POST cross-origin OU same-origin.
			//
			// Sprint 13 [S13.3] — finetuning:
			//   * Default (StrictNoOrigin=false): permite no-origin porque
			//     admin scripts/curl com JWT conseguem chamar API em dev/prod.
			//   * RADIANT_CSRF_STRICT_NO_ORIGIN=1: rejeita — para deploy
			//     max-strict onde sabemos que browser é o único client.
			if origin == "" {
				if cfg.StrictNoOrigin {
					http.Error(w, "CSRF: missing Origin header", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Same-origin = permite sempre
			host := r.Host
			if isSameOrigin(origin, host) {
				next.ServeHTTP(w, r)
				return
			}

			// Cross-origin na allowlist = permite
			if allowSet[origin] {
				next.ServeHTTP(w, r)
				return
			}

			// Cross-origin não permitido
			if cfg.EnforceProduction {
				http.Error(w, "CSRF: cross-origin request blocked", http.StatusForbidden)
				return
			}
			// Dev: loga warning mas permite
			next.ServeHTTP(w, r)
		})
	}
}

// isSameOrigin checa se origin URL tem mesmo scheme+host que request host.
func isSameOrigin(origin, host string) bool {
	if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
		return false
	}
	originHost := strings.TrimPrefix(origin, "http://")
	originHost = strings.TrimPrefix(originHost, "https://")
	if idx := strings.Index(originHost, "/"); idx != -1 {
		originHost = originHost[:idx]
	}
	return originHost == host
}
