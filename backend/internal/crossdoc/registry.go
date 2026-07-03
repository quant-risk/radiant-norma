// Registry agrega CrossDocRules indexadas por código.
//
// Mesmo padrão de rules.Registry do package audit/rules — mas separado
// em package raiz (crossdoc) para que engine e rules possam usar.
package crossdoc

// Registry indexa regras cross-document.
type Registry struct {
	rules map[string]CrossDocRule
}

func NewRegistry() *Registry {
	return &Registry{rules: make(map[string]CrossDocRule)}
}

func (r *Registry) Register(rule CrossDocRule) {
	r.rules[rule.Code()] = rule
}

func (r *Registry) Get(code string) CrossDocRule {
	return r.rules[code]
}

func (r *Registry) All() []CrossDocRule {
	out := make([]CrossDocRule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, rule)
	}
	return out
}

// Codes retorna códigos registrados.
func (r *Registry) Codes() []string {
	out := make([]string, 0, len(r.rules))
	for k := range r.rules {
		out = append(out, k)
	}
	return out
}
