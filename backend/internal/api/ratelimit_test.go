// Package api — testes do rate limiter (memory + Redis backend).
//
// Sprint 16 — v3.6.0 [S16.1]: valida que ambos backends têm semântica
// equivalente: N calls dentro de Window → allow, N+1 → deny com
// retryAfter > 0. Redis usa miniredis (in-process Redis fake) pra
// rodar sem dependência de Docker.
package api

import (
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func discardSlog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// Memory backend tests
// =============================================================================

func TestMemoryRateLimiter_Allows(t *testing.T) {
	rl := newMemoryRateLimiter()
	max := DefaultRateLimits[bucketHeavy].Max
	for i := 0; i < max; i++ {
		allowed, _ := rl.Allow(bucketHeavy, "demo")
		if !allowed {
			t.Fatalf("call #%d deveria passar (max=%d)", i+1, max)
		}
	}
}

func TestMemoryRateLimiter_BlocksAtMax(t *testing.T) {
	rl := newMemoryRateLimiter()
	max := DefaultRateLimits[bucketHeavy].Max
	window := DefaultRateLimits[bucketHeavy].Window
	for i := 0; i < max; i++ {
		_, _ = rl.Allow(bucketHeavy, "demo")
	}
	allowed, retryAfter := rl.Allow(bucketHeavy, "demo")
	if allowed {
		t.Fatal("call após max deveria ser bloqueada")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter deveria ser > 0, got %v", retryAfter)
	}
	if retryAfter > window {
		t.Errorf("retryAfter (%v) não deveria exceder window (%v)", retryAfter, window)
	}
}

func TestMemoryRateLimiter_DifferentIFIDsIndependent(t *testing.T) {
	rl := newMemoryRateLimiter()
	max := DefaultRateLimits[bucketHeavy].Max
	for i := 0; i < max; i++ {
		_, _ = rl.Allow(bucketHeavy, "demo")
	}
	allowed, _ := rl.Allow(bucketHeavy, "demo-bank")
	if !allowed {
		t.Fatal("outra IF não deveria herdar rate limit")
	}
}

func TestMemoryRateLimiter_UnknownBucketPasses(t *testing.T) {
	rl := newMemoryRateLimiter()
	allowed, _ := rl.Allow("unknown-bucket", "demo")
	if !allowed {
		t.Fatal("bucket desconhecido deveria passar (fail-open)")
	}
}

func TestMemoryRateLimiter_Backend(t *testing.T) {
	rl := newMemoryRateLimiter()
	if rl.Backend() != "memory" {
		t.Errorf("Backend()=%q, want \"memory\"", rl.Backend())
	}
}

// =============================================================================
// Redis backend tests (com miniredis)
// =============================================================================

func newTestRedisLimiter(t *testing.T, limits map[pathBucket]RateLimit) (*RedisRateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	// miniredis adiciona :0 port auto; usa Addr direto
	opts := &redis.Options{Addr: mr.Addr()}
	client := redis.NewClient(opts)

	rl := &RedisRateLimiter{
		Client:    client,
		Limits:    limits,
		Script:    redis.NewScript(LuaIncrWithTTL),
		Logger:    discardSlog(),
		KeyPrefix: "rl:",
	}
	t.Cleanup(func() { _ = rl.Close() })
	return rl, mr
}

func TestRedisRateLimiter_Allows(t *testing.T) {
	rl, _ := newTestRedisLimiter(t, DefaultRateLimits)
	max := DefaultRateLimits[bucketHeavy].Max
	for i := 0; i < max; i++ {
		allowed, _ := rl.Allow(bucketHeavy, "demo")
		if !allowed {
			t.Fatalf("call #%d deveria passar", i+1)
		}
	}
}

func TestRedisRateLimiter_BlocksAtMax(t *testing.T) {
	rl, _ := newTestRedisLimiter(t, DefaultRateLimits)
	max := DefaultRateLimits[bucketHeavy].Max
	for i := 0; i < max; i++ {
		_, _ = rl.Allow(bucketHeavy, "demo")
	}
	allowed, retryAfter := rl.Allow(bucketHeavy, "demo")
	if allowed {
		t.Fatal("call após max deveria ser bloqueada")
	}
	if retryAfter <= 0 {
		t.Errorf("retryAfter deveria ser > 0, got %v", retryAfter)
	}
}

func TestRedisRateLimiter_DifferentIFIDsIndependent(t *testing.T) {
	rl, _ := newTestRedisLimiter(t, DefaultRateLimits)
	max := DefaultRateLimits[bucketHeavy].Max
	for i := 0; i < max; i++ {
		_, _ = rl.Allow(bucketHeavy, "demo")
	}
	allowed, _ := rl.Allow(bucketHeavy, "demo-bank")
	if !allowed {
		t.Fatal("outra IF não deveria herdar rate limit")
	}
}

func TestRedisRateLimiter_Backend(t *testing.T) {
	rl, _ := newTestRedisLimiter(t, DefaultRateLimits)
	if rl.Backend() != "redis" {
		t.Errorf("Backend()=%q, want \"redis\"", rl.Backend())
	}
}

func TestRedisRateLimiter_TTLExpires(t *testing.T) {
	rl, mr := newTestRedisLimiter(t, DefaultRateLimits)

	// Janela de 1s (mínimo que faz sentido em Redis — int() trunca).
	// Teste valida que após TTL, contador reseta.
	shortLimits := map[pathBucket]RateLimit{
		bucketHeavy: {Max: 2, Window: 1 * time.Second},
	}
	rl.Limits = shortLimits

	// 2 calls OK
	_, _ = rl.Allow(bucketHeavy, "demo")
	_, _ = rl.Allow(bucketHeavy, "demo")

	// 3ª bloqueada
	allowed, _ := rl.Allow(bucketHeavy, "demo")
	if allowed {
		t.Fatal("3ª call deveria estar bloqueada")
	}

	// Avança tempo do miniredis em 1.5s — TTL expira, contador reseta
	mr.FastForward(1500 * time.Millisecond)
	allowed, _ = rl.Allow(bucketHeavy, "demo")
	if !allowed {
		t.Fatal("após TTL expirar, call deveria passar")
	}
}

func TestRedisRateLimiter_FailOpenOnRedisDown(t *testing.T) {
	rl, mr := newTestRedisLimiter(t, DefaultRateLimits)
	mr.Close() // fecha Redis antes do Allow

	// Com Redis down, retorna (true, 0) por fail-open
	allowed, retryAfter := rl.Allow(bucketHeavy, "demo")
	if !allowed {
		t.Fatal("Redis down deveria fail-open (allow)")
	}
	if retryAfter != 0 {
		t.Errorf("fail-open deveria ter retryAfter=0, got %v", retryAfter)
	}
}

// =============================================================================
// Factory tests
// =============================================================================

func TestNewRateLimiterFromEnv_MemoryDefault(t *testing.T) {
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "")
	t.Setenv("RADIANT_REDIS_URL", "")

	rl, err := NewRateLimiterFromEnv()
	if err != nil {
		t.Fatalf("NewRateLimiterFromEnv: %v", err)
	}
	if rl.Backend() != "memory" {
		t.Errorf("Backend()=%q, want \"memory\"", rl.Backend())
	}
}

func TestNewRateLimiterFromEnv_MemoryExplicit(t *testing.T) {
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "memory")
	rl, err := NewRateLimiterFromEnv()
	if err != nil {
		t.Fatalf("NewRateLimiterFromEnv: %v", err)
	}
	if rl.Backend() != "memory" {
		t.Errorf("Backend()=%q, want \"memory\"", rl.Backend())
	}
}

func TestNewRateLimiterFromEnv_RedisRequiresURL(t *testing.T) {
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "redis")
	t.Setenv("RADIANT_REDIS_URL", "")
	_, err := NewRateLimiterFromEnv()
	if err == nil {
		t.Fatal("redis backend sem URL deveria retornar erro")
	}
}

func TestNewRateLimiterFromEnv_RedisBadURL(t *testing.T) {
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "redis")
	t.Setenv("RADIANT_REDIS_URL", "not-a-valid-url")
	_, err := NewRateLimiterFromEnv()
	if err == nil {
		t.Fatal("URL inválida deveria retornar erro")
	}
}

func TestNewRateLimiterFromEnv_UnknownBackend(t *testing.T) {
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "mongodb")
	_, err := NewRateLimiterFromEnv()
	if err == nil {
		t.Fatal("backend desconhecido deveria retornar erro")
	}
}

func TestNewRateLimiterFromEnv_RedisWithMiniredis(t *testing.T) {
	mr := miniredis.RunT(t)
	t.Setenv("RADIANT_RATE_LIMIT_BACKEND", "redis")
	t.Setenv("RADIANT_REDIS_URL", "redis://"+mr.Addr())

	rl, err := NewRateLimiterFromEnv()
	if err != nil {
		t.Fatalf("NewRateLimiterFromEnv: %v", err)
	}
	if rl.Backend() != "redis" {
		t.Errorf("Backend()=%q, want \"redis\"", rl.Backend())
	}
	t.Cleanup(func() {
		if rrl, ok := rl.(*RedisRateLimiter); ok {
			_ = rrl.Close()
		}
	})

	// Smoke: 1 call deve passar
	allowed, _ := rl.Allow(bucketHeavy, "demo")
	if !allowed {
		t.Fatal("primeira call deveria passar")
	}
}
// TestRedisRateLimiter_ConcurrentStress valida que sob carga concorrente
// intensa (50 goroutines × 100 calls), Redis Lua script é atômico —
// não double-counting, não race em calls/locks.
//
// Esperado: bucketHeavy.Max=10 → apenas 10 calls allowed, 4990 denied.
// Sem race detector flag. Sem double-count (Lua script atomic).
func TestRedisRateLimiter_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skip stress test em -short mode")
	}

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := &RedisRateLimiter{
		Client:    client,
		Limits:    DefaultRateLimits,
		Script:    redis.NewScript(LuaIncrWithTTL),
		Logger:    discardSlog(),
		KeyPrefix: "stress:",
	}
	t.Cleanup(func() { _ = rl.Close() })

	const (
		goroutines        = 50
		callsPerGoroutine = 100
	)

	var (
		allowed atomic.Int64
		denied  atomic.Int64
		wg      sync.WaitGroup
	)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := 0; c < callsPerGoroutine; c++ {
				ok, _ := rl.Allow(bucketHeavy, "stress-demo")
				if ok {
					allowed.Add(1)
				} else {
					denied.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	total := allowed.Load() + denied.Load()
	expectedTotal := int64(goroutines * callsPerGoroutine)
	if total != expectedTotal {
		t.Fatalf("total mismatch: got %d, want %d", total, expectedTotal)
	}

	// bucketHeavy.Max=10. 50×100=5000 calls. Apenas 10 allowed, 4990 denied.
	if allowed.Load() != 10 {
		t.Errorf("allowed=%d, want 10 (heavy bucket max)", allowed.Load())
	}
	if denied.Load() != expectedTotal-10 {
		t.Errorf("denied=%d, want %d", denied.Load(), expectedTotal-10)
	}

	t.Logf("stress test OK: %d allowed, %d denied (de %d total)",
		allowed.Load(), denied.Load(), total)
}

// TestMemoryRateLimiter_ConcurrentStress valida que memory backend também
// é thread-safe sob carga concorrente. Mutex deve serializar correctly.
func TestMemoryRateLimiter_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skip stress test em -short mode")
	}

	rl := newMemoryRateLimiter()

	const (
		goroutines        = 50
		callsPerGoroutine = 100
	)

	var (
		allowed atomic.Int64
		denied  atomic.Int64
		wg      sync.WaitGroup
	)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := 0; c < callsPerGoroutine; c++ {
				ok, _ := rl.Allow(bucketHeavy, "stress-mem-demo")
				if ok {
					allowed.Add(1)
				} else {
					denied.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	total := allowed.Load() + denied.Load()
	expectedTotal := int64(goroutines * callsPerGoroutine)
	if total != expectedTotal {
		t.Fatalf("total mismatch: got %d, want %d", total, expectedTotal)
	}

	// Sem Redis TTL, todas as 5000 calls caem na mesma janela → só 10 allowed.
	if allowed.Load() != 10 {
		t.Errorf("allowed=%d, want 10 (heavy bucket max, single window)", allowed.Load())
	}

	t.Logf("memory stress test OK: %d allowed, %d denied",
		allowed.Load(), denied.Load())
}
