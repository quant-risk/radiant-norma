// Regras cross-document iniciais 3040 ↔ 4111 ↔ DRSAC (Sprint 6 v1.5.0).
//
// Cada regra:
//   - Code: XD-NNN (namespace próprio)
//   - Description: human-readable
//   - Severity: E (erro) / A (aviso) / I (informativo)
//   - RequiredDocs: CADOCs que devem estar presentes
//   - Apply: lógica de validação cruzando os docs
package rules

import (
	"context"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
)

// XD-001 — Total de operações no 3040 deve bater com total de clientes no 4111.
//
// Justificativa: mesmo IF, mesma data-base — totais devem ser próximos.
// Tolerância de 5% para cobrir casos onde cliente tem múltiplas operações
// (1 cliente → N operações no 3040 vs 1 cliente no 4111).
type TotalOperacoes3040Consistente4111 struct{}

func (TotalOperacoes3040Consistente4111) Code() string {
	return "XD-001"
}
func (TotalOperacoes3040Consistente4111) Description() string {
	return "Total de operações no 3040 deve ser próximo (±5%) do total de clientes no 4111"
}
func (TotalOperacoes3040Consistente4111) Severity() string { return "A" }
func (TotalOperacoes3040Consistente4111) RequiredDocs() []string {
	return []string{"3040", "4111"}
}
func (TotalOperacoes3040Consistente4111) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml3040 := docs.Get("3040")
	xml4111 := docs.Get("4111")
	ops := crossdoc.ExtractSumOfTag(xml3040, "Agreg", "QtdOp")
	clients := crossdoc.ExtractSumOfTag(xml4111, "Cliente", "QtdCli")
	if ops == 0 {
		return crossdoc.NewError("XD-001", "A", "3040 sem operações detectadas")
	}
	if clients == 0 {
		return crossdoc.NewError("XD-001", "A", "4111 sem clientes detectados")
	}
	diff := ops - clients
	if diff < 0 {
		diff = -diff
	}
	ratio := diff / clients
	if ratio > 0.05 {
		return crossdoc.NewError("XD-001", "A",
			fmt.Sprintf("discrepância %.1f%% entre ops 3040 (%.0f) e clients 4111 (%.0f) — tol. 5%%",
				ratio*100, ops, clients))
	}
	return nil
}

// XD-002 — Modalidade 0213 (cheque especial) no 3040 deve ter flag correspondente no 4111.
//
// Justificativa: 4111 lista modalidades de crédito contratadas. Se 3040
// reporta modalidade 0213 (cheque especial) com v150 > 0 (vencido > 90d),
// 4111 deve ter esse contrato flagged como inadimplente.
type Modalidade0213FlagChequeEspecial struct{}

func (Modalidade0213FlagChequeEspecial) Code() string {
	return "XD-002"
}
func (Modalidade0213FlagChequeEspecial) Description() string {
	return "Modalidade 0213 (cheque especial) no 3040 deve estar flagged no 4111"
}
func (Modalidade0213FlagChequeEspecial) Severity() string { return "A" }
func (Modalidade0213FlagChequeEspecial) RequiredDocs() []string {
	return []string{"3040", "4111"}
}
func (Modalidade0213FlagChequeEspecial) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml3040 := docs.Get("3040")

	// Conta quantos <Agreg> têm Mod="0213"
	count0213 := 0
	agregCount := 0
	currentMod := ""
	for line := range iterateXMLElements(xml3040, "Agreg") {
		agregCount++
		if line.Mod != "" {
			currentMod = line.Mod
		}
		if currentMod == "0213" {
			count0213++
		}
	}

	// Se 3040 não tem cheque especial, regra é N/A (skip)
	if count0213 == 0 {
		return nil
	}

	// Verifica 4111 tem flag CE (cheque especial)
	xml4111 := docs.Get("4111")
	if crossdoc.ExtractTextBetween(xml4111, "FlagChequeEspecial") == "" &&
		crossdoc.CountTag(xml4111, "Modalidade0213") == 0 {
		return crossdoc.NewError("XD-002", "A",
			fmt.Sprintf("3040 reporta %d ocorrências de Mod 0213 mas 4111 não tem flag correspondente",
				count0213))
	}
	return nil
}

// XD-003 — Subsegmento DRSAC ESG deve ser compatível com classificação de risco 3040.
//
// Justificativa: IFs com subsegmento "S2" ou "S3" no DRSAC (maior risco)
// devem ter ScoreRisco mais alto no 3040.
type DRSACSubsegmentoClassificacaoRisco struct{}

func (DRSACSubsegmentoClassificacaoRisco) Code() string {
	return "XD-003"
}
func (DRSACSubsegmentoClassificacaoRisco) Description() string {
	return "Subsegmento DRSAC ESG deve ser compatível com classificação de risco no 3040"
}
func (DRSACSubsegmentoClassificacaoRisco) Severity() string { return "I" }
func (DRSACSubsegmentoClassificacaoRisco) RequiredDocs() []string {
	return []string{"3040", "2030"}
}
func (DRSACSubsegmentoClassificacaoRisco) Apply(_ context.Context, docs *crossdoc.DocSet) error {
	xml3040 := docs.Get("3040")
	xml2030 := docs.Get("2030")

	subseg := crossdoc.ExtractTextBetween(xml2030, "Subsegmento")
	if subseg == "" {
		return nil // DRSAC sem subsegmento → não valida
	}

	// Subsegmentos de maior risco: S4, S5
	if subseg != "S4" && subseg != "S5" {
		return nil
	}

	// 3040 deveria ter score de risco médio acima de X
	score := crossdoc.ExtractTextBetween(xml3040, "ScoreRiscoMedio")
	if score == "" {
		return nil // sem score → skip
	}

	// S4/S5 → score deve ser >= 0.7 (alto risco)
	var f float64
	_, _ = fmt.Sscanf(score, "%f", &f)
	if f < 0.7 {
		return crossdoc.NewError("XD-003", "I",
			fmt.Sprintf("DRSAC Subsegmento=%s mas ScoreRiscoMedio 3040=%s (esperado ≥0.7)",
				subseg, score))
	}
	return nil
}

// ===========================
// Helpers internos — XML line scanning
// ===========================

type xmlLine struct {
	Mod string
}

func iterateXMLElements(xmlContent, parentTag string) <-chan xmlLine {
	out := make(chan xmlLine)
	go func() {
		defer close(out)
		current := xmlLine{}
		openTag := "<" + parentTag + ">"
		idx := 0
		for {
			start := indexFrom(xmlContent, openTag, idx)
			if start == -1 {
				break
			}
			end := indexFrom(xmlContent, "</"+parentTag+">", start)
			if end == -1 {
				break
			}
			content := xmlContent[start+len(openTag) : end]
			// Extrai Mod (se houver)
			if mod := crossdoc.ExtractTextBetween(content, "Mod"); mod != "" {
				current.Mod = mod
				out <- current
			}
			idx = end + 1
		}
	}()
	return out
}

func indexFrom(s, sub string, from int) int {
	if from >= len(s) {
		return -1
	}
	idx := indexOf(sub, s[from:])
	if idx == -1 {
		return -1
	}
	return idx + from
}

func indexOf(sub, s string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
