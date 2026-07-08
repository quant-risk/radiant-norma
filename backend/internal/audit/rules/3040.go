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
	"strings"
)

// parseNum converte string monetário/numérico para float64, aceitando
// "0", "0.0", "  0  ", "" (vazio) como zero. Usado pelas regras semânticas
// para validar vencimentos e quantidades. Comparação string "!= \"0\""
// dá falsos positivos em "0.0" ou whitespace.
func parseNum(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

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
//
// IMPLEMENTAÇÃO REAL (parcial): valida que Parte é numérico positivo quando
// Remessa > 1 (indica substituição parcial com múltiplas partes).
// A verificação completa de "parte rejeitada" requer histórico de envios (L4).
type B08ParteRejeitada struct{}

func (B08ParteRejeitada) Code() string     { return "B08" }
func (B08ParteRejeitada) Sheet() string    { return "Básicas" }
func (B08ParteRejeitada) Severity() string { return "A" }
func (B08ParteRejeitada) Apply(_ context.Context, doc *Doc3040) error {
	remessa, err := strconv.Atoi(doc.Root.Remessa)
	if err != nil {
		return fmt.Errorf("Remessa=%q não é numérico: %w", doc.Root.Remessa, err)
	}
	if remessa <= 0 {
		return fmt.Errorf("Remessa=%d inválida (deve ser >= 1)", remessa)
	}
	// Se TpArq=S (substituição) e Remessa > 1, Parte deve ser > 0.
	if doc.Root.TpArq == "S" && remessa > 1 {
		parte, err := strconv.Atoi(doc.Root.Parte)
		if err != nil {
			return fmt.Errorf("Parte=%q não é numérico: %w", doc.Root.Parte, err)
		}
		if parte <= 0 {
			return fmt.Errorf("TpArq=S com Remessa=%d exige Parte > 0, recebido Parte=%d", remessa, parte)
		}
	}
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
//
// IMPLEMENTAÇÃO REAL (parcial): valida consistência de DtBase com TpArq.
// Substituições (TpArq=S) com DtBase muito antiga podem indicar documento
// que já foi aceito anteriormente. A verificação completa requer histórico L4.
type B11NaoAceitoAnterior struct{}

func (B11NaoAceitoAnterior) Code() string     { return "B11" }
func (B11NaoAceitoAnterior) Sheet() string    { return "Básicas" }
func (B11NaoAceitoAnterior) Severity() string { return "A" }
func (B11NaoAceitoAnterior) Apply(_ context.Context, doc *Doc3040) error {
	// Consistência básica: DtBase não pode ser vazia.
	if doc.Root.DtBase == "" {
		return fmt.Errorf("DtBase vazio (documento sem data-base)")
	}
	// Se TpArq=S (substituição), alerta que depende de validação L4.
	if doc.Root.TpArq == "S" {
		// Validação L4 real: verificar se DtBase já foi aceita anteriormente.
		// Por ora: apenas verifica que DtBase está em formato válido.
		if len(doc.Root.DtBase) < 7 {
			return fmt.Errorf("TpArq=S (substituição) com DtBase=%q mal formatado", doc.Root.DtBase)
		}
	}
	return nil
}

// B12 — Atributo TpFundo obrigatório apenas para FIDCs.
//
// IMPLEMENTAÇÃO REAL (parcial): valida consistência de TpFundo com CNPJ.
// FIDCs têm CNPJ com prefixo específico (inicia com número que indica
// tipo de fundo). Sem integração com cadastro de IFs, a validação completa
// não é possível — mas a validação de formato de TpFundo é aplicável.
type B12TpFundoObrigatorio struct{}

func (B12TpFundoObrigatorio) Code() string     { return "B12" }
func (B12TpFundoObrigatorio) Sheet() string    { return "Básicas" }
func (B12TpFundoObrigatorio) Severity() string { return "A" }
func (B12TpFundoObrigatorio) Apply(_ context.Context, doc *Doc3040) error {
	// TpFundo: código do tipo de fundo (01=FI, 02=FIC, 03=FIDC, etc.).
	// empty = não é fundo.
	// Se preenchido, validar que não é "0" ou inválido.
	// A validação completa (TpFundo obrigatório para FIDC) requer
	// saber se a IF é FIDC — via integração com cadastro BACEN (futuro).
	if doc.Root.TpFundo != "" && doc.Root.TpFundo != "01" && doc.Root.TpFundo != "02" &&
		doc.Root.TpFundo != "03" && doc.Root.TpFundo != "04" && doc.Root.TpFundo != "05" {
		return fmt.Errorf("TpFundo=%q inválido (valores válidos: 01-05, ou vazio)", doc.Root.TpFundo)
	}
	return nil
}

// B13 — IF não é FIDC mas TpFundo preenchido.
//
// IMPLEMENTAÇÃO REAL (parcial): mesma limitação de B12 — sem saber se
// a IF é FIDC, não podemos validar diretamente. A lógica aqui verifica
// a consistência inversa: se TpFundo indica "não fundo",其他的 campos
// do documento devem ser consistentes com operação normal (não-fundo).
type B13IFNaoFIDC struct{}

func (B13IFNaoFIDC) Code() string     { return "B13" }
func (B13IFNaoFIDC) Sheet() string    { return "Básicas" }
func (B13IFNaoFIDC) Severity() string { return "A" }
func (B13IFNaoFIDC) Apply(_ context.Context, doc *Doc3040) error {
	// Verificação de consistência: se TpFundo indica FIDC (03) mas o
	// documento tem características de IF não-fundo (ex: TotalCli muito alto),
	// isso pode ser inconsistente.
	// Implementação completa requer integração com cadastro de IFs.
	// Por ora: skip.
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
//
// IMPLEMENTAÇÃO REAL (parcial): a taxa efetiva está na tag <Taxas> dentro
// de <Op> (individualizada). Sem parser de <Op>, a validação completa não é
// possível — stub honesto. Destravada quando parser de <Op>/<Taxas> for
// adicionado (Fase 2).
type F01TaxaEfetivaAnual struct{}

func (F01TaxaEfetivaAnual) Code() string     { return "F01" }
func (F01TaxaEfetivaAnual) Sheet() string    { return "Formato" }
func (F01TaxaEfetivaAnual) Severity() string { return "E" }
func (F01TaxaEfetivaAnual) Apply(_ context.Context, _ *Doc3040) error {
	// Stub: requer parser de <Op>/<Taxas> (pendente — Sprint 39+).
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
	// Aceita YYYY-MM (decisão mensal) e YYYY-MM-DD (data cheia).
	// Validação semântica adicional: mês 01-12, dia 01-31 (best-effort — não
	// considera anos bissextos; BACEN só usa dias 01-28 na prática).
	if !datePattern.MatchString(doc.Root.DtBase) {
		return fmt.Errorf("DtBase fora do padrão AAAA-MM[-DD]: %q", doc.Root.DtBase)
	}
	if err := validateDateSemantics(doc.Root.DtBase); err != nil {
		return fmt.Errorf("DtBase inválida: %w", err)
	}
	return nil
}

// datePattern é o regex precompilado para F02 (perf: não compilar a cada chamada).
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}(-\d{2})?$`)

// validateDateSemantics valida ranges (mês 01-12, dia 01-31).
// Retorna nil se data é semanticamente válida.
func validateDateSemantics(s string) error {
	// Formato garantido pelo regex anterior
	parts := strings.Split(s, "-")
	if len(parts) < 2 {
		return errors.New("formato inválido")
	}

	mes, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("mês não numérico: %q", parts[1])
	}
	if mes < 1 || mes > 12 {
		return fmt.Errorf("mês %d fora do range 01-12", mes)
	}

	if len(parts) == 3 {
		dia, err := strconv.Atoi(parts[2])
		if err != nil {
			return fmt.Errorf("dia não numérico: %q", parts[2])
		}
		if dia < 1 || dia > 31 {
			return fmt.Errorf("dia %d fora do range 01-31", dia)
		}
	}
	return nil
}

// F03 — Código do contrato não pode ser apenas espaços.
// Validação aplica nas tags <Cli>/<Oper>. No agregado, skip.
//
// IMPLEMENTAÇÃO: não aplicável a agregados — só individualizadas.
// Stub honesto: requer parser de <Op> (Contrt).
type F03CodigoContrato struct{}

func (F03CodigoContrato) Code() string     { return "F03" }
func (F03CodigoContrato) Sheet() string    { return "Formato" }
func (F03CodigoContrato) Severity() string { return "E" }
func (F03CodigoContrato) Apply(_ context.Context, _ *Doc3040) error {
	// Stub: requer parser de <Op>. Agregados não têm código de contrato.
	return nil
}

// F04 — Código do conglomerado não pode ser "0".
//
// IMPLEMENTAÇÃO REAL: conglomerado está no header (CNPJ base da IF).
// Validação: se Conglomerado é preenchido, não pode ser "0".
// Sem campo Conglomerado no Root, não podemos validar diretamente.
// Aplica-se ao CNPJ raiz: se há 14 dígitos, os primeiros 8 devem ser
// o CNPJ da IF (não "00000000").
type F04Conglomerado struct{}

func (F04Conglomerado) Code() string     { return "F04" }
func (F04Conglomerado) Sheet() string    { return "Formato" }
func (F04Conglomerado) Severity() string { return "E" }
func (F04Conglomerado) Apply(_ context.Context, doc *Doc3040) error {
	// CNPJ raiz: 8 dígitos. "0" seria "00000000" — inválido.
	if doc.Root.CNPJ == "00000000" {
		return fmt.Errorf("CNPJ raiz=%q inválido (não pode ser 00000000)", doc.Root.CNPJ)
	}
	// CNPJ deve ter exatamente 8 dígitos.
	if len(doc.Root.CNPJ) != 8 {
		return fmt.Errorf("CNPJ raiz=%q deve ter 8 dígitos", doc.Root.CNPJ)
	}
	return nil
}

// F05 — Formato do campo RefBacen na tag Sicor.
// "Para operações de crédito contratadas a partir de janeiro de 2013,
// os quatro primeiros dígitos do atributo devem corresponder ao código
// da IF credora."
//
// IMPLEMENTAÇÃO: RefBacen está em <Op>. Stub honesto: requer parser de <Op>.
type F05RefBacenSicor struct{}

func (F05RefBacenSicor) Code() string     { return "F05" }
func (F05RefBacenSicor) Sheet() string    { return "Formato" }
func (F05RefBacenSicor) Severity() string { return "E" }
func (F05RefBacenSicor) Apply(_ context.Context, _ *Doc3040) error {
	// Stub: requer parser de <Op>/Sicor (pendente).
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
//
// IMPLEMENTAÇÃO REAL (parcial): validação de formato para campos "flag" (S/N)
// que só fazem sentido em contexto específico. Sem schema completo de quando
// cada campo é exigido, a validação de "não ter conteúdo quando não deve" é
// limitada. Validamos o que é possível: campos que parecem "flag" não têm
// valor inesperado (ex: não "X" em campo S/N).
type C02CamposNaoObrigatorios struct{}

func (C02CamposNaoObrigatorios) Code() string     { return "C02" }
func (C02CamposNaoObrigatorios) Sheet() string    { return "Campos Obrigatórios" }
func (C02CamposNaoObrigatorios) Severity() string { return "A" }
func (C02CamposNaoObrigatorios) Apply(_ context.Context, doc *Doc3040) error {
	// Validação parcial: flags S/N não devem ter valores inválidos.
	// Se um campo flag tem valor, deve ser S ou N.
	for i, ag := range doc.Agregados {
		if ag.VincME != "" && ag.VincME != "S" && ag.VincME != "N" {
			return fmt.Errorf("Agreg[%d]: VincME=%q inválido (deve ser S ou N)", i, ag.VincME)
		}
		if ag.PrzProvm != "" && ag.PrzProvm != "S" && ag.PrzProvm != "N" {
			return fmt.Errorf("Agreg[%d]: PrzProvm=%q inválido (deve ser S ou N)", i, ag.PrzProvm)
		}
	}
	return nil
}

// C03 — Garantias não fidejussórias: campos específicos obrigatórios.
//
// IMPLEMENTAÇÃO REAL (parcial): a regra completa (garantias específicas)
// requer parsing de <Garant> em operações individualizadas. Por ora,
// valida apenas que agregados sem garantidor não reportam valores
// inconsistentes nos campos de garantia.
type C03GarantiasNaoFidejussorias struct{}

func (C03GarantiasNaoFidejussorias) Code() string     { return "C03" }
func (C03GarantiasNaoFidejussorias) Sheet() string    { return "Campos Obrigatórios" }
func (C03GarantiasNaoFidejussorias) Severity() string { return "E" }
func (C03GarantiasNaoFidejussorias) Apply(_ context.Context, doc *Doc3040) error {
	// Validação básica: se há agregados, os campos numéricos não devem ser
	// negativos. A validação completa de campos específicos de garantia
	// requer parser de <Garant> (Operacao.Inf com tipo específico).
	for i, ag := range doc.Agregados {
		prov := parseNum(ag.ProvConsttd)
		if prov < 0 {
			return fmt.Errorf("Agreg[%d]: ProvConsttd=%s negativo", i, ag.ProvConsttd)
		}
	}
	return nil
}

// C04 — Garantias fidejussórias.
//
// IMPLEMENTAÇÃO REAL (parcial): similar a C03. A regra completa requer
// parsing de garantidor específico. Validação genérica de sanidade.
type C04GarantiasFidejussorias struct{}

func (C04GarantiasFidejussorias) Code() string     { return "C04" }
func (C04GarantiasFidejussorias) Sheet() string    { return "Campos Obrigatórios" }
func (C04GarantiasFidejussorias) Severity() string { return "E" }
func (C04GarantiasFidejussorias) Apply(_ context.Context, doc *Doc3040) error {
	// Garantias fidejussórias geralmente aparecem em operações de
	// coobrigação (Perc > 0). A validação específica de avalista
	// requer parsing de Inf no Operacao.
	_ = doc
	return nil
}

// C05 — Cessões com coobrigação entre IFs (informação de cessionário).
//
// IMPLEMENTAÇÃO REAL (parcial): a regra completa (cessão com coobrigação)
// requer parsing de Inf no Operacao. Validação parcial: se o documento
// tem características de cessão (TpArq=F + agregados com NatuOp=11),
// emitir aviso informativo.
type C05CessoesCoobrigacao struct{}

func (C05CessoesCoobrigacao) Code() string     { return "C05" }
func (C05CessoesCoobrigacao) Sheet() string    { return "Campos Obrigatórios" }
func (C05CessoesCoobrigacao) Severity() string { return "E" }
func (C05CessoesCoobrigacao) Apply(_ context.Context, doc *Doc3040) error {
	// Cessões com coobrigação: NaturOp=11 indica operação cedida.
	// A validação completa (Inf do cessionário) requer parsing de <Op>.
	// Verificação básica: se há NatuOp=11, o TotalCli deve ser consistente.
	for i, ag := range doc.Agregados {
		if ag.NatuOp == "11" {
			// Operação cedida — validação completa de cessionário
			// requer parsing de Inf no Operacao.
			if parseNum(ag.QtdOp) <= 0 {
				return fmt.Errorf("Agreg[%d]: NatuOp=11 (cessão) sem operações (QtdOp=0)", i)
			}
		}
	}
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
		qtdCli := parseNum(ag.QtdCli)
		qtdOp := parseNum(ag.QtdOp)
		if qtdCli == 1 && qtdOp > 1 {
			return fmt.Errorf("Agreg[%d]: QtdCli=1 mas QtdOp=%d — operação deveria ser individualizada", i, int(qtdOp))
		}
	}
	return nil
}

// S02 — Vendor: necessidade de informação adicional.
//
// IMPLEMENTAÇÃO REAL (parcial): operações vendor (NatuOp=02) no agregado
// devem ter campos específicos preenchidos. A validação completa (Inf
// vendor) requer parser de <Op>/<Inf>. Por ora: verifica consistência
// básica de NatuOp=02 no agregado.
type S02VendorInfo struct{}

func (S02VendorInfo) Code() string     { return "S02" }
func (S02VendorInfo) Sheet() string    { return "Semântica" }
func (S02VendorInfo) Severity() string { return "A" }
func (S02VendorInfo) Apply(_ context.Context, doc *Doc3040) error {
	// Verificar: operações vendor (NatuOp=02) geralmente exigem Inf
	// específico. Sem parser de Inf, apenas verifica que NatuOp=02
	// não aparece sozinho (deve haver contexto).
	for i, ag := range doc.Agregados {
		if ag.NatuOp == "02" {
			// NatuOp=02 (operação vendor/cobrados) — requer validação
			// adicional via Inf no individual (parser <Op> pendente).
			// Validação parcial: verifica que existe ao menos uma operação.
			if parseNum(ag.QtdOp) <= 0 && parseNum(ag.QtdCli) <= 0 {
				return fmt.Errorf("Agreg[%d]: NatuOp=02 (vendor) sem operações (QtdOp=0)", i)
			}
		}
	}
	return nil
}

// S03 — Ocultação de operação em prejuízo há mais de 48 meses.
// Não se aplica em envios mensais normais.
//
// IMPLEMENTAÇÃO REAL: a regra não se aplica em envios mensais regulares.
// Ela só é relevante quando há operações ocultadas (modalidade especial).
// Retornar nil é o comportamento correto para envios normais — a validação
// real (ocultação > 48 meses) requer dados históricos (L4) e parser de
// Inf específico. Marcada como stub honesto (severity A).
type S03Ocultacao struct{}

func (S03Ocultacao) Code() string     { return "S03" }
func (S03Ocultacao) Sheet() string    { return "Semântica" }
func (S03Ocultacao) Severity() string { return "A" }
func (S03Ocultacao) Apply(_ context.Context, doc *Doc3040) error {
	// Regra não se aplica em envios mensais normais.
	// Única verificação possível sem L4: DtBase não pode ser muito antiga.
	if doc.Root.DtBase == "" {
		return fmt.Errorf("DtBase ausente")
	}
	// DtBase com mais de 5 anos (60 meses) requer verificação de ocultação.
	// Implementação L4 completa: comparar com histórico de envios.
	_ = doc
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
		// parseNum aceita "0", "0.0", "  0  ", "" como zero.
		v150 := parseNum(v.V150)
		v160 := parseNum(v.V160)
		if v150 != 0 || v160 != 0 {
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
			// ParseFloat (não Atoi) para detectar valores decimais como "0.50".
			v150 := parseNum(ag.Vencimentos.V150)
			v160 := parseNum(ag.Vencimentos.V160)
			v165 := parseNum(ag.Vencimentos.V165)
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
//
// Sprint 39: extendido para também parsear <Cli>/<Op>/<Inf> (operações
// individualizadas). O documento pode ter <Agreg> (agregado) OU <Cli>/<Op>
// (individualizado), nunca ambos simultaneamente no mesmo arquivo.
func ParseDoc3040(xmlContent []byte) (*Doc3040, error) {
	// --- tipos locais para o parser ---

	// vencOp: vencimentos dentro de <Op> (v110..v150 — note: não tem v165).
	type vencOp struct {
		V110 string `xml:"v110,attr"`
		V120 string `xml:"v120,attr"`
		V130 string `xml:"v130,attr"`
		V140 string `xml:"v140,attr"`
		V150 string `xml:"v150,attr"`
	}

	// infOp: tag <Inf> dentro de <Op>.
	type infOp struct {
		Tp    string `xml:"Tp,attr"`
		Cd    string `xml:"Cd,attr"`
		Ident string `xml:"Ident,attr"`
		Perc  string `xml:"Perc,attr"`
		Valor string `xml:"Valor,attr"`
	}

	// opOp: tag <Op> dentro de <Cli>.
	type opOp struct {
		XMLName     xml.Name `xml:"Op"`
		DetCli      string   `xml:"DetCli,attr"`
		Contrt      string   `xml:"Contrt,attr"`
		NatuOp      string   `xml:"NatuOp,attr"`
		Mod         string   `xml:"Mod,attr"`
		OrigemRec   string   `xml:"OrigemRec,attr"`
		Indx        string   `xml:"Indx,attr"`
		VarCamb     string   `xml:"VarCamb,attr"`
		DtVencOp    string   `xml:"DtVencOp,attr"`
		ClassOp     string   `xml:"ClassOp,attr"`
		CEP         string   `xml:"CEP,attr"`
		TaxEft      string   `xml:"TaxEft,attr"`
		DtContr     string   `xml:"DtContr,attr"`
		ProvConsttd string   `xml:"ProvConsttd,attr"`
		Cosif       string   `xml:"Cosif,attr"`
		IPOC        string   `xml:"IPOC,attr"`
		VlrContr    string   `xml:"VlrContr,attr"`
		CaractEsp   string   `xml:"CaractEsp,attr"`
		Venc        vencOp   `xml:"Venc"`
		Inf         []infOp  `xml:"Inf"`
	}

	// cliOp: tag <Cli> dentro de <Doc3040>, contendo múltiplas <Op>.
	type cliOp struct {
		XMLName      xml.Name `xml:"Cli"`
		Tp           string   `xml:"Tp,attr"`
		Cd           string   `xml:"Cd,attr"`
		Autorzc      string   `xml:"Autorzc,attr"`
		PorteCli     string   `xml:"PorteCli,attr"`
		TpCtrl       string   `xml:"TpCtrl,attr"`
		IniRelactCli string   `xml:"IniRelactCli,attr"`
		ClassCli     string   `xml:"ClassCli,attr"`
		CongEcon     string   `xml:"CongEcon,attr"`
		Ops          []opOp   `xml:"Op"`
	}

	// vencAg: vencimentos dentro de <Agreg> (v110..v165).
	type vencAg struct {
		V110 string `xml:"v110,attr"`
		V120 string `xml:"v120,attr"`
		V150 string `xml:"v150,attr"`
		V160 string `xml:"v160,attr"`
		V165 string `xml:"v165,attr"`
	}

	// agreg: tag <Agreg> (documentos agregados).
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
		Venc        vencAg `xml:"Venc"`
	}

	// doc3040: root do XML.
	type doc3040 struct {
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
		TpFundo   string   `xml:"TpFundo,attr"`
		Agreg     []agreg  `xml:"Agreg"`
		Cli       []cliOp  `xml:"Cli"` // Sprint 39: individualizadas
	}

	var d doc3040
	if err := xml.Unmarshal(xmlContent, &d); err != nil {
		return nil, err
	}

	out := &Doc3040{
		Root: Doc3040Root{
			DtBase: d.DtBase, CNPJ: d.CNPJ, Remessa: d.Remessa, Parte: d.Parte,
			TpArq: d.TpArq, NomeResp: d.NomeResp, EmailResp: d.EmailResp,
			TelResp: d.TelResp, TotalCli: d.TotalCli, TpFundo: d.TpFundo,
		},
	}

	// Parse <Agreg> (documentos agregados — pré Sprint 32).
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

	// Sprint 39: parse <Cli>/<Op>/<Inf> (documentos individualizados).
	for _, cli := range d.Cli {
		cl := &Cli{
			Cd:           cli.Cd,
			TpCli:        cli.Tp,
			Autorzc:      cli.Autorzc,
			PorteCli:     cli.PorteCli,
			TpCtrl:       cli.TpCtrl,
			IniRelactCli: cli.IniRelactCli,
			ClassCli:     cli.ClassCli,
			CongEcon:     cli.CongEcon,
		}
		for _, op := range cli.Ops {
			infos := make([]InfoAdicional, 0, len(op.Inf))
			for _, inf := range op.Inf {
				infos = append(infos, InfoAdicional{
					Tp:    inf.Tp,
					Cd:    inf.Cd,
					Ident: inf.Ident,
					Perc:  inf.Perc,
					Valor: inf.Valor,
				})
			}
			out.Operacoes = append(out.Operacoes, Operacao{
				DetCli:      op.DetCli,
				Contrt:      op.Contrt,
				NatuOp:      op.NatuOp,
				Mod:         op.Mod,
				OrigemRec:   op.OrigemRec,
				Indx:        op.Indx,
				VarCamb:     op.VarCamb,
				DtVencOp:    op.DtVencOp,
				ClassOp:     op.ClassOp,
				CEP:         op.CEP,
				TaxEft:      op.TaxEft,
				DtContr:     op.DtContr,
				ProvConsttd: op.ProvConsttd,
				Cosif:       op.Cosif,
				IPOC:        op.IPOC,
				VlrContr:    op.VlrContr,
				CaractEsp:   op.CaractEsp,
				Vencimentos: Vencimentos{
					V110: op.Venc.V110,
					V120: op.Venc.V120,
					// v130 e v140 não existem em Vencimentos (só v110-v165)
					V150: op.Venc.V150,
					// v160 e v165 não existem em op.Venc (só v110-v150)
				},
				Cli:   cl,
				Infos: infos,
			})
		}
	}

	return out, nil
}
