// Testes de cache stampede do CadocListCache.
//
// Validação 22 (F22.2): singleflight deve evitar thundering herd.
// N goroutines em cache miss simultâneo devem disparar fetch() apenas 1x.
package schema_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/schema"
)

// TestCadocListCache_NoStampede_Concurrent: 200 goroutines chamam
// GetOrFetch simultaneamente com cache expirado.
//
// Validação 22 (F22.2): singleflight garante que fetch() roda APENAS
// 1 vez (independente da concorrência). Sem singleflight, 200
// goroutines fariam fetch() → DOS-via-DB.
func TestCadocListCache_NoStampede_Concurrent(t *testing.T) {
	c := schema.NewCadocListCache(100 * time.Millisecond)
	var fetchCount int64

	// Simular fetch() lento que conta invocações.
	fetch := func() ([]string, error) {
		atomic.AddInt64(&fetchCount, 1)
		time.Sleep(50 * time.Millisecond) // simula DB latency
		return []string{"3040", "3050", "4111"}, nil
	}

	// Espera cache expirar.
	time.Sleep(150 * time.Millisecond)

	const N = 200
	var wg sync.WaitGroup
	wg.Add(N)
	results := make([][]string, N)
	errs := make([]error, N)

	for i := 0; i < N; i++ {
		go func(idx int) {
			defer wg.Done()
			r, err := c.GetOrFetch(fetch)
			results[idx] = r
			errs[idx] = err
		}(i)
	}
	wg.Wait()

	// Validação: fetch() chamado exatamente 1 vez.
	count := atomic.LoadInt64(&fetchCount)
	if count != 1 {
		t.Errorf("fetch() chamado %d vezes com singleflight (esperado 1) — cache stampede não mitigado", count)
	}

	// Validação: todas as goroutines receberam o mesmo resultado.
	for i, r := range results {
		if errs[i] != nil {
			t.Errorf("goroutine %d: erro %v", i, errs[i])
		}
		if len(r) != 3 {
			t.Errorf("goroutine %d: esperado 3 cadocs, got %d", i, len(r))
		}
	}
}