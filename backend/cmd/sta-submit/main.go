// Command sta-submit — CLI standalone para envio de CADOC ao BACEN STA WS.
//
// Uso:
//
//	sta-submit --xml-file=/path/to/cadoc3040.xml \
//	           --cadoc-code=3040 \
//	           --data-base=2024-12 \
//	           --cnpj=demo-bank
//
// Variáveis de ambiente:
//
//	RADIANT_STA_BACKEND         stub (default) | ws (BACEN real)
//	RADIANT_STA_WS_URL          https://sta-h.bcb.gov.br/staws (homolog)
//	RADIANT_STA_SISBACEN_USER   formato UUUUUDDDD.operador
//	RADIANT_STA_SISBACEN_PASSWORD senha Sisbacen
//	RADIANT_STA_TIMEOUT_SECONDS default 30s
//
// Exit codes:
//
//	0  aceito pelo BACEN
//	1  rejeitado pelo BACEN / erro de transporte
//	2  erro de validação client-side (input inválido)
//	3  erro BACEN formal (rejeição com status >= 500 ou similar)
//
// Reusa sta.NewClientFromEnv (Sprint 18+) — mesma fábrica que cmd/api usa.
//
// Referência: SPRINT_26_RESEARCH.md + manual BACEN §5.1+§5.2.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

// Exit codes (consistente com cmd/senhaws-rotate).
const (
	exitOK          = 0
	exitRejected    = 1 // rejeitado OU transporte
	exitClientError = 2
	exitBACENError  = 3
)

// config agrega inputs (flags + env vars).
type config struct {
	xmlFile   string
	cadocCode string
	dataBase  string
	cnpj      string
	quiet     bool
	timeout   time.Duration
}

// staClient é interface mínima que runSubmit usa (test injection point).
// Em produção, sta.NewClientFromEnv retorna sta.Client que implementa isso.
type staClient interface {
	Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error)
}

// staNewClientFromEnv é variável de função para permitir injeção em tests.
// Default é sta.NewClientFromEnv. Tests sobrescrevem pra mock httptest.
var staNewClientFromEnv func(logger *slog.Logger) (staClient, error) = func(logger *slog.Logger) (staClient, error) {
	c, err := sta.NewClientFromEnv(logger)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// loadConfig faz parse de flags + env vars.
func loadConfig(args []string) (*config, error) {
	fs := flag.NewFlagSet("sta-submit", flag.ContinueOnError)
	cfg := &config{}
	fs.StringVar(&cfg.xmlFile, "xml-file", os.Getenv("STA_SUBMIT_XML_FILE"), "Caminho do arquivo XML do CADOC (env: STA_SUBMIT_XML_FILE)")
	fs.StringVar(&cfg.cadocCode, "cadoc-code", envOrDefault("STA_SUBMIT_CADOC_CODE", "3040"), "Código do CADOC (env: STA_SUBMIT_CADOC_CODE, default 3040)")
	fs.StringVar(&cfg.dataBase, "data-base", os.Getenv("STA_SUBMIT_DATA_BASE"), "Data-base no formato YYYY-MM (env: STA_SUBMIT_DATA_BASE)")
	fs.StringVar(&cfg.cnpj, "cnpj", envOrDefault("STA_SUBMIT_CNPJ", "demo-bank"), "CNPJ do IF (env: STA_SUBMIT_CNPJ, default demo-bank)")
	fs.BoolVar(&cfg.quiet, "quiet", false, "Silencia logs de stderr")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	cfg.timeout = 30 * time.Second // herdado do NewClientFromEnv (Sprint 18)
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
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// runSubmit implementa o subcomando submit (único).
func runSubmit(ctx context.Context, cfg *config, logger *slog.Logger) int {
	// Validação client-side.
	if cfg.xmlFile == "" {
		fmt.Fprintf(os.Stderr, "config invalida: --xml-file requerido\n")
		return exitClientError
	}
	if cfg.dataBase == "" {
		fmt.Fprintf(os.Stderr, "config invalida: --data-base requerido (formato YYYY-MM)\n")
		return exitClientError
	}

	// Lê XML do arquivo.
	xmlBytes, err := os.ReadFile(cfg.xmlFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config invalida: erro lendo %s: %v\n", cfg.xmlFile, err)
		return exitClientError
	}
	if len(xmlBytes) == 0 {
		fmt.Fprintf(os.Stderr, "config invalida: arquivo %s vazio\n", cfg.xmlFile)
		return exitClientError
	}

	// Cria cliente STA via factory injetável (mesma usada pelo cmd/api).
	client, err := staNewClientFromEnv(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro criando STA client: %v\n", err)
		return exitClientError
	}

	// Submete.
	sub := &sta.Submission{
		CadocCode: cfg.cadocCode,
		DataBase:  cfg.dataBase,
		CNPJ:      cfg.cnpj,
		XML:       string(xmlBytes),
	}

	result, err := client.Submit(ctx, sub)
	if err != nil {
		var staErr *sta.STAError
		if errors.As(err, &staErr) {
			fmt.Fprintf(os.Stderr, "erro BACEN STA %d: %s\n", staErr.StatusCode, staErr.Message)
			return exitBACENError
		}
		fmt.Fprintf(os.Stderr, "erro transporte: %v\n", err)
		return exitRejected
	}

	// Sucesso ou rejeição formal.
	if result.Accepted {
		fmt.Printf("protocol_sta=%s  status=accepted\n", result.ProtocolSTA)
		return exitOK
	}

	// Rejeitado — result.Rejection != nil.
	if result.Rejection != nil {
		fmt.Printf("protocol_sta=%s  status=rejected  code=%s  message=%s\n",
			result.ProtocolSTA, result.Rejection.Code, result.Rejection.Message)
	} else {
		fmt.Printf("protocol_sta=%s  status=rejected  (sem motivo)\n", result.ProtocolSTA)
	}
	return exitRejected
}

// usage imprime help.
func usage() {
	fmt.Fprintf(os.Stderr, `Usage: sta-submit [flags]

Flags:
  --xml-file    Caminho do arquivo XML do CADOC (env: STA_SUBMIT_XML_FILE)
  --cadoc-code  Código do CADOC (env: STA_SUBMIT_CADOC_CODE, default 3040)
  --data-base   Data-base YYYY-MM (env: STA_SUBMIT_DATA_BASE)
  --cnpj        CNPJ do IF (env: STA_SUBMIT_CNPJ, default demo-bank)
  --quiet       Silencia logs de stderr

Variáveis de ambiente STA (RADIANT_STA_*):
  RADIANT_STA_BACKEND         stub (default) | ws (BACEN real)
  RADIANT_STA_WS_URL          https://sta-h.bcb.gov.br/staws
  RADIANT_STA_SISBACEN_USER   formato UUUUUDDDD.operador
  RADIANT_STA_SISBACEN_PASSWORD senha Sisbacen
  RADIANT_STA_TIMEOUT_SECONDS default 30s

Exit codes:
  0  aceito pelo BACEN
  1  rejeitado pelo BACEN / erro de transporte
  2  erro de validação client-side
  3  erro BACEN formal
`)
}

func main() {
	cfg, err := loadConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flag parse: %v\n", err)
		usage()
		os.Exit(exitClientError)
	}

	logger := newLogger(cfg.quiet)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	os.Exit(runSubmit(ctx, cfg, logger))
}
