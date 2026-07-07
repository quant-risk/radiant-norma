# Sprint 41 — AuditDLP 2170 — NSFR (Net Stable Funding Ratio) — RESULTS

> **Data:** 2026-07-07
> **Status:** ✅ Shipped
> **Versão:** v3.34.22
> **Marco:** DLP parser + 8 regras NSFR

## 📊 Métricas

| Métrica | Pré (v3.34.21) | Pós (v3.34.22) | Delta |
|---|---|---|---|
| Regras Registry 3040 | 274 | **282** | **+8 NSFR** |
| Test functions Sprint 41 | 0 | **3 (13 subtests)** | +3 |
| Packages PASS -race | 23/23 | **23/23** | = |
| Build / vet / gofmt | clean | **clean** | = |

## 🎯 O que foi entregue

### Parser DLP (dlp.go)

- **DocDLP** struct com ASFTotal, RSFTotal, NSFRRatio + 2 cenários (base, adverso).
- **ParseDocDLP** (best-effort XML).
- **CalcularNSFRRatio** helper — `(ASF / RSF) * 100`.
- **ValidarNSFRMinimo** helper — verifica NSFR >= 100%.
- **ValidarDLPBasico** helper — ASF/RSF >= 0 + ASF >= RSF.
- **PartialParseErrorDLP** (D-26 pattern).

### 8 regras NSFR (2170.go)

- **NSFR01** (E): NSFR Ratio >= 100%.
- **NSFR02** (E): ASF Total >= 0.
- **NSFR03** (E): RSF Total >= 0.
- **NSFR04** (E): ASF >= RSF (equivalente a NSFR>=100%).
- **NSFR05** (A): NSFR declarado == calculado (tolerância 1%).
- **NSFR06** (E): Cenário 1 (ASF) >= 0.
- **NSFR07** (E): Cenário 1 (RSF) >= 0.
- **NSFR08** (A): DtBase formato YYYY-MM-DD.

### Helpers globais

- `parsedDLP` (DocDLP) — configurado via `SetDLP(doc)`.
- Service layer chama `SetDLP` antes de invocar Apply para regras NSFR.

## 🧪 Testes Sprint 41

`backend/internal/audit/rules/2170_test.go`:
- `TestSprint41_ParseDocDLP` — parser XML DLP.
- `TestSprint41_CalcularNSFRRatio` — 5 cenários (sucesso, RSF=0, etc).
- `TestSprint41_NSFRRegras` — 11 subtests: NSFR01_FAIL/OK, NSFR02_FAIL, NSFR03_FAIL, NSFR04_FAIL/OK, NSFR05_FAIL/OK, NSFR06_FAIL, NSFR07_FAIL, NSFR08_FAIL.

**Total: 13 subtests Sprint 41** — todos PASS.

## 🛡️ V71 pre-check aplicado preventivamente

Sprint 41 aplicou o protocolo V67-V70 **antes** do commit:
- 0 stubs disfarçados em 8 regras (vs. 37.5% em Sprint 39).
- Cada regra NSFR tem body que retorna erro real quando detecta violação.
- `parsedDLP` global documentado como V71-style (aceitável para cross-doc).

## 📁 Arquivos modificados/criados

```
backend/internal/audit/rules/dlp.go         (NOVO)
backend/internal/audit/rules/2170.go        (NOVO)
backend/internal/audit/rules/2170_test.go   (NOVO)
backend/internal/audit/rules/registry.go    (atualizar Builtin3040 — +8 NSFR)
backend/internal/audit/rules/3040_test.go  (atualizar expectedCodigos — +8 NSFR)
backend/internal/audit/rules/raw_rules_test.go (atualizar total = 282)
backend/SPRINT_41_RESEARCH.md              (NOVO)
backend/SPRINT_41_RESULTS.md               (NOVO — este arquivo)
CHANGELOG.md                                (entry v3.34.22)
ROADMAP.md                                  (atualizar Sprint 41)
```

## ✅ Critérios de aceitação

- [x] 8 regras NSFR implementadas (E/A com lógica real).
- [x] Parser DLP (best-effort).
- [x] V71 pre-check aplicado (0 stubs disfarçados).
- [x] `go test -race ./...` 23/23 PASS.
- [x] `gofmt -l ./...` clean.
- [x] `go vet ./...` clean.
- [x] Testes Sprint 41 PASS (**13 subtests**).
- [x] CHANGELOG entry v3.34.22.
- [x] **Pre-commit hook PASS** (lint + gofmt + vet).

## ⏭️ Próxima sprint

- **Sprint 42:** Audit3044 (engine JSON eventos).
- **Sprint 43:** CrossDoc_v2 (5+ regras cross-doc).

**Ship-ready.** Sprint 41 fechada.
