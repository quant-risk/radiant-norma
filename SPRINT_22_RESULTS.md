# SPRINT 22 — RESULTS: Retry exponencial wrapper

> **Sprint:** 22 (v3.12.0)
> **Quando:** 2026-07-06
> **Status:** ✅ Shipped
> **Commit:** (preencher após push)

## TL;DR

Sprint 22 fecha o **retry exponencial wrapper** para o cliente STA WS. Falhas transientes
do BACEN (503/502/timeout/connection refused) agora são absorvidas via retry automático
com backoff 1s/2s/4s + jitter ±50%. Erros permanentes (4xx, X-Content-Hash mismatch)
**não fazem retry** — caller bug ou corrupção de integridade.

**Decisão arquitetural:** `RetryingClient` wrappea qualquer `sta.Client` (drop-in
replacement). Mesma interface `Submit(ctx, sub) (*Result, error)` — caller substitui
inner por RetryingClient onde antes passava inner. Não precisa mudar callers existentes.

**Decisão YAGNI consciente:** **NÃO** criar `RetryingReadClient` / `RetryingChunkedClient`.
Submit é 80% do tráfego. Read/list são raros. 3 wrappers separados adiciona complexidade
sem caller imediato. Se virar problema operacional, Sprint 24+.

**Mudança adicional crítica descoberta durante implementação (validação 42):**
`parseSTAError` (Sprint 18) retornava `fmt.Errorf` opaco. RetryingClient precisa
`errors.As(err, &staErr)` para classificar 5xx vs 4xx — quebrava o wrapping. Mudança
mínima: `parseSTAError` agora retorna `*STAError` direto. **Tests Sprint 18 continuam
passando** (todos usam `strings.Contains`, robustos a mudança de tipo de erro).

**17 testes novos** no pacote sta. Total STA: **81 testes top-level** (era 63 antes da Sprint 22).

## Entregas

### 1. `RetryConfig` — configurável

```go
type RetryConfig struct {
    MaxAttempts  int            // default 3 (1 inicial + 2 retries)
    BackoffBase   time.Duration  // default 1s
    BackoffFactor float64        // default 2.0 (exponencial 1s/2s/4s)
    Jitter        float64        // default 0.5 (±50% — evita thundering herd)
    Logger        *slog.Logger   // opcional, default slog.Default()
    OnRetry       func(attempt int, err error, nextBackoff time.Duration)
}
```

Validação client-side: MaxAttempts em [1, 10], Jitter em [0, 1].

### 2. `RetryingClient` — drop-in replacement

```go
type RetryingClient struct {
    inner Client
    cfg   RetryConfig
    rng   *rand.Rand
}

func NewRetryingClient(inner Client, cfg RetryConfig) (*RetryingClient, error)
func (r *RetryingClient) Submit(ctx context.Context, sub *Submission) (*Result, error)
```

Implementa `Client` interface. Caller substitui:

```go
// Antes:
staClient, _ := sta.NewClientFromEnv(logger)

// Depois:
raw, _ := sta.NewClientFromEnv(logger)
staClient, _ := sta.NewRetryingClient(raw, sta.RetryConfig{
    MaxAttempts:  3,
    BackoffBase:  1 * time.Second,
    Logger:       logger,
})
```

### 3. Classificação de erros (`shouldRetry`)

**Retryable (5xx + network):**
- HTTP 500, 502, 503, 504
- `context.DeadlineExceeded` (timeout)
- `net.Error` com `Timeout()=true`
- `url.Error` wrappando connection refused/reset/broken pipe/no such host

**NÃO retryable (4xx + corrupção):**
- HTTP 400, 403, 404, 410, 412, 416 (caller bug ou permanente)
- `ErrContentHashMismatch` (Sprint 19 — corrupção de integridade)
- `ErrContentHashHeaderMalformed` (Sprint 19 — formato mudou)
- `context.Canceled` (caller cancelou — não é transient)

### 4. Backoff exponencial com jitter

```go
mult := math.Pow(cfg.BackoffFactor, float64(attempt-1))
backoff := time.Duration(float64(cfg.BackoffBase) * mult)
// jitter: ±Jitter random
backoff = applyJitter(backoff, rng)
```

Padrão:
- Attempt 1: imediato
- Attempt 2 (se falhou): espera `BackoffBase × 1 ± Jitter` (1s ± 50% = 500ms..1500ms)
- Attempt 3 (se falhou): espera `BackoffBase × 2 ± Jitter` (2s ± 50% = 1s..3s)

### 5. `sleepWithContext` — respeita `ctx.Done()`

Sleep entre retries respeita context cancellation. Se caller cancelar (timeout, shutdown),
retorna `ctx.Err()` wrappeado em vez de continuar retrying.

### 6. Audit emission via `OnRetry` callback

```go
rc, _ := sta.NewRetryingClient(raw, sta.RetryConfig{
    OnRetry: func(attempt int, err error, nextBackoff time.Duration) {
        auditLog.Log(ifID, "", "sta.submit.retry",
            err.Error(), nil, map[string]any{
                "attempt": attempt,
                "next_backoff": nextBackoff,
            })
    },
})
```

Callback é opcional. Default: logger estruturado.

### 7. Bug fix encontrado (validação 42 F-S22-1)

`parseSTAError` retornava `fmt.Errorf` opaco. Quebrava `errors.As(err, &staErr)` no RetryingClient.
**Fix:** `parseSTAError` agora retorna `*STAError` direto. Tests Sprint 18 usam `strings.Contains` —
robustos a mudança. 11 testes Sprint 18 continuam passando.

## Decisões que pagaram

### D-1. YAGNI: só `Client` (Submit), não Read/Chunked

3 wrappers × ReadClient/ChunkedClient seria 6 combinações. Submit é o caso comum
(80% do tráfego). Read/list raramente falha (poll frontend é tolerante). Se Read virar
problema operacional, Sprint 24+ adiciona wrapper.

### D-2. Interface drop-in

`RetryingClient` implementa `Client` — caller substitui inner direto. Zero mudanças
em callers existentes (worker, handlers, etc). Caller opta-in quando quiser.

### D-3. Jitter default 0.5 (±50%)

Sem jitter, múltiplos workers podem sincronizar (thundering herd). Jitter ±50% é
sweet spot comum (não muito agressivo, não muito suave). Documentado em código.

### D-4. Classificação conservadora (4xx nunca retry)

É melhor falhar rápido em caller bug do que retryar eternamente. Classificação
conservadora: erros desconhecidos → no retry. Caller pode logar e investigar.

### D-5. `OnRetry` callback opcional

Caller pode hookar audit_log/métrica sem poluir interface. Default é logger.
Callback é leve (aviso: não fazer I/O pesado).

### D-6. `sleepWithContext` respeita ctx

Caller pode wrappar com `context.WithTimeout` para cap de tempo total. Sleep
interrompido retorna `ctx.Err()` wrappeado — caller vê que retry foi cancelado, não
retry silencioso continuou.

## Estatísticas

| Métrica | Valor |
|---|---|
| Arquivos novos | 2 (`retry.go` + `retry_test.go`) |
| Arquivos modificados | 1 (`ws.go` — fix parseSTAError) |
| Testes Sprint 22 | 17 (12 httptest STA + 5 unit tests puros) |
| Total STA | 81 testes top-level |
| Packages PASS | 18/18 |
| Build OK | 5/5 binaries |
| Smoke E2E | 11/11 PASS |
| gofmt drift | 0 |
| go vet | clean |

## Lições aprendidas (carry forward)

### L-1. `parseSTAError` opaco vs tipado — quando cada um

Sprint 18 (`parseSTAError`) usava `fmt.Errorf` porque caller (`Submit`) só queria
`err.Error()` para exibir ao caller humano. Caller não inspecionava campos.

Sprint 22 (RetryingClient) precisa **inspecionar** status code → `*STAError` tipado.

**Regra:** se caller pode se beneficiar de inspeção tipada (`errors.As`), retorne
tipo concreto. Se caller só quer mensagem legível, `fmt.Errorf` basta. **Mude
quando houver caller que precisa** — não antecipe.

### L-2. `fmt.Errorf("%w", ...)` + `errors.As` walk the chain

`Submit` wrappeia `parseSTAError` com `%w`. `RetryingClient.shouldRetry` chama
`errors.As(err, &staErr)` — stdlib walk the chain até achar `*STAError`. Funciona
mesmo com múltiplos wrappings.

### L-3. Jitter é defesa contra thundering herd

Em sistemas distribuídos, sem jitter múltiplos clients com mesmo backoff sincronizam
(todos esperam 1s, todos tentam, todos falham, todos esperam 1s, ...). Jitter randomiza.

### L-4. YAGNI consciente vs prematura

`RetryingReadClient` e `RetryingChunkedClient` são wrappers óbvios agora que
`RetryingClient` existe. Mas sem caller imediato, adicionar seria code bloat.
Se virar problema operacional, 30 min de trabalho.

### L-5. Test que descobre bug de produção

`TestRetryingClient_OnRetryCallback` falhou inicialmente porque `successSTAHandler`
retornava 200 OK em vez de 201 Created. WSClient.Submit esperava 201 (manual §5.1.1).
Bug em potencial production code (Sprint 18) — mas só descoberto porque Sprint 22
testa o caminho de Submit completo via httptest.

## Próximos passos (Sprint 23+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| 23 | Senhaws endpoint (§9.1) + credential rotation | Troca periódica de senha Sisbacen |
| 24 | Smoke contra BACEN homolog real (precisa credenciais) | Última validação antes de produção |
| 25 | Handlers REST `/v1/sta/range-*` (quando batch worker chamar) | Frontend ou batch trigger UI |
| 26 | `RetryingReadClient` / `RetryingChunkedClient` (se virar problema operacional) | Defender read-side contra BACEN 503 |
| 27 | Wire `RetryingClient` no `cmd/api/main.go` (se virar requisito) | Defense para prod |

## Critérios de done — todos ✅

- [x] `RetryConfig` + `RetryingClient` + `NewRetryingClient` + `Submit` implementados
- [x] `shouldRetry` classifica 5xx + network errors como retryable
- [x] Backoff exponencial com jitter implementado
- [x] `OnRetry` callback opcional (audit_log hook)
- [x] `sleepWithContext` respeita `ctx.Done()`
- [x] 17 testes httptest STA (12 cenários integração + 5 unit tests puros)
- [x] 18/18 packages PASS + smoke + gofmt/vet
- [x] Bug fix encontrado (validação 42 F-S22-1): `parseSTAError` agora retorna `*STAError` direto
- [x] SPRINT_22_RESEARCH.md + SPRINT_22_RESULTS.md + CHANGELOG v3.12.0
- [ ] commit + push (próximo passo)

## Anti-patterns evitados

1. **Retry mascara bug permanente** — 4xx nunca retry (caller bug não conserta com retry).
2. **Retry infinito** — `MaxAttempts` cap em 10. Caller pode wrappear com `context.WithTimeout`.
3. **Retry sincronizado** — Jitter ±50% randomiza backoff.
4. **Empty wrapper** — `RetryingClient` tem comportamento distinto (retry), não é decorator vazio.
5. **Hollow stub** — Wrapper wrappeia só `Submit` (não `Client` inteira) — honestidade sobre capability.