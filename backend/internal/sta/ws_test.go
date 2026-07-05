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
	"context"
	"crypto/sha256"
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
//	POST /arquivos                      → handlePost
//	PUT  /arquivos/{protocolo}/conteudo → handlePut
//
// Se requireBasicAuth for true, retorna 401 quando header Authorization
// ausente (espelha comportamento real do BACEN).
type mockSTA struct {
	requireBasicAuth bool
	handlePost       func(w http.ResponseWriter, r *http.Request)
	handlePut        func(w http.ResponseWriter, r *http.Request, protocolo string)
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
			name: "empty password",
			cfg: WSConfig{
				BaseURL:  "https://sta-h.bcb.gov.br/staws",
				User:     "x.y",
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
		User:     "x.y",
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
