# Sprint 34 — Carry-over 3050 Fase 6 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 34 (Carry-over 3050 — fechar em 100%)
> **Pré-requisito:** v3.34.9 (commit f497d0a / tag v3.34.9)
> **Marco esperado:** 153 → 170 regras 3050 (90% → **100%** cobertura catálogo TXB_V11)

## 🎯 Escopo Fase 6

17 regras adicionais para fechar 100%:

### Tier 1 — Lógica pura sem infra (4 regras)

| Cod | Sev | Regra | Origem |
|---|---|---|---|
| 3050-S12 | A | przMedCarteira condicional quando sldCarAtiva > 0 (refina S23) | 3025 |
| 3050-S14 | E | txMaxima > txMinima (cruzada, refina S12 stub) | 3055 |
| 3050-H19 | A | apenas 1 elemento `<referencia>` por doc (parser change) | formato |
| 3050-H20 | A | 1 elemento `<diario>` e 1 `<mensal>` por referencia (parser change) | formato |

### Tier 2 — Matriz modalidade × encargo adicionais (13 stubs)

| Cod | Sev | Regra |
|---|---|---|
| 3050-S71 | I | financBensVeiculos apenas prefixado |
| 3050-S72 | I | arrendamentoVeiculos apenas prefixado |
| 3050-S73 | I | leasingVeiculos apenas prefixado |
| 3050-S74 | I | credConsigFuncPublico apenas prefixado |
| 3050-S75 | I | credRuralCusteioInvestComerc consolidação |
| 3050-S76 | I | microFinancMicroCred consolidação |
| 3050-S77 | I | capGirTop rotativo IPCA/IGP-M bloqueio |
| 3050-S78 | I | imobResidComercFinancBens consolidação |
| 3050-S79 | I | financImobReformaApenasPref |
| 3050-S80 | I | cheqEspecialCheqPrefDat bloqueio |
| 3050-S81 | I | garantiasConsolidado (multi-modalidade) |
| 3050-S82 | I | matrizConsolidadaFinal (encerramento matriz 2001) |
| 3050-S83 | I | periodicidadeAnualBACEN (anual vs mensal) |

### Carry-over permanente (5 regras — não factíveis nesta sprint)

- **S02** (Doc não esperado) — precisa DB infra (`SELECT MAX(data_base) FROM envios WHERE if_id=X AND cadoc_code='3050'`)
- **S06** (Substituição sem original) — mesma infra
- **S10** (Doc anterior não enviado) — mesma infra
- **S36** (indRemessa=I apenas primeira vez) — mesma infra
- **S38** (Doc único por CNPJ+dataBase) — mesma infra

**Total:** 17 regras novas (4 Tier 1 + 13 Tier 2). 153 + 17 = **170 (100%)**.

## 🏗️ Decisões técnicas

### DT-34 — RawXML em Doc3050Root

Para implementar H19/H20 (contar elementos XML), parser precisa expor o XML bruto. Adicionar campo `RawXML []byte` em `Doc3050Root`, populado pelo parser:

```go
type Doc3050Root struct {
    CNPJ, DataBase, IndRemessa, NmContato, TelContato string
    Encoding string
    BomPresent bool
    RawXML []byte // NOVO — XML bruto pra validações estruturais
}
```

Parser: `doc.Root.RawXML = data` antes do loop.

### DT-35 — S12 e S14 (lógica pura)

**S12 (PrzMed se Sld):** quando sldCarAtiva > 0, przMedCarteira deve estar preenchido (já parcialmente coberto por S23 — S12 complementa para Mensal com sldBaiPrejuizo).

**S14 (txMaxima > txMinima):** quando ambos preenchidos, txMaxima > txMinima (regra 3055 — cruzadas).

### DT-36 — Matriz S71-S83 (stubs informativos)

Última onda de consolidações matriz 2001 (2001 × 134 → 32 + 13 = 45 stubs cobrindo combinações distintas). Cobre as combinações remanescentes que S39-S70 não representou (veículos, leasing, microFinanc, garantias, periodicidade anual).

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.9) | Pós esperado |
|---|---|---|
| Regras 3050 | 153 | **170** |
| Cobertura catálogo 3050 | 90% | **100%** |
| Coverage `internal/audit/rules` | 70.9% | **70-71%** |
| Test functions Fase 6 | 0 | **~10** (4 Tier 1 + integração) |
| Test functions total 3050 | 117 | **127** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3050.go              (+RawXML em Doc3050Root, +S12/S14/H19/H20/S71-S83 = 17 regras)
backend/internal/audit/rules/3050_fase6_test.go   (NOVO — 10 testes table-driven)
backend/internal/audit/rules/3050_test.go         (atualizar TestBuiltin3050_TotalRulesIs 153→170)
CHANGELOG.md                                       (entry v3.34.10)
backend/SPRINT_34_RESULTS.md                       (NOVO)
```

## 🎯 Self-verify

- [ ] `grep -c "Register3050"` = 170
- [ ] `grep -c "^func Test"` em 3050_fase6_test.go ≥ 10
- [ ] `go test -race ./...` 23/23 PASS
- [ ] Para S12/S14: testar com casos válidos e inválidos
- [ ] Para H19/H20: testar XML com 0, 1, e 2 elementos

## ⏭️ Após Fase 6 (Sprint 33/34 totalmente fechados)

Sprint 33 (Audit3050) fechado em 100%. Próxima sprint:
- **Sprint 35 — AuditDLO 2061** (próximo CADOC)
- **Sprint 36 — FrontendNext** (Next.js 15)
- **Sprint 37 — Carry-over infra** (DB historico_envios para S02/S06/S10/S36/S38)
EOF