-- Migration 012 — Sprint 13 (v3.5.2) [S14.3]:
-- Row Level Security (RLS) policies Postgres para tenant isolation em
-- camada de banco.
--
-- @postgres-only
-- Marcador indica que migrate.go deve skipar esta migration em SQLite
-- (driver de teste). SQLite não tem RLS. Em produção Postgres, aplicar
-- manualmente: `psql -f migrations/012_rls_policies.sql -d radiant`
-- ou via pipeline CI dedicada.
--
-- CRITICAL (audit S-B): Postgres não tinha RLS. Toda tenancy dependia
-- de WHERE if_id=? em cada query — bug class (LGPD risk). Esta
-- migration cria policies USING (if_id = current_setting('app.if_id'))
-- para tabelas tenant-scoped.
--
-- IMPORTANTE — modelo incremental:
--   * ENABLE RLS (permissivo): policies existem mas não são FORÇADAS.
--     Queries sem SET app.if_id retornam TODAS as rows (sem regressão).
--   * FORCE RLS: queries SEMPRE avaliam policy. Aplicação precisa
--     fazer tx.Exec("SET LOCAL app.if_id = ?", ifID) em cada
--     transação que toca tabela tenant-scoped. Sem isso, queries
--     falham/viram 0 rows.
--
-- Estamos na fase ENABLE. Para ativar FORCE, fazer follow-up Sprint
-- que adiciona SET LOCAL em db.BeginTx wrappers + testes de
-- cross-tenant attempt (deveria dar 0 rows).
--
-- Tabelas com RLS:
--   envios, audit_log, audit_events, rule_failures, disabled_rules,
--   acknowledged_recommendations
--
-- Tabelas SEM RLS (são globais ou single-tenant por design):
--   ifs (claramente precisa de admin only), schema_versions,
--   criticas, radar_alerts (regulatory), radar_baselines,
--   schema_migrations

-- ============================================================
-- envios
-- ============================================================
ALTER TABLE envios ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS envios_tenant_isolation ON envios;
CREATE POLICY envios_tenant_isolation ON envios
    USING (if_id = current_setting('app.if_id', true));

-- ============================================================
-- audit_log
-- ============================================================
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_log_tenant_isolation ON audit_log;
CREATE POLICY audit_log_tenant_isolation ON audit_log
    USING (
        if_id IS NULL  -- global/system actions (admin pre-fix)
        OR if_id = current_setting('app.if_id', true)
    );

-- ============================================================
-- audit_events
-- ============================================================
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS audit_events_tenant_isolation ON audit_events;
CREATE POLICY audit_events_tenant_isolation ON audit_events
    USING (
        if_id IS NULL
        OR if_id = current_setting('app.if_id', true)
    );

-- ============================================================
-- rule_failures
-- ============================================================
ALTER TABLE rule_failures ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS rule_failures_tenant_isolation ON rule_failures;
CREATE POLICY rule_failures_tenant_isolation ON rule_failures
    USING (if_id = current_setting('app.if_id', true));

-- ============================================================
-- disabled_rules
-- ============================================================
ALTER TABLE disabled_rules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS disabled_rules_tenant_isolation ON disabled_rules;
CREATE POLICY disabled_rules_tenant_isolation ON disabled_rules
    USING (if_id = current_setting('app.if_id', true));

-- ============================================================
-- acknowledged_recommendations
-- ============================================================
ALTER TABLE acknowledged_recommendations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS ack_rec_tenant_isolation ON acknowledged_recommendations;
CREATE POLICY ack_rec_tenant_isolation ON acknowledged_recommendations
    USING (if_id = current_setting('app.if_id', true));

-- ============================================================
-- GRANTS: role que app usa precisa SELECT/INSERT/UPDATE/DELETE nas
-- tabelas com RLS. Postgres default é PUBLIC + table owner. Se a
-- app usa role dedicada (recomendado), adicionar GRANT explícito.
-- Skip aqui — fora de escopo sem saber o role name.
-- ============================================================

-- ============================================================
-- Comentário para tabelas: documentar intenção.
-- ============================================================
COMMENT ON POLICY envios_tenant_isolation ON envios IS
    'Sprint 13 [S14.3]: tenant isolation opt-in. Para enforcement real (FORCE), set SET LOCAL app.if_id em tx.';
