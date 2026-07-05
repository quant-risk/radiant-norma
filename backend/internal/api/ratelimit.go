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
// Estratégia: in-memory token bucket por (method, path-bucket, ifID).
// Path-bucket = categoria derivada do path (HEAVY/MUTATE/READ/EXPORT).
// Sem bucket compartilhado entre IFs (1 IF não afeta outra).
//
// Em prod com múltiplas réplicas API, in-memory NÃO é suficiente
// (cada réplica tem contador próprio). Para produção real, integrar
// Redis com INCR+EXPIRE. Pattern aqui é compatível: substituir
// `Allow(key)` por Redis EVAL "INCR ... EXPIRE ...".

package api

import (
	"net/http"
	"strconv"
	"sync"
	"time"
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
const MaxKeysRateLimiter = 10_000

// apiRateLimiter é rate limiter in-memory por chave (bucket+ifID).
//
// Política: máximo `limit.max` calls por `limit.window` por ifID dentro
// de um bucket. Max keys: MaxKeysRateLimiter. LRU eviction simples.
//
// Thread-safe: sync.Mutex serializa Allow().
// Performance: O(N) por call onde N = calls na janela (range + count).
// Para escala Radiant Norma atual (<100 IFs piloto), suficiente.
type apiRateLimiter struct {
	mu       sync.Mutex
	calls    map[string][]time.Time // key=bucket+":"+ifID
	limits   map[pathBucket]RateLimit
	lastEvict map[string]time.Time // for LRU on overflow
}

// newAPIRateLimiter cria limiter com policy padrão.
func newAPIRateLimiter() *apiRateLimiter {
	r := &apiRateLimiter{
		calls:     make(map[string][]time.Time),
		limits:    DefaultRateLimits,
		lastEvict: make(map[string]time.Time),
	}
	return r
}

// Allow checa se uma call pelo bucket+ifID é permitida. Retorna
// (allowed, retryAfter).
func (r *apiRateLimiter) Allow(bucket pathBucket, ifID string) (bool, time.Duration) {
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
func (r *apiRateLimiter) evictOldestLocked() {
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
func rateLimitMiddleware(limiter *apiRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ignora health/ready.
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
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
			allowed, retryAfter := limiter.Allow(bucket, ifID)
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
