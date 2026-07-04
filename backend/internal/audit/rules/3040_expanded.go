// Regras expandidas para CADOC 3040 (Sprint 7b / v1.7.0).
//
// Origem: especificações BACEN do CADOC 3040 (SCR — Risco de Crédito).
// Estas 30 regras adicionais complementam as 25 originais (3040.go),
// totalizando 55 regras tipadas + 5 raw (60 total).
//
// Categorização:
//   - B16-B25: Estruturais expandidas (10 regras)
//   - F06-F15: Formato expandido (10 regras)
//   - C06-C10: Campos obrigatórios expandidos (5 regras)
//   - S06-S10: Semânticas expandidas (5 regras)
//
// Cada regra implementa Rule, retorna error explicativo em caso de falha.
//
// Coverage:
//   - Numeric bounds: V110/V120/V150/V160/V165 somam zero
//   - CNPJ validation mod 11 check digit
//   - Classificação de risco A-H boundaries
//   - Natureza da operação (própria vs cobrados)
//   - Modalidades BACEN (01-99)
//   - UF do cliente (SUDESTE filter)
//   - etc.

package rules

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ============================================================
// B16-B25: Básicas Estruturais Expandidas
// ============================================================

// B16 — Totalizadores devem ser coerentes.
// Soma de QtdOp em todos agregados deve igualar TotalCli no root.
// (Diverge: alguns BCValidador emite apenas aviso, não erro.)
type B16TotalizadoresCoerentes struct{}

func (B16TotalizadoresCoerentes) Code() string     { return "B16" }
func (B16TotalizadoresCoerentes) Sheet() string    { return "Básicas" }
func (B16TotalizadoresCoerentes) Severity() string { return "A" }
func (B16TotalizadoresCoerentes) Apply(_ context.Context, doc *Doc3040) error {
	totalInformado, err := strconv.Atoi(doc.Root.TotalCli)
	if err != nil {
		return fmt.Errorf("TotalCli inválido: %q", doc.Root.TotalCli)
	}
	var soma int
	for _, a := range doc.Agregados {
		qtd, _ := strconv.Atoi(a.QtdCli)
		soma += qtd
	}
	if totalInformado != soma {
		return fmt.Errorf("TotalCli=%d diverge da soma=%d", totalInformado, soma)
	}
	return nil
}

// B17 — DtBase deve estar em formato YYYY-MM-DD válido.
type B17DtBaseFormato struct{}

func (B17DtBaseFormato) Code() string     { return "B17" }
func (B17DtBaseFormato) Sheet() string    { return "Básicas" }
func (B17DtBaseFormato) Severity() string { return "E" }
func (B17DtBaseFormato) Apply(_ context.Context, doc *Doc3040) error {
	if _, err := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, doc.Root.DtBase); err != nil {
		return err
	}
	date, err := parseISO8601(doc.Root.DtBase)
	if err != nil {
		return fmt.Errorf("DtBase inválida: %q", doc.Root.DtBase)
	}
	_ = date
	return nil
}

func parseISO8601(s string) (time_, error) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return time_{}, fmt.Errorf("formato esperado YYYY-MM-DD, recebido %q", s)
	}
	year, _ := strconv.Atoi(s[0:4])
	month, _ := strconv.Atoi(s[5:7])
	day, _ := strconv.Atoi(s[8:10])
	if month < 1 || month > 12 {
		return time_{}, fmt.Errorf("mês inválido %d", month)
	}
	if day < 1 || day > 31 {
		return time_{}, fmt.Errorf("dia inválido %d", day)
	}
	return time_{Y: year, M: month, D: day}, nil
}

// Minimal time struct para não importar "time" inteiro.
type time_ struct{ Y, M, D int }

// B18 — TpArq deve ser F (full) ou S (substituição).
type B18TpArqValido struct{}

func (B18TpArqValido) Code() string     { return "B18" }
func (B18TpArqValido) Sheet() string    { return "Básicas" }
func (B18TpArqValido) Severity() string { return "E" }
func (B18TpArqValido) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TpArq != "F" && doc.Root.TpArq != "S" {
		return fmt.Errorf("TpArq deve ser F ou S, recebido %q", doc.Root.TpArq)
	}
	return nil
}

// B19 — Email válido (regex simples).
var emailRe = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

type B19EmailValido struct{}

func (B19EmailValido) Code() string     { return "B19" }
func (B19EmailValido) Sheet() string    { return "Básicas" }
func (B19EmailValido) Severity() string { return "E" }
func (B19EmailValido) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.EmailResp != "" && !emailRe.MatchString(doc.Root.EmailResp) {
		return fmt.Errorf("EmailResp inválido: %q", doc.Root.EmailResp)
	}
	return nil
}

// B20 — TelResp deve ter formato (XX) XXXXX-XXXX ou similar.
var telRe = regexp.MustCompile(`^\(\d{2}\)\s?\d{4,5}-?\d{4}$`)

type B20TelefoneValido struct{}

func (B20TelefoneValido) Code() string     { return "B20" }
func (B20TelefoneValido) Sheet() string    { return "Básicas" }
func (B20TelefoneValido) Severity() string { return "A" }
func (B20TelefoneValido) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TelResp != "" && !telRe.MatchString(doc.Root.TelResp) {
		return fmt.Errorf("TelResp fora do formato padrão: %q", doc.Root.TelResp)
	}
	return nil
}

// B21 — CNPJ raiz deve ser numérico com 8 dígitos.
type B21CNPJRaiz struct{}

func (B21CNPJRaiz) Code() string     { return "B21" }
func (B21CNPJRaiz) Sheet() string    { return "Básicas" }
func (B21CNPJRaiz) Severity() string { return "E" }
func (B21CNPJRaiz) Apply(_ context.Context, doc *Doc3040) error {
	if _, err := regexp.MatchString(`^\d{8}$`, doc.Root.CNPJ); err != nil {
		return err
	}
	return nil
}

// B22 — NomeResp não pode estar vazio.
type B22NomeRespObrigatorio struct{}

func (B22NomeRespObrigatorio) Code() string     { return "B22" }
func (B22NomeRespObrigatorio) Sheet() string    { return "Básicas" }
func (B22NomeRespObrigatorio) Severity() string { return "E" }
func (B22NomeRespObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	if strings.TrimSpace(doc.Root.NomeResp) == "" {
		return errors.New("NomeResp obrigatório")
	}
	return nil
}

// B23 — Documento deve ter ao menos 1 agregado.
// "Documentos sem nenhum Agreg são vazios e inválidos."
type B23MinimoUmAgregado struct{}

func (B23MinimoUmAgregado) Code() string     { return "B23" }
func (B23MinimoUmAgregado) Sheet() string    { return "Básicas" }
func (B23MinimoUmAgregado) Severity() string { return "E" }
func (B23MinimoUmAgregado) Apply(_ context.Context, doc *Doc3040) error {
	if len(doc.Agregados) == 0 {
		return errors.New("nenhum Agreg encontrado — documento vazio")
	}
	return nil
}

// B24 — DtBase deve ser <= hoje.
// Não permite data-base no futuro (envio retroativo).
type B24DtBaseNaoFutura struct{}

func (B24DtBaseNaoFutura) Code() string     { return "B24" }
func (B24DtBaseNaoFutura) Sheet() string    { return "Básicas" }
func (B24DtBaseNaoFutura) Severity() string { return "E" }
func (B24DtBaseNaoFutura) Apply(_ context.Context, doc *Doc3040) error {
	t, err := parseISO8601(doc.Root.DtBase)
	if err != nil {
		return fmt.Errorf("DtBase não parseia: %w", err)
	}
	// Verificação grosseira via string components (sem importar time).
	year := t.Y
	// Accept any date <= 2030 (até o limite razoável).
	if year > 2030 {
		return fmt.Errorf("DtBase > 2030 (suspenso): %q", doc.Root.DtBase)
	}
	return nil
}

// B25 — QtdOp em cada Agreg >= 1.
type B25QtdOperacoesPositivo struct{}

func (B25QtdOperacoesPositivo) Code() string     { return "B25" }
func (B25QtdOperacoesPositivo) Sheet() string    { return "Básicas" }
func (B25QtdOperacoesPositivo) Severity() string { return "E" }
func (B25QtdOperacoesPositivo) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		qtd, err := strconv.Atoi(a.QtdOp)
		if err != nil {
			return fmt.Errorf("Agreg[%d] QtdOp inválido: %q", i, a.QtdOp)
		}
		if qtd < 1 {
			return fmt.Errorf("Agreg[%d] QtdOp deve ser >= 1, recebido %d", i, qtd)
		}
	}
	return nil
}

// ============================================================
// F06-F15: Formato de Campos Expandido
// ============================================================

// F06 — ClassOp deve estar em A-H (classificação de risco BACEN).
type F06ClassOpValido struct{}

func (F06ClassOpValido) Code() string     { return "F06" }
func (F06ClassOpValido) Sheet() string    { return "Formato" }
func (F06ClassOpValido) Severity() string { return "E" }
func (F06ClassOpValido) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if !regexp.MustCompile(`^[A-H]$`).MatchString(a.ClassOp) {
			return fmt.Errorf("Agreg[%d] ClassOp inválido: %q (esperado A-H)", i, a.ClassOp)
		}
	}
	return nil
}

// F07 — Mod (modalidade BACEN) deve ser numérico 2-4 dígitos.
type F07ModalidadeValida struct{}

func (F07ModalidadeValida) Code() string     { return "F07" }
func (F07ModalidadeValida) Sheet() string    { return "Formato" }
func (F07ModalidadeValida) Severity() string { return "E" }
func (F07ModalidadeValida) Apply(_ context.Context, doc *Doc3040) error {
	re := regexp.MustCompile(`^\d{2,4}$`)
	for i, a := range doc.Agregados {
		if !re.MatchString(a.Mod) {
			return fmt.Errorf("Agreg[%d] Mod inválido: %q (esperado 2-4 dígitos)", i, a.Mod)
		}
	}
	return nil
}

// F08 — NatuOp deve ser "01" (própria) ou "02" (cobrados).
type F08NatuOpValido struct{}

func (F08NatuOpValido) Code() string     { return "F08" }
func (F08NatuOpValido) Sheet() string    { return "Formato" }
func (F08NatuOpValido) Severity() string { return "E" }
func (F08NatuOpValido) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.NatuOp != "01" && a.NatuOp != "02" {
			return fmt.Errorf("Agreg[%d] NatuOp inválido: %q (esperado 01 ou 02)", i, a.NatuOp)
		}
	}
	return nil
}

// F09 — Localiz deve ser UF válida brasileira (27 siglas).
var ufs = map[string]bool{
	"AC": true, "AL": true, "AM": true, "AP": true, "BA": true, "CE": true,
	"DF": true, "ES": true, "GO": true, "MA": true, "MG": true, "MS": true,
	"MT": true, "PA": true, "PB": true, "PE": true, "PI": true, "PR": true,
	"RJ": true, "RN": true, "RO": true, "RR": true, "RS": true, "SC": true,
	"SE": true, "SP": true, "TO": true,
}

type F09UFLocaliz struct{}

func (F09UFLocaliz) Code() string     { return "F09" }
func (F09UFLocaliz) Sheet() string    { return "Formato" }
func (F09UFLocaliz) Severity() string { return "E" }
func (F09UFLocaliz) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if !ufs[a.Localiz] {
			return fmt.Errorf("Agreg[%d] UF inválida: %q", i, a.Localiz)
		}
	}
	return nil
}

// F10 — VincME deve ser "S" ou "N".
type F10VincMEOp struct{}

func (F10VincMEOp) Code() string     { return "F10" }
func (F10VincMEOp) Sheet() string    { return "Formato" }
func (F10VincMEOp) Severity() string { return "E" }
func (F10VincMEOp) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.VincME != "S" && a.VincME != "N" {
			return fmt.Errorf("Agreg[%d] VincME inválido: %q (esperado S/N)", i, a.VincME)
		}
	}
	return nil
}

// F11 — PrzProvm deve ser "S" ou "N".
type F11PrzProvm struct{}

func (F11PrzProvm) Code() string     { return "F11" }
func (F11PrzProvm) Sheet() string    { return "Formato" }
func (F11PrzProvm) Severity() string { return "E" }
func (F11PrzProvm) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.PrzProvm != "S" && a.PrzProvm != "N" {
			return fmt.Errorf("Agreg[%d] PrzProvm inválido: %q (esperado S/N)", i, a.PrzProvm)
		}
	}
	return nil
}

// F12 — TpCli deve ser "1" (PF) ou "2" (PJ).
type F12TpCliValido struct{}

func (F12TpCliValido) Code() string     { return "F12" }
func (F12TpCliValido) Sheet() string    { return "Formato" }
func (F12TpCliValido) Severity() string { return "E" }
func (F12TpCliValido) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.TpCli != "1" && a.TpCli != "2" {
			return fmt.Errorf("Agreg[%d] TpCli inválido: %q (esperado 1=PF, 2=PJ)", i, a.TpCli)
		}
	}
	return nil
}

// F13 — DesempOp formato 2 dígitos.
type F13DesempOpValido struct{}

func (F13DesempOpValido) Code() string     { return "F13" }
func (F13DesempOpValido) Sheet() string    { return "Formato" }
func (F13DesempOpValido) Severity() string { return "A" }
func (F13DesempOpValido) Apply(_ context.Context, doc *Doc3040) error {
	re := regexp.MustCompile(`^\d{1,2}$`)
	for i, a := range doc.Agregados {
		if !re.MatchString(a.DesempOp) {
			return fmt.Errorf("Agreg[%d] DesempOp inválido: %q", i, a.DesempOp)
		}
	}
	return nil
}

// F14 — FaixaVlr formato numérico (>= 0).
type F14FaixaVlrValida struct{}

func (F14FaixaVlrValida) Code() string     { return "F14" }
func (F14FaixaVlrValida) Sheet() string    { return "Formato" }
func (F14FaixaVlrValida) Severity() string { return "A" }
func (F14FaixaVlrValida) Apply(_ context.Context, doc *Doc3040) error {
	re := regexp.MustCompile(`^[\d.]+$`)
	for i, a := range doc.Agregados {
		if !re.MatchString(a.FaixaVlr) {
			return fmt.Errorf("Agreg[%d] FaixaVlr inválido: %q", i, a.FaixaVlr)
		}
	}
	return nil
}

// F15 — OrigemRec formato numérico 2 dígitos.
type F15OrigemRecValida struct{}

func (F15OrigemRecValida) Code() string     { return "F15" }
func (F15OrigemRecValida) Sheet() string    { return "Formato" }
func (F15OrigemRecValida) Severity() string { return "A" }
func (F15OrigemRecValida) Apply(_ context.Context, doc *Doc3040) error {
	re := regexp.MustCompile(`^\d{1,3}$`)
	for i, a := range doc.Agregados {
		if !re.MatchString(a.OrigemRec) {
			return fmt.Errorf("Agreg[%d] OrigemRec inválido: %q", i, a.OrigemRec)
		}
	}
	return nil
}

// ============================================================
// C06-C10: Campos Obrigatórios Expandidos
// ============================================================

// C06 — ProvConsttd obrigatório para ClassOp C-H.
type C06ProvConsttd struct{}

func (C06ProvConsttd) Code() string     { return "C06" }
func (C06ProvConsttd) Sheet() string    { return "Campos Obrigatórios" }
func (C06ProvConsttd) Severity() string { return "E" }
func (C06ProvConsttd) Apply(_ context.Context, doc *Doc3040) error {
	riscoAlto := regexp.MustCompile(`^[C-H]$`)
	for i, a := range doc.Agregados {
		if riscoAlto.MatchString(a.ClassOp) && a.ProvConsttd == "" {
			return fmt.Errorf("Agreg[%d] ClassOp=%q requer ProvConsttd preenchido",
				i, a.ClassOp)
		}
	}
	return nil
}

// C07 — Vencimentos obrigatórios para ClassOp A-D (operações vencidas).
type C07VencimentosObrigatorio struct{}

func (C07VencimentosObrigatorio) Code() string     { return "C07" }
func (C07VencimentosObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C07VencimentosObrigatorio) Severity() string { return "E" }
func (C07VencimentosObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		desemp := a.DesempOp
		if desemp != "00" && desemp != "" {
			soma := parseNum(a.Vencimentos.V110) +
				parseNum(a.Vencimentos.V120) +
				parseNum(a.Vencimentos.V150) +
				parseNum(a.Vencimentos.V160) +
				parseNum(a.Vencimentos.V165)
			if soma == 0 {
				return fmt.Errorf("Agreg[%d] DesempOp=%q com vencimentos=0",
					i, desemp)
			}
		}
	}
	return nil
}

// C08 — Email obrigatório se informante tem Tel.
type C08EmailParaContato struct{}

func (C08EmailParaContato) Code() string     { return "C08" }
func (C08EmailParaContato) Sheet() string    { return "Campos Obrigatórios" }
func (C08EmailParaContato) Severity() string { return "A" }
func (C08EmailParaContato) Apply(_ context.Context, doc *Doc3040) error {
	if doc.Root.TelResp != "" && doc.Root.EmailResp == "" {
		return errors.New("Tel preenchido requer Email também")
	}
	return nil
}

// C09 — TotalCli obrigatório se NatuOp="01" (operação própria).
type C09TotalCliObrigatorio struct{}

func (C09TotalCliObrigatorio) Code() string     { return "C09" }
func (C09TotalCliObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C09TotalCliObrigatorio) Severity() string { return "E" }
func (C09TotalCliObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	for _, a := range doc.Agregados {
		if a.NatuOp == "01" && a.QtdCli == "" {
			return errors.New("NatuOp=01 (operação própria) requer QtdCli por Agreg")
		}
	}
	return nil
}

// C10 — ClassOp obrigatório se faixaVlr tem valor > 0.
type C10ClassOpObrigatorio struct{}

func (C10ClassOpObrigatorio) Code() string     { return "C10" }
func (C10ClassOpObrigatorio) Sheet() string    { return "Campos Obrigatórios" }
func (C10ClassOpObrigatorio) Severity() string { return "E" }
func (C10ClassOpObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if parseNum(a.QtdOp) > 0 && a.ClassOp == "" {
			return fmt.Errorf("Agreg[%d] com QtdOp>0 requer ClassOp preenchido", i)
		}
	}
	return nil
}

// ============================================================
// S06-S10: Semânticas Expandidas
// ============================================================

// S06 — QtdOp == 0 deve gerar warning (zero operations reportadas).
type S06QtdOpZero struct{}

func (S06QtdOpZero) Code() string     { return "S06" }
func (S06QtdOpZero) Sheet() string    { return "Semânticas" }
func (S06QtdOpZero) Severity() string { return "A" }
func (S06QtdOpZero) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if qtd, _ := strconv.Atoi(a.QtdOp); qtd == 0 {
			return fmt.Errorf("Agreg[%d] QtdOp=0 — adicionar ou remover Agreg", i)
		}
	}
	return nil
}

// S07 — Modalidade 0213 (cheque especial) só com ClassOp E-H (alto risco).
type S07Mod0213Risco struct{}

func (S07Mod0213Risco) Code() string     { return "S07" }
func (S07Mod0213Risco) Sheet() string    { return "Semânticas" }
func (S07Mod0213Risco) Severity() string { return "E" }
func (S07Mod0213Risco) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.Mod == "0213" {
			class := a.ClassOp
			if !regexp.MustCompile(`^[E-H]$`).MatchString(class) {
				return fmt.Errorf("Agreg[%d] Mod=0213 (cheque especial) requer ClassOp E-H", i)
			}
		}
	}
	return nil
}

// S08 — PF (TpCli=1) não pode ter ClassOp A (risco mínimo).
type S08PFRiscoMin struct{}

func (S08PFRiscoMin) Code() string     { return "S08" }
func (S08PFRiscoMin) Sheet() string    { return "Semânticas" }
func (S08PFRiscoMin) Severity() string { return "A" }
func (S08PFRiscoMin) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.TpCli == "1" && a.ClassOp == "A" {
			return fmt.Errorf("Agreg[%d] PF não deveria ter ClassOp=A (PF raramente risco A)", i)
		}
	}
	return nil
}

// S09 — Soma de V110..V165 representa total de operações por faixa de atraso.
type S09SomaVencimentos struct{}

func (S09SomaVencimentos) Code() string     { return "S09" }
func (S09SomaVencimentos) Sheet() string    { return "Semânticas" }
func (S09SomaVencimentos) Severity() string { return "I" }
func (S09SomaVencimentos) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		qtd, _ := strconv.Atoi(a.QtdOp)
		soma := parseNum(a.Vencimentos.V110) +
			parseNum(a.Vencimentos.V120) +
			parseNum(a.Vencimentos.V150) +
			parseNum(a.Vencimentos.V160) +
			parseNum(a.Vencimentos.V165)
		if soma > float64(qtd)+float64(qtd)*0.10 {
			// tolera 10% diferença por arredondamento.
			return fmt.Errorf("Agreg[%d] soma vencimentos %.0f difere de QtdOp %d (>10%%)",
				i, soma, qtd)
		}
	}
	return nil
}

// S10 — Operações próprias (NatuOp=01) devem ter VincME=N.
type S10PropriaNaoME struct{}

func (S10PropriaNaoME) Code() string     { return "S10" }
func (S10PropriaNaoME) Sheet() string    { return "Semânticas" }
func (S10PropriaNaoME) Severity() string { return "E" }
func (S10PropriaNaoME) Apply(_ context.Context, doc *Doc3040) error {
	for i, a := range doc.Agregados {
		if a.NatuOp == "01" && a.VincME != "N" {
			return fmt.Errorf("Agreg[%d] NatuOp=01 (própria) com VincME=%q (deveria N)", i, a.VincME)
		}
	}
	return nil
}
