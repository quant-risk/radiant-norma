// Package loggerutil provides helpers to prevent credential leakage in logs.
//
// Validação 15/16 (F15.1, F16.5): without sanitizing err.Error()
// before logging, drivers expose metadata (DSN, user, database).
// pgx example: "failed to connect to `user=user database=db`".
//
// SafeError / Wrap are wrappers that run regex passes to redact
// common credential patterns before logging.
package loggerutil

import (
	"errors"
	"regexp"
)

// dsnCanonical captura URLs com prefixo protocol://user:pass@host.
//
// Exemplo: postgres://user:secret123@db:5432/radiant
var dsnCanonical = regexp.MustCompile("(?i)(postgres|postgresql|mysql|mariadb|redis|mongodb)://[^@\\s]+@")

// pgxKeyValue captura formato key=value emitido por pgx em mensagens
// de erro. Exemplo: "user=user database=db port=5432 sslmode=disable".
//
// CRÍTICO: rodar ANTES de passwordKV ou password=X fica mascarado
// mas user=database continuam expostos.
var pgxKeyValue = regexp.MustCompile("(?i)(?:user|database|db|host|server|addr|port)=([^\\s`,;]+)")

// passwordKV captura password=X solto.
var passwordKV = regexp.MustCompile("(?i)\\b(password|passwd|pwd|secret)=([^&\\s,;]+)")

// passwordInQuery captura ?password=X / &password=Y em query strings.
var passwordInQuery = regexp.MustCompile("(?i)([?&](?:password|pwd|pass)=)[^&\\s,;]+")

// SafeError returns err.Error() with credentials replaced by [REDACTED].
//
// Use BEFORE logging err from drivers, HTTP clients, or any source
// that may include DSN/URL with password.
//
// Returns "" if err is nil.
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

	// 1st: DSN canônico com prefixo protocol://user:pass@host.
	msg = dsnCanonical.ReplaceAllString(msg, "$1://[REDACTED]@")

	// 2nd: pgx key=value (user=X database=Y).
	// Rodar ANTES de passwordKV — senão password=X fica mascarado
	// mas user=database ficam expostos.
	msg = pgxKeyValue.ReplaceAllString(msg, "${1}=[REDACTED]")

	// 3rd: password=X solto.
	msg = passwordKV.ReplaceAllString(msg, "${1}=[REDACTED]")
	msg = passwordInQuery.ReplaceAllString(msg, "${1}[REDACTED]")

	return msg
}

// Wrap returns a new error with format "<safeMsg>: <SafeError(err)>".
//
// Se err é nil, retorna errors.New(safeMsg).
func Wrap(safeMsg string, err error) error {
	if err == nil {
		return errors.New(safeMsg)
	}
	return errors.New(safeMsg + ": " + SafeError(err))
}
