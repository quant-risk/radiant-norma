# Phase 1.2 — Unificar parser + generator (canonical root tag)

> **Status**: shipped
> **Branch**: `remediation/gates-1-14`
> **Closes gate**: gate #3 from `RELATORIO_FINAL.md` ("validator deve usar generator como fonte canônica de root tags")
> **Benchmark coverage**: `VAL-ROOT-{2030,2060,...,4111}` and `GEN-*` round-trip tests.

## Contexto

Antes desta fase, o `Norma Audit` (`audit/service.go`) mantinha um mapa
`expectedRootTag()` hardcoded para cada CADOC. O generator (`generator/gen*/`)
produzia XML com root tags que podiam divergir deste mapa, criando um
descasamento entre o que o generator gerava e o que o validator esperava.

Exemplo do problema:
- Generator 4111 gerava `<Documento4111>`
- `expectedRootTag("4111")` retornava `"Documento4111"` (por coincidência)
- Mas se houvesse divergência futura (ex: generator mudasse para `<Documento>`),
  o validator aprovaria algo que não corresponde ao que foi gerado.

## Solução

Phase 1.2 centraliza a root tag canônica no `CADOCGenerator`:

1. **`CADOCGenerator.RootTag()`** adicionado à interface
2. **Cada generator implementa** `RootTag()` retornando sua root tag canônica
3. **`audit.Service`** recebe `*generator.Registry` via `SetGeneratorRegistry()`
4. **`expectedRootTag()`** vira método do `Service` que delega ao generator quando disponível
5. **Fallback legacy** mantêm comportamento para CADOCs sem generator

## Mudanças de código

### `backend/internal/generator/interface.go`
```go
type CADOCGenerator interface {
    // ... métodos existentes ...
    
    // RootTag retorna a tag raiz canônica do XML gerado.
    // O Norma Audit (L1 validator) usa este método como fonte canônica.
    RootTag() string
}
```

### Generators implementados
| Generator | Root Tag |
|-----------|----------|
| gen2030 | `DocumentoDRSAC` |
| gen2060 | `Doc2060` |
| gen2061 | `documentoDLO` |
| gen2062 | `documentoDLI` |
| gen2070 | `documentoDDR` |
| gen2160 | `documentoDRL` |
| gen2170 | `documentoDLP` |
| gen3040 | `Doc3040` |
| gen3050 | `DocTXB` |
| gen4111 | `Documento4111` |

### `backend/internal/audit/service.go`
```go
type Service struct {
    // ...
    genReg *generator.Registry // Sprint 57: generator registry (fonte canônica de root tags)
}

func (s *Service) SetGeneratorRegistry(r *generator.Registry) {
    s.genReg = r
}

func (s *Service) expectedRootTag(cadoc string) string {
    // Fase 1.2: usa generator como fonte canônica se disponível.
    if s.genReg != nil {
        g := s.genReg.Get(cadoc)
        if g != nil {
            return g.RootTag()
        }
    }
    // Fallback legacy para CADOCs sem generator.
    // ...
}
```

### `backend/cmd/api/main.go`
```go
// Phase 1.2: injeta generator registry no audit service.
audSvc.SetGeneratorRegistry(genReg)
```

## Evidência de testes

Os testes `TestPhase1_1_RejectsEmptyRootCadoc4111` mostram o comportamento correto:

```
=== RUN   TestPhase1_1_RejectsEmptyRootCadoc4111/<?xml_version="1.0"?><Documento/>
ERROR audit L1 parse failed cadoc=4111 err="esperado elemento raiz <Documento4111> mas tem <Documento>"
--- PASS: TestPhase1_1_RejectsEmptyRootCadoc4111/<?xml_version="1.0"?><Documento/> (0.00s)

=== RUN   TestPhase1_1_RejectsEmptyRootCadoc4111/<?xml_version="1.0"?><Documento4111/>
ERROR audit L1 parse failed cadoc=4111 err="documento Documento4111 vazio (sem atributos e sem elementos filhos)"
--- PASS: TestPhase1_1_RejectsEmptyRootCadoc4111/<?xml_version="1.0"?><Documento4111/> (0.00s)
```

O validator agora:
1. Rejeita `<Documento/>` (root tag errada) com mensagem clara
2. Rejeita `<Documento4111/>` vazio (empty doc, fail-closed)

## O que não está em Phase 1.2

- Phase 1.3: `/v1/validate` usa `ValidateFull` (L1→L4)
- Phase 1.4: whitelist de versão no generator
- Phase 1.5: required fields enforced + data_base obrigatório
- Phase 1.6: `/v1/validate` exige `data_base` e `versao_layout`
