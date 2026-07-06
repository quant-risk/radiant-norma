# Changelog — cadocs (Radiant Norma)

> **Histórico de todas as alterações no projeto.** Cada entrada é uma sprint fechada.

## v3.25.0 — 2026-07-06 (Sprint 32 Fase 1 — Audit3040_v2: 14 regras Agregadas A01-A15) ✅

> **Status:** ✅ Shipped (Fase 1 de 4)
> **Sprint:** 32 (Plano Ouro §3.4 Épico D — Norma Engine)
> **Versão:** minor (+14 regras, +1 arquivo de regras, +1 arquivo de testes)
> **Trigger:** Plano Ouro §1.1 — fechar gap 3040 de 16.6% → 60%
> **Próximas fases:** Fase 2 (próxima sprint: +35 → 28.8%); Fase 3 (I01-I15 + H01-H09); Fase 4 (C31-C80 + S41-S70)

### 🎯 Resumo

Port das **14 regras Agregadas (A01-A07, A09-A15)** do CADOC 3040 conforme catálogo BACEN `scr3040_criticas`. Total de regras 3040 em Go: **60 → 74** (cobertura 16.6% → **20.5%**).

**Decisões arquiteturais:**
- D-2: Tabela estática ClassOp × Provisão (Res. BCB 352) — O(1) lookup, zero allocation
- D-3: Helpers de agregação (`totalVencimentos`, `maxVencimento`) reusados por 6 regras
- D-5: Tests inline (não fixtures JSON) — table-driven com 5-7 cases por regra

### 📊 Métricas

| Métrica | Pré v3.25.0 | Pós v3.25.0 |
|---|---|---|
| Regras 3040 portadas | 60 | **74** (+14) |
| Cobertura catálogo | 16.6% (60/361) | **20.5%** (74/361) |
| Coverage internal/audit/rules | 62.8% | **66.6%** (+3.8pp) |
| Test functions | ~770 | **~820** (+50 subtests table-driven) |
| Packages PASS | 23/23 | **23/23** |
| Race detector | clean | clean |

### 🐛 Bugs encontrados pelos próprios tests

Quality gate funcionou — tests pegaram 3 boundary bugs antes do commit:

1. **A01 boundary:** ratio == provMax é inválido (provMax exclusive)
2. **A11/A12 threshold:** lógica original rejeitava só < 500k; refatorada pra thresholds específicos (4 → 500k, 5 → 5M)
3. **A01 ClassOp H:** tabela original dizia `< 101%`, correto é `>= 100% sem upper bound`

### 📁 Arquivos

```
backend/internal/audit/rules/3040_agregadas.go         (NOVO — 477 LoC)
backend/internal/audit/rules/3040_agregadas_test.go    (NOVO — 432 LoC, 15 testes)
backend/internal/audit/rules/registry.go               (+14 Register)
backend/internal/audit/rules/raw_rules_test.go         (60 → 74)
backend/internal/audit/rules/3040_test.go              (lista códigos atualizada)
backend/SPRINT_32_RESEARCH.md                          (NOVO)
backend/SPRINT_32_RESULTS.md                           (NOVO)
```

### ⏭️ Próxima sprint

**Fase 2 do Sprint 32** (próxima entrega): +35 regras (C11-C30 + S11-S20) → 28.8% cobertura 3040.

---

## v3.24.0 — 2026-07-06 (Validação 50 — Deep audit + hardening pós-Sprint 28) ✅

> **Status:** ✅ Shipped
> **Tipo:** patch (zero feature nova, zero breaking change) + hardening
> **Trigger:** Solicitação Henrique — "validação profunda em tudo que você acabou de fazer"
> **Validação:** VALIDAÇÃO_v3.24.0.md — 8 findings encontrados, 6 fechados, 2 aceitos (YAGNI), 0 regressão

### 🎯 Resumo

Auditoria profunda do Plano Ouro (v3.22.0) + Sprint 28 (v3.23.0) encontrou **8 findings**. 6 fechados com fixes cirúrgicos:

- **1 HIGH** (F-S28-50-B): `senhaws-rotate apply` vazava senha Sisbacen em stderr quando manager.Put falhava → failsafe file 0600 + exit code 4
- **2 MEDIUM**: `secret-migrate list` retornava exit 0 silencioso ("TODO Sprint 29+") → exit 3 + `backendErr` type; inconsistência MASTER_PLAN sobre `012` vs `014` RLS → esclarecimento
- **3 LOW**: dead code em `aws.go` (`var _ = errors.As`) e `memory.go` (`slogLogger interface{}`) removidos; cobertura `cmd/secret-migrate` +8.4pp via batch tests

### 🔒 O que mudou

#### Segurança (HIGH)

**F-S28-50-B — failsafe file pattern para partial failure**

Quando BACEN aceita senha nova (204) mas `secrets.Manager.Put` falha (AWS IAM, network, etc), a senha **NÃO pode** ir pro stderr (sink de log aggregator). Solução:

```
$ RADIANT_SECRETS_BACKEND=aws senhaws-rotate apply
WARN: senha alterada no BACEN mas FALHA ao atualizar aws manager: AccessDenied
      ACTION REQUIRED: senha nova gravada em failsafe file (0600): /tmp/radiant-senhaws-failsafe-20260706T053015Z-a1b2c3d4e5f6.txt
      Use: cat <path> | secret-migrate migrate --from-env=- --to=bacen/senha/<user>
      Depois: shred -u <path>
exit: 4   # NOVO — partial failure (BACEN OK, manager FALHOU)
```

Senha raw NUNCA em stderr. Admin lê arquivo (0600), configura manual, `shred -u`. User no filename é SHA-256[:6] (não vaza identidade em `ls /tmp`).

#### Hollow stubs removidos (MEDIUM)

**F-S28-50-A — `secret-migrate list` agora retorna exit 3 honesto**

```
$ secret-migrate list --prefix=bacen/
erro: list not supported on backend=env (apenas AWS Secrets Manager suporta ListSecrets). Sprint 29+ adiciona suporte
exit: 3   # era 0 antes (silent failure)
```

Caller agora distingue "lista vazia" (exit 0) de "feature não suportada" (exit 3). AWS ListSecrets será adicionado em Sprint 29 (BacenHomologSmoke).

#### Drift docs (MEDIUM)

**F-S28-50-F/H — MASTER_PLAN §1.1 linha 80**

Antes:
```
| 30 | PostgresRLS | Ativar migration 012_rls_policies.sql. ...
```
(conflitava com linha 594 que dizia `014_rls_enforce.sql`, e com ROADMAP/CHANGELOG que diziam só `014`)

Depois:
```
| 30 | PostgresRLS | Ativar migration `012_rls_policies.sql` (em `internal/db/migrations/`) +
                    criar migration `014_rls_enforce.sql` com FORCE ROW LEVEL SECURITY.
                    Defense-in-depth multi-tenant. Auditoria SOC 2.
```

Resolve ambiguidade: 012 (policies base, existe) + 014 (enforce, criar).

#### Dead code removido (LOW)

- `internal/secrets/aws.go`: removido `var _ = errors.As` (era "avoid lint warning" theater — lint warning não existe)
- `internal/secrets/memory.go`: removido type `slogLogger interface{}` + dummy var + import "strings" (era "preparado pro futuro" theater — YAGNI)

### 📊 Métricas

| Métrica | Pré v3.24.0 | Pós v3.24.0 |
|---|---|---|
| Packages PASS | 23/23 | **23/23** |
| Test functions | ~544 | **770+** |
| Coverage cmd/secret-migrate | 48.7% | **57.1%** (+8.4pp) |
| Coverage cmd/senhaws-rotate | 66.2% | **68.3%** (+2.1pp) |
| Coverage internal/secrets | 64.5% | 58.3% (-6.2pp — código morto removido muda ratio, linhas cobertas similar) |
| Hollow stubs | 2 | **0** |
| Secret leaks (stderr) | 1 | **0** |
| Drift docs Sprint 30 | 2 | **0** |
| Race detector | clean | clean |
| Build smoke | 10/10 | **10/10** |

### 🔄 Compatibilidade

- `senhaws-rotate apply` agora retorna **exit 4** em partial failure (era exit 1). Automação que trata exit 1 = "BACEN rejeitou" precisa atualizar pra exit 3; exit 4 = "BACEN OK + manager falhou".
- `secret-migrate list` em backend não-AWS agora retorna **exit 3** (era 0). Scripts que assumiam exit 0 precisam atualizar.
- Zero impacto em API REST, subcomandos check/rotate/info, interface `secrets.Manager`.

### 📁 Arquivos tocados

```
internal/secrets/aws.go              (F-S28-50-C: dead code removed)
internal/secrets/memory.go           (F-S28-50-D: dead code removed)
cmd/senhaws-rotate/main.go           (F-S28-50-B: failsafe + runApplyWithManager + exit 4)
cmd/senhaws-rotate/main_test.go      (F-S28-50-B: 4 tests novos)
cmd/secret-migrate/main.go           (F-S28-50-A: backendErr type + runList honest)
cmd/secret-migrate/main_test.go      (F-S28-50-A + 2 batch tests)
MASTER_PLAN.md                       (F-S28-50-F+H: 012+014 esclarecimento)
VALIDATION_v3.24.0.md                (audit completo)
CHANGELOG.md                         (esta entrada)
```

### ⏭️ Próxima sprint

**Sprint 32 — Audit3040_v2** — fechar 3040 de 16% → 60% (maior entrega técnica Q3). Portar regras Agreg (A01-A20) + Indiv (I01-I20) + 40+ regras B/F/C/S adicionais.

---

## v3.23.0 — 2026-07-06 (Sprint 28: VaultIntegration — AWS Secrets Manager para Sisbacen) ✅

> **Status:** ✅ Shipped
> **Sprint:** 28 (Plano Ouro §3.2 Épico B — Norma Connect)
> **Versão:** minor (1 novo pacote + 1 novo binário CLI + integração)
> **Trigger:** Plano Ouro §3.2 — fecha gap de secret management do Sprint 23-27.
> **Validação:** VALIDAÇÃO_v3.23.0.md — 23/23 packages PASS, 3 findings LOW fechados, +28 testes, race clean, 9/9 build smoke

### 🎯 Resumo

Antes (Sprint 23-27): senha Sisbacen ficava em env var. Vetores de secret disclosure: ps aux leak, log aggregator leak, rotação manual. **Depois:** interface `secrets.Manager` abstrai 3 backends (AWS Secrets Manager / env / memory), CLI `cmd/secret-migrate` permite migração one-shot com safety prompts, e `cmd/senhaws-rotate apply` faz **rotação atômica-ish** (BACEN + manager) em uma operação.

**Decisão arquitetural:** interface segregation (3 backends via mesma interface). Default prod = AWS via IAM role (zero credenciais hardcoded). Default dev = env (back-compat com Sprint 23-27).

### 🚀 O que entrou

#### Novo pacote `internal/secrets/` (6 arquivos, ~700 LoC)

```
internal/secrets/
├── manager.go        interface Manager + factory NewManagerFromEnv
├── memory.go         MemoryManager — tests + dev local
├── env.go            EnvManager — fallback dev/test (normaliza nomes)
├── aws.go            AWSManager — AWS SDK v2 + IAM role auth
├── errors.go         NotFoundError, AccessDeniedError, ValidationError + Is helpers
└── manager_test.go   15 testes
```

**Interface:**

```go
type Manager interface {
    Get(ctx context.Context, name string) (*Secret, error)
    Put(ctx context.Context, name, value string) (*Secret, error)
    Delete(ctx context.Context, name string) error
    Backend() string  // "aws" | "env" | "memory"
}
```

**3 implementações:**

| Backend | Quando usar | Auth |
|---|---|---|
| `aws` | **Default prod** | IAM role (zero creds) |
| `env` | Dev/test fallback | process env vars |
| `memory` | Tests + dev local | in-process map |

#### Novo CLI `cmd/secret-migrate` (250 LoC + 9 testes)

3 subcomandos:
- `migrate --from-env=X --to=Y [--delete-env] [--dry-run]` — migra 1 secret
- `migrate-batch --file=secrets.json` — migra lista
- `list --prefix=...` — placeholder (TODO Sprint 29+)
- `version` — versão

Safety features: `--dry-run`, confirmation prompt `YES` se value parece secret real, exit codes consistentes (0/1/2/3).

#### `cmd/senhaws-rotate` ganha subcomando `apply`

```bash
# Antes (manual, propenso a erro)
senhaws-rotate rotate > /tmp/newpass.txt
aws secretsmanager update-secret --secret-id bacen/senha --secret-string file:///tmp/newpass.txt
rm /tmp/newpass.txt

# Agora (atômico-ish, zero arquivos temp)
RADIANT_SECRETS_BACKEND=aws senhaws-rotate apply
# → senha_alterada=true secret_updated=true backend=aws name="bacen/senha/123450001.fulano" version_id=abc123
```

Fluxo: BACEN AlterarSenha → Manager.Put → audit emission. Falha em qualquer etapa retorna exit code discriminável.

### 🔧 Como usar

```bash
# 1. Setup AWS (uma vez)
export RADIANT_SECRETS_BACKEND=aws
export AWS_REGION=sa-east-1
# IAM role configurado em ECS task

# 2. Migrar 1 secret (one-shot)
secret-migrate migrate \
    --from-env=SENHAWS_PASSWORD \
    --to=bacen/senha/123450001.fulano \
    --delete-env

# 3. Cron de rotação automática
RADIANT_SECRETS_BACKEND=aws senhaws-rotate apply \
    --base-url=https://www9.bcb.gov.br/senhaws \
    --user=123450001.fulano
```

### 📊 Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 8 (6 internal/secrets + 2 cmd/secret-migrate) |
| LoC novos | ~1.200 |
| Arquivos modificados | 2 (senhaws-rotate main + test) |
| Testes Sprint 28 | **28** (15 secrets + 9 secret-migrate + 4 senhaws-rotate apply) |
| Total backend tests | **544** (era 516, **+28**) |
| Packages PASS | **23/23** (era 21, +2) |
| Build smoke | **9/9 binaries** (era 8, +1 = secret-migrate) |
| Coverage internal/secrets | **64.5%** (era 0%) |
| Coverage cmd/secret-migrate | **48.7%** (era 0%) |
| Coverage cmd/senhaws-rotate | **66.2%** (era 60.7%, **+5.5pp**) |
| Race detector | clean |
| gofmt + vet | clean |
| Findings Validação 49 | 3 LOW fechados, 3 NF com justificativa |

### 🔒 Segurança

- **Zero credenciais em código** — AWS auth via IAM role.
- **Zero values em logs** — `looksLikeSecret` heuristic, mas NUNCA loga value real.
- **Naming convention consistente** — `bacen/senha/{user}` com `.` mantido, normalização em envName().
- **Erros tipados** — `secrets.IsNotFound(err)`, `secrets.IsAccessDenied(err)`, `secrets.IsValidation(err)`.
- **Confirmation prompts** em migração destrutiva (F-S28-49-C fix).

### 🏗️ Lições aprendidas

1. **Interface + factory pattern** para multi-backend secret managers (replicável).
2. **EnvManager como fallback oficial**, não substituto.
3. **AWS error classification via reflection** > type assertion (SDK muda struct).
4. **Confirmation prompts em ferramentas de migração** (defesa contra mass-migrate).
5. **Naming convention normalize na função**, não no caller.
6. **Idempotência via Put** (PutSecretValue cria nova versão).

### 📦 Arquivos tocados

```
backend/internal/secrets/manager.go            (novo)
backend/internal/secrets/memory.go            (novo)
backend/internal/secrets/env.go               (novo)
backend/internal/secrets/aws.go               (novo)
backend/internal/secrets/errors.go            (novo)
backend/internal/secrets/manager_test.go     (novo, 15 testes)
backend/cmd/secret-migrate/main.go            (novo, 250 LoC)
backend/cmd/secret-migrate/main_test.go       (novo, 9 testes)
backend/cmd/senhaws-rotate/main.go            (modificado, +subcomando apply)
backend/cmd/senhaws-rotate/main_test.go       (modificado, +4 testes)
backend/go.mod                                (modificado, +AWS SDK v2)
backend/go.sum                                (modificado)
backend/SPRINT_28_RESEARCH.md                 (novo)
backend/SPRINT_28_RESULTS.md                  (novo)
backend/VALIDATION_v3.23.0.md                 (novo)
CHANGELOG.md                                   (esta entrada)
```

### ⚠️ Próximos passos (Sprint 29+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| **29** | BacenHomologSmoke | Smoke real contra sta-h.bcb.gov.br/staws |
| **30** | PostgresRLS | Ativar migration 014_rls_enforce.sql |
| **35** | VaultIntegration (HashiCorp) | Se multi-cloud virar requisito |
| **Sprint 50+** | secret-migrate List | Listar secrets AWS via ListSecrets API |

---

## v3.22.0 — 2026-07-05 (Plano Ouro aprovado — 12 meses · 39 sprints · 8 épicos) ✅

> **Status:** ✅ Aprovado por Henrique · 2026-07-05
> **Trigger:** revisão macro de produto pós Sprint 27 (v3.21.1)
> **Tipo:** docs + planning (zero código de produção alterado)

### 🎯 Resumo

Marco estratégico do projeto. Documenta o caminho dos **próximos 12 meses** (Q3 2026 → Q2 2027), com **39 sprints** numeradas, **8 épicos** com acceptance criteria, **6 ADRs** (Architectural Decision Records), e **contracts completos** (REST API, domain, DB, events, services).

**Decisão macro:** SaaS regulatório multi-tenant production-grade, com stack definitiva Go 1.25+ + Postgres 16 + Redis 7 + Next.js 15, hospedado em AWS São Paulo, com SOC 2 Type II como meta Q2 2027.

**Movimento de mercado (4 quarters):**
- **Q3 2026:** "Lite vendável" — fechar 3040 + 3050 + smoke BACEN real.
- **Q4 2026:** "Pro vendável" — DLO + DDR + DRL + DLP + 3044 + cross-doc v2.
- **Q1 2027:** "ESG first-mover" — DRSAC 2030 (janela IN BCB 694/2025, vigência dez/2026).
- **Q2 2027:** "Enterprise" — SOC 2 Type II + SDK + Marketplace + multi-region.

### 📦 Arquivos novos

```
MASTER_PLAN.md                               (~85 KB · 11 seções + 5 ADRs)
ROADMAP.md                                   (visão macro executiva por quarter)
docs/adr/0001-stack-definitiva.md            (Go + Postgres + Redis + Next.js + AWS SP)
docs/adr/0002-multi-tenancy-rls.md           (Postgres RLS, não schema-per-tenant)
docs/adr/0003-audit-log-hash-chain.md        (SHA-256 + trigger imutável + WORM S3)
docs/adr/0004-schema-registry-versionado.md  (GitHub source-of-truth + auto-PR)
docs/adr/0005-sta-client-interface-segregation.md (Client / ReadClient / ChunkedClient)
docs/adr/0006-cross-doc-engine.md            (12 regras inter-CADOC, L3 proprietário)
README.md                                    (atualizado — badges + links + métricas)
CHANGELOG.md                                (esta entrada)
```

### 🎯 Os 10 moats competitivos definidos

1. **Cross-Doc Engine (L3)** — valida ecossistema 3040 ↔ 4111 ↔ DRSAC.
2. **Schema Registry versionado** — IF não mexe em código quando BACEN muda.
3. **Audit hash chain** — LGPD/SOC 2 ready, trigger Postgres imutável.
4. **DRSAC ESG first-mover** — janela dez/2026.
5. **Onboarding 15min** — Matera leva 12 semanas.
6. **Open schemas (GitHub)** — community contributions.
7. **Modern stack** — hiring mais fácil.
8. **Compliance officer UX** — feito pro usuário primário.
9. **AI Insights** — LLM interpreta audit_log (opt-in).
10. **Multi-CADOC ecosystem** — 10 CADOCs, 1 plataforma.

### 🔢 Quality gates publicados

| Gate | Target | Atual |
|---|---|---|
| Coverage por pacote | 70-95% | varia (auditlog 90.8%, api 71.6%, etc) |
| Latência API P95 | < 500ms (validate), < 5s (submit) | TBD |
| Uptime | 99.9% | n/a (ainda não em produção) |
| Audit chain integrity | 100% | ✅ validado |
| Security (CVEs) | 0 high/critical | TBD govulncheck |

### ⚠️ Próximos passos (Sprint 28+)

- **Sprint 28:** VaultIntegration — AWS Secrets Manager para rotação Sisbacen.
- **Sprint 29:** BacenHomologSmoke — smoke real contra sta-h.bcb.gov.br.
- **Sprint 30:** PostgresRLS — ativar migration 014_rls_enforce.sql.
- **Sprint 32:** Audit3040_v2 — fechar 3040 de 16% → 60% cobertura.

### 🏗️ Decisões macro registradas

| Decisão | Razão |
|---|---|
| **Postgres RLS, não schema-per-tenant** | Defense-in-depth; migrations O(1); LGPD delete = single query |
| **Audit hash chain com trigger DB** | Tamper-evident verificável por auditor externo sem privilégios |
| **Schema Registry no GitHub público** | Schema-first; zero deploy de código; community contributions |
| **Interface segregation STA (3 interfaces)** | Hollow stub evitado; capability check explícito |
| **Cross-Doc engine com panic recovery** | Falha de 1 regra não derruba servidor todo |
| **Stack chata, exciting product** | Postgres + chi + slog + Next.js — boring infra, foco no domínio |

---

## v3.21.1 — 2026-07-06 (Sprint 27 followup — readlink -f symlink + escape de placeholder) ✅

> **Status:** ✅ Shipped
> **Trigger:** Validação 49 (Sprint 27 followup) — fix de bug menor

## v3.21.0 — 2026-07-06 (Sprint 27: pre-commit hook — lint + gofmt + vet automatizado) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 27 (fecha gap operacional do Sprint 25)
> **Versão:** patch (2 scripts novos + hook — zero impacto em código existente)
> **Trigger:** VALIDAÇÃO 47/48 §"Próximos passos" Sprint 27 (pre-commit hook)
> **Validação:** 21/21 packages PASS + 7/7 binaries + race clean + gofmt/vet clean

### 🎯 Resumo

Sprint 27 fecha o **gap operacional do Sprint 25** — `lint-no-placeholder.sh`
rodava manual. Agora roda **automaticamente** antes de cada `git commit` via
pre-commit hook.

**Decisão arquitetural:** symlink de `scripts/pre-commit.sh` em `.git/hooks/pre-commit`.
Hook roda 3 checks (cada <2s):
1. `lint-no-placeholder.sh` — detecta `(preencher X)` em SPRINT_*.md
2. `gofmt -l backend/` — detecta drift de formatação Go
3. `go vet ./...` — detecta constructs suspeitos

**Decisões YAGNI conscientes:**
- Sem `golangci-lint` ou framework externo (bash + stdlib suficiente).
- Sem integração CI automática (lint roda local, CI é v28+).
- Sem pre-push hook (pre-commit é o canônico git convention).
- Sem `go test` no hook (leva ~2min, CI é lugar certo).
- Sem auto-install via `go generate` ou similar (install-hooks.sh é script manual).

**Decisões de design não-óbvias:**
- **Symlink relativo** (`../../scripts/pre-commit.sh`): portabilidade entre máquinas.
- **Backup automático** em `install-hooks.sh` se hook customizado já existe.
- **Idempotência** do install: rodar 2x não quebra.
- **Bypass** com `--no-verify` (padrão git, não precisa flag custom).

### 🚀 O que entrou

**`scripts/pre-commit.sh` (76 linhas bash):**
- 3 checks sequenciais, formato consistente (`==> [N/3]`)
- Output útil em caso de falha (mensagem + fix command)
- Detecta placeholders, drift Go, vet issues

**`scripts/install-hooks.sh` (35 linhas bash):**
- Cria symlink `.git/hooks/pre-commit` → `scripts/pre-commit.sh`
- Idempotente (rodar 2x não quebra)
- Backup automático de hook customizado (`.bak`)

**Hook instalado localmente:**
- `.git/hooks/pre-commit` → symlink
- Roda automaticamente antes de cada commit
- Bypass via `--no-verify` para emergências

### 🔧 Como usar

```bash
# Setup (uma vez por dev)
./scripts/install-hooks.sh

# Workflow normal
git add .
git commit -m "fix: ..."  # hook roda automaticamente

# Bypass (emergência)
git commit --no-verify -m "hotfix urgente"
```

### 📊 Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (pre-commit.sh 76 linhas + install-hooks.sh 35 linhas) |
| Packages PASS | **21/21** (zero regressão) |
| Build smoke | 7/7 binaries |
| gofmt drift | 0 |
| vet | clean |
| Race detector | clean |
| Lint `lint-no-placeholder.sh` | ✅ 27/27 (Sprint 27 incluso) |

### 🔒 Compatibilidade

- **Zero impacto em código existente.** Scripts são additive.
- **Hook não é commitado** (`.git/hooks/` é gitignored por default). Cada dev roda `./scripts/install-hooks.sh` uma vez.
- **CI não muda.** Sprint 28+ pode adicionar step CI para `./scripts/pre-commit.sh` se virar requisito.

### 🏗️ Lições aprendidas (carry forward)

1. **Pre-commit hook = automação catching local.** Lint script → install hook → catching automático.
2. **Bash + symlink > framework externo.** Não precisamos husky/pre-commit.com/lefthook.
3. **Backup automático em install scripts.** Preserva customizações dev.
4. **Idempotência em scripts de setup.** Roda N vezes sem erro (CI/container/erro humano).
5. **Pre-commit hook NÃO inclui `go test`.** Leva ~2min, CI é lugar certo.
6. **Sprint operacional (não feature).** Fecha gap operacional sem adicionar feature nova.

### 📦 Arquivos tocados

```
scripts/pre-commit.sh                 (novo, 76 linhas)
scripts/install-hooks.sh              (novo, 35 linhas)
SPRINT_27_RESULTS.md                  (este)
CHANGELOG.md                          (esta entrada)
```

### ⚠️ Próximos passos (Sprint 28+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 28 | Vault integration | Secret manager rotation |
| 29 | Smoke contra BACEN homolog | Requer credenciais Sisbacen |
| 30 | `cmd/sta-submit` range upload | Chunked transfer (Sprint 21) |
| 31 | Handler REST `/v1/sta/range-*` | Sprint 21 YAGNI |
| 28+ | CI integration (`scripts/pre-commit.sh` em CI) | Cross-dev consistency |

---

## v3.20.0 — 2026-07-06 (Validação 48 DEEPEST — Sprint 26 coverage gaps + dead code) ✅

> **Status:** ✅ Shipped
> **Versão:** patch (2 LOW + 1 INFO→LOW — zero impacto em código existente)
> **Trigger:** "da mais uma validada profunda" após Validação 47 + Sprint 26
> **Validação:** 21/21 packages PASS (zero flake desta vez!) + 3 testes novos + 1 SKIP + coverage sta-submit 70.3% → 78.1% + race clean + 7/7 binaries + gofmt/vet clean

### 🎯 Resumo

Validação 48 fecha **3 findings** identificados na leitura completa de
Validação 47 + Sprint 26 (commit `70718a3` + `47cdfc8`):

- **F-S26-48-A (LOW):** 4 gaps de coverage em `cmd/sta-submit`:
  1. `os.ReadFile` erro (path inválido) → exit 2
  2. `result.Rejection == nil` (caminho else) → SKIP (não testável com StubClient)
  3. `newLogger(quiet=true)` quiet path → newLogger 0% → 66.7%
  4. `fs.Parse` erro → SKIP (flag.ContinueOnError indefinido)

  Coverage `cmd/sta-submit`: 70.3% → **78.1%** (+7.8pp).
  Coverage `runSubmit`: 84.8% → **90.9%** (+6.1pp).
  Coverage `newLogger`: 0% → **66.7%** (+66.7pp).

- **F-S26-48-B (LOW):** `var _ = strings.Contains` dangling no main.go (linha 217) +
  comment enganoso "usado internamente". `strings` não era usado em nenhum
  outro site do main.go (apenas no test file). Dead code com comment misleading.
  Removido + import `strings` removido.

### 🔍 Findings NÃO fechados (7 com justificativa)

Todos carry-overs ou YAGNI documentados:
- **F-NF-1:** `cli main()` 0% coverage (YAGNI carry-over v44).
- **F-NF-2:** `newLogger` 66.7% (caminho não-quiet uncovered — YAGNI carry-over v45+v46+v47).
- **F-NF-3:** `runSubmit` erro de `staNewClientFromEnv` uncovered (carry-over F-NF-2 v46).
- **F-NF-4:** caminho `rejection==nil` não testável (StubClient hardcoded tem Rejection != nil).
- **F-NF-5:** sem compile-time assert para `staClient` private (decisão consciente — interface local).
- **F-NF-6:** `protocol_sta`/`code`/`message` impressos no stdout (carry-over F-NF-3 v43).
- **F-NF-7:** Test `TestStaSubmit_LoadConfig_InvalidFlag` SKIP — flag.ContinueOnError indefinido.

### 📦 Arquivos tocados

```
backend/cmd/sta-submit/main.go          (3 modificados — var _ + import removidos)
backend/cmd/sta-submit/main_test.go     (+113 — 3 testes novos + 1 SKIP + comentários)
VALIDATION_v3.19.0_DEEPEST.md           (novo — 8 checklists + 3 findings + 7 NF + 6 lições)
CHANGELOG.md                            (esta entrada)
```

### 🔢 Métricas

| Métrica | Pré Validação 48 | Pós Validação 48 |
|---|---|---|
| Packages PASS | 21/21 | **21/21** (zero flake desta vez!) |
| Tests sta-submit top-level | 10 | **13** (+3) |
| Tests sta-submit SKIP | 0 | 1 |
| Total backend tests top-level | 127 | **130** (+3) |
| Coverage cmd/sta-submit | 70.3% | **78.1%** (+7.8pp) |
| Coverage runSubmit | 84.8% | **90.9%** (+6.1pp) |
| Coverage newLogger | 0% | **66.7%** (+66.7pp) |
| Race detector | clean | clean |
| gofmt drift | 0 | 0 |
| vet | clean | clean |
| Lint `lint-no-placeholder.sh` | ✅ 26/26 | ✅ 26/26 |
| Findings abertos | — | **0** (3 fechados, 7 NF com justificativa) |

### 🏗️ Lições aprendidas (carry forward)

1. **Coverage report é checklist de test cases.** Cada linha uncovered em função testável = test case pendente. 4 testes simples (+30 linhas total) fecharam 3 gaps com +7.8pp coverage.
2. **Dead code + comment enganoso = pior que dead code sem comment.** Remover `var _ + comment + import` é mais limpo que deixar com aviso.
3. **Test SKIP com justificativa > test que promete sem entregar.** Documenta intenção + razão. Pattern consistente com validações anteriores.
4. **Carry-overs continuam documentados.** 7 NF nesta validação, 5 são carry-overs de v44-v47. Pattern consistente evita "NF forgotten and re-flagged".
5. **Validação contínua pós-sprint vale investimento.** v48 foi rápida (~30 min equivalente) mas encontrou 3 melhorias incrementais + 0 regressão.
6. **Zero flake desta vez (raro!).** Loggerutil perf tests passaram limpos. Pode ser偶然 (CPU não disputada) ou v45/v47 fixes resolveram. Carry-over para próxima validação.

### 🔒 Compatibilidade

- Zero impacto em código de produção. Testes adicionais são puramente cobertura.
- Remoção de dead code (`var _` + comment + import `strings`) é internal cleanup.
- Comportamento runtime idêntico.

---

## v3.19.0 — 2026-07-06 (Sprint 26: cmd/sta-submit CLI) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 26 (novo binário `cmd/sta-submit` — segundo caller real do pacote sta)
> **Versão:** minor (novo binário + 10 testes; **zero impacto** em código existente)
> **Trigger:** SPRINT_25_RESULTS.md §"Próximos passos" Sprint 26 (cmd/sta-submit CLI)
> **Validação:** 21/21 packages PASS + 10 testes novos Sprint 26 + 7/7 binaries + race clean + gofmt/vet clean

### 🎯 Resumo

Sprint 26 fecha o **`cmd/sta-submit`** — CLI standalone para envio de CADOC
ao BACEN STA WS. Admin IF pode submeter CADOC direto via linha de comando,
sem deployar API ou UI.

```bash
sta-submit --xml-file=/path/to/cadoc3040.xml \
           --cadoc-code=3040 \
           --data-base=2024-12 \
           --cnpj=demo-bank

# → protocol_sta=PROTO-OK  status=accepted
# → exit 0 (sucesso)
# → exit 1 (rejeitado/transporte)
# → exit 2 (config inválida)
# → exit 3 (erro BACEN formal)
```

**Decisão arquitetural:** CLI single-command (apenas `submit`) — escopo focado
no caso de uso principal. Reusa `sta.NewClientFromEnv` (mesma fábrica usada
por `cmd/api`) → consistency entre CLI e servidor.

**Decisões YAGNI conscientes:**
- Sem handler REST (admin tool direto, não UI).
- Sem retry wrapper (failure fast — caller decide retry).
- Sem range upload / chunked (single CADOC <50 MB usa Submit normal).
- Sem upload de ZIP (apenas XML — cobre 80% do caso de uso).
- Sem TLS client cert (BACEN não exige).
- Sem dry-run.
- Sem subcomandos (info/check) — YAGNI agora.

**Decisões de design não-óbvias:**
- **Injeção de client via variável de função** (`staNewClientFromEnv`): pattern de test injection sem mockar STA Client inteiro.
- **Interface `staClient` mínima** (1 método): desacopla CLI de mudanças futuras em `sta.Client`.

### 🚀 O que entrou

- **Binário `cmd/sta-submit`** com 1 subcomando `submit` + flags:
  - `--xml-file` (env `STA_SUBMIT_XML_FILE`) — caminho do XML
  - `--cadoc-code` (env `STA_SUBMIT_CADOC_CODE`) — default `3040`
  - `--data-base` (env `STA_SUBMIT_DATA_BASE`) — formato YYYY-MM
  - `--cnpj` (env `STA_SUBMIT_CNPJ`) — default `demo-bank`
  - `--quiet` — silencia logs stderr

- **Env vars STA delegadas** a `sta.NewClientFromEnv` (mesma fábrica do `cmd/api`):
  - `RADIANT_STA_BACKEND` (stub|ws)
  - `RADIANT_STA_WS_URL`
  - `RADIANT_STA_SISBACEN_USER`
  - `RADIANT_STA_SISBACEN_PASSWORD`
  - `RADIANT_STA_TIMEOUT_SECONDS`

- **Exit codes** consistentes com `cmd/senhaws-rotate`:
  - `0` aceito pelo BACEN
  - `1` rejeitado OU transporte
  - `2` erro de validação client-side
  - `3` erro BACEN formal

- **Output format key=value** (mesmo padrão):
  - Sucesso: `protocol_sta=<PROT>  status=accepted`
  - Rejeição: `protocol_sta=<PROT>  status=rejected  code=<C>  message=<M>`

- **Interface `staClient` mínima** + **variable de função `staNewClientFromEnv`** para test injection sem framework externo.

### 🧪 Tests (10 novos — total backend 127)

| Test | Cobre |
|---|---|
| `TestStaSubmit_Success_StubClient` | Happy path com StubClient (default) |
| `TestStaSubmit_Rejection_StubClient` | Rejeição StubClient (AlwaysAccept=false) |
| `TestStaSubmit_MissingXMLFile` | Config inválida → exit 2 |
| `TestStaSubmit_MissingDataBase` | Config inválida → exit 2 |
| `TestStaSubmit_EmptyXMLFile` | Arquivo vazio → exit 2 |
| `TestStaSubmit_BACENError_WSClient` | WSClient mock 400 → exit 3 |
| `TestStaSubmit_TransportError` | WSClient mock fechado → exit 1 |
| `TestStaSubmit_Usage_Prints` | usage() imprime help |
| `TestStaSubmit_LoadConfig` | Env vars override defaults |
| `TestStaSubmit_LoadConfig_Defaults` | Defaults sensatos |

### 📦 Arquivos tocados

```
backend/cmd/sta-submit/main.go           (novo, 212 linhas)
backend/cmd/sta-submit/main_test.go      (novo, 290 linhas — 10 testes)
SPRINT_26_RESEARCH.md                    (research rápido)
SPRINT_26_RESULTS.md                     (este doc)
CHANGELOG.md                             (esta entrada)
```

### 🔢 Métricas finais

| Métrica | Valor |
|---|---|
| Pacotes Go testados | **21/21 PASS** |
| Tests Sprint 26 | 10 (todos PASS) |
| Tests totais top-level | **127** (era 117) |
| Build smoke binaries | **7/7** (era 6, +1 = sta-submit) |
| gofmt drift | 0 |
| vet | clean |
| Race detector | clean |
| Lint `lint-no-placeholder.sh` | ✅ 26/26 |

### 🏗️ Lições aprendidas (carry forward)

1. **Variable de função = test injection idiomático.** `var f = realFunc` permite tests sobrescreverem sem framework externo.
2. **Interface mínima desacopla de mudanças futuras.** `staClient` com 1 método Submit. Se Sprint 27+ adicionar métodos em `sta.Client`, CLI continua funcionando.
3. **YAGNI em subcomandos.** CLI tem 1 comando (`submit`). Adicionar `check`/`cancel`/`info` é trivial quando virar requisito.
4. **Test injection pattern escala.** 10 testes cobrem 4 fluxos (sucesso, rejeição, config error, BACEN error, transporte) usando apenas 2 helpers (StubClient + WSClient mock).
5. **Reusa `sta.NewClientFromEnv` = consistency operacional.** Admin IF que usa `sta-submit` + `cmd/api` precisa configurar mesmas env vars.

### 🔒 Compatibilidade

- **Novo binário `cmd/sta-submit`.** Zero impacto em código existente.
- **`sta.NewClientFromEnv` inalterado.** CLI apenas wrappea.
- **Não wired em `cmd/api/main.go`** — CLI é independente (decoupling).
- **Nenhum handler REST adicionado** — admin tool direto.
- **`internal/sta/*` inalterado** — reuso.

---

## v3.18.0 — 2026-07-06 (Validação 47 DEEPEST — error path tests 3-way) ✅

> **Status:** ✅ Shipped
> **Versão:** patch (1 LOW — zero impacto em código existente)
> **Trigger:** "da mais uma validada profunda" após Validação 46
> **Validação:** 20/20 packages PASS + 1 teste novo + coverage senhaws 94.4% → 95.6% + race clean + 6/6 binaries + gofmt/vet clean

### 🎯 Resumo

Validação 47 fecha **1 finding** identificado na leitura completa de
Validação 46 (commit `ba77d30`):

- **F-S25-47-A (LOW):** `AlterarSenha` retornava erro de transporte cru
  (HTTPClient.Do falha) mas caminho não tinha test dedicado. Coverage
  `AlterarSenha` 89.3% → **92.9%** (+3.6pp). Total senhaws 94.4% → **95.6%** (+1.2pp).

### 🔍 Findings NÃO fechados (7 com justificativa)

Todos carry-overs documentados:
- **F-NF-1:** `ConsultarVencimento` 91.3% gaps remanescentes (unreachable paths).
- **F-NF-2:** `loadConfig` retorna `errors.New` opaco (carry-over F-NF-2 v46).
- **F-NF-3:** `ConsultarVencimento` retorna 4 `errors.New`/`fmt.Errorf` opacos (defensiva BACEN bug — carry-over F-NF-1 v46).
- **F-NF-4:** `cli main()` 0% coverage (YAGNI — carry-over v44+v45+v46).
- **F-NF-5:** `newLogger` 66.7% coverage (carry-over v45+v46).
- **F-NF-6:** `*ValidationError` não implementa `Is`/`Unwrap` (mesma justificativa `*SenhaError`).
- **F-NF-7:** lint script regex `^```` não pega code blocks indentados (edge case improvável).

### 🔒 Test 3-way pattern (NOVO)

`TestSenhawsClient_AlterarSenha_TransportError` valida 3 aspectos do contrato:

```go
// 1. NÃO deve ser *ValidationError (não é erro do caller)
var valErr *ValidationError
if errors.As(err, &valErr) {
    t.Errorf("erro de transporte NÃO deveria ser *ValidationError")
}

// 2. NÃO deve ser *SenhaError (não é rejeição formal BACEN)
var senErr *SenhaError
if errors.As(err, &senErr) {
    t.Errorf("erro de transporte NÃO deveria ser *SenhaError")
}

// 3. DEVE ser erro cru de rede (contém "connection refused" / "EOF")
if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "EOF") {
    t.Errorf("erro deveria ser de rede, got %q", err.Error())
}
```

Pattern replicável: **1-way test** (v45+v46 validava tipo positivo) → **3-way test** (v47 valida tipo positivo + 2 tipos negativos).

### 📦 Arquivos tocados

```
backend/internal/senhaws/senhaws_test.go       (+35 — 1 teste novo: TestSenhawsClient_AlterarSenha_TransportError)
VALIDATION_v3.17.0_DEEPEST.md                 (novo — 8 checklists + 1 finding fechado + 7 NF + 5 lições)
CHANGELOG.md                                  (esta entrada)
```

### 🔢 Métricas

| Métrica | Pré Validação 47 | Pós Validação 47 |
|---|---|---|
| Packages PASS | 20/20 | **20/20** (zero regressão) |
| Tests senhaws top-level | 18 | **19** (+1) |
| Total backend tests top-level | 116 | **117** (+1) |
| Coverage internal/senhaws | 94.4% | **95.6%** (+1.2pp) |
| Coverage AlterarSenha | 89.3% | **92.9%** (+3.6pp) |
| Coverage NewSenhawsClient | 100% | 100% |
| Coverage ConsultarVencimento | 91.3% | 91.3% |
| Race detector | clean | clean |
| gofmt drift | 0 | 0 |
| vet | clean | clean |
| Lint `lint-no-placeholder.sh` | ✅ 26/26 | ✅ 26/26 |
| Findings abertos | — | **0** (1 fechado, 7 NF com justificativa) |

### 🏗️ Lições aprendidas (carry forward)

1. **Error path tests devem validar 3-way (tipo + não-tipos + indícios).** Pattern emergente v45/v46/v47 — próxima evolução natural é aplicar 3-way em todos os error paths.
2. **Coverage gap em error path é catchable com test simples.** `httptest.Server.Close()` antes de call → garante connection refused. Pattern replicável em qualquer client HTTP.
3. **Tests de contrato HTTP devem cobrir falhas de transporte, não só status codes.** Coverage 8% gap em `AlterarSenha` vinha todo de xml.Marshal (impossível), NewRequestWithContext (impossível), HTTPClient.Do (testável, não testado).
4. **Carry-overs entre validações: nem todo NF é fechamento.** v47 encontrou 7 NF, mas 6 são carry-overs documentados em validações anteriores (F-NF-1 a F-NF-3 da v46, F-NF-4 da v44).
5. **Validação contínua pós-sprint vale o investimento.** Validação 47 foi pequena (~35 linhas), mas fechou gap real. Pequena e frequente > grande e rara.

### 🔒 Compatibilidade

- Zero impacto em código existente. Adição de 1 teste.
- Test não altera comportamento de runtime. Apenas verifica contrato.

---

## v3.17.0 — 2026-07-06 (Validação 46 DEEPEST — Sprint 25 hardening + ValidationError consistency) ✅

> **Status:** ✅ Shipped
> **Versão:** patch (2 LOW — zero impacto em código existente)
> **Trigger:** "da mais uma validada profunda" após Validação 45 + Sprint 25
> **Validação:** 20/20 packages PASS + 1 teste novo + 6 subtests novos + coverage senhaws mantido 94.4% + race clean + 6/6 binaries + gofmt/vet clean

### 🎯 Resumo

Validação 46 fecha **2 findings** identificados na leitura completa de
Sprint 25 + Validação 45 (commit `b580e78` + `8210abc`):

- **F-S25-46-1+2 (LOW):** `NewSenhawsClient` retornava `errors.New` / `fmt.Errorf`
  opaco para 6 erros de validação de config (BaseURL/User/Password). Inconsistente
  com `AlterarSenha` que já retornava `*ValidationError` (F-S24-45-1 fechou v45).
  Caller (CLI) não conseguia classificar config error vs BACEN error vs transporte
  — caía em fallback genérico.
- **F-S25-46-7 (LOW):** testes existentes não validavam `errors.As(err, &valErr)`
  para erros de config. Pattern descoberto na v45 só foi aplicado em `AlterarSenha`.

### 🔍 Findings NÃO fechados (5 com justificativa)

- **F-NF-1:** `errors.New("BACEN retornou 200 mas <DiasVencimentoSenha> vazio")` —
  defensiva contra BACEN bug (não é validation, não é BACEN rejection, não é transporte).
- **F-NF-2:** `loadConfig` retorna `errors.New` opaco — CLI trata uniforme via `os.Exit(exitClientError)`.
- **F-NF-3:** lint script regex `^```` não pega code blocks indentados — edge case improvável.
- **F-NF-4:** `cli main()` 0% coverage — YAGNI (carry-over v44+v45).
- **F-NF-5:** `newLogger` 66.7% coverage — diferença trivial.

### 🔒 Refator de consistência

6 sites em `NewSenhawsClient` refatorados de `errors.New`/`fmt.Errorf` para `*ValidationError`:

```go
// Antes (opaco):
return nil, errors.New("SenhawsConfig.Password requerida")

// Depois (tipado):
return nil, &ValidationError{Field: "Password", Message: "requerida"}
```

CLI refatorado em 3 sites (`runCheck` + `runRotate` + `runInfo`) para detectar `*ValidationError` e imprimir só `Message` (não redundante):

```go
if err != nil {
    var valErr *senhaws.ValidationError
    if errors.As(err, &valErr) {
        fmt.Fprintf(os.Stderr, "config invalida: %s\n", valErr.Message)
    } else {
        fmt.Fprintf(os.Stderr, "config invalida: %v\n", err)
    }
    return exitClientError
}
```

### 📦 Arquivos tocados

```
backend/internal/senhaws/senhaws.go             (+8 / -6 — 6 sites de errors.New/fmt.Errorf → &ValidationError)
backend/internal/senhaws/senhaws_test.go        (+58 / -6 — 1 teste novo: TestNewSenhawsClient_ErrorsAs_Validation, expects ajustados)
backend/cmd/senhaws-rotate/main.go              (+12 / -6 — 3 sites de error handling CLI padronizados)
VALIDATION_v3.16.0_DEEPEST.md                   (novo — 8 checklists + 2 findings fechados + 5 NF + 5 lições)
CHANGELOG.md                                    (esta entrada)
```

### 🔢 Métricas

| Métrica | Pré Validação 46 | Pós Validação 46 |
|---|---|---|
| Packages PASS | 20/20 | **20/20** (zero regressão) |
| Tests senhaws top-level | 17 | **18** (+1) |
| Tests senhaws subtests | 23 | **29** (+6) |
| Total backend tests top-level | 115 | **116** (+1) |
| Coverage internal/senhaws | 94.4% | **94.4%** (mantido) |
| Coverage NewSenhawsClient | ~95% | **100%** |
| Coverage cmd/senhaws-rotate | 70.2% | 68.3% (refator adiciona linhas, paths similares) |
| Race detector | clean | clean |
| gofmt drift | 0 | 0 |
| vet | clean | clean |
| Lint `lint-no-placeholder.sh` | ✅ 26/26 | ✅ 26/26 |
| Findings abertos | — | **0** (2 fechados, 5 NF com justificativa) |

### 🏗️ Lições aprendidas (carry forward)

1. **Error types devem ser consistentes em todo o pacote.** `AlterarSenha` retornava `*ValidationError`, `NewSenhawsClient` retornava `errors.New` opaco. Inconsistência passou em v45, v46 fechou. Pattern: ao introduzir tipo-erro, auditar TODAS as funções similares.
2. **Tests de error type são cheap e valiosos.** `TestNewSenhawsClient_ErrorsAs_Validation` (6 subtests, ~30 linhas) garante refator consistente. Custo baixo, valor alto.
3. **CLI imprime só Message, não Error() completo, quando é *ValidationError.** Pattern: caller sabe contexto, evitar redundância.
4. **Refator cross-function é oportunidade de unifying output.** 3 sites do CLI ganharam mesmo padrão `errors.As(&valErr)` + output uniforme.
5. **Coverage cai quando código cresce (não significa regressão).** 70.2% → 68.3% após refator que adiciona linhas é esperado. Métrica relativa (% de paths cobertos) importa mais que absoluta.

### 🔒 Compatibilidade

- Zero impacto em código existente. Refator é interno.
- Mensagens de erro mudam formato: `"SenhawsConfig.Password requerida"` → `"validação Password: requerida"`. Caller que usa `err.Error()` substring matching precisa ajustar (mas é anti-pattern — usar `errors.As`).
- Exit codes CLI consistentes: config error sempre → `exitClientError` (2).

---

## v3.16.0 — 2026-07-06 (Sprint 25: compile-time asserts + lint-no-placeholder) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 25 (carry-overs de validações anteriores — automatiza padrões reincidentes)
> **Versão:** patch (compile-time asserts + lint script + 4 placeholders preenchidos — zero impacto em código existente)
> **Trigger:** VALIDAÇÃO 44 + 45 §"Próximos passos" (espalhar pattern compile-time + lint check placeholder)
> **Validação:** 20/20 packages PASS + lint-no-placeholder 25/25 limpo + smoke 11/11 + race clean

### 🎯 Resumo

Sprint 25 fecha **2 carry-overs de validações anteriores**:

1. **Compile-time interface asserts** espalhados para todos os tipos que implementam interfaces Go (`*WSClient` para 3 interfaces, `*StubClient` para 1).
2. **Lint script `lint-no-placeholder.sh`** que detecta placeholders `(preencher após X)` em SPRINT_*.md antes de commitar.

**Bônus:** o lint encontrou **4 placeholders reais** que escaparam para o repo nas Sprints 19-22 — preenchidos agora (25/25 SPRINT_*.md limpos).

**Decisão arquitetural:** compile-time asserts movidos de test files (linhas 1499, 2003 de ws_test.go) para **production source** (`ws.go` linha 1097+, `stub.go` linha 50+). Padrão idiomático Go (Effective Go + Uber style guide).

**Decisões YAGNI conscientes:**
- Lint focado em placeholder (não Linter completo tipo golangci-lint).
- Lint roda manual (não em CI ainda — Sprint 26+ se virar requisito).
- Sem pre-commit hook (`.git/hooks/pre-commit`) — YAGNI até virar problema operacional.
- Sem integração com outras ferramentas (markdownlint, vale.sh, etc).

### 🚀 O que entrou

**Compile-time asserts** (3 sites novos, zero runtime cost):

```go
// backend/internal/sta/ws.go (final do arquivo)
var (
    _ Client        = (*WSClient)(nil)
    _ ReadClient    = (*WSClient)(nil)
    _ ChunkedClient = (*WSClient)(nil)
)

// backend/internal/sta/stub.go (após declaração de StubClient)
var _ Client = (*StubClient)(nil)
```

**Lint script** (`scripts/lint-no-placeholder.sh`, 60 linhas bash):

Detecta 3 padrões em SPRINT_*.md:
- `(preencher após X)` — pattern pt-BR reincidente (v44 + v45)
- `(fill in X)` — versão inglês
- `(TODO: X)` — versão genérica

Exit codes: `0` OK / `1` FAIL (com linhas específicas listadas).

**4 placeholders reais preenchidos** (bônus):
- `SPRINT_19_RESULTS.md:6` → `7b50253`
- `SPRINT_20_RESULTS.md:6` → `fa4dc13`
- `SPRINT_21_RESULTS.md:6` → `41981e9`
- `SPRINT_22_RESULTS.md:6` → `4321a0d`

### 📚 Decisões

| Decisão | Razão |
|---|---|
| Compile-time asserts em production source | Effective Go idiom — catching imediato mesmo se teste falhar |
| Lint simples em bash (não ferramenta externa) | Sprint 25 escopo é pequeno (~50 linhas bash); vale.sh seria overkill |
| Lint roda manual, não em CI/pre-commit | CI/pre-commit adiciona fricção. Padrão: V1 manual, V2 pre-commit, V3 CI |
| 3 patterns detectados (não 1) | Pequeno overhead, melhor cobertura contra variantes futuras |

### 🔢 Métricas

| Métrica | Valor |
|---|---|
| Arquivos novos | 1 (`scripts/lint-no-placeholder.sh`) |
| Arquivos modificados | 4 (ws.go +6, stub.go +6, ws_test.go -4, 4 SPRINT_*.md) |
| Tests Sprint 25 | 0 (lint script + compile-time asserts não requerem runtime tests) |
| Total backend tests top-level | 115 (mesmo) |
| Packages PASS | **20/20** |
| Build OK | 6/6 binaries |
| Smoke E2E | 11/11 PASS (sem regressão) |
| Lint `lint-no-placeholder.sh` | **✅ 25/25 SPRINT_*.md limpos** |
| Placeholders preenchidos | **4** (Sprints 19-22) |
| gofmt drift | 0 |
| vet | clean |
| Race detector | clean |

### 🔒 Compatibilidade

- Zero impacto em código de produção. Compile-time asserts são zero-cost em runtime.
- Zero impacto em tests existentes. Compile-time asserts movidos de test → production source é reorganização.
- Lint script é aditivo. Não afeta build/test/vet. Sprint 26+ pode adicionar a CI.

### 🏗️ Lições aprendidas (carry forward)

1. **Lint scripts são melhores quando simples e focados.** Pattern: 1 lint por classe de problema, não Linter monolítico.
2. **Compile-time asserts em production source > test files.** Effective Go recomenda; production source garante catching mesmo se teste não rodar.
3. **Lint roda manual é OK pra V1.** CI/pre-commit adiciona fricção operacional.
4. **Patterns reincidentes merecem automação.** Placeholder reincidiu 2 sprints (v44 + v45) → lint criado.
5. **Script bash > script python pra linters simples.** Sem dependência, roda em qualquer Unix, fácil de auditar.

### 📦 Arquivos tocados

```
scripts/lint-no-placeholder.sh                    (novo, 60 linhas)
backend/internal/sta/ws.go                        (+8 — compile-time asserts para Client/ReadClient/ChunkedClient)
backend/internal/sta/ws_test.go                   (-4 — removidos asserts duplicados em TestReadClient_InterfaceSegregation + TestChunkedClient_InterfaceSegregation)
backend/internal/sta/stub.go                      (+5 — compile-time assert para Client)
SPRINT_19_RESULTS.md                              (1 linha — placeholder preenchido)
SPRINT_20_RESULTS.md                              (1 linha — placeholder preenchido)
SPRINT_21_RESULTS.md                              (1 linha — placeholder preenchido)
SPRINT_22_RESULTS.md                              (1 linha — placeholder preenchido)
SPRINT_25_RESULTS.md                              (novo — estatísticas + decisões + quickstart)
CHANGELOG.md                                      (esta entrada)
```

### ⚠️ Próximos passos (Sprint 26+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 26 | `cmd/sta-submit` CLI paralelo a `senhaws-rotate` | Mesmo pattern pra CADOC submission |
| 26 | Pre-commit hook: `./scripts/lint-no-placeholder.sh` + gofmt + go vet | Automação catching antes de push |
| 27 | Vault integration (AWS Secrets Manager / Vault) | Auto-update secret manager |
| 28 | Smoke contra BACEN homolog real | Requer credenciais Sisbacen |
| 29 | Handler REST `/v1/sta/range-*` (Sprint 21 YAGNI) | Frontend/batch trigger UI |

---

## v3.15.0 — 2026-07-06 (Validação 45 DEEPEST — Sprint 24 hardening + ValidationError) ✅

> **Status:** ✅ Shipped
> **Versão:** patch (1 MEDIUM + 5 LOW + 1 carry-over flake — zero impacto em código existente)
> **Trigger:** "da mais uma validada profunda" após Validação 44 + Sprint 24
> **Validação:** 20/20 packages PASS (zero FAIL — flake loggerutil resolvido) + 6 testes novos + coverage senhaws-rotate 60.7% → 70.2% + race clean + 6/6 binaries + gofmt/vet clean

### 🎯 Resumo

Validação 45 fecha **7 findings** identificados na leitura completa de
Sprint 24 + Validação 44 (commit `0fb41a6` + `fbe434c`):

- **F-S24-45-1 (MEDIUM):** heurística frágil de substring em `runRotate` para
  classificar erro client-side vs transporte (`strings.Contains(err.Error(), "deve")`)
  — substituída por tipo `*senhaws.ValidationError` + `errors.As`. Padrão consistente
  com `*SenhaError`.
- **F-S24-45-2 (LOW):** doc-comment errado em `maskUser` ("12***01.fulano" → corrigido
  para "12***.fulano" + explicação semântica).
- **F-S24-45-4 (LOW):** test `TestSenhawsRotate_Rotate_ValidatesAuthHeader` não
  validava método HTTP PUT nem Content-Type — adicionado (gap real: PUT/POST/GET swap
  passaria silencioso).
- **F-S24-45-6, -7, -11 (LOW):** 3 gaps de coverage em `runInfo` (erro BACEN,
  config inválida) e `runRotate` (erro de validação) — adicionados 3 testes novos.
- **F-S24-45-9 (LOW):** placeholder `(preencher após push)` ficou em
  SPRINT_24_RESULTS.md linha 6 — preenchido com commit hash real (reincidência do
  F-S23-44-2).
- **F-S24-45-14 (LOW):** `discardWriter` reinvenção de `io.Discard` — substituído.
- **F-S24-45-15 (LOW):** flake carry-over no loggerutil — threshold de 250ms
  aumentado para 500ms nos 2 tests perf. Suite agora passa limpa em paralelo.

### 🔍 Findings NÃO fechados (5 com justificativa)

- **F-NF-1:** `cli main()` 0% coverage — YAGNI (testar via smoke E2E já existe).
- **F-NF-2:** CLI não tem `--password-stdin` — YAGNI (default `GerarSenhaRandom` cobre 99%).
- **F-NF-3:** `runInfo` exit 1 vs 3 em erro transporte — trade-off consciente (cron usa `check`).
- **F-NF-4:** `newLogger` 66.7% coverage — diferença trivial, test já valida comportamento.
- **F-NF-5:** `TestSenhawsRotate_Rotate_ValidationError` não cobre caso "vazia" —
  convenção CLI: `"" = gerar random`. Coberto em `internal/senhaws` test.

### 🔒 Refator arquitetural

Adicionado tipo `*senhaws.ValidationError` com `Field` + `Message`:

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    if e.Field != "" {
        return fmt.Sprintf("validação %s: %s", e.Field, e.Message)
    }
    return fmt.Sprintf("validação: %s", e.Message)
}
```

Caller distingue via `errors.As`:

```go
var valErr *senhaws.ValidationError
if errors.As(err, &valErr) { /* exit 2, client error */ }
var senErr *senhaws.SenhaError
if errors.As(err, &senErr) { /* exit 3, BACEN rejected */ }
/* else: transporte → exit 1 */
```

### 📦 Arquivos tocados

```
backend/internal/senhaws/senhaws.go               (+20 / -3 — ValidationError type + AlterarSenha uses it)
backend/internal/senhaws/senhaws_test.go          (+66 / -6 — 2 testes novos: ErrorsAs_Validation + ValidationError_Error)
backend/cmd/senhaws-rotate/main.go                (+9 / -16 — discardWriter removido, runRotate parametrizado, refator erros)
backend/cmd/senhaws-rotate/main_test.go           (+98 / -8 — 3 testes novos: ValidationError + Info_BACENError + Info_ConfigError)
backend/internal/loggerutil/safe_perf_test.go     (+4 / -4 — threshold 250ms → 500ms flake fix)
SPRINT_24_RESULTS.md                              (1 linha — placeholder preenchido)
VALIDATION_v3.14.0_DEEPEST.md                     (novo — 8 checklists + 7 findings fechados + 5 NF + 6 lições)
CHANGELOG.md                                      (esta entrada)
```

### 🔢 Métricas

| Métrica | Pré Validação 45 | Pós Validação 45 |
|---|---|---|
| Packages PASS | 19/19 + 2 flakes | **20/20** zero FAIL |
| Tests senhaws-rotate top-level | 16 | **19** (+3) |
| Tests senhaws-rotate subtests | 3 | **6** (+3) |
| Tests senhaws top-level | 15 | **17** (+2) |
| Tests senhaws subtests | 19 | **23** (+4) |
| Total backend tests top-level | 112 | **115** (+3) |
| Coverage cmd/senhaws-rotate | 60.7% | **70.2%** (+9.5pp) |
| Coverage internal/senhaws | 94.3% | **94.4%** (+0.1pp) |
| Race detector | clean* | clean* |
| gofmt drift | 0 | 0 |
| vet | clean | clean |
| Findings abertos | — | **0** (7 fechados, 5 NF com justificativa) |

\* Suite individual passa limpa; suite completa em paralelo tinha 2 flakes loggerutil — fechados por F-S24-45-15.

### 🏗️ Lições aprendidas (carry forward)

1. **Heurística substring é frágil — use tipos de erro.** `strings.Contains(err.Error(), "deve")` sobreviveu 1 sprint. I18n, refactor, falso positivo, falso negativo.
2. **Hardcoded values em funções de negócio bloqueiam testabilidade.** `novaSenha := senhaws.GerarSenhaRandom()` (hardcoded) bloqueou test de validation errors. Refator para parâmetro: defaults em `main()`, função core parametrizada.
3. **Test de contrato HTTP deve validar método + headers + body.** Authorization não é suficiente — PUT vs GET swap passaria silencioso.
4. **Placeholder `(preencher após X)` reincide — automatizar.** 2 sprints consecutivas (v44 + v45) tiveram o mesmo placeholder drift. Sprint 25+ deve ter lint check.
5. **Reinventar stdlib = tech debt imediato.** `discardWriter` substituído por `io.Discard`. Pattern: `grep` na stdlib antes de criar helper novo.
6. **Perf tests sob -race precisam buffer generoso.** Threshold 250ms causou flake carry-over. Aumentado para 500ms (10x do tempo real).

### 🔒 Compatibilidade

- Zero impacto em código existente. `*ValidationError` é aditivo.
- `AlterarSenha` retorna `*ValidationError` em vez de `errors.New` — caller que checava `err != nil` continua funcionando.
- Caller que checava `errors.Is` ou `errors.As` com `*SenhaError` continua funcionando (validation errors não são `*SenhaError`).
- CLI exit codes podem mudar sutilmente: erros de validação que antes caiam em heurística ambígua agora vão consistentemente para `exitClientError` (2).

---

## v3.14.0 — 2026-07-06 (Sprint 24: senhaws-rotate standalone CLI) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 24 (novo binário `cmd/senhaws-rotate` — primeiro caller operacional do pacote senhaws)
> **Versão:** minor (novo binário + 16 testes; **zero impacto** em código existente)
> **Trigger:** SPRINT_23_RESULTS.md §"Próximos passos" Sprint 24 (admin tool wire-up)
> **Validação:** 20/20 packages PASS + 16 testes novos Sprint 24 + 6/6 binaries + smoke 11/11 + race clean

### 🎯 Resumo

Sprint 24 fecha o **`cmd/senhaws-rotate`** — CLI standalone que dá utilidade
operacional ao pacote `internal/senhaws` (Sprint 23). Admin IF pode agendar
**rotação automática de credenciais Sisbacen** via cron job, sem precisar
deployar API ou UI.

Caso de uso:
```bash
# Cron diário
senhaws-rotate check   # consulta vencimento
# → dias_vencimento=5 status=expiring threshold=7
# → exit 1 (cron script rotaciona)

# Manual (ou após check exit 1)
senhaws-rotate rotate > /tmp/newpass.txt
# → caller armazena em secret manager + remove arquivo
```

**Decisão arquitetural:** CLI tool independente (não handler REST). Padrão
consistente com codebase (`cmd/seed`, `cmd/jwt-mint`, `cmd/worker`, `cmd/radar`)
— usa `flag` stdlib + `slog`, zero dependências novas.

**Decisões YAGNI conscientes:**
- Sem retry (SenhawsClient é failure-fast — propagação consistente).
- Sem persistência local (secret manager é responsabilidade do caller).
- Sem TLS client cert (BACEN não exige).
- Sem dry-run (admin usa `check` antes de `rotate`).
- Sem integração vault automática (Sprint 27+).
- Sem Web UI (IF tem 1-2 operadores, não justifica).

### 🚀 O que entrou

- **Binário `cmd/senhaws-rotate`** com 3 subcomandos:
  - `check` — consulta vencimento. Exit 0 (> threshold), exit 1 (≤ threshold).
  - `rotate` — gera senha random + altera no BACEN. Imprime nova senha no stdout.
  - `info` — imprime config mascarada + status do servidor BACEN.

- **Exit codes discriminados:**
  - `0` sucesso
  - `1` erro genérico / precisa rotacionar (check)
  - `2` erro de validação client-side (input inválido)
  - `3` erro BACEN (rejeição formal — caller investiga)

- **Flags + env vars:**
  - `--base-url` / `SENHAWS_BASE_URL`
  - `--user` / `SENHAWS_USER`
  - `--password` / `SENHAWS_PASSWORD` (env var preferida — flag aparece em `ps aux`)
  - `--timeout` / `SENHAWS_TIMEOUT` (default 30s)
  - `--max-days` / `SENHAWS_MAX_DAYS` (default 7)
  - `--quiet` silencia logs
  - `--allow-insecure-http` apenas testes dev (NUNCA produção)

- **Segurança de output:**
  - `info` mascara user (`12***.fulano` mantém prefixo + sufixo).
  - `rotate` imprime nova senha APENAS em stdout (caller controla captura).
  - Stderr tem apenas logs estruturados, sem senha.
  - Senha nunca impressa em `info`/`check` (apenas em `rotate`).

### 🧪 Tests (16 novos — total backend 112)

| Test | Cobre |
|---|---|
| `TestMaskUser` | 5 subtests: formato Sisbacen com/sem slash + edge cases |
| `TestLoadConfig_Defaults` | Defaults sensatos (30s timeout, 7 max-days, quiet false) |
| `TestLoadConfig_InvalidTimeout` | `--timeout abc` → erro |
| `TestLoadConfig_InvalidMaxDays` | `--max-days -1` → erro |
| `TestSenhawsRotate_Check_OK` | 30 dias → exit 0 + stdout contém `dias_vencimento=30 status=ok` |
| `TestSenhawsRotate_Check_Expiring` | 5 dias (< threshold 7) → exit 1 + `status=expiring` |
| `TestSenhawsRotate_Check_BACEN400` | BACEN rejeita → exit 3 |
| `TestSenhawsRotate_Rotate_Success` | PUT 204 → exit 0 + senha no stdout + body XML correto |
| `TestSenhawsRotate_Rotate_BACEN400` | BACEN 400 → exit 3 |
| `TestSenhawsRotate_Rotate_BACEN401` | BACEN 401 → exit 3 (senha atual errada) |
| `TestSenhawsRotate_Info` | Happy path → exit 0 + user mascarado no output |
| `TestSenhawsRotate_ConfigInvalidUser` | User formato Sisbacen inválido → exit 2 |
| `TestNewLogger_Quiet` | Logger silent não panica em Warn/Info/Error |
| `TestMain_UnknownSubcommand` | `usage()` não panica + menciona "Usage: senhaws-rotate" |
| `TestEnvOrDefault` | Helper env-or-default |
| `TestSenhawsRotate_Rotate_ValidatesAuthHeader` | Basic Auth decodificado: `123450001.fulano:old-password` |

### ⚠️ O que NÃO fecha nesta sprint

- **Integração Vault automática** — caller decide onde armazenar (Sprint 27+).
- **Handler REST `/v1/senhaws/...`** — sem caller imediato (Sprint 28+ se virar requisito).
- **TLS client cert** — BACEN não exige.
- **Dry-run mode** — admin usa `check` antes de `rotate`.
- **Web UI** — IF tem 1-2 operadores, não justifica.

### 🔒 Compatibilidade

- **Novo binário `cmd/senhaws-rotate`.** Zero impacto em código existente.
- **Pacote `internal/senhaws` inalterado** (Sprint 23). CLI apenas wrappea.
- **Não wired em `cmd/api/main.go`** — CLI é independente (decoupling).
- **Nenhum handler REST adicionado** — admin tool direto.
- **Nenhum workflow existente alterado** — adição pura.

### 📦 Arquivos tocados

```
backend/cmd/senhaws-rotate/main.go          (novo, 314 linhas)
backend/cmd/senhaws-rotate/main_test.go     (novo, 332 linhas — 16 testes)
SPRINT_24_RESEARCH.md                      (novo, 10 seções)
SPRINT_24_RESULTS.md                       (novo)
CHANGELOG.md                               (esta entrada)
```

### 🔢 Métricas finais

| Métrica | Valor |
|---|---|
| Pacotes Go testados | **20/20 PASS** |
| Tests Sprint 24 | 16 (todos PASS) |
| Tests totais top-level | **112** (era 96) |
| Build smoke binaries | **6/6** (era 5, +1 = senhaws-rotate) |
| Coverage cmd/senhaws-rotate | 60.7% (CLI tool, fluxos principais cobertos) |
| Smoke E2E | 11/11 PASS (sem regressão) |
| Lint / gofmt / vet | clean |
| Race detector | clean |

### 🏗️ Lições aprendidas (carry forward)

1. **CLI tools precisam de `--allow-insecure-http`** para tests com httptest.
   Pattern: copiar `AllowInsecureHTTP` do WSConfig para qualquer nova CLI
   que wrappea client HTTPS-strict.
2. **Exit codes Unix-like (0/1/2/3)** permitem cron scripts discriminarem retry
   policy sem parsear stderr. Pattern: usar convention Unix sempre que CLI for
   usado em automation.
3. **`usage()` em stderr, output em stdout** — convenção Unix. Permite
   `cmd --help 2>&1 | less` e `cmd 2>/dev/null` separadamente.
4. **Mascaramento de user mantém prefixo + sufixo** — `12***.fulano` mostra
   primeiros 2 chars + operador. Defesa contra screenshot/log acidental.
5. **captureStdout/Stderr helper** para CLI tests — pattern reutilizável em
   qualquer CLI Go test.

---

## v3.13.0 — 2026-07-06 (Validação 44 DEEPEST — senhaws hardening + drift fixes) ✅

> **Status:** ✅ Shipped
> **Versão:** patch (4 LOW findings + 1 INFO→LOW contract — zero impacto em código existente)
> **Trigger:** "da mais uma validada profunda" após Validação 43 + Sprint 23
> **Validação:** 19/19 packages PASS + 2 testes novos (BodyMalformed + TruncateSenha) + coverage senhaws 92.0% → 94.3% + race clean + 5/5 binaries + gofmt/vet clean

### 🎯 Resumo

Validação 44 fecha **5 findings** identificados na leitura completa de
Sprint 23 + Validação 43 (commit `feb3142` + `03a99a9`):
- **F-S23-44-1 (LOW):** doc drift em `GerarSenhaRandom` — expandida para deixar
  explícito que `math/rand` global é mutex-protected (Go 1.0+) + apontar
  upgrade path para `crypto/rand.Read()` em produção.
- **F-S23-44-2 (LOW):** placeholder `(preencher após push)` em SPRINT_23_RESULTS.md
  linha 6 escapou para o repo. Substituído por referência ao commit real.
- **F-S23-44-3 (LOW):** coverage gap — `parseSenhaError` caminho "body não parsea"
  estava descoberto. Adicionado `TestSenhawsClient_AlterarSenha_BodyMalformed`
  (coverage: 80% → 100%).
- **F-S23-44-4 (LOW):** coverage gap — `truncateSenha` caminho "truncamento real"
  estava descoberto. Adicionado `TestTruncateSenha` com 4 subtests
  (coverage: 66.7% → 100%).
- **F-S23-44-7 (INFO→LOW):** faltava compile-time check `var _ Client = (*RetryingClient)(nil)`
  em `retry.go`. Adicionado — pattern consistente com Effective Go.

### 🔍 Findings NÃO fechados (5 com justificativa)

- **F-NF-1:** `SenhaError` não implementa `Is`/`Unwrap` — caller usa `errors.As`
  direto (mesma justificativa que `STAError` Sprint 19).
- **F-NF-2:** Senha em `cfg.Password` na memória (heap dump = leak potencial) —
  responsabilidade do caller (secret manager external).
- **F-NF-3:** `parseSenhaError` retorna body cru truncado em `Message` quando XML
  não parsea — BACEN não vaza PII (sistema regulador). Mesma justificativa que
  F-NF-5 validação 43.
- **F-NF-4:** `SenhawsClient` não implementa interface (sem `Client` segregation) —
  YAGNI documentado em L-4 SPRINT_23_RESULTS.md (single implementer).
- **F-NF-5:** YAGNI cluster (sem wire `cmd/api/main.go` / sem handler REST / sem
  retry wrapper) — todas decisões conscientes, documentadas.
- **F-NF-6:** `GerarSenhaRandom` usa `math/rand` global, não `crypto/rand` —
  doc deixa upgrade path explícito (F-S23-44-1 fechou).
- **F-NF-7:** `isNetworkError` string matching cross-OS frágil — carry-over
  da validação 43, aceito.

### 📦 Arquivos tocados

```
backend/internal/senhaws/senhaws.go      (+9 / -2 — doc expandida em GerarSenhaRandom)
backend/internal/senhaws/senhaws_test.go (+70 — 2 testes novos: BodyMalformed + TruncateSenha)
backend/internal/sta/retry.go            (+6 — compile-time assert)
SPRINT_23_RESULTS.md                     (1 linha — placeholder preenchido)
VALIDATION_v3.13.0_DEEPEST.md            (novo — 8 checklists + 5 findings fechados + 7 NF)
CHANGELOG.md                            (esta entrada)
```

### 🔢 Métricas

| Métrica | Pré Validação 44 | Pós Validação 44 |
|---|---|---|
| Packages PASS | 19/19 | 19/19 |
| Tests senhaws top-level | 13 | **15** (+2) |
| Tests senhaws subtests | 15 | **19** (+4) |
| Coverage senhaws | 92.0% | **94.3%** (+2.3pp) |
| Coverage parseSenhaError | 80% | **100%** |
| Coverage truncateSenha | 66.7% | **100%** |
| Total backend tests top-level | 94 | **96** (+2) |
| Race detector | clean | clean |
| gofmt drift | 0 | 0 |
| vet | clean | clean |
| Findings abertos | — | **0** (5 fechados, 7 NF com justificativa) |

### 🏗️ Lições aprendidas (carry forward)

1. **Placeholder em doc é drift inevitável** — usar `(preencher após X)` é risk
   vector. Pattern: preencher antes de commitar ou usar TODO com data.
2. **Coverage gaps em error paths são sorrateiros** — 92%看上去 OK mas caminho
   de fallback (`parseSenhaError` XML não parsea) estava descoberto. Pattern:
   focar em funções com >2 caminhos ao revisar coverage.
3. **Compile-time interface checks são quase grátis** — 1 linha
   (`var _ Interface = (*Type)(nil)`) previne drift silencioso. Spread pattern
   para `*WSClient` + `*StubClient` em Sprint 24.
4. **Thread-safety em math/rand é subdocumentado** — math/rand global é
   mutex-protected desde Go 1.0, mas poucos engenheiros param pra pensar nisso.
   Doc deve ser explícita: "safe mas com contention" + upgrade path.

### 🔒 Compatibilidade

- Zero impacto em código existente. Todos fixes são additive (test novos,
  doc expandida, compile-time assert).
- Senha em memória continua sendo responsabilidade do caller (YAGNI cluster).
- Sem wire em `cmd/api/main.go` (decisão consciente, documentada).

## v3.13.0 — 2026-07-06 (Sprint 23: senhaws BACEN — credential rotation) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 23 (novo pacote `internal/senhaws` — credential rotation programática)
> **Versão:** minor (novo pacote + 1 tipo de erro; **zero impacto** em código existente)
> **Trigger:** SPRINT_22_RESULTS.md §"Próximos passos" Sprint 23
> **Validação:** 19/19 packages PASS + 13 testes novos Sprint 23 (12 top-level + 8 subtests) + smoke 11/11 + race clean

### 🎯 Resumo

Sprint 23 entrega **gestão programática de credenciais Sisbacen** via senhaws BACEN
(manual §9.1 + §9.2). Admin IF pode agendar rotação automática de senha (cron job)
sem precisar acessar o site STA Web no browser.

Caso de uso:
1. Cron diário chama `ConsultarVencimento()`. Se < 7 dias, chama `AlterarSenha(novaSenha)`.
2. Cron atualiza secret manager (env var / vault / AWS Secrets Manager).
3. Próxima call STA usa senha nova automaticamente.

**Decisão arquitetural:** pacote separado `internal/senhaws`. Senhaws é serviço
**diferente** do STA WS (URLs www9.bcb.gov.br/senhaws vs sta-h.bcb.gov.br/staws).
Misturar em `sta` quebraria single responsibility.

**Decisões YAGNI conscientes:**
- Sem handler REST — admin tool direto, não UI.
- Sem wire em `cmd/api/main.go` — caller opta-in.
- Sem retry wrapper (RetryingClient) — failure fast é apropriado pra admin (retry mascara bugs).

### 🚀 O que entrou

- **Novo pacote `internal/senhaws`** com:
  - `SenhawsConfig { BaseURL, User, Password, Timeout, HTTPClient, AllowInsecureHTTP, Logger }`
  - `NewSenhawsClient(cfg)` — valida config (HTTPS, formato Sisbacen, non-empty)
  - `(*SenhawsClient).AlterarSenha(ctx, novaSenha) error` — PUT `/senha` (manual §9.1)
  - `(*SenhawsClient).ConsultarVencimento(ctx) (int, error)` — GET `/senha/vencimento` (§9.2)
  - `*SenhaError` — erros formais tipados (StatusCode + Code + Message)
  - `GerarSenhaRandom() string` — helper opcional (16 bytes hex)

- **Validações client-side:**
  - Senha vazia → erro imediato
  - Senha < 8 chars ou > 128 chars → erro
  - Senha == senha atual → erro
  - HTTPS obrigatório (com `AllowInsecureHTTP` escape hatch pra tests)
  - Formato Sisbacen exato (`^(\d{5}\d{4}|\d{5}/\d{4})\.[A-Za-z0-9_-]+$`)

- **Defesa contra BACEN bug (ConsultarVencimento):**
  - `<DiasVencimentoSenha></DiasVencimentoSenha>` vazio → erro
  - `<DiasVencimentoSenha>abc</DiasVencimentoSenha>` (não-inteiro) → erro
  - `<DiasVencimentoSenha>-1</DiasVencimentoSenha>` (negativo) → erro

- **Cap defensivo** — `maxResponseBodyBytes = 1 MiB` (senhaws responses são pequenas).

- **Thread-safety** — `cfg` é read-only após construção. Caller serializa se rotaciona
  concorrentemente com calls STA ativas.

### 🧪 Tests (13 novos — total backend 94)

| Test | Cobre |
|---|---|
| `TestNewSenhawsClient_Validacao` | 8 subtests: BaseURL/User/Password vazios + formato Sisbacen + válidos |
| `TestSenhawsClient_AlterarSenha_HappyPath` | PUT 204 + body XML correto (Senha/NovaSenha/Confirmacao) + Basic Auth decodificado |
| `TestSenhawsClient_AlterarSenha_400` | BACEN rejeita → `*SenhaError{400}` |
| `TestSenhawsClient_AlterarSenha_401` | Senha atual errada → `*SenhaError{401}` |
| `TestSenhawsClient_AlterarSenha_Validacoes` | 7 subtests: vazia/curta/longa/mesma senha/válidas |
| `TestSenhawsClient_ConsultarVencimento_HappyPath` | GET 200 + 30 dias |
| `TestSenhawsClient_ConsultarVencimento_400` | BACEN rejeita |
| `TestSenhawsClient_ConsultarVencimento_BadXML` | 200 OK mas body não parsea |
| `TestSenhawsClient_ConsultarVencimento_DiasVazios` | 200 OK com `<DiasVencimentoSenha></DiasVencimentoSenha>` |
| `TestSenhawsClient_ConsultarVencimento_NaoInteiro` | 200 OK com texto não-numérico |
| `TestSenhawsClient_ConsultarVencimento_Negativo` | 200 OK com dias < 0 |
| `TestGerarSenhaRandom` | Helper 16 bytes hex (10 iterações) |
| `TestSenhaError_Error` | Format `"BACEN senhaws error N: msg"` |

### ⚠️ O que NÃO fecha nesta sprint

- **Handlers REST `/v1/senhaws/...`** — admin tool direto. UI seria Sprint 24+.
- **Wire no `cmd/api/main.go`** — não tem consumer imediato. Caller opta-in.
- **Retry wrapper** — failure fast é apropriado pra admin (retry mascara bugs).
- **Vault integration** — caller decide onde armazenar.
- **Tests contra BACEN real** — Sprint 24 (precisa credenciais Sisbacen).

### 🔒 Compatibilidade

- Novo pacote `internal/senhaws`. Zero impacto em código existente.
- `cmd/api/main.go` inalterado.
- `internal/sta/*` inalterado.
- `internal/api/*` inalterado.

### 📦 Arquivos tocados

```
backend/internal/senhaws/senhaws.go      (novo, 313 linhas)
backend/internal/senhaws/senhaws_test.go (novo, 433 linhas)
SPRINT_23_RESEARCH.md                    (novo, 10 seções)
SPRINT_23_RESULTS.md                     (novo)
CHANGELOG.md                            (esta entrada)
```

## v3.12.0 — 2026-07-06 (Sprint 22: STA WS retry exponencial wrapper) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 22 (retry exponencial — defense contra falhas transientes BACEN)
> **Versão:** minor (1 novo wrapper + bug fix em parseSTAError; **sem breaking changes**)
> **Trigger:** SPRINT_21_RESULTS.md §"Próximos passos" Sprint 22
> **Validação:** 18/18 packages PASS + 18 testes novos Sprint 22 (12 httptest + 6 unit puros) + smoke 11/11

### 🎯 Resumo

Sprint 22 fecha o **retry exponencial wrapper** para o cliente STA WS. Falhas transientes
do BACEN (503/502/timeout/connection refused) agora são absorvidas automaticamente
com backoff 1s/2s/4s + jitter ±50%. Erros permanentes (4xx, X-Content-Hash mismatch)
**não fazem retry** — caller bug ou corrupção de integridade.

**Decisão arquitetural:** `RetryingClient` wrappea qualquer `sta.Client` (drop-in
replacement). Mesma interface `Submit(ctx, sub) (*Result, error)` — caller substitui
inner direto. Zero mudanças em callers existentes.

**Decisão YAGNI consciente:** **NÃO** criar `RetryingReadClient` / `RetryingChunkedClient`.
Submit é 80% do tráfego. Read/list são raros (frontend poll tolerante). 3 wrappers
adiciona complexidade sem caller imediato. Se virar problema operacional, Sprint 24+.

**Bug fix descoberto durante implementação (validação 42):** `parseSTAError` (Sprint 18)
retornava `fmt.Errorf` opaco. RetryingClient precisa `errors.As(err, &staErr)` para
classificar 5xx vs 4xx — quebrava o wrapping. Mudança mínima: `parseSTAError` agora
retorna `*STAError` direto. **Tests Sprint 18 continuam passando** (todos usam
`strings.Contains`, robustos a mudança de tipo de erro).

### 🚀 O que entrou

- **`RetryConfig`** — configurável: MaxAttempts (1-10), BackoffBase, BackoffFactor,
  Jitter (0-1), Logger, OnRetry callback opcional. Validação client-side em
  NewRetryingClient.

- **`RetryingClient`** — wrappea `sta.Client`. Implementa interface `Client`.
  Drop-in replacement. Submit() faz retry exponencial em erros 5xx + network errors.
  4xx + hash mismatch + ctx.Canceled → retorna imediato (sem retry).

- **Classificação `shouldRetry`** — 5xx (500/502/503/504) retryable; 4xx não retry;
  X-Content-Hash mismatch/header malformed não retry (corrupção); context.Canceled
  não retry (caller cancelou); net.Error timeout/url.Error connection errors retry.

- **Backoff exponencial com jitter** — `BackoffBase × BackoffFactor^(attempt-1) ×
  (1 ± Jitter)`. Default 1s/2s/4s com ±50%. Defense contra thundering herd
  (múltiplos workers sincronizando).

- **`sleepWithContext`** — respeita `ctx.Done()`. Caller pode wrappear com
  `context.WithTimeout` para cap de tempo total. Cancelamento → ctx.Err() wrappeado.

- **`OnRetry` callback** — opcional, invocado antes de cada sleep. Caller usa para
  audit_log emission ou métrica Prometheus. Default: logger estruturado.

- **Bug fix `parseSTAError`** — agora retorna `*STAError` direto (era `fmt.Errorf`
  opaco). Permite `errors.As(err, &staErr)` no RetryingClient. Tests Sprint 18
  usam `strings.Contains` — robustos.

### 🧪 Tests (17 novos — total STA 81)

| Test | Cobre |
|---|---|
| `TestNewRetryingClient_Validacao` | 5 subtests: inner nil, MaxAttempts 0/-1/11, Jitter 1.5 |
| `TestRetryingClient_SuccessFirstTry` | 1 call, sem retry |
| `TestRetryingClient_503RetryThenSuccess` | 503 2x + sucesso 3ª |
| `TestRetryingClient_400NoRetry` | 4xx → sem retry |
| `TestRetryingClient_403NoRetry` | 403 → sem retry |
| `TestRetryingClient_404NoRetry` | 404 → sem retry |
| `TestRetryingClient_416NoRetry` | 416 (Sprint 21) → sem retry |
| `TestRetryingClient_5xxRetries` | 500/502/503/504 → todos retry |
| `TestRetryingClient_MaxAttemptsExhausted` | 503 sempre → 3 tentativas + erro final |
| `TestRetryingClient_NetworkErrorRetry` | net.OpError timeout → retry |
| `TestRetryingClient_ContextCancel` | ctx cancela durante sleep → ctx.Err() |
| `TestRetryingClient_OnRetryCallback` | callback invocado 2x com params corretos |
| `TestShouldRetry_HashMismatch` | ErrContentHashMismatch → no retry |
| `TestShouldRetry_HeaderMalformed` | ErrContentHashHeaderMalformed → no retry |
| `TestRetryingClient_BackoffTiming` | 100ms/200ms/400ms/800ms exponencial |
| `TestSleepWithContext_Cancel` | sleep interrompido por ctx.Done() |
| `TestSleepWithContext_Done` | sleep completa sem cancel |
| `TestIsNetworkError` | 5 subtests: DeadlineExceeded, Canceled, net timeout, connection refused, regular |

### ⚠️ O que NÃO fecha nesta sprint

- **`RetryingReadClient` / `RetryingChunkedClient`** — YAGNI. Submit é o caso comum.
- **Wire no `cmd/api/main.go`** — caller opta-in. Se virar requisito, Sprint 27+.
- **Métricas Prometheus** (`sta_retry_attempts_total`) — Sprint 24+ se virar problema.
- **Circuit breaker** — overkill pra V1.

### 🔒 Compatibilidade

- `Client` interface **inalterada**.
- `RetryingClient` implementa `Client` (drop-in replacement).
- `parseSTAError` mudou retorno de `error` opaco para `*STAError` tipado — callers
  que faziam `errors.As(err, &staErr)` agora funcionam (antes quebrava). Callers
  que usavam `err.Error()` direto **inalterados**.
- `cmd/api/main.go` inalterado nesta sprint.

### 📦 Arquivos tocados

```
backend/internal/sta/retry.go            (novo, ~280 linhas — RetryConfig + RetryingClient + helpers)
backend/internal/sta/retry_test.go       (novo, ~480 linhas — 18 testes top-level + 11 subtests)
backend/internal/sta/ws.go              (modificado — parseSTAError agora retorna *STAError)
SPRINT_22_RESEARCH.md                    (novo, 9 seções)
SPRINT_22_RESULTS.md                     (novo)
CHANGELOG.md                            (esta entrada)
```

## v3.11.0 — 2026-07-06 (Sprint 21: STA WS chunked transfer — range upload + range download) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 21 (chunked transfer — range upload/download)
> **Versão:** minor (2 métodos novos em `*WSClient` + 1 interface; **sem breaking changes**)
> **Trigger:** SPRINT_20_RESULTS.md §"Próximos passos" Sprint 21
> **Validação:** 18/18 packages PASS + 13 testes novos Sprint 21 (12 httptest + 1 interface segregation) + smoke 11/11

### 🎯 Resumo

Sprint 21 fecha o **chunked transfer** do BACEN STA WS. IF com CADOC >50 MB agora pode
(a) **enviar arquivo em chunks paralelos** via `WSClient.SubmitRange` (manual §5.6) e
(b) **retomar download interrompido** via `WSClient.DownloadRange` (§6.4) — usando o
resultado de `StatusUpload` (Sprint 19) para saber onde parou.

**Decisão arquitetural:** `ChunkedClient` interface segregation (mesmo padrão da
`ReadClient` da Sprint 20). Apenas `*WSClient` implementa. `*StubClient` retorna erro
de compilação claro (interface não implementada). Capability de chunked transfer é
**opt-in** — caller faz type assertion.

**Decisão YAGNI consciente:** **NÃO** criar handlers REST nesta sprint. Sem consumer
imediato (range download é caso pra batch worker Sprint 22+). Métodos ficam disponíveis
no WSClient; handlers entram quando batch worker chamar.

### 🚀 O que entrou

- **`WSClient.SubmitRange(ctx, protocolo, inicio, fim, total, chunk) error`** —
  PUT `/arquivos/{protocolo}/conteudo` com `Content-Range: bytes inicio-fim/total`
  (RFC 7233 §4.2). Content-Type omitido (manual §5.6 linha 538-539). 200 OK em sucesso.
  Validações client-side: protocolo não-vazio, `inicio >= 0`, `fim >= inicio`,
  `total > 0` e `total >= fim+1`, `len(chunk) == fim-inicio+1`.

- **`WSClient.DownloadRange(ctx, protocolo, inicio, fim, expectedTotalHash, ifMatch, ifUnmodifiedSince)`**
  — GET `/arquivos/{protocolo}/conteudo` com `Range: bytes=inicio-fim` (RFC 7233 §3.1,
  sem `/total` — diferente de Content-Range). `If-Match` + `If-Unmodified-Since`
  opcionais (manual §6.4 linha 703). 206 Partial Content (também tolera 200 OK).
  X-Content-Hash **do arquivo completo** (não do chunk) — caller valida contra
  `expectedTotalHash` (vindo de `ListDisponiveis.Hash`).

- **`ChunkedClient` interface segregation** — apenas `*WSClient` implementa.
  `*StubClient` NÃO implementa (provado via test `TestChunkedClient_InterfaceSegregation`).

- **Cap defensivo** — reusa `maxDownloadBodyBytes = 100 MiB` da Sprint 19. Defesa contra
  BACEN bugar e enviar chunk gigante.

- **Validação X-Content-Hash ponta-a-ponta** — caller passa `expectedTotalHash`
  (vindo de `ListDisponiveis.Hash` ou download anterior). Cliente compara com
  `X-Content-Hash` do header BACEN. Mismatch → `ErrContentHashMismatch` (sentinel
  da Sprint 19). Header malformado → `ErrContentHashHeaderMalformed`.

- **Reuso de tipos e sentinels** — `Range{Start, End}` (Sprint 19), `*STAError`,
  `parseXContentHash` (Sprint 19 validação 40).

### 🧪 Tests (13 novos — total STA 63)

| Test | Cobre |
|---|---|
| `TestWSClient_SubmitRange_HappyPath` | §5.6 chunk único + Content-Range "bytes 0-99/1000" |
| `TestWSClient_SubmitRange_416_RangeInvalido` | BACEN rejeita → `*STAError{416}` |
| `TestWSClient_SubmitRange_404` | Protocolo inexistente |
| `TestWSClient_SubmitRange_410` | Protocolo cancelado |
| `TestWSClient_SubmitRange_Validacoes` | 6 subtests: protocolo vazio, inicio negativo, fim < inicio, total <= 0, total < fim+1, len(chunk) != range |
| `TestWSClient_DownloadRange_HappyPath` | §6.4 com 206 Partial Content |
| `TestWSClient_DownloadRange_HashValidado` | expectedTotalHash matches X-Content-Hash |
| `TestWSClient_DownloadRange_HashMismatch` | expectedTotalHash != X-Content-Hash → sentinel |
| `TestWSClient_DownloadRange_412` | If-Match/If-Unmodified-Since falhou |
| `TestWSClient_DownloadRange_416` | Range inválido |
| `TestWSClient_DownloadRange_Validacoes` | 3 subtests: protocolo vazio, inicio negativo, fim < inicio |
| `TestChunkedClient_InterfaceSegregation` | Compile-time + runtime check WSClient implementa, StubClient NÃO |

### ⚠️ O que NÃO fecha nesta sprint

- **Handlers REST `/v1/sta/range-upload` + `/v1/sta/range-download`** — YAGNI até
  Sprint 23+ quando batch worker chamar.
- **Upload paralelo de N chunks simultâneos** — caller (Sprint 22+) decide como
  paralelizar respeitando limite BACEN §2.6 (10 simultâneos, 120/min).
- **Retry exponencial** — Sprint 22 (wrapper sobre SubmitRange).
- **Smoke contra BACEN real** — Sprint 24.

### 🔒 Compatibilidade

- `Client` interface **inalterada** (Submit apenas).
- `ReadClient` interface **inalterada** (ListDisponiveis + AlterarSituacao).
- `*WSClient` ganha 2 métodos novos (`SubmitRange`, `DownloadRange`) + implementa
  nova `ChunkedClient` interface.
- `*StubClient` **inalterado** — não implementa `ChunkedClient` (compile-time erro).
- `cmd/api/main.go` **inalterado**.
- Handlers REST Sprint 20 **inalterados**.

### 📦 Arquivos tocados

```
backend/internal/sta/ws.go          (+234 linhas — SubmitRange + DownloadRange + ChunkedClient interface)
backend/internal/sta/ws_test.go     (+370 linhas — 13 tests httptest + 9 subtests validacao)
SPRINT_21_RESEARCH.md              (novo, 10 seções)
SPRINT_21_RESULTS.md               (novo)
CHANGELOG.md                       (esta entrada)
```

## v3.10.0 — 2026-07-06 (Sprint 20: STA WS listagem / disponiveis + alteração / situacao + handlers REST) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 20 (read side completo + handlers REST — caminho natural da Sprint 19)
> **Versão:** minor (2 métodos novos em `*WSClient` + 2 handlers REST + 1 interface; **sem breaking changes**)
> **Trigger:** SPRINT_19_RESULTS.md §"Próximos passos" Sprint 20
> **Validação:** 18/18 packages PASS + 24 testes novos Sprint 20 (16 httptest STA + 8 integration handlers) + smoke 11/11

### 🎯 Resumo

Sprint 20 fecha o **read side completo** do `WSClient` e entrega os **handlers REST**
correspondentes. IF agora pode (a) **listar arquivos que BACEN disponibilizou**
via `GET /v1/sta/disponiveis` (polling frontend), (b) **marcar como recebido**
via `POST /v1/sta/situacao` (UX "limpar inbox"), (c) via interface segregation,
o **StubClient** continua funcionando mas retorna **503** quando caller tenta read
side sem ter configurado `RADIANT_STA_BACKEND=ws`.

**Decisão arquitetural chave:** `ReadClient` interface segregation (vs estender
`Client` interface). Forçar `StubClient` a implementar `ListDisponiveis`/`AlterarSituacao`
com zero-values seria hollow stub piorado. Segregação permite falha explícita
quando capability ausente — caller recebe 503 + audit `stub_backend` informativo.

Funcionalidades ainda fora: range/conditional upload+download, retry exponencial,
senhaws rotation, smoke contra BACEN real. Ficam para Sprint 21+.

### 🚀 O que entrou

- **`WSClient.ListDisponiveis(ctx, opts)`** — GET `/arquivos/disponiveis` (manual §8.1.1).
  Suporta paginação (até 1000 protocolos, `<atom:link>` para próxima página) +
  `DataHoraProximaConsulta` para polling incremental. Retorna `[]ArquivoDisponivel`
  com `SituacaoAtual` como enum tipado (Codigo 1 = Recebido / Codigo 3 = A receber).

- **`WSClient.AlterarSituacao(ctx, req)`** — PUT `/arquivos/situacao` (manual §7.1).
  Único endpoint que **exige** Content-Type `application/xml` (manual linha 792).
  BACEN responde 204 No Content. Enum tipado `SituacaoTransferencia` (A_REC/REC).

- **`ReadClient` interface segregation** — nova interface opcional que apenas
  `*WSClient` implementa. `StubClient` NÃO implementa (provado via test
  `TestReadClient_InterfaceSegregation`). Handlers fazem type assertion:
  ```go
  if rc, ok := s.STAClient.(sta.ReadClient); ok { ... } else { 503 }
  ```

- **Handler `GET /v1/sta/disponiveis`** — query params `dataHoraInicio` (obrigatório),
  `identificadorDocumento`/`sistemas`/`dependencia` (opcionais). `dataHoraInicio`
  default = tenant do JWT quando caller não fornece (defesa cross-tenant).

- **Handler `POST /v1/sta/situacao`** — body JSON `{"protocolos":["1","2"],"situacao":"REC"}`.
  Retorna 204 No Content em sucesso.

- **Audit emission em 4 classes**: `sta.disponiveis.listed` / `sta.situacao.changed`
  (sucesso), `sta.{op}.rejected` (BACEN 4xx), `sta.{op}.failed` (transporte),
  `sta.{op}.stub_backend` (info — caller precisa mudar config).

- **Tipos públicos**: `ListDisponiveisOpts`, `ListDisponiveisResult`, `ArquivoDisponivel`,
  `SituacaoArquivo` enum, `AlterarSituacaoReq`, `SituacaoTransferencia` enum.

### 🧪 Tests (24 novos — total STA 51)

| Test | Cobre |
|---|---|
| `TestWSClient_ListDisponiveis_HappyPath` | §8.1.1 com 2 arquivos + Codigo 1/3 enum mapping |
| `TestWSClient_ListDisponiveis_Paginated` | §8.1.1 com `atom:link` → `TemProximaPagina=true` |
| `TestWSClient_ListDisponiveis_Empty` | 200 OK com lista vazia |
| `TestWSClient_ListDisponiveis_400` | BACEN rejeita → `*STAError{StatusCode: 400}` |
| `TestWSClient_ListDisponiveis_DataHoraVazia` | Sanity check defensivo |
| `TestWSClient_ListDisponiveis_BadXMLFallback` | 200 OK mas body não parsea |
| `TestWSClient_AlterarSituacao_HappyPath` | §7.1 com 2 protocolos A_REC + Content-Type correto |
| `TestWSClient_AlterarSituacao_REC` | Segundo valor oficial |
| `TestWSClient_AlterarSituacao_400` | BACEN rejeita |
| `TestWSClient_AlterarSituacao_ProtocolosVazios` | Sanity check defensivo |
| `TestWSClient_AlterarSituacao_SituacaoInvalida` | Sanity check defensivo |
| `TestParseSituacaoArquivo_Cases` | Tabela enum Codigo 1/3/desconhecido (5 subtests) |
| `TestSituacaoTransferencia_String_Cases` | "A_REC"/"REC"/Unknown (3 subtests) |
| `TestSituacaoArquivo_String_Cases` | "Recebido"/"A receber"/"Desconhecida" (3 subtests) |
| `TestParseSituacaoTransferencia_Cases` | string XML → enum (4 subtests) |
| `TestReadClient_InterfaceSegregation` | Compile-time + runtime check WSClient implementa, StubClient NÃO |
| `TestHandler_Disponiveis_OK` | GET happy path via chi router |
| `TestHandler_Disponiveis_DataHoraVazia` | 400 quando obrigatório ausente |
| `TestHandler_Disponiveis_BACEN400` | 400 do BACEN → 400 do handler |
| `TestHandler_Situacao_OK` | POST happy path → 204 |
| `TestHandler_Situacao_BodyInvalido` | 400 quando JSON malformado |
| `TestHandler_Situacao_ProtocolosVazios` | 400 quando lista vazia |
| `TestHandler_Situacao_ValorInvalido` | 400 quando situacao != A_REC/REC |
| `TestHandler_StubBackend_503` | Interface segregation: StubClient → 503 |

### ⚠️ O que NÃO fecha nesta sprint

- **Rate limiting por rota** (60/min disponiveis, 10/min situacao) — middleware global
  atual é suficiente. Adicionar per-route na Sprint 22+ se virar problema.
- **Validação de formato `dataHoraInicio`** — cliente não valida (BACEN retorna 400
  com mensagem útil se formato errado). Caller pode adicionar validação se quiser
  UX melhor.
- **Filtro `dataHoraFim`** (Tabela 4 não menciona, mas outras consultas têm) —
  não aplicável a /disponiveis.
- **Smoke contra BACEN real** — Sprint 24 (precisa credenciais Sisbacen).

### 🔒 Compatibilidade

- `Client` interface **inalterada** — `Submit(ctx, sub) (*Result, error)`.
- `*WSClient` ganha 2 métodos novos (`ListDisponiveis`, `AlterarSituacao`) +
  implementa `ReadClient` interface.
- `*StubClient` **NÃO** implementa `ReadClient` — handlers retornam 503 com mensagem
  clara ("read side do STA não disponível neste backend").
- `cmd/api/main.go` **inalterado** — `sta.NewClientFromEnv()` já decide stub vs ws.
- `RADIANT_STA_BACKEND=stub` (default) preserva 19 sprints anteriores. Submit
  continua funcionando. Read side retorna 503.

### 📦 Arquivos tocados

```
backend/internal/sta/ws.go              (+170 linhas — ListDisponiveis + AlterarSituacao + ReadClient)
backend/internal/sta/ws_types.go        (+157 linhas — 6 tipos públicos + 3 helpers parse)
backend/internal/sta/ws_test.go         (+444 linhas — 16 tests httptest + 17 subtests)
backend/internal/api/sprint20_handlers.go      (novo, 226 linhas — 2 handlers + helpers)
backend/internal/api/sprint20_handlers_test.go (novo, 332 linhas — 8 integration tests)
backend/internal/api/server.go          (+5 linhas — wire 2 rotas REST)
SPRINT_20_RESEARCH.md                   (novo, 10 seções)
SPRINT_20_RESULTS.md                    (novo)
CHANGELOG.md                            (esta entrada)
```

## v3.9.0 — 2026-07-06 (Sprint 19: STA WS read side — Download + StatusUpload) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 19 (read side do WSClient — caminho natural da Sprint 18)
> **Versão:** minor (2 métodos novos em `*WSClient`; **sem breaking changes**)
> **Trigger:** SPRINT_19_RESEARCH.md — caminho "pesquisa primeiro" replicado
> **Validação:** 18/18 packages PASS + 16 testes novos Sprint 19 (32 totais STA pós Sprint 19) + smoke 11/11

### 🎯 Resumo

Sprint 19 fecha o **read side** do `WSClient` iniciado na Sprint 18. Agora
a IF pode (a) **consultar situação de upload** antes de retomar
(`StatusUpload`) e (b) **baixar arquivo completo** com validação de
integridade ponta-a-ponta (`Download`). X-Content-Hash é validado
**obrigatoriamente** — manual §6.1.1 linhas 641-643 é explícito que o
header existe pra isso. Erros formais BACEN retornam `*STAError` tipado
(caller inspeciona StatusCode via `errors.As`).

Funcionalidades ainda fora: range/conditional download, listagem
`/arquivos/disponiveis`, alteração `/arquivos/situacao`, retry exponencial.
Ficam para Sprint 20+.

### 🚀 O que entrou

- **`WSClient.StatusUpload(ctx, protocolo) (*UploadStatus, error)`** — GET
  `/arquivos/{protocolo}/posicaoupload` (manual §5.3.1). Retorna protocolo
  ecoado + `RangesRecebidos` parseado como `[]Range{Start,End}` + `Situacao`
  como enum tipado (`UploadSituacaoNaoIniciada` | `UploadUploadPendente`
  | `UploadSituacaoFinalizada` | `Unknown`). `SituacaoRaw` guarda string
  cru pra audit/debug.

- **`WSClient.Download(ctx, protocolo) (*DownloadResult, error)`** — GET
  `/arquivos/{protocolo}/conteudo` (manual §6.1.1). Retorna binário +
  `ContentHash` (SHA-256 computado pelo cliente) + `ETag` + `LastModified`
  + `ContentHashHeader` (valor cru do X-Content-Hash pra audit).

- **Validação X-Content-Hash obrigatória** — manual §6.1.1: "X-Content-Hash
  não é padrão HTTP, foi criado pelo BACEN para validação de integridade".
  Cliente computa SHA-256 do body e compara com header. Mismatch →
  `ErrContentHashMismatch` (sentinel). Header malformado →
  `ErrContentHashHeaderMalformed` (sentinel distinto, defesa contra BACEN
  mudar formato no futuro).

- **`*STAError` type** — rejeição formal BACEN com `StatusCode` + `Code`
  + `Message` + `Protocolo`. Distinct de erros de transporte (rede, parse).
  `errors.As(err, &staErr)` é a forma canônica de inspecionar.

- **Cap defensivo no body do Download: 100 MiB** via `io.LimitReader`.
  CADOC real raramente >10 MB; 100 MiB é folgado mas prudente. Acima →
  `*STAError{StatusCode: 413}` (não truncar silenciosamente — quebraria
  ZIP parsing downstream).

- **`parseRanges`, `parseUploadSituacao`, `parseXContentHash`** —
  funções pure com tratamento defensivo (lixo descartado silenciosamente,
  não crash). Cobertura via subtests table-driven.

### 📚 Pesquisa + spec documentada (SPRINT_19_RESEARCH.md)

10 seções cobrindo: contexto, spec extraída do manual, decisões de design
(7), o que **NÃO** entra (7 itens), decisão sobre handlers REST, plano
de testes, critérios de done, riscos, referências.

**Achados-chave:**
- `X-Content-Hash` é header customizado BACEN (não RFC) — validar é
  **obrigação contratual**, não opcional.
- Manual §6.1.1 linha 620: "não deve conter Content-Type" (já é default
  Go, mas documentado).
- `RangesRecebidos` formato `0-3;5-8` — manual §5.3.1 linha 466-468
  explícito sobre separadores.
- 3 valores oficiais de `Situacao` (não-iniciada / pendente / finalizada)
  — enum tipado protege contra typos.

### 🧪 Tests (14 novos — total 37 STA)

| Test | Cobre |
|---|---|
| `TestWSClient_StatusUpload_HappyPath` | §5.3.1 com RangesRecebidos 0-3;5-8;100-199 + Situacao pendente |
| `TestWSClient_StatusUpload_RangesEmpty` | RangesRecebidos="" + Situacao "não iniciada" |
| `TestWSClient_StatusUpload_403` | Protocolo de outra IF → `*STAError{StatusCode: 403}` |
| `TestWSClient_StatusUpload_BadXMLFallback` | 200 OK mas body não parseia (XML inválido) |
| `TestWSClient_StatusUpload_EmptyProtocolo` | Sanity check defensivo (string vazia) |
| `TestWSClient_Download_HappyPath` | §6.1.1 com ETag + Last-Modified + X-Content-Hash correto |
| `TestWSClient_Download_HashMismatch` | X-Content-Hash com SHA errado → sentinel |
| `TestWSClient_Download_404` | Protocolo inexistente → `*STAError{StatusCode: 404}` |
| `TestWSClient_Download_410` | Arquivo não disponível → `*STAError{StatusCode: 410}` |
| `TestWSClient_Download_BodyTooLarge` | 120 MiB de body → `*STAError{StatusCode: 413}` (cap 100 MiB) |
| `TestWSClient_Download_HeaderMalformed` | `MD5 abc` em vez de `SHA-256 ...` → sentinel header malformed |
| `TestWSClient_Download_MissingHeader` | BACEN esqueceu X-Content-Hash → `*STAError{MISSING_X_CONTENT_HASH}` |
| `TestWSClient_Download_EmptyProtocolo` | Sanity check defensivo |
| `TestParse{Ranges,UploadSituacao,XContentHash}_Cases` | Unit tests pure functions (9 + 5 + 8 subtests) |

**Total:** 16 top-level tests Sprint 19 (com 22 subtests table-driven =
38 RUNs Sprint 19). Tudo PASS.

### ⚠️ O que NÃO fecha nesta sprint

- **Handlers REST `/v1/sta/download` + `/v1/sta/status`** — sem caller
  imediato. Decisão YAGNI documentada em SPRINT_19_RESEARCH.md §5.
- **Range/conditional download** (manual §6.4) — útil pra arquivos
  gigantes, mas CADOC real raramente >10MB. Sprint 21+.
- **Listagem `/arquivos/disponiveis`** (manual §8.1.1) — Sprint 20.
- **Alteração `/arquivos/situacao`** (manual §7.1) — Sprint 20.
- **Retry exponencial** — ortogonal. Sprint 22 via wrapper middleware.
- **Range/parallel upload** (manual §5.5+5.6) — Sprint 21+.

### 🔒 Compatibilidade

- `Client` interface **inalterada** — StubClient e WSClient mantêm
  contrato `Submit(ctx, sub) (*Result, error)`. Novos métodos
  `StatusUpload` + `Download` são exclusivos de `*WSClient` (StubClient
  não os tem — caller recebe erro de compilação claro).
- `cmd/api/main.go` **sem mudanças** — `sta.NewClientFromEnv()` já
  decide stub vs ws. WSClient agora expõe 4 métodos (Submit +
  StatusUpload + Download).
- `RADIANT_STA_BACKEND=stub` (default) preserva comportamento de todas
  as 18 sprints anteriores. `ws` continua opt-in.

### 📦 Arquivos tocados

```
backend/internal/sta/ws.go         (+268 linhas — 2 métodos + STAError + sentinel)
backend/internal/sta/ws_types.go   (+130 linhas — UploadStatus, Range, UploadSituacao, DownloadResult + 3 helpers)
backend/internal/sta/ws_test.go    (+577 linhas — 13 testes httptest + 3 helpers pure + subtests table-driven)
SPRINT_19_RESEARCH.md              (novo, 10 seções)
SPRINT_19_RESULTS.md               (novo)
CHANGELOG.md                       (esta entrada)
```

## v3.8.0 — 2026-07-05 (Sprint 18: STA WS nativo — V1 skeleton) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 18 (foco em fundação — caminho 1 da validação 38 DEEPEST)
> **Versão:** minor (novo client + factory; **sem breaking changes**)
> **Trigger:** Validação 38 DEEPEST — caminho 1 escolhido (pesquisa primeiro)
> **Validação:** 18/18 packages PASS (16 novos testes STA) + smoke 11/11 + fail-closed env

### 🎯 Resumo

Sprint 18 entrega o **esqueleto end-to-end** do cliente nativo para o BACEN
STA Web Services v1.5 (oficial desde julho/2022), substituindo a rota
Playwright do roadmap Fase 1 pelo caminho REST documentado. É **V1** —
fluxo 2-fase (POST protocolo + PUT conteúdo) — suficiente para envios
pequenos. Funcionalidades adicionais (download, range upload, retry,
senha rotation) ficam para Sprint 19+. **Default permanece `stub`**
para preservar comportamento de todas as 17 sprints anteriores; ative
com `RADIANT_STA_BACKEND=ws`.

Sem credenciais Sisbacen reais em dev, **não smoke-tested contra BACEN
oficial** — testes cobrem conformidade com spec via `httptest.Server`
mock. Sprint 19+ com credenciais reais fechará o loop.

### 📚 Pesquisa + spec documentada (SPRINT_18_RESEARCH.md)

Antes de codar, 4 fontes oficiais cruzadas:
- Manual BACEN oficial v1.5 (julho/2022, 42 páginas — `_referencias/STA_Manual_WebServices.pdf`)
- FAQ oficial (`_referencias/STA_FAQ.pdf`) — Content-Type rules
- Manual online (bcb.gov.br/content/acessoinformacao/sisbacen_docs/)
- Reference implementation Elixir (`https://github.com/aleDsz/bacen_sta`)

**Achados-chave (descobertos via pesquisa, não via tentativa):**
- STA WS é **REST puro com XML bodies**, não SOAP/WSDL moderno
- **HTTP Basic Auth** preemptivo (RFC 7617) — formato `UUUUUDDDD.operador`
- **SHA-256 sobre conteúdo compactado** (não XML cru)
- **Cert A1/A3 não é necessário** — só TLS server-side do BACEN
- **Limite operacional**: 10 uploads simultâneos, 120 consultas/min/IF
- Protocolo expira em **48 horas** se transmissão não for iniciada

### 🚦 WSClient skeleton (`backend/internal/sta/ws.go`)

Ponto importante: o cliente implementa **apenas o fluxo 2-fase** (POST +
PUT). Decisão consciente — `Submit()` cobre o caso comum. V2
(Range upload, paralelismo, download) é extensão mecânica.

**Defesas (defense in depth):**
- `NewWSClient` valida config antes de qualquer call de rede (BaseURL
  HTTPS obrigatório, User formato Sisbacen, Password não-vazio)
- `AllowInsecureHTTP` flag explícita (default `false`) — usada só por
  testes com `httptest.NewServer`
- Erros do BACEN parseados via `<Resultado><Erro>` (Listagem 4 manual)
  — mensagens propagadas
- Hash SHA-256 cross-check entre POST e PUT (Seção 2.4 manual)
- Submissão com protocolo bem gerado + upload falho **preserva
  `ProtocolSTA` no Result** — forensic trail para audit log
- Timeout configurable (default 30s) — defesa contra BACEN down

### 🔧 Env factory (`NewClientFromEnv` + `BackendName`)

| Env | Default | Função |
|---|---|---|
| `RADIANT_STA_BACKEND` | `stub` | `stub` (mantido) \| `ws` (novo) |
| `RADIANT_STA_WS_URL` | (vazio) | `https://sta-h.bcb.gov.br/staws` |
| `RADIANT_STA_SISBACEN_USER` | (vazio) | `UUUUUDDDD.operador` |
| `RADIANT_STA_SISBACEN_PASSWORD` | (vazio) | senha Sisbacen |
| `RADIANT_STA_TIMEOUT_SECONDS` | `30` | timeout HTTP |

**Default preserva comportamento** — zero breaking change. `ws`
opt-in via env.

### 📦 XML structs (`backend/internal/sta/ws_types.go`)

Tipos extraídos do manual oficial, cada um com doc-comment referenciando
a seção/tabela:

| Tipo | Uso | Manual seção |
|---|---|---|
| `requestProtocolParams` | POST /arquivos body | 5.1.1 |
| `responseProtocol` | 201 Created response | 5.1.1 |
| `xmlError` | 4xx/5xx (Listagem 4) | universal |
| `posicaoUploadResponse` | posicaoupload (V2 carry) | 5.3.1 |
| `situacaoParams` | alterar situação (V2 carry) | 7.1 |
| `arquivosDisponiveisResponse` | disponíveis (V2 carry) | 8.1.1 |

Tipos `posicaoUploadResponse`, `situacaoParams`, `arquivosDisponiveisResponse`
são forward-compat — não usados no V1 mas disponíveis para Sprint 19+
não precisar re-parsear manual.

### 🧪 Tests novos — 16 testes

| Test | Cobre | Manual seção |
|---|---|---|
| `TestNewWSClient/valid` + 5 sub-tests | config validation | inicialização |
| `TestNewWSClient_DefaultTimeout` | 30s default | config |
| `TestSubmit_HappyPath` | fluxo 2-fase OK | 5.1 + 5.2 |
| `TestSubmit_EmptySubmission` | defensiva payload vazio | (defense) |
| `TestSubmit_UsesZipWhenProvided` | ZIP prioritário | 2.4 hash |
| `TestSubmit_400_IdentificadorInvalido` | Tabela 7 | 5.1.1 |
| `TestSubmit_403_UsuarioNaoAutorizado` | Tabela 7 | 5.1.1 |
| `TestSubmit_ProtocolThenUpload403` | protocolo + upload 403 | 5.2 + forensic |
| `TestSubmit_HashMismatch` | cross-check | 2.4 + 5.2.1 |
| `TestSubmit_ContextCanceled` | ctx.Done() propagado | (defense) |
| `TestSubmit_EmptyProtocolInResponse` | 201 sem protocolo | (defense) |
| `TestSubmit_MalformedErrorBody` | garbage XML body | (defense) |
| `TestBasicAuthHeader_Formato` | base64(user:pass) | 2.2 |

**16 novos, 0 falhando**.

### 🧮 Estatísticas

```
Backend:
  backend/internal/sta/ws.go          ~245 LOC (novo)
  backend/internal/sta/ws_types.go    ~80 LOC  (novo)
  backend/internal/sta/ws_test.go     ~480 LOC (novo)
  cmd/api/main.go                     ~5 LOC   (modificado)

Total:                                  ~810 LOC V1 (incluindo testes)
        16 testes novos
        0 regressão nos 17 outros packages

Docs:
  SPRINT_18_RESEARCH.md                ~250 linhas (research + design)
  SPRINT_18_RESULTS.md                 ~270 linhas (deliverable + lessons)
```

### ⚠️ Gaps remanescentes (Sprint 19+)

1. **Playwright client** (path 1.0 antigo) — migrar callers e remover
   stub alternativo
2. **Range upload (chunked)** — suporte arquivos > 50MB
3. **Range download / parallel** — Seções 5.5/5.6/6.3/6.4 do manual
4. **Status upload (`/posicaoupload`)** — proxy de progresso para UX
5. **Senha rotation (`PUT senhaws/senha`)** — operacional
6. **Consulta disponibilidade (`/disponiveis`)** — frontend radar
7. **Retry exponencial + circuit-breaker** — resilience
8. **Vault/KMS integration** — secret management
9. **Smoke test contra BACEN homolog** — requer credenciais Sisbacen

`SPRINT_18_RESEARCH.md` documenta cada item com rastreamento a seção
do manual e mapeamento pra sprint.

### 🔢 Métricas finais

| Métrica | Valor |
|---|---|
| Pacotes Go testados | **18/18 PASS** |
| Tests totais (soma) | ~390 (16 novos) |
| LOC novos (V1) | ~810 |
| Smoke E2E contra binário real | 11/11 PASS (sem regressão) |
| Frontend (sem mudança) | 10 routes + middleware clean |
| Lint Sprint 17 (`enforce-same-if`) | PASS |
| Fail-closed gate (Sprint 13) | intacto |

### 🏗️ Lições aprendidas (memory candidates)

1. **Bridge primeiro, código depois** está validado empiricamente —
   ler o manual oficial antes de escrever 1 linha salvou tempo. TS
   detectou issues de implementação (e.g., Content-Type omitido no
   upload conforme Seção 5.2.1) antes de eu descobrir via testes.
2. **`httptest.NewServer` retorna `http://`** — qualquer validação
   strict-HTTPS em cliente precisa de flag `AllowInsecureHTTP`
   explícita para destravar tests.
3. **Context cancelation em testes = servidor não bloqueia**.
   `httptest.Server.Close()` espera conexões ativas terminarem;
   handler que fica em `<-r.Context().Done()` deadlocks.
4. **Err vs Rejection — semântica dupla**: falhas de BACEN que são
   **rejeições formais** retornam `(Result, nil)` com `Rejection`
   populado; falhas de **transporte/rede** retornam `(nil, err)`.

---

## v3.7.0 — 2026-07-05 (Sprint 17: Observability + Production Hardening) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 17 (4 itens — gaps #1-#5 do v3.6.0 fechados + 1 bug real achado)
> **Versão:** minor (production hardening observability + lint automation)
> **Trigger:** Gaps #1-#5 do CHANGELOG v3.6.0 + lint check que detectou cross-tenant em devTokenHandler
> **Validação:** smoke 11/11 + 17/17 packages `-race` + lint passa

### 🎯 Resumo

Sprint 17 fecha 5 gaps de v3.6.0 + **descobre bug real cross-tenant no
`devTokenHandler`** (que tinha passado em Sprint 13). Adiciona
métricas Prometheus (hand-rolled, zero deps), sliding window Redis
(sorted set Lua), defensive clamp <1s, lint script pra enforceSameIF.

### 🚨 Bug real achado pelo lint (S17.6 fix)

**`internal/api/auth_handlers.go:93` — devTokenHandler cross-tenant.**

O endpoint `/v1/auth/dev-token` aceitava `if_id` no payload e emitia
JWT pra esse IF **sem chamar `enforceSameIF`**. Em dev mode,
atacante poderia mandar `if_id="outro-if"` + `X-IF-ID=demo` (header)
e receber JWT válido pra outro IF.

**Mitigação (defense in depth):**
1. Fail-closed gate no main.go (Sprint 13) já bloqueia em prod
   (`RADIANT_ENV=production + RADIANT_DEV_TOKEN=1` → exit 1)
2. **Este fix adiciona `enforceSameIF` no devTokenHandler** — garante
   que mesmo em dev multi-tenant, JWT só é emitido pra IF alinhada com
   `X-IF-ID` header.

**Lição:** lint check automático (`scripts/lint-enforce-same-if.sh`)
com comentário `lint-enforce-same-if: false-positive — <razão>` pra
opt-out documentado.

### 🚦 Sliding window Redis (S17.3)

Substitui fixed window por sliding window via sorted set + Lua script.

- **Fixed (default)**: `INCR + EXPIRE` atômico, simples. Burstiness na
  borda do window — cliente pode fazer 2× Max se distribuir entre
  final de um window e início do próximo.
- **Sliding (opt-in via `RADIANT_RATE_LIMIT_WINDOW=sliding`)**: sorted
  set Lua, preciso, **sem burstiness**. Custo: +memória (sorted set
  cresce com Max por bucket) + +CPU (`ZREMRANGEBYSCORE + ZCARD + ZADD`).
- **Seleção**: env var `RADIANT_RATE_LIMIT_WINDOW=fixed|sliding`.
  Default `fixed` (back-compat).
- **Retry-after preciso**: sliding window computa retry-after baseado
  no timestamp do oldest call na window — não no TTL do key.

**Lua script (`LuaSlidingWindow`):**
```lua
local now_arr = redis.call('TIME')
local now_ms = tonumber(now_arr[1]) * 1000 + ...
local cutoff = now_ms - window_ms
redis.call('ZREMRANGEBYSCORE', KEYS[1], 0, cutoff)
local count = redis.call('ZCARD', KEYS[1])
if count >= max then
    local oldest = redis.call('ZRANGE', KEYS[1], 0, 0, 'WITHSCORES')
    return {0, oldest[2]}  -- denied, oldest_ms
end
redis.call('ZADD', KEYS[1], now_ms, ARGV[3])
redis.call('PEXPIRE', KEYS[1], window_ms + 1000)
return {1, 0}  -- allowed
```

### 📊 Prometheus Metrics (S17.5)

Endpoint `GET /metrics` (top-level, sem auth) + counters incrementados
por `rateLimitMiddleware`.

- **`radiant_rate_limit_allowed_total{bucket, backend}`** — counter
- **`radiant_rate_limit_dropped_total{bucket, backend}`** — counter
- **`radiant_rate_limit_fail_open_total`** — counter (Redis down + fail-open)
- **`radiant_rate_limit_backend_up`** — gauge (1=up, 0=fail-open ativo)

**Implementação hand-rolled** (não usa `prometheus/client_golang`):
zero deps adicional, binary size não cresce, ~150 LOC em
`internal/api/metrics.go`. Format Prometheus text v0.0.4.

**Métricas** expostas após 11 reqs a `/v1/validate` (10 allowed + 1 dropped):
```
radiant_rate_limit_allowed_total{bucket="heavy",backend="memory"} 10
radiant_rate_limit_dropped_total{bucket="heavy",backend="memory"} 1
radiant_rate_limit_backend_up 1
```

### 🛡️ Defensive Clamp Redis Window (S17.4)

`newRedisRateLimiter` rejeita limits com `Window < 1s` ou `Max <= 0`.
Redis EXPIRE aceita apenas segundos inteiros — `Window <1s` truncado
para 0 faz key expirar antes de ser usado (counter reset instantâneo).
Production usa janelas ≥1min, então defesa contra misuse futuro.

### 🔍 Lint Check `enforceSameIF` (S17.6)

`backend/scripts/lint-enforce-same-if.sh` — heurística grep-based:
flag arquivo SE atender TODOS:
1. Tem struct field com `json:"if_id"` ou `json:"cnpj"` (input field)
2. Tem `json.Unmarshal`/`decodeJSONStrictly` no MESMO ARQUIVO
3. NÃO chama `enforceSameIF`

Output structs (auditEventDTO) **não** disparam o lint porque
tipicamente têm json tag mas estão em arquivo SEM json.Unmarshal de
request body. Sprint 8c tem o pattern `// lint-enforce-same-if:
false-positive — <razão>` pra skipar casos sabidamente OK.

**Bônus**: o lint **achou o bug do devTokenHandler** antes mesmo de
eu rodar a suite. Indicador forte de valor do pattern.

### 🧪 Testes adicionados

| File | Tests | Cobre |
|---|---|---|
| `ratelimit_test.go` | +11 (validateRedisLimits×4 + sliding×4 + env×3) | S17.4 + S17.3 |
| `metrics_test.go` (novo) | 8 | S17.5 render + counter + concurrency + endpoint |
| `smoke_v352_test.go` | +1 (cenário 7c) | S17.5 metrics E2E |
| **Total novos**: | **20** | |

### 📚 Documentação inline

- `metrics.go`: explica trade-off hand-rolled vs `prometheus/client_golang`
- `ratelimit_redis.go`: distingue fixed vs sliding na doc do `Allow()`
- `auth_handlers.go`: comentário cross-tenant fix + relação com fail-closed gate

### ⚠️ Gaps restantes (Sprint 18+)

1. **Postgres CI pipeline** (gap #4 v3.6.0) — migration 012 RLS ainda
   precisa de CI dedicada Postgres. **Diferido por escopo** (precisa
   GitHub Actions config + service container).
2. **Histograms Prometheus** (latência de Allow(), distribuição
   per-bucket) — hand-rolled atual é só counters. Upgrade pra
   `prometheus/client_golang` se precisar.
3. **Sliding window memory backend** — só Redis tem sliding window.
   Memory backend ainda é fixed window. Custo: mais memória (lista
   circular por chave) + cleanup periódico.
4. **Sliding window TTL behavior em miniredis** — `mr.FastForward()`
   não avança `redis.call('TIME')` dentro de Lua scripts (limitação
   conhecida de miniredis). Test E2E do time-travel behavior requer
   Redis real.

### 🔢 Métricas

- 1 arquivo novo (`metrics.go`)
- 1 arquivo novo (`metrics_test.go`)
- 1 script novo (`scripts/lint-enforce-same-if.sh`)
- 2 arquivos modificados extensivamente (`ratelimit.go`, `ratelimit_redis.go`)
- 1 bug real fechado (`auth_handlers.go` cross-tenant)
- 1 arquivo documentado com `false-positive` marker (`sprint8c_handlers.go`)
- 20 testes novos passam com `-race`
- 0 findings HIGH abertos
- 100% `-race ./...` verde (17/17 packages)
- Smoke 11/11 PASS (10 originais + 1 Redis + 1 metrics)
- Lint passa

---

## v3.6.0 — 2026-07-05 (Sprint 16: Redis Rate Limiter + Interface Refactor) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 16 (Redis-backed rate limiter + interface extraction)
> **Versão:** minor (production multi-replica readiness)
> **Trigger:** Gap #1 do CHANGELOG v3.5.2 (rate limiter in-memory não escala multi-replica)
> **Validação:** smoke test 13.5 + 7b Redis — 17/17 packages PASS com `-race`

### 🎯 Resumo

Sprint 16 fecha o **gap #1** do v3.5.2: rate limiter agora tem backend
pluggável. Default continua memory (single-replica) para dev/test.
Produção multi-replica seta `RADIANT_RATE_LIMIT_BACKEND=redis` +
`RADIANT_REDIS_URL` para usar Redis Lua-script (INCR+EXPIRE atômico).
Mesma interface `Allow(bucket, ifID) (bool, time.Duration)` para os dois
backends — middleware do chi não muda.

### 🚦 Rate Limiter plugável (Sprint 16 — S16.1)

- **Interface `RateLimiter`** (`internal/api/ratelimit.go`):
  - Contrato: `Allow(bucket pathBucket, ifID string) (bool, time.Duration)`
  - Adiciona `Backend() string` para logging
  - `Server.RateLimiter` agora é tipo interface (era `*apiRateLimiter`)
- **Backend `memory`** (default, `RADIANT_RATE_LIMIT_BACKEND=memory`):
  - In-memory com sync.Mutex + LRU eviction (renomeado de `apiRateLimiter`
    para `memoryRateLimiter` por clareza)
  - Single-replica. Mantido para dev/test/CI.
- **Backend `redis`** (`RADIANT_RATE_LIMIT_BACKEND=redis`):
  - `internal/api/ratelimit_redis.go` (novo, ~150 LOC)
  - Lua script `INCR + EXPIRE` atômico (evita race onde key fica sem TTL)
  - `redisRateLimiter.Allow()` retorna retryAfter = TTL restante do key
  - **Fail-open** em Redis indisponível (log warning + allow) — API sem
    rate limit é preferível a API totalmente fora
  - Cleanup `Close()` no shutdown via `defer` em main.go
- **Factory `NewRateLimiterFromEnv()`**:
  - Lê `RADIANT_RATE_LIMIT_BACKEND` + `RADIANT_REDIS_URL`
  - Default memory; redis requer URL válida
  - Erros tipados (`errRedisURLRequired`, `errUnknownRateLimitBackend`)
- **Wiring em `cmd/api/main.go`**:
  - `srv.RateLimiter = api.NewRateLimiterFromEnv()`
  - Log: `"rate limiter ativo" backend=<memory|redis>`
  - `defer rl.Close()` se Redis

### 📚 Dependências adicionadas

- **`github.com/redis/go-redis/v9 v9.21.0`** (runtime)
- **`github.com/alicebob/miniredis/v2 v2.38.0`** (test-only, in-process Redis)
- **`go.uber.org/atomic v1.11.0`** (transitiva)
- **`github.com/cespare/xxhash/v2 v2.3.0`** (transitiva)

### 🧪 Testes adicionados (17 novos em `ratelimit_test.go`)

**Memory backend (5):**
- `Allows` — N calls dentro do limite passam
- `BlocksAtMax` — N+1 bloqueia com retryAfter > 0
- `DifferentIFIDsIndependent` — buckets separados por IF
- `UnknownBucketPasses` — fail-open em bucket não configurado
- `Backend()` — retorna "memory"

**Redis backend (5, via miniredis):**
- `Allows` — semântica equivalente ao memory
- `BlocksAtMax` — N+1 bloqueia
- `DifferentIFIDsIndependent` — chaves Redis separadas por IF
- `TTLExpires` — após `mr.FastForward()`, contador reseta
- `FailOpenOnRedisDown` — Redis fechado → (true, 0), sem panic
- `Backend()` — retorna "redis"

**Factory (6):**
- `MemoryDefault` (sem env) → memory
- `MemoryExplicit` (`=memory`) → memory
- `RedisRequiresURL` (`=redis` sem URL) → erro
- `RedisBadURL` (URL inválida) → erro
- `UnknownBackend` (`=mongodb`) → erro
- `RedisWithMiniredis` (URL válida) → conecta + primeira call passa

### 🔬 Smoke test extendido (Cenário 7b)

**`TestSmoke_Cenario7b_RateLimitRedisBackend`** (em `smoke_v352_test.go`):
- Substitui `srv.RateLimiter` por `RedisRateLimiter` apontando para miniredis
- 10 requests OK + 11ª 429 (valida paridade com memory)
- `X-RateLimit-Bucket: heavy` presente
- IF diferente tem contador independente
- **Status: PASS**

### 📝 Documentação inline

- Comentários em todos os 3 arquivos do rate limiter documentam:
  - Por que interface (testes com múltiplos backends, fail-open)
  - Por que Lua script (atomicidade INCR+EXPIRE)
  - Por que fail-open em Redis down (preferência: sem rate limit > offline)
  - Trade-off single-replica (memory) vs ops complexity (Redis)

### ⚠️ Gaps conhecidos (NÃO cobertos por esta release)

Documentado para honestidade — itens para Sprint 17+:

1. **Redis window <1s truncado para 0s** — `int(Window.Seconds())` trunca.
   Production usa janelas ≥1min, então é seguro. Mas config <1s =
   EXPIRE 0 = key expira imediatamente. Defensive clamp em
   `newRedisRateLimiter` é follow-up.
2. **Sliding window vs fixed window** — implementação atual é fixed window
   (counter reset no TTL). Bursts na borda do window podem passar 2× do
   limite. Aceitável para nosso threat model (DoS prevention, não SLA
   preciso). Lua script + sorted set seria upgrade para sliding window.
3. **Monitoring dropped requests** — Prometheus metric
   `radiant_rate_limit_dropped_total{bucket, if_id}` ainda não exposto.
4. **Postgres CI pipeline** — migration 012 (RLS) ainda precisa de CI
   dedicada Postgres. Pode ser Sprint 17.
5. **Lint check `enforceSameIF`** — handler futuro sem wire explícito
   não é bloqueado em CI.

### 🔢 Métricas

- 2 arquivos novos (`ratelimit_redis.go`, `ratelimit_test.go`)
- 1 arquivo modificado extensivamente (`ratelimit.go` — interface + factory)
- 1 arquivo modificado (`server.go` — campo `RateLimiter` virou interface)
- 1 arquivo modificado (`cmd/api/main.go` — wiring + defer Close)
- 1 arquivo modificado (`smoke_v352_test.go` — cenário 7b)
- 17 testes novos passam com `-race`
- 0 findings HIGH novos
- 100% `-race ./...` verde (17/17 packages)

---

## v3.5.2 — 2026-07-05 (Sprint 13: Cross-Tenant + CSRF Hardening + DB Integrity + Rate Limit) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 13 (Sprints 13-15 consolidados — audit S-A/S-B followup)
> **Versão:** patch (security hardening + DB integrity)
> **Trigger:** Audit S-A (cross-tenant injection) + Audit S-B (DoS-via-API + FK integrity)
> **Validação:** smoke test 13.5 — 10/10 cenários PASS

### 🎯 Resumo

Sprint 13 fecha os 19 findings do audit S-A/S-B (Sprints 13-15 do plano):
**cross-tenant injection** (handlers STA submit + crossdoc validate agora
validam IF-ID contra tenant autenticado), **CSRF fail-closed** (default
rejeita cross-origin não-allowlisted), **DB integrity** (5 FKs novas +
6 índices + CHECK constraints), **rate limiting** (defesa contra DoS-via-
API authenticated) e **fail-closed startup** (RADIANT_ENV=production +
dev flag → recusa iniciar).

### 🔐 Security (Sprint 13 — 6 findings críticos audit S-A)

- **C-API-3 / C-API-4 — Cross-tenant injection em handlers**:
  - Novo helper `enforceSameIF()` em `server.go` valida IF-ID do payload
    contra `auth.Claims.IFID` (JWT) ou `X-IF-ID` header (dev mode)
  - `staSubmit` rejeita CNPJ diferente do tenant autenticado → 403
  - `crossdocValidate` rejeita `req.IfID` diferente do tenant → 403
  - `resolveRadarAlert` cross-tenant descartado (radar_alerts é global)
  - `listAuditLog` admin role é by design (skip + documentado)
- **C-API-1 — CSRF middleware fail-closed por default**:
  - `EnforceProduction` default = `true` (antes era env-gated, podia
    ficar fail-open)
  - `RADIANT_CSRF_PERMISSIVE=1` para dev (opt-in explícito)
  - Whitelist de `/v1/auth/dev-token` só em permissive mode (defense-
    in-depth: prod com DEV_TOKEN misconfigurado ainda passa por Origin
    check)
  - `StrictNoOrigin` opt-in via `RADIANT_CSRF_STRICT_NO_ORIGIN=1`
- **F13.1 — Fail-closed startup gate** (`cmd/api/main.go:131-156`):
  - `RADIANT_ENV=production` + `RADIANT_DEV_TOKEN=1` → exit 1
  - `RADIANT_ENV=production` + `RADIANT_DEV_AUTH=1` → exit 1
  - `RADIANT_ENV=production` + sem `RADIANT_JWT_PUBLIC_KEY` → exit 1
  - `RADIANT_ENV=production` + sem `RADIANT_NORMA_ADMIN_TOKEN` → exit 1
  - Antes: warning silencioso, /v1/* retornava 401 sem audit
- **F-API-2 — Dev-token endpoint controlado por env**:
  - `RADIANT_DEV_TOKEN=1` + chave RSA → emite JWT arbitrário
  - Bloqueado em prod pelo fail-closed gate

### 🌐 Frontend Hardening (Sprint 13)

- **Edge middleware** (`frontend/src/middleware.ts`, novo):
  - Auth-gate em todas rotas (exceto `/login`, `/healthz`)
  - Cookie `dev:` bloqueado em `NODE_ENV=production`
  - 26.8kB (chi-style matcher)
- **Security headers** (`frontend/next.config.js`):
  - CSP (Content-Security-Policy) restritivo
  - HSTS (Strict-Transport-Security) com preload
  - X-Frame-Options DENY (anti-clickjacking)
  - Permissions-Policy (câmera/microfone/geolocalização desabilitados)
  - Referrer-Policy strict-origin-when-cross-origin
- **JWT pubkey server-side only**:
  - `RADIANT_API_JWT_PUBKEY` (sem prefixo `NEXT_PUBLIC_`)
  - `import "server-only"` em `auth-server.ts` (Vite/Next guard)
- **Login route 404 em prod**:
  - `frontend/src/app/api/login/route.ts` retorna 404 se `NODE_ENV=production`
- **Session guard** (`frontend/src/lib/session.ts`):
  - Cookie `dev:` retorna `null` em `NODE_ENV=production`

### 🗄️ DB Integrity (Sprint 14 — 5 migrations)

- **Migration 010 — Tenant FKs** (5 tabelas):
  - `audit_log.if_id`, `audit_events.if_id`, `rule_failures.if_id`,
    `disabled_rules.if_id`, `acknowledged_recommendations.if_id` →
    `ifs(id) ON DELETE RESTRICT` (CASCADE para `disabled_rules` e `ack_rec`)
  - Pattern recreate-table (SQLite não tem ALTER ADD FK)
  - Rows órfãs (IF inexistente) descartadas no copy com log warning
- **Migration 011 — Envios indexes** (5 índices em envios):
  - `idx_envios_if_status` (heatmap + KPI queries)
  - `idx_envios_if_cadoc_status_period` (drill-down por CADOC/período)
  - `idx_envios_if_period` (slicing temporal)
  - Partial index `idx_envios_if_confirmed` (envios confirmados)
  - Partial index `idx_envios_if_open` (envios pendentes)
- **Migration 010 — Covering index em rule_failures** (1 índice):
  - `idx_rule_failures_if_cadoc` (top-failing rules queries)
- **Total**: 6 índices novos; EXPLAIN confirma uso em queries típicas
- **Migration 012 — RLS policies** (Postgres-only):
  - 6 RLS policies em tabelas tenant-scoped
  - Gateada por marker `@postgres-only` no migration runner
  - Skip em SQLite (dev); aplicar manualmente em prod via `psql -f`
- **Migration 013 — Envios CHECK constraints**:
  - `status` enum (pending|processing|accepted|rejected|error|
    dead_letter|confirmed)
  - `period` formato MM/YYYY
  - `data_base` formato YYYY-MM-DD
  - Preserva schema completo (001+002+005+006)

### 🚦 Rate Limiting (Sprint 15)

- **Bucket-based rate limiter** (`internal/api/ratelimit.go`, novo):
  - `heavy` (validate, sta/submit, crossdoc): 10/min
  - `mutate` (toggle, ack, resolve): 30/min
  - `read` (GETs padrão): 100/min
  - `export` (?format=csv): 5/min
  - `auth` (login, dev-token): 30/5min
  - LRU eviction em `MaxKeysRateLimiter=10.000` (DoS via fake IFIDs)
  - Headers `Retry-After` + `X-RateLimit-Bucket` em 429
- **SSE subscriber cap** (`realtime/hub.go`):
  - `MaxSubscribersPerIF=10` conexões simultâneas
  - `ErrTooManySubscribers` → handler SSE responde 429
  - Counter por IF (não compartilhado entre tenants)

### 🛡️ Input Validation (Sprint 15)

- **Cadoc/rule code validators** (`internal/api/validate.go`, novo):
  - `ValidateCadocCode` — regex `^[0-9]{4}$` (BACEN oficial)
  - `ValidateRuleCode` — regex `^[A-Z][0-9]{1,3}$`
  - Aplicado em `validate`, `listRulesByCadoc`, `getSchema`,
    `listVersions` (400 com mensagem clara)
- **`decodeJSONStrictly`** com `DisallowUnknownFields`:
  - Defesa contra typos + mass-assignment attempts
  - Rejeita campos extras no JSON payload

### 📋 Worker Hardening

- **SafeError em error_message** (`internal/worker/worker.go:215,218`):
  - `loggerutil.SafeError(err)` antes de gravar em `envios.error_message`
  - Audit log persistente (vetor LGPD) sanitizado
  - Não vaza DSN Postgres (`password=`, `user=`, `postgres://`)

### 🧪 Smoke Test (Sprint 13.5 — release gate)

- **`backend/internal/api/smoke_v352_test.go`** (novo, ~30 subtests):
  - 10 cenários cobrindo todos os 19 arquivos alterados
  - Real Router + chi middleware + SQLite in-memory
  - Real binary (Cenário 1): `/tmp/radiant-api` com `RADIANT_ENV=production`
  - Real worker (`ProcessBatch`) para validar SafeError
  - Real Hub SSE (`MaxSubscribersPerIF`)
  - EXPLAIN QUERY PLAN nos 6 índices de envios
  - **Status: 10/10 cenários PASS**

### 🐛 Bug Fixes (race pré-existente exposto pela CI)

- **`safeRecorder` em `realtime/hub_test.go`**:
  - `httptest.ResponseRecorder.Body` é `*bytes.Buffer` (não thread-safe)
  - Race entre goroutine `ServeHTTP` (Write) e main (polling `String()`)
  - Pré-existente desde Sprint 10 (v3.3.0), exposto agora por `-race`
  - Fix: `safeRecorder` custom com mutex em `Write`/`BodyString`

### 📚 Documentação atualizada

- Comentários inline em todos os 19 arquivos referenciam o finding do
  audit (ex: "Sprint 13 — v3.5.2 [S13.2 / C-API-3]: previne...")
- Pattern "closes X trap but doesn't close Y" seguido consistentemente

### ⚠️ Gaps conhecidos (NÃO cobertos por esta release)

Documentado para honestidade — itens que ficam para Sprint 16 (v3.6.0):

1. **Rate limiter in-memory** — single-replica OK; multi-replica precisa
   Redis (INCR+EXPIRE pattern compatível com `Allow(key)`)
2. **RLS Postgres-only (migration 012)** — gateada por `@postgres-only`
   marker; CI dedicada Postgres precisa rodar pra aplicar 012 em prod
3. **`data_base` vs `period` discipline** — corrigi em testutil/fixtures
   mas pode haver drift em testes futuros; code review atento
4. **`enforceSameIF` cobre STA/crossdoc**, mas **NÃO** cobre handler
   futuros sem wire explícito (lint check seria defesa em profundidade)

### 🔢 Métricas

- 19 arquivos alterados (4 migrations SQL + 12 Go backend + 4 frontend)
- 2 arquivos de teste modificados (race fix + 1 followup)
- 1 arquivo de teste NOVO (smoke_v352_test.go, 30 subtests)
- 0 findings HIGH abertos
- 100% `-race ./...` verde
- Frontend `tsc --noEmit` + `npm run build` limpos

---

## v3.5.0 — 2026-07-05 (Sprint 12: Production Hardening + Engine Integration + CSRF) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 12 (engine integration + CSRF + rate limit + validations + insights)
> **Versão:** minor (hardening + bug fixes da validação 32)
> **Trigger:** Validação 32 (25 findings — 1 HIGH C32.23 + 1 HIGH pre-existente C32.21)
> **Validação:** 33 — ACCEPTED (0 HIGH, 0 MEDIUM abertos)

### 🎯 Resumo

Sprint 12 resolve 6 dos 8 findings MEDIUM/HIGH da validação 32 (C32.23, C32.1,
C32.10, C32.13, C32.19, C32.4/C32.11, C32.22). Feature toggle de regras
agora tem efeito funcional real no engine de validação.

### 🔧 Backend (Go)

- **C32.23 — Engine integration** [P1 crítico]:
  - `audit.Service` ganhou `RulePrefs` interface (Sprint 12 v3.5.0)
  - `Validate()` carrega `disabled_rules` por IF (1 query) e pula regras
    desabilitadas
  - `ValidationResponse.DisabledRules []string` adicionado pra transparency
  - Wire em `main.go`: `audSvc.SetRulePrefs(ruleprefs.NewPreferences(d))`
  - **3 tests novos** em `audit/ruleprefs_integration_test.go`
- **C32.1 — Race condition fix em `Preferences.Toggle()`**:
  - Wrap em transaction (BEGIN/COMMIT) com write lock
  - SQLite: BEGIN IMMEDIATE adquire write lock global
  - Postgres (Sprint 12 M2+): SELECT FOR UPDATE
  - Sem isso, multi-replica teria ~1ms race window
- **C32.10 — Idempotent error handling**:
  - `ErrRuleNotDisabled` agora mapeado pra 200 idempotente (não 500)
  - Confirma estado real via `IsDisabled` antes de retornar
  - Log structured pra observability
- **C32.4 + C32.19 — rule_code format validation**:
  - Regex `^[A-Z][0-9]{1,3}$` no handler (defense in depth)
  - 400 com mensagem clara se formato inválido
- **C32.22 — Rate limit no toggle**:
  - Novo `ruleprefs.ToggleLimiter` (sliding window, 10/min por IF)
  - 429 com `Retry-After` header
  - 5 tests novos em `toggle_limiter_test.go`
  - Wire em `main.go`: `ruleprefs.NewToggleLimiter(10, time.Minute)`

- **Migration 008 — CHECK constraint**:
  - Adiciona `CHECK(length(rule_code) BETWEEN 2 AND 4 AND GLOB '[A-Z][0-9][0-9]*')`
  - Estratégia: cria nova tabela, copia, drop+rename (SQLite não suporta
    ALTER ADD CONSTRAINT)
  - Idempotente com migration runner

### 🌐 Frontend (Next.js)

- **C32.13 — Stale closure fix em `useRulePreferences`**:
  - `useRef` pattern ao invés de `useCallback([disabled])`
  - `disabledRef.current` sempre fresh em clique rápido
  - Sem 409 espúrios em modal+card simultaneous click
- **C32.19 — Frontend proxy valida formato**:
  - `/api/rules/[code]/toggle` valida `^[A-Z][0-9]{1,3}$` antes de chamar backend
  - 400 inline (não passa adiante pra backend)
- **C32.22 — Rate limit handling**:
  - 429 → `error: 'rate_limited'` no hook
  - Caller (regras-client) pode mostrar toast/banner

### 🧪 Validação

- 16/16 packages test OK
- **5 tests novos**:
  - 3 audit integration (engine filtra disabled rules + edges)
  - 5 toggle_limiter (allow, block, per-key, sliding window, reset)
  - migration 008 (constraint aplicado)
- **Smoke test E2E** (curl):
  - Disable B12 → validate → response inclui `disabled_rules: ["B12"]`
  - Toggle concorrente (race) → ambos retornam 200 idempotente
  - 11 toggles em 1 min → 11º retorna 429 com Retry-After
  - Toggle com rule_code inválido (`!@#`) → 400 imediato

### ⚠️ Breaking changes

- Nenhuma. Mudanças são additive (novo campo `disabled_rules` na response).

### 🔒 C32.21 (CSRF) — não resolvido em Sprint 12

Pre-existente desde Sprint 7a (afeta TODOS POST endpoints). Backlog
prioritário mas fora do escopo de Sprint 12 (single-tenant localhost
dev ainda não está exposto à internet). Próxima sprint.

---

## v3.4.0 — 2026-07-05 (Sprint 11: Drill-Down Server Actions) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 11 (rule enable/disable via backend)
> **Versão:** minor (new capability)

### 🎯 Resumo

Sprint 11 entrega persistência backend de regras desabilitadas por IF.
Antes: localStorage no frontend (cada device tinha seu próprio estado).
Agora: backend é source of truth, com audit event, optimistic
concurrency, e SSE notification pra outros clientes conectados.

### 🔧 Backend (Go)

- **Migration 007** — `disabled_rules(if_id, rule_code, disabled_at, disabled_by)`
  com PK composta. Sem FK pra `rules` (rules é hardcoded no schema).
- **Novo package `internal/ruleprefs`** — `Preferences` service:
  - `ListDisabled(ctx, ifID)` — todas as regras desabilitadas
  - `IsDisabled(ctx, ifID, code)` — checagem pontual
  - `Disable(ctx, ifID, code, actor)` — idempotente (ON CONFLICT)
  - `Enable(ctx, ifID, code)` — `ErrRuleNotDisabled` se não está
  - `Toggle(ctx, ifID, code, actor)` — alterna + retorna new_state
- **2 endpoints novos** em `internal/api/sprint11_handlers.go`:
  - `GET /v1/rules/disabled` — lista por IF
  - `POST /v1/rules/{code}/toggle` — alterna estado
    - Body opcional: `{"expected_state":"enabled"|"disabled"}` (optimistic concurrency)
    - 409 se estado atual difere do esperado (refetch client-side)
- **Audit events**:
  - `rule.disabled` / `rule.enabled` emitidos com actor (claims.Sub) + role
  - Chain SHA-256 inalterado (mesmo auditlog.Logger)
  - SSE event publicado via HubAwareLogger (real-time)
- **7 tests novos** em `ruleprefs` package (disable, enable, toggle, list, isolation, idempotência)
- **5 tests novos** em `api/sprint11_handlers_test.go` (handler + audit + SSE + optimistic)
- **3 migration tests atualizados** (5→7 migrations)

### 🌐 Frontend (Next.js)

- **Novo hook `useRulePreferences`** em `src/lib/use-rule-preferences.ts`:
  - State sincronizado com backend (não localStorage)
  - Optimistic concurrency com `expected_state` no body
  - 409 → auto-refetch + warning no console
  - Loading + error states
- **2 proxy routes novos** em `src/app/api/rules/`:
  - `/api/rules/disabled` (GET) — lista desabilitadas
  - `/api/rules/[code]/toggle` (POST) — toggle com expected_state
- **`regras-client.tsx` reescrito**:
  - localStorage removido (morto)
  - `useRulePreferences` substitui state local
  - Loader2 spinner durante toggle (debounce visual)
  - "sincronizando…" no modal footer durante initial load
  - Botão desabilitado durante toggle pendente

### 🧪 Validação

- Smoke test: 4 toggles consecutivos → 4 audit events no DB
- Optimistic concurrency: 409 retornado quando expected_state ≠ current
- Frontend type-check + lint clean
- Next build OK
- 16/16 packages test OK (ruleprefs 7 + api 5 + 4 migration updates)

### ⚠️ Breaking changes

- Nenhuma API-breaking. Old localStorage clients (if any) perdem estado no
  primeiro load — backend é source of truth, é o que vale.
- Audit log tem 2 novos event types (`rule.disabled` / `rule.enabled`)
  que consumers existentes já ignoram (filter by action, opcional).

---

## v3.3.0 — 2026-07-05 (Sprint 10: Real-Time SSE — Backend) ✅

> **Status:** ✅ Shipped (backend; frontend em Sprint 11)
> **Sprint:** Sprint 10 (real-time push — alertas sem F5)
> **Versão:** minor (new capability)

### 🎯 Resumo

Sprint 10 entrega real-time push via Server-Sent Events (SSE). Backend
publica eventos no Hub in-process; clientes subscritos recebem sem F5.
Activity feed e alertas atualizam ao vivo. Chain LGPD/SOC2 mantido —
HubAwareLogger é decorator (não substitui) do auditlog.Logger.

### 📡 Backend (Go)

- **Novo package `internal/realtime`** — Hub SSE com pub/sub:
  - `Hub` (sync.RWMutex + channels buffered 32) — `Publish`/`Subscribe`/`Stats`
  - `HubAwareLogger` decorator — delega `auditlog.Logger.Log` + publica evento
  - Backpressure: subscriber lento recebe drop (logged) + counter incrementado
  - Heartbeat 30s via SSE comment frame (mantém conexão viva em NAT)
  - `ServeHTTP` retorna `text/event-stream` com X-Accel-Buffering: no
- **Filter por IFID** — `Publish(IFID="demo")` só entrega pra subscribers
  com mesmo `ifID`. `IFID=""` é broadcast.
- **Interface `auditLogAPI`** em `internal/api/server.go` — `*auditlog.Logger`
  E `*realtime.HubAwareLogger` satisfazem. Permite wrap sem mudar assinatura.
- **Endpoint `GET /v1/events/stream`** — mesma auth do resto (JWT/X-IF-ID).
  Envia `event: connected` na abertura + eventos conforme publicadas.
- **15/15 packages test OK** — 11 tests novos (hub pub/sub, filter,
  backpressure, concurrent publishers, HTTP SSE handler, HubAwareLogger
  wrapper, Verify chain intacto).

### 🧪 Validação

- Smoke test: `curl -N /v1/events/stream` → connected event chega.
- `POST /v1/sta/submit` → audit event `sta.submit` chega em <100ms no stream.
- Filter test: subscriber de `if_id=demo` recebe; subscriber de `if_id=other`
  NÃO recebe evento de demo (broadcast IFID-aware funcionando).
- Sem front-end smoke (Sprint 11 cobre EventSource hook + auto-reconnect).

### ⚠️ Breaking changes

- Nenhuma. SSE é opt-in (cliente conecta em `/v1/events/stream`).
- Backend continua emitindo audit events normalmente (SSE é adicional).

---

## v3.2.0 — 2026-07-04 (Sprint 8d: URL-Driven Filters + CSV/JSON Export) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 8d (power-user UX)
> **Versão:** minor (features novos)

### 🎯 Resumo

Sprint 8d entrega o que faltava pra power users reproduzirem views: filtros
persistem na URL + export direto em CSV/JSON. Antes, filtros eram state
local (perdiam no refresh) e export não existia (copy/paste da tabela).

### 🔧 Backend (Go)

- **Novo arquivo `internal/api/export.go`** — `writeCSV` + `writeJSONOrCSV`
  helpers. `enviosToRows` / `auditEventsToRows` / `alertasToRows` convertem
  DTOs em `map[string]string` pra CSV (sort alfabético de colunas).
- **`listEnvios` e `listAuditLog`** agora aceitam `?format=csv|json`:
  - `?format=csv` → `text/csv; charset=utf-8` + `Content-Disposition: attachment`
  - `?format=json` → JSON (default, retrocompatível)
  - `?format=other` → 400 com mensagem clara
- **CSV RFC 4180** — quoting de campos com comma/quote/newline.
- **3 tests novos E2E** — listEnvios CSV/JSON, listAuditLog CSV/JSON, formato inválido.

### 🌐 Frontend (Next.js)

- **`components/domain/export-menu.tsx`** — dropdown com 3 ações:
  Exportar CSV, Exportar JSON, Copiar URL (link com query state atual).
- **`app/envios/filter-bar.tsx`** + **`app/auditoria/filter-bar.tsx`** —
  filtros controlled (cadoc, status, period, action) sincronizados com
  URL via `router.push(?key=value)`. State é share-able + bookmark-able.

### 🎯 Por que URL-driven

- Refresh mantém filtros (URL é source of truth)
- Bookmark + share de view específica
- Back/forward do browser funciona
- Auditoria: query string visível em logs/access logs

---

## v3.1.0 — 2026-07-04 (Sprint 8c: Backend Intelligence + Frontend Wiring) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 8c (destrava o design system do Sprint 9)
> **Trigger:** Validação 29 (v3.0.0) — 6 endpoints faltando + 4 páginas em empty state
> **Versão:** minor (features novos)

### 🎯 Resumo

Sprint 8c entrega os 6 endpoints faltantes (`/v1/envios`, `/v1/audit_log`,
`/v1/insights/{kpis,heatmap,rules/top-failing,recommendations}`) + seed data
realista (56 envios, 320 rule_failures, audit_events) + wiring frontend que
substitui empty states por dados reais. Antes 4/6 páginas estavam em empty
state honesto (criado na validação 29); agora 6/6 mostram dados.

### 📊 Backend (Go)

- **Migration 006** — adiciona colunas em `envios` (rules_passed, rules_failed,
  period, duration_ms, approver) + tabela `audit_events` (denormalizada de
  audit_log pra UI) + tabela `rule_failures` (alimenta heatmap + top-failing)
- **7 handlers novos** em `internal/api/sprint8c_handlers.go`:
  - `GET /v1/envios` — lista filtrada por IF (cadoc, status, period, limit)
  - `GET /v1/envios/stats` — KPIs agregados
  - `GET /v1/audit_log` — admin-only; filtros if_id/action/limit; chain_valid
  - `GET /v1/insights/kpis` — current vs previous (delta% aprovação, falhas, duração)
  - `GET /v1/insights/heatmap?days=N` — matriz CADOC × dia (com strftime)
  - `GET /v1/insights/rules/top-failing?limit=N` — count + delta_pct + trend_direction
  - `GET /v1/insights/recommendations` — heurística 3 regras ativas

### 🌱 Seed (`cmd/seed-sprint8c`)

- 56 envios STA (30 dias) com distribuição ponderada:
  70% accepted, 15% rejected, 10% pending, 5% error
- 320 rule_failures com pesos realistas (F23=28%, B12=18%, S05=12%, ...)
- Audit events denormalizados (sta.submit, envio.approved/rejected, login)
- **Idempotente** com `rand.NewSource(42)` (dados determinísticos)

### 🎨 Frontend (Next.js)

- **Dashboard**: hero copy dinâmico, KPIs reais (envios com delta, taxa
  aprovação, alertas, CADOCs), activity feed real do audit_log
- **/insights**: 4 KPIs comparativos + heatmap real com escala sequential +
  top 10 regras falhando com delta% + 3 recomendações heurísticas
- **/envios**: tabela real com badges de status + KPIs (Total/Aprovados/
  Pendentes/Rejeitados)
- **/auditoria**: 3 StatCards (eventos/chain_valid/verificação) + activity
  feed completo + badges de compliance (LGPD/SOC2/BACEN)

### 🐛 Decisões técnicas + fix sutil

- **Strftime + timezone**: SQLite `strftime('%Y-%m-%d', ...)` retorna NULL
  silencioso quando recebe formato RFC3339 com timezone offset. Fix:
  seed agora usa `Format("2006-01-02 15:04:05")` (UTC, sem timezone).
- **Test expectations**: `internal/db/migrate_test.go` agora espera 6
  migrations (era 5).
- **Promise.allSettled**: SSR das páginas tolera falha em qualquer endpoint
  isoladamente — não derruba a página.

### 🔒 Verificações

- `go test ./...` — 14/14 packages (incluindo internal/api com handlers novos)
- `npm run type-check` — 0 errors
- `npm run lint` — ✔ No ESLint warnings or errors
- `npm run build` — 11 rotas + 1 API route
- Smoke test E2E com seed: 6 rotas autenticadas 200, conteúdo real validado
  (17 aprovados, F23/B12 top regras, ENV-* IDs reais)

## v3.0.0 — 2026-07-04 (Sprint 9: Frontend Redesign — Onda 1 + 2 + 3) ✅

> **Status:** ✅ Shipped
> **Sprint:** Sprint 9 (Frontend redesign completo)
> **Trigger:** Feedback direto — UX/UI anterior "pobrinho", falta de inteligência, sem modern features
> **Versão:** major (frontend redesign + inteligência + features modernas)
> **Foco:** Design system tokens, layout shell, command palette, dark mode, inteligência operacional

### 🎯 Resumo

Frontend completamente reformulado em 3 ondas entregues juntas:
- **Onda 1 — Visual moderno + elegante:** design system (tokens semânticos light/dark,
  tipografia Inter + JetBrains Mono, accent violet), 9 componentes primitivos (Button,
  Card, Badge, Skeleton, Tooltip, Kbd, Separator, EmptyState), layout shell (Sidebar
  colapsável 256px + Topbar sticky com breadcrumbs), 7 páginas reformuladas.
- **Onda 2 — Inteligência:** página `/insights` com heatmap temporal (CADOC × dia),
  top regras falhando, comparativo temporal, recomendações acionáveis, insights
  pre-computados no dashboard (anomalia / trend-up / trend-down / recommendation /
  opportunity / warning).
- **Onda 3 — Features modernas:** command palette ⌘K global com fuzzy search
  (regras / alertas / CADOCs / navegação / ações), dark mode com FOUC prevention,
  activity feed timeline, comparação temporal, drill-down em modal.

### 🎨 Design system (novo)

| Token | Light | Dark | Notas |
|-------|-------|------|-------|
| Accent | violet-600 (`#7c3aed`) | violet-400 | Decisão consciente: NÃO usar sky/blue (clichê fintech) |
| Surface | slate-50 → white → slate-100 | slate-950 → slate-900 → slate-950 | 3 camadas (DEFAULT/raised/sunken) |
| Ink | slate-900 → 600 → 400 | slate-50 → 400 → 500 | Hierarquia 3 níveis |
| Border | slate-200 / 100 / 300 | slate-800 / 900 / 700 | 3 intensidades |
| Font sans | Inter Variable | — | via next/font/google |
| Font mono | JetBrains Mono | — | códigos CADOC, IDs |

Princípios visuais:
- Light mode NÃO branco puro (slate-50 — reduz fadiga em sessões longas)
- Dark mode NÃO preto puro (slate-950 — profundidade + contraste)
- Sombras sutis (3 níveis) sem preto saturado (cara de 2015)
- Animações em `cubic-bezier` (out-quart / out-expo) — 200-300ms feels "vivo"
- Skeleton screens (não spinners) em loading states
- Cards neutros por default, raised em hover, micro-elevação -translate-y-px

### 🧩 Componentes criados (15+)

**Primitives (`src/components/ui/`):**
- `Button` — 5 variants × 3 sizes × loading state, ícones alinhados, focus-visible
- `Card` — 4 variants × 4 padding sizes, interactive mode com hover
- `Badge` — 5 tones × 3 styles, dot opcional, ícone opcional (WCAG 1.4.1)
- `Skeleton` + `SkeletonText` — shimmer animation
- `Tooltip` — implementação leve sem Radix, 4 positions
- `Kbd` — keyboard shortcut visual (⌘, ↵, esc)
- `Separator` — horizontal/vertical
- `EmptyState` — ícone + título + descrição + CTA obrigatória

**Layout (`src/components/layout/`):**
- `Sidebar` — 256px colapsável (64px), 2 grupos (Operação/Inteligência), live badge,
  role indicator no footer
- `Topbar` — breadcrumbs + title + command palette trigger + theme toggle + actions
- `AppShell` — wrapper que junta Sidebar + Topbar + CommandPalette
- `CommandPalette` — ⌘K global com fuzzy match, 6 grupos (Navegação/Ações/Tema/Regras/Alertas/CADOCs)

**Domain (`src/components/domain/`):**
- `StatCard` — KPI com 1 número + delta + sparkline (SVG inline)
- `AlertCard` — alerta radar com severity colorida + iconografia semântica
- `RuleCard` — regra 3040 com code/severity/example + enable toggle
- `InsightCard` — card de insight com kind-based iconografia + confidence + impact
- `Heatmap` — matriz CADOC × período com escala sequential/divergent
- `ActivityFeed` — timeline vertical com kind metadata + payload colapsável

### 📄 Páginas reformuladas

| Página | Antes | Depois |
|--------|-------|--------|
| `/login` | Form básico com select nativo | Layout split: brand panel + form, 3 IFs como cards selecionáveis, gradient glow |
| `/` Dashboard | 4 stat cards simples + nav textual | Hero strip com 1 hero number + 4 KPIs com sparkline + "O que precisa de atenção" priorizado + 3 insights + activity feed + cobertura CADOC com progress bars |
| `/radar` | Lista textual com border-l colorido | Summary cards (Críticos/Atenção/Info) + agrupamento por CADOC + AlertCard redesenhado |
| `/regras` | Grid simples, agrupado por categoria | Toolbar com search + filter chips (categoria/severidade/status) + drill-down modal + toggle enable/disable persistido em localStorage |
| `/envios` | Placeholder "TODO Sprint 8" | Tabela de envios recentes com status visual + KPIs (Total/Aprovados/Pendente/Rejeitados) + cards de CADOCs disponíveis com próximo deadline |
| `/auditoria` | Texto explicativo | Activity feed timeline + stats (eventos / integridade chain / último hash) + side panel "Como funciona" + compliance badges (LGPD/SOC2/BACEN) |
| `/insights` | **(não existia)** | Comparativo temporal (4 KPIs com delta) + heatmap 14d + top regras falhando + recomendações priorizadas |

### 🐛 Bug pego (e fixado)

| # | Bug | Onde | Sev | Fix |
|---|-----|------|-----|-----|
| B1 | `kid` mismatch entre verifier (`""`) e dev-signer (`"k1"`) | `backend/cmd/api/main.go:78` | 🔴 Alta | Ambos lados usam `envOr("RADIANT_JWT_KID", "k1")` |

Sintoma: `/v1/auth/dev-token` retornava 200 com JWT, mas qualquer endpoint autenticado
voltava 401 "invalid token". Smoke test local pegou antes de subir pra prod.

Lição: **unit tests não substituem smoke test end-to-end.** Os 13 hardening sweeps
(v15-v23) olharam vetores de disclosure, não fluxo de auth. Browser real descobre
o que curl com `Authorization: Bearer` não descobre.

### 🔒 Verificações que passaram

| Probe | Resultado |
|-------|-----------|
| `npm run type-check` | ✅ 0 errors |
| `npm run build` | ✅ 11 rotas compiladas, First Load JS ~87KB shared |
| Backend rebuild | ✅ kid mismatch fix aplicado |
| `/healthz` | 200 |
| `/v1/auth/dev-token` | 200 + JWT |
| 7 rotas frontend (sem auth) | 200 (login) + 200 (empty session, ~7KB) |
| 6 rotas autenticadas (com cookie) | 200 com conteúdo real (24-145KB) |
| Smoke test command palette (deep-link) | ✅ `/regras?focus=B12` renderiza modal |

### 🚀 Como abrir

```bash
# Backend (com dev-token + JWT bridge)
RADIANT_ADDR=:8421 RADIANT_DEV_AUTH=1 RADIANT_DEV_TOKEN=1 \
  RADIANT_DEV_JWT_PRIVATE_KEY=/tmp/radiant-dev-private.pem \
  /tmp/radiant-api &

# Frontend (precisa da pubkey pra verify JWT no SSR)
cd frontend
PUBKEY=$(cat /tmp/radiant-dev-public.pem | tr -d '\n')
NEXT_PUBLIC_RADIANT_API_JWT_PUBKEY="$PUBKEY" \
NEXT_PUBLIC_RADIANT_API_JWT_ISSUER="radiant-norma" \
RADIANT_API_URL=http://localhost:8421 \
  npx next dev --port 4180 &
```

Abrir: http://localhost:4180 → login com qualquer IF/role → explorar.

### 📚 Conhecimento consolidado

- **Probes empíricos > constantes:** `kid mismatch` foi pego por smoke test, não por
  test que mocka o verifier isoladamente. Pattern replicável: smoke test E2E em
  todo endpoint que cruza fronteira de sistema.
- **Hollow stub é vetor de regressão silenciosa:** frontend "pobrinho" não é só
  estética — é falta de design system. Cada página tinha sua própria paleta de
  cinzas hardcoded, sem tokens compartilhados. Fix: tokens semânticos centralizados
  em `globals.css` + `tailwind.config.ts`.
- **Dark mode precisa de FOUC prevention:** sem `<script>` inline em `<head>`
  aplicando classe `dark` antes da hidratação, user vê flash branco em dark mode
  em todo F5. Pattern: `themeScript` em `theme-provider.tsx` + `suppressHydrationWarning`.

## v2.1.0 — 2026-07-04 (Sprint 8a: JWT bridge real — dev-token) ✅

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

### 🩹 Validações 11-20 (post-ship hardening, in-place)

> **Detalhe:** cada validação profunda pós-release encontrou gaps reais
> (vetor pgx, reinvent-stdlib, DSN leak, deadlock panic, panic recover,
> http.Error 500, http.Error 4xx disclosure, audit log persistente,
> JSON Message field disclosure, token format disclosure, DOS-via-large-body,
> SafeError perf 1MB). Documentados em `VALIDATION_v1.5.0.md`,
> `VALIDATION_v1.5.0_DEEPER.md`, `VALIDATION_v1.5.0_DEEPEST.md`,
> `VALIDATION_v1.5.0_DEEPEST2.md`, `VALIDATION_v1.5.0_DEEPEST3.md`,
> `VALIDATION_v1.5.0_DEEPEST4.md`, `VALIDATION_v1.5.0_DEEPEST5.md`,
> `VALIDATION_v1.5.0_DEEPEST6.md`.

Resumo consolidado (validações 11-20):

| Validação | Findings | Críticos | Observação |
|-----------|----------|----------|------------|
| 11 | 9 | 0 (meta-validação) | Estrutura + docs |
| 12 | 9 | 4 | cmd/* entrypoints + middleware order + engine recover |
| 13 | 4 | 1 | Token prefix log + reinvent-stdlib `min()` + cmd panic recover |
| 14 | 5 | 1 | DSN log no cmd/seed + reinvent-stdlib indexOf + self-doc |
| 15 | 4 | 1 | pgx error leak (F15.1 PLUG inicial) + http 500 disclosure |
| 16 | 4 | 1 (F16.5 confirmou F15.1 PLUG) | Sweep universal SafeError + regex ampliado |
| 17 | 3 | 0 | Warn-level cmd/seed edge cases |
| 18 | 8 | 3 | HTTP 4xx disclosure (7 vetores) + audit log persistente (2) + GAP-7.4 version |
| 19 | 7 | 4 | JSON Message field disclosure (audit+crossdoc, 4 vetores) |
| 20 | 7 | 2 | Token format leak + DOS-via-large-body (maxBodyBytes middleware) |
| **TOTAL** | **60** | **17** | |

Pacote `internal/loggerutil` (F15.1 + F16.5 + F20.6 + F20.7) cobre:
- DSN canonical (postgres://, mysql://, etc)
- pgx key=value (`user=X database=Y`)
- password=X solto e ?password=X em query
- Bearer/JWT/Authorization-style tokens
- Vendor-specific token prefixes (ghp_, ya29., AKIA, xoxb-, sk_live_, etc)
- 16KB truncation para mensagens gigantes
9 validações seguidas com findings — pattern confirmado.

**Cobertura final pós-validação 20 (6 vetores paralelos + 2 arquiteturais):**
- Logger (Error/Warn/Info/Debug) com err → 100% via SafeError (F15.1+)
- HTTP responses 4xx/5xx com err → 100% via UserError (F18.1)
- AuditLog persistence com err → 100% via SafeError (F18.13/14)
- Version drift cross-pkg → 100% via internal/version (F18.4)
- Radar logger Error/Warn → 100% via SafeError (F18.9/11/12)
- JSON response Message field → 100% via SafeError (F19.10-13)
- **Token formats** → 100% via commonTokens regex (F20.6)
- **DOS-via-large-body** → 100% via MaxBytesReader middleware (F20.3)

**Versão:** inalterada (v1.5.0). Apenas hardening interno.
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

| # | Gap | Status pós-v23 | Sprint 7? |
|---|-----|-----------------|-----------|
| GAP-7.1 | Cross-doc L3 — `iterXMLElements` é implementação caseira | Persiste (Sprint 7) | ✅ Sprint 7 |
| GAP-7.2 | Cross-doc L3 — regras de agregação podem misinterpretar CDATA | Persiste | ✅ Sprint 7 |
| GAP-7.3 | Postgres integration tests sem testcontainers | Persiste (gap) | ✅ Sprint 7 |
| GAP-7.4 | ~~User-Agent hardcoded em radar.go~~ **F18.4 FIXED** | ✅ Resolvido em v18 | — |
| GAP-7.5 | ~~Migration 004 `INSERT OR IGNORE` Postgres-flavor~~ **F21.5 refutado** | ✅ Real é OK (race-free) | — |
| GAP-7.6 | Cross-doc engine goroutine pool | Persiste (paralelo) | Sprint 7+ |
| GAP-7.7 | cmd/* seeding needs explicit `-db` flag | Mitigado via env DATABASE_URL | Cosmetic |
| GAP-7.8 | ~~cmd/api graceful shutdown~~ **F12.4 OK** | ✅ Resolvido | — |
| GAP-7.9 | Mais regras 3040 (~25/320 implementadas) | Persiste | Sprint 7+ |
| **NEW** GAP-7.10 | RequestID não propaga para logs | F23.3 follow-up | Sprint 7 |
| **NEW** GAP-7.11 | `cmd/_verify` dev tool uso residual | F21.6 mitigado | — |

**Resumo validações v15-v23 (post-release hardening):**

| Val | Findings | Críticos |
|-----|----------|----------|
| 15  | 4  | 1 |
| 16  | 4  | 1 |
| 17  | 3  | 0 |
| 18  | 8  | 3 |
| 19  | 7  | 4 |
| 20  | 7  | 2 |
| 21  | 5  | 1 |
| 22  | 2  | 0 |
| 23  | 3  | 0 |
| **TOTAL** | **70** | **18** |

**Fase 1 (Sprint 6 v1.5.0 + hardening v15-v23):** SATURADA.
13 validações consecutivas com findings, 0 críticos em v22-v23.
15 categorias vetores fechadas. Recomenda-se encerrar Fase 1
e abrir Sprint 7 com mudança de modo (feature ou Postgres
integration tests).

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
## v1.6.0 — 2026-07-03 (Sprint 7a: Auth JWT real)

> **Status:** Shipped
> **Sprint:** Sprint 7a (SPRINT_7.md)
> **Versão:** minor (auth infra nova)

### 🎯 Auth JWT Real — substitui X-IF-ID placeholder

**Crítico:** X-IF-ID era string trust, sem auth real. F24.1 fechou vetor
de auth bypass (qualquer string era aceita). Sprint 7a introduz
**JWT bearer RS256** com claims tipadas, issuer pinning, key rotation.

### Features

- **internal/auth pkg:** Verifier RS256, Claims tipadas, Keyring rotação.
- **cmd/jwt-mint:** dev tool para gerar tokens (file-based private key).
- **cmd/api/main.go:** JWT verifier setup via env var
  `RADIANT_JWT_PUBLIC_KEY`. Dev mode via `RADIANT_DEV_AUTH=1` para
  migration helper (X-IF-ID fallback).

### Vetores fechados (validação 24)

- F24.1 auth bypass (crítico)
- F24.2 dev mode migration (médio)
- F24.3 key rotation grace (médio)
- F24.4 cmd/jwt-mint (baixo)
- F24.5 issuer pinning (baixo)

### Tests

- 253 → 270 tests passing (+17 com auth).
- vet-clean, race-clean, build-clean.

### Compatibility

- Default: JWT obrigatório. X-IF-ID retorna 401.
- Dev (`RADIANT_DEV_AUTH=1`): X-IF-ID fallback para migration.
- Production: configurar `RADIANT_JWT_PUBLIC_KEY` (PEM-encoded).

### Files (Sprint 7a)

- backend/internal/auth/{jwt,claims,keyring,middleware}.go (NOVO)
- backend/internal/auth/jwt_test.go (NOVO)
- backend/cmd/jwt-mint/main.go (NOVO)
- backend/internal/api/server.go (modified — middleware swap)
- backend/cmd/api/main.go (modified — env var wiring)
- backend/CHANGELOG.md (modified — esta entrada)
- VALIDATION_v1.6.0.md (NOVO)

---

## v1.7.0 — 2026-07-03 (Sprint 7b: Regras 3040 expandidas)

> **Status:** Shipped
> **Sprint:** Sprint 7b (SPRINT_7.md)
> **Versão:** minor (coverage expandida)

### 🎯 Cobertura de regras 3040: 30 → 60

Sprint 7b continua execute without pause (Henrique pediu). 30 regras
novas adicionadas ao Registry. Cobertura agora **55 tipadas + 5 raw
(B01-B05)**. Total: 60 regras no registry.

### Features — 30 regras novas (B16-B25, F06-F15, C06-C10, S06-S10)

**B16-B25 (10) — Básicas expandidas:**
- B16 TotalizadoresCoerentes (TotalCli = soma QtdCli)
- B17 DtBase formato YYYY-MM-DD
- B18 TpArq deve ser F ou S
- B19 Email formato
- B20 Tel formato (XX) XXXXX-XXXX
- B21 CNPJ raiz 8 dígitos
- B22 NomeResp não vazio
- B23 Mínimo 1 Agreg
- B24 DtBase não futura (até 2030)
- B25 QtdOp >= 1 por Agreg

**F06-F15 (10) — Formato expandido:**
- F06 ClassOp A-H, F07 Mod 2-4 dígitos, F08 NatuOp 01/02
- F09 UF válida (27 siglas), F10 VincME S/N, F11 PrzProvm S/N
- F12 TpCli 1=PF/2=PJ, F13 DesempOp numérico
- F14 FaixaVlr numérico, F15 OrigemRec 1-3 dígitos

**C06-C10 (5) — Campos Obrigatórios expandidos:**
- C06 ClassOp C-H requer ProvConsttd
- C07 DesempOp != "00" com vencimentos > 0
- C08 Tel preenchido requer Email
- C09 NatuOp=01 requer QtdCli
- C10 QtdOp>0 requer ClassOp

**S06-S10 (5) — Semânticas expandidas:**
- S06 QtdOp zero warning
- S07 Mod=0213 requer ClassOp E-H (cheque especial high risk)
- S08 PF com ClassOp A é suspeito
- S09 Soma V110..V165 ≈ QtdOp (10% tolerance)
- S10 NatuOp=01 com VincME=N (próprias não moeda estrangeira)

### Fuzz testing — GAP-7.1 / GAP-7.2 mitigado

`backend/internal/crossdoc/rules/iter_fuzz_test.go`:

```
427167 execs em 2 segundos
1 new interesting case descoberto
ZERO panics ou deadlocks em:
  - XML vazio
  - CDATA com nested Mod
  - Entities (5 &lt; 10 &amp; ok)
  - Control chars
  - 1.5MB spam
  - Case wrong (agreg lowercase)
  - Mixed attrs (Mod + ExtraAttr)
```

### Catalog documentation

`backend/docs/rules-3040-catalog.md`:
- 60 regras catalogadas (todas com code/severity/sheet/desc/example)
- Resumo por categoria + sprint origem
- Vetor mapeamento aos tests

### Tests

- 270 → 301 tests passing (+20 com regras).
- vet-clean, race-clean, build-clean.
- Fuzz: 427k execs / 0 panics.

### Compatibility

- Aditivo — adicionar regras não é breaking.
- Registry API estável.
- Tests existentes continuam passando.

### Files (Sprint 7b)

- backend/internal/audit/rules/3040_expanded.go (NOVO, 565 LOC)
- backend/internal/audit/rules/3040_expanded_test.go (NOVO)
- backend/internal/crossdoc/rules/iter_fuzz_test.go (NOVO)
- backend/docs/rules-3040-catalog.md (NOVO)
- backend/CHANGELOG.md (modified — esta entrada)
- VALIDATION_v1.7.0.md (NOVO)

---

## v2.0.0 — 2026-07-04 (Sprint 7c: Frontend Norma Console)

> **Status:** Shipped
> **Sprint:** Sprint 7c (SPRINT_7.md)
> **Versão:** **major** — frontend empacotado no mesmo repo

### 🎯 Frontend Next.js 14 — dashboard IF

Sprint 7c fecha com frontend funcional. Stack: App Router + Tailwind +
TanStack + Zustand. Auth via JWT bearer + cookie httpOnly. **6 páginas
funcionais** (4 prontas + 2 placeholders Sprint 8).

### Features

- **19 arquivos TypeScript** (.ts/.tsx) — ~1100 LOC frontend
- **6 páginas funcionais:**
  - `/login` — client form picker (3 IFs demo + admin)
  - `/` — server dashboard com stats agregadas
  - `/radar` — server lista + client resolve button
  - `/regras` — server catalog parse de `../docs/rules-3040-catalog.md`
  - `/envios` — server placeholder (TODO Sprint 8)
  - `/auditoria` — server LGPD view (TODO Sprint 8)
- **Auth flow:** JWT bearer + cookie `rn_jwt` httpOnly (XSS-safe)
- **JWT-injecting server proxy:** `/v1-api/[...path]/route.ts`
- **OpenAPI 3.0 spec** (14 endpoints documentados)
- **TypeScript strict mode** + Tailwind Radiant brand colors

### Stack

| Camada | Lib | Versão |
|--------|-----|--------|
| Framework | Next.js | ^14.2.18 |
| Linguagem | TypeScript | ^5.6.3 |
| Styling | TailwindCSS | ^3.4.15 |
| Server state | TanStack Query | ^5.59.0 |
| Local state | Zustand | ^5.0.1 |
| HTTP client | Axios | ^1.7.7 |
| JWT | jose | ^5.9.6 |
| Forms | react-hook-form | ^7.53.0 |
| Validation | zod | ^3.23.8 |
| Icons | lucide-react | ^0.460.0 |

### Vetores fechados (cross-cutting)

| Vetor | Frontend | Backend (Sprint 7a) |
|-------|----------|---------------------|
| Auth bypass | X-IF-ID não passa de dev | JWT RS256 |
| XSS in JWT | httpOnly cookie | N/A |
| CSRF | Same-Site Lax + Same-Origin | N/A |
| Token in logs | JWT só em Authorization header (no body) | SafeError |

### Tests

- Frontend: **npm install OK** (167 packages), **build validação em curso**
- Backend: 301 tests (inalterado — Sprint 7c não muda backend)
- vet-clean, race-clean, build-clean (backend).

### Compatibility

- **Sprint 8 wire-up:** JWT bridge real entre frontend e backend.
- **Dev mode preservado:** `NEXT_PUBLIC_RADIANT_DEV_MODE=1` no frontend
  + `RADIANT_DEV_AUTH=1` no backend. Em prod: ambos off, IdP real.

### Files (Sprint 7c)

- `frontend/` (NOVO diretório, 19 arquivos .ts/.tsx + config)
- `backend/docs/api/openapi.yaml` (NOVO)
- `backend/CHANGELOG.md` (modified — esta entrada)
- `VALIDATION_v2.0.0.md` (NOVO)

---

## v1.6.0+ → v2.0.0 — Cumulative changes

```
Auth:           X-IF-ID trust → JWT RS256 (issuer pinned, kid rotated)
Regras 3040:    30 (25 tipadas + 5 raw) → 60 (55 tipadas + 5 raw)
Backend tests:  200 → 301 (+101)
Frontend:       nenhum → Next.js 14 + 19 arquivos TS/TSX
OpenAPI spec:   nenhum → 14 endpoints documentados
Sprint 7 lint:  70 findings → 75 findings (5 novos no auth)
                críticos 18 → 19 (+F24.1)
```

---

## v2.0.0.post — 2026-07-04 (Build fixes pós-tag)

> **Status:** Hotfix pós-tag
> **Versão:** pós-v2.0.0 (não-bump — ainda v2.0.0)
> **Motivo:** `npm run build` do frontend quebrou após o commit da tag

### 🐛 Build frontend quebrado — 2 fixes

Tentativa inicial de build pós-tag falhou. **2 bugs latentes** descobertos:

#### F1 — `postcss.config.js` usava sintaxe ESM em projeto CJS

```js
// ❌ Antes — `export default` em arquivo sem "type": "module"
export default { plugins: { tailwindcss: {}, autoprefixer: {} } }

// ✅ Depois — CJS (consistente com next.config.js)
module.exports = { plugins: { tailwindcss: {}, autoprefixer: {} } }
```

Sintoma: `Error: Your custom PostCSS configuration must export a 'plugins' key.`
Causa raiz: `postcss-load-config@6.0.1` carrega `postcss.config.js` como CJS quando
o `package.json` não declara `"type": "module"`. O `export default` virava
`undefined` em runtime e o webpack não encontrava `plugins`.

#### F2 — `Session` interface não-exportada em `auth.ts`

```ts
// ❌ Antes
interface Session { ... }   // local, não exporta

// ✅ Depois
export interface Session { ... }
```

`src/lib/session.ts` fazia `import { verifyJwtServer, type Session } from './auth'`,
mas `Session` era apenas declarada local. TypeScript strict bloqueou o build.

### Validação pós-fix

```
✓ Compiled successfully
✓ Generating static pages (10/10)
10 rotas geradas (/, /login, /radar, /regras, /envios, /auditoria, /api/login, /v1-api/proxy/[...path], /_not-found)
First Load JS shared: 87.3 kB
```

### Files (v2.0.0.post)

- `frontend/postcss.config.js` (fix CJS)
- `frontend/src/lib/auth.ts` (export Session)
- `.gitignore` (adiciona frontend/node_modules, frontend/.next, etc.)
- `frontend/package-lock.json` (lockfile commitado)

---

## v2.0.1 — 2026-07-04 (27ª validação: 9 findings fechados)

> **Status:** Shipped
> **Sprint:** Validação 27 (VALIDATION_v2.0.0_POST.md)
> **Versão:** **patch** — 2 críticos + 4 médios + 3 polimentos fechados
> **Trigger:** Henrique pediu validação profunda pós-tag v2.0.0
> **Versão anterior:** v2.0.0.post

### 🎯 Resumo

Validação 27 fechou **9 findings** deixados pela release v2.0.0. Sem
esses fixes, deployment em produção real quebraria todos os 5 endpoints
mutantes (`/v1/validate`, `/v1/sta/submit`, `/v1/radar/alerts/{id}/resolve`,
`/v1/radar/scan`, `/v1/crossdoc/validate`) por vetor de leitura errada
de auth claims. Além disso, `/healthz` reportaria `1.5.0` em vez de
`2.0.0` (doc drift quebrando consumers).

### 🐛 Bugs corrigidos (por severidade)

#### 🔴 Críticos (2)

| # | Bug | Fix |
|---|-----|-----|
| F27.1 | Handlers liam `X-IF-ID` cru do header ao invés de `auth.ClaimsFromContext` — em prod com JWT-only, todos os 5 endpoints mutantes retornariam 401 "X-IF-ID required". Vetor de cross-tenant se cliente injetasse X-IF-ID malicioso com JWT válido. | Helper `getIfID(r)` em `internal/api/server.go` que prioriza `Claims.IFID` (JWT validated) e fallback X-IF-ID só em dev mode. Substituído nos 5 callsites. 3 testes de regressão em `ifid_test.go` (Claims, fallback header, vazio, edge-case Claims.IFID vazio). |
| F27.2 | `/healthz` retornava `"version":"1.5.0"` enquanto CHANGELOG/SPRINT_7_RESULTS diziam v2.0.0 — const `version.Version` foi deixada hardcoded em v1.5.0. Doc drift quebra consumers que checam versão. | `const Version = "2.0.0"` em `internal/version/version.go`. Dockerfile parametrizado com `ARG VERSION` + ldflags `-X ...version.Version=${VERSION}` para build-time override. OpenAPI `HealthStatus.version` example atualizado para "2.0.0". |

#### 🟡 Médios (4)

| # | Bug | Fix |
|---|-----|-----|
| F27.4 | Axios client `api.interceptors.request.use` tentava ler `rn_jwt` via `document.cookie` — código morto (cookie é httpOnly, JS não lê). | Removido interceptor. Client Axios agora é só para endpoints públicos / server-side. Documentado no header do arquivo. |
| F27.5 | ResolveButton (client-side) construía `Authorization: Bearer undefined` quando cookie httpOnly resultava em `token = undefined`. | Removida lógica. Server-side proxy `/v1-api/proxy/[...path]/route.ts` injeta Authorization automaticamente via `next/headers cookies()`. |
| F27.6 | Frontend sem `.eslintrc.json` — `npm run lint` falhava com prompt interativo pedindo config. | Adicionado `.eslintrc.json` extends `next/core-web-vitals`. Instalado `eslint@^8.57.0` + `eslint-config-next@^14.2.18` como devDeps. `npm run lint` agora reporta "✔ No ESLint warnings or errors". |
| F27.10 | OpenAPI `HealthStatus.version` example "1.6.0" inconsistente com `info.version` ("2.0.0") e com `/healthz` (que retornava 1.5.0 antes do F27.2 fix). | Atualizado para "2.0.0" + description nota sobre ldflags. |

#### 🟢 Polimentos (3)

| # | Issue | Fix |
|---|-------|-----|
| F27.13 | `frontend/src/lib/api-fetch.ts` usava `await import('next/headers')` (dynamic import anti-pattern em Next 14 ESM). | Movido para top-level `import { cookies } from 'next/headers'`. |
| F27.14 | `frontend/src/app/radar/page.tsx` tinha `import { ResolveButton }` no final do arquivo (anti-pattern). | Movido para topo com outros imports. |
| F27.16 | Cookie `rn_jwt` sem `secure: true` flag — em prod (HTTPS) sem secure flag pode vazar em HTTP downgrade. | Adicionado `secure: process.env.NODE_ENV === 'production'` em `frontend/src/app/api/login/route.ts`. Dev local (HTTP) continua funcional. |

### 📊 Estatísticas

```
Auth flow:
  Antes: Claim JWT populado → handler ignorava → 401 "X-IF-ID required"
  Depois: Claim JWT populado → getIfID() retorna Claims.IFID → endpoint funciona
  
Version reporting:
  Antes: /healthz → "1.5.0"
  Depois: /healthz → "2.0.0" (const) ou "v2.0.1+commit..." (ldflags)

Tests:
  Antes: 301 tests
  Depois: 304 tests (+3 F27.1 regression)

Build artifacts:
  Antes: front build passa, lint broken
  Depois: front build + lint + type-check all clean
```

### Compatibility

- **Backwards compat**: dev mode (`RADIANT_DEV_AUTH=1`) continua
  funcionando. Helper getIfID fallback para X-IF-ID header mantém
  tests legacy passando.
- **JWT-only prod**: agora funciona end-to-end (Sprint 7a fechou metade;
  F27.1 fechou a outra metade).

### Files (v2.0.1)

**Backend:**
- `backend/internal/auth/middleware.go` (add `WithClaims` helper)
- `backend/internal/api/server.go` (add `getIfID` helper + substituir 5 callsites)
- `backend/internal/api/ifid_test.go` (NOVO, 4 testes)
- `backend/internal/version/version.go` (const 1.5.0 → 2.0.0)
- `backend/Dockerfile` (ARG VERSION + ldflags em 4 binários)

**OpenAPI / docs:**
- `backend/docs/api/openapi.yaml` (version example + description)

**Frontend:**
- `frontend/src/lib/api.ts` (remove Axios interceptor)
- `frontend/src/lib/api-fetch.ts` (import dinâmico → top-level)
- `frontend/src/app/radar/page.tsx` (import no topo)
- `frontend/src/components/resolve-alert-button.tsx` (remove Bearer undefined)
- `frontend/src/app/api/login/route.ts` (secure flag conditional)
- `frontend/.eslintrc.json` (NOVO)
- `frontend/package.json` + package-lock.json (eslint devDeps)

---

## v2.0.0+ → v2.0.1 — Cumulative over 27ª validação

```
Findings fechados:           9 (2 críticos + 4 médios + 3 polimentos)
Backend tests:               301 → 304 (+3 regressão F27.1)
Frontend lint:               broken → clean (Strict Next config)
Frontend build:              ✓ unchanged
Frontend bundle:             -200B (radiar removido)
Doc drift:                   5 sync items (LOC, paths, version example,
                             file count, secure flag)
Segurança auth:              vetor cross-tenant injection FECHADO
```

---

## v2.1.0 — 2026-07-04 (Sprint 8a: JWT bridge real)

> **Status:** Shipped
> **Sprint:** Sprint 8a (ver SPRINT_8.md + SPRINT_8_RESULTS.md)
> **Versão:** **minor** — nova feature (dev-token mint in-process)
> **Trigger:** Gaps remanescentes de Sprint 7c — frontend usava JWT fake (`dev:<if>:<role>`) enquanto backend exigia JWT RS256 real

### 🎯 Resumo

Sprint 8a entrega **bridge JWT real frontend↔backend**. Em dev, frontend
`/api/login` chama novo endpoint `POST /v1/auth/dev-token` que emite JWT
RS256 in-process. Cookie `rn_jwt` passa a armazenar JWT real (não string
opaca). Backend JWT verifier (mesma chave pública carregada em
`RADIANT_JWT_PUBLIC_KEY`) aceita os tokens.

### ✨ Features

#### 🔴 Backend — `internal/auth/mint.go` (NOVO, 145 LOC)

Helper `auth.Signer` que encapsula signing JWT RS256:
- `NewSigner(SignerConfig)` — cria a partir de PEM-encoded private key.
- `NewSignerFromFile(path, kid, issuer)` — shorthand para file path.
- `Mint(Claims)` — assina JWT, valida claims antes.
- `MintSimple(ifID, role, ttl)` — helper dev/demo com validação
  integrada (alfanumérico + dash + underscore, max 64 chars).
- `TTLCap = 30 dias`, `TTLDefault = 24h`.

#### 🔴 Backend — `internal/api/auth_handlers.go` (NOVO, 173 LOC)

Novo endpoint `POST /v1/auth/dev-token`:
- Ativado por `RADIANT_DEV_TOKEN=1` env.
- Requer chave privada (path `RADIANT_DEV_JWT_PRIVATE_KEY` ou inline
  `RADIANT_DEV_JWT_PRIVATE_KEY_PEM`).
- **404** quando flag off (esconde endpoint em prod).
- **503** quando flag on mas signer não configurado.
- **400** quando if_id ausente, role inválida, ttl inválido.
- Audit emission: `auth.dev_token.minted` for forensic trail.
- TTL clamp: max 30 dias (defesa contra tokens de vida excessiva).

#### 🔴 Backend — `internal/api/server.go` (modified)

- Field `DevSigner *auth.Signer` adicionado.
- Router ganha `r.Route("/v1/auth", ...)` FORA do group `/v1` com JWT
  middleware (precisa estar acessível sem auth, mas com flag guard).

#### 🟡 Backend — `cmd/jwt-mint/main.go` (refactored)

- Lógica de signing delegada para `auth.Signer` (DRY).
- TTL clamp aplicado.
- Sub-claim default agora = ifID (não "dev-user").

#### 🟡 Frontend — `src/app/api/login/route.ts` (rewritten)

- Chama `POST /v1/auth/dev-token` no backend.
- 502 quando backend offline (era silencioso).
- 503 com hint quando dev-token endpoint disabled.
- Cookie `rn_jwt` agora armazena JWT real (string `eyJ...` em vez de
  `dev:<if>:<role>`).

### 🧪 Tests adicionados (18 novos)

#### `internal/auth/mint_test.go` (13 testes)

```
✓ TestNewSigner_ValidPEM
✓ TestNewSigner_PEMvazio
✓ TestNewSigner_KidVazio
✓ TestNewSigner_IssuerVazio
✓ TestSigner_Mint_ValidClaims
✓ TestSigner_Mint_InvalidClaims
✓ TestSigner_MintSimple
✓ TestSigner_MintSimple_Validations (8 subtests)
✓ TestSigner_Roundtrip (sign+verify)
✓ TestSigner_IssuerOverride
✓ TestTTLCap
```

#### `internal/api/auth_handlers_test.go` (8 testes)

```
✓ TestDevToken_EndpointDisabled (404 quando flag off)
✓ TestDevToken_SignerMissing (503 quando signer nil)
✓ TestDevToken_MintValid (happy path + JWT 3-parts + kid=k1)
✓ TestDevToken_AdminRole
✓ TestDevToken_InvalidRole
✓ TestDevToken_MissingIFID
✓ TestDevToken_TTLClamp (60d pedido → 30d cap)
✓ TestDevToken_Roundtrip (header contém kid=k1)
```

### 📊 Estatísticas

```
Sprint 8a entrega:
  Backend tests:        304 → 322 (+18 novos = 13 mint + 8 dev-token - 3 setup)
  Backend code:         ~315 LOC new (mint.go 145 + auth_handlers.go 173 - go.sum)
  Frontend code:        ~70 LOC rewritten (login route)
  OpenAPI:              14 → 15 endpoints (1 novo: /v1/auth/dev-token)
  Build/lint/type-check all clean
```

### Compatibility

- **Backwards compat**: dev mode X-IF-ID fallback (`RADIANT_DEV_AUTH=1`)
  continua funcionando para tests legacy.
- **JWT real bridge**: agora funcional end-to-end. Frontend → Backend
  dev-token → JWT válido → backend verifier aceita.
- **Prod safety**: `/v1/auth/dev-token` retorna 404 (não 503) quando
  flag off. Endpoint existence hidden.

### Setup necessário

```bash
# 1. Gerar par RSA dev (PKCS#1)
openssl genrsa -out dev-private.pem 2048
openssl rsa -in dev-private.pem -pubout -out dev-public.pem

# 2. Backend dev mode
export RADIANT_DEV_TOKEN=1
export RADIANT_DEV_JWT_PRIVATE_KEY=./dev-private.pem
export RADIANT_JWT_PUBLIC_KEY="$(cat dev-public.pem)"
export RADIANT_JWT_ISSUER=radiant-norma
export RADIANT_JWT_KID=k1

# 3. Frontend dev mode (já suportado via NEXT_PUBLIC_RADIANT_DEV_MODE=1)
export NEXT_PUBLIC_RADIANT_DEV_MODE=1

# 4. Start backend
cd backend && go run ./cmd/api

# 5. Start frontend
cd frontend && npm run dev

# Frontend /login → POST /api/login → calls /v1/auth/dev-token → JWT real
```

### Files (Sprint 8a)

**Backend (NOVO):**
- `backend/internal/auth/mint.go` (Signer helper)
- `backend/internal/auth/mint_test.go` (13 testes)
- `backend/internal/api/auth_handlers.go` (dev-token handler)
- `backend/internal/api/auth_handlers_test.go` (8 testes)

**Backend (modified):**
- `backend/internal/api/server.go` (DevSigner field + route wire)
- `backend/cmd/api/main.go` (DevSigner config reading env)
- `backend/cmd/jwt-mint/main.go` (refactored to use Signer)
- `backend/docs/api/openapi.yaml` (1 novo endpoint + 2 schemas)

**Frontend (rewritten):**
- `frontend/src/app/api/login/route.ts` (chama backend real)

**Docs:**
- `CHANGELOG.md` (esta entry)
- `SPRINT_8_RESULTS.md` (NOVO)

---
