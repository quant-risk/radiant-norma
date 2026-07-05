// Sprint 10 — handler SSE /v1/events/stream.
//
// Wrapper que delega ao Hub. Incluído em sprint8c_handlers.go pra
// manter todos os handlers novos juntos (consistência).

package api

import (
	"net/http"
)

// eventsStreamHandler expõe o Hub SSE como http.Handler.
// Auth vem do middleware global (acima); request context carrega IF.
// Filtro por IF acontece dentro do Hub (Publish ignora subscribers
// de outras IFs).
func (s *Server) eventsStreamHandler(w http.ResponseWriter, r *http.Request) {
	if s.EventsHub == nil {
		http.Error(w, "events hub não inicializado", http.StatusServiceUnavailable)
		return
	}
	s.EventsHub.ServeHTTP(w, r)
}