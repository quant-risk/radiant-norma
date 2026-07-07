-- Migration 22 — Sprint 64: Pilot4 — Banco S3-S4 onboarding.
--
-- Adiciona:
--   1. pilot_programs       — programas piloto (ativa/inativa)
--   2. pilot_participants  — bancos em piloto (S3/S4)
--   3. ifs.segment         — segmento do tenant (s1|s2|s3|s4)
--   4. onboarding_steps     — passos de onboarding por tenant

-- ============================================================
-- Segmento do tenant
-- ============================================================
ALTER TABLE ifs ADD COLUMN segment TEXT NOT NULL DEFAULT 's1';

-- ============================================================
-- Programas piloto
-- ============================================================
CREATE TABLE IF NOT EXISTS pilot_programs (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,                  -- ex: "Banco S3-S4 Pilot 4"
    description TEXT,
    start_date  DATETIME,
    end_date    DATETIME,
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Participantes de programas piloto
-- ============================================================
CREATE TABLE IF NOT EXISTS pilot_participants (
    id          TEXT PRIMARY KEY,
    program_id  TEXT NOT NULL REFERENCES pilot_programs(id) ON DELETE CASCADE,
    if_id       TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'onboarding', -- onboarding|active|churned
    joined_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes       TEXT,
    UNIQUE(program_id, if_id)
);

CREATE INDEX IF NOT EXISTS idx_participants_if   ON pilot_participants(if_id);
CREATE INDEX IF NOT EXISTS idx_participants_prog ON pilot_participants(program_id);

-- ============================================================
-- Passos de onboarding por tenant
-- ============================================================
CREATE TABLE IF NOT EXISTS onboarding_steps (
    id          TEXT PRIMARY KEY,
    if_id       TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    step_key    TEXT NOT NULL,    -- "docs_submitted","cadoc_tested","production_approved"
    status      TEXT NOT NULL DEFAULT 'pending', -- pending|in_progress|completed|skipped
    completed_at DATETIME,
    notes       TEXT,
    UNIQUE(if_id, step_key)
);

CREATE INDEX IF NOT EXISTS idx_onboarding_if ON onboarding_steps(if_id);
