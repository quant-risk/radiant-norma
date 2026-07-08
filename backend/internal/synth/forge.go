package synth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
)

// Forge é o orchestrator principal do AuditForge.
// Executa o loop iterativo: Challenger → Weak → Judge → dataset.
type Forge struct {
	cfg        Config
	generator  *Generator
	weakSolver *WeakSolver
	judge      *Judge
	logger     *slog.Logger
}

// Config configura o Forge e o Generator.
type Config struct {
	Cadoc       CadocType
	LLM         LLMClient
	Count       int
	FailureRate float64 // fração de casos que devem falhar (0.0-1.0)
}

// NewForge cria um novo Forge.
func NewForge(cfg Config, auditSvc *audit.Service, logger *slog.Logger) *Forge {
	return &Forge{
		cfg:        cfg,
		generator:  NewGenerator(GeneratorConfig(cfg)),
		weakSolver: NewWeakSolver(auditSvc),
		judge:      NewJudge(cfg.LLM),
		logger:     logger,
	}
}

// Run executa o loop e retorna os casos gerados.
func (f *Forge) Run(ctx context.Context) ([]Case, error) {
	f.logger.Info("AuditForge starting",
		"cadoc", f.cfg.Cadoc,
		"count", f.cfg.Count,
		"failure_rate", f.cfg.FailureRate,
		"model", f.cfg.LLM.Model())

	cases, err := f.generator.Generate(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	// Weak solve: valida cada caso com o audit.Service real
	var failures, passes int
	for i := range cases {
		_, err := f.weakSolver.Solve(ctx, &cases[i])
		if err != nil {
			f.logger.Warn("weak solve failed", "id", cases[i].ID, "err", err)
		}
		if cases[i].IsFailure {
			failures++
		} else {
			passes++
		}

		// Judge: avalia realismo e dificuldade (apenas para casos de falha)
		if cases[i].IsFailure {
			if err := f.judge.JudgeCase(ctx, &cases[i]); err != nil {
				f.logger.Warn("judge failed", "id", cases[i].ID, "err", err)
			}
		}
	}

	f.logger.Info("AuditForge complete",
		"total", len(cases),
		"failures", failures,
		"passes", passes,
		"judged", failures)

	return cases, nil
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
