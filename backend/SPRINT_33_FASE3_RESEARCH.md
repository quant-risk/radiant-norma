# Sprint 33 Fase 3 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 33 Fase 3 (continuação direta Fase 2)
> **Pré-requisito:** v3.34.2 (commit 1980f35 / tag v3.34.2) — drift validação corrigido
> **Marco esperado:** 56 → 90+ regras 3050 (cobertura 32.9% → 53%+)

## 🎯 Escopo Fase 3

Esta fase entrega:
1. **H10-H15 Header** (6 regras) — length max adicional em campos do header (cnpj length, indRemessa variants, encoding UTF-8, espaços duplicados).
2. **I15-I28 Individuais** (14 regras) — sub-modalidades específicas (desDuplicatas, desCheques, vendor, compror, carCrd, etc) × validação.
3. **S29-S32 Sistema** (4 regras) — periodicidade básica (dataBase deve ser fim de mês BACEN, deve estar em janela de envio, etc).
4. **Carry-over** (3 implementações): S09 (DiasUteis), S13 (ÚltimoDiaUtil), S24 (txMedJurosAjustada ≤ txMedJuros) — deixar de ser stub.

**Alvo:** 56 → 81+ regras 3050 (cobertura 47.6%+).

## 📋 Mapeamento regra → categoria

### H10-H15 Header (6 regras)

Baseado nas seções "3. Informações a serem reportadas" e "5. Formato de campo" do catálogo TXB_V11:

| Cod | Sev | Regra | Origem BACEN |
|---|---|---|---|
| 3050-H10 | A | cnpjInstituicao length = 8 (raiz CNPJ BACEN) | 2005 |
| 3050-H11 | A | cnpjInstituicao all-digits (sem letras/símbolos) | 2005 |
| 3050-H12 | E | dataBase formato YYYY-MM-DD rigoroso (não 2024-1-1) | formato |
| 3050-H13 | A | indRemessa ∈ {I, A, S} case-sensitive | 3052 |
| 3050-H14 | A | nmContato trim sem espaços duplicados | formato |
| 3050-H15 | A | telContato trim sem caracteres não-numéricos residuais | formato |

### I15-I28 Individuais (14 regras)

Uma por sub-modalidade × validação. Catálogo TXB_V11 §4 lista 36 regras 3003-3059; já implementei I01-I14 (limites). Agora I15-I28 cobre regras de inconsistência por modalidade:

| Cod | Sev | Regra | Modalidade-alvo |
|---|---|---|---|
| 3050-I15 | E | sldCarAtiva ≥ 0 em desDuplicatas | desDuplicatas |
| 3050-I16 | E | vlrConcessoes ≥ 0 em desCheques | desCheques |
| 3050-I17 | E | txMedJuros ≥ 0 em vendor | vendor |
| 3050-I18 | E | przDecMedConcessoes ≥ 0 em compror | compror |
| 3050-I19 | E | sldCarAtiva ≥ 0 em carCrd (cartão crédito) | carCrd |
| 3050-I20 | E | vlrConcessoes ≥ 0 em carCrd | carCrd |
| 3050-I21 | A | txMedJuros ≤ 100% em todas modalidades | (cruzada) |
| 3050-I22 | A | txMedEncOperacionais ≤ 50% em todas modalidades | (cruzada) |
| 3050-I23 | E | przDecMedConcessoes ≤ 5000 dias em capGir | capGir* |
| 3050-I24 | E | qtdNovContratos ≥ 0 em todas modalidades | (cruzada) |
| 3050-I25 | E | sldCedido ≥ 0 em todas modalidades | (cruzada) |
| 3050-I26 | E | sldAdquirido ≥ 0 em todas modalidades | (cruzada) |
| 3050-I27 | A | sldCarAtiva > 0 → modalidades com txMaxima > txMinima | (cruzada) |
| 3050-I28 | A | indRemessa = "I" → qtdNovContratos ≥ 1 | (cruzada) |

### S29-S32 Sistema (4 regras)

Periodicidade + calendário simplificado (sem dependência de calendário BACEN externo):

| Cod | Sev | Regra | Origem |
|---|---|---|---|
| 3050-S29 | E | dataBase deve estar entre 2009-01-01 e hoje+30 (não-futuro distante) | 2009/2010 |
| 3050-S30 | A | dataBase deve ser primeiro dia subsequente ao último dataBase recebido | 2001/periodicidade |
| 3050-S31 | I | doc.AnteriorRef presente quando indRemessa = "S" (substituição) | 2001 |
| 3050-S32 | A | qtdTotal modalidades > 0 (sanity: doc não-vazio) | formato |

### Carry-over (3 implementações)

| Cod | Sev | Mudança | Origem |
|---|---|---|---|
| 3050-S09 | E | DiasUteis: dataBase é dia útil BACEN (placeholder + lista hardcoded feriados nacionais) | 2009/calendário |
| 3050-S13 | E | ÚltimoDiaUtil: dataBase é último dia útil do mês BACEN | 3031-3035 |
| 3050-S24 | E | txMedJurosAjustada ≤ txMedJuros (precisa parser expor `txMedJurosAjustada`) | 3051 |

**Nota:** S09 e S13 dependem de um helper `IsDiaUtilBACEN(date time.Time) bool` que vou criar com feriados nacionais hardcoded (placeholder consciente, evolução futura = API BACEN ou tabela). S24 depende do parser expor o campo — vou verificar se precisa.

## 🏗️ Decisões técnicas

### DT-28: Helper `IsDiaUtilBACEN`

```go
// isDiaUtilBACEN retorna true se data for dia útil no calendário BACEN
// (feriados nacionais hardcoded — placeholder consciente, evolução futura
// = API BACEN ou tabela anual atualizável).
//
// Feriados nacionais fixos (lei federal):
//   01/01 — Confraternização Universal
//   21/04 — Tiradentes
//   01/05 — Dia do Trabalho
//   07/09 — Independência
//   12/10 — Nossa Senhora Aparecida
//   02/11 — Finados
//   15/11 — Proclamação República
//   25/12 — Natal
//
// Feriados móveis (Páscoa, Carnaval, Sexta-Feira Santa, Corpus Christi)
// calculados por algoritmo de Gauss (Computus).
```

**Stub honesto:** usar lista fixa de feriados nacionais (lei federal) + algoritmo de Gauss pra feriados móveis. Suficiente pra Fase 3; evolução futura = API BACEN.

### DT-29: txMedJurosAjustada no parser

Verificar se parser 3050 já expõe este campo. Se não, adicionar como `*float64` opcional em `Modalidade` (D-26 permite nil-safe). Carry-over pra S24.

### DT-30: I21-I22 (txMedJuros ≤ 100% / txMedEncOperacionais ≤ 50%)

Essas são "cruzadas" no sentido que aplicam a **todas** as modalidades. Implementação: loop sobre `doc.Diario`+`doc.Mensal`, validar cada uma. Semelhante a I11-I14 (limites BACEN).

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.2) | Pós esperado |
|---|---|---|
| Regras 3050 | 56 | **81+** (+25) |
| Cobertura catálogo 3050 | 32.9% | **47.6%+** |
| Coverage `internal/audit/rules` | 72.1% | **70-72%** (stubs implementados caem cobertura) |
| Test functions Fase 3 | 0 | **22-25** |
| Test functions total 3050 | 46 | **68-71** |
| Packages PASS -race | 23/23 | **23/23** |
| vet + gofmt | clean | **clean** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3050.go              (+H10-H15 +I15-I28 +S29-S32 +S09/S13/S24 carry-over)
backend/internal/audit/rules/3050_fase3_test.go   (NOVO — ~22-25 testes table-driven)
backend/internal/audit/rules/3050_helpers.go      (NOVO — isDiaUtilBACEN + helper feriados)
backend/internal/audit/rules/3050_test.go         (atualizar TestBuiltin3050_TotalRulesIs)
backend/SPRINT_33_FASE3_RESULTS.md                (NOVO — após implementação)
CHANGELOG.md                                       (entry v3.34.3)
```

## ⚠️ Riscos identificados

1. **S09/S13 feriados:** lista hardcoded de feriados nacionais é correto mas incompleta (feriados estaduais/municipais existem). Documentar como placeholder consciente. Mitigação: comentário claro + evolução futura documentada.

2. **S24 txMedJurosAjustada:** se parser não expõe o campo, S24 vira S09-equivalente (stub honesto). Verificar parser primeiro.

3. **I27/I28 (cruzadas complexas):** testar com casos onde a regra **não** dispara (sanity). Caso contrário, regra dispara sempre e coverage "verde" mas qualidade ruim.

4. **Coverage caindo:** espera-se coverage -1pp a -2pp porque stubs viram código real (mais linhas). Aceitável.

## 🎯 Self-verify (regra HOT memory)

Antes do commit:
- [ ] `grep -c "^func Test" backend/internal/audit/rules/3050_fase3_test.go` confere com soma de regras.
- [ ] `grep Register3050` no Builtin3050 confere com soma esperada (81+).
- [ ] `go test -race -count=1 -v ./internal/audit/rules/` — todos PASS.
- [ ] `go vet ./...` clean.
- [ ] `gofmt -l ./internal/audit/rules/` clean.
- [ ] Para cada carry-over (S09, S13, S24): testar dia útil REAL (ex: 2024-12-25 = Natal = não útil) e último dia útil REAL (ex: 2024-04-30 = último dia útil do mês abril 2024).

## ⏭️ Próxima sprint (Fase 4 — opcional)

Se Fase 3 entregar 81+ regras, Fase 4 pode:
- **H16-H25** Header avançado (encoding UTF-8 BOM, namespaces XML)
- **S33-S44** Sistema adicional (matriz 2001 × 134 regras-stub com XSD enforcement — stubs severity "I")
- **I29-I60** Individuais adicionais (sub-modalidades específicas que faltaram em I15-I28)
- **Alvo:** 81 → 170 regras (100% cobertura, mesmo que via stubs informativos).