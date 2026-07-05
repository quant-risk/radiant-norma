-- Migration 013 — Sprint 13 (v3.5.2) [S14.4]:
-- CHECK constraints em envios (status + period + data_base).
--
-- Audit S-B [MEDIUM]: envios.status sem CHECK constraint — typos
-- passam silenciosamente, worker fica em loop.
-- Audit S-B [MEDIUM]: envios.period aceita qualquer string livre;
-- convention é 'MM/YYYY' mas DB não enforça.
-- Audit S-B [MEDIUM]: envios.data_base TEXT sem CHECK (deveria ser
-- YYYY-MM-DD).
--
-- Defense-in-depth: backend valida via app code, mas DB enforçar
-- garante integridade mesmo se handler tiver bug.
--
-- Pattern: SQLite não tem ALTER TABLE ADD CONSTRAINT → recreate-table.

-- ============================================================
-- envios: status enum + period format + data_base format
-- Schema completo (001 + 006 + 010) preservado: rules_passed,
-- rules_failed, duration_ms, approver (Sprint 8c).
-- ============================================================
DROP TABLE IF EXISTS envios_new;
CREATE TABLE envios_new (
    id              TEXT PRIMARY KEY,           -- UUID
    if_id           TEXT NOT NULL REFERENCES ifs(id),
    cadoc_code      TEXT NOT NULL,
    data_base       TEXT NOT NULL CHECK (
        data_base GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
    ),
    remessa         INTEGER NOT NULL DEFAULT 1,
    xml_hash        TEXT NOT NULL,
    zip_hash        TEXT NOT NULL,
    xml_content     TEXT NOT NULL DEFAULT '',  -- 002: full XML
    zip_content     BLOB,                       -- 002: ZIP file
    protocol_sta    TEXT,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending','validated','sent','accepted','rejected',
                   'error','processing','dead_letter')
    ),
    error_message   TEXT,
    error_code      TEXT,
    sent_at         DATETIME,
    confirmed_at    DATETIME,
    period          TEXT NOT NULL DEFAULT '00/0000' CHECK (
        period GLOB '[0-9][0-9]/[0-9][0-9][0-9][0-9]'
    ),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- Sprint 8c columns (006):
    rules_passed    INTEGER NOT NULL DEFAULT 0,
    rules_failed    INTEGER NOT NULL DEFAULT 0,
    duration_ms     INTEGER,
    approver        TEXT,
    -- Worker hardening columns (005):
    attempts        INTEGER NOT NULL DEFAULT 0,
    next_retry_at   DATETIME,
    processing_started_at DATETIME
);
INSERT INTO envios_new (id, if_id, cadoc_code, data_base, remessa, xml_hash,
                          zip_hash, xml_content, zip_content, protocol_sta,
                          status, error_message, error_code, sent_at,
                          confirmed_at, period, created_at,
                          rules_passed, rules_failed, duration_ms, approver,
                          attempts, next_retry_at, processing_started_at)
SELECT id, if_id, cadoc_code, data_base, remessa, xml_hash,
       zip_hash, COALESCE(xml_content, ''), zip_content, protocol_sta,
       -- Sanitize status: se row já tem status fora da whitelist,
       -- coerce para 'error' (preserva audit, sinaliza bug histórico).
       CASE WHEN status IN ('pending','validated','sent','accepted','rejected',
                            'error','processing','dead_letter')
            THEN status ELSE 'error' END,
       error_message, error_code, sent_at, confirmed_at,
       -- Sanitize period: se row tem period fora do formato,
       -- coerce para '00/0000' (preserva row, indica dado sujo).
       CASE WHEN period GLOB '[0-9][0-9]/[0-9][0-9][0-9][0-9]'
            THEN period ELSE '00/0000' END,
       created_at,
       COALESCE(rules_passed, 0),
       COALESCE(rules_failed, 0),
       duration_ms,
       approver,
       COALESCE(attempts, 0),
       next_retry_at,
       processing_started_at
FROM envios;
DROP TABLE envios;
ALTER TABLE envios_new RENAME TO envios;

-- Re-cria índices perdidos no recreate, incluindo worker hardening
-- (idx_envios_pending_retry de 005).
CREATE INDEX IF NOT EXISTS idx_envios_pending_retry
    ON envios(status, next_retry_at)
    WHERE status = 'pending';

-- Re-cria índices perdidos no recreate (idx_envios_if, idx_envios_cadoc,
-- idx_envios_if_status, idx_envios_if_cadoc_status_period, idx_envios_if_confirmed,
-- idx_envios_if_period, idx_envios_if_open).
CREATE INDEX IF NOT EXISTS idx_envios_if ON envios(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_envios_cadoc ON envios(cadoc_code, data_base);
CREATE INDEX IF NOT EXISTS idx_envios_if_status ON envios(if_id, status);
CREATE INDEX IF NOT EXISTS idx_envios_if_cadoc_status_period
    ON envios(if_id, cadoc_code, status, period);
CREATE INDEX IF NOT EXISTS idx_envios_if_confirmed
    ON envios(if_id, confirmed_at DESC)
    WHERE confirmed_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_envios_if_period ON envios(if_id, period);
CREATE INDEX IF NOT EXISTS idx_envios_if_open
    ON envios(if_id, status, created_at DESC)
    WHERE status IN ('pending', 'error', 'processing');
