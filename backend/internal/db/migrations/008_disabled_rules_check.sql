-- Migration 008 — Sprint 12: CHECK constraint em disabled_rules.rule_code.
--
-- C32.4 + C32.11: defesa em profundidade no DB layer. Backend valida via
-- regex, mas CHECK constraint garante integridade mesmo se handler
-- tiver bug. Migration idempotente.
--
-- Padrão BACEN: [A-Z][0-9]{1,3} (B12, F23, S05, C001).
--
-- IMPORTANTE: SQLite não suporta ALTER TABLE ADD CONSTRAINT diretamente.
-- Estratégia: cria nova tabela, copia dados, drop original, renomeia.
-- Migration runner já wrappa em transaction — não usar BEGIN/COMMIT aqui.

CREATE TABLE IF NOT EXISTS disabled_rules_new (
    if_id        TEXT NOT NULL,
    rule_code    TEXT NOT NULL CHECK (
        length(rule_code) BETWEEN 2 AND 4
        AND rule_code GLOB '[A-Z][0-9][0-9]*'
    ),
    disabled_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_by  TEXT NOT NULL,

    PRIMARY KEY (if_id, rule_code)
);

INSERT OR IGNORE INTO disabled_rules_new
    SELECT if_id, rule_code, disabled_at, disabled_by FROM disabled_rules;

DROP TABLE IF EXISTS disabled_rules;

ALTER TABLE disabled_rules_new RENAME TO disabled_rules;

CREATE INDEX IF NOT EXISTS idx_disabled_rules_if
    ON disabled_rules(if_id);
