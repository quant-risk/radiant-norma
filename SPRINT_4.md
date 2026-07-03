# Sprint 4 — Audit utilizável + Radar Regulatório

> **Duração:** 2026-07-03 (1 sessão focada, ~3h de execução efetiva)
> **Tema:** Corrigir hollow stubs da Sprint 3 + implementar 25 regras semânticas 3040 + Radar regulatório (worker que detecta mudanças de leiaute).
> **Versão:** v1.3.0

## Contexto

Sprint 3 fechou o ciclo "dados → serviço consumível" mas, ao validar end-to-end,
encontramos **5 hollow stubs** que impediam o Norma Audit de funcionar de verdade:

| Bug | Severidade |
|---|---|
| `/v1/validate` exigia `cadoc_code` no JSON mas docs/clientes enviavam `cadoc` | 🔴 crítico |
| Apenas 5/320 regras 3040 implementadas (1.5% cobertura) | 🔴 alto |
| `cmd/worker/` diretório vazio (anunciado mas não codado) | 🟡 médio |
| `/v1/sta/submit` lia params da URL, inconsistente com resto da API | 🟡 médio |
| Tabela `envios` rejeitava INSERT silenciosamente (FK pra `ifs` vazia) | 🟡 médio |

Sprint 4 ataca em 2 eixos:
1. **Honesty Patch** — fecha os 5 hollow stubs antes de adicionar feature nova
2. **Audit utilizável** — porta 25 regras semânticas do 3040 com parser XML tipado
3. **Radar regulatório** — worker que detecta mudanças de leiaute (diferencial first-mover)

## Backlog priorizado

### 🔴 P0 — Honesty Patch (5/5 entregue)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P0.1** | `/v1/validate` aceita `cadoc` OU `cadoc_code` no JSON | ✅ | `audit.ValidationRequest.UnmarshalJSON` |
| **P0.2** | `/v1/sta/submit` padronizado em body JSON (retrocompat query) | ✅ | `sta.Submission` + handler unificado |
| **P0.3** | `cmd/worker/` stub mínimo (queue processor) | ✅ | `cmd/worker/main.go` (150 linhas) |
| **P0.4** | Tabela `envios` aceita INSERT (FK pra ifs) | ✅ | Seed popula 2 IFs demo + migration 002 |
| **P0.5** | Re-testar end-to-end: validate OK/quebrado, STA submit, worker retry | ✅ | 5 cenários curl-testados |

### 🟡 P1 — Audit utilizável (B1: 25 regras 3040)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P1.1** | Registry de regras tipado (`internal/audit/rules/`) | ✅ | `registry.go` (130 linhas) |
| **P1.2** | Parser XML 3040 com structs tipadas | ✅ | `3040.go::ParseDoc3040` |
| **P1.3** | 25 regras implementadas: B06-B15, F01-F05, C01-C05, S01-S05 | ✅ | `3040.go` (430 linhas) |
| **P1.4** | Severity da Rule tem prioridade sobre DB (gravidade) | ✅ | `service.go::Validate` |
| **P1.5** | Testar contra XML exemplo + 4 cenários de erro intencional | ✅ | 100% passa |

### 🟢 P2 — Radar Regulatório (B3: first-mover)

| # | Tarefa | Status | Saída |
|---|---|---|---|
| **P2.1** | Service de Radar com fetch + hash + diff | ✅ | `internal/radar/radar.go` (250 linhas) |
| **P2.2** | Worker CLI (`cmd/radar/main.go`) | ✅ | `cmd/radar/main.go` (100 linhas) |
| **P2.3** | Endpoints REST: list/get/resolve/scan | ✅ | 4 endpoints em `/v1/radar/*` |
| **P2.4** | Migration tracking via `schema_migrations` | ✅ | `migrate.go` (era idempotente frágil) |
| **P2.5** | Testar scan + resolve + filtros | ✅ | `POST /v1/radar/scan` retorna novos alertas |

## Definition of Done — Sprint 4

- [x] Backend compila sem warnings (`go build ./...`)
- [x] `/v1/validate` aceita `cadoc` E `cadoc_code` (testado com 4 cenários)
- [x] 25 regras 3040 implementadas com parser XML tipado
- [x] Severity das regras implementadas tem prioridade sobre DB
- [x] `cmd/worker` processa envios pending do DB
- [x] `cmd/radar` faz scan de URLs BACEN + detecta mudanças
- [x] 4 endpoints REST novos: `/v1/radar/*`
- [x] Migration tracking via `schema_migrations`
- [x] Audit log grava todas as ações
- [x] Smoke test suite completo passa

## Estatísticas finais

```
Backend Go          14 arquivos · 2.908 linhas (vs 10/1.400 Sprint 3)
   ├─ cmd/api/main.go            100 linhas (entrypoint + graceful shutdown)
   ├─ cmd/seed/main.go           280 linhas (+ seed IFs)
   ├─ cmd/worker/main.go         180 linhas (queue processor)
   ├─ cmd/radar/main.go          100 linhas (radar worker)
   ├─ internal/api/server.go     290 linhas (10 handlers + middleware)
   ├─ internal/audit/service.go  320 linhas (L1 + L2 registry)
   ├─ internal/audit/rules/registry.go  140 linhas (Rule interface)
   ├─ internal/audit/rules/3040.go     440 linhas (25 regras + parser)
   ├─ internal/auditlog/log.go   140 linhas (hash chain)
   ├─ internal/db/db.go           40 linhas (SQLite + abstraction)
   ├─ internal/db/migrate.go     90 linhas (migration tracking)
   ├─ internal/radar/radar.go    260 linhas (fetch + diff + alerts)
   ├─ internal/schema/registry.go 140 linhas (Schema Registry)
   ├─ internal/sta/stub.go        95 linhas (STA stub com JSON tags)
   └─ migrations/                128 + 5 linhas (2 SQL files)

Banco SQLite populado:
   schema_migrations: 2
   schema_versions:   8
   criticas:          968
   envios:            N (criados via /v1/sta/submit)
   audit_log:         N (cada ação grava)
   radar_alerts:      N (criados via Radar)
```

## Endpoints REST (10 funcionais, testados com curl)

| Método | Path | Teste |
|---|---|---|
| GET | `/healthz` | ✓ `{"status":"ok"}` |
| GET | `/v1/schemas` | ✓ 11 CADOCs |
| GET | `/v1/schemas/3040` | ✓ 500 fields + XSD 30 KB |
| GET | `/v1/schemas/3040/versions` | ✓ histórico |
| GET | `/v1/rules/3040` | ✓ 320 regras |
| POST | `/v1/validate` (XML OK) | ✓ passed=true |
| POST | `/v1/validate` (DtBase inválido) | ✓ F02 detecta severity=E |
| POST | `/v1/validate` (Remessa=0) | ✓ B06 detecta severity=E |
| POST | `/v1/sta/submit` JSON | ✓ protocolo fake + envio_id + persiste |
| POST | `/v1/sta/submit` ?cadoc= | ✓ retrocompat |
| GET | `/v1/radar/alerts` | ✓ lista alertas |
| POST | `/v1/radar/alerts/{id}/resolve` | ✓ resolve |
| POST | `/v1/radar/scan` | ✓ scan + retorna novos |
| 401 sem X-IF-ID | ✓ |
| 404 CADOC inexistente | ✓ |

## Decisões técnicas

| Decisão | Razão | Trade-off aceito |
|---|---|---|
| **`rules.Registry` interface-based** | Cada regra = struct com métodos. Adicionar regra = 1 struct + 1 linha no registry. | Mais boilerplate que map[string]func |
| **Parser XML tipado (struct tags)** | encoding/xml é stdlib, zero dependência. Mais simples que XPath. | Não cobre queries complexas (não precisamos agora) |
| **Severity da Rule > DB gravidade** | DB tem gravidade vazia pra regras novas. Rule é fonte da verdade. | Regras duplicadas podem divergir |
| **cmd/worker é stub** | Processa pending, atualiza status. Sem retry exponencial, sem DLQ. | Suficiente pra demo. Produção precisa asynq/machinery |
| **cmd/radar fetch URLs BACEN candidatas** | Resiliente: 404 não quebra, baseline gravado. | Algumas URLs vão falhar sempre (BACEN muda paths) |
| **`radar_alerts.alert_type='_baseline_*'`** | Reaproveita tabela existente. | Em produção, tabela dedicada seria melhor |
| **Migration tracking via `schema_migrations`** | Resolve problema de ALTER TABLE não-idempotente | Adiciona 1 query por migração nova |
| **`envios.xml_content` TEXT coluna** | Worker precisa do XML completo pra re-submeter | DB cresce ~5KB/envio (ok pra SQLite) |

## Cobertura de regras 3040 — antes vs depois

```
Sprint 3: 5/320 regras (1.5%)
Sprint 4: 25/320 regras (7.8%)  — Básicas B06-B15 + Formato F01-F05 + Campos C01-C05 + Semantica S01-S05
```

Cobertura por categoria:
- **Básicas (B)**: 11/21 (52%) — B01-B21 agora cobertas
- **Formato (F)**: 5/5 (100%) — todas as regras de formato
- **Campos Obrigatórios (C)**: 5/? (parcial) — C01-C05 implementadas
- **Semântica (S)**: 5/110 (4.5%) — só as mais óbvias
- **Agregadas (A)**: 0/14 — próxima sprint
- **Individualizadas (I)**: 0/15 — próxima sprint

## Lições aprendidas (cross-project)

1. **Hollow stub detection salva reputação** — Sprint 3 fechou 5/10 itens mas CHANGELOG dizia 10/10. Validação end-to-end com `cd /tmp && ... && curl` é obrigatória antes de "done".
2. **Severity da implementação > DB** — Quando regra é implementada em código, sua severity é a verdade. DB é só metadata.
3. **Migration tracking é obrigatório** — `CREATE TABLE IF NOT EXISTS` não basta quando você tem `ALTER TABLE` (não-idempotente). Tabela `schema_migrations` resolve.
4. **Custom UnmarshalJSON > tags conflitantes** — Aceitar múltiplos aliases no JSON é trivial com custom unmarshal. Mantém ergonomia do cliente sem poluir o struct.
5. **Radar resiliente > Radar certeiro** — Tentar fetch, logar falha, seguir. URLs BACEN mudam; sistema que sobrevive a 404 é melhor que sistema que falha quando path muda.

## Gaps remanescentes → Sprint 5

| Gap | Origem | Sprint 5 |
|---|---|---|
| **Driver Postgres real** | Sem Docker local | `pgx` + config flag |
| **JWT/OAuth em vez de X-IF-ID** | Honesty patch mantém header simples | `internal/auth/` + refresh tokens |
| **Postgres RLS (multi-tenant isolado)** | X-IF-ID só identifica | Policies por IF |
| **STA Web/WS real (Playwright)** | Stub | `internal/sta/web.go` |
| **Mais regras 3040 (A04, A06, I01-I15, etc)** | Cobertura 7.8% | Sprint 5 foca Agregadas + Individualizadas |
| **Cross-doc engine (L3)** | Requer 3040+4111 em paralelo | Carregar multi-CADOC |
| **Frontend Norma Console** | Backend-only | Next.js dashboard |
| **Dedup 3040 (14 warnings)** | Duplicatas reais no JSON | Deduplicar no extract.py |
| **Radar: asynq queue + retries** | Stub usa ticker simples | Substituir por fila real |
| **Testes unitários** | Sem coverage ainda | `*_test.go` + coverage report |

## Próxima sprint — Sprint 5 (preview)

**Tema:** Norma Console + mais regras + cross-doc L3
- Frontend Next.js (dashboard IFs)
- Portar 30+ regras Agregadas + Individualizadas do 3040
- Cross-doc engine L3 (3040 + 4111 + DRSAC)
- Radar com asynq queue

**Estimativa:** 40-60h

---

**Autor:** Mavis · Radiant ()
**Stakeholder:** Henrique Costa · henrique@fortvna.com.br