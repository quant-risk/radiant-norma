// Tests for SafeError + Wrap — confirmam que DSNs são sanitizados e
// outros conteúdos permanecem intactos.
package loggerutil_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
)

func TestSafeError_Nil(t *testing.T) {
	if got := loggerutil.SafeError(nil); got != "" {
		t.Errorf("SafeError(nil) = %q, want \"\"", got)
	}
}

func TestSafeError_PostgresURL(t *testing.T) {
	err := errors.New("failed to connect to postgres://app:secret123@db.example.com:5432/radiant")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "secret123") {
		t.Errorf("SafeError vaza password: %q", got)
	}
	if strings.Contains(got, "app:") {
		t.Errorf("SafeError vaza user: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("SafeError não marca redacted: %q", got)
	}
}

func TestSafeError_MySQLURL(t *testing.T) {
	err := errors.New("mysql://root:hunter2@db:3306/foo")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "hunter2") {
		t.Errorf("SafeError vaza password MySQL: %q", got)
	}
	if strings.Contains(got, "root:") {
		t.Errorf("SafeError vaza user MySQL: %q", got)
	}
}

func TestSafeError_RedisURL(t *testing.T) {
	err := errors.New("redis://default:abc123@redis.example.com:6379")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "abc123") {
		t.Errorf("SafeError vaza password Redis: %q", got)
	}
}

func TestSafeError_PlainError(t *testing.T) {
	err := errors.New("schema 'foo' não encontrado")
	got := loggerutil.SafeError(err)
	if got != "schema 'foo' não encontrado" {
		t.Errorf("SafeError deveria preservar plain error: %q", got)
	}
}

func TestSafeError_QueryStringPassword(t *testing.T) {
	// Menos comum, mas possível em URLs com query strings.
	err := errors.New("invalid connection: ?password=secret&sslmode=disable")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "secret") {
		t.Errorf("SafeError vaza password=... em query: %q", got)
	}
}

func TestWrap_Nil(t *testing.T) {
	err := loggerutil.Wrap("falhou", nil)
	if err == nil {
		t.Fatal("Wrap(nil) deveria retornar erro")
	}
	if err.Error() != "falhou" {
		t.Errorf("Wrap(nil) = %q, want 'falhou'", err.Error())
	}
}

func TestWrap_ErrWithDSN(t *testing.T) {
	inner := errors.New("connect: postgres://user:PASS@host/db")
	wrapped := loggerutil.Wrap("open db", inner)
	if strings.Contains(wrapped.Error(), "PASS") {
		t.Errorf("Wrap vaza password: %q", wrapped.Error())
	}
	if !strings.Contains(wrapped.Error(), "[REDACTED]") {
		t.Errorf("Wrap não marca redacted: %q", wrapped.Error())
	}
	if !strings.Contains(wrapped.Error(), "open db") {
		t.Errorf("Wrap não inclui mensagem: %q", wrapped.Error())
	}
}
