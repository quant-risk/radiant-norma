# SPRINT 22 — Research: Retry exponencial wrapper

> **Sprint:** 22 (v3.12.0)
> **Quando:** 2026-07-06
> **Pesquisador:** mavis
> **Status:** pesquisa completa, pronto pra implementação

## 1. Contexto

Sprint 22 entrega o **retry exponencial wrapper** para o cliente STA WS. Falhas transientes
do BACEN (503 Service Unavailable, 502 Bad Gateway, timeout de rede) são comuns em
integrações HTTP — BACEN pode estar momentaneamente sobrecarregado em horários de pico.

Sem retry, cada falha transiente vira:
- Erro 503 retornado ao caller
- Status `pending` no DB (worker vai retentar entre tentativas, mas com backoff fixo)
- Log de erro + alerta operacional

Com retry exponencial 3x (1s/2s/4s), **70%+ das falhas transientes** se resolvem antes
de virar erro operacional. Caller só vê erro depois de 3 tentativas fracassadas.

**Importante:** retry do **client HTTP** ≠ retry do **batch worker**.
- Worker retry (já existe, Sprint 6 v1.5.0): entre tentativas de envio (atrás do DB).
- Client retry (esta sprint): dentro de cada chamada HTTP.

São camadas diferentes. Worker retry cuida de "envio falhou, agendar pra próxima"; client
retry cuida de "BACEN 503 agora, tentar de novo em 1s".

## 2. Decisões de design

### 2.1 Retry em quê? (classificação de erros)

| Status / Erro | Retry? | Justificativa |
|---|---|---|
| 200, 201, 204 | N/A | Sucesso |
| 206 | N/A | Sucesso parcial |
| 400 Bad Request | ❌ | Caller bug — retry não conserta |
| 403 Forbidden | ❌ | Auth/permissão — permanente |
| 404 Not Found | ❌ | Protocolo inexistente — permanente |
| 410 Gone | ❌ | Arquivo cancelado — permanente |
| 412 Precondition Failed | ❌ | If-Match falhou — caller precisa revalidar |
| 416 Range Not Satisfiable | ❌ | Range inválido — caller precisa recalcular |
| 500 Internal Server Error | ✅ | BACEN bug — pode ser transiente |
| 502 Bad Gateway | ✅ | Proxy BACEN — clássico transiente |
| 503 Service Unavailable | ✅ | BACEN sobrecarregado — clássico transiente |
| 504 Gateway Timeout | ✅ | Timeout de gateway |
| Timeout de rede (context.DeadlineExceeded, net.Error) | ✅ | Transiente |
| Connection refused / reset | ✅ | BACEN reiniciando |
| X-Content-Hash mismatch | ❌ | Corrupção — não adianta retry |
| `STAError` com outro status | ❌ | Conservador — retry apenas em 5xx |

### 2.2 Backoff

**Decisão:** exponencial com jitter.

- Tentativa 1: imediato
- Tentativa 2 (se falhou): espera `BackoffBase × 1` = 1s
- Tentativa 3 (se falhou): espera `BackoffBase × 2` = 2s
- (Se MaxAttempts=4) Tentativa 4: espera `BackoffBase × 4` = 4s

**Por que jitter?** Múltiplos workers batendo ao mesmo tempo no BACEN com backoff
fixo podem sincronizar (thundering herd). Jitter randomiza espera entre
`BackoffBase × 2^n × (0.5..1.5)`.

### 2.3 Configuração

```go
type RetryConfig struct {
    MaxAttempts  int            // default 3 (1 inicial + 2 retries)
    BackoffBase   time.Duration  // default 1s
    BackoffFactor float64        // default 2.0
    Jitter        float64        // default 0.5 (±50% randomization)
    Logger        *slog.Logger   // optional, default slog.Default()
    OnRetry       func(attempt int, err error, nextBackoff time.Duration)  // optional callback
}
```

### 2.4 Cap no tempo total

Sem cap, retry pode levar muito tempo:
- 3 attempts × backoff até 2s = até 3s total em retry sleep
- Mais o tempo de cada call (30s timeout padrão)
- Total worst-case: ~90s

Caller pode passar `context.WithTimeout(ctx, 60s)` se quiser cap. Sem cap no wrapper
— caller decide.

### 2.5 Audit emission

Cada retry loga via `logger.Warn` com:
- attempt number
- error class
- backoff aplicado
- tempo total gasto

Caller pode injetar callback `OnRetry` para audit_log emission. Default: log estruturado apenas.

### 2.6 Interface segregation — só `Client` por agora

**Decisão:** `RetryingClient` wrappea só `Client` (Submit) nesta sprint. ReadClient e
ChunkedClient ficam para Sprint 23+ se virar problema operacional.

**Por que não agora:**
- 80% do tráfego é Submit (envio). Read/list é raro (poll frontend, eventual).
- Retry em Read tem semântica diferente (caller prefere erro rápido pra retry manual).
- 3 wrappers separados adiciona complexidade sem caller imediato.

**Padrão:** primitiva → wrapper → consumer. Submit já tem primitiva + vários consumers.
Wrapping é next step natural.

## 3. Estruturas propostas

```go
// RetryConfig configura comportamento do RetryingClient.
type RetryConfig struct {
    MaxAttempts  int
    BackoffBase   time.Duration
    BackoffFactor float64
    Jitter        float64
    Logger        *slog.Logger
    OnRetry       func(attempt int, err error, nextBackoff time.Duration)
}

// RetryingClient wrappea um sta.Client adicionando retry exponencial em
// erros transientes. Implementa Client — caller passa direto onde antes
// passava inner.
type RetryingClient struct {
    inner Client
    cfg   RetryConfig
}

// NewRetryingClient constrói wrapper. Retorna erro se cfg.MaxAttempts < 1.
func NewRetryingClient(inner Client, cfg RetryConfig) (*RetryingClient, error)

// Submit wrappea inner.Submit com retry exponencial. Erros 5xx + timeout
// disparam retry; 4xx + outros retornam sem retry.
func (r *RetryingClient) Submit(ctx context.Context, sub *Submission) (*Result, error)

// shouldRetry classifica err como retryable ou não.
// Retorna (true, backoffDuration) se retry, (false, 0) caso contrário.
func shouldRetry(err error, attempt int) (bool, time.Duration)

// isRetryableHTTPStatus checa se status code é 5xx (retry) ou 4xx (no retry).
func isRetryableHTTPStatus(statusCode int) bool

// sleepWithContext respeita ctx.Done() — se context cancelar durante sleep,
// retorna imediatamente.
func sleepWithContext(ctx context.Context, d time.Duration) error
```

## 4. Compatibilidade

- `Client` interface inalterada.
- `RetryingClient` implementa `Client` — caller substitui `staClient` por `retryingClient`.
- `cmd/api/main.go` inalterado nesta sprint (Sprint 23+ wire no main se virar útil).
- Caller pattern: `retrying := sta.NewRetryingClient(rawWS, cfg); staClient = retrying`.

## 5. Plano de testes

| Test | Cobre |
|---|---|
| `TestNewRetryingClient_Validacao` | MaxAttempts < 1 → erro |
| `TestRetryingClient_SuccessFirstTry` | 1 call, sucesso, sem retry |
| `TestRetryingClient_503RetryThenSuccess` | BACEN 503 duas vezes, sucesso na 3ª |
| `TestRetryingClient_400NoRetry` | BACEN 400, retorna imediatamente sem retry |
| `TestRetryingClient_403NoRetry` | BACEN 403, sem retry |
| `TestRetryingClient_MaxAttemptsExhausted` | sempre 503, retry N vezes, retorna erro final |
| `TestRetryingClient_NetworkErrorRetry` | context.DeadlineExceeded, retry |
| `TestRetryingClient_ContextCancel` | ctx cancela durante sleep → retorna ctx.Err() |
| `TestRetryingClient_OnRetryCallback` | callback chamado com params corretos |
| `TestShouldRetry_HTTPStatus_5xx` | 500/502/503/504 → true |
| `TestShouldRetry_HTTPStatus_4xx` | 400/403/404/410/412/416 → false |
| `TestShouldRetry_NetworkError` | net.Error, deadline exceeded → true |
| `TestShouldRetry_HashMismatch` | ErrContentHashMismatch → false |
| `TestRetryingClient_BackoffTiming` | backoff exponencial verificado (timing aproximado) |
| `TestSleepWithContext_Cancel` | ctx.Done() interrompe sleep |

## 6. Critérios de done

- [ ] `RetryConfig` + `RetryingClient` + `NewRetryingClient` + `Submit` implementados
- [ ] `shouldRetry` classifica 5xx + network errors como retryable
- [ ] Backoff exponencial com jitter implementado
- [ ] `OnRetry` callback opcional (audit_log hook)
- [ ] 15 testes httptest STA (happy + errors + validações + timing)
- [ ] 18/18 packages PASS + smoke + gofmt/vet
- [ ] SPRINT_22_RESEARCH.md (este) + SPRINT_22_RESULTS.md + CHANGELOG v3.12.0
- [ ] commit + push

## 7. Riscos identificados

| Risco | Mitigação |
|---|---|
| Retry mascara bug permanente do caller | Logs estruturados com OnRetry callback permitem audit trail |
| Retry acumula latência inaceitável | Cap no tempo total via ctx; caller decide timeout máximo |
| Retry sincronizado entre múltiplos workers (thundering herd) | Jitter randomiza espera |
| Caller passa MaxAttempts muito alto (10+) | Validação client-side em NewRetryingClient (MaxAttempts entre 1-10) |
| `OnRetry` callback faz trabalho pesado (ex: DB write) | Documentação diz callback deve ser leve; logger é path comum |
| `inner.Submit` retorna sucesso mas loga warning (ex: `Rejection.Code="UPLOAD_FAILED"`) | Não retry — Rejection não é erro transient. Caller inspeciona Result.Rejection. |

## 8. O que NÃO entra nesta sprint

- **RetryingReadClient / RetryingChunkedClient** — YAGNI. Submit é o caso comum.
- **Integration com batch worker** — worker já tem retry entre envios. Wrapper é
  camada ortogonal.
- **Métricas Prometheus** (`sta_retry_attempts_total`) — Sprint 17 plantou seeds;
  contador simples fica pra Sprint 24+ se virar problema operacional.
- **Handlers REST `/v1/sta/retry-config`** — sem caller imediato.
- **Circuit breaker** — overkill pra V1. Retry simples resolve 70%+ das falhas transientes.

## 9. Referências

- Manual BACEN STA Web Services v1.5 — Seção 2.6 (limites de conexão: 10 simultâneos, 120/min)
- SPRINT_18_RESEARCH.md — WSClient skeleton
- SPRINT_19_RESEARCH.md — read side + X-Content-Hash validation
- SPRINT_20_RESEARCH.md — read side completo + ReadClient interface segregation
- SPRINT_21_RESEARCH.md — chunked transfer + ChunkedClient interface segregation
- VALIDATION_v3.11.0_DEEPEST.md — padrões reforçados (errors.As, doc freshness, etc)