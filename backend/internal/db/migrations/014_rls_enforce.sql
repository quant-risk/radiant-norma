-- Migration 014 — Sprint 30 (v3.33.0) — FORCE ROW LEVEL SECURITY.
--
-- @postgres-only
-- Marcador indica que migrate.go deve skipar esta migration em SQLite.
-- SQLite não tem FORCE RLS (apenas ENABLE que é per-table e opcional).
-- Em produção Postgres, esta migration transforma policies opt-in
-- (criadas em 012) em MANDATORY para TODAS as roles, incluindo table owner.
--
-- CRITICAL (audit SOC 2 / LGPD): até 012, policies existiam mas só eram
-- avaliadas para non-owner roles. Se app usava role = table owner,
-- bypassava RLS completamente. Esta migration fecha o gap.
--
-- IMPORTANTE: a partir desta migration, TODA query a tabela tenant-scoped
-- DEVE estar em transação que fez `SET LOCAL app.if_id = <ifID>` ANTES.
-- Sem isso:
--   - Postgres retorna 0 rows (policy USING falha porque app.if_id é NULL).
--   - Aplicação vai parecer "perdeu dados" — fail-loud > fail-silent.
--
-- Helper centralizado em internal/db/tenant.go::WithTenantTx encapsula
-- BeginTx + SET LOCAL + Commit/Rollback. Toda chamada deve passar por lá.
--
-- Whitelist GLOBAL (não recebe FORCE — são tabelas cross-tenant por design):
--   ifs                       — admin only, sem tenant scope
--   schema_versions           — versionamento de schema (auditoria)
--   criticas                  — catálogo BACEN, imutável cross-tenant
--   radar_alerts              — alertas regulatórios BACEN, cross-tenant
--   radar_baselines           — baseline regulatório, cross-tenant
--   schema_migrations         — controle interno migrate.go
--
-- Tabelas com FORCE (todas que estão em 012):
--   envios, audit_log, audit_events, rule_failures, disabled_rules,
--   acknowledged_recommendations

-- ============================================================
-- FORCE RLS: policies passam a ser avaliadas para TODOS, incluindo owner.
-- ============================================================

-- envios
ALTER TABLE envios FORCE ROW LEVEL SECURITY;

-- audit_log (if_id IS NULL é admin/system — escapável)
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;

-- audit_events (if_id IS NULL é admin/system — escapável)
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

-- rule_failures
ALTER TABLE rule_failures FORCE ROW LEVEL SECURITY;

-- disabled_rules
ALTER TABLE disabled_rules FORCE ROW LEVEL SECURITY;

-- acknowledged_recommendations
ALTER TABLE acknowledged_recommendations FORCE ROW LEVEL SECURITY;

-- ============================================================
-- Comentários: documentar intenção + estado pós-FORCE.
-- ============================================================

COMMENT ON POLICY envios_tenant_isolation ON envios IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED (FORCE). Helper WithTenantTx REQUIRED para acesso.';

COMMENT ON POLICY audit_log_tenant_isolation ON audit_log IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED. Admin actions (if_id IS NULL) seguem acessíveis.';

COMMENT ON POLICY audit_events_tenant_isolation ON audit_events IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED. Admin actions (if_id IS NULL) seguem acessíveis.';

COMMENT ON POLICY rule_failures_tenant_isolation ON rule_failures IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED.';

COMMENT ON POLICY disabled_rules_tenant_isolation ON disabled_rules IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED.';

COMMENT ON POLICY ack_rec_tenant_isolation ON acknowledged_recommendations IS
    'Sprint 30 [S30.3]: tenant isolation ENFORCED.';