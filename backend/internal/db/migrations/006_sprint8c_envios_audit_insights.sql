-- Migration 006 — Sprint 8c: enriquecimento de envios + tabelas para insights.
--
-- Adiciona colunas que frontend precisa pra /envios (Sprint 8c), e
-- prepara estrutura para audit_events legíveis pelo frontend (não
-- confundir com audit_log que é a chain hash-only — esta tabela é
-- denormalizada pra queries).
--
-- Trigger: Sprint 9 v3.0.0 frontend redesign tem empty states em
-- /envios e /insights porque backend não expunha dados. Migration 006
-- + cmd/seed + handlers novos destravam.

-- ============================================================
-- ENVIOS — enriquece com regras e metadados do STA
-- ============================================================
ALTER TABLE envios ADD COLUMN rules_passed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE envios ADD COLUMN rules_failed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE envios ADD COLUMN period TEXT;                   -- 'MM/YYYY' (e.g., '05/2026')
ALTER TABLE envios ADD COLUMN duration_ms INTEGER;           -- tempo de validação
ALTER TABLE envios ADD COLUMN approver TEXT;                 -- quem aprovou (system, user)

CREATE INDEX IF NOT EXISTS idx_envios_period ON envios(if_id, period);

-- ============================================================
-- AUDIT EVENTS — view legível do audit_log (denormalizada)
-- ============================================================
-- Razão: audit_log tem prev_hash + chain — ótimo pra integridade
-- (LGPD/SOC2) mas pesado pra queries e payload é só hash.
-- Esta tabela denormaliza eventos com payload completo pra UI
-- (Dashboard activity feed, /auditoria page).
--
-- Sync: cmd/api emite INSERT em ambas (audit_log pra chain +
-- audit_events pra UI). Mesmo created_at → mesmo source.
CREATE TABLE IF NOT EXISTS audit_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_log_id    INTEGER NOT NULL REFERENCES audit_log(id),
    if_id           TEXT,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,        -- 'envio.created', 'envio.approved', etc
    target          TEXT,
    description     TEXT,                 -- copy legível
    payload         TEXT,                 -- JSON context
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_events_if ON audit_events(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_events_created ON audit_events(created_at DESC);

-- ============================================================
-- RULE FAILURES — para top-failing + heatmap
-- ============================================================
-- Cada falha detectada em uma validação gera 1 row aqui.
-- Permite heatmap temporal + top regras falhando sem precisar
-- parsear XMLs novamente.
CREATE TABLE IF NOT EXISTS rule_failures (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    envio_id        TEXT NOT NULL REFERENCES envios(id),
    if_id           TEXT NOT NULL,
    cadoc_code      TEXT NOT NULL,
    rule_code       TEXT NOT NULL,        -- 'F23', 'B12', etc
    rule_severity   TEXT NOT NULL,        -- 'E' | 'A' | 'I'
    failed_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rule_failures_cadoc ON rule_failures(cadoc_code, failed_at);
CREATE INDEX IF NOT EXISTS idx_rule_failures_rule ON rule_failures(rule_code, failed_at);
CREATE INDEX IF NOT EXISTS idx_rule_failures_if ON rule_failures(if_id, failed_at);