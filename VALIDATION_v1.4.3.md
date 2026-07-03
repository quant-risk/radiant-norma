# VALIDATION v1.4.3 — 10ª validação profunda (Sprint 5 v1.4.4)

> **Status:** DRAFT — superseded by validation pass 10
> **Data:** 2026-07-03
> **Trigger:** Release v1.4.3 introduziu `api.Version` constante + `itoa` wrapper.
> Henrique pediu 10ª validação em tudo que eu acabei de fazer (v1.4.3) antes de Sprint 6.
> **Escopo:** revisão profunda de `internal/api/{server,server_test}.go`,
> `internal/radar/radar.go`, `VALIDATION_v1.4.2.md` (auto-referência!),
> `SPRINT_6.md`, `CHANGELOG.md`, MEMORY.md
> **Versão proposta:** v1.4.4 (patch — fixes de F10.6, F10.9-F10.15)

## 🎯 Resumo executivo

**Estado real da v1.4.3:**
- ✅ 99 testes, ~70% coverage média
- ✅ F9.22-F9.24 aplicados (Version constant + itoa wrapper)
- ✅ F9.14 + F9.16 corrigidos (VALIDATION_v1.4.1.md self-inconsistencies)
- ⚠️ **`itoa` wrapper ficou redundante** após F9.24 — podia ser removido direto
- ⚠️ **User-Agent hardcoded em radar.go** com versão stale ("1.3")
- ⚠️ **VALIDATION_v1.4.2.md tem 5 self-inconsistencies** (auto-referência falhou DE NOVO)

**Após validação 10 (v1.4.4):**
- ✅ F10.6: itoa wrapper removido completamente — `strconv.FormatInt` direto
- ✅ F10.9: User-Agent bumped para v1.4.4 (mas ainda hardcoded — gap arquitetural)
- ✅ F10.3 + F10.11-F10.15: self-inconsistências corrigidas
- 📊 99 testes (mantido), coverage mantida

## 🔍 Findings da validação 10

### ✅ F10.6 — `itoa` wrapper redundante após F9.24

**Severidade:** 🟢 Baixa (cleanup)

**Local:** `internal/api/server_test.go:185-205`

**Problema:** F9.24 transformou `itoa` (21 linhas reinventando stdlib) em wrapper
deprecado (3 linhas). Mas o comentário `Deprecated: use strconv.FormatInt(n, 10)`
é pista que o wrapper não é necessário — todos os call sites são internos ao
test file. Wrapper deprecado interno = dead code.

**Fix (v1.4.4):** wrapper removido completamente. `strconv.FormatInt(n, 10)`
usado direto nos 2 call sites (linhas 103, 147 originalmente).

**Padrão:** quando você deprecia algo interno, remova-o em vez de manter
"para compatibilidade" — call sites internos podem ser atualizados.

### 🟡 F10.9 — User-Agent hardcoded com versão stale

**Severidade:** 🟡 Média (visível em produção — BACEN vê versão errada)

**Local:** `internal/radar/radar.go:204`

**Problema:**
```go
req.Header.Set("User-Agent", "Mozilla/5.0 (Radiant-Norma-Radar/1.3; +https://fortvna.com.br)")
```

Versão stale ("1.3") visível em TODO request que o radar faz pro BACEN. BACEN pode
logar isso e detectar inconsistência entre a versão reportada em `/healthz` e a
reportada em User-Agent.

**Fix (v1.4.4):** atualizado para "1.4.4" hardcoded. **Gap arquitetural conhecido**:
deveria referenciar `api.Version` mas isso requer criar `internal/version` package
compartilhado (ou re-export). Vai pra Sprint 6 backlog como F10.10.

### 🟢 F10.3 — Docstring `Cobertura Sprint 5 v1.4.1` stale

**Severidade:** 🟢 Baixa (doc)

**Local:** `internal/api/server_test.go:3`

**Problema:** docstring do package diz "Cobertura Sprint 5 v1.4.1" mas release
atual é v1.4.4. **Pattern memory:** "Docstrings de teste envelhecem rápido".

**Fix (v1.4.4):** removido o version stamp da docstring. Agora descreve o que
os testes fazem sem ancorar em release específica.

### 🟢 F10.11 — VALIDATION_v1.4.2.md F9.24 diz "wrapper deprecado" mas foi removido

**Severidade:** 🟢 Baixa (doc — auto-referência)

**Local:** `VALIDATION_v1.4.2.md:96-102`

**Problema:** doc da validação 9 diz que F9.24 criou "wrapper deprecado
(mantém call sites funcionando) sobre stdlib". Mas a validação 10 removeu
o wrapper completamente (F10.6). Doc stale.

**Fix (v1.4.4):** atualizado para "inicialmente wrapper deprecado... na v1.4.4
wrapper completamente removido".

### 🟢 F10.12 — VALIDATION_v1.4.2.md lista "itoa vira wrapper"

**Severidade:** 🟢 Baixa (doc)

**Local:** `VALIDATION_v1.4.2.md:174`

**Problema:** diz "`internal/api/server_test.go` — ... `itoa` vira wrapper"
mas o itoa foi removido na v1.4.4.

**Fix:** corrigido para "itoa wrapper removido completamente (v1.4.4)".

### 🟢 F10.13-F10.14 — VALIDATION_v1.4.2.md "Próxima validação" stale

**Severidade:** 🟢 Baixa (doc)

**Local:** `VALIDATION_v1.4.2.md:222-224`

**Problema:** diz "Próxima validação (10ª) vai rodar DEPOIS da Sprint 6
fechar gaps novos". Mas 10ª validação foi rodada (esta). Auto-referência
falhou de novo.

**Fix:** atualizado para mencionar que validação 10 rodou.

### 🟢 F10.15 — SPRINT_6.md trigger stale

**Severidade:** 🟢 Baixa (doc)

**Local:** `SPRINT_6.md:7`

**Problema:** doc diz "Trigger: 7ª validação profunda (v1.4.1) fechou gaps
de audit emission; gaps remanescentes viram Sprint 6". Mas 8ª (v1.4.2),
9ª (v1.4.3), 10ª (v1.4.4) validações também fecharam gaps.

**Fix:** atualizado para mencionar v1.4.1 → v1.4.4 cronologicamente.

### 🟡 F10.10 — Gap arquitetural: Version ainda hardcoded em radar.go

**Severidade:** 🟡 Média (architectural debt)

**Local:** `internal/radar/radar.go:204`

**Problema:** mesmo após F9.22 (api.Version), User-Agent do radar continua
hardcoded. **Causa:** radar package não pode importar api package (dependência
unilateral — api depende de radar, não o contrário). Para compartilhar a
constante, seria preciso:

**Opção A:** criar `internal/version/version.go`:
```go
package version
const Version = "1.4.4"
```
E usar tanto em `api.Version` (re-export ou referência) quanto em radar.

**Opção B:** definir `Version` em radar package também, com sincronização
manual (ruim — reintroduz F8.4).

**Opção C:** build-time injection via `-ldflags "-X main.Version=..."`
(padrão Go para binários).

**Decisão Sprint 5:** Opção A é a cleanest. Vai pra Sprint 6 backlog
como gap arquitetural. Por agora (v1.4.4), atualizado hardcoded.

### ✅ F10.1-F10.5 — Outras verificações (não-bugs ou fixes já feitos)

- **F10.1** — server.go importa strconv, usado em 3 lugares. OK.
- **F10.2** — server_test.go importa strconv, usado em 2 call sites via FormatInt. OK.
- **F10.4** — Histórico referência v1.4.1 OK (é história, não-claim).
- **F10.5** — Histórico referência v1.4.3 OK (é história, não-claim).

### ✅ F10.16-F10.20 — Verificações de padrões memory

| Pattern | Aplicação | Status |
|---|---|---|
| Slice truncation `[s:N]` | grep todo repo non-test | 0 ocorrências (clean) |
| Audit emission surface | mapear mutantes | 4/4 mutantes emitem |
| PT-BR ASCII tokenisers | grep | 0 matches (não-aplicável) |
| json.RawMessage gotcha | usos | 2 legítimos (pass-through) |
| DOS-via-API risk | mapear endpoints | 1 identificado (R1, Sprint 6) |

## 📊 Estatísticas finais

```
Antes (v1.4.3):              Depois (v1.4.4):
  Testes:    99                 Testes:    99 (mantido)
  Coverage:  ~70% média         Coverage:  ~70% média (mantido)
  itoa:      3 linhas (wrapper) itoa:      0 linhas (removido)
  User-Agent: "1.3"             User-Agent: "1.4.4"
  Self-doc:  5 inconsistencies  Self-doc:  0
```

**Linhas modificadas:**
- `internal/api/server_test.go`: -7 linhas (wrapper removido, call sites inlined)
- `internal/api/server.go`: 1 linha (Version bumped)
- `internal/radar/radar.go`: 1 linha (User-Agent bumped)
- `VALIDATION_v1.4.2.md`: 4 seções corrigidas
- `SPRINT_6.md`: 1 linha corrigida

## 📂 Arquivos modificados/criados (v1.4.4)

**Código:**
- `internal/api/server.go` — Version bumped "1.4.3" → "1.4.4"
- `internal/api/server_test.go` — itoa wrapper removido completamente
- `internal/radar/radar.go` — User-Agent bumped "1.3" → "1.4.4"

**Docs:**
- `VALIDATION_v1.4.2.md` — 5 self-inconsistencies corrigidas (F10.11-F10.15)
- `SPRINT_6.md` — trigger atualizado para mencionar v1.4.1-v1.4.4
- `CHANGELOG.md` — entrada v1.4.4 adicionada

## 🏗️ Lições aprendidas

**1. Wrapper deprecado interno = dead code.**
F10.6 confirmou: quando você deprecia algo que só é usado internamente,
**remova** em vez de manter "para compatibilidade". Compatibilidade só faz
sentido para API pública ou para call sites fora do seu controle.

**2. Auto-referência em docs falha repetidamente.**
Memory pattern diz "validar 9 pegou self-ref do 8". Validação 10 pegou
self-ref do 9. **Pattern confirma:** toda validação deve cross-check os
docs da validação anterior com `grep -n "<symbol>" <file>`.

**3. Single source of truth requer refator arquitetural.**
F10.10 mostra: api.Version não se propaga automaticamente para radar.
`go test` passa mas User-Agent fica stale. **Pattern:** quando você
introduz constante compartilhada, verificar TODOS os lugares que usam
versão hardcoded em runtime-visible places (User-Agent, log, metadata).

**4. Validação em cascata não é opcional — confirmado novamente.**
10ª validação pegou 5+ issues que escaparam da 9ª. 4 validações na mesma
release (v1.4.0 → v1.4.4) encontraram **~30 issues**. Cada release
merece sua validação.

## 🚧 Gaps remanescentes (Sprint 6 backlog consolidado)

| # | Gap | Origem | Sprint 6 |
|---|---|---|---|
| F10.10 | Version compartilhada api ↔ radar | Validação 10 | Hardening — `internal/version` package |
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

- **v1.4.4 marcada para commit local** após este doc
- **Sprint 6** vai priorizar F10.10 (version shared) + R1 + F3 + W1+W2 + F6 + L3 + PG
- **Próxima validação (11ª)** vai rodar após Sprint 6 fechar gaps novos

---

**Decisão:** Fixar F10.6, F10.9-F10.15 (já feito). Documentar F10.10 como gap
arquitetural para Sprint 6. Não promover a "accepted" até validação 11 fechar
gaps novos da Sprint 6.