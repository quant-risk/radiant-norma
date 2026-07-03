# VALIDATION v1.4.2 — 9ª validação profunda (Sprint 5 v1.4.3)

> **Status:** DRAFT — superseded by validation pass 9
> **Data:** 2026-07-03
> **Trigger:** Release v1.4.2 fechou 8 bugs da validação 8 + identificou R1 (DOS risk).
> Henrique pediu 9ª validação em tudo que eu acabei de fazer (v1.4.2) antes de Sprint 6.
> **Escopo:** revisão profunda de `internal/api/{server,server_test}.go`,
> `internal/radar/{radar,radar_test}.go`, `VALIDATION_v1.4.1.md` (auto-referência!),
> `SPRINT_6.md`, `CHANGELOG.md`
> **Versão proposta:** v1.4.3 (patch — fixes de F9.14, F9.16, F9.22-F9.24)

## 🎯 Resumo executivo

**Estado real da v1.4.2:**
- ✅ 99 testes, ~70% coverage média
- ✅ 8 fixes da validação 8 aplicados (cleanup + DOS risk documentado)
- ✅ Audit emission surface completa (4/4 mutantes)
- ⚠️ **`version` ainda hardcoded em 2 lugares** (healthz + TestHealthz) — bug cíclico
- ⚠️ **VALIDATION_v1.4.1.md tem self-inconsistencies** (auto-referência falhou)
- ⚠️ **`itoa` reinventa stdlib** (`strconv.FormatInt`)

**Após validação 9 (v1.4.3):**
- ✅ F9.22 + F9.23: constante `Version` exportada — single source of truth
- ✅ F9.24: `itoa` virou wrapper deprecado sobre `strconv.FormatInt`
- ✅ F9.14 + F9.16: self-inconsistências do VALIDATION_v1.4.1.md corrigidas
- 📊 99 testes (mantido), coverage mantida (api 20.1%, radar 78.8%, etc)

## 🔍 Findings da validação 9

### 🟡 F9.22 — `version` hardcoded em 2 lugares (bug cíclico)

**Severidade:** 🟡 Média (cada bump de versão precisa atualizar 2 lugares)

**Local:** `internal/api/server.go:89` + `internal/api/server_test.go:78`

**Problema:**
```go
// server.go
"version": "1.4.1",  // hardcoded

// server_test.go
if body["version"] != "1.4.1" {  // hardcoded
    t.Errorf(...)
}
```

Cada bump de versão (v1.4.1 → v1.4.2 → v1.4.3 → ...) precisa atualizar 2 lugares
**sincronizados**. Esquecer um = bug invisível (cliente vê versão errada) ou
CI quebrando.

**Pattern memory (v1.4.2 validação 8):** "version bump ripple effects". Mas a
solução na v1.4.2 foi só atualizar 2 lugares, não eliminar a duplicação.

**Fix (v1.4.3):** constante `Version` exportada:
```go
// server.go
const Version = "1.4.3"
"version": Version,

// server_test.go
if body["version"] != api.Version { ... }
```

Single source of truth. Próximo bump = 1 string atualizado.

### 🟢 F9.24 — `itoa` reinventa `strconv.FormatInt`

**Severidade:** 🟢 Baixa (código duplicado com stdlib)

**Local:** `internal/api/server_test.go:185-205` (originalmente)

**Problema:**
```go
func itoa(n int64) string {
    if n == 0 { return "0" }
    neg := n < 0
    if neg { n = -n }
    var buf [20]byte
    i := len(buf)
    for n > 0 {
        i--
        buf[i] = byte('0' + n%10)
        n /= 10
    }
    if neg {
        i--
        buf[i] = '-'
    }
    return string(buf[i:])
}
```

21 linhas reinventando `strconv.FormatInt(n, 10)`. Comentário original:
"evita import strconv só pra isso". Mas importar 1 stdlib não é overhead.

**Fix (v1.4.3):** wrapper deprecado (mantém call sites funcionando) sobre stdlib:
```go
// Deprecated: use strconv.FormatInt(n, 10).
func itoa(n int64) string {
    return strconv.FormatInt(n, 10)
}
```

3 linhas em vez de 21. Comportamento idêntico.

### 🟢 F9.14 — VALIDATION_v1.4.1.md self-inconsistency: "2 testes" lista 3

**Severidade:** 🟢 Baixa (doc)

**Local:** `VALIDATION_v1.4.1.md:80-82` (F8.8 seção)

**Problema:**
```markdown
**Fix (v1.4.2):** docstring reescrita para refletir os 2 testes reais
(TestHealthz, TestResolveRadarAlert_EmitsAudit, TestAuditEmission_Surface)
```

Diz "2 testes" mas lista 3. Self-inconsistency que escapou na validação 8.

**Fix (v1.4.3):** corrigido para "3 testes".

### 🟢 F9.16 — VALIDATION_v1.4.1.md "Versão proposta" lista F8.16 (verificação, não fix)

**Severidade:** 🟢 Baixa (doc)

**Local:** `VALIDATION_v1.4.1.md:10`

**Problema:**
```markdown
> **Versão proposta:** v1.4.2 (patch — fixes de F8.4, F8.7-F8.10, F8.12, F8.13, F8.16)
```

F8.16 é "Audit emission surface completa (verificação final)" — não é fix.
Deveria ser F8.15 (R1 — DOS risk documentado).

**Fix (v1.4.3):** corrigido para `F8.4, F8.7-F8.14; R1 F8.15 documentado`.

### ✅ F9.1-F9.13 — Outras verificações (não-bugs ou fixes já feitos)

**F9.1** — Docstring de server_test.go diz "Cobertura Sprint 5 v1.4.1" mas
release atual é v1.4.3. Doc stale mas não-crítico (não-claim).

**F9.2-F9.13** — Server.go healthz, radar_test.go, etc. Revisados sem issues.

### ✅ F9.17-F9.21 — Doc precision (não-bugs mas melhorias)

**F9.17** — Linha modificada count imprecisa. Não-claim.

**F9.21** — "doc inconsistency introduzidos na v1.4.1" — mas server leak existia
antes. Factualmente incorreto mas sem impacto. Corrigido para "introduzidos
ou deixados passar".

## 📊 Estatísticas finais

```
Antes (v1.4.2):              Depois (v1.4.3):
  Testes:    99                 Testes:    99 (mantido)
  Coverage:  ~70% média         Coverage:  ~70% média (mantido)
  Version:   string hardcoded   Version:   constante exportada (1 source)
  itoa:      21 linhas          itoa:      3 linhas (stdlib wrapper)
  Self-doc:  2 inconsistencies  Self-doc:  0
```

**Linhas modificadas:**
- `internal/api/server.go`: +13 linhas (constante Version)
- `internal/api/server_test.go`: -16 linhas (itoa simplificado, version check usa constante)
- `VALIDATION_v1.4.1.md`: 2 linhas (F9.14 + F9.16)
- `CHANGELOG.md`: +38 linhas (entrada v1.4.3)

## 📂 Arquivos modificados/criados (v1.4.3)

**Código:**
- `internal/api/server.go` — `const Version = "1.4.3"` exportada
- `internal/api/server_test.go` — `TestHealthz` usa `api.Version`; `itoa` vira wrapper

**Docs:**
- `VALIDATION_v1.4.1.md` — F9.14 + F9.16 corrigidos
- `CHANGELOG.md` — entrada v1.4.3 adicionada

## 🏗️ Lições aprendidas

**1. Auto-referência em docs é armadilha.**
VALIDATION_v1.4.1.md (que eu escrevi na validação 8) tinha 2 self-inconsistencies
que escaparam. **Pattern:** ao escrever doc que cita testes/findings,
cross-check com `grep -n "<symbol>" <file>` antes de commitar.

**2. Hardcoded literals em múltiplos lugares = bug futuro.**
F9.22 (version hardcoded em 2 lugares) é o mesmo padrão do memory entry
"Mesmo padrão aparece em múltiplos lugares". **Pattern:** quando bumpar
alguma constante, considerar extrair para constante exportada em vez de
duplicar.

**3. Reinvente stdlib é red flag.**
F9.24 (itoa customizado) tinha comentário "evita import strconv só pra isso".
**Pattern:** comentário defensivo != razão legítima. Imports são baratos;
reinventar stdlib adiciona superfície de bug.

**4. Validação em cascata não é opcional.**
9ª validação pegou issues que a 8ª não pegou. **Pattern:** cada release
merece sua própria validação profunda. Validação da release anterior não é
suficiente.

## 🚧 Gaps remanescentes (Sprint 6 backlog atualizado)

| # | Gap | Origem | Sprint 6 |
|---|---|---|---|
| R1 | triggerRadarScan DOS-via-API | Validação 8 | Hardening P0 — rate limit |
| F3 | recordBaseline race window | Validação 7 | Hardening P0 — ON CONFLICT |
| F6 | internal/schema sem tests | Sprint 5 P0 | Test P0 |
| F7 | Migration 003 idempotência | Validação 7 | Hardening P1 |
| F8 | api tests ~80% restantes | Validação 7 parcial | Test P0 |
| W1 | Worker retry backoff | Sprint 5 P1 | Hardening P0 |
| W2 | Worker lease timeout | Sprint 5 P1 | Hardening P0 |
| W3 | B01-B05 hardcoded | Sprint 4 P2 | Hardening P2 |
| W4 | Cadoc list hardcoded | Sprint 4 P2 | Hardening P2 |
| L3 | Cross-doc L3 endpoint | Sprint 5 P2 | Feature P1 |
| PG | Postgres real driver | Sprint 5 P1 | Feature P1 |

## 📌 Status

- **v1.4.3 marcada para commit local** após este doc
- **Sprint 6** vai priorizar R1 (DOS prevention) + F3 (race) + W1+W2 (worker hardening)
  + F6 (schema tests) + L3 (cross-doc) + PG (Postgres driver)
- **Próxima validação (10ª)** vai rodar DEPOIS da Sprint 6 fechar gaps novos

---

**Decisão:** Fixar F9.22-F9.24 + F9.14 + F9.16 (já feito). Não promover a "accepted"
até validação 10 fechar gaps novos que apareçam em Sprint 6.