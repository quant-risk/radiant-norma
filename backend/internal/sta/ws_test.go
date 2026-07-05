// Package sta — testes do WSClient (Sprint 18 / v3.8.0).
//
// Estratégia: mock do BACEN STA Web Services via httptest.Server. Cada teste
// cobre uma classe de comportamento documentada nas Tabelas 5-8 do manual
// oficial e a Seção 2.6 (limits).
//
// NÃO testamos contra BACEN real (sem credenciais Sisbacen em dev). Esses
// testes provam conformidade com a spec documentada; smoke contra Bacen real
// é Sprint 19+ (quando credenciais forem obtidas).
package sta

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockSTA é um handler configurável que simula o BACEN STA WS para fins
// de teste.
//
// Roteamento:
//
//	POST /arquivos                                       → handlePost
//	PUT  /arquivos/{protocolo}/conteudo                  → handlePut
//	GET  /arquivos/{protocolo}/conteudo                  → handleGetConteudo
//	GET  /arquivos/{protocolo}/posicaoupload             → handleGetPosicao
//
// Se requireBasicAuth for true, retorna 401 quando header Authorization
// ausente (espelha comportamento real do BACEN).
type mockSTA struct {
	requireBasicAuth  bool
	handlePost        func(w http.ResponseWriter, r *http.Request)
	handlePut         func(w http.ResponseWriter, r *http.Request, protocolo string)
	handleGetConteudo func(w http.ResponseWriter, r *http.Request, protocolo string)
	handleGetPosicao  func(w http.ResponseWriter, r *http.Request, protocolo string)
}

func (m *mockSTA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.requireBasicAuth {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="sta"`)
			http.Error(w, "missing Authorization", http.StatusUnauthorized)
			return
		}
	}

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/arquivos":
		if m.handlePost != nil {
			m.handlePost(w, r)
			return
		}
	case r.Method == http.MethodPut &&
		strings.HasPrefix(r.URL.Path, "/arquivos/") &&
		strings.HasSuffix(r.URL.Path, "/conteudo"):
		protocolo := strings.TrimPrefix(r.URL.Path, "/arquivos/")
		protocolo = strings.TrimSuffix(protocolo, "/conteudo")
		if m.handlePut != nil {
			m.handlePut(w, r, protocolo)
			return
		}
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/arquivos/") &&
		strings.HasSuffix(r.URL.Path, "/posicaoupload"):
		protocolo := strings.TrimPrefix(r.URL.Path, "/arquivos/")
		protocolo = strings.TrimSuffix(protocolo, "/posicaoupload")
		if m.handleGetPosicao != nil {
			m.handleGetPosicao(w, r, protocolo)
			return
		}
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/arquivos/") &&
		strings.HasSuffix(r.URL.Path, "/conteudo"):
		protocolo := strings.TrimPrefix(r.URL.Path, "/arquivos/")
		protocolo = strings.TrimSuffix(protocolo, "/conteudo")
		if m.handleGetConteudo != nil {
			m.handleGetConteudo(w, r, protocolo)
			return
		}
	}
	http.Error(w, "not implemented in mock: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// newMockSTA cria mockSTA server + WSClient apontando pra ele.
func newMockSTA(t *testing.T, m *mockSTA) (*httptest.Server, *WSClient) {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)

	client, err := NewWSClient(WSConfig{
		BaseURL:           srv.URL,
		User:              "12345/0001.fulano",
		Password:          "senha-test",
		Timeout:           5 * time.Second,
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewWSClient falhou: %v", err)
	}
	return srv, client
}

// successHandler retorna mockSTA "happy path". Pós-POST retorna protocolo
// 123 e PUT aceita qualquer coisa.
func successHandler() *mockSTA {
	return &mockSTA{
		requireBasicAuth: true,
		handlePost: func(w http.ResponseWriter, r *http.Request) {
			if ct := r.Header.Get("Content-Type"); ct != "application/xml" {
				http.Error(w, "Content-Type esperado application/xml, got "+ct, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado xmlns:atom="http://www.w3.org/2005/Atom">
<Protocolo>123</Protocolo>
<atom:link href="http://example/arquivos/123/conteudo" rel="conteudo" type="application/octet-stream"/>
</Resultado>`))
		},
		handlePut: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			if protocolo != "123" {
				http.Error(w, "protocolo inesperado: "+protocolo, http.StatusBadRequest)
				return
			}
			if r.ContentLength <= 0 {
				http.Error(w, "ContentLength esperado >0", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		},
	}
}

func sampleSubmission() *Submission {
	return &Submission{
		CadocCode: "3040",
		DataBase:  "2024-12",
		CNPJ:      "demo-bank",
		XML:       "<root><Doc3040/></root>",
	}
}

// ============================================================
// NewWSClient tests
// ============================================================

func TestNewWSClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WSConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws",
				User:     "12345/0001.fulano",
				Password: "senha",
			},
		},
		{
			name: "empty baseURL",
			cfg: WSConfig{
				User:     "x.y",
				Password: "p",
			},
			wantErr: "BaseURL requerida",
		},
		{
			name: "http not https",
			cfg: WSConfig{
				BaseURL:  "http://insecure/staws",
				User:     "x.y",
				Password: "p",
			},
			wantErr: "deve usar HTTPS",
		},
		{
			name: "trailing slash",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws/",
				User:     "x.y",
				Password: "p",
			},
			wantErr: "não deve terminar com /",
		},
		{
			name: "user sem dot",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws",
				User:     "123450001",
				Password: "p",
			},
			wantErr: "UUUUUDDDD.operador",
		},
		{
			name: "user formato Sisbacen inválido",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws",
				User:     "12345.0001.fulano",
				Password: "p",
			},
			wantErr: "formato Sisbacen exato",
		},
		{
			name: "empty password",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws",
				User:     "123450001.fulano",
				Password: "",
			},
			wantErr: "Password requerida",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWSClient(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("esperava sucesso, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava erro contendo %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("esperava erro contendo %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestNewWSClient_DefaultTimeout(t *testing.T) {
	c, err := NewWSClient(WSConfig{
		BaseURL:  "https://sta-h.bcb.gov.br/staws",
		User:     "123450001.fulano",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}
	if c.cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout default = %v, esperado 30s", c.cfg.Timeout)
	}
	if c.cfg.HTTPClient == nil {
		t.Error("HTTPClient nil após NewWSClient")
	}
}

// ============================================================
// Submit — happy path + behavior variants
// ============================================================

func TestSubmit_HappyPath(t *testing.T) {
	_, client := newMockSTA(t, successHandler())

	res, err := client.Submit(context.Background(), sampleSubmission())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !res.Accepted {
		t.Errorf("esperava Accepted=true, got false (rejection=%+v)", res.Rejection)
	}
	if res.ProtocolSTA != "123" {
		t.Errorf("ProtocolSTA = %q, esperado 123", res.ProtocolSTA)
	}
}

func TestSubmit_EmptySubmission(t *testing.T) {
	_, client := newMockSTA(t, successHandler())
	res, err := client.Submit(context.Background(), &Submission{})
	if err == nil {
		t.Fatalf("esperava erro em submission vazio, got res=%+v", res)
	}
	if !strings.Contains(err.Error(), "STA submission vazia") {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestSubmit_UsesZipWhenProvided(t *testing.T) {
	var seenHash string
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		var got struct {
			Hash string `xml:"Hash"`
		}
		if err := xml.NewDecoder(r.Body).Decode(&got); err != nil {
			http.Error(w, "bad xml: "+err.Error(), http.StatusBadRequest)
			return
		}
		seenHash = got.Hash
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Resultado><Protocolo>1</Protocolo></Resultado>`))
	}
	mock.handlePut = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.WriteHeader(http.StatusOK)
	}
	_, client := newMockSTA(t, mock)

	zipData := []byte("PAYLOAD ZIP")
	sub := &Submission{
		CadocCode: "3040",
		DataBase:  "2024-12",
		CNPJ:      "demo",
		XML:       "<root/>",
		Zip:       zipData,
	}
	_, err := client.Submit(context.Background(), sub)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	expected := sha256.Sum256(zipData)
	expectedHex := hex.EncodeToString(expected[:])
	if seenHash != expectedHex {
		t.Errorf("Hash enviado = %q, esperado SHA-256 do ZIP = %q", seenHash, expectedHex)
	}
}

// ============================================================
// Submit — error mapping (Tabela 5-8 do manual)
// ============================================================

func TestSubmit_400_IdentificadorInvalido(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>Parâmetro 'IdentificadorDocumento' inválido</Descricao></Erro></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Submit(context.Background(), sampleSubmission())
	if err == nil {
		t.Fatal("esperava erro em 400")
	}
	if !strings.Contains(err.Error(), "IdentificadorDocumento") {
		t.Errorf("erro deve conter mensagem BACEN, got %v", err)
	}
}

func TestSubmit_403_UsuarioNaoAutorizado(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>403</Codigo><Descricao>Usuário não autorizado a transmitir o arquivo 9999</Descricao></Erro></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Submit(context.Background(), sampleSubmission())
	if err == nil {
		t.Fatal("esperava erro em 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("erro deve mencionar 403, got %v", err)
	}
	if !strings.Contains(err.Error(), "não autorizado") {
		t.Errorf("erro deve conter descrição BACEN, got %v", err)
	}
}

func TestSubmit_ProtocolThenUpload403(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Resultado><Protocolo>456</Protocolo></Resultado>`))
	}
	mock.handlePut = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>403</Codigo><Descricao>Protocolo não pertence à instituição</Descricao></Erro></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	res, err := client.Submit(context.Background(), sampleSubmission())
	// Submit retorna (Result, nil) — protocolo existe, falha de upload mapeada.
	if err != nil {
		t.Fatalf("Submit deveria retornar Result+nil em upload falho, got err: %v", err)
	}
	if res.ProtocolSTA != "456" {
		t.Errorf("ProtocolSTA = %q, esperado 456 (preservado para forensic)", res.ProtocolSTA)
	}
	if res.Accepted {
		t.Error("Accepted deveria ser false em upload falho")
	}
	if res.Rejection == nil {
		t.Fatal("Rejection deveria estar populado")
	}
	if res.Rejection.Code != "UPLOAD_FAILED" {
		t.Errorf("Rejection.Code = %q, esperado UPLOAD_FAILED", res.Rejection.Code)
	}
	if !strings.Contains(res.Rejection.Message, "não pertence") {
		t.Errorf("Rejection.Message deve conter descrição BACEN, got %q", res.Rejection.Message)
	}
}

func TestSubmit_HashMismatch(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Resultado><Protocolo>789</Protocolo></Resultado>`))
	}
	mock.handlePut = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>Parâmetro 'Hash' não confere com conteúdo enviado</Descricao></Erro></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	res, err := client.Submit(context.Background(), sampleSubmission())
	// Fase 1 OK → protocolo existe. Fase 2 falha → Submit retorna
	// Result+nil com Rejection populado.
	if err != nil {
		t.Fatalf("Submit deveria retornar Result+nil em upload falho, got err: %v", err)
	}
	if res == nil {
		t.Fatal("res é nil")
	}
	if res.ProtocolSTA != "789" {
		t.Errorf("ProtocolSTA = %q, esperado 789", res.ProtocolSTA)
	}
	if res.Accepted {
		t.Error("Accepted deveria ser false em hash mismatch")
	}
	if res.Rejection == nil {
		t.Fatal("Rejection deveria estar populado")
	}
	if !strings.Contains(res.Rejection.Message, "Hash") {
		t.Errorf("Rejection.Message deve mencionar Hash, got %q", res.Rejection.Message)
	}
}

// ============================================================
// Submit — edge cases / defense in depth
// ============================================================

func TestSubmit_ContextCanceled(t *testing.T) {
	mock := successHandler()
	// Handler devolve imediatamente (não bloqueia) — o test valida que
	// mesmo assim o client vê o context timeout na sua chamada. O servidor
	// não precisa simular lentidão; basta cancelar o client.
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_, client := newMockSTA(t, mock)

	// Cria context e cancela ANTES de chamar Submit — cenário extremo onde
	// request nem sai do client.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Submit(ctx, sampleSubmission())
	if err == nil {
		t.Fatal("esperava erro de context cancelado")
	}
	if !strings.Contains(err.Error(), "context") &&
		!strings.Contains(err.Error(), "canceled") &&
		!strings.Contains(err.Error(), "Service Unavailable") {
		t.Errorf("erro deve mencionar context/cancelamento ou 503, got %v", err)
	}
}

func TestSubmit_EmptyProtocolInResponse(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Resultado><Protocolo></Protocolo></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	res, err := client.Submit(context.Background(), sampleSubmission())
	// Fase 1 falha (protocolo vazio) → Submit retorna (nil, err).
	if err == nil {
		t.Fatalf("esperava err não-nil, got res=%+v", res)
	}
	if res != nil {
		t.Errorf("res deveria ser nil em erro de fase 1, got %+v", res)
	}
	if !strings.Contains(err.Error(), "<Protocolo>") {
		t.Errorf("err deve mencionar tag Protocolo vazia, got %v", err)
	}
}

func TestSubmit_MalformedErrorBody(t *testing.T) {
	mock := successHandler()
	mock.handlePost = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not xml at all, just garbage bytes`))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Submit(context.Background(), sampleSubmission())
	if err == nil {
		t.Fatal("esperava erro de 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("erro deve mencionar 500, got %v", err)
	}
}

// TestBasicAuthHeader_VerificaHeader — prova que o header montado é base64
// correto de "user:pass".
func TestBasicAuthHeader_Formato(t *testing.T) {
	c, _ := NewWSClient(WSConfig{
		BaseURL:  "https://example.invalid/staws",
		User:     "12345/0001.fulano",
		Password: "minha-senha-secreta",
		Timeout:  1 * time.Second,
	})
	header := c.basicAuthHeader()
	if !strings.HasPrefix(header, "Basic ") {
		t.Fatalf("header deve começar com 'Basic ', got %q", header)
	}
	encoded := strings.TrimPrefix(header, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode falhou: %v", err)
	}
	want := "12345/0001.fulano:minha-senha-secreta"
	if string(decoded) != want {
		t.Errorf("decoded = %q, esperado %q", string(decoded), want)
	}
}

// TestNewWSClient_AcceptsFormatosUsuarioVariados — Validação 39 (F-2):
// regex aceita tanto formato concatenado (UUUUUDDDD) quanto com slash
// (UUUUU/DDDD) — ambos comuns em docs BACEN.
func TestNewWSClient_AcceptsFormatosUsuarioVariados(t *testing.T) {
	for _, user := range []string{
		"123450001.fulano",   // concatenado
		"12345/0001.fulano",  // com slash
		"00000/0000.root",    // zeros OK
		"99999/1234.admin-x", // dash no operador
	} {
		_, err := NewWSClient(WSConfig{
			BaseURL:  "https://sta-h.bcb.gov.br/staws",
			User:     user,
			Password: "p",
		})
		if err != nil {
			t.Errorf("user %q deveria ser aceito, got %v", user, err)
		}
	}
}

// TestNewWSClient_ForceHTTP1 — Validação 39 (F-3): Transport deve
// desabilitar HTTP/2 (Manual Seção 2.5 — BACEN só suporta HTTP/1.1).
func TestNewWSClient_ForceHTTP1(t *testing.T) {
	c, err := NewWSClient(WSConfig{
		BaseURL:  "https://sta-h.bcb.gov.br/staws",
		User:     "123450001.fulano",
		Password: "p",
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}
	transport, ok := c.cfg.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("HTTPClient.Transport não é *http.Transport")
	}
	if transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 deveria ser false (BACEN só suporta HTTP/1.1)")
	}
	if transport.TLSClientConfig == nil {
		t.Error("TLSClientConfig deveria estar setado (TLS 1.2 mínimo)")
	}
	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %v, esperado TLS 1.2", transport.TLSClientConfig.MinVersion)
	}
}

// TestSubmit_RespostaEnormeCapada — Validação 39 (F-1): BACEN misbehaving
// retornando 20 MiB de body após protocolo válido deve ser capado a 10 MiB
// sem crashar.
func TestSubmit_RespostaEnormeCapada(t *testing.T) {
	mock := &mockSTA{
		requireBasicAuth: true,
		handlePost: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusCreated)
			// Header válido contendo <Protocolo>1</Protocolo>.
			header := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Resultado><Protocolo>1</Protocolo></Resultado>`)
			w.Write(header)
			// Padding: 20 MiB de espaço — acima do cap de 10 MiB.
			padding := bytes.Repeat([]byte(" "), 20<<20)
			w.Write(padding)
		},
		handlePut: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			w.WriteHeader(http.StatusOK)
		},
	}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	client, err := NewWSClient(WSConfig{
		BaseURL:           srv.URL,
		User:              "12345/0001.fulano",
		Password:          "senha",
		Timeout:           5 * time.Second,
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}

	// Verifica que NÃO crasha. Pode retornar sucesso (se parsear
	// parcial) ou erro de parse (se truncado no meio do XML). Ambos OK
	// — o importante é a defesa contra OOM.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic em resposta gigante: %v", r)
		}
	}()
	_, _ = client.Submit(context.Background(), &Submission{
		CadocCode: "3040",
		DataBase:  "2024-12",
		CNPJ:      "demo",
		XML:       "<root/>",
	})
}

// ============================================================
// Sprint 19 (v3.9.0) — read side: StatusUpload + Download
//
// Spec: SPRINT_19_RESEARCH.md §2.1 (Seção 5.3.1 manual BACEN) + §2.2
// (Seção 6.1.1 manual BACEN). Cada teste abaixo mapeia uma linha da tabela
// de erros esperada ou um caso happy path.
// ============================================================

// sucessoPosicaoUploadHandler retorna mockSTA configurado para responder
// 200 OK com XML de posicaoupload completo (RangesRecebidos + Situacao).
// Content-Type do GET omitido conforme manual §5.3.1.
func sucessoPosicaoUploadHandler() *mockSTA {
	return &mockSTA{
		requireBasicAuth: true,
		handleGetPosicao: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			if r.Header.Get("Content-Type") != "" {
				http.Error(w, "Content-Type deveria ser omitido (manual §5.3.1)", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
<Protocolo>123</Protocolo>
<RangesRecebidos>0-3;5-8;100-199</RangesRecebidos>
<Situacao>Transmissão pendente</Situacao>
</Resultado>`))
		},
	}
}

// TestWSClient_StatusUpload_HappyPath — Seção 5.3.1.
func TestWSClient_StatusUpload_HappyPath(t *testing.T) {
	_, client := newMockSTA(t, sucessoPosicaoUploadHandler())

	status, err := client.StatusUpload(context.Background(), "123")
	if err != nil {
		t.Fatalf("StatusUpload: %v", err)
	}
	if status == nil {
		t.Fatal("status nil")
	}
	if status.Protocolo != "123" {
		t.Errorf("Protocolo = %q, esperado 123", status.Protocolo)
	}
	if got, want := len(status.RangesRecebidos), 3; got != want {
		t.Errorf("len(RangesRecebidos) = %d, esperado %d", got, want)
	}
	if status.RangesRecebidos[0].Start != 0 || status.RangesRecebidos[0].End != 3 {
		t.Errorf("RangesRecebidos[0] = %+v, esperado {0,3}", status.RangesRecebidos[0])
	}
	if status.RangesRecebidos[1].Start != 5 || status.RangesRecebidos[1].End != 8 {
		t.Errorf("RangesRecebidos[1] = %+v, esperado {5,8}", status.RangesRecebidos[1])
	}
	if status.RangesRecebidos[2].Start != 100 || status.RangesRecebidos[2].End != 199 {
		t.Errorf("RangesRecebidos[2] = %+v, esperado {100,199}", status.RangesRecebidos[2])
	}
	if status.Situacao != UploadUploadPendente {
		t.Errorf("Situacao = %v, esperado UploadUploadPendente (%q)",
			status.Situacao, status.Situacao.String())
	}
	if status.SituacaoRaw != "Transmissão pendente" {
		t.Errorf("SituacaoRaw = %q, esperado %q", status.SituacaoRaw, "Transmissão pendente")
	}
}

// TestWSClient_StatusUpload_RangesEmpty — RangesRecebidos vazio + Situacao
// "Transmissão não iniciada" (caso real de novo protocolo).
func TestWSClient_StatusUpload_RangesEmpty(t *testing.T) {
	mock := sucessoPosicaoUploadHandler()
	mock.handleGetPosicao = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
<Protocolo>456</Protocolo>
<RangesRecebidos></RangesRecebidos>
<Situacao>Transmissão não iniciada</Situacao>
</Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	status, err := client.StatusUpload(context.Background(), "456")
	if err != nil {
		t.Fatalf("StatusUpload: %v", err)
	}
	if len(status.RangesRecebidos) != 0 {
		t.Errorf("RangesRecebidos deveria ser vazio, got %+v", status.RangesRecebidos)
	}
	if status.Situacao != UploadSituacaoNaoIniciada {
		t.Errorf("Situacao = %v, esperado UploadSituacaoNaoIniciada", status.Situacao)
	}
}

// TestWSClient_StatusUpload_403 — protocolo de outra IF (Seção 5.3.1).
func TestWSClient_StatusUpload_403(t *testing.T) {
	mock := sucessoPosicaoUploadHandler()
	mock.handleGetPosicao = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>403</Codigo><Descricao>Protocolo não pertence à instituição</Descricao></Erro></Resultado>`))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.StatusUpload(context.Background(), "999")
	if err == nil {
		t.Fatal("esperava erro em 403")
	}
	var staErr *STAError
	if !errorsAs(err, &staErr) {
		t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
	}
	if staErr.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d, esperado 403", staErr.StatusCode)
	}
	if !strings.Contains(staErr.Message, "não pertence") {
		t.Errorf("Message deve mencionar 'não pertence', got %q", staErr.Message)
	}
	if staErr.Protocolo != "999" {
		t.Errorf("Protocolo ecoado = %q, esperado 999", staErr.Protocolo)
	}
}

// TestWSClient_StatusUpload_BadXMLFallback — 200 OK mas body não parseia.
// Esperado: erro de parse (não STAError), porque status HTTP é 200.
func TestWSClient_StatusUpload_BadXMLFallback(t *testing.T) {
	mock := sucessoPosicaoUploadHandler()
	mock.handleGetPosicao = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid xml`))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.StatusUpload(context.Background(), "123")
	if err == nil {
		t.Fatal("esperava erro de parse")
	}
	if !strings.Contains(err.Error(), "parse posicaoupload") {
		t.Errorf("erro deve mencionar parse posicaoupload, got %v", err)
	}
}

// TestWSClient_StatusUpload_EmptyProtocolo — sanity check defensivo.
func TestWSClient_StatusUpload_EmptyProtocolo(t *testing.T) {
	c, _ := NewWSClient(WSConfig{
		BaseURL:           "http://127.0.0.1:0",
		User:              "123450001.x",
		Password:          "p",
		AllowInsecureHTTP: true,
	})
	_, err := c.StatusUpload(context.Background(), "")
	if err == nil {
		t.Fatal("esperava erro em protocolo vazio")
	}
	if !strings.Contains(err.Error(), "protocolo requerido") {
		t.Errorf("erro deveria mencionar 'protocolo requerido', got %v", err)
	}
}

// sucessoDownloadHandler retorna mockSTA configurado para responder 200 OK
// com headers completos (ETag, Last-Modified, X-Content-Hash) + body ZIP.
//
// `content` é o body que será enviado (default "ZIP binário fake").
// Se contentHeader hash não bater com SHA-256(content), use
// sucessoDownloadHandlerCustomHash pra simular bug BACEN.
func sucessoDownloadHandler() (*mockSTA, []byte) {
	content := []byte("PK\x03\x04 ZIP binário fake")
	sum := sha256.Sum256(content)
	hash := "SHA-256 " + hex.EncodeToString(sum[:])

	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			if r.Header.Get("Content-Type") != "" {
				http.Error(w, "Content-Type deveria ser omitido (manual §6.1.1)", http.StatusBadRequest)
				return
			}
			w.Header().Set("ETag", `"abc123"`)
			w.Header().Set("Last-Modified", "Thu, 01 Dec 2022 12:00:00 GMT")
			w.Header().Set("X-Content-Hash", hash)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(content)
		},
	}
	return mock, content
}

// TestWSClient_Download_HappyPath — Seção 6.1.1.
func TestWSClient_Download_HappyPath(t *testing.T) {
	mock, expectedContent := sucessoDownloadHandler()
	_, client := newMockSTA(t, mock)

	res, err := client.Download(context.Background(), "123")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if res == nil {
		t.Fatal("res nil")
	}
	if !bytes.Equal(res.Conteudo, expectedContent) {
		t.Errorf("Conteúdo = %q, esperado %q", res.Conteudo, expectedContent)
	}
	if res.ContentHash == "" {
		t.Error("ContentHash vazio")
	}
	wantSum := sha256.Sum256(expectedContent)
	wantHex := hex.EncodeToString(wantSum[:])
	if res.ContentHash != wantHex {
		t.Errorf("ContentHash = %q, esperado %q", res.ContentHash, wantHex)
	}
	if res.ETag != `"abc123"` {
		t.Errorf("ETag = %q, esperado %q", res.ETag, `"abc123"`)
	}
	if res.LastModified != "Thu, 01 Dec 2022 12:00:00 GMT" {
		t.Errorf("LastModified = %q", res.LastModified)
	}
	if !strings.HasPrefix(res.ContentHashHeader, "SHA-256 ") {
		t.Errorf("ContentHashHeader = %q, esperado prefixo SHA-256", res.ContentHashHeader)
	}
}

// TestWSClient_Download_HashMismatch — X-Content-Hash com valor errado
// (cenário: BACEN bugou ou proxy transparente corrompeu body).
func TestWSClient_Download_HashMismatch(t *testing.T) {
	mock, _ := sucessoDownloadHandler()
	mock.handleGetConteudo = func(w http.ResponseWriter, r *http.Request, protocolo string) {
		// Hash intencionalmente errado (todos zeros).
		w.Header().Set("X-Content-Hash", "SHA-256 0000000000000000000000000000000000000000000000000000000000000000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body real"))
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Download(context.Background(), "123")
	if err == nil {
		t.Fatal("esperava erro de hash mismatch")
	}
	if !errorsIs(err, ErrContentHashMismatch) {
		t.Fatalf("erro deveria ser ErrContentHashMismatch, got %v", err)
	}
}

// TestWSClient_Download_404 — protocolo inexistente (Seção 6.1.1).
func TestWSClient_Download_404(t *testing.T) {
	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>404</Codigo><Descricao>Protocolo não encontrado</Descricao></Erro></Resultado>`))
		},
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Download(context.Background(), "999")
	if err == nil {
		t.Fatal("esperava erro em 404")
	}
	var staErr *STAError
	if !errorsAs(err, &staErr) {
		t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
	}
	if staErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, esperado 404", staErr.StatusCode)
	}
	if !strings.Contains(staErr.Message, "não encontrado") {
		t.Errorf("Message = %q", staErr.Message)
	}
}

// TestWSClient_Download_410 — arquivo não disponível (Seção 6.1.1).
func TestWSClient_Download_410(t *testing.T) {
	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			w.WriteHeader(http.StatusGone)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>410</Codigo><Descricao>O arquivo não está disponível para download</Descricao></Erro></Resultado>`))
		},
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Download(context.Background(), "888")
	if err == nil {
		t.Fatal("esperava erro em 410")
	}
	var staErr *STAError
	if !errorsAs(err, &staErr) {
		t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
	}
	if staErr.StatusCode != http.StatusGone {
		t.Errorf("StatusCode = %d, esperado 410", staErr.StatusCode)
	}
}

// TestWSClient_Download_BodyTooLarge — BACEN misbehaving: body > 100 MiB
// deve ser capado, não estourar memória. Esperado *STAError 413.
func TestWSClient_Download_BodyTooLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skip em -short; aloca 100+ MiB")
	}
	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			// Content-Length FALSO (BACEN bug): diz 1 KB mas envia 120 MiB.
			// Cap do cliente é em bytes lidos, não em Content-Length.
			hash := strings.Repeat("0", 64)
			w.Header().Set("X-Content-Hash", "SHA-256 "+hash)
			w.WriteHeader(http.StatusOK)
			// 120 MiB (> maxDownloadBodyBytes de 100 MiB).
			padding := bytes.Repeat([]byte("X"), 120<<20)
			_, _ = w.Write(padding)
		},
	}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	client, err := NewWSClient(WSConfig{
		BaseURL:           srv.URL,
		User:              "12345/0001.fulano",
		Password:          "senha",
		Timeout:           30 * time.Second,
		AllowInsecureHTTP: true,
	})
	if err != nil {
		t.Fatalf("NewWSClient: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic em body gigante: %v", r)
		}
	}()

	_, err = client.Download(context.Background(), "123")
	if err == nil {
		t.Fatal("esperava erro de body grande")
	}
	var staErr *STAError
	if !errorsAs(err, &staErr) {
		t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
	}
	if staErr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("StatusCode = %d, esperado 413", staErr.StatusCode)
	}
	if !strings.Contains(staErr.Message, "cap") {
		t.Errorf("Message deve mencionar cap defensivo, got %q", staErr.Message)
	}
}

// TestWSClient_Download_HeaderMalformed — BACEN mudou formato do header
// (defesa contra versionamento futuro).
func TestWSClient_Download_HeaderMalformed(t *testing.T) {
	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			// Formato errado: MD5 em vez de SHA-256 (cenário hipotético BACEN v2.0).
			w.Header().Set("X-Content-Hash", "MD5 abcdef0123456789")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("body"))
		},
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Download(context.Background(), "123")
	if err == nil {
		t.Fatal("esperava erro de header malformado")
	}
	if !errorsIs(err, ErrContentHashHeaderMalformed) {
		t.Fatalf("erro deveria ser ErrContentHashHeaderMalformed, got %v", err)
	}
}

// TestWSClient_Download_MissingHeader — BACEN esqueceu de mandar header
// (defesa contra regressão).
func TestWSClient_Download_MissingHeader(t *testing.T) {
	mock := &mockSTA{
		requireBasicAuth: true,
		handleGetConteudo: func(w http.ResponseWriter, r *http.Request, protocolo string) {
			// Sem X-Content-Hash.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("body"))
		},
	}
	_, client := newMockSTA(t, mock)

	_, err := client.Download(context.Background(), "123")
	if err == nil {
		t.Fatal("esperava erro de header ausente")
	}
	var staErr *STAError
	if !errorsAs(err, &staErr) {
		t.Fatalf("erro deveria ser *STAError, got %T: %v", err, err)
	}
	if staErr.Code != "MISSING_X_CONTENT_HASH" {
		t.Errorf("Code = %q, esperado MISSING_X_CONTENT_HASH", staErr.Code)
	}
}

// TestWSClient_Download_EmptyProtocolo — sanity check defensivo.
func TestWSClient_Download_EmptyProtocolo(t *testing.T) {
	c, _ := NewWSClient(WSConfig{
		BaseURL:           "http://127.0.0.1:0",
		User:              "123450001.x",
		Password:          "p",
		AllowInsecureHTTP: true,
	})
	_, err := c.Download(context.Background(), "")
	if err == nil {
		t.Fatal("esperava erro em protocolo vazio")
	}
	if !strings.Contains(err.Error(), "protocolo requerido") {
		t.Errorf("erro deveria mencionar 'protocolo requerido', got %v", err)
	}
}

// ============================================================
// Helpers para tests Sprint 19 (errorsAs/errorsIs wrapped)
// ============================================================
//
// Go 1.13+ tem errors.As/Is mas o file usa imports compactos — encapsulamos
// aqui pra evitar adicionar mais imports e manter diff pequeno.

func errorsAs(err error, target interface{}) bool {
	for err != nil {
		if e, ok := err.(*STAError); ok {
			if t, ok := target.(**STAError); ok {
				*t = e
				return true
			}
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// ============================================================
// Unit tests — pure functions (parseRanges, parseUploadSituacao, parseXContentHash)
// ============================================================

func TestParseRanges_Cases(t *testing.T) {
	tests := []struct {
		in   string
		want []Range
	}{
		{"", nil},
		{"0-3", []Range{{0, 3}}},
		{"0-3;5-8", []Range{{0, 3}, {5, 8}}},
		{"100-199;200-299", []Range{{100, 199}, {200, 299}}},
		// Defense: lixo descartado silenciosamente.
		{"abc-def", nil},
		{"0-3;;5-8", []Range{{0, 3}, {5, 8}}},
		{"0-3;malformed", []Range{{0, 3}}},
		{"5-3", nil},  // end < start → descarta
		{"-1-3", nil}, // start negativo (parse fail) → descarta
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := parseRanges(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, esperado %d (got=%+v)", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %+v, esperado %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseUploadSituacao_Cases(t *testing.T) {
	tests := []struct {
		in   string
		want UploadSituacao
	}{
		{"", UploadSituacaoUnknown},
		{"Transmissão não iniciada", UploadSituacaoNaoIniciada},
		{"Transmissão pendente", UploadUploadPendente},
		{"Transmissão finalizada", UploadSituacaoFinalizada},
		{"valor futuro desconhecido", UploadSituacaoUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseUploadSituacao(tt.in); got != tt.want {
				t.Errorf("got %v, esperado %v", got, tt.want)
			}
		})
	}
}

func TestParseXContentHash_Cases(t *testing.T) {
	validHash := strings.Repeat("a", 64)
	validHeader := "SHA-256 " + validHash

	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"valid", validHeader, validHash, false},
		{"valid uppercase hash", "SHA-256 " + strings.ToUpper(validHash), validHash, false},
		{"valid SHA-256 case insensitive", "sha-256 " + validHash, validHash, false},
		{"empty", "", "", true},
		{"no space", "SHA-256" + validHash, "", true},
		{"wrong algorithm", "MD5 " + validHash, "", true},
		{"too short hash", "SHA-256 abc", "", true},
		{"non-hex char", "SHA-256 " + strings.Repeat("z", 64), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseXContentHash(tt.header)
			if tt.wantErr {
				if err == nil {
					t.Errorf("esperava erro, got hash=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("não esperava erro, got %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, esperado %q", got, tt.want)
			}
		})
	}
}
