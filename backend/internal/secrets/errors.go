package secrets

import (
	"errors"
	"fmt"
)

// NotFoundError indica que o secret não existe no backend.
//
// Caller pode usar errors.As(err, &nf) para classificar:
//
//	if errors.As(err, &secrets.NotFoundError{}) {
//	    // secret não existe — talvez criar?
//	}
type NotFoundError struct {
	Name    string
	Backend string
	Cause   error
}

func (e *NotFoundError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("secret %q not found in %s: %v", e.Name, e.Backend, e.Cause)
	}
	return fmt.Sprintf("secret %q not found in %s", e.Name, e.Backend)
}

func (e *NotFoundError) Unwrap() error { return e.Cause }

// AccessDeniedError indica que o backend recusou acesso (IAM, ACL, etc).
//
// Não confundir com autenticação (que é diferente). AccessDenied é pós-auth:
// o caller tem identidade mas falta permissão.
type AccessDeniedError struct {
	Name    string
	Backend string
	Cause   error
}

func (e *AccessDeniedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("access denied to secret %q in %s: %v", e.Name, e.Backend, e.Cause)
	}
	return fmt.Sprintf("access denied to secret %q in %s", e.Name, e.Backend)
}

func (e *AccessDeniedError) Unwrap() error { return e.Cause }

// ValidationError indica que input do caller é inválido (nome vazio,
// value vazio, caracteres proibidos, etc).
//
// Não retryable. Caller deve corrigir input.
type ValidationError struct {
	Name   string
	Reason string
	Cause  error
}

func (e *ValidationError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("validation error: %s", e.Reason)
	}
	return fmt.Sprintf("validation error for secret %q: %s", e.Name, e.Reason)
}

func (e *ValidationError) Unwrap() error { return e.Cause }

// Compile-time guarantees que erros satisfazem error interface e são
// identificáveis via errors.As.
var (
	_ error = (*NotFoundError)(nil)
	_ error = (*AccessDeniedError)(nil)
	_ error = (*ValidationError)(nil)
)

// Helper: IsNotFound é sugar pra errors.As.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

// Helper: IsAccessDenied é sugar pra errors.As.
func IsAccessDenied(err error) bool {
	var ad *AccessDeniedError
	return errors.As(err, &ad)
}

// Helper: IsValidation é sugar pra errors.As.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}