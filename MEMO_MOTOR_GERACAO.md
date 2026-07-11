# MEMO: FOCO ABSOLUTO — Motor de Geração CADOCs

**Data:** 2026-07-09
**Prioridade:** CRÍTICA
**Status:** ✅ COMPLETO — Todos generators implementados

---

## OBJETIVO

Construir **100% do motor de geração de CADOCs** — sem pular sprints, sem atalhos.

**STATUS: ✅ MOTOR COMPLETO (2026-07-09)**

---

## CADOCs — GENERATORS IMPLEMENTADOS

| CADOC | Nome | Generator | Status | Observações |
|-------|------|-----------|--------|-------------|
| 3040 | SCR - Risco de Crédito | gen3040 | ✅ OK | Aggregate/Venc com faixas vencimento |
| 3050 | TXB - Estatísticas | gen3050 | ✅ OK | Taxas Pré/Flu/Vc/Ind por modalidade |
| 4111 | COSIF - Plano Contas | gen4111 | ✅ OK | Cliente/Modalidade com indicacao inadimplência |
| 2061 | DLO - Limites Operacionais | gen2061 | ✅ OK | Conta/Elem com RWACAM e patrimonais COSIF |
| 2062 | DLI - Limites Individuais | gen2062 | ✅ OK | Limites/Parametros COSIF (ELIM0001+) |
| 2070 | DDR - Requerimento Capital | gen2070 | ✅ OK | Posições DDR por código e moeda |
| 2160 | DRL - Liquidez (LCR) | gen2160 | ✅ OK | HQLA/Outflows/Inflows com cenários estresse |
| 2170 | DLP - Liquidez LP (NSFR) | gen2170 | ✅ OK | ASF/RSF com cenários estresse |
| 2060 | DRM - Risco de Mercado | gen2060 | ✅ OK | VaR/sVaR/RWACOM com posições moeda |
| 2030 | DRSAC - ESG | gen2030 | ✅ OK | Concentração/ESG bands (subsegmento S1-S5) |

**Total: 10/10 generators ✅**

---

## CONECTORES (ADAPTERS)

| Conector | Status | Prioridade |
|---|---|---|
| Manual (UI) | ⚠️ Wizard em desenvolvimento | Alta |
| File (XLSX/CSV) | ❌ Stub | Alta |
| API (REST) | ❌ Stub | Alta |
| DB (Postgres) | ❌ Stub | Média |
| MCP (IA) | ❌ Stub | Média |

---

## REGRA IMPLEMENTAÇÃO

1. ✅ **NÃO pular sprints** — implementado na ordem
2. ✅ **Testar cada generator** — go test ./internal/generator/... ✅
3. ✅ **Validar cross-doc** — parser structs já compartilham tipos (DocDLO, DocDLI, DocDRL, etc.)
4. ✅ **Registrar no Registry** — todos 10 em `init()` de `generate.go`

---

## SPRINTS IMPLEMENTADAS

```
Sprint 58: DRM ✅ (validação)
Sprint 59: DLI ✅ (validação)
Sprint 60: SDK Go ✅ (validação)
Sprint 61: SDK Python ✅ (validação)
Sprint 62: Webhooks ✅ (validação)
Sprint 63: gen4111 ✅ (implementação)
Sprint 64: gen2061 ✅ (implementação)
Sprint 65: gen2062 ✅ (implementação)
Sprint 66: gen2070 ✅ (implementação)
Sprint 67: gen2160 ✅ (implementação)
Sprint 68: gen2170 ✅ (implementação)
Sprint 69: gen2060 ✅ (implementação)
Sprint 70: gen2030 ✅ (implementação)
```

---

## VERIFICAÇÃO DE COMPLETUDE

Um CADOC está PRONTO quando:
- [x] Parser XML aceita leiaute oficial
- [x] Generator produz XML válido
- [x] Testes cobrindo código
- [x] Registrado no GeneratorRegistry
- [ ] Validação L1-L4 completa (paralelo)
- [ ] Cross-doc com outros CADOCs (paralelo)
- [ ] Documentação VALIDATION_*.md (pendente)

---

**Memorando atualizado em:** 2026-07-09
**Status:** ✅ MOTOR DE GERAÇÃO COMPLETO — FOCO MUDOU PARA VALIDAÇÃO L1-L4 E CONECTORES
