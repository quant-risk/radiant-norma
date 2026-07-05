// Package sta — RetryingClient wrapper.
//
// Sprint 22 (v3.12.0): adiciona retry exponencial em erros transientes
// (5xx + timeout/erros de rede). Erros 4xx + X-Content-Hash mismatch
// NÃO fazem retry (caller bug / corrupção permanente — não adianta retry).
//
// Worker retry (Sprint 6 v1.5.0) é camada separada — entre envios no DB.
// Client retry é dentro de cada chamada HTTP. Worker + client retry são
// complementares.
//
// Referência: SPRINT_22_RESEARCH.md §2 (decisões de design).
package sta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// RetryConfig configura o RetryingClient.
type RetryConfig struct {
	// MaxAttempts é o número MÁXIMO de tentativas (incluindo a primeira).
	// Default 3 (1 inicial + 2 retries). Validado: 1 <= MaxAttempts <= 10.
	MaxAttempts int

	// BackoffBase é o tempo de espera base entre tentativas.
	// Default 1s. Backoff exponencial: BackoffBase * 2^(attempt-1).
	BackoffBase time.Duration

	// BackoffFactor controla a taxa de crescimento do backoff.
	// Default 2.0 (exponencial: 1s, 2s, 4s, 8s, ...).
	BackoffFactor float64

	// Jitter randomiza o backoff em ±Jitter (0..1).
	// Default 0.5 (±50% — evita thundering herd). 0 desabilita jitter.
	Jitter float64

	// Logger é o logger estruturado para emissão de retry attempts.
	// Opcional. Default slog.Default().
	Logger *slog.Logger

	// OnRetry é callback opcional invocado antes de cada sleep de retry.
	// Use para audit_log emission ou métrica Prometheus.
	// Deve ser leve — não fazer I/O pesado (DB write, HTTP call).
	// Parâmetros: attempt (1-indexed, após primeira falha), err (causa), nextBackoff (sleep planejado).
	OnRetry func(attempt int, err error, nextBackoff time.Duration)
}

// RetryingClient wrappea um Client adicionando retry exponencial em
// erros transientes.
//
// Implementa Client — caller substitui inner por RetryingClient onde antes
// passava inner. Drop-in replacement.
//
// Thread-safe: rng é protegido por mutex. Submit pode ser chamado de
// múltiplas goroutines simultaneamente (batch worker paralelo, por ex.).
type RetryingClient struct {
	inner Client
	cfg   RetryConfig
	rngMu sync.Mutex
	rng   *rand.Rand
}

// NewRetryingClient valida config e constrói wrapper. Retorna erro descritivo
// se cfg inválida (sem fazer call de rede).
func NewRetryingClient(inner Client, cfg RetryConfig) (*RetryingClient, error) {
	if inner == nil {
		return nil, errors.New("inner client requerido (nil)")
	}
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.MaxAttempts < 1 || cfg.MaxAttempts > 10 {
		return nil, fmt.Errorf("MaxAttempts deve estar em [1, 10] (got %d)", cfg.MaxAttempts)
	}
	if cfg.BackoffBase == 0 {
		cfg.BackoffBase = 1 * time.Second
	}
	if cfg.BackoffFactor == 0 {
		cfg.BackoffFactor = 2.0
	}
	if cfg.Jitter == 0 {
		// Default 0.5 = ±50%. Escolha comum em sistemas distribuídos
		// (Sufficient randomization without excessive spread).
		cfg.Jitter = 0.5
	}
	if cfg.Jitter < 0 || cfg.Jitter > 1 {
		return nil, fmt.Errorf("Jitter deve estar em [0, 1] (got %f)", cfg.Jitter)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	// rng seeded com UnixNano() — não-cryptographic, suficiente para jitter.
	return &RetryingClient{
		inner: inner,
		cfg:   cfg,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

// Submit wrappea inner.Submit com retry exponencial.
//
// Loop: tenta inner.Submit; se sucesso ou err não-retryable, retorna imediato.
// Se retryable, calcula backoff (com jitter), chama OnRetry callback,
// aguarda (respeitando ctx.Done()), tenta de novo.
//
// Limite: cfg.MaxAttempts. Se exceder, retorna último erro wrappeado.
//
// NÃO wrappea inner.Submit que retorna Result com Accepted=false —
// Rejection indica falha de upload (BACEN rejeitou), não é erro transient.
// Caller inspeciona Result.Rejection.
func (r *RetryingClient) Submit(ctx context.Context, sub *Submission) (*Result, error) {
	if sub == nil {
		return nil, errors.New("Submission não pode ser nil")
	}

	var lastErr error
	for attempt := 1; attempt <= r.cfg.MaxAttempts; attempt++ {
		result, err := r.inner.Submit(ctx, sub)
		if err == nil {
			// Sucesso. Rejection (se houver) NÃO dispara retry —
			// caller inspeciona result.Rejection.
			return result, nil
		}

		lastErr = err

		// Última tentativa — não adianta retry.
		if attempt >= r.cfg.MaxAttempts {
			break
		}

		// Erro retryable?
		retryable, backoff := r.shouldRetry(err, attempt)
		if !retryable {
			// Erro permanente (4xx, hash mismatch, etc.) — caller bug.
			return nil, err
		}

		// Aplica jitter.
		backoff = r.applyJitter(backoff)

		r.cfg.Logger.Warn("STA submit retry",
			"attempt", attempt,
			"next_attempt", attempt+1,
			"backoff", backoff,
			"err", err,
		)

		// Callback opcional (audit_log, métrica).
		if r.cfg.OnRetry != nil {
			r.cfg.OnRetry(attempt, err, backoff)
		}

		// Aguarda próximo attempt. Respeita ctx.Done() — se caller
		// cancelar (timeout, shutdown), retorna imediatamente com ctx.Err().
		if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
			return nil, fmt.Errorf("STA submit cancelled during retry sleep (attempt %d): %w", attempt, sleepErr)
		}
	}

	// Todas as tentativas falharam. Retorna último erro wrappeado.
	return nil, fmt.Errorf("STA submit falhou após %d tentativas: %w", r.cfg.MaxAttempts, lastErr)
}

// shouldRetry classifica err como retryable ou não.
//
// Retorna (true, backoffDuration) se retry, (false, 0) caso contrário.
//
// Regras:
//   - 5xx (500, 502, 503, 504) → retry (BACEN transiente).
//   - 4xx → não retry (caller bug, permanente).
//   - X-Content-Hash mismatch (Sprint 19) → não retry (corrupção).
//   - Timeout / network error → retry.
//   - Outros → não retry (conservador).
func (r *RetryingClient) shouldRetry(err error, attempt int) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	// Sentinel errors da Sprint 19: corrupção de integridade não é transient.
	if errors.Is(err, ErrContentHashMismatch) || errors.Is(err, ErrContentHashHeaderMalformed) {
		return false, 0
	}

	// STAError tipado — classifica por status code.
	var staErr *STAError
	if errors.As(err, &staErr) {
		if staErr.StatusCode >= 500 && staErr.StatusCode <= 599 {
			backoff := r.computeBackoff(attempt)
			return true, backoff
		}
		// 4xx e outros não-5xx → não retry.
		return false, 0
	}

	// Network errors / timeout → retry.
	if isNetworkError(err) {
		backoff := r.computeBackoff(attempt)
		return true, backoff
	}

	// Conservador: erros desconhecidos não fazem retry.
	return false, 0
}

// isNetworkError detecta timeouts + connection errors.
//
// Cobre:
//   - context.DeadlineExceeded (timeout do request)
//   - net.Error com Timeout() == true
//   - net.OpError com classe de erro de rede
//   - Connection refused/reset (url.Error wrapping connection errors)
func isNetworkError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		// context.Canceled NÃO é transient — caller cancelou.
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// url.Error wrappa connection errors em net package.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// url.Error.Timeout() delega para netErr.Timeout() quando wrappado.
		if urlErr.Timeout() {
			return true
		}
		// Connection refused/reset — sem flag Timeout(), mas string match.
		s := urlErr.Err.Error()
		if strings.Contains(s, "connection refused") ||
			strings.Contains(s, "connection reset") ||
			strings.Contains(s, "broken pipe") ||
			strings.Contains(s, "no such host") {
			return true
		}
	}
	return false
}

// computeBackoff retorna BackoffBase × BackoffFactor^(attempt-1).
// attempt é 1-indexed: attempt=1 → BackoffBase, attempt=2 → 2×Base, etc.
func (r *RetryingClient) computeBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	mult := math.Pow(r.cfg.BackoffFactor, float64(attempt-1))
	return time.Duration(float64(r.cfg.BackoffBase) * mult)
}

// applyJitter randomiza backoff em ±Jitter (0..1).
// BackoffBase=1s, Jitter=0.5 → [500ms, 1500ms].
//
// Thread-safe via rngMu (rand.Rand não é thread-safe).
func (r *RetryingClient) applyJitter(backoff time.Duration) time.Duration {
	if r.cfg.Jitter == 0 {
		return backoff
	}
	r.rngMu.Lock()
	j := (r.rng.Float64()*2 - 1) * r.cfg.Jitter
	r.rngMu.Unlock()
	jittered := float64(backoff) * (1 + j)
	if jittered < 0 {
		jittered = 0
	}
	return time.Duration(jittered)
}

// sleepWithContext aguarda d ou até ctx.Done().
// Retorna ctx.Err() se ctx cancelar antes de d expirar; nil caso contrário.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats helpers — expostos para test/debug.

// MaxAttempts retorna o número máximo de tentativas configuradas.
func (r *RetryingClient) MaxAttempts() int { return r.cfg.MaxAttempts }

// Inner retorna o client wrappeado (para debug).
func (r *RetryingClient) Inner() Client { return r.inner }

// Compile-time guarantee: *RetryingClient implementa sta.Client.
// Permite drop-in replacement sem erro de compilação silencioso.
var _ Client = (*RetryingClient)(nil)
