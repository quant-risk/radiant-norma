# ADR-0006: Cross-Doc Engine — diferencial proprietário (L3)

> **Status:** Aceito
> **Data:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

BCValidador oficial valida 1 documento por vez. Matera, Mitra, cadoc.ai também. Nenhum valida **o ecossistema inteiro** (3040 ↔ 4111 ↔ DRSAC ↔ 3050 ↔ 2160 ↔ 2170).

Isso é uma dor real: IF submete 3040 OK, submete 4111 OK, mas no relatório consolidado BACEN aparece divergência que gerou multa. Hoje a IF descobre isso **quando BACEN recusa o relatório consolidado** — tarde demais.

## Decisão

Engine dedicada de cross-doc validation que executa regras inter-CADOC em paralelo, com panic recovery por regra (validação 12) e auditoria completa.

**Arquitetura:**

```go
type CrossDocRule interface {
    Code() string
    Description() string
    Severity() string
    RequiredDocs() []string  // cadoc_codes necessários
    Apply(ctx context.Context, docs *DocSet) ([]CrossDocError, error)
}

type DocSet struct {
    docs map[string]Document  // cadoc_code -> parsed
}

type Engine struct {
    registry *Registry
    logger   *slog.Logger
}

func (e *Engine) Validate(ctx context.Context, req *ValidationRequest) *ValidationResponse {
    docs := &DocSet{Cadocs: req.Cadocs}

    var todo []CrossDocRule
    var skipped []string
    for _, rule := range e.registry.All() {
        if !allRequiredPresent(rule.RequiredDocs(), docs) {
            skipped = append(skipped, rule.Code())
            continue
        }
        todo = append(todo, rule)
    }

    // Parallel execution com panic recovery
    var mu sync.Mutex
    var wg sync.WaitGroup
    for _, rule := range todo {
        rule := rule
        wg.Add(1)
        go func() {
            defer wg.Done()
            defer func() {
                if r := recover(); r != nil {
                    logger.Error("crossdoc rule panic recovered", "rule", rule.Code(), "panic", r)
                    mu.Lock()
                    resp.Errors = append(resp.Errors, ValidationError{
                        Code: rule.Code(), Severity: "E",
                        Message: "internal error (recovered from panic)",
                    })
                    mu.Unlock()
                }
            }()
            errs := rule.Apply(ctx, docs)
            mu.Lock()
            defer mu.Unlock()
            // ... append results
        }()
    }
    wg.Wait()

    resp.RulesSkip = skipped
    resp.Passed = len(resp.Errors) == 0
    resp.DurationMs = time.Since(start).Milliseconds()
    return resp
}
```

**Re cross-doc target (Sprint 43):**

| Code | Cadocs | Descrição |
|---|---|---|
| XD01 | 3040 ↔ 4111 | Saldo 3040 = Saldo 4111 (mesma data-base) |
| XD02 | 3040 ↔ 3050 | Total 3040 = Somatório 3050 (modalidade × UF) |
| XD03 | 2160 ↔ 2170 | LCR ≥ 100% (consistência LCR/NSFR) |
| XD04 | 3040 ↔ DRSAC | IPOC × saldo 3040 × setor DRSAC coerente |
| XD05 | 4111 ↔ DRSAC | Operações ESG-classificadas coerentes |
| XD06 | 3050 ↔ 2160 | APR usados no LCR |
| XD07 | 3040 ↔ 4111 ↔ 3050 | Triangulação completa |
| XD08 | 2061 ↔ 2070 | Limites operacionais vs requerimento capital |
| XD09 | 2160 ↔ 3040 | Liquidez × risco de crédito (estresse) |
| XD10 | DRSAC ↔ 3040 | Score ESG × taxa de inadimplência |
| XD11 | DRSAC ↔ 4111 | Operações ESG × contrapartes |
| XD12 | Todos | Data-base consistente entre CADOCs |

## Consequências

**Positivas:**
- ✅ **Moat competitivo único**: zero concorrentes têm.
- ✅ Catch de erros inter-CADOC **antes** do envio.
- ✅ Paralelização (goroutine pool) — latência P95 < 5s pra tripla (3040+4111+DRSAC).
- ✅ Audit log de execuções (rastreabilidade forense).
- ✅ Plugin architecture (registry) — IFs podem adicionar regras custom.

**Negativas:**
- ❌ Regras inter-CADOC exigem entendimento profundo de negócio regulatório.
- ❌ Se 1 CADOC está mal parseado, propaga erro pra todas as regras que dependem dele.
- ❌ Manutenção: cada nova regra exige conhecimento cross-domain.

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Validar pós-envio (relatório consolidado BACEN)** | Tarde demais. IF já foi multada. |
| **Stored procedures no DB** | Não portável, difícil de testar. |
| **LLM interpretando audit_log** | Complementar, não substituto. |

## Notas de implementação

- Cada regra implementa `RequiredDocs()` para pré-filtro (skip se docs ausentes).
- Timeout por regra: 5s default, configurável.
- Resultado inclui `rules_run`, `rules_skipped`, `errors`, `warnings` — estrutura idêntica ao validate single-CADOC pra UX consistente.
- Audit emission: `crossdoc.validated` com cadocs + passed + counts.
- Open source da engine (community contributions de regras).