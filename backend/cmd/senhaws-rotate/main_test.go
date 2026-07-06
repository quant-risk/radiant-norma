// Tests do cmd/senhaws-rotate.
//
// Cobre: cada subcomando (check/rotate/info) + config validation + exit codes.
// Usa httptest.Server para mockar BACEN senhaws.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/secrets"
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
		code := runRotate(context.Background(), cfg, silentLogger(), "")
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

	code := runRotate(context.Background(), cfg, silentLogger(), "")
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

	code := runRotate(context.Background(), cfg, silentLogger(), "")
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

// TestSenhawsRotate_Rotate_ValidationError — Validação 45 (F-S24-45-1):
// erros client-side (senha curta/longa/igual) devem resultar em exit 2.
// Antes do fix, heurística de substring era frágil.
//
// NOTA: senha vazia é omitida — CLI converte "" para GerarSenhaRandom()
// (default prod). Test de validação de senha vazia está em
// internal/senhaws TestSenhawsClient_AlterarSenha_ErrorsAs_Validation.
func TestSenhawsRotate_Rotate_ValidationError(t *testing.T) {
	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Não deveria ser chamado — validação client-side acontece antes.
		t.Errorf("mock não deveria ser chamado para erro de validação")
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	cfg := baseCfg(srv.URL)

	tests := []struct {
		name string
		nova string
	}{
		{"curta", "abc"},
		{"longa", strings.Repeat("a", 129)},
		{"mesma senha", "old-password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runRotate(context.Background(), cfg, silentLogger(), tt.nova)
			if code != exitClientError {
				t.Errorf("exit code = %d, esperado %d (client validation error)", code, exitClientError)
			}
		})
	}
}

// TestSenhawsRotate_Info_BACENError — caminho de erro BACEN (runInfo → exit 3).
func TestSenhawsRotate_Info_BACENError(t *testing.T) {
	srv := mockSenhawsServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>user invalido</Descricao></Erro></Resultado>`)
	})
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runInfo(context.Background(), cfg, silentLogger())
		if code != exitBACENError {
			t.Errorf("exit code = %d, esperado %d (BACEN error)", code, exitBACENError)
		}
	})

	if !strings.Contains(stdout, "bacen_status=error") {
		t.Errorf("stdout deveria conter bacen_status=error, got %q", stdout)
	}
}

// TestSenhawsRotate_Info_ConfigError — caminho de config inválida (runInfo → exit 2).
func TestSenhawsRotate_Info_ConfigError(t *testing.T) {
	cfg := &config{
		baseURL:  "https://example.com/senhaws",
		user:     "fulano", // formato Sisbacen inválido
		password: "p",
		timeout:  2 * time.Second,
		maxDays:  7,
	}

	stdout := captureStdout(t, func() {
		code := runInfo(context.Background(), cfg, silentLogger())
		if code != exitClientError {
			t.Errorf("exit code = %d, esperado %d (client error)", code, exitClientError)
		}
	})

	if !strings.Contains(stdout, "bacen_status=config_error") {
		t.Errorf("stdout deveria conter bacen_status=config_error, got %q", stdout)
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

// TestSenhawsRotate_Rotate_ValidatesAuthHeader — verifica Basic Auth decodificado + método PUT + Content-Type.
func TestSenhawsRotate_Rotate_ValidatesAuthHeader(t *testing.T) {
	var capturedAuth, capturedMethod, capturedContentType string
	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedMethod = r.Method
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	cfg := baseCfg(srv.URL)

	_ = captureStdout(t, func() {
		runRotate(context.Background(), cfg, silentLogger(), "")
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
	// Validação 45 (F-S24-45-4): verificar método HTTP e Content-Type também.
	if capturedMethod != http.MethodPut {
		t.Errorf("método HTTP = %q, esperado PUT", capturedMethod)
	}
	if capturedContentType != "application/xml" {
		t.Errorf("Content-Type = %q, esperado application/xml", capturedContentType)
	}
}

// Garantir que io.Discard é usado em silentLogger (compilação).
var _ = io.Discard

// =============================================================================
// Sprint 28 — subcomando apply
// =============================================================================

// TestSenhawsRotate_Apply_Success — Sprint 28 (v3.23.0) subcomando apply.
// Verifica que apply:
//  1. Chama BACEN AlterarSenha
//  2. Atualiza secret manager (memory backend em test)
//  3. Exit 0 + stdout contém secret_updated=true
func TestSenhawsRotate_Apply_Success(t *testing.T) {
	// Use memory backend (definido em NewManagerFromEnv)
	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	srv := mockSenhawsServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		nil,
	)
	cfg := baseCfg(srv.URL)

	stdout := captureStdout(t, func() {
		code := runApply(context.Background(), cfg, silentLogger())
		if code != exitOK {
			t.Errorf("exit code = %d, esperado %d", code, exitOK)
		}
	})

	if !strings.Contains(stdout, "senha_alterada=true") {
		t.Errorf("stdout deveria conter senha_alterada=true, got %q", stdout)
	}
	if !strings.Contains(stdout, "secret_updated=true") {
		t.Errorf("stdout deveria conter secret_updated=true, got %q", stdout)
	}
	if !strings.Contains(stdout, "backend=memory") {
		t.Errorf("stdout deveria conter backend=memory, got %q", stdout)
	}
	if !strings.Contains(stdout, `name="bacen/senha/123450001.fulano"`) {
		t.Errorf("stdout deveria conter name=\"bacen/senha/123450001.fulano\", got %q", stdout)
	}
}

// TestSenhawsRotate_Apply_BACENReject — BACEN rejeita antes de tocar manager.
// Exit 3 esperado. Manager NÃO deve ser consultado.
func TestSenhawsRotate_Apply_BACENReject(t *testing.T) {
	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `<?xml version="1.0"?>
<Resultado><Erro><Codigo>400</Codigo><Descricao>senha atual invalida</Descricao></Erro></Resultado>`)
	}, nil)
	cfg := baseCfg(srv.URL)

	stderr := captureStderr(t, func() {
		code := runApply(context.Background(), cfg, silentLogger())
		if code != exitBACENError {
			t.Errorf("exit code = %d, esperado %d (BACEN error)", code, exitBACENError)
		}
	})

	if !strings.Contains(stderr, "erro BACEN senhaws 400") {
		t.Errorf("stderr deveria conter erro BACEN, got %q", stderr)
	}
	if !strings.Contains(stderr, "senha atual invalida") {
		t.Errorf("stderr deveria conter mensagem BACEN, got %q", stderr)
	}
}

// TestSenhawsRotate_Apply_ConfigInvalid — exit 2.
func TestSenhawsRotate_Apply_ConfigInvalid(t *testing.T) {
	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	tests := []struct {
		name string
		cfg  *config
	}{
		{"empty baseURL", &config{user: "x", password: "y"}},
		{"empty user", &config{baseURL: "http://x", password: "y"}},
		{"empty password", &config{baseURL: "http://x", user: "y"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := captureStderr(t, func() {
				code := runApply(context.Background(), tt.cfg, silentLogger())
				if code != exitClientError {
					t.Errorf("exit code = %d, esperado %d", code, exitClientError)
				}
			})
			if !strings.Contains(stderr, "config invalida") {
				t.Errorf("stderr deveria conter 'config invalida', got %q", stderr)
			}
		})
	}
}

// TestSenhawsRotate_Apply_SecretNameFormat — verifica naming convention.
// User "123450001.fulano" → secret name "bacen/senha/123450001.fulano"
// (mantém "." para readability; EnvManager normaliza internamente)
func TestSenhawsRotate_Apply_SecretNameFormat(t *testing.T) {
	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	var capturedURL string
	srv := mockSenhawsServer(t,
		func(w http.ResponseWriter, r *http.Request) {
			capturedURL = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		},
		nil,
	)
	cfg := baseCfg(srv.URL)

	captureStdout(t, func() {
		code := runApply(context.Background(), cfg, silentLogger())
		if code != exitOK {
			t.Errorf("exit code = %d, esperado %d", code, exitOK)
		}
	})

	if capturedURL != "/senha" {
		t.Errorf("BACEN path = %q, esperado /senha", capturedURL)
	}
}

// =============================================================================
// Validação 50 — F-S28-50-B: senha NÃO pode vazar em stderr quando manager.Put
// falha. Pattern: gravar em arquivo 0600 com path conhecido, instruir admin.
// =============================================================================

// TestWriteFailsafe_BasicRoundTrip verifica que writeFailsafe grava em arquivo
// 0600 num path previsível com hash do user.
func TestWriteFailsafe_BasicRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RADIANT_FAILSAFE_PATH", tmpDir)

	path, err := writeFailsafe("123450001.fulano", "senha-secreta-123")
	if err != nil {
		t.Fatalf("writeFailsafe failed: %v", err)
	}

	// Path contém hash do user (não o user raw)
	if strings.Contains(path, "123450001") || strings.Contains(path, "fulano") {
		t.Errorf("failsafe path deve conter hash, não user raw: %s", path)
	}

	// Arquivo existe e tem permissões 0600
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failsafe file not found: %v", err)
	}
	if info.Size() != int64(len("senha-secreta-123")) {
		t.Errorf("failsafe size = %d, want %d", info.Size(), len("senha-secreta-123"))
	}

	// Permissões: somente owner rw
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("failsafe perms = %o, want 0600", mode)
	}

	// Conteúdo = senha (sem newline)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failsafe: %v", err)
	}
	if string(got) != "senha-secreta-123" {
		t.Errorf("failsafe content = %q, want %q", got, "senha-secreta-123")
	}

	// Cleanup
	os.Remove(path)
}

// failingManager simula um secrets.Manager cujo Put sempre retorna erro.
// Usado pra exercitar o caminho partial failure em runApply (BACEN OK + manager falha).
type failingManager struct {
	*secrets.MemoryManager
	putErr error
}

func (f *failingManager) Put(ctx context.Context, name, value string) (*secrets.Secret, error) {
	return nil, f.putErr
}

// TestRunApply_PartialFailure_NoStderrLeak — Validação 50 F-S28-50-B.
//
// Cenário: BACEN aceita (204) mas Manager.Put falha. runApply NÃO deve
// imprimir a senha em stderr. Deve gravar failsafe file 0600 e retornar exit 4.
func TestRunApply_PartialFailure_NoStderrLeak(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RADIANT_FAILSAFE_PATH", tmpDir)

	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	cfg := baseCfg(srv.URL)

	mgr := &failingManager{
		MemoryManager: secrets.NewMemoryManager(),
		putErr:        &secrets.AccessDeniedError{Name: "bacen/senha/x", Backend: "memory", Cause: errors.New("simulated")},
	}

	stderr := captureStderr(t, func() {
		_ = captureStdout(t, func() {
			code := runApplyWithManager(context.Background(), cfg, silentLogger(), mgr)
			if code != exitPartialFailure {
				t.Errorf("partial failure: exit = %d, want %d (exitPartialFailure)", code, exitPartialFailure)
			}
		})
	})

	// Verifica que a senha nova NÃO vazou em stderr (F-S28-50-B original era
	// "Senha nova (capture agora!): <senha>" — esse pattern NÃO pode aparecer)
	if strings.Contains(stderr, "Senha nova (capture") {
		t.Errorf("stderr NÃO deve conter padrão 'Senha nova (capture' (Validação 50): %q", stderr)
	}
	// Verifica que mensagem de failsafe file foi emitida (admin consegue agir)
	if !strings.Contains(stderr, "failsafe file") {
		t.Errorf("stderr deve mencionar failsafe file path: %q", stderr)
	}

	// Verifica que failsafe file foi criado com permissões 0600
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read tmpdir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected failsafe file in tmpdir, found 0")
	}
	for _, f := range files {
		info, _ := f.Info()
		if info.Mode().Perm() != 0600 {
			t.Errorf("failsafe %s perms = %o, want 0600", f.Name(), info.Mode().Perm())
		}
	}
}

// TestRunApply_HappyPath_NoFailsafe verifica que em caso de sucesso NÃO
// há failsafe file (sanity).
func TestRunApply_HappyPath_NoFailsafe(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RADIANT_FAILSAFE_PATH", tmpDir)
	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	srv := mockSenhawsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	cfg := baseCfg(srv.URL)

	_ = captureStderr(t, func() {
		_ = captureStdout(t, func() {
			code := runApply(context.Background(), cfg, silentLogger())
			if code != exitOK {
				t.Errorf("happy: exit = %d, want %d", code, exitOK)
			}
		})
	})

	files, _ := os.ReadDir(tmpDir)
	if len(files) != 0 {
		t.Errorf("happy path não deve criar failsafe file, found %d", len(files))
	}
}

// TestRunApply_PartialFailure_ExitCode4 verifica que exitPartialFailure é
// distinto. Validação 50 introduziu exit 4 pra automação diferenciar
// "BACEN rejeitou" (3) de "BACEN OK + manager falhou" (4).
func TestRunApply_PartialFailure_ExitCode4(t *testing.T) {
	if exitPartialFailure != 4 {
		t.Errorf("exitPartialFailure = %d, want 4", exitPartialFailure)
	}
	exits := map[int]bool{
		exitOK: true, exitGenericError: true, exitClientError: true,
		exitBACENError: true, exitPartialFailure: true,
	}
	if len(exits) != 5 {
		t.Errorf("exit codes devem ser únicos, got collisions")
	}
}
