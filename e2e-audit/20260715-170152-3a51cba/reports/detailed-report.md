# Radiant Norma — Relatório Detalhado de Auditoria E2E

> **Run:** 20260715-170152-3a51cba
> **Data:** 2026-07-15
> **HEAD:** 3a51cba4ce1945c4e554915131617089c9d061bb
> **Working tree:** sujo (13 arquivos modificados em backend/internal/)
> **Modo:** interativo em chunks (BENCHs executados em /tmp/radiant-build/backend)
> **Status:** **FECHADO** — Fase B completa; BENCHs 00, 02, 03, 04, 05, 09 executados em runtime
> **Score final:** 50-69 (PARCIALMENTE FUNCIONAL)

---

## 1. Inventário de claims

**Total**: 279 claims testáveis extraídos de 3 passes:

| Pass | Origem | Claims |
|---|---|---|
| 1 | README, REDESIGN, 8 ADRs | 120 |
| 2 | OpenAPI v1.yaml, LLM/Postgres/Catálogo docs | 77 |
| 3 | ROADMAP, MASTER_PLAN, CHANGELOG (v3.34.44 → v3.36.4) | 82 |

**Distribuição por peso**: 116× peso 5 (core), 136× peso 3 (supporting), 27× peso 1 (marketing).
**Distribuição por categoria**: validacao 52, deployment 52, geracao 51, seguranca 40, ui 26, integracao 21, ingestao 15, contratos 8, compliance 7, negocio 6, observabilidade 1.

## 2. Contradições inter-fonte (resumo das mais graves)

| ID | Tema | Fontes em conflito |
|---|---|---|
| CLM-A0-027/038/047/051/062/198 | Regras 3040 portadas | 25, 60, 126, 266, 275, 275/361, 320 — **oito números distintos** |
| CLM-A0-039/088/273 | Cross-doc entregues | README 1 / ROADMAP 8 / ADR-0006 12 / **Real 25** |
| CLM-A0-014/048/272 | Generators entregues | README 10 / CHANGELOG 1 (3040) / **Real 10 registrados** |
| CLM-A0-100/185/249 | Conectores de ingestão | README "4 stubs + 1 Manual" / LLM docs 5 / **Real 5 com Fetch** |
| CLM-A0-017/069 | Versão Go | README 1.22+ / ADR 1.25+ / go.mod 1.25.0 / instalado 1.26.4 |
| CLM-A0-041/056 | Versão Next.js | README 14 / gaps 15 / **a confirmar** |
| CLM-A0-169 | Prefixo /v1 | OpenAPI mistura `/marketplace` e `/v1/marketplace/*` |
| CLM-A0-193 | Total endpoints | README "20+" / OpenAPI "49 operações em 46 paths" |

## 3. Achados P0 já confirmados (CHANGELOG v3.36.3/v3.36.4)

| Achado | Severidade | Evidência |
|---|---|---|
| `generator.NewRegistry()` retornava registry VAZIO até v3.36.3 | **P0** | CHANGELOG.md:42-48 |
| Manual + File adapters FALTAVAM (só 3 de 5) | **P0** | CHANGELOG.md:48-50 |
| Data race em `staRangeUpload` response | **P0** | CHANGELOG.md:17-20 |
| `err.Error()` leak em `staRangeInit` expondo URL/hostname/status | **P0** | CHANGELOG.md:26-28 |
| Pilot endpoints auth bypass | **P0** | CHANGELOG.md v3.34.52 |
| `audit/rules` cobertura 62.8% < mínimo 85% | **P0** | MASTER_PLAN.md §5.1 |
| `crossdoc/rules` cobertura 28.3% < mínimo 70% | **P0** | MASTER_PLAN.md §5.1 |

## 4. Estado do baseline (Fase B — COMPLETA)

Backend copiado para `/tmp/radiant-build/backend` (filesystem local APFS) e testado em runtime:

- `go mod download` ✅
- `go build ./...` ✅ EXIT 0 — todos os 287 arquivos Go compilam
- `go vet ./...` ✅ EXIT 0 — sem warnings
- `go test -list ./...` → **1244 testes top-level** (claim README "516" subestima 2.4x)
- `go test ./...` → **51 packages PASS, 0 FAIL**
- `go test -cover ./...` → cobertura medida em todos os packages (ver `benchmarks/coverage-all.txt`)
- `go test -race ./internal/...` → **CLEAN** em 6 pacotes críticos (sta, auditlog, crossdoc, crossdoc/rules, audit, audit/rules)

## 5. Resultados BENCH (Fase D)

| BENCH | Status | Resultado runtime |
|---|---|---|
| BENCH-00 build/docs | ✅ | go build EXIT 0, vet EXIT 0, 1244 testes, 51 packages PASS |
| BENCH-01 jornada UI | ❌ | Não executado (Playwright instalado mas tempo esgotado) |
| BENCH-02 generators | ✅ | 10 generators com Generate real + xml.Marshal + tests PASS |
| BENCH-03 cobertura | ✅ | 51 packages cobertura medida; 4 abaixo do mínimo |
| BENCH-04 ingest | ✅ | FileAdapter Fetch_CSV/XLSX + ValidateConfig + HealthCheck PASS |
| BENCH-05 cross-doc | ✅ | 25 regras + 15 testes (Meta, Apply, BuiltinRegistry) PASS |
| BENCH-06 STA real | ❌ | Hetzner credentials recusadas; STA real não testado |
| BENCH-07 frontend | ⚠️ | Router (77 rotas) vs OpenAPI (49 ops) contado por grep |
| BENCH-08 security | ❌ | Não executado runtime |
| BENCH-09 deploy | ✅ | Race detector CLEAN; build clean |
| BENCH-10 supporting | ❌ | Não exercitados |
| BENCH-11 senhaws | ❌ | Não executado |

## 6. Próximas fases (se retomar)

- BENCH-01 (jornada UI completa)
- BENCH-06 (STA real após homolog BACEN)
- BENCH-08 (security runtime: cross-tenant, JWT, CSRF, RLS, IDOR, etc.)
- BENCH-10 (módulos supporting: insights, marketplace, pilot, SOC2)
- BENCH-11 (senhaws rotate + secret-migrate)