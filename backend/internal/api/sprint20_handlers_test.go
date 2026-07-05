// Package api — Sprint 20 (v3.10.0) integration tests.
//
// Cobre:
//
//   - staDisponiveisHandler: GET /v1/sta/disponiveis
//   - staSituacaoHandler: POST /v1/sta/situacao
//   - Interface segregation: StubClient retorna 503, WSClient funciona
//
// Estratégia: WSClient contra httptest.Server local (não chama BACEN real).
package api_test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/radar"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// newTestServerWithWS monta um *api.Server com WSClient real contra um
// mockSTA httptest.Server (BACEN fake). Substitui newTestServer pra testes
// que precisam do read side funcional.
func newTestServerWithWS(t *testing.T, mockHandler http.Handler) (*api.Server, *httptest.Server, *sta.WSClient) {
	t.Helper()
	// Habilita dev mode (X-IF-ID em vez de JWT) — server_test.go padrão.
	t.Setenv("RADIANT_DEV_AUTH", "1")
	d := testutil.NewTestDB(t)

	testIFs := []string{"demo", "demo-bank", "system"}
	for i, ifID := range testIFs {
		cnpj := fmt.Sprintf("%08d", i+1)
		_, _ = d.Exec(`INSERT OR IGNORE INTO ifs (id, cnpj, nome, tipo, segmento, plano)
			VALUES (?, ?, ?, 'SCD', 'S5', 'pro')`, ifID, cnpj, "Test "+ifID)
	}

	bacen := httptest.NewServer(mockHandler)
	t.Cleanup(bacen.Close)

	wsClient, err := sta.NewWSClient(sta.WSConfig{
		BaseURL:           bacen.URL,
		User:              "12345/0001.fulano",
		Password:          "test-pwd",
		Timeout:           5 * time.Second,
		AllowInsecureHTTP: true,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
				ForceAttemptHTTP2: false,
			},
		},
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}

	schReg := schema.New(d)
	audSvc := audit.New(d)
	audLog := auditlog.New(d)
	radarSvc := radar.New(d, 1)

	srv := api.NewServer(d, schReg, audSvc, audLog, wsClient, radarSvc, nil, nil, nil)
	srv.ScanLimiter = radar.NewScanLimiter(1 * time.Minute)
	srv.ScanCache = radar.NewScanCache(5 * time.Minute)
	srv.AdminAuth = &radar.AdminAuth{Token: "test-admin-token"}
	srv.CadocListCache = schema.NewCadocListCache(5 * time.Minute)

	return srv, bacen, wsClient
}

// mockSTA20 é um handler configurável que simula BACEN STA WS pra testes
// dos handlers REST. Implementação local (a versão em internal/sta/ws_test.go
// é private ao package sta).
type mockSTA20 struct {
	handleGetDisponiveis func(w http.ResponseWriter, r *http.Request)
	handlePutSituacao    func(w http.ResponseWriter, r *http.Request)
}

func (m *mockSTA20) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
		w.Header().Set("WWW-Authenticate", `Basic realm="sta"`)
		http.Error(w, "missing Authorization", http.StatusUnauthorized)
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/arquivos/disponiveis":
		if m.handleGetDisponiveis != nil {
			m.handleGetDisponiveis(w, r)
			return
		}
	case r.Method == http.MethodPut && r.URL.Path == "/arquivos/situacao":
		if m.handlePutSituacao != nil {
			m.handlePutSituacao(w, r)
			return
		}
	}
	http.Error(w, "not implemented in mock: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// successDisponiveisHandler retorna mockSTA que responde 200 OK com 1 arquivo.
func successDisponiveisHandler() *mockSTA20 {
	return &mockSTA20{
		handleGetDisponiveis: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("dataHoraInicio") == "" {
				http.Error(w, "dataHoraInicio obrigatório", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
<DataHoraProximaConsulta>2024-12-01T10:00:00.001</DataHoraProximaConsulta>
<Arquivo>
<Protocolo>42</Protocolo>
<TipoArquivo>ACOS011</TipoArquivo>
<CodigoDocumento>3040</CodigoDocumento>
<Sistema>CCS</Sistema>
<TamanhoArquivo>1024</TamanhoArquivo>
<Hash>abc123</Hash>
<SituacaoAtual><Codigo>3</Codigo><Descricao>A receber</Descricao></SituacaoAtual>
<DataHoraDisponibilizacao>2024-11-30T10:00:00.000</DataHoraDisponibilizacao>
</Arquivo>
</Resultado>`))
		},
	}
}

// successAlterarSituacaoHandler retorna mockSTA que responde 204 No Content.
func successAlterarSituacaoHandler() *mockSTA20 {
	return &mockSTA20{
		handlePutSituacao: func(w http.ResponseWriter, r *http.Request) {
			if ct := r.Header.Get("Content-Type"); ct != "application/xml" {
				http.Error(w, "Content-Type esperado application/xml, got "+ct, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}

// withDevAuth injeta header X-IF-ID para passar middleware auth em dev mode.
// (Validação 27 — getIfID prioriza JWT mas em dev mode X-IF-ID basta.)
func withDevAuth(r *http.Request, ifID string) *http.Request {
	r.Header.Set("X-IF-ID", ifID)
	return r
}

// TestHandler_Disponiveis_OK — Sprint 20: GET /v1/sta/disponiveis happy path.
func TestHandler_Disponiveis_OK(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successDisponiveisHandler())

	req := httptest.NewRequest(http.MethodGet,
		"/v1/sta/disponiveis?dataHoraInicio=2024-11-01T00:00:00.000", nil)
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Arquivos []struct {
			Protocolo     string `json:"protocolo"`
			TipoArquivo   string `json:"tipo_arquivo"`
			SituacaoAtual string `json:"situacao_atual"`
		} `json:"arquivos"`
		DataHoraProximaConsulta string `json:"data_hora_proxima_consulta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v (body=%s)", err, rec.Body.String())
	}
	if len(resp.Arquivos) != 1 {
		t.Errorf("len(arquivos) = %d, esperado 1", len(resp.Arquivos))
	}
	if resp.Arquivos[0].Protocolo != "42" {
		t.Errorf("protocolo = %q, esperado 42", resp.Arquivos[0].Protocolo)
	}
	if resp.Arquivos[0].SituacaoAtual != "A receber" {
		t.Errorf("situacao_atual = %q, esperado 'A receber'", resp.Arquivos[0].SituacaoAtual)
	}
}

// TestHandler_Disponiveis_DataHoraVazia — 400 quando obrigatório ausente.
func TestHandler_Disponiveis_DataHoraVazia(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successDisponiveisHandler())

	req := httptest.NewRequest(http.MethodGet, "/v1/sta/disponiveis", nil)
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "dataHoraInicio obrigatório") {
		t.Errorf("body deve mencionar dataHoraInicio, got %q", rec.Body.String())
	}
}

// TestHandler_Disponiveis_BACEN400 — 400 do BACEN → 400 do handler.
func TestHandler_Disponiveis_BACEN400(t *testing.T) {
	mock := successDisponiveisHandler()
	mock.handleGetDisponiveis = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>dataHoraInicio em formato inválido</Descricao></Erro></Resultado>`))
	}
	srv, _, _ := newTestServerWithWS(t, mock)

	req := httptest.NewRequest(http.MethodGet,
		"/v1/sta/disponiveis?dataHoraInicio=formato-invalido", nil)
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado 400 (body=%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "BACEN rejeitou") {
		t.Errorf("body deve mencionar rejeição BACEN, got %q", rec.Body.String())
	}
}

// TestHandler_Situacao_OK — POST /v1/sta/situacao happy path → 204.
func TestHandler_Situacao_OK(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successAlterarSituacaoHandler())

	body := []byte(`{"protocolos":["1","2"],"situacao":"REC"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sta/situacao", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, esperado 204 (body=%s)", rec.Code, rec.Body.String())
	}
}

// TestHandler_Situacao_BodyInvalido — 400 quando JSON malformado.
func TestHandler_Situacao_BodyInvalido(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successAlterarSituacaoHandler())

	body := []byte(`{invalid json`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sta/situacao", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado 400", rec.Code)
	}
}

// TestHandler_Situacao_ProtocolosVazios — 400 quando lista vazia.
func TestHandler_Situacao_ProtocolosVazios(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successAlterarSituacaoHandler())

	body := []byte(`{"protocolos":[],"situacao":"REC"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sta/situacao", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado 400", rec.Code)
	}
}

// TestHandler_Situacao_ValorInvalido — 400 quando situacao != A_REC/REC.
func TestHandler_Situacao_ValorInvalido(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successAlterarSituacaoHandler())

	body := []byte(`{"protocolos":["1"],"situacao":"FOO"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/sta/situacao", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, esperado 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "A_REC") {
		t.Errorf("body deve mencionar valores válidos, got %q", rec.Body.String())
	}
}

// TestHandler_StubBackend_503 — interface segregation: StubClient → 503.
func TestHandler_StubBackend_503(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet,
		"/v1/sta/disponiveis?dataHoraInicio=2024-11-01T00:00:00.000", nil)
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, esperado 503 (StubClient não implementa ReadClient)",
			rec.Code)
	}
}

// TestHandler_Disponiveis_CrossTenant_403 — Validação 41 finding F-S20-41:
// caller passa dependencia != tenant autenticado → 403 via enforceSameIF.
// Sem isso, IF_A poderia listar arquivos de IF_B via query param.
func TestHandler_Disponiveis_CrossTenant_403(t *testing.T) {
	srv, _, _ := newTestServerWithWS(t, successDisponiveisHandler())

	req := httptest.NewRequest(http.MethodGet,
		"/v1/sta/disponiveis?dataHoraInicio=2024-11-01T00:00:00.000&dependencia=OTHER_TENANT", nil)
	req = withDevAuth(req, "demo-bank")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, esperado 403 (cross-tenant bloqueado por enforceSameIF)",
			rec.Code)
	}
}
