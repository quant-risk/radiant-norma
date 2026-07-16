# Defeitos — Auditoria E2E Radiant Norma

> Run: 20260715-170152-3a51cba
> Data: 2026-07-15
> Status: índice completo — 11 defeitos (7 P0, 4 P2)
> Fonte: CHANGELOG v3.36.3/v3.36.4/v3.34.52 + medições runtime (cobertura por pacote)

---

## DEFECT-001 (P0) — `generator.NewRegistry()` retornava registry VAZIO

- **severidade:** P0 — crítico
- **confiança:** alta (confirmado pelo CHANGELOG)
- **claims afetados:** CLM-A0-014, CLM-A0-048, CLM-A0-248, CLM-A0-272
- **ambiente/commit:** HEAD 3a51cba + working tree dirty
- **pré-condições:** nenhuma
- **fixture:** nenhuma (defeito de código)
- **passos:**
  1. Antes do fix v3.36.3: chamar `generator.NewRegistry()` e tentar `Registry.Get("3040")` em produção.
  2. Esperado: retorna instância de `Gen3040`.
  3. Observado: retorna nil.
- **esperado:** Registry pré-populado com 10 generators via `RegisterDefaults`.
- **observado:** Registry vazio; wiring em `cmd/api/main.go` era cosmético.
- **evidência:** CHANGELOG.md linha 42-48 (v3.36.3 fix 1).
- **frequência:** 100% até v3.36.2.
- **impacto:** Generator 3040 e os outros 9 não funcionavam; rotas `/v1/generate/*` retornavam erro silencioso.
- **escopo:** todas as chamadas `Registry.Get(cadoc)` antes do fix.
- **hipótese de causa:** init() de blank imports vs dependency injection; corrigido em v3.36.3 com `generator.RegisterDefaults(r, []CADOCGenerator{...})`.
- **workaround:** nenhum antes do fix.
- **status:** CORRIGIDO em v3.36.3 (CHANGELOG confirma +30 testes).

---

## DEFECT-002 (P0) — Manual + File adapters FALTANDO

- **severidade:** P0 — crítico
- **confiança:** alta
- **claims afetados:** CLM-A0-049, CLM-A0-100, CLM-A0-185, CLM-A0-249, CLM-A0-257
- **ambiente/commit:** HEAD 3a51cba + working tree dirty
- **fixture:** nenhuma (defeito de código)
- **passos:**
  1. Antes do fix v3.36.3: verificar presença de `ManualAdapter` e `FileAdapter` no código.
  2. Esperado: ambos implementados.
  3. Observado: apenas DB, API, MCP existiam (3 de 5).
- **esperado:** 5 conectores funcionais.
- **observado:** 3 stubs + 0 Manual + 0 File; apenas um subconjunto.
- **evidência:** CHANGELOG.md linha 48-50 (v3.36.3 fix 2); LLM_INTEGRATION_GUIDE.md:55 ("MCPAdapter retorna ErrNotImplemented" — desatualizado); ON_PREMISE_LLM_SPEC.md:17 (afirma 5 conectores incluindo MCP — otimista).
- **frequência:** 100% antes do fix.
- **impacto:** Wizard `/console/generate` oferecia opções sem implementação; UX promete e não entrega.
- **escopo:** rotas de ingestão manual e upload de arquivos.
- **hipótese de causa:** backlog intencional para Sprint 57 follow-up.
- **workaround:** usar APIAdapter ou DBAdapter como proxy.
- **status:** CORRIGIDO em v3.36.3.

---

## DEFECT-003 (P0) — Data race em `staRangeUpload` response

- **severidade:** P0 — crítico
- **confiança:** alta
- **claims afetados:** CLM-A0-043, CLM-A0-245
- **ambiente/commit:** HEAD 3a51cba + working tree dirty
- **fixture:** nenhuma (defeito de concorrência)
- **passos:**
  1. Disparar 2 PUTs concorrentes no mesmo protocolo via `/v1/sta/range-upload`.
  2. Antes do fix v3.36.4: `Session.ReceivedBytes/Ranges/Status` lidos FORA do `sessionsMu.Lock`.
  3. Observado: data race detectada por `go test -race`.
- **esperado:** todas as leituras e escritas protegidas por `sessionsMu`.
- **observado:** leitura sem lock após PUTs concorrentes.
- **evidência:** CHANGELOG.md linha 17-20 (v3.36.4 fix H2); arquivo dirty `backend/internal/api/sta_range_handlers.go`.
- **frequência:** 100% em cargas concorrentes.
- **impacto:** corrupção de estado, respostas incorretas, panic intermitente.
- **escopo:** todas as sessões STA Range Upload.
- **hipótese de causa:** design inicial sem considerar concorrência.
- **workaround:** nenhum.
- **status:** CORRIGIDO em v3.36.4 (snapshot completo sob lock + cópia do slice `Ranges`).
- **validação runtime pós-fix:** `go test -race ./internal/sta/` exit 0 (clean).

---

## DEFECT-004 (P0) — `err.Error()` leak expondo URL/hostname/status do BACEN

- **severidade:** P0 — crítico (viola F18.1 do MASTER_PLAN)
- **confiança:** alta
- **claims afetados:** CLM-A0-068, CLM-A0-218, CLM-A0-247
- **ambiente/commit:** HEAD 3a51cba + working tree dirty
- **fixture:** nenhuma (defeito de info disclosure)
- **passos:**
  1. Disparar POST `/v1/sta/range-init` com BACEN mockado retornando 5xx.
  2. Antes do fix v3.36.4: response body inclui "BACEN rejeitou init: <err.Error>()".
  3. Observado: URL/hostname/status expostos ao cliente.
- **esperado:** mensagem genérica "serviço indisponível"; log detalhado apenas server-side.
- **observado:** `s.userError()` retornava `err.Error()` raw.
- **evidência:** CHANGELOG.md linha 26-28 (v3.36.4 fix M5).
- **frequência:** 100% em falhas de BACEN.
- **impacto:** Information disclosure; atacante aprende topologia interna.
- **escopo:** todas as respostas de erro STA Range.
- **hipótese de causa:** debug deixado em prod.
- **workaround:** nenhum sem patch.
- **status:** CORRIGIDO em v3.36.4 (log server-side detalhado + `s.userError()` genérico).

---

## DEFECT-005 (P0) — Pilot endpoints auth bypass

- **severidade:** P0 — crítico (auth bypass)
- **confiança:** alta
- **claims afetados:** CLM-A0-068, CLM-A0-271
- **ambiente/commit:** HEAD 3a51cba + working tree dirty
- **fixture:** nenhuma (defeito de auth)
- **passos:**
  1. Antes do fix v3.34.52: chamar `/v1/pilot/*` sem token.
  2. Observado: retorna 200.
- **esperado:** 401 Unauthorized.
- **observado:** bypass total.
- **evidência:** CHANGELOG.md v3.34.52 linha 696.
- **frequência:** 100% antes do fix.
- **impacto:** qualquer um pode criar/ler/modificar dados de piloto.
- **escopo:** endpoints de pilot.
- **hipótese de causa:** middleware não aplicado.
- **workaround:** nenhum.
- **status:** CORRIGIDO em v3.34.52.

---

## DEFECT-006 (P0) — Cobertura `audit/rules = 61.6%` < mínimo 85%

- **severidade:** P0 — bloqueador (CI deveria falhar)
- **confiança:** alta (medido em runtime, não só declarado)
- **claims afetados:** CLM-A0-033, CLM-A0-034, CLM-A0-226, CLM-A0-276
- **ambiente/commit:** HEAD 3a51cba (build em /tmp/radiant-build/backend)
- **medição:** `go test -count=1 -cover ./internal/audit/rules/` → **61.6%**
- **passos:**
  1. Rodar `go test -cover ./internal/audit/rules/`.
  2. Esperado: ≥ 85%.
  3. Observado: 61.6%.
- **evidência:** `benchmarks/coverage-all.txt`
- **impacto:** mutações podem não ser detectadas; cobertura insuficiente para o claim de "production-grade".
- **escopo:** todas as regras 3040/2070/3026.
- **status:** **NÃO CORRIGIDO — CONFIRMADO EM RUNTIME**. CI deveria estar falhando. Verificar se gate de cobertura está aplicado.

---

## DEFECT-007 (P0) — Cobertura `crossdoc/rules = 23.4%` < mínimo 70%

- **severidade:** P0 — bloqueador (CI deveria falhar)
- **confiança:** alta (medido em runtime)
- **claims afetados:** CLM-A0-039, CLM-A0-088, CLM-A0-273, CLM-A0-276
- **medição:** `go test -count=1 -cover ./internal/crossdoc/rules/` → **23.4%**
- **passos:**
  1. Rodar `go test -cover ./internal/crossdoc/rules/`.
  2. Esperado: ≥ 70%.
  3. Observado: 23.4%.
- **evidência:** `benchmarks/coverage-all.txt`
- **impacto:** regras cross-doc (XD01-XD12 + DRSAC + 4111) podem falhar silenciosamente; risco regulatório elevado (cross-doc é o **moat proprietário**).
- **status:** **NÃO CORRIGIDO — CONFIRMADO EM RUNTIME**. CI deveria estar falhando.

---

## DEFECT-008 (P0) — Cobertura `audit = 25.1%` < mínimo 80%

- **severidade:** P0 — bloqueador (CI deveria falhar)
- **confiança:** alta (medido em runtime)
- **claims afetados:** CLM-A0-226, CLM-A0-276
- **medição:** `go test -count=1 -cover ./internal/audit/` → **25.1%**
- **gap:** -54.9pp (mais grave que audit/rules)
- **evidência:** `benchmarks/coverage-all.txt`
- **impacto:** L1/L2/L3/L4 layers têm testes fracos; mutações podem não ser detectadas em runtime.
- **status:** **NÃO CORRIGIDO — CONFIRMADO EM RUNTIME**.

---

## DEFECT-009 (P0) — Cobertura `api = 58.0%` < mínimo 80%

- **severidade:** P0 — bloqueador (CI deveria falhar)
- **confiança:** alta (medido em runtime)
- **claims afetados:** CLM-A0-043, CLM-A0-226
- **medição:** `go test -count=1 -cover ./internal/api/` → **58.0%**
- **gap:** -22.0pp
- **evidência:** `benchmarks/coverage-all.txt`
- **impacto:** 77 rotas REST + 49 operações OpenAPI com testes fracos; risco de regressões em rotas críticas.
- **status:** **NÃO CORRIGIDO — CONFIRMADO EM RUNTIME**.

---

## DEFECT-010 (P2) — OpenAPI mistura prefixo `/v1`

- **severidade:** P2 — médio
- **confiança:** alta (visível no spec)
- **claims afetados:** CLM-A0-169
- **passos:**
  1. Inspecionar `docs/openapi/v1.yaml`.
  2. Observado: paths `/marketplace`, `/webhooks` (sem prefixo) vs `/v1/marketplace/*`, `/v1/webhooks/*` (com prefixo).
- **evidência:** docs/openapi/v1.yaml linhas 1239, 1312, 1340, 1381, 1406, 1458, 1480, 1518, 1548.
- **impacto:** inconsistência no contrato; clientes podem esperar que todos tenham `/v1`.
- **status:** ABERTO.

---

## DEFECT-011 (P2) — `MCP_<NAME>_ENDPOINT` env var não documentada

- **severidade:** P2 — médio (risco operacional)
- **confiança:** alta
- **claims afetados:** CLM-A0-105, CLM-A0-185
- **passos:**
  1. Tentar usar MCPAdapter em produção.
  2. Observado: retorna erro "servidor MCP X não encontrado (configure a variável MCP_<NAME>_ENDPOINT)".
- **evidência:** backend/internal/ingest/adapter.go:1042 (resolveEndpoint).
- **impacto:** setup confuso; nenhuma documentação em README, ADR ou LLM Integration Guide.
- **status:** ABERTO.

---

## DEFECT-012 (P3) — Claim "516 testes" diverge de 1244 reais (subestima 2.4x)

- **severidade:** P3 — baixo (métrica)
- **confiança:** alta
- **claims afetados:** CLM-A0-033, CLM-A0-230
- **medição:** `go test -list '.*' ./... | wc -l` em /tmp/radiant-build/backend → **1244 testes top-level**
- **passos:**
  1. README linha 230 declara "516 testes top-level".
  2. `go test -list` retorna 1244.
- **evidência:** `benchmarks/test-list.txt`
- **impacto:** métrica desatualizada; ou README conta apenas testes sem subtests, ou subestima.
- **status:** ABERTO — atualizar README ou investigar contagem alternativa.

---

## Resumo

| Severidade | Total | Corrigidos | Em aberto |
|---|---|---|---|
| P0 | 9 | 5 | 4 (audit/rules, crossdoc/rules, audit, api — todas por cobertura) |
| P2 | 2 | 0 | 2 |
| P3 | 1 | 0 | 1 |
| **Total** | **12** | **5** | **7** |

**Hard gates acionados em runtime**: 4 pacotes com cobertura abaixo do mínimo declarado no MASTER_PLAN §5.1 (audit/rules, crossdoc/rules, audit, api). CI deveria estar falhando — verificar gate de cobertura.