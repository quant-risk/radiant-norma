# SPRINT 32 RESEARCH — Audit3040_v2 — Portar regras restantes 3040

> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Data:** 2026-07-06
> **Trigger:** Plano Ouro §3.4 — fechar gap de cobertura 3040 de 16.6% → 60%
> **Status:** Research completo. Implementação Fase 1 entregue.

## TL;DR

Catálogo 3040 tem **361 críticas** (BACEN scr3040_criticas). Hoje **60 estão portadas em Go** (16.6%) — B01-B25 (parcial), F01-F15, C01-C10, S01-S10. Sprint 32 visa **+160 regras** para chegar a ~60% (220/361).

**Escopo da sprint em fases (entrega incremental):**

| Fase | Regras | Acumulado | Cobertura |
|---|---|---|---|
| Pré Sprint 32 | 60 | 60 | 16.6% |
| **Fase 1 (esta entrega)** | **15 (A01-A15)** | **75** | **20.8%** |
| Fase 2 (próxima sprint) | +35 (B16-B25 + C11-C30 + S11-S20) | 110 | 30.5% |
| Fase 3 | +50 (A16-A30 + B26-B35 + I01-I15) | 160 | 44.3% |
| Fase 4 | +60 (C31-C80 + S21-S70) | 220 | **60.9%** ✓ target |

**Rationale pra faseamento:** portar 160 regras numa única sprint = ~2-3 horas leitura de catálogo + 4-6 horas implementação + 2-3 horas tests + 1-2 horas validação. Não cabe em sessão única sem comprometer qualidade. Melhor entregar Fase 1 sólida e continuar incrementalmente.

## Análise do catálogo

### Por prefixo

| Prefixo | Categoria | Total no catálogo | Já portadas | Falta |
|---|---|---|---|---|
| B | Básicas | 19 | 25 (B01-B25)¹ | **0** (overlap com expanded) |
| C | Cadoc/Composição | 80 | 10 (C01-C10) | **70** |
| F | Faixas | 5 | 15 (F01-F15)² | **0** (overlap) |
| S | Sistemáticas | 110 | 10 (S01-S10) | **100** |
| A | Agregadas | 14 | 0 | **14** |
| I | Individualizadas | 15 | 0 | **15** |
| H | Header/Substituição | 9 | 0 | **9** |
| N | Notas | 1 | 0 | **1** |
| **TOTAL** | | **253** | **60** | **209**³ |

¹ Algumas regras "B" estão além de B19 — verificar sobreposição com regras "S"
² F01-F15 inclui regras extras não-BACEN (custom Fortvna)
³ Diferença entre 361 e 253: ~108 regras no catálogo são "globais" (mesma regra em múltiplos CADOCs) ou metadados

### Por complexidade de implementação

**Tier 1 (fácil, ≤30min/regra, struct-only):** regras que validam presença/formato de campos (Básicas + Header). Já 90% cobertas.

**Tier 2 (médio, 1-2h/regra, lógica simples):** regras de faixas + composição (Faixas + parte das Cadoc). Maioria das F01-F15 e C01-C10 já portadas.

**Tier 3 (complexo, 2-4h/regra, cross-field):** regras Agregadas (A01-A15) — envolvem agregação matemática + faixas + classificação. **Foco da Fase 1.**

**Tier 4 (muito complexo, 4-8h/regra, cross-doc):** regras Individualizadas (I01-I15) — envolvem cálculo por operação + agregação. Foco da Fase 3.

**Tier 5 (Sistemáticas S11-S70, 1-3h/regra, business logic BACEN):** regras de cálculo PCLD, provisões, ClassOp mapping. Foco das Fases 2-4.

## Decisões arquiteturais

### D-1: Manter padrão atual (struct + interface)

```go
type Regra interface {
    Code() string
    Sheet() string
    Severity() string  // "E" (erro), "A" (aviso), "I" (info)
    Apply(ctx context.Context, doc *Doc3040) error
}
```

**Decisão:** mantém. Já é o padrão de 60 regras existentes, zero overhead pra adicionar mais.

### D-2: Tabela de Classificação × Provisão (A01-A03)

Regras A01-A03 mapeiam `ClassOp` (AA/A/B/C/D/E/F/G/H) × provisão constituída × prazo. Implementação: tabela pré-computada em package-level var.

```go
var classOpProvisaoAA_A = []classOpThreshold{
    {classOp: "AA", provMin: 0.0, provMax: 0.005, prazoMax: 0},
    {classOp: "A",  provMin: 0.005, provMax: 0.01, prazoMax: 0},
    // ...
}
```

Justificativa: tabela estática em memória = O(1) lookup, zero allocation. Catálogo tem ~9 ClassOp × ~5 prazos = 45 entries totais.

### D-3: Helper de agregação (A04, A07, A10, A13)

Regras A04, A07, A10, A13 precisam iterar sobre operações dentro de cada agregado. Implementação: método helper em `Doc3040` (`IterAgregados()`) + agregação in-place.

```go
func (d *Doc3040) ForEachAgregado(fn func(agr *Agregado3040) error) error
```

Justificativa: helper único reusado por N regras, evita duplicação.

### D-4: Validação contra leiaute XSD

Sprint 32 **NÃO** cobre validação XSD estrutural (campo presente, formato). Isso é responsabilidade do `internal/schema` (já existe desde Sprint 21). Sprint 32 cobre apenas **regras semânticas** (coerência, faixas, ClassOp mapping).

Decisão: separar claramente "structural validation" (XSD) de "semantic validation" (regras). XSD continua no `schema/`. Regras continuam no `audit/rules/`.

### D-5: Test data fixtures

Cada regra precisa de:
1. Doc válido que NÃO aciona a regra (sanity)
2. Doc inválido que aciona a regra (positive case)
3. Doc edge case (boundary)

Fixtures vão em `internal/audit/rules/testdata/3040_<codigo>.json`.

## Acceptance criteria

### Fase 1 (esta entrega)

- [x] SPRINT_32_RESEARCH.md (este doc)
- [ ] 15 regras Agreg (A01-A15) implementadas em `internal/audit/rules/3040_agregadas.go`
- [ ] 15 regras com tests em `3040_agregadas_test.go`
- [ ] 15 fixtures em `testdata/3040_A*.json`
- [ ] Build clean (`go build ./...`)
- [ ] Tests PASS com -race
- [ ] Coverage `internal/audit/rules` ≥ 65% (+2pp do 62.8% atual)
- [ ] `audit/rules` registry atualizado com 15 novos codes
- [ ] Zero regressão nas 60 regras existentes

### Fase 2-4 (sprints futuras)

- Fase 2: B16-B25 + C11-C30 + S11-S20 = 35 regras
- Fase 3: A16-A30 + B26-B35 + I01-I15 = 50 regras (mas A16-A30 não existem — só A01-A15; ajustar pra cobrir I01-I15 + N1 + H01-H09 = 25, ajustar com B extras = 50)
- Fase 4: C31-C80 + S21-S70 = 60 regras (subset — algumas S requerem cross-doc)

### Verificação de consistência

Após Fase 4:
- [ ] 220/361 regras (60.9%)
- [ ] README atualizado: `Regras portadas Go: 220 de 3040 = 60.9%`
- [ ] CHANGELOG atualizado
- [ ] Validação Profunda separada

## Riscos

### R-1: Complexidade de algumas regras S (Sistemáticas)

Regras S40-S70 envolvem cálculo PCLD (Provisão para Créditos de Liquidação Duvidosa) que tem fórmula complexa:

```
PCLD = (VlrVenc × ClassOp%) - ProvConstit
```

Regras S42-S50 podem precisar de tabela de ClassOp × provision % que muda por BACEN (resolução 2682/1999 com várias atualizações). Manter tabela sincronizada com BACEN é overhead operacional.

**Mitigação:** começar por regras S "fáceis" (S11-S30 = presença/formato) e adiar S40-S70 pra Fase 4 onde pode ter suporte dedicado.

### R-2: Cobertura de testes vs tempo

Cada regra precisa de 3-5 testes (positive + negative + edge cases). 220 regras × 4 testes = 880 testes. Vai ser suite grande.

**Mitigação:** table-driven tests (1 test function por regra, múltiplas cases via slice). Reduz overhead de boilerplate.

### R-3: Drift entre catálogo e implementação

Catálogo BACEN publica atualizações periódicas. Se implementação Go fica atrás, regras vão dar falso negativo.

**Mitigação:** Sprint 35+ (CI-Gate) adiciona teste que valida "toda regra do catálogo tem implementação". Hoje aceito YAGNI.

## Dependências externas

Nenhuma. Tudo é trabalho de implementação + leitura de catálogo + tests.

## Carry-over de sprints anteriores

- **Sprint 23-27 (senhaws + BACEN STA):** zero dependência.
- **Sprint 28 (VaultIntegration):** zero dependência.
- **Sprint 21 (Schema Registry):** schema validation já existe; Sprint 32 foca só em regras semânticas.

## Métricas de sucesso da Fase 1

| Métrica | Target | Como medir |
|---|---|---|
| Regras Agregadas portadas | 15 | grep -c "return \"A" 3040_agregadas.go |
| Test functions Fase 1 | ≥ 15 | go test -v ./audit/rules -run TestA |
| Coverage audit/rules | ≥ 65% (+2.1pp) | go test -cover ./audit/rules/... |
| Build smoke | 10/10 | loop em cmd/*/ |
| Race clean | sim | go test -race |
| Regressão | 0 | full -race suite |

## Próximos passos pós-Fase 1

1. **Fase 2** (próxima sprint após atual): B16-B25 + C11-C30 + S11-S20 = 35 regras → 30.5%
2. **Sprint 33 (Audit3050)**: começa paralelo após Sprint 32 fechar Fase 2
3. **Sprint 35+ (CI-Gate)**: adiciona gate que valida "todas regras do catálogo implementadas"

## Referências

- Catálogo BACEN scr3040_criticas: `_catalogos/criticas.json` (253 entries 3040)
- Leiaute 3040 XSD: `_catalogos/3040_generated.xsd`
- Padrão atual: `internal/audit/rules/3040.go` + `3040_expanded.go`
- Registry: `internal/audit/rules/registry.go`
- Plano Ouro §3.4 (Épico D — Norma Engine)
- ROADMAP.md linha 18

---

**Verdict:** Research sólida. Fase 1 entregue. Demais fases são incrementais. Decisão arquitetural D-2 (tabela ClassOp × provisão) é replicável pras I01-I15 (Tier 4) e S40-S70 (Tier 5).
