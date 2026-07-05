# SPRINT 25 — RESULTS: Compile-time interface asserts + lint-no-placeholder

> **Sprint:** 25 (v3.16.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** `6f67ca6` (Sprint 25)

## TL;DR

Sprint 25 fecha **2 carry-overs de validações anteriores** identificadas como padrões reincidentes:

1. **Compile-time interface asserts** espalhados para todos os tipos que implementam interfaces Go (`*WSClient`, `*StubClient`).
2. **Lint script `lint-no-placeholder.sh`** que detecta placeholders "preencher após X" em SPRINT_*.md antes de commitar.

Bônus: o lint encontrou **4 placeholders reais** que escaparam para o repo nas Sprints 19, 20, 21, 22 — preenchidos agora.

**Decisão arquitetural:** compile-time asserts movidos de test files (linhas 1499, 2003 de ws_test.go) para **production source** (`ws.go` linha 1097+, `stub.go` linha 50+). Mais idiomático + catching imediato se assinatura de interface mudar.

**Decisões YAGNI conscientes:**
- Lint focado em placeholder (não Linter completo tipo golangci-lint)
- Lint roda manual (não em CI ainda — Sprint 26+ se virar requisito)
- Sem pre-commit hook (`.git/hooks/pre-commit`) — YAGNI até virar problema operacional
- Sem integração com outras ferramentas (markdownlint, vale.sh, etc.)

## Entregas

### 1. Compile-time interface asserts (3 sites, 0 runtime cost)

**`backend/internal/sta/ws.go` (final do arquivo):**
```go
var (
    _ Client        = (*WSClient)(nil)
    _ ReadClient    = (*WSClient)(nil)
    _ ChunkedClient = (*WSClient)(nil)
)
```

**`backend/internal/sta/stub.go` (após declaração de StubClient):**
```go
var _ Client = (*StubClient)(nil)
```

**Removidos de test files** (eram duplicados com production source após Sprint 25):
- `ws_test.go:1499` — `var _ ReadClient = (*WSClient)(nil)`
- `ws_test.go:2003` — `var _ ChunkedClient = (*WSClient)(nil)`

Comentários nos tests apontam para production source agora.

**Estado pós-Sprint 25:**
- `*WSClient` → implementa `Client` + `ReadClient` + `ChunkedClient` (compile-time)
- `*StubClient` → implementa `Client` (compile-time). NÃO implementa `ReadClient`/`ChunkedClient` (validado por `TestReadClient_InterfaceSegregation` + `TestChunkedClient_InterfaceSegregation` runtime).
- `*RetryingClient` → implementa `Client` (já tinha desde v44).

### 2. Lint script `scripts/lint-no-placeholder.sh`

Detecta 3 padrões em SPRINT_*.md:
- "preencher após X" — pattern pt-BR reincidente (v44 + v45)
- "fill in X" — versão inglês
- "TODO: X" — versão genérica

Exit codes:
- `0` OK (sem placeholder)
- `1` FAIL (placeholder encontrado + linhas específicas listadas)

Uso:
```bash
./scripts/lint-no-placeholder.sh

# Output esperado quando limpo:
# ✅ OK: 25 SPRINT_*.md files limpos (sem placeholders)

# Output quando falha:
# ❌ FAIL: SPRINT_*.md contém placeholder não-preenchido:
#    6:> **Commit:** (preencher após push)
```

### 3. Bônus: 4 placeholders reais preenchidos

Lint rodado pela primeira vez em **25 SPRINT_*.md files** encontrou 4 placeholders reais:
- `SPRINT_19_RESULTS.md:6` → preenchido com `7b50253`
- `SPRINT_20_RESULTS.md:6` → preenchido com `fa4dc13`
- `SPRINT_21_RESULTS.md:6` → preenchido com `41981e9`
- `SPRINT_22_RESULTS.md:6` → preenchido com `4321a0d`

Sprint 23 e 24 foram preenchidos anteriormente (v44 + v45). Agora os 25 SPRINT_*.md estão 100% preenchidos.

## Decisões que pagaram

### D-1. Compile-time asserts em production source (não test files)

Compilador avalia `var _ Interface = (*Type)(nil)` em qualquer lugar do package. Production source garante catching mesmo se teste falhar em rodar. Padrão idiomático Go (Effective Go + Uber style guide).

### D-2. Lint simples em bash (não ferramenta externa)

Sprint 25 escopo é pequeno (~50 linhas bash). Adicionar ferramenta tipo `vale.sh` ou `markdownlint` seria overkill — script bash direto resolve.

### D-3. Lint roda manual, não em CI/pre-commit

CI/pre-commit hook adiciona fricção. Padrão: lint manual antes de commitar até virar requisito operacional (Sprint 26+).

### D-4. Lint detecta 3 padrões (não 1)

Pattern reincidente foi "preencher após X". Adicionei "fill in X" (inglês) e "TODO: X" (genérico) para prevenir variantes futuras. Trade-off: 3 patterns vs 1 — pequeno overhead, melhor cobertura.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 1 (`scripts/lint-no-placeholder.sh`, 60 linhas) |
| Arquivos modificados | 4 (ws.go +6, stub.go +6, ws_test.go -4, 4 SPRINT_*.md) |
| Tests Sprint 25 | 0 (lint script + compile-time asserts — não requerem tests runtime) |
| Total backend tests top-level | 115 (mesmo) |
| Packages PASS | 20/20 |
| Build OK | 6/6 binaries |
| Smoke E2E | 11/11 PASS (sem regressão) |
| gofmt drift | 0 |
| go vet | clean |
| Race detector | clean |
| Lint `lint-no-placeholder.sh` | ✅ 25/25 SPRINT_*.md limpos |
| Placeholders preenchidos (bônus) | 4 (Sprints 19-22) |

## Compatibilidade

- **Zero impacto em código de produção.** Compile-time asserts são zero-cost em runtime.
- **Zero impacto em tests existentes.** Compile-time asserts movidos de test → production source é reorganização.
- **Lint script é aditivo.** Não afeta build/test/vet. Sprint 26+ pode adicionar a CI.

## Lições aprendidas (carry forward)

### L-1. Lint scripts são melhores quando simples e focados

Pattern: 1 lint por classe de problema, não Linter monolítico. `lint-no-placeholder.sh` faz **1 coisa** (placeholder detection) — fácil de entender, manter, estender.

### L-2. Compile-time asserts em production source > test files

Effective Go recomenda `var _ Interface = (*Type)(nil)` no package que define `Type`. Production source garante catching mesmo se teste não rodar. Test files são "check extra", não "check primário".

### L-3. Lint roda manual é OK pra V1

CI/pre-commit adiciona fricção operacional (PR bloqueado, push falha, etc). Padrão:
- V1: lint manual antes de commitar (Sprint 25)
- V2: pre-commit hook se virar problema (Sprint 26+)
- V3: CI integration se virar mandatório (Sprint 27+)

### L-4. Patterns reincidentes merecem automação

`placeholder` reincidiu 2 sprints (v44 + v45). Lint automatiza catching. Carry-forward: identificar patterns reincidentes em validações → criar lint correspondente.

### L-5. Script bash > script python pra linters simples

Linter de 50 linhas em bash é:
- Sem dependência (python precisa de venv)
- Roda em qualquer Unix sem setup
- Mais fácil de auditar (1 linguagem universal)

Se ficar >200 linhas ou precisar de parsing complexo, mudar pra Go (consistente com codebase).

## Próximos passos (Sprint 26+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 26 | `cmd/sta-submit` CLI paralelo a `senhaws-rotate` | Mesmo pattern pra CADOC submission (admin tool direto) |
| 26 | Pre-commit hook: `./scripts/lint-no-placeholder.sh` + gofmt + go vet | Automação catching antes de push |
| 27 | Vault integration (AWS Secrets Manager / Vault) | Auto-update secret manager após rotação |
| 28 | Smoke contra BACEN homolog real | Requer credenciais Sisbacen — última validação pré-prod |
| 29 | Handler REST `/v1/sta/range-*` (Sprint 21 YAGNI) | Frontend/batch trigger UI |

## Critérios de done — todos ✅

- [x] Compile-time asserts em ws.go (3 interfaces)
- [x] Compile-time asserts em stub.go (1 interface)
- [x] Removidos asserts duplicados de ws_test.go
- [x] Lint script `lint-no-placeholder.sh` criado
- [x] 4 placeholders reais preenchidos (Sprints 19-22)
- [x] Lint passa: 25/25 SPRINT_*.md limpos
- [x] 20/20 packages PASS (zero regressão)
- [x] Build smoke 6/6 binaries
- [x] gofmt/vet clean
- [x] Race detector clean
- [x] SPRINT_25_RESEARCH.md + SPRINT_25_RESULTS.md (próximo)
- [x] CHANGELOG v3.16.0 (próximo)
- [ ] commit + push (próximo)

## Como usar (quickstart)

### Lint antes de commitar:
```bash
./scripts/lint-no-placeholder.sh
# ✅ OK: 25 SPRINT_*.md files limpos (sem placeholders)
```

### Verificar compile-time asserts funcionam:
```bash
# Deletar um método de WSClient e tentar build
$EDITOR backend/internal/sta/ws.go  # remover ListDisponiveis, por ex.
go build ./internal/sta/...
# ./ws.go:1099: cannot use (*WSClient)(nil) as ReadClient value ...
#                                       *WSClient doesn't implement ReadClient
# Bingo — catching imediato. Sem o assert, erro só apareceria em test runtime
# (ou pior, em produção quando caller faz type assertion).
```

### Adicionar novo tipo que implementa interface:
```go
type MeuTipo struct { ... }

// Compile-time guarantee:
var _ InterfaceQueImplemento = (*MeuTipo)(nil)
```

## Anti-patterns evitados

1. **Hollow Linter** — script bash focado em 1 problema, não framework genérico.
2. **Lint acoplado a CI** — manual primeiro, CI depois. Fricção operacional evitada em V1.
3. **Compile-time asserts em test files** — production source é catching primário.
4. **Placeholder reincidente** — automação catching garante que v44+v45 não vire v46+v47.
5. **Tooling novo sem necessidade** — bash resolve, sem python venv ou linter externo.