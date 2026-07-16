# Product Model — Radiant Norma (estado real observado)

> **Gerado em:** 2026-07-15 (audit run 20260715-170152-3a51cba)
> **Branch/HEAD:** main @ 3a51cba4ce1945c4e554915131617089c9d061bb
> **Working tree:** sujo — 13 arquivos modificados (em backend/internal/)
> **Método:** inspeção direta do código via `find`/`grep` sobre `backend/`, `frontend/`, `sdk/`

---

## 1. Inventário de código (real)

### 1.1 Backend Go

| Item | Valor real | Origem |
|---|---|---|
| Arquivos `.go` totais | **287** | `find backend -name '*.go' \| wc -l` |
| Linhas totais | (medido após Fase B) | `find … -exec wc -l + \| tail -1` |
| Binários (cmd/*) | **13**: `_verify, api, generator-server, jwt-mint, radar, secret-migrate, seed, seed-demo, seed-sprint8c, senhaws-rotate, sta-submit, synth-gen, worker` | `ls backend/cmd/` |
| Pacotes internal/ | **35** | `ls backend/internal/` |
| Pacotes internal/api/ (handlers) | **~46 arquivos** incluindo tests | `ls backend/internal/api/ \| wc -l` |
| Migrations SQL | **26** (001 → 026) em `backend/internal/db/migrations/` | `find backend -name '*.sql'` |
| Coverage declarado pelo MASTER_PLAN | `audit/rules 62.8% (min 85%)`, `crossdoc/rules 28.3% (min 70%)`, `db 68.4% (min 75%)`, `auth 71.8% (min 85%)`, `api 71.6% (min 80%)` | MASTER_PLAN §5.1 |

### 1.2 Generators registrados (v3.36.3+)

10 generators em `backend/internal/generator/gen*/`:

| CADOC | Diretório | Tem generator.go | Tem test |
|---|---|---|---|
| 2030 DRSAC | `gen2030/` | ✅ | ✅ |
| 2060 DRM | `gen2060/` | ✅ | (não confirmado) |
| 2061 DLO | `gen2061/` | ✅ | (não confirmado) |
| 2062 DLI | `gen2062/` | ✅ | (não confirmado) |
| 2070 DDR | `gen2070/` | ✅ | (não confirmado) |
| 2160 DRL | `gen2160/` | ✅ | (não confirmado) |
| 2170 DLP | `gen2170/` | ✅ | (não confirmado) |
| 3040 SCR | `gen3040/` | ✅ | ✅ |
| 3050 TXB | `gen3050/` | ✅ | (não confirmado) |
| 4111 COSIF | `gen4111/` | ✅ | (não confirmado) |

Interface `CADOCGenerator` em `backend/internal/generator/interface.go` com:
- `RegisterDefaults(r *Registry, generators []CADOCGenerator)`
- `Register(g CADOCGenerator)`
- `Get(cadocCode string) CADOCGenerator`
- `List() []CADOCGenerator`
- `IsRegistered(cadocCode string) bool`

**Histórico crítico**: até v3.36.2 `generator.NewRegistry()` retornava registry VAZIO (CLM-A0-248); wiring em `cmd/api/main.go` era cosmético. Fix em v3.36.3 chama `RegisterDefaults`.

### 1.3 Adapters de ingestão

Localização: `backend/internal/ingest/adapter.go` (1315 linhas, monolítico) + `file_adapter.go` (518 linhas, separado).

5 adapters declarados, todos com `Fetch`, `ValidateConfig`, `HealthCheck`:

| Adapter | Localização | Status observado |
|---|---|---|
| ManualAdapter | `adapter.go:233` | Implementado; cria Canonical vazio com `cfg.Manual.Fields` em `Extra` |
| FileAdapter | `file_adapter.go` (arquivo separado) | Implementado; parseia CSV (RFC 4180), JSON, XLSX flattened |
| APIAdapter | `adapter.go:312` | Implementado (~300 linhas para Fetch); HTTP REST |
| DBAdapter | `adapter.go:676` | Implementado (~260 linhas); PostgreSQL/MySQL |
| MCPAdapter | `adapter.go:989` | Implementado (~280 linhas); JSON-RPC 2.0 para endpoint MCP |

`ErrNotImplemented` está declarado mas **não é retornado por nenhum dos Fetch()** (verificação por inspeção).

### 1.4 Validação / audit

`backend/internal/audit/`:
- `service.go` — orquestrador
- `xsd_validator.go` — L1 XSD
- `rules/` — **27 arquivos de produção** + 24 de teste
- `l4/` — engine de comparação histórica (alert.go, diff_expanded.go, dlo_diff.go, engine.go)

Regras em `backend/internal/audit/rules/`:
- `3040.go`, `3040_expanded.go`, `3040_fase4.go`, `3040_agregadas.go`, `3040_individuais.go`, `3040_sistematicas.go`, `3040_sprint36.go`, `3040_sprint37.go`
- `2070.go`, `3026.go`
- (sem `3050.go`, `3044.go`, `4111.go`, `drsac.go` explícito na lista de production files — pode estar em sub-pacotes)

**Inconsistência crítica**: cobertura declarada `audit/rules = 62.8%` está **abaixo do mínimo 85%** (MASTER_PLAN §5.1). CI deveria estar falhando.

### 1.5 Cross-doc (L3)

`backend/internal/crossdoc/rules/registry.go` confirma:

**25 regras cross-doc registradas** (Sprint 52 v3.34.33):
- **3 iniciais**: `TotalOperacoes3040Consistente4111`, `Modalidade0213FlagChequeEspecial`, `DRSACSubsegmentoClassificacaoRisco`
- **8 DRSAC↔SCR** (XD-DR01..DR08): IPOC, saldo, cliente, CNAE, risco social, ambiental, TVM, green
- **5 4111↔3040** (XD-4111-01..05): CNPJ, totais clientes vs ops, inadimplentes, data-base, zerados
- **9 XD02-XD12** (engine genérico): XD02 total 3040↔3050, XD03 LCR↔NSFR, XD06 APR em LCR, XD07 triangulação 3040↔4111↔3050, XD08 limites↔capital, XD09 liquidez↔risco, XD10 ESG↔inadimplência, XD11 ESG↔4111, XD12 data-base consistente

**Veredito**: contradiz todas as fontes externas. README: 1 entregue; ROADMAP: 8 XD01-XD08; ADR-0006: 12 XD01-XD12; **Real: 25** (3+8+5+9).

### 1.6 STA (Submissão ao BACEN)

`backend/internal/sta/`:
- `stub.go` — StubClient (apenas write side, conforme ADR-0005)
- `ws.go` — WSClient (read + write + chunked)
- `ws_types.go` — tipos JSON
- `retry.go` — backoff exponencial
- `smoke_test.go`, `retry_test.go`, `ws_test.go`

ADR-0005 confirmado: 3 interfaces segregadas (Client/ReadClient/ChunkedClient).

**Achado v3.36.4 fix H2**: data race em `staRangeUpload` response — `session.ReceivedBytes/Ranges/Status` lidos FORA do lock. Corrigido em v3.36.4.

### 1.7 Schema registry

`backend/internal/schema/`: `registry.go`, `changelog.go`, `cache_stampede_test.go`.

### 1.8 Frontend (Next.js)

`frontend/src/`: estrutura `app/`, `components/`, `lib/`, `middleware.ts`.

README linha 238 diz **56 arquivos TS/TSX · 7.108 LoC · Next.js 14**.
Gaps conhecidos linha 335 dizem **Next.js 15**.

**Inconsistência**: precisa `grep "next": " package.json` para confirmar versão real.

### 1.9 SDKs

- `sdk/go/` — Go SDK oficial (`client.go`, `types.go`, `radiant/` subdir)
- `sdk/py/` — Python SDK (mencionado pelo README)
- `sdk/python/` — **segundo** Python SDK (não mencionado pelo README)

### 1.10 Modulos "supporting" anunciados

`internal/marketplace/` — Sprint 62 ✅ (publish/install/rate)
`internal/multiregion/` — Sprint 63 ✅ (BR-SP1/SP2 replication)
`internal/soc2/` — Sprint 65 ✅ (continuous evidence collection)
`internal/pilot/` — Sprint 64 ✅ (v3.34.46)
`internal/branding/` — Sprint 46 ✅ (white-label)
`internal/billing/` — Stripe (Sprint 45 ✅)
`internal/secrets/` — AWS Secrets Manager / Vault (Sprint 28 ✅)
`internal/senhaws/` — Bacen senhas rotation (Sprint 23+ ✅)
`internal/webhook/` — Sprint 61 ✅ (registry + delivery worker + REST API)
`internal/insights/` — AI Insights (Sprint 53 ✅)
`internal/realtime/` — SSE

---

## 2. Fluxo real (descoberto)

```text
fonte (Manual/File/API/DB/MCP)
   ↓ SourceAdapter.Fetch
CanonicalDocument (internal/canonical)
   ↓ CADOCGenerator.Generate (registry.Get)
GeneratedDoc { XML, ZIP, SHA256, FieldMap, Errors }
   ↓ POST /v1/validate (ValidateFull)
   ├─ L1 XSD (internal/audit/xsd_validator)
   ├─ L2 Semântico (internal/audit/service + rules/)
   ├─ L3 Cross-doc (internal/crossdoc/engine + rules/)
   └─ L4 Histórico (internal/audit/l4/engine)
   ↓ /v1/sta/submit
STA Client (WSClient ou StubClient)
   ↓
BACEN (sta.bcb.gov.br ou sta-h)
   ↓
auditlog.HashChain (internal/auditlog)
   ↓
audit_log (SQLite/Postgres, append-only)
   ↓
SSE → frontend (internal/realtime, /v1/events/stream)
```

---

## 3. Trust boundaries

1. **Internet ↔ API**: Cloudflare → CloudFront → API (TLS 1.3, HSTS, CSP, X-Frame-Options)
2. **Frontend ↔ API**: JWT ou cookie httpOnly + SameSite=Lax/Strict
3. **API ↔ DB**: pgx/v5 (Postgres) ou modernc.org/sqlite (SQLite)
4. **API ↔ Redis**: go-redis/v9
5. **API ↔ Ollama/MCP**: loopback HTTP (planejado, não ativo)
6. **API ↔ BACEN STA**: TLS + mTLS (quando ativado), com senhaws rotacionado via Vault/Secrets Manager
7. **Tenants**: isolados por `SET LOCAL app.current_if_id` em **toda** transação + Postgres RLS

---

## 4. Bifurcações observadas

- **SQLite vs Postgres**: dual driver, detecção por prefixo da DSN. Migration 004 com `INSERT OR IGNORE` **não roda em Postgres** (limitação documentada em `docs/postgres-setup.md`).
- **STA Stub vs WS**: `RADIANT_STA_BACKEND=ws` ativa WSClient; default é StubClient (apenas write side).
- **5 conectores × Manual-only vs todos**: a v3.36.0 dizia apenas Manual funcional; v3.36.3 adicionou Manual+File ao registry, mas **TODOS os 5 estão com Fetch implementado** (verificado por inspeção).
- **Dev cookie vs JWT prod**: v3.35.5 introduziu login direto via cookie dev (sem JWT bridge). P5 do MASTER_PLAN diz "Defense-in-depth em secrets" — simplificação de dev é aceitável desde que prod force JWT.

---

## 5. Componentes não wired / stubs residuais

- **`MCPAdapter.resolveEndpoint`** depende de variável de ambiente `MCP_<NAME>_ENDPOINT` que **não está documentada em lugar algum** — produção real exige configuração manual.
- **`APIAdapter.Fetch`** é o maior dos 5 (~285 linhas para Fetch), com parsing de GET/POST/paginação. **Risco**: complexidade alta, sem cobertura de testes dedicada aparente.
- **Sprint 32 finalizou com 266 regras portadas** (ROADMAP). Nenhuma fonte externa bate: README diz 275; gaps conhecidos dizem 275/361; CLI retorna 320; catálogo Sprint 7b declara 60. **Conflito grave** — auditoria precisa abrir `internal/audit/rules/` e contar `func` exports.
- **Cobertura `audit/rules = 62.8%` < mínimo 85%** (MASTER_PLAN §5.1) — não-conformidade declarada.
- **Cobertura `crossdoc/rules = 28.3%` < mínimo 70%** — não-conformidade declarada.

---

## 6. Achados P0 a investigar (continuação Fase B/D)

| ID | Achado | Origem |
|---|---|---|
| P0-1 | `audit/rules` 62.8% < 85% mínimo | MASTER_PLAN §5.1 |
| P0-2 | `crossdoc/rules` 28.3% < 70% mínimo | MASTER_PLAN §5.1 |
| P0-3 | 13 arquivos dirty em backend/internal/ — incluindo `xd_rules.go`, `sta_range_handlers.go`, `insights_llm_handlers.go` | git status |
| P0-4 | OpenAPI mistura `/marketplace` e `/v1/marketplace/*` (paths sem prefixo `/v1`) | openapi/v1.yaml |
| P0-5 | Adapter `ErrNotImplemented` declarado mas não retornado — código morto | ingest/adapter.go |
| P0-6 | Migration 004 com `INSERT OR IGNORE` quebra em Postgres | docs/postgres-setup.md |
| P0-7 | `MCP_<NAME>_ENDPOINT` env var não documentada | ingest/adapter.go:1042 |
| P0-8 | `dev-private.pem` no repo (verificar se é teste ou produção) | backend/dev-private.pem |