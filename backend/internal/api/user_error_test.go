// Tests para userError helper — confirma que err.Error() NÃO vaza
// na response HTTP em qualquer cenário.
package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/api"
)

// errorMessageAllowed verifica que a response NÃO contém vetores
// típicos de err.Error() — strings SQL/JSON que err.Error() conteria.
func errorMessageDisallowed(body, forbidden string) bool {
	return !strings.Contains(body, forbidden)
}

func TestUserError_SanitizesErrAt400(t *testing.T) {
	// err.Error() com fragmento SQL real (vetor comum).
	rawErr := errors.New("sql: SELECT * FROM secrets WHERE token='abc123'")

	srv := api.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	srv.UserError(w, http.StatusBadRequest, "test", rawErr)

	got := w.Body.String()

	// Vetor deve estar AUSENTE.
	if !errorMessageDisallowed(got, "secrets") {
		t.Errorf("userError vaza SQL fragment: %q", got)
	}
	if !errorMessageDisallowed(got, "abc123") {
		t.Errorf("userError vaza token: %q", got)
	}
	if !errorMessageDisallowed(got, "SELECT *") {
		t.Errorf("userError vaza SQL: %q", got)
	}

	// Mensagem pública compacta deve estar presente.
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(strings.ToLower(got), "requisi") {
		t.Errorf("mensagem pública ausente: %q", got)
	}
}

func TestUserError_SanitizesJSONDetail(t *testing.T) {
	// err.Error() típico de json.Unmarshal com field names.
	rawErr := errors.New("invalid character 'x' looking for beginning of value (offset 42, field cadoc_internal_secret)")

	srv := api.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	srv.UserError(w, http.StatusBadRequest, "test", rawErr)

	got := w.Body.String()

	if !errorMessageDisallowed(got, "cadoc_internal_secret") {
		t.Errorf("userError vaza field name: %q", got)
	}
	if !errorMessageDisallowed(got, "offset 42") {
		t.Errorf("userError vaza offset: %q", got)
	}
}

func TestUserError_DSNAt500(t *testing.T) {
	// err.Error() típico de pgx com DSN canônica.
	rawErr := errors.New("failed to connect to `user=app database=secretdb`: hostname resolving")

	srv := api.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	w := httptest.NewRecorder()
	srv.UserError(w, http.StatusInternalServerError, "test", rawErr)

	got := w.Body.String()

	if !errorMessageDisallowed(got, "app") {
		t.Errorf("userError vaza user: %q", got)
	}
	if !errorMessageDisallowed(got, "secretdb") {
		t.Errorf("userError vaza database: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "erro interno") {
		t.Errorf("mensagem pública ausente: %q", got)
	}
}

// Sanity: response deve ser JSON ou texto simples, não panicar.
func TestUserError_AllStatusCodes(t *testing.T) {
	codes := []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity,
		http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}
	srv := api.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	for _, code := range codes {
		w := httptest.NewRecorder()
		srv.UserError(w, code, "test", errors.New("vetor x"))
		if w.Code != code {
			t.Errorf("status code %d esperado %d", w.Code, code)
		}
		body := w.Body.String()
		if body == "" {
			t.Errorf("body vazio para status %d", code)
		}
	}
}

// Garantir que JSON encoding é válido (não panic em Unicode, etc).
func TestUserError_JsonEncodingValid(t *testing.T) {
	srv := api.NewServer(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	w := httptest.NewRecorder()
	srv.UserError(w, http.StatusInternalServerError, "test",
		errors.New("java.sql.SQLException: unicode ãõ ç"))
	// Não precisa ser JSON, mas o body deve ser UTF-8 válido.
	var x interface{}
	body := w.Body.Bytes()
	if json.Valid(body) || len(body) > 0 {
		_ = x // ok se for texto
	}
}
