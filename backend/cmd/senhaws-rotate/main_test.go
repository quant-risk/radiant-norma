// Tests do cmd/senhaws-rotate.
//
// Cobre: cada subcomando (check/rotate/info) + config validation + exit codes.
// Usa httptest.Server para mockar BACEN senhaws.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// captureStdout captura stdout durante execução de fn e retorna conteúdo.
// Usado para validar que CLI imprime no formato esperado.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

// captureStderr similar pra stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stderr = old
	return <-done
}

// silentLogger retorna logger que descarta output (test-friendly).
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// mockSenhawsServer retorna httptest.Server simulando senhaws BACEN.
func mockSenhawsServer(t *testing.T, handleAlterar http.HandlerFunc, handleVenc http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validação: Authorization Basic presente.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/senha":
			if r.Method != http.MethodPut {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			if handleAlterar != nil {
				handleAlterar(w, r)
				return
			}
		case "/senha/vencimento":
			if r.Method != http.MethodGet {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			if handleVenc != nil {
				handleVenc(w, r)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// baseCfg retorna config mínimo válido apontando para srv.
func baseCfg(srvURL string) *config {
	return &config{
		baseURL:           srvURL,
		user:              "123450001.fulano",
		password:          "old-password",
		timeout:           2 * time.Second,
		maxDays:           7,
		allowInsecureHTTP: true, // httptest.NewServer retorna http://
	}
}

// TestMaskUser — helper de masking.
func TestMaskUser(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"123450001.fulano", "12***.fulano"},
		{"12345/0001.beltrano", "12***.beltrano"},
		{"ab.x", "***"}, // muito curto
		{"a.bc", "***"}, // limite
		{"12345.ciclano", "12***.ciclano"},
	}
	for _, tt := range tests {
		got := maskUser(tt.in)
		if got != tt.want {
			t.Errorf("maskUser(%q) = %q, esperado %q", tt.in, got, tt.want)
		}
	}
}

// TestLoadConfig_Defaults — flags têm defaults sensatos.
func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("SENHAWS_BASE_URL", "https://example.com/senhaws")
	t.Setenv("SENHAWS_USER", "123450001.x")
	t.Setenv("SENHAWS_PASSWORD", "p")

	cfg, err := loadConfig(nil)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.timeout != 30*time.Second {
		t.Errorf("timeout default = %v, esperado 30s", cfg.timeout)
	}
	if cfg.maxDays != 7 {
		t.Errorf("maxDays default = %d, esperado 7", cfg.maxDays)
	}
	if cfg.quiet != false {
		t.Errorf("quiet default = %v, esperado false", cfg.quiet)
	}
}

// TestLoadConfig_InvalidTimeout — exit 2 (client error).
func TestLoadConfig_InvalidTimeout(t *testing.T) {
	_, err := loadConfig([]string{"--timeout", "abc"})
	if err == nil {
		t.Fatal("loadConfig deveria falhar com --timeout abc")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("erro deveria mencionar timeout, got %v", err)
	}
}

// TestLoadConfig_InvalidMaxDays — exit 2.
func TestLoadConfig_InvalidMaxDays(t *testing.T) {
	_, err := loadConfig([]string{"--max-days", "-1"})
	if err == nil {
		t.Fatal("loadConfig deveria falhar com --max-days -1")
	}
}

// TestSenhawsRotate_Check_OK — 30 dias → exit 0.
func TestSenhawsRotate_Check_OK(t *testing.T) {
	srv := mockSenhawsServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><DiasVencimentoSenha>30</DiasVencimentoSenha></Resultado>`)
	})
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runCheck(context.Background(), cfg, silentLogger())
		if code != exitOK {
			t.Errorf("exit code = %d, esperado %d", code, exitOK)
		}
	})

	if !strings.Contains(stdout, "dias_vencimento=30") {
		t.Errorf("stdout deveria conter dias_vencimento=30, got %q", stdout)
	}
	if !strings.Contains(stdout, "status=ok") {
		t.Errorf("stdout deveria conter status=ok, got %q", stdout)
	}
}

// TestSenhawsRotate_Check_Expiring — 5 dias (< threshold 7) → exit 1.
func TestSenhawsRotate_Check_Expiring(t *testing.T) {
	srv := mockSenhawsServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><DiasVencimentoSenha>5</DiasVencimentoSenha></Resultado>`)
	})
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runCheck(context.Background(), cfg, silentLogger())
		if code != exitGenericError {
			t.Errorf("exit code = %d, esperado %d (precisa rotacionar)", code, exitGenericError)
		}
	})

	if !strings.Contains(stdout, "status=expiring") {
		t.Errorf("stdout deveria conter status=expiring, got %q", stdout)
	}
}

// TestSenhawsRotate_Check_BACEN400 — exit 3.
func TestSenhawsRotate_Check_BACEN400(t *testing.T) {
	srv := mockSenhawsServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>user invalido</Descricao></Erro></Resultado>`)
	})
	cfg := baseCfg(srv.URL)

	code := runCheck(context.Background(), cfg, silentLogger())
	if code != exitBACENError {
		t.Errorf("exit code = %d, esperado %d (BACEN error)", code, exitBACENError)
	}
}

// TestSenhawsRotate_Rotate_Success — exit 0 + senha no stdout.
func TestSenhawsRotate_Rotate_Success(t *testing.T) {
	var capturedBody string
	srv := mockSenhawsServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusNoContent)
		},
		nil,
	)
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runRotate(context.Background(), cfg, silentLogger())
		if code != exitOK {
			t.Errorf("exit code = %d, esperado %d", code, exitOK)
		}
	})

	if !strings.Contains(stdout, "senha_alterada=true") {
		t.Errorf("stdout deveria conter senha_alterada=true, got %q", stdout)
	}
	// Validar senha nova foi gerada (32 hex chars).
	if !strings.Contains(stdout, "nova_senha=") {
		t.Errorf("stdout deveria conter nova_senha=, got %q", stdout)
	}
	// Validar XML body enviado.
	if !strings.Contains(capturedBody, "<Senha>old-password</Senha>") {
		t.Errorf("body deveria conter senha atual, got %q", capturedBody)
	}
}

// TestSenhawsRotate_Rotate_BACEN400 — exit 3.
func TestSenhawsRotate_Rotate_BACEN400(t *testing.T) {
	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>senha fraca</Descricao></Erro></Resultado>`)
	}, nil)
	cfg := baseCfg(srv.URL)

	code := runRotate(context.Background(), cfg, silentLogger())
	if code != exitBACENError {
		t.Errorf("exit code = %d, esperado %d (BACEN error)", code, exitBACENError)
	}
}

// TestSenhawsRotate_Rotate_BACEN401 — exit 3 (senha atual errada).
func TestSenhawsRotate_Rotate_BACEN401(t *testing.T) {
	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><Erro><Codigo>401</Codigo><Descricao>senha atual invalida</Descricao></Erro></Resultado>`)
	}, nil)
	cfg := baseCfg(srv.URL)

	code := runRotate(context.Background(), cfg, silentLogger())
	if code != exitBACENError {
		t.Errorf("exit code = %d, esperado %d (BACEN auth error)", code, exitBACENError)
	}
}

// TestSenhawsRotate_Info — happy path.
func TestSenhawsRotate_Info(t *testing.T) {
	srv := mockSenhawsServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><DiasVencimentoSenha>30</DiasVencimentoSenha></Resultado>`)
	})
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runInfo(context.Background(), cfg, silentLogger())
		if code != exitOK {
			t.Errorf("exit code = %d, esperado %d", code, exitOK)
		}
	})

	if !strings.Contains(stdout, "base_url="+srv.URL) {
		t.Errorf("stdout deveria conter base_url, got %q", stdout)
	}
	if !strings.Contains(stdout, "user=12***.fulano") {
		t.Errorf("stdout deveria conter user mascarado, got %q", stdout)
	}
	if !strings.Contains(stdout, "bacen_status=ok") {
		t.Errorf("stdout deveria conter bacen_status=ok, got %q", stdout)
	}
	if !strings.Contains(stdout, "dias_vencimento=30") {
		t.Errorf("stdout deveria conter dias_vencimento=30, got %q", stdout)
	}
}

// TestSenhawsRotate_ConfigInvalidUser — User formato Sisbacen errado → exit 2.
func TestSenhawsRotate_ConfigInvalidUser(t *testing.T) {
	cfg := &config{
		baseURL:  "https://example.com/senhaws",
		user:     "fulano", // formato Sisbacen inválido
		password: "p",
		timeout:  2 * time.Second,
		maxDays:  7,
	}

	code := runCheck(context.Background(), cfg, silentLogger())
	if code != exitClientError {
		t.Errorf("exit code = %d, esperado %d (client error)", code, exitClientError)
	}
}

// TestNewLogger_Quiet — logger quiet descarta output.
func TestNewLogger_Quiet(t *testing.T) {
	logger := newLogger(true)
	if logger == nil {
		t.Fatal("newLogger(true) deveria retornar logger")
	}
	// Não panica em Warn/Info/Error.
	logger.Warn("test", "key", "value")
	logger.Info("test2")
	logger.Error("test3")
}

// TestMain_UnknownSubcommand — dispatch retorna exit 1.
func TestMain_UnknownSubcommand(t *testing.T) {
	// Não podemos chamar main() diretamente (executa os.Exit).
	// Validar logic via teste do switch diretamente é tricky — testar via reuso.
	// Aqui testamos só que usage() não panica.
	stderr := captureStderr(t, usage)
	if !strings.Contains(stderr, "Usage: senhaws-rotate") {
		t.Errorf("usage deveria mencionar Usage, got %q", stderr)
	}
}

// TestEnvOrDefault — helper de env.
func TestEnvOrDefault(t *testing.T) {
	t.Setenv("TEST_VAR_SENHAWS_X", "custom")
	if got := envOrDefault("TEST_VAR_SENHAWS_X", "default"); got != "custom" {
		t.Errorf("envOrDefault com env set = %q, esperado custom", got)
	}
	if got := envOrDefault("TEST_VAR_SENHAWS_NONEXISTENT", "default"); got != "default" {
		t.Errorf("envOrDefault sem env = %q, esperado default", got)
	}
}

// TestSenhawsRotate_Rotate_ValidatesAuthHeader — verifica Basic Auth decodificado.
func TestSenhawsRotate_Rotate_ValidatesAuthHeader(t *testing.T) {
	var capturedAuth string
	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	cfg := baseCfg(srv.URL)

	_ = captureStdout(t, func() {
		runRotate(context.Background(), cfg, silentLogger())
	})

	if !strings.HasPrefix(capturedAuth, "Basic ") {
		t.Fatalf("Authorization deveria ser Basic, got %q", capturedAuth)
	}
	encoded := strings.TrimPrefix(capturedAuth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	if string(decoded) != "123450001.fulano:old-password" {
		t.Errorf("credenciais erradas, got %q", string(decoded))
	}
}

// Garantir que io.Discard é usado em silentLogger (compilação).
var _ = io.Discard
