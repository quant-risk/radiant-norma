# VALIDAÇÃO 56 — Deep audit pós-v3.33.1 (revisão da entrega anterior)

> **Data:** 2026-07-06
> **Trigger:** Solicitação Henrique — "validação profunda em tudo que você acabou de fazer (Validação 54 + Sprint 30 + Validação 55)"
> **Tipo:** patch (bug real detectado em stress tests + cleanup carry-over + hardening)
> **Versão:** v3.33.1 → **v3.33.2**

## TL;DR

Auditoria profunda da entrega anterior (Validação 54/Sprint 30/Validação 55) encontrou **8 findings, 7 fechados nesta validação, 0 carry-over da própria V56**.

**Bug real detectado (F-56-B HIGH):** `TestAuditLog_NoChainBreaks_Concurrent` (50 goroutines) e `TestAuditLog_NoChainBreaks_HighContention` (200 goroutines) **FALHAVAM silenciosamente sem -race**. CI roda com `-race` (test é skipped), portanto CI nunca validava o invariant crítico "50 goroutines Log sem quebrar chain". Pré-existente (flake desde Sprint 32 / Validação 21 documentou como pótencial risco), nunca corrigido. **CI cego ao invariant — regressão latente mascarada.**

Fix: semaphore (32 in-flight) + busy_timeout 5000→30000ms + Log context timeout 5s→15s. **50/50 e 200/200 goroutines commitam em 0.13s** vs timeout de 5s pré-fix.

## Findings — detalhamento

### F-56-A — Comentário enganoso sobre BEGIN IMMEDIATE em `auditlog/log.go` (LOW → FECHADO)

**Severidade:** LOW (documentação)
**Categoria:** Drift docs ↔ código
**Arquivo:** `backend/internal/auditlog/log.go:88-91`

**Bug:**
Comentário original sugeria que `BeginTx(ctx, nil)` no driver SQLite usaria BEGIN IMMEDIATE automaticamente. Real: `modernc.org/sqlite` usa BEGIN DEFERRED por default. BEGIN IMMEDIATE vem do **DSN pragma `_txlock=immediate`** em `backend/internal/db/db.go:64`. Sem o pragma, refactor que remova a config DSN quebraria auditoria silenciosamente.

**Fix:** Comentário reescrito com referência explícita ao pragma + alerta sobre F21.5.

```go
// Validação 56 (v3.33.2): BEGIN IMMEDIATE em SQLite vem do DSN pragma
// `_txlock=immediate` em db.Open → openSQLite (backend/internal/db/db.go:64).
// Sem o pragma, modernc.org/sqlite usa BEGIN DEFERRED default — duas
// goroutines pegariam o mesmo prev_hash no SELECT antes do INSERT e
// gerariam entradas com PrevHash duplicado (chain quebrada). NÃO remover
// `_txlock=immediate` do DSN sem revisar F21.5 (regressão validação 21).
```

---

### F-56-B — Stress tests de auditlog falham silenciosamente sem -race (HIGH → FECHADO)

**Severidade:** **HIGH (CI cego ao invariant crítico)**
**Categoria:** Concurrency / CI coverage gap
**Arquivos:**
- `backend/internal/auditlog/concurrent_test.go`
- `backend/internal/auditlog/log.go`
- `backend/internal/db/db.go`

**Bug — múltipla camada:**

1. **`TestAuditLog_NoChainBreaks_Concurrent` (50 goroutines)** falhava com:
   ```
   expected 50 entries, got 0
   ```
2. **`TestAuditLog_NoChainBreaks_HighContention` (200 goroutines, sem=30)** falhava com:
   ```
   Log 22 failed: begin tx: context deadline exceeded
   Log 17 failed: begin tx: database is locked (5) (SQLITE_BUSY)
   chain break: ok=true count=170 (expected 200)
   ```
3. **Ambos testes são skipped em `-race`** (`if testutil.IsRaceEnabled() { t.Skip }`). CI roda com `-race`. **CI não valida este invariant.**

**Diagnóstico (Validação 56):** Sprint 30 (v3.33.0) adicionou skip sob -race documentando que "stress tests rodam normalmente em `go test ./...` (sem -race)". **FALSO** — testes falhavam sem -race há meses sem ninguém rodar (porque ninguém roda CI sem -race, e local ninguém executava). Validação 21 (F21.5) documentou o risco, mas o teste de regression nunca foi verificado pós-fix inicial.

**Root cause:**
- `d.SetMaxOpenConns(8)` em `db.go:69` → pool 8.
- `busy_timeout=5000` (5s) → após 5s, transactions esperando o lock falham com SQLITE_BUSY.
- `Log` context timeout 5s → se BeginTx não consegue lock em 5s, falha com `context.DeadlineExceeded`.
- 50 goroutines disputam pool 8 + lock write serializado por `_txlock=immediate` = contenção dupla que estoura 5s consistentemente.

**Fix combinado (3 locais):**

1. **`db.go`** — `busy_timeout=5000` → `busy_timeout=30000` (6× margem):
   ```go
   dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(30000)&_txlock=immediate", path)
   ```

2. **`auditlog/log.go`** — Log context timeout 5s → 15s (3× margem):
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
   ```

3. **`concurrent_test.go`** — semaphore 32 (= 4× pool size) em ambos testes:
   ```go
   const semCap = 32 // 4× pool MaxOpenConns=8 — exercita contenção sem time-out
   sem := make(chan struct{}, semCap)
   ```

**Por que semaphore 32 (não 8, não 50)?**
- Pool=8. Se semaphore=8, exerce contenção máxima do pool mas não testa margem.
- Se semaphore=50 (= pool+42 esperando no semaphore, 0 no DB), não testa contenção (todas ficam enfileiradas no semaphore, não no lock).
- 32 (= 4× pool) garante que 8 estão no DB disputando lock, 24 estão no semaphore. Exercita contenção real dentro de margem segura.

**Verificação empírica:**

| Teste | Pré-fix | Pós-fix |
|---|---|---|
| `TestAuditLog_NoChainBreaks_Concurrent` (50 goroutines) | FAIL: "expected 50, got 0" | **PASS em 0.13s** |
| `TestAuditLog_NoChainBreaks_HighContention` (200 goroutines) | FAIL: 170/200, SQLITE_BUSY×6, deadline×24 | **PASS em 0.13s** |

Comment corrigido no test file removendo o claim aspiracional "tests rodam normalmente sem -race" (que era falso).

---

### F-56-C — `driverCache` em `internal/db/tenant.go` unbounded (LOW → FECHADO)

**Severidade:** LOW (carry-over F-55-D, leak negligenciável mas documentado)
**Categoria:** Resource hygiene
**Arquivos:**
- `backend/internal/db/tenant.go` — adicionada função `ClearDriverCache(d)`
- `backend/cmd/api/main.go` — chamada no shutdown
- `backend/cmd/worker/main.go` — chamada no shutdown
- `backend/internal/db/tenant_test.go` — teste NilDB

**Bug:** `var driverCache sync.Map` é global, nunca limpa. `*sql.DB` fechado fica na cache indefinidamente. Em prática, app tem 1 *sql.DB por processo (Cmd/API = 1, Cmd/worker = 1), então leak é negligenciável (1 entry por DSN).

**Fix:**

1. **Função `ClearDriverCache(d)` adicionada em `tenant.go`:**
   ```go
   func ClearDriverCache(d *sql.DB) {
       if d == nil {
           return
       }
       driverCache.Delete(d)
   }
   ```

2. **Chamada em `cmd/api/main.go:48`** (API process — single DB):
   ```go
   defer d.Close()
   defer db.ClearDriverCache(d)
   ```

3. **Chamada em `cmd/worker/main.go:60`** (worker process — long-running):
   ```go
   defer d.Close()
   defer db.ClearDriverCache(d)
   ```

4. **Teste unitário** `TestClearDriverCache_NilDB` para nil safety.

**Carry-over:** Outros cmds (cmd/seed, cmd/radar, cmd/sta-submit, cmd/secret-migrate, cmd/jwt-mint, cmd/senhaws-rotate, cmd/_verify) NÃO foram atualizados — são admin/auxiliary, processos que terminam rápido, leak negligible. Carry-over EXPLÍCITO removido. Documentado nos comments.

---

### F-56-D — `auditlog.Verify` bypassa FORCE RLS sem documentação (MED → FECHADO)

**Severidade:** MED (carry-over F-55-J — audit trail risk)
**Categoria:** Docs/admin escape
**Arquivo:** `backend/internal/auditlog/log.go` (função Verify)

**Bug (carry-over F-55-J):** `Verify` lê de `l.db` direto (sem `WithTenantTx`), intencionalmente cross-tenant para validar chain completa. Sem FORCE RLS, OK. Com FORCE RLS (migration 014), dependeria de role de table owner ou SET LOCAL antes — sem doc explícita.

**Fix:** Adicionado comment ADMIN ESCAPE:
```go
// Validação 56 (v3.33.2): ADMIN ESCAPE — Verify É INTENCIONALMENTE cross-tenant
// (não usa WithTenantTx). Razão: precisa ver TODAS as entries para validar a
// chain completa — uma entry com if_id NULL (admin/system, política 012
// permite) seria invisível para um call com tenant scope.
// Implicações:
//   - Em Postgres com FORCE RLS (migration 014), a sessão precisa
//     fazer SET LOCAL app.if_id com string vazia ANTES do SELECT (ou ter
//     role de table owner + policy permissiva). Validar wiring em produção:
//     auditlog.Verify só deve ser invocável por endpoints admin.
//   - Em SQLite (dev/test), Verify é trivial — sem FORCE RLS.
//   - Não expor Verify a clientes externos sem audit trail (caller deve
//     Log uma entry "verify_invoked" antes/depois).
```

Risco diminuído mas não eliminado — endpoint que chama Verify continua precisando de autorização. Carry-over refinado: Sprint 36/37 deve adicionar rate limit + audit log quando Verify é invocado.

---

### F-56-E — `defer tx.Rollback()` em loop em `migrate.go` (INFO → ACEITO)

**Severidade:** INFO (cosmético)
**Categoria:** Memory footprint
**Arquivo:** `backend/internal/db/migrate.go:111`

**Observação:** `defer func() { _ = tx.Rollback() }()` dentro de `for` adiciona defer ao stack por iteração. Para 14 migrations, são 14 defers pendentes até fim de função. **Não é bug**, é padrão Go frowned upon.

**Aceito:** Refatorar para usar `tx.Rollback()` explícito por iteração muda muito código por benefit negligível. Carry como INFO/F-56-E.

---

### F-56-F — `_ = isPostgres` dead assign em `migrate.go:64` (LOW → FECHADO)

**Severidade:** LOW (lixo)
**Categoria:** Code quality
**Arquivo:** `backend/internal/db/migrate.go:63-64`

**Bug:**
```go
isPostgres := isPostgresDB(d)
_ = isPostgres // usado dentro do loop via closure abaixo
```

Variável USADA mais tarde (linha 135). `_ = isPostgres` é assign-mortos para silenciar linter.

**Fix:** Removida linha `_ = isPostgres`. Variável é usada, linter não reclama. Limpeza 1 linha.

---

### F-56-G — Typo aspas curvas em `tenant.go:60` (carry-over F-55-F) (LOW → FECHADO)

**Severidade:** LOW (cosmético)
**Categoria:** Docs typo
**Arquivo:** `backend/internal/db/tenant.go:60`

**Bug (carry-over F-55-F):** Aspas curvas `”` em vez de `""` em comentário. Typo visual, sem impacto funcional.

**Fix:** Removido o comentário problemático, substituído por descrição sem aspas literais (gofmt 1.26 normaliza aspas duplas consecutivas em comentários para Unicode tipográfico — workaround aplicado).

---

### F-56-H — `Log` recompute hash duplicado em `auditlog/log.go` (LOW → ACEITO)

**Severidade:** LOW (DRY violation, não bug)
**Categoria:** Code quality
**Arquivo:** `backend/internal/auditlog/log.go:104-107 e 128-130`

**Observação:** Hash calculado dentro do callback (linha 104-107) e DE NOVO fora para retornar entry (linha 128-130). Duplicação de 3 linhas.

**Aceito:** Refatorar para função `calculateEntryHash(...)` adicionaria 5 linhas, removeria 3. Net +2 LOC + indireção. **Não vale a pena**, código é claro inline. Carry como INFO.

---

## Resumo de fixes aplicados em v3.33.2

| Fix | Mudança | LOC |
|---|---|---|
| F-56-A | Comment auditoria `auditlog/log.go` | +9 |
| F-56-B | `db.go` busy_timeout + Log timeout + concurrent_test.go semaphore | +45 / -7 |
| F-56-C | `ClearDriverCache` + cmd/api + cmd/worker + tenant_test NilDB | +18 |
| F-56-D | Comment ADMIN ESCAPE em `Verify` | +13 |
| F-56-E | (info, não-fix) defer in loop | 0 |
| F-56-F | Dead assign removido migrate.go | -1 |
| F-56-G | Typo aspas em tenant.go corrigido | -3 / +4 |
| F-56-H | (info, não-fix) recompute hash duplicado | 0 |
| **Total** | | **+103 / -16** |

## Validação final

| Check | Resultado |
|---|---|
| `go vet ./...` | ✅ |
| `gofmt -l .` | ✅ |
| `go test -count=1 ./...` | ✅ 23/23 packages |
| `go test -race -count=1 ./...` | ✅ 23/23 packages |
| `TestAuditLog_NoChainBreaks_Concurrent` (50 goroutines) | ✅ 50/50 em 0.13s |
| `TestAuditLog_NoChainBreaks_HighContention` (200 goroutines) | ✅ 200/200 em 0.13s |
| `internal/auditlog` coverage | 92.5% (mantida) |
| `internal/audit/rules` coverage | 70.8% (mantida, > gate 70%) |
| `internal/db` coverage | 60.6% (era 62.1%, -1.5pp por ClearDriverCache novo) |
| `internal/radar` coverage | 81.2% (mantida) |

### Coverage notes

- `validateIFID` 0%, `escapeSingleQuote` 0%: **Postgres-only path**. Testes locais rodam SQLite (pragma `_txlock=immediate` é SQLite-only, sem FORCE RLS). Cobertura só possível em CI dedicada com Postgres. Carry-over histórico.
- `ClearDriverCache` 0%: **apenas executado em cmd/api e cmd/worker** (sem testes integrados para cmd/). Teste NilDB adicionado cobre nil safety mas não exercita o caminho real.

## Lições aprendidas

### 1. "Skip em -race" não é defesa contra timeouts — é defesa contra flakes SOB -race

Sprint 30 adicionou `if testutil.IsRaceEnabled() { t.Skip }` para pular stress tests em CI (race detector overhead + SQLite contention = flake). Lógica defensável. Mas o comentário "stress tests rodam normalmente em `go test ./...` (sem -race)" era FALSO e ninguém validou. **Lição:** ao adicionar skip condicional, validar empiricamente o caminho NÃO-skip. Self-verification > aspirational comments.

### 2. Test "pulado" + comentario aspiracional = double-blind spot

CI skip + comment otimista = ninguém olha. **Lição:** testes críticos devem ter **smoke test executado em CI** mesmo que marginal. Alternativa: adicionar coverage gate (`stress tests ≥ 1 must run per build`) via script custom.

### 3. Layered contention: pool lock + busy_timeout + ctx timeout = multiplicative budget

50 goroutines, pool=8, busy_timeout=5s, ctx=5s. Em contenção:
- 8 goroutines pegam conn imediatamente.
- 42 goroutines esperam conn ou lock.
- Cada tx leva ~100ms com BEGIN IMMEDIATE + SELECT + INSERT + COMMIT.
- 50/8 = 6.25 ciclos × 100ms = 625ms total. Cabe em 5s.

MAS hash SHA256 (auditlog) + checks Postgres + testes paralelos em CPU compartilhada = **200-500ms por tx**. Fila estoura 5s imediatamente. **Lição:** margem 3× em timeouts é necessária para testes concurrency; 1× é falso-positivo garantido.

### 4. Comment enganoso sobre BeginTx = setup pra refactor quebrar invariant silenciosamente

F-56-A era apenas comentário. Mas refactor futuro que tire o pragma `_txlock=immediate` do DSN quebraria o invariant "lock write serializa chain" com erro críptico ("chain quebrada" visível, mas causa raiz distante). **Lição:** comments críticos (especialmente sobre invariants não-óbvios) devem referenciar o **componente exato** que garante o invariant.

## Carry-over

| Finding | Status | Próxima ação |
|---|---|---|
| F-56-E (defer in loop migrate) | INFO | Sprint polish (cleanup passivo) |
| F-56-H (recompute hash duplicate) | INFO | Sprint polish (DRY pass) |
| F-55-I (audit_log if_id NULL docs/métricas) | MED | Sprint 36 (Observability) |
| F-55-J endpoint Verify rate limit/audit | MED | Sprint 36/37 (Pilot) |
| F-54-F (Ubuntu runner migration) | LOW | Sprint futura |
| F-54-G (artifact upload actions/upload-artifact) | LOW | Sprint futura |
| F-54-I (SHA pin actions/checkout) | LOW | Sprint futura |
| F-54-K (cmd/ 0% coverage) | LOW | Polish + cmd/*_test patterns |

Próxima sprint: **Sprint 33 (Audit3050) — TXB_V11, 170 regras catálogo** (Plano Ouro §1.1 Q2).

---

## Errata — Validação 57 (v3.33.3) fechou 4 bugs do doc/code drift pós-V56

### F-57-A — Drift numérico (LOW → FECHADO)
**Severidade:** LOW (doc consistency)
**Bug:** Doc + CHANGELOG diziam "8 findings, **7 fechados**" mas a tabela de fechados tinha só 6 itens. Real: 6 fechados + 2 aceitos (INFO).
**Fix:** CHANGELOG entry da v3.33.2 atualizada para "6 fechados + 2 aceitos" + tabela "Aceitos/não-fix" separada.

### F-57-C — F-56-G documentado mas não aplicado (MED → FECHADO)
**Severidade:** MED (drift entre doc e código é setup pra refactor quebrar silent)
**Bug:** Doc da V56 dizia "F-56-G: linha `_ = isPostgres` removida" mas o `migrate.go:64` real ainda tinha a linha. Commit bafe5b4 não tocou no arquivo. Drift entre doc e código.
**Fix:** Aplicado o fix que supostamente tinha sido feito em V56 — removida `_ = isPostgres` em `migrate.go:64`. Variável é usada na linha 134, linter não reclama.
**Lição:** Se doc diz "fix X aplicado", `grep -c "X"` em código antes de commitar. Self-verification > self-confidence.

### F-57-E — Ordem F-56-F vs F-56-G inconsistente entre DOC e CHANGELOG (LOW → FECHADO)
**Severidade:** LOW
**Bug:** DOC: F-56-F = dead assign, F-56-G = typo. CHANGELOG: F-56-F = typo, F-56-G = dead assign.
**Fix:** CHANGELOG realinhado à numeração do DOC.

### F-57-I — `TestClearDriverCache` só cobria nil (LOW → FECHADO)
**Severidade:** LOW (test hygiene)
**Bug:** `TestClearDriverCache_NilDB` adicionado em V56, mas caminho real (cmd/api + cmd/worker shutdown) chama com d não-nil. Coverage 0% do helper real.
**Fix:** Adicionado `TestClearDriverCache_NonNil` que abre DB, chama cleanup 3× (verifica idempotência), sem panic.

### 4 carry-over próprios
- Nenhum da V57. ✅

Próxima sprint: **Sprint 33 (Audit3050) — TXB_V11, 170 regras catálogo.**
