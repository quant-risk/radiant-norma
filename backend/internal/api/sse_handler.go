// Sprint 10 — handler SSE /v1/events/stream.
//
// Wrapper que delega ao Hub. Antes de delegar, popula context com
// IFID lido do Claims JWT (pra evitar import cycle: realtime não pode
// importar auth).

package api

import (
	"context"
	"net/http"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
)

// ifIDContextKey é a chave usada pra injetar IFID no context
// antes do Hub ServeHTTP. Realtime package lê essa chave.
//
// Mantida como raw string pra casar com hub.getIfID (que lê
// r.Context().Value("if_id") com string-typed key).
const ifIDContextKey = "if_id"

// eventsStreamHandler expõe o Hub SSE como http.Handler.
// Auth vem do middleware global (acima); request context carrega IF.
// Filtro por IF acontece dentro do Hub (Publish ignora subscribers
// de outras IFs).
//
// Se Claims existirem no context (auth middleware populou), IFID é
// extraído e injetado em context com chave ifIDContextKey. Caso contrário,
// fallback X-IF-ID header (dev mode).
func (s *Server) eventsStreamHandler(w http.ResponseWriter, r *http.Request) {
	if s.EventsHub == nil {
		http.Error(w, "events hub não inicializado", http.StatusServiceUnavailable)
		return
	}

	// Resolve IFID do JWT claims (preferred) ou X-IF-ID fallback.
	ifID := ""
	if claims, err := auth.ClaimsFromContext(r.Context()); err == nil && claims != nil {
		ifID = claims.IFID
	}
	if ifID == "" {
		ifID = r.Header.Get("X-IF-ID")
	}

	// Injeta IFID no context (raw string key, matches hub.getIfID).
	ctx := context.WithValue(r.Context(), ifIDContextKey, ifID)
	r = r.WithContext(ctx)

	s.EventsHub.ServeHTTP(w, r)
}