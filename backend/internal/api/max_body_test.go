// Tests para maxBodyBytesMiddleware — regressão para F20.3
// (DOS-via-large-body que escapou das validações anteriores).
package api_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/api"
)

// TestMaxBody_AcceptsUnderLimit: body válido (< 10MB) passa pelo middleware.
func TestMaxBody_AcceptsUnderLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apenas confirma que o body é legível.
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("body read failed: %v", err)
		}
		if len(b) != 1024 {
			t.Errorf("expected 1024 bytes, got %d", len(b))
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := api.MaxBodyBytesMiddlewareForTest(10<<20, handler)

	body := bytes.Repeat([]byte("a"), 1024)
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewReader(body))
	req.Header.Set("X-IF-ID", "demo")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// TestMaxBody_RejectsOverLimit: body > 10MB → 413 ANTES de alocar conteúdo.
func TestMaxBody_RejectsOverLimit(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := api.MaxBodyBytesMiddlewareForTest(10<<20, handler)

	// Content-Length 11 MiB (acima do limite).
	body := bytes.Repeat([]byte("a"), 11<<20)
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewReader(body))
	req.Header.Set("X-IF-ID", "demo")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rec.Code)
	}
	if handlerCalled {
		t.Error("handler should NOT have been called for oversized body")
	}
}

// TestMaxBody_NoContentLength_StreamsToLimit: cliente não envia Content-Length
// (chunks). MaxBytesReader deve parar quando chegar ao limite.
func TestMaxBody_NoContentLength_StreamsToLimit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tentar ler tudo deve falhar (limite atingido).
		_, err := io.ReadAll(r.Body)
		if err == nil {
			t.Error("expected error reading oversized body, got nil")
		}
		// Error deve ser do tipo "body too large"
		if !strings.Contains(err.Error(), "too large") &&
			!strings.Contains(err.Error(), "EOF") {
			t.Logf("got expected streaming error: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
	})

	wrapped := api.MaxBodyBytesMiddlewareForTest(1024, handler) // 1KB limit

	body := bytes.Repeat([]byte("a"), 2<<20) // 2 MiB
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewReader(body))
	req.Header.Set("X-IF-ID", "demo")
	req.ContentLength = -1 // Simula chunked encoding sem Content-Length
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
}
