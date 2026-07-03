-- Migration 003: Remove UNIQUE constraint em radar_alerts.
--
-- Problema: UNIQUE(cadoc_code, alert_type, detected_at, title) causa
-- "constraint failed" em scans consecutivos muito rápidos (mesmo
-- detected_at em microssegundos próximos).
--
-- Solução: alertas com mesmo título mas detectados em momentos diferentes
-- DEVEM poder coexistir. UNIQUE não tem propósito útil aqui — alertas
-- são eventos, não entidades deduplicáveis.
--
-- Substitui por INDEX (não-único) para queries de "último alerta por cadoc".
--
-- Detectado por: TestScanSource_HashChanged_CreatesAlert (Sprint 5 v1.4.0).

DROP INDEX IF EXISTS idx_radar_unique;

CREATE INDEX IF NOT EXISTS idx_radar_cadoc_type
    ON radar_alerts(cadoc_code, alert_type, detected_at DESC);

-- Cria tabela nova sem UNIQUE constraint
CREATE TABLE radar_alerts_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cadoc_code      TEXT NOT NULL,
    alert_type      TEXT NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'info',
    title           TEXT NOT NULL,
    description     TEXT,
    source_url      TEXT,
    detected_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at     DATETIME
);

INSERT INTO radar_alerts_new
    SELECT id, cadoc_code, alert_type, severity, title, description, source_url, detected_at, resolved_at
    FROM radar_alerts;

DROP TABLE radar_alerts;

ALTER TABLE radar_alerts_new RENAME TO radar_alerts;

-- Recria índices (mantém performance, remove UNIQUE)
CREATE INDEX IF NOT EXISTS idx_radar_cadoc ON radar_alerts(cadoc_code, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_radar_unresolved ON radar_alerts(resolved_at) WHERE resolved_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_radar_cadoc_type ON radar_alerts(cadoc_code, alert_type, detected_at DESC);