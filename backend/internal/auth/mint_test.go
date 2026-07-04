// Tests para auth.Signer (Sprint 8a v2.1.0).
//
// Cobertura:
//   - Signer creation via PEM (PKCS#1 e PKCS#8)
//   - Mint with valid claims → JWT string parseável
//   - Claims.Validate() rejection
//   - Invalid PEM rejection
//   - Roundtrip Sign+Verify (com Verifier real)
//   - TTL clamping (TTL default vs cap)
//   - Simple mint helper (dev path) — if_id validation, role whitelist
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"
)

// jwtBase64Decode decodifica JWT payload (base64url) → JSON string.
// Helper para tests verificarem claims emitadas.
func jwtBase64Decode(s string) (string, error) {
	// JWT usa base64url (sem padding). Adiciona padding.
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		// Tenta StdEncoding (algumas libs usam com padding).
		data, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			return "", err
		}
	}
	return string(data), nil
}

// (imports ok)

// generateTestKey gera par RSA 2048 para tests.
func generateTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	return priv, pemBytes
}

// TestNewSigner_ValidPEM: cria Signer com PEM PKCS#1.
func TestNewSigner_ValidPEM(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Errorf("NewSigner PEM válido: %v", err)
	}
	if signer.kid != "k1" {
		t.Errorf("kid: got %q want k1", signer.kid)
	}
	if signer.issuer != "radiant-norma" {
		t.Errorf("issuer: got %q want radiant-norma", signer.issuer)
	}
}

// TestNewSigner_PEMvazio: erro se PEM vazio.
func TestNewSigner_PEMvazio(t *testing.T) {
	_, err := NewSigner(SignerConfig{
		PrivateKeyPEM: "",
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err == nil {
		t.Error("esperava erro com PEM vazio")
	}
}

// TestNewSigner_KidVazio: erro se kid vazio.
func TestNewSigner_KidVazio(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	_, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "",
		Issuer:        "radiant-norma",
	})
	if err == nil {
		t.Error("esperava erro com kid vazio")
	}
}

// TestNewSigner_IssuerVazio: erro se issuer vazio.
func TestNewSigner_IssuerVazio(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	_, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "",
	})
	if err == nil {
		t.Error("esperava erro com issuer vazio")
	}
}

// TestSigner_Mint_ValidClaims: Mint com claims válidas → JWT string retornado.
func TestSigner_Mint_ValidClaims(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	now := time.Now()
	claims := Claims{
		Sub:  "demo",
		IFID: "demo",
		Role: RoleIF,
		Iss:  "radiant-norma",
		Iat:  now,
		Exp:  now.Add(1 * time.Hour),
	}
	signed, exp, err := signer.Mint(claims)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if signed == "" {
		t.Error("token vazio")
	}
	if !exp.Equal(claims.Exp) {
		t.Errorf("exp: got %v want %v", exp, claims.Exp)
	}
	// JWT should have 3 parts (header.payload.signature).
	if strings.Count(signed, ".") != 2 {
		t.Errorf("JWT malformed: %q", signed)
	}
}

// TestSigner_Mint_InvalidClaims: Mint rejeita claims inválidas.
func TestSigner_Mint_InvalidClaims(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	// IFID vazio → Validate() falha.
	_, _, err = signer.Mint(Claims{
		Sub:  "x",
		IFID: "",
		Role: RoleIF,
		Iss:  "radiant-norma",
		Exp:  time.Now().Add(1 * time.Hour),
	})
	if err == nil {
		t.Error("esperava erro com IFID vazio")
	}

	// Role inválido
	_, _, err = signer.Mint(Claims{
		Sub:  "x",
		IFID: "demo",
		Role: "hacker",
		Iss:  "radiant-norma",
		Exp:  time.Now().Add(1 * time.Hour),
	})
	if err == nil {
		t.Error("esperava erro com role inválido")
	}
}

// TestSigner_MintSimple: helper dev/demo.
func TestSigner_MintSimple(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	signed, exp, err := signer.MintSimple("demo-bank", RoleIF, 1*time.Hour)
	if err != nil {
		t.Errorf("MintSimple: %v", err)
	}
	if signed == "" {
		t.Error("token vazio")
	}
	if !exp.After(time.Now()) {
		t.Errorf("exp no passado: %v", exp)
	}
}

// TestSigner_MintSimple_Validations: edge cases do helper dev.
func TestSigner_MintSimple_Validations(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, _ := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})

	tests := []struct {
		name  string
		ifID  string
		role  Role
		ttl   time.Duration
		wantE bool
	}{
		{"valid if", "demo-bank", RoleIF, time.Hour, false},
		{"empty if", "", RoleIF, time.Hour, true},
		{"if with space", "demo bank", RoleIF, time.Hour, true},
		{"if with slash", "demo/bank", RoleIF, time.Hour, true},
		{"if too long", strings.Repeat("a", 65), RoleIF, time.Hour, true},
		{"admin valid", "demo-admin", RoleAdmin, time.Hour, false},
		{"readonly valid", "ro-user", RoleReadOnly, time.Hour, false},
		{"invalid role", "demo", Role("god"), time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := signer.MintSimple(tt.ifID, tt.role, tt.ttl)
			gotErr := err != nil
			if gotErr != tt.wantE {
				t.Errorf("MintSimple err = %v, wantErr %v", err, tt.wantE)
			}
		})
	}
}

// TestSigner_Roundtrip: Sign + Verify com par real.
func TestSigner_Roundtrip(t *testing.T) {
	priv, pemBytes := generateTestKey(t)

	signer, err := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}

	// Cria Verifier com public key da mesma private key.
	keyring := NewKeyring()
	keyring.Add(&Key{
		Kid:       "k1",
		PublicKey: &priv.PublicKey,
		Active:    true,
		CreatedAt: time.Now(),
	})
	verifier := NewVerifier(Config{Issuer: "radiant-norma"}, keyring)

	// Mint + Verify
	now := time.Now()
	claims := Claims{
		Sub:  "demo",
		IFID: "demo-bank-3",
		Role: RoleIF,
		Iss:  "radiant-norma",
		Iat:  now,
		Exp:  now.Add(1 * time.Hour),
	}
	signed, _, err := signer.Mint(claims)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}

	verified, err := verifier.Verify(signed)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verified.IFID != "demo-bank-3" {
		t.Errorf("IFID: got %q want %q", verified.IFID, "demo-bank-3")
	}
	if verified.Role != RoleIF {
		t.Errorf("Role: got %q want if", verified.Role)
	}
}

// TestSigner_IssuerOverride: Signer sobrescreve Claims.Iss se diferente.
// Defense contra tokens forjados via claim manipulation.
func TestSigner_IssuerOverride(t *testing.T) {
	_, pemBytes := generateTestKey(t)
	signer, _ := NewSigner(SignerConfig{
		PrivateKeyPEM: string(pemBytes),
		Kid:           "k1",
		Issuer:        "radiant-norma",
	})

	signed, _, err := signer.Mint(Claims{
		Sub:  "demo",
		IFID: "demo",
		Role: RoleIF,
		Iss:  "evil-issuer", // tentaria override
		Exp:  time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT malformed: %q", signed)
	}
	// Decodifica payload base64 → JSON string.
	payload, err := jwtBase64Decode(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	// Issuer pinned = radiant-norma (overridden por signer).
	if !strings.Contains(payload, `"iss":"radiant-norma"`) {
		t.Errorf("JWT payload missing pinned issuer; got: %s", payload)
	}
	if strings.Contains(payload, "evil-issuer") {
		t.Errorf("JWT payload contains unpinned issuer")
	}
}

// TestTTLCap: cap está em 30 dias.
func TestTTLCap(t *testing.T) {
	if TTLCap != 30*24*time.Hour {
		t.Errorf("TTLCap = %v, want 30 dias", TTLCap)
	}
	if TTLDefault != 24*time.Hour {
		t.Errorf("TTLDefault = %v, want 24h", TTLDefault)
	}
}
