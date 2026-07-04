// Package auth — JWT signing helpers.
//
// Sprint 8a (v2.1.0): factoring out JWT signing logic from cmd/jwt-mint
// so the API server can mint dev tokens in-process via /v1/auth/dev-token.
//
// Em produção: tokens são emitidos por IdP externo (Keycloak/Okta/etc).
// Em dev: cmd/jwt-mint (CLI) ou /v1/auth/dev-token (frontend bridge) usam
// este helper para gerar tokens de demo / fixtures.
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Signer assina JWTs RS256 usando uma chave RSA privada.
//
// Por design: o Signer detém chave privada (sensível) — NUNCA retornar
// para HTTP handlers, sempre usado server-side. Reuso por múltiplos
// goroutines é seguro (rsa.Sign é internally thread-safe; jwt-go usa
// uma goroutine-local random source via crypto/rand).
type Signer struct {
	priv    *rsa.PrivateKey
	kid     string
	issuer  string
	now     func() time.Time
	jtiRand func() string
}

// SignerConfig configura um Signer.
//
//   - PrivateKeyPEM: chave RSA privada em formato PEM (PKCS#1 ou PKCS#8).
//     Aceita também []byte com conteúdo PEM, ou path via load helpers.
//   - Kid: Key ID. Único dentro do Keyring compatível (verificação usa
//     mesmo kid para matching).
//   - Issuer: issuer (iss claim). Verifier exige match.
//   - Now: clock function (default time.Now). Override em tests para
//     timestamps determinísticos.
//   - JTIrand: random source para gerar JTI único (default crypto/rand).
type SignerConfig struct {
	PrivateKeyPEM string // PEM-encoded RSA private key (PKCS#1 ou PKCS#8)
	Kid           string
	Issuer        string
	Now           func() time.Time
	JTIrand       func() string
}

// NewSigner cria Signer a partir de PEM-encoded private key.
//
// Valida formato PEM (PKCS#1 e PKCS#8 suportados). Retorna erro se
// chave ausente, mal-formed, ou não-RSA.
func NewSigner(cfg SignerConfig) (*Signer, error) {
	if cfg.PrivateKeyPEM == "" {
		return nil, errors.New("auth.Signer: PrivateKeyPEM required")
	}
	if cfg.Kid == "" {
		return nil, errors.New("auth.Signer: Kid required")
	}
	if cfg.Issuer == "" {
		return nil, errors.New("auth.Signer: Issuer required")
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	jtiRand := cfg.JTIrand
	if jtiRand == nil {
		jtiRand = defaultJTIRand
	}

	block, _ := pem.Decode([]byte(cfg.PrivateKeyPEM))
	if block == nil {
		return nil, errors.New("auth.Signer: invalid PEM encoding")
	}
	priv, err := parseRSAPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth.Signer: parse private key: %w", err)
	}

	return &Signer{
		priv:    priv,
		kid:     cfg.Kid,
		issuer:  cfg.Issuer,
		now:     now,
		jtiRand: jtiRand,
	}, nil
}

// NewSignerFromFile carrega chave privada de arquivo PEM path.
//
// Same as NewSigner mas lê arquivo primeiro. Mensagem de erro inclui path.
func NewSignerFromFile(path string, kid, issuer string) (*Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("auth: read private key %s: %w", path, err)
	}
	return NewSigner(SignerConfig{
		PrivateKeyPEM: string(data),
		Kid:           kid,
		Issuer:        issuer,
	})
}

// parseRSAPrivateKey parseia chave RSA privada em PKCS#1 ou PKCS#8.
// Suporta ambos formatos porque cmd/jwt-mint pode receber de qualquer
// ferramenta (openssl default PKCS#1 vs ssh-keygen default PKCS#8).
func parseRSAPrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if priv, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return priv, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaPriv, ok := k.(*rsa.PrivateKey); ok {
			return rsaPriv, nil
		}
		return nil, errors.New("not RSA private key (PKCS#8)")
	}
	return nil, errors.New("not RSA private key (PKCS#1 nor PKCS#8)")
}

// Mint assina JWT com claims pré-preenchidas.
//
// Valida claims antes de assinar (sub, if_id, role, iss, exp requeridos).
// Adiciona `iat` se Claims.Iat for zero (default now).
//
// Returns: (signed JWT string, expiration time, error).
func (s *Signer) Mint(claims Claims) (string, time.Time, error) {
	if err := claims.Validate(); err != nil {
		return "", time.Time{}, fmt.Errorf("auth.Signer.Mint: %w", err)
	}
	if claims.Iss == "" {
		claims.Iss = s.issuer
	}
	claims.Iss = s.issuer // pin even if Claims.Iss was different
	if claims.Iat.IsZero() {
		claims.Iat = s.now()
	}

	registered := jwtv5.MapClaims{
		"sub":   claims.Sub,
		"if_id": claims.IFID,
		"role":  string(claims.Role),
		"iss":   claims.Iss,
		"exp":   claims.Exp.Unix(),
		"iat":   claims.Iat.Unix(),
		"jti":   s.jtiRand(),
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, registered)
	token.Header["kid"] = s.kid

	signed, err := token.SignedString(s.priv)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth.Signer.Mint: sign: %w", err)
	}
	return signed, claims.Exp, nil
}

// MintSimple é helper para dev/demo: gera Claims com Sub=IFID, role,
// issuer=s.issuer, e TTL. Returns signed JWT + exp time.
//
// Valida IFID (max 64 chars, alfanumérico + dash + underscore — match
// com validação X-IF-ID frontend).
func (s *Signer) MintSimple(ifID string, role Role, ttl time.Duration) (string, time.Time, error) {
	if ifID == "" {
		return "", time.Time{}, errors.New("auth.Signer.MintSimple: ifID required")
	}
	if len(ifID) > 64 {
		return "", time.Time{}, errors.New("auth.Signer.MintSimple: ifID too long (max 64)")
	}
	for _, c := range ifID {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_'
		if !ok {
			return "", time.Time{}, errors.New("auth.Signer.MintSimple: invalid character in ifID")
		}
	}
	switch role {
	case RoleIF, RoleAdmin, RoleReadOnly:
	default:
		return "", time.Time{}, fmt.Errorf("auth.Signer.MintSimple: invalid role %q", role)
	}
	now := s.now()
	claims := Claims{
		Sub:  ifID,
		IFID: ifID,
		Role: role,
		Iss:  s.issuer,
		Iat:  now,
		Exp:  now.Add(ttl),
	}
	return s.Mint(claims)
}

// defaultJTIRand gera 16 random bytes (32 hex chars) por token.
// Suficiente para unicidade em escala de demo (collision < 1e-30).
func defaultJTIRand() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: timestamp-based. Não deveria acontecer em runtime normal.
		return fmt.Sprintf("jti-fallback-%d", time.Now().UnixNano())
	}
	return "jti-" + hex.EncodeToString(b)
}

// TTLCap limita TTL máximo permitido em chamadas de dev-token.
// Previne geração de tokens com vida excessiva em ambiente dev
// (ex: alguém passar --ttl=8760h por engano).
const TTLCap = 30 * 24 * time.Hour // 30 dias

// TTLDefault é TTL sugerido para tokens dev/demo.
const TTLDefault = 24 * time.Hour
