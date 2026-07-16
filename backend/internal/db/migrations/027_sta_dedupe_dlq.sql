-- Migration 027 — Phase 4: STA deduplication + DLQ visibility
--
-- Deduplication:
--   - idempotency_key: allows clients to pass explicit idempotency key
--     to prevent double-submission on retries (e.g., network timeout → retry).
--   - Unique partial index on (if_id, idempotency_key) WHERE idempotency_key IS NOT NULL
--     → same key + same tenant = duplicate (rejected at DB level).
--   - xml_hash dedup still done at handler level (for submissions without key).
--
-- DLQ visibility:
--   - idx_envios_dead_letter:快速 access dead_letter envios by tenant.
--   - No schema change needed — dead_letter status already in CHECK (013).

-- idempotency_key column (Phase 4)
ALTER TABLE envios ADD COLUMN idempotency_key TEXT;

-- Unique constraint on idempotency_key per tenant (where key is not null)
-- Prevents DB-level duplicate rejection on resubmit with same key.
CREATE UNIQUE INDEX IF NOT EXISTS idx_envios_idempotency_key
    ON envios(if_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key != '';

-- Fast access to dead_letter envios by tenant (for GET /v1/envios/dlq)
CREATE INDEX IF NOT EXISTS idx_envios_dead_letter
    ON envios(if_id, created_at DESC)
    WHERE status = 'dead_letter';

-- Fast access to accepted/rejected envios by tenant+hash (for dedup query)
CREATE INDEX IF NOT EXISTS idx_envios_if_cadoc_db_hash
    ON envios(if_id, cadoc_code, data_base, xml_hash)
    WHERE status IN ('pending', 'accepted', 'rejected');
