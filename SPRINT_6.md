# Sprint 6 — Hardening + Postgres + Cross-Doc L3

> **Data proposta:** 2026-07-04+
> **Status:** Proposta (aguarda aprovação)
> **Tema:** Encerrar gaps de hardening + setup multi-DB + diferencial proprietário
> **Versão alvo:** v1.5.0 (minor — features novos)
> **Trigger:** 7ª validação profunda (v1.4.1) fechou gaps de audit emission;
> gaps remanescentes viram Sprint 6

## 🎯 Por que Sprint 6 AGORA

**v1.4.1 fechou auditoria mas 9 gaps permanecem** (ver `VALIDATION_v1.4.0.md`):

| Categoria | Gap | Origem |
|---|---|---|
| Hardening | F3 recordBaseline race window | Validação 7 |
| Hardening | W1+W2 worker retry/lease | Sprint 5 P1 atrasado |
| Hardening | W3 B01-B05 hardcoded | Sprint 4 P2 |
| Hardening | W4 cadoc list hardcoded | Sprint 4 P2 |
| Hardening | R1 triggerRadarScan DOS-via-API | **Validação 8 (v1.4.2)** |
| Test | F6 schema/registry sem tests | Sprint 5 P0 pulado |
| Test | F8 api tests restantes (~80%) | Validação 7 parcial |
| Test | F7 migration idempotência | Validação 7 |
| Feature | L3 cross-doc endpoint | Sprint 5 P2 atrasado |
| Infra | PG driver real + Docker | Sprint 5 P1 atrasado |

**Diferencial competitivo:** Cross-doc L3 (validar 3040 ↔ 4111 ↔ DRSAC) é o que
separa Radiant Norma do BCValidador. Já atrasado 1 sprint.

## 🏛️ Tema da Sprint

**4 frentes paralelas:**

### 🔴 Frente 1 — Hardening P0 (8h)

Fecha gaps críticos que viraram código de produção em Sprint 4 sem hardening
suficiente.

**Escopo:**

- **F3** — `recordBaseline` race window
  - Solução: usar `BEGIN IMMEDIATE` + check + insert numa única tx
  - Postgres: `INSERT ... ON CONFLICT (cadoc_code, alert_type) DO UPDATE`
  - SQLite: `INSERT OR IGNORE` + fallback UPDATE
  - Test: `TestRecordBaseline_Concurrent` (50 goroutines, 1 baseline final)

- **W1** — Worker retry com backoff
  - Adicionar `attempts INT NOT NULL DEFAULT 0` em envios
  - `max_attempts = 5` com exponential backoff (1m, 5m, 30m, 2h, 12h)
  - Audit emission: `envio.retry.scheduled` com attempt count
  - Test: `TestWorker_RetryBackoff` — falha 3x → success na 4ª

- **W2** — Worker lease timeout
  - Stuck detection: envios em `processing` há > 5min → reset pra `pending`
  - Implementar como "sweeper" no boot + a cada 1min
  - Test: `TestWorker_LeaseTimeout` — simula worker crash, verifica reset

- **W3** — B01-B05 hardcoded → registry
  - Mover inline checks em `service.go::applyRegra` (linhas 336-362) para o registry
  - Manter `Builtin3040()` como ponto único de verdade
  - Test: regressão B01-B05 + integração com audit/service

- **W4** — Cadoc list hardcoded → DB
  - Substituir `[]string{"3040", "3050", ...}` em server.go:95, 130
  - Carregar de `schema_versions` (cadocs com versão registrada) ou `criticas`
  - Cachear em memória (5min TTL)
  - Test: handler retorna cadocs do DB

- **R1** — `triggerRadarScan` DOS-via-API risk (descoberto na 8ª validação)
  - Cada POST `/v1/radar/scan` chama `ScanOnce(ctx, nil)` que dispara 3 HTTP
    requests pra bc.gov.br. Sem rate limiting. Vetor de DOS em produção.
  - Solução: rate limit por IF (1 scan/min), cache 5min (último resultado),
    exigir role "admin" no auth (Sprint 6 inclui JWT/OAuth real)
  - Test: `TestTriggerRadarScan_RateLimit` — 5 requests em 1s → 4 com 429

### 🟡 Frente 2 — Testes restantes (6h)

Fecha gaps de cobertura que ficaram de Sprint 5.

**Escopo:**

- **F6** — `internal/schema/registry_test.go`
  - `TestGetEffective` (data exata, data passada, data futura, sem versão)
  - `TestList` (ordenação DESC, multi-version, single-version)
  - `TestInsert` (idempotência UNIQUE(cadoc, effective_from))
  - `TestGetEffective_NoRows` (cadoc inexistente)

- **F7** — Migration 003 idempotência
  - `TestMigrate_Idempotent` — aplica 2x, sem erro
  - `TestMigrate_003_RecreateFromCorrupted` — drop+rebuild
  - Já existe `db/migrate_test.go`? NÃO (gap). Criar.

- **F8** — api tests E2E (~80% restantes)
  - `TestValidate_E2E` — POST /v1/validate, headers, body, audit
  - `TestSTASubmit_E2E` — POST /v1/sta/submit, DB insertion
  - `TestListRules_E2E` — GET /v1/rules/{cadoc}?enabled=true
  - `TestListSchemas_E2E` — GET /v1/schemas/{cadoc}
  - `TestAuthMiddleware_Rejects` — sem X-IF-ID = 401

### 🟢 Frente 3 — Cross-Doc L3 (8h)

**Diferencial proprietário.** Implementa o que BCValidador não tem.

**Escopo:**

- `internal/crossdoc/` package novo
- `CrossDocRule` interface (similar a Rule mas recebe múltiplos docs)
- `internal/crossdoc/rules/3040_4111.go` — 3-5 regras iniciais:
  1. Total de operações 3040 = total de clientes 4111
  2. Modalidades 0213 (cheque especial) batem com flag em 4111
  3. Subsegmento DRSAC ESG compatível com classificação de risco 3040
- `internal/crossdoc/engine.go` — orquestrador (carrega múltiplos docs em paralelo)
- Endpoint `POST /v1/crossdoc/validate` — recebe `{cadocs: {3040: xml, 4111: xml}}`
- Audit: `crossdoc.validated` com metadata `{cadocs: [...], passed: bool, errors: n}`

**Diferencial vs BCValidador:** BCValidador valida UM CADOC por vez.
Radiant Norma valida o ecossistema inteiro (L3 é exclusivo).

### 🔵 Frente 4 — Postgres driver (6h)

Infraestrutura pra produção. Sem Docker local até Sprint 6 = spike fica em SQLite.

**Escopo:**

- Adicionar `github.com/jackc/pgx/v5` em go.mod
- `db.Open` detecta driver via DSN prefix (`postgres://` vs `file:`)
- DSN Postgres: `postgres://user:pass@localhost:5432/radiant?sslmode=disable`
- DSN SQLite: inalterado
- Migration system compatível com ambos (sem features SQLite-specific em 003)
- `docker-compose.yml` na raiz: postgres:16 + radiant-api + radiant-worker
- `Makefile`: target `db-up` (docker compose up), `db-down`, `db-migrate`
- Test: `TestOpen_PostgresDSN` — verifica parsing

## 📦 Entregas previstas

### Documentação
- `SPRINT_6.md` (este doc, atualizado durante sprint)
- `VALIDATION_v1.5.0.md` (validação profunda da release)
- `CHANGELOG.md` (entrada v1.5.0)
- `docs/postgres-setup.md` (como rodar Postgres local)
- `docs/crossdoc-rules.md` (como adicionar regras cross-doc)

### Código

**Frente 1 (hardening) — 6 arquivos:**
- `internal/radar/radar.go` (F3: ON CONFLICT)
- `internal/db/migrations/004_envios_attempts.sql` (W1: nova coluna)
- `cmd/worker/main.go` (W1+W2: backoff + lease)
- `internal/audit/rules/registry.go` + 5 regras novas (W3)
- `internal/api/server.go` (W4: cadoc list do DB)
- `internal/schema/registry.go` (carregar cadocs)

**Frente 2 (tests) — 3 arquivos:**
- `internal/schema/registry_test.go` (F6, 200+ linhas)
- `internal/db/migrate_test.go` (F7, 150+ linhas)
- `internal/api/server_test.go` (F8, +400 linhas)

**Frente 3 (cross-doc) — 5 arquivos:**
- `internal/crossdoc/engine.go` (orquestrador)
- `internal/crossdoc/rule.go` (interface)
- `internal/crossdoc/rules/3040_4111.go` (3-5 regras)
- `internal/crossdoc/rules/registry.go`
- `internal/api/server.go` (endpoint /v1/crossdoc/validate)

**Frente 4 (postgres) — 3 arquivos:**
- `docker-compose.yml`
- `internal/db/db.go` (detecção driver)
- `Makefile` (alvos db-up, db-down, db-migrate)

### CI/CD
- `.github/workflows/test.yml` — adicionar Postgres service container
- `Makefile` — `make db-up` antes de integration tests
- README atualizado com Postgres quickstart

## 📊 Estatísticas previstas (final Sprint 6)

```
Backend Go:
  Linhas: ~4.200 → ~6.500 (+55%)
  Packages: 9 → 10 (+crossdoc)
  Coverage: ~70% → ~75% (+schema, +db migrate, +api restante)
  Regras cross-doc: 0 → 3-5
  Drivers DB: 1 (SQLite) → 2 (+Postgres)

Documentação:
  Markdowns principais: 5 → 7 (+postgres-setup, +crossdoc-rules)
  Validações: 5 → 6 (v1.5.0 doc)
```

## 🎯 Critérios de aceite Sprint 6

### Must-have (P0)
- [ ] F3 race fix + regression test
- [ ] W1 worker backoff + tests
- [ ] W2 worker lease timeout + tests
- [ ] **R1** triggerRadarScan rate limit (DOS prevention) + tests
- [ ] F6 schema tests (≥70% coverage em `internal/schema`)
- [ ] Postgres driver funciona end-to-end (docker compose up + migrate + API up)

### Should-have (P1)
- [ ] W3 B01-B05 no registry
- [ ] W4 cadoc list do DB
- [ ] F7 migration idempotência test
- [ ] F8 api tests ≥60% coverage em `internal/api`
- [ ] Cross-doc L3 com 3+ regras + endpoint funcional

### Nice-to-have (P2)
- [ ] Mutation testing (go-mutesting) em audit/rules
- [ ] Property-based testing em radar (gopter)
- [ ] Benchmark race detector (comparar SQLite vs Postgres)
- [ ] Frontend Next.js dashboard (adiar pra Sprint 7?)

## 🛠️ Stack & ferramentas

**Mantém:**
- Go 1.24+, chi router, SQLite (modernc.org/sqlite pure-Go)
- slog, embed.FS, stdlib testing

**Novas:**
- `github.com/jackc/pgx/v5` (Postgres driver — pure Go, sem CGo)
- `github.com/stretchr/testify` (opcional, P2)
- Docker + docker-compose (Postgres local)

**Cross-doc:**
- Carregar múltiplos CADOCs em paralelo (goroutines + errgroup)
- Compartilhar Doc3040 + Doc4111 entre regras via context

## 📚 Referências

- Go testing: https://pkg.go.dev/testing
- SQLite in-memory: `file::memory:?cache=shared`
- Race detector: https://go.dev/blog/race-detector
- pgx: https://github.com/jackc/pgx
- Postgres ON CONFLICT: https://www.postgresql.org/docs/current/sql-insert.html
- docker-compose: https://docs.docker.com/compose/

---

**Decisão:** Aguardando aprovação do Henrique para iniciar Sprint 6.
Estimativa: 28h de trabalho focado (4 frentes × 6-8h cada, em paralelo).

**Priorização recomendada:**
1. 🔴 Hardening P0 (8h) — fecha gaps críticos
2. 🟡 Testes restantes (6h) — fecha cobertura
3. 🟢 Cross-doc L3 (8h) — diferencial competitivo
4. 🔵 Postgres (6h) — produção

**Trade-off:** Se tempo curto, focar em P0+P1 (14h), Postgres vira Sprint 7.