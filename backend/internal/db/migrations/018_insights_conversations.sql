-- Migration 18 — Sprint 53: AIInsights_v1 — LLM interpreta audit_log.
--
-- Adds:
--   1. ifs.llm_insights_enabled  — feature flag opt-in (default false)
--   2. insights_conversations    — conversa por tenant (suporta context histórico)

-- ============================================================
-- Feature flag na tabela de tenants
-- ============================================================
ALTER TABLE ifs ADD COLUMN llm_insights_enabled INTEGER NOT NULL DEFAULT 0;

-- ============================================================
-- Conversa por tenant (armazena histórico user/assistant)
-- ============================================================
CREATE TABLE IF NOT EXISTS insights_conversations (
    id           TEXT PRIMARY KEY,  -- uuid v4
    if_id        TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    role         TEXT NOT NULL,    -- 'user' | 'assistant'
    content      TEXT NOT NULL,    -- texto da mensagem
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conv_if ON insights_conversations(if_id, created_at);

-- Cleanup: remove conversas com mais de 90 dias (bot cleanup)
-- Trigger: job externo ou cron call /v1/admin/insights/cleanup
