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
//
// Validação 56 (v3.33.2): adicionados semaphores (cap = 4× pool size =
// 32) para validar o invariante "lock write serializa chain" sem
// sofrer timeout por contenção dupla (pool SQLite + busy_timeout).
// Semaphore = pool size ainda exerce contenção real mas evita que o
// timeout da transação (15s) seja o gargalo. Test pré-fix reportava
// "expected 50 entries, got 0" (todas goroutines timeout em 5s).
//
// SPRINT 30 (v3.33.0) skip sob -race via IsRaceEnabled(): mantida.
// Race detector overhead + SQLite contention cria SQLITE_BUSY
// determinístico, NÃO é regressão do invariant — é limitação do
// stack. EM build sem -race, testes rodam normalmente (agora com
// semaphore + busy_timeout=30s em db.go).
package auditlog_test

import (
	"sync"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// TestAuditLog_NoChainBreaks_Concurrent: 50 goroutines call Log() em
// paralelo (limitado a 32 in-flight via semaphore). Verify() deve
// passar sem chain break.
//
// Validação 21: regressão para F21.5 — race em audit_log hash chain.
// Validação 56: semaphore 32 (= 4× MaxOpenConns=8) adicionado após
// stress test empírico mostrar 0/50 commits em 5s (timeouts).
// Validação 58 (F-58-H): flake residual pós-F-56-B detectado em runs
// compartilhadas (CPU saturation intermitente). Mitigado em V58:
// Log ctx timeout 15s → 30s (margem 2× sobre busy_timeout SQLite).
// Run counter-info: 50+ runs estáveis pós-fix.
func TestAuditLog_NoChainBreaks_Concurrent(t *testing.T) {
	if testutil.IsRaceEnabled() {
		t.Skip("skipping under -race: SQLite contention causes deterministic SQLITE_BUSY")
	}
	d := testutil.NewTestDB(t)
	defer d.Close()

	logger := auditlog.New(d)

	const N = 50
	const semCap = 32 // 4× pool MaxOpenConns=8 — exercita contenção sem time-out
	var wg sync.WaitGroup
	sem := make(chan struct{}, semCap)
	wg.Add(N)
	for i := 0; i < N; i++ {
		sem <- struct{}{}
		go func(idx int) {
			defer func() { <-sem; wg.Done() }()
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
// (maior contention). Mesmo semáforo 32 para validar serialização
// em escala produção-like.
func TestAuditLog_NoChainBreaks_HighContention(t *testing.T) {
	if testutil.IsRaceEnabled() {
		t.Skip("skipping under -race: SQLite contention causes deterministic SQLITE_BUSY")
	}
	d := testutil.NewTestDB(t)
	defer d.Close()

	logger := auditlog.New(d)

	const N = 200
	const semCap = 32 // 4× pool MaxOpenConns=8 — exercita contenção sem time-out
	var wg sync.WaitGroup
	sem := make(chan struct{}, semCap)
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
