// Command senhaws-rotate — CLI standalone para gestão de credenciais Sisbacen.
//
// Uso:
//
//	check   — consulta dias até vencimento da senha
//	rotate  — gera nova senha + altera no BACEN
//	apply   — gera nova senha + altera no BACEN + atualiza secrets manager (Sprint 28+)
//	info    — imprime config (mascarada) + status do servidor
//
// Exemplos:
//
//	# Checar vencimento (cron diário)
//	senhaws-rotate check
//	# → imprime: dias_vencimento=30  status=ok  threshold=7
//	# → exit 0 se > threshold, exit 1 se <= threshold
//
//	# Rotacionar senha (modo antigo — caller gerencia secret manager)
//	senhaws-rotate rotate > /tmp/newpass.txt
//	# → imprime: senha_alterada=true  nova_senha=abc123...
//	# → caller armazena /tmp/newpass.txt em secret manager e remove o arquivo
//
//	# Rotacionar + atualizar secret manager (Sprint 28+ — RECOMENDADO)
//	RADIANT_SECRETS_BACKEND=aws senhaws-rotate apply
//	# → imprime: senha_alterada=true secret_updated=true backend=aws name="bacen/senha/..."
//	# → exit 0
//
//	# Debug
//	senhaws-rotate info
//	# → imprime: base_url, user mascarado, timeout, status BACEN
//
// Variáveis de ambiente (alternativa a flags):
//
//	SENHAWS_BASE_URL    https://www9.bcb.gov.br/senhaws (homol) ou www3 (prod)
//	SENHAWS_USER        formato UUUUUDDDD.operador
//	SENHAWS_PASSWORD    senha Sisbacen ATUAL — NÃO log (F13.8)
//	SENHAWS_TIMEOUT     default 30s
//	SENHAWS_MAX_DAYS    threshold para check exit code, default 7
//	RADIANT_SECRETS_BACKEND   "env" (default) | "aws" | "memory"
//	AWS_REGION                obrigatório para RADIANT_SECRETS_BACKEND=aws
//
// Exit codes:
//
//	0  sucesso
//	1  erro genérico / precisa rotacionar (check)
//	2  erro de validação client-side (input inválido)
//	3  erro BACEN (rejeição formal — caller investiga)
//
// Referência: SPRINT_24_RESEARCH.md + SPRINT_28_RESEARCH.md + manual BACEN §9.1+§9.2.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/secrets"
	"github.com/fortvna/radiant-norma/backend/internal/senhaws"
)

// Exit codes (consistente com convention Unix).
const (
	exitOK           = 0
	exitGenericError = 1
	exitClientError  = 2
	exitBACENError   = 3
)

// config agrega inputs (flags + env vars).
type config struct {
	baseURL           string
	user              string
	password          string
	timeout           time.Duration
	maxDays           int
	quiet             bool
	allowInsecureHTTP bool
}

// loadConfig faz parse de flags + env vars. Env vars têm prioridade (padrão cmd/api).
func loadConfig(args []string) (*config, error) {
	fs := flag.NewFlagSet("senhaws-rotate", flag.ContinueOnError)
	cfg := &config{}
	fs.StringVar(&cfg.baseURL, "base-url", os.Getenv("SENHAWS_BASE_URL"), "BaseURL do senhaws BACEN (env: SENHAWS_BASE_URL)")
	fs.StringVar(&cfg.user, "user", os.Getenv("SENHAWS_USER"), "Usuário Sisbacen formato UUUUUDDDD.operador (env: SENHAWS_USER)")
	fs.StringVar(&cfg.password, "password", os.Getenv("SENHAWS_PASSWORD"), "Senha Sisbacen ATUAL (env: SENHAWS_PASSWORD). NÃO usar em linha de comando — preferir env var")
	timeoutStr := fs.String("timeout", envOrDefault("SENHAWS_TIMEOUT", "30s"), "Timeout HTTP (env: SENHAWS_TIMEOUT)")
	maxDaysStr := fs.String("max-days", envOrDefault("SENHAWS_MAX_DAYS", "7"), "Threshold em dias para check exit 1 (env: SENHAWS_MAX_DAYS)")
	fs.BoolVar(&cfg.quiet, "quiet", false, "Silencia logs de stderr (apenas stdout)")
	fs.BoolVar(&cfg.allowInsecureHTTP, "allow-insecure-http", false, "Permite BaseURL HTTP (apenas para testes dev com httptest). NUNCA em produção.")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --timeout: %w", err)
	}
	cfg.timeout = timeout

	var maxDays int
	if _, err := fmt.Sscanf(*maxDaysStr, "%d", &maxDays); err != nil {
		return nil, fmt.Errorf("invalid --max-days: %w", err)
	}
	if maxDays < 0 {
		return nil, errors.New("--max-days deve ser >= 0")
	}
	cfg.maxDays = maxDays

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// newLogger cria logger estruturado respeitando --quiet.
func newLogger(quiet bool) *slog.Logger {
	if quiet {
		// Silent logger — descarta todos os logs (io.Discard stdlib).
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// maskUser mascara user Sisbacen mantendo prefixo + sufixo.
// Ex: "123450001.fulano" → "12***.fulano" (mostra primeiros 2 chars + operador).
// Defesa contra screenshot/log acidental.
func maskUser(user string) string {
	at := strings.IndexByte(user, '.')
	if at <= 2 {
		return "***"
	}
	return user[:2] + "***" + user[at:]
}

// runCheck implementa subcomando check.
func runCheck(ctx context.Context, cfg *config, logger *slog.Logger) int {
	client, err := senhaws.NewSenhawsClient(senhaws.SenhawsConfig{
		BaseURL:           cfg.baseURL,
		User:              cfg.user,
		Password:          cfg.password,
		Timeout:           cfg.timeout,
		Logger:            logger,
		AllowInsecureHTTP: cfg.allowInsecureHTTP,
	})
	if err != nil {
		// Validação 46 (F-S25-46-7): erros de config são *ValidationError agora.
		var valErr *senhaws.ValidationError
		if errors.As(err, &valErr) {
			fmt.Fprintf(os.Stderr, "config invalida: %s\n", valErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
		}
		return exitClientError
	}

	dias, err := client.ConsultarVencimento(ctx)
	if err != nil {
		// Erro BACEN tipado vs transporte.
		var senErr *senhaws.SenhaError
		if errors.As(err, &senErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN senhaws %d: %s\n", senErr.StatusCode, senErr.Message)
			return exitBACENError
		}
		fmt.Fprintf(os.Stderr, "erro transporte: %v\n", err)
		return exitGenericError
	}

	status := "ok"
	exitCode := exitOK
	if dias <= cfg.maxDays {
		status = "expiring"
		exitCode = exitGenericError // 1 = precisa rotacionar
	}

	fmt.Printf("dias_vencimento=%d  status=%s  threshold=%d\n", dias, status, cfg.maxDays)
	return exitCode
}

// runRotate implementa subcomando rotate.
//
// novaSenha: se vazia, gera senha random via GerarSenhaRandom (default prod).
// Se passada, usa o valor (útil para testes + caller que quer senha custom).
func runRotate(ctx context.Context, cfg *config, logger *slog.Logger, novaSenha string) int {
	client, err := senhaws.NewSenhawsClient(senhaws.SenhawsConfig{
		BaseURL:           cfg.baseURL,
		User:              cfg.user,
		Password:          cfg.password,
		Timeout:           cfg.timeout,
		Logger:            logger,
		AllowInsecureHTTP: cfg.allowInsecureHTTP,
	})
	if err != nil {
		// Validação 46 (F-S25-46-7): erros de config são *ValidationError agora.
		var valErr *senhaws.ValidationError
		if errors.As(err, &valErr) {
			fmt.Fprintf(os.Stderr, "config invalida: %s\n", valErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
		}
		return exitClientError
	}

	if novaSenha == "" {
		novaSenha = senhaws.GerarSenhaRandom()
	}

	if err := client.AlterarSenha(ctx, novaSenha); err != nil {
		var senErr *senhaws.SenhaError
		if errors.As(err, &senErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN senhaws %d: %s\n", senErr.StatusCode, senErr.Message)
			return exitBACENError
		}
		var valErr *senhaws.ValidationError
		if errors.As(err, &valErr) {
			fmt.Fprintf(os.Stderr, "erro de validacao: %s\n", valErr.Message)
			return exitClientError
		}
		fmt.Fprintf(os.Stderr, "erro transporte: %v\n", err)
		return exitGenericError
	}

	// Sucesso — imprime nova senha no stdout para caller capturar.
	// Caller DEVE redirecionar > /tmp/newpass.txt e armazenar em secret manager.
	fmt.Printf("senha_alterada=true  nova_senha=%s\n", novaSenha)
	return exitOK
}

// runApply implementa subcomando apply (Sprint 28 — v3.23.0).
//
// Combina rotate + secret-manager update em uma operação atômica-ish.
//
// Fluxo:
//   1. AlterarSenha no BACEN
//   2. Put no secrets manager configurado
//   3. Audit log (futuro — Sprint 28 fica só no stderr)
//
// Failure modes:
//
//	- BACEN aceita + Manager.Put falha: imprime warning, exit 1 (caller deve
//	  re-executar apply — Manager.Put é idempotente via SecretId+SecretString).
//	- BACEN rejeita: exit 3 (sem side effect no manager).
//	- Config inválida: exit 2.
func runApply(ctx context.Context, cfg *config, logger *slog.Logger) int {
	// 1. Validate config
	if cfg.baseURL == "" || cfg.user == "" || cfg.password == "" {
		fmt.Fprintf(os.Stderr, "config invalida: --base-url, --user, --password são obrigatórios\n")
		return exitClientError
	}

	// 2. Init secrets manager
	mgr, err := secrets.NewManagerFromEnv(ctx, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "secrets manager init failed (RADIANT_SECRETS_BACKEND=%q): %v\n",
			os.Getenv("RADIANT_SECRETS_BACKEND"), err)
		fmt.Fprintf(os.Stderr, "  hint: para AWS, configure AWS_REGION. Para dev/test, use RADIANT_SECRETS_BACKEND=memory ou unset (env fallback).\n")
		return exitClientError
	}
	logger.Info("secrets manager ativo", "backend", mgr.Backend())

	// 3. Init senhaws client
	client, err := senhaws.NewSenhawsClient(senhaws.SenhawsConfig{
		BaseURL:           cfg.baseURL,
		User:              cfg.user,
		Password:          cfg.password,
		Timeout:           cfg.timeout,
		Logger:            logger,
		AllowInsecureHTTP: cfg.allowInsecureHTTP,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "senhaws client init: %v\n", err)
		return exitClientError
	}

	// 4. Generate new password
	novaSenha := senhaws.GerarSenhaRandom()
	logger.Info("gerando nova senha + alterando no BACEN", "user", maskUser(cfg.user))

	// 5. AlterarSenha no BACEN
	if err := client.AlterarSenha(ctx, novaSenha); err != nil {
		var senErr *senhaws.SenhaError
		if errors.As(err, &senErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN senhaws %d: %s\n", senErr.StatusCode, senErr.Message)
			return exitBACENError
		}
		var valErr *senhaws.ValidationError
		if errors.As(err, &valErr) {
			fmt.Fprintf(os.Stderr, "erro de validacao: %s\n", valErr.Message)
			return exitClientError
		}
		fmt.Fprintf(os.Stderr, "erro transporte BACEN: %v\n", err)
		return exitGenericError
	}

	logger.Info("senha alterada no BACEN com sucesso")

	// 6. Update secrets manager
	// Naming convention: bacen/senha/{user} onde user mantém formato Sisbacen
	// UUUUUDDDD.operador. EnvManager normaliza "." → "_" em env vars
	// (via envName), mas secret name mantém "." para readability no AWS Console.
	secretName := fmt.Sprintf("bacen/senha/%s", cfg.user)

	updated, err := mgr.Put(ctx, secretName, novaSenha)
	if err != nil {
		// CRITICAL: BACEN accepted but manager failed.
		// Caller must re-execute apply (Put is idempotent).
		fmt.Fprintf(os.Stderr, "WARN: senha alterada no BACEN mas FALHA ao atualizar %s manager: %v\n", mgr.Backend(), err)
		fmt.Fprintf(os.Stderr, "      nova senha está apenas no BACEN. Re-execute `senhaws-rotate apply` para persistir.\n")
		fmt.Fprintf(os.Stderr, "      Senha nova (capture agora!): %s\n", novaSenha)
		return exitGenericError
	}

	logger.Info("secrets manager atualizado",
		"name", secretName,
		"backend", mgr.Backend(),
		"version_id", updated.VersionID,
	)

	fmt.Printf("senha_alterada=true  secret_updated=true  backend=%s  name=%q  version_id=%s\n",
		mgr.Backend(), secretName, updated.VersionID)
	return exitOK
}

// runInfo implementa subcomando info.
func runInfo(ctx context.Context, cfg *config, logger *slog.Logger) int {
	fmt.Printf("base_url=%s\n", cfg.baseURL)
	fmt.Printf("user=%s  (mascarado)\n", maskUser(cfg.user))
	fmt.Printf("timeout=%s\n", cfg.timeout)
	fmt.Printf("max_days=%d\n", cfg.maxDays)

	client, err := senhaws.NewSenhawsClient(senhaws.SenhawsConfig{
		BaseURL:           cfg.baseURL,
		User:              cfg.user,
		Password:          cfg.password,
		Timeout:           cfg.timeout,
		Logger:            logger,
		AllowInsecureHTTP: cfg.allowInsecureHTTP,
	})
	if err != nil {
		fmt.Printf("bacen_status=config_error\n")
		// Validação 46 (F-S25-46-7): erros de config são *ValidationError agora.
		var valErr *senhaws.ValidationError
		if errors.As(err, &valErr) {
			fmt.Fprintf(os.Stderr, "config invalida: %s\n", valErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
		}
		return exitClientError
	}

	dias, err := client.ConsultarVencimento(ctx)
	if err != nil {
		fmt.Printf("bacen_status=error\n")
		var senErr *senhaws.SenhaError
		if errors.As(err, &senErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN senhaws %d: %s\n", senErr.StatusCode, senErr.Message)
			return exitBACENError
		}
		fmt.Fprintf(os.Stderr, "erro transporte: %v\n", err)
		return exitGenericError
	}

	fmt.Printf("bacen_status=ok  dias_vencimento=%d\n", dias)
	return exitOK
}

// usage imprime help.
func usage() {
	fmt.Fprintf(os.Stderr, `Usage: senhaws-rotate <subcommand> [flags]

Subcommands:
  check    Consulta dias até vencimento. Exit 0 se > max-days, exit 1 se <= max-days.
  rotate   Gera nova senha + altera no BACEN. Imprime nova senha no stdout.
  apply    Gera nova senha + altera no BACEN + atualiza secrets manager. (Sprint 28)
  info     Imprime config mascarada + status do servidor BACEN.

Flags:
  --base-url             URL do senhaws BACEN (env: SENHAWS_BASE_URL)
  --user                 Usuário Sisbacen (env: SENHAWS_USER)
  --password             Senha Sisbacen ATUAL (env: SENHAWS_PASSWORD). NÃO usar em linha de comando.
  --timeout              Timeout HTTP (env: SENHAWS_TIMEOUT, default 30s)
  --max-days             Threshold em dias para check (env: SENHAWS_MAX_DAYS, default 7)
  --quiet                Silencia logs de stderr
  --allow-insecure-http  Permite BaseURL HTTP (apenas para testes dev). NUNCA em produção.

Exit codes:
  0  sucesso
  1  erro genérico / precisa rotacionar (check)
  2  erro de validação client-side
  3  erro BACEN (rejeição formal)

Variáveis de ambiente: SENHAWS_BASE_URL, SENHAWS_USER, SENHAWS_PASSWORD,
SENHAWS_TIMEOUT, SENHAWS_MAX_DAYS.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitGenericError)
	}

	subcommand := os.Args[1]
	cfg, err := loadConfig(os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flag parse: %v\n", err)
		usage()
		os.Exit(exitClientError)
	}

	logger := newLogger(cfg.quiet)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout+5*time.Second)
	defer cancel()

	var exitCode int
	switch subcommand {
	case "check":
		exitCode = runCheck(ctx, cfg, logger)
	case "rotate":
		exitCode = runRotate(ctx, cfg, logger, "")
	case "apply":
		exitCode = runApply(ctx, cfg, logger)
	case "info":
		exitCode = runInfo(ctx, cfg, logger)
	case "-h", "--help", "help":
		usage()
		exitCode = exitOK
	default:
		fmt.Fprintf(os.Stderr, "subcomando desconhecido: %q\n", subcommand)
		usage()
		exitCode = exitGenericError
	}

	os.Exit(exitCode)
}
