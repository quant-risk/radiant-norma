# VALIDAÇÃO 54 — Deep audit pós-v3.32.0 (Sprint 35 CI-Gate)

> **Data:** 2026-07-06
> **Versão alvo:** v3.32.0 (Sprint 35 — CI-Gate expandido)
> **Tipo:** patch (workflow hardening)
> **Trigger:** Solicitação Henrique — "validação profunda em tudo que você acabou de fazer"

## TL;DR

Sprint 35 (v3.32.0) entregou CI-Gate expandido com 11 steps, mas auditoria profunda encontrou **9 findings — 2 fechados nesta validação (v3.32.1 hotfix), 4 fechados por reverter F-54-H, 3 carry-over**.

**Bug crítico detectado:** drift check do step 3040 tinha `working-directory: backend` mas abria `CHANGELOG.md` (que está na RAIZ). **No CI, esse step SEMPRE falharia** com FileNotFoundError. Drift check que nunca rodou em produção é o pior tipo de gate morto.

## Findings — detalhamento

### F-54-A — drift check quebra em CI (HIGH → FECHADO)

**Severidade:** HIGH (gate morto = pior classe)
**Categoria:** Drift detection broken-by-design

**Bug:**
```yaml
- name: 3040 rule count drift check (Sprint 35)
  working-directory: backend  # ← CWD = backend/
  run: |
    CLAIMED=$(python3 -c "
    ...
    with open('CHANGELOG.md') as f:  # ← tenta abrir backend/CHANGELOG.md
        ...
    ")
```

**Sintoma:** FileNotFoundError → exit 1 → step fails → CI quebra.

**Como passou despercebido em v3.32.0:** Testei localmente com `python3 ...` rodando da raiz do repo (sem `working-directory: backend`), deu 126=126, OK. **Não testei com o step completo no working-directory real.**

**Diagnóstico (Validação 54):**
```bash
$ cd backend && python3 -c "open('CHANGELOG.md')"
FileNotFoundError: [Errno 2] No such file or directory: 'CHANGELOG.md'
```

**Fix:**
```yaml
with open('../CHANGELOG.md') as f:  # ← caminho relativo ao working-directory
```

**Verificação:**
```bash
$ cd backend && python3 -c "
import re
with open('../CHANGELOG.md') as f: text = f.read()
matches = [m.group(2) for line in text.split('\n') if 'Total 3040' in line 
            for m in [re.search(r'\*\*(\d+).+?(\d+)\*\*', line)] if m]
print(matches[0])
"
126
```

---

### F-54-B — `permissions:` não declarado (MED → FECHADO)

**Severidade:** MED (defense-in-depth)
**Categoria:** Workflow security

**Bug:** Sem `permissions:` declarado, GitHub Actions usa default = `write` (mais permissivo). Para CI que só precisa ler código, deveria ser `contents: read`.

**Fix:**
```yaml
permissions:
  contents: read
```

**Justificativa:** Workload CI = checkout + go build/test. Não precisa write em contents, issues, pull-requests, etc.

---

### F-54-C — `timeout-minutes` não declarado (MED → FECHADO)

**Severidade:** MED (custo)
**Categoria:** Workflow resource limits

**Bug:** Sem timeout, job pode rodar indefinidamente. Custo = runner $$/minuto.

**Fix:**
```yaml
timeout-minutes: 15
```

**Justificativa:** Total atual ~155s. Margem 5x. Se passar de 15min, alguma coisa está errada (loop infinito, deadlock test, hang de network).

---

### F-54-D — `concurrency:` não declarado (MED → FECHADO)

**Severidade:** MED (desperdício de quota)
**Categoria:** Workflow scheduling

**Bug:** Múltiplos PRs contra main rodam simultaneamente. Cada push subsequente dispara novo run mesmo se old runs não terminaram.

**Fix:**
```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

**Justificativa:** Cancela runs antigos quando novo commit chega. Economiza runner minutes (especialmente em pushes forçados de "fix typo" subsequentes).

---

### F-54-E — Build loop hardcoded (MED → FECHADO)

**Severidade:** MED (drift silencioso)
**Categoria:** Build maintenance

**Bug original (v3.32.0):**
```yaml
for bin in api worker radar jwt-mint secret-migrate senhaws-rotate sta-submit seed seed-sprint8c _verify; do
  echo "Building cmd/$bin..."
  go build -o /tmp/ci-bin-$bin ./cmd/$bin
done
```

Se novo `cmd/<novo>/` for adicionado sem editar YAML, **drift silencioso** — CI não pega. Falsa sensação de cobertura.

**Fix:**
```yaml
shopt -s nullglob
bins=(cmd/*/)
if [ ${#bins[@]} -eq 0 ]; then
  echo "❌ No cmd/*/ directories found"
  exit 1
fi
for dir in "${bins[@]}"; do
  bin=$(basename "$dir")
  echo "Building cmd/$bin..."
  go build -o /tmp/ci-bin-$bin "./cmd/$bin"
done
echo "All ${#bins[@]} binaries built OK"
```

**Justificativa:** Glob dinâmico = adicionou novo `cmd/<novo>/` → próximo CI builda automaticamente. **Fail-loud** se `cmd/` vazio (impossível mas paranoia).

---

### F-54-F — `runs-on: macos-latest` (INFO → CARRY-OVER)

**Severidade:** INFO (custo / não-crítico)
**Categoria:** Infrastructure

**Observação:** macos-latest = 2x mais caro que ubuntu-latest. Repo é Go puro com SQLite pure-Go (modernc.org), não tem deps macOS-specific.

**Não-fixado porque:**
1. Não estava em escopo da Validação 54 (que é hotfix de CI broken)
2. Trocar OS pode introduzir race conditions Linux-vs-macOS diferentes (CI preexistente valida em macOS, mudar agora cria vetor de regressão não-testado)
3. Decisão arquitetural que deve ser sprint dedicada

**Carry-over:** Adicionar em Sprint 36 (Observability) — quando vamos mexer em CI infra de qualquer jeito, é hora natural.

---

### F-54-G — coverage.txt não é artifact (INFO → CARRY-OVER)

**Severidade:** INFO (UX)
**Categoria:** Observability

**Observação:** `coverage.txt` é gerado mas descartado. Se CI falhar e quiser post-mortem de regressão de coverage, perde histórico.

**Não-fixado porque:** YAGNI por enquanto. Em Sprint 36 (Observability) adicionamos artifact upload junto com outras melhorias de visibilidade.

---

### F-54-H — race+cover combinados quebraram Coverage gates (LOW → REVERTED)

**Severidade:** LOW (meu próprio bug introduzido em F-54-H tentativa)
**Categoria:** Refactor regression

**Tentativa de fix em v3.32.1 draft:**
```yaml
- run: |
    go test -race -count=1 -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tee coverage.txt
```

**Problema:** `go tool cover -func` produz saída POR FUNÇÃO, não por pacote:
```
github.com/fortvna/radiant-norma/backend/internal/auditlog/log.go:42:		New		100.0%
```

Não tem `coverage: X.X% of statements` agregado. **Coverage gates (line 113/122/131) faziam `grep "internal/auditlog"` + pegavam PRIMEIRA match (100%)** em vez do agregado por pacote (90.8%).

**Sintoma:** Coverage gate sempre passaria porque pega primeira função = 100%. Falsa sensação de segurança = pior que sem gate.

**Fix:** **REVERTI** para dois steps separados (race + cover), conforme v3.32.0 original:
```yaml
- name: go test -race
  run: go test -race -count=1 ./...
- name: Coverage report
  run: go test -cover ./... | tee coverage.txt
```

**Custo:** ~155s duplicado por run (testes rodam 2x). Aceitável enquanto cobertura real funciona.

**Lição:** Optimização prematura quebrou invariante. **Testes devem validar invariantes, não otimizar tempo.** Se otimizar, validar manualmente parsing.

---

### F-54-I — Actions pinadas em major version (INFO → CARRY-OVER)

**Severidade:** INFO (supply-chain)
**Categoria:** Workflow security

**Observação:** `actions/checkout@v4` e `actions/setup-go@v5` pinados em major version. Supply-chain best-practice = SHA pin (`@v4.1.7` SHA exato).

**Não-fixado porque:**
1. Major version é convenção comum e aceitável para actions oficiais
2. SHA pin requer update manual periódico
3. Decisão security-vs-maintenance que merece discussão

**Carry-over:** Considerar quando Sprint 36 (Observability) ou em revisão trimestral de CI.

---

### F-54-J — Placeholder lint sem working-directory (INFO → ACEITO)

**Severidade:** INFO
**Categoria:** Workflow structure

**Observação:** Step `Placeholder lint (SPRINT_*.md)` não tem `working-directory`. Em CI, `bash scripts/lint-no-placeholder.sh` resolve relativo ao workspace root (que é repo root após checkout). Funciona.

**Aceito porque:** Inconsistência menor (todos os outros steps usam `working-directory: backend`). Lint script tem lógica própria de `REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"` que torna-o robusto.

---

### F-54-K — cmd/ packages com 0% coverage (INFO → ACEITO)

**Severidade:** INFO
**Categoria:** Test coverage

**Observação:** Pacotes `cmd/*` mostram 0.0% coverage. Normal — são main packages (entry points).

**Aceito porque:** Padrão Go idiomático. Testar main() requer refactor para extrair runnable. YAGNI.

---

## Resumo de fixes aplicados em v3.32.1

| Fix | Mudança | LOC |
|---|---|---|
| F-54-A | `open('CHANGELOG.md')` → `open('../CHANGELOG.md')` | 1 |
| F-54-B | Adicionado `permissions: contents: read` | 3 |
| F-54-C | Adicionado `timeout-minutes: 15` | 2 |
| F-54-D | Adicionado `concurrency: cancel-in-progress` | 3 |
| F-54-E | Build loop hardcoded → glob dinâmico | +9 |
| F-54-H | Revertido race+cover combinados | -2 |
| Header | Comentário Validação 54 | +9 |
| **Total** | Workflow | **+25 net** |

## Validação final

| Check | Resultado |
|---|---|
| YAML válido | ✅ |
| Steps | **11** |
| go vet | ✅ clean |
| gofmt | ✅ clean |
| Placeholder lint | ✅ 28 SPRINT_*.md limpos |
| Build 10 binaries (glob) | ✅ 10/10 |
| go test -race | ✅ 23/23 packages |
| go test -cover | ✅ auditlog=90.8%, radar=81.2%, audit/rules=70.8% |
| Drift check (com fix F-54-A) | ✅ **126 = 126** |

## Lição aprendida

**Drift check é o tipo de gate que pode estar "ali看起来 bonito" mas nunca ter rodado em produção.** O fato de eu ter testado v3.32.0 localmente (onde `working-directory: backend` não estava aplicado) e ter dado OK mascarou o bug. **Lesson:** validar com `act` localmente ou garantir que simulação reproduz EXATAMENTE o ambiente CI.

Padrão universal: **gate que nunca rodou = pior que sem gate** (falsa sensação de segurança). Auditorias profundas pós-release devem SEMPRE rodar o gate completo uma vez, não só validar logic isoladamente.

## Próximos passos

1. ✅ Commit v3.32.1 com fixes F-54-A..E
2. ✅ Tag v3.32.1 + push
3. ⏭️ Sprint 30 (PostgresRLS) — ativar `012_rls_policies.sql` + criar `014_rls_enforce.sql`

## Carry-over

| Finding | Para Sprint |
|---|---|
| F-54-F (Ubuntu) | Sprint 36 (Observability) ou sprint dedicada |
| F-54-G (artifact upload) | Sprint 36 |
| F-54-I (SHA pin) | Revisão trimestral CI |