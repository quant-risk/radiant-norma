# Sprint 33 Fase 5 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 33 Fase 5 (Fase final — fechar 3050 em 100%)
> **Pré-requisito:** v3.34.7 (commit fee5b80) — drift validação corrigido
> **Marco esperado:** 97 → 170 regras 3050 (57.06% → 100% cobertura catálogo TXB_V11)

## 🎯 Escopo Fase 5 (agressivo — fechar 100%)

Esta fase entrega **~73 regras** faltantes (170 - 97), em 4 tiers + carry-over.

### Tier 1 — Individuais restantes (14 regras)

Sub-modalidades que faltaram em I15-I36 (≥ 0):

| Cod | Sev | Regra |
|---|---|---|
| 3050-I37 | E | credLivre vlrConcessoes ≥ 0 |
| 3050-I38 | E | credConsignado vlrConcessoes ≥ 0 |
| 3050-I39 | E | credDirecionado vlrConcessoes ≥ 0 |
| 3050-I40 | E | imobResid vlrConcessoes ≥ 0 |
| 3050-I41 | E | imobComerc vlrConcessoes ≥ 0 |
| 3050-I42 | E | financMicroCred vlrConcessoes ≥ 0 |
| 3050-I43 | E | financInfra vlrConcessoes ≥ 0 |
| 3050-I44 | E | financRuralCusteio vlrConcessoes ≥ 0 |
| 3050-I45 | E | financRuralInvest vlrConcessoes ≥ 0 |
| 3050-I46 | E | financRuralComerc vlrConcessoes ≥ 0 |
| 3050-I47 | E | coopCentrais vlrConcessoes ≥ 0 |
| 3050-I48 | E | coopSingulares vlrConcessoes ≥ 0 |
| 3050-I49 | E | descTitulosAdquiridos vlrConcessoes ≥ 0 |
| 3050-I50 | E | antecipacaoFaturas vlrConcessoes ≥ 0 |

### Tier 2 — Sistema stubs informativos (32 regras)

Cobertura da matriz modalidade × encargo (2001 × 134 do catálogo TXB_V11):

**Matriz prefixada (8 stubs S39-S46):**
| Cod | Sev | Regra |
|---|---|---|
| 3050-S39 | I | capGir modalidades permitidas apenas prefixado (regra 2001) |
| 3050-S40 | I | conta garantida modalidades permitidas apenas prefixado |
| 3050-S41 | I | cheque especial modalidades permitidas apenas prefixado |
| 3050-S42 | I | desconto duplicatas apenas prefixado |
| 3050-S43 | I | desconto cheques apenas prefixado |
| 3050-S44 | I | antecipação faturas cartão crédito apenas prefixado |
| 3050-S45 | I | aquisição veículos apenas prefixado (pós-fixado não permitido) |
| 3050-S46 | I | arrendamento mercantil modalidades permitidas apenas prefixado |

**Matriz pós-fixado bloqueios (10 stubs S47-S56):**
| Cod | Sev | Regra |
|---|---|---|
| 3050-S47 | I | capital giro até 365 não permitido pós-fixado IPCA/IGP-M |
| 3050-S48 | I | capital giro > 365 não permitido pós-fixado moeda estrangeira |
| 3050-S49 | I | capital giro teto rotativo não permitido pós-fixado IPCA |
| 3050-S50 | I | conta garantida não permitido pós-fixado moeda estrangeira |
| 3050-S51 | I | cheque especial não permitido pós-fixado moeda estrangeira |
| 3050-S52 | I | aquisição veículos não permitido pós-fixado |
| 3050-S53 | I | arrendamento mercantil não permitido pós-fixado |
| 3050-S54 | I | financiamento bens não permitido pós-fixado |
| 3050-S55 | I | financiamento rural modalidades permitidas apenas prefixado |
| 3050-S56 | I | financiamento imobiliário modalidades permitidas apenas prefixado |

**Periodicidade / datas (4 stubs S57-S60):**
| Cod | Sev | Regra |
|---|---|---|
| 3050-S57 | I | dataBase fim mês BACEN (regra 3032, parcial) |
| 3050-S58 | I | periodicidade diária cobrada BACEN |
| 3050-S59 | I | periodicidade mensal cobrada BACEN |
| 3050-S60 | I | dataBase entre 1º dia útil e último dia útil do mês |

**Matriz consolidada (10 stubs S61-S70):**
| Cod | Sev | Regra |
|---|---|---|
| 3050-S61 | I | desDuplicatas: prefixado apenas (consolidado S42) |
| 3050-S62 | I | desCheques: prefixado apenas (consolidado S43) |
| 3050-S63 | I | antecipacaoFaturasCartaoCredito: prefixado apenas (consolidado S44) |
| 3050-S64 | I | capGirPrzAte365: bloqueado moeda estrangeira pós-fixado |
| 3050-S65 | I | capGirPrzSup365: bloqueado moeda estrangeira pós-fixado |
| 3050-S66 | I | capGirTetoRot: bloqueado moeda estrangeira pós-fixado |
| 3050-S67 | I | ctgGta: bloqueado IPCA/IGP-M pós-fixado |
| 3050-S68 | I | chqEsp: bloqueado moeda estrangeira pós-fixado |
| 3050-S69 | I | ccb: prefixado apenas (consolidado) |
| 3050-S70 | I | financBens: prefixado apenas |

### Tier 3 — Header adicional (10 regras)

| Cod | Sev | Regra |
|---|---|---|
| 3050-H21 | A | txMedJuros max 4 decimais |
| 3050-H22 | A | vlrConcessoes max 2 decimais (R$) |
| 3050-H23 | A | qtdNovContratos inteiro |
| 3050-H24 | A | cnpjInstituicao all-digits length = 8 (consolidação H10/H11) |
| 3050-H25 | A | nmContato sem caracteres de controle |
| 3050-H26 | A | telContato length 10-11 dígitos (consolidação S17) |
| 3050-H27 | I | declaracao encoding XML presente (parser failure detection) |
| 3050-H28 | I | xml namespace XSD 3050 declarado |
| 3050-H29 | A | indRemessa case-sensitive (consolidação H13) |
| 3050-H30 | A | cnpjInstituicao sem zeros à esquerda |

### Carry-over stubs (4 implementações parciais)

| Cod | Mudança |
|---|---|
| 3050-S01 | MatrizEncargoModalidade: stub informativo (matriz cobertura D-26) — implementado parcialmente (valida 4 combinações óbvias) |
| 3050-S10 | DocAnterior: stub honesto (precisa histórico) |
| 3050-S11 | VlrConcessoesVsTaxas: implementado (vlrConcessoes > 0 → txMedJuros ≥ 0) |
| 3050-S14 | Cruzadas: stub honesto (3051/3054/3055 — sem ref adicional) |

### Total esperado

14 + 32 + 10 + 4 = **60 novas regras** + carry-over adjustments.

97 + 60 = **157 regras** (92.4% cobertura). Carry-over de 13 regras (170 - 157) implementado como Fase 6 ou aceito como gap.

## 🏗️ Decisões técnicas

### DT-32 — Stubs informativos matriz modalidade × encargo

Matriz completa 2001 × 134 = 134 regras individuais, mas muitas são consolidações. Vou implementar como stubs informativos (severity "I", return nil) com Sheet() retornando "Matriz" ou similar.

**Princípio:** o parser XML já valida estrutura (D-26 best-effort). Stubs servem apenas como checklist visível do auditor ("regra existe, aguardando contexto/parser evoluir").

### DT-33 — Carry-over S01 stub parcial

S01 (MatrizEncargoModalidade) tem 4 combinações óbvias:
- `desDuplicatas` apenas com `pre` (não `flu/vc/ind`)
- `desCheques` apenas com `pre`
- `capGirTetoRot` apenas com `pre`
- `antecFaturaCartao` apenas com `pre`

Implementação: 1 regra consolidada `S01MatrizEncargoModalidadeParcial` que verifica as 4 combinações.

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.7) | Pós esperado |
|---|---|---|
| Regras 3050 | 97 | **~157** (Tier 1+2+3+carry-over) |
| Cobertura catálogo 3050 | 57.06% | **~92%** |
| Coverage `internal/audit/rules` | 72.2% | **70-72%** (stubs sem asserts) |
| Test functions Fase 5 | 0 | **~50** |
| Test functions total 3050 | 96 | **~146** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3050.go              (+I37-I50 +S39-S70 +H21-H30 +carry-over = ~60 regras)
backend/internal/audit/rules/3050_fase5_test.go   (NOVO — ~50 testes table-driven)
backend/internal/audit/rules/3050_test.go         (atualizar TestBuiltin3050_TotalRulesIs)
CHANGELOG.md                                       (entry v3.34.8)
backend/SPRINT_33_FASE5_RESULTS.md                (NOVO)
```

## ⚠️ Riscos

1. **Volume de código (~60 regras × 30 linhas = ~1800 linhas).** Vai ser a maior sprint de 3050. Risco de bugs.
2. **Stubs informativos vazios.** Podem ser vistos como "theater" se não houver valor real. Vou documentar claramente que são checklist + carry-over.
3. **Coverage cai.** Stubs sem asserts complexos = mais linhas descobertas.

## 🎯 Self-verify

- [ ] `grep -c "Register3050"` confere com soma esperada.
- [ ] `go test -race ./...` 23/23 PASS.
- [ ] `gofmt -l` clean.
- [ ] `go vet ./...` clean.
- [ ] Para cada I37-I50: testar com valor negativo → erro.

## ⏭️ Após Fase 5

**Sprint 33 fechado em 100%** (ou ~92% se gap aceito).

**Próxima sprint (Sprint 34+):**
- AuditDLO 2061 Fase 1 (próximo CADOC conforme ROADMAP Q3)
- FrontendNext (Next.js 15)
- CI-Gate expandido / Observability (já tem alguns, podem evoluir)