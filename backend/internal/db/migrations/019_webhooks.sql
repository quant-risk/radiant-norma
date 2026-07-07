-- Migration 19 — Sprint 61: Webhooks Outbound
--
-- Adiciona:
--   1. webhooks          — registro de webhooks por tenant
--   2. webhook_deliveries — log de tentativas de entrega (com retry)

-- ============================================================
-- Webhook registrations
-- ============================================================
CREATE TABLE IF NOT EXISTS webhooks (
    id          TEXT PRIMARY KEY,
    if_id       TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    secret      TEXT,                     -- HMAC secret (opcional, vazio = sem assinatura)
    events      TEXT NOT NULL,            -- csv: "validation.completed,schema.changed" ou "*"
    description TEXT,
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_if    ON webhooks(if_id, active);
CREATE INDEX IF NOT EXISTS idx_webhook_events ON webhooks(events);

-- ============================================================
-- Delivery log
-- ============================================================
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id               TEXT PRIMARY KEY,
    webhook_id       TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event            TEXT NOT NULL,  -- "validation.completed", "schema.changed", etc.
    payload          TEXT NOT NULL,  -- json do evento
    status           TEXT NOT NULL,  -- pending|success|failed|retrying
    http_status      INTEGER,         -- código HTTP da resposta (se attempt > 0)
    response_body    TEXT,            -- primeiros 2KB da resposta
    attempt          INTEGER NOT NULL DEFAULT 0,
    next_retry_at    DATETIME,        -- próximo retry (se status=retrying)
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivered_at     DATETIME         -- timestamp final (success ou descarte)
);

CREATE INDEX IF NOT EXISTS idx_delivery_webhook ON webhook_deliveries(webhook_id, created_at);
CREATE INDEX IF NOT EXISTS idx_delivery_status ON webhook_deliveries(status, next_retry_at)
    WHERE status IN ('pending', 'retrying');
