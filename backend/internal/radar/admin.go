// Package radar — DOS-via-API hardening (R1, Sprint 6 v1.5.0).
//
// Componentes:
//   - ScanLimiter: rate limit 1 scan/min por IF (token bucket simplificado)
//   - ScanCache: cache de último resultado por 5min (evita refazer HTTP fetch)
//   - AdminAuth: autenticação admin (header X-Admin ou Bearer token) —
//     protege contra triggerRadarScan ser usado como vetor de DOS
//
// Background (VALIDATION_v1.4.2.md R1):
//   POST /v1/radar/scan chama ScanOnce(ctx, nil) → 3 HTTP requests pra
//   bc.gov.br. Sem rate limit + sem auth admin = vetor de DOS contra BACEN
//   + inchaço da tabela radar_alerts + auditlog hash chain attack vector.
//
// Security stance: FAIL CLOSED. Se ADMIN_TOKEN não está configurada, o
// endpoint retorna 503 "admin auth não configurada". Melhor que fail open
// (autoriza qualquer um) em produção.
package radar

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// ============================
// ScanLimiter
// ============================

// ScanLimiter é um rate limiter in-memory por chave (IF).
//
// Política: máximo 1 scan por cooldown (1 minuto default).
// Map é unbounded — em produção com muitas IFs, considerar eviction LRU.
// Para o spike atual (< 100 IFs), OK.
type ScanLimiter struct {
	mu       sync.Mutex
	lastCall map[string]time.Time
	cooldown time.Duration
}

// NewScanLimiter cria limiter com cooldown dado.
func NewScanLimiter(cooldown time.Duration) *ScanLimiter {
	return &ScanLimiter{
		lastCall: make(map[string]time.Time),
		cooldown: cooldown,
	}
}

// Allow checa se a chave pode fazer scan agora.
//
// Retorna (true, 0) se permitido e atualiza lastCall.
// Retorna (false, retryAfter) se bloqueado, com tempo até poder tentar.
func (l *ScanLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	last, ok := l.lastCall[key]
	if ok {
		elapsed := now.Sub(last)
		if elapsed < l.cooldown {
			return false, l.cooldown - elapsed
		}
	}
	l.lastCall[key] = now
	return true, 0
}

// Reset limpa estado (usado em testes).
func (l *ScanLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lastCall = make(map[string]time.Time)
}

// ============================
// ScanCache
// ============================

// ScanCache guarda o último resultado de ScanOnce para evitar refazer
// HTTP fetches em chamadas repetidas dentro do TTL.
//
// Em produção: trocar para Redis (in-memory não escala horizontalmente).
type ScanCache struct {
	mu         sync.Mutex
	alerts     []Alert
	scannedAt  time.Time
	ttl        time.Duration
}

// NewScanCache cria cache com TTL.
func NewScanCache(ttl time.Duration) *ScanCache {
	return &ScanCache{ttl: ttl}
}

// Get retorna alerts se cache válido, senão (nil, false).
//
// Sempre retorna slice (não nil) quando válido, pra JSON serializar [].
func (c *ScanCache) Get() ([]Alert, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.alerts == nil || time.Since(c.scannedAt) > c.ttl {
		return nil, false
	}
	// Copia para evitar race com mutações externas
	out := make([]Alert, len(c.alerts))
	copy(out, c.alerts)
	return out, true
}

// Put armazena alerts no cache.
func (c *ScanCache) Put(alerts []Alert) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alerts = make([]Alert, len(alerts))
	copy(c.alerts, alerts)
	c.scannedAt = time.Now()
}

// Invalidate limpa cache (usado em testes ou após mudanças externas).
func (c *ScanCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alerts = nil
	c.scannedAt = time.Time{}
}

// ScannedAt retorna timestamp do último scan cacheado (para telemetria).
func (c *ScanCache) ScannedAt() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.scannedAt
}

// ============================
// AdminAuth
// ============================

// AdminAuth valida que request tem permissão admin.
//
// Política FAIL CLOSED: se Token está vazio, IsAdmin retorna false.
// Em produção, cmd/api/main.go carrega ADMIN_TOKEN de env var.
//
// Aceita 2 formas:
//   1. Header X-Admin: <token>
//   2. Authorization: Bearer <token>
//
// Comparação constant-time (segurança contra timing attack).
type AdminAuth struct {
	Token string
}

// IsAdmin retorna true se request tem token admin válido.
func (a *AdminAuth) IsAdmin(r *http.Request) bool {
	if a.Token == "" {
		return false // fail closed
	}
	// Header direto
	if h := r.Header.Get("X-Admin"); h != "" {
		return constantTimeEqual(h, a.Token)
	}
	// Bearer token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return constantTimeEqual(auth[7:], a.Token)
	}
	return false
}

// Challenge retorna header WWW-Authenticate quando auth falha.
func (a *AdminAuth) Challenge() string {
	return `{"error":"admin token required (X-Admin header or Authorization: Bearer)"}`
}

// constantTimeEqual compara 2 strings em tempo constante (evita timing attack).
// Retorna true se iguais.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
