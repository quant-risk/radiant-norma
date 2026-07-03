# Sprint 5 — Testes Unitários + Cross-Doc L3 + Hardening

> **Data proposta:** 2026-07-03+
> **Status:** Proposta (aguarda aprovação)
> **Tema:** Maturidade de engenharia (testes + hardening) + diferencial proprietário (L3 cross-doc)
> **Duração estimada:** 1-2 semanas

## 🎯 Por que Sprint 5 AGORA

**Sprint 4 fechou 5 sprints de backlog com 28 bugs em 5 validações manuais.**
O padrão está claro: **validação manual encontra 80% dos bugs em 4-5 passadas,
depois satura**. As 28 correções foram suficientes para deixar o backend
funcional, mas:

1. **Não tem testes automatizados.** Cada nova feature precisa de mais 4-5
   passadas manuais para validar.
2. **Race condition crítica** (v1.3.5) passou 4 validações porque testes eram
   sequenciais. **Stress test com goroutines precisa estar no CI.**
3. **Diferencial competitivo** (L3 cross-doc) ainda não foi implementado.
   É o que distingue Radiant Norma do BCValidador Java proprietário.

## 🏛️ Tema da Sprint

**3 frentes paralelas:**

### 🔴 Frente 1 — Testes unitários (P0 — bloqueia tudo)

Sem testes, **não dá pra confiar** no backend pra produção. Sprint 4 fechou
Sprint 0-3; Sprint 5 fecha "como validar sem 4 passadas manuais?".

**Escopo:**
- `auditlog/log_test.go` — race condition (50 goroutines concorrentes)
- `audit/rules/3040_test.go` — todas as 25 regras (canônicos + edge cases)
- `audit/service_test.go` — Validate, LoadCriticas, applyRegra, UnmarshalJSON
- `radar/radar_test.go` — scanSource, recordBaseline, ListAlerts, ResolveAlert
- `db/migrate_test.go` — idempotência, concurrent apply
- `schema/registry_test.go` — GetEffective, List, Insert

**Pattern:** SQLite in-memory (`:memory:`) por teste + helpers de setup
(`newTestDB(t *testing.T) *sql.DB`). Usa `modernc.org/sqlite` que já é pure-Go.

**Coverage target:** ≥60% no `audit/`, ≥50% no `auditlog/`, ≥40% no resto.

**Race detector:** `go test -race ./...` em CI.

### 🟡 Frente 2 — Hardening dos gaps identificados

Sprint 4 deixou 5 gaps em aberto (documentados em VALIDATION_v1.3.5.md):

| Gap | Origem | Solução |
|---|---|---|
| Worker retry sem backoff/limite | worker/main.go:162 | Adicionar `attempts` + `max_attempts=3` + exponential backoff |
| Worker lease timeout | worker/main.go:136-145 | Reset `processing` → `pending` após 5min stuck |
| Radar scanSource race | radar/radar.go:134-196 | Mover para `asynq` queue com serialização por source |
| B01-B05 hardcoded | service.go:328-354 | Mover para registry (consistência) |
| Server cadoc list hardcoded | server.go:94,130 | Carregar do DB |

**Critério:** cada gap vira teste de regressão (Frente 1) + fix.

### 🟢 Frente 3 — Cross-doc L3 (diferencial proprietário)

**Cross-doc L3** = validar 3040 ↔ 4111 ↔ DRSAC em paralelo.

**Exemplos concretos:**
- Total de operações no 3040 = total de clientes no 4111?
- Modalidades no 3040 (cheque especial 0213) batem com flag no 4111?
- DRSAC ESG (Sprint 2 catalogado) tem vínculo com classificação de risco no 3040?

**Escopo Sprint 5:**
- Interface `CrossDocRule` (similar à `Rule`)
- `internal/crossdoc/` package novo
- `rules/3040_4111.go` — 3-5 regras cross-doc iniciais
- `internal/crossdoc/engine.go` — orquestrador (carrega múltiplos docs, executa regras)
- Endpoint `POST /v1/crossdoc/validate` — recebe múltiplos CADOCs

**Diferencial vs BCValidador:** BCValidador valida UM CADOC por vez.
Radiant Norma valida o ecossistema inteiro (L3 é exclusivo).

## 📦 Entregas previstas

### Documentação
- `SPRINT_5.md` (este doc, atualizado durante sprint)
- `VALIDATION_v1.4.0.md` (validação profunda de v1.4.0)
- `CHANGELOG.md` (entrada v1.4.0)
- `docs/testing.md` (como rodar testes, helpers, padrões)

### Código

**Frente 1 (testes) — 8 arquivos:**
- `internal/auditlog/log_test.go` (350+ linhas, race test + chain test)
- `internal/audit/rules/3040_test.go` (800+ linhas, 25 regras × 5 edge cases)
- `internal/audit/service_test.go` (200+ linhas, Validate + applyRegra)
- `internal/radar/radar_test.go` (250+ linhas, scan + baseline + resolve)
- `internal/db/migrate_test.go` (100+ linhas, idempotente)
- `internal/schema/registry_test.go` (150+ linhas, GetEffective)
- `internal/api/server_test.go` (200+ linhas, httptest end-to-end)
- `internal/testutil/db.go` (helper, 50 linhas)

**Frente 2 (hardening) — 5 arquivos:**
- `internal/db/migrate.go` (add `attempts` column em envios)
- `cmd/worker/main.go` (backoff + lease recovery)
- `internal/radar/radar.go` (serializar scan por source)
- `internal/audit/rules/registry.go` (B01-B05 movidos pra cá)
- `internal/api/server.go` (carregar cadocs do DB)

**Frente 3 (cross-doc) — 4 arquivos:**
- `internal/crossdoc/engine.go` (orquestrador)
- `internal/crossdoc/rules/3040_4111.go` (regras iniciais)
- `internal/crossdoc/rules/registry.go` (registry cross-doc)
- `cmd/api/server.go` (endpoint `/v1/crossdoc/validate`)

### CI/CD
- `.github/workflows/test.yml` (go test -race -cover, golangci-lint)
- `Makefile` (alvo `test`, `test-race`, `test-cover`, `lint`)
- README atualizado com badges de CI

## 📊 Estatísticas previstas (final Sprint 5)

```
Backend Go:
  Linhas: 3.198 → ~5.500 (+72% via testes)
  Packages: 8 → 9 (+crossdoc)
  Coverage: 0% → ≥55%
  Race tests: 0 → 8+ (auditlog, worker, radar, migrate)

End-to-end:
  Endpoints: 13 → 14 (+/v1/crossdoc/validate)
  Regras 3040: 25 → 25 (mantém, foco em cobertura)
  Regras cross-doc: 0 → 3-5 iniciais

Documentação:
  Markdowns principais: 4 → 5 (+testing.md)
  Validações: 4 → 5 (v1.4.0 doc)
```

## 🚧 Gaps remanescentes (vão pra Sprint 6)

| Gap | Por quê | Sprint 6 |
|---|---|---|
| Driver Postgres real (pgx) | Sem Docker local | Sprint 6 com Docker compose |
| JWT/OAuth em vez de X-IF-ID | Mantido header simples | Sprint 6 com auth completo |
| STA Web Services real | Stub OK pra demo | Sprint 6 com Web/WS |
| Mais regras 3040 (295 faltam) | 25/320, foco em hardening | Sprint 6 ou 7 |
| Frontend Norma Console | Backend-only | Sprint 6 ou 7 (Next.js) |

## 🎯 Critérios de aceite Sprint 5

### Must-have (P0)
- [ ] `go test ./...` passa 100%
- [ ] `go test -race ./...` passa 100% (detecta races)
- [ ] `go test -cover ./...` ≥55%
- [ ] `golangci-lint run` clean
- [ ] CI workflow roda em PR

### Should-have (P1)
- [ ] 5 gaps de hardening corrigidos + testes
- [ ] Frontend L3 cross-doc (3+ regras, 1 endpoint)
- [ ] Documentação atualizada

### Nice-to-have (P2)
- [ ] Benchmarks (paralelizar vs serial no auditlog)
- [ ] Mutation testing (go-mutesting)
- [ ] Property-based testing (gopter)

## 🛠️ Stack & ferramentas

**Mantém:**
- Go 1.24+, chi router, SQLite (modernc.org/sqlite pure-Go)
- stdlib `testing` (sem framework externo — Go idiom)

**Novas:**
- `github.com/stretchr/testify` (assertions mais legíveis — opcional)
- `golangci-lint` (CI lint)
- `github.com/golang-migrate/migrate` (não, manter embed.FS atual)

**Para cross-doc:**
- Carregar múltiplos CADOCs em paralelo (goroutines + errgroup)
- Compartilhar Doc3040 + Doc4111 entre regras via context

## 🚀 Como começar (handoff)

1. **Setup de testutil** — helper `newTestDB(t *testing.T)` que retorna SQLite
   in-memory já migrado
2. **Teste de regressão** — começar pelo `auditlog/log_test.go` (race é P0)
3. **Bateria de regras** — `audit/rules/3040_test.go` com tabelas de casos
4. **CI antes de feature** — garantir que `go test -race` rode em PR antes
   de implementar cross-doc
5. **Cross-doc por último** — após testes garantirem que L1+L2 não quebram

## 📚 Referências

- Go testing: https://pkg.go.dev/testing
- SQLite in-memory: `file::memory:?cache=shared` (driver modernc)
- Race detector: https://go.dev/blog/race-detector
- testify: https://github.com/stretchr/testify (assert/require/mock)
- golangci-lint: https://golangci-lint.run/

---

**Decisão:** Aguardando aprovação do Henrique para iniciar Sprint 5.
Estimativa: 8-12h de trabalho focado (em paralelo com outras tasks Fortvna).

**Priorização recomendada:**
1. 🔴 Testes P0 (4h) — bloqueia confiança em qualquer feature
2. 🟡 Hardening (2h) — fecha gaps do Sprint 4
3. 🟢 Cross-doc L3 (4-6h) — diferencial competitivo

**Trade-off:** Se tempo curto, focar em P0+P1 (8h), cross-doc vira Sprint 6.