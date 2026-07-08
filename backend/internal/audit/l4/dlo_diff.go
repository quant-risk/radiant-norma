// Diff para CADOC 2061 (DLO) e 2062 (DLI).
//
// Usa os parsers existentes (rules.ParseDocDLO, ParseDocDLI) para extrair
// campos agregados e detectar variação.
package l4

import (
	"fmt"
	"math"

	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
)

// extractDLOChanges extrai e compara campos agregados do DLO (2061).
func (e *Engine) extractDLOChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDocDLO([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLO prev: %w", err)
	}
	currDoc, err := rules.ParseDocDLO([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLO curr: %w", err)
	}

	var changes []FieldChange

	// 800.01 — Capital Principal
	changes = appendFieldChange(changes, "2061", "800.01",
		prevDoc.Accounts["800.01"], currDoc.Accounts["800.01"])

	// 800.02 — Capital Complementar
	changes = appendFieldChange(changes, "2061", "800.02",
		prevDoc.Accounts["800.02"], currDoc.Accounts["800.02"])

	// 111.01 — Depósitos
	changes = appendFieldChange(changes, "2061", "111.01",
		prevDoc.Accounts["111.01"], currDoc.Accounts["111.01"])

	// 111.02 — Aplicações Financeiras
	changes = appendFieldChange(changes, "2061", "111.02",
		prevDoc.Accounts["111.02"], currDoc.Accounts["111.02"])

	// Patrimonio (campo direto)
	changes = appendFieldChange(changes, "2061", "Patrimonio",
		prevDoc.Patrimonio, currDoc.Patrimonio)

	// LimiteTotal
	changes = appendFieldChange(changes, "2061", "LimiteTotal",
		prevDoc.LimiteTotal, currDoc.LimiteTotal)

	// Conta770 (totais de conta 770)
	changes = appendFieldChange(changes, "2061", "Conta770",
		prevDoc.Conta770, currDoc.Conta770)

	return changes, nil
}

// extractDLIChanges extrai e compara campos agregados do DLI (2062).
func (e *Engine) extractDLIChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDocDLI([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLI prev: %w", err)
	}
	currDoc, err := rules.ParseDocDLI([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLI curr: %w", err)
	}

	var changes []FieldChange

	// PLA (6.10.01 + 6.10.02 - 6.10.90)
	prevPLA := rules.SomaPLA(prevDoc.Accounts)
	currPLA := rules.SomaPLA(currDoc.Accounts)
	changes = appendFieldChange(changes, "2062", "PLA", prevPLA, currPLA)

	// Margem PLA (6.00.00)
	changes = appendFieldChange(changes, "2062", "6.00.00",
		prevDoc.Accounts["6.00.00"], currDoc.Accounts["6.00.00"])

	// Capital Realizado (8.10.01)
	prevCap := rules.SomaCapitalRealizado(prevDoc.Accounts)
	currCap := rules.SomaCapitalRealizado(currDoc.Accounts)
	changes = appendFieldChange(changes, "2062", "8.10.01", prevCap, currCap)

	// Requerimento PLA (6.90.00)
	changes = appendFieldChange(changes, "2062", "6.90.00",
		prevDoc.Accounts["6.90.00"], currDoc.Accounts["6.90.00"])

	// Limites específicos (seexistirem)
	// 20.00 — Partes Relacionadas
	changes = appendFieldChange(changes, "2062", "20.00",
		prevDoc.Accounts["20.00.00"], currDoc.Accounts["20.00.00"])

	// 34.00 — Empréstimo TVM
	changes = appendFieldChange(changes, "2062", "34.00",
		prevDoc.Accounts["34.00.00"], currDoc.Accounts["34.00.00"])

	return changes, nil
}

// extractDRLChanges extrai e compara campos do DRL (2160 — LCR).
func (e *Engine) extractDRLChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDocDRL([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRL prev: %w", err)
	}
	currDoc, err := rules.ParseDocDRL([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DRL curr: %w", err)
	}

	var changes []FieldChange

	// HQLA (High Quality Liquid Assets)
	changes = appendFieldChange(changes, "2160", "HQLA",
		prevDoc.HQLA, currDoc.HQLA)

	// Outflows
	changes = appendFieldChange(changes, "2160", "Outflows",
		prevDoc.Outflows, currDoc.Outflows)

	// Inflows
	changes = appendFieldChange(changes, "2160", "Inflows",
		prevDoc.Inflows, currDoc.Inflows)

	// LCR Ratio
	changes = appendFieldChange(changes, "2160", "LCRRatio",
		prevDoc.LCRRatio, currDoc.LCRRatio)

	return changes, nil
}

// extractDLPChanges extrai e compara campos do DLP (2170 — NSFR).
func (e *Engine) extractDLPChanges(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDocDLP([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLP prev: %w", err)
	}
	currDoc, err := rules.ParseDocDLP([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse DLP curr: %w", err)
	}

	var changes []FieldChange

	// ASF (Available Stable Funding)
	changes = appendFieldChange(changes, "2170", "ASFTotal",
		prevDoc.ASFTotal, currDoc.ASFTotal)

	// RSF (Required Stable Funding)
	changes = appendFieldChange(changes, "2170", "RSFTotal",
		prevDoc.RSFTotal, currDoc.RSFTotal)

	// NSFR Ratio
	changes = appendFieldChange(changes, "2170", "NSFRRatio",
		prevDoc.NSFRRatio, currDoc.NSFRRatio)

	return changes, nil
}

// extract3040Changes extrai e compara campos do 3040.
func (e *Engine) extract3040Changes(prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	prevDoc, err := rules.ParseDoc3040([]byte(prev.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3040 prev: %w", err)
	}
	currDoc, err := rules.ParseDoc3040([]byte(curr.XMLContent))
	if err != nil {
		return nil, fmt.Errorf("parse 3040 curr: %w", err)
	}

	var changes []FieldChange

	// Conta770-like: soma dos agregados por modalidade
	// RWACAM agregado (se existir)
	// Por enquanto, compara contagem de operações
	changes = appendFieldChange(changes, "3040", "Agregados",
		float64(len(prevDoc.Agregados)), float64(len(currDoc.Agregados)))
	changes = appendFieldChange(changes, "3040", "Operacoes",
		float64(len(prevDoc.Operacoes)), float64(len(currDoc.Operacoes)))

	return changes, nil
}

// appendFieldChange adiciona uma mudança se for significativa (> 0.01% variação ou cruzou zero).
func appendFieldChange(changes []FieldChange, cadoc, field string, prev, curr float64) []FieldChange {
	// Ignora se ambos zero
	if prev == 0 && curr == 0 {
		return changes
	}
	// Ignora se são exatamente iguais
	if prev == curr {
		return changes
	}
	dp := deltaPercent(prev, curr)
	// Só adiciona se variação > 0.01% ou cruzou zero
	if math.Abs(dp) > 0.01 || (prev == 0 || curr == 0) {
		changes = append(changes, FieldChange{
			CadocCode: cadoc,
			Field:     field,
			Previous:  prev,
			Current:   curr,
			DeltaPct:  dp,
		})
	}
	return changes
}
