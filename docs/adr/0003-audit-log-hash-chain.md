# ADR-0003: Audit log — hash chain SHA-256 + imutável via trigger

> **Status:** Aceito
> **Data:** 2026-07-05
> **Decisor(es):** Henrique Costa · Mavis

## Contexto

LGPD + SOC 2 + auditoria BACEN exigem que toda mutação seja registrada de forma:
- **Tamper-evident** (qualquer alteração invalida cadeia)
- **Imutável** (não pode ser editada/deletada por ninguém, inclusive DBA)
- **Verificável** (auditor externo pode validar sem privilégios especiais)

O BCValidador oficial (Java) não tem isso. Concorrentes SaaS (Matera, Mitra) têm audit log mas editável.

## Decisão

Cada entry do audit log tem:

```
entry_hash = SHA-256(prev_hash ‖ payload_hash ‖ metadata ‖ actor ‖ action ‖ target ‖ if_id ‖ timestamp)
```

Três mecanismos叠加 (defense-in-depth):

### 1. Hash chain na app

```go
func (l *Logger) Log(ctx context.Context, ifID, actor, action, target string, payload []byte, metadata any) (*Entry, error) {
    tx, err := l.db.BeginTx(ctx, nil)
    if err != nil { return nil, err }
    defer tx.Rollback()

    // BEGIN IMMEDIATE garante serialização (SQLite) / row lock (Postgres)
    var prevHash string
    err = tx.QueryRowContext(ctx, `SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1 FOR UPDATE`).Scan(&prevHash)
    if errors.Is(err, sql.ErrNoRows) {
        prevHash = strings.Repeat("0", 64) // Genesis
    }

    payloadHash := sha256.Sum256(payload)
    timestamp := time.Now().UTC().Format(time.RFC3339Nano)
    concat := prevHash + hex.EncodeToString(payloadHash[:]) + metadataJSON + actor + action + target + ifID + timestamp
    entryHash := sha256.Sum256([]byte(concat))

    tx.ExecContext(ctx, `INSERT INTO audit_log ...`, ifID, actor, action, target, payloadHash, prevHash, entryHash, metadataJSON, timestamp)
    return tx.Commit()
}
```

### 2. Trigger Postgres bloqueia UPDATE/DELETE

```sql
CREATE OR REPLACE FUNCTION audit_log_immutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log é imutável';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update
    BEFORE UPDATE OR DELETE ON tenant_data.audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_immutable();
```

### 3. WORM storage como cold archive (S3 Object Lock)

Audit log com > 90 dias → exporta pra S3 com Object Lock (Compliance mode). Read-only imutável por X anos.

## Consequências

**Positivas:**
- ✅ Tamper-evident verificável: comando `cmd/_verify` valida O(n) e mostra cadeia quebrada.
- ✅ Imutável via DB (não convenção): DBA não consegue editar.
- ✅ SOC 2 Type II satisfeito (tamper-evident log é requirement explícito).
- ✅ LGPD: registo de tratamento de dados evidenciado.
- ✅ Auditoria forense: extração completa de qualquer envio em < 60s.

**Negativas:**
- ❌ Storage cresce rápido: ~1KB/entry, 10k entries/dia = 10MB/dia = 3.6GB/ano. Mitigável: archiving S3 + retention policy.
- ❌ Performance: INSERT serializado (BEGIN IMMEDIATE). Mitigável: batch inserts em alta carga, separar tabela por mês (partitioning).
- ❌ Recovery de disaster: precisa restaurar DB + replay audit chain.

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| **Blockchain privada** | Overkill. SHA-256 chain + trigger DB é suficiente pra auditoria. |
| **WORM storage primário** | Latência alta pra query. Usar como cold archive, não hot path. |
| **Audit log em append-only file** | Não escala horizontalmente, backup nightmare. |

## Notas de implementação

- Partitioning por mês: `audit_log_2026_07`, `audit_log_2026_08`, etc. Trigger herdado.
- Compressão: `pg_lz` ou zstd em colunas `metadata` (JSONB).
- Retention: hot 90d Postgres, warm 1y S3, cold 7y S3 Glacier.
- Backup: PITR (Point-in-Time Recovery) no RDS + WAL archiving.
- Verification CLI: `cmd/_verify` rodado em CI diariamente + on-demand via API admin.