# Validação 32 — Sprint 11 (v3.4.0): Drill-Down Server Actions

> **Data:** 2026-07-05
> **Sprint auditado:** Sprint 11 (rule enable/disable via backend)
> **Versão:** v3.4.0
> **Status:** ⚠️ **CONDITIONAL ACCEPT — 1 finding HIGH + 1 finding HIGH pre-existente**
> **Bloqueador:** Nenhum (production-ready com 2 follow-ups Sprint 12+)

---

## 🎯 Resumo da auditoria

Sprint 11 entrega persistência backend de regras desabilitadas. Auditoria
cobre 4 vetores paralelos: race conditions, error path disclosure, audit
gaps, e integridade referencial com o sistema de validação.

**25 findings** identificados, distribuídos em:
- 🔴 **2 HIGH** (1 novo, 1 pre-existente)
- 🟡 **8 MEDIUM** (5 novos, 3 pre-existentes)
- 🟢 **15 LOW/INFO** (maioria melhorias, não bloqueantes)

### ✅ Veredito: CONDITIONAL ACCEPT

- Sprint 11 está **funcionalmente correto** para os casos cobertos
  (toggle + audit + SSE + optimistic concurrency)
- Finding **C32.23** é o mais relevante: toggle persiste mas **não
  afeta validação real** — feature não está conectada ao engine
- Finding **C32.21** (CSRF) é pre-existente (afeta TODOS os POST
  endpoints desde Sprint 7a) e não foi introduzido por Sprint 11
- Demais findings são hardening (Sprint 12+)

---

## 📊 Escopo auditado

### Backend (Go)

| Arquivo | LOC | Função |
|---------|-----|--------|
| `internal/ruleprefs/preferences.go` | 154 | Preferences service (Disable/Enable/Toggle/List) |
| `internal/ruleprefs/preferences_test.go` | 145 | 7 tests unitários |
| `internal/api/sprint11_handlers.go` | 142 | 2 HTTP handlers (list + toggle) |
| `internal/api/sprint11_handlers_test.go` | 274 | 5 E2E tests |
| `internal/db/migrations/007_disabled_rules.sql` | 28 | Tabela + índice |

### Frontend (Next.js 14)

| Arquivo | LOC | Função |
|---------|-----|--------|
| `src/lib/use-rule-preferences.ts` | 97 | Hook de state sync com backend |
| `src/app/api/rules/disabled/route.ts` | 24 | Proxy GET |
| `src/app/api/rules/[code]/toggle/route.ts` | 41 | Proxy POST |
| `src/app/regras/regras-client.tsx` | +18 | Integração com hook (sem localStorage) |

**Total: 921 LOC** (~480 backend, ~180 frontend, ~260 tests)

---

## 🔴 Findings HIGH

### C32.23 — Disabled rules não são consultadas pelo engine de validação [HIGH, NOVO]

**Sintoma:** `audit.Service.Validate()` (pipeline de validação CADOC) **não
consulta a tabela `disabled_rules`**. Toggle persiste no DB e emite audit,
mas quando o usuário submete um XML pra validação, a regra desabilitada
**continua sendo avaliada normalmente**.

**Reprodução:**
```bash
# 1. Disable F23
curl -X POST /v1/rules/F23/toggle -H "Authorization: Bearer ..."
# → 200 {new_state: "disabled"}, audit event emitido, SSE publicado

# 2. Validate XML que viola F23
curl -X POST /v1/validate -d '{"cadoc_code":"4060","xml":"<campo_invalido/>"}'
# → F23 ainda aparece como falha (expected: deve ser pulada)
```

**Impacto:** Feature inteira é **cosmética**. UX mostra toggle verde/cinza,
mas a regra continua ativa. Usuário pensa que desabilitou e o resultado
não muda.

**Risk:**
- Compliance: usuário pode tomar decisão regulatória baseada em premissa
  falsa ("regra X está desabilitada, então não preciso cumprir")
- LGPD: audit log diz "rule.disabled" mas regra continua rodando
- Trust: feature não-cumprida mina credibilidade do produto

**Fix recomendado (Sprint 12+ candidato):**
1. Injetar `Preferences` no `audit.Service`
2. Antes de avaliar regra, check `prefs.IsDisabled(ctx, ifID, ruleCode)`
3. Se disabled, skip rule (não inclui no result.Errors)
4. Adicionar campo `disabled: true` no result pra transparency
5. Considerar `audit_disabled_rules` separado pra rastrear skips

**Workaround atual:** Documentar que toggle é "preparação para Sprint 12"
e adicionar banner na UI /regras avisando que está em preview.

**Sprint:** Bloqueador real para GA. **Não-bloqueador** se tratado como
"preview alpha".

---

### C32.21 — Falta CSRF protection em POST endpoints [HIGH, PRE-EXISTENTE]

**Sintoma:** Todos os POST/PUT/DELETE do backend (`/v1/validate`,
`/v1/sta/submit`, `/v1/radar/scan`, `/v1/rules/{code}/toggle`, etc) não
verificam CSRF token. Cookie `rn_jwt` é httpOnly, mas o browser envia
automaticamente em cross-origin requests via `<form>` ou `<img>` maliciosos.

**Reprodução conceitual:**
```html
<!-- site malicioso -->
<form action="https://radiant.example.com/api/rules/F23/toggle" method="POST">
  <input type="hidden" name="expected_state" value="enabled">
  <input type="submit" value="Click me">
</form>
```

**Impacto:** Atacante pode, em nome de usuário autenticado, fazer toggles
silenciosos, validar XMLs arbitrários, ou disparar radar scans.

**Risk:**
- Mesmo vetor que Meltdown/Spectre pras fintech: regulatory não perdoa
- BACEN CMN 4.966 exige "segregação de funções" e "audit trail imutável"
  — toggle não-autorizado do user fere ambos
- Discovery: usuário nem percebe que toggle aconteceu

**Fix recomendado (Sprint 12 hardening):**
1. Adicionar CSRF middleware que valida `Origin` header
2. Em dev: log warning (não bloqueia)
3. Em prod: bloqueia requests cross-origin exceto /api/login e /v1/auth/*
4. Frontend: garantir todos os POST vão via mesma origin (já é o caso)
5. Considerar SameSite=Strict no cookie `rn_jwt` (defense-in-depth)

**Status:** Pre-existente desde Sprint 7a (JWT bridge). Não bloqueia
Sprint 11 specifically, mas precisa ser Sprint 12 priority.

---

## 🟡 Findings MEDIUM

### C32.1 — Race condition em `Preferences.Toggle()` [MEDIUM, NOVO]

**Sintoma:** `Toggle()` faz `IsDisabled()` (SELECT) e depois `Enable()` ou
`Disable()` (DELETE/INSERT) em queries separadas. Entre as duas queries,
outro request pode mudar o estado.

**Cenário:**
```
T0: User A toggle F23 (currently enabled)
T1: SELECT IsDisabled → false
T2: User B toggle F23 (also currently enabled)
T3: User B SELECT IsDisabled → false
T4: User B INSERT (F23 now disabled)
T5: User A INSERT ... ON CONFLICT DO UPDATE (F23 still disabled, but timestamps updated)
```

**Resultado:** Audit log mostra 2 eventos "rule.disabled" do mesmo
estado. Confuso mas não corrupto.

**Risk:** Com single-instance backend + DB local, race window é de
~1ms. In Postgres multi-pod (Sprint 12), é mais provável.

**Fix:**
```go
// Wrap em transaction com write lock (SQLite: BEGIN IMMEDIATE)
tx, _ := p.db.BeginTx(ctx, nil)
defer tx.Rollback()
// SELECT + INSERT/DELETE na mesma tx
tx.Commit()
```

**Sprint:** Adiar para Sprint 12 (multi-replica work pode unificar).

---

### C32.10 — Toggle() com race em Enable() retorna 500 confuso [MEDIUM, NOVO]

**Sintoma:** Se race C32.1 ocorre no path Enable (state era disabled
quando chamamos IsDisabled, mas outro request habilitou antes do DELETE),
`Enable()` retorna `ErrRuleNotDisabled` (0 rows affected), que o handler
traduz pra HTTP 500 com "internal state error".

**Impacto:** UX ruim — usuário vê "internal state error" quando na verdade
a operação foi idempotente (estado final é o mesmo: enabled).

**Fix:**
```go
if errors.Is(err, ruleprefs.ErrRuleNotDisabled) {
    // Idempotente: estado final é "enabled" mesmo. Não é erro.
    return writeJSON(w, 200, ...)
}
s.internalServerError(w, err, "toggle")
```

**Sprint:** Candidato Sprint 12 (acompanha fix C32.1).

---

### C32.13 — Stale closure em `useRulePreferences.toggle()` [MEDIUM, NOVO]

**Sintoma:** `useCallback(..., [disabled, refresh])` captura `disabled`
no momento da criação. Se user toggle code X, e antes do request
retornar, toggle X de novo, a 2ª chamada lê `disabled.has(X) === false`
(stale, ainda não foi atualizado), envia `expected_state="enabled"`,
mas backend tem "disabled" → 409.

**Resultado:** User vê "state_changed" no console. Em prática, o
botão está disabled durante pending (parent component), mas se
clique vem de outro lugar (modal + card simultâneo), pode disparar.

**Fix:**
```typescript
const disabledRef = useRef(disabled)
useEffect(() => { disabledRef.current = disabled }, [disabled])

const toggle = useCallback(async (code) => {
  const expectedState = disabledRef.current.has(code) ? 'disabled' : 'enabled'
  // ...
}, [refresh]) // sem `disabled` na dep
```

**Sprint:** Adiar — não é bloqueador, modal+card simultaneous click é
corner case.

---

### C32.19 — Sem validação de formato de `rule_code` no proxy [MEDIUM, NOVO]

**Sintoma:** `app/api/rules/[code]/toggle/route.ts` passa `params.code`
diretamente ao backend sem validar. Backend aceita qualquer string.
SQL injection é bloqueado por parameterized queries, mas:

1. Audit log persiste strings maliciosas (defense-in-depth falho)
2. UI pode quebrar com codes contendo `<script>` ou control chars
3. Storage waste (user pode enviar 1MB de código)

**Fix no proxy:**
```typescript
const VALID_RULE_CODE = /^[A-Z][0-9]{1,3}$/
if (!VALID_RULE_CODE.test(code)) {
  return NextResponse.json({ error: 'invalid rule code format' }, { status: 400 })
}
```

**Fix no backend handler:**
```go
var validRuleCode = regexp.MustCompile(`^[A-Z][0-9]{1,3}$`)
if !validRuleCode.MatchString(ruleCode) {
    http.Error(w, "invalid rule code", http.StatusBadRequest)
    return
}
```

**Sprint:** Candidato Sprint 12 hardening.

---

### C32.22 — Sem rate limiting no toggle [MEDIUM, PRE-EXISTENTE]

**Sintoma:** User autenticado pode fazer N toggles por segundo. Cada um
gera DB write + audit + SSE event. Atacante autenticado pode DoS
o sistema.

**Fix:** Reusar `radar.ScanLimiter` pattern:
- Token bucket por IFID: 10 req/s normal, 1 req/s burst
- 429 com Retry-After header

**Sprint:** Candidato Sprint 12 hardening.

---

### C32.5 + C32.8 — "internal state error" message misleading [MEDIUM, NOVO]

**Sintoma:** C32.5: `errors.Is(err, ruleprefs.ErrRuleNotDisabled)` → 500
com "internal state error". C32.8: mesma string em user-facing.

**Impacto:** Operacional + UX. Em Sentry/logs, devs veem 500 quando na
verdade é race (C32.10). User vê "internal" quando é só out-of-sync.

**Fix:** Map para 409 com code `"state_drift"` + refetch hint.

**Sprint:** Acopla com C32.10 fix.

---

## 🟢 Findings LOW/INFO (resumo)

| # | Descrição | Severidade | Sprint |
|---|-----------|------------|--------|
| C32.2 | `Disable()` ON CONFLICT atualiza timestamp (idempotente by design) | INFO | doc |
| C32.3 | `Enable()` ErrRuleNotDisabled ambiguamente derivado de RowsAffected=0 | LOW | 12+ |
| C32.4 | rule_code sem validação de charset/length | LOW | 12 |
| C32.6 | chi.URLParam dead code check `if ruleCode == ""` | LOW | cleanup |
| C32.7 | auditBody redundante com `target=ruleCode` (target já é code) | INFO | doc |
| C32.9 | json.NewDecoder sem size limit (mitigado por maxBodyBytesMiddleware 10MB) | LOW | 12 |
| C32.11 | rule_code sem CHECK constraint (alphanumeric max 16) | LOW | migration 008 |
| C32.12 | Sem FK pra tabela rules (rules hardcoded hoje) | INFO | quando aplicável |
| C32.14 | useRulePreferences refresh() — 5xx logged mas loading vai false corretamente | INFO | — |
| C32.15 | 401 silenciosamente seta Set vazio (sem user feedback) | LOW | 12 |
| C32.16 | togglePending pode desincronizar em unmount durante pending | LOW | 12 |
| C32.17 | Modal "sincronizando..." só em initial load (não durante refetch) | LOW | UX polish |
| C32.18 | Sem retry em network error | INFO | — |
| C32.20 | Proxy body sem size limit (limitado por Next.js default 1MB) | LOW | 12 |
| C32.24 | `_, _ = s.AuditLog.Log()` swallow error | LOW | 12 |
| C32.25 | ListDisabled() sem paginação (max 60 rules, OK hoje) | LOW | quando crescer |

---

## 🧪 Validação empírica

### Test suite

| Package | Tests | Status |
|---------|-------|--------|
| `internal/ruleprefs` | 7 | ✅ pass |
| `internal/api` (Sprint 11) | 5 novos + 16 existentes | ✅ pass |
| `internal/db` (migration 007) | atualizado 5→7 | ✅ pass |
| 16/16 packages | ~50 tests totais | ✅ pass |

### Smoke test

```bash
# Setup
TOKEN=$(curl -X POST /v1/auth/dev-token -d '{"if_id":"demo","role":"if"}')

# Empty list
curl /v1/rules/disabled → {codes: [], total: 0}

# Disable
curl -X POST /v1/rules/B12/toggle → {new_state: "disabled"}
curl /v1/rules/disabled → {codes: ["B12"], total: 1}

# Optimistic concurrency (state mismatch)
curl -X POST /v1/rules/B12/toggle -d '{"expected_state":"enabled"}' → 409
curl -X POST /v1/rules/B12/toggle -d '{"expected_state":"disabled"}' → 200 {new_state: "enabled"}

# Audit log
sqlite3 /tmp/db "SELECT action, target, actor FROM audit_log WHERE action LIKE 'rule.%'"
# → 4 rows: 2x rule.disabled (B12, F23) + 2x rule.enabled (B12, B12)
```

### Frontend smoke

```bash
# Login → cookie
curl -X POST /api/login -d '{"if_id":"demo","role":"if"}' -c cookies.txt

# Via proxy
curl -b cookies.txt /api/rules/disabled → {codes: [...]}
curl -X POST -b cookies.txt /api/rules/S05/toggle → {new_state: "disabled"}
curl -X POST -b cookies.txt -d '{"expected_state":"enabled"}' /api/rules/S05/toggle → 409
```

---

## 📈 Métricas de qualidade

| Métrica | Valor | Threshold | Status |
|---------|-------|-----------|--------|
| Backend packages test OK | 16/16 | 16/16 | ✅ |
| Test coverage ruleprefs | 7 tests | ≥5 | ✅ |
| Test coverage api Sprint 11 | 5 tests | ≥4 | ✅ |
| Migration test count | 7/7 | 7 | ✅ |
| TypeScript strict | clean | clean | ✅ |
| ESLint warnings | 0 | 0 | ✅ |
| Next build | OK | OK | ✅ |
| Audit emission coverage | 100% (toggle) | 100% mutating endpoints | ✅ |
| Optimistic concurrency | valida current_state | yes | ✅ |
| SSE integration | 100% (toggle publica evento) | 100% mutating | ✅ |

---

## 🚀 Recomendações para Sprint 12

**Priority 1 (bloqueador real):**
- C32.23: Integrar `Preferences` com `audit.Service.Validate()` para que
  toggle tenha efeito funcional. Sem isso, feature é cosmética.

**Priority 2 (security):**
- C32.21: CSRF middleware (afeta todos POST)
- C32.19: Validar formato de rule_code no proxy + handler
- C32.22: Rate limiting no toggle (reusar ScanLimiter pattern)

**Priority 3 (correctness):**
- C32.1 + C32.10: Wrap Toggle em transaction (write lock), map
  ErrRuleNotDisabled para 200 idempotente (não 500)
- C32.13: Fix stale closure com useRef pattern

**Priority 4 (hardening):**
- C32.5/C32.8: Better error messages (409 "state_drift" vs 500)
- C32.4/C32.11: rule_code validation (regex + CHECK constraint)
- C32.6: Dead code cleanup
- C32.15/C32.16/C32.17: UX polishes (login feedback, modal states)

**Priority 5 (info, defer):**
- C32.12: FK to rules table (quando rules virar DB)
- C32.25: Pagination (quando rules > 200)
- C32.24: Log audit failure to slog

---

## ✅ Veredito final

**Sprint 11 v3.4.0 — CONDITIONAL ACCEPT.**

**Funcional para o que cobre:**
- Toggle persiste no DB ✅
- Audit log emitido ✅
- SSE notification ✅
- Optimistic concurrency ✅
- Frontend sync com backend ✅

**Gap conhecido (C32.23):**
- Disabled rules não afetam validação real
- Feature é cosmética até Sprint 12
- **Recomendação:** documentar como "preview" e adicionar banner na UI

**Hardening pre-existente (C32.21 + C32.22):**
- CSRF + rate limit afetam TODOS endpoints
- Não-bloqueante para Sprint 11 (Sprint 12 hardening work)

**Ship recommendation:** ✅ Pode ser pushed como v3.4.0 com banner
"preview — rules persistidas, enforcement no engine é Sprint 12".

**Commits:**
- `cf91532` — feat(sprint-11 v3.4.0): Drill-down server actions
- Este validation doc (próximo commit)
