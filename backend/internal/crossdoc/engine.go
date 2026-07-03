// Engine orquestra validação cross-document.
//
// Sprint 6 v1.5.0 (Cross-Doc L3 — diferencial proprietário):
//
// Carrega múltiplos documentos em paralelo, executa regras cross-doc do
// Registry, agrega resultados em um ValidationResponse.
//
// Diferencial vs BCValidador: BCValidador valida UM CADOC por vez.
// Radiant Norma valida o ecossistema inteiro — checa consistência entre
// 3040, 4111, DRSAC etc.
package crossdoc

import (
	"context"
	"sync"
	"time"
)

// ValidationRequest é a entrada do engine.
type ValidationRequest struct {
	Cadocs map[string]string `json:"cadocs"` // cadoc_code → XML raw
}

// ValidationError de uma regra cross-doc.
type ValidationError struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ValidationResponse é a saída.
type ValidationResponse struct {
	Passed     bool              `json:"passed"`
	Errors     []ValidationError `json:"errors"`
	Warnings   []ValidationError `json:"warnings"`
	RulesRun   []string          `json:"rules_run"` // codes que executaram (sem skip)
	RulesSkip  []string          `json:"rules_skipped"`
	DurationMs int64             `json:"duration_ms"`
	ExecutedAt time.Time         `json:"executed_at"`
}

// Engine orquestra cross-doc validation.
type Engine struct {
	registry *Registry
}

func NewEngine(reg *Registry) *Engine {
	return &Engine{registry: reg}
}

// Validate executa todas as regras do registry em paralelo (limitado a
// numCPU goroutines) e agrega resultados.
//
// Estratégia:
//   - Carrega DocSet (validate input)
//   - Para cada regra: check RequiredDocs no DocSet
//   - Skip regra se algum doc obrigatório faltar
//   - Run regra em goroutine (limitado)
//   - WaitGroup espera todas terminarem
func (e *Engine) Validate(ctx context.Context, req *ValidationRequest) *ValidationResponse {
	start := time.Now()
	resp := &ValidationResponse{
		Passed:     true,
		Errors:     []ValidationError{},
		Warnings:   []ValidationError{},
		RulesRun:   []string{},
		RulesSkip:  []string{},
		ExecutedAt: start,
	}

	if req.Cadocs == nil {
		resp.Passed = false
		resp.DurationMs = time.Since(start).Milliseconds()
		return resp
	}

	docs := &DocSet{Cadocs: req.Cadocs}

	// Coleta regras aplicáveis
	var todo []CrossDocRule
	var skipped []string
	for _, rule := range e.registry.All() {
		if !allRequiredPresent(rule.RequiredDocs(), docs) {
			skipped = append(skipped, rule.Code())
			continue
		}
		todo = append(todo, rule)
	}

	// Executa em paralelo (limitado)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, rule := range todo {
		rule := rule
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := rule.Apply(ctx, docs)
			mu.Lock()
			defer mu.Unlock()
			resp.RulesRun = append(resp.RulesRun, rule.Code())
			if err != nil {
				if cdErr, ok := err.(*Error); ok {
					ve := ValidationError{
						Code:     cdErr.Code,
						Severity: cdErr.Severity,
						Message:  cdErr.Message,
					}
					if cdErr.Severity == "E" {
						resp.Errors = append(resp.Errors, ve)
					} else {
						resp.Warnings = append(resp.Warnings, ve)
					}
				} else {
					// Erro genérico → warning
					resp.Warnings = append(resp.Warnings, ValidationError{
						Code:     rule.Code(),
						Severity: rule.Severity(),
						Message:  err.Error(),
					})
				}
			}
		}()
	}
	wg.Wait()

	resp.RulesSkip = skipped
	resp.Passed = len(resp.Errors) == 0
	resp.DurationMs = time.Since(start).Milliseconds()
	return resp
}

// allRequiredPresent retorna true se todos CADOCs requeridos estão em docs.
func allRequiredPresent(required []string, docs *DocSet) bool {
	for _, c := range required {
		if !docs.Has(c) {
			return false
		}
	}
	return true
}
