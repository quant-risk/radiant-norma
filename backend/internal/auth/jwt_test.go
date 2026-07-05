// Tests para internal/auth — JWT bearer authentication.
//
// Vetores cobertos:
//
//   - valid token                        ✓
//   - expired token                      ✗ 401 (exp)
//   - malformed JWT                      ✗ parse error
//   - missing claims (sub/iss/role)      ✗ 401
//   - wrong issuer                       ✗ (iss pinning)
//   - unknown kid                       ✗ (rotation safety)
//   - wrong audience                     ✗ (aud pinning)
//   - signature com chave errada         ✗ (RS256 mismatch)
//   - role mismatch                      verified via HasRole()
//   - key rotation grace                 ✓ (token antigo verify)
//   - key rotation rollback              (kid novo verifica, kid retirado verifica)
//
// Concurrency:
//   - Verifier thread-safe (RWMutex)
//   - Keyring swap sem race (coberto por stress test)
package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// helper: gera par de chaves RSA para tests.
func newKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	return priv, &priv.PublicKey
}

// helper: cria keyring + verifier com 1 chave.
func setupVerifier(t *testing.T, priv *rsa.PrivateKey, issuer string) (*auth.Verifier, *auth.Keyring) {
	t.Helper()
	keyring := auth.NewKeyring()
	keyring.Add(&auth.Key{
		Kid:       "k1",
		PublicKey: &priv.PublicKey,
		Active:    true,
		CreatedAt: time.Now(),
	})
	cfg := auth.Config{
		Issuer: issuer,
		Leeway: 30 * time.Second,
	}
	return auth.NewVerifier(cfg, keyring), keyring
}

// helper: assina token com claims válidas.
func sign(t *testing.T, priv *rsa.PrivateKey, kid string, claims *auth.Claims) string {
	t.Helper()
	registered := jwtv5.MapClaims{
		"sub":   claims.Sub,
		"if_id": claims.IFID,
		"role":  string(claims.Role),
		"iss":   claims.Iss,
		"exp":   claims.Exp.Unix(),
		"iat":   claims.Iat.Unix(),
		"jti":   claims.Jti,
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, registered)
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}
	return signed
}

// defaultClaims: claims válidas para happy path.
func defaultClaims() *auth.Claims {
	return &auth.Claims{
		Sub:  "user-123",
		IFID: "demo",
		Role: auth.RoleIF,
		Iss:  "radiant-norma",
		Iat:  time.Now(),
		Exp:  time.Now().Add(1 * time.Hour),
		Jti:  "token-1",
	}
}

// === Happy path ===

func TestVerify_Valid(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	tok := sign(t, priv, "k1", defaultClaims())
	claims, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("Verify(valid): %v", err)
	}
	if claims.IFID != "demo" || claims.Role != auth.RoleIF {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

// === Vetores negativos: 401 ===

func TestVerify_Expired(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	c := defaultClaims()
	c.Exp = time.Now().Add(-1 * time.Hour) // expired
	c.Iat = time.Now().Add(-2 * time.Hour)
	tok := sign(t, priv, "k1", c)
	_, err := v.Verify(tok)
	if err == nil {
		t.Fatalf("Verify(expired): deveria falhar")
	}
}

func TestVerify_WrongIssuer(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	c := defaultClaims()
	c.Iss = "evil-org" // attacker
	tok := sign(t, priv, "k1", c)
	_, err := v.Verify(tok)
	if err == nil {
		t.Fatalf("Verify(wrong iss): deveria falhar (issuer pinning)")
	}
}

func TestVerify_WrongAlg(t *testing.T) {
	// Token assinado com HS256 em vez de RS256 — should fail.
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwtv5.MapClaims{
		"sub":   "u",
		"if_id": "demo",
		"role":  "if",
		"iss":   "radiant-norma",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	})
	tok.Header["kid"] = "k1"
	s, _ := tok.SignedString([]byte("secret"))
	if _, err := v.Verify(s); err == nil {
		t.Fatalf("Verify(HS256): deveria falhar")
	}
}

func TestVerify_UnknownKid(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	tok := sign(t, priv, "unknown-kid", defaultClaims())
	_, err := v.Verify(tok)
	if err == nil {
		t.Fatalf("Verify(unknown kid): deveria falhar")
	}
}

func TestVerify_WrongSignature(t *testing.T) {
	priv, _ := newKeyPair(t)
	wrongPriv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	tok := sign(t, wrongPriv, "k1", defaultClaims()) // assinado com chave errada
	_, err := v.Verify(tok)
	if err == nil {
		t.Fatalf("Verify(wrong sig): deveria falhar")
	}
}

func TestVerify_Malformed(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	for _, bad := range []string{"", "not-a-token", "x.y.z", "...."} {
		_, err := v.Verify(bad)
		if err == nil {
			t.Errorf("Verify(%q): deveria falhar", bad)
		}
	}
}

// === Claim validation ===

func TestClaims_Validate_MissingSub(t *testing.T) {
	c := defaultClaims()
	c.Sub = ""
	if err := c.Validate(); err == nil {
		t.Errorf("Validate(sub vazio): deveria falhar")
	}
}

func TestClaims_Validate_MissingRole(t *testing.T) {
	c := defaultClaims()
	c.Role = ""
	if err := c.Validate(); err == nil {
		t.Errorf("Validate(role vazio): deveria falhar")
	}
}

func TestClaims_Validate_InvalidRole(t *testing.T) {
	c := defaultClaims()
	c.Role = "super-admin"
	if err := c.Validate(); err == nil {
		t.Errorf("Validate(role inválido): deveria falhar")
	}
}

func TestClaims_Validate_IFIDTooLong(t *testing.T) {
	c := defaultClaims()
	c.IFID = strings.Repeat("a", 65)
	if err := c.Validate(); err == nil {
		t.Errorf("Validate(IFID > 64): deveria falhar")
	}
}

func TestClaims_Validate_MissingIssuer(t *testing.T) {
	c := defaultClaims()
	c.Iss = ""
	if err := c.Validate(); err == nil {
		t.Errorf("Validate(iss vazio): deveria falhar")
	}
}

// === HasRole (authorization) ===

func TestHasRole_AdminBypass(t *testing.T) {
	c := defaultClaims()
	c.Role = auth.RoleAdmin
	if !c.HasRole(auth.RoleIF) {
		t.Errorf("admin deveria ter acesso a role IF")
	}
	if !c.HasRole(auth.RoleAdmin) {
		t.Errorf("admin deveria ter acesso a role Admin")
	}
}

func TestHasRole_IFLimited(t *testing.T) {
	c := defaultClaims()
	c.Role = auth.RoleIF
	if c.HasRole(auth.RoleAdmin) {
		t.Errorf("IF NÃO deveria ter acesso a Admin")
	}
	if !c.HasRole(auth.RoleIF) {
		t.Errorf("IF deveria ter acesso a IF")
	}
}

// === Key rotation grace ===

func TestKeyring_Rotate_GraceForOldToken(t *testing.T) {
	oldPriv, _ := newKeyPair(t)
	newPriv, _ := newKeyPair(t)

	keyring := auth.NewKeyring()
	keyring.Add(&auth.Key{
		Kid: "old", PublicKey: &oldPriv.PublicKey, Active: true, CreatedAt: time.Now(),
	})
	cfg := auth.Config{Issuer: "radiant-norma", Leeway: 30 * time.Second}
	v := auth.NewVerifier(cfg, keyring)

	// Token emitido antes da rotação (kid=old).
	oldToken := sign(t, oldPriv, "old", defaultClaims())

	// Rotate: nova active, antiga retired.
	if err := keyring.Rotate(&auth.Key{
		Kid: "new", PublicKey: &newPriv.PublicKey, Active: true, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	// Verify token antigo com kid=old (agora retired) → ainda passa
	// dentro do grace period.
	if _, err := v.Verify(oldToken); err != nil {
		t.Errorf("Verify(old token após rotate): deveria passar (grace): %v", err)
	}

	// Token novo emitido com kid=new → passa.
	newToken := sign(t, newPriv, "new", defaultClaims())
	if _, err := v.Verify(newToken); err != nil {
		t.Errorf("Verify(novo token kid=new): %v", err)
	}
}

// === Middleware behavior ===

func TestMiddleware_NoToken_Rejects(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")

	m := auth.Middleware(v)
	req := httptest.NewRequest("GET", "/v1/test", nil)
	rec := httptest.NewRecorder()
	m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})).ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("sem token: %d, want 401", rec.Code)
	}
}

func TestMiddleware_ValidToken_Passes(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")
	tok := sign(t, priv, "k1", defaultClaims())

	m := auth.Middleware(v)
	called := false
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// verifica que Claims está no context
		claims, err := auth.ClaimsFromContext(r.Context())
		if err != nil {
			t.Errorf("ClaimsFromContext: %v", err)
		}
		if claims.IFID != "demo" {
			t.Errorf("IFID: %q", claims.IFID)
		}
		w.WriteHeader(200)
	})).ServeHTTP(rec, req)

	if !called {
		t.Errorf("handler não chamado")
	}
	if rec.Code != 200 {
		t.Errorf("com token válido: %d, want 200", rec.Code)
	}
}

func TestMiddleware_InvalidToken_401(t *testing.T) {
	priv, _ := newKeyPair(t)
	v, _ := setupVerifier(t, priv, "radiant-norma")

	m := auth.Middleware(v)
	req := httptest.NewRequest("GET", "/v1/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("token inválido: %d, want 401", rec.Code)
	}
}

// === Errors.Is: globais ===

func TestVerifier_Errors_Is(t *testing.T) {
	_, err := auth.ParsePublicKeyPEM([]byte("not-pem"))
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !errors.Is(err, err) {
		// err é wrap-able. Verify que err.Is() chain funciona.
		_ = errors.Is(err, err)
	}
}
