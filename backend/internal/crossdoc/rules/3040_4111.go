// Regras cross-document iniciais 3040 ↔ 4111 ↔ DRSAC (Sprint 6 v1.5.0).
//
// Cada regra:
//   - Code: XD-NNN (namespace próprio)
//   - Description: human-readable
//   - Severity: E (erro) / A (aviso) / I (informativo)
//   - RequiredDocs: CADOCs que devem estar presentes
//   - Apply: lógica de validação cruzando os docs
//
// Sprint 72: refatorado para usar bacen.Doc3040 e doc4111 via xml.Unmarshal,
// eliminando string-scraping com regex frágil.
package rules

import (
	"context"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/bacen"
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	"github.com/fortvna/radiant-norma/backend/internal/doc4111"
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
	if xml3040 == "" || xml4111 == "" {
		return nil
	}

	// Usa typed unmarshal — xml.Unmarshal sobre Doc3040 e Documento4111.
	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return crossdoc.NewError("XD-001", "A", "3040 parse error: "+err.Error())
	}
	d4111, err := doc4111.ParseFromBytes([]byte(xml4111))
	if err != nil {
		return crossdoc.NewError("XD-001", "A", "4111 parse error: "+err.Error())
	}

	ops := doc3040.QtdOpTotal()
	clients := doc4111.ExtractQtdTotal(d4111)

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
	if xml3040 == "" {
		return nil
	}

	doc3040, err := bacen.Parse3040([]byte(xml3040))
	if err != nil {
		return nil
	}

	// Itera sobre Agreg[] usando struct tipada (sem string-scraping).
	count0213 := 0
	for _, a := range doc3040.Agregadas {
		if a.Mod == "0213" {
			count0213++
		}
	}

	// Se 3040 não tem cheque especial, regra é N/A.
	if count0213 == 0 {
		return nil
	}

	xml4111 := docs.Get("4111")
	if xml4111 == "" {
		return nil
	}

	d4111, err := doc4111.ParseFromBytes([]byte(xml4111))
	if err != nil {
		return nil
	}

	if !doc4111.HasModalidadeInadimplente(d4111) {
		return crossdoc.NewError("XD-002", "A",
			fmt.Sprintf("3040 reporta %d ocorrências de Mod 0213 mas 4111 não tem flag correspondente",
				count0213))
	}
	return nil
}

// XD-003 — Subsegmento DRSAC ESG deve ser compatível com classificação de risco 3040.
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
	xml2030 := docs.Get("2030")
	if xml2030 == "" {
		return nil
	}

	// Extrai Subsegmento do XML 2030 via string scan simples
	// (não vale a pena criar outro tipo para uma string só).
	subseg := crossdoc.ExtractTextBetween(xml2030, "Subsegmento")
	if subseg == "" {
		return nil
	}
	if subseg != "S4" && subseg != "S5" {
		return nil
	}

	xml3040 := docs.Get("3040")
	if xml3040 == "" {
		return nil
	}
	score := crossdoc.ExtractTextBetween(xml3040, "ScoreRiscoMedio")
	if score == "" {
		return nil
	}

	var f float64
	fmt.Sscanf(score, "%f", &f)
	if f < 0.7 {
		return crossdoc.NewError("XD-003", "I",
			fmt.Sprintf("DRSAC Subsegmento=%s mas ScoreRiscoMedio 3040=%s (esperado ≥0.7)",
				subseg, score))
	}
	return nil
}

var _ = bacen.Parse3040
