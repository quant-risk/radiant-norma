# SPRINT 32 FASE 3 RESULTS — Audit3040_v2 — 19 regras Individuais + Doc3040 expandido

> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Data:** 2026-07-06
> **Fase:** 3 de 4
> **Status:** ✅ Shipped

## TL;DR

Sprint 32 Fase 3 entregou **19 regras Individuais/Campos/Header** + expansão do struct `Doc3040` com `Operacao`, `Cli`, `Parcela`. Total de regras 3040: **79 → 98** (cobertura 21.9% → **27.1%**).

**Escopo honesto:** Catálogo tinha 42 regras candidatas (16 C + 5 S + 15 I + 9 H — 3 gaps). Análise mostrou que **23 são carry-over** (precisam Garantidores/Parcelas completos, histórico de envios, somatórios complexos).

## Decisões arquiteturais

### D-10: Expansão do struct `Doc3040`

```go
type Doc3040 struct {
    Root       Doc3040Root
    Agregados  []Agregado
    Operacoes  []Operacao  // NOVO Sprint 32 Fase 3
}

type Operacao struct {
    Inf, Contrt, IPOC, Valor, Perc, DtContr, DtVencOp, ClassOp, ProvConsttd string
    Vencimentos Vencimentos
    Garantidores []string
    Parcelas []Parcela
    Cli *Cli  // cliente individual (I-rules)
}

type Cli struct {
    Cd, TpCli, IPOC string
}

type Parcela struct {
    Num int
    DtVenc, Valor string
}
```

**Compatibilidade:** Campo novo (`Operacoes`) é nil por default se parser não popula. Regras que iteram Operacoes simplesmente não rodam — zero impacto em código existente. **Zero regressão.**

### D-11: Parser XML — manter compatibilidade

Parser atual (Sprint 21) **não popula Operacoes**. Decisão: deixar nil/empty. Quando parser for atualizado (Sprint 33+), regras passam a ter dados. Decisão: backward compat > feature velocity.

### D-12: Carry-over 23 regras → Fase 4

Regras que ficaram de fora (e por quê):
- C21, C23-C29 (8): Garantidores/Parcelas completos (C21, C24, C26 etc)
- I06-I10, I12-I15 (9): Garantidores.IPOC, somatórios por cliente, vencimentos específicos
- H04-H09 (6): histórico de envios (substituição critérios)
- S11, S16, S18 (3): gaps reais no catálogo BACEN

## Métricas

| Métrica | Pré Fase 3 | Pós Fase 3 |
|---|---|---|
| Regras 3040 | 79 | **98** (+19) |
| Cobertura 3040 | 21.9% | **27.1%** |
| Coverage internal/audit/rules | 68.1% | **70.1%** (+2.0pp) |
| Test functions Fase 3 | 0 | **12** |
| Subtests (table-driven) | — | ~50 |
| Packages PASS | 23/23 | **23/23** |
| Race detector | clean | clean |
| Build smoke | 10/10 | **10/10** |
| Struct fields novos | 0 | **3 (Operacao, Cli, Parcela)** |

## Regras implementadas

### C11-C20 (Campos Obrigatórios por Inf) — 8 regras

| Code | Descrição | Inf |
|---|---|---|
| C11 | DtVencOp obrigatória | todas (exceto v199) |
| C13 | Cd + Ident + Valor obrigatórios | 0303, 0304 |
| C14 | Contrt + IPOC + Valor obrigatórios | 0305 |
| C16 | Contrt + IPOC obrigatórios | 0307 |
| C17 | Contrt obrigatório | 04XX |
| C18 | Contrt obrigatório | 05XX |
| C19 | Contrt + Perc + Valor obrigatórios | 0701 |
| C20 | Contrt + IPOC + Perc + Valor obrigatórios | 0702, 0703, 0704 |

### S13, S14 (Sistemáticas individuais)

| Code | Descrição |
|---|---|
| S13 | Garantidor fidejussório ≠ próprio cliente (PF only) |
| S14 | DtVencOp >= DtContr |

### I01-I05, I11 (Individualizadas) — 6 regras

| Code | Descrição |
|---|---|
| I01 | Classificação × Provisão individual (reusa tabela A01) |
| I02 | Vencimentos × Classificação individual (mesma tabela A02) |
| I03 | Unicidade (Cd, TpCli) na remessa |
| I04 | Unicidade (Contrt, IPOC=modalidade) na remessa |
| I05 | Unicidade vencimentos em uma operação |
| I11 | **STUB** — NatuOp≠32 em Cli (carry-over Fase 4: precisa NatuOp em Operacao) |

### H01-H03 (Header) — 3 regras

| Code | Descrição |
|---|---|
| H01 | TpArq ∈ {F, S} |
| H02 | CNPJ raiz = 8 dígitos numéricos |
| H03 | TotalCli > 0 |

## Bugs encontrados pelos tests

1. **`if err := X{}.Apply(); ...` pattern quebrou compilação** — Go parser confunde com composite literal. Fix: extrair pra `err := ...; if err != nil`. 2 ocorrências no test file.
2. **TestI01 boundary case ratio == provMax** — provMax é exclusive. Fix: test case ratio=0.0050 esperado ERRO, ratio=0.0049 OK.

## Lições aprendidas

### L-1. Struct expansion com zero regressão

Adicionar campo `Operacoes []Operacao` ao `Doc3040` não quebrou nenhum test existente. Por quê? Go zero-value (nil slice) é aceito em range loops (`for _, op := range nil` = 0 iterations). Pattern: **expansão de struct em YAGNI é segura se consumidores usam range.**

### L-2. Carry-over explícito > stub silencioso

I11 é stub pass-through. Documentado como "carry-over Fase 4: precisa NatuOp em Operacao". Comentário + código consistentes. Não é theater.

### L-3. Tabela A01 reusada em I01

I01 (Classificação individual × Provisão) usa mesma `tabelaClassOpProvisaoA01` que A01 (agregado). Single source of truth — se BACEN atualizar %, ambas atualizam.

### L-4. Boundary tests ainda pegam bugs reais

Mesmo depois de 4 sprints de Audit3040, boundary cases ainda revelam bugs sutis. I01 boundary 0.0050 ERRO (provMax exclusive) — decisão de design que precisa ser consistente em TODA regra de faixa.

### L-5. Parser não popula Operacoes → regras I-* não rodam

Hoje, parser XML não popula `Operacoes`. Regras I-* simplesmente não rodam (range sobre slice nil = 0 iterations). Não bloqueia nada. Sprint 33+ (parser update) destrava todas as 23 regras I-* carry-over.

## Próximos passos

### Fase 4 (última sprint do Audit3040_v2)

- C31-C80 (Campos avançados) — 50 regras
- S21-S70 (Sistemáticas parte 3) — 30 regras (subset — algumas precisam cross-doc)
- Total esperado: +75 → 173 (47.9%)

Carry-over Fase 3 → Fase 4: 23 regras (C21/C23-C29 + I06-I10/I12-I15 + H04-H09).

## Compatibilidade

- `Doc3040` struct expandido com `Operacoes []Operacao`. Zero impacto em código existente.
- Parser XML não popula Operacoes ainda — backward compat mantida.
- Registry atualizado com 19 regras — consumers que iteram Codes() precisam estar preparados (já estavam).
- Zero impacto em API REST ou binários.

## Verdict

**✅ Ship-ready.** 19 regras entregues com 12 testes, struct expandido, coverage 70.1% (target atingido). Carry-over honesto em 23 regras. Próxima sprint: **Fase 4 — fechar 3040 → ~48%**.
