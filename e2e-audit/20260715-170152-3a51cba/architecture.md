# Architecture — Radiant Norma (estado real observado)

> **Gerado em:** 2026-07-15
> **Run:** 20260715-170152-3a51cba
> **Método:** inspeção de `backend/`, `frontend/`, `sdk/`, ADRs, OpenAPI

---

## Diagrama de componentes (real)

```
┌────────────────────────────────────────────────────────────────────┐
│                       EDGE / INFRA                                 │
│  Cloudflare (DNS+WAF+DDoS) → CloudFront (CDN+TLS) → API GW         │
└─────────────────────────────┬──────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│              Next.js Console (frontend)                            │
│  - app/ (rotas)   - components/   - lib/   - middleware.ts        │
│  - JWT ou cookie httpOnly                                          │
└─────────────────────────────┬──────────────────────────────────────┘
                              │ HTTPS · JWT
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│          Radiant Norma API — cmd/api/main.go (Go chi v5)           │
│  Middleware: CSRF, RateLimit (in-memory + Redis), CORS, Auth       │
│  Handlers (backend/internal/api/):                                 │
│    auth_handlers, csrf, export, generate, insights_handlers,       │
│    insights_llm_handlers, marketplace_handlers, metrics,           │
│    ratelimit (+_redis), schema_changelog_handlers, server.go,      │
│    soc2_handlers, sprint11/20/73/8c_handlers, sse_handler,         │
│    sta_range_handlers, validate, webhook_handlers,                 │
│    wizard_handlers, readyz                                         │
└──────┬──────────────────────────────────────────┬──────────────────┘
       │                                          │
       ▼                                          ▼
┌─────────────────────────────┐    ┌──────────────────────────────────┐
│  Ingest — internal/ingest   │    │  Validation — internal/audit     │
│  SourceAdapter (5)          │    │  L1 XSD (xsd_validator)          │
│   - Manual                  │    │  L2 Sem (service + rules/)       │
│   - File  (file_adapter.go) │    │     27 prod + 24 test files      │
│   - API                     │    │  L3 Crossdoc — engine + rules/   │
│   - DB                      │    │     25 cross-doc rules           │
│   - MCP (JSON-RPC 2.0)      │    │  L4 Historical — l4/engine       │
└─────────────┬───────────────┘    └──────────────┬──────────────────┘
              │ CanonicalDocument                 │
              ▼                                   │
┌─────────────────────────────────────────────────┘
│  Generation — internal/generator/
│  Interface CADOCGenerator:
│    CadocCode, Generate, RequiredFields,
│    SupportedVersions, EstimateComplexity
│  Registry com RegisterDefaults (10 generators):
│    2030 DRSAC, 2060 DRM, 2061 DLO, 2062 DLI,
│    2070 DDR, 2160 DRL, 2170 DLP, 3040 SCR,
│    3050 TXB, 4111 COSIF
│  GeneratedDoc { XML, ZIP, SHA256, FieldMap, Errors }
│  Wizard — generator/wizard/session.go
│  Validation — generator/validation/validate.go
└─────────────┬───────────────────────────────────────┘
              │ XML/ZIP
              ▼
┌────────────────────────────────────────────────────────────────────┐
│   STA Client — internal/sta/                                        │
│   3 interfaces segregadas (ADR-0005):                              │
│     Client (Submit), ReadClient (List/AlterarSituacao),            │
│     ChunkedClient (SubmitRange/DownloadRange)                       │
│   Implementações:                                                   │
│     StubClient (write side) | WSClient (read+write+chunked)        │
│   Retry com backoff exponencial + jitter + circuit breaker          │
│   State machine: pending → processing → accepted/rejected/error     │
└─────────────┬───────────────────────────────────────────────────────┘
              │ HTTPS/TLS+mTLS (quando ativo)
              ▼
       ┌──────────────────────────────────────┐
       │ BACEN STA — sta.bcb.gov.br           │
       │ Senhaws — www9.bcb.gov.br            │
       └──────────────────────────────────────┘

       ╔════════════════════════════════════╗
       ║   Persistência e side-effects      ║
       ╠════════════════════════════════════╣
       ║ DB — modernc.org/sqlite OR         ║
       ║     pgx/v5 (Postgres 16)           ║
       ║   26 migrations (001-026)          ║
       ║   FORCE RLS em 6 tabelas (mig 014) ║
       ║                                     ║
       ║ Cache — Redis 7 (rate limit, SSE)  ║
       ║                                     ║
       ║ AuditLog — internal/auditlog       ║
       ║   SHA-256(prev || payload || meta) ║
       ║                                     ║
       ║ Realtime — internal/realtime        ║
       ║   SSE pub/sub Redis                ║
       ║                                     ║
       ║ Worker — internal/worker           ║
       ║   cmd/worker (background)           ║
       ║                                     ║
       ║ Radar — internal/radar             ║
       ║   fetch BACEN + SHA-256 diff       ║
       ║   cmd/radar (cron)                  ║
       ╚════════════════════════════════════╝

       ╔════════════════════════════════════╗
       ║   Suporte / "supporting"            ║
       ╠════════════════════════════════════╣
       ║ Insights (LLM) — internal/insights ║
       ║ Marketplace — internal/marketplace ║
       ║ Multiregion — internal/multiregion  ║
       ║ SOC2 — internal/soc2               ║
       ║ Pilot — internal/pilot             ║
       ║ Branding — internal/branding       ║
       ║ Billing — internal/billing         ║
       ║ Secrets — internal/secrets         ║
       ║ Senhaws — internal/senhaws         ║
       ║ Webhook — internal/webhook         ║
       ║ Synth (Autodata) — internal/synth  ║
       ║ Schema changelog — schema/         ║
       ║ Ruleprefs — internal/ruleprefs     ║
       ║ Testutil — internal/testutil       ║
       ║ Version — internal/version         ║
       ║ Loggerutil — internal/loggerutil   ║
       ╚════════════════════════════════════╝
```

---

## Inventário de rotas registradas (real)

Total **49 operações** declaradas no OpenAPI. Mapeamento estimado por handler:

| Categoria OpenAPI | Handler arquivo (backend/internal/api/) |
|---|---|
| Health (/healthz, /readyz) | server.go, metrics.go |
| Schema (/v1/schemas, /v1/schemas/{c}) | sprint8c_handlers.go (provável) |
| Schema changelog | schema_changelog_handlers.go |
| Rules (/v1/rules, /v1/rules/{c}) | sprint8c_handlers.go |
| Validate | validate.go |
| Generate | generate.go |
| Generate batch | generate.go |
| Generate history | sprint73_handlers.go |
| Cross-doc | sprint73_handlers.go |
| STA | sprint11_handlers.go (auth+submit básico) + sprint20_handlers.go (submit/zip) + sta_range_handlers.go |
| Radar | sprint8c_handlers.go (alerts/scan) |
| L4 compare | sprint73_handlers.go |
| Envios | sprint11_handlers.go (envios) + sprint20_handlers.go |
| Audit log | sprint11_handlers.go (audit) |
| Insights | insights_handlers.go + insights_llm_handlers.go |
| Marketplace | marketplace_handlers.go |
| Webhooks | webhook_handlers.go |
| Pilot | (interno, possivelmente sprint73) |
| Wizard | wizard_handlers.go |
| Auth | auth_handlers.go |
| CSRF | csrf.go |
| RateLimit | ratelimit.go + ratelimit_redis.go |
| Export | export.go |
| SOC2 | soc2_handlers.go |

(A confirmar Fase D BENCH-07 com diff router real ↔ OpenAPI ↔ SDK ↔ UI.)

---

## Inconsistências e gaps de cobertura

### Cobertura de regras (8 números distintos)

| Fonte | Regras 3040 | Confiabilidade |
|---|---|---|
| README linha 54 (tabela) | **275** | Baixa |
| README linha 158 (curl) | **320** | Baixa |
| README linha 235 (métricas) | **126** | Média |
| README linha 328 (gaps) | **275/361** (76.2%) | Média |
| ROADMAP Sprint 32 | **266** (7 fases 60→266) | Média |
| Diagrama arquitetura (linha 93) | **25** | Baixa |
| `docs/rules-3040-catalog.md` (Sprint 7b) | **60** (5 raw + 55 tipadas) | Alta (escrito para o catálogo) |
| Catálogo bruto declarado | **361** | Média |
| Real (a contar) | **?** (a medir na Fase B) | A confirmar |

### Cobertura de pacotes (MASTER_PLAN §5.1)

| Pacote | Mínimo | Atual declarado | Gap |
|---|---|---|---|
| `audit/rules` | 85% | 62.8% | **-22.2%** (NÃO CONFORME) |
| `audit` | 80% | 77.0% | -3.0% |
| `auditlog` | 95% | 90.8% | -4.2% |
| `auth` | 85% | 71.8% | -13.2% |
| `crossdoc` | 80% | 85.0% | +5.0% |
| `crossdoc/rules` | 70% | 28.3% | **-41.7%** (NÃO CONFORME) |
| `db` | 75% | 68.4% | -6.6% |
| `insights` | 80% | 82.9% | +2.9% |
| `loggerutil` | 95% | 96.2% | +1.2% |
| `radar` | 80% | 81.2% | +1.2% |
| `realtime` | 80% | 79.1% | -0.9% |
| `ruleprefs` | 75% | 60.5% | -14.5% |
| `schema` | 80% | 81.6% | +1.6% |
| `senhaws` | 90% | 95.6% | +5.6% |
| `sta` | 85% | 80.0% | -5.0% |
| `worker` | 80% | 86.1% | +6.1% |
| `api` | 80% | 71.6% | -8.4% |

**NÃO CONFORMES**:
- `audit/rules` (-22.2%)
- `crossdoc/rules` (-41.7%)

`auth`, `ruleprefs`, `sta`, `api`, `audit`, `db` também abaixo do mínimo declarado.

### Outras contradições

| Tópico | README/CHANGELOG | ROADMAP/MASTER_PLAN | Real |
|---|---|---|---|
| Generators entregues | 10 | 10 (Epic I) | 10 registrados (v3.36.3+) |
| Conectores 5: status | 4 stubs + 1 Manual | 5 planejados | 5 com Fetch implementado |
| Cross-doc entregues | 1 (3040↔4111) | 8 XD01-XD08 | **25** (3+8+5+9) |
| Regras 3040 portadas | 275 | 266 (76%) | a contar |
| DRSAC 2030 | 0 regras | 35 regras (D01-D35) | a confirmar |
| Frontend | Next.js 14 | Next.js 15 | a confirmar |
| Versão /healthz | 1.2.0 | 3.36.2 | a confirmar |

---

## Roadmap de execução da auditoria

Próximas fases após A0.3:
1. **Fase B** — `go build ./...`, `go test ./...`, `go test -race ./...`, `go test -cover`, `go vet`, contagem real de regras
2. **Fase C** — fixtures sintéticas + oráculos independentes
3. **Fase D** — 12 benchmarks (BENCH-00 a BENCH-11)
4. **Reports** — scorecard, hard gates, executive, truth-verdict