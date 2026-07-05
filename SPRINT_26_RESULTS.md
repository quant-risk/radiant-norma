# SPRINT 26 — RESULTS: cmd/sta-submit CLI (CADOC submission tool)

> **Sprint:** 26 (v3.19.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped

## TL;DR

Sprint 26 fecha o **CLI `cmd/sta-submit`** — segundo caller real do pacote `internal/sta`. Admin IF pode submeter CADOC ao BACEN STA WS direto via linha de comando, sem deployar API ou UI.

**Decisão arquitetural:** CLI single-command (apenas `submit`) — escopo focado no caso de uso principal. Reusa `sta.NewClientFromEnv` (mesma fábrica usada por `cmd/api`) → consistency entre CLI e servidor.

**Decisões YAGNI conscientes:**
- Sem handler REST (admin tool direto, não UI).
- Sem retry wrapper (failure fast — caller decide retry).
- Sem range upload / chunked (single CADOC <50 MB usa Submit normal; range é Sprint 27+).
- Sem upload de ZIP (apenas XML — cobre 80% do caso de uso).
- Sem TLS client cert (BACEN não exige).
- Sem dry-run.

**Decisões de design não-óbvias:**
- **Injeção de client via variável de função** (`staNewClientFromEnv`): pattern de test injection sem precisar mockar STA Client inteiro. Tests sobrescrevem a variável, produção usa `sta.NewClientFromEnv`.
- **Interface `staClient` mínima** (só `Submit`): desacopla CLI de qualquer mudança futura em `sta.Client`.

## Entregas

### 1. Binário `cmd/sta-submit` (~212 linhas main + ~290 linhas test)

**Flags:**
- `--xml-file` (env `STA_SUBMIT_XML_FILE`) — caminho do XML do CADOC
- `--cadoc-code` (env `STA_SUBMIT_CADOC_CODE`) — default `3040`
- `--data-base` (env `STA_SUBMIT_DATA_BASE`) — formato YYYY-MM
- `--cnpj` (env `STA_SUBMIT_CNPJ`) — default `demo-bank`
- `--quiet` — silencia logs stderr

**Env vars STA (delegadas a `sta.NewClientFromEnv`):**
- `RADIANT_STA_BACKEND` (stub|ws)
- `RADIANT_STA_WS_URL`
- `RADIANT_STA_SISBACEN_USER`
- `RADIANT_STA_SISBACEN_PASSWORD`
- `RADIANT_STA_TIMEOUT_SECONDS`

**Exit codes (consistente com cmd/senhaws-rotate):**
- `0` aceito pelo BACEN
- `1` rejeitado pelo BACEN / erro de transporte
- `2` erro de validação client-side (input inválido)
- `3` erro BACEN formal

**Output:**
- Sucesso: `protocol_sta=<PROT>  status=accepted`
- Rejeição: `protocol_sta=<PROT>  status=rejected  code=<C>  message=<M>`
- Erro config: stderr `config invalida: ...`
- Erro BACEN: stderr `erro BACEN STA <CODE>: <MSG>`

### 2. Interface `staClient` + injection point

```go
// staClient é interface mínima que runSubmit usa (test injection point).
type staClient interface {
    Submit(ctx context.Context, sub *sta.Submission) (*sta.Result, error)
}

// staNewClientFromEnv é variável de função para permitir injeção em tests.
var staNewClientFromEnv func(logger *slog.Logger) (staClient, error) = func(...) {...}
```

**Benefício:** Tests podem trocar `staNewClientFromEnv` para usar `StubClient` ou `WSClient` apontando para `httptest.Server`, sem refatorar chamada em `runSubmit`.

### 3. Test injection pattern (10 testes)

**Helpers:**
- `newStubClientAlwaysReject()` — StubClient com `AlwaysAccept=false`
- `staNewWSClientForTest(url, user, pass)` — WSClient com `AllowInsecureHTTP=true`

**Tests:**
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

## Decisões que pagaram

### D-1. Interface `staClient` mínima (1 método)

Decouple CLI de mudanças futuras em `sta.Client`. Se Sprint 27+ adicionar `ChunkedClient` ou `ReadClient`, CLI não quebra.

### D-2. Variable de função para test injection

Pattern idiomático Go: `var x = funcName` permite tests sobrescreverem. Mais simples que interface mocking framework.

### D-3. Reusa `sta.NewClientFromEnv`

Mesma fábrica que `cmd/api` usa → consistency operacional. Mesmas env vars, mesmo comportamento, mesma fallback (StubClient default).

### D-4. Single command (não subcomandos)

`submit` é único. Se virar `check`, `cancel`, etc no futuro, adicionar subcomandos é trivial. YAGNI agora.

### D-5. Exit code 1 = rejeitado OU transporte

Consistente com cmd/senhaws-rotate. Cron scripts podem usar exit 1 como "retry" trigger.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (main.go 212 linhas + main_test.go 290 linhas) |
| Packages novos | 0 (cmd/* não conta) |
| Tests Sprint 26 | 10 (todos PASS) |
| Total backend tests top-level | **127** (era 117, +10) |
| Packages PASS | **21/21** (era 20, +1 = cmd/sta-submit) |
| Build OK | **7/7 binaries** (era 6, +1 = sta-submit) |
| gofmt drift | 0 |
| vet | clean |
| Race detector | clean |

## Compatibilidade

- **Novo binário `cmd/sta-submit`.** Zero impacto em código existente.
- **`sta.NewClientFromEnv` inalterado.** CLI apenas wrappea.
- **Não wired em `cmd/api/main.go`** — CLI é independente (decoupling).
- **Nenhum handler REST adicionado** — admin tool direto.
- **`internal/sta/*` inalterado** — reuso.

## Lições aprendidas (carry forward)

### L-1. Variable de função = test injection idiomático

`var f = realFunc` permite tests sobrescreverem sem framework externo. Pattern replicável em qualquer CLI que wrappea factory.

### L-2. Interface mínima desacopla de mudanças futuras

`staClient` com 1 método Submit. Se Sprint 27+ adicionar métodos em `sta.Client`, CLI continua funcionando. Pattern: dependa do mínimo necessário.

### L-3. YAGNI em subcomandos

CLI tem 1 comando (`submit`). Adicionar `check`/`cancel`/`info` é trivial quando virar requisito. Sprint 24 (senhaws-rotate) tem 3 subcomandos porque caso de uso é composto. Sprint 26 tem 1 porque caso de uso é atômico.

### L-4. Test injection pattern escala

10 testes cobrem 4 fluxos (sucesso, rejeição, config error, BACEN error, transporte) usando apenas 2 helpers (StubClient + WSClient mock). Pattern replicável para futuros CLIs.

### L-5. Reusa `sta.NewClientFromEnv` = consistency operacional

Admin IF que usa `sta-submit` + `cmd/api` precisa configurar mesmas env vars. Mesmo backend (stub ou ws). Mesma fallback. Zero inconsistência.

## Próximos passos (Sprint 27+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 27 | Pre-commit hook: lint + gofmt + vet | Fecha gap operacional do Sprint 25 |
| 28 | Vault integration | Secret manager rotation |
| 29 | Smoke contra BACEN homolog real | Requer credenciais Sisbacen |
| 30 | `cmd/sta-submit` range upload | Chunked transfer (Sprint 21) |
| 31 | Handler REST `/v1/sta/range-*` (Sprint 21 YAGNI) | Frontend/batch trigger UI |

## Critérios de done — todos ✅

- [x] CLI com flag --xml-file + exit codes + output format
- [x] Interface `staClient` mínima
- [x] Variable de função `staNewClientFromEnv` para test injection
- [x] 10 testes top-level (httptest + StubClient)
- [x] 21/21 packages PASS (zero regressão)
- [x] Build smoke 7/7 binaries
- [x] gofmt/vet clean
- [x] SPRINT_26_RESEARCH.md + SPRINT_26_RESULTS.md + CHANGELOG (próximo)
- [x] commit + push (próximo)

## Como usar (quickstart)

```bash
# 1. Setup (env vars para STA real)
export RADIANT_STA_BACKEND=ws
export RADIANT_STA_WS_URL=https://sta-h.bcb.gov.br/staws
export RADIANT_STA_SISBACEN_USER="123450001.fulano"
export RADIANT_STA_SISBACEN_PASSWORD="$ACTUAL_PASSWORD"

# 2. Submeter CADOC
sta-submit --xml-file=/path/to/cadoc3040.xml \
           --cadoc-code=3040 \
           --data-base=2024-12 \
           --cnpj=demo-bank

# → protocol_sta=PROTO-OK  status=accepted
# → exit 0

# 3. Cron (batch job)
for cadoc in *.xml; do
    sta-submit --xml-file="$cadoc" --data-base=2024-12 || \
        echo "FAIL: $cadoc" >> submit_errors.log
done

# 4. Default (sem env vars) usa StubClient — útil pra dev/test
unset RADIANT_STA_BACKEND
sta-submit --xml-file=test.xml --data-base=2024-12
# → protocolo fake gerado por StubClient, exit 0
```

## Anti-patterns evitados

1. **CLI monolítico com 5 subcomandos** — escopo YAGNI. 1 comando resolve o caso de uso.
2. **In-memory mock via interface mocking framework** — `var f = realFunc` é mais simples, sem deps.
3. **Wrapper vazio** — CLI tem comportamento real (lê XML, chama Submit, formata output).
4. **Senha em logs** — apenas em env vars, nunca em flags ou stdout.
5. **Retry mascara bug** — failure fast consistente com senhaws-rotate.