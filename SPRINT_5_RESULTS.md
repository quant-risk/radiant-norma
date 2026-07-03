# Sprint 5 — Testes Unitários (v1.4.0)

> **Data:** 2026-07-03
> **Status:** ✅ Concluída
> **Tema:** Maturidade de engenharia — testes automatizados + hardening
> **Resultado:** **86 testes, ~70% coverage média, 5 bugs latentes detectados e corrigidos**

## 🎯 Objetivo da sprint

Fechar o gap de "validação manual encontra 80% em 4-5 passadas, depois satura"
documentado em SPRINT_5.md. Implementar testes unitários + CI para detectar
regressões automaticamente.

## 🏛️ Entregas

### 🔴 P0 — Testes unitários

| Package | Arquivo | Testes | Coverage |
|---|---|---|---|
| `internal/testutil` | `db.go` | 1 | 45.0% |
| `internal/auditlog` | `log_test.go` | 9 | 90.8% |
| `internal/audit/rules` | `3040_test.go` | 56 | 63.2% |
| `internal/audit` | `service_test.go` | 12 | 68.7% |
| `internal/radar` | `radar_test.go` | 8 | 78.1% |
| **TOTAL** | **5 arquivos** | **86 testes** | **~70% média** |

### 🔴 Bônus — 5 bugs latentes detectados pelos testes

| # | Bug | Severidade | Arquivo | Fix |
|---|---|---|---|---|
| 1 | **auditlog.Verify panica em tampering** (slice `[:12]` em string < 12 chars) | 🔴 Alta | `auditlog/log.go:165,182` | Use `%q` em vez de `[:12]` |
| 2 | **F02Datas aceita mês 13, dia 32** (só validava formato, não ranges) | 🟡 Média | `audit/rules/3040.go::F02Datas` | Função `validateDateSemantics` |
| 3 | **recordBaseline cria `_baseline_drsac faq`** (espaço em vez de underscore) | 🟡 Média | `radar/radar.go::recordBaseline` | Helper `baselineTypeFor` com replacer |
| 4 | **radar_alerts UNIQUE constraint viola em scans rápidos** (mesmo detected_at + title) | 🔴 Alta | `db/migrations/003_radar_no_unique.sql` | Migration remove UNIQUE |
| 5 | **LoadCriticas quebra em `mensagem_erro=NULL`** (converting NULL to string) | 🔴 Alta | `audit/service.go::LoadCriticas` | `sql.NullString` para campo opcional |

**Total: 3 bugs ALTO + 2 bugs MÉDIO encontrados pelos testes.** Sem testes,
esses bugs iriam pra produção.

### 🟡 P1 — Hardening + DX

- **`internal/testutil/db.go`** — helper `NewTestDB(t)` retorna SQLite in-memory com migrations aplicadas, fecha em `t.Cleanup`
- **`Makefile`** — alvos `test`, `test-race`, `test-cover`, `lint`, `vet`, `fmt`, `seed`, `run-api`, `clean`
- **`.github/workflows/test.yml`** — CI roda `go vet`, `go test -race`, coverage thresholds (auditlog ≥85%, radar ≥70%), `gofmt` check
- **`backend/README.md`** — quickstart + architecture + critical rules
- **Healthz `version: "1.4.0"`** — bump

## 📊 Estatísticas

```
Antes (v1.3.6):                Depois (v1.4.0):
  Testes:    0                   Testes:    86
  Coverage:  0%                  Coverage:  ~70% média
  Files:    ~20                  Files:    25 (+5 test files)
  LOC:      3.215                LOC:      ~4.200 (+985)
  CI:       nenhum               CI:       GitHub Actions
  Makefile: nenhum               Makefile: 9 alvos
```

## 📂 Arquivos modificados/criados

**Código:**
- `internal/testutil/db.go` (novo, 50 linhas)
- `internal/auditlog/log_test.go` (novo, 200 linhas)
- `internal/audit/rules/3040_test.go` (novo, 350 linhas)
- `internal/audit/service_test.go` (novo, 220 linhas)
- `internal/radar/radar_test.go` (novo, 270 linhas)
- `internal/audit/rules/3040.go` (modificado, +validateDateSemantics)
- `internal/audit/service.go` (modificado, LoadCriticas usa NullString)
- `internal/auditlog/log.go` (modificado, Verify não panica)
- `internal/radar/radar.go` (modificado, baselineTypeFor helper)
- `internal/db/migrations/003_radar_no_unique.sql` (novo, remove UNIQUE)
- `internal/api/server.go` (modificado, version 1.3.6 → 1.4.0)

**Infra/Docs:**
- `backend/Makefile` (novo, 9 alvos)
- `backend/README.md` (novo, quickstart)
- `.github/workflows/test.yml` (novo, CI workflow)

## 📊 Suite de regressão E2E

```
✓ Validate XML oficial           → passed=true, 0 erros
✓ Healthz v1.4.0                 → uptime=2, version=1.4.0
✓ Audit chain após 5 validates   → 6 entries, válida
✓ Worker 3 envios processados    → status=accepted
✓ Radar idempotente              → 1 baseline após 3 scans
✓ F02 mês 13 detectado           → erro E (era bug, agora fixed)
✓ Stress 50 concurrent           → 50/50 chain válida (v1.3.5 fix mantido)
✓ go vet clean
✓ gofmt clean
✓ 86 testes passando
```

## 🚧 Gaps remanescentes (vão pra Sprint 6)

| Gap | Sprint 5 origem | Sprint 6 |
|---|---|---|
| Worker retry sem backoff/limite | P2 (atrasado) | Sprint 6 — adicionar `attempts` column |
| Worker lease timeout | P2 (atrasado) | Sprint 6 — reset stuck processing |
| B01-B05 hardcoded | Sprint 4 P2 | Sprint 6 — mover pro registry |
| Cadoc list hardcoded | Sprint 4 P2 | Sprint 6 — carregar do DB |
| Mais regras 3040 (295 faltam) | Sprint 4 P2 | Sprint 6 — Sprint de regras |
| Cross-doc L3 | Sprint 5 P2 (atrasado) | Sprint 6 — endpoint `/v1/crossdoc/validate` |
| Driver Postgres real | Sprint 4 P2 | Sprint 6 — pgx + Docker compose |
| Frontend Norma Console | Sprint 4 P2 | Sprint 7 — Next.js dashboard |

## 🏗️ Lições aprendidas

**1. Testes encontram bugs que validação manual não pega.**
5 bugs latentes em ~1.000 linhas de código novo (testes). Sem testes, todos
iriam pra produção. Padrão: **regressão com assertion forte > validação
manual repetida**.

**2. Helpers de teste pagam dividendo.**
`testutil.NewTestDB(t)` reduziu boilerplate de 20+ linhas por teste para 1.
Total economizado: ~400 linhas.

**3. `go test -race` é obrigatório em qualquer código concorrente.**
O `auditlog/log_test.go::TestLog_Concurrent` (100 goroutines) é o teste
mais valioso do projeto: regressão direta da race condition do v1.3.5.

**4. `sql.NullString` é mandatório para campos opcionais.**
A regra é: se o campo pode ser NULL no DB (mesmo que "nunca deveria ser"),
use `sql.NullString` no Scan. Sem isso, 1 INSERT manual ou migration antiga
quebra toda a validação (L2-LOAD).

**5. Regex não valida ranges.**
`^\d{4}-\d{2}(-\d{2})?$` aceita "2020-13" e "2020-01-32". Sempre combinar
regex com validação semântica de ranges.

**6. UNIQUE constraints em `(time, string)` são frágeis.**
`UNIQUE(cadoc_code, alert_type, detected_at, title)` viola em scans rápidos
com mesmo `time.Now()`. Removido na migration 003.

## 📌 Próximos passos

Sprint 6 vai focar em:
1. **Hardening worker** — backoff + lease + dead-letter
2. **Driver Postgres** — `pgx` + Docker compose
3. **Frontend** — Next.js dashboard (Sprint 7 ou 6?)
4. **Cross-doc L3** — endpoint novo
5. **Schema tests** — `internal/schema/registry_test.go` (F6 do VALIDATION_v1.4.0)
6. **Record baseline race** — `INSERT ON CONFLICT` ou tx serializada (F3 do VALIDATION_v1.4.0)

A infraestrutura de testes v1.4.0 + v1.4.1 dá confiança pra essas mudanças. Cada
nova feature terá regression test desde o primeiro commit.

---

## 🆕 v1.4.1 — Pós-validação profunda 7

> **Status:** patch aplicado após 7ª validação (ver `VALIDATION_v1.4.0.md`)
> **Trigger:** Henrique pediu validação profunda + commit local + segue pra Sprint 6
> **Mudanças:** F1 (ShortHash helper) + F2 (audit emission radar endpoints) + 13 testes novos

### Fixes
- **F1** (🟡 Média): `radar.go` tinha 4× `s[:12]` em hash — mesmo padrão do bug #1 v1.4.0
- **F2** (🟡 Média): `resolveRadarAlert` + `triggerRadarScan` não emitiam audit

### Tests (+13)
- `internal/api/server_test.go` (NOVO): healthz + audit emission canary (3 testes)
- `internal/radar/radar_test.go` (+4 testes): ShortHash normal/short/never-panics + RecordBaseline corrompido

### Estatísticas
```
Testes:    86 → 99 (+13)
Packages:  5 com testes → 6 com testes (+api)
Coverage:  radar 78.1% → 78.8%, api NOVO 20.1%
```

### Gaps remanescentes (Sprint 6 backlog)
| Gap | Validação | Prioridade Sprint 6 |
|---|---|---|
| F3 — recordBaseline race window | 7ª | P0 |
| F6 — internal/schema sem tests | 7ª | P0 |
| W1+W2 — worker hardening | 5ª | P0 |
| L3 — cross-doc endpoint | 5ª | P1 |
| PG — Postgres driver | 5ª | P1 |
| F8 — api tests restantes | 7ª | P2 |

---

**Decisão Sprint 5 fechada:** P0 completo + v1.4.1 patch aplicado. 99 testes + CI
workflow + Makefile + README + audit emission surface completa. Suficiente pra Sprint 6.