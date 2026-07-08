// cmd/synth-gen: CLI do AuditForge — gerador de envios CADOC sintéticos.
//
// Uso:
//
//	synth-gen --cadoc 3040 --count 100 --failure-rate 0.3
//
// Variáveis de ambiente:
//
//	LLM_API_KEY      — API key do LLM (MiniMax ou OpenAI)
//	LLM_PROVIDER     — "minimax" (default) ou "openai"
//	LLM_MODEL        — modelo a usar (default: MiniMax-Text-01 ou gpt-4o-mini)
//
// Output:
//
//	--output-dir ./synth-output/  (default: ./synth-out/)
//
// Formato de output:
//
//	<synth-out>/
//	  all_cases.json     — todos os casos gerados
//	  failures.json      — apenas casos que violam regras
//	  stats.json         — estatísticas do run
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/insights"
	"github.com/fortvna/radiant-norma/backend/internal/synth"
)

func main() {
	cadoc := flag.String("cadoc", "3040", "tipo de CADOC (3040, 3050, 4111)")
	count := flag.Int("count", 50, "número de casos a gerar")
	failureRate := flag.Float64("failure-rate", 0.3, "fração de casos que devem falhar (0.0-1.0)")
	outputDir := flag.String("output-dir", "./synth-out", "diretório de output")
	dbPath := flag.String("db", "radiant.db", "path para o banco SQLite")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx := context.Background()

	// Conecta ao DB
	d, err := db.Open(*dbPath)
	if err != nil {
		logger.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	// Garante schema
	if err := db.Migrate(d); err != nil {
		logger.Error("db migrate", "err", err)
		os.Exit(1)
	}

	// Services
	auditSvc := audit.New(d)

	// LLM client — required
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		fmt.Println("ERRO: LLM_API_KEY não setada. Usage: LLM_API_KEY=... synth-gen --cadoc 3040 --count 50")
		os.Exit(1)
	}
	provider := os.Getenv("LLM_PROVIDER")
	model := os.Getenv("LLM_MODEL")

	var llm insights.LLMClient
	switch provider {
	case "openai":
		m := model
		if m == "" {
			m = "gpt-4o-mini"
		}
		baseURL := os.Getenv("LLM_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		llm = insights.NewOpenAIChat(apiKey, m, baseURL)
	default: // minimax
		cfg := insights.MiniMaxConfig{APIKey: apiKey}
		if model != "" {
			cfg.Model = model
		}
		if baseURL := os.Getenv("LLM_BASE_URL"); baseURL != "" {
			cfg.BaseURL = baseURL
		}
		llm = insights.NewMiniMaxChat(cfg)
	}

	// Forge
	forge := synth.NewForge(synth.Config{
		Cadoc:       synth.CadocType(*cadoc),
		LLM:         llm,
		Count:       *count,
		FailureRate: *failureRate,
	}, auditSvc, logger)

	cases, err := forge.Run(ctx)
	if err != nil {
		logger.Error("forge run", "err", err)
		os.Exit(1)
	}

	// Output
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Error("mkdir output", "err", err)
		os.Exit(1)
	}

	// all_cases.json
	allBytes, err := synth.ExportJSON(cases)
	if err != nil {
		logger.Error("export all_cases", "err", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "all_cases.json"), allBytes, 0644); err != nil {
		logger.Error("write all_cases.json", "err", err)
		os.Exit(1)
	}

	// failures.json
	failures := synth.ExportFailures(cases)
	failBytes, err := json.MarshalIndent(failures, "", "  ")
	if err != nil {
		logger.Error("marshal failures", "err", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "failures.json"), failBytes, 0644); err != nil {
		logger.Error("write failures.json", "err", err)
		os.Exit(1)
	}

	// stats.json
	stats := buildStats(cases, failures)
	statsBytes, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		logger.Error("marshal stats", "err", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, "stats.json"), statsBytes, 0644); err != nil {
		logger.Error("write stats.json", "err", err)
		os.Exit(1)
	}

	logger.Info("done",
		"output", *outputDir,
		"total", len(cases),
		"failures", len(failures))
}

func buildStats(all, failures []synth.Case) map[string]any {
	byRule := map[string]int{}
	byRealism := map[string]int{}
	byDifficulty := map[string]int{}

	for _, c := range failures {
		if c.RuleCode != "" {
			byRule[c.RuleCode]++
		}
		if c.Realism != "" {
			byRealism[c.Realism]++
		}
		if c.Difficulty != "" {
			byDifficulty[c.Difficulty]++
		}
	}

	return map[string]any{
		"total":         len(all),
		"failures":      len(failures),
		"passes":        len(all) - len(failures),
		"by_rule":       byRule,
		"by_realism":    byRealism,
		"by_difficulty": byDifficulty,
	}
}
