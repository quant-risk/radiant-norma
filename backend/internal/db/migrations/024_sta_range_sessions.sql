-- Migration 24 — Sprint 31: RangeUploadAPI
--
-- Sessões de upload chunkado via STA (Seção 5.6 do manual BACEN).
--
-- Flow:
--   1. POST /v1/sta/range/init  → cria sessão, pede protocolo BACEN
--   2. PUT  /v1/sta/range/{protocolo} → faz upload de chunk (Content-Range)
--   3. GET  /v1/sta/range/{protocolo} → status dos chunks recebidos (resume)
--
-- A tabela é owned pelo tenant (if_id) para permitir que diferentes IFs
-- retomem uploads independentes. Protocolo é único no BACEN.
--
-- RangesJSON armazena []Range como JSON array (serialização de sta.Range).
-- TTL de 24h: sessões abandonadas são limpas por job periódica.

-- ============================================================
-- Range sessions
-- ============================================================
CREATE TABLE IF NOT EXISTS range_sessions (
    id              TEXT PRIMARY KEY,
    if_id           TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    protocolo       TEXT NOT NULL,         -- protocolo BACEN (único)
    total_bytes     INTEGER NOT NULL,
    received_bytes  INTEGER NOT NULL DEFAULT 0,
    ranges_json     TEXT NOT NULL DEFAULT '[]',  -- JSON array of {start,end}
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending|complete|failed|abandoned
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

-- Lookup rápido por protocolo (único no BACEN).
CREATE UNIQUE INDEX IF NOT EXISTS idx_range_protocolo ON range_sessions(protocolo);
-- TTL cleanup.
CREATE INDEX IF NOT EXISTS idx_range_created ON range_sessions(created_at);
-- Lista sessões ativas do tenant.
CREATE INDEX IF NOT EXISTS idx_range_if ON range_sessions(if_id, status);
