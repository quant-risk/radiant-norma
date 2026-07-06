// cmd/secret-migrate: CLI one-shot para migrar secrets de env vars → AWS Secrets Manager.
//
// Uso típico:
//
//	# Setup
//	export RADIANT_SECRETS_BACKEND=aws
//	export AWS_REGION=sa-east-1
//	# (IAM role já configurada)
//
//	# Migrar 1 secret
//	secret-migrate migrate \
//	    --from-env=RADIANT_SECRET_BACEN_SENHA_123450001_FULANO \
//	    --to=bacen/senha/123450001.fulano
//
//	# Migrar lista de secrets (batch)
//	secret-migrate migrate-batch --file=secrets.json
//
//	# Listar secrets já migrados
//	secret-migrate list --prefix=bacen/   [Sprint 29+: AWS ListSecrets]
//
// Decisão Sprint 28: tool de migração é one-shot, NÃO roda em prod como daemon.
// Apaga env var do processo após sucesso (com --delete-env).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/fortvna/radiant-norma/backend/internal/secrets"
)

const usage = `secret-migrate — CLI para migrar secrets entre backends.

Subcomandos:
  migrate         Migra 1 secret de env var → backend configurado
  migrate-batch   Migra lista de secrets de arquivo JSON
  list            Lista secrets no backend (filtro por prefix) [AWS only — Sprint 29+]
  version         Imprime versão

Env vars:
  RADIANT_SECRETS_BACKEND   "env" | "aws" | "memory" (default: env)
  AWS_REGION                obrigatório para backend=aws
  AWS_*                     outras vars de AWS SDK chain

Exit codes:
  0  sucesso
  1  erro genérico
  2  erro de validação (input inválido)
  3  erro de backend (AWS access denied, feature não suportada, etc)
`

type migrateConfig struct {
	fromEnv   string
	toName    string
	deleteEnv bool
	dryRun    bool
	quiet     bool
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	switch subcommand {
	case "migrate":
		if err := runMigrate(args, logger); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(exitCode(err))
		}
	case "migrate-batch":
		if err := runMigrateBatch(args, logger); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(exitCode(err))
		}
	case "list":
		if err := runList(args, logger); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %v\n", err)
			os.Exit(exitCode(err))
		}
	case "version":
		fmt.Println("secret-migrate v3.23.0 (Sprint 28)")
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "subcomando desconhecido: %q\n\n%s", subcommand, usage)
		os.Exit(2)
	}
}

func runMigrate(args []string, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	var cfg migrateConfig
	fs.StringVar(&cfg.fromEnv, "from-env", "", "env var name to read value from (required)")
	fs.StringVar(&cfg.toName, "to", "", "secret name in backend (required)")
	fs.BoolVar(&cfg.deleteEnv, "delete-env", false, "unset env var after successful migration")
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "show what would happen without doing it")
	fs.BoolVar(&cfg.quiet, "quiet", false, "silence info logs")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parse: %w", err)
	}

	if cfg.fromEnv == "" {
		return &validationErr{msg: "--from-env required"}
	}
	if cfg.toName == "" {
		return &validationErr{msg: "--to required"}
	}

	ctx := context.Background()
	mgr, err := secrets.NewManagerFromEnv(ctx, logger)
	if err != nil {
		return fmt.Errorf("init manager: %w", err)
	}

	logger.Info("secret-migrate",
		"from_env", cfg.fromEnv,
		"to_name", cfg.toName,
		"backend", mgr.Backend(),
		"dry_run", cfg.dryRun,
	)

	value := os.Getenv(cfg.fromEnv)
	if value == "" {
		return &validationErr{msg: fmt.Sprintf("env var %q is empty or not set", cfg.fromEnv)}
	}

	if cfg.dryRun {
		fmt.Printf("DRY-RUN: would migrate %s → %q (value length: %d chars) in backend=%s\n",
			cfg.fromEnv, cfg.toName, len(value), mgr.Backend())
		return nil
	}

	// SAFETY: warn if env value looks like a real password (avoid mass-migrate disaster)
	if looksLikeSecret(value) {
		fmt.Fprintf(os.Stderr, "AVISO: env var %q value parece ser um secret real (length=%d, has-mixed-case=%v).\n",
			cfg.fromEnv, len(value), hasMixedCase(value))
		fmt.Fprintf(os.Stderr, "Confirme a operação digitando 'YES' (case-sensitive): ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "YES" {
			return &validationErr{msg: "user cancelled (confirmation failed)"}
		}
	}

	// Put no backend
	s, err := mgr.Put(ctx, cfg.toName, value)
	if err != nil {
		return fmt.Errorf("put failed: %w", err)
	}

	fmt.Printf("migrated=true backend=%s name=%q version_id=%s created_at=%s\n",
		mgr.Backend(), s.Name, s.VersionID, s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))

	// Validação 49 (F-S28-49-C): confirmação adicional quando --delete-env + value parece secret real.
	// Defesa contra typo (env var errada = secret válido some sem warning).
	if cfg.deleteEnv && looksLikeSecret(value) {
		fmt.Fprintf(os.Stderr, "Confirmar remoção de %q do env do processo (digite 'YES'): ", cfg.fromEnv)
		var confirmDelete string
		fmt.Scanln(&confirmDelete)
		if confirmDelete != "YES" {
			fmt.Fprintf(os.Stderr, "AVISO: secret migrado para backend, mas env var NÃO foi removida. Remova manualmente com: unset %s\n", cfg.fromEnv)
			return nil
		}
	}

	if cfg.deleteEnv {
		if err := os.Unsetenv(cfg.fromEnv); err != nil {
			logger.Warn("failed to unset env var", "env", cfg.fromEnv, "err", err)
		} else {
			logger.Info("env var unset", "env", cfg.fromEnv)
		}
	}

	return nil
}

func runMigrateBatch(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("migrate-batch", flag.ContinueOnError)
	var file string
	var dryRun bool
	fs.StringVar(&file, "file", "", "JSON file with [{from_env, to_name, delete_env}] entries (required)")
	fs.BoolVar(&dryRun, "dry-run", false, "show what would happen")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parse: %w", err)
	}
	if file == "" {
		return &validationErr{msg: "--file required"}
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var entries []struct {
		FromEnv   string `json:"from_env"`
		ToName    string `json:"to_name"`
		DeleteEnv bool   `json:"delete_env"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	ctx := context.Background()
	mgr, err := secrets.NewManagerFromEnv(ctx, logger)
	if err != nil {
		return fmt.Errorf("init manager: %w", err)
	}

	fmt.Printf("backend=%s entries=%d dry_run=%v\n\n", mgr.Backend(), len(entries), dryRun)

	successCount := 0
	for i, e := range entries {
		fmt.Printf("[%d/%d] %s → %q ... ", i+1, len(entries), e.FromEnv, e.ToName)
		if dryRun {
			fmt.Println("(dry-run, skipped)")
			continue
		}
		val := os.Getenv(e.FromEnv)
		if val == "" {
			fmt.Println("SKIP (env var empty)")
			continue
		}
		_, err := mgr.Put(ctx, e.ToName, val)
		if err != nil {
			fmt.Printf("FAIL: %v\n", err)
			continue
		}
		if e.DeleteEnv {
			_ = os.Unsetenv(e.FromEnv)
		}
		fmt.Println("OK")
		successCount++
	}

	fmt.Printf("\nsummary: success=%d total=%d\n", successCount, len(entries))
	return nil
}

func runList(args []string, logger *slog.Logger) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	var prefix string
	fs.StringVar(&prefix, "prefix", "", "filter secrets by prefix (e.g. 'bacen/')")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parse: %w", err)
	}

	ctx := context.Background()
	mgr, err := secrets.NewManagerFromEnv(ctx, logger)
	if err != nil {
		return fmt.Errorf("init manager: %w", err)
	}

	// Validação 50 (F-S28-50-A): list requer backend AWS — EnvManager/MemoryManager
	// não implementam listagem. Retornar erro tipado (exit 3) em vez de exit 0
	// silencioso, pra caller distinguir "funcionou, lista vazia" de
	// "feature não suportada neste backend".
	if mgr.Backend() != secrets.BackendAWS {
		return &backendErr{msg: fmt.Sprintf("list not supported on backend=%s (apenas AWS Secrets Manager suporta ListSecrets). Sprint 29+ adiciona suporte", mgr.Backend())}
	}

	// AWS path: AWSManager.List() será implementado em Sprint 29 (BACEN homolog smoke
	// precisa de listagem operacional). Por ora, retorna erro transparente.
	return &backendErr{msg: fmt.Sprintf("AWS ListSecrets ainda não implementado (Sprint 29+). Backend=%s, prefix=%q",
		mgr.Backend(), prefix)}
}

// looksLikeSecret heurística simples: > 8 chars + mixed case OR contém dígito.
func looksLikeSecret(s string) bool {
	if len(s) < 8 {
		return false
	}
	return hasMixedCase(s) || containsDigit(s)
}

func hasMixedCase(s string) bool {
	hasUpper, hasLower := false, false
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if hasUpper && hasLower {
			return true
		}
	}
	return false
}

func containsDigit(s string) bool {
	return strings.ContainsAny(s, "0123456789")
}

// exitCode mapeia erros para exit codes consistentes com outros CLIs (senhaws-rotate, sta-submit).
func exitCode(err error) int {
	if _, ok := err.(*validationErr); ok {
		return 2
	}
	if _, ok := err.(*backendErr); ok {
		return 3
	}
	if secrets.IsNotFound(err) || secrets.IsAccessDenied(err) {
		return 3
	}
	return 1
}

type validationErr struct{ msg string }

func (e *validationErr) Error() string { return e.msg }
func (e *validationErr) Is(target error) bool {
	_, ok := target.(*validationErr)
	return ok
}

// backendErr sinaliza erro operacional do backend (feature não suportada,
// AWS access denied, etc). Exit 3 — caller distingue de erro genérico (1)
// e validação (2). Validação 50: usado por runList pra reportar "list not
// supported" de forma honesta.
type backendErr struct{ msg string }

func (e *backendErr) Error() string { return e.msg }
func (e *backendErr) Is(target error) bool {
	_, ok := target.(*backendErr)
	return ok
}
