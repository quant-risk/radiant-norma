# Sprint 3 — Infraestrutura backend Go + API REST

> **Duração:** 2026-07-03 (1 sessão focada, ~25 min de execução efetiva)
> **Tema:** Construir a infraestrutura backend do Radiant Sentinel — API REST, Schema Registry, Sentinel Audit, STA stub. Postgres-ready (SQLite no spike).

## Contexto

Sprint 1 entregou base de conhecimento (catálogo BACEN). Sprint 2 entregou Sentinel Audit spike (CLI Go + XSD gerado + 1.099 regras extraídas). **Sprint 3 fecha o ciclo "dados → serviço consumível"**: o catálogo JSON agora vira API REST com audit log.

## Backlog priorizado

### 🔴 P0 — Fundação (3/3 entregue)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P0.1** | Inicializar projeto backend Go (cmd/api, cmd/seed, internal/*) | ✅ | `backend/go.mod`, 10 arquivos Go |
| **P0.2** | Migrations SQLite/Postgres-ready (5 tabelas) | ✅ | `internal/db/migrations/001_initial.sql` |
| **P0.3** | Worker de seed (JSON → DB) | ✅ | `cmd/seed/main.go` importou 968 críticas + 8 schemas |

### 🟡 P1 — API REST (1/1 entregue)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P1.1** | chi router com 7 endpoints | ✅ | `internal/api/server.go` (250 linhas) |

### 🟢 P2 — Sentinel Audit microservice (2/2 entregue)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P2.1** | Ler críticas do DB, expor POST /v1/validate | ✅ | `internal/audit/service.go` (200 linhas) |
| **P2.2** | Teste end-to-end: XML válido passa, XML quebrado detecta | ✅ | XML 3040 exemplo: 0 erros/2ms; XML quebrado: L1-PARSE |

### 🔵 P3 — Multi-tenant + LGPD (2/2 entregue)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P3.1** | Middleware X-IF-ID + audit log hash chain | ✅ | `internal/auditlog/log.go` (120 linhas) |
| **P3.2** | STA stub (interface + mock) | ✅ | `internal/sta/stub.go` (80 linhas) |

## Definition of Done — Sprint 3

- [x] Backend Go compila sem warnings (`go build ./...`)
- [x] API REST responde em `/healthz` com `{"status":"ok","version":"1.2.0"}`
- [x] 968 críticas importadas no DB (de 6 CADOCs: 3040, 3050, 2061, 2070, 2060, 3044)
- [x] 8 schemas importados no DB
- [x] Sentinel Audit valida XML válido (3040 exemplo, 4832 B): **passed=true, 0 erros, 2ms**
- [x] Sentinel Audit detecta XML quebrado (22 B): **L1-PARSE + B04**
- [x] Audit log com hash chain funcionando (5 entries, chain válido)
- [x] STA stub gera protocolo fake
- [x] Auth: 401 sem X-IF-ID, 404 pra CADOC inexistente
- [x] CHANGELOG v1.2.0 + PDFs regenerados + commits no git

## Estatísticas finais

```
Backend: 10 arquivos Go (~1.400 linhas)
   ├─ cmd/api/main.go              80 linhas (entrypoint + graceful shutdown)
   ├─ cmd/seed/main.go             250 linhas (seed JSON → DB)
   ├─ internal/api/server.go       250 linhas (7 handlers + middleware)
   ├─ internal/audit/service.go    200 linhas (Sentinel Audit)
   ├─ internal/schema/registry.go  140 linhas (Schema Registry)
   ├─ internal/auditlog/log.go     120 linhas (hash chain)
   ├─ internal/sta/stub.go         80 linhas (STA mock)
   ├─ internal/db/db.go            40 linhas (SQLite + abstraction)
   ├─ internal/db/migrate.go       45 linhas (embed.FS migrations)
   └─ migrations/001_initial.sql   110 linhas (5 tabelas)

Banco: SQLite 118 KB
   ├─ schema_versions: 8
   ├─ criticas:        968
   └─ audit_log:       5 (geradas em testes)
```

## Endpoints REST (7 funcionais, testados com curl)

| Método | Path | Teste |
|---|---|---|
| GET | `/healthz` | ✓ `{"status":"ok","version":"1.2.0"}` |
| GET | `/v1/schemas` | ✓ lista 11 CADOCs |
| GET | `/v1/schemas/3040` | ✓ 500 fields + XSD 30 KB |
| GET | `/v1/schemas/3040/versions` | ✓ histórico |
| GET | `/v1/rules/3040` | ✓ **320 regras** carregadas |
| POST | `/v1/validate` (XML OK) | ✓ passed=true, 2ms |
| POST | `/v1/validate` (XML quebrado) | ✓ erros detectados |
| POST | `/v1/sta/submit` | ✓ protocolo fake |
| sem X-IF-ID | ✓ 401 | |
| cadoc inválido | ✓ 404 | |

## Decisões técnicas

| Decisão | Razão | Trade-off aceito |
|---|---|---|
| **SQLite com modernc.org/sqlite** | Pure-Go, sem CGo, ambiente sem Postgres/Docker | Trocar driver pra `pgx` em prod |
| **chi router** | Stdlib-compatible, leve | Poderia usar stdlib `http.ServeMux` (Go 1.22+) |
| **embed.FS migrations** | Self-contained binary | Versão mobile-friendly |
| **X-IF-ID em vez de JWT** | Spike | Sprint 4: JWT + OAuth2 |
| **STA stub antes de Playwright** | Testa end-to-end sem dependência externa | Não exercita STA real ainda |
| **Audit log hash chain (não append-only)** | Tamper-evident | Custo: ~10% slower writes |

## Decisão de linguagem: Go vs Rust

**Resposta documentada em CHANGELOG § v1.2.0**: Go foi escolha pragmática. Time já usa Go (radiant-harness), mercado financeiro BR é Go-heavy, compilação 10x mais rápida, contratação trivial.

Rust faria sentido se:
- HFT/low-latency (< 1ms SLA)
- WASM primário
- Libs públicas (parser de BACEN XML como crate)

**Radiant Sentinel não se encaixa em nenhum.** Latência de network (50ms+) >> GC pauses (1ms).

## Histórico de commits (Sprint 3)

```
61a92cd chore: remover radiant.db do tracking (runtime artifact)
3a1aa7b feat(sprint-3): backend Go com API REST + Schema Registry + Sentinel Audit + STA stub
```

## Gaps remanescentes → Sprint 4

| Gap | Origem | Sprint 4 |
|---|---|---|
| Driver Postgres real | Sem Docker local | `pgx` + config flag |
| JWT/OAuth em vez de X-IF-ID | Spike | `internal/auth/` |
| Postgres RLS (multi-tenant isolado) | X-IF-ID só identifica | Policies por IF |
| STA Web/WS real (Playwright) | Stub | `internal/sta/web.go` |
| Portar 30+ regras semânticas do 3040 | Só B01-B05 implementadas | `internal/audit/rules/3040.go` |
| Cross-doc engine (L3) | Requer 3040+4111 em paralelo | Carregar multi-CADOC |
| Frontend (Sentinel Console) | Backend-only | Next.js dashboard |
| BCValidador real (vs Go) | Docker ausente | Dockerfile + CI |
| Testes unitários | Sem coverage ainda | `*_test.go` + coverage report |

## Próxima sprint — Sprint 4 (preview)

**Tema:** Autenticação real + multi-tenant isolado + STA real
- JWT + OAuth2 com refresh tokens
- Postgres RLS policies
- STA Web client (Playwright)
- Portar 30+ regras semânticas do 3040 (C01-C30)
- Radar regulatório worker (detecta mudanças de leiaute)

**Estimativa:** 30-40h

---

**Autor:** Mavis · Radiant Risk Solutions (marca da Fortvna)
**Stakeholder:** Henrique Costa · henrique@fortvna.com.br