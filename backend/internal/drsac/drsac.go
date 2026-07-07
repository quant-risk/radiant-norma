// DRSAC service — integra com o motor de validação do Radiant Norma.
//
//nolint:revive
package drsac

import (
	"context"
	"fmt"
)

// Critica representa uma crítica encontrada na validação.
type Critica struct {
	Code     string // Código da crítica (ex: D01, D02...)
	Severity string // HIGH, MEDIUM, LOW
	Message  string // Descrição da crítica
	Path     string // XPath-like ao campo
}

// Result é o resultado completo da validação DRSAC.
type Result struct {
	Valid    bool
	Criticas []Critica
	Doc      *DocumentoDRSAC
}

// Rule é a interface implementada por todas as regras DRSAC.
type Rule interface {
	Code() string
	Severity() string
	Message() string
	Apply(context.Context, *DocumentoDRSAC) error
}

// ValidateDocument é o entry point principal para validar um documento DRSAC.
// Parsing + validação de domínio + validação de regras de negócio.
func ValidateDocument(ctx context.Context, data []byte) (*Result, error) {
	doc, err := ParseFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse DRSAC: %w", err)
	}

	var criticas []Critica

	// Validação de domínio (anexos 01-20)
	if err := Validate(doc); err != nil {
		if ve, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range ve.Unwrap() {
				if ve, ok := e.(*ValidationError); ok {
					criticas = append(criticas, Critica{
						Code:     "D01",
						Severity: "HIGH",
						Message:  ve.Error(),
						Path:     ve.Path,
					})
				}
			}
		} else {
			criticas = append(criticas, Critica{
				Code:     "D01",
				Severity: "HIGH",
				Message:  err.Error(),
			})
		}
	}

	// Regras de catálogo (D01-D35)
	criticas = append(criticas, ValidateRules(ctx, doc)...)

	// Regras cross-field
	criticas = append(criticas, crossFieldRules(doc)...)

	return &Result{
		Valid:    len(criticas) == 0,
		Criticas: criticas,
		Doc:      doc,
	}, nil
}

// ValidateRules executa todas as regras D01-D35 contra o documento.
// Retorna lista de Critica.
func ValidateRules(ctx context.Context, doc *DocumentoDRSAC) []Critica {
	rules := []Rule{
		// Estrutura (D01-D10)
		D01{}, D02{}, D03{}, D04{}, D05{}, D06{}, D07{}, D08{}, D09{}, D10{},
		// Riscos (D11-D16)
		D11{}, D12{}, D13{}, D14{}, D15{}, D16{},
		// Consistência 98/99 (D17-D18)
		D17{}, D18{},
		// GEE (D19-D20)
		D19{}, D20{},
		// Localização (D21-D25)
		D21{}, D22{}, D23{}, D24{}, D25{},
		// TVM (D26-D28)
		D26{}, D27{}, D28{},
		// Setores (D29-D31)
		D29{}, D30{}, D31{},
		// AgrMit / ContribPositiva (D32-D35)
		D32{}, D33{}, D34{}, D35{},
	}

	var criticas []Critica
	for _, rule := range rules {
		if err := rule.Apply(ctx, doc); err != nil {
			criticas = append(criticas, Critica{
				Code:     rule.Code(),
				Severity: rule.Severity(),
				Message:  fmt.Sprintf("%s: %v", rule.Message(), err),
			})
		}
	}
	return criticas
}

// crossFieldRules valida regras que dependem de múltiplos campos.
//
//nolint:revive
func crossFieldRules(doc *DocumentoDRSAC) []Critica {
	var criticas []Critica

	for i, cl := range doc.Clientes {
		path := fmt.Sprintf("/DocumentoDRSAC/Clientes/Cliente[%d]", i+1)

		for j, op := range cl.ExpAtivos.ExpOperCred {
			opPath := fmt.Sprintf("%s/ExpAtivos/ExpOperCred[%d]", path, j+1)

			// Regra: GEE valores só requeridos quando situação = 01 ou 02
			if op.HistAbsorEmissGEE != nil {
				if op.HistAbsorEmissGEE.Valor != "" &&
					op.HistAbsorEmissGEE.Sit != GEESitAbsorcao &&
					op.HistAbsorEmissGEE.Sit != GEESitEmissao {
					criticas = append(criticas, Critica{
						Code:     "DR01",
						Severity: "MEDIUM",
						Message:  "HistAbsorEmissGEE.valor só é requerido quando situacao é 01 (Absorção) ou 02 (Emissão)",
						Path:     opPath + "/HistAbsorEmissGEE",
					})
				}
			}

			// Regra: Mitigador existe → risk physical deve ter av diferente de 98/99
			if op.MitRiscClimFis != nil && op.MitRiscClimFis.Exist == MitigadorExiste {
				if op.RiscClimFis.Av == AvNaoAvaliado || op.RiscClimFis.Av == AvForaEscopo {
					criticas = append(criticas, Critica{
						Code:     "DR02",
						Severity: "HIGH",
						Message:  "Mitigador existe (01) mas RiscClimFis.av é 98 ou 99 — inconsistente",
						Path:     opPath + "/MitRiscClimFis",
					})
				}
			}

			// Regra: Contribuição positiva → saldo > 0
			if op.ContribPositiva != nil && op.Saldo == "0.00" {
				criticas = append(criticas, Critica{
					Code:     "DR03",
					Severity: "LOW",
					Message:  "ContribPositiva preenchido mas saldo da operação é zero",
					Path:     opPath + "/ContribPositiva",
				})
			}
		}
	}

	return criticas
}

// IntegrateWithAudit registra o tipo de documento no audit trail.
// (placeholder — implementado quando DRSAC for integrado ao engine principal)
func IntegrateWithAudit() {
	// TODO: quando DRSAC for integrado ao engine de validação principal,
	// registrar aqui que o audit trail sabe que "2030" = "DocumentoDRSAC"
}
