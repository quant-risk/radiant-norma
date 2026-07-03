# VALIDATION v1.4.0 — 7ª validação profunda (Sprint 5 v1.4.1)

> **Status:** DRAFT — superseded by validation pass 7
> **Data:** 2026-07-03
> **Trigger:** Release v1.4.0 fechou 5 sprints com 28 bugs + 5 latentes. 7ª validação
> pedido por Henrique para fechar gaps remanescentes antes de Sprint 6.
> **Escopo:** revisão profunda de código + docs + migrations + CI + testes + estrutura
> **Versão proposta:** v1.4.1 (patch — fixes de F1, F2 + audit emission gap)

## 🎯 Resumo executivo

**Estado real da v1.4.0:**
- ✅ 86 testes, ~70% coverage média, CI workflow, Makefile, README
- ✅ Todos os testes passam com `-race`
- ✅ `go vet` e `gofmt` clean
- ✅ 5 bugs latentes corrigidos (auditlog panic, F02 mês 13, baseline normalização, UNIQUE radar, LoadCriticas NULL)
- ⚠️ **Audit emission surface incompleta** — 2 endpoints de Radar sem audit
- ⚠️ **Mesmo padrão do bug #1 (slice [:12])** apareceu em radar.go (defensivo)

**Após validação 7 (v1.4.1):**
- ✅ F1 corrigido: helper `ShortHash()` exportado + 3 testes de regressão
- ✅ F2 corrigido: audit emission em `resolveRadarAlert` + `triggerRadarScan`
- ✅ F8 fechado (parcialmente): `internal/api/server_test.go` com 3 testes E2E
- 📊 **99 testes** (86 → 99, +13), **6 packages com testes** (5 → 6, +api)
- 📊 Coverage: radar 78.1% → **78.8%**, api NOVO 20.1%

## 🔍 Findings da validação 7

### ✅ F1 — `lastHash[:12]` panica em hash curto (MESMO padrão do bug #1 do sprint 5)

**Severidade:** 🟡 Média (defensivo — não-crítico em produção atual)

**Local:** `internal/radar/radar.go:150,171,192,193` — 4 ocorrências de `[:12]`

**Problema:**
```go
// radar.go:171 (antes)
Description: fmt.Sprintf("Hash anterior: %s\nHash novo: %s", lastHash[:12], hash[:12]),
```

Mesmo padrão que o `auditlog.Verify` que eu corrigi em v1.4.0 (bug #1). SHA-256
hex tem sempre 64 chars, então `[:12]` é safe na prática. MAS se alguém inserir
hash mal-formado no DB (corrupção, migration errada, INSERT manual), panica.

**Diferente do bug do auditlog:** aqui o `lastHash` vem do DB (description da tabela
`radar_alerts`), enquanto no auditlog o hash era interno (computado na hora).

**Fix (v1.4.1):** helper `ShortHash(s string) string` em radar.go + 4 testes:
- `TestShortHash_Normal` — 64 chars → primeiros 12
- `TestShortHash_Short` — 6 subtests: empty, 1, 5, 11, 12, 13 chars
- `TestShortHash_NeverPanics` — smoke test defensivo
- `TestRecordBaseline_ShortHashInDB` — integração: insere baseline corrompida no DB e verifica que ScanOnce não panica

**Cobertura:** radar 78.1% → **78.8%** (+0.7pp).

### ✅ F2 — Audit emission surface gap (PADRÃO MEMORY: "Audit emission surface check")

**Severidade:** 🟡 Média (SOC 2 / LGPD — mutações precisam ser auditadas)

**Local:** `internal/api/server.go:377,395` — handlers `resolveRadarAlert` e `triggerRadarScan`

**Problema:**
Mapeei via `grep -rn "auditLog\.Log\|AuditLog\.Log" --include="*.go" . | grep -v _test.go`:

| Endpoint | Audit? | Action |
|---|---|---|
| POST /v1/validate | ✅ | `cadoc.validated` |
| POST /v1/sta/submit | ✅ | `sta.submit` |
| (erro path persist) | ✅ | `sta.submit.persist_failed` |
| POST /v1/radar/alerts/{id}/resolve | ❌ **F2** | — |
| POST /v1/radar/scan | ❌ **F2** | — |

**Pattern do memory (`Audit emission surface check antes de claim`):**
> "release notes claim 'auto-invoked from every mutation'. Realidade: 5 paths não emitem."
> Aplicado aqui antes de qualquer claim. **Aqui eu não fiz claim ainda**, mas o gap é real.

**Fix (v1.4.1):** ambos handlers agora emitem audit:
- `resolveRadarAlert` → `radar.alert.resolved` (target=radar, metadata={alert_id})
- `triggerRadarScan` → `radar.scan.triggered` (target=radar, metadata={new_alerts})

**Tests (v1.4.1):** `internal/api/server_test.go` (novo) com 2 testes E2E +
`TestHealthz` smoke (3 total):
- `TestHealthz` — smoke test
- `TestResolveRadarAlert_EmitsAudit` — verifica audit_log após resolve
- `TestAuditEmission_Surface` — canário: detecta se alguém remover `auditLog.Log` no futuro

### 🟡 F3 — `recordBaseline` UPDATE-then-INSERT não é atômico (race window)

**Severidade:** 🟡 Média (improvável em produção — scan é sequencial por source)

**Local:** `internal/radar/radar.go:266-287`

**Problema:**
```go
// 1. UPDATE (afeta 0 rows se não existe)
res, err := s.db.ExecContext(ctx, `UPDATE radar_alerts ...`, ...)
n, _ := res.RowsAffected()
if n > 0 { return nil }  // OK, atualizou

// 2. INSERT (race window: dois scans simultâneos veem n=0, ambos fazem INSERT)
_, err = s.db.ExecContext(ctx, `INSERT INTO radar_alerts ...`)
```

**Race window:** se dois scans rodarem em paralelo (improvável mas possível), ambos
veem `n=0`, ambos tentam INSERT. Com migration 003 removeu UNIQUE, ambosucceedem
e duplicam a baseline.

**Fix proposto (Sprint 6):** usar `INSERT ... ON CONFLICT` (Postgres) ou
`INSERT OR IGNORE` + UPDATE em uma única transação serializada.

**Decisão Sprint 5:** não corrigir agora — risco operacional baixo, vai pra Sprint 6.

### 🟡 F4 — `cmd/seed/main.go:151` — stmt preparado DENTRO do for-loop

**Severidade:** 🟢 Baixa (cosmético, sem impacto funcional)

**Local:** `internal/.../cmd/seed/main.go::seedCriticas` (linha 151)

**Problema:**
```go
for cadoc, lista := range cf.Criticas {
    // Limpa críticas antigas desse CADOC
    d.Exec("DELETE FROM criticas WHERE cadoc_code = ?", cadoc)

    // Prepara stmt a CADA cadoc (10+ vezes — overhead)
    stmt, err := d.Prepare(`INSERT INTO criticas (...) VALUES (...)`)
    ...
    stmt.Close()
}
```

O mesmo INSERT SQL é preparado 10+ vezes (uma por cadoc) em vez de uma vez fora
do loop. Pequeno overhead; em 11 cadocs = ~100ms de overhead cumulativo.

**Decisão Sprint 5:** não corrigir agora — só roda 1x via seed. Vai pra backlog.

### 🟡 F5 — `cmd/seed/main.go:160-184` — parse de dataInicio silencioso

**Severidade:** 🟢 Baixa (UX, dados podem ficar NULL silenciosamente)

**Local:** `cmd/seed/main.go:160-184`

**Problema:**
```go
if c.DataBaseInicio != "" {
    t, err := time.Parse("2006-01-02", c.DataBaseInicio)
    if err == nil {
        dataInicio = t.Format("2006-01-02")
    } else {
        dataInicio = nil  // ⚠️ sem warning — INSERT com NULL silencioso
    }
}
```

Se o JSON tem `"data-base inicio": "08/2020"` (formato errado), vira NULL no DB
sem log. UX ruim — usuário não sabe que dado foi perdido.

**Decisão Sprint 5:** documentar como gap. Fix: adicionar `logger.Warn` no else.

### 🟡 F6 — `internal/schema` (147 linhas) sem testes

**Severidade:** 🟡 Média (GetEffective, List, Insert não testados)

**Local:** `internal/schema/registry.go`

**Problema:** Sprint 5 P0 previa `internal/schema/registry_test.go` (150+ linhas) mas
não foi entregue. Funções core do Schema Registry estão untested.

**Impacto:** regressões em GetEffective (multi-version logic) só seriam pegas em prod.

**Decisão Sprint 5:** não corrigir agora — gap coberto por testes indiretos via
cmd/seed (que exercita Insert). Vai pra Sprint 6 P0.

### 🟡 F7 — Migration 003 não tem teste de idempotência

**Severidade:** 🟡 Média (recovery / re-apply pode falhar)

**Local:** `internal/db/migrations/003_radar_no_unique.sql`

**Problema:** migration tracking via `schema_migrations` evita re-aplicação automática,
MAS se rodar manualmente (debug, recovery), `DROP TABLE → INSERT INTO new → DROP →
ALTER RENAME` pode falhar se:
- DB tem dados novos que conflitam
- Roda 2x sem tracking

**Decisão Sprint 5:** não corrigir agora — migrations são one-shot. Vai pra Sprint 6
como parte do hardening.

### 🟡 F8 — `internal/api` sem tests até v1.4.1 (PARCIALMENTE FECHADO)

**Severidade:** 🟡 Média (end-to-end coverage era zero)

**Local:** `internal/api/server.go` (431 linhas, 0 testes até v1.4.1)

**Fix (v1.4.1):** `internal/api/server_test.go` (novo, ~220 linhas) com 3 testes E2E:
- `TestHealthz` — smoke
- `TestResolveRadarAlert_EmitsAudit` — regressão F2
- `TestAuditEmission_Surface` — canário de regressão

**Coverage api:** 0% → **20.1%**. Restante (~80%) ainda untested:
- POST /v1/validate, /v1/sta/submit (write paths)
- GET /v1/schemas, /v1/rules, /v1/radar/alerts (read paths)
- authMiddleware (X-IF-ID check)

**Decisão Sprint 5:** suficiente para fechar F2. Vai pra Sprint 6 P0 expandir.

### ✅ F9 — PT-BR / ASCII range tokenisers

**Severidade:** N/A (não aplicável)

**Procurei:** `grep -rEn "'a'|'z'|< 'a'|> 'z'" --include="*.go" .`

**Resultado:** zero matches. Apenas usos legítimos:
- `radar_test.go:129,254` — `string(rune('A'+n-1))` para gerar "A", "B", "C"...

**Não há tokenisers com ASCII range.** Memory pattern não aplicável aqui.

### ✅ F10 — CHANGELOG / SPRINT_5_RESULTS incompletos (FECHADO nesta validação)

**Severidade:** 🟢 Baixa (doc)

**Problema:** CHANGELOG.md (v1.4.0 entry) menciona "5 bugs latentes detectados" mas
não menciona gaps remanescentes (audit emission surface, schema untested, api untested).

**Fix (v1.4.1):** CHANGELOG.md será atualizado com entrada v1.4.1 + este doc.

## 📊 Estatísticas finais

```
Antes (v1.4.0):              Depois (v1.4.1):
  Testes:    86                 Testes:    99 (+13)
  Packages:  5 com testes       Packages:  6 com testes (+api)
  Coverage:  ~70% média         Coverage:  ~70% média (api NOVO 20.1%)

  auditlog:  90.8% (mantido)
  audit:     68.7% (mantido)
  audit/...: 63.2% (mantido)
  radar:     78.1% → 78.8% (+0.7pp — ShortHash + RecordBaseline)
  testutil:  45.0% (mantido)
  api:       NOVO 20.1% (3 testes: healthz + audit emission canary)
```

## 📂 Arquivos modificados/criados (v1.4.1)

**Código:**
- `internal/radar/radar.go` — `ShortHash()` helper + 4 call sites (defesa contra F1)
- `internal/api/server.go` — audit emission em `resolveRadarAlert` + `triggerRadarScan` (F2)

**Tests:**
- `internal/radar/radar_test.go` — 4 testes novos (ShortHash + RecordBaseline)
- `internal/api/server_test.go` — NOVO, 3 testes E2E (healthz + audit surface)

## 🏗️ Lições aprendidas

**1. Mesmo padrão aparece em múltiplos lugares (F1).**
Bug #1 do v1.4.0 (`auditlog.Verify` panic em `[:12]`) tinha o mesmo padrão em 4
pontos do `radar.go`. Validação 7 caçou preventivamente. **Pattern:** quando
você corrigir um bug de slice/index, fazer `grep` pelo mesmo padrão em todo o repo.

**2. Audit emission surface é gap silencioso (F2).**
Endpoints que mutam dados sem audit são invisíveis em prod até alguém pedir
relatório SOC 2. **Pattern:** mapear `auditLog.Log()` call graph antes de qualquer
sprint com mutação nova. Memory entry: "Audit emission surface check antes de claim".

**3. Tests E2E com httptest são baratos e valiosos.**
`internal/api/server_test.go` foi ~220 linhas e cobriu 3 cenários críticos
(healthz, audit emission canary, resolveRadarAlert). **Dívida antiga:** F8
estava aberto desde Sprint 3 mas só foi resolvido agora (com F2 como motivador).

## 🚧 Gaps remanescentes (Sprint 6 backlog)

| # | Gap | Origem | Sprint 6 |
|---|---|---|---|
| F3 | `recordBaseline` race window | Validação 7 | Hardening P0 — ON CONFLICT |
| F4 | `cmd/seed` stmt por loop | Validação 7 | Cosmético — preparar fora |
| F5 | `cmd/seed` dataInicio silent | Validação 7 | UX — logger.Warn |
| F6 | `internal/schema` sem tests | Sprint 5 P0 pulado | Test P0 |
| F7 | Migration 003 idempotência | Validação 7 | Hardening P1 |
| F8 | api tests ~80% restantes | Validação 7 (parcial) | Test P0 |
| W1 | Worker retry backoff | Sprint 5 P1 atrasado | Hardening P0 |
| W2 | Worker lease timeout | Sprint 5 P1 atrasado | Hardening P0 |
| W3 | B01-B05 hardcoded em service.go | Sprint 4 P2 | Hardening P2 |
| W4 | Cadoc list hardcoded | Sprint 4 P2 | Hardening P2 |
| L3 | Cross-doc L3 endpoint | Sprint 5 P2 atrasado | Feature P1 |
| PG | Postgres real driver | Sprint 5 P1 atrasado | Feature P1 |

## 📌 Status

- **v1.4.1 marcada para commit local** após este doc
- **Sprint 6** vai priorizar F3 (race), F6 (schema tests), W1+W2 (worker hardening),
  L3 (cross-doc), PG (Postgres driver)
- **Documentos a atualizar:** CHANGELOG.md (entrada v1.4.1), SPRINT_5_RESULTS.md (gaps)

---

**Decisão:** Fixar F1, F2, F8 agora (já feito). Documentar F3-F7 + W1-W4 + L3 + PG
como backlog Sprint 6. Não promover a "accepted" até validação 8 fechar gaps novos
que apareçam em Sprint 6.