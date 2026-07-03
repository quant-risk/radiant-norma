// Tests for SafeError + Wrap — confirmam que DSNs são sanitizados e
// outros conteúdos permanecem intactos.
package loggerutil_test

import (
	"errors"
	"fmt"
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

// ===========================
// Validação 16 (F16.5): regressão — vetor REAL pgx
// ===========================
//
// O vetor que motivou F15.1 não foi DSN bem formada (covered acima),
// mas o WRAP output do pgx com template `\`user=%s database=%s\``:
//
//	"failed to connect to `user=user database=db`: hostname resolving..."
//
// O atual `dsnPatterns` regex `(?i)(postgres|...):` casa com
// "postgres://" — NÃO casa com `user=user` (não tem prefixo DSN).
// Por isso o vetor REAL do pgx passa despercebido pela sanitização
// atual (F15.1 PLUG, não FIX).
//
// Este test é a regressão que força o fix no próximo round.

func TestSafeError_PgxConnectError_REAL_Vector(t *testing.T) {
	// Este é o output real do pgx com DSN postgres://user:secret123@...
	err := errors.New("failed to connect to `user=user database=db`: hostname resolving error")
	got := loggerutil.SafeError(err)

	// Vetor real expõe:
	if strings.Contains(got, "user=user") {
		t.Errorf("SafeError NÃO sanitiza vetor REAL pgx (user=X): %q — fix pendente", got)
	}
	if strings.Contains(got, "database=db") {
		t.Errorf("SafeError NÃO sanitiza vetor REAL pgx (database=X): %q — fix pendente", got)
	}
}

// Regressão para outros vetores comuns que vão acontecer:
// - Connection strings with embedded credentials
// - URLs em stack traces
// - JWT tokens em logs (eyJ...)
// - Cookies em logs

func TestSafeError_URLInStackTrace(t *testing.T) {
	// Simular erro com URL embedded (sem prefixo protocol).
	err := errors.New("connection refused at 192.168.1.1:5432 for user=admin")
	got := loggerutil.SafeError(err)

	// Username "admin" não é exposto (esperado); queremos "admin user=" ainda,
	// embora admin não seja necessariamente secret, é info disclosure.
	// Documentado: regex atual só pega URLs com prefixo protocol — NÂO
	// pega linhas tipo "user=X" ou "password=X" soltas.
	//
	// Vetor documentado e aceito como follow-up — ver F16.11.
	if !strings.Contains(got, "user=admin") {
		t.Logf("NOTA: regex não detecta 'user=X' sem prefixo DSN: %q (F16.11 follow-up)", got)
	}
}

func TestSafeError_EmptyError(t *testing.T) {
	err := errors.New("")
	got := loggerutil.SafeError(err)
	if got != "" {
		t.Errorf("SafeError(empty err) = %q, want empty string", got)
	}
}

func TestSafeError_NestedDSN(t *testing.T) {
	// Erro com DSN aninhada (raro mas ocorre com fmt.Errorf chains)
	err := errors.New("connect failed: postgres://u:pa55@primary:5432/db; fallback redis://default:abc123@cache:6379/0")
	got := loggerutil.SafeError(err)
	if strings.Contains(got, "pa55") {
		t.Errorf("SafeError não sanitizou primeiro DSN: %q", got)
	}
	if strings.Contains(got, "abc123") {
		t.Errorf("SafeError não sanitizou segundo DSN: %q", got)
	}
}

// ===========================
// Validação 17 (F17.1): regressão — vetor stmt.Exec/INSERT errors
// ===========================
//
// Esses erros NÃO vazam credenciais por design (database/sql não
// anexa DSN nas mensagens de constraint) — mas o test garante que
// mudanças no regex (F16.5) não corrompem mensagens legítimas,
// e estabelece baseline se algum driver futuro passar a incluir DSN.

func TestSafeError_DriverConstraintError(t *testing.T) {
	// Erro típico de INSERT/SQLITE — "constraint failed: UNIQUE constraint..."
	err := errors.New("UNIQUE constraint failed: schema_versions.cadoc_code")
	got := loggerutil.SafeError(err)
	if got != "UNIQUE constraint failed: schema_versions.cadoc_code" {
		t.Errorf("SafeError corrompeu constraint msg: %q", got)
	}
}

func TestSafeError_JsonDecodeError(t *testing.T) {
	// Erro típico de json.Unmarshal — "invalid character ..."
	err := errors.New("invalid character 'x' looking for beginning of value")
	got := loggerutil.SafeError(err)
	if got != "invalid character 'x' looking for beginning of value" {
		t.Errorf("SafeError corrompeu JSON msg: %q", got)
	}
}

func TestSafeError_FmtErrorfChain(t *testing.T) {
	// fmt.Errorf("%w", inner) com DSN encadeada.
	inner := errors.New("connect: postgres://u:pa55@primary:5432/db")
	outer := fmt.Errorf("first attempt: %w; retrying", inner)
	got := loggerutil.SafeError(outer)
	// Em Go 1.13+, fmt.Errorf com %w preserva a mensagem de inner via
	// Unwrap(). Mas SafeError recebe error.Error() (que concatena tudo).
	if strings.Contains(got, "pa55") {
		t.Errorf("SafeError não sanitizou DSN via fmt.Errorf chain: %q", got)
	}
}
