package synth

import (
	"context"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
)

// WeakSolver valida um caso usando o audit.Service real.
type WeakSolver struct {
	auditSvc *audit.Service
}

// NewWeakSolver cria um weak solver usando o audit.Service real.
func NewWeakSolver(auditSvc *audit.Service) *WeakSolver {
	return &WeakSolver{auditSvc: auditSvc}
}

// Solve valida um caso e retorna as regras que falharam (ou nil se válido).
func (w *WeakSolver) Solve(ctx context.Context, c *Case) ([]string, error) {
	req := &audit.ValidationRequest{
		CadocCode: c.Cadoc,
		XML:       c.XML,
		IfID:      "synth-forge",
	}

	resp, err := w.auditSvc.Validate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("audit validate: %w", err)
	}

	if !resp.Passed && len(resp.Errors) > 0 {
		c.IsFailure = true
		var codes []string
		for _, e := range resp.Errors {
			codes = append(codes, e.Critica.Codigo)
		}
		c.RuleCode = codes[0]
		c.ErrorMsg = resp.Errors[0].Message
		return codes, nil
	}

	c.IsFailure = false
	return nil, nil
}
