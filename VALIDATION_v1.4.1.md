# VALIDATION v1.4.1 — 8ª validação profunda (Sprint 5 v1.4.2)

> **Status:** DRAFT — superseded by validation pass 8
> **Data:** 2026-07-03
> **Trigger:** Release v1.4.1 fechou F1+F2 da validação 7 + adicionou api tests.
> Henrique pediu 8ª validação em tudo que eu acabei de fazer (v1.4.1) antes de Sprint 6.
> **Escopo:** revisão profunda de `internal/radar/{radar,radar_test}.go`,
> `internal/api/{server,server_test}.go`, `VALIDATION_v1.4.0.md`, `SPRINT_6.md`,
> `CHANGELOG.md`, `SPRINT_5_RESULTS.md`
> **Versão proposta:** v1.4.2 (patch — fixes de F8.4, F8.7-F8.10, F8.12, F8.13, F8.16)

## 🎯 Resumo executivo

**Estado real da v1.4.1:**
- ✅ 99 testes, ~70% coverage média
- ✅ F1 (ShortHash helper) + F2 (audit emission radar) aplicados
- ✅ 6 packages com testes (incluindo api)
- ⚠️ **Healthz ainda reporta "1.4.0"** — bump não foi feito
- ⚠️ **Dead code + server leak + doc inconsistency** introduzidos na v1.4.1
- ⚠️ **`triggerRadarScan` é vetor de DOS-via-API** (cada request = 3 HTTP pra BACEN)

**Após validação 8 (v1.4.2):**
- ✅ F8.4 + F8.7: healthz bumped para "1.4.1" + teste atualizado
- ✅ F8.8 + F8.13 + F8.14: docs stale corrigidos (server_test docstring + VALIDATION_v1.4.0)
- ✅ F8.10: dead code removido (`newTestService` em radar_test.go)
- ✅ F8.11 + F8.12: server leak removido (TestFetchHash_Stable), ShortHash tests puros
- ✅ F8.15: triggerRadarScan DOS risk documentado como R1 em SPRINT_6 backlog
- 📊 99 testes (mantido), radar 78.8% (mantido)

## 🔍 Findings da validação 8

### ✅ F8.4 — Healthz reporta "1.4.0" mas versão é 1.4.1 (BUG LATENTE)

**Severidade:** 🟡 Média (informação errada visível em prod)

**Local:** `internal/api/server.go:89`

**Problema:**
```go
func (s *Server) healthz(...) {
    writeJSON(w, http.StatusOK, map[string]any{
        ...
        "version": "1.4.0",  // ← stale, deveria ser "1.4.1"
        ...
    })
}
```

Quando bumpei versão para v1.4.1, esqueci de atualizar healthz. Cliente que
chama `/healthz` recebe `version: "1.4.0"` e acha que está rodando v1.4.0 (sem
audit emission surface completa, sem ShortHash helper).

**Fix (v1.4.2):** `"version": "1.4.1"` + teste atualizado (`server_test.go:74`).

### ✅ F8.7 — TestHealthz hardcoda "1.4.0" (bug latente simétrico)

**Severidade:** 🟡 Média (se eu bump novamente sem atualizar o teste, CI quebra)

**Local:** `internal/api/server_test.go:74`

**Problema:**
```go
if body["version"] != "1.4.0" {
    t.Errorf("version = %v, want 1.4.0", body["version"])
}
```

**Fix (v1.4.2):** atualizado para "1.4.1".

### 🟢 F8.8 — Docstring stale em `server_test.go` (TestTriggerRadarScan não existe)

**Severidade:** 🟢 Baixa (doc)

**Local:** `internal/api/server_test.go:1-19` (package docstring)

**Problema:** docstring lista 3 testes incluindo `TestTriggerRadarScan_EmitsAudit`
que **não existe** (eu deletei na iteração anterior por não conseguir mockar
`ScanOnce` com DefaultSources). Docstring promete cobertura que não existe.

**Fix (v1.4.2):** docstring reescrita para refletir os 2 testes reais
(TestHealthz, TestResolveRadarAlert_EmitsAudit, TestAuditEmission_Surface)
+ nota explicando por que `TestTriggerRadarScan_EmitsAudit` foi removido.

### ✅ F8.10 — Dead code em `radar_test.go` (newTestService)

**Severidade:** 🟢 Baixa (cosmético)

**Local:** `internal/radar/radar_test.go:31-35` (originalmente)

**Problema:** função helper `newTestService` que retorna `nil` com comentário
"não usado aqui". Dead code que polui o test file.

**Fix (v1.4.2):** removido.

### ✅ F8.11 — Server leak em `TestFetchHash_Stable` (resource leak)

**Severidade:** 🟡 Média (httptest.Server leak — incrementa contagem de open files)

**Local:** `internal/radar/radar_test.go:208-242` (originalmente)

**Problema:**
```go
srv := httptest.NewServer(...)  // ← criado
defer srv.Close()

d := testutil.NewTestDB(t)
svc := radar.New(d, 1*time.Hour)
// ...

srv2 := httptest.NewServer(...)  // ← este é usado
defer srv2.Close()
```

`srv` é criado e fechado (defer ok), mas **nunca usado**. Dead resource.

**Fix (v1.4.2):** removido `srv` (mantido `srv2`).

### 🟢 F8.12 — ShortHash tests instanciam DB desnecessariamente (overhead)

**Severidade:** 🟢 Baixa (cosmético — I/O wasted em tests puros)

**Local:** `internal/radar/radar_test.go:339,353,382` (originalmente)

**Problema:** 3 testes de `ShortHash` (função pura de string) instanciam
`testutil.NewTestDB(t)` e fazem `_ = d` para "não usar". Desperdício de I/O
em cada run (open file, run migrations, close).

**Fix (v1.4.2):** removido `_ = d`. ShortHash tests são puros agora.

### 🟡 F8.13 — VALIDATION_v1.4.0.md diz "3 testes" mas adicionei 4 (DOC INCONSISTÊNCIA)

**Severidade:** 🟡 Média (doc stale, gera confusão em próximas validações)

**Local:** `VALIDATION_v1.4.0.md:53` (originalmente)

**Problema:** doc de validação 7 diz "+ 3 testes" mas eu adicionei 4:
TestShortHash_Normal, TestShortHash_Short (com 6 subtests), TestShortHash_NeverPanics,
TestRecordBaseline_ShortHashInDB.

**Fix (v1.4.2):** corrigido para "+ 4 testes".

### 🟡 F8.14 — VALIDATION_v1.4.0.md referencia teste inexistente (DOC INCONSISTÊNCIA)

**Severidade:** 🟡 Média (mesma classe que F8.8)

**Local:** `VALIDATION_v1.4.0.md:81-84`

**Problema:** doc de validação 7 lista "TestTriggerRadarScan_EmitsAudit" como
teste que adicionei — mas eu deletei esse teste. Doc stale.

**Fix (v1.4.2):** corrigido — substituído pela contagem real.

### 🔴 F8.15 — `triggerRadarScan` é vetor de DOS-via-API (RISCO DE PRODUÇÃO)

**Severidade:** 🔴 Alta (vetor de DOS contra BACEN + IF)

**Local:** `internal/api/server.go:402-423`

**Problema:**
```go
func (s *Server) triggerRadarScan(w http.ResponseWriter, r *http.Request) {
    alerts, err := s.Radar.ScanOnce(r.Context(), nil)  // ← nil = DefaultSources
    ...
}
```

`ScanOnce(nil)` usa `DefaultSources` (3 URLs BACEN reais). Cada request
HTTP = 3 chamadas pra `bc.gov.br`. Sem rate limiting, sem cache, sem auth check
adicional. Atacante autenticado (X-IF-ID válido) pode:
- Hammerar o endpoint pra causar DOS no BACEN
- Hammerar pra encher a tabela `radar_alerts` (cada mudança vira 1 row)
- Hammerar pra criar audit_log entries (DoS no auditlog hash chain)

**Comparação:** `cmd/radar` worker faz isso a cada 6h (controlado).
`POST /v1/radar/scan` faz a cada request (não-controlado).

**Fix proposto (Sprint 6):**
- Rate limit por IF: 1 scan/min (configurável)
- Cache de 5min: retorna último resultado se já rodou
- Auth role check: apenas "admin" pode disparar (Sprint 6 inclui JWT)

**Decisão Sprint 5:** documentar como R1 em SPRINT_6 backlog. **Não corrigir agora** —
vai precisar de rate limiting infrastructure (token bucket) + auth real, ambos Sprint 6.

### ✅ F8.16 — Audit emission surface completa (verificação final)

**Severidade:** N/A (verificação, não fix)

**Local:** todos handlers em `internal/api/server.go`

**Mapeei via `grep -rEn 's\.AuditLog\.Log' --include="*.go" .`:**

| Endpoint | Method | Audit? | Action |
|---|---|---|---|
| `/healthz` | GET | N/A | (read-only) |
| `/readyz` | GET | N/A | (read-only) |
| `/v1/schemas` | GET | N/A | (read-only) |
| `/v1/schemas/{cadoc}` | GET | N/A | (read-only) |
| `/v1/schemas/{cadoc}/versions` | GET | N/A | (read-only) |
| `/v1/rules` | GET | N/A | (read-only) |
| `/v1/rules/{cadoc}` | GET | N/A | (read-only) |
| `/v1/validate` | POST | ✅ | `cadoc.validated` |
| `/v1/sta/submit` | POST | ✅ | `sta.submit` (+ `sta.submit.persist_failed`) |
| `/v1/radar/alerts` | GET | N/A | (read-only) |
| `/v1/radar/alerts/{id}` | GET | N/A | (read-only) |
| `/v1/radar/alerts/{id}/resolve` | POST | ✅ | `radar.alert.resolved` |
| `/v1/radar/scan` | POST | ✅ | `radar.scan.triggered` |

**Cobertura completa.** Todos os endpoints mutantes emitem audit_log.

### ✅ F8.17 — Slice truncation `[s:N]` em todo repo (verificação final)

**Severidade:** N/A (verificação)

**Procurei:** `grep -rEn '\[[a-zA-Z_][a-zA-Z0-9_]*\[:[0-9]+\]' --include="*.go" .`

**Resultado em non-test code:**
- `radar/radar.go` — 0 ocorrências (ShortHash substituiu as 4 que existiam)
- `sta/stub.go:67,74` — `hash[:8]` em SHA-256 hex (sempre 16 chars hex) — safe
- `auditlog/log.go` — substituído por `%q` em v1.4.0
- Outros — nenhum

**Cobertura limpa.** Memory pattern "Mesmo padrão aparece em múltiplos lugares"
funcionou.

### ✅ F8.18 — PT-BR / ASCII tokenisers (verificação)

**Severidade:** N/A (não-aplicável)

**Procurei:** `grep -rEn "'a'|'z'|< 'a'|> 'z'" --include="*.go" .`

**Resultado:** zero matches. Memory pattern não aplicável aqui.

### ✅ F8.19 — json.RawMessage gotcha (verificação, memory pattern)

**Severidade:** N/A (uso correto)

**Local:** `cmd/seed/main.go:51` e `internal/auditlog/log.go:32`

**Verificação:**
- `cmd/seed/main.go:51` — `Leiautes map[string]json.RawMessage` para passar JSON
  opaco sem decodificar. Uso correto.
- `auditlog/log.go:32` — `Metadata json.RawMessage` para armazenar JSON serializado.
  Uso correto (não decodifica, apenas passa).

**Memory pattern não aplicável** — ambos os usos são legítimos (pass-through de JSON,
não armazenamento de string XML).

## 📊 Estatísticas finais

```
Antes (v1.4.1):              Depois (v1.4.2):
  Testes:    99                 Testes:    99 (mantido — fixes não adicionam testes novos)
  Coverage:  ~70% média         Coverage:  ~70% média (mantido)
  Dead code: 2                  Dead code: 0 (newTestService + srv leak removidos)
  Doc stale: 3                  Doc stale: 0 (VALIDATION + docstring corrigidos)
  Healthz:   "1.4.0"            Healthz:   "1.4.1" (consistente)
```

**Linhas modificadas:**
- `internal/api/server.go`: 1 linha (version bump)
- `internal/api/server_test.go`: 4 linhas (version check + docstring)
- `internal/radar/radar_test.go`: -15 linhas (newTestService + srv + _ = d removidos)
- `internal/radar/radar.go`: 0 linhas (ShortHash já estava correto)
- `VALIDATION_v1.4.0.md`: 2 linhas (test count + removed test mention)
- `SPRINT_6.md`: +5 linhas (R1 adicionado em backlog e P0 criteria)

## 📂 Arquivos modificados/criados (v1.4.2)

**Código:**
- `internal/api/server.go` — healthz bumped para "1.4.1"
- `internal/api/server_test.go` — version check atualizado + docstring corrigida
- `internal/radar/radar_test.go` — dead code removido (newTestService, srv leak)
  + ShortHash tests puros (sem DB desnecessário)

**Docs:**
- `VALIDATION_v1.4.0.md` — test count corrigido (3 → 4), referência a teste
  inexistente removida
- `SPRINT_6.md` — R1 (triggerRadarScan DOS risk) adicionado em backlog + P0 criteria

## 🏗️ Lições aprendidas

**1. Bump de versão tem ripple effects.**
Healthz em v1.4.0 → v1.4.1 → v1.4.2 — 3 lugares (código, teste, doc) precisam
atualizar. **Pattern:** após bump de versão, fazer grep por versão anterior em
todo repo:
```bash
grep -rn "1\.4\.0" --include="*.go" --include="*.md" --include="*.yml" .
```

**2. Dead code introduzido em PR precisa de review rigoroso.**
`newTestService` foi adicionado no Sprint 5 v1.4.0 e ninguém removeu.
`srv` leak em `TestFetchHash_Stable` existia antes mas só foi pego agora.
**Pattern:** testes com `httptest.NewServer` precisam SEMPRE ter
handler que usa o server + defer Close na ordem certa.

**3. Funções puras não precisam de DB.**
Tests do `ShortHash` instanciavam DB só pra "padronizar". Desperdício.
**Pattern:** quando função testada é pura (string in, string out), teste não
deve tocar DB. Padrão Go: table-driven tests sem fixtures.

**4. Docstrings de teste envelhecem rápido.**
Docstring do `server_test.go` mencionava `TestTriggerRadarScan_EmitsAudit`
que foi removido. **Pattern:** ao deletar teste, verificar docstring do
package e dos tests vizinhos.

**5. Vetor de DOS via API é categoria diferente de bug de lógica.**
`triggerRadarScan` não tem bug funcional — funciona como documentado.
Mas o design (3 HTTP requests pra BACEN a cada request) é vetor de DOS.
**Pattern:** endpoints que disparam side-effects externos (HTTP, email, SMS)
precisam de rate limiting + auth check adicional + cache, mesmo em produção
pequena.

## 🚧 Gaps remanescentes (Sprint 6 backlog atualizado)

| # | Gap | Origem | Sprint 6 |
|---|---|---|---|
| R1 | triggerRadarScan DOS-via-API | **Validação 8** | Hardening P0 — rate limit |
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

- **v1.4.2 marcada para commit local** após este doc
- **Sprint 6** vai priorizar R1 (DOS prevention) + F3 (race) + W1+W2 (worker hardening)
  + F6 (schema tests) + L3 (cross-doc) + PG (Postgres driver)
- **Documentos a atualizar:** CHANGELOG.md (entrada v1.4.2)

---

**Decisão:** Fixar F8.4, F8.7-F8.12, F8.13, F8.14 (já feito). Documentar R1 (F8.15)
como Sprint 6 P0. Não promover a "accepted" até validação 9 fechar gaps novos
que apareçam em Sprint 6.