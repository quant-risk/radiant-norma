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
		// Coleta todos os erros como críticas
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

	// Regras cross-field (disponíveis mesmo sem XSD oficial)
	criticas = append(criticas, crossFieldRules(doc)...)

	return &Result{
		Valid:    len(criticas) == 0,
		Criticas: criticas,
		Doc:      doc,
	}, nil
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

	// Regra: IPOC de operação deve existir em algum registro (cross-cliente)
	seenIPOCs := make(map[string]bool)
	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC != "" {
				seenIPOCs[op.IPOC] = true
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
