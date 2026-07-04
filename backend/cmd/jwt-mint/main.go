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
// Sprint 8a (v2.1.0): lógica de signing delegada para auth.Signer
// (DRY com /v1/auth/dev-token in-process mint).
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/auth"
)

func main() {
	var (
		privKeyPath = flag.String("private-key", "./dev-private.pem", "caminho da private key (PEM)")
		kid         = flag.String("kid", "k1", "key ID")
		issuer      = flag.String("issuer", "radiant-norma", "issuer (iss claim)")
		ifID        = flag.String("if", "demo", "tenant identifier (if_id claim)")
		role        = flag.String("role", "if", "role (if/admin/readonly)")
		sub         = flag.String("sub", "dev-user", "subject (sub claim)")
		ttl         = flag.Duration("ttl", auth.TTLDefault, "token TTL (cap: 30 dias)")
		quiet       = flag.Bool("quiet", false, "only print token (sem logs)")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Validação de role
	switch auth.Role(*role) {
	case auth.RoleIF, auth.RoleAdmin, auth.RoleReadOnly:
	default:
		logger.Error("role inválido", "got", *role, "want", "if|admin|readonly")
		os.Exit(1)
	}

	// Cap TTL pra evitar geração acidental de tokens de vida excessiva.
	if *ttl > auth.TTLCap {
		logger.Warn("ttl além do cap, clamped", "got", *ttl, "cap", auth.TTLCap)
		*ttl = auth.TTLCap
	}

	signer, err := auth.NewSignerFromFile(*privKeyPath, *kid, *issuer)
	if err != nil {
		logger.Error("load signer", "err", err, "path", *privKeyPath)
		os.Exit(1)
	}

	// Allow --sub override of sub claim (default uses if_id).
	subClaim := *sub
	if subClaim == "" {
		subClaim = *ifID
	}

	claims := auth.Claims{
		Sub:  subClaim,
		IFID: *ifID,
		Role: auth.Role(*role),
		Iss:  *issuer,
		Exp:  time.Now().Add(*ttl),
	}

	signed, exp, err := signer.Mint(claims)
	if err != nil {
		logger.Error("mint", "err", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "=== JWT minted (Sprint 8a v2.1.0) ===\n")
		fmt.Fprintf(os.Stderr, "kid:       %s\n", *kid)
		fmt.Fprintf(os.Stderr, "issuer:    %s\n", *issuer)
		fmt.Fprintf(os.Stderr, "if_id:     %s\n", *ifID)
		fmt.Fprintf(os.Stderr, "role:      %s\n", *role)
		fmt.Fprintf(os.Stderr, "sub:       %s\n", subClaim)
		fmt.Fprintf(os.Stderr, "ttl:       %s\n", *ttl)
		fmt.Fprintf(os.Stderr, "expires:   %s\n", exp.Format(time.RFC3339))
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		fmt.Fprintf(os.Stderr, "  curl -H 'Authorization: Bearer <TOKEN>' \\\n")
		fmt.Fprintf(os.Stderr, "       http://localhost:8080/v1/schemas\n\n")
		fmt.Fprintf(os.Stderr, "Token:\n")
	}
	fmt.Println(signed)
}

// init: shell-completion hint quando --help.
func init() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-") &&
		!strings.Contains(strings.Join(os.Args, " "), "--help") {
		// only the -h/--help flag triggers this — already handled by flag.Parse
	}
}
