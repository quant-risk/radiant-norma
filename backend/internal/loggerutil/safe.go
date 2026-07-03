// Package internal/loggerutil — helpers para evitar vazamento de credenciais em logs.
//
// Validação 15 (F15.1 fix): pgx error messages incluem parte da DSN
// (user + database name) quando db.Ping() falha. Em produção, esses
// valores aparecem em stack traces e logs (vazamento de reconnaissance
// info). Memory pattern "secret em logs = disclosure" expandido.
//
// Helper:
//   - SafeError(err) retorna err.Error() com DSN-like substrings
//     substituídas por [REDACTED]. Use antes de logar.
//
// Detecta:
//   - postgres://user:pass@host:port/db  →  postgres://[REDACTED]
//   - postgresql://...
//   - mysql://...
//   - redis://...   (auth em URL)
//   - mongodb://...
//   - URL-encoded queries com password=...
//
// Se não detectar padrão, retorna err.Error() original.
package loggerutil

import (
	"errors"
	"regexp"
)

// dsnPatterns captura prefixos de URL que tipicamente têm credenciais.
var dsnPatterns = regexp.MustCompile(`(?i)(postgres|postgresql|mysql|mariadb|redis|mongodb)://[^@\s]+@`)

// errorWithPassword captura parâmetros de query com password.
// Exemplos: ?password=x, &password=y, ?sslmode=disable&password=z.
var errorWithPassword = regexp.MustCompile(`(?i)([?&](?:password|pwd|pass)=)[^&\s]+`)

// SafeError retorna err.Error() com credenciais substituídas por
// [REDACTED]. Use ANTES de logar erros de DB, HTTP, ou qualquer
// fonte que possa incluir DSN/URL com password.
//
// Se err é nil, retorna "".
//
// NOTA: não é bulletproof — erros estruturados (com fields
// separados) podem contornar regex. Para cobertura total, sanitize
// no nível do driver (pgx conn string parser) ou use logs estruturados
// com ignore fields.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	msg = dsnPatterns.ReplaceAllString(msg, "$1://[REDACTED]@")
	msg = errorWithPassword.ReplaceAllString(msg, "${1}[REDACTED]")
	return msg
}

// Wrap é wrapper de conveniência para fmt.Errorf com sanitização.
//
//	wrap := loggerutil.Wrap(err, "open db failed")
//	logger.Error("...", "err", wrap)
//
// Equivalente a fmt.Errorf("%s: %w", safeMsg, err) mas com sanitização.
func Wrap(safeMsg string, err error) error {
	if err == nil {
		return errors.New(safeMsg)
	}
	return errors.New(safeMsg + ": " + SafeError(err))
}
