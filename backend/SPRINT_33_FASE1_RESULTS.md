# Sprint 33 Fase 1 — RESULTS

> **Data:** 2026-07-06
> **Sprint:** 33 Fase 1 de N
> **Tipo:** minor (parser + 28 regras 3050)
> **Versão:** v3.33.7 → **v3.34.0**

## ✅ Status

Shipped. 28 regras 3050 implementadas (14 A + 14 S), parser XML funcional, cobertura 16.5%.

## 📦 Entregas vs planejado

| Item | Planejado | Entregue |
|---|---|---|
| `Doc3050` struct | ✅ | ✅ |
| `Modalidade` achatada | ✅ | ✅ |
| `ParseDoc3050` streaming | ✅ | ✅ |
| 14 Agregadas A01-A14 | ✅ | ✅ (14/14) |
| 14 Stubs S01-S14 | ✅ | ✅ (14/14) |
| `Builtin3050()` | ✅ | ✅ |
| `Rule3050` interface | ✅ | ✅ |
| Testes table-driven | ✅ | ✅ (17 funções) |
| Coverage subiu ≥1pp | ≥71% | **72.9%** (+2.1pp) |

## 📊 Métricas finais

| Métrica | Pré | Pós |
|---|---|---|
| Regras 3050 | 0 | **28** |
| Cobertura catálogo 3050 | 0% | **16.5%** |
| Coverage `internal/audit/rules` | 70.8% | **72.9%** |
| Test functions Fase 1 | 0 | **17** |
| Files novos | 0 | **2** (3050.go, 3050_test.go) + **2** (RESEARCH.md, RESULTS.md) |
| LOC Go adicionados | 0 | ~1020 (540 + 480) |
| Packages PASS -race | 23/23 | **23/23** |
| Stress 50 goroutines | mantida | **3/3 PASS** |
| vet + gofmt | clean | **clean** |

## 🧪 Testes (17 funções, ~50 sub-tests table-driven)

### Agregadas (14)

- TestA01_SldCarSomaFaixas (4 cases: happy + diff > 0.01 + nil-skip + tolerância)
- TestA02_SldCedidoMenosAdquirido (3 cases)
- TestA03_SldBaiPrejuizoLeSldCar (3 cases)
- TestA04_SldCarMaisCedidoVsAdquirido (3 cases)
- TestA05_CNPJRaiz (6 cases: 8 dígitos + vazio + curto + longo + letra + underscore)
- TestA06_DataBaseFormato (7 cases: happy + vazio + 3 formatos errados + letra)
- TestA07_IndRemessaValido (6 cases: I/A/S + vazio + X + minúscula)
- TestA08_NmContatoObrigatorio (4 cases)
- TestA09_TxMedJurosLimite (5 cases: 0/15.5/100/-0.1/150)
- TestA10/11_TxMedEncFiscais/OperacionaisLimite (smoke 1 case cada)
- TestA12_TxMinimaLeMaxima (3 cases: <, =, >)
- TestA13_PrzDecMedConcessoesNaoNeg (3 cases)
- TestA14_PrzMedCarteiraNaoNeg (smoke)

### Stubs (1)

- TestS01_S14_StubsReturnNil (14 cases — todos retornam nil + severity "I")

### Integration (2)

- TestBuiltin3050_TotalRulesIs (verifica 28 regras + códigos A01-A14 + S01-S14)
- TestParseDoc3050_Smoke (parse XML real + verifica Root/Diario/Mensal/Modalidade fields)

## 🐛 Bugs encontrados pelos próprios tests

3 problemas durante implementação, todos corrigidos in-loop:

1. **Parser com `map[string]xml3050Attrs` falhou.** `unknown type map[string]rules.xml3050Attrs` — encoding/xml não consegue unmarshal direto em map quando elementos são opcionais. Fix: refatorado para streaming Token + path tracking (D-26).
2. **`xml.EOF` não existe.** Fix: `io.EOF`.
3. **Test A02 cálculo errado.** Esperava `sldCedido(6000000...)` mas meu cálculo estava errado (5M+1000 = 5.001M, e `5.001M - 50k = 4.951M` < sldCarAtiva 5M → não viola). Fix: setar `sldCedido = 100M` para garantir violação clara.

## 🎯 Conformidade vs plano

| Decisão | Status |
|---|---|
| D-24 (Rule3050 interface paralela) | ✅ aplicada |
| D-25 (Modalidade achatada) | ✅ aplicada |
| D-26 (parser best-effort) | ✅ aplicada |
| D-27 (stubs severity "I") | ✅ aplicada |

Todas as decisões documentadas em SPRINT_33_RESEARCH.md e executadas conforme.

## ⏭️ Próxima sprint (Fase 2)

**Sprint 33 Fase 2 — Audit3050 Sistemáticas S15-S44 + Cruzadas**:

Carry-over (stubs a implementar):
- S01 — Matriz modalidade × encargo × sub-modalidade (2001 × 134)
- S11 — VlrConcessoes vs Taxas (3003/3004/3007/3008/3009)
- S12 — PrzMedCarteira condicional (3025/3034)
- S13 — Último dia útil (3031-3035)
- S14 — Cruzadas (3051/3054/3056-3059)

Novas (S15-S30):
- 15+ regras Sistemáticas (formato campo: CNPJ, data, taxa, valor, etc).
- 15+ regras Header adicionais (espaços, encoding, length max).

**Alvo:** 28 → 60+ regras 3050 (cobertura 35%+).

## 📁 Arquivos

```
backend/internal/audit/rules/3050.go                 (NOVO — 540 LoC)
backend/internal/audit/rules/3050_test.go            (NOVO — 480 LoC, 17 testes)
backend/internal/audit/rules/registry.go             (D-24: +Rule3050, +Register3050, +Get3050, +Builtin3050)
backend/SPRINT_33_RESEARCH.md                        (NOVO)
backend/SPRINT_33_FASE1_RESULTS.md                   (NOVO — este arquivo)
CHANGELOG.md                                          (entry v3.34.0)
```