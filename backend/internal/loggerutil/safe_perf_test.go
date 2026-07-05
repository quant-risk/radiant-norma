// Tests de stress do SafeError — vetores REAL e performance.
// Validação 20 (F20.5): confirmar que regex robusto não vira
// gargalo sob mensagem gigante.
package loggerutil_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
)

// 16KB string — tamanho típico de um error message real.
const bigBlock = "lorem ipsum dolor sit amet, consectetur adipiscing elit. " // 56 chars
const bigBlocks = 290                                                        // 290 * 56 ≈ 16.2KB

func buildBigMsg() string {
	var b strings.Builder
	for i := 0; i < bigBlocks; i++ {
		b.WriteString(bigBlock)
	}
	b.WriteString(" ERROR postgres://user:secret@host:5432/db oauth_token=ya29.abc123")
	return b.String()
}

// TestSafeError_TypicalMessage_Performance: 16KB string (tamanho típico
// pós-truncamento) passa pelo regex em < 50ms (sob -race detector).
//
// Validação 21 (F20.7): confirma que fix de 16KB truncation funciona.
// Mensagens >16KB são truncadas, então testamos do tamanho real esperado.
func TestSafeError_TypicalMessage_Performance(t *testing.T) {
	bigMsg := buildBigMsg()
	if len(bigMsg) < 16000 || len(bigMsg) > 17000 {
		t.Fatalf("test setup: expected 16KB message, got %d bytes", len(bigMsg))
	}
	start := time.Now()
	got := loggerutil.SafeError(errors.New(bigMsg))
	elapsed := time.Since(start)

	// Validação 45 (F-S24-45-15): threshold aumentado de 250ms para 500ms.
	// -race detector adiciona ~10x overhead. Em suite completa (20 packages
	// em paralelo), CPU disputada causa picos até ~400ms. Sem -race seria
	// <5ms. O ponto é detectar regressões cataclísmicas, não otimizar
	// microperformance.
	if elapsed > 500*time.Millisecond {
		t.Errorf("SafeError lento demais: %v em 16KB (esperado <500ms sob -race)", elapsed)
	}
	if strings.Contains(got, "secret") {
		t.Errorf("vaza password")
	}
	if strings.Contains(got, "ya29") {
		t.Errorf("vaza token")
	}
	t.Logf("SafeError processou %d bytes em %v", len(bigMsg), elapsed)
}

// TestSafeError_OversizedMessage_Performance: mensagem gigante (>16KB)
// é truncada ANTES de regex passes. Performance esperada: similar
// a 16KB (~50ms sob -race), não proporcional ao tamanho original.
func TestSafeError_OversizedMessage_Performance(t *testing.T) {
	huge := buildHuge()
	start := time.Now()
	got := loggerutil.SafeError(errors.New(huge))
	elapsed := time.Since(start)

	// Validação 45 (F-S24-45-15): threshold aumentado de 250ms para 500ms
	// (mesma justificativa do test anterior).
	if elapsed > 500*time.Millisecond {
		t.Errorf("SafeError lento em oversized: %v (esperado <500ms sob -race)", elapsed)
	}
	if !strings.Contains(got, "[TRUNCATED") {
		t.Errorf("oversized message não foi truncada: len=%d", len(got))
	}
	t.Logf("SafeError processou oversized %d → %d bytes em %v", len(huge), len(got), elapsed)
}

func buildHuge() string {
	var b strings.Builder
	for i := 0; i < 20000; i++ {
		b.WriteString(bigBlock)
	}
	b.WriteString(" postgres://user:secret@h/d")
	return b.String()
}

// TestSafeError_MultipleVetors: 1 mensagem com múltiplos vetores.
func TestSafeError_MultipleVetors(t *testing.T) {
	errStr := "\n[2026-07-03 22:00] attempt 1: postgres://app:pa55@primary:5432/db\n" +
		"[2026-07-03 22:01] attempt 2: redis://default:r3dis@cache:6379\n" +
		"[2026-07-03 22:02] failed: `user=app database=secretdb`\n" +
		"[2026-07-03 22:03] retry with password=hunter2\n" +
		"[2026-07-03 22:04] ?token=ghp_abc123&secret=ghp_def456\n" +
		"[2026-07-03 22:05] X-Admin-Token: ya29.abc123\n"
	err := errors.New(errStr)
	got := loggerutil.SafeError(err)
	for _, mustNotContain := range []string{"pa55", "r3dis", "hunter2", "ghp_abc", "ghp_def", "ya29"} {
		if strings.Contains(got, mustNotContain) {
			t.Errorf("vaza %s em message: vetor presente", mustNotContain)
		}
	}
}

// TestSafeError_NilAndEmpty: nil e empty error não panica.
func TestSafeError_NilAndEmpty(t *testing.T) {
	if got := loggerutil.SafeError(nil); got != "" {
		t.Errorf("SafeError(nil) = %q, want empty", got)
	}
	if got := loggerutil.SafeError(errors.New("")); got != "" {
		t.Errorf("SafeError(empty) = %q, want empty", got)
	}
}

// TestSafeError_UnicodeSafe: regex não quebra com Unicode.
func TestSafeError_UnicodeSafe(t *testing.T) {
	err := errors.New("connection failed: postgres://user:secret@host/db ãõ ç 中文 العربية")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "secret") {
		t.Errorf("vaza secret em Unicode context")
	}
	if !strings.Contains(got, "postgres://[REDACTED]@") {
		t.Errorf("DSN não foi reescrito em Unicode: %q", got)
	}
}

// Benchmark para confirmar custo em hot path.
func BenchmarkSafeError(b *testing.B) {
	err := errors.New("failed: postgres://user:secret@host:5432/db user=user database=db password=secret")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = loggerutil.SafeError(err)
	}
}

// TestSafeError_FmtErrorfChain_PerfVersion.
func TestSafeError_FmtErrorfChain_PerfVersion(t *testing.T) {
	inner := errors.New("wrapped: postgres://u:p@h/d")
	outer := fmt.Errorf("outer context: %w with extra", inner)
	got := loggerutil.SafeError(outer)
	if strings.Contains(got, "p@h") {
		t.Errorf("vaza password em fmt.Errorf chain")
	}
}
