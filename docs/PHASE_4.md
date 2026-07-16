# Phase 4: STA persist + dedupe + retry + DLQ

**Data:** 2026-07-16
**Status:** ✅ SHIPPED

---

## Objetivo

Garantir que nenhuma submissão STA se perca, que duplicados sejam rejeitados
de forma idempotente, e que falhas permanentes cheguem a um DLQ visível.

---

## O que existia

- `cmd/worker` já tinha retry com backoff exponencial (5 tentativas: 1m, 5m, 30m, 2h, 12h).
- `cmd/worker` já tinha lease sweeper para detectar workers crashed.
- `status='dead_letter'` já existia como status terminal (migration 005).
- `envios` schema tinha `attempts`, `next_retry_at`, `processing_started_at` (migration 005).

**O que NÃO existia:**
1. Deduplicação no ponto de entrada (`staSubmit`).
2. Visibilidade do DLQ via API REST.
3. Retry manual de itens DLQ.
4. Índice para buscar envios mortos rapidamente.
5. Coluna `idempotency_key` para dedup explícito.

---

## O que foi implementado

### 4.1 Deduplicação em `staSubmit` (server.go)

Dois níveis de dedup:

**Nível 1 — Idempotency Key (X-Idempotency-Key header)**
```
Header: X-Idempotency-Key: {client-generated-uuid}
```
Se um segundo request chegar com a mesma key + mesmo tenant, retorna
o resultado da submissão original sem chamar o STA (idempotent replay).

**Nível 2 — xml_hash dedup**
Se nenhum idempotency key for fornecido, verifica se existe envio com
mesmo `(if_id, cadoc_code, data_base, xml_hash)` e status `pending|accepted|rejected`.
Rejeita o duplicates sem chamar STA.

Ambos os níveis logam em `audit_log` com action `sta.submit.idempotent_replay`
e `sta.submit.dedup`.

### 4.2 Coluna `idempotency_key` (migration 027)

```sql
ALTER TABLE envios ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_envios_idempotency_key
    ON envios(if_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key != '';
```

### 4.3 Novos índices (migration 027)

```sql
-- Acesso rápido a dead_letter por tenant
CREATE INDEX idx_envios_dead_letter
    ON envios(if_id, created_at DESC)
    WHERE status = 'dead_letter';

-- Acesso rápido a envios processáveis para dedup query
CREATE INDEX idx_envios_if_cadoc_db_hash
    ON envios(if_id, cadoc_code, data_base, xml_hash)
    WHERE status IN ('pending', 'accepted', 'rejected');
```

### 4.4 Endpoint `GET /v1/envios/dlq` (admin only)

Lista envios com `status = 'dead_letter'` para o tenant autenticado.
Resposta:

```json
{
  "envios": [
    {
      "id": "env-abc123",
      "cadoc_code": "3040",
      "data_base": "2026-06",
      "period": "06/2026",
      "status": "dead_letter",
      "error_code": "STA_SUBMIT_FAILED",
      "error_message": "...",
      "attempts": 5,
      "created_at": "2026-07-16T10:00:00Z"
    }
  ],
  "total": 1
}
```

### 4.5 Endpoint `POST /v1/envios/{id}/retry` (admin only)

Marca um envio `dead_letter` como `pending` e reseta `attempts = 0`.
O worker imediatamente o picking up.

Pré-condição: envio deve ter `status = 'dead_letter'`.
Retorna 409 Conflict se o envio não estiver em dead_letter.

### 4.6 `GET /v1/envios?status=dead_letter`

O filtro de status agora aceita `dead_letter` como valor válido.
O DTO `envioDTO` ganhou campo `attempts` para mostrar contagem de tentativas.

### 4.7 `GET /v1/envios/stats` — dead_letter count

A resposta agora inclui `"dead_letter": N` além dos demais status.

---

## Arquivos alterados

| Arquivo | Mudança |
|---|---|
| `internal/api/server.go` | `staSubmit` com dedup; `checkIdempotencyKey`, `checkXmlHashDedup`, `nullString`, `listDLQ`, `retryDLQ` handlers |
| `internal/api/sprint8c_handlers.go` | `envioDTO.attempts`; query SELECT com `attempts`; stats com `dead_letter` |
| `internal/api/server_e2e_test.go` | Test fix: `data_base` required para `TestValidate_ValidCadocEmitsAudit` |
| `internal/db/migrations/027_sta_dedupe_dlq.sql` | **NOVO** — idempotency_key + índices |

---

## Testes

```bash
# API
go test ./internal/api/...    # ✅ ok (7.8s)

# Worker
go test ./internal/worker/...  # ✅ ok (0.8s)

# STA
go test ./internal/sta/...    # ✅ ok (1.0s)
```

---

## Contrato de API

### `POST /v1/sta/submit`

**Headers adicionais aceitos:**
- `X-Idempotency-Key: {uuid}` — chave de idempotência (opcional)

**Respostas de dedup (HTTP 200):**
```json
{
  "protocol_sta": "20260716012345abc",
  "accepted": true,
  "rejection": null,
  "envio_id": "env-abc123",
  "dedup": "idempotency_key"   // ou "xml_hash"
}
```

### `GET /v1/envios/dlq`

**Auth:** admin only (JWT role = `admin`)

### `POST /v1/envios/{id}/retry`

**Auth:** admin only (JWT role = `admin`)
**Pré-condição:** `status = 'dead_letter'`
**Resposta 200:**
```json
{"envio_id": "env-abc123", "status": "pending", "message": "envio requeued for retry"}
```
**Resposta 409:** `{ "error": "cannot retry envios with status 'accepted' (only dead_letter)" }`

---

## Notas de operação

- O retry manual via `POST /v1/envios/{id}/retry` não requer restart do worker.
- O worker vai picking up o item resetado no próximo tick (máx. 30s).
- Deduplicação por `xml_hash` é por `(if_id, cadoc_code, data_base)` — ou seja,
  mesmo XML para mesmo mês = duplicate. Isso pode ser muito agressivo em
  alguns cenários; avaliar se `data_base` deve ser removido do dedup key.
- ODLQ não é automático — depende de admin visualizar via `GET /v1/envios/dlq`
  e acionar `POST /v1/envios/{id}/retry` manualmente. Automação de DLQ
  (alertas, webhooks) fica para Phase 5 (webhooks).
