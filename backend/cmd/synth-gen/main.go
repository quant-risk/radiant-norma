// cmd/synth-gen: CLI do AuditForge — gerador de envios CADOC sintéticos.
//
// Implementa o loop Agentic Self-Instruct do paper Autodata:
// Challenger → Weak Solver → Strong Solver → Judge → (feedback) → Challenger
//
// Paper: "Autodata: An agentic data scientist to create high quality synthetic data"
// (FAIR Meta, Jun 2026) — https://arxiv.org/abs/2606.25996
//
// Uso:
//
//	synth-gen --cadoc 3040 --count 100
//
// Variáveis de ambiente:
//
//	LLM_API_KEY      — API key do LLM (MiniMax ou OpenAI)
//	LLM_PROVIDER     — "minimax" (default) ou "openai"
//	LLM_MODEL        — modelo a usar (default: MiniMax-Text-01 ou gpt-4o-mini)
//
// Output:
//
//	--output-dir ./synth-out/  (default: ./synth-out/)
//
// Formato de output:
//
//	<synth-out>/
//	  all_cases.json     — todos os casos gerados (com gap, rounds, judge feedback)
//	  accepted.json     — apenas casos aceitos pelo judge
//	  rejected.json     — casos rejeitados após MaxRounds
//	  stats.json        — estatísticas do run (gap avg, rounds avg, etc)
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
	maxRounds := flag.Int("max-rounds", 5, "máximo de iterações do loop por caso")
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

	// Forge com loop Agentic Self-Instruct
	forge := synth.NewForge(synth.Config{
		Cadoc:     synth.CadocType(*cadoc),
		LLM:       llm,
		Count:     *count,
		MaxRounds: *maxRounds,
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

	// accepted.json — casos onde Judge aceitou
	var accepted, rejected []synth.Case
	for _, c := range cases {
		if c.Realism != "" {
			accepted = append(accepted, c)
		} else {
			rejected = append(rejected, c)
		}
	}

	accBytes, _ := json.MarshalIndent(accepted, "", "  ")
	if err := os.WriteFile(filepath.Join(*outputDir, "accepted.json"), accBytes, 0644); err != nil {
		logger.Error("write accepted.json", "err", err)
		os.Exit(1)
	}

	rejBytes, _ := json.MarshalIndent(rejected, "", "  ")
	if err := os.WriteFile(filepath.Join(*outputDir, "rejected.json"), rejBytes, 0644); err != nil {
		logger.Error("write rejected.json", "err", err)
		os.Exit(1)
	}

	// stats.json
	stats := buildStats(cases, accepted)
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
		"accepted", len(accepted),
		"rejected", len(rejected))
}

func buildStats(all, accepted []synth.Case) map[string]any {
	byRule := map[string]int{}
	byRealism := map[string]int{}
	byDifficulty := map[string]int{}
	var totalGap, totalRounds float64
	judged := 0

	for _, c := range accepted {
		if c.RuleCode != "" {
			byRule[c.RuleCode]++
		}
		if c.Realism != "" {
			byRealism[c.Realism]++
		}
		if c.Difficulty != "" {
			byDifficulty[c.Difficulty]++
		}
		totalGap += c.StrongScore - c.WeakScore
		totalRounds += float64(c.Rounds)
		judged++
	}

	gapAvg := 0.0
	roundsAvg := 0.0
	if judged > 0 {
		gapAvg = totalGap / float64(judged)
		roundsAvg = totalRounds / float64(judged)
	}

	return map[string]any{
		"total":           len(all),
		"accepted":        len(accepted),
		"rejected":        len(all) - len(accepted),
		"acceptance_rate": float64(len(accepted)) / float64(len(all)),
		"gap_avg":         gapAvg,
		"rounds_avg":      roundsAvg,
		"by_rule":         byRule,
		"by_realism":      byRealism,
		"by_difficulty":   byDifficulty,
	}
}
