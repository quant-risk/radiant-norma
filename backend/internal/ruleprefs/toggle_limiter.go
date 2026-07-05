// Package ruleprefs — ToggleLimiter: rate limiter para toggle de regras.
//
// Sprint 12 (v3.5.0) — C32.22: rate limit por IFID para /v1/rules/{code}/toggle.
// Atacante autenticado pode DoS o sistema com toggles spam (cada um gera
// DB write + audit + SSE event). Política: 10 toggles por minuto por IF.
//
// Pattern reusado de radar.ScanLimiter — in-memory map, sync.Mutex.
// Em produção com muitas IFs (> 1000), considerar Redis. Para escala
// atual do Radiant Norma (< 100 IFs piloto), in-memory é OK.
//
// Sprint 12 v3.5.1 — C34.11: max keys (10k) + log warning quando excede
// + LRU eviction simples. DoS protection contra fake if_ids infinitos.

package ruleprefs

import (
	"log/slog"
	"sync"
	"time"
)

// MaxKeysToggleLimiter é o limite de keys distintos antes de LRU eviction.
const MaxKeysToggleLimiter = 10_000

// ToggleLimiter é rate limiter in-memory por chave (IF).
//
// Política: máximo `maxPerWindow` toggles por `window` (default 10/min).
// Max keys: MaxKeysToggleLimiter (10k). Se exceder, drop keys mais antigos.
type ToggleLimiter struct {
	mu           sync.Mutex
	calls        map[string][]time.Time // key=if_id, value=call timestamps
	maxPerWindow int
	window       time.Duration
}

// NewToggleLimiter cria limiter com janela e max calls.
func NewToggleLimiter(maxPerWindow int, window time.Duration) *ToggleLimiter {
	return &ToggleLimiter{
		calls:        make(map[string][]time.Time),
		maxPerWindow: maxPerWindow,
		window:       window,
	}
}

// Allow checa se a chave pode fazer toggle agora.
//
// C34.11: ao inserir nova key quando map > MaxKeysToggleLimiter, faz
// LRU eviction (drop keys mais antigos com calls mais antigos).
func (l *ToggleLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	calls := l.calls[key]

	// Drop entries fora da janela
	cutoff := now.Add(-l.window)
	filtered := calls[:0]
	for _, t := range calls {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	calls = filtered

	if len(calls) >= l.maxPerWindow {
		// Calcula retry_after = (oldest + window) - now
		oldest := calls[0]
		retryAfter := oldest.Add(l.window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		l.calls[key] = calls
		return false, retryAfter
	}

	calls = append(calls, now)
	l.calls[key] = calls

	// C34.11: se atingiu max keys E é primeira vez desta key, faz LRU eviction.
	if len(l.calls) > MaxKeysToggleLimiter {
		l.evictOldest()
	}

	return true, 0
}

// evictOldest remove as keys mais antigas (call mais antigo) até ficar
// dentro do limite. Roda sob lock.
func (l *ToggleLimiter) evictOldest() {
	for len(l.calls) > MaxKeysToggleLimiter {
		var oldestKey string
		var oldestTime time.Time
		first := true
		for k, calls := range l.calls {
			if len(calls) == 0 {
				// Empty entry — pode remover direto
				delete(l.calls, k)
				continue
			}
			if first || calls[0].Before(oldestTime) {
				oldestKey = k
				oldestTime = calls[0]
				first = false
			}
		}
		if oldestKey == "" {
			break // all empty, deleted some
		}
		delete(l.calls, oldestKey)
	}
	slog.Default().Warn("ToggleLimiter exceeded max keys — LRU eviction triggered",
		"current_size", len(l.calls),
		"max_keys", MaxKeysToggleLimiter)
}

// Stats retorna métricas pra observability.
func (l *ToggleLimiter) Stats() (size int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.calls)
}

// Reset limpa estado (usado em testes).
func (l *ToggleLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = make(map[string][]time.Time)
}
