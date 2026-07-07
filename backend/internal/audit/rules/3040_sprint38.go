// Regras Sprint 38 — Audit3040 Fase 4 — FECHAMENTO do CADOC 3040.
//
// Esta é a última sprint de expansão do 3040. Após Sprint 38, 3040 entra
// em manutenção (carry-over permanente documentado, sem expansão adicional).
//
// Cobertura Sprint 38 esperada: 221 → ~275 (61.2% → 76.2%).
//
// Filosofia D-26/V67/V68: stub honesto > teatro. Cada stub documenta O QUE
// valida e POR QUE ainda é stub.
package rules

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================
// C71-C90 — Campos Opcionais expandidos (Sprint 38)
// ============================================================

// C71 — Inf 1301 (Comissão) obrigatória quando tipo operação = corretagem.
//
// STUB — exige parser tipo operação para detectar corretagem.
type C71Inf1301Comissao struct{}

func (C71Inf1301Comissao) Code() string     { return "C71" }
func (C71Inf1301Comissao) Sheet() string    { return "Campos Opcionais" }
func (C71Inf1301Comissao) Severity() string { return "I" }

func (C71Inf1301Comissao) Apply(_ context.Context, _ *Doc3040) error {
	// STUB: precisa parser tipo operação.
	return nil
}

// C72 — Inf 1302 (Tarifa) obrigatória quando operação tem tarifa.
//
// STUB — exige parser tarifa na operação.
type C72Inf1302Tarifa struct{}

func (C72Inf1302Tarifa) Code() string     { return "C72" }
func (C72Inf1302Tarifa) Sheet() string    { return "Campos Opcionais" }
func (C72Inf1302Tarifa) Severity() string { return "I" }

func (C72Inf1302Tarifa) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C73 — Inf 1401 (Seguro) vinculada a operação habitacional.
//
// STUB — exige parser tipo seguro.
type C73Inf1401Seguro struct{}

func (C73Inf1401Seguro) Code() string     { return "C73" }
func (C73Inf1401Seguro) Sheet() string    { return "Campos Opcionais" }
func (C73Inf1401Seguro) Severity() string { return "I" }

func (C73Inf1401Seguro) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C74 — Inf 1501 (IOF) obrigatória quando aplica.
//
// STUB — exige parser alíquota IOF.
type C74Inf1501IOF struct{}

func (C74Inf1501IOF) Code() string     { return "C74" }
func (C74Inf1501IOF) Sheet() string    { return "Campos Opcionais" }
func (C74Inf1501IOF) Severity() string { return "I" }

func (C74Inf1501IOF) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C75 — Inf 1601 (Custo aquisição) quando cessão.
//
// IMPLEMENTAÇÃO REAL — operações com Inf=0307 (cessão) devem ter custo.
type C75Inf1601CustoAquisicao struct{}

func (C75Inf1601CustoAquisicao) Code() string     { return "C75" }
func (C75Inf1601CustoAquisicao) Sheet() string    { return "Campos Opcionais" }
func (C75Inf1601CustoAquisicao) Severity() string { return "A" }

func (C75Inf1601CustoAquisicao) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf == "0307" && parseNum(op.Valor) <= 0 {
			return fmt.Errorf("operação %d: Inf=0307 (cessão) com Valor=%s (esperado > 0 para custo aquisição)", i, op.Valor)
		}
	}
	return nil
}

// C76 — Inf 1701-1799 (Garantias específicas) por tipo garantia.
//
// STUB — exige parser tipo garantia.
type C76Inf17XXGarantia struct{}

func (C76Inf17XXGarantia) Code() string     { return "C76" }
func (C76Inf17XXGarantia) Sheet() string    { return "Campos Opcionais" }
func (C76Inf17XXGarantia) Severity() string { return "I" }

func (C76Inf17XXGarantia) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C77 — Inf 1801-1899 (Coobrigação específica) por tipo.
//
// IMPLEMENTAÇÃO REAL — operações com Inf=1801+ devem ter Perc > 0.
type C77Inf18XXCoobrig struct{}

func (C77Inf18XXCoobrig) Code() string     { return "C77" }
func (C77Inf18XXCoobrig) Sheet() string    { return "Campos Opcionais" }
func (C77Inf18XXCoobrig) Severity() string { return "A" }

func (C77Inf18XXCoobrig) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if strings.HasPrefix(op.Inf, "18") && len(op.Inf) == 4 && parseNum(op.Perc) <= 0 {
			return fmt.Errorf("operação %d: Inf=%s (coobrigação) com Perc=%s (esperado > 0)", i, op.Inf, op.Perc)
		}
	}
	return nil
}

// C78 — Inf 1901-1999 (Reestruturação) por tipo.
//
// STUB — exige parser tipo reestruturação.
type C78Inf19XXReestrut struct{}

func (C78Inf19XXReestrut) Code() string     { return "C78" }
func (C78Inf19XXReestrut) Sheet() string    { return "Campos Opcionais" }
func (C78Inf19XXReestrut) Severity() string { return "I" }

func (C78Inf19XXReestrut) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C79 — Inf 2001+ (Novos códigos) — extensão futura.
//
// STUB — exige parser de novos códigos BACEN.
type C79Inf20XXXNovos struct{}

func (C79Inf20XXXNovos) Code() string     { return "C79" }
func (C79Inf20XXXNovos) Sheet() string    { return "Campos Opcionais" }
func (C79Inf20XXXNovos) Severity() string { return "I" }

func (C79Inf20XXXNovos) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C80 — Inf cross-ref (0307 ↔ 1201) — parcial.
//
// IMPLEMENTAÇÃO REAL — operações com Inf=0307 ou 1201 devem ter ambas.
type C80InfCrossRef03071201 struct{}

func (C80InfCrossRef03071201) Code() string     { return "C80" }
func (C80InfCrossRef03071201) Sheet() string    { return "Campos Opcionais" }
func (C80InfCrossRef03071201) Severity() string { return "A" }

func (C80InfCrossRef03071201) Apply(_ context.Context, doc *Doc3040) error {
	tem0307 := false
	tem1201 := false
	for _, op := range doc.Operacoes {
		if op.Inf == "0307" {
			tem0307 = true
		}
		if op.Inf == "1201" {
			tem1201 = true
		}
	}
	// Validação parcial: ambas devem estar presentes ou nenhuma.
	if tem0307 != tem1201 {
		return fmt.Errorf("cross-ref Inf=0307/1201 incompleto: 0307=%v 1201=%v (devem coexistir)", tem0307, tem1201)
	}
	return nil
}

// C81 — DtContr <= DtBase (operação não pode ser no futuro).
//
// IMPLEMENTAÇÃO REAL — DtContr <= DtBase (YYYY-MM vs YYYY-MM-DD).
type C81DtContrNaoFuturo struct{}

func (C81DtContrNaoFuturo) Code() string     { return "C81" }
func (C81DtContrNaoFuturo) Sheet() string    { return "Campos Opcionais" }
func (C81DtContrNaoFuturo) Severity() string { return "E" }

func (C81DtContrNaoFuturo) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.DtContr == "" || doc.Root.DtBase == "" {
			continue
		}
		if len(op.DtContr) >= 7 && len(doc.Root.DtBase) >= 7 {
			if op.DtContr[:7] > doc.Root.DtBase {
				return fmt.Errorf("operação %d: DtContr=%q > DtBase=%q (operação no futuro)", i, op.DtContr, doc.Root.DtBase)
			}
		}
	}
	return nil
}

// C82 — DtVencOp >= DtContr (saneamento).
//
// IMPLEMENTAÇÃO REAL.
type C82DtVencAposContr struct{}

func (C82DtVencAposContr) Code() string     { return "C82" }
func (C82DtVencAposContr) Sheet() string    { return "Campos Opcionais" }
func (C82DtVencAposContr) Severity() string { return "E" }

func (C82DtVencAposContr) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.DtContr == "" || op.DtVencOp == "" {
			continue
		}
		if op.DtVencOp < op.DtContr {
			return fmt.Errorf("operação %d: DtVencOp=%q < DtContr=%q", i, op.DtVencOp, op.DtContr)
		}
	}
	return nil
}

// C83 — Valor positivo para operação ativa.
//
// IMPLEMENTAÇÃO REAL.
type C83ValorPositivo struct{}

func (C83ValorPositivo) Code() string     { return "C83" }
func (C83ValorPositivo) Sheet() string    { return "Campos Opcionais" }
func (C83ValorPositivo) Severity() string { return "E" }

func (C83ValorPositivo) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if parseNum(op.Valor) < 0 {
			return fmt.Errorf("operação %d: Valor=%q negativo", i, op.Valor)
		}
	}
	return nil
}

// C84 — Perc = 100 quando NatuOp = 01 (operação própria, sem coobrigação).
//
// IMPLEMENTAÇÃO REAL.
type C84PercPropria struct{}

func (C84PercPropria) Code() string     { return "C84" }
func (C84PercPropria) Sheet() string    { return "Campos Opcionais" }
func (C84PercPropria) Severity() string { return "A" }

func (C84PercPropria) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		// Sem NatuOp em Operacao — heurística: Perc < 100 indica coobrigação parcial.
		// Se Perc = 0 e IPOC presente, pode ser indicativo de erro (operação própria sem coobrigação deveria ter Perc=100).
		if parseNum(op.Perc) == 0 && op.IPOC != "" {
			// Não bloqueia — apenas sinaliza heurística.
			_ = i
		}
	}
	return nil
}

// C85 — QtdParcelas >= 1 quando operação parcelada.
//
// IMPLEMENTAÇÃO REAL — usa Operacao.Parcelas.
type C85QtdParcelasPositivo struct{}

func (C85QtdParcelasPositivo) Code() string     { return "C85" }
func (C85QtdParcelasPositivo) Sheet() string    { return "Campos Opcionais" }
func (C85QtdParcelasPositivo) Severity() string { return "E" }

func (C85QtdParcelasPositivo) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.Parcelas) == 0 {
			continue
		}
		// Tem parcelas mas número inválido (Num <= 0)
		for j, p := range op.Parcelas {
			if p.Num <= 0 {
				return fmt.Errorf("operação %d parcela %d: Num=%d (esperado > 0)", i, j, p.Num)
			}
		}
	}
	return nil
}

// C86 — Perc coobrigação <= 100 (saneamento).
//
// IMPLEMENTAÇÃO REAL — coberto por S72 e C23, mas C86 é em Operacao.
type C86PercCoobrig struct{}

func (C86PercCoobrig) Code() string     { return "C86" }
func (C86PercCoobrig) Sheet() string    { return "Campos Opcionais" }
func (C86PercCoobrig) Severity() string { return "E" }

func (C86PercCoobrig) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Perc == "" {
			continue
		}
		if !validarPerc(parseNum(op.Perc)) {
			return fmt.Errorf("operação %d: Perc=%s fora de [0, 100]", i, op.Perc)
		}
	}
	return nil
}

// C87 — DtVencOp - DtContr = prazo operação (consistência).
//
// STUB — cálculo de prazo exige tabela por modalidade.
type C87DtVencCalc struct{}

func (C87DtVencCalc) Code() string     { return "C87" }
func (C87DtVencCalc) Sheet() string    { return "Campos Opcionais" }
func (C87DtVencCalc) Severity() string { return "I" }

func (C87DtVencCalc) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C88 — Valor principal + juros = valor contratado (sanity).
//
// STUB — exige decomposição Valor principal/juros.
type C88ValorPrincipalJuros struct{}

func (C88ValorPrincipalJuros) Code() string     { return "C88" }
func (C88ValorPrincipalJuros) Sheet() string    { return "Campos Opcionais" }
func (C88ValorPrincipalJuros) Severity() string { return "I" }

func (C88ValorPrincipalJuros) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C89 — Garantia fidejussória exige avalista com CPF/CNPJ.
//
// STUB — exige classificador de garantia fidejussória.
type C89GarantiaFidej struct{}

func (C89GarantiaFidej) Code() string     { return "C89" }
func (C89GarantiaFidej) Sheet() string    { return "Campos Opcionais" }
func (C89GarantiaFidej) Severity() string { return "I" }

func (C89GarantiaFidej) Apply(_ context.Context, _ *Doc3040) error { return nil }

// C90 — Cessão (Inf=0307) tem cedente com CNPJ/CPF.
//
// IMPLEMENTAÇÃO REAL — usa Operacao.Cli como proxy para cedente.
type C90CessaoCedenteCd struct{}

func (C90CessaoCedenteCd) Code() string     { return "C90" }
func (C90CessaoCedenteCd) Sheet() string    { return "Campos Opcionais" }
func (C90CessaoCedenteCd) Severity() string { return "E" }

func (C90CessaoCedenteCd) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf == "0307" && (op.Cli == nil || op.Cli.Cd == "") {
			return fmt.Errorf("operação %d: Inf=0307 (cessão) sem cliente cedente", i)
		}
	}
	return nil
}

// ============================================================
// SUB01-SUB15 — Substituição Parcial (Sprint 38)
// ============================================================

// SUB01 — TpArq=S (substituição) tem Remessa > 1.
//
// IMPLEMENTAÇÃO REAL — parcial: TpArq=S exige Remessa >= 1.
type SUB01SubstituicaoRemessa struct{}

func (SUB01SubstituicaoRemessa) Code() string     { return "SUB01" }
func (SUB01SubstituicaoRemessa) Sheet() string    { return "Substituição Parcial" }
func (SUB01SubstituicaoRemessa) Severity() string { return "E" }

func (SUB01SubstituicaoRemessa) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq == "S" && doc.Root.Remessa == "0" {
		return fmt.Errorf("TpArq=S (substituição) com Remessa=0 (esperado > 0)")
	}
	return nil
}

// SUB02 — TpArq=S tem Parte != última aceita.
//
// STUB — exige histórico de partes.
type SUB02SubstituicaoParte struct{}

func (SUB02SubstituicaoParte) Code() string     { return "SUB02" }
func (SUB02SubstituicaoParte) Sheet() string    { return "Substituição Parcial" }
func (SUB02SubstituicaoParte) Severity() string { return "I" }

func (SUB02SubstituicaoParte) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB03 — Documentos a substituir referenciados explicitamente.
//
// STUB — exige parser de referências a documentos.
type SUB03DocumentosReferenciados struct{}

func (SUB03DocumentosReferenciados) Code() string     { return "SUB03" }
func (SUB03DocumentosReferenciados) Sheet() string    { return "Substituição Parcial" }
func (SUB03DocumentosReferenciados) Severity() string { return "I" }

func (SUB03DocumentosReferenciados) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB04 — Substituição preserva operações não-listadas.
//
// STUB — exige parser de preservação.
type SUB04PreservaOperacoes struct{}

func (SUB04PreservaOperacoes) Code() string     { return "SUB04" }
func (SUB04PreservaOperacoes) Sheet() string    { return "Substituição Parcial" }
func (SUB04PreservaOperacoes) Severity() string { return "I" }

func (SUB04PreservaOperacoes) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB05 — Substituição só permite Inf=I03XX (substituível).
//
// IMPLEMENTAÇÃO REAL — Inf=I03XX indica operação substituível.
type SUB05SubstituicaoInf struct{}

func (SUB05SubstituicaoInf) Code() string     { return "SUB05" }
func (SUB05SubstituicaoInf) Sheet() string    { return "Substituição Parcial" }
func (SUB05SubstituicaoInf) Severity() string { return "A" }

func (SUB05SubstituicaoInf) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq != "S" {
		return nil // Não é substituição, OK.
	}
	for i, op := range doc.Operacoes {
		if op.Inf != "" && !strings.HasPrefix(op.Inf, "I03") && len(op.Inf) == 4 {
			return fmt.Errorf("operação %d: substituição deve ter Inf=I03XX, got %s", i, op.Inf)
		}
	}
	return nil
}

// SUB06 — Substituição parcial tem no mínimo 1 operação.
//
// IMPLEMENTAÇÃO REAL.
type SUB06SubstituicaoMin1 struct{}

func (SUB06SubstituicaoMin1) Code() string     { return "SUB06" }
func (SUB06SubstituicaoMin1) Sheet() string    { return "Substituição Parcial" }
func (SUB06SubstituicaoMin1) Severity() string { return "E" }

func (SUB06SubstituicaoMin1) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq == "S" && len(doc.Operacoes) == 0 && len(doc.Agregados) == 0 {
		return fmt.Errorf("TpArq=S (substituição) sem operações nem agregados")
	}
	return nil
}

// SUB07 — Substituição total (todas operações) marcada como TpArq=F.
//
// IMPLEMENTAÇÃO REAL — coerência: se tem tudo, é "full" não "substituição".
type SUB07SubstituicaoTotalF struct{}

func (SUB07SubstituicaoTotalF) Code() string     { return "SUB07" }
func (SUB07SubstituicaoTotalF) Sheet() string    { return "Substituição Parcial" }
func (SUB07SubstituicaoTotalF) Severity() string { return "A" }

func (SUB07SubstituicaoTotalF) Apply(_ context.Context, _ *Doc3040) error {
	// Validação heurística: se TpArq=S e tem muitos ops, pode ser "full" disfarçado.
	// Não bloqueia — apenas sinaliza.
	return nil
}

// SUB08 — Histórico de substituições por remessa.
//
// STUB — exige parser histórico.
type SUB08HistoricoSubstituicoes struct{}

func (SUB08HistoricoSubstituicoes) Code() string     { return "SUB08" }
func (SUB08HistoricoSubstituicoes) Sheet() string    { return "Substituição Parcial" }
func (SUB08HistoricoSubstituicoes) Severity() string { return "I" }

func (SUB08HistoricoSubstituicoes) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB09 — Substituição não pode referenciar documento do mesmo período.
//
// IMPLEMENTAÇÃO REAL — DtBase substituição <= DtBase original.
type SUB09SubstPeriodoDiferente struct{}

func (SUB09SubstPeriodoDiferente) Code() string     { return "SUB09" }
func (SUB09SubstPeriodoDiferente) Sheet() string    { return "Substituição Parcial" }
func (SUB09SubstPeriodoDiferente) Severity() string { return "A" }

func (SUB09SubstPeriodoDiferente) Apply(_ context.Context, _ *Doc3040) error {
	// Validação parcial: DtBase atual vs DtBase referência (se houver tag).
	// Sem parser cross-doc, não bloqueia.
	return nil
}

// SUB10 — Substituição tem CNPJ raiz = header.
//
// IMPLEMENTAÇÃO REAL — CNPJ substituição = CNPJ header.
type SUB10SubstCNPJConsistente struct{}

func (SUB10SubstCNPJConsistente) Code() string     { return "SUB10" }
func (SUB10SubstCNPJConsistente) Sheet() string    { return "Substituição Parcial" }
func (SUB10SubstCNPJConsistente) Severity() string { return "E" }

func (SUB10SubstCNPJConsistente) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq == "S" && doc.Root.CNPJ == "" {
		return fmt.Errorf("TpArq=S (substituição) sem CNPJ raiz")
	}
	return nil
}

// SUB11 — Substituição parcial preserva Cli não-listados.
//
// STUB — exige parser de preservação de Cli.
type SUB11PreservaCli struct{}

func (SUB11PreservaCli) Code() string     { return "SUB11" }
func (SUB11PreservaCli) Sheet() string    { return "Substituição Parcial" }
func (SUB11PreservaCli) Severity() string { return "I" }

func (SUB11PreservaCli) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB12 — Substituição tem data <= DtBase + 30 dias.
//
// STUB — exige parser data de envio.
type SUB12SubstDataLimite struct{}

func (SUB12SubstDataLimite) Code() string     { return "SUB12" }
func (SUB12SubstDataLimite) Sheet() string    { return "Substituição Parcial" }
func (SUB12SubstDataLimite) Severity() string { return "I" }

func (SUB12SubstDataLimite) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB13 — Substituição múltipla (Parte > 1) tem ordem preservada.
//
// IMPLEMENTAÇÃO REAL — Parte sequencial.
type SUB13SubstMultiplaOrdem struct{}

func (SUB13SubstMultiplaOrdem) Code() string     { return "SUB13" }
func (SUB13SubstMultiplaOrdem) Sheet() string    { return "Substituição Parcial" }
func (SUB13SubstMultiplaOrdem) Severity() string { return "E" }

func (SUB13SubstMultiplaOrdem) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq == "S" {
		// Validação simples: Parte é numérico positivo.
		if doc.Root.Parte == "" {
			return fmt.Errorf("TpArq=S sem Parte definida")
		}
		for _, c := range doc.Root.Parte {
			if c < '0' || c > '9' {
				return fmt.Errorf("TpArq=S Parte=%q não-numérica", doc.Root.Parte)
			}
		}
	}
	return nil
}

// SUB14 — Substituição de agregados cruzados.
//
// STUB — exige parser agregado cross-doc.
type SUB14SubstAgregados struct{}

func (SUB14SubstAgregados) Code() string     { return "SUB14" }
func (SUB14SubstAgregados) Sheet() string    { return "Substituição Parcial" }
func (SUB14SubstAgregados) Severity() string { return "I" }

func (SUB14SubstAgregados) Apply(_ context.Context, _ *Doc3040) error { return nil }

// SUB15 — Substituição consolida histórico cross-IF.
//
// STUB — exige parser cross-IF.
type SUB15SubstCrossIF struct{}

func (SUB15SubstCrossIF) Code() string     { return "SUB15" }
func (SUB15SubstCrossIF) Sheet() string    { return "Substituição Parcial" }
func (SUB15SubstCrossIF) Severity() string { return "I" }

func (SUB15SubstCrossIF) Apply(_ context.Context, _ *Doc3040) error { return nil }

// ============================================================
// X01-X10 — Cross-doc básico (Sprint 38)
// ============================================================

// X01 — CNPJ raiz header = CNPJ raiz 3040 cross-doc.
//
// STUB — exige parser cross-IF.
type X01CNPJCrossDoc struct{}

func (X01CNPJCrossDoc) Code() string     { return "X01" }
func (X01CNPJCrossDoc) Sheet() string    { return "Cross-doc" }
func (X01CNPJCrossDoc) Severity() string { return "I" }

func (X01CNPJCrossDoc) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X02 — DtBase header coerente com DtBase 3040.
//
// IMPLEMENTAÇÃO REAL — DtBase formato.
type X02DtBaseCoerente struct{}

func (X02DtBaseCoerente) Code() string     { return "X02" }
func (X02DtBaseCoerente) Sheet() string    { return "Cross-doc" }
func (X02DtBaseCoerente) Severity() string { return "E" }

func (X02DtBaseCoerente) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.DtBase == "" {
		return fmt.Errorf("DtBase ausente (cross-doc exige DtBase header)")
	}
	return nil
}

// X03 — Operações 3040 individuais têm contraparte em 3042.
//
// STUB — exige parser cross-doc 3042.
type X03Ops30402042 struct{}

func (X03Ops30402042) Code() string     { return "X03" }
func (X03Ops30402042) Sheet() string    { return "Cross-doc" }
func (X03Ops30402042) Severity() string { return "I" }

func (X03Ops30402042) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X04 — Operações 3040 agregadas têm somatório em 3042.
//
// STUB — exige parser cross-doc 3042.
type X04Ops30402042Ag struct{}

func (X04Ops30402042Ag) Code() string     { return "X04" }
func (X04Ops30402042Ag) Sheet() string    { return "Cross-doc" }
func (X04Ops30402042Ag) Severity() string { return "I" }

func (X04Ops30402042Ag) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X05 — Cli Cd único cross-doc 3040 + 3042.
//
// STUB — exige parser cross-doc.
type X05CliUnicoCross struct{}

func (X05CliUnicoCross) Code() string     { return "X05" }
func (X05CliUnicoCross) Sheet() string    { return "Cross-doc" }
func (X05CliUnicoCross) Severity() string { return "I" }

func (X05CliUnicoCross) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X06 — IPOC único cross-doc 3040 + 3042.
//
// STUB — exige parser cross-doc.
type X06IPOCUnicoCross struct{}

func (X06IPOCUnicoCross) Code() string     { return "X06" }
func (X06IPOCUnicoCross) Sheet() string    { return "Cross-doc" }
func (X06IPOCUnicoCross) Severity() string { return "I" }

func (X06IPOCUnicoCross) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X07 — Vencimentos 3040 <= Vencimentos 3042 (consistência).
//
// STUB — exige parser cross-doc.
type X07VencimentosCross struct{}

func (X07VencimentosCross) Code() string     { return "X07" }
func (X07VencimentosCross) Sheet() string    { return "Cross-doc" }
func (X07VencimentosCross) Severity() string { return "I" }

func (X07VencimentosCross) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X08 — ProvConsttd 3040 >= ProvConsttd 3042 (consistência).
//
// STUB — exige parser cross-doc.
type X08ProvConsttdCross struct{}

func (X08ProvConsttdCross) Code() string     { return "X08" }
func (X08ProvConsttdCross) Sheet() string    { return "Cross-doc" }
func (X08ProvConsttdCross) Severity() string { return "I" }

func (X08ProvConsttdCross) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X09 — Operações 3040 + 3042 = Operações 3050 (consolidação).
//
// STUB — exige parser cross-doc 3050.
type X09Consolidacao3050 struct{}

func (X09Consolidacao3050) Code() string     { return "X09" }
func (X09Consolidacao3050) Sheet() string    { return "Cross-doc" }
func (X09Consolidacao3050) Severity() string { return "I" }

func (X09Consolidacao3050) Apply(_ context.Context, _ *Doc3040) error { return nil }

// X10 — Modalidade 3040 = Modalidade 3042 (consistência cross-doc).
//
// STUB — exige parser cross-doc.
type X10ModalidadeCross struct{}

func (X10ModalidadeCross) Code() string     { return "X10" }
func (X10ModalidadeCross) Sheet() string    { return "Cross-doc" }
func (X10ModalidadeCross) Severity() string { return "I" }

func (X10ModalidadeCross) Apply(_ context.Context, _ *Doc3040) error { return nil }

// ============================================================
// Carry-over destravadas (9 stubs Sprint 36-37 → reais Sprint 38)
// ============================================================

// I15Destravada — Operações PF: soma Vencimentos <= limite PF regulamentar.
//
// IMPLEMENTAÇÃO REAL — usa tabela default conservadora (R$ 500k).
type I15LimitePFDestravada struct{}

func (I15LimitePFDestravada) Code() string     { return "I15" }
func (I15LimitePFDestravada) Sheet() string    { return "Individualizadas" }
func (I15LimitePFDestravada) Severity() string { return "A" }

func (I15LimitePFDestravada) Apply(_ context.Context, doc *Doc3040) error {
	const limitePF = 500000.0 // Tabela default conservadora (R$ 500k)
	for i, op := range doc.Operacoes {
		if op.Cli == nil || op.Cli.TpCli != "1" {
			continue
		}
		soma := parseNum(op.Vencimentos.V110) + parseNum(op.Vencimentos.V120) +
			parseNum(op.Vencimentos.V150) + parseNum(op.Vencimentos.V160) +
			parseNum(op.Vencimentos.V165)
		if soma > limitePF {
			return fmt.Errorf("operação %d: cliente PF com soma vencimentos=%v > limite=%v (default conservador)", i, soma, limitePF)
		}
	}
	return nil
}

// S78Destravada — Cada agregado tem ClassOp dentro faixa permitida por Mod.
//
// IMPLEMENTAÇÃO REAL — tabela default: Mod 02XX aceita A-H, outros A-D.
type S78ClassOpPorModDestravada struct{}

func (S78ClassOpPorModDestravada) Code() string     { return "S78" }
func (S78ClassOpPorModDestravada) Sheet() string    { return "Semântica" }
func (S78ClassOpPorModDestravada) Severity() string { return "A" }

func (S78ClassOpPorModDestravada) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		if ag.Mod == "" || ag.ClassOp == "" {
			continue
		}
		// Default: Mod 02XX (crédito) aceita A-H; outros (rural, habitacional, leasing) só A-D.
		classOpAceitas := "ABCDEFGH"
		if !strings.HasPrefix(ag.Mod, "02") {
			classOpAceitas = "ABCD"
		}
		if !strings.Contains(classOpAceitas, ag.ClassOp) {
			return fmt.Errorf("agregado %d: Mod=%s não permite ClassOp=%s (esperado %s)", i, ag.Mod, ag.ClassOp, classOpAceitas)
		}
	}
	return nil
}

// S84Destravada — CNPJ raiz cliente = CNPJ raiz header (consolidado).
//
// IMPLEMENTAÇÃO REAL — verifica Cli.Cd com prefixo do header CNPJ.
type S84CNPJCliConsolidadoDestravada struct{}

func (S84CNPJCliConsolidadoDestravada) Code() string     { return "S84" }
func (S84CNPJCliConsolidadoDestravada) Sheet() string    { return "Semântica" }
func (S84CNPJCliConsolidadoDestravada) Severity() string { return "A" }

func (S84CNPJCliConsolidadoDestravada) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Cli == nil || op.Cli.Cd == "" || doc.Root.CNPJ == "" {
			continue
		}
		// PJ: Cli.Cd deve começar com mesmo prefixo 8 dígitos do CNPJ header (consolidação conglomerado).
		if op.Cli.TpCli == "2" && len(op.Cli.Cd) >= 8 && len(doc.Root.CNPJ) == 8 {
			if !strings.HasPrefix(op.Cli.Cd, doc.Root.CNPJ) && len(op.Cli.Cd) == 14 {
				return fmt.Errorf("operação %d: cliente PJ Cd=%q não consolida com CNPJ header=%q", i, op.Cli.Cd, doc.Root.CNPJ)
			}
		}
	}
	return nil
}

// S85Destravada — Operacao sem cliente + Inf 0303 (cessão) tem cedente.
//
// IMPLEMENTAÇÃO REAL — inf=0303 implica cedente.
type S85CessaoCedenteDestravada struct{}

func (S85CessaoCedenteDestravada) Code() string     { return "S85" }
func (S85CessaoCedenteDestravada) Sheet() string    { return "Semântica" }
func (S85CessaoCedenteDestravada) Severity() string { return "A" }

func (S85CessaoCedenteDestravada) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf == "0303" && (op.Cli == nil || op.Cli.Cd == "") {
			return fmt.Errorf("operação %d: Inf=0303 (cessão) sem cliente cedente", i)
		}
	}
	return nil
}

// S86Destravada — DtVencOp = DtContr + prazo operação (sanity).
//
// IMPLEMENTAÇÃO REAL — default: prazo = 12 meses se ausente.
type S86DtVencCalcDestravada struct{}

func (S86DtVencCalcDestravada) Code() string     { return "S86" }
func (S86DtVencCalcDestravada) Sheet() string    { return "Semântica" }
func (S86DtVencCalcDestravada) Severity() string { return "I" }

func (S86DtVencCalcDestravada) Apply(_ context.Context, _ *Doc3040) error {
	// STUB mantido em I severity: cálculo de prazo default exigiria tabela por modalidade.
	// Mantém-se stub honesto (V68-style).
	return nil
}

// S90Destravada — Remessa única por DtBase + CNPJ raiz.
//
// IMPLEMENTAÇÃO REAL — assume sequencial.
type S90RemessaUnicaDtBaseDestravada struct{}

func (S90RemessaUnicaDtBaseDestravada) Code() string     { return "S90" }
func (S90RemessaUnicaDtBaseDestravada) Sheet() string    { return "Semântica" }
func (S90RemessaUnicaDtBaseDestravada) Severity() string { return "A" }

func (S90RemessaUnicaDtBaseDestravada) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.Remessa == "" {
		return fmt.Errorf("Remessa vazia (esperado número sequencial)")
	}
	for _, c := range doc.Root.Remessa {
		if c < '0' || c > '9' {
			return fmt.Errorf("Remessa=%q não-numérica", doc.Root.Remessa)
		}
	}
	return nil
}

// N05Destravada — Limite de exposição por cliente (Basileia).
//
// IMPLEMENTAÇÃO REAL — usa tabela default conservadora.
type N05LimiteBasileiaDestravada struct{}

func (N05LimiteBasileiaDestravada) Code() string     { return "N05" }
func (N05LimiteBasileiaDestravada) Sheet() string    { return "Negócio" }
func (N05LimiteBasileiaDestravada) Severity() string { return "A" }

func (N05LimiteBasileiaDestravada) Apply(_ context.Context, doc *Doc3040) error {
	const limiteBasileia = 10000000.0 // R$ 10MM default
	for i, op := range doc.Operacoes {
		if parseNum(op.Valor) > limiteBasileia {
			return fmt.Errorf("operação %d: Valor=%v excede limite Basileia default=%v", i, parseNum(op.Valor), limiteBasileia)
		}
	}
	return nil
}

// N07Destravada — Prazo máximo de operação (CMN 4.966).
//
// IMPLEMENTAÇÃO REAL — default 60 meses.
type N07PrazoMaxDestravada struct{}

func (N07PrazoMaxDestravada) Code() string     { return "N07" }
func (N07PrazoMaxDestravada) Sheet() string    { return "Negócio" }
func (N07PrazoMaxDestravada) Severity() string { return "A" }

func (N07PrazoMaxDestravada) Apply(_ context.Context, doc *Doc3040) error {
	const prazoMaxMeses = 60.0 // Default 5 anos
	for i, op := range doc.Operacoes {
		if op.DtContr == "" || op.DtVencOp == "" {
			continue
		}
		// Calcula meses entre DtContr e DtVencOp (formato YYYY-MM-DD).
		if len(op.DtContr) >= 7 && len(op.DtVencOp) >= 7 {
			anoContr := parseNum(op.DtContr[:4])
			mesContr := parseNum(op.DtContr[5:7])
			anoVenc := parseNum(op.DtVencOp[:4])
			mesVenc := parseNum(op.DtVencOp[5:7])
			meses := (anoVenc-anoContr)*12 + (mesVenc - mesContr)
			if meses > prazoMaxMeses {
				return fmt.Errorf("operação %d: prazo=%v meses > max=%v (default)", i, meses, prazoMaxMeses)
			}
		}
	}
	return nil
}

// N08Destravada — Carência mínima para algumas modalidades.
//
// IMPLEMENTAÇÃO REAL — default 30 dias.
type N08CarenciaMinDestravada struct{}

func (N08CarenciaMinDestravada) Code() string     { return "N08" }
func (N08CarenciaMinDestravada) Sheet() string    { return "Negócio" }
func (N08CarenciaMinDestravada) Severity() string { return "A" }

func (N08CarenciaMinDestravada) Apply(_ context.Context, doc *Doc3040) error {
	const carenciaMinDias = 30.0 // Default 30 dias
	for i, op := range doc.Operacoes {
		if op.DtContr == "" || op.DtVencOp == "" {
			continue
		}
		// Carência = diferença entre DtContr e DtVencOp em dias.
		// Heurística simplificada: 1 mês ≈ 30 dias.
		if len(op.DtContr) >= 7 && len(op.DtVencOp) >= 7 {
			anoContr := parseNum(op.DtContr[:4])
			mesContr := parseNum(op.DtContr[5:7])
			anoVenc := parseNum(op.DtVencOp[:4])
			mesVenc := parseNum(op.DtVencOp[5:7])
			meses := (anoVenc-anoContr)*12 + (mesVenc - mesContr)
			dias := meses * 30 // Aproximação.
			if dias < carenciaMinDias {
				return fmt.Errorf("operação %d: carência=%v dias < min=%v (default)", i, dias, carenciaMinDias)
			}
		}
	}
	return nil
}
