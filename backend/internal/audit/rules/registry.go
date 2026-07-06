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
	rules    map[string]Rule
	rawRules map[string]RawRule
}

// NewRegistry cria um registry vazio.
func NewRegistry() *Registry {
	return &Registry{
		rules:    make(map[string]Rule),
		rawRules: make(map[string]RawRule),
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

// Get retorna a regra tipada (Doc3040) por código.
func (r *Registry) Get(code string) Rule {
	return r.rules[code]
}

// GetRaw retorna a regra raw (XML bruto) por código.
func (r *Registry) GetRaw(code string) RawRule {
	return r.rawRules[code]
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
//
// Cobertura catálogo: 74/361 = 20.5% (era 16.6%).
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

	return r
}
