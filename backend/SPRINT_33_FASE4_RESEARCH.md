# Sprint 33 Fase 4 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 33 Fase 4 (Fase final 3050)
> **Pré-requisito:** v3.34.5 (commit 1fb2e72 / tag v3.34.5) — drift validação corrigido
> **Marco esperado:** 80 → 116+ regras 3050 (47.06% → 68.2%+ cobertura)

## 🎯 Escopo Fase 4 (enxuto)

Esta fase entrega **36 novas regras** (Tier 1 + Tier 2) + 1 carry-over fix:

### Tier 1 (18 entregas — alta prioridade, garantido)

**Header H16-H20 (5 regras) — formato avançado:**
| Cod | Sev | Regra |
|---|---|---|
| 3050-H16 | E | encoding XML declarado = "UTF-8" |
| 3050-H17 | A | sem BOM UTF-8 nos primeiros 3 bytes |
| 3050-H18 | E | raiz XML = `<DocTXB>` |
| 3050-H19 | A | apenas 1 elemento `<referencia>` por doc |
| 3050-H20 | A | 1 elemento `<diario>` e 1 `<mensal>` por referencia |

**Sistema S33-S36 (4 regras) — periodicidade + sanity:**
| Cod | Sev | Regra |
|---|---|---|
| 3050-S33 | A | dataBase não pode ser > 1 ano atrás (sanity) |
| 3050-S34 | E | dataBase consistente entre Diario/Mensal (mesma dataBase implícita) |
| 3050-S35 | A | modalidades únicas por Codigo+Encargo+TipoCli dentro do mesmo periodo (refina S26) |
| 3050-S36 | I | stub: indRemessa=I apenas primeira vez (carry-over, precisa histórico) |

**Individuais I29-I36 (8 regras) — sub-modalidades específicas:**
| Cod | Sev | Regra |
|---|---|---|
| 3050-I29 | E | aquVeiculos vlrConcessoes ≥ 0 |
| 3050-I30 | E | arrMerVeiculos vlrConcessoes ≥ 0 |
| 3050-I31 | E | arrMerOutros vlrConcessoes ≥ 0 |
| 3050-I32 | E | capGirTetoRot sldCarAtiva ≥ 0 |
| 3050-I33 | E | chqEsp sldCarAtiva ≥ 0 |
| 3050-I34 | E | ctgGta sldCarAtiva ≥ 0 |
| 3050-I35 | E | FinancBens vlrConcessoes ≥ 0 |
| 3050-I36 | E | ccb przDec ≥ 0 |

**Carry-over fix:**
- IsUltimoDiaUtilMes edge case: se último dia do mês cai em sábado, retornar a sexta anterior como "último dia útil".

### Tier 2 (18 entregas — desejável se sobrar tempo)

**Header H21-H25 (5):**
- H21 txMedJuros max 4 decimais
- H22 vlrConcessoes max 2 decimais (R$)
- H23 qtdNovContratos inteiro
- H24 cnpj length max 8 (sanity, complementa H10)
- H25 nmContato sem caracteres de controle

**Sistema S37-S40 (4):**
- S37 primeiro envio sem AnteriorRef
- S38 substituição preserva qtdCli/qtdOp
- S39 DocTXB unico por CNPJ+dataBase (sanity)
- S40 txMedJuros > txMedJurosAjustada (consistência)

**Individuais I37-I46 (10):**
- I37-I40: contGarantida/capGir/credLiv/credConsig vlrConcessoes ≥ 0
- I41-I44: txMedJuros min ≥ 0 em 4 modalidades específicas
- I45-I46: sldBaiPrejuizo = 0 em 2 modalidades específicas

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.5) | Pós Tier 1 | Pós Tier 1+2 |
|---|---|---|---|
| Regras 3050 | 80 | **98** | **116** |
| Cobertura catálogo 3050 | 47.06% | **57.6%** | **68.2%** |
| Coverage `internal/audit/rules` | 72.5% | **71-73%** | **71-73%** |
| Test functions Fase 4 | 0 | **18** | **36** |
| Test functions total 3050 | 76 | **94** | **112** |
| Packages PASS -race | 23/23 | **23/23** | **23/23** |

## 🔧 Carry-over técnico

### Edge case IsUltimoDiaUtilMes

**Problema:** se último dia do mês cai em sábado (ex: 2025-05-31), BACEN real considera sexta anterior (2025-05-30) como último dia útil. Implementação atual retorna false para ambos, **não bate com semântica BACEN**.

**Fix:** alterar `IsUltimoDiaUtilMes` para:
1. Calcular último dia do mês
2. Se último dia é dia útil → retorna data == último dia
3. Se último dia NÃO é dia útil → voltar dias até achar último dia útil; retorna data == esse dia

```go
func IsUltimoDiaUtilMes(data time.Time) bool {
    ano, mes, _ := data.Date()
    primeiroProximo := time.Date(ano, mes+1, 1, 0, 0, 0, 0, time.UTC)
    ultimo := primeiroProximo.AddDate(0, 0, -1)
    // Encontra último dia útil do mês (varre do último dia do mês pra trás)
    for d := ultimo; !d.Before(time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)); d = d.AddDate(0, 0, -1) {
        if IsDiaUtilBACEN(d) {
            return data.Equal(d)
        }
    }
    return false // não deveria acontecer (sempre tem dia útil)
}
```

**Teste novo:** TestIsUltimoDiaUtilMes_EdgeCaseSabado (2025-05-31 = sábado → 2025-05-30 = sexta deve retornar true).

## 🏗️ Decisões técnicas

### DT-31 — Header avançado não precisa parser change

H16-H25 validam o `xml.Decoder` ou estrutura genérica do XML. Vou usar `encoding/xml.Decoder` em uma função helper `ValidateXMLStructure([]byte) error` que retorna issues como erro descritivo.

**Trade-off:** adiciona complexidade. Alternativa: validar no momento do parse (D-26 best-effort). Vou implementar uma validação leve em `Apply3050` das regras H, lendo `doc.Root` populado pelo parser.

Wait — `doc.Root` tem `CNPJ/DataBase/IndRemessa/NmContato/TelContato`. Não tem o encoding. Pra H16-H20 preciso estender Doc3050Root ou expor via novo campo.

**Solução pragmática:** estender Doc3050Root com `RawXML []byte` (opcional) OU implementar as regras H16-H20 dentro do parser (`ParseDoc3050` faz validações + retorna erro parcial).

**Decisão:** estender `Doc3050Root` com `Encoding string` (parsing do header `<?xml version="1.0" encoding="UTF-8"?>`). H16 valida; demais usam o que está em Root.

### DT-32 — S35 (modalidades únicas) refina S26

S26 já valida unicidade Codigo+Encargo+TipoCli. S35 adiciona: **dentro do mesmo periodo** (Diario ou Mensal) — S26 valida cross-periodo, S35 within-periodo.

**Realidade:** S26 já cobre. S35 seria redundante. Vou **remover S35 do escopo** e substituir por S38 (DocTXB unico por CNPJ+dataBase) que é genuinamente novo.

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3050.go              (+H16-H20 +S33-S34 +S36 +I29-I36 = 18 regras Tier 1)
backend/internal/audit/rules/3050_helpers.go      (fix IsUltimoDiaUtilMes edge case)
backend/internal/audit/rules/3050_fase4_test.go   (NOVO — 18 testes Tier 1)
backend/internal/audit/rules/3050.go              (+H21-H25 +S37-S40 +I37-I46 = 18 regras Tier 2 — se tempo permitir)
backend/internal/audit/rules/3050_test.go         (atualizar TestBuiltin3050_TotalRulesIs)
CHANGELOG.md                                       (entry v3.34.6)
backend/SPRINT_33_FASE4_RESULTS.md                (NOVO)
```

## ⚠️ Riscos identificados

1. **Edge case fix pode quebrar tests existentes.** `TestIsUltimoDiaUtilMes` tem casos que dependem do comportamento antigo. Vou validar cada caso:
   - `2024-04-30` (terça) → último dia útil = 30 (sem mudança)
   - `2024-12-31` (terça) → último dia útil = 31 (sem mudança)
   - `2024-04-29` → não-último (sem mudança)
   - `2024-02-29` (bissexto, quarta) → último = 29 (sem mudança)
   - `2023-02-28` (não-bissexto, terça) → último = 28 (sem mudança)
   - **NOVO caso edge:** `2025-05-31` (sábado) → último = 30 (sexta) — único caso que muda

2. **H16-H20 precisam parser change.** Adicionar `Encoding` em Doc3050Root é mínimo mas quebra API. Carry-over se muitos tests falharem.

3. **Volume de código.** 36 regras × ~30 linhas cada = ~1080 linhas Go + ~540 linhas testes = ~1620 linhas. Considerável mas factível.

## 🎯 Self-verify (regra HOT memory)

Antes do commit:
- [ ] `grep -c "Register3050"` no 3050.go confere com Builtin3050 atual.
- [ ] `grep -c "^func Test"` no 3050_fase4_test.go = 18 (Tier 1) ou 36 (Tier 1+2).
- [ ] `go test -race ./internal/audit/rules/...` todos PASS.
- [ ] `go test -race ./...` 23/23 packages PASS.
- [ ] `gofmt -l ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] `go tool cover -func` total ≥ 71%.
- [ ] Para edge case fix: testar 2025-05-31 (sábado) → IsUltimoDiaUtilMes retorna true para 2025-05-30.

## ⏭️ Após Fase 4 (se Sprint 33 fechar em 80%+)

**Opções:**
- **Fase 5** (carry-over restantes): fechar 100% (170 regras) — +54 regras stubs informativos S45-S90 + I47-I80.
- **Sprint 34 — AuditDLO 2061** (próximo CADOC): parser + 30+ regras iniciais.
- **Sprint 34 — FrontendNext**: migração Next.js 15.

**Recomendação:** se cobertura ≥ 70% com regras reais (não stubs), abrir Sprint 34 (AuditDLO) para diversificar workstream. Carry-over 3050 → Sprint 35+.