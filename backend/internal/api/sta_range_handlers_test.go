// sta_range_handlers_test.go — Sprint 57 v3.36.3: testes para RangeUploadAPI.
//
// Foco em contratos dos 3 handlers sem usar BACEN real:
//   - staRangeInit: validação body, cadoc_code, BACEN client capability
//   - staRangeUpload: Content-Range parsing, BACEN PUT, persistência
//   - staRangeStatus: lookup de sessão em memória
//
// Estes testes usam stubBACEN que implementa sta.Client + sta.RangeUploader.
// Não testam wiring de DB/Redis (handlers são nil-safe via s.DB nil-check).
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/go-chi/chi/v5"
)

// stubRangeUploader implementa sta.RangeUploader + sta.ReadClient + sta.ChunkedClient.
type stubRangeUploader struct {
	initFunc     func(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (string, error)
	submitFunc   func(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error
	statusFunc   func(ctx context.Context, protocolo string) (*sta.UploadStatus, error)
	downloadFunc func(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*sta.DownloadResult, error)
	listFunc     func(ctx context.Context, opts sta.ListDisponiveisOpts) (*sta.ListDisponiveisResult, error)
	alterarFunc  func(ctx context.Context, req sta.AlterarSituacaoReq) error
}

func (s *stubRangeUploader) InitRangeSession(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (string, error) {
	if s.initFunc != nil {
		return s.initFunc(ctx, cadocCode, hashHex, totalBytes)
	}
	return "PROT-12345", nil
}

func (s *stubRangeUploader) SubmitRange(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error {
	if s.submitFunc != nil {
		return s.submitFunc(ctx, protocolo, inicio, fim, total, chunk)
	}
	return nil
}

func (s *stubRangeUploader) StatusUpload(ctx context.Context, protocolo string) (*sta.UploadStatus, error) {
	if s.statusFunc != nil {
		return s.statusFunc(ctx, protocolo)
	}
	return &sta.UploadStatus{Protocolo: protocolo, Situacao: sta.UploadSituacaoFinalizada}, nil
}

func (s *stubRangeUploader) DownloadRange(ctx context.Context, protocolo string, inicio, fim int64, expectedTotalHash, ifMatch, ifUnmodifiedSince string) (*sta.DownloadResult, error) {
	if s.downloadFunc != nil {
		return s.downloadFunc(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince)
	}
	return nil, errors.New("not implemented")
}

func (s *stubRangeUploader) ListDisponiveis(ctx context.Context, opts sta.ListDisponiveisOpts) (*sta.ListDisponiveisResult, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func (s *stubRangeUploader) AlterarSituacao(ctx context.Context, req sta.AlterarSituacaoReq) error {
	if s.alterarFunc != nil {
		return s.alterarFunc(ctx, req)
	}
	return errors.New("not implemented")
}

// stubSTA é um STAClient mínimo sem chunked support (apenas Submit stub).
type stubSTA struct{}

func (s *stubSTA) Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error) {
	return &sta.Result{ProtocolSTA: "X", Accepted: true}, nil
}

// stubBACEN suporta RangeUploader + Submit (implementa sta.Client).
type stubBACEN struct {
	stubRangeUploader
}

func (s *stubBACEN) Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error) {
	return &sta.Result{ProtocolSTA: "BACEN-X", Accepted: true}, nil
}

// novoServerRange cria um Server com STAClient mockado (DB = nil → persistência skipped).
func novoServerRange(t *testing.T, client sta.Client) *Server {
	t.Helper()
	s := &Server{STAClient: client}
	return s
}

// seedActiveSession pré-popula activeSessions para testes de staRangeUpload/Status.
// NOTA: cleanup é responsabilidade do teste (chamar cleanupActiveSession) para
// evitar deadlock com t.Cleanup + handler's sessionsMu.
func seedActiveSession(t *testing.T, protocolo, ifID string) {
	t.Helper()
	sessionsMu.Lock()
	activeSessions[protocolo] = &rangeSession{
		ID:         "sess-" + protocolo,
		IfID:       ifID,
		Protocolo:  protocolo,
		CadocCode:  "3040",
		TotalBytes: 1000,
		Ranges:     []sta.Range{},
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	sessionsMu.Unlock()
}

// cleanupActiveSession remove sessão pré-populada (call after test).
func cleanupActiveSession(protocolo string) {
	sessionsMu.Lock()
	delete(activeSessions, protocolo)
	sessionsMu.Unlock()
}

// doRangeRequest monta um request HTTP com chi.URLParam routing context.
// Para handler init (sem param), passa protocolo="".
func doRangeRequest(t *testing.T, method, path, protocolo string, body io.Reader, headers map[string]string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if protocolo != "" {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("protocolo", protocolo)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// =============================================================================
// staRangeInit tests
// =============================================================================

func TestStaRangeInit_MissingBody(t *testing.T) {
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "", nil,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}

func TestStaRangeInit_MissingCadoc(t *testing.T) {
	s := novoServerRange(t, &stubBACEN{})
	body := strings.NewReader(`{"total_bytes": 1024}`)
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "", body,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (missing cadoc_code)", w.Code)
	}
}

func TestStaRangeInit_InvalidJSON(t *testing.T) {
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "",
		strings.NewReader(`not json`), map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (invalid JSON)", w.Code)
	}
}

func TestStaRangeInit_STAClientNoRangeSupport(t *testing.T) {
	// stubSTA não implementa RangeUploader → 503
	s := novoServerRange(t, &stubSTA{})
	body := strings.NewReader(`{"cadoc_code": "3040"}`)
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "", body,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d, want 503 (STA without chunked support)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "chunked") {
		t.Errorf("body should mention chunked: %q", w.Body.String())
	}
}

func TestStaRangeInit_BACENError(t *testing.T) {
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			initFunc: func(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (string, error) {
				return "", errors.New("BACEN offline")
			},
		},
	}
	s := novoServerRange(t, bacen)
	body := strings.NewReader(`{"cadoc_code": "3040", "total_bytes": 1024}`)
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "", body,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d, want 502 (BACEN error)", w.Code)
	}
}

func TestStaRangeInit_Success(t *testing.T) {
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			initFunc: func(ctx context.Context, cadocCode, hashHex string, totalBytes int64) (string, error) {
				return "PROT-99999", nil
			},
		},
	}
	s := novoServerRange(t, bacen)
	body := strings.NewReader(`{"cadoc_code": "3040", "hash_hex": "abc", "total_bytes": 1048576}`)
	req := doRangeRequest(t, http.MethodPost, "/v1/sta/range/init", "", body,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeInit(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status=%d, want 201 (got body %s)", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["protocolo"] != "PROT-99999" {
		t.Errorf("protocolo=%v, want PROT-99999", resp["protocolo"])
	}
}

// =============================================================================
// staRangeUpload tests
// =============================================================================

func TestStaRangeUpload_MissingProtocolo(t *testing.T) {
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "", bytes.NewReader([]byte("data")),
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (missing protocolo)", w.Code)
	}
}

func TestStaRangeUpload_MissingContentRange(t *testing.T) {
	seedActiveSession(t, "PROT-12345", "test-if")
	defer cleanupActiveSession("PROT-12345")
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "PROT-12345",
		bytes.NewReader([]byte("data")),
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (missing Content-Range)", w.Code)
	}
}

func TestStaRangeUpload_InvalidContentRange(t *testing.T) {
	seedActiveSession(t, "PROT-12345", "test-if")
	defer cleanupActiveSession("PROT-12345")
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "PROT-12345",
		bytes.NewReader([]byte("data")),
		map[string]string{"Content-Range": "not-a-valid-range", "X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (invalid Content-Range)", w.Code)
	}
}

func TestStaRangeUpload_STAClientNoRangeSupport(t *testing.T) {
	seedActiveSession(t, "PROT-12345", "test-if")
	defer cleanupActiveSession("PROT-12345")
	s := novoServerRange(t, &stubSTA{})
	chunk := make([]byte, 100) // match Content-Range bytes 0-99/1000
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "PROT-12345",
		bytes.NewReader(chunk),
		map[string]string{"Content-Range": "bytes 0-99/1000", "X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d, want 503", w.Code)
	}
}

func TestStaRangeUpload_BACENError(t *testing.T) {
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			submitFunc: func(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error {
				return errors.New("BACEN 500")
			},
		},
	}
	seedActiveSession(t, "PROT-12345", "test-if")
	defer cleanupActiveSession("PROT-12345")
	s := novoServerRange(t, bacen)
	chunk := make([]byte, 10)
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "PROT-12345",
		bytes.NewReader(chunk),
		map[string]string{"Content-Range": "bytes 0-9/100", "X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status=%d, want 502 (BACEN 500)", w.Code)
	}
}

func TestStaRangeUpload_Success(t *testing.T) {
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			submitFunc: func(ctx context.Context, protocolo string, inicio, fim, total int64, chunk []byte) error {
				if inicio != 0 || fim != 99 || total != 1000 {
					return fmt.Errorf("unexpected range %d-%d/%d", inicio, fim, total)
				}
				if int64(len(chunk)) != 100 {
					return fmt.Errorf("chunk len=%d, want 100", len(chunk))
				}
				return nil
			},
		},
	}
	seedActiveSession(t, "PROT-12345", "test-if")
	defer cleanupActiveSession("PROT-12345")
	s := novoServerRange(t, bacen)
	chunk := make([]byte, 100)
	req := doRangeRequest(t, http.MethodPut, "/v1/sta/range/", "PROT-12345",
		bytes.NewReader(chunk),
		map[string]string{"Content-Range": "bytes 0-99/1000", "X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeUpload(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want 200 (got %s)", w.Code, w.Body.String())
	}
}

// =============================================================================
// staRangeStatus tests
// =============================================================================

func TestStaRangeStatus_MissingProtocolo(t *testing.T) {
	s := novoServerRange(t, &stubBACEN{})
	req := doRangeRequest(t, http.MethodGet, "/v1/sta/range/", "", nil,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeStatus(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400 (missing protocolo)", w.Code)
	}
}

func TestStaRangeStatus_BACENError(t *testing.T) {
	// Sem seed de sessão em memória → BACEN falha → fallback DB → fallback memory → 404.
	// (Handler faz fallback em vez de retornar 502 — DB e memória estão vazios no teste.)
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			statusFunc: func(ctx context.Context, protocolo string) (*sta.UploadStatus, error) {
				return nil, errors.New("BACEN timeout")
			},
		},
	}
	s := novoServerRange(t, bacen)
	req := doRangeRequest(t, http.MethodGet, "/v1/sta/range/", "PROT-X", nil,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeStatus(w, req)

	// Esperado: 404 "sessão não encontrada" (handler tenta BACEN, depois DB, depois memory).
	if w.Code != http.StatusNotFound {
		t.Errorf("status=%d, want 404 (fallback BACEN → DB → memory)", w.Code)
	}
}

func TestStaRangeStatus_Success(t *testing.T) {
	bacen := &stubBACEN{
		stubRangeUploader: stubRangeUploader{
			statusFunc: func(ctx context.Context, protocolo string) (*sta.UploadStatus, error) {
				return &sta.UploadStatus{
					Protocolo: protocolo,
					Situacao:  sta.UploadSituacaoFinalizada,
				}, nil
			},
		},
	}
	s := novoServerRange(t, bacen)
	req := doRangeRequest(t, http.MethodGet, "/v1/sta/range/", "PROT-OK", nil,
		map[string]string{"X-IF-ID": "test-if"})
	w := httptest.NewRecorder()
	s.staRangeStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status=%d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "PROT-OK") {
		t.Errorf("body should echo protocolo: %s", w.Body.String())
	}
}
