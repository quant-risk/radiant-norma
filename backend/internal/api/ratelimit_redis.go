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
	"time"

	"github.com/redis/go-redis/v9"
)

// LuaIncrWithTTL é o script Lua que faz INCR + EXPIRE atomicamente.
// Retorna [current_count, ttl_seconds].
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

// RedisRateLimiter implementa RateLimiter usando Redis como store.
//
// Thread-safe: go-redis.Client é thread-safe. Script pré-carregado
// (redis.NewScript) usa EVALSHA cache, evitando reenvio do script.
//
// Exportado (capital R) porque main.go precisa chamar Close() no shutdown
// e tests precisam injetar client próprio (ex: miniredis).
type RedisRateLimiter struct {
	Client    *redis.Client
	Limits    map[pathBucket]RateLimit
	Script    *redis.Script
	Logger    *slog.Logger
	KeyPrefix string
}

// newRedisRateLimiter cria limiter conectado ao Redis em redisURL.
//
// Limiter NÃO toca Redis no construtor — primeira chamada de Allow()
// estabelece conexão. Isso permite que API faça boot sem Redis
// disponível (útil pra startup em ambiente com race em deps).
//
// Caller é responsável por validar redisURL antes (factory
// newRateLimiterFromEnv já exige RADIANT_REDIS_URL não-vazio).
func newRedisRateLimiter(redisURL string, limits map[pathBucket]RateLimit) (*RedisRateLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url %q: %w", redisURL, err)
	}
	client := redis.NewClient(opts)
	return &RedisRateLimiter{
		Client:    client,
		Limits:    limits,
		Script:    redis.NewScript(LuaIncrWithTTL),
		Logger:    slog.Default(),
		KeyPrefix: "rl:",
	}, nil
}

// Allow checa rate limit via Redis Lua script.
//
// Comportamento:
//   - Redis OK + dentro do limite: retorna (true, 0)
//   - Redis OK + excedeu limite: retorna (false, ttl restante)
//   - Redis indisponível: loga warning + retorna (true, 0) (fail-open)
//
// Fail-open em Redis down é decisão consciente: API sem rate limit é
// melhor que API totalmente fora. Em prod, monitoring deve flagar
// Allow() errors como alerta (log estruturado com "redis_rate_limit_err").
func (r *RedisRateLimiter) Allow(bucket pathBucket, ifID string) (bool, time.Duration) {
	limit, ok := r.Limits[bucket]
	if !ok {
		return true, 0 // bucket desconhecido = passa
	}

	key := r.KeyPrefix + string(bucket) + ":" + ifID
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run Lua script (EVALSHA cache, fallback EVAL na 1ª chamada)
	res, err := r.Script.Run(ctx, r.Client, []string{key}, int(limit.Window.Seconds())).Result()
	if err != nil {
		r.Logger.Warn("redis rate limit failed (fail-open)", "err", err, "bucket", bucket, "if_id", ifID)
		return true, 0
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) != 2 {
		r.Logger.Warn("redis rate limit unexpected result (fail-open)", "res", res)
		return true, 0
	}

	current, _ := arr[0].(int64)
	ttl, _ := arr[1].(int64)

	if current > int64(limit.Max) {
		retryAfter := time.Duration(ttl) * time.Second
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}

	return true, 0
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