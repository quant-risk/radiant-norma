// Command senhaws-rotate — CLI standalone para gestão de credenciais Sisbacen.
//
// Uso:
//
//	check   — consulta dias até vencimento da senha
//	rotate  — gera nova senha + altera no BACEN
//	info    — imprime config (mascarada) + status do servidor
//
// Exemplos:
//
//	# Checar vencimento (cron diário)
//	senhaws-rotate check
//	# → imprime: dias_vencimento=30  status=ok  threshold=7
//	# → exit 0 se > threshold, exit 1 se <= threshold
//
//	# Rotacionar senha
//	senhaws-rotate rotate > /tmp/newpass.txt
//	# → imprime: senha_alterada=true  nova_senha=abc123...
//	# → caller armazena /tmp/newpass.txt em secret manager e remove o arquivo
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
//
// Exit codes:
//
//	0  sucesso
//	1  erro genérico / precisa rotacionar (check)
//	2  erro de validação client-side (input inválido)
//	3  erro BACEN (rejeição formal — caller investiga)
//
// Referência: SPRINT_24_RESEARCH.md + manual BACEN §9.1+§9.2.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

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
		// Silent logger — slog.New(slog.DiscardHandler equivalent via NewTextHandler(io.Discard).
		return slog.New(slog.NewTextHandler(discardWriter{}, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// discardWriter é io.Writer que descarta tudo (silencia logs).
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// maskUser mascara user Sisbacen mantendo prefixo + suffixo.
// Ex: "123450001.fulano" → "12***01.fulano".
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
		fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
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
func runRotate(ctx context.Context, cfg *config, logger *slog.Logger) int {
	client, err := senhaws.NewSenhawsClient(senhaws.SenhawsConfig{
		BaseURL:           cfg.baseURL,
		User:              cfg.user,
		Password:          cfg.password,
		Timeout:           cfg.timeout,
		Logger:            logger,
		AllowInsecureHTTP: cfg.allowInsecureHTTP,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
		return exitClientError
	}

	novaSenha := senhaws.GerarSenhaRandom()

	if err := client.AlterarSenha(ctx, novaSenha); err != nil {
		var senErr *senhaws.SenhaError
		if errors.As(err, &senErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN senhaws %d: %s\n", senErr.StatusCode, senErr.Message)
			return exitBACENError
		}
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		// Erro client-side (validação) vs transporte. Client-side retorna errors.New
		// ou fmt.Errorf direto — mensagem menciona "deve ter" ou "nao pode" — heurística.
		if strings.Contains(err.Error(), "deve") || strings.Contains(err.Error(), "não pode") || strings.Contains(err.Error(), "diferente") {
			return exitClientError
		}
		return exitGenericError
	}

	// Sucesso — imprime nova senha no stdout para caller capturar.
	// Caller DEVE redirecionar > /tmp/newpass.txt e armazenar em secret manager.
	fmt.Printf("senha_alterada=true  nova_senha=%s\n", novaSenha)
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
		fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
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
		exitCode = runRotate(ctx, cfg, logger)
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
