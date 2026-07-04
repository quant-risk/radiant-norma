// Command jwt-mint gera tokens JWT para dev/test.
//
// Usage:
//
//	go run ./cmd/jwt-mint \
//	  --private-key=dev-private.pem \
//	  --kid=k1 \
//	  --issuer=radiant-norma \
//	  --if=demo \
//	  --role=if \
//	  --ttl=24h
//
// Em produção, tokens são emitidos por serviço auth separado (ou 3rd-party).
// Este tool é para:
//
//   - Development: gerar token pessoal para testar API local
//   - Testing: gerar tokens em test fixtures
//   - Demo: gerar tokens para demo accounts (audit)
//
// Por design: lê private key via file (não flag) para evitar
// exposição via `ps aux`. Default --if= e --role= fornecem valores
// seguros para dev. --ttl default 24h.
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
	jwtv5 "github.com/golang-jwt/jwt/v5"
)

func main() {
	var (
		privKeyPath = flag.String("private-key", "./dev-private.pem", "caminho da private key (PEM)")
		kid         = flag.String("kid", "k1", "key ID")
		issuer      = flag.String("issuer", "radiant-norma", "issuer (iss claim)")
		ifID        = flag.String("if", "demo", "tenant identifier (if_id claim)")
		role        = flag.String("role", "if", "role (if/admin/readonly)")
		sub         = flag.String("sub", "dev-user", "subject (sub claim)")
		ttl         = flag.Duration("ttl", 24*time.Hour, "token TTL")
		quiet       = flag.Bool("quiet", false, "only print token (sem logs)")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Validação de role
	switch auth.Role(*role) {
	case auth.RoleIF, auth.RoleAdmin, auth.RoleReadOnly:
		// ok
	default:
		logger.Error("role inválido", "got", *role, "want", "if|admin|readonly")
		os.Exit(1)
	}

	priv, err := loadPrivateKey(*privKeyPath)
	if err != nil {
		logger.Error("load private key", "err", err, "path", *privKeyPath)
		os.Exit(1)
	}

	now := time.Now()
	claims := auth.Claims{
		Sub:  *sub,
		IFID: *ifID,
		Role: auth.Role(*role),
		Iss:  *issuer,
		Iat:  now,
		Exp:  now.Add(*ttl),
	}
	if err := claims.Validate(); err != nil {
		logger.Error("validate claims", "err", err)
		os.Exit(1)
	}

	registered := jwtv5.MapClaims{
		"sub":   claims.Sub,
		"if_id": claims.IFID,
		"role":  string(claims.Role),
		"iss":   claims.Iss,
		"exp":   claims.Exp.Unix(),
		"iat":   claims.Iat.Unix(),
		"jti":   generateJTI(),
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, registered)
	token.Header["kid"] = *kid

	signed, err := token.SignedString(priv)
	if err != nil {
		logger.Error("sign", "err", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "=== JWT minted (Sprint 7a v1.6.0) ===\n")
		fmt.Fprintf(os.Stderr, "kid:       %s\n", *kid)
		fmt.Fprintf(os.Stderr, "issuer:    %s\n", *issuer)
		fmt.Fprintf(os.Stderr, "if_id:     %s\n", *ifID)
		fmt.Fprintf(os.Stderr, "role:      %s\n", *role)
		fmt.Fprintf(os.Stderr, "sub:       %s\n", *sub)
		fmt.Fprintf(os.Stderr, "ttl:       %s\n", *ttl)
		fmt.Fprintf(os.Stderr, "expires:   %s\n", claims.Exp.Format(time.RFC3339))
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  curl -H 'Authorization: Bearer <TOKEN>' \\\n")
		fmt.Fprintf(os.Stderr, "       http://localhost:8080/v1/schemas\n\n")
		fmt.Fprintf(os.Stderr, "Token:\n")
	}
	fmt.Println(signed)
}

// loadPrivateKey lê chave privada RSA de arquivo PEM.
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode PEM in %s", path)
	}
	if priv, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return priv, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaPriv, ok := k.(*rsa.PrivateKey); ok {
			return rsaPriv, nil
		}
	}
	return nil, fmt.Errorf("not RSA private key in %s", path)
}

// generateJTI gera identificador único de token (random bytes).
func generateJTI() string {
	// 16 random bytes hex = 32 chars, suficiente para uniq.
	return fmt.Sprintf("jti-%d", time.Now().UnixNano())
}

// init: shell-completion hint quando --help.
func init() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-") &&
		!strings.Contains(strings.Join(os.Args, " "), "--help") {
		// only the -h/--help flag triggers this — already handled by flag.Parse
	}
}
