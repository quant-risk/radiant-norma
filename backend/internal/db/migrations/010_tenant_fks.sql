-- Migration 010 — Sprint 13 (v3.5.2) [S14.1]:
-- Adiciona FOREIGN KEY (if_id) REFERENCES ifs(id) em 5 tabelas tenant-scoped.
--
-- CRITICAL (audit S-B / v3.4.0): integridade referencial era gap de DB.
-- Tabelas sem FK: audit_log, audit_events, rule_failures, disabled_rules,
-- acknowledged_recommendations. IF deletada deixava rows órfãs. Toda
-- tenancy dependia de WHERE if_id=? em cada query — mas sem FK, INSERT
-- podia gravar if_id de IF que não existe (LGPD risk + bug class).
--
-- Strategy: recreate-table pattern (SQLite não tem ALTER TABLE ADD FK).
-- Pattern é portável pra Postgres também (Postgres suporta ALTER ADD FK
-- mas pra manter migration driver-agnostic, usamos recreate-table).
--
-- Cada seção:
--   1. Cria tabela _new com schema idêntico + FK nova
--   2. Copia APENAS rows que satisfazem a constraint
--   3. Drop original, renomeia _new → original
--   4. Recria índices perdidos
--
-- ON DELETE RESTRICT: IFs são tenant root, não podem ser deletados
-- enquanto têm dados (FORCE tenant isolation no DB layer).
-- audit_log.if_id é nullable (admin/system actions); copia WHERE
-- if_id IS NULL OR matches.
--
-- AUDIT: rows perdidas (if_id órfão ou IF inexistente) devem ser logadas
-- pelo runner. C34.11 documenta gap do contador de skipped rows — fora
-- do escopo desta migration.

-- ============================================================
-- Section 1: audit_log.if_id → ifs(id) ON DELETE RESTRICT
-- audit_log.if_id é nullable (admin/system actions passam NULL).
-- ============================================================
DROP TABLE IF EXISTS audit_log_new;
CREATE TABLE audit_log_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    if_id           TEXT REFERENCES ifs(id) ON DELETE RESTRICT,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    target          TEXT,
    payload_hash    TEXT NOT NULL,
    prev_hash       TEXT NOT NULL,
    entry_hash      TEXT NOT NULL,
    metadata        TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO audit_log_new (id, if_id, actor, action, target, payload_hash,
                            prev_hash, entry_hash, metadata, created_at)
SELECT id, if_id, actor, action, target, payload_hash,
       prev_hash, entry_hash, metadata, created_at
FROM audit_log
WHERE if_id IS NULL OR if_id IN (SELECT id FROM ifs);
DROP TABLE audit_log;
ALTER TABLE audit_log_new RENAME TO audit_log;

CREATE INDEX IF NOT EXISTS idx_audit_if ON audit_log(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_chain ON audit_log(id);

-- ============================================================
-- Section 2: audit_events.if_id → ifs(id) ON DELETE RESTRICT
-- audit_events.if_id é nullable (denormalização de audit_log).
-- ============================================================
DROP TABLE IF EXISTS audit_events_new;
CREATE TABLE audit_events_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    audit_log_id    INTEGER NOT NULL REFERENCES audit_log(id) ON DELETE CASCADE,
    if_id           TEXT REFERENCES ifs(id) ON DELETE RESTRICT,
    actor           TEXT NOT NULL,
    action          TEXT NOT NULL,
    target          TEXT,
    description     TEXT,
    payload         TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO audit_events_new (id, audit_log_id, if_id, actor, action, target,
                                description, payload, created_at)
SELECT id, audit_log_id, if_id, actor, action, target,
       description, payload, created_at
FROM audit_events
WHERE if_id IS NULL OR if_id IN (SELECT id FROM ifs);
DROP TABLE audit_events;
ALTER TABLE audit_events_new RENAME TO audit_events;

CREATE INDEX IF NOT EXISTS idx_audit_events_if ON audit_events(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);

-- ============================================================
-- Section 3: rule_failures.if_id → ifs(id) ON DELETE RESTRICT
-- rule_failures.if_id é NOT NULL (cada falha pertence a uma IF).
-- ============================================================
DROP TABLE IF EXISTS rule_failures_new;
CREATE TABLE rule_failures_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    envio_id        TEXT NOT NULL REFERENCES envios(id) ON DELETE CASCADE,
    if_id           TEXT NOT NULL REFERENCES ifs(id) ON DELETE RESTRICT,
    cadoc_code      TEXT NOT NULL,
    rule_code       TEXT NOT NULL,
    rule_severity   TEXT NOT NULL,
    failed_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO rule_failures_new (id, envio_id, if_id, cadoc_code, rule_code,
                                 rule_severity, failed_at)
SELECT id, envio_id, if_id, cadoc_code, rule_code,
       rule_severity, failed_at
FROM rule_failures
WHERE if_id IN (SELECT id FROM ifs);
DROP TABLE rule_failures;
ALTER TABLE rule_failures_new RENAME TO rule_failures;

CREATE INDEX IF NOT EXISTS idx_rule_failures_envio ON rule_failures(envio_id);
CREATE INDEX IF NOT EXISTS idx_rule_failures_if ON rule_failures(if_id, failed_at);
-- Sprint 13 [S14.2]: covering index for heatmap/top-failing queries
CREATE INDEX IF NOT EXISTS idx_rule_failures_if_cadoc
    ON rule_failures(if_id, cadoc_code, failed_at);

-- ============================================================
-- Section 4: disabled_rules.if_id → ifs(id) ON DELETE CASCADE
-- Sprint 12 — regra desabilitada por IF. Cascade: se IF é deletada,
-- suas preferências não fazem sentido.
-- ============================================================
DROP TABLE IF EXISTS disabled_rules_new;
CREATE TABLE disabled_rules_new (
    if_id           TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    rule_code       TEXT NOT NULL CHECK (
        length(rule_code) BETWEEN 2 AND 4
        AND rule_code GLOB '[A-Z][0-9][0-9]*'
    ),
    disabled_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_by     TEXT NOT NULL,
    PRIMARY KEY (if_id, rule_code)
);
INSERT INTO disabled_rules_new (if_id, rule_code, disabled_at, disabled_by)
SELECT if_id, rule_code, disabled_at, disabled_by
FROM disabled_rules
WHERE if_id IN (SELECT id FROM ifs);
DROP TABLE disabled_rules;
ALTER TABLE disabled_rules_new RENAME TO disabled_rules;

CREATE INDEX IF NOT EXISTS idx_disabled_rules_if ON disabled_rules(if_id);

-- ============================================================
-- Section 5: acknowledged_recommendations.if_id → ifs(id) ON DELETE CASCADE
-- user ack/dismiss de insights recommendations. Cascade com IF.
-- ============================================================
DROP TABLE IF EXISTS acknowledged_recommendations_new;
CREATE TABLE acknowledged_recommendations_new (
    if_id           TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    rec_id          TEXT NOT NULL,
    acknowledged_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acknowledged_by TEXT NOT NULL,
    PRIMARY KEY (if_id, rec_id)
);
INSERT INTO acknowledged_recommendations_new
    (if_id, rec_id, acknowledged_at, acknowledged_by)
SELECT if_id, rec_id, acknowledged_at, acknowledged_by
FROM acknowledged_recommendations
WHERE if_id IN (SELECT id FROM ifs);
DROP TABLE acknowledged_recommendations;
ALTER TABLE acknowledged_recommendations_new RENAME TO acknowledged_recommendations;

CREATE INDEX IF NOT EXISTS idx_ack_rec_if
    ON acknowledged_recommendations(if_id);
