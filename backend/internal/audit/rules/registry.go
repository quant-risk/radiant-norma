// Package rules implementa regras de validação semânticas portadas dos
// catálogos BACEN para execução em Go.
//
// Cada CADOC tem seu próprio arquivo (3040.go, 3050.go, etc) com regras
// tipadas. O Registry agrega todas elas e o service.go consulta via codigo.
//
// Convenção de naming:
//
//	CODIGO_<NOME>  ex: F02_DT_BASE
//
// Cada regra implementa a interface Rule:
//
//	type Rule interface {
//	    Code() string                       // "F02"
//	    Sheet() string                      // "Formato"
//	    Severity() string                   // "E" | "A" | "I"
//	    Apply(ctx, doc *Doc3040) error      // nil se OK
//	}
package rules

import (
	"context"
)

// Doc3040 é o documento CADOC 3040 parseado.
type Doc3040 struct {
	Root      Doc3040Root
	Agregados []Agregado
	Operacoes []Operacao // Sprint 32 Fase 3 — operações individuais
}

// Doc3040Root é a tag <Doc3040>.
type Doc3040Root struct {
	DtBase    string // YYYY-MM
	CNPJ      string // 8 dígitos
	Remessa   string
	Parte     string
	TpArq     string // F (full) ou S (substituição)
	NomeResp  string
	EmailResp string
	TelResp   string
	TotalCli  string
}

// Agregado é a tag <Agreg>.
type Agregado struct {
	NatuOp      string // 01 = própria, 02 = cobrados
	Mod         string // modalidade BACEN
	OrigemRec   string
	VincME      string // S/N
	ClassOp     string // A,B,C,D,E,F,G,H
	FaixaVlr    string
	PrzProvm    string // S/N
	Localiz     string // UF
	TpCli       string // 1=PF, 2=PJ
	DesempOp    string // 01 a vencer, 02 vencida 15-30, etc
	ProvConsttd string
	QtdOp       string
	QtdCli      string

	Vencimentos Vencimentos
}

// Vencimentos é a tag <Venc>.
type Vencimentos struct {
	V110 string // até 14 dias
	V120 string // 15-30
	V150 string // 31-60
	V160 string // 61-90
	V165 string // > 90
}

// Operacao representa uma operação individual dentro do documento 3040.
// Sprint 32 Fase 3 — adicionado para suportar C11-C30, S13/S14, I01-I15.
//
// Nem todas as regras operam em Operacao. Regras de Agregado continuam
// usando Agregado. Regras individuais (I-*) usam Operacao + Cli.
type Operacao struct {
	Inf         string // código da informação adicional (0303, 0304, 0701, etc)
	Contrt      string // código do contrato
	IPOC        string // código IPOC (I14 — unicidade na remessa)
	Valor       string // valor contratado/negociado/recomprado
	Perc        string // percentual de coobrigação
	DtContr     string // data contratação (YYYY-MM-DD)
	DtVencOp    string // data vencimento operação
	ClassOp     string // classificação individual
	ProvConsttd string // provisão individual

	Vencimentos Vencimentos // V110-V165 individuais

	// Lista de identificadores de garantidores fidejussórios
	// (S13 — "garantidor fidejussório ≠ próprio cliente")
	Garantidores []string

	// Lista de parcelas (S12 — DtVencOp compatível com fluxo de parcelas)
	Parcelas []Parcela

	// Cliente individual (I-rules). Nil se operação sem cliente explícito.
	Cli *Cli
}

// Cli representa cliente individual em uma operação.
// Sprint 32 Fase 3 — I-rules precisam Cd (CPF/CNPJ) + TpCli + IPOC.
type Cli struct {
	Cd    string // 11 dígitos PF / 8 dígitos PJ
	TpCli string // 1=PF, 2=PJ
	IPOC  string // código IPOC
}

// Parcela representa uma parcela individual de uma operação.
// Sprint 32 Fase 3 — S12 valida DtVencOp >= max(DtVenc das parcelas).
type Parcela struct {
	Num    int
	DtVenc string
	Valor  string
}

// Rule é a interface de uma regra de validação.
type Rule interface {
	Code() string
	Sheet() string
	Severity() string // E (Erro bloqueante), A (Aviso), I (Informativo)
	Apply(ctx context.Context, doc *Doc3040) error
}

// RawRule é a interface para regras que operam no XML bruto, sem parser
// tipado. Usado para B01-B05 (básicas estruturais: declaração XML, tamanho).
//
// Sprint 6 v1.5.0 (W3): movidas do hardcode em audit/service.go::applyRegra
// para registro unificado. Mantemos interface separada para evitar refactor
// das 25 regras já portadas (que operam em *Doc3040 tipado).
type RawRule interface {
	Code() string
	Sheet() string
	Severity() string
	ApplyRaw(ctx context.Context, xmlContent string) error
}

// RawRuleFunc adapter permite usar func como RawRule.
type RawRuleFunc struct {
	C       string
	Sht     string
	Sev     string
	ApplyFn func(ctx context.Context, xmlContent string) error
}

func (r RawRuleFunc) Code() string     { return r.C }
func (r RawRuleFunc) Sheet() string    { return r.Sht }
func (r RawRuleFunc) Severity() string { return r.Sev }
func (r RawRuleFunc) ApplyRaw(ctx context.Context, s string) error {
	return r.ApplyFn(ctx, s)
}

// Registry agrega regras indexadas por código.
type Registry struct {
	rules     map[string]Rule
	rawRules  map[string]RawRule
	rules3050 map[string]Rule3050 // Sprint 33 Fase 1 — regras CADOC 3050
	rules2070 map[string]Rule2070 // Sprint 35 Fase 1 — regras CADOC 2070 (DDR)
	rules3044 map[string]Rule3044 // Sprint 42 — regras CADOC 3044 (JSON)
}

// NewRegistry cria um registry vazio.
func NewRegistry() *Registry {
	return &Registry{
		rules:     make(map[string]Rule),
		rawRules:  make(map[string]RawRule),
		rules3050: make(map[string]Rule3050),
		rules2070: make(map[string]Rule2070),
		rules3044: make(map[string]Rule3044),
	}
}

// Register adiciona uma regra tipada (*Doc3040).
func (r *Registry) Register(rule Rule) {
	r.rules[rule.Code()] = rule
}

// RegisterRaw adiciona uma regra que opera em XML bruto.
//
// Sprint 6 v1.5.0 (W3): usado para B01-B05 que não precisam de parser
// tipado do 3040.
func (r *Registry) RegisterRaw(rule RawRule) {
	r.rawRules[rule.Code()] = rule
}

// Register3050 adiciona uma regra tipada para CADOC 3050 (TXB).
//
// Sprint 33 Fase 1 (D-24): interface paralela Rule3050 para não quebrar
// Rule existente (3040). Registry indexa ambos em maps separados.
func (r *Registry) Register3050(rule Rule3050) {
	r.rules3050[rule.Code()] = rule
}

// Get3050 retorna a regra 3050 por código.
func (r *Registry) Get3050(code string) Rule3050 {
	return r.rules3050[code]
}

// Codes3050 retorna todos os códigos 3050 registrados.
func (r *Registry) Codes3050() []string {
	out := make([]string, 0, len(r.rules3050))
	for k := range r.rules3050 {
		out = append(out, k)
	}
	return out
}

// All3050 retorna todas as regras 3050 (útil para inventário).
func (r *Registry) All3050() []Rule3050 {
	out := make([]string, 0, len(r.rules3050))
	_ = out // unused
	out3050 := make([]Rule3050, 0, len(r.rules3050))
	for _, r := range r.rules3050 {
		out3050 = append(out3050, r)
	}
	return out3050
}

// Get retorna a regra tipada (Doc3040) por código.
func (r *Registry) Get(code string) Rule {
	return r.rules[code]
}

// GetRaw retorna a regra raw (XML bruto) por código.
func (r *Registry) GetRaw(code string) RawRule {
	return r.rawRules[code]
}

// Register2070 adiciona uma regra tipada para CADOC 2070 (DDR).
//
// Sprint 35 Fase 1 — DT-36: interface paralela Rule2070.
func (r *Registry) Register2070(rule Rule2070) {
	r.rules2070[rule.Code()] = rule
}

// Get2070 retorna a regra 2070 por código.
func (r *Registry) Get2070(code string) Rule2070 {
	return r.rules2070[code]
}

// Codes2070 retorna todos os códigos 2070 registrados.
func (r *Registry) Codes2070() []string {
	out := make([]string, 0, len(r.rules2070))
	for k := range r.rules2070 {
		out = append(out, k)
	}
	return out
}

// All2070 retorna todas as regras 2070 (útil para inventário).
func (r *Registry) All2070() []Rule2070 {
	out := make([]Rule2070, 0, len(r.rules2070))
	for _, r := range r.rules2070 {
		out = append(out, r)
	}
	return out
}

// Register3044 adiciona uma regra tipada para CADOC 3044 (JSON).
//
// Sprint 42: interface Rule3044 para regras de eventos (T01-T19).
func (r *Registry) Register3044(rule Rule3044) {
	r.rules3044[rule.Code()] = rule
}

// Get3044 retorna a regra 3044 por código.
func (r *Registry) Get3044(code string) Rule3044 {
	return r.rules3044[code]
}

// Codes3044 retorna todos os códigos 3044 registrados.
func (r *Registry) Codes3044() []string {
	out := make([]string, 0, len(r.rules3044))
	for k := range r.rules3044 {
		out = append(out, k)
	}
	return out
}

// All3044 retorna todas as regras 3044 (útil para inventário).
func (r *Registry) All3044() []Rule3044 {
	out := make([]Rule3044, 0, len(r.rules3044))
	for _, r := range r.rules3044 {
		out = append(out, r)
	}
	return out
}

// Codes retorna todos os códigos registrados (tipadas + raw).
func (r *Registry) Codes() []string {
	out := make([]string, 0, len(r.rules)+len(r.rawRules))
	for k := range r.rules {
		out = append(out, k)
	}
	for k := range r.rawRules {
		out = append(out, k)
	}
	return out
}

// All retorna todas as regras (tipadas — útil para inventário).
func (r *Registry) All() []Rule {
	out := make([]Rule, 0, len(r.rules))
	for _, r := range r.rules {
		out = append(out, r)
	}
	return out
}

// Builtin3040 retorna o registry com as regras 3040 implementadas.
//
// Cobre:
//   - Básicas raw: B01-B05 (opera em XML bruto, sem parser tipado)
//   - Básicas tipadas: B06-B15 (contadores, limites)
//   - Formato: F01-F05 (taxa, data, contrato, conglomerado, RefBacen)
//   - Campos Obrigatórios: C01-C05 (obrigatoriedade condicional)
//   - Semântica: S01-S05 (semântica geral)
//
// Total pré-Sprint 32: 60 regras (Sprint 7b v1.7.0).
// Sprint 32 Fase 1: +14 regras Agregadas (A01-A07, A09-A15) → 74 regras.
// Sprint 32 Fase 2: +5 regras Sistemáticas (S12 stub, S15, S17, S19, S20) → 79 regras.
// Sprint 32 Fase 3: +19 regras Individuais/Campos Op/Header (C11-C20, S13, S14, I01-I05, I11, H01-H03) → 98 regras.
// Sprint 32 Fase 4: +28 regras (C31-C40/C51-C55/S21-S46/S69-S70 — 14 completas + 14 stubs) → 126 regras.
// Sprint 36 Fase 2: +51 regras (C21-C30/C41-C50/C56-C70/H04-H09/N01-N10 — 23 completas + 28 stubs I) → 177 regras.
// V67: recontagem — 23 reais (severity E/A detectam violação) + 28 stubs (severity I retornam nil).
// Híbridas (severity I com lógica parcial): C23, C43, C64.
// Sprint 37 Fase 3: +44 novas (I06-I15/A16-A30/S71-S90) + 5 destravadas sobrescrevem stubs originais
// (C44/C46/C57/C62/C68 stub → C44Destravada/.../C68Destravada real). Total Registry: 221.
// Sprint 38 Fase 4: +45 novas (C71-C90/SUB01-SUB15/X01-X10) + 9 destravadas sobrescrevem stubs (Sprint 36-37).
// Total Registry: 275 (última sprint de expansão do 3040).
//
// Cobertura catálogo: 275/361 = 76.2%.
//
// Cobertura catálogo: 177/361 = 49.0%.
func Builtin3040() *Registry {
	r := NewRegistry()

	// Básicas raw B01-B05 (Sprint 6 v1.5.0 / W3)
	r.RegisterRaw(B01ArquivoXMLValido{})
	r.RegisterRaw(B02EstruturaBasica{})
	r.RegisterRaw(B03TamanhoArquivo{})
	r.RegisterRaw(B04CodificacaoDeclarada{})
	r.RegisterRaw(B05ArquivoNaoVazio{})

	// Básicas B06-B15 (Sprint 4)
	r.Register(B06RemessaIncompativel{})
	r.Register(B07ComposicaoRemessa{})
	r.Register(B08ParteRejeitada{})
	r.Register(B09MaxErros{})
	r.Register(B10MaxAvisos{})
	r.Register(B11NaoAceitoAnterior{})
	r.Register(B12TpFundoObrigatorio{})
	r.Register(B13IFNaoFIDC{})
	r.Register(B14MaxOpDivergentes3042{})
	r.Register(B15MaxOpDivergentes3040{})

	// Formato F01-F05 (Sprint 4)
	r.Register(F01TaxaEfetivaAnual{})
	r.Register(F02Datas{})
	r.Register(F03CodigoContrato{})
	r.Register(F04Conglomerado{})
	r.Register(F05RefBacenSicor{})

	// Campos Obrigatórios C01-C05 (Sprint 4)
	r.Register(C01CamposObrigatoriosPJ{})
	r.Register(C02CamposNaoObrigatorios{})
	r.Register(C03GarantiasNaoFidejussorias{})
	r.Register(C04GarantiasFidejussorias{})
	r.Register(C05CessoesCoobrigacao{})

	// Semântica S01-S05 (Sprint 4)
	r.Register(S01DetalhamentoCliente{})
	r.Register(S02VendorInfo{})
	r.Register(S03Ocultacao{})
	r.Register(S04CreditoALiberar{})
	r.Register(S05LimiteCredito{})

	// Sprint 7b / v1.7.0 — Básicas expandidas B16-B25 (10 regras)
	r.Register(B16TotalizadoresCoerentes{})
	r.Register(B17DtBaseFormato{})
	r.Register(B18TpArqValido{})
	r.Register(B19EmailValido{})
	r.Register(B20TelefoneValido{})
	r.Register(B21CNPJRaiz{})
	r.Register(B22NomeRespObrigatorio{})
	r.Register(B23MinimoUmAgregado{})
	r.Register(B24DtBaseNaoFutura{})
	r.Register(B25QtdOperacoesPositivo{})

	// Sprint 7b / v1.7.0 — Formato expandido F06-F15 (10 regras)
	r.Register(F06ClassOpValido{})
	r.Register(F07ModalidadeValida{})
	r.Register(F08NatuOpValido{})
	r.Register(F09UFLocaliz{})
	r.Register(F10VincMEOp{})
	r.Register(F11PrzProvm{})
	r.Register(F12TpCliValido{})
	r.Register(F13DesempOpValido{})
	r.Register(F14FaixaVlrValida{})
	r.Register(F15OrigemRecValida{})

	// Sprint 7b / v1.7.0 — Campos Obrigatórios C06-C10 (5 regras)
	r.Register(C06ProvConsttd{})
	r.Register(C07VencimentosObrigatorio{})
	r.Register(C08EmailParaContato{})
	r.Register(C09TotalCliObrigatorio{})
	r.Register(C10ClassOpObrigatorio{})

	// Sprint 7b / v1.7.0 — Semânticas S06-S10 (5 regras)
	r.Register(S06QtdOpZero{})
	r.Register(S07Mod0213Risco{})
	r.Register(S08PFRiscoMin{})
	r.Register(S09SomaVencimentos{})
	r.Register(S10PropriaNaoME{})

	// Sprint 32 / v3.25.0 — Agregadas A01-A15 (14 regras; A08 não existe no catálogo)
	// Classificação × Provisão × Vencimentos (Tier 3 — Agregadas)
	r.Register(A01ClassOpProvisao{})
	r.Register(A02ClassOpVencSemPrazo{})
	r.Register(A03ClassOpVencComPrazo{})
	r.Register(A04MinimoVencimento{})
	r.Register(A05NatuOpLocaliz{})
	r.Register(A06DesempOpVenc{})
	r.Register(A07AgregadoDuplicado{})
	// A08 não consta no catálogo BACEN scr3040_criticas (pula para A09)
	r.Register(A09FaixaVlrMedia{})
	r.Register(A10QtdOpMaiorQtdCli{})
	r.Register(A11FaixaAltVencMedioBaixo{})
	r.Register(A12FaixaAltRiscoMedio{})
	r.Register(A13RiscoMedioMin{})
	r.Register(A14LocalizExterior{})
	r.Register(A15AgregadoDuplicadoCompleto{})

	// Sprint 32 / v3.27.0 Fase 2 — Sistemáticas S12, S15, S17, S19, S20 (5 regras; S11/S13/S14/S16/S18 não implementadas nesta fase)
	// S12 é stub pass-through (carry-over Fase 3: precisa Operacao.Parcelas)
	r.Register(S12DtVencCompativelParcelas{})
	r.Register(S15DtContrNaoFutura{})
	r.Register(S17CdTamanhoPorTpCli{})
	r.Register(S19DtBaseMinima{})
	r.Register(S20VencimentosHH{})

	// Sprint 32 / v3.29.0 Fase 3 — Individuais + Campos Op + Header (19 regras)
	// Adicionou Doc3040.Operacoes []Operacao + Operacao.Cli *Cli + Operacao.Parcelas []Parcela
	// C11-C20 (8 regras subset — carry-over C21/C23-C29)
	r.Register(C11DtVencObrigatoria{})
	r.Register(C13Inf0303Cessao{})
	r.Register(C14Inf0305Renegociacao{})
	r.Register(C16Inf0307{})
	r.Register(C17Inf04XX{})
	r.Register(C18Inf05XX{})
	r.Register(C19Inf0701{})
	r.Register(C20Inf0702{})
	// S13, S14 (Sistemáticas individuais)
	r.Register(S13GarantidorNaoCliente{})
	r.Register(S14DtVencMaiorDtContr{})
	// I01, I02, I03, I04, I05, I11 (Individualizadas subset)
	r.Register(I01ClassOpProvisaoIndividual{})
	r.Register(I02ClassOpVencIndividual{})
	r.Register(I03CliTpCliUnico{})
	r.Register(I04ContratoModalidadeUnico{})
	r.Register(I05VencimentosUnicos{})
	r.Register(I11CliNaoNatuOp32{}) // stub — carry-over Fase 4 (precisa NatuOp em Operacao)
	// H01, H02, H03 (Header)
	r.Register(H01TpArqValido{})
	r.Register(H02CNPJRaiz{})
	r.Register(H03TotalCliPositivo{})

	// Sprint 32 / v3.30.0 Fase 4 — Fechamento Sprint 32 (subset C31-C55, S21-S46, S69-S70)
	// 28 regras: 14 completas + 14 stubs com carry-over documentado
	// C31-C40 (subset de 14) — FatAnual, Perc Indexador, Inf 1201-1203
	r.Register(C31FaturamentoObrigatorio{})
	r.Register(C32PercIndexadorObrigatorio{})
	r.Register(C33DiasAtrasoObrigatorio{}) // stub — requer DiaAtraso em Operacao
	r.Register(C34Inf1201Coobrigacao{})
	r.Register(C35Inf1201Obrigatorio{})
	r.Register(C36IdentCedenteObrigatorio{})
	r.Register(C37Inf1202{})
	r.Register(C38Pacote1512{}) // stub — parser cruzamento pacotes
	r.Register(C39Inf1203{})
	r.Register(C40Inf1201CdIdent{})
	// C51-C55 (5 regras) — Inf específicas adicionais
	r.Register(C51Inf0313{})
	r.Register(C52Inf04Excluindo0406{})
	r.Register(C54Inf18XX{})
	r.Register(C55Inf1999{})
	// S21-S46 (subset) — Modalidade × natureza + Ident formato
	r.Register(S21Mod15SemVenc310{})
	r.Register(S22Mod1511NaoPF{})
	r.Register(S25CNPJCabecalhoDiferente{})
	r.Register(S26NatuOp02TemInf{})  // stub — requer NatuOp em Operacao
	r.Register(S33Inf0101Natureza{}) // stub — idem
	r.Register(S34CdCessao{})        // stub — cruzamento original/cedida
	r.Register(S41IdentCNPJ8Digitos{})
	r.Register(S42CedenteIgualCabecalho{})
	r.Register(S43CedenteIgualCliente{})
	r.Register(S44CaractEsp35{}) // stub — requer CaractEsp em Operacao
	r.Register(S45IdentCPFouCNPJ{})
	r.Register(S46CdFormatoData{})
	// S69-S70 (Fechamento)
	r.Register(S69ClassOpHHProvZero{})
	r.Register(S70IntramesDtContr{}) // stub — requer DtIntrames

	// Sprint 36 / v3.34.13 Fase 2 — Expansão 3040 (51 regras: 30 reais + 21 stubs I)
	// C21-C30: Campos Obrigatórios adicionais (Inf 0101, 0308, 0313, 0501, 0703-1101)
	r.Register(C21Inf0101NatuOp01{})
	r.Register(C22Inf0308Garantia{})
	r.Register(C23Inf0313Perc{})
	r.Register(C24Inf0501Reneg{})
	r.Register(C25Inf0703DtLib{})
	r.Register(C26Inf0704Refin{})
	r.Register(C27Inf0801Vinculo{})
	r.Register(C28Inf0901Rural{})
	r.Register(C29Inf1001Habit{})
	r.Register(C30Inf1101Leasing{})
	// C41-C50: Campos Opcionais com condicionalidade (10 regras; 9 reais, 1 stub)
	r.Register(C41ClassOpPorMod{})
	r.Register(C42ProvConsttdClassOp{})
	r.Register(C43VencimentosPrazo{})
	r.Register(C44LocalizPF{})
	r.Register(C45VincMEMod{})
	r.Register(C46OrigemRecBNDES{})
	r.Register(C47FaixaVlrClassOp{})
	r.Register(C48PrzProvmClassOp{})
	r.Register(C49TpCliQtdCli{})
	r.Register(C50DesempOpVenc{})
	// C56-C70: Campos cross-doc / cross-Operacao (15 regras; 4 reais, 11 stubs)
	r.Register(C56Inf0213Rel0307{})
	r.Register(C57Inf0307Rel1201{})
	r.Register(C58IPOCUnicoRemessa{})
	r.Register(C59ContratoUnicoIPOCDt{})
	r.Register(C60DtContrSaneamento{})
	r.Register(C61DtVencPosContr{})
	r.Register(C62ClassOpIndAg{})
	r.Register(C63ProvIndAg{})
	r.Register(C64VencIndSomaAg{})
	r.Register(C65QtdCliIndAg{})
	r.Register(C66CliObrigInfI03{})
	r.Register(C67CliCdFormato{})
	r.Register(C68CliIPOCEqual{})
	r.Register(C69ParcelaDtVencOp{})
	r.Register(C70GarantidorFidej{})
	// H04-H09: Header (6 regras; 5 reais, 1 stub)
	r.Register(H04DtBasePeriodo{})
	r.Register(H05CNPJRaiz8Dig{})
	r.Register(H06RemessaNumerica{})
	r.Register(H07ParteNumerica{})
	r.Register(H08TpArqHeader{})
	r.Register(H09TotalCliSomaAg{})
	// N01-N10: Regras de Negócio (10 regras; 2 reais, 8 stubs)
	r.Register(N01CliUnicoRemessa{})
	r.Register(N02CliMesmoClassOp{})
	r.Register(N03LimitePorCli{})
	r.Register(N04ConcentracaoMod{})
	r.Register(N05LimiteBasileia{})
	r.Register(N06ProvMinClassOp{})
	r.Register(N07PrazoMax{})
	r.Register(N08CarenciaMin{})
	r.Register(N09IdadeCli{})
	r.Register(N10ConsolidacaoConglomerado{})

	// Sprint 37 / v3.34.15 Fase 3 — Expansão 3040 (50 regras: 41 reais + 9 stubs I)
	// I06-I15: Individualizadas (9 reais + 1 stub I15)
	r.Register(I06ContratoModPJ{})
	r.Register(I07IPOCCliUnico{})
	r.Register(I08ProvIndPositiva{})
	r.Register(I09VencIndClassA{})
	r.Register(I10IPOCFormato{})
	r.Register(I12CliIPOCIgualOpIPOC{})
	r.Register(I13DtVencJanela5Anos{})
	r.Register(I14IPOCBemFormado{})
	r.Register(I15LimitePF{}) // stub
	// A16-A30: Agregadas expandidas (15 reais)
	r.Register(A16ClassOpFaixaVlr{})
	r.Register(A17QtdOpSomaInd{})
	r.Register(A18QtdCliSomaInd{})
	r.Register(A19ModNatuOpValido{})
	r.Register(A20PrzProvmClassOp{})
	r.Register(A21LocalizUFValida{})
	r.Register(A22TpCliPFTemLocaliz{})
	r.Register(A23TpCliPJTemLocaliz{})
	r.Register(A24DesempOpValido{})
	r.Register(A25ClassOpAgIgualInd{})
	r.Register(A26NatuOp02OrigemRec{})
	r.Register(A27VincMEModME{})
	r.Register(A28FaixaVlrSeq{})
	r.Register(A29QtdCliExigeCampos{})
	r.Register(A30ProvAgSomaInd{})
	// S71-S90: Semântica expandida (16 reais + 4 stubs)
	r.Register(S71ValorPositivoQtdOp{})
	r.Register(S72PercRange{})
	r.Register(S73DtContrDentroAno{})
	r.Register(S74VencimentosNaoNegativos{})
	r.Register(S75TotalCliConsistente{})
	r.Register(S76ParteNumericaSeq{})
	r.Register(S77SubstituicaoRemessa{})
	r.Register(S78ClassOpPorModValido{}) // stub
	r.Register(S79DtBaseAtual{})         // stub parcial
	r.Register(S80QtdOpNaoNegativo{})
	r.Register(S81VencimentosOrdem{})
	r.Register(S82ValorMaiorVencimentos{})
	r.Register(S83QtdCliInteiro{})
	r.Register(S84CNPJCliConsolidado{}) // stub
	r.Register(S85CessaoCedente{})      // stub
	r.Register(S86DtVencCalc{})         // stub
	r.Register(S87QtdOpInteiro{})
	r.Register(S88VencimentosSoma{})
	r.Register(S89ClassOpVincME{})
	r.Register(S90RemessaUnicaDtBase{}) // stub
	// Carry-over destravadas (5 stubs Sprint 36 → reais Sprint 37)
	r.Register(C44LocalizPFDestravada{})
	r.Register(C46OrigemRecBNDESDestravada{})
	r.Register(C57Inf0307Rel1201Destravada{})
	r.Register(C62ClassOpIndAgDestravada{})
	r.Register(C68CliIPOCEqualDestravada{})

	// Sprint 38 / v3.34.17 Fase 4 — FECHAMENTO do 3040 (45 novas + 9 destravadas)
	// C71-C90: Campos Opcionais expandidos (10 reais + 10 stubs I)
	r.Register(C71Inf1301Comissao{})
	r.Register(C72Inf1302Tarifa{})
	r.Register(C73Inf1401Seguro{})
	r.Register(C74Inf1501IOF{})
	r.Register(C75Inf1601CustoAquisicao{})
	r.Register(C76Inf17XXGarantia{})
	r.Register(C77Inf18XXCoobrig{})
	r.Register(C78Inf19XXReestrut{})
	r.Register(C79Inf20XXXNovos{})
	r.Register(C80InfCrossRef03071201{})
	r.Register(C81DtContrNaoFuturo{})
	r.Register(C82DtVencAposContr{})
	r.Register(C83ValorPositivo{})
	r.Register(C84PercPropria{})
	r.Register(C85QtdParcelasPositivo{})
	r.Register(C86PercCoobrig{})
	r.Register(C87DtVencCalc{})
	r.Register(C88ValorPrincipalJuros{})
	r.Register(C89GarantiaFidej{})
	r.Register(C90CessaoCedenteCd{})
	// SUB01-SUB15: Substituição Parcial (7 reais + 8 stubs I)
	r.Register(SUB01SubstituicaoRemessa{})
	r.Register(SUB02SubstituicaoParte{})
	r.Register(SUB03DocumentosReferenciados{})
	r.Register(SUB04PreservaOperacoes{})
	r.Register(SUB05SubstituicaoInf{})
	r.Register(SUB06SubstituicaoMin1{})
	r.Register(SUB07SubstituicaoTotalF{})
	r.Register(SUB08HistoricoSubstituicoes{})
	r.Register(SUB09SubstPeriodoDiferente{})
	r.Register(SUB10SubstCNPJConsistente{})
	r.Register(SUB11PreservaCli{})
	r.Register(SUB12SubstDataLimite{})
	r.Register(SUB13SubstMultiplaOrdem{})
	r.Register(SUB14SubstAgregados{})
	r.Register(SUB15SubstCrossIF{})
	// X01-X10: Cross-doc básico (1 real + 9 stubs I)
	r.Register(X01CNPJCrossDoc{})
	r.Register(X02DtBaseCoerente{})
	r.Register(X03Ops30402042{})
	r.Register(X04Ops30402042Ag{})
	r.Register(X05CliUnicoCross{})
	r.Register(X06IPOCUnicoCross{})
	r.Register(X07VencimentosCross{})
	r.Register(X08ProvConsttdCross{})
	r.Register(X09Consolidacao3050{})
	r.Register(X10ModalidadeCross{})
	// Carry-over destravadas (9 stubs Sprint 36-37 → reais Sprint 38)
	r.Register(I15LimitePFDestravada{})
	r.Register(S78ClassOpPorModDestravada{})
	r.Register(S84CNPJCliConsolidadoDestravada{})
	r.Register(S85CessaoCedenteDestravada{})
	r.Register(S86DtVencCalcDestravada{})
	r.Register(S90RemessaUnicaDtBaseDestravada{})
	r.Register(N05LimiteBasileiaDestravada{})
	r.Register(N07PrazoMaxDestravada{})
	r.Register(N08CarenciaMinDestravada{})

	// Sprint 40 / v3.34.21 — AuditDRL 2160 (LCR — Liquidity Coverage Ratio)
	// 8 regras: LCR01-LCR08 (E/A com lógica real).
	r.Register(LCR01{})
	r.Register(LCR02{})
	r.Register(LCR03{})
	r.Register(LCR04{})
	r.Register(LCR05{})
	r.Register(LCR06{})
	r.Register(LCR07{})
	r.Register(LCR08{})

	// Sprint 41 / v3.34.22 — AuditDLP 2170 (NSFR — Net Stable Funding Ratio)
	// 8 regras: NSFR01-NSFR08 (E/A com lógica real).
	r.Register(NSFR01{})
	r.Register(NSFR02{})
	r.Register(NSFR03{})
	r.Register(NSFR04{})
	r.Register(NSFR05{})
	r.Register(NSFR06{})
	r.Register(NSFR07{})
	r.Register(NSFR08{})

	// Sprint 42 / v3.34.23 — Audit3044 (Engine JSON — Eventos de Operações)
	// 17 regras: T01-T19 (T18/T19 carry-over: dependem de DB lookup).
	r.Register3044(T01{})
	r.Register3044(T02{})
	r.Register3044(T03{})
	r.Register3044(T04{})
	r.Register3044(T05{})
	r.Register3044(T06{})
	r.Register3044(T07{})
	r.Register3044(T08{})
	r.Register3044(T11{})
	r.Register3044(T12{})
	r.Register3044(T13{})
	r.Register3044(T14{})
	r.Register3044(T15{})
	r.Register3044(T16{})
	r.Register3044(T17{})
	// T18, T19: carry-over (DB lookup — implementar quando DB layer pronto)
	r.Register3044(T18{})
	r.Register3044(T19{})

	return r
}
