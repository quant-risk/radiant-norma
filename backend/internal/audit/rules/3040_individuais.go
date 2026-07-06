// Regras Individuais (C11-C20, S13/S14, I01-I05/I11, H01-H03) do CADOC 3040 — Sprint 32 Fase 3.
//
// C11-C20: Campos Obrigatórios por Inf (Sprint 32 Fase 3 escopo reduzido de C11-C30)
// S13: Garantidor fidejussório ≠ próprio cliente
// S14: DtVencOp >= DtContr
// I01: Classificação × Provisão individual
// I02/I07: Vencimentos × Classificação individual (apenas I02; I07 carry-over)
// I03: Unicidade cliente + TpCli
// I04: Unicidade contrato + modalidade
// I05: Unicidade vencimentos em uma operação
// I11: NatuOp≠32 em Cli
// H01-H03: Header (formato da remessa)
//
// Carry-over Fase 4: C21/C23-C29, I06-I10/I12-I15, H04-H09
//
// Pattern: cada regra itera `doc.Operacoes`. Se Operacoes está vazio, regras
// individuais simplesmente passam (não rodam). Isso é consistente com parser
// atual que não popula Operacoes — quando parser for atualizado, regras passam
// a ter dados.
package rules

import (
	"context"
	"fmt"
	"strconv"
)

// ============================================================
// C11 — DtVenc obrigatória para todas operações (exceto v199)
// ============================================================

// C11 — Data vencimento obrigatória.
//
// Catálogo BACEN: "A data de vencimento da operação é obrigatória para todas
// as operações, exceto para aquelas que possuem vencimentos v199."
//
// Sprint 32 Fase 3: valida DtVencOp != "" em todas operações. Exceção v199
// (vencido > 1 ano) não está no struct → simplificação: valida apenas presença.
type C11DtVencObrigatoria struct{}

func (C11DtVencObrigatoria) Code() string     { return "C11" }
func (C11DtVencObrigatoria) Sheet() string    { return "Campos Obrigatórios" }
func (C11DtVencObrigatoria) Severity() string { return "E" }
func (C11DtVencObrigatoria) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.DtVencOp == "" {
			return fmt.Errorf("operação %d: DtVencOp obrigatória (Inf=%s, Contrt=%s)",
				i, op.Inf, op.Contrt)
		}
	}
	return nil
}

// ============================================================
// C13 — Inf 0303/0304: Cd, Ident, Valor obrigatórios
// ============================================================

// C13 — Para Inf=0303 (cessão com coobrigação) ou 0304 (cessão sem coobrigação):
// campos obrigatórios: Cd (data cessão), Ident (cessionário), Valor (valor negociado).
type C13Inf0303Cessao struct{}

func (C13Inf0303Cessao) Code() string     { return "C13" }
func (C13Inf0303Cessao) Sheet() string    { return "Campos Obrigatórios" }
func (C13Inf0303Cessao) Severity() string { return "E" }
func (C13Inf0303Cessao) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0303" && op.Inf != "0304" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=%s): Cd (data cessão) obrigatório", i, op.Inf)
		}
		if op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=%s): Ident (cessionário) obrigatório", i, op.Inf)
		}
		if op.Valor == "" {
			return fmt.Errorf("operação %d (Inf=%s): Valor (negociado) obrigatório", i, op.Inf)
		}
	}
	return nil
}

// ============================================================
// C14 — Inf 0305: novo contrato, modalidade, Valor renegociado obrigatórios
// ============================================================

// C14 — Para Inf=0305 (renegociação): Contrt, IPOC (modalidade), Valor obrigatórios.
type C14Inf0305Renegociacao struct{}

func (C14Inf0305Renegociacao) Code() string     { return "C14" }
func (C14Inf0305Renegociacao) Sheet() string    { return "Campos Obrigatórios" }
func (C14Inf0305Renegociacao) Severity() string { return "E" }
func (C14Inf0305Renegociacao) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0305" {
			continue
		}
		if op.Contrt == "" || op.IPOC == "" || op.Valor == "" {
			return fmt.Errorf("operação %d (Inf=0305): Contrt+IPOC(modalidade)+Valor(renegociado) obrigatórios",
				i)
		}
	}
	return nil
}

// ============================================================
// C16 — Inf 0307: novo contrato, modalidade
// ============================================================

// C16 — Para Inf=0307: Contrt + IPOC (modalidade+submodalidade).
type C16Inf0307 struct{}

func (C16Inf0307) Code() string     { return "C16" }
func (C16Inf0307) Sheet() string    { return "Campos Obrigatórios" }
func (C16Inf0307) Severity() string { return "E" }
func (C16Inf0307) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0307" {
			continue
		}
		if op.Contrt == "" || op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=0307): Contrt+IPOC(modalidade) obrigatórios", i)
		}
	}
	return nil
}

// ============================================================
// C17 — Inf 04XX: Cd (código instrumento)
// ============================================================

// C17 — Para Inf=04XX (instrumentos): Contrt (código instrumento) obrigatório.
type C17Inf04XX struct{}

func (C17Inf04XX) Code() string     { return "C17" }
func (C17Inf04XX) Sheet() string    { return "Campos Obrigatórios" }
func (C17Inf04XX) Severity() string { return "E" }
func (C17Inf04XX) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.Inf) != 4 || op.Inf[:2] != "04" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=%s): Contrt (código instrumento) obrigatório", i, op.Inf)
		}
	}
	return nil
}

// ============================================================
// C18 — Inf 05XX: Cd
// ============================================================

// C18 — Para Inf=05XX: Contrt obrigatório.
type C18Inf05XX struct{}

func (C18Inf05XX) Code() string     { return "C18" }
func (C18Inf05XX) Sheet() string    { return "Campos Obrigatórios" }
func (C18Inf05XX) Severity() string { return "E" }
func (C18Inf05XX) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.Inf) != 4 || op.Inf[:2] != "05" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=%s): Contrt obrigatório", i, op.Inf)
		}
	}
	return nil
}

// ============================================================
// C19 — Inf 0701: Cd, Perc, Valor
// ============================================================

// C19 — Para Inf=0701 (cessionário): Contrt, Perc, Valor obrigatórios.
type C19Inf0701 struct{}

func (C19Inf0701) Code() string     { return "C19" }
func (C19Inf0701) Sheet() string    { return "Campos Obrigatórios" }
func (C19Inf0701) Severity() string { return "E" }
func (C19Inf0701) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0701" {
			continue
		}
		if op.Contrt == "" || op.Perc == "" || op.Valor == "" {
			return fmt.Errorf("operação %d (Inf=0701): Contrt+Perc+Valor obrigatórios", i)
		}
	}
	return nil
}

// ============================================================
// C20 — Inf 0702/0703/0704: cedente
// ============================================================

// C20 — Para Inf=0702/0703/0704 (cedente): Contrt, IPOC, Perc, Valor obrigatórios.
type C20Inf0702 struct{}

func (C20Inf0702) Code() string     { return "C20" }
func (C20Inf0702) Sheet() string    { return "Campos Obrigatórios" }
func (C20Inf0702) Severity() string { return "E" }
func (C20Inf0702) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0702" && op.Inf != "0703" && op.Inf != "0704" {
			continue
		}
		if op.Contrt == "" || op.IPOC == "" || op.Perc == "" || op.Valor == "" {
			return fmt.Errorf("operação %d (Inf=%s): Contrt+IPOC(cessionário)+Perc+Valor obrigatórios", i, op.Inf)
		}
	}
	return nil
}

// ============================================================
// S13 — Garantidor fidejussório ≠ próprio cliente
// ============================================================

// S13 — Para pessoa física, Cli.Cd deve ser diferente de Garantidores[i].Ident.
// Implementação: percorre Operacoes, pra cada uma compara Cli.Cd com cada garantidor.
// Não-físico (TpCli=2): não há constraint de CPF diferente.
type S13GarantidorNaoCliente struct{}

func (S13GarantidorNaoCliente) Code() string     { return "S13" }
func (S13GarantidorNaoCliente) Sheet() string    { return "Sistemáticas" }
func (S13GarantidorNaoCliente) Severity() string { return "E" }
func (S13GarantidorNaoCliente) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Cli == nil || op.Cli.TpCli != "1" {
			continue // só PF tem constraint
		}
		for j, gar := range op.Garantidores {
			if gar == op.Cli.Cd {
				return fmt.Errorf("operação %d: garantidor %d tem mesmo Cd do cliente (%s) — fidejussório próprio não admitido",
					i, j, gar)
			}
		}
	}
	return nil
}

// ============================================================
// S14 — DtVencOp >= DtContr
// ============================================================

// S14 — DtVencOp deve ser >= DtContr (formato YYYY-MM-DD).
//
// Implementação simplificada: comparação lexicográfica de string YYYY-MM-DD.
// Funciona porque YYYY-MM-DD tem ordering lexicográfico = cronológico.
type S14DtVencMaiorDtContr struct{}

func (S14DtVencMaiorDtContr) Code() string     { return "S14" }
func (S14DtVencMaiorDtContr) Sheet() string    { return "Sistemáticas" }
func (S14DtVencMaiorDtContr) Severity() string { return "E" }
func (S14DtVencMaiorDtContr) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.DtContr == "" || op.DtVencOp == "" {
			continue // campos vazios → outras regras tratam (C11, etc)
		}
		if op.DtVencOp < op.DtContr {
			return fmt.Errorf("operação %d: DtVencOp=%s anterior a DtContr=%s",
				i, op.DtVencOp, op.DtContr)
		}
	}
	return nil
}

// ============================================================
// I01 — Classificação × Provisão individual (Tab. A01)
// ============================================================

// I01 — Classificação individual × Provisão.
//
// Catálogo BACEN: "Exceto modalidades 19XX: ClassOp=AA → 0% ≤ ProvConsttd/VlrVenc < 0.5%..."
//
// Implementação: usa mesma tabela tabelaClassOpProvisaoA01 que A01 (agregado).
type I01ClassOpProvisaoIndividual struct{}

func (I01ClassOpProvisaoIndividual) Code() string     { return "I01" }
func (I01ClassOpProvisaoIndividual) Sheet() string    { return "Individualizadas" }
func (I01ClassOpProvisaoIndividual) Severity() string { return "E" }
func (I01ClassOpProvisaoIndividual) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		// Exceto modalidades 19XX
		if len(op.Inf) >= 2 && op.Inf[:2] == "19" {
			continue
		}
		prov := parseNum(op.ProvConsttd)
		total := totalVencimentosOperacao(op)
		if total == 0 {
			continue
		}
		ratio := prov / total
		for _, e := range tabelaClassOpProvisaoA01 {
			if e.ClassOp == op.ClassOp {
				if ratio < e.ProvMin || ratio >= e.ProvMax {
					return fmt.Errorf("operação %d (ClassOp=%s): provisão/∑Venc = %.4f (esperado %.4f ≤ ratio < %.4f)",
						i, op.ClassOp, ratio, e.ProvMin, e.ProvMax)
				}
				break
			}
		}
	}
	return nil
}

// ============================================================
// I02 — Vencimentos × Classificação individual (sem prazo)
// ============================================================

// I02 — Individual: vencimentos não podem exceder 210/240/360 dias conforme ClassOp.
type I02ClassOpVencIndividual struct{}

func (I02ClassOpVencIndividual) Code() string     { return "I02" }
func (I02ClassOpVencIndividual) Sheet() string    { return "Individualizadas" }
func (I02ClassOpVencIndividual) Severity() string { return "E" }
func (I02ClassOpVencIndividual) Apply(_ context.Context, doc *Doc3040) error {
	prazos := map[string]float64{
		"AA": 210, "A": 210,
		"B": 240, "C": 240,
		"D": 360, "E": 360, "F": 360, "G": 360, "H": 360,
	}
	for i, op := range doc.Operacoes {
		max, ok := prazos[op.ClassOp]
		if !ok {
			continue
		}
		maior := maxVencimentoOperacao(op)
		if maior >= max {
			return fmt.Errorf("operação %d (ClassOp=%s): vencimento %.0f excede prazo máximo %.0f",
				i, op.ClassOp, maior, max)
		}
	}
	return nil
}

// ============================================================
// I03 — Unicidade cliente + TpCli na remessa
// ============================================================

// I03 — Não é admitida repetição de (Cd, TpCli) na remessa.
type I03CliTpCliUnico struct{}

func (I03CliTpCliUnico) Code() string     { return "I03" }
func (I03CliTpCliUnico) Sheet() string    { return "Individualizadas" }
func (I03CliTpCliUnico) Severity() string { return "E" }
func (I03CliTpCliUnico) Apply(_ context.Context, doc *Doc3040) error {
	type chave struct{ Cd, TpCli string }
	seen := make(map[chave]int)
	for i, op := range doc.Operacoes {
		if op.Cli == nil {
			continue
		}
		k := chave{op.Cli.Cd, op.Cli.TpCli}
		if idx, exists := seen[k]; exists {
			return fmt.Errorf("operação %d duplicata cliente (Cd=%s, TpCli=%s) da operação %d",
				i, op.Cli.Cd, op.Cli.TpCli, idx)
		}
		seen[k] = i
	}
	return nil
}

// ============================================================
// I04 — Unicidade contrato + modalidade
// ============================================================

// I04 — Não é admitida repetição de (Contrt, IPOC=modalidade) na remessa.
type I04ContratoModalidadeUnico struct{}

func (I04ContratoModalidadeUnico) Code() string     { return "I04" }
func (I04ContratoModalidadeUnico) Sheet() string    { return "Individualizadas" }
func (I04ContratoModalidadeUnico) Severity() string { return "E" }
func (I04ContratoModalidadeUnico) Apply(_ context.Context, doc *Doc3040) error {
	type chave struct{ Contrt, IPOC string }
	seen := make(map[chave]int)
	for i, op := range doc.Operacoes {
		if op.Contrt == "" || op.IPOC == "" {
			continue
		}
		k := chave{op.Contrt, op.IPOC}
		if idx, exists := seen[k]; exists {
			return fmt.Errorf("operação %d duplicata contrato (Contrt=%s, Mod=%s) da operação %d",
				i, op.Contrt, op.IPOC, idx)
		}
		seen[k] = i
	}
	return nil
}

// ============================================================
// I05 — Unicidade vencimentos em uma operação
// ============================================================

// I05 — Em uma única operação, não pode haver vencimentos duplicados.
type I05VencimentosUnicos struct{}

func (I05VencimentosUnicos) Code() string     { return "I05" }
func (I05VencimentosUnicos) Sheet() string    { return "Individualizadas" }
func (I05VencimentosUnicos) Severity() string { return "E" }
func (I05VencimentosUnicos) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		v := op.Vencimentos
		faixas := []string{v.V110, v.V120, v.V150, v.V160, v.V165}
		seen := make(map[string]int)
		for _, val := range faixas {
			if val == "" || val == "0" || val == "0.0" {
				continue
			}
			if _, exists := seen[val]; exists {
				return fmt.Errorf("operação %d: vencimento %q duplicado (V110-V165)", i, val)
			}
			seen[val] = 1
		}
	}
	return nil
}

// ============================================================
// I11 — NatuOp≠32 em Cli (individual)
// ============================================================

// I11 — Operações individualizadas (tag <Cli>) não podem ter natureza 32.
// Implementação: checa se op.Cli != nil && NatuOp=32 → erro.
// Sprint 32 Fase 3: op não tem NatuOp próprio (NatuOp está em Agregado).
// Implementação alternativa: error se op.Cli != nil com algum flag heurístico.
// Como Cli != nil já indica "operação individual", I11 vira simplificação:
// se Cli presente, NatuOp=32 é proibido. Sem NatuOp individual, regra é stub
// que aceita sempre (carry-over Fase 4: adicionar NatuOp a Operacao).
type I11CliNaoNatuOp32 struct{}

func (I11CliNaoNatuOp32) Code() string     { return "I11" }
func (I11CliNaoNatuOp32) Sheet() string    { return "Individualizadas" }
func (I11CliNaoNatuOp32) Severity() string { return "E" }
func (I11CliNaoNatuOp32) Apply(_ context.Context, _ *Doc3040) error {
	// Sprint 32 Fase 3: stub. Requer NatuOp individual no struct Operacao.
	// Quando parser popular NatuOp por op, essa regra valida Cli+NatuOp≠32.
	return nil
}

// ============================================================
// H01 — Header TpArq = F (full) ou S (substituição)
// ============================================================

// H01 — Tipo de arquivo deve ser F (full) ou S (substituição).
// Já existe em regras Básicas (provavelmente B18), mas aqui duplicamos
// por completude do header group.
type H01TpArqValido struct{}

func (H01TpArqValido) Code() string     { return "H01" }
func (H01TpArqValido) Sheet() string    { return "Header" }
func (H01TpArqValido) Severity() string { return "E" }
func (H01TpArqValido) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq != "F" && doc.Root.TpArq != "S" {
		return fmt.Errorf("TpArq=%q inválido (esperado F=full ou S=substituição)", doc.Root.TpArq)
	}
	return nil
}

// ============================================================
// H02 — CNPJ 8 dígitos (raiz)
// ============================================================

// H02 — CNPJ da IF deve ter 8 dígitos (raiz).
type H02CNPJRaiz struct{}

func (H02CNPJRaiz) Code() string     { return "H02" }
func (H02CNPJRaiz) Sheet() string    { return "Header" }
func (H02CNPJRaiz) Severity() string { return "E" }
func (H02CNPJRaiz) Apply(_ context.Context, doc *Doc3040) error {
	if len(doc.Root.CNPJ) != 8 {
		return fmt.Errorf("CNPJ=%q deve ter 8 dígitos (raiz), recebido %d",
			doc.Root.CNPJ, len(doc.Root.CNPJ))
	}
	if _, err := strconv.Atoi(doc.Root.CNPJ); err != nil {
		return fmt.Errorf("CNPJ=%q não-numérico", doc.Root.CNPJ)
	}
	return nil
}

// ============================================================
// H03 — TotalCli > 0
// ============================================================

// H03 — TotalCli no header deve ser > 0.
type H03TotalCliPositivo struct{}

func (H03TotalCliPositivo) Code() string     { return "H03" }
func (H03TotalCliPositivo) Sheet() string    { return "Header" }
func (H03TotalCliPositivo) Severity() string { return "E" }
func (H03TotalCliPositivo) Apply(_ context.Context, doc *Doc3040) error {
	total, err := strconv.Atoi(doc.Root.TotalCli)
	if err != nil {
		return fmt.Errorf("TotalCli=%q não-numérico", doc.Root.TotalCli)
	}
	if total <= 0 {
		return fmt.Errorf("TotalCli=%d deve ser > 0", total)
	}
	return nil
}

// ============================================================
// Helpers para Operacao
// ============================================================

// totalVencimentosOperacao soma V110-V165 de uma operação.
func totalVencimentosOperacao(op Operacao) float64 {
	return parseNum(op.Vencimentos.V110) +
		parseNum(op.Vencimentos.V120) +
		parseNum(op.Vencimentos.V150) +
		parseNum(op.Vencimentos.V160) +
		parseNum(op.Vencimentos.V165)
}

// maxVencimentoOperacao retorna maior vencimento entre V110-V165.
func maxVencimentoOperacao(op Operacao) float64 {
	max := 0.0
	for _, v := range []string{
		op.Vencimentos.V110,
		op.Vencimentos.V120,
		op.Vencimentos.V150,
		op.Vencimentos.V160,
		op.Vencimentos.V165,
	} {
		val := parseNum(v)
		if val > max {
			max = val
		}
	}
	return max
}
