# Changelog — cadocs (Radiant Norma)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

## v1.5.0 — 2026-07-03 (Sprint 6: Hardening P0 + Cross-Doc L3 + Postgres driver) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 6 (ver `SPRINT_6.md` + `SPRINT_6_RESULTS.md`)
> **Trigger:** 11 gaps acumulados de v1.4.1-v1.4.4 + DOS-via-API risk (R1)
> **Versão:** minor (features novos)

### 🎯 Resumo

Hardening crítico (P0): F3 race fix, W1+W2 worker hardening, R1 DOS-via-API
prevention. Testes restantes (F6, F7, F8). Diferencial proprietário cross-doc
L3 com 3 regras iniciais. Driver dual SQLite/Postgres via DSN detection.

### 🐛 Bugs corrigidos

| # | Bug | Sev | Origem | Fix |
|---|---|---|---|---|
| F3.1 | `recordBaseline` UPDATE+INSERT race window | 🔴 Alta | Validação 7 | INSERT ... ON CONFLICT em tabela `radar_baselines` |
| R1.1 | `triggerRadarScan` DOS-via-API | 🔴 Alta | Validação 8 | Auth admin + rate limit 1/min + cache 5min (FAIL CLOSED) |
| F8.1 | `LoadCriticas` Scan fail em `descricao`/`regra` NULL | 🔴 Alta | Validação 11 (auto) | sql.NullString para `regra`/`descricao` (mesmo padrão v1.4.0 #1) |

### ✅ Entregas por frente

#### 🔴 Frente 1 — Hardening P0

- **F3 — Radar race fix:** nova tabela `radar_baselines` com PK composta
  `(cadoc_code, alert_type)`. Migration 004 migra baselines antigas de
  `radar_alerts`. `RecordBaseline` usa `INSERT ... ON CONFLICT DO UPDATE`.
  50 goroutines concorrentes → 1 baseline (regressão coberta).
- **W1 — Worker retry/backoff:** migrations 005 adiciona `attempts` +
  `next_retry_at` + `processing_started_at` em `envios`. Backoff
  exponencial 1m/5m/30m/2h/12h, dead-letter após 5 tentativas.
- **W2 — Worker lease timeout:** sweeper a cada 1min resseta envios em
  `processing` há > 5min para `pending` (assume crash).
- **R1 — DOS-via-API prevention:**
  - `AdminAuth` FAIL CLOSED (sem `RADIANT_NORMA_ADMIN_TOKEN` env var → 401).
  - `ScanLimiter` (1 scan/min por IF) — header `Retry-After` em 429.
  - `ScanCache` (5min TTL) — reduz HTTP requests ao BACEN.
  - Audit emission: `radar.scan.triggered` vs `radar.scan.cached`.

#### 🟡 Frente 2 — Testes

- **F6:** 14 testes em `internal/schema/registry_test.go` —
  GetEffective (data exata/passada/futura/sem-data), Insert (UNIQUE
  constraint), List (ordenação DESC), end-to-end.
- **F7:** 6 testes em `internal/db/migrate_test.go` — applier, idempotência
  (rodar 2x), recreate from corrupted, fresh DB, race concurrent 2x,
  schema_migrations table creation.
- **F8:** 17 testes em `internal/api/server_e2e_test.go` — AuthMiddleware
  4 endpoints, /v1/validate (4 casos), /v1/sta/submit (2 casos),
  /v1/schemas, /v1/rules, /v1/schemas/{cadoc}, /v1/radar/alerts/{id},
  enabled filter.

#### 🟢 Frente 3 — Cross-Doc L3 (diferencial proprietário)

- **Novo package `internal/crossdoc/`** com interface `CrossDocRule`,
  Registry, Engine (orquestra paralelo).
- **3 regras iniciais** (`XD-001`, `XD-002`, `XD-003`):
  - `XD-001`: Total ops 3040 vs clients 4111 (tolerância 5%, severity A).
  - `XD-002`: Modalidade 0213 (cheque especial) flag no 4111.
  - `XD-003`: Subsegmento DRSAC ESG (S4/S5) compatível com score ≥0.7.
- **Endpoint `POST /v1/crossdoc/validate`** recebe
  `{cadocs: {3040: xml, 4111: xml, 2030: xml}}` e retorna
  ValidationResponse com passed/errors/warnings/rules_run/rules_skipped.
- **Audit:** `crossdoc.validated` com metadata `{cadocs, passed, errors,
  warnings, rules_run, rules_skip}`.

#### 🔵 Frente 4 — Postgres driver

- **`db.Open` detecta DSN**:
  - `postgres://` ou `postgresql://` → pgx/v5 (database/sql bridge).
  - `file:` ou path cru → SQLite (preserva comportamento v1.4.x).
- **Pool diferenciado**: SQLite 8/2 (writes serializados) vs Postgres 25/5.
- **`Backend(dsn)` helper** retorna `"sqlite"` ou `"postgres"` (logging).
- **`docker-compose.yml`** (raiz): Postgres:16-alpine + serviços opcionais
  api/worker via profile `prod`.
- **`docs/postgres-setup.md`** quickstart + limitações.

#### 🟢 W3 — B01-B05 → registry (refator arquitetural)

- Nova interface `RawRule` em `audit/rules/registry.go` (opera em XML
  bruto, não *Doc3040 tipado).
- `RawRuleFunc` adapter permite usar func como RawRule.
- `Registry` agora dual map (`rules` + `rawRules`).
- `audit/service.go::applyRegra` remove ~30 linhas de if B01-B05 inline.

#### 🟢 W4 — cadoc list dinâmico (DB + cache)

- `schema.Registry.ListCadocs()` faz `SELECT DISTINCT cadoc_code` UNION
  `schema_versions + criticas`.
- `CadocListCache` in-memory 5min (mesmo padrão do ScanCache do R1).
- `internal/api/server.go::cadocsWithCache` abstrai cache vs DB.
- `listSchemas` e `listRules` consultam ambos via cache.

### 📊 Estatísticas

```
Testes:        99 (v1.4.4) → 213 RUN / 164 únicos (v1.5.0)
                          (+65% únicos, +115% runs c/ subtests)
Coverage:      ~70% média → ~75% média (medida por package, ver SPRINT_6_RESULTS)
Packages:      5 c/ tests → 10 c/ tests    (de 12 totais)
LOC:           ~4.200 → ~6.500             (+55%)
Commits:       10 commits Sprint 6 (v1.4.3 e v1.4.4 são anteriores à tag v1.4.4)
Migrations:    3 (001-003) → 5 (001-005)
Regras audit:  25 tipadas → 25 tipadas + 5 raw (B01-B05)
Regras cross:  0 → 3 (XD-001/002/003)
DB drivers:    1 (SQLite) → 2 (+Postgres)
Endpoints:     13 → 14 (+/v1/crossdoc/validate)
Tables:        7 → 8 (+radar_baselines)
```

### 🏗️ Lições aprendidas (memory entries candidatam)

1. **DOS-via-API rate limiting é obrigatório desde o dia 1** —
   agora coberto com FAIL CLOSED, audit, e testes de regressão.
2. **textual vs datetime comparison em SQLite** —
   `time.Now()` via driver modernc formatado em RFC3339 vs
   `CURRENT_TIMESTAMP` do SQLite em formato com espaço → comparação
   `<=` falhava silenciosamente. Solução: `DATETIME(CURRENT_TIMESTAMP,
   '+N seconds')` no próprio SQL.
3. **Dual registry pra regras que operam em representações diferentes**
   (tipada vs raw) sem forçar refactor de N regras já implementadas.
4. **Tests E2E pegam bugs latentes** que unit tests não pegam —
   `LoadCriticas` faltava NullString em `regra` e `descricao`.

### ⚠️ Gaps remanescentes (Sprint 7 backlog)

| # | Gap | Por que | Sprint 7? |
|---|-----|---------|-----------|
| GAP-7.1 | Cross-doc L3 — `iterXMLElements` é implementação caseira (não usa encoding/xml robustamente) | Funciona para BACEN XMLs típicos, edge cases podem falhar | ✅ Sprint 7 |
| GAP-7.2 | Cross-doc L3 — regras baseadas em agregação de tags podem misinterpretar CDATA ou entities | Tolerável para XML plano | ✅ Sprint 7 (test fuzzer) |
| GAP-7.3 | Postgres integration tests sem testcontainers | Dependência de Postgres rodando quebraria CI | ✅ Sprint 7 |
| GAP-7.4 | Migration 004 (`INSERT OR IGNORE`) não roda em Postgres puro | Pequena diferença SQL | ✅ Sprint 7 (migração Postgres-flavor) |
| GAP-7.5 | User-Agent hardcoded em radar.go (não usa api.Version) | Gap arquitetural conhecido desde v1.4.3 F10.10 | ✅ Sprint 7 (refactor internal/version) |
| GAP-7.6 | Cross-doc engine limita goroutines | Pode DOS em request com 100 CADOCs | ✅ Sprint 7 (limit concurrency) |
| GAP-7.7 | cmd/api não lê DATABASE_URL automaticamente (precisa `-db` flag) | DX | ✅ Sprint 7 |
| GAP-7.8 | Backend Frontend Norma Console (Next.js) | Backend-only | ✅ Sprint 7+ |
| GAP-7.9 | Mais regras 3040 (320 total, ~25 implementadas) | Foco em hardening | ✅ Sprint 7 (sprint de regras) |

### 📂 Commits Sprint 6

1. `feat(v1.5.0)` F6 schema tests + version bump
2. `fix(v1.5.0)` F3 race fix recordBaseline
3. `feat(v1.5.0)` W1+W2 worker hardening
4. `fix(v1.5.0)` R1 DOS-via-API prevention
5. `refactor(v1.5.0)` W3 B01-B05 → registry
6. `feat(v1.5.0)` W4 cadoc list do DB
7. `test(v1.5.0)` F7 migrate tests
8. `test(v1.5.0)` F8 E2E coverage + bug LoadCriticas
9. `feat(v1.5.0)` Cross-Doc L3
10. `feat(v1.5.0)` Postgres driver

### 📂 Arquivos modificados/criados (Sprint 6)

**Código (10+ arquivos):**
- `backend/internal/db/migrations/004_radar_baselines.sql` (NOVO)
- `backend/internal/db/migrations/005_worker_hardening.sql` (NOVO)
- `backend/internal/worker/worker.go` (NOVO, ~250 LOC)
- `backend/internal/worker/worker_test.go` (NOVO, ~390 LOC)
- `backend/internal/radar/admin.go` (NOVO, ~130 LOC)
- `backend/internal/radar/admin_test.go` (NOVO, ~150 LOC)
- `backend/internal/audit/rules/basic_rules.go` (NOVO, ~80 LOC)
- `backend/internal/audit/rules/raw_rules_test.go` (NOVO, ~180 LOC)
- `backend/internal/crossdoc/crossdoc.go` (NOVO, ~150 LOC)
- `backend/internal/crossdoc/engine.go` (NOVO, ~120 LOC)
- `backend/internal/crossdoc/registry.go` (NOVO, ~50 LOC)
- `backend/internal/crossdoc/crossdoc_test.go` (NOVO, ~230 LOC)
- `backend/internal/crossdoc/rules/3040_4111.go` (NOVO, ~170 LOC)
- `backend/internal/crossdoc/rules/registry.go` (NOVO, ~30 LOC)
- `backend/internal/schema/registry_test.go` (NOVO/COMPLETO)
- `backend/internal/api/server_test.go` (atualizado)
- `backend/internal/api/server_e2e_test.go` (NOVO)
- `backend/internal/api/server_admin_test.go` (NOVO)
- `backend/internal/api/server.go` (modificado)
- `backend/internal/audit/service.go` (modificado — W3 + F8.1 fix)
- `backend/internal/audit/rules/registry.go` (modificado — W3)
- `backend/internal/audit/rules/3040_test.go` (modificado)
- `backend/internal/radar/radar.go` (modificado — F3 + version bump)
- `backend/internal/radar/radar_test.go` (modificado — F3 tests)
- `backend/internal/schema/registry.go` (modificado — W4)
- `backend/internal/db/db.go` (modificado — Postgres driver)
- `backend/internal/db/migrate_test.go` (NOVO)
- `backend/cmd/worker/main.go` (modificado — sweeper loop)

**Infra/Docs:**
- `docker-compose.yml` (NOVO)
- `docs/postgres-setup.md` (NOVO)
- `SPRINT_6.md` (atualizado — status Aprovada)
- `VALIDATION_v1.5.0.md` (NOVO — esta validação)
- `SPRINT_6_RESULTS.md` (NOVO — resultados finais)

---

## v1.4.4 — 2026-07-03 (Validação profunda 10: itoa removed + User-Agent bump + self-doc fix)
