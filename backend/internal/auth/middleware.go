// Package auth — chi middleware para JWT bearer + X-IF-ID fallback.
//
// Substitui authMiddleware em server.go. Política:
//
//   - Default: exige JWT válido no header `Authorization: Bearer <token>`.
//   - Dev mode (env RADIANT_DEV_AUTH=1): aceita X-IF-ID como fallback
//     para migration de clients.
//   - Errors → 401 com mensagem genérica via UserError.
package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

// now() helper para tests determinísticos. Production usa time.Now() direto.
var now = time.Now

type ctxKey string

const (
	// ctxClaimsKey é a chave para o *Claims no context.
	ctxClaimsKey ctxKey = "auth.claims"
	// devAuthFlag é o env var que ativa X-IF-ID fallback.
	devAuthFlag = "RADIANT_DEV_AUTH"
)

// Middleware extrai JWT do header e popula context com Claims.
//
// Requisitos:
//   - Header `Authorization: Bearer <token>`
//   - Token válido (assinatura, issuer, expiry)
//   - Claim role apropriada para endpoint (role check é no handler)
//
// Em dev mode (RADIANT_DEV_AUTH=1), aceita X-IF-ID no header como
// fallback. Dev mode emite warning nos logs (TODO: emitir via logger).
func Middleware(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Tenta JWT primeiro.
			if v != nil {
				token := extractBearer(r.Header.Get("Authorization"))
				if token != "" {
					claims, err := v.Verify(token)
					if err == nil {
						ctx := context.WithValue(r.Context(), ctxClaimsKey, claims)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
					// JWT presente mas inválido → 401 direto (não tentar
					// X-IF-ID fallback).
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
			}

			// 2. Fallback X-IF-ID em dev mode.
			if os.Getenv(devAuthFlag) == "1" {
				ifID := r.Header.Get("X-IF-ID")
				if ifID != "" {
					claims := &Claims{
						Sub:  ifID,
						IFID: ifID,
						Role: RoleIF,
						Iss:  "dev",
						Iat:  now(),
						Exp:  now().Add(1 * time.Hour),
					}
					ctx := context.WithValue(r.Context(), ctxClaimsKey, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 3. Sem creds — 401.
			http.Error(w, "authentication required", http.StatusUnauthorized)
		})
	}
}

// extractBearer extrai token do header Authorization. Suporta também
// token passado via header `Authorization: <token>` (sem prefixo) para
// dev convenience.
func extractBearer(h string) string {
	if h == "" {
		return ""
	}
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if strings.HasPrefix(h, "bearer ") {
		return strings.TrimPrefix(h, "bearer ")
	}
	return h // dev mode sem prefix
}

// ClaimsFromContext retorna Claims do context (populado por middleware).
// Retorna erro se não houver — caller decide se 401 ou 403.
func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	c, ok := ctx.Value(ctxClaimsKey).(*Claims)
	if !ok {
		return nil, errors.New("auth: claims não encontradas no context")
	}
	return c, nil
}
