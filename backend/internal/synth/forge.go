package synth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
)

// Forge é o orchestrator principal do AuditForge.
// Implementa o loop Agentic Self-Instruct do paper Autodata:
//
//	Challenger → Weak Solver → Strong Solver → Judge → (feedback) → Challenger
//
// A key metric é o "gap": strong_score - weak_score.
// Um bom caso para training tem: gap > 0.2, realism >= medium, difficulty >= easy.
//
// Paper: "Autodata: An agentic data scientist to create high quality synthetic data"
// (FAIR Meta, Jun 2026) — https://arxiv.org/abs/2606.25996
type Forge struct {
	cfg          Config
	generator    *Generator
	weakSolver   *WeakSolver
	strongSolver *StrongSolver
	judge        *Judge
	logger       *slog.Logger
}

// Config configura o Forge.
type Config struct {
	Cadoc       CadocType
	LLM         LLMClient
	Count       int
	FailureRate float64
	MaxRounds   int // iterações do loop por caso (default 5)
}

// NewForge cria um novo Forge.
func NewForge(cfg Config, auditSvc *audit.Service, logger *slog.Logger) *Forge {
	if cfg.MaxRounds == 0 {
		cfg.MaxRounds = 5
	}
	return &Forge{
		cfg:          cfg,
		generator:    NewGenerator(GeneratorConfig(cfg)),
		weakSolver:   NewWeakSolver(auditSvc),
		strongSolver: NewStrongSolver(),
		judge:        NewJudge(cfg.LLM),
		logger:       logger,
	}
}

// Run executa o loop Agentic Self-Instruct completo.
func (f *Forge) Run(ctx context.Context) ([]Case, error) {
	f.logger.Info("AuditForge starting",
		"cadoc", f.cfg.Cadoc,
		"count", f.cfg.Count,
		"max_rounds", f.cfg.MaxRounds,
		"failure_rate", f.cfg.FailureRate,
		"model", f.cfg.LLM.Model())

	var cases []Case
	var totalRounds int

	for i := 0; i < f.cfg.Count; i++ {
		f.logger.Info("generating case", "index", i+1, "total", f.cfg.Count)

		c, rounds, err := f.generateWithLoop(ctx, i)
		totalRounds += rounds

		if err != nil {
			f.logger.Warn("case generation failed", "id", c.ID, "err", err)
		}

		cases = append(cases, *c)
	}

	accepted := 0
	for _, c := range cases {
		if c.Realism != "" { // foi julgado
			accepted++
		}
	}

	f.logger.Info("AuditForge complete",
		"total", len(cases),
		"accepted", accepted,
		"rejected", len(cases)-accepted,
		"total_rounds", totalRounds,
		"avg_rounds", float64(totalRounds)/float64(len(cases)))

	return cases, nil
}

// generateWithLoop implementa o Agentic Self-Instruct loop para um caso:
// 1. Challenger gera XML
// 2. Weak+Strong solvers avaliam
// 3. Judge decide (accept/reject + feedback)
// 4. Se reject: feedback → Challenger (próxima round)
// Repete até MaxRounds ou accept.
func (f *Forge) generateWithLoop(ctx context.Context, idx int) (*Case, int, error) {
	id := fmt.Sprintf("synth-%s-%03d", f.cfg.Cadoc, idx+1)
	feedback := ""

	for round := 1; round <= f.cfg.MaxRounds; round++ {
		// 1. Challenger gera XML
		prompt := challengePrompt(f.cfg.Cadoc, feedback)
		resp, err := f.cfg.LLM.Chat(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			return nil, round - 1, fmt.Errorf("challenger round %d: %w", round, err)
		}

		xml := extractXML(resp)
		if xml == "" || len(xml) < 20 {
			feedback = "XML vazio ou inválido. Gere um XML completo com todos os campos obrigatórios."
			continue
		}

		c := &Case{
			ID:          id,
			Cadoc:       string(f.cfg.Cadoc),
			XML:         xml,
			GeneratedBy: f.cfg.LLM.Model(),
			Rounds:      round,
		}

		// 2. Weak solver avalia
		weakScore, err := f.weakSolver.Solve(ctx, c)
		if err != nil {
			f.logger.Warn("weak solve failed", "id", id, "round", round, "err", err)
		}
		c.WeakScore = weakScore

		// 3. Strong solver avalia
		strongScore, err := f.strongSolver.Solve(ctx, c)
		if err != nil {
			f.logger.Warn("strong solve failed", "id", id, "round", round, "err", err)
		}
		c.StrongScore = strongScore

		// 4. Judge avalia
		result, err := f.judge.JudgeCase(ctx, c)
		if err != nil {
			f.logger.Warn("judge failed", "id", id, "round", round, "err", err)
			feedback = "Erro no judge. Tente um caso diferente com mais variação nos valores."
			continue
		}

		c.Realism = result.Realism
		c.Difficulty = result.Difficulty
		c.JudgeFeedback = result.Feedback

		if result.Accept {
			// Caso aceito — fim do loop
			f.logger.Info("case accepted", "id", id, "round", round, "gap", result.Gap)
			return c, round, nil
		}

		// Rejeitado — feedback para próxima round
		feedback = result.Feedback
		f.logger.Info("case rejected", "id", id, "round", round, "feedback", feedback)
	}

	// Esgotou rounds — retorna último caso (pode não ser ideal)
	return &Case{
		ID:          id,
		Cadoc:       string(f.cfg.Cadoc),
		XML:         "",
		GeneratedBy: f.cfg.LLM.Model(),
		Rounds:      f.cfg.MaxRounds,
	}, f.cfg.MaxRounds, nil
}

// ExportJSON exporta os casos para JSON.
func ExportJSON(cases []Case) ([]byte, error) {
	return json.MarshalIndent(cases, "", "  ")
}

// ExportFailures exporta apenas os casos que falham regras.
func ExportFailures(cases []Case) []Case {
	var out []Case
	for _, c := range cases {
		if c.IsFailure {
			out = append(out, c)
		}
	}
	return out
}
