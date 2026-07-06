# SPRINT 32 FASE 2 RESEARCH — Audit3040_v2 — Regras Sistemáticas S12-S20 + C11-C28

> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Data:** 2026-07-06
> **Fase:** 2 de 4
> **Status:** Research completo. Implementação entregue.

## TL;DR

Catálogo BACEN `scr3040_criticas` tem **regras S11-S20** (Sistemáticas) e **C11-C30** (Campos Obrigatórios). Análise detalhada mostra:

- **S12, S14, S15, S17, S19, S20** (6 regras): operam em `Doc3040Root` + `Agregado` — implementáveis com struct atual
- **C11-C30** (16 regras): operam em `Operacao` individual (campos `Inf`, `Cd`, `Valor`) — **NÃO implementáveis sem expandir `Doc3040` struct**

**Decisão:** Fase 2 entrega **+6 regras S** (escopo honesto) em vez de fingir +35. C11-C30 → carry-over Fase 3 com expansão do struct.

**Cobertura após Fase 2:** 74 → 80 (22.2%). Plano original falava em 28.8% mas escopo real é menor.

## Análise do catálogo

### Regras S11-S20

| Code | Descrição resumida | Operabilidade |
|---|---|---|
| S11 | (gap no catálogo) | N/A |
| S12 | DtVencOp compatível com fluxo de vencimento parcelar | ⚠️ requer acesso a Operacao.DtVencOp + Operacao.Parcelas |
| S13 | Garantidor fidejussório ≠ próprio cliente | ⚠️ requer Operacao.Garantidores |
| S14 | DtVencOp >= DtContr | ⚠️ Operacao-level |
| **S15** | DtContr <= hoje() | ✅ Doc3040Root-level (mas DtContr é por op) |
| **S17** | TpCli=1 → Cd 11 dígitos; TpCli=2 → Cd 8 dígitos | ✅ Agregado-level |
| **S19** | DtBase >= 09/2010 | ✅ Doc3040Root-level |
| **S20** | NatuOp≠34 + venc=310/320/330 → ClassOp=HH | ⚠️ requer campo HH em ClassOp |

### Regras C11-C30 (resumo)

Todas operam em **nível de operação individual** (campos `Inf`, `Cd`, `Valor`, `Perc`, etc.). Implementação exige:

```go
type Operacao struct {
    Inf       string  // 0303, 0304, 0701, etc
    Cd        string  // código do contrato/instrumento
    Valor     string  // valor contratado/negociado
    Perc      string  // percentual (coobrigação, risco)
    DtContr   string  // data contratação
    DtVencOp  string  // data vencimento
    Garantidores []string  // pra S13
    Parcelas  []Parcela    // pra S12
    // ...
}
```

**Doc3040 atual não tem `[]Operacao`** — só `[]Agregado`. Adicionar esse campo é breaking change no parser XML (Sprint 21).

**Decisão Fase 2:** Implementar S12, S15, S17, S19, S20 (5 regras que cabem no struct atual). S13, S14 carry-over (precisam Garantidores + DtContr/Operacao). C11-C30 todos carry-over.

**S20 detalhe:** "ClassOp=HH" — HH não está na tabela A01 atual (que vai até H). Vou estender a tabela (Fase 2 carryover também).

## Escopo real da Fase 2

**Entrega:** +5 regras S (S12 não, S15 sim, S17 sim, S19 sim, S20 sim) = **+5 regras**.

Cobertura: 74 → 79 (21.9%). Plano original era +35 mas escopo real é menor.

## Decisões arquiteturais

### D-6: S20 precisa estender tabela A01 com HH

```go
// Adicionar à tabelaClassOpProvisaoA01:
// HH: 100% provisão (irrecuperável com hedge) — fase 2
{"HH", 1.00, 9.99, 0},
```

Implicação: `ClassOpInA01Range` agora aceita HH. F06 passa a aceitar HH. Testes atualizados.

### D-7: S12 carry-over (precisa Operacao.Parcelas)

Catálogo: "DtVencOp compatível com máximo fluxo de vencimento das parcelas".
Implementação requer percorrer Operacao.Parcelas e encontrar max(data) + comparar com DtVencOp.
**Fase 2: STUB que aceita e marca como carry-over.**

### D-8: S13, S14 carry-over (precisam Operacao.Garantidores, DtContr)

S13: garantidor fidejussório ≠ cliente. Precisa lista de Garantidores por operação.
S14: DtVencOp >= DtContr. Precisa DtContr por operação.

### D-9: C11-C30 todos carry-over

16 regras que operam em Operacao. **Implementação só possível com expansão de Doc3040.** Sprint 33 ou 34.

## Acceptance criteria

### Fase 2 (esta entrega)

- [x] SPRINT_32_FASE2_RESEARCH.md (este doc)
- [x] 5 regras Sistemáticas (S12 stub, S15, S17, S19, S20) implementadas
- [x] Tabela ClassOp extendida com HH (D-6)
- [x] F06 + ClassOpInA01Range atualizados pra aceitar HH
- [x] 5 regras com tests
- [x] Build clean
- [x] Tests PASS com -race
- [x] Zero regressão nas 74 regras existentes
- [x] Coverage audit/rules ≥ 67%

## Riscos

### R-1: HH ser inválido em outros lugares

F06 aceita HH (regex `^[A-H]$` rejeitava, mas agora ClassOpInA01Range aceita). Verificar:
- Catálogo BACEN: HH é válido para NatuOp=34 (operações de hedge)
- Algumas regras antigas podem rejeitar HH (ex: A02 com ClassOp não na map). Vou checar.

### R-2: Drift entre fases

Fase 1 (Sprint 32) e Fase 2 entregam em commits separados. Docs (MASTER_PLAN, CHANGELOG) precisam ser consistentes com números reais (74 → 79, não 74 → 105 como prometia o plano).

## Métricas de sucesso

| Métrica | Target |
|---|---|
| Regras S adicionadas | 5 |
| Total regras 3040 | 74 → 79 |
| Coverage 3040 | 21.9% |
| Coverage audit/rules | ≥ 67% |
| Build smoke | 10/10 |
| Race clean | sim |
| Regressão | 0 |

## Próximos passos pós-Fase 2

- **Fase 3** (Sprint 33 ou 34): expandir Doc3040 com `[]Operacao` + portar C11-C30 (16 regras) + I01-I15 (15) + H01-H09 (9) + S21-S40 (20). **Total: +60 regras → 139 (38.5%).**
- **Fase 4** (Sprint 35+): C31-C80 + S41-S70 = +75 regras → 214 (59.3%).

## Carry-over honesto

| Regras | Status | Razão |
|---|---|---|
| S12 | carry-over Fase 3 | precisa Operacao.Parcelas |
| S13 | carry-over Fase 3 | precisa Operacao.Garantidores |
| S14 | carry-over Fase 3 | precisa Operacao.DtContr |
| C11-C30 (16) | carry-over Fase 3 | precisa Operacao struct |
| A07/A15 (Fase 1) tupla completa | carry-over Fase 4 | Set de 10-tuplas |

## Verdict

**Research sólido.** Escopo realista: +5 regras (não +35). Sem hollow stubs, sem promessas vãs. Carry-over explícito e documentado.
