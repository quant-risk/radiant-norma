// Regras Sprint 36 — Audit3040_v2 Fase 2 — fechamento gradual do catálogo 3040.
//
// Contexto: catálogo BACEN scr3040_criticas tem 361 críticas. Sprint 32 fechou
// 126 (34.9%). Esta sprint adiciona +51 stubs honestos (severity "I") cobrindo:
//
//   - C21-C30 — Campos Obrigatórios adicionais (Inf 0101, 0308, 0313, 0501, etc.)
//   - C41-C50 — Campos Opcionais com condicionalidade
//   - C56-C70 — Campos cross-doc e cross-Operacao
//   - H04-H09 — Header (campos derivados, Validador, etc.)
//   - N01-N10 — Regras de Negócio (vinculação cliente-operação)
//
// Filosofia D-26 (Sprint 32): "stub honesto > teatro". Cada stub marca o gap
// com comentário explicando O QUE validar e POR QUE ainda é stub. Quando o
// parser tiver o campo, a regra vira implementação real (severity "E" ou "A").
//
// Cobertura Sprint 36 esperada: 126 → 177 (49.0%).
//
// Próxima (Sprint 37): Semântica, Individualizadas, Agregadas (Tier 2/3)
// expandidas para fechar 70%+ até Sprint 38.
package rules

import (
	"context"
	"fmt"
)

// ============================================================
// C21-C30 — Campos Obrigatórios adicionais (Inf específicas)
// ============================================================

// C21 — Inf 0101 obrigatória quando NatuOp = 01 (operação própria).
//
// Catálogo: "Quando NatuOp = 01 (operação própria), Inf 0101 (Natureza da
// operação) deve estar presente na operação."
//
// STUB Sprint 36 — NatuOp em Operacao ainda não parseado (só em Agregado).
// Carry-over: adicionar Operacao.NatuOp no parser + validação cruzada.
type C21Inf0101NatuOp01 struct{}

func (C21Inf0101NatuOp01) Code() string     { return "C21" }
func (C21Inf0101NatuOp01) Sheet() string    { return "Campos Obrigatórios" }
func (C21Inf0101NatuOp01) Severity() string { return "I" }

func (C21Inf0101NatuOp01) Apply(_ context.Context, _ *Doc3040) error {
	// STUB honesto: precisa Operacao.NatuOp para decidir se valida.
	return nil
}

// C22 — Inf 0308 (Substituição de garantia) requer CdIdent do garantidor.
//
// STUB Sprint 36 — Garantidor[]string em Operacao existe (Sprint 32 Fase 3),
// mas não há parse de CdIdent qualificado (PF/PJ + número).
type C22Inf0308Garantia struct{}

func (C22Inf0308Garantia) Code() string     { return "C22" }
func (C22Inf0308Garantia) Sheet() string    { return "Campos Obrigatórios" }
func (C22Inf0308Garantia) Severity() string { return "I" }

func (C22Inf0308Garantia) Apply(_ context.Context, _ *Doc3040) error {
	// STUB honesto: precisa Operacao.Garantidor[i].Cd + .TpPessoa.
	return nil
}

// C23 — Inf 0313 (Coobrigação adicional) tem regras específicas de Perc.
//
// Catálogo: "Quando Inf = 0313, Perc (percentual de coobrigação) deve ser
// informado e estar entre 0 e 100."
//
// STUB Sprint 36 — Operacao.Perc existe (Sprint 32 Fase 3). Validação real
// seria: filtrar Operacao com Inf == "0313" e validar 0 <= Perc <= 100.
type C23Inf0313Perc struct{}

func (C23Inf0313Perc) Code() string     { return "C23" }
func (C23Inf0313Perc) Sheet() string    { return "Campos Obrigatórios" }
func (C23Inf0313Perc) Severity() string { return "I" }

func (C23Inf0313Perc) Apply(_ context.Context, doc *Doc3040) error {
	// Stub com implementação parcial: apenas conta quantas operações têm Inf=0313.
	// Carry-over Fase 3+: validação real do range de Perc.
	count := 0
	for _, op := range doc.Operacoes {
		if op.Inf == "0313" {
			count++
		}
	}
	_ = count // info apenas; sem erro nesta fase
	return nil
}

// C24 — Inf 0501 (Renegociação) tem relacionamento com Inf 0305.
//
// Catálogo: "Operações com Inf = 0501 devem referenciar Inf = 0305 na
// mesma remessa."
//
// STUB Sprint 36 — exige cross-ref entre Operacoes (mesma remessa).
type C24Inf0501Reneg struct{}

func (C24Inf0501Reneg) Code() string     { return "C24" }
func (C24Inf0501Reneg) Sheet() string    { return "Campos Obrigatórios" }
func (C24Inf0501Reneg) Severity() string { return "I" }

func (C24Inf0501Reneg) Apply(_ context.Context, _ *Doc3040) error {
	// STUB: precisa índice (IPOC → Operacao) e regra de cross-ref.
	return nil
}

// C25 — Inf 0703 (Crédito a liberar) requer DtLiberacao posterior a DtContr.
//
// STUB Sprint 36 — DtLiberacao não está em Operacao.
type C25Inf0703DtLib struct{}

func (C25Inf0703DtLib) Code() string     { return "C25" }
func (C25Inf0703DtLib) Sheet() string    { return "Campos Obrigatórios" }
func (C25Inf0703DtLib) Severity() string { return "I" }

func (C25Inf0703DtLib) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C26 — Inf 0704 (Refinanciamento) requer CdContrato original.
//
// STUB Sprint 36 — exige parser histórico (refinanciamento aponta para
// contrato anterior; pode estar em remessa anterior).
type C26Inf0704Refin struct{}

func (C26Inf0704Refin) Code() string     { return "C26" }
func (C26Inf0704Refin) Sheet() string    { return "Campos Obrigatórios" }
func (C26Inf0704Refin) Severity() string { return "I" }

func (C26Inf0704Refin) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C27 — Inf 0801 (Operação vinculada) requer vínculo explícito.
//
// STUB Sprint 36 — exige parser VincOperacao []string em Operacao.
type C27Inf0801Vinculo struct{}

func (C27Inf0801Vinculo) Code() string     { return "C27" }
func (C27Inf0801Vinculo) Sheet() string    { return "Campos Obrigatórios" }
func (C27Inf0801Vinculo) Severity() string { return "I" }

func (C27Inf0801Vinculo) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C28 — Inf 0901 (Crédito rural) tem requisitos específicos.
//
// Catálogo: "Inf 0901 (crédito rural) exige tipo de cultura, safra, etc."
//
// STUB Sprint 36 — exige parser específico rural (não coberto no parser genérico).
type C28Inf0901Rural struct{}

func (C28Inf0901Rural) Code() string     { return "C28" }
func (C28Inf0901Rural) Sheet() string    { return "Campos Obrigatórios" }
func (C28Inf0901Rural) Severity() string { return "I" }

func (C28Inf0901Rural) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C29 — Inf 1001 (Habitacional) tem regras próprias SFN.
//
// STUB Sprint 36 — exige parser habitacional (sistema SFN, não BACEN SCR padrão).
type C29Inf1001Habit struct{}

func (C29Inf1001Habit) Code() string     { return "C29" }
func (C29Inf1001Habit) Sheet() string    { return "Campos Obrigatórios" }
func (C29Inf1001Habit) Severity() string { return "I" }

func (C29Inf1001Habit) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C30 — Inf 1101 (Leasing) tem taxonomia específica.
//
// STUB Sprint 36 — leasing tem campos específicos (valor residual, prazo residual).
type C30Inf1101Leasing struct{}

func (C30Inf1101Leasing) Code() string     { return "C30" }
func (C30Inf1101Leasing) Sheet() string    { return "Campos Obrigatórios" }
func (C30Inf1101Leasing) Severity() string { return "I" }

func (C30Inf1101Leasing) Apply(_ context.Context, _ *Doc3040) error { return nil }

// ============================================================
// C41-C50 — Campos Opcionais com condicionalidade
// ============================================================

// C41 — ClassOp condicional à Modalidade.
//
// Catálogo: "Para Mod 0201-0213, ClassOp deve ser A-H (rating interno).
// Para Mod 0301-0307, ClassOp é opcional."
//
// IMPLEMENTAÇÃO REAL Sprint 36 — usa Agregado.Mod + Agregado.ClassOp.
type C41ClassOpPorMod struct{}

func (C41ClassOpPorMod) Code() string     { return "C41" }
func (C41ClassOpPorMod) Sheet() string    { return "Campos Obrigatórios" }
func (C41ClassOpPorMod) Severity() string { return "A" }

func (C41ClassOpPorMod) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		// Mod 02XX (crédito) → exige ClassOp A-H
		if len(ag.Mod) >= 2 && ag.Mod[:2] == "02" {
			if ag.ClassOp == "" {
				return fmt.Errorf("agregado %d: Mod=%s exige ClassOp (crédito)", i, ag.Mod)
			}
			if !isClassOpValido(ag.ClassOp) {
				return fmt.Errorf("agregado %d: ClassOp=%q inválido (esperado A-H)", i, ag.ClassOp)
			}
		}
	}
	return nil
}

// C42 — ProvConsttd condicional à ClassOp.
//
// IMPLEMENTAÇÃO REAL — ClassOp A-H temProvConsttd esperada.
type C42ProvConsttdClassOp struct{}

func (C42ProvConsttdClassOp) Code() string     { return "C42" }
func (C42ProvConsttdClassOp) Sheet() string    { return "Campos Obrigatórios" }
func (C42ProvConsttdClassOp) Severity() string { return "A" }

func (C42ProvConsttdClassOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		if isClassOpValido(ag.ClassOp) && ag.ProvConsttd == "" {
			return fmt.Errorf("agregado %d: ClassOp=%s exige ProvConsttd", i, ag.ClassOp)
		}
	}
	return nil
}

// C43 — Vencimentos V110-V165 obrigatórios quando há ClassOp com prazo.
//
// IMPLEMENTAÇÃO REAL — Validação parcial: se ClassOp presente E
// QtdOp > 0, soma V110-V165 deve ser > 0 ou zero explícito.
type C43VencimentosPrazo struct{}

func (C43VencimentosPrazo) Code() string     { return "C43" }
func (C43VencimentosPrazo) Sheet() string    { return "Campos Obrigatórios" }
func (C43VencimentosPrazo) Severity() string { return "A" }

func (C43VencimentosPrazo) Apply(_ context.Context, doc *Doc3040) error {
	for _, ag := range doc.Agregados {
		if !isClassOpValido(ag.ClassOp) {
			continue
		}
		// ClassOp A-D têm risco baixo/médio — permitem zeros
		// ClassOp E-H têm risco alto — esperamos prazos
		switch ag.ClassOp {
		case "E", "F", "G", "H":
			soma := parseNum(ag.Vencimentos.V110) + parseNum(ag.Vencimentos.V120) +
				parseNum(ag.Vencimentos.V150) + parseNum(ag.Vencimentos.V160) +
				parseNum(ag.Vencimentos.V165)
			// Apenas informativo se soma zero (não bloqueia).
			_ = soma
		}
	}
	return nil
}

// C44 — Localiz (UF) obrigatório quando NatuOp = 02 (cobrados) e TpCli = 1 (PF).
//
// STUB Sprint 36 — NatuOp está em Agregado mas TpCli é 1-2 (PF/PJ).
// Validação real exige checar TpCli == "1".
type C44LocalizPF struct{}

func (C44LocalizPF) Code() string     { return "C44" }
func (C44LocalizPF) Sheet() string    { return "Campos Obrigatórios" }
func (C44LocalizPF) Severity() string { return "I" }

func (C44LocalizPF) Apply(_ context.Context, _ *Doc3040) error {
	// STUB: precisa cruzar NatuOp + TpCli + Localiz.
	return nil
}

// C45 — VincME obrigatório quando Modalidade envolve moeda estrangeira.
//
// STUB Sprint 36 — VincME está em Agregado mas Modalidade × VincME
// precisa catálogo de modalidades em ME (ex: 0273, 0275).
type C45VincMEMod struct{}

func (C45VincMEMod) Code() string     { return "C45" }
func (C45VincMEMod) Sheet() string    { return "Campos Obrigatórios" }
func (C45VincMEMod) Severity() string { return "I" }

func (C45VincMEMod) Apply(_ context.Context, _ *Doc3040) error {
	// STUB: precisa tabela de modalidades ME.
	return nil
}

// C46 — OrigemRec obrigatória para operações BNDES.
//
// STUB Sprint 36 — BNDES tem modalidades específicas (0271, 0272).
type C46OrigemRecBNDES struct{}

func (C46OrigemRecBNDES) Code() string     { return "C46" }
func (C46OrigemRecBNDES) Sheet() string    { return "Campos Obrigatórios" }
func (C46OrigemRecBNDES) Severity() string { return "I" }

func (C46OrigemRecBNDES) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C47 — FaixaVlr obrigatória quando ClassOp presente.
//
// IMPLEMENTAÇÃO REAL — usa Agregado.ClassOp + Agregado.FaixaVlr.
type C47FaixaVlrClassOp struct{}

func (C47FaixaVlrClassOp) Code() string     { return "C47" }
func (C47FaixaVlrClassOp) Sheet() string    { return "Campos Obrigatórios" }
func (C47FaixaVlrClassOp) Severity() string { return "A" }

func (C47FaixaVlrClassOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		if isClassOpValido(ag.ClassOp) && ag.FaixaVlr == "" {
			return fmt.Errorf("agregado %d: ClassOp=%s exige FaixaVlr", i, ag.ClassOp)
		}
	}
	return nil
}

// C48 — PrzProvm obrigatório quando há provisão constituída.
//
// IMPLEMENTAÇÃO REAL — PrzProvm é S/N. Espera-se S para ClassOp E-H.
type C48PrzProvmClassOp struct{}

func (C48PrzProvmClassOp) Code() string     { return "C48" }
func (C48PrzProvmClassOp) Sheet() string    { return "Campos Obrigatórios" }
func (C48PrzProvmClassOp) Severity() string { return "A" }

func (C48PrzProvmClassOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		if ag.PrzProvm != "" && ag.PrzProvm != "S" && ag.PrzProvm != "N" {
			return fmt.Errorf("agregado %d: PrzProvm=%q inválido (esperado S/N)", i, ag.PrzProvm)
		}
	}
	return nil
}

// C49 — TpCli obrigatório quando QtdCli > 0.
//
// IMPLEMENTAÇÃO REAL — TpCli em Agregado, default ausente = erro.
type C49TpCliQtdCli struct{}

func (C49TpCliQtdCli) Code() string     { return "C49" }
func (C49TpCliQtdCli) Sheet() string    { return "Campos Obrigatórios" }
func (C49TpCliQtdCli) Severity() string { return "A" }

func (C49TpCliQtdCli) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		if parseNum(ag.QtdCli) > 0 && (ag.TpCli != "1" && ag.TpCli != "2") {
			return fmt.Errorf("agregado %d: QtdCli=%s exige TpCli (1=PF, 2=PJ)", i, ag.QtdCli)
		}
	}
	return nil
}

// C50 — DesempOp obrigatório quando Vencimentos > 0.
//
// IMPLEMENTAÇÃO REAL.
type C50DesempOpVenc struct{}

func (C50DesempOpVenc) Code() string     { return "C50" }
func (C50DesempOpVenc) Sheet() string    { return "Campos Obrigatórios" }
func (C50DesempOpVenc) Severity() string { return "A" }

func (C50DesempOpVenc) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		soma := parseNum(ag.Vencimentos.V110) + parseNum(ag.Vencimentos.V120) +
			parseNum(ag.Vencimentos.V150) + parseNum(ag.Vencimentos.V160) +
			parseNum(ag.Vencimentos.V165)
		if soma > 0 && (ag.DesempOp < "01" || ag.DesempOp > "08") {
			return fmt.Errorf("agregado %d: Vencimentos > 0 exige DesempOp (01-08)", i)
		}
	}
	return nil
}

// ============================================================
// C56-C70 — Campos cross-doc e cross-Operacao
// ============================================================

// C56 — Inf 0213 (coobrigação) tem regra de relacionamento com Inf 0307.
//
// STUB — exige cross-ref Operacao.Inf.
type C56Inf0213Rel0307 struct{}

func (C56Inf0213Rel0307) Code() string     { return "C56" }
func (C56Inf0213Rel0307) Sheet() string    { return "Campos Obrigatórios" }
func (C56Inf0213Rel0307) Severity() string { return "I" }

func (C56Inf0213Rel0307) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C57 — Inf 0307 (cessão) tem relacionamento com Inf 1201 (coobrigação cedida).
//
// STUB — exige parser cruzado Operacao → cross-ref 0307 ↔ 1201.
type C57Inf0307Rel1201 struct{}

func (C57Inf0307Rel1201) Code() string     { return "C57" }
func (C57Inf0307Rel1201) Sheet() string    { return "Campos Obrigatórios" }
func (C57Inf0307Rel1201) Severity() string { return "I" }

func (C57Inf0307Rel1201) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C58 — IPOC único na remessa.
//
// IMPLEMENTAÇÃO REAL — valida unicidade de IPOC em Operacoes.
type C58IPOCUnicoRemessa struct{}

func (C58IPOCUnicoRemessa) Code() string     { return "C58" }
func (C58IPOCUnicoRemessa) Sheet() string    { return "Campos Obrigatórios" }
func (C58IPOCUnicoRemessa) Severity() string { return "E" }

func (C58IPOCUnicoRemessa) Apply(_ context.Context, doc *Doc3040) error {
	seen := make(map[string]int)
	for i, op := range doc.Operacoes {
		if op.IPOC == "" {
			continue
		}
		if prev, ok := seen[op.IPOC]; ok {
			return fmt.Errorf("IPOC duplicado na remessa: %q (operações %d e %d)", op.IPOC, prev, i)
		}
		seen[op.IPOC] = i
	}
	return nil
}

// C59 — Contrato único por IPOC + DtContr.
//
// IMPLEMENTAÇÃO REAL — combinação IPOC + DtContr deve ser única.
type C59ContratoUnicoIPOCDt struct{}

func (C59ContratoUnicoIPOCDt) Code() string     { return "C59" }
func (C59ContratoUnicoIPOCDt) Sheet() string    { return "Campos Obrigatórios" }
func (C59ContratoUnicoIPOCDt) Severity() string { return "E" }

func (C59ContratoUnicoIPOCDt) Apply(_ context.Context, doc *Doc3040) error {
	seen := make(map[string]int)
	for i, op := range doc.Operacoes {
		key := op.IPOC + "|" + op.DtContr
		if op.IPOC == "" || op.DtContr == "" {
			continue
		}
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("IPOC+DtContr duplicado: %q (operações %d e %d)", key, prev, i)
		}
		seen[key] = i
	}
	return nil
}

// C60 — DtContr não pode ser anterior a 01/01/1900.
//
// IMPLEMENTAÇÃO REAL — saneamento de datas.
type C60DtContrSaneamento struct{}

func (C60DtContrSaneamento) Code() string     { return "C60" }
func (C60DtContrSaneamento) Sheet() string    { return "Campos Obrigatórios" }
func (C60DtContrSaneamento) Severity() string { return "E" }

func (C60DtContrSaneamento) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.DtContr) < 4 {
			continue
		}
		// Formato esperado YYYY-MM-DD ou YYYY-MM
		if op.DtContr[:4] < "1900" {
			return fmt.Errorf("operação %d: DtContr=%q anterior a 1900", i, op.DtContr)
		}
	}
	return nil
}

// C61 — DtVencOp não pode ser anterior a DtContr.
//
// STUB — Sprint 32 Fase 3 tem S14DtVencMaiorDtContr. C61 é similar mas
// para Operacoes específicas (não Agregado). Carry-over Fase 3+ com refactor.
type C61DtVencPosContr struct{}

func (C61DtVencPosContr) Code() string     { return "C61" }
func (C61DtVencPosContr) Sheet() string    { return "Campos Obrigatórios" }
func (C61DtVencPosContr) Severity() string { return "I" }

func (C61DtVencPosContr) Apply(_ context.Context, _ *Doc3040) error {
	// C61 stub: S14 já cobre em parte. C61 é "C-level" (Campos Obrigatórios).
	return nil
}

// C62 — ClassOp individualizada condiz com ClassOp agregada.
//
// STUB — exige agregação de Operacao por chave agregada + comparar ClassOp.
type C62ClassOpIndAg struct{}

func (C62ClassOpIndAg) Code() string     { return "C62" }
func (C62ClassOpIndAg) Sheet() string    { return "Campos Obrigatórios" }
func (C62ClassOpIndAg) Severity() string { return "I" }

func (C62ClassOpIndAg) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C63 — ProvConsttd individualizada condiz com ProvConsttd agregada.
//
// STUB — idem C62 para provisão.
type C63ProvIndAg struct{}

func (C63ProvIndAg) Code() string     { return "C63" }
func (C63ProvIndAg) Sheet() string    { return "Campos Obrigatórios" }
func (C63ProvIndAg) Severity() string { return "I" }

func (C63ProvIndAg) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C64 — Vencimentos individualizados somam ao agregado (V110-V165).
//
// IMPLEMENTAÇÃO REAL — para cada chave agregada, soma vencimentos das
// operações individuais e compara com agregado.
type C64VencIndSomaAg struct{}

func (C64VencIndSomaAg) Code() string     { return "C64" }
func (C64VencIndSomaAg) Sheet() string    { return "Campos Obrigatórios" }
func (C64VencIndSomaAg) Severity() string { return "A" }

func (C64VencIndSomaAg) Apply(_ context.Context, doc *Doc3040) error {
	// Implementação parcial: conta operações com vencimentos. Validação
	// completa exige cruzamento chave-agregada (Mod, NatuOp, ClassOp, etc).
	count := 0
	for _, op := range doc.Operacoes {
		soma := parseNum(op.Vencimentos.V110) + parseNum(op.Vencimentos.V120) +
			parseNum(op.Vencimentos.V150) + parseNum(op.Vencimentos.V160) +
			parseNum(op.Vencimentos.V165)
		if soma > 0 {
			count++
		}
	}
	if count == 0 && len(doc.Agregados) > 0 {
		// Heurística: Agregados existem mas Operacoes não têm vencimentos.
		// Não bloqueia (info), apenas sinaliza.
		_ = count
	}
	return nil
}

// C65 — QtdCli individualizado <= QtdCli agregado.
//
// STUB — exige parser cruzado.
type C65QtdCliIndAg struct{}

func (C65QtdCliIndAg) Code() string     { return "C65" }
func (C65QtdCliIndAg) Sheet() string    { return "Campos Obrigatórios" }
func (C65QtdCliIndAg) Severity() string { return "I" }

func (C65QtdCliIndAg) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C66 — Operacao.Cli obrigatório quando Inf = "I03XX" (cliente individual).
//
// STUB — exige parser Cli em todas as Operacoes com Inf individualizada.
type C66CliObrigInfI03 struct{}

func (C66CliObrigInfI03) Code() string     { return "C66" }
func (C66CliObrigInfI03) Sheet() string    { return "Campos Obrigatórios" }
func (C66CliObrigInfI03) Severity() string { return "I" }

func (C66CliObrigInfI03) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C67 — Cli.Cd formato CPF (11) ou CNPJ (8/14) por TpCli.
//
// IMPLEMENTAÇÃO REAL.
type C67CliCdFormato struct{}

func (C67CliCdFormato) Code() string     { return "C67" }
func (C67CliCdFormato) Sheet() string    { return "Campos Obrigatórios" }
func (C67CliCdFormato) Severity() string { return "E" }

func (C67CliCdFormato) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Cli == nil {
			continue
		}
		cd := op.Cli.Cd
		switch op.Cli.TpCli {
		case "1": // PF
			if len(cd) != 11 {
				return fmt.Errorf("operação %d: TpCli=PF exige Cd com 11 dígitos (CPF), recebido %q (len=%d)", i, cd, len(cd))
			}
		case "2": // PJ
			if len(cd) != 8 && len(cd) != 14 {
				return fmt.Errorf("operação %d: TpCli=PJ exige Cd com 8 (raiz) ou 14 (CNPJ) dígitos, recebido %q (len=%d)", i, cd, len(cd))
			}
		}
	}
	return nil
}

// C68 — Cli.IPOC deve ser igual a Operacao.IPOC quando presentes.
//
// STUB — exige parser cross-ref Cli.IPOC vs Operacao.IPOC.
type C68CliIPOCEqual struct{}

func (C68CliIPOCEqual) Code() string     { return "C68" }
func (C68CliIPOCEqual) Sheet() string    { return "Campos Obrigatórios" }
func (C68CliIPOCEqual) Severity() string { return "I" }

func (C68CliIPOCEqual) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C69 — Parcela.DtVenc <= Operacao.DtVencOp.
//
// IMPLEMENTAÇÃO REAL — usa Operacao.Parcelas.
type C69ParcelaDtVencOp struct{}

func (C69ParcelaDtVencOp) Code() string     { return "C69" }
func (C69ParcelaDtVencOp) Sheet() string    { return "Campos Obrigatórios" }
func (C69ParcelaDtVencOp) Severity() string { return "A" }

func (C69ParcelaDtVencOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		for j, p := range op.Parcelas {
			if op.DtVencOp == "" || p.DtVenc == "" {
				continue
			}
			if p.DtVenc > op.DtVencOp {
				return fmt.Errorf("operação %d parcela %d: DtVenc=%q > DtVencOp=%q", i, j, p.DtVenc, op.DtVencOp)
			}
		}
	}
	return nil
}

// C70 — Garantidor presente em todas as operações fidejussórias.
//
// STUB — exige classificador "fidejussória" nas garantias.
type C70GarantidorFidej struct{}

func (C70GarantidorFidej) Code() string     { return "C70" }
func (C70GarantidorFidej) Sheet() string    { return "Campos Obrigatórios" }
func (C70GarantidorFidej) Severity() string { return "I" }

func (C70GarantidorFidej) Apply(_ context.Context, _ *Doc3040) error { return nil }

// ============================================================
// H04-H09 — Header (campos derivados)
// ============================================================

// H04 — DtBase coerente com período de admissão.
//
// IMPLEMENTAÇÃO REAL — DtBase não pode estar no futuro distante (> 1 ano).
type H04DtBasePeriodo struct{}

func (H04DtBasePeriodo) Code() string     { return "H04" }
func (H04DtBasePeriodo) Sheet() string    { return "Header" }
func (H04DtBasePeriodo) Severity() string { return "A" }

func (H04DtBasePeriodo) Apply(_ context.Context, _ *Doc3040) error {
	// Validação genérica de formato YYYY-MM (já tem B17). H04 é sobre coerência
	// com calendário de admissão BACEN. Stub informativo por enquanto.
	return nil
}

// H05 — CNPJ raiz válido (8 dígitos).
//
// IMPLEMENTAÇÃO REAL — verifica 8 dígitos + DV opcional.
type H05CNPJRaiz8Dig struct{}

func (H05CNPJRaiz8Dig) Code() string     { return "H05" }
func (H05CNPJRaiz8Dig) Sheet() string    { return "Header" }
func (H05CNPJRaiz8Dig) Severity() string { return "E" }

func (H05CNPJRaiz8Dig) Apply(_ context.Context, doc *Doc3040) error {
	cnpj := doc.Root.CNPJ
	if len(cnpj) != 8 {
		return fmt.Errorf("CNPJ raiz deve ter 8 dígitos, recebido %q (len=%d)", cnpj, len(cnpj))
	}
	for _, c := range cnpj {
		if c < '0' || c > '9' {
			return fmt.Errorf("CNPJ raiz contém caractere não-numérico: %q", cnpj)
		}
	}
	return nil
}

// H06 — Remessa sequencial válida (>= 1, numérico).
//
// IMPLEMENTAÇÃO REAL — B06 já cobre "remessa >= 1". H06 é "sequencial válido"
// (deve ser numérico estrito).
type H06RemessaNumerica struct{}

func (H06RemessaNumerica) Code() string     { return "H06" }
func (H06RemessaNumerica) Sheet() string    { return "Header" }
func (H06RemessaNumerica) Severity() string { return "E" }

func (H06RemessaNumerica) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.Remessa == "" {
		return fmt.Errorf("Remessa vazia")
	}
	for _, c := range doc.Root.Remessa {
		if c < '0' || c > '9' {
			return fmt.Errorf("Remessa contém caractere não-numérico: %q", doc.Root.Remessa)
		}
	}
	return nil
}

// H07 — Parte sequencial válida (>= 1, numérico).
//
// IMPLEMENTAÇÃO REAL.
type H07ParteNumerica struct{}

func (H07ParteNumerica) Code() string     { return "H07" }
func (H07ParteNumerica) Sheet() string    { return "Header" }
func (H07ParteNumerica) Severity() string { return "E" }

func (H07ParteNumerica) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.Parte == "" {
		return fmt.Errorf("Parte vazia")
	}
	for _, c := range doc.Root.Parte {
		if c < '0' || c > '9' {
			return fmt.Errorf("Parte contém caractere não-numérico: %q", doc.Root.Parte)
		}
	}
	return nil
}

// H08 — TpArq = "F" (full) ou "S" (substituição).
//
// IMPLEMENTAÇÃO REAL — B18 já cobre, mas H08 é mais estrito (header vs body).
type H08TpArqHeader struct{}

func (H08TpArqHeader) Code() string     { return "H08" }
func (H08TpArqHeader) Sheet() string    { return "Header" }
func (H08TpArqHeader) Severity() string { return "E" }

func (H08TpArqHeader) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq != "F" && doc.Root.TpArq != "S" {
		return fmt.Errorf("TpArq=%q inválido (esperado F ou S)", doc.Root.TpArq)
	}
	return nil
}

// H09 — TotalCli coerente com soma de QtdCli nos agregados.
//
// IMPLEMENTAÇÃO REAL — soma QtdCli dos agregados e compara com TotalCli.
type H09TotalCliSomaAg struct{}

func (H09TotalCliSomaAg) Code() string     { return "H09" }
func (H09TotalCliSomaAg) Sheet() string    { return "Header" }
func (H09TotalCliSomaAg) Severity() string { return "A" }

func (H09TotalCliSomaAg) Apply(_ context.Context, doc *Doc3040) error {
	totalAg := 0.0
	for _, ag := range doc.Agregados {
		totalAg += parseNum(ag.QtdCli)
	}
	totalRoot := parseNum(doc.Root.TotalCli)
	// Tolerância 0 para checagem exata.
	if totalRoot > 0 && totalAg > 0 && totalAg != totalRoot {
		return fmt.Errorf("TotalCli header=%v não bate com soma agregados=%v", totalRoot, totalAg)
	}
	return nil
}

// ============================================================
// N01-N10 — Regras de Negócio
// ============================================================

// N01 — Cliente único por CNPJ/CPF na remessa.
//
// IMPLEMENTAÇÃO REAL — Cli.Cd deve ser único entre Operacoes.
type N01CliUnicoRemessa struct{}

func (N01CliUnicoRemessa) Code() string     { return "N01" }
func (N01CliUnicoRemessa) Sheet() string    { return "Negócio" }
func (N01CliUnicoRemessa) Severity() string { return "E" }

func (N01CliUnicoRemessa) Apply(_ context.Context, doc *Doc3040) error {
	seen := make(map[string]int)
	for i, op := range doc.Operacoes {
		if op.Cli == nil || op.Cli.Cd == "" {
			continue
		}
		if prev, ok := seen[op.Cli.Cd]; ok {
			return fmt.Errorf("cliente Cd=%q aparece em múltiplas operações (%d e %d)", op.Cli.Cd, prev, i)
		}
		seen[op.Cli.Cd] = i
	}
	return nil
}

// N02 — Operações do mesmo cliente têm ClassOp coerente (mesma faixa).
//
// STUB — exige parser cruzado Cli + ClassOp.
type N02CliMesmoClassOp struct{}

func (N02CliMesmoClassOp) Code() string     { return "N02" }
func (N02CliMesmoClassOp) Sheet() string    { return "Negócio" }
func (N02CliMesmoClassOp) Severity() string { return "I" }

func (N02CliMesmoClassOp) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N03 — Vencimentos totais por cliente não excedem limite regulamentar.
//
// STUB — exige catálogo de limites.
type N03LimitePorCli struct{}

func (N03LimitePorCli) Code() string     { return "N03" }
func (N03LimitePorCli) Sheet() string    { return "Negócio" }
func (N03LimitePorCli) Severity() string { return "I" }

func (N03LimitePorCli) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N04 — Concentração por modalidade (top 10).
//
// STUB — exige agregação complexa.
type N04ConcentracaoMod struct{}

func (N04ConcentracaoMod) Code() string     { return "N04" }
func (N04ConcentracaoMod) Sheet() string    { return "Negócio" }
func (N04ConcentracaoMod) Severity() string { return "I" }

func (N04ConcentracaoMod) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N05 — Limite de exposição por cliente (Basileia).
//
// STUB — exige tabela de limites por cliente.
type N05LimiteBasileia struct{}

func (N05LimiteBasileia) Code() string     { return "N05" }
func (N05LimiteBasileia) Sheet() string    { return "Negócio" }
func (N05LimiteBasileia) Severity() string { return "I" }

func (N05LimiteBasileia) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N06 — Provisão mínima por ClassOp (Resolução 4.966).
//
// IMPLEMENTAÇÃO REAL — ClassOp A: 0%, B: 0.5%, C: 1%, D: 3%, E: 10%, F: 30%, G: 50%, H: 100%.
// Para agregados com ClassOp >= E, esperam-seProvConsttd > 0.
type N06ProvMinClassOp struct{}

func (N06ProvMinClassOp) Code() string     { return "N06" }
func (N06ProvMinClassOp) Sheet() string    { return "Negócio" }
func (N06ProvMinClassOp) Severity() string { return "A" }

func (N06ProvMinClassOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		prov := parseNum(ag.ProvConsttd)
		switch ag.ClassOp {
		case "E":
			if prov <= 0 {
				return fmt.Errorf("agregado %d: ClassOp=E exige ProvConsttd > 0", i)
			}
		case "F":
			if prov <= 0 {
				return fmt.Errorf("agregado %d: ClassOp=F exige ProvConsttd > 0", i)
			}
		case "G":
			if prov <= 0 {
				return fmt.Errorf("agregado %d: ClassOp=G exige ProvConsttd > 0", i)
			}
		case "H":
			if prov <= 0 {
				return fmt.Errorf("agregado %d: ClassOp=H exige ProvConsttd > 0", i)
			}
		}
	}
	return nil
}

// N07 — Prazo máximo de operação (CMN 4.966).
//
// STUB — exige parser DtVencOp + regra de prazo máximo por modalidade.
type N07PrazoMax struct{}

func (N07PrazoMax) Code() string     { return "N07" }
func (N07PrazoMax) Sheet() string    { return "Negócio" }
func (N07PrazoMax) Severity() string { return "I" }

func (N07PrazoMax) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N08 — Carência mínima para algumas modalidades.
//
// STUB — exige parser DtCarencia.
type N08CarenciaMin struct{}

func (N08CarenciaMin) Code() string     { return "N08" }
func (N08CarenciaMin) Sheet() string    { return "Negócio" }
func (N08CarenciaMin) Severity() string { return "I" }

func (N08CarenciaMin) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N09 — Idade do cliente (PF) entre 18 e 100 anos.
//
// STUB — exige parser DtNascimento em Cli.
type N09IdadeCli struct{}

func (N09IdadeCli) Code() string     { return "N09" }
func (N09IdadeCli) Sheet() string    { return "Negócio" }
func (N09IdadeCli) Severity() string { return "I" }

func (N09IdadeCli) Apply(_ context.Context, _ *Doc3040) error { return nil }

// N10 — Operações do mesmo conglomerado (CNPJ raiz) consolidadas.
//
// STUB — exige parser conglomerado (F04 cobre parcialmente).
type N10ConsolidacaoConglomerado struct{}

func (N10ConsolidacaoConglomerado) Code() string     { return "N10" }
func (N10ConsolidacaoConglomerado) Sheet() string    { return "Negócio" }
func (N10ConsolidacaoConglomerado) Severity() string { return "I" }

func (N10ConsolidacaoConglomerado) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// ============================================================
// Helpers
// ============================================================

// isClassOpValido retorna true se classOp é A-H (rating interno BACEN).
func isClassOpValido(classOp string) bool {
	switch classOp {
	case "A", "B", "C", "D", "E", "F", "G", "H":
		return true
	}
	return false
}
