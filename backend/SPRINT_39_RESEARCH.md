# Sprint 39 — AuditDDR Fase 2 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 39 (AuditDDR Fase 2 — parser DRM/DLO + cross-doc)
> **Pré-requisito:** v3.34.18 (V69 fechou drift do Sprint 38)
> **Marco:** 11 → 18+ regras DDR + parser DRM + parser DLO

## 🎯 Escopo Sprint 39 Fase 2

**Adicionar parsers cross-doc** para fechar DDR (CADOC 2070) com DRM (Risco Mercado) e DLO (Limites Operacionais):

1. **DocDRM** — subset de risco mercado (RWAJUR1-4, VaR, sVaR, RWACOM, Posições moedas).
2. **DocDLO** — subset de limites operacionais (Conta770, LimiteTotal, Patrimonio).
3. **7 regras cross-doc** entre DDR + DRM + DLO.
4. **Helper ValidadorDRMStrict** para uso futuro em service layer.

## 🏗️ Decisões técnicas

### DT-46 — Cross-doc via globais package-level

`parsedDRM` e `parsedDLO` são variáveis package-level, configuradas via `SetDRM(doc)` e `SetDLO(doc)`. Service layer chama antes de invocar Apply2070.

**Rationale:** evita adicionar interface Apply2070 com 2 params adicionais (mudança breaking). Globais são OK para este caso (single-threaded validation).

### DT-47 — Tolerância 10% para discrepâncias cross-doc

Cross-doc entre DDR e DRM/DLO não exige match exato (round-trip de XML para XML pode introduzir rounding). Tolerância 10% é razoável para razoável primeiros estágios.

### DT-48 — Parser best-effort com PartialParseError

Mesmo padrão do ParseDoc2070 (D-26). Tolerância a XML malformado parcial.

## 📋 Regras cross-doc (7)

| Cod | Sev | Regra |
|---|---|---|
| C4693-crossdoc | E | Patrimônio Líquido Exterior (DDR 161000+181000) vs DLO.Patrimonio |
| C4678-crossdoc | A | RWAJUR2+3+4 DDR vs DRM |
| C4679-crossdoc | A | Descasamento vertical vs DRM |
| C4684-crossdoc | A | VaR (RWAJUR1) vs DDR |
| C4685-crossdoc | A | sVaR vs DDR |
| C4686-crossdoc | E | Posições moedas DRM têm contraparte DDR |
| C4763-crossdoc | A | Saldo conta 770 DLO vs DDR |
| drm-strict | I | Helper ValidadorDRMStrict |

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.18) | Pós esperado |
|---|---|---|
| Regras DDR (2070) | 11 | **18** |
| Parsers cross-doc | 0 | **2** (DRM + DLO) |
| Test functions Sprint 39 | 0 | **4 (11 subtests)** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar

```
backend/internal/audit/rules/drm.go                       (NOVO — DocDRM + ParseDocDRM)
backend/internal/audit/rules/dlo.go                       (NOVO — DocDLO + ParseDocDLO)
backend/internal/audit/rules/2070_crossdoc.go             (NOVO — 7 regras + helper)
backend/internal/audit/rules/2070_crossdoc_test.go        (NOVO — 11 subtests)
backend/SPRINT_39_RESEARCH.md                            (NOVO — este arquivo)
backend/SPRINT_39_RESULTS.md                             (NOVO — após implementação)
```

## ⏭️ Após Sprint 39

- **Sprint 40:** AuditDRL (2160 LCR modelos II).
- **Sprint 41:** AuditDLP (2170 NSFR).
- **Sprint 42:** Audit3044 (engine JSON eventos).