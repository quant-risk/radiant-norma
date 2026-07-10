# MEMO: FOCO ABSOLUTO — Motor de Geração CADOCs

**Data:** 2026-07-09
**Prioridade:** CRÍTICA
**Status:** EM ANDAMENTO

---

## OBJETIVO

Construir **100% do motor de geração de CADOCs** — sem pular sprints, sem atalhos. Só parar quando TODOS os CADOCs estiverem funcionando.

---

## CADOCs QUE PRECISAM DE GENERATOR

| CADOC | Nome | Generator Status | Prioridade |
|---|---|---|---|
| 3040 | SCR - Risco de Crédito | ✅ gen3040 OK | — |
| 3050 | TXB - Estatísticas | ✅ gen3050 OK | — |
| 4111 | COSIF - Plano Contas | ❌ Falta | Alta |
| 2061 | DLO - Limites Operacionais | ❌ Falta | Alta |
| 2062 | DLI - Limites Individuais | ❌ Falta | Alta |
| 2070 | DDR - Requerimento Capital | ❌ Falta | Alta |
| 2160 | DRL - Liquidez (LCR) | ❌ Falta | Alta |
| 2170 | DLP - Liquidez LP (NSFR) | ❌ Falta | Alta |
| 2060 | DRM - Risco de Mercado | ❌ Falta | Alta |
| 2030 | DRSAC - ESG | ❌ Falta | Baixa (material não público) |

---

## CONECTORES (ADAPTERS)

| Conector | Status | Prioridade |
|---|---|---|
| Manual (UI) | ⚠️ Wizard | Alta |
| File (XLSX/CSV) | ❌ Stub | Alta |
| API (REST) | ❌ Stub | Alta |
| DB (Postgres) | ❌ Stub | Média |
| MCP (IA) | ❌ Stub | Média |

---

## REGRAS

1. **NÃO pular sprints** — implementar na ordem
2. **Testar cada generator** antes de ir para o próximo
3. **Validar cross-doc** — garantir que CADOCs se conversam
4. **Documentar tudo** — VALIDATION_*.md para cada sprint

---

## ROADMAP

```
Sprint 58: DRM ✅ (validacao)
Sprint 59: DLI ✅ (validacao)
Sprint 60: SDK Go ✅ (validacao)
Sprint 61: SDK Python ✅ (validacao)
Sprint 62: Webhooks ✅ (validacao)

PRÓXIMOS (implementar generator):
- Sprint 63: gen4111 (COSIF)
- Sprint 64: gen2061 (DLO)
- Sprint 65: gen2062 (DLI)
- Sprint 66: gen2070 (DDR)
- Sprint 67: gen2160 (DRL)
- Sprint 68: gen2170 (DLP)
- Sprint 69: gen2060 (DRM)
- Sprint 70: gen2030 (DRSAC) [se material ficar disponível]

FASE 2: Conectores
- File Adapter (XLSX/CSV)
- API Adapter
- DB Adapter
- MCP Adapter
```

---

## VERIFICAÇÃO DE COMPLETUDE

Um CADOC está PRONTO quando:
- [ ] Parser XML aceita leiaute oficial
- [ ] Generator produz XML válido
- [ ] Validação L1-L4 passa
- [ ] Testes cobrindo >80%
- [ ] Cross-doc com outros CADOCs
- [ ] Documentação VALIDATION_*.md

---

**Memorando criado em:** 2026-07-09
**Revisar em:** Sprint 63+
