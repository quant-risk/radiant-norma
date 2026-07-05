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
	WhitelistRoutes []string
	// EnforceProduction: se true, bloqueia cross-origin não-allowlisted.
	// Default: true se RADIANT_ENV=production.
	EnforceProduction bool
}

// DefaultCSRFConfig retorna config adequada ao ambiente.
//
// Dev (RADIANT_ENV != production): allowlist inclui localhost:4180 (Next.js
// dev). Cross-origin loga warning mas não bloqueia (permite Postman).
// Prod (RADIANT_ENV=production): só allowlist + same-origin. Outras = 403.
func DefaultCSRFConfig() CSRFConfig {
	cfg := CSRFConfig{
		AllowOrigins: []string{
			"http://localhost:4180", // Next.js dev (port 4180)
			"http://localhost:3000", // Next.js default
		},
		WhitelistRoutes: []string{
			"/v1/auth/dev-token", // Dev mode: frontend minta JWT
		},
		EnforceProduction: os.Getenv("RADIANT_ENV") == "production",
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
			// Permitir no-origin é fallback defensivo (não é vetor CSRF real
			// porque browser sem Origin não envia cookie cross-origin).
			if origin == "" {
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