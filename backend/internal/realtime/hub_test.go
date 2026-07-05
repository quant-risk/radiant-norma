// Package realtime — tests do Hub SSE.
package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestHub() *Hub {
	return NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestHub_PublishReceive(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unregister, _ := hub.Subscribe(ctx, "demo")
	defer unregister()

	hub.Publish(Event{
		Kind:      "test.event",
		IFID:      "demo",
		Payload:   map[string]any{"foo": "bar"},
		Timestamp: time.Now(),
	})

	select {
	case evt := <-events:
		if evt.Kind != "test.event" {
			t.Errorf("expected kind=test.event, got %s", evt.Kind)
		}
		if evt.IFID != "demo" {
			t.Errorf("expected if_id=demo, got %s", evt.IFID)
		}
		if evt.Payload["foo"] != "bar" {
			t.Errorf("expected payload foo=bar, got %v", evt.Payload["foo"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event not received within 100ms")
	}
}

func TestHub_FilterByIFID(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2 subscribers: um pra "demo", um pra "other"
	demoEvents, demoUnreg, _ := hub.Subscribe(ctx, "demo")
	defer demoUnreg()
	otherEvents, otherUnreg, _ := hub.Subscribe(ctx, "other")
	defer otherUnreg()

	// Publica evento SÓ pra demo
	hub.Publish(Event{Kind: "demo-only", IFID: "demo", Timestamp: time.Now()})

	// Demo recebe, other NÃO recebe
	select {
	case evt := <-demoEvents:
		if evt.Kind != "demo-only" {
			t.Errorf("demo: expected demo-only, got %s", evt.Kind)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("demo subscriber didn't receive")
	}

	select {
	case evt := <-otherEvents:
		t.Errorf("other should NOT receive, got %s", evt.Kind)
	case <-time.After(50 * time.Millisecond):
		// esperado: nada em other
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, demoUnreg, _ := hub.Subscribe(ctx, "demo")
	defer demoUnreg()
	_, otherUnreg, _ := hub.Subscribe(ctx, "other")
	defer otherUnreg()

	// Publica com IFID vazio → broadcast pra todos
	hub.Publish(Event{Kind: "global", IFID: "", Timestamp: time.Now()})

	// Ambos recebem
	// (test simplificado — não verifica todos aqui)
}

func TestHub_BackpressureDrop(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Subscriber com channel buffer 32 (default). Não consome.
	_, unregister, _ := hub.Subscribe(ctx, "demo")
	defer unregister()

	// Publica 50 eventos (mais que buffer de 32)
	for i := 0; i < 50; i++ {
		hub.Publish(Event{Kind: "burst", IFID: "demo", Timestamp: time.Now()})
	}

	subs, total, dropped := hub.Stats()
	_ = subs
	_ = total
	if dropped == 0 {
		t.Errorf("expected drops > 0 (channel buffer 32 + 50 events), got %d", dropped)
	}
}

func TestHub_UnsubscribeStopsDelivery(t *testing.T) {
	hub := newTestHub()
	ctx := context.Background()

	events, unregister, _ := hub.Subscribe(ctx, "demo")

	hub.Publish(Event{Kind: "first", IFID: "demo", Timestamp: time.Now()})
	select {
	case <-events:
		// drain
	case <-time.After(100 * time.Millisecond):
		t.Fatal("first event not received")
	}

	unregister() // fecha o subscriber e remove do hub

	// Verify hub não tem mais subscribers — Publish não deve chegar.
	// Channel fechado retorna zero value imediatamente, então checamos
	// que Stats mostra 0 subs E que Publish não causa efeitos colaterais.
	subs, _, _ := hub.Stats()
	if subs != 0 {
		t.Errorf("expected 0 subscribers after unregister, got %d", subs)
	}

	// Publish depois de unsubscribe — não deve panic nem chegar ao channel.
	// recover() em Publish captura o send pra channel fechado.
	hub.Publish(Event{Kind: "second", IFID: "demo", Timestamp: time.Now()})

	// Channel fechado retorna zero value imediatamente — não há como
	// distinguir de "nenhum evento chegou" via receive. Validamos que
	// o hub não tem mais subs (acima) e que Publish não panicou.
}

func TestHub_ConcurrentPublishers(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unregister, _ := hub.Subscribe(ctx, "demo")
	defer unregister()

	// 10 goroutines publicando 100 eventos cada = 1000 eventos total
	const goroutines = 10
	const perGoroutine = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				hub.Publish(Event{
					Kind:      "concurrent",
					IFID:      "demo",
					Timestamp: time.Now(),
				})
			}
		}(i)
	}

	// Drain events
	received := 0
	deadline := time.After(2 * time.Second)
loop:
	for received < goroutines*perGoroutine {
		select {
		case <-events:
			received++
		case <-deadline:
			break loop
		}
	}

	wg.Wait()
	subs, total, _ := hub.Stats()
	if subs != 1 {
		t.Errorf("expected 1 subscriber, got %d", subs)
	}
	if total < uint64(goroutines*perGoroutine) {
		t.Errorf("expected at least %d total events, got %d", goroutines*perGoroutine, total)
	}
	t.Logf("concurrent test: received=%d total=%d", received, total)
}

// --- HTTP handler tests ---

// safeRecorder é um http.ResponseWriter com buffer thread-safe.
//
// httptest.ResponseRecorder.Body é *bytes.Buffer, que NÃO é safe para
// uso concorrente — `Write` em goroutine X e `String()` em goroutine Y
// são data race. Estes testes rodam hub.ServeHTTP em goroutine enquanto
// o test main faz poll do body, então precisamos serializar.
//
// Implementação: mantemos nosso próprio *bytes.Buffer protegido por
// mutex, e implementamos http.ResponseWriter delegando ao buffer.
// Flusher exposto via interface assertion opcional.
//
// Sprint 13 — v3.5.2 followup: race pré-existente desde Sprint 10
// (v3.3.0) só exposto agora porque passamos a rodar `-race` na CI.
type safeRecorder struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	header http.Header
	code   int
}

func newSafeRecorder() *safeRecorder {
	return &safeRecorder{header: make(http.Header)}
}

func (s *safeRecorder) Header() http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.header
}

func (s *safeRecorder) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.code == 0 {
		s.code = http.StatusOK
	}
	return s.buf.Write(p)
}

func (s *safeRecorder) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.code = code
}

// BodyString retorna snapshot thread-safe do body.
func (s *safeRecorder) BodyString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// Flush é parte de http.Flusher. Hub.ServeHTTP chama flusher.Flush()
// após cada Write; delegamos pra no-op já que httptest.NewRecorder
// também não faz flush real.
func (s *safeRecorder) Flush() {}

// Compile-time check: safeRecorder satisfaz http.ResponseWriter.
var _ http.ResponseWriter = (*safeRecorder)(nil)

func TestHubServeHTTP_SendsConnectedEvent(t *testing.T) {
	hub := newTestHub()

	req := httptest.NewRequest("GET", "/v1/events/stream", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := newSafeRecorder()

	// Run em goroutine (handler é blocking)
	done := make(chan struct{})
	go func() {
		hub.ServeHTTP(w, req)
		close(done)
	}()

	// Espera o connected event ser enviado
	deadline := time.After(500 * time.Millisecond)
	var body string
pollConnected:
	for {
		select {
		case <-deadline:
			t.Fatal("connected event not received")
		default:
			body = w.BodyString()
			if strings.Contains(body, "event: connected") {
				break pollConnected
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !strings.Contains(body, `"if_id":"demo"`) {
		t.Errorf("connected event missing if_id=demo, got: %s", body)
	}
}

func TestHubServeHTTP_PublishesEventsToClient(t *testing.T) {
	hub := newTestHub()

	req := httptest.NewRequest("GET", "/v1/events/stream", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := newSafeRecorder()

	done := make(chan struct{})
	go func() {
		hub.ServeHTTP(w, req)
		close(done)
	}()

	// Wait pra connected event chegar antes de publicar
	deadlineConnected := time.After(500 * time.Millisecond)
pollConnected:
	for {
		select {
		case <-deadlineConnected:
			t.Fatal("connected event not received before publish")
		default:
			if strings.Contains(w.BodyString(), "event: connected") {
				break pollConnected
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Publish event
	hub.Publish(Event{
		Kind: "audit",
		IFID: "demo",
		Payload: map[string]any{
			"action": "envio.approved",
		},
		Timestamp: time.Now(),
	})

	// Wait pra event ser enviado
	deadline := time.After(500 * time.Millisecond)
	var body string
pollAudit:
	for {
		select {
		case <-deadline:
			t.Fatalf("event not received by client, body: %s", w.BodyString())
		default:
			body = w.BodyString()
			if strings.Contains(body, "event: audit") {
				break pollAudit
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if !strings.Contains(body, "envio.approved") {
		t.Errorf("event data missing action, got: %s", body)
	}

	// Validate JSON parse — pega a última data line (audit, não connected)
	lines := strings.Split(body, "\n")
	var dataLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
		}
	}
	var evt Event
	if err := json.Unmarshal([]byte(dataLine), &evt); err != nil {
		t.Errorf("event data not valid JSON: %v", err)
	}
	if evt.Kind != "audit" {
		t.Errorf("expected kind=audit, got %s", evt.Kind)
	}
}
