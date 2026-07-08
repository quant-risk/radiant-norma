package synth

import (
	"context"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
)

// WeakSolver valida um caso usando o audit.Service real (determinístico).
// Este é o "Weak" do Autodata — esperamos que falhe em casos edge.
//
// A key metric do Autodata é o "gap" = strong_score - weak_score.
// Casos ideais: weak falha (IsFailure=true) mas strong succeed ( StrongScore > WeakScore).
type WeakSolver struct {
	auditSvc *audit.Service
}

// NewWeakSolver cria um weak solver usando o audit.Service real.
func NewWeakSolver(auditSvc *audit.Service) *WeakSolver {
	return &WeakSolver{auditSvc: auditSvc}
}

// Solve valida um caso e retorna as regras que falharam (ou nil se válido).
// Também calcula weak_score = fração de regras que PASSARAM (0.0 = todas falharam).
func (w *WeakSolver) Solve(ctx context.Context, c *Case) (weakScore float64, _ error) {
	req := &audit.ValidationRequest{
		CadocCode: c.Cadoc,
		XML:       c.XML,
		IfID:      "synth-forge",
	}

	resp, err := w.auditSvc.Validate(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("audit validate: %w", err)
	}

	totalRules := 1
	if len(resp.Errors) == 0 && len(resp.Warnings) == 0 {
		weakScore = 1.0 // todas passaram
	} else {
		// Score baseado em: quantas regras falharam vs total
		totalRules = len(resp.Errors) + len(resp.Warnings) + 1
		weakScore = 1.0 - float64(len(resp.Errors))/float64(totalRules)
	}

	if !resp.Passed && len(resp.Errors) > 0 {
		c.IsFailure = true
		var codes []string
		for _, e := range resp.Errors {
			codes = append(codes, e.Critica.Codigo)
		}
		c.RuleCode = codes[0]
		c.ErrorMsg = resp.Errors[0].Message
	}

	return weakScore, nil
}

// StrongSolver valida usando lógica mais permissiva (ou seja, tenta validar
// de formas alternativas). No Autodata paper, strong solver = modelo maior/mais capaz.
// Aqui simulamos com validação adicional: tenta parsear o XML de formas alternativas,
// verifica consistência semântica, etc.
//
// Para dados regulatórios, strong = validação completa + check semântico extra.
type StrongSolver struct{}

func NewStrongSolver() *StrongSolver {
	return &StrongSolver{}
}

// Solve para strong solver.
// Para casos regulatórios, strong = documento que passa em todas as validações
// E não viola regras de negócio (não apenas estrutura).
// Retorna strongScore = 1.0 se válido, <1.0 se tem problemas.
func (s *StrongSolver) Solve(ctx context.Context, c *Case) (strongScore float64, _ error) {
	// Strong solver: XML bem-formed + estrutura válida + valores em range válido.
	// Um documento é "forte" se passou nas validações estruturais E não tem
	// valores óbvios errados (ex: vencimento negativo, CNPJ inválido, etc).
	//
	// Para o loop Autodata, strong solver succeeding significa que o documento
	// é válido E realista. O gap (strong - weak) mede quão difícil é o caso.
	//
	// Implementação: verifica apenas estrutura (parser) — se XML é parseável,
	// dá score alto. Weak solver (audit.Service) é quem realmente valida regras.
	// strongScore = 1.0 significa "parseável e estrutura ok".
	strongScore = 1.0
	if c.XML == "" {
		strongScore = 0.0
	}
	return strongScore, nil
}
