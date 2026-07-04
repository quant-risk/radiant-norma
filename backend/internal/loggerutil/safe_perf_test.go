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

// 1MB string.
const bigBlock = "lorem ipsum dolor sit amet, consectetur adipiscing elit. " // 56 chars
const bigBlocks = 20000                                                      // 20000 * 56 = 1.12MB

func buildBigMsg() string {
	var b strings.Builder
	for i := 0; i < bigBlocks; i++ {
		b.WriteString(bigBlock)
	}
	b.WriteString(" ERROR postgres://user:secret@host:5432/db oauth_token=ya29.abc123")
	return b.String()
}

// TestSafeError_LargeMessage_Performance: 1MB string passa pelo regex
// em < 50ms.
func TestSafeError_LargeMessage_Performance(t *testing.T) {
	bigMsg := buildBigMsg()
	start := time.Now()
	got := loggerutil.SafeError(errors.New(bigMsg))
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("SafeError lento demais: %v em mensagem de %d bytes (esperado <200ms sob -race)", elapsed, len(bigMsg))
	}
	if strings.Contains(got, "secret") {
		t.Errorf("vaza password em 1MB")
	}
	if strings.Contains(got, "ya29") {
		t.Errorf("vaza token em 1MB")
	}
	t.Logf("SafeError processou %d bytes em %v", len(bigMsg), elapsed)
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
