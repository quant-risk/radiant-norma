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

// BuiltinRegistry retorna um *crossdoc.Registry pré-populado com as
// 3 regras iniciais.
func BuiltinRegistry() *crossdoc.Registry {
	r := crossdoc.NewRegistry()
	RegisterInitialRules(r)
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

