-- Migration 015: Stripe Billing
-- Sprint 45 — StripeBilling
--
-- Adiciona colunas Stripe na tabela ifs para billing integrado.
-- Suporta: customer ID, subscription ID, price ID, trial end.
--
-- NOTA: Esta migration é idempotente por design (migration runner
-- verifica se já foi aplicada antes de executar).
-- Não usa BEGIN/COMMIT — runner gerencia transação.

-- Stripe customer + subscription IDs
ALTER TABLE ifs ADD COLUMN stripe_customer_id TEXT;
ALTER TABLE ifs ADD COLUMN stripe_subscription_id TEXT;
ALTER TABLE ifs ADD COLUMN stripe_plan_id TEXT;
ALTER TABLE ifs ADD COLUMN billing_email TEXT;
ALTER TABLE ifs ADD COLUMN trial_ends_at DATETIME;

-- billing_events table para audit trail
CREATE TABLE IF NOT EXISTS billing_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id      TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    stripe_event_id TEXT,
    payload        TEXT,
    processed_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(stripe_event_id)
);

-- Índices (ignora se já existirem)
CREATE INDEX IF NOT EXISTS idx_ifs_stripe_customer ON ifs(stripe_customer_id);
CREATE INDEX IF NOT EXISTS idx_ifs_stripe_subscription ON ifs(stripe_subscription_id);
CREATE INDEX IF NOT EXISTS idx_billing_events_tenant ON billing_events(tenant_id);
