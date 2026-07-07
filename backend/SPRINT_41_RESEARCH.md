# Sprint 41 — RESEARCH — AuditDLP 2170 (NSFR — Net Stable Funding Ratio)

> **Data:** 2026-07-07
> **Sprint:** 41
> **Domínio:** DLP 2170
> **Versão atual:** v3.34.21
> **Próxima:** v3.34.22

---

## 1. Contexto

### O que é NSFR?

**NSFR (Net Stable Funding Ratio)** é uma razão de Funding Estável definida na **BACEN Res. 4.542/2017** (complementar à Res. 4.605 do LCR). Garante que instituições mantêm funding estável suficiente para suportar seus ativos de longo prazo por pelo menos 1 ano.

**Fórmula:**
```
NSFR = (Available Stable Funding / Required Stable Funding) × 100%  ≥ 100%
```

- **ASF (Available Stable Funding):** soma ponderada de passivos e capital com pesos de estabilidade (fator 0-100%)
- **RSF (Required Stable Funding):** soma ponderada de ativos com pesos de liquidez (fator 0-100%)

### Fontes de dados

| Fonte | Descrição |
|---|---|
| `2170-DLP/DLP_2170_Instrucoes_Brasil.pdf` | Manual de instruções BACEN (pt-BR) |
| `2170-DLP/DLP_2170_Instrucoes_OrientacoesGerais.pdf` | Orientações gerais de preenchimento |
| `2170-DLP/DLP_2170_Workshop.pdf` | Apresentação de workshop BACEN |
| `2170-DLP/DLP_2170_ModeloCalculo.xlsx` | Planilha de modelo de cálculo |
| `_normativos/CC_3958_DLP.pdf` | Circular Conjunta 3958 (fundamento legal) |

---

## 2. Estrutura do DLP 2170 (NSFR)

### Campos principais do DLP

```xml
<DocDLP cnpj="..." dataBase="...">
  <ASFTotal valor="5000.00"/>         <!-- Available Stable Funding Total -->
  <RSFTotal valor="4000.00"/>         <!-- Required Stable Funding Total -->
  <NSFRRatio valor="125.00"/>         <!-- Ratio ASF/RSF × 100 -->
  <ASFItens>
    <ASFItem codigo="1" descricao="Capital" valor="1000.00" fator="1.00"/>
    <ASFItem codigo="2" descricao="FinanciamentosLongoPrazo" valor="2000.00" fator="0.80"/>
    <ASFItem codigo="3" descricao="DepositosEstaveis" valor="1500.00" fator="0.95"/>
    <!-- ... outros itens ASF -->
  </ASFItens>
  <RSFItens>
    <RSFItem codigo="1" descricao="AtivosLiquidos" valor="500.00" fator="0.00"/>
    <RSFItem codigo="2" descricao="EmprestimosLongoPrazo" valor="2000.00" fator="0.50"/>
    <!-- ... outros itens RSF -->
  </RSFItens>
</DocDLP>
```

### Composição ASF (Available Stable Funding)

| Item ASF | Fator |
|---|---|
| Capital próprio (Patrimônio) | 100% |
| Financiamentos de longo prazo (>12 meses) | 100% |
| Financiamentos de 6-12 meses | 50% |
| Financiamentos <6 meses | 0% |
| Depósitos estáveis (demanda) | 95-100% |
| Depósitos menos estáveis | 90% |
| Recursos de outras instituições (<6 meses) | 0% |

### Composição RSF (Required Stable Funding)

| Item RSF | Fator |
|---|---|
| Caixa e equivalentes | 0% |
| Títulos públicos federais | 5% |
| Empréstimos com гарантия (colateral) | 10-15% |
| Empréstimos de longo prazo (>12 meses) | 50-65% |
| Empréstimos de curto prazo (<1 ano) | 100% |
| Ativos illíquidos | 100% |
| Ações e instrumentos de capital | 100% |

---

## 3. Regras NSFR planejadas

### Fase 1 — 8 regras NSFR (catálogo básico)

| Cod | Sev | Regra | Lógica |
|---|---|---|---|
| **NSFR01** | E | NSFR Ratio ≥ 100% | ASF/RSF × 100 >= 100 |
| **NSFR02** | E | ASF Total >= 0 | Não pode ser negativo |
| **NSFR03** | E | RSF Total >= 0 | Não pode ser negativo |
| **NSFR04** | E | ASF >= RSF (equivalente a NSFR>=100%) | Consistência |
| **NSFR05** | A | NSFR declarado == calculado | Tolerância 1% |
| **NSFR06** | E | Cenário 1 (ASF) >= 0 | Consistência intra-documento |
| **NSFR07** | E | Cenário 2 (RSF) >= 0 | Consistência intra-documento |
| **NSFR08** | A | DtBase formato YYYY-MM-DD | Validação de formato |

### Fase 2 (carry-over, se necessário)

| Cod | Sev | Regra |
|---|---|---|
| NSFR09 | E | Item ASF Individual >= 0 |
| NSFR10 | E | Item RSF Individual >= 0 |
| NSFR11 | A | Soma itens ASF = ASF Total (tolerância) |
| NSFR12 | A | Soma itens RSF = RSF Total (tolerância) |
| NSFR13 | A | DtRef dentro do exercício |

---

## 4. Arquitetura

### Novos arquivos

```
backend/internal/audit/rules/
  dlp.go        (NOVO — DocDLP + ParseDocDLP + helpers)
  2170.go       (NOVO — 8 regras NSFR)
  2170_test.go  (NOVO — testes)
```

### Registry update

```go
// registry.go — adicionar Builtin2170
var Builtin2170 = []interface{}{
    &NSFR01{}, &NSFR02{}, &NSFR03{},
    &NSFR04{}, &NSFR05{}, &NSFR06{},
    &NSFR07{}, &NSFR08{},
}
```

### Parser DLP (dlp.go)

Estrutura `DocDLP` análoga a `DocDRL`/`DocDRM`:
- Root (CNPJ, DataBase)
- ASFTotal, RSFTotal, NSFRRatio
- ASFItens, RSFItens (listas de itens com fator)

Helpers:
- `CalcularNSFRRatio(asf, rsf float64) float64`
- `ValidarNSFRMinimo(doc *DocDLP) error`
- `ValidarDLPBasico(doc *DocDLP) error`

---

## 5. Critérios de aceitação

- [ ] Parser DLP (best-effort XML) implementado
- [ ] 8 regras NSFR com lógica real (0 stubs disfarçados)
- [ ] V70 pre-check aplicado antes do commit
- [ ] `go test -race ./...` 23/23 PASS
- [ ] `go vet ./...` clean
- [ ] `gofmt -l ./...` clean
- [ ] Testes Sprint 41 PASS (≥10 subtests)
- [ ] CHANGELOG entry v3.34.22
- [ ] ROADMAP.md atualizado (Sprint 41 ✅)

---

## 6. Timeline estimado

| Dia | Atividade |
|---|---|
| Dia 1 | RESEARCH.md + parser dlp.go |
| Dia 1-2 | 8 regras NSFR em 2170.go |
| Dia 2 | Testes 2170_test.go |
| Dia 2-3 | Validação V71 (drift check) |
| Dia 3 | RESULTS.md + CHANGELOG + ship |

---

## 7. Riscos

| Risco | Mitigação |
|---|---|
| PDF/DLP não legível por máquina | Parser best-effort como DRM/DLO |
| Falta de XSD oficial | Usar modelo de planilha Excel como grounding |
| Itens ASF/RSF complexos (múltiplos fatores) | Fase 1: validar só totais; Fase 2: granular |
