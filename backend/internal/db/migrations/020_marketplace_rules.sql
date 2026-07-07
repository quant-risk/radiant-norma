-- Migration 20 — Sprint 62: Marketplace — Catálogo de regras customizadas.
--
-- Adiciona:
--   1. marketplace_rules     — catálogo de regras compartilháveis
--   2. marketplace_installs — regras instaladas por tenant
--   3. marketplace_ratings  — ratings (1-5 estrelas) por tenant por regra

-- ============================================================
-- Catálogo de regras compartilháveis
-- ============================================================
CREATE TABLE IF NOT EXISTS marketplace_rules (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT,
    code          TEXT NOT NULL,        -- ex: "CUSTOM_001"
    cadoc         TEXT NOT NULL,       -- ex: "3040"
    rule_type     TEXT NOT NULL,       -- 'format' | 'semantic' | 'crossdoc' | 'raw'
    config        TEXT,                -- JSON com parâmetros da regra
    author_if_id  TEXT NOT NULL,       -- IF que publicou
    author_name   TEXT,                -- nome do autor/companhia
    rating        REAL NOT NULL DEFAULT 0,
    rating_count  INTEGER NOT NULL DEFAULT 0,
    install_count INTEGER NOT NULL DEFAULT 0,
    tags          TEXT,                -- csv: "credito,limite,risco"
    active        INTEGER NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_marketplace_cadoc   ON marketplace_rules(cadoc, active);
CREATE INDEX IF NOT EXISTS idx_marketplace_author  ON marketplace_rules(author_if_id);
CREATE INDEX IF NOT EXISTS idx_marketplace_rating  ON marketplace_rules(rating DESC, install_count DESC)
    WHERE active = 1;

-- ============================================================
-- Regras instaladas por tenant
-- ============================================================
CREATE TABLE IF NOT EXISTS marketplace_installs (
    id           TEXT PRIMARY KEY,
    rule_id      TEXT NOT NULL REFERENCES marketplace_rules(id) ON DELETE CASCADE,
    if_id        TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rule_id, if_id)
);

CREATE INDEX IF NOT EXISTS idx_installs_if    ON marketplace_installs(if_id);
CREATE INDEX IF NOT EXISTS idx_installs_rule ON marketplace_installs(rule_id);

-- ============================================================
-- Ratings por tenant por regra
-- ============================================================
CREATE TABLE IF NOT EXISTS marketplace_ratings (
    rule_id  TEXT NOT NULL REFERENCES marketplace_rules(id) ON DELETE CASCADE,
    if_id    TEXT NOT NULL REFERENCES ifs(id) ON DELETE CASCADE,
    stars    INTEGER NOT NULL CHECK(stars >= 1 AND stars <= 5),
    PRIMARY KEY(rule_id, if_id)
);

CREATE INDEX IF NOT EXISTS idx_ratings_rule ON marketplace_ratings(rule_id);
