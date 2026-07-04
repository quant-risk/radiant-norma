// Package auth — JWT RS256 verifier.
package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config configura o Verifier.
//
// Issuer: pin iss claim. Tokens com iss diferente são rejeitados.
//
// Audience: opcional. Se setado, token aud claim deve bater.
//
// Leeway: quanto tempo aceitamos de skew de relógio entre issuer e
// verifier. Default: 30s.
type Config struct {
	Issuer   string
	Audience string
	Leeway   time.Duration
}

// Verifier verifica e parseia tokens JWT usando chaves públicas RS256.
//
// Concurrency: thread-safe. Keyring atualizado via Rotate sem race
// (uso de sync.RWMutex).
type Verifier struct {
	mu       sync.RWMutex
	config   Config
	keyring  *Keyring
	parser   *jwt.Parser
}

// NewVerifier cria verifier com keyring de 1 chave (sem rotação).
//
// kid: Key ID. Multiple keys podem coexistir (key rotation grace).
// publicKeyPEM: chave pública RSA em formato PEM.
func NewVerifier(config Config, keyring *Keyring) *Verifier {
	if config.Leeway == 0 {
		config.Leeway = 30 * time.Second
	}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(config.Leeway),
	}
	if config.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(config.Issuer))
	}
	if config.Audience != "" {
		opts = append(opts, jwt.WithAudience(config.Audience))
	}
	opts = append(opts, jwt.WithExpirationRequired())

	return &Verifier{
		config:  config,
		keyring: keyring,
		parser:  jwt.NewParser(opts...),
	}
}

// Verify valida o token e retorna as claims em tipo seguro.
//
// Errors wrapam jwt.ErrToken* para comparação. Use errors.Is.
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	keyfunc := func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("auth: token sem kid — required for rotation")
		}
		v.mu.RLock()
		defer v.mu.RUnlock()
		key, ok := v.keyring.Get(kid)
		if !ok {
			return nil, fmt.Errorf("auth: kid %q não registrado", kid)
		}
		return key.PublicKey, nil
	}

	token, err := v.parser.Parse(tokenString, keyfunc)
	if err != nil {
		return nil, fmt.Errorf("auth: parse token: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("auth: claims unexpected type")
	}

	claims := &Claims{}
	if err := mapClaimsToStruct(mapClaims, claims); err != nil {
		return nil, fmt.Errorf("auth: claims invalid: %w", err)
	}

	if err := claims.Validate(); err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	return claims, nil
}

// mapClaimsToStruct converte jwt.MapClaims para Claims.
//
// Tratamento especial:
//   - exp / iat como time.Time (não int64)
//   - role como Role (não string)
func mapClaimsToStruct(m jwt.MapClaims, c *Claims) error {
	if v, ok := m["sub"].(string); ok {
		c.Sub = v
	}
	if v, ok := m["if_id"].(string); ok {
		c.IFID = v
	}
	if v, ok := m["role"].(string); ok {
		c.Role = Role(v)
	}
	if v, ok := m["iss"].(string); ok {
		c.Iss = v
	}
	if v, ok := m["exp"].(float64); ok {
		c.Exp = time.Unix(int64(v), 0)
	}
	if v, ok := m["iat"].(float64); ok {
		c.Iat = time.Unix(int64(v), 0)
	}
	if v, ok := m["jti"].(string); ok {
		c.Jti = v
	}
	return nil
}

// ParsePublicKeyPEM parseia chave pública RSA em formato PEM
// (ex: "-----BEGIN PUBLIC KEY-----...").
func ParsePublicKeyPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("auth: failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("auth: parse PKIX public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("auth: not RSA public key")
	}
	return rsaPub, nil
}
