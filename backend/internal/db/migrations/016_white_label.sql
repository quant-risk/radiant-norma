-- Migration 016: WhiteLabel Branding
-- Sprint 46 — WhiteLabel
--
-- Adiciona colunas de branding na tabela ifs para WhiteLabel:
-- logo, cores primárias/secundárias, domínio customizado, slug.
--
-- Rollback: DROP COLUMN (PostgreSQL 14+).

-- Branding columns
ALTER TABLE ifs ADD COLUMN logo_url TEXT;
ALTER TABLE ifs ADD COLUMN primary_color TEXT DEFAULT '#3b6ef5';
ALTER TABLE ifs ADD COLUMN secondary_color TEXT DEFAULT '#1a2a5e';
ALTER TABLE ifs ADD COLUMN custom_domain TEXT;
ALTER TABLE ifs ADD COLUMN tenant_slug TEXT;

-- Índices
CREATE UNIQUE INDEX IF NOT EXISTS idx_ifs_tenant_slug ON ifs(tenant_slug)
    WHERE tenant_slug IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ifs_custom_domain ON ifs(custom_domain)
    WHERE custom_domain IS NOT NULL;
