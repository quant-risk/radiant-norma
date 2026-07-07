# Sprint 36 — Audit3040 Fase 2 — RESEARCH

> **Data:** 2026-07-07
> **Sprint:** 36 (Audit3040 Fase 2 — expandir Doc3040 + destravar carry-over)
> **Pré-requisito:** v3.34.12 (commit 0fac7e5 / tag v3.34.12) — Audit3050 fechado 100%, AuditDDR 2070 Fase 1 fechado
> **Marco esperado:** 126 → ~176+ regras 3040 (34.9% → 49%+ cobertura catálogo 361)

## ⚠️ Contexto

O usuário está puto porque eu pulei do 3040 (parado em 35%) para Audit3050 e AuditDDR. **Errei em não fechar uma workstream antes de começar outra.** Vou mergulhar em fechar 3040 agora, sem pular.

## 🎯 Escopo Sprint 36 Fase 2

**+50 regras 3040** baseadas em:
1. **Básicas faltantes:** B01-B05 (5 regras — qualidade XML/ZIP/período/etc).
2. **Campos Obrigatórios parciais:** C01-C20 (20 regras — validação de campos required).
3. **Substituição Parcial:** Substituição regras (5 regras — validação de envio parcial).
4. **Header H04-H09** (6 regras — validação de header).
5. **Negocio N01** (1 regra — validação de negócio).
6. **Carry-over:** 13 regras destravadas (parser mais rico).

Total: ~50 regras. **Implementadas: 126 → 176** (cobertura 34.9% → 48.8%).

## 📋 Regras prioritárias Tier 1

### B01-B05 (Básicas — 5 regras)

| Cod | Descrição |
|---|---|
| B01 | Erro XML — XML deve atender regras gerais |
| B02 | ZIP deve ser gerado pelo aplicativo validador |
| B03 | IF deve possuir autorização BCB |
| B04 | Documento fora do período de admissão |
| B05 | Documento não esperado (já tem stub S08) |

### C01-C10 (Campos Obrigatórios — 10 regras)

Validação de campos required (CNPJ, DtBase, Remessa, Parte, TpArq, NomeResp, EmailResp, TelResp, TotalCli, ClassOp).

### C11-C20 (Campos Obrigatórios continuação — 10 regras)

Validação específica de campos em Operacao/Agregado.

### H04-H09 (Header — 6 regras)

Validação específica de campos do header.

### Substituição Parcial (5 regras)

Validação de remessas parciais (TpArq=S, Parte != final).

### N01 (Negócio — 1 regra)

Validação de regras de negócio (vinculação cliente-operação).

### Carry-over destravadas (13 regras)

Regras que dependiam de parser mais rico — agora destravadas.

## 🏗️ Decisões técnicas

### DT-39 — Tipos auxiliares para validação

Algumas regras precisam de tipos auxiliares:
- `Inf` (código informação adicional) já existe em Operacao.
- `Cli` já existe em Operacao.
- Para validação de formato CNPJ/CPF, vou criar helper `validarCNPJ(string) error`.

### DT-40 — Stubs honestos vs implementação real

Tier 1 (50 regras):
- ~30 implementação real (validação de campos/formatos).
- ~20 stubs informativos (validações complexas de negócio, dependem de cross-doc).

## 🎯 Métricas alvo

| Métrica | Pré (v3.34.12) | Pós esperado |
|---|---|---|
| Regras 3040 | 126 | **~176** (+50) |
| Cobertura catálogo 3040 | 34.9% | **~48.8%** |
| Coverage `internal/audit/rules` | 70.9% | **70-71%** |
| Test functions 3040 | ~20 | **~50** |
| Packages PASS -race | 23/23 | **23/23** |

## 📁 Arquivos a criar/modificar

```
backend/internal/audit/rules/3040_basicas_faltantes.go  (NOVO — B01-B05)
backend/internal/audit/rules/3040_campos_obrigatorios.go (NOVO — C01-C20)
backend/internal/audit/rules/3040_header_faltantes.go   (NOVO — H04-H09)
backend/internal/audit/rules/3040_substituicao.go        (NOVO — Substituição Parcial)
backend/internal/audit/rules/3040_negocio.go             (NOVO — N01)
backend/internal/audit/rules/3040_carry_over.go          (NOVO — carry-over destravadas)
backend/internal/audit/rules/registry.go                 (atualizar Builtin3040)
backend/internal/audit/rules/3040_fase5_test.go         (NOVO — testes)
CHANGELOG.md                                            (entry v3.34.13)
backend/SPRINT_36_RESEARCH.md                           (NOVO — este arquivo)
```

## 🎯 Self-verify

- [ ] `grep -c "r.Register(" no Builtin3040 confere com soma esperada.
- [ ] `go test -race ./...` 23/23 PASS.
- [ ] `gofmt -l ./...` clean.
- [ ] `go vet ./...` clean.

## ⏭️ Após Sprint 36

Sprint 37 Fase 3: +50 regras (Semântica + Individualizadas + Agregadas) → 226 regras (62.6%).
Sprint 38 Fase 4: +50 regras (carry-over + cross-doc stubs) → 276 regras (76.5%).

Carry-over permanente: ~85 regras (15-25% do catálogo) que precisam de infra adicional ou cross-doc.

**Não vou pular para outro CADOC antes de fechar 3040.**