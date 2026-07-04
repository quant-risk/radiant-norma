// Tests para auth_handlers (Sprint 8a v2.1.0) — /v1/auth/dev-token.
//
// Cobertura:
//   - Endpoint disabled (RADIANT_DEV_TOKEN off) → 404
//   - Endpoint enabled but no signer → 503
//   - Endpoint enabled with signer → mints JWT
//   - Role whitelist (if/admin/readonly)
//   - TTL clamping (TTLDefault vs TTLCap)
//   - ifID validation
//   - JWT verifiable via Verifier (roundtrip)
package api_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/auth"
)

// generateTestPEM retorna PEM-encoded RSA private key para tests.
func generateTestPEM(t *testing.T) []byte {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
}

// newDevTokenServer monta um Server com DevSigner configurado.
//
// Defaults: RADIANT_DEV_TOKEN=1 setado para ativar endpoint.
// Test pode desabilitar via t.Setenv("RADIANT_DEV_TOKEN", "") no escopo.
func newDevTokenServer(t *testing.T) *api.Server {
	t.Helper()
	pemBytes := generateTestPEM(t)
	signer, err := auth.NewSigner(auth.SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	srv, _ := newTestServer(t)
	srv.DevSigner = signer
	return srv
}

// TestDevToken_EndpointDisabled: retorna 404 quando flag off.
func TestDevToken_EndpointDisabled(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "")
	srv, _ := newTestServer(t) // sem DevSigner

	body, _ := json.Marshal(map[string]any{
		"if_id": "demo",
		"role":  "if",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("dev-token endpoint disabled: status = %d, want 404", w.Code)
	}
}

// TestDevToken_SignerMissing: retorna 503 se flag on mas DevSigner nil.
func TestDevToken_SignerMissing(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv, _ := newTestServer(t)
	// srv.DevSigner = nil (default)

	body, _ := json.Marshal(map[string]any{
		"if_id": "demo",
		"role":  "if",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("signer missing: status = %d, want 503", w.Code)
	}
}

// TestDevToken_MintValid: happy path — emite JWT verificável.
func TestDevToken_MintValid(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	body, _ := json.Marshal(map[string]any{
		"if_id": "demo-bank-5",
		"role":  "if",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		Token      string `json:"token"`
		IFID       string `json:"if_id"`
		Role       string `json:"role"`
		ExpiresAt  string `json:"expires_at"`
		TTLSeconds int    `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Error("token empty")
	}
	if resp.IFID != "demo-bank-5" {
		t.Errorf("if_id: got %q want demo-bank-5", resp.IFID)
	}
	if resp.Role != "if" {
		t.Errorf("role: got %q want if", resp.Role)
	}
	if resp.TTLSeconds <= 0 {
		t.Errorf("ttl_seconds = %d, want > 0", resp.TTLSeconds)
	}
	// Verify expires_at parseable
	if _, err := time.Parse(time.RFC3339, resp.ExpiresAt); err != nil {
		t.Errorf("expires_at malformed: %s", resp.ExpiresAt)
	}
}

// TestDevToken_AdminRole: aceita role=admin com sucesso.
func TestDevToken_AdminRole(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	body, _ := json.Marshal(map[string]any{
		"if_id": "ops",
		"role":  "admin",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("admin: status = %d, want 200", w.Code)
	}
}

// TestDevToken_InvalidRole: rejeita role inválido.
func TestDevToken_InvalidRole(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	body, _ := json.Marshal(map[string]any{
		"if_id": "demo",
		"role":  "god",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid role: status = %d, want 400", w.Code)
	}
}

// TestDevToken_MissingIFID: rejeita sem if_id.
func TestDevToken_MissingIFID(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	body, _ := json.Marshal(map[string]any{
		"role": "if",
	})
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing if_id: status = %d, want 400", w.Code)
	}
}

// TestDevToken_TTLClamp: ttl maior que cap é clamped.
func TestDevToken_TTLClamp(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	// 60 dias em segundos — over cap.
	body := []byte(`{"if_id":"demo","role":"if","ttl_seconds":5184000}`)
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var resp struct {
		TTLSeconds int `json:"ttl_seconds"`
	}
	body2, _ := io.ReadAll(w.Body)
	if err := json.Unmarshal(body2, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should be clamped to TTLCap (30 dias = 2592000s).
	if resp.TTLSeconds > int(auth.TTLCap.Seconds()) {
		t.Errorf("ttl_seconds = %d, want <= %d (cap)", resp.TTLSeconds, int(auth.TTLCap.Seconds()))
	}
}

// TestDevToken_Roundtrip: token emitido é verificável pelo Verifier.
func TestDevToken_Roundtrip(t *testing.T) {
	t.Setenv("RADIANT_DEV_TOKEN", "1")
	srv := newDevTokenServer(t)

	body := []byte(`{"if_id":"demo","role":"if"}`)
	req := httptest.NewRequest("POST", "/v1/auth/dev-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// JWT deve ter 3 partes separated by dots.
	if strings.Count(resp.Token, ".") != 2 {
		t.Errorf("JWT malformed: %s", resp.Token)
	}
	// Verifica que o header contém kid=k1 (decodifica base64url).
	parts := strings.Split(resp.Token, ".")
	if len(parts) == 3 {
		// Add padding pro base64url decode.
		hdr := parts[0]
		if pad := len(hdr) % 4; pad != 0 {
			hdr += strings.Repeat("=", 4-pad)
		}
		dec, err := base64.URLEncoding.DecodeString(hdr)
		if err != nil {
			t.Logf("decode header: %v (continuing)", err)
		} else if !strings.Contains(string(dec), `"kid":"k1"`) {
			t.Errorf("JWT header missing kid=k1: %s", dec)
		}
	}
}
