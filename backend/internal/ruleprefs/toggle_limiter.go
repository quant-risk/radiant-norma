// Package ruleprefs — ToggleLimiter: rate limiter para toggle de regras.
//
// Sprint 12 (v3.5.0) — C32.22: rate limit por IFID para /v1/rules/{code}/toggle.
// Atacante autenticado pode DoS o sistema com toggles spam (cada um gera
// DB write + audit + SSE event). Política: 10 toggles por minuto por IF.
//
// Pattern reusado de radar.ScanLimiter — in-memory map, sync.Mutex.
// Em produção com muitas IFs (> 1000), considerar Redis. Para escala
// atual do Radiant Norma (< 100 IFs piloto), in-memory é OK.

package ruleprefs

import (
	"sync"
	"time"
)

// ToggleLimiter é rate limiter in-memory por chave (IF).
//
// Política: máximo `maxPerWindow` toggles por `window` (default 10/min).
// Map é unbounded — em produção com muitas IFs, considerar eviction LRU.
type ToggleLimiter struct {
	mu          sync.Mutex
	calls       map[string][]time.Time // key=if_id, value=call timestamps
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
// Retorna (true, 0) se permitido e registra o call.
// Retorna (false, retryAfter) se rate-limited, com tempo até poder tentar.
//
// Algoritmo: sliding window. Mantém lista de timestamps de calls na
// janela. Se len < max, permite e append. Se len == max, verifica se
// o mais antigo está fora da janela — se sim, drop e append; senão
// bloqueia.
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
	return true, 0
}

// Reset limpa estado (usado em testes).
func (l *ToggleLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = make(map[string][]time.Time)
}