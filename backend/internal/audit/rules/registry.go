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
	NatuOp     string // 01 = própria, 02 = cobrados
	Mod        string // modalidade BACEN
	OrigemRec  string
	VincME     string // S/N
	ClassOp    string // A,B,C,D,E,F,G,H
	FaixaVlr   string
	PrzProvm   string // S/N
	Localiz    string // UF
	TpCli      string // 1=PF, 2=PJ
	DesempOp   string // 01 a vencer, 02 vencida 15-30, etc
	ProvConsttd string
	QtdOp      string
	QtdCli     string

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

// Registry agrega regras indexadas por código.
type Registry struct {
	rules map[string]Rule
}

// NewRegistry cria um registry vazio.
func NewRegistry() *Registry {
	return &Registry{rules: make(map[string]Rule)}
}

// Register adiciona uma regra.
func (r *Registry) Register(rule Rule) {
	r.rules[rule.Code()] = rule
}

// Get retorna a regra de um código (ou nil se não existir).
func (r *Registry) Get(code string) Rule {
	return r.rules[code]
}

// Codes retorna todos os códigos registrados.
func (r *Registry) Codes() []string {
	out := make([]string, 0, len(r.rules))
	for k := range r.rules {
		out = append(out, k)
	}
	return out
}

// All retorna todas as regras.
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
//   - Básicas: B06-B15 (contadores, limites)
//   - Formato: F01-F05 (taxa, data, contrato, conglomerado, RefBacen)
//   - Campos Obrigatórios: C01-C05 (obrigatoriedade condicional)
//   - Semântica: S01-S05 (semântica geral)
//
// Total: 25 regras. Adicionar mais em sprints seguintes.
func Builtin3040() *Registry {
	r := NewRegistry()

	// Básicas B06-B15
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

	// Formato F01-F05
	r.Register(F01TaxaEfetivaAnual{})
	r.Register(F02Datas{})
	r.Register(F03CodigoContrato{})
	r.Register(F04Conglomerado{})
	r.Register(F05RefBacenSicor{})

	// Campos Obrigatórios C01-C05
	r.Register(C01CamposObrigatoriosPJ{})
	r.Register(C02CamposNaoObrigatorios{})
	r.Register(C03GarantiasNaoFidejussorias{})
	r.Register(C04GarantiasFidejussorias{})
	r.Register(C05CessoesCoobrigacao{})

	// Semântica S01-S05
	r.Register(S01DetalhamentoCliente{})
	r.Register(S02VendorInfo{})
	r.Register(S03Ocultacao{})
	r.Register(S04CreditoALiberar{})
	r.Register(S05LimiteCredito{})

	return r
}