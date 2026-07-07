-- Migration 21 — Sprint 63: MultiRegion — BR-SP1 / BR-SP2 replication.
--
-- Adiciona:
--   1. ifs.region              — região do tenant (br-sp1 | br-sp2)
--   2. region_events         — log de eventos de replicação entre regiões
--   3. replication_status    — status de replicação por região

-- ============================================================
-- Região do tenant
-- ============================================================
ALTER TABLE ifs ADD COLUMN region TEXT NOT NULL DEFAULT 'br-sp1';

-- ============================================================
-- Event log de replicação entre regiões
-- ============================================================
CREATE TABLE IF NOT EXISTS region_events (
    id          TEXT PRIMARY KEY,
    region_from TEXT NOT NULL,  -- região de origem
    region_to   TEXT NOT NULL, -- região de destino
    event_type  TEXT NOT NULL,  -- 'tenant.created', 'envio.created', 'validation.completed'
    entity_type TEXT NOT NULL,   -- 'tenant', 'envio', 'audit_log'
    entity_id   TEXT NOT NULL,
    payload     TEXT,            -- json opcional
    status      TEXT NOT NULL DEFAULT 'pending', -- pending|synced|failed
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    synced_at   DATETIME
);

CREATE INDEX IF NOT EXISTS idx_region_events_status ON region_events(status, created_at);
CREATE INDEX IF NOT EXISTS idx_region_events_entity  ON region_events(entity_type, entity_id);

-- ============================================================
-- Status de replicação por região
-- ============================================================
CREATE TABLE IF NOT EXISTS replication_status (
    region       TEXT PRIMARY KEY,
    last_sync_at DATETIME,
    lag_seconds  INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'healthy', -- healthy|degraded|offline
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
