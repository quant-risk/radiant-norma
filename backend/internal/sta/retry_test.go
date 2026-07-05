// Package sta — tests do RetryingClient (Sprint 22).
//
// Cobre: classificação de erros (retryable vs permanente), backoff exponencial
// com jitter, integração com httptest.Server simulando BACEN 5xx transientes.
package sta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// captureRetry é uma captura simples para testes do callback OnRetry.
type captureRetry struct {
	attempts []int
	errs     []error
	backs    []time.Duration
}

// retryServer retorna um httptest.Server que responde com base no estado
// armazenado no pointer `state`. Cada call atômica incrementa `calls`.
// Quando `failTimes` > 0, retorna 503 com corpo XML; depois de `failTimes`
// sucessos, passa para o `successHandler`.
type retryServer struct {
	calls          int32
	failTimes      int32
	failStatus     int
	successHandler http.HandlerFunc
	mu             chan struct{} // serializa failTimes check
}

func (s *retryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := atomic.AddInt32(&s.calls, 1)

	// Validação: ainda dentro da janela de falhas?
	if n <= s.failTimes {
		w.WriteHeader(int(s.failStatus))
		_, _ = io.WriteString(w, fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>%d</Codigo><Descricao>transient failure %d</Descricao></Erro></Resultado>`, s.failStatus, n))
		return
	}
	s.successHandler(w, r)
}

// successSTAHandler retorna 201 Created (Fase 1 do Submit, manual §5.1.1) + XML.
// Também responde 200 OK para PUT de conteúdo (Fase 2).
func successSTAHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado><Protocolo>PROTO-OK</Protocolo></Resultado>`)
		case http.MethodPut:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
}

// newRetryingClientForTest monta um WSClient contra httptest.Server que
// responde conforme configurado, embrulhado em RetryingClient.
//
// Por que WSClient (não StubClient)? StubClient valida "submission sem
// XML nem ZIP" antes de retornar — não chama rede. Para exercitar o
// caminho de retry com httptest, precisamos de WSClient real.
func newRetryingClientForTest(t *testing.T, cfg RetryConfig, failTimes int32, failStatus int) (*httptest.Server, *RetryingClient, *WSClient) {
	t.Helper()

	srv := httptest.NewServer(&retryServer{
		failTimes:      failTimes,
		failStatus:     failStatus,
		successHandler: successSTAHandler(),
	})
	t.Cleanup(srv.Close)

	ws, err := NewWSClient(WSConfig{
		BaseURL:           srv.URL,
		User:              "12345/0001.fulano",
		Password:          "test-pwd",
		Timeout:           2 * time.Second,
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}

	rc, err := NewRetryingClient(ws, cfg)
	if err != nil {
		t.Fatalf("NewRetryingClient: %v", err)
	}
	return srv, rc, ws
}

// TestNewRetryingClient_Validacao — configs inválidas rejeitadas.
func TestNewRetryingClient_Validacao(t *testing.T) {
	stub := NewStubClient()
	tests := []struct {
		name    string
		cfg     RetryConfig
		wantErr string
	}{
		{"inner nil", RetryConfig{}, "inner client requerido"},
		{"MaxAttempts 0 usa default 3", RetryConfig{}, ""},
		{"MaxAttempts -1", RetryConfig{MaxAttempts: -1}, "MaxAttempts deve estar"},
		{"MaxAttempts 11", RetryConfig{MaxAttempts: 11}, "MaxAttempts deve estar"},
		{"Jitter 1.5", RetryConfig{Jitter: 1.5}, "Jitter deve estar"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Para testar inner nil, passa nil explicitamente.
			var inner Client = stub
			if tt.name == "inner nil" {
				inner = nil
			}
			_, err := NewRetryingClient(inner, tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("esperava sucesso, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava erro contendo %q", tt.wantErr)
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("erro deveria mencionar %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// sampleSubmissionForRetry retorna Submission válida (XML não-vazio).
// WSClient.Submit valida XML/ZIP não-vazio antes de chamar BACEN.
func sampleSubmissionForRetry() *Submission {
	return &Submission{
		CadocCode: "3040",
		DataBase:  "2024-12",
		CNPJ:      "demo-bank",
		XML:       "<root><Doc3040/></root>",
	}
}

// TestRetryingClient_SuccessFirstTry — BACEN sucesso primeira tentativa.
func TestRetryingClient_SuccessFirstTry(t *testing.T) {
	_, rc, _ := newRetryingClientForTest(t, RetryConfig{MaxAttempts: 3}, 0, 503)

	res, err := rc.Submit(context.Background(), sampleSubmissionForRetry())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !res.Accepted {
		t.Errorf("esperava Accepted=true, got %+v", res)
	}
	if res.ProtocolSTA != "PROTO-OK" {
		t.Errorf("ProtocolSTA = %q, esperado PROTO-OK", res.ProtocolSTA)
	}
}

// TestRetryingClient_503RetryThenSuccess — BACEN 503 2x, sucesso na 3ª.
func TestRetryingClient_503RetryThenSuccess(t *testing.T) {
	_, rc, _ := newRetryingClientForTest(t,
		RetryConfig{
			MaxAttempts:   3,
			BackoffBase:   1 * time.Millisecond, // acelera test
			BackoffFactor: 2.0,
			Jitter:        0,
		},
		2, 503) // 2 falhas 503, depois sucesso

	res, err := rc.Submit(context.Background(), sampleSubmissionForRetry())
	if err != nil {
		t.Fatalf("Submit após retries: %v", err)
	}
	if !res.Accepted {
		t.Errorf("esperava Accepted=true após 503 retries, got %+v", res)
	}
}

// TestRetryingClient_400NoRetry — BACEN 400, sem retry.
func TestRetryingClient_400NoRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>parâmetro inválido</Descricao></Erro></Resultado>`)
	}))
	t.Cleanup(srv.Close)

	rc, err := NewRetryingClient(NewStubClient(), RetryConfig{
		MaxAttempts: 3,
		BackoffBase: 1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewRetryingClient: %v", err)
	}

	// Como o StubClient NÃO usa o httptest server (ele tem lógica própria),
	// este test valida classificação de erro via stub + STAError gerado
	// diretamente. Vou trocar a abordagem: usar WSClient contra mock.
	_ = srv // unused

	// Em vez disso: testar via erro STAError mock injetado.
	t.Run("via shouldRetry", func(t *testing.T) {
		_, _, _ = newRetryingClientForTest(t, RetryConfig{MaxAttempts: 3}, 0, 503)
		// shouldRetry retorna false para 4xx
		staErr := &STAError{StatusCode: 400, Message: "param inválido"}
		retryable, _ := rc.shouldRetry(staErr, 1)
		if retryable {
			t.Error("4xx NÃO deveria fazer retry")
		}
	})
}

// TestRetryingClient_403NoRetry — via shouldRetry direto.
func TestRetryingClient_403NoRetry(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	staErr := &STAError{StatusCode: 403, Message: "não autorizado"}
	retryable, _ := rc.shouldRetry(staErr, 1)
	if retryable {
		t.Error("403 NÃO deveria fazer retry")
	}
}

// TestRetryingClient_404NoRetry — protocolo inexistente.
func TestRetryingClient_404NoRetry(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	staErr := &STAError{StatusCode: 404, Message: "não encontrado"}
	retryable, _ := rc.shouldRetry(staErr, 1)
	if retryable {
		t.Error("404 NÃO deveria fazer retry")
	}
}

// TestRetryingClient_416NoRetry — range inválido (Sprint 21).
func TestRetryingClient_416NoRetry(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	staErr := &STAError{StatusCode: 416, Message: "range inválido"}
	retryable, _ := rc.shouldRetry(staErr, 1)
	if retryable {
		t.Error("416 NÃO deveria fazer retry")
	}
}

// TestRetryingClient_5xxRetries — 500/502/503/504 → retry.
func TestRetryingClient_5xxRetries(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	for _, status := range []int{500, 502, 503, 504} {
		staErr := &STAError{StatusCode: status, Message: "transient"}
		retryable, backoff := rc.shouldRetry(staErr, 1)
		if !retryable {
			t.Errorf("status %d deveria fazer retry", status)
		}
		if backoff <= 0 {
			t.Errorf("status %d: backoff deveria ser > 0, got %v", status, backoff)
		}
	}
}

// TestRetryingClient_MaxAttemptsExhausted — sempre 503, retry N vezes, retorna erro final.
func TestRetryingClient_MaxAttemptsExhausted(t *testing.T) {
	_, rc, _ := newRetryingClientForTest(t,
		RetryConfig{
			MaxAttempts:   3,
			BackoffBase:   1 * time.Millisecond,
			BackoffFactor: 2.0,
			Jitter:        0,
		},
		99, 503) // 99 falhas (cobre todas as 3 tentativas)

	_, err := rc.Submit(context.Background(), sampleSubmissionForRetry())
	if err == nil {
		t.Fatal("esperava erro após MaxAttempts")
	}
	if !contains(err.Error(), "3 tentativas") {
		t.Errorf("erro deveria mencionar tentativas, got %v", err)
	}
}

// TestRetryingClient_NetworkErrorRetry — context.DeadlineExceeded → retry.
func TestRetryingClient_NetworkErrorRetry(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})

	// timeout error wrappado
	timeoutErr := &net.OpError{
		Op:  "dial",
		Err: timeoutError{}, // mock timeout
	}
	retryable, backoff := rc.shouldRetry(timeoutErr, 1)
	if !retryable {
		t.Error("net.OpError timeout deveria fazer retry")
	}
	if backoff <= 0 {
		t.Error("backoff deveria ser > 0")
	}
}

// timeoutError implementa net.Error com Timeout()=true.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

// TestRetryingClient_ContextCancel — ctx cancela durante sleep → retorna ctx.Err().
//
// Requer WSClient real contra httptest server (StubClient retorna Accepted=true
// imediato, não entra no caminho de retry).
func TestRetryingClient_ContextCancel(t *testing.T) {
	// Mock que SEMPRE retorna 503 — força retry sleep.
	_, rc, _ := newRetryingClientForTest(t,
		RetryConfig{
			MaxAttempts:   3,
			BackoffBase:   5 * time.Second, // sleep longo se não for interrompido
			BackoffFactor: 1.0,
			Jitter:        0,
		},
		99, 503)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := rc.Submit(ctx, sampleSubmissionForRetry())
	if err == nil {
		t.Fatal("esperava erro de context cancel")
	}
	if !contains(err.Error(), "cancelled") && !errors.Is(err, context.Canceled) {
		t.Errorf("erro deveria mencionar cancel, got %v", err)
	}
}

// TestRetryingClient_OnRetryCallback — callback invocado com params corretos.
func TestRetryingClient_OnRetryCallback(t *testing.T) {
	captured := &captureRetry{}
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{
		MaxAttempts:   3,
		BackoffBase:   1 * time.Millisecond,
		BackoffFactor: 2.0,
		Jitter:        0,
		OnRetry: func(attempt int, err error, nextBackoff time.Duration) {
			captured.attempts = append(captured.attempts, attempt)
			captured.errs = append(captured.errs, err)
			captured.backs = append(captured.backs, nextBackoff)
		},
	})
	_, rc, _ = newRetryingClientForTest(t,
		RetryConfig{
			MaxAttempts:   3,
			BackoffBase:   1 * time.Millisecond,
			BackoffFactor: 2.0,
			Jitter:        0,
			OnRetry: func(attempt int, err error, nextBackoff time.Duration) {
				captured.attempts = append(captured.attempts, attempt)
				captured.errs = append(captured.errs, err)
				captured.backs = append(captured.backs, nextBackoff)
			},
		},
		2, 503)

	_, err := rc.Submit(context.Background(), sampleSubmissionForRetry())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if len(captured.attempts) != 2 {
		t.Errorf("OnRetry deveria ter sido chamado 2x (entre 3 tentativas), got %d", len(captured.attempts))
	}
	if captured.attempts[0] != 1 {
		t.Errorf("primeiro callback attempt=%d, esperado 1", captured.attempts[0])
	}
	if captured.attempts[1] != 2 {
		t.Errorf("segundo callback attempt=%d, esperado 2", captured.attempts[1])
	}
}

// TestShouldRetry_HashMismatch — ErrContentHashMismatch NÃO retryable.
func TestShouldRetry_HashMismatch(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	wrapped := fmt.Errorf("wrap: %w", ErrContentHashMismatch)
	retryable, _ := rc.shouldRetry(wrapped, 1)
	if retryable {
		t.Error("ErrContentHashMismatch NÃO deveria fazer retry (corrupção)")
	}
}

// TestShouldRetry_HeaderMalformed — ErrContentHashHeaderMalformed NÃO retryable.
func TestShouldRetry_HeaderMalformed(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{MaxAttempts: 3})
	wrapped := fmt.Errorf("wrap: %w", ErrContentHashHeaderMalformed)
	retryable, _ := rc.shouldRetry(wrapped, 1)
	if retryable {
		t.Error("ErrContentHashHeaderMalformed NÃO deveria fazer retry (formato mudou)")
	}
}

// TestRetryingClient_BackoffTiming — backoff exponencial (sem jitter).
func TestRetryingClient_BackoffTiming(t *testing.T) {
	rc, _ := NewRetryingClient(NewStubClient(), RetryConfig{
		MaxAttempts:   5,
		BackoffBase:   100 * time.Millisecond,
		BackoffFactor: 2.0,
		Jitter:        0,
	})
	// attempt=1 → 100ms, attempt=2 → 200ms, attempt=3 → 400ms
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
	}
	for _, tt := range tests {
		got := rc.computeBackoff(tt.attempt)
		if got != tt.want {
			t.Errorf("attempt %d: got %v, esperado %v", tt.attempt, got, tt.want)
		}
	}
}

// TestSleepWithContext_Cancel — sleep interrompido por ctx.Done().
func TestSleepWithContext_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	err := sleepWithContext(ctx, 5*time.Second)
	elapsed := time.Since(start)
	if err == nil {
		t.Error("sleep deveria ter retornado erro")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("erro deveria ser context.Canceled, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("sleep não foi interrompido cedo (elapsed=%v)", elapsed)
	}
}

// TestSleepWithContext_Done — sleep completa sem cancel.
func TestSleepWithContext_Done(t *testing.T) {
	err := sleepWithContext(context.Background(), 10*time.Millisecond)
	if err != nil {
		t.Errorf("sleep deveria completar nil, got %v", err)
	}
}

// TestIsNetworkError — cobre os 5 caminhos.
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"context.DeadlineExceeded", context.DeadlineExceeded, true},
		{"context.Canceled", context.Canceled, false},
		{"net.Error timeout", timeoutError{}, true},
		{"connection refused (string)", errors.New("dial tcp: connection refused"), false}, // Não wrappeado em url.Error
		{"regular error", errors.New("some other error"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkError(tt.err); got != tt.want {
				t.Errorf("isNetworkError(%v) = %v, esperado %v", tt.err, got, tt.want)
			}
		})
	}
}

// contains é helper local (strings.Contains é muito verboso pra testes).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > 0 && len(substr) > 0 && sFind(s, substr)))
}

// sFind evita import extra de strings. Implementação naive (single-pass).
func sFind(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// _ para evitar unused import warnings em builds que removem algum dos testes.
var _ = slog.Default
var _ = http.MethodGet
var _ atomic.Int32
