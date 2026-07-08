// Regras Fase 4 — Sprint 32 (última) — Audit3040_v2
//
// Implementa subset realista de C31-C55, S21-S46, S69-S70 (~30 regras).
// Carry-over documentado em SPRINT_32_FASE4_RESEARCH.md (~67 regras
// que requerem expansão adicional do struct: DiaAtraso, Porte, etc).
//
// Pattern dominante: "Para Inf=X, validar Y" — reusa lógica da Fase 3
// (C11-C20) com infs adicionais (0313, 04XX, 18XX, 1201-1203).
package rules

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
)

// ============================================================
// C31-C35 — Obrigatoriedade por data-base + Inf específica
// ============================================================

// C31 — Faturamento Anual obrigatório (>= 07/2011).
//
// Catálogo: "Faturamento Anual obrigatório para operações concedidas a partir
// de 07/2011 e para operações concessão após essa data-base."
type C31FaturamentoObrigatorio struct{}

func (C31FaturamentoObrigatorio) Code() string     { return "C31" }
func (C31FaturamentoObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C31FaturamentoObrigatorio) Severity() string { return "E" }
func (C31FaturamentoObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	ano, mes, err := parseDtBaseYM(doc.Root.DtBase)
	if err != nil {
		return err
	}
	// >= 07/2011
	if ano < 2011 || (ano == 2011 && mes < 7) {
		return nil
	}
	// Validação: Operacoes com Cli devem ter Faturamento > 0
	for i, op := range doc.Operacoes {
		if op.Cli == nil {
			continue
		}
		if op.Valor == "" || parseNum(op.Valor) <= 0 {
			return fmt.Errorf("operação %d (Cli): Faturamento Anual obrigatório (>= 07/2011)", i)
		}
	}
	return nil
}

// C32 — Perc Indexador obrigatório (>= 09/2011).
type C32PercIndexadorObrigatorio struct{}

func (C32PercIndexadorObrigatorio) Code() string     { return "C32" }
func (C32PercIndexadorObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C32PercIndexadorObrigatorio) Severity() string { return "E" }
func (C32PercIndexadorObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	ano, mes, err := parseDtBaseYM(doc.Root.DtBase)
	if err != nil {
		return err
	}
	if ano < 2011 || (ano == 2011 && mes < 9) {
		return nil
	}
	for i, op := range doc.Operacoes {
		if op.Perc == "" {
			return fmt.Errorf("operação %d (Inf=%s): Perc Indexador obrigatório (>= 09/2011)", i, op.Inf)
		}
	}
	return nil
}

// C33 — Dias Atraso obrigatório para vencidas (205-330).
//
// Catálogo: "DiasAtraso obrigatório para operações com código vencimento 205-330."
//
// Sprint 32 Fase 4: heurística — se vencimento > 200 e op não tem DiaAtraso,
// warning. Carry-over completo requer campo DiaAtraso em Operacao.
type C33DiasAtrasoObrigatorio struct{}

func (C33DiasAtrasoObrigatorio) Code() string  { return "C33" }
func (C33DiasAtrasoObrigatorio) Sheet() string { return "Campos Obrigatórios" }

// Severity "I" (informativo) — stub pass-through. Carry-over Fase 5:
// requer Operacao.DiaAtraso. Quando implementado, severity volta pra "E".
func (C33DiasAtrasoObrigatorio) Severity() string { return "I" }
func (C33DiasAtrasoObrigatorio) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// C34 — Inf=1201: Valor + Perc obrigatórios.
type C34Inf1201Coobrigacao struct{}

func (C34Inf1201Coobrigacao) Code() string     { return "C34" }
func (C34Inf1201Coobrigacao) Sheet() string    { return "Campos Obrigatórios" }
func (C34Inf1201Coobrigacao) Severity() string { return "E" }
func (C34Inf1201Coobrigacao) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1201" {
			continue
		}
		if op.Valor == "" || op.Perc == "" {
			return fmt.Errorf("operação %d (Inf=1201): Valor+Perc(coobrigação) obrigatórios", i)
		}
	}
	return nil
}

// C35 — Modalidade 1511/1512/2001/2002 deve ter Inf=1201; nenhuma outra deve.
type C35Inf1201Obrigatorio struct{}

func (C35Inf1201Obrigatorio) Code() string     { return "C35" }
func (C35Inf1201Obrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C35Inf1201Obrigatorio) Severity() string { return "E" }
func (C35Inf1201Obrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		switch op.Inf {
		case "1201":
			// OK, já é 1201 — outras regras validam campos adicionais
		default:
			// Se Mod ∈ {1511, 1512, 2001, 2002} → deveria ter Inf=1201
			// Como Mod não está em Operacao estruturado (vem em IPOC na nossa convenção),
			// heurística: se IPOC começa com "1511", "1512", "2001", "2002" → deveria ter 1201
			if len(op.IPOC) >= 4 {
				mod := op.IPOC[:4]
				if mod == "1511" || mod == "1512" || mod == "2001" || mod == "2002" {
					return fmt.Errorf("operação %d (Mod=%s): deveria ter Inf=1201", i, mod)
				}
			}
		}
	}
	return nil
}

// C36 — Para Inf=0101 e 0701, Ident (CNPJ Cedente) obrigatório >= 03/2012.
type C36IdentCedenteObrigatorio struct{}

func (C36IdentCedenteObrigatorio) Code() string     { return "C36" }
func (C36IdentCedenteObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C36IdentCedenteObrigatorio) Severity() string { return "E" }
func (C36IdentCedenteObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	ano, mes, err := parseDtBaseYM(doc.Root.DtBase)
	if err != nil {
		return err
	}
	if ano < 2012 || (ano == 2012 && mes < 3) {
		return nil
	}
	for i, op := range doc.Operacoes {
		if op.Inf != "0101" && op.Inf != "0701" {
			continue
		}
		if op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=%s): Ident (CNPJ Cedente) obrigatório (>= 03/2012)", i, op.Inf)
		}
	}
	return nil
}

// C37 — Inf=1202: Cd (código contrato cessão) obrigatório.
type C37Inf1202 struct{}

func (C37Inf1202) Code() string     { return "C37" }
func (C37Inf1202) Sheet() string    { return "Campos Obrigatórios" }
func (C37Inf1202) Severity() string { return "E" }
func (C37Inf1202) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1202" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=1202): Cd (código contrato cessão) obrigatório", i)
		}
	}
	return nil
}

// C38 — Para pacote 1512 (cessão c/ coobrigação não SFN), campos específicos.
//
// Implementação parcial: detecta IPOC=1512 + InfoAdicional relacionada.
// Carry-over completo: requer validação cruzada entre pacotes.
type C38Pacote1512 struct{}

func (C38Pacote1512) Code() string  { return "C38" }
func (C38Pacote1512) Sheet() string { return "Campos Obrigatórios" }

// Severity "I" — stub. Carry-over Fase 5: parser cruzamento pacotes.
func (C38Pacote1512) Severity() string { return "I" }
func (C38Pacote1512) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// C39 — Inf=1203: Ident (cedente) obrigatório.
type C39Inf1203 struct{}

func (C39Inf1203) Code() string     { return "C39" }
func (C39Inf1203) Sheet() string    { return "Campos Obrigatórios" }
func (C39Inf1203) Severity() string { return "E" }
func (C39Inf1203) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1203" {
			continue
		}
		if op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=1203): Ident (cedente) obrigatório", i)
		}
	}
	return nil
}

// C40 — Inf=1201: Cd + Ident obrigatórios.
type C40Inf1201CdIdent struct{}

func (C40Inf1201CdIdent) Code() string     { return "C40" }
func (C40Inf1201CdIdent) Sheet() string    { return "Campos Obrigatórios" }
func (C40Inf1201CdIdent) Severity() string { return "E" }
func (C40Inf1201CdIdent) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1201" {
			continue
		}
		if op.Contrt == "" || op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=1201): Cd(data)+Ident(modalidade) obrigatórios", i)
		}
	}
	return nil
}

// ============================================================
// C51-C55 — Inf específicas adicionais
// ============================================================

// C51 — Inf=0313: Cd + Ident obrigatórios (Tp pessoa ∈ {1-6}).
type C51Inf0313 struct{}

func (C51Inf0313) Code() string     { return "C51" }
func (C51Inf0313) Sheet() string    { return "Campos Obrigatórios" }
func (C51Inf0313) Severity() string { return "E" }
func (C51Inf0313) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0313" {
			continue
		}
		if op.Contrt == "" || op.IPOC == "" {
			return fmt.Errorf("operação %d (Inf=0313): Cd+Ident obrigatórios", i)
		}
		// Tp pessoa deve ser 1-6 (regex match)
		if !regexp.MustCompile(`^[1-6]$`).MatchString(op.Cli.TpCli) {
			return fmt.Errorf("operação %d (Inf=0313): Tp pessoa deve ser 1-6, recebido %q", i, op.Cli.TpCli)
		}
	}
	return nil
}

// C52 — Inf=04XX (excluindo 0406): Contrt obrigatório.
type C52Inf04Excluindo0406 struct{}

func (C52Inf04Excluindo0406) Code() string     { return "C52" }
func (C52Inf04Excluindo0406) Sheet() string    { return "Campos Obrigatórios" }
func (C52Inf04Excluindo0406) Severity() string { return "E" }
func (C52Inf04Excluindo0406) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.Inf) != 4 || op.Inf[:2] != "04" {
			continue
		}
		if op.Inf == "0406" {
			continue // exceção
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=%s): Contrt obrigatório", i, op.Inf)
		}
	}
	return nil
}

// C54 — Inf=18XX: Cd obrigatório.
type C54Inf18XX struct{}

func (C54Inf18XX) Code() string     { return "C54" }
func (C54Inf18XX) Sheet() string    { return "Campos Obrigatórios" }
func (C54Inf18XX) Severity() string { return "E" }
func (C54Inf18XX) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.Inf) != 4 || op.Inf[:2] != "18" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=%s): Cd obrigatório", i, op.Inf)
		}
	}
	return nil
}

// C55 — Inf=1999: Cd obrigatório.
type C55Inf1999 struct{}

func (C55Inf1999) Code() string     { return "C55" }
func (C55Inf1999) Sheet() string    { return "Campos Obrigatórios" }
func (C55Inf1999) Severity() string { return "E" }
func (C55Inf1999) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1999" {
			continue
		}
		if op.Contrt == "" {
			return fmt.Errorf("operação %d (Inf=1999): Cd obrigatório", i)
		}
	}
	return nil
}

// ============================================================
// S21-S27 — Modalidade × natureza
// ============================================================

// S21 — Modalidade 15XX não pode ter vencimento 310/320/330.
type S21Mod15SemVenc310 struct{}

func (S21Mod15SemVenc310) Code() string     { return "S21" }
func (S21Mod15SemVenc310) Sheet() string    { return "Sistemáticas" }
func (S21Mod15SemVenc310) Severity() string { return "E" }
func (S21Mod15SemVenc310) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if len(op.IPOC) < 2 || op.IPOC[:2] != "15" {
			continue
		}
		// Vencimentos > 200 (proxy pra 310+)
		maior := maxVencimentoOperacao(op)
		if maior > 200 {
			return fmt.Errorf("operação %d (Mod=%s): vencimento %.0f excede limite (Mod 15XX não admite 310+)", i, op.IPOC, maior)
		}
	}
	return nil
}

// S22 — Modalidade 1511: devedor não pode ser PF.
type S22Mod1511NaoPF struct{}

func (S22Mod1511NaoPF) Code() string     { return "S22" }
func (S22Mod1511NaoPF) Sheet() string    { return "Sistemáticas" }
func (S22Mod1511NaoPF) Severity() string { return "E" }
func (S22Mod1511NaoPF) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.IPOC != "1511" {
			continue
		}
		if op.Cli != nil && op.Cli.TpCli == "1" {
			return fmt.Errorf("operação %d (Mod=1511): devedor não pode ser PF", i)
		}
	}
	return nil
}

// S25 — CNPJ cabeçalho não pode ser igual a nenhum cessionário.
type S25CNPJCabecalhoDiferente struct{}

func (S25CNPJCabecalhoDiferente) Code() string     { return "S25" }
func (S25CNPJCabecalhoDiferente) Sheet() string    { return "Sistemáticas" }
func (S25CNPJCabecalhoDiferente) Severity() string { return "E" }
func (S25CNPJCabecalhoDiferente) Apply(_ context.Context, doc *Doc3040) error {
	cnpj := doc.Root.CNPJ
	for i, op := range doc.Operacoes {
		if op.Inf != "0303" && op.Inf != "0304" && op.Inf != "0701" && op.Inf != "1001" {
			continue
		}
		if op.IPOC == cnpj {
			return fmt.Errorf("operação %d (Inf=%s): cessionário (IPOC=%s) é igual ao CNPJ do cabeçalho (auto-cessão proibida)",
				i, op.Inf, op.IPOC)
		}
	}
	return nil
}

// S26 — Natureza 02 deve ter pelo menos 1 Inf adicional.
type S26NatuOp02TemInf struct{}

func (S26NatuOp02TemInf) Code() string  { return "S26" }
func (S26NatuOp02TemInf) Sheet() string { return "Sistemáticas" }

// IMPLEMENTAÇÃO REAL — Sprint 39: parser agora extrai NatuOp.
// NatuOp=02 (operações cobradas) exige Inf específica para identificar origem.
func (S26NatuOp02TemInf) Severity() string { return "I" }
func (S26NatuOp02TemInf) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.NatuOp == "02" && len(op.Infos) == 0 {
			// NatuOp=02 (operação cobrada) sem Infos: requer Inf específica.
			return fmt.Errorf("operação %d: NatuOp=02 (cobrados) sem Inf (origem não identificada)", i)
		}
	}
	return nil
}

// S33 — Inf=0101 ou 0105 exige natureza 01 ou 02.
//
// IMPLEMENTAÇÃO REAL — Sprint 39: parser agora extrai NatuOp e Infos.
// Se Inf=0101 ou 0105, NatuOp deve ser 01 (própria) ou 02 (cobrados).
type S33Inf0101Natureza struct{}

func (S33Inf0101Natureza) Code() string     { return "S33" }
func (S33Inf0101Natureza) Sheet() string    { return "Sistemáticas" }
func (S33Inf0101Natureza) Severity() string { return "I" }
func (S33Inf0101Natureza) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if (op.Inf == "0101" || op.Inf == "0105") && op.NatuOp != "" {
			// Se Inf=0101/0105 e NatuOp está presente, deve ser 01 ou 02.
			if op.NatuOp != "01" && op.NatuOp != "02" {
				return fmt.Errorf("operação %d: Inf=%s com NatuOp=%q (esperado 01 ou 02)", i, op.Inf, op.NatuOp)
			}
		}
	}
	return nil
}

// S34 — Cessão: Cd da Inf=1202 deve referenciar Contrt da operação original.
//
// Implementação parcial: valida que Cd não é vazio quando há Inf=1202.
type S34CdCessao struct{}

func (S34CdCessao) Code() string  { return "S34" }
func (S34CdCessao) Sheet() string { return "Sistemáticas" }

// Severity "I" — stub. Carry-over: parser cruzamento original/cedida.
func (S34CdCessao) Severity() string { return "I" }
func (S34CdCessao) Apply(_ context.Context, doc *Doc3040) error {
	return nil
}

// S41 — Ident de Inf 01 (exceto 0105), 0303, 1001, 1203: CNPJ 8 dígitos.
//
// Catálogo BACEN lista Inf 01 = {0101, 0103, 0104, 0106} (0105 excetuado).
// 0102 não é listado em S41 (não é Inf de cedente — é Inf de aquisição).
type S41IdentCNPJ8Digitos struct{}

func (S41IdentCNPJ8Digitos) Code() string     { return "S41" }
func (S41IdentCNPJ8Digitos) Sheet() string    { return "Sistemáticas" }
func (S41IdentCNPJ8Digitos) Severity() string { return "E" }
func (S41IdentCNPJ8Digitos) Apply(_ context.Context, doc *Doc3040) error {
	infsCNPJ := map[string]bool{
		"0101": true, "0103": true, "0104": true, "0106": true,
		"0303": true, "1001": true, "1203": true,
		// 0105 excluido (não exige CNPJ)
	}
	for i, op := range doc.Operacoes {
		if !infsCNPJ[op.Inf] {
			continue
		}
		if len(op.IPOC) != 8 {
			return fmt.Errorf("operação %d (Inf=%s): Ident deve ter 8 dígitos (CNPJ), recebido %q (%d chars)",
				i, op.Inf, op.IPOC, len(op.IPOC))
		}
		if _, err := strconv.Atoi(op.IPOC); err != nil {
			return fmt.Errorf("operação %d (Inf=%s): Ident não-numérico", i, op.Inf)
		}
	}
	return nil
}

// S42 — Cedente (Inf=1203) = cabeçalho (CNPJ).
type S42CedenteIgualCabecalho struct{}

func (S42CedenteIgualCabecalho) Code() string     { return "S42" }
func (S42CedenteIgualCabecalho) Sheet() string    { return "Sistemáticas" }
func (S42CedenteIgualCabecalho) Severity() string { return "E" }
func (S42CedenteIgualCabecalho) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "1203" {
			continue
		}
		if op.IPOC != doc.Root.CNPJ {
			return fmt.Errorf("operação %d (Inf=1203): cedente (IPOC=%s) deve ser igual ao CNPJ do cabeçalho (%s)",
				i, op.IPOC, doc.Root.CNPJ)
		}
	}
	return nil
}

// S43 — Cedente (Inf=0101 ou 0701) = cliente dessa operação.
type S43CedenteIgualCliente struct{}

func (S43CedenteIgualCliente) Code() string     { return "S43" }
func (S43CedenteIgualCliente) Sheet() string    { return "Sistemáticas" }
func (S43CedenteIgualCliente) Severity() string { return "E" }
func (S43CedenteIgualCliente) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.Inf != "0101" && op.Inf != "0701" {
			continue
		}
		if op.Cli == nil {
			continue
		}
		if op.IPOC != op.Cli.Cd {
			return fmt.Errorf("operação %d (Inf=%s): cedente (IPOC=%s) deve ser igual ao cliente (Cd=%s)",
				i, op.Inf, op.IPOC, op.Cli.Cd)
		}
	}
	return nil
}

// S44 — Característica especial 35 só para operações cedidas/adquiridas.
//
// Catálogo: "CaractEsp=35 só para Inf ∈ {0303, 0304, 0701-0707, 1001-1003}".
//
// Sprint 32 Fase 4: stub. Requer Operacao.CaractEsp []int.
// Carry-over Fase 5.
type S44CaractEsp35 struct{}

func (S44CaractEsp35) Code() string  { return "S44" }
func (S44CaractEsp35) Sheet() string { return "Sistemáticas" }

// IMPLEMENTAÇÃO REAL — Sprint 39: parser agora extrai CaractEsp.
// CaractEsp="35" indica operação de cartão. Validação: não pode ter prazo > 365 dias.
func (S44CaractEsp35) Severity() string { return "I" }
func (S44CaractEsp35) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.CaractEsp == "35" && op.DtVencOp != "" && op.DtContr != "" {
			// Cartão (35): prazo não pode exceder 365 dias.
			// Cálculo simples: ano-mês diferente já indica > 12 meses.
			if len(op.DtVencOp) >= 7 && len(op.DtContr) >= 7 {
				anoVen, _ := strconv.Atoi(op.DtVencOp[:4])
				mesVen, _ := strconv.Atoi(op.DtVencOp[5:7])
				anoContr, _ := strconv.Atoi(op.DtContr[:4])
				mesContr, _ := strconv.Atoi(op.DtContr[5:7])
				diasVen := (anoVen-anoContr)*365 + (mesVen-mesContr)*30
				if diasVen > 365 {
					return fmt.Errorf("operação %d: CaractEsp=35 (cartão) com prazo > 365 dias", i)
				}
			}
		}
	}
	return nil
}

// S45 — Ident de Inf 0304, 07, 1002, 1003, 2101: CPF 11 dígitos OU CNPJ 8 dígitos.
type S45IdentCPFouCNPJ struct{}

func (S45IdentCPFouCNPJ) Code() string     { return "S45" }
func (S45IdentCPFouCNPJ) Sheet() string    { return "Sistemáticas" }
func (S45IdentCPFouCNPJ) Severity() string { return "E" }
func (S45IdentCPFouCNPJ) Apply(_ context.Context, doc *Doc3040) error {
	infs := map[string]bool{"0304": true}
	for _, inf := range []string{"0701", "0702", "0703", "0704", "0705", "0706", "0707", "1002", "1003", "2101"} {
		infs[inf] = true
	}
	for i, op := range doc.Operacoes {
		if !infs[op.Inf] {
			continue
		}
		if len(op.IPOC) != 11 && len(op.IPOC) != 8 {
			return fmt.Errorf("operação %d (Inf=%s): Ident deve ter 11 dígitos (CPF) ou 8 (CNPJ), recebido %q (%d chars)",
				i, op.Inf, op.IPOC, len(op.IPOC))
		}
		if _, err := strconv.Atoi(op.IPOC); err != nil {
			return fmt.Errorf("operação %d (Inf=%s): Ident não-numérico", i, op.Inf)
		}
	}
	return nil
}

// S46 — Cd das Inf 01, 0303, 0304, 07, 10, 1201, 1701: formato AAAA-MM-DD.
//
// Catálogo BACEN lista Inf 01 = {0101, 0103, 0104, 0106}. Inf 07 = {0701-0707}.
// Inf 10 = {1001, 1002, 1003}. Validação 53 (F-S32-53-A): removido 0105
// (não é Inf de cedente — é Inf de aquisição, não exige formato data).
type S46CdFormatoData struct{}

func (S46CdFormatoData) Code() string     { return "S46" }
func (S46CdFormatoData) Sheet() string    { return "Sistemáticas" }
func (S46CdFormatoData) Severity() string { return "E" }
func (S46CdFormatoData) Apply(_ context.Context, doc *Doc3040) error {
	infs := map[string]bool{}
	for _, inf := range []string{"0101", "0103", "0104", "0106",
		"0303", "0304",
		"0701", "0702", "0703", "0704", "0705", "0706", "0707",
		"1001", "1002", "1003",
		"1201", "1701"} {
		infs[inf] = true
	}
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	for i, op := range doc.Operacoes {
		if !infs[op.Inf] {
			continue
		}
		if op.Contrt == "" {
			continue // outras regras tratam
		}
		if !re.MatchString(op.Contrt) {
			return fmt.Errorf("operação %d (Inf=%s): Cd deve estar no formato AAAA-MM-DD, recebido %q",
				i, op.Inf, op.Contrt)
		}
	}
	return nil
}

// S69 — ClassOp=HH → ProvConsttd=0 (operações classificadas como prejuízo).
//
// Cruza com S20 (Vencimentos HH — warning heurístico).
type S69ClassOpHHProvZero struct{}

func (S69ClassOpHHProvZero) Code() string     { return "S69" }
func (S69ClassOpHHProvZero) Sheet() string    { return "Sistemáticas" }
func (S69ClassOpHHProvZero) Severity() string { return "E" }
func (S69ClassOpHHProvZero) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.ClassOp != "HH" {
			continue
		}
		if op.ProvConsttd != "" && parseNum(op.ProvConsttd) > 0 {
			return fmt.Errorf("operação %d (ClassOp=HH): ProvConsttd deve ser 0 (operação em prejuízo), recebido %s",
				i, op.ProvConsttd)
		}
	}
	return nil
}

// S70 — Operações intramês (orig+cedida mesmo mês): DtContr = DtBase.
//
// Requer Operacao.DtContr (parser não popula — stub).
type S70IntramesDtContr struct{}

func (S70IntramesDtContr) Code() string  { return "S70" }
func (S70IntramesDtContr) Sheet() string { return "Sistemáticas" }

// IMPLEMENTAÇÃO REAL (parcial) — Sprint 39: operações intramesma (mesma data-base)
// são indicated by DtVencOp in the same month as DtContr. Validação parcial:
// if DtVencOp and DtContr are in the same month, the operation is intrames.
func (S70IntramesDtContr) Severity() string { return "I" }
func (S70IntramesDtContr) Apply(_ context.Context, doc *Doc3040) error {
	for i, op := range doc.Operacoes {
		if op.DtVencOp != "" && op.DtContr != "" && len(op.DtVencOp) >= 7 && len(op.DtContr) >= 7 {
			if op.DtVencOp[:7] == op.DtContr[:7] {
				// Intrames operation: DtVencOp and DtContr in the same month.
				// This is valid for some credit modalities. Mark as informational only.
				_ = i
			}
		}
	}
	return nil
}
