# Sprint 43 — CrossDoc_v2 — Regras Cross-Documento DRL/DLP × 3044 — RESULTS

> **Data:** 2026-07-07
> **Status:** ✅ Shipped
> **Versão:** v3.34.24
> **Marco:** Cross-doc DRL/DLP × 3044 (XD01-XD08)

## 📊 Métricas

| Métrica | Pré (v3.34.23) | Pós (v3.34.24) | Delta |
|---|---|---|---|
| Regras cross-doc XD01-XD08 | 0 | **8** | +8 |
| Test subtests Sprint 43 | 0 | **16** | +16 |
| Packages PASS -race | 23/23 | **23/23** | = |
| Build / vet / gofmt | clean | **clean** | = |

## 🎯 O que foi entregue

### 8 regras Cross-Doc (crossdoc_liquidity.go)

| Cod | Sev | Descrição |
|---|---|---|
| **XD01** | E | CNPJ DRL == DLP == 3044 |
| **XD02** | E | DtBase DRL == DLP == dataSaldoDevedor 3044 |
| **XD03** | A | Soma saldoDevedor 3044 >= HQLA DRL |
| **XD04** | A | NSFR/LCR consistentes (LCR<80% E NSFR>120% = alerta) |
| **XD05** | A | Soma pagamentos 3044 <= Outflows DRL |
| **XD06** | E | IPOC em 3044 existe no histórico (carry-over) |
| **XD07** | E | Atraso 3044 consistente com DRL/DLP (carry-over) |
| **XD08** | A | Consistência prazos 3044 vs DRL/DLP (carry-over) |

### Helpers

- `parsed3044` global (set via `Set3044`).
- `Set3044(doc *Doc3044)` — configura 3044 para validações cross-doc.

## 🧪 Testes Sprint 43

`backend/internal/audit/rules/crossdoc_liquidity_test.go`:
- `TestSprint43_XD01_CNPJMismatch` — 3 subtests (DRL×3044 mismatch, DRL×DLP mismatch, OK)
- `TestSprint43_XD02_DataMismatch` — 3 subtests (DRL×DLP mismatch, DRL×3044 mismatch, OK)
- `TestSprint43_XD03_SaldoDevedorMaiorHQLA` — 2 subtests (soma>HQLA, soma<HQLA OK)
- `TestSprint43_XD04_InconsistenciaLCRNSFR` — 3 subtests (LCR<80% NSFR>120%, ambos baixos, OK)
- `TestSprint43_XD05_PagamentosMaiorOutflows` — 2 subtests (pag>Outflows, pag<Outflows OK)
- `TestSprint43_XD06_CarryOver` — 1 subtest
- `TestSprint43_XD07_CarryOver` — 1 subtest
- `TestSprint43_XD08_CarryOver` — 1 subtest

**Total: 16 subtests Sprint 43** — todos PASS.

## 🛡️ V74 pre-check aplicado preventivamente

- 0 stubs disfarçados em regras XD01-XD05 (lógica real).
- XD06/XD07/XD08: carry-over honesto (retornam nil explicitamente, documentado).
- `parsed3044` global: nomeado diferente de `parsedDRL`/`parsedDLP` para evitar conflito de declaração.

## 📁 Arquivos modificados/criados

```
backend/internal/audit/rules/crossdoc_liquidity.go   (NOVO — XD01-XD08 + Set3044)
backend/internal/audit/rules/crossdoc_liquidity_test.go (NOVO — 16 subtests)
backend/internal/version/version.go                   (3.34.24)
backend/SPRINT_43_RESEARCH.md                       (NOVO)
backend/SPRINT_43_RESULTS.md                       (NOVO — este arquivo)
CHANGELOG.md                                    (entry v3.34.24)
ROADMAP.md                                      (Sprint 43 ✅)
```

## ✅ Critérios de aceitação

- [x] 5 regras XD01-XD05 com lógica real.
- [x] XD06-XD08 carry-over honesto (documentado).
- [x] V74 pre-check aplicado (0 stubs disfarçados).
- [x] `go test -race ./...` 23/23 PASS.
- [x] `gofmt -l ./...` clean.
- [x] `go vet ./...` clean.
- [x] Testes Sprint 43 PASS (**16 subtests**).
- [x] CHANGELOG entry v3.34.24.
- [x] ROADMAP updated (Sprint 43 ✅).

## ⏭️ Próxima sprint

- **Sprint 44:** Radar_v2 (Diff semântico + auto-PR).
- **Sprint 45:** StripeBilling (Lite/Pro/Scale/Enterprise).

**Ship-ready.** Sprint 43 fechada.
