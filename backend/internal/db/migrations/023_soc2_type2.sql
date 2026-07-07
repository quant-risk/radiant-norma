-- Migration 23 — Sprint 65: SOC 2 Type II — Continuous evidence collection.
--
-- Adiciona:
--   1. soc2_evidence_log     — log imutável de evidências coletadas
--   2. soc2_control_status  — status contínuo de cada controle por período
--   3. soc2_findings        — findings de auditoria levantados

-- ============================================================
-- Log de evidências (imutável — append only)
-- ============================================================
CREATE TABLE IF NOT EXISTS soc2_evidence_log (
    id            TEXT PRIMARY KEY,
    control_id   TEXT NOT NULL,   -- ex: "CC6.1"
    criterion    TEXT NOT NULL,   -- ex: "CC6"
    evidence_type TEXT NOT NULL,   -- 'automated_check' | 'manual_review' | 'system_log' | 'document'
    evidence     TEXT NOT NULL,   -- json com detalhes da evidência
    result       TEXT NOT NULL,   -- 'pass' | 'fail' | 'warning' | 'not_applicable'
    metadata     TEXT,            -- json adicional (hash do log, etc.)
    collected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    collected_by TEXT NOT NULL     -- 'system' | 'auditor_id'
);

CREATE INDEX IF NOT EXISTS idx_evidence_control ON soc2_evidence_log(control_id, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_evidence_criterion ON soc2_evidence_log(criterion, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_evidence_result  ON soc2_evidence_log(result, collected_at DESC);

-- ============================================================
-- Status de controle por período
-- ============================================================
CREATE TABLE IF NOT EXISTS soc2_control_status (
    id           TEXT PRIMARY KEY,
    control_id   TEXT NOT NULL,
    period_start DATETIME NOT NULL,
    period_end   DATETIME NOT NULL,
    status       TEXT NOT NULL,    -- 'compliant' | 'non_compliant' | 'in_progress' | 'not_applicable'
    findings     INTEGER NOT NULL DEFAULT 0,
    last_evidence_at DATETIME,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(control_id, period_start, period_end)
);

CREATE INDEX IF NOT EXISTS idx_control_status_control ON soc2_control_status(control_id, period_start DESC);

-- ============================================================
-- Findings de auditoria
-- ============================================================
CREATE TABLE IF NOT EXISTS soc2_findings (
    id           TEXT PRIMARY KEY,
    control_id   TEXT NOT NULL,
    finding_id   TEXT NOT NULL,   -- ex: "CC6.1-2026-Q2-001"
    severity     TEXT NOT NULL,   -- 'critical' | 'high' | 'medium' | 'low'
    description  TEXT NOT NULL,
    evidence_ref TEXT,            -- id da evidência relacionada
    status       TEXT NOT NULL DEFAULT 'open', -- open | in_resolution | closed | accepted_risk
    discovered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at  DATETIME,
    resolved_by  TEXT,
    notes        TEXT
);

CREATE INDEX IF NOT EXISTS idx_findings_control ON soc2_findings(control_id, discovered_at DESC);
CREATE INDEX IF NOT EXISTS idx_findings_status  ON soc2_findings(status, discovered_at DESC);
