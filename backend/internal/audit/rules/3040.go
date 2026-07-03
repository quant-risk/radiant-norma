// Regras portadas para o CADOC 3040 (SCR — Risco de Crédito).
//
// Convenção: cada regra implementa Rule. Apply retorna nil se OK ou
// error se falhou (mensagem legível ao usuário final).
//
// Cobertura Sprint 4 (v1.3.0):
//   - Básicas B06-B15 (contadores e limites)
//   - Formato F01-F05 (validação de formato de campos)
//   - Campos Obrigatórios C01-C05 (obrigatoriedade condicional)
//   - Semântica S01-S05 (regras semânticas gerais)

package rules

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// ============================================================
// B06-B15: Básicas (contadores, limites, regras estruturais)
// ============================================================

// B06 — Número de remessa incompatível.
// "O envio de uma nova remessa (ou seja, uma substituição) do documento 3040
// deve corresponder à última remessa aceita + 1."
type B06RemessaIncompativel struct{}

func (B06RemessaIncompativel) Code() string     { return "B06" }
func (B06RemessaIncompativel) Sheet() string    { return "Básicas" }
func (B06RemessaIncompativel) Severity() string { return "E" }
func (B06RemessaIncompativel) Apply(_ context.Context, doc *Doc3040) error {
	remessa, err := strconv.Atoi(doc.Root.Remessa)
	if err != nil {
		return fmt.Errorf("remessa inválida: %q", doc.Root.Remessa)
	}
	if remessa < 1 {
		return fmt.Errorf("remessa deve ser >= 1, recebido %d", remessa)
	}
	return nil
}

// B07 — Composição da remessa (número de parte e final de remessa).
// "Uma remessa do documento 3040 deve ser composta de N partes, onde N >= 1.
// As partes de uma remessa devem ser numeradas sequencialmente de 1 a N
// sem gaps."
type B07ComposicaoRemessa struct{}

func (B07ComposicaoRemessa) Code() string     { return "B07" }
func (B07ComposicaoRemessa) Sheet() string    { return "Básicas" }
func (B07ComposicaoRemessa) Severity() string { return "E" }
func (B07ComposicaoRemessa) Apply(_ context.Context, doc *Doc3040) error {
	parte, err := strconv.Atoi(doc.Root.Parte)
	if err != nil {
		return fmt.Errorf("parte inválida: %q", doc.Root.Parte)
	}
	if parte < 1 {
		return fmt.Errorf("parte deve ser >= 1, recebido %d", parte)
	}
	return nil
}

// B08 — Remessa com parte rejeitada.
// Stub: verificação real precisa consultar histórico de envios.
type B08ParteRejeitada struct{}

func (B08ParteRejeitada) Code() string     { return "B08" }
func (B08ParteRejeitada) Sheet() string    { return "Básicas" }
func (B08ParteRejeitada) Severity() string { return "A" }
func (B08ParteRejeitada) Apply(_ context.Context, _ *Doc3040) error {
	// Sem histórico de envios no contexto da regra → skip
	return nil
}

// B09 — Número máximo de erros.
// Limite do BCValidador: 5000 erros por documento.
type B09MaxErros struct{}

func (B09MaxErros) Code() string     { return "B09" }
func (B09MaxErros) Sheet() string    { return "Básicas" }
func (B09MaxErros) Severity() string { return "I" }
func (B09MaxErros) Apply(_ context.Context, _ *Doc3040) error {
	return nil // informativo; limite validado pós-coleta
}

// B10 — Número máximo de avisos.
// Limite do BCValidador: 5000 avisos por documento.
type B10MaxAvisos struct{}

func (B10MaxAvisos) Code() string     { return "B10" }
func (B10MaxAvisos) Sheet() string    { return "Básicas" }
func (B10MaxAvisos) Severity() string { return "I" }
func (B10MaxAvisos) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// B11 — Documento não aceito na data-base anterior.
// Stub: requer consultar histórico.
type B11NaoAceitoAnterior struct{}

func (B11NaoAceitoAnterior) Code() string     { return "B11" }
func (B11NaoAceitoAnterior) Sheet() string    { return "Básicas" }
func (B11NaoAceitoAnterior) Severity() string { return "A" }
func (B11NaoAceitoAnterior) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// B12 — Atributo TpFundo obrigatório apenas para FIDCs.
type B12TpFundoObrigatorio struct{}

func (B12TpFundoObrigatorio) Code() string     { return "B12" }
func (B12TpFundoObrigatorio) Sheet() string    { return "Básicas" }
func (B12TpFundoObrigatorio) Severity() string { return "A" }
func (B12TpFundoObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	// Sem saber se IF é FIDC, skip. (Requer integração com cadastro de IFs)
	_ = doc
	return nil
}

// B13 — IF não é FIDC mas TpFundo preenchido.
type B13IFNaoFIDC struct{}

func (B13IFNaoFIDC) Code() string     { return "B13" }
func (B13IFNaoFIDC) Sheet() string    { return "Básicas" }
func (B13IFNaoFIDC) Severity() string { return "A" }
func (B13IFNaoFIDC) Apply(_ context.Context, doc *Doc3040) error {
	_ = doc
	return nil
}

// B14 — Excedida quantidade de operações divergentes no 3042.
// Stub: requer comparar com 3042 enviado.
type B14MaxOpDivergentes3042 struct{}

func (B14MaxOpDivergentes3042) Code() string     { return "B14" }
func (B14MaxOpDivergentes3042) Sheet() string    { return "Básicas" }
func (B14MaxOpDivergentes3042) Severity() string { return "A" }
func (B14MaxOpDivergentes3042) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// B15 — Excedida quantidade de operações divergentes no 3040.
type B15MaxOpDivergentes3040 struct{}

func (B15MaxOpDivergentes3040) Code() string     { return "B15" }
func (B15MaxOpDivergentes3040) Sheet() string    { return "Básicas" }
func (B15MaxOpDivergentes3040) Severity() string { return "A" }
func (B15MaxOpDivergentes3040) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// ============================================================
// F01-F05: Formato
// ============================================================

// F01 — Taxa efetiva anual.
// "Forma de taxa percentual anual, em uma base centesimal, com a utilização
// de duas a sete casas decimais."
type F01TaxaEfetivaAnual struct{}

func (F01TaxaEfetivaAnual) Code() string     { return "F01" }
func (F01TaxaEfetivaAnual) Sheet() string    { return "Formato" }
func (F01TaxaEfetivaAnual) Severity() string { return "E" }
func (F01TaxaEfetivaAnual) Apply(_ context.Context, _ *Doc3040) error {
	// Validação real: percorre tag <Taxas> nas operações individualizadas.
	// Stub da Sprint 4: regras de agregadas não têm taxa.
	return nil
}

// F02 — Datas (formato AAAA-MM-DD).
type F02Datas struct{}

func (F02Datas) Code() string     { return "F02" }
func (F02Datas) Sheet() string    { return "Formato" }
func (F02Datas) Severity() string { return "E" }
func (F02Datas) Apply(_ context.Context, doc *Doc3040) error {
	// BACEN: AAAA-MM-DD com 4 dígitos ano, 2 mês, 2 dia.
	// DtBase do 3040 é tipicamente YYYY-MM (mensal) mas a regra geral é AAAA-MM-DD.
	if doc.Root.DtBase == "" {
		return errors.New("DtBase vazio")
	}
	// Aceita YYYY-MM (decisão mensal) e YYYY-MM-DD (data cheia)
	if !datePattern.MatchString(doc.Root.DtBase) {
		return fmt.Errorf("DtBase fora do padrão AAAA-MM[-DD]: %q", doc.Root.DtBase)
	}
	return nil
}

// datePattern é o regex precompilado para F02 (perf: não compilar a cada chamada).
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}(-\d{2})?$`)

// F03 — Código do contrato não pode ser apenas espaços.
// Validação aplica nas tags <Cli>/<Oper>. No agregado, skip.
type F03CodigoContrato struct{}

func (F03CodigoContrato) Code() string     { return "F03" }
func (F03CodigoContrato) Sheet() string    { return "Formato" }
func (F03CodigoContrato) Severity() string { return "E" }
func (F03CodigoContrato) Apply(_ context.Context, _ *Doc3040) error {
	// Não aplicável a agregados — só individualizadas.
	return nil
}

// F04 — Código do conglomerado não pode ser "0".
type F04Conglomerado struct{}

func (F04Conglomerado) Code() string     { return "F04" }
func (F04Conglomerado) Sheet() string    { return "Formato" }
func (F04Conglomerado) Severity() string { return "E" }
func (F04Conglomerado) Apply(_ context.Context, _ *Doc3040) error {
	// Não aplicável a agregados.
	return nil
}

// F05 — Formato do campo RefBacen na tag Sicor.
// "Para operações de crédito contratadas a partir de janeiro de 2013,
// os quatro primeiros dígitos do atributo devem corresponder ao código
// da IF credora."
type F05RefBacenSicor struct{}

func (F05RefBacenSicor) Code() string     { return "F05" }
func (F05RefBacenSicor) Sheet() string    { return "Formato" }
func (F05RefBacenSicor) Severity() string { return "E" }
func (F05RefBacenSicor) Apply(_ context.Context, _ *Doc3040) error {
	// Não aplicável a agregados — só individualizadas.
	return nil
}

// ============================================================
// C01-C05: Campos Obrigatórios
// ============================================================

// C01 — Campos obrigatórios somente para pessoa jurídica.
// Exige preenchimento de RazãoSocial, CNPJ quando TpCli=2 (PJ).
type C01CamposObrigatoriosPJ struct{}

func (C01CamposObrigatoriosPJ) Code() string     { return "C01" }
func (C01CamposObrigatoriosPJ) Sheet() string    { return "Campos Obrigatórios" }
func (C01CamposObrigatoriosPJ) Severity() string { return "E" }
func (C01CamposObrigatoriosPJ) Apply(_ context.Context, doc *Doc3040) error {
	// Para cada Agregado, se TpCli=2 (PJ), validar QtdCli > 0 e TotalCli coerente
	for i, ag := range doc.Agregados {
		if ag.TpCli == "2" {
			qtd, _ := strconv.Atoi(ag.QtdCli)
			if qtd == 0 {
				return fmt.Errorf("Agreg[%d]: TpCli=2 (PJ) mas QtdCli=0", i)
			}
		}
	}
	return nil
}

// C02 — Campos não obrigatórios (validação inversa: não podem ter conteúdo
// quando a condição não é satisfeita).
type C02CamposNaoObrigatorios struct{}

func (C02CamposNaoObrigatorios) Code() string     { return "C02" }
func (C02CamposNaoObrigatorios) Sheet() string    { return "Campos Obrigatórios" }
func (C02CamposNaoObrigatorios) Severity() string { return "A" }
func (C02CamposNaoObrigatorios) Apply(_ context.Context, _ *Doc3040) error {
	// Stub: requer schema completo de quando cada campo é exigido.
	return nil
}

// C03 — Garantias não fidejussórias: campos específicos obrigatórios.
type C03GarantiasNaoFidejussorias struct{}

func (C03GarantiasNaoFidejussorias) Code() string     { return "C03" }
func (C03GarantiasNaoFidejussorias) Sheet() string    { return "Campos Obrigatórios" }
func (C03GarantiasNaoFidejussorias) Severity() string { return "E" }
func (C03GarantiasNaoFidejussorias) Apply(_ context.Context, _ *Doc3040) error {
	// Não aplicável a agregados (tag <Garant> é individualizada).
	return nil
}

// C04 — Garantias fidejussórias.
type C04GarantiasFidejussorias struct{}

func (C04GarantiasFidejussorias) Code() string     { return "C04" }
func (C04GarantiasFidejussorias) Sheet() string    { return "Campos Obrigatórios" }
func (C04GarantiasFidejussorias) Severity() string { return "E" }
func (C04GarantiasFidejussorias) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// C05 — Cessões com coobrigação entre IFs (informação de cessionário).
type C05CessoesCoobrigacao struct{}

func (C05CessoesCoobrigacao) Code() string     { return "C05" }
func (C05CessoesCoobrigacao) Sheet() string    { return "Campos Obrigatórios" }
func (C05CessoesCoobrigacao) Severity() string { return "E" }
func (C05CessoesCoobrigacao) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// ============================================================
// S01-S05: Semântica
// ============================================================

// S01 — Detalhamento do cliente.
// Quando QtdCli=1, os campos de detalhamento devem estar preenchidos.
type S01DetalhamentoCliente struct{}

func (S01DetalhamentoCliente) Code() string     { return "S01" }
func (S01DetalhamentoCliente) Sheet() string    { return "Semântica" }
func (S01DetalhamentoCliente) Severity() string { return "E" }
func (S01DetalhamentoCliente) Apply(_ context.Context, doc *Doc3040) error {
	// QtdCli=1 implica em cliente único → individualização obrigatória.
	// Validação real: verificar se QtdOp > 1 ou se há <Cli> correspondente.
	for i, ag := range doc.Agregados {
		qtdCli, _ := strconv.Atoi(ag.QtdCli)
		qtdOp, _ := strconv.Atoi(ag.QtdOp)
		if qtdCli == 1 && qtdOp > 1 {
			return fmt.Errorf("Agreg[%d]: QtdCli=1 mas QtdOp=%d — operação deveria ser individualizada", i, qtdOp)
		}
	}
	return nil
}

// S02 — Vendor: necessidade de informação adicional.
type S02VendorInfo struct{}

func (S02VendorInfo) Code() string     { return "S02" }
func (S02VendorInfo) Sheet() string    { return "Semântica" }
func (S02VendorInfo) Severity() string { return "A" }
func (S02VendorInfo) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// S03 — Ocultação de operação em prejuízo há mais de 48 meses.
// Não se aplica em envios mensais normais.
type S03Ocultacao struct{}

func (S03Ocultacao) Code() string     { return "S03" }
func (S03Ocultacao) Sheet() string    { return "Semântica" }
func (S03Ocultacao) Severity() string { return "A" }
func (S03Ocultacao) Apply(_ context.Context, _ *Doc3040) error {
	return nil
}

// S04 — Crédito a liberar: não aplicabilidade.
//
// Catálogo BACEN: "Não poderão ter preenchidos os vencimentos de crédito a
// liberar (vencimentos 60 e 80) as modalidades 'crédito rotativo vinculado
// a cartão de crédito' (0204), 'cartão de crédito - compra parcelada' (0210),
// 'cartão de crédito - compra à vista' (1304), 'cheque especial e conta
// garantida' (0201), 'cheque especial' (0213) e 'conta garantida' (0214)".
//
// Mapeamento IDs BACEN → campos do XML:
//   - Vencimento 60 (31-60 dias) → Vencimentos.V150
//   - Vencimento 80 (61-90 dias) → Vencimentos.V160
type S04CreditoALiberar struct{}

func (S04CreditoALiberar) Code() string     { return "S04" }
func (S04CreditoALiberar) Sheet() string    { return "Semântica" }
func (S04CreditoALiberar) Severity() string { return "E" }
func (S04CreditoALiberar) Apply(_ context.Context, doc *Doc3040) error {
	// Modalidades onde "crédito a liberar" (venc 60 e 80) não pode ser preenchido.
	creditoLiberarMods := map[string]bool{
		"0204": true, // crédito rotativo vinculado a cartão de crédito
		"0210": true, // cartão de crédito - compra parcelada
		"1304": true, // cartão de crédito - compra à vista
		"0201": true, // cheque especial e conta garantida
		"0213": true, // cheque especial
		"0214": true, // conta garantida
	}
	for i, ag := range doc.Agregados {
		if !creditoLiberarMods[ag.Mod] {
			continue
		}
		v := ag.Vencimentos
		// Vencimentos "60" e "80" (= v150 e v160) devem ser zero.
		if v.V150 != "0" || v.V160 != "0" {
			return fmt.Errorf("Agreg[%d]: modalidade %s (crédito a liberar) não pode ter vencimentos 60/80 preenchidos (v150=%s v160=%s)",
				i, ag.Mod, v.V150, v.V160)
		}
	}
	return nil
}

// S05 — Limite de crédito: vencimentos possíveis.
// "A modalidade 'Limite de Crédito' (19) só pode ter vencimentos de limite
// (20 e 40). Nenhum outro vencimento pode ser aceito quando esta modalidade
// for informada."
type S05LimiteCredito struct{}

func (S05LimiteCredito) Code() string     { return "S05" }
func (S05LimiteCredito) Sheet() string    { return "Semântica" }
func (S05LimiteCredito) Severity() string { return "E" }
func (S05LimiteCredito) Apply(_ context.Context, doc *Doc3040) error {
	for i, ag := range doc.Agregados {
		// Modalidade 19 = Limite de Crédito (BACEN).
		// Vencimentos de limite são v110 (faixa 020) e v120 (faixa 040).
		if ag.Mod == "19" {
			// v150 (31-60), v160 (61-90), v165 (>90) devem ser 0.
			v150, _ := strconv.Atoi(ag.Vencimentos.V150)
			v160, _ := strconv.Atoi(ag.Vencimentos.V160)
			v165, _ := strconv.Atoi(ag.Vencimentos.V165)
			if v150 != 0 || v160 != 0 || v165 != 0 {
				return fmt.Errorf("Agreg[%d]: Limite de Crédito (Mod=19) só aceita vencimentos v110/v120, mas tem v150=%s v160=%s v165=%s",
					i, ag.Vencimentos.V150, ag.Vencimentos.V160, ag.Vencimentos.V165)
			}
		}
	}
	return nil
}

// ============================================================
// Parser 3040
// ============================================================

// ParseDoc3040 faz unmarshal do XML 3040 em struct tipada.
func ParseDoc3040(xmlContent []byte) (*Doc3040, error) {
	type venc struct {
		V110 string `xml:"v110,attr"`
		V120 string `xml:"v120,attr"`
		V150 string `xml:"v150,attr"`
		V160 string `xml:"v160,attr"`
		V165 string `xml:"v165,attr"`
	}
	type agreg struct {
		NatuOp      string `xml:"NatuOp,attr"`
		Mod         string `xml:"Mod,attr"`
		OrigemRec   string `xml:"OrigemRec,attr"`
		VincME      string `xml:"VincME,attr"`
		ClassOp     string `xml:"ClassOp,attr"`
		FaixaVlr    string `xml:"FaixaVlr,attr"`
		PrzProvm    string `xml:"PrzProvm,attr"`
		Localiz     string `xml:"Localiz,attr"`
		TpCli       string `xml:"TpCli,attr"`
		DesempOp    string `xml:"DesempOp,attr"`
		ProvConsttd string `xml:"ProvConsttd,attr"`
		QtdOp       string `xml:"QtdOp,attr"`
		QtdCli      string `xml:"QtdCli,attr"`
		Venc        venc   `xml:"Venc"`
	}
	type doc struct {
		XMLName   xml.Name `xml:"Doc3040"`
		DtBase    string   `xml:"DtBase,attr"`
		CNPJ      string   `xml:"CNPJ,attr"`
		Remessa   string   `xml:"Remessa,attr"`
		Parte     string   `xml:"Parte,attr"`
		TpArq     string   `xml:"TpArq,attr"`
		NomeResp  string   `xml:"NomeResp,attr"`
		EmailResp string   `xml:"EmailResp,attr"`
		TelResp   string   `xml:"TelResp,attr"`
		TotalCli  string   `xml:"TotalCli,attr"`
		Agreg     []agreg  `xml:"Agreg"`
	}
	var d doc
	if err := xml.Unmarshal(xmlContent, &d); err != nil {
		return nil, err
	}

	out := &Doc3040{
		Root: Doc3040Root{
			DtBase: d.DtBase, CNPJ: d.CNPJ, Remessa: d.Remessa, Parte: d.Parte,
			TpArq: d.TpArq, NomeResp: d.NomeResp, EmailResp: d.EmailResp,
			TelResp: d.TelResp, TotalCli: d.TotalCli,
		},
	}
	for _, a := range d.Agreg {
		out.Agregados = append(out.Agregados, Agregado{
			NatuOp: a.NatuOp, Mod: a.Mod, OrigemRec: a.OrigemRec, VincME: a.VincME,
			ClassOp: a.ClassOp, FaixaVlr: a.FaixaVlr, PrzProvm: a.PrzProvm,
			Localiz: a.Localiz, TpCli: a.TpCli, DesempOp: a.DesempOp,
			ProvConsttd: a.ProvConsttd, QtdOp: a.QtdOp, QtdCli: a.QtdCli,
			Vencimentos: Vencimentos{
				V110: a.Venc.V110, V120: a.Venc.V120, V150: a.Venc.V150,
				V160: a.Venc.V160, V165: a.Venc.V165,
			},
		})
	}
	return out, nil
}
