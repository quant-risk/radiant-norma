-- Migration 011 — Sprint 13 (v3.5.2) [S14.2]:
-- Covering indexes para queries críticas de envios (listEnvios,
-- enviosAggregate, insightsKPIs, insightsHeatmap).
--
-- Audit S-B: queries atuais faziam seq scan em produção. Com volume
-- de envios STA (milhões de rows em IFs grandes), performance degrada
-- linearmente sem índice apropriado.
--
-- Indexes adicionados (HIGH do audit, v3.5.0 deepsweep):
--   * (if_id, status) — cobere listEnvios + enviosStats
--   * (if_id, cadoc_code, status, period) — cobere listEnvios
--     (sprint8c_handlers.go:81-96) — sequência exata de filtros.
--   * (if_id, confirmed_at DESC) — partial WHERE confirmed_at NOT NULL;
--     cobere enviosAggregate + insightsKPIs WHERE confirmed_at range
--     (sprint8c_handlers.go:357-369, 412-426).
--   * (if_id, period) — cobere GROUP BY period em dashboard.
--
-- Notas:
--   * idx_envios_status (single-col, low-cardinality) é DROPado —
--     wasted space + queries pelo composite já cobriam.
--   * Event-ordering COALESCE(confirmed_at, sent_at, created_at) DESC
--     não tem functional index (SQLite antigo; Postgres OK). Trade-off
--     aceito: order-by usa idx_envios_if_cadoc_status_period + sort
--     in-memory por event_at. Beats seq scan.

-- Drop idx_envios_status (low-cardinality single-col).
-- SQLite não tem IF EXISTS para DROP INDEX até 3.35; IF EXISTS é
-- aceito em Postgres. Migration runner roda uma única vez por nome
-- de arquivo, então DROP sem IF EXISTS em SQLite funciona (DROP IF
-- EXISTS é mais portável).
DROP INDEX IF EXISTS idx_envios_status;
CREATE INDEX IF NOT EXISTS idx_envios_if_status
    ON envios(if_id, status);

CREATE INDEX IF NOT EXISTS idx_envios_if_cadoc_status_period
    ON envios(if_id, cadoc_code, status, period);

-- Partial index — só rows já confirmadas (a maioria das analíticas
-- filtra por confirmed_at BETWEEN ?).
CREATE INDEX IF NOT EXISTS idx_envios_if_confirmed
    ON envios(if_id, confirmed_at DESC)
    WHERE confirmed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_envios_if_period
    ON envios(if_id, period);

-- Index para listRadarAlerts equivalent em envios status flow
-- (sender dashboards): SELECT WHERE status='pending' OR 'error'.
-- Partial por low-cardinality ainda ajuda (pending/errors são minority).
CREATE INDEX IF NOT EXISTS idx_envios_if_open
    ON envios(if_id, status, created_at DESC)
    WHERE status IN ('pending', 'error', 'processing');
