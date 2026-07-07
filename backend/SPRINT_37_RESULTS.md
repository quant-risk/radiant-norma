# Sprint 37 — Audit3040 Fase 3 — RESULTS

> **Data:** 2026-07-07
> **Status:** ✅ Shipped
> **Versão:** v3.34.15
> **Marco:** 177 → 221 regras 3040 (49.0% → 61.2%)

## 📊 Métricas

| Métrica | Pré (v3.34.14) | Pós (v3.34.15) | Delta |
|---|---|---|---|
| Regras 3040 | 177 | **221** | **+44 + 5 destravadas** |
| Cobertura catálogo | 49.0% | **61.2%** | **+12.2 pp** |
| Coverage `internal/audit/rules` | 71.0% | **68.2%** | -2.8 pp¹ |
| Test functions Sprint 37 | 0 | **3 (32 subtests)** | +3 |
| Packages PASS -race | 23/23 | **23/23** | = |
| Build / vet / gofmt | clean | **clean** | = |

¹ Coverage caiu porque adicionei 49 regras mas só ~70% delas tem testes específicos. Recuperei de 66.8% (sem testes novos) para 68.2% com 32 subtests.

## 🎯 O que foi entregue

### 49 regras novas em `internal/audit/rules/3040_sprint37.go`

| Categoria | Códigos | Reais | Híbridas | Stubs I |
|---|---|---|---|---|
| Individualizadas | I06-I15 | 8 (I06-I10, I12-I14) | 0 | 1 (I15) |
| Agregadas expandidas | A16-A30 | 14 (A16-A18, A19-A30) | 0 | 0 |
| Semântica expandida | S71-S90 | 16 (S71-S77, S80-S83, S87-S89) | 1 (S79) | 3 (S78, S84-S86, S90) |
| Destravadas (override stubs) | C44, C46, C57, C62, C68 | 5 (todas com lógica) | 0 | 0 |
| **Total** | — | **43** | **1** | **5** |

> **Nota:** as 5 destravadas **sobrescrevem** as stubs originais com mesmo Code (Register indexa por Code). Total Registry final = 221 (5 raw + 216 tipadas), não 226.

### Helpers em `internal/audit/rules/3040_helpers.go`

Validadores reutilizáveis para evitar duplicação:
- `validarUF(string) bool` — 27 UFs brasileiras + EX.
- `validarIPOC(string) bool` — alfanumérico 8-20 chars.
- `validarModNatuOp(mod, natuOp) bool` — combinação regulamentar (50+ combinações).
- `validarPerc(float64) bool` — range [0, 100].
- `isVencimentoOrdemCronologica(Vencimentos) bool` — V110 < V120 < V150 < V160 < V165.
- `isFaixaVlrValida(string) bool` — 01-13.
- `addAnos(string, int) string` — adiciona anos a uma string YYYY.

## 🎯 Decisões técnicas

### DT-41 — Helpers centralizados

Validadores de UF, IPOC, Mod×NatuOp, Perc, Vencimentos, FaixaVlr ficaram em `3040_helpers.go` para reuso. Cada helper tem testes no `TestSprint37_Helpers`.

### DT-42 — Destravadas sobrescrevem stubs

A Sprint 37 marca 5 stubs Sprint 36 (C44, C46, C57, C62, C68) como "destravadas". Em vez de criar códigos novos (C44D etc), sobrescrevi as stubs com versões que detectam violação real. Filosofia: **uma regra por Code, melhor stub → real**.

### DT-43 — S79 como híbrida

S79DtBaseAtual é declarada como `severity "A"` mas o body não detecta violação (precisa data atual que não temos na struct). V67-style: declarei como híbrida com severidade I. No teste, removida da lista de stubs puros.

## 🧪 Testes Sprint 37

`backend/internal/audit/rules/3040_sprint37_test.go`:
- `TestSprint37_ReaisDetectamViolacoes` — 26 subtests: I06, I07, I08, I10, I12, I13, I14, A19, A21, A24, A29, S72, S76, S81, S83, S89, C44Destravada, C46Destravada, etc.
- `TestSprint37_StubsReturnNil` — 6 stubs Sprint 37: I15, S78, S84, S85, S86, S90.
- `TestSprint37_Helpers` — 6 subtests para validarUF, validarIPOC, validarModNatuOp, validarPerc, addAnos, isVencimentoOrdemCronologica, isFaixaVlrValida.

**Total: 32 subtests Sprint 37** — todos PASS.

## 📁 Arquivos modificados/criados

```
backend/internal/audit/rules/3040_sprint37.go         (NOVO — 49 regras)
backend/internal/audit/rules/3040_sprint37_test.go    (NOVO — 32 subtests)
backend/internal/audit/rules/3040_helpers.go         (NOVO — helpers)
backend/internal/audit/rules/registry.go              (modificado — Builtin3040 + comentário cobertura 221/361)
backend/internal/audit/rules/3040_test.go            (modificado — expectedCodigos)
backend/internal/audit/rules/raw_rules_test.go       (modificado — total = 221)
backend/SPRINT_37_RESEARCH.md                        (NOVO — planejamento)
backend/SPRINT_37_RESULTS.md                         (NOVO — este arquivo)
CHANGELOG.md                                        (entry v3.34.15)
ROADMAP.md                                          (atualizar Sprint 32/37)
```

## 🚫 Carry-over permanente (após Sprint 37)

Stubs Sprint 37 que permanecem (S78, S84-S86, S90, I15) cobrem ~75 regras do catálogo que dependem de:
- **Parser expandido** (Operacao.NatuOp, Cli.DtNascimento, Operacao.Cedente).
- **Cross-doc pesado** (DRM, DLO, histórico de remessas).
- **Catálogo de modalidades** (Rural, Habitacional, Leasing).
- **Limites regulatórios** (Basileia, CMN 4.966 com tabela dinâmica).
- **Data atual** (S79, validação de atraso de envio).

**Estimativa Sprint 38:** destravar mais 5-10 stubs com parser expandido, fechando 3040 em ~76% (275/361).

## ⚠️ Notas honestas

1. **5 destravadas sobrescrevem stubs** — não são "+5 regras adicionais", são versões melhores das 5 stubs existentes. Total Registry = 221 (não 226).
2. **Coverage caiu 71.0% → 68.2%** — Sprint adicionou muitas regras (>10% do package) e nem todas têm testes específicos.
3. **S79DtBaseAtual é híbrida** — declara severity "A" mas body retorna nil. Carry-over para parser de data atual.
4. **Cobertura efetiva** (regras com Apply() que detecta violação real): ~155/361 = **43.0%** (não 61.2% — 61% é cobertura "registrada").
5. **Carry-over permanente** estimado: ~75 regras (~21% do catálogo).

## ✅ Critérios de aceitação

- [x] 49 regras novas implementadas (43 reais + 1 híbrida + 5 stubs I).
- [x] 5 destravadas sobrescrevem stubs originais (intencional).
- [x] `go test -race ./...` 23/23 PASS.
- [x] `gofmt -l ./...` clean.
- [x] `go vet ./...` clean.
- [x] Coverage `internal/audit/rules` >= 65% (**68.2%**).
- [x] Comentário de cobertura em Builtin3040 atualizado (221/361).
- [x] Testes Sprint 37 PASS (**32 subtests**).
- [x] CHANGELOG entry v3.34.15.
- [x] Carry-over documentado com caminho de resolução.
- [x] **Pre-commit hook PASS** (lint + gofmt + vet).

**Ship-ready.** Sprint 37 fechada. Próxima: Sprint 38 Fase 4 (última do 3040).