// Package senhaws — tests do SenhawsClient (Sprint 23).
//
// Cobre: validação de config, AlterarSenha happy + errors, ConsultarVencimento,
// validações client-side de tamanho de senha.
package senhaws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockSenhaws é handler configurável para testes httptest.
type mockSenhaws struct {
	handleAlterar func(w http.ResponseWriter, r *http.Request)
	handleVenc    func(w http.ResponseWriter, r *http.Request)
}

func (m *mockSenhaws) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validação: Authorization Basic presente.
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		w.Header().Set("WWW-Authenticate", `Basic realm="senhaws"`)
		http.Error(w, "missing Authorization", http.StatusUnauthorized)
		return
	}

	switch r.URL.Path {
	case "/senha":
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if m.handleAlterar != nil {
			m.handleAlterar(w, r)
			return
		}
	case "/senha/vencimento":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if m.handleVenc != nil {
			m.handleVenc(w, r)
			return
		}
	}
	http.Error(w, "not implemented: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
}

// successSenhaMock retorna mock com handlers de sucesso para ambos endpoints.
// Usado em testes que precisam passar por validações client-side e chegar
// ao BACEN (mock) sem se preocupar com a resposta específica.
func successSenhaMock() *mockSenhaws {
	return &mockSenhaws{
		handleAlterar: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado><DiasVencimentoSenha>30</DiasVencimentoSenha></Resultado>`)
		},
	}
}

// newSenhawsClientForTest monta SenhawsClient apontando para mockSenhaws.
func newSenhawsClientForTest(t *testing.T, m *mockSenhaws) (*httptest.Server, *SenhawsClient) {
	t.Helper()

	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)

	sc, err := NewSenhawsClient(SenhawsConfig{
		BaseURL:           srv.URL,
		User:              "123450001.fulano",
		Password:          "old-password",
		Timeout:           2 * time.Second,
		AllowInsecureHTTP: true, // httptest.NewServer retorna http://
	})
	if err != nil {
		t.Fatalf("NewSenhawsClient: %v", err)
	}
	return srv, sc
}

// TestNewSenhawsClient_Validacao — configs inválidas rejeitadas.
func TestNewSenhawsClient_Validacao(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SenhawsConfig
		wantErr string
	}{
		{"BaseURL vazio", SenhawsConfig{User: "123450001.x", Password: "p"}, "BaseURL requerida"},
		{"BaseURL http (sem TLS)", SenhawsConfig{
			BaseURL: "http://example.com/senhaws", User: "123450001.x", Password: "p",
		}, "deve usar HTTPS"},
		{"BaseURL trailing slash", SenhawsConfig{
			BaseURL: "https://example.com/senhaws/", User: "123450001.x", Password: "p",
		}, "não deve terminar com /"},
		{"User vazio", SenhawsConfig{
			BaseURL: "https://example.com/senhaws", Password: "p",
		}, "User requerida"},
		{"User formato Sisbacen inválido", SenhawsConfig{
			BaseURL: "https://example.com/senhaws", User: "fulano", Password: "p",
		}, "formato Sisbacen inválido"},
		{"Password vazio", SenhawsConfig{
			BaseURL: "https://example.com/senhaws", User: "123450001.x",
		}, "Password requerida"},
		{"válido", SenhawsConfig{
			BaseURL: "https://example.com/senhaws", User: "123450001.x", Password: "p",
		}, ""},
		{"válido com slash", SenhawsConfig{
			BaseURL: "https://example.com/senhaws", User: "12345/0001.x", Password: "p",
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSenhawsClient(tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("esperava sucesso, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava erro contendo %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("erro deveria mencionar %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestSenhawsClient_AlterarSenha_HappyPath — PUT 204 No Content.
func TestSenhawsClient_AlterarSenha_HappyPath(t *testing.T) {
	var capturedBody string
	var capturedAuth string
	mock := &mockSenhaws{
		handleAlterar: func(w http.ResponseWriter, r *http.Request) {
			capturedAuth = r.Header.Get("Authorization")
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			if ct := r.Header.Get("Content-Type"); ct != "application/xml" {
				http.Error(w, "Content-Type errado: "+ct, http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	err := sc.AlterarSenha(context.Background(), "new-password-12345")
	if err != nil {
		t.Fatalf("AlterarSenha: %v", err)
	}

	// Verifica body XML enviado.
	if !strings.Contains(capturedBody, "<Senha>old-password</Senha>") {
		t.Errorf("body deveria conter senha atual, got %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "<NovaSenha>new-password-12345</NovaSenha>") {
		t.Errorf("body deveria conter nova senha, got %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "<ConfirmacaoNovaSenha>new-password-12345</ConfirmacaoNovaSenha>") {
		t.Errorf("body deveria conter confirmação, got %q", capturedBody)
	}
	if !strings.HasPrefix(capturedAuth, "Basic ") {
		t.Errorf("Authorization deveria ser Basic, got %q", capturedAuth)
	}
	// Verifica decodificação base64.
	encoded := strings.TrimPrefix(capturedAuth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if string(decoded) != "123450001.fulano:old-password" {
		t.Errorf("credenciais erradas, got %q", string(decoded))
	}
}

// TestSenhawsClient_AlterarSenha_400 — BACEN rejeita.
func TestSenhawsClient_AlterarSenha_400(t *testing.T) {
	mock := &mockSenhaws{
		handleAlterar: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>Nova senha não atende requisitos</Descricao></Erro></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	err := sc.AlterarSenha(context.Background(), "weak-pwd")
	if err == nil {
		t.Fatal("esperava erro em 400")
	}
	var senErr *SenhaError
	if !errors.As(err, &senErr) {
		t.Fatalf("erro deveria ser *SenhaError, got %T: %v", err, err)
	}
	if senErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, esperado 400", senErr.StatusCode)
	}
}

// TestSenhawsClient_AlterarSenha_401 — senha atual errada.
func TestSenhawsClient_AlterarSenha_401(t *testing.T) {
	mock := &mockSenhaws{
		handleAlterar: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>401</Codigo><Descricao>Senha atual inválida</Descricao></Erro></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	err := sc.AlterarSenha(context.Background(), "new-password-12345")
	if err == nil {
		t.Fatal("esperava erro em 401")
	}
	var senErr *SenhaError
	if !errors.As(err, &senErr) {
		t.Fatalf("erro deveria ser *SenhaError, got %T", err)
	}
	if senErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, esperado 401", senErr.StatusCode)
	}
}

// TestSenhawsClient_AlterarSenha_Validacoes — client-side checks.
func TestSenhawsClient_AlterarSenha_Validacoes(t *testing.T) {
	mock := successSenhaMock()
	_, sc := newSenhawsClientForTest(t, mock)

	tests := []struct {
		name    string
		nova    string
		wantErr string
	}{
		{"vazia", "", "não pode ser vazia"},
		{"curta (< 8)", "abc", "mínimo 8 chars"},
		{"longa (> 128)", strings.Repeat("a", 129), "máximo 128 chars"},
		{"mesma senha atual", "old-password", "diferente da senha atual"},
		{"válida", "new-password-12345", ""},
		{"válida 8 chars", "abcd1234", ""},
		{"válida 128 chars", strings.Repeat("a", 128), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sc.AlterarSenha(context.Background(), tt.nova)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("esperava sucesso, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("esperava erro contendo %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("erro deveria mencionar %q, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestSenhawsClient_ConsultarVencimento_HappyPath — GET 200 + dias.
func TestSenhawsClient_ConsultarVencimento_HappyPath(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado>
<DiasVencimentoSenha>30</DiasVencimentoSenha>
</Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	dias, err := sc.ConsultarVencimento(context.Background())
	if err != nil {
		t.Fatalf("ConsultarVencimento: %v", err)
	}
	if dias != 30 {
		t.Errorf("dias = %d, esperado 30", dias)
	}
}

// TestSenhawsClient_ConsultarVencimento_400 — BACEN rejeita.
func TestSenhawsClient_ConsultarVencimento_400(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>Usuário não autenticado</Descricao></Erro></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	_, err := sc.ConsultarVencimento(context.Background())
	if err == nil {
		t.Fatal("esperava erro em 400")
	}
	var senErr *SenhaError
	if !errors.As(err, &senErr) {
		t.Fatalf("erro deveria ser *SenhaError, got %T", err)
	}
	if senErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, esperado 400", senErr.StatusCode)
	}
}

// TestSenhawsClient_ConsultarVencimento_BadXML — 200 OK mas body não parsea.
func TestSenhawsClient_ConsultarVencimento_BadXML(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `not valid xml`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	_, err := sc.ConsultarVencimento(context.Background())
	if err == nil {
		t.Fatal("esperava erro de parse")
	}
	if !strings.Contains(err.Error(), "parse vencimento") {
		t.Errorf("erro deveria mencionar parse, got %v", err)
	}
}

// TestSenhawsClient_ConsultarVencimento_DiasVazios — 200 OK com <DiasVencimentoSenha></DiasVencimentoSenha>.
func TestSenhawsClient_ConsultarVencimento_DiasVazios(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado><DiasVencimentoSenha></DiasVencimentoSenha></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	_, err := sc.ConsultarVencimento(context.Background())
	if err == nil {
		t.Fatal("esperava erro de DiasVencimentoSenha vazio")
	}
	if !strings.Contains(err.Error(), "vazio") {
		t.Errorf("erro deveria mencionar vazio, got %v", err)
	}
}

// TestSenhawsClient_ConsultarVencimento_NaoInteiro — 200 OK com texto não-numérico.
func TestSenhawsClient_ConsultarVencimento_NaoInteiro(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado><DiasVencimentoSenha>abc</DiasVencimentoSenha></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	_, err := sc.ConsultarVencimento(context.Background())
	if err == nil {
		t.Fatal("esperava erro de parse int")
	}
	if !strings.Contains(err.Error(), "não é inteiro") {
		t.Errorf("erro deveria mencionar não-inteiro, got %v", err)
	}
}

// TestSenhawsClient_ConsultarVencimento_Negativo — 200 OK com dias < 0.
func TestSenhawsClient_ConsultarVencimento_Negativo(t *testing.T) {
	mock := &mockSenhaws{
		handleVenc: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Resultado><DiasVencimentoSenha>-1</DiasVencimentoSenha></Resultado>`)
		},
	}
	_, sc := newSenhawsClientForTest(t, mock)

	_, err := sc.ConsultarVencimento(context.Background())
	if err == nil {
		t.Fatal("esperava erro de dias negativo")
	}
	if !strings.Contains(err.Error(), "negativo") {
		t.Errorf("erro deveria mencionar negativo, got %v", err)
	}
}

// TestGerarSenhaRandom — sanity check do helper.
func TestGerarSenhaRandom(t *testing.T) {
	for i := 0; i < 10; i++ {
		s := GerarSenhaRandom()
		// 16 bytes hex = 32 chars.
		if len(s) != 32 {
			t.Errorf("len = %d, esperado 32", len(s))
		}
		// Apenas hex chars.
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("char não-hex em %q: %c", s, c)
			}
		}
	}
}

// TestSenhaError_Error — formato do Error().
func TestSenhaError_Error(t *testing.T) {
	e := &SenhaError{StatusCode: 400, Code: "400", Message: "senha fraca"}
	want := "BACEN senhaws error 400: senha fraca"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, esperado %q", got, want)
	}
}

// fmt usado em tests?
var _ = fmt.Sprintf
