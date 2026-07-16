// Package api — rate limiter middleware.
//
// Sprint 13 — v3.5.2 [S15.1]: defense em profundidade contra DoS-via-API
// authenticated. Audit S-B/H [HIGH]: validate, sta/submit, crossdoc,
// envios?format=csv, audit_log não tinham rate limit. Atacante
// autenticado podia:
//   * hammer /v1/validate (10MB body parse por call)
//   * hammer /v1/sta/submit (XML parse + audit + DB write)
//   * hammer /v1/crossdoc/validate (N CADOCs × regras × goroutines)
//   * exfiltrar audit_log inteiro via many concurrent CSV exports
//
// Estratégia: token bucket por (method, path-bucket, ifID).
// Path-bucket = categoria derivada do path (HEAVY/MUTATE/READ/EXPORT).
// Sem bucket compartilhado entre IFs (1 IF não afeta outra).
//
// Sprint 16 — v3.6.0 [S16.1]: backend plugável.
//   - memoryRateLimiter: in-memory, single-replica. Default em dev/test.
//   - redisRateLimiter: distribuído via Redis INCR+EXPIRE. Default em prod
//     multi-replica. Ver ratelimit_redis.go.
//
// Seleção via env RADIANT_RATE_LIMIT_BACKEND=memory|redis (default memory).
// Redis URL via RADIANT_REDIS_URL=redis://host:port/db.

package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Erros retornados por newRateLimiterFromEnv.
var (
	errRedisURLRequired        = errors.New("RADIANT_RATE_LIMIT_BACKEND=redis requer RADIANT_REDIS_URL")
	errUnknownRateLimitBackend = errors.New("RADIANT_RATE_LIMIT_BACKEND deve ser 'memory' ou 'redis'")
)

// Path bucket categoriza endpoints para ter limite apropriado.
type pathBucket string

const (
	bucketHeavy  pathBucket = "heavy"  // validate, sta/submit, crossdoc
	bucketMutate pathBucket = "mutate" // toggle, ack, resolve
	bucketRead   pathBucket = "read"   // GETs padrões
	bucketExport pathBucket = "export" // ?format=csv em envios/audit_log
	bucketAuth   pathBucket = "auth"   // login, dev-token
)

// RateLimiter é a interface pública do limiter.
//
// Implementações: memoryRateLimiter (in-memory, single-replica) e
// redisRateLimiter (distribuído, multi-replica). Server.RateLimiter é
// deste tipo — middleware funciona igual independente do backend.
//
// Retorna (allowed, retryAfter):
//   - allowed=true: request passa
//   - allowed=false: request bloqueado; retryAfter = tempo até próximo slot
type RateLimiter interface {
	Allow(bucket pathBucket, ifID string) (allowed bool, retryAfter time.Duration)
	// Backend retorna nome do backend ("memory" ou "redis") para logging.
	Backend() string
}

// DefaultRateLimits é a política de rate limit por bucket.
//
// Cada IF tem 1 bucket por categoria (não 1 bucket por endpoint) — evita
// explosion de keys se cada endpoint tiver bucket próprio, e ainda
// protege DoS. Atacante authenticado que hammer /v1/validate + /v1/sta/
// submit compartilha o bucket `heavy`.
var DefaultRateLimits = map[pathBucket]RateLimit{
	bucketHeavy:  {Max: 10, Window: time.Minute},
	bucketMutate: {Max: 30, Window: time.Minute},
	bucketRead:   {Max: 100, Window: time.Minute},
	bucketExport: {Max: 5, Window: time.Minute},
	bucketAuth:   {Max: 30, Window: 5 * time.Minute},
}

// RateLimit define política: max N por window por (bucket, ifID).
type RateLimit struct {
	Max    int
	Window time.Duration
}

// MaxKeysRateLimiter é o limite de keys distintos antes de LRU eviction.
//
// Mitiga DoS via fake if_ids infinitos (audit S-B / C34.11).
// Só aplicável ao backend in-memory — Redis tem seu próprio TTL.
const MaxKeysRateLimiter = 10_000

// memoryRateLimiter é rate limiter in-memory por chave (bucket+ifID).
//
// Política: máximo `limit.max` calls por `limit.window` por ifID dentro
// de um bucket. Max keys: MaxKeysRateLimiter. LRU eviction simples.
//
// Thread-safe: sync.Mutex serializa Allow().
// Performance: O(N) por call onde N = calls na janela (range + count).
// Para escala Radiant Norma atual (<100 IFs piloto), suficiente.
//
// Em prod com múltiplas réplicas API, contadores NÃO são compartilhados
// — cada réplica tem seu próprio map. Para prod multi-replica, use
// redisRateLimiter (RADIANT_RATE_LIMIT_BACKEND=redis).
type memoryRateLimiter struct {
	mu        sync.Mutex
	calls     map[string][]time.Time // key=bucket+":"+ifID
	limits    map[pathBucket]RateLimit
	lastEvict map[string]time.Time // for LRU on overflow
}

// newMemoryRateLimiter cria limiter in-memory com policy padrão.
func newMemoryRateLimiter() *memoryRateLimiter {
	return &memoryRateLimiter{
		calls:     make(map[string][]time.Time),
		limits:    DefaultRateLimits,
		lastEvict: make(map[string]time.Time),
	}
}

// Allow checa se uma call pelo bucket+ifID é permitida.
func (r *memoryRateLimiter) Allow(bucket pathBucket, ifID string) (bool, time.Duration) {
	limit, ok := r.limits[bucket]
	if !ok {
		return true, 0 // bucket desconhecido = passa (fail-open só em dev)
	}

	key := string(bucket) + ":" + ifID
	now := time.Now()
	cutoff := now.Add(-limit.Window)

	r.mu.Lock()
	defer r.mu.Unlock()

	// LRU eviction se número de keys explode (proteção DoS via fake if_ids).
	if len(r.calls) >= MaxKeysRateLimiter {
		if _, exists := r.calls[key]; !exists {
			r.evictOldestLocked()
		}
	}

	timestamps := r.calls[key]
	// Corta timestamps fora da janela (in-place).
	filtered := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	timestamps = filtered

	if len(timestamps) >= limit.Max {
		// Próximo slot libera no tempo (timestamps[0] + window).
		retryAfter := timestamps[0].Add(limit.Window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		r.calls[key] = timestamps
		r.lastEvict[key] = now
		return false, retryAfter
	}

	timestamps = append(timestamps, now)
	r.calls[key] = timestamps
	r.lastEvict[key] = now
	return true, 0
}

// evictOldestLocked remove key com lastEvict mais antigo. Requer lock.
func (r *memoryRateLimiter) evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, t := range r.lastEvict {
		if oldestKey == "" || t.Before(oldestTime) {
			oldestKey = k
			oldestTime = t
		}
	}
	if oldestKey != "" {
		delete(r.calls, oldestKey)
		delete(r.lastEvict, oldestKey)
	}
}

// Backend retorna nome do backend para logging.
func (r *memoryRateLimiter) Backend() string { return "memory" }

// bucketForPath classifica path em bucket para rate limit.
func bucketForPath(method, path string, isExport bool) pathBucket {
	// Export via query tem precedência: ?format=csv exporta tudo.
	if isExport {
		return bucketExport
	}

	// Auth endpoints.
	if path == "/v1/auth/dev-token" || path == "/api/login" {
		return bucketAuth
	}

	// Heavy: validate, sta/submit, crossdoc/validate.
	if path == "/v1/validate" || path == "/v1/sta/submit" || path == "/v1/crossdoc/validate" {
		return bucketHeavy
	}

	// Mutate: toggle, ack, resolve, radar/scan.
	if method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete {
		return bucketMutate
	}

	return bucketRead
}

// rateLimitMiddleware é chi middleware que aplica rate limiter por IFID.
//
// Sprint 17 — v3.7.0 [S17.5]: métricas opcionais via Metrics param.
// Se metrics == nil, middleware funciona normalmente (sem incrementar).
// Caller (NewServer) decide se quer métricas.
func rateLimitMiddleware(limiter RateLimiter, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ignora health/ready + metrics + status endpoint (evita ruído).
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" || r.URL.Path == "/metrics" || r.URL.Path == "/status" {
				next.ServeHTTP(w, r)
				return
			}

			// Extrai IFID (claims > header) — mesmo helper do CSRF/handler.
			ifID := getIfID(r)
			if ifID == "" {
				next.ServeHTTP(w, r)
				return
			}

			isExport := r.URL.Query().Get("format") != ""
			bucket := bucketForPath(r.Method, r.URL.Path, isExport)
			backend := limiter.Backend()
			allowed, retryAfter := limiter.Allow(bucket, ifID)
			if metrics != nil {
				if allowed {
					metrics.IncAllowed(string(bucket), backend)
				} else {
					metrics.IncDropped(string(bucket), backend)
				}
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
				w.Header().Set("X-RateLimit-Bucket", string(bucket))
				http.Error(w,
					`{"error":"rate limit exceeded","bucket":"`+string(bucket)+
						`","retry_after_seconds":`+strconv.Itoa(int(retryAfter.Seconds())+1)+`}`,
					http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NewRateLimiterFromEnv cria limiter baseado em RADIANT_RATE_LIMIT_BACKEND.
//
//   - "memory" (default): in-memory, single-replica. Dev/test/CI.
//   - "redis": distribuído via Redis. Prod multi-replica.
//     Requer RADIANT_REDIS_URL=redis://host:port/db.
//
// Para backend redis, opcionalmente RADIANT_RATE_LIMIT_WINDOW=fixed|sliding:
//   - "fixed" (default): INCR+EXPIRE. Simples, burstiness na borda.
//   - "sliding": sorted set Lua. Preciso, sem burstiness. +CPU/+mem.
//
// Em prod sem Redis configurado, retorna erro — fail-closed evita
// deploy silencioso em modo "memory" (contadores não compartilhados).
func NewRateLimiterFromEnv() (RateLimiter, error) {
	backend := os.Getenv("RADIANT_RATE_LIMIT_BACKEND")
	if backend == "" || backend == "memory" {
		return newMemoryRateLimiter(), nil
	}
	if backend == "redis" {
		redisURL := os.Getenv("RADIANT_REDIS_URL")
		if redisURL == "" {
			return nil, errRedisURLRequired
		}
		rl, err := newRedisRateLimiter(redisURL, DefaultRateLimits)
		if err != nil {
			return nil, err
		}
		// Window type (Sprint 17 — v3.7.0 — S17.3)
		switch os.Getenv("RADIANT_RATE_LIMIT_WINDOW") {
		case "", "fixed":
			rl.WindowType = WindowTypeFixed
		case "sliding":
			rl.WindowType = WindowTypeSliding
		default:
			return nil, fmt.Errorf("RADIANT_RATE_LIMIT_WINDOW deve ser 'fixed' ou 'sliding'")
		}
		return rl, nil
	}
	return nil, errUnknownRateLimitBackend
}
