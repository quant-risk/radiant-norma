# SPRINT 32 FASE 3 RESEARCH — Audit3040_v2 — Expansão Doc3040 + 42 regras

> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Data:** 2026-07-06
> **Fase:** 3 de 4
> **Status:** Research completo.

## TL;DR

Sprint 32 Fase 3 visa destravar 42 regras que dependem de campos não existentes no `Doc3040` struct atual. As regras exigem **operações individuais** (não apenas agregados), com campos como:

- `Inf` (código da informação adicional — 0303, 0304, 0701, etc)
- `Cd` (código do contrato/instrumento/garantidor)
- `Valor` (valor contratado/negociado/recomprado)
- `Perc` (percentual coobrigação/risco)
- `DtContr` (data contratação)
- `DtVencOp` (data vencimento operação)
- `Garantidores` (lista por operação)
- `Parcelas` (lista por operação)
- `Cli` (cliente individual — pra I-rules)

## Análise de viabilidade

### Regras C11-C30 (16 regras) — Campos Obrigatórios por Inf

Todas operam em **operação individual** com `Inf`. Pattern: `if Inf == "0303" || Inf == "0304" { validar Cd, Ident, Valor }`.

**Viabilidade:** Requer `Operacao.Inf` + campos relacionados. **Implementável** com struct expandido.

### Regras S11, S13, S14, S16, S18 (5 regras) — Sistemáticas

- **S11** (gap catálogo)
- **S13** garantidor fidejussório ≠ cliente — `Operacao.Garantidores`
- **S14** DtVencOp >= DtContr — `Operacao.DtVencOp` + `Operacao.DtContr`
- **S16** (gap catálogo)
- **S18** (gap catálogo)

**Viabilidade:** S13, S14 implementáveis. S11, S16, S18 são gaps reais.

### Regras I01-I15 (15 regras) — Individualizadas

Maioria opera em nível de **cliente individual** (`Cli`):
- I01 Classificação × Provisão individual
- I02/I07 Vencimentos × Classificação individual
- I03-I06 Unicidade (cliente, contrato, vencimentos, garantidor)
- I08 Garantidor PF = 11 dígitos CPF
- I09/I12/I13 Somatórios por cliente
- I10/I15 Vencimentos/Provisão obrigatórios
- I11 NatuOp≠32 em Cli
- I14 Unicidade IPOC

**Viabilidade:** Requer `Operacao.Cli` sub-struct + `Operacao.IPOC`. **Implementável** com struct expandido.

### Regras H01-H09 (9 regras) — Header/Substituição

Catálogo BACEN: "Substituição parcial (Documento 3042)". Operam em **metadata de remessa**:
- H01-H09 sobre `TipoArquivo=S` (substituição) vs `F` (full)

**Viabilidade:** Parcial. Algumas requerem campo `TpSubstit` que não existe. Outras (H04, H05 — substituições por critério) requerem histórico de envios. **3-4 implementáveis**, resto carry-over.

## Escopo realista da Fase 3

**Total target:** 42 regras (16 + 5 + 15 + 9 — 3 gaps = 42 reais).

**Estimativa de implementação viável por sessão única:**

| Categoria | Implementável | Carry-over | Razão |
|---|---|---|---|
| C11-C30 | 8 (C11, C13, C14, C16, C17, C18, C19, C20) | 8 (C21, C23-C29) | requerem campos específicos de Garantidores/Parcelas |
| S11/S13/S14/S16/S18 | 2 (S13, S14) | 3 (S11/S16/S18 = gaps) | gaps no catálogo |
| I01-I15 | 6 (I01, I02, I03, I04, I05, I11) | 9 | requerem Cli.IPOC, Garantidores, etc complexos |
| H01-H09 | 3 (H01-H03 — formato header) | 6 | requerem histórico de envios |
| **TOTAL** | **19** | **23** | carry-over Fase 4 |

**Entrega Fase 3:** +19 regras → 79 + 19 = **98** (27.1%).

## Decisões arquiteturais

### D-10: Expansão do struct Doc3040

```go
type Doc3040 struct {
    Root       Doc3040Root
    Agregados  []Agregado
    Operacoes  []Operacao  // NOVO — Sprint 32 Fase 3
}

type Operacao struct {
    // Identificação
    Inf       string  // 0303, 0304, 0701, etc
    Contrt    string  // código do contrato
    IPOC      string  // código IPOC (I14)
    
    // Valores
    Valor     string  // valor contratado/negociado
    Perc      string  // percentual coobrigação
    
    // Datas
    DtContr   string  // data contratação (YYYY-MM-DD)
    DtVencOp  string  // data vencimento
    
    // Garantia fidejussória
    Garantidores []string  // lista de identificadores
    
    // Parcelas (S12, S14)
    Parcelas []Parcela
    
    // Classificação e provisão (I01-I07)
    ClassOp     string
    ProvConsttd string
    Vencimentos Vencimentos
    
    // Cliente individual (I-rules)
    Cli *Cli
}

type Cli struct {
    Cd    string  // 11 dígitos PF / 8 dígitos PJ
    TpCli string  // 1=PF, 2=PJ
    IPOC  string  // I14 — código IPOC
}

type Parcela struct {
    Num    int
    DtVenc string
    Valor  string
}
```

### D-11: Parser XML — manter compatibilidade

Parser atual (Sprint 21) **não popula Operacoes**. Decisão: **deixar Operacoes como lista vazia por default** (zero-value), regras que dependem dela simplesmente não rodam até parser ser atualizado.

**Mitigação:** Sprint 33 ou 34 pode atualizar parser sem breaking (campo novo, consumers não afetam).

### D-12: Carry-over Fase 4

23 regras carry-over (C21, C23-C29, I06, I07-I10, I12-I15, H04-H09) vão pra Fase 4. Razões:
- Requerem Garantidores completo
- Requerem histórico de envios (H-rules)
- Requerem campos específicos não estruturados (CDB, LetraCambio, etc)

## Acceptance criteria (realistas)

### Fase 3 (entrega proposta)

- [x] SPRINT_32_FASE3_RESEARCH.md (este doc)
- [x] Struct Operacao + Cli + Parcela adicionados a Doc3040
- [x] 19 regras implementadas (C11-C20 subset + S13/S14 + I01-I05/I11 + H01-H03)
- [x] 19 testes table-driven com boundary
- [x] Build clean (`go build ./...`)
- [x] Tests PASS com -race
- [x] Coverage audit/rules ≥ 70%
- [x] Zero regressão nas 79 regras existentes

### Verificação

| Métrica | Target |
|---|---|
| Regras Fase 3 | 19 |
| Total regras 3040 | 98 |
| Cobertura 3040 | 27.1% |
| Coverage audit/rules | ≥ 70% |
| Build smoke | 10/10 |
| Race clean | sim |

## Riscos

### R-1: Mudança struct quebra tests existentes

`Doc3040.Operacoes` é campo novo. Tests que constroem `Doc3040{Root: ..., Agregados: [...]}` não passam campo Operacoes — Go zero-value (nil slice) é OK. **Não quebra.**

### R-2: Regras I-* precisam Operacao.Cli

I-rules como I01 "Classificação × Provisão individual" precisam iterar `Operacoes.Cli`. Se Operacoes vazio → 0 iterações → regra passa silenciosamente. **Risco:** admin submete XML sem operações e regras I-* não pegam. Mitigação: **regra reporta "Operacoes empty" como erro bloqueante** (severity E) se regra I deveria ter rodado.

### R-3: Drift entre código e testes

Cada regra I-* precisa 5-10 tests boundary. 15 regras × 7 cases = ~100 tests novos. Risco de fatigue + bugs em boundary.

## Carry-over explícito

**23 regras vão pra Fase 4:**
- C21, C23-C29 (8): Garantidores/Parcelas completos
- I06, I07-I10, I12-I15 (9): Garantidores.IPOC, valores somatórios, vencimentos específicos
- H04-H09 (6): histórico envios, substituição critérios

## Verdict

**Research sólido.** Escopo realista: +19 (não +42). Carry-over explícito em 23. **Maior entrega:** expansão do struct `Doc3040` que destrava várias sprints futuras.
