-- Migration 009 — Sprint 12 (opcional): acknowledge de recommendations.
--
-- Recommendations em /v1/insights/recommendations são computadas
-- dinamicamente (heurística). Sem persistência, user não tem como marcar
-- como "vista/fechada" — refaz toda vez que abre a página.
--
-- Esta tabela rastreia: por IF, qual recommendation ID foi acknowledge
-- e por quem (claims.Sub).
--
-- PK: (if_id, rec_id). Idempotente via ON CONFLICT.

CREATE TABLE IF NOT EXISTS acknowledged_recommendations (
    if_id           TEXT NOT NULL,
    rec_id          TEXT NOT NULL,
    acknowledged_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_by TEXT NOT NULL,

    PRIMARY KEY (if_id, rec_id)
);

CREATE INDEX IF NOT EXISTS idx_ack_rec_if
    ON acknowledged_recommendations(if_id);
