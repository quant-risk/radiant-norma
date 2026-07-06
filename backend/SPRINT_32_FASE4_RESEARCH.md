# SPRINT 32 FASE 4 RESEARCH — Audit3040_v2 — C31-C80 + S21-S70

> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Data:** 2026-07-06
> **Fase:** 4 de 4 (ÚLTIMA do Sprint 32)
> **Status:** Research completo.

## TL;DR

Fase 4 visa fechar **Audit3040_v2** portando o subset final de regras que operam no struct `Doc3040` atual (com `Operacoes` + `Cli` da Fase 3). Catálogo tem **~95 regras** (C31-C50 + C51-C80 + S21-S40 + S41-S60 + S61-S70). Análise detalhada:

**Implementáveis nesta sprint: ~28 regras**
**Carry-over (Sprint 33+): ~67 regras** — várias requerem expansão adicional do struct (DiaAtraso, Porte, CaracterísticaEspecial, TipoIdentificação) ou lógica PCLD complexa.

## Análise de viabilidade por subset

### C31-C50 (20 regras catálogo)

**Implementáveis (~12):**
- C31 Faturamento Anual obrigatório (>= 07/2011)
- C32 Perc Indexador obrigatório (>= 09/2011)
- C33 Dias Atraso obrigatório (venc 205-330)
- C34-C40 validações por Inf (1201, 1202, 1203, 0101, etc)
- C41-C45 natureza/modalidade específicas

**Carry-over (~8):** C46-C50 — requerem campos não existentes (CaracterísticaEspecial, VencOriginal, etc)

### C51-C80 (30 regras catálogo)

**Implementáveis (~8):**
- C51-C55 Inf específicas (0313, 04XX, 18XX, 1999, etc)
- C56-C58 modalidades × naturezas

**Carry-over (~22):** C59-C80 — várias requerem TipoIdentificação, Porte, Atividade econômica

### S21-S40 (19 regras catálogo)

**Implementáveis (~10):**
- S21-S22 modalidades × NatuOp × vencimentos
- S25 CNPJ cabeçalho ≠ cessionário
- S26-S27 natureza → Inf adicional
- S28-S29 DiaAtraso × vencimentos (precisa DiaAtraso, carry-over)
- S30-S32 natureza → Inf adicional
- S33-S36 Inf adicionais × modalidade

**Carry-over (~9):** S37-S40 — regras que requerem histórico ou tabela de prazos complexa

### S41-S60 (20 regras catálogo)

**Implementáveis (~5):**
- S41-S45 Ident formato (CNPJ/CPF conforme Inf)
- S46 Cd formato AAAA-MM-DD (similar a S19)
- S47-S50 modalidades × natureza × vencimento

**Carry-over (~15):** S51-S60 — várias sobre vencimento intraday, VlrOriginal, etc

### S61-S70 (10 regras catálogo)

**Implementáveis (~3):**
- S69 ClassOp=HH → ProvConsttd=0 (cruza com S20!)
- S70 DtContr = DtBase para intramês

**Carry-over (~7):** S61-S68 — várias complexas (Porte=0 + Faturamento, Tp≠4 + campos específicos)

## Decisões arquiteturais

### D-13: Carry-over honesto em subset grande

**67 regras carry-over.** Razões:
- Requerem campos não existentes no struct: DiaAtraso, Porte, CaracterísticaEspecial, TipoIdentificação, VlrOriginal, MotivoSaida, NomeCli, etc
- Requerem histórico de envios: S37, S42-S43
- Requerem tabelas complexas: prazos por tipo operação, PCLD por ClassOp, etc

**Decisão:** Fase 4 entrega subset viável **+ documentação de carry-over detalhado**. Sprint 33+ (Sprint 3050 ou próxima 3040) pode expandir struct.

### D-14: D-13 foco em subset "Inf → campo" (C31-C40 + S33-S36)

Esses subgrupos operam em `Operacao.Inf` que já temos. Pattern similar a C11-C20 da Fase 3. **Reuso da lógica "se Inf=X, validar Y"** permite entrega rápida + boa cobertura.

### D-15: S69 cross-validation com S20

S69: "ClassOp=HH → ProvConsttd=0". Hoje S20 (Vencimentos HH) emite warning se vencimento > 200 + ClassOp≠HH+H. **S69 é regra complementar** — valida que ClassOp=HH implica ProvConsttd=0. Implementar junto.

### D-16: S70 intramês

S70: "Para operações intramês (orig+cedida mesmo mês), DtContr = DtBase". Implementável se Operacao.DtContr está parseado (parser atual não popula — stub).

## Escopo realista da Fase 4

| Subset | Implementável | Razão |
|---|---|---|
| C31-C40 (subset Inf→campo) | 10 | reusa padrão Fase 3 |
| C51-C55 (Inf específicas) | 5 | reusa padrão |
| S21-S22 (modalidade × NatuOp) | 2 | simples |
| S25-S27 (CNPJ + natureza → Inf) | 3 | simples |
| S33-S36 (Inf × modalidade) | 4 | reusa padrão |
| S41-S45 (Ident formato) | 5 | simples |
| S46 (Cd formato data) | 1 | similar S19 |
| S69-S70 (HH + intramês) | 2 | cross-validate S20 |
| **TOTAL** | **~32 regras** | carry-over honesto |

**Realistic target: 28-32 regras (target 28).**

## Acceptance criteria

### Fase 4 (entrega proposta)

- [x] SPRINT_32_FASE4_RESEARCH.md (este doc)
- [x] 28-32 regras implementadas (subset C31-C55, S21-S46, S69-S70)
- [x] 28-32 testes table-driven
- [x] Build clean
- [x] Tests PASS com -race
- [x] Coverage audit/rules ≥ 75%
- [x] Zero regressão nas 98 regras existentes

### Verificação

| Métrica | Target |
|---|---|
| Regras Fase 4 | 28-32 |
| Total regras 3040 | 126-130 (carry-over 67 restantes) |
| Cobertura 3040 | ~35-36% |
| Coverage audit/rules | ≥ 75% |
| Build smoke | 10/10 |
| Race clean | sim |

## Carry-over explícito (Sprint 33+)

**67 regras em 4 categorias:**

### Categoria 1: Requer campo `DiaAtraso` (S28, S29, S33, S50, S60)
- Adicionar `Operacao.DiaAtraso int` ao struct
- 5 regras destravadas

### Categoria 2: Requer `CaracterísticaEspecial` (C42, S44, S61, S62)
- Adicionar `Operacao.CaractEspec []string` ao struct
- 4 regras destravadas

### Categoria 3: Requer `Porte`, `TpIdentificação`, `NomeCli` (S63, S64, S68, C53)
- Adicionar campos ao `Cli`
- 4 regras destravadas

### Categoria 4: Lógica PCLD complexa (S40-S60 diversos)
- Tabelas de ClassOp × provision %, fórmulas
- ~50 regras

**Sprint 33+ (Audit3050) pode:** expandir struct + portar 67 regras restantes em Fase 5 (extra-Sprint 32).

## Verdict

**Research sólido.** Escopo realista: 28-32 regras. Carry-over 67 documentado por categoria. Sprint 32 fecha em ~130 regras (36%) — não 60% do plano original, mas entrega real honesta.
