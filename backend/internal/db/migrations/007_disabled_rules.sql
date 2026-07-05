-- Migration 007 — Sprint 11: persistência de regras desabilitadas por IF.
--
-- Antes: regras habilitadas/desabilitadas ficavam em localStorage do browser
-- (frontend-only). Problema: cada device/usuário tem estado separado, sem
-- auditoria, sem compartilhamento entre membros do mesmo IF.
--
-- Agora: backend tem source of truth. Toggle persiste em DB, emite audit
-- event (rule.disabled / rule.enabled), frontend consulta + atualiza.
--
-- Schema:
--   disabled_rules(if_id, rule_code, disabled_at, disabled_by)
--   PK: (if_id, rule_code) — 1 row por IF×rule
--
-- Não precisa de FK pra rules table (rules é hardcoded do schema, não tem
-- cadastro no DB). Validação: se regra for removida do schema, fica órfã
-- na tabela mas inofensiva (não é mais referenciada em /v1/rules).

CREATE TABLE IF NOT EXISTS disabled_rules (
    if_id        TEXT NOT NULL,
    rule_code    TEXT NOT NULL,
    disabled_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_by  TEXT NOT NULL,  -- user email ou sub do JWT

    PRIMARY KEY (if_id, rule_code)
);

CREATE INDEX IF NOT EXISTS idx_disabled_rules_if
    ON disabled_rules(if_id);
