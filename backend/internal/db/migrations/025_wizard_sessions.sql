-- Migration 25 — Sprint 57: NormaGeneratorFoundation — wizard state machine
--
-- Wizard sessions para geração de CADOCs em múltiplos passos:
--
-- Steps: select_cadoc → select_source → map_fields → preview → generate
--
-- TTL: sessões abandonadas (>2h since last update) são limpas por job periódica.
-- Sessões completadas ou failed são mantidas por 24h para debug.

-- ============================================================
-- Wizard sessions
-- ============================================================
CREATE TABLE IF NOT EXISTS wizard_sessions (
    id              TEXT PRIMARY KEY,
    if_id           TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    step            TEXT NOT NULL DEFAULT 'select_cadoc',
    -- step: select_cadoc | select_source | map_fields | preview | generate
    cadoc_code      TEXT,                  -- CADOC selecionado (ex: "3040")
    source_type     TEXT,                  -- source seleccionado: manual|file|api|db|mcp
    canonical_json  TEXT,                  -- CanonicalDocument em progresso (JSON)
    generated_xml  TEXT,                  -- XML gerado (quando step=generate)
    field_mapping  TEXT,                  -- JSON do field mapping ({cosif_field: value})
    errors          TEXT,                  -- JSON array de erros
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_wizard_if_step ON wizard_sessions(if_id, step);
CREATE INDEX IF NOT EXISTS idx_wizard_updated ON wizard_sessions(updated_at);
