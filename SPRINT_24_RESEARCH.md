# SPRINT 24 — Research: cmd/senhaws-rotate (standalone CLI)

> **Sprint:** 24 (v3.14.0)
> **Quando:** 2026-07-06
> **Pesquisador:** mavis
> **Status:** research completa, pronto pra implementação
> **Trigger:** SPRINT_23_RESULTS.md §"Próximos passos" Sprint 24 (admin tool wire-up) + VALIDAÇÃO 44 (CLI dá utilidade real ao SenhawsClient)

## 1. Contexto

Sprint 23 entregou o pacote `internal/senhaws` com `AlterarSenha` + `ConsultarVencimento`.
Sprint 24 dá **utilidade operacional**: CLI standalone que admin IF pode rodar em cron job
para gestão automática de credenciais Sisbacen.

**Caso de uso real:**
- Cron diário chama `senhaws-rotate check` → vê quantos dias faltam
- Se `< 7 dias`, operador roda `senhaws-rotate rotate` → nova senha gerada + alterada no BACEN
- CLI imprime nova senha no stdout (em formato seguro) → caller redireciona pra secret manager

**Por que CLI e não handler REST:**
- Tool de **operação admin**, não de UI de usuário
- IF tem 1-2 operadores de BACEN — não justifica UI
- CLI é composable: `senhaws-rotate check && senhaws-rotate rotate` em shell
- Sem dependência de API estar UP pra rotacionar (decoupling)

## 2. Escopo da Sprint 24

### 2.1 Subcomandos

| Comando | O que faz |
|---|---|
| `senhaws-rotate check` | Chama `ConsultarVencimento`. Imprime dias restantes. Exit 0 se > 7, exit 1 se ≤ 7 (rotacionar). |
| `senhaws-rotate rotate` | Gera senha random + chama `AlterarSenha`. Imprime nova senha no stdout (formato seguro). Exit 0 em sucesso, exit 1 em erro. |
| `senhaws-rotate info` | Imprime config (BaseURL, User mascarado) + status do servidor BACEN (via check). Útil pra debug. |

### 2.2 Flags

```
--base-url    (env SENHAWS_BASE_URL)  https://www9.bcb.gov.br/senhaws (homol) | www3 (prod)
--user        (env SENHAWS_USER)       formato UUUUUDDDD.operador
--password    (env SENHAWS_PASSWORD)   senha Sisbacen ATUAL — NÃO log (F13.8)
--timeout     (env SENHAWS_TIMEOUT)    default 30s
--max-days    (env SENHAWS_MAX_DAYS)   threshold para check exit code, default 7
--quiet                            silencia logs (apenas stdout)
```

### 2.3 Output format

**`check`:**
```
dias_vencimento=30  status=ok  threshold=7
```

Exit code:
- 0: dias > max-days
- 1: dias ≤ max-days (precisa rotacionar) OR erro BACEN

**`rotate`:**
```
senha_alterada=true  nova_senha=abc123def456789012345678901234ab
```

Exit code:
- 0: sucesso
- 1: erro (BACEN rejeitou, rede falhou, validação client-side falhou)

**`info`:**
```
base_url=https://www9.bcb.gov.br/senhaws
user=12***01.fulano  (mascarado)
timeout=30s
bacen_status=ok  dias_vencimento=30
```

Exit code:
- 0: BACEN respondeu
- 1: erro de comunicação

### 2.4 Segurança de output

**`rotate` imprime senha nova no stdout.** Isso é intencional:
- Caller (cron script) captura stdout → escreve em secret manager
- Stderr tem apenas logs estruturados (sem senha)
- Senha vai por stdout (NÃO por logger) para evitar log retention

**Risco:** stdout de processo fica em histórico do shell (`~/.bash_history` se caller não usa `HISTCONTROL=ignorespace`). Documentar: caller DEVE usar `senhaws-rotate rotate > /tmp/newpass.txt` e limpar arquivo após usar.

**Defesa:** máscara de senha no info command (user mascarado). Senha em si só vai raw no `rotate` (necessário pra caller capturar).

## 3. Decisões de design

### 3.1 `flag` stdlib (não cobra)

**Decisão:** usar `flag` stdlib.

**Razão:** codebase tem `cmd/seed`, `cmd/jwt-mint`, `cmd/seed-sprint8c`, `cmd/radar`, `cmd/worker`, `cmd/_verify` — todos usam `flag` stdlib. Padrão consistente. Adicionar cobra seria 1 dependência nova + pattern drift.

### 3.2 Subcomandos via `os.Args[1]`

**Decisão:** dispatch manual em `switch os.Args[1]`.

**Razão:** mesma razão de D-3.1. Cobra é overkill para 3 subcomandos.

### 3.3 Exit codes específicos

**Decisão:**
- 0: sucesso (check: dias > threshold, rotate: senha alterada, info: BACEN respondeu)
- 1: erro genérico
- 2: erro de validação client-side (input inválido)
- 3: erro BACEN (rejeição formal)

**Razão:** cron scripts podem discriminar retry. Senha com 0 dias vencidos = exit 1 (rotacionar). Senha com erro de input = exit 2 (admin corrige). Senha com 401 do BACEN = exit 3 (senha atual errada, admin investiga).

### 3.4 Sem retry wrapper (consistente com SenhawsClient)

**Decisão:** CLI não tem `--retry` flag.

**Razão:** SenhawsClient é failure-fast (decisão YAGNI Sprint 23). CLI herdou essa decisão. Se falhar, admin re-executa.

### 3.5 Sem persistência local

**Decisão:** CLI é stateless.

**Razão:** secret manager é responsabilidade do caller. CLI apenas imprime nova senha → caller armazena. Sem SQLite, sem state file.

### 3.6 Senha random: `GerarSenhaRandom` ou flag `--password-stdin`?

**Decisão:** default é `GerarSenhaRandom` (helper do pacote senhaws). Flag `--password-stdin` permite caller passar senha custom (ex: gerada por `crypto/rand` ou via Vault).

**Razão:** `GerarSenhaRandom` é suficiente para 99% dos casos. Flag cobre edge cases (cripto-strong, integration com vault).

## 4. Estrutura proposta

```
backend/cmd/senhaws-rotate/
├── main.go          (~250 linhas — CLI principal + 3 subcomandos)
├── main_test.go     (~150 linhas — integration tests com httptest)
└── README.md        (uso + exemplos)

SPRINT_24_RESEARCH.md  (este)
SPRINT_24_RESULTS.md   (entregável)
```

## 5. Compatibilidade

- Novo binário `cmd/senhaws-rotate`. Zero impacto em código existente.
- Pacote `internal/senhaws` inalterado.
- Não wired em `cmd/api/main.go` (CLI é independente).
- Não interfere com nenhum workflow existente.

## 6. Plano de testes

| Test | Cobre |
|---|---|
| `TestSenhawsRotate_Check_OK` | check com 30 dias → exit 0 |
| `TestSenhawsRotate_Check_Expiring` | check com 5 dias → exit 1 |
| `TestSenhawsRotate_Check_BACENError` | check com 400 → exit 1 |
| `TestSenhawsRotate_Rotate_Success` | rotate happy path → exit 0 + senha no stdout |
| `TestSenhawsRotate_Rotate_BACENRejeita` | rotate com 400 → exit 3 |
| `TestSenhawsRotate_Rotate_AuthFalha` | rotate com 401 → exit 3 |
| `TestSenhawsRotate_Rotate_CurtaSenha` | input < 8 chars → exit 2 |
| `TestSenhawsRotate_Info` | info happy path |
| `TestSenhawsRotate_ConfigInvalid` | User formato Sisbacen errado → exit 2 |
| `TestSenhawsRotate_Quiet` | --quiet silencia logs |

## 7. Critérios de done

- [x] Research + design (este doc)
- [ ] CLI com 3 subcomandos (check/rotate/info)
- [ ] 10 testes top-level (httptest + unit)
- [ ] Build OK + gofmt/vet clean
- [ ] SPRINT_24_RESULTS.md + CHANGELOG v3.14.0
- [ ] commit + push

## 8. Riscos

| Risco | Mitigação |
|---|---|
| Senha em stdout vaza em shell history | Doc explícito: caller usa `> /tmp/newpass.txt` e limpa após usar. Não usar em interactive shell. |
| Senha em process listing (ps aux) | Senha é argumento de flag `--password` ou env var, NÃO position arg. `ps` mostra flags, mas valor de `--password` aparece. Caller DEVE usar env var (não flag) em produção. |
| CLI rodado como root inadvertidamente | Doc: requer file mode 0700 no binary se rodado como root. Caller gerencia. |
| BACEN down → CLI hang | Timeout configurável (default 30s). CLI não bloqueia eternamente. |
| Cron script com senha errada fica em loop | SenhawsClient retorna erro → CLI exit 1 → caller decide retry policy. CLI não tem loop interno. |

## 9. O que NÃO entra nesta sprint

- **Integração Vault automática** — caller decide onde armazenar (Sprint 27+).
- **Web UI** — IF tem 1-2 operadores, não justifica.
- **Métricas Prometheus** — CLI não é HTTP server.
- **Multi-tenant batch** — 1 CLI instance = 1 IF. Multi-tenant é caso de uso de API (handler), não CLI.
- **TLS client cert** — BACEN não exige (TLS só server-side).
- **Dry-run mode** — admin pode rodar `check` antes de `rotate` para verificar.

## 10. Referências

- Manual BACEN STA Web Services v1.5 §9 (mesmo da Sprint 23)
- SPRINT_23_RESULTS.md — pacote senhaws + YAGNI cluster
- cmd/jwt-mint — pattern de CLI com flag + env vars + logger
- cmd/seed — pattern de CLI com flag stdlib
- VALIDAÇÃO 44 — lições (placeholder drift, coverage gaps, compile-time asserts)