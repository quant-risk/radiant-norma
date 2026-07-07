# Sprint 35 — AuditDDR 2070 Fase 1 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 35 (AuditDDR 2070 — Requerimento Capital Diário)
> **Pré-requisito:** v3.34.11 (commit 900b141 / tag v3.34.11) — Sprint 33/34 (Audit3050) TOTALMENTE FECHADO
> **Marco esperado:** 0 → 11 regras DDR 2070 (100% catálogo DDR)

## 🎯 Escopo Fase 1

Esta fase entrega:
1. **Doc2070 + DDR struct** + tipos auxiliares.
2. **ParseDoc2070** — parser XML best-effort.
3. **11 regras DDR 2070** (severity E/A/I conforme catálogo).
4. **Builtin2070() *Registry** — registry paralelo.
5. **Testes table-driven** (~10 funções).

## 📋 Catálogo DDR 2070 (11 regras)

| Cod BACEN | Sev | Descrição resumida | Base |
|---|---|---|---|
| 4678 | I | Exposição líquida RWAJUR2/3/4 consistente com DRM | DRM |
| 4679 | I | Descasamento vertical RWAJUR2/3/4 consistente com DRM | DRM |
| 4680 | I | Descasamento horizontal dentro zona RWAJUR2/3/4 consistente com DRM | DRM |
| 4681 | I | Descasamento horizontal entre zonas RWAJUR2/3/4 consistente com DRM | DRM |
| 4682 | I | Exposição bruta RWACOM consistente com DRM | DRM |
| 4684 | I | VaR (RWAJUR1) consistente com DRM | DRM |
| 4685 | I | sVaR (RWAJUR1) consistente com DRM | DRM |
| 4686 | I | Posições moedas DRM consistentes com DDR | DRM |
| 4693 | E | Patrimônio líquido exterior inconsistente | DDR |
| 4751 | I | Chaves duplicadas entre posição e moeda | DDR |
| 4763 | I | Saldo conta 770 DLO/2061 inconsistente com DDR | DLO |

## 📊 Análise de complexidade

**Regras baseadas em cross-doc (DRM/DLO):** 9 de 11 (4678-4686, 4763). Implementação completa requer parser DRM/DLO e queries cruzadas — fora do escopo da Fase 1. **Fase 1: stubs informativos** (severity I, return nil + comentário explicando dependência cross-doc).

**Regras DDR-internas:** 2 de 11 (4693, 4751). Implementáveis:
- **4693 (severity E):** valida que 181000 <= 161000 (soma posições vendidas).
- **4751 (severity I):** valida unicidade de chave (posição + moeda) na DDR.

**Stub carry-over:** 9 regras baseadas em cross-doc ficam como stubs informativos (severity I) até Fase 2 (parser DRM/DLO).

**Implementação Tier 1 (Fase 1):**
- Doc2070 + DDR struct
- ParseDoc2070 (best-effort)
- 4693 (severity E — Patrimônio líquido)
- 4751 (severity I — Chaves duplicadas)
- 9 stubs cross-doc (4678-4686, 4763)
- Builtin2070 com 11 regras
- ~10 testes table-driven

## 🏗️ Decisões técnicas

### DT-36 — Interface paralela Rule2070

Reutilizar pattern D-24/D-26/D-27 do 3050:
- `Rule2070 interface { Code/Sheet/Severity/Apply2070 }`
- `Registry.Register2070(rule Rule2070)`
- `Builtin2070() *Registry`

### DT-37 — Doc2070 com DDR achatada (D-25)

Estrutura similar a Doc3050:
```go
type Doc2070 struct {
    Root   Doc2070Root
    Diario []DDR // modalidades DDR achatadas
}

type Doc2070Root struct {
    CNPJ       string
    DataBase   string
    IndRemessa string
    NmContato  string
    TelContato string
}

type DDR struct {
    Codigo     string  // código exposição (161000, 181000, etc)
    Moeda      string
    Valor      *float64
    // outros campos conforme XSD
}
```

### DT-38 — Parser best-effort (D-26)

`ParseDoc2070(data []byte) (*Doc2070, error)` — segue pattern do `ParseDoc3050`.

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.11) | Pós esperado |
|---|---|---|
| Regras 2070 | 0 | **11** |
| Cobertura catálogo DDR | 0% | **100%** (11/11) |
| Coverage `internal/audit/rules` | 70.7% | **70-71%** |
| Test functions DDR | 0 | **~10** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar

```
backend/internal/audit/rules/2070.go              (NOVO — Doc2070 + parser + 11 regras)
backend/internal/audit/rules/2070_test.go         (NOVO — 10 testes table-driven)
backend/internal/audit/rules/registry.go          (D-36: +Rule2070 interface + Register2070 + Builtin2070)
CHANGELOG.md                                       (entry v3.34.12)
backend/SPRINT_35_RESEARCH.md                     (NOVO — este arquivo)
backend/SPRINT_35_RESULTS.md                      (NOVO — após implementação)
```

## 🎯 Self-verify

- [ ] `grep -c "Register2070("` = 11
- [ ] `grep -c "^func Test"` em 2070_test.go ≥ 10
- [ ] `go test -race ./...` 23/23 PASS
- [ ] Para 4693: testar com 161000=100, 181000=200 → erro (inconsistência)
- [ ] Para 4751: testar com chave duplicada → erro

## ⏭️ Após Fase 1

Sprint 35 Fase 2 (futuro):
- Parser DRM 2060 (cross-doc)
- Parser DLO 2061 (cross-doc)
- Implementar regras cross-doc (4678-4686, 4763) com queries reais