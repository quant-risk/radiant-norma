// Registry agrega CrossDocRules indexadas por código.
//
// Mesmo padrão de rules.Registry — mas separa namespaces (XD-*)
package rules

import (
	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
)

// Convenience helpers — Registry é o tipo do package crossdoc raiz.
// Aqui só exportamos wrapper que cria Builtin cross-doc registry
// já populado com as regras iniciais.

// BuiltinRegistry retorna um *crossdoc.Registry pré-populado com
// todas as regras cross-doc (iniciais + DRSAC + 4111).
// Sprint 52 v3.34.33.
func BuiltinRegistry() *crossdoc.Registry {
	r := crossdoc.NewRegistry()
	RegisterInitialRules(r)
	RegisterDRSACCrossDocRules(r)
	Register4111CrossDocRules(r)
	return r
}

// RegisterInitialRules adiciona as 3 regras iniciais ao registry.
//
// Sprint 6 v1.5.0: regras de exemplo focadas em 3040 ↔ 4111 ↔ DRSAC.
func RegisterInitialRules(r *crossdoc.Registry) {
	r.Register(TotalOperacoes3040Consistente4111{})
	r.Register(Modalidade0213FlagChequeEspecial{})
	r.Register(DRSACSubsegmentoClassificacaoRisco{})
}

// RegisterDRSACCrossDocRules registra as 8 regras cross-doc DRSAC↔SCR.
// Sprint 52 v3.34.33.
func RegisterDRSACCrossDocRules(r *crossdoc.Registry) {
	r.Register(XDDR01IPOCExistsInSCR{})
	r.Register(XDDR02SaldoConsistente{})
	r.Register(XDDR03ClienteExisteNoSCR{})
	r.Register(XDDR04SetorCNAEConsistente{})
	r.Register(XDDR05RiscoSocialAlto{})
	r.Register(XDDR06RiscoAmbiental{})
	r.Register(XDDR07TotalTVMConsistente{})
	r.Register(XDDR08ContribPositivaGreen{})
}

// Register4111CrossDocRules registra as 5 regras cross-doc 4111↔3040.
// Sprint 52 v3.34.33.
func Register4111CrossDocRules(r *crossdoc.Registry) {
	r.Register(XD4111CNPJConsistente{})
	r.Register(XD4111TotalClientesvsOps{})
	r.Register(XD4111Inadimplentesvs3040{})
	r.Register(XD4111DataBaseConsistente{})
	r.Register(XD4111Zeradovs3040{})
}
