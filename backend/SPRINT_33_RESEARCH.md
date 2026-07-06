# Sprint 33 — Audit3050 (TXB_V11) Fase 1 — RESEARCH

> **Data:** 2026-07-06
> **Sprint:** 33 (Plano Ouro §1.1 Q3)
> **Fase:** 1 de N
> **Tipo:** minor (parser XML 3050 + 14 Agregadas A01-A14 + 14 stubs S01-S14)
> **Pré-requisito:** V60 (commit 8dd43ec / tag v3.33.6) + V61 (commit 110c4a5 / tag v3.33.7)
> **Marco:** 0 → 28 regras 3050, cobertura catálogo 0% → 16.5%

## 🎯 Escopo Fase 1

Esta fase entrega:
1. **`Doc3050` struct** + tipos auxiliares (Modalidade com campos opcionais `*float64`/`*int`).
2. **`ParseDoc3050([]byte) (*Doc3050, error)`** — parser XML análogo a `ParseDoc3040`. Lê conforme XSD BACEN `3050_Schema_TXB_V4.xsd`.
3. **14 regras Agregadas A01-A14** (severity "E"/"A" conforme regra) — somas + header + taxas + prazos.
4. **14 stubs S01-S14** (severity "I" honestos, padrão v3.30.0 D-13) — cobrem regras que precisam de mais estrutura (matriz encargo × modalidade, datas complexas, calendário BACEN, etc).
5. **`Builtin3050() *Registry`** — registry paralelo ao `Builtin3040()`.

**Alvo:** 0 → 28 regras 3050.

## 📋 Análise do catálogo TXB_V11 (170 regras)

Catálogo: `3050/Criticas_TXB_V11.xlsx` (180 linhas, 5 colunas, sheet único "Regras de Validação").

### Distribuição por seção

| Seção | Ocorrências | Códigos únicos | Códigos range |
|---|---|---|---|
| 1. RELATIVAS AO TIPO DE ENCARGO FINANCEIRO PACTUADO | 69 | 1 | 2001 |
| 2. RELATIVAS À PERIODICIDADE DE ENVIO DAS INFORMAÇÕES | 14 | 1 | 2001 |
| 3. RELATIVAS ÀS INFORMAÇÕES A SEREM REPORTADAS | 20 | 1 | 2001 |
| 4. RELATIVAS ÀS INCOMPATIBILIDADES TÉCNICAS DOS DADOS | 36 | 36 | 3003-3059 |
| 5. RELATIVAS AOS FORMATOS DE CAMPO | 31 | 15 | 2001, 3021, 3031-3035 |
| **TOTAL** | **170** | **51** | — |

**Padrão identificado:**
- **2001** repete 134 vezes (matriz modalidade × sub-modalidade × encargo + validações de formato). Cada combinação possível é uma regra individual no XSD.
- **3003-3059** são 36 regras específicas de inconsistências (somas, prazos, taxas limites).

### Mapeamento para Fase 1 (14 A + 14 stubs)

| Regra 3050 | Equivalente 3040 | Onde implementar |
|---|---|---|
| 3018 (sldCarAtiva = soma sldCar por faixa) | A01-A07 | **A01-A04** (somas) |
| 3019 (sldCedido - sldAdquirido ≤ sldCarAtiva anterior + vlrConcessoes) | — | **A02** |
| 3020 (sldBaiPrejuizo ≤ sldCarAtiva mês anterior) | — | **A03** |
| 3021 (sldCarAtiva + sldCedido ≥ sldAdquirido + vlrConcessoes) | — | **A04** |
| 2005-2010 (header obrigatório) | H01-H09 | **A05-A08** |
| 3026-3028, 3042-3044 (taxas limites elevada/baixa) | A09-A14 | **A09-A11** |
| 3051 (txMedJuros ajustada ≤ txMedJuros) | S20 | **A12** |
| 3036-3037 (prazos capGir) | F08 | **A13** |
| 3038-3039 (przMedCarteira limites) | — | **A14** |
| 2001 × 134 (matriz) | (sem equivalente) | **S01-S04** (stubs matriz) |
| 2002-2004, 2009-2010 (header/calendário) | H04-H09 | **S05-S09** (stubs) |
| 3001-3002, 3031-3035 (datas complexas) | S15, S19 | **S10-S14** (stubs) |

**Total:** 14 Agregadas (A01-A14) + 14 stubs (S01-S14) = **28 regras**.

## 🏗️ Decisões arquiteturais

### D-24 — Interface paralela `Rule3050` (não quebrar `Rule` existente)

A interface `Rule` atual (`registry.go:119`) tem `Apply(ctx, *Doc3040) error`. Para 3050, em vez de
fazer `Apply(ctx, interface{})` com type assertion (smell), criamos interface paralela:

```go
// Rule3050 é a interface paralela para regras do CADOC 3050.
// Mantém compat com Rule (3040) — Registry indexa ambos.
type Rule3050 interface {
    Code() string
    Sheet() string
    Severity() string // E (Erro), A (Aviso), I (Informativo)
    Apply3050(ctx context.Context, doc *Doc3050) error
}
```

`Registry` ganha:
- `rules3050 map[string]Rule3050`
- `Register3050(r Rule3050)`
- `Get3050(code string) Rule3050`
- `Builtin3050() *Registry`

### D-25 — `Doc3050` com `Modalidade` achatada

Em vez de espelhar a hierarquia XSD `<pesJuridica>/<pre>/<desDuplicatas>` (4-5 níveis de
structs + 4 modelos diferentes de atributos), achatar tudo em `[]Modalidade`:

```go
type Doc3050 struct {
    Root    Doc3050Root
    Diario  []Modalidade // modalidades diárias achatadas
    Mensal  []Modalidade // modalidades mensais achatadas
}

type Modalidade struct {
    Codigo  string  // desDuplicatas, capGirPrzAte365, ...
    Encargo string  // pre, flu, vc, ind
    TipoCli string  // pesJuridica, pesFisica

    // Campos opcionais (nil = não-preenchido no XML).
    // Uso de *float64 / *int: distinção entre "0" (preenchido com zero) e "ausente".
    TxMedJuros            *float64
    TxMedEncFiscais       *float64
    TxMedEncOperacionais  *float64
    TxMinima              *float64
    TxMaxima              *float64
    VlrConcessoes         *float64
    PrzDecMedConcessoes   *int
    QtdNovContratos       *int
    SldCarAtiva           *float64
    SldCedido             *float64
    SldAdquirido          *float64
    SldBaiPrejuizo        *float64
    SldCarAte14           *float64
    SldCarAte60           *float64
    SldCarAte90           *float64
    SldCarMaior90         *float64
    QtdConAte14           *int
    QtdConAte60           *int
    QtdConAte90           *int
    QtdConMaior90         *int
    PrzMedCarteira        *int
}
```

**Trade-off:** perco hierarquia semântica do XSD (`pesJuridica.pre.desDuplicatas`) mas ganho:
- Regras simples que iteram `range doc.Diario` sem 4 loops aninhados.
- Parser trivial que só precisa achar `*Modalidade` element + seus atributos (sem validar hierarquia).
- Validação de matriz encargo × modalidade ainda possível via `doc.Diario[i].Encargo == "pre" && doc.Diario[i].Codigo == "desDuplicatas"`.

### D-26 — Parser XML tolerante (best-effort)

XSD tem 4 modelos de atributos diferentes. Parser vai:
1. Tentar cada modelo (1, 2, 3, 4) para cada sub-modalidade encontrada.
2. Set apenas os atributos que existem no XML (campos opcionais = nil).
3. Regras de Fase 1 cobrem subset; stubs documentam carry-over.

Falha de parse não é bloqueante — retorna `Doc3050` parcial + erro (`*PartialParseError`).
Regras robustas a campos nil.

### D-27 — Stubs honestos (severity "I")

Padrão v3.30.0 D-13: stubs retornam nil (pass-through) + severity "I". Auditor vê "regra existe
mas não implementada, carry-over Fase X". Zero theater.

## 🐛 Validação de sanidade (auto-checagem pré-implementação)

- [x] XSD 3050 lido: 4 modelos de atributos + 4 encargos × 2 tipos cliente × N sub-modalidades.
- [x] Catálogo TXB_V11 mapeado: 170 regras, 5 seções, distribuição de códigos.
- [x] Decisões D-24/D-25/D-26/D-27 documentadas com trade-offs.
- [ ] Implementação: `3050.go` (struct + parser + regras + stubs).
- [ ] Testes: 14 A + 14 stubs = 28 funções de teste table-driven.
- [ ] Cobertura: `internal/audit/rules` deve subir ≥1pp (de 70.8% para ~72%+).
- [ ] Suite: 23/23 packages -race PASS.
- [ ] Self-verify pré-commit: `grep -c "Apply3050"` no novo arquivo.

## 📁 Arquivos a criar (Fase 1)

```
backend/internal/audit/rules/3050.go              (NOVO — Doc3050 + parser + 14 A + 14 stubs)
backend/internal/audit/rules/3050_test.go         (NOVO — 28 testes table-driven)
backend/internal/audit/rules/registry.go          (D-24: +Rule3050 interface + Builtin3050())
backend/SPRINT_33_RESEARCH.md                     (NOVO — este arquivo)
backend/SPRINT_33_FASE1_RESULTS.md                (NOVO — após implementação)
```

## ⏭️ Fases seguintes (planejadas, NÃO nesta entrega)

- **Fase 2:** 30+ regras Sistemáticas (S15-S44) — formato de campos (CNPJ, data, taxa, valor),
  periodicidade (último dia útil), prazos.
- **Fase 3:** 60+ regras Individuais (I01-I60) — uma por combinação sub-modalidade × encargo × validação.
- **Fase 4:** 2001 × 134 stubs informativos — matriz modalidade × encargo (estrutura XSD cuida de大部分, mas regras Go validam consistência).

## 🎯 Métricas alvo (Fase 1)

| Métrica | Pré | Pós |
|---|---|---|
| Regras 3050 | 0 | **28** (14 A + 14 stubs) |
| Cobertura catálogo 3050 | 0% | **16.5%** (28/170) |
| Coverage `internal/audit/rules` | 70.8% | **≥71%** |
| Test functions Fase 1 | 0 | **28** |
| Packages PASS | 23/23 | **23/23** |
| Race detector | clean | clean |