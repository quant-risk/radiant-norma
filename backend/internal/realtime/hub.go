// Package realtime — Server-Sent Events (SSE) hub para push.
//
// Sprint 10: user pediu "real-time via SSE — alertas chegam sem F5,
// activity feed ao vivo". Implementação:
//
//   1. Hub em memória com pub/sub (cada Subscribe retorna channel).
//   2. Endpoint SSE /v1/events/stream mantém conexão HTTP aberta e
//      envia cada evento como `event: <kind>\ndata: <json>\n\n`.
//   3. AuditLogger é decorado com HubAwareLogger que publica cada
//      evento no hub depois de gravar no DB (mantém LGPD/SOC2 chain).
//
// Concurrency: Hub usa sync.RWMutex + channels buffered. Subscriber
// fecha automaticamente quando client desconecta (request context).
//
// Backpressure: se subscriber não consegue consumir (channel cheio),
// recebe evento "dropped" e counter incrementa. Nunca bloqueia publisher.

package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Event representa 1 evento a ser enviado via SSE.
type Event struct {
	Kind      string         `json:"kind"` // "audit", "radar.detected", etc
	IFID      string         `json:"if_id,omitempty"`
	Payload   map[string]any `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
}

// subscriber é 1 cliente SSE conectado. Channel buffered pra absorver
// picos curtos; se encher, evento é dropado (logged).
type subscriber struct {
	id     string
	ch     chan Event
	ifID   string
	ctx    context.Context
	atomic.Bool
}

// Hub gerencia todos os subscribers ativos.
type Hub struct {
	mu          sync.RWMutex
	subs        map[*subscriber]struct{}
	totalEvents atomic.Uint64
	dropped     atomic.Uint64
	logger      *slog.Logger
}

// NewHub cria hub vazio.
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		subs:   make(map[*subscriber]struct{}),
		logger: logger,
	}
}

// Subscribe registra novo subscriber. Retorna channel que recebe eventos
// + função Unregister pra cleanup.
//
// Channel tem buffer 32 — se subscriber demorar mais que isso pra consumir,
// eventos são dropados (logged) e contador incrementa.
func (h *Hub) Subscribe(ctx context.Context, ifID string) (<-chan Event, func()) {
	sub := &subscriber{
		id:   randomID(),
		ch:   make(chan Event, 32),
		ifID: ifID,
		ctx:  ctx,
	}

	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()

	h.logger.Info("sse subscriber added", "sub_id", sub.id, "if_id", ifID, "total", len(h.subs))

	unregister := func() {
		h.mu.Lock()
		if _, ok := h.subs[sub]; ok {
			delete(h.subs, sub)
			close(sub.ch)
		}
		h.mu.Unlock()
		h.logger.Info("sse subscriber removed", "sub_id", sub.id, "total", len(h.subs))
	}

	// Auto-unregister quando context cancelar (client disconnect)
	go func() {
		<-ctx.Done()
		unregister()
	}()

	return sub.ch, unregister
}

// Publish envia evento pra todos os subscribers relevantes.
// Se ifID != "", apenas subscribers com mesmo ifID recebem.
// Se ifID == "", broadcast pra todos.
//
// Thread-safety: Publish segura h.mu.RLock pra snapshot dos subscribers,
// depois itera sem lock. Send pra channel fechado é detectado via
// recover() — channel fechado significa subscriber desconectou entre
// snapshot e send, e evento é silently dropped.
func (h *Hub) Publish(evt Event) {
	h.totalEvents.Add(1)

	h.mu.RLock()
	subs := make([]*subscriber, 0, len(h.subs))
	for sub := range h.subs {
		if evt.IFID == "" || sub.ifID == "" || sub.ifID == evt.IFID {
			subs = append(subs, sub)
		}
	}
	h.mu.RUnlock()

	for _, sub := range subs {
		func() {
			defer func() {
				// Channel fechado = subscriber desconectou entre snapshot e send.
				// Silently drop. Panic recovery é a única forma segura de detectar
				// isso em Go (select { case ch <- x: } panics em closed channel).
				_ = recover()
			}()
			select {
			case sub.ch <- evt:
				// ok
			default:
				// Channel cheio: drop. Log + counter pra observability.
				h.dropped.Add(1)
				h.logger.Warn("sse subscriber channel full — event dropped",
					"sub_id", sub.id,
					"event_kind", evt.Kind,
					"dropped_total", h.dropped.Load())
			}
		}()
	}
}

// Stats retorna métricas pra health-check / observability.
func (h *Hub) Stats() (subs int, totalEvents, dropped uint64) {
	h.mu.RLock()
	subs = len(h.subs)
	h.mu.RUnlock()
	return subs, h.totalEvents.Load(), h.dropped.Load()
}

// --- HTTP handler ---

// ServeHTTP implementa http.Handler para SSE stream.
//
// Endpoint: GET /v1/events/stream
// Auth: mesma auth do resto (JWT/X-IF-ID via middleware global).
// Headers:
//   - Content-Type: text/event-stream
//   - Cache-Control: no-cache
//   - Connection: keep-alive
//   - X-Accel-Buffering: no (desabilita buffering em nginx-like)
//
// Cada evento: "event: <kind>\ndata: <json>\n\n". Heartbeat a cada 30s
// ("event: heartbeat\ndata: {}\n\n") pra manter conexão viva em NATs.
//
// IFID resolution (em ordem):
//   1. Header X-IF-ID (dev fallback / direct header)
//   2. Context value key "if_id" (populado pelo api/sse_handler.go
//      lendo Claims do JWT middleware — evita import cycle)
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ifID := getIfID(r)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	events, unregister := h.Subscribe(r.Context(), ifID)
	defer unregister()

	// Envia evento "connected" inicial pra confirmar stream
	_, _ = w.Write([]byte("event: connected\ndata: {\"if_id\":\"" + ifID + "\"}\n\n"))
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	notif := r.Context().Done()
	for {
		select {
		case <-notif:
			// Client desconectou — unregister via defer
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				h.logger.Warn("sse marshal failed", "err", err, "kind", evt.Kind)
				continue
			}
			_, _ = w.Write([]byte("event: " + evt.Kind + "\ndata: "))
			_, _ = w.Write(data)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		case <-ticker.C:
			// Heartbeat: comment frame (SSE specification allows comments
			// starting with ':' to keep connection alive without data).
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}

// --- helpers ---

// getIfID lê IF do contexto (injetado pelo api/sse_handler.go) ou
// X-IF-ID fallback (dev mode).
//
// Recebe string key "if_id" — context key type é opaque (string-typed
// no caller). Isso evita import cycle (realtime não importa auth).
func getIfID(r *http.Request) string {
	if v := r.Context().Value("if_id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return r.Header.Get("X-IF-ID")
}

// randomID gera identificador único (não precisa ser cryptographically
// seguro — só pra logs/observability).
var counter atomic.Uint64

func randomID() string {
	return formatUint(counter.Add(1))
}

func formatUint(n uint64) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = chars[n%36]
		n /= 36
	}
	return string(out)
}