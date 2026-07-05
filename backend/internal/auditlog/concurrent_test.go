// Testes de stress concurrent para auditlog.
//
// Validação 21 (F21.5): confirmar que BEGIN IMMEDIATE é efetivo em
// goroutines paralelas. Sem isso, duas goroutines pegariam o mesmo
// prev_hash no SELECT antes do INSERT — chain quebrada + entries
// duplicadas.
//
// Bug hipotético:
//
//	goroutine A: BEGIN DEFERRED
//	goroutine B: BEGIN DEFERRED
//	A: SELECT entry_hash → h1
//	B: SELECT entry_hash → h1 (mesmo)
//	A: INSERT(entry_hash=h1) OK
//	B: INSERT(entry_hash=h1) — MESMO prev_hash, chain quebrada
//
// Consequência: Verify() vai falhar em qualquer entry > primeira.
package auditlog_test

import (
	"sync"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// TestAuditLog_NoChainBreaks_Concurrent: 50 goroutines call Log()
// em paralelo. Verify() deve passar sem chain break.
//
// Validação 21: regressão para F21.5 — race em audit_log hash chain.
func TestAuditLog_NoChainBreaks_Concurrent(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	logger := auditlog.New(d)

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := logger.Log(
				"if-demo", "test-actor",
				"test.action",
				"target-"+intToStr(idx),
				[]byte("payload-"+intToStr(idx)),
				map[string]any{"i": idx},
			)
			if err != nil {
				t.Errorf("Log %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify deve passar — chain válida.
	ok, count, err := logger.Verify()
	if err != nil {
		t.Fatalf("Verify failed com %d entries (chain possivelmente quebrada): %v", count, err)
	}
	if !ok {
		t.Fatalf("Verify returned ok=false com %d entries (chain break detected)", count)
	}
	if count != N {
		t.Errorf("expected %d entries, got %d", N, count)
	}
}

// TestAuditLog_NoChainBreaks_HighContention: 200 goroutines
// (maior contention). Deve passar mesmo com SQLite serializando.
func TestAuditLog_NoChainBreaks_HighContention(t *testing.T) {
	d := testutil.NewTestDB(t)
	defer d.Close()

	logger := auditlog.New(d)

	const N = 200
	var wg sync.WaitGroup
	sem := make(chan struct{}, 30) // limit concurrency 30 in flight
	wg.Add(N)
	for i := 0; i < N; i++ {
		sem <- struct{}{}
		go func(idx int) {
			defer func() { <-sem; wg.Done() }()
			_, err := logger.Log(
				"if-demo", "test-actor",
				"test.action",
				"target",
				[]byte("payload"),
				nil,
			)
			if err != nil {
				t.Errorf("Log %d failed: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	ok, count, err := logger.Verify()
	if err != nil {
		t.Fatalf("Verify failed com %d entries (chain possivelmente quebrada): %v", count, err)
	}
	if !ok || count != N {
		t.Errorf("chain break: ok=%v count=%d (expected %d)", ok, count, N)
	}
}

// intToStr evita import strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
