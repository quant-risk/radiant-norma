// Package loggerutil provides helpers to prevent credential leakage in logs.
//
// Validação 15/16/20 (F15.1, F16.5, F20.6): without sanitizing err.Error()
// before logging, drivers/token-leak expose metadata.
//
// SafeError / Wrap run regex passes to redact common credential
// patterns before logging.
package loggerutil

import (
	"errors"
	"regexp"
)

// maxSafeErrorBytes limita o tamanho do erro que SafeError processa.
// Validação 20 (F20.7): sem limite, mensagens gigantes (>1MB) podem
// custar 200ms+ e travar logging em hot path. 16KB é suficiente
// para a maioria dos err.Error() razoáveis. Mensagens maiores são
// truncadas com indicador.
const maxSafeErrorBytes = 16 * 1024

// dsnCanonical captura URLs com prefixo protocol://user:pass@host.
var dsnCanonical = regexp.MustCompile("(?i)(postgres|postgresql|mysql|mariadb|redis|mongodb)://[^@\\s]+@")

// pgxKeyValue captura formato key=value emitido por pgx.
var pgxKeyValue = regexp.MustCompile("(?i)(?:user|database|db|host|server|addr|port)=([^\\s`,;]+)")

// passwordKV captura password=X solto.
var passwordKV = regexp.MustCompile("(?i)\\b(password|passwd|pwd|secret)=([^&\\s,;]+)")

// passwordInQuery captura ?password=X / &password=Y em query strings.
var passwordInQuery = regexp.MustCompile("(?i)([?&](?:password|pwd|pass)=)[^&\\s,;]+")

// bearerToken captura Bearer/JWT em Authorization-style headers.
//
// Validação 20 (F20.6): Authorization: Bearer eyJ... é vetor comum.
// Match só em contextos `Bearer XXX=` ou `token XXX=` / `jwt XXX=`.
var bearerToken = regexp.MustCompile("(?i)\\b(bearer|token|jwt|auth|authorization)\\b[=:\\s]+([A-Za-z0-9\\-._~+/]+=*)")

// commonTokens captura token formats comuns que vazam.
//
// Validação 20 (F20.6): regex anterior só pegava password=X solto.
// Token formats (GitHub PAT, AWS key, Google OAuth, Stripe) passam
// despercebidos. Cada novo vendor tem prefix específico.
var commonTokens = regexp.MustCompile("\\b(ghp_|gho_|ghu_|ghs_|ghf_|ya29\\.|xox[a-z]-|xapp-|AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|sk_live_|sk_test_|pk_live_|rk_live_)[A-Za-z0-9]*")

// SafeError returns err.Error() with credentials replaced by [REDACTED].
//
// Use BEFORE logging err from drivers, HTTP clients, or any source
// that may include DSN/URL with password.
//
// Returns "" if err is nil.
//
// Message length é truncada em maxSafeErrorBytes (16KB). Mensagens
// maiores são cortadas com "... [TRUNCATED] ..." para evitar
// gargalo de CPU/memória.
//
// NÃO é bulletproof — erros estruturados (com fields separados)
// podem contornar regex. Para cobertura total, sanitize no nível
// do driver (pgx conn string parser) ou use logs estruturados
// com ignore fields.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// Validação 20 (F20.7): truncar mensagem gigante.
	truncated := false
	if len(msg) > maxSafeErrorBytes {
		msg = msg[:maxSafeErrorBytes]
		truncated = true
	}

	// 1st: DSN canônico.
	msg = dsnCanonical.ReplaceAllString(msg, "$1://[REDACTED]@")

	// 2nd: pgx key=value (user=X database=Y) — antes de passwordKV.
	msg = pgxKeyValue.ReplaceAllString(msg, "${1}=[REDACTED]")

	// 3rd: password=X solto.
	msg = passwordKV.ReplaceAllString(msg, "${1}=[REDACTED]")
	msg = passwordInQuery.ReplaceAllString(msg, "${1}[REDACTED]")

	// 4th: tokens (Bearer/JWT/etc) — Authorization headers.
	msg = bearerToken.ReplaceAllString(msg, "$1=[REDACTED]")

	// 5th: token formats específicos por vendor.
	msg = commonTokens.ReplaceAllString(msg, "[REDACTED_TOKEN]")

	if truncated {
		msg += " ... [TRUNCATED to " + intToStr(maxSafeErrorBytes) + " bytes]"
	}
	return msg
}

// intToStr evita import strconv (helper).
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

// Wrap returns a new error with format "<safeMsg>: <SafeError(err)>".
func Wrap(safeMsg string, err error) error {
	if err == nil {
		return errors.New(safeMsg)
	}
	return errors.New(safeMsg + ": " + SafeError(err))
}
