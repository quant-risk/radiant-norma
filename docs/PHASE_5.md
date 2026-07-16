# Phase 5: Webhook inicializar + assinatura + idempotência

**Data:** 2026-07-16
**Status:** ✅ SHIPPED

---

## Objetivo

Corrigir bugs críticos no sistema de webhooks outbound existente e garantir que
eventos de submissão STA sejam disparados.

---

## O que existia

- Migration 019 (`019_webhooks.sql`) com schema `webhooks` + `webhook_deliveries`.
- `webhook.Service.Dispatch` — itera webhooks registrados e chama `ds.Enqueue`.
- `webhook.Dispatcher` com 4 workers e retry com backoff.
- `Deliver` e `EnqueueAndInsert` (no code) — pattern para insert-before-enqueue.
- `isRetryable(err)` — decidia retry baseado apenas em mensagem de erro.

**Eventos definidos mas nunca disparados:**
- `submission.accepted`
- `submission.rejected`

---

## Bugs corrigidos

### B1: `Dispatch` não criava registro de delivery antes de enfileirar

**Antes:**
```go
// Dispatch chamava ds.Enqueue(deliveryID, ...) mas nunca fazia o INSERT
// na tabela webhook_deliveries. O worker (processJob) fazia:
//   UPDATE webhook_deliveries SET status='pending' WHERE id=? AND webhook_id=?
// e recebia RowsAffected=0 porque o registro não existia → log "delivery record
// not found, skipping".
```

**Depois:**
```go
// Novo método EnqueueAndInsert: cria o registro E enfileira.
func (d *Dispatcher) EnqueueAndInsert(webhookID, event, payload string) string {
    id := newID()
    d.db.ExecContext(ctx,
        `INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, ...) VALUES ...`)
    d.Enqueue(id, webhookID, event, payload)
    return id
}

// Dispatch agora usa EnqueueAndInsert.
```

**Arquivo:** `internal/webhook/dispatcher.go`

### B2: `isRetryable` ignorava HTTP status code

**Antes:**
```go
// Só olhava a mensagem de erro (network error → retry; qualquer texto com 4xx
// passava como retry porque mensagem não continha "400" explicitamente).
```

**Depois:**
```go
func isRetryable(err error, status int) bool {
    if err == nil { return false }
    // 429 → retry
    if status == 429 { return true }
    // 5xx → retry
    if status >= 500 && status < 600 { return true }
    // 4xx (exceto 429) → não retry
    if status >= 400 && status < 500 { return false }
    // network error → retry
    return containsAny(err.Error(), "timeout", "deadline", ...)
}
```

**Arquivo:** `internal/webhook/helpers.go`

### B3: `staSubmit` nunca disparava webhooks de submissão

**Antes:**
```go
// DispatchSubmissionAccepted e DispatchSubmissionRejected existiam no service
// mas nunca eram chamadas do handler de STA submit.
```

**Depois:**
```go
// Após persistir envío, staSubmit agora dispara:
if result.Accepted {
    s.Webhook.DispatchSubmissionAccepted(...)
} else {
    s.Webhook.DispatchSubmissionRejected(...)
}
```

**Arquivo:** `internal/api/server.go`

---

## Assinatura HMAC

O header `X-Radiant-Signature: sha256={hmac}` já estava implementado em `helpers.go:deliver`:

```go
if secret != "" {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(payload))
    sig := hex.EncodeToString(h.Sum(nil))
    req.Header.Set("X-Radiant-Signature", "sha256="+sig)
}
```

O `secret` é armazenado em texto plano no DB (column `secret` em `webhooks`).
Assinatura usa HMAC-SHA256 sobre o body JSON cru.

---

## Eventos disponíveis

| Evento | Disparado em | Payload |
|---|---|---|
| `validation.completed` | POST /v1/validate | `EventValidationCompleted` |
| `schema.changed` | publicação de schema | `EventSchemaChanged` |
| `radar.change_detected` | scan radar | `EventRadarChangeDetected` |
| `submission.accepted` | STA accept | `EventSubmissionAccepted` |
| `submission.rejected` | STA reject | `EventSubmissionRejected` |
| `envio.dead_letter` | worker DLQ | (Phase 7 — não implementado ainda) |

---

## Arquivos alterados

| Arquivo | Mudança |
|---|---|
| `internal/webhook/dispatcher.go` | `EnqueueAndInsert`; `Dispatch` agora chama `EnqueueAndInsert` |
| `internal/webhook/helpers.go` | `isRetryable(err, status int)` — status-aware |
| `internal/webhook/service.go` | `Deliver` usa `EnqueueAndInsert`; `Dispatch` usa `EnqueueAndInsert` |
| `internal/api/server.go` | `staSubmit` chama `DispatchSubmissionAccepted/Rejected` |
| `internal/webhook/service_test.go` | teste `isRetryable` com status codes |

---

## Testes

```bash
go test ./internal/webhook/...    # ✅ ok (0.4s)
go test ./internal/api/...        # ✅ ok (9.7s)
```

---

## Headers de webhook entregue ao receiver

```
POST {url_do_cliente}
Content-Type: application/json
X-Radiant-Event: submission.accepted
X-Radiant-Timestamp: 1752650400
X-Radiant-Signature: sha256=abc123...   (se secret configurado)
```

Body: JSON do evento correspondente (`EventSubmissionAccepted`, etc).

---

## Retentativas

- Máximo: 5 tentativas
- Backoff: 1m, 5m, 15m, 30m, 60m
- Status finais: `success` | `failed` (após 5 tentativas)
- Após DLQ de webhook (envio que excedeu retries) → evento `envio.dead_letter` (Phase 7)
