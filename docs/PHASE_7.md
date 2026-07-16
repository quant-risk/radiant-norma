# Phase 7: Auditoria + Insights em fonte única

**Data:** 2026-07-16
**Status:** ✅ SHIPPED

---

## Objetivo

Unificar o sistema de auditoria: toda entrada de auditoria deve ser escrita
UMA vez e populart ambas as tabelas `audit_log` (chain hash, tamper-evident)
e `audit_events` (view denormalizada, legível para UI).

---

## O que existia

**`audit_log` (chain hash table):**
- `auditlog.Logger.Log` escrever apenas nela
- Hash chain com `prev_hash` + `entry_hash` (SHA-256)
- tamper-evident para LGPD/SOC2
- Verificado por `auditlog.Verify()`

**`audit_events` (denormalizado legível):**
- Migration 006 criou a tabela
- migration 010 adicionou FK `audit_log_id → audit_log.id`
- migration 012/014 habilitou RLS
- **NINGUÉM escrevia nela** — seed scripts populavam manualmente

**Consequência:** `/v1/audit_log` lia de `audit_events` (que estava vazio em produção);
`audit_log` era write-only exceto para o `Verify` admin.

---

## O que foi implementado

### 7.1 `auditlog.Logger.Log` agora escreve em ambas as tabelas

**Antes:**
```go
// Só insertava em audit_log
tx.ExecContext(ctx, `INSERT INTO audit_log (...) VALUES (...)`)
```

**Depois (same transaction):**
```go
// 1. audit_log (chain)
tx.ExecContext(ctx, `INSERT INTO audit_log (...) VALUES (...)`)
id, _ = res.LastInsertId()

// 2. audit_events (denormalizado legível) — mesmo TX, mesmo timestamp
_, err = tx.ExecContext(ctx, `
    INSERT INTO audit_events (audit_log_id, if_id, actor, action, target,
                              description, payload, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, id, ...)
```

Se qualquer INSERT falhar, o transaction rollback aborta ambas as operações
— atomicidade garantida pelo SQLite/Postgres transaction.

**Novo helper `extractDescription`:**
Extrai `"description"` do metadata map quando presente, populando
`audit_events.description` automaticamente.

---

## Como ficou a arquitetura de auditoria

```
Handler chama s.AuditLog.Log(ifID, actor, action, target, payload, metadata)
         │
         ▼
┌─────────────────────────────────────────────┐
│         auditlog.Logger.Log (1 call)          │
│                                              │
│  ┌──────────────────────────────────────┐   │
│  │ BEGIN TRANSACTION (WITH TENANT TX)    │   │
│  │                                      │   │
│  │  1. INSERT INTO audit_log (...)      │   │  ← hash chain (tamper-evident)
│  │     id = last_insert_rowid()         │   │
│  │                                      │   │
│  │  2. INSERT INTO audit_events (...)  │   │  ← denormalizado legível (Phase 7 fix)
│  │     audit_log_id = id               │   │
│  │                                      │   │
│  │ COMMIT                               │   │
│  └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
         │
         ▼
/v1/audit_log ← lê de audit_events (legível, indexado)
/v1/audit_log/verify ← lê de audit_log (chain verification)
```

---

## Eventos agora populados em `audit_events`

Todas as ações que antes iam só para `audit_log` agora também populam `audit_events`:

| Ação | Disparado por |
|---|---|
| `cadoc.validated` | `POST /v1/validate` |
| `sta.submit` | `POST /v1/sta/submit` |
| `sta.submit.idempotent_replay` | idempotency dedup |
| `sta.submit.dedup` | xml_hash dedup |
| `sta.submit.persist_failed` | persist error |
| `envio.retry.dlq` | manual DLQ retry |
| `radar.alert.resolved` | resolve alert |
| `radar.scan.triggered` | trigger scan |
| `radar.scan.cached` | cached scan |
| `crossdoc.validated` | crossdoc validation |

---

## Arquivos alterados

| Arquivo | Mudança |
|---|---|
| `internal/auditlog/log.go` | `Log` agora faz INSERT em `audit_events` no mesmo TX; novo helper `extractDescription` |

---

## Testes

```bash
go test ./internal/auditlog/...  # ✅ chain integrity tests still pass
go test ./internal/api/...        # ✅ (inclui audit_log handlers)
```

---

## Notas operacionais

- **Backward compatible**: existing `Verify()` continua funcionando — não depende de `audit_events`.
- **Seed scripts** (`cmd/seed-sprint8c`, `cmd/seed-demo`) continuam populando `audit_events`
  diretamente com placeholders (não chamam `AuditLog.Log`). Isso é intentional — seed
  é dados de demonstração, não produção.
- **Atomicidade**: se `audit_events` INSERT falhar, o TX faz rollback e
  `audit_log` também não é escrito. Isso mantém consistência mas significa
  que uma falha de INSERT no `audit_events` (ex: payload muito grande) pode
  bloquear o `audit_log`. Em prática, `payload TEXT` no SQLite aceita até ~1GB.
