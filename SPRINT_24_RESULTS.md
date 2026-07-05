# SPRINT 24 — RESULTS: cmd/senhaws-rotate (standalone CLI)

> **Sprint:** 24 (v3.14.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** `0fb41a6` (Sprint 24) → ver VALIDAÇÃO 45 para commits subsequentes

## TL;DR

Sprint 24 fecha o **standalone CLI `senhaws-rotate`** que dá utilidade operacional
ao pacote `internal/senhaws` (Sprint 23). Admin IF pode agendar rotação
automática de credenciais Sisbacen via cron job, sem deployar API ou UI.

**Decisão arquitetural:** CLI tool independente (não handler REST), composto por
3 subcomandos (`check`/`rotate`/`info`). Padrão consistente com codebase
(`cmd/seed`, `cmd/jwt-mint`, `cmd/worker`, `cmd/radar`) — usa `flag` stdlib
+ `slog`, sem dependências adicionais.

**Decisões YAGNI conscientes:**
- Sem retry (SenhawsClient é failure-fast por design — propagação consistente).
- Sem persistência local (secret manager é responsabilidade do caller).
- Sem TLS client cert (BACEN não exige).
- Sem dry-run (admin usa `check` antes de `rotate`).
- Sem integração vault automática (Sprint 27+).
- Sem Web UI (IF tem 1-2 operadores, não justifica).

## Entregas

### 1. Binário `cmd/senhaws-rotate` (~310 linhas)

3 subcomandos:

| Comando | O que faz | Exit codes |
|---|---|---|
| `check` | `ConsultarVencimento` → imprime dias restantes | 0 (> threshold), 1 (≤ threshold) |
| `rotate` | `GerarSenhaRandom` + `AlterarSenha` → imprime nova senha no stdout | 0 (sucesso), 3 (BACEN rejeitou) |
| `info` | Imprime config mascarada + status BACEN | 0 (BACEN OK), 2/3 (erros) |

### 2. Flags + env vars (consistente com cmd/api)

| Flag | Env var | Default | Notas |
|---|---|---|---|
| `--base-url` | `SENHAWS_BASE_URL` | (vazio) | Obrigatório. https://www9.bcb.gov.br/senhaws (homol) ou www3 (prod) |
| `--user` | `SENHAWS_USER` | (vazio) | Obrigatório. Formato UUUUUDDDD.operador |
| `--password` | `SENHAWS_PASSWORD` | (vazio) | **Env var preferida** — flag aparece em `ps aux` |
| `--timeout` | `SENHAWS_TIMEOUT` | 30s | Timeout HTTP |
| `--max-days` | `SENHAWS_MAX_DAYS` | 7 | Threshold para `check` exit 1 |
| `--quiet` | — | false | Silencia logs de stderr |
| `--allow-insecure-http` | — | false | Apenas testes dev. NUNCA produção. |

### 3. Exit codes discriminados

```
0  sucesso
1  erro genérico / precisa rotacionar (check)
2  erro de validação client-side (input inválido)
3  erro BACEN (rejeição formal — caller investiga)
```

Permite cron scripts discriminar retry policy:
- `senhaws-rotate check` exit 1 → script rotaciona automaticamente
- exit 2 → admin corrige config (não retry)
- exit 3 → admin investiga log BACEN (não retry)

### 4. Segurança de output

- **`check`:** imprime `dias_vencimento=N status=ok threshold=M` no stdout. Sem dados sensíveis.
- **`rotate`:** imprime `nova_senha=...` no stdout. Caller DEVE redirecionar `> /tmp/newpass.txt`
  e armazenar em secret manager antes de remover arquivo. NÃO usar em interactive shell
  (history leak).
- **`info`:** imprime `user=12***.fulano` (mascarado, mantém prefixo + sufixo). Senha nunca impressa.
- **Stderr:** apenas logs estruturados, sem senha.

### 5. Build + binário standalone

```bash
go build -o bin/senhaws-rotate ./cmd/senhaws-rotate/
# Resultado: binário ~10MB, zero deps runtime além de Go stdlib + pacote interno senhaws
```

## Decisões que pagaram

### D-1. CLI e não handler REST

Tool de operação admin, não de UI de usuário. Composable em shell (`check && rotate`).
Sem dependência de API estar UP pra rotacionar (decoupling operacional).

### D-2. `flag` stdlib (não cobra)

Codebase tem 6 binários (`seed`, `jwt-mint`, `seed-sprint8c`, `radar`, `worker`, `_verify`)
— todos com `flag` stdlib. Adicionar cobra seria 1 dep nova + pattern drift.

### D-3. Exit codes discriminados (0/1/2/3)

Cron scripts podem discriminar retry policy sem parsear mensagem. Pattern consistente
com Unix convention (0=ok, 1=genérico, 2=misuse, 3+=específicos).

### D-4. `--allow-insecure-http` (consistente com WSConfig)

Tests precisam rodar com httptest (HTTP). Production usa HTTPS. Flag explícita
mesma naming convention do `WSConfig.AllowInsecureHTTP` (validação 39). Default
false — NUNCA setado em prod.

### D-5. Sem retry wrapper

SenhawsClient é failure-fast por design (Sprint 23 decisão). CLI herdou.
Se falhar, admin re-executa. Retry mascararia bugs (caller esqueceu de atualizar
secret manager e fica em loop infinito).

### D-6. Subcomando `info` separado (não `--verbose` flag)

`info` tem semântica distinta: imprimir config + status. Não é "verbose mode" do check.
Separar em subcomando torna uso auto-documentado.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (`main.go` 314 linhas + `main_test.go` 332 linhas) |
| Packages novos | 0 (cmd/* não conta como package) |
| Testes Sprint 24 | 16 top-level (todos PASS) |
| Total backend tests top-level | **96 → 112** (+16) |
| Packages PASS | **20/20** (era 19, +1 = cmd/senhaws-rotate) |
| Build OK | **6/6 binaries** (era 5, +1 = senhaws-rotate) |
| Smoke E2E | 11/11 PASS (sem regressão) |
| Coverage cmd/senhaws-rotate | 60.7% (CLI tool — fluxos principais cobertos) |
| gofmt drift | 0 |
| go vet | clean |
| Race detector | clean |

## Compatibilidade

- **Novo binário `cmd/senhaws-rotate`.** Zero impacto em código existente.
- **Pacote `internal/senhaws` inalterado** (Sprint 23). CLI apenas wrappea.
- **Não wired em `cmd/api/main.go`** — CLI é independente (decoupling).
- **Nenhum handler REST adicionado** — admin tool direto.
- **Nenhum workflow existente alterado** — adição pura.

## Lições aprendidas (carry forward)

### L-1. CLI tools precisam de flag `--allow-insecure-http` para testes

Tests com `httptest.NewServer` retornam HTTP. CLI tool que valida HTTPS strict
precisa de escape hatch. Pattern: copiar `AllowInsecureHTTP` do WSConfig para
qualquer nova CLI que wrappea client HTTPS-strict.

### L-2. Exit codes Unix-like são poderosos para automation

4 exit codes (0/1/2/3) permitem cron scripts discriminarem retry policy sem
parsear stderr. Pattern: usar convention Unix sempre que CLI for usado em automation.

### L-3. `usage()` em stderr, output em stdout

Convenção Unix: usage/help vai em stderr (file descriptor 2), resultados em stdout
(fd 1). Permite `cmd --help 2>&1 | less` e `cmd 2>/dev/null` separadamente.

### L-4. Mascaramento de user mantém prefixo + sufixo

`maskUser("123450001.fulano")` → `"12***.fulano"` — mostra primeiros 2 chars + operador.
Defesa contra screenshot/log acidental. Útil em qualquer output que mostra user identifier.

### L-5. captureStdout/Stderr helper para CLI tests

Tests de CLI precisam capturar fd 1 e fd 2 separadamente. Helper pattern:

```go
old := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w
defer func() { os.Stdout = old }()
// run code
w.Close()
buf.ReadFrom(r)
```

Reutilizável em qualquer CLI Go test.

## Próximos passos (Sprint 25+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 25 | Compile-time asserts para `*WSClient` + `*StubClient` | Espalhar pattern introduzido na validação 44 |
| 26 | `cmd/sta-submit` CLI paralelo a `senhaws-rotate` | Mesmo pattern para envio de CADOC (sem UI) |
| 27 | Vault integration (AWS Secrets Manager / Vault) | Auto-update secret manager após rotação |
| 28 | Handler REST `/v1/senhaws/...` (se virar requisito) | Frontend UI para admin |
| 29 | Smoke contra BACEN homolog real | Requer credenciais Sisbacen — última validação pré-prod |

## Critérios de done — todos ✅

- [x] CLI com 3 subcomandos (check/rotate/info)
- [x] Flags + env vars consistentes com codebase
- [x] Exit codes discriminados (0/1/2/3)
- [x] Mascaramento de user em output
- [x] 16 testes top-level (httptest + unit + helpers)
- [x] 20/20 packages PASS (zero regressão)
- [x] Build smoke 6/6 binaries
- [x] gofmt/vet clean
- [x] SPRINT_24_RESEARCH.md + SPRINT_24_RESULTS.md
- [ ] CHANGELOG v3.14.0 (próximo passo)
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Hollow CLI stub** — CLI tem comportamento real (3 subcomandos funcionais + exit codes + masked output).
2. **Senha em stderr/log** — senha só vai em stdout (subcomando `rotate`), caller controla captura.
3. **Senha em `ps aux`** — flag `--password` documentada como "NÃO usar" — preferir env var.
4. **Retry mascara bug** — failure fast consistente com SenhawsClient.
5. **Wrapper vazio** — CLI não wrappea RetryingClient (decisão consciente).
6. **HTTPS check quebra tests** — `--allow-insecure-http` flag consistente com WSConfig.
7. **User em output sem máscara** — `maskUser()` esconde primeiros chars.

## Como usar (quickstart)

```bash
# 1. Setup (env vars preferidas)
export SENHAWS_BASE_URL="https://www9.bcb.gov.br/senhaws"  # homologação
export SENHAWS_USER="123450001.fulano"
export SENHAWS_PASSWORD="$ACTUAL_SISBACEN_PASSWORD"

# 2. Checar vencimento (cron diário)
senhaws-rotate check
# → dias_vencimento=30  status=ok  threshold=7
# → exit 0

# 3. Rotacionar (quando check indica expirando)
senhaws-rotate rotate > /tmp/newpass.txt
# exit 0
# /tmp/newpass.txt contém: senha_alterada=true  nova_senha=abc123...

# 4. Caller armazena em secret manager
aws secretsmanager update-secret --secret-id bacen/senha --secret-string file:///tmp/newpass.txt
rm /tmp/newpass.txt  # cleanup obrigatório

# 5. Próxima call STA usa senha nova automaticamente
```

## Resumo dos próximos passos (commit + push)

Vou:
1. Atualizar CHANGELOG.md com entry v3.14.0
2. Smoke final (full test + build + vet)
3. Commit + push origin