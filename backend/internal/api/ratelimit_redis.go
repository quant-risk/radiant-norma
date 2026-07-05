// Package api — Redis-backed rate limiter.
//
// Sprint 16 — v3.6.0 [S16.1]: produção multi-replica.
// In-memory limiter (ver ratelimit.go) é single-replica — cada API
// replica tem seu próprio contador, atacante autenticado pode
// distribuir carga entre réplicas e bypass do limite.
//
// Redis-backed usa INCR + EXPIRE atômicos via Lua script. Cada call
// incrementa um counter com TTL = window; quando TTL expira, counter
// é resetado. Sliding window é aproximado — atômico o suficiente
// pra rate limiting (burst nas bordas é aceitável, comportamento
// equivalente ao in-memory).
//
// Fail-closed strategy: se Redis está down, Allow() retorna erro.
// Caller (rateLimitMiddleware) interpreta erro como allow (fail-open)
// porque API sem rate limit ainda é preferível a API totalmente fora.
// Log do erro é responsabilidade do caller.
//
// Configuração via env:
//   RADIANT_RATE_LIMIT_BACKEND=redis
//   RADIANT_REDIS_URL=redis://host:port/db

package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// LuaIncrWithTTL é o script Lua que faz INCR + EXPIRE atomicamente.
// Retorna [current_count, ttl_seconds].
//
// FIXED WINDOW (Sprint 16). Simples mas tem burstiness na borda do
// window — cliente pode fazer 2× Max calls se distribuir entre o final
// de um window e o início do próximo. Aceitável pra DoS prevention.
//
// Por que Lua? Garante atomicidade entre INCR e EXPIRE — sem script,
// existe race onde key é INCR mas EXPIRE falha, key fica sem TTL.
// Script roda no servidor Redis em single-thread, sem race.
//
// Exportado porque tests (smoke + ratelimit_test) precisam injetar em
// RedisRateLimiter custom (ex: miniredis) sem ter que parsear string.
const LuaIncrWithTTL = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('TTL', KEYS[1])
return {current, ttl}
`

// LuaSlidingWindow é o script Lua que implementa sliding window via
// sorted set. Cada call adiciona entry com score=timestamp_ms; entries
// mais velhas que (now - window) são removidas. Count = ZCARD.
//
// SLIDING WINDOW (Sprint 17 — v3.7.0 — S17.3). Mais preciso que fixed
// window — sem burstiness na borda. Custo: +memória (sorted set cresce
// com Max por bucket) + +CPU (ZREMRANGEBYSCORE + ZCARD + ZADD por call).
//
// Retorna [allowed (0|1), oldest_ms_or_zero].
//   - allowed=1, oldest=0: request passou
//   - allowed=0, oldest>0: bloqueado; oldest = ms timestamp do call
//     mais antigo na window (pra calcular retry-after preciso)
//
// KEYS[1] = bucket:ifid
// ARGV[1] = max (int)
// ARGV[2] = window_seconds (int)
// ARGV[3] = member (string único pra evitar collision no sorted set)
const LuaSlidingWindow = `
local now_arr = redis.call('TIME')
local now_ms = tonumber(now_arr[1]) * 1000 + math.floor(tonumber(now_arr[2]) / 1000)
local window_ms = tonumber(ARGV[2]) * 1000
local cutoff = now_ms - window_ms
local max_count = tonumber(ARGV[1])

-- Remove entries older than window
redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, cutoff)

-- Count current entries in window
local count = redis.call('ZCARD', KEYS[1])

if count >= max_count then
    -- Get oldest score for retry-after computation
    local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
    local oldest_ms = tonumber(oldest[2])
    return {0, oldest_ms}
end

-- Allow: add current timestamp to set with unique member
redis.call('ZADD', KEYS[1], now_ms, ARGV[3])
-- TTL = window + 1s buffer (cleanup se cliente parar de chamar)
redis.call('PEXPIRE', KEYS[1], window_ms + 1000)
return {1, 0}
`

// RedisRateLimiter implementa RateLimiter usando Redis como store.
//
// Thread-safe: go-redis.Client é thread-safe. Script pré-carregado
// (redis.NewScript) usa EVALSHA cache, evitando reenvio do script.
//
// Exportado (capital R) porque main.go precisa chamar Close() no shutdown
// e tests precisam injetar client próprio (ex: miniredis).
//
// Sprint 17 — v3.7.0 [S17.5]: campo Metrics opcional. Quando setado,
// IncFailOpen é chamado quando Redis retorna erro (fail-open path).
//
// Sprint 17 — v3.7.0 [S17.3]: campo WindowType seleciona algoritmo.
// "fixed" (default, INCR+EXPIRE) ou "sliding" (sorted set).
type RedisRateLimiter struct {
	Client        *redis.Client
	Limits        map[pathBucket]RateLimit
	Script        *redis.Script // fixed window (default)
	SlidingScript *redis.Script // sliding window (Sprint 17)
	Logger        *slog.Logger
	KeyPrefix     string
	Metrics       *Metrics // opcional; nil = sem instrumentação
	WindowType    WindowType

	// memberCounter gera member único para sorted set (sliding window).
	// Atomic pra thread-safety entre goroutines concorrentes.
	memberCounter atomic.Uint64
}

// WindowType define algoritmo de janela do Redis limiter.
// "fixed" (Sprint 16 — v3.6.0): INCR+EXPIRE, simples.
// "sliding" (Sprint 17 — v3.7.0): sorted set Lua, sem burstiness.
type WindowType string

const (
	WindowTypeFixed   WindowType = "fixed"
	WindowTypeSliding WindowType = "sliding"
)

// newRedisRateLimiter cria limiter conectado ao Redis em redisURL.
//
// Limiter NÃO toca Redis no construtor — primeira chamada de Allow()
// estabelece conexão. Isso permite que API faça boot sem Redis
// disponível (útil pra startup em ambiente com race em deps).
//
// Caller é responsável por validar redisURL antes (factory
// newRateLimiterFromEnv já exige RADIANT_REDIS_URL não-vazio).
//
// Validação de limits: Window deve ser ≥1s. Redis EXPIRE aceita
// apenas segundos inteiros — Window <1s seria truncado para 0
// (key expira imediatamente, contadores resetam a cada call).
// Production usa janelas ≥1min (DefaultRateLimits), então é defesa
// contra misuse futuro.
func newRedisRateLimiter(redisURL string, limits map[pathBucket]RateLimit) (*RedisRateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url %q: %w", redisURL, err)
	}
	if err := validateRedisLimits(limits); err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)
	return &RedisRateLimiter{
		Client:        client,
		Limits:        limits,
		Script:        redis.NewScript(LuaIncrWithTTL),
		SlidingScript: redis.NewScript(LuaSlidingWindow),
		Logger:        slog.Default(),
		KeyPrefix:     "rl:",
		WindowType:    WindowTypeFixed, // default; altere via SetWindowType
	}, nil
}

// validateRedisLimits garante que cada bucket tem Window ≥1s.
// Redis EXPIRE aceita segundos inteiros; <1s truncado para 0 é
// comportamento perigoso (key expira antes de ser usado).
func validateRedisLimits(limits map[pathBucket]RateLimit) error {
	const minWindow = time.Second
	for bucket, lim := range limits {
		if lim.Window < minWindow {
			return fmt.Errorf("bucket %q: window %v < 1s mínimo (Redis EXPIRE aceita segundos inteiros)",
				bucket, lim.Window)
		}
		if lim.Max <= 0 {
			return fmt.Errorf("bucket %q: max %d deve ser > 0", bucket, lim.Max)
		}
	}
	return nil
}

// Allow checa rate limit via Redis Lua script.
//
// Comportamento:
//   - Redis OK + dentro do limite: retorna (true, 0)
//   - Redis OK + excedeu limite: retorna (false, retry-after)
//   - Redis indisponível: loga warning + retorna (true, 0) (fail-open)
//
// Fail-open em Redis down é decisão consciente: API sem rate limit é
// melhor que API totalmente fora. Em prod, monitoring deve flagar
// Allow() errors como alerta (log estruturado com "redis_rate_limit_err").
//
// Algoritmo controlado por WindowType (Sprint 17 — v3.7.0 — S17.3):
//   - "fixed" (default): INCR+EXPIRE. retry-after = TTL restante.
//   - "sliding": sorted set Lua. retry-after = tempo até oldest call
//     sair da window (mais preciso, sem burstiness na borda).
func (r *RedisRateLimiter) Allow(bucket pathBucket, ifID string) (bool, time.Duration) {
	limit, ok := r.Limits[bucket]
	if !ok {
		return true, 0 // bucket desconhecido = passa
	}

	key := r.KeyPrefix + string(bucket) + ":" + ifID
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Seleciona script baseado em WindowType.
	var res interface{}
	var err error
	switch r.WindowType {
	case WindowTypeSliding:
		// Member único por call pra evitar collision no sorted set
		// (ZADD com mesmo score+member é no-op).
		member := fmt.Sprintf("%d-%s-%s-%d", time.Now().UnixNano(), string(bucket), ifID, r.memberCounter.Add(1))
		res, err = r.SlidingScript.Run(ctx, r.Client, []string{key},
			limit.Max, int(limit.Window.Seconds()), member).Result()
	default: // WindowTypeFixed
		res, err = r.Script.Run(ctx, r.Client, []string{key}, int(limit.Window.Seconds())).Result()
	}

	if err != nil {
		r.Logger.Warn("redis rate limit failed (fail-open)", "err", err, "bucket", bucket, "if_id", ifID)
		if r.Metrics != nil {
			r.Metrics.IncFailOpen("redis")
			r.Metrics.SetBackendUp(false)
		}
		return true, 0
	}

	if r.Metrics != nil {
		r.Metrics.SetBackendUp(true)
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) != 2 {
		r.Logger.Warn("redis rate limit unexpected result (fail-open)", "res", res)
		if r.Metrics != nil {
			r.Metrics.IncFailOpen("redis")
		}
		return true, 0
	}

	allowedFlag, _ := arr[0].(int64)

	switch r.WindowType {
	case WindowTypeSliding:
		// Lua returns {1, 0} allowed ou {0, oldest_ms} denied.
		if allowedFlag == 0 {
			oldestMS, _ := arr[1].(int64)
			// retry-after = (oldest_ms + window_ms) - now_ms
			nowMS := time.Now().UnixMilli()
			windowMS := limit.Window.Milliseconds()
			retryMS := (oldestMS + windowMS) - nowMS
			if retryMS < 0 {
				retryMS = 0
			}
			retryAfter := time.Duration(retryMS) * time.Millisecond
			if retryAfter < time.Second {
				retryAfter = time.Second
			}
			return false, retryAfter
		}
		return true, 0
	default: // WindowTypeFixed
		// Lua returns {count, ttl_seconds}. count > Max → denied.
		if allowedFlag > int64(limit.Max) {
			ttl, _ := arr[1].(int64)
			retryAfter := time.Duration(ttl) * time.Second
			if retryAfter < time.Second {
				retryAfter = time.Second
			}
			return false, retryAfter
		}
		return true, 0
	}
}

// Close fecha conexão com Redis. Chame em defer no main.go shutdown.
func (r *RedisRateLimiter) Close() error {
	return r.Client.Close()
}

// Backend retorna nome do backend para logging.
func (r *RedisRateLimiter) Backend() string { return "redis" }

// Compile-time check: RedisRateLimiter implementa RateLimiter.
var _ RateLimiter = (*RedisRateLimiter)(nil)

// suppress unused-import warning for errors in case future code needs it.
var _ = errors.New
