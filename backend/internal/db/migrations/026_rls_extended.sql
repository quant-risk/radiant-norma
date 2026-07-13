-- Migration 026 — Sprint 79: Extended RLS Enforcement
--
-- @postgres-only
--
-- Extende FORCE ROW LEVEL SECURITY (014) para tabelas tenant-scoped
-- criadas APÓS a migração 014 (014 já cobria envios, audit_log,
-- audit_events, rule_failures, disabled_rules, acknowledged_recommendations).
--
-- Novas tabelas com if_id que precisam de RLS:
--   wizard_sessions     (025) — wizard state por tenant
--   range_sessions     (024) — STA chunked upload por tenant
--   webhooks            (019) — webhook registrations por tenant
--   insights_conversations (018) — conversas de insights por tenant
--
-- Sem RLS nestas tabelas, um tenant poderia ler/escrever dados de outro
-- (violação LGPD / SOC 2).
--
-- Esta migration:
--   1. ENABLE ROW LEVEL SECURITY + cria policies USING para cada tabela
--   2. FORCE ROW LEVEL SECURITY em todas (mesmo comportamento de 014)
--
-- Para detalhes sobre SET LOCAL app.if_id → usar helper db.WithoutTx()
-- em internal/db/tenant.go.

-- ============================================================
-- wizard_sessions
-- ============================================================
ALTER TABLE wizard_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS wizard_sessions_tenant_isolation ON wizard_sessions;
CREATE POLICY wizard_sessions_tenant_isolation ON wizard_sessions
    USING (if_id = current_setting('app.if_id', true));
ALTER TABLE wizard_sessions FORCE ROW LEVEL SECURITY;

COMMENT ON POLICY wizard_sessions_tenant_isolation ON wizard_sessions IS
    'Sprint 79: tenant isolation ENFORCED (FORCE). Wizard sessions are tenant-scoped.';

-- ============================================================
-- range_sessions
-- ============================================================
ALTER TABLE range_sessions ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS range_sessions_tenant_isolation ON range_sessions;
CREATE POLICY range_sessions_tenant_isolation ON range_sessions
    USING (if_id = current_setting('app.if_id', true));
ALTER TABLE range_sessions FORCE ROW LEVEL SECURITY;

COMMENT ON POLICY range_sessions_tenant_isolation ON range_sessions IS
    'Sprint 79: tenant isolation ENFORCED (FORCE). Range upload sessions are tenant-scoped.';

-- ============================================================
-- webhooks
-- ============================================================
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS webhooks_tenant_isolation ON webhooks;
CREATE POLICY webhooks_tenant_isolation ON webhooks
    USING (if_id = current_setting('app.if_id', true));
ALTER TABLE webhooks FORCE ROW LEVEL SECURITY;

COMMENT ON POLICY webhooks_tenant_isolation ON webhooks IS
    'Sprint 79: tenant isolation ENFORCED (FORCE). Webhooks are tenant-scoped.';

-- ============================================================
-- insights_conversations
-- ============================================================
ALTER TABLE insights_conversations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS insights_conversations_tenant_isolation ON insights_conversations;
CREATE POLICY insights_conversations_tenant_isolation ON insights_conversations
    USING (if_id = current_setting('app.if_id', true));
ALTER TABLE insights_conversations FORCE ROW LEVEL SECURITY;

COMMENT ON POLICY insights_conversations_tenant_isolation ON insights_conversations IS
    'Sprint 79: tenant isolation ENFORCED (FORCE). Insights conversations are tenant-scoped.';
