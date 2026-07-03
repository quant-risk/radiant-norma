-- Radiant Norma — Schema inicial v1
-- Sprint 3 (P0.2)

-- ============================================================
-- INSTITUIÇÕES FINANCEIRAS (multi-tenant)
-- ============================================================
CREATE TABLE IF NOT EXISTS ifs (
    id              TEXT PRIMARY KEY,           -- UUID
    cnpj            TEXT NOT NULL UNIQUE,       -- CNPJ raiz (8 dígitos)
    nome            TEXT NOT NULL,
    tipo            TEXT NOT NULL,              -- SCD, IP, SEP, BANCO_S3, etc
    segmento        TEXT,                       -- S1, S2, S3, S4, S5
    plano           TEXT NOT NULL DEFAULT 'lite', -- lite, pro, scale, enterprise
    sta_user        TEXT,                       -- Sisbacen user
    sta_service     TEXT DEFAULT 'PSTA300',     -- STA service ID
    cert_a1_path    TEXT,                       -- caminho do cert A1 (se configurado)
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at      DATETIME                    -- soft-delete
);

CREATE INDEX IF NOT EXISTS idx_ifs_cnpj ON ifs(cnpj);
CREATE INDEX IF NOT EXISTS idx_ifs_deleted ON ifs(deleted_at);

-- ============================================================
-- SCHEMA REGISTRY — versionamento por data-base
-- ============================================================
CREATE TABLE IF NOT EXISTS schema_versions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cadoc_code      TEXT NOT NULL,              -- '3040', '3050', '2030'
    effective_from  DATE NOT NULL,              -- YYYY-MM-DD
    source_uri      TEXT NOT NULL,              -- SHA do arquivo XLS/XSD original
    fields_json     TEXT NOT NULL,              -- [{tag, attr, type, required, domain}]
    xsd             TEXT,                       -- XSD gerado
    changelog       TEXT,                       -- mudanças vs versão anterior
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cadoc_code, effective_from)
);

CREATE INDEX IF NOT EXISTS idx_schema_cadoc ON schema_versions(cadoc_code, effective_from DESC);

-- ============================================================
-- CRÍTICAS (regras de validação)
-- ============================================================
CREATE TABLE IF NOT EXISTS criticas (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cadoc_code      TEXT NOT NULL,
    sheet           TEXT NOT NULL,
    codigo          TEXT NOT NULL,
    regra           TEXT,
    descricao       TEXT,
    gravidade       TEXT,                       -- E (erro), A (aviso), I (informativo)
    tipo_indicio    TEXT,                       -- O (obrigatoriedade), etc
    data_base_inicio DATE,                      -- ISO 8601 YYYY-MM-DD
    data_base_fim   DATE,
    mensagem_erro   TEXT,
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    source          TEXT,                       -- planilha ou PDF origem
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cadoc_code, sheet, codigo)
);

CREATE INDEX IF NOT EXISTS idx_criticas_cadoc ON criticas(cadoc_code, codigo);
CREATE INDEX IF NOT EXISTS idx_criticas_enabled ON criticas(enabled);

-- ============================================================
-- ENVIOS (histórico de submissões STA)
-- ============================================================
CREATE TABLE IF NOT EXISTS envios (
    id              TEXT PRIMARY KEY,           -- UUID
    if_id           TEXT NOT NULL REFERENCES ifs(id),
    cadoc_code      TEXT NOT NULL,
    data_base       TEXT NOT NULL,              -- YYYY-MM-DD
    remessa         INTEGER NOT NULL DEFAULT 1,
    xml_hash        TEXT NOT NULL,              -- SHA-256 do XML enviado
    zip_hash        TEXT NOT NULL,              -- SHA-256 do ZIP enviado
    protocol_sta    TEXT,                       -- protocolo STA (até 18 dígitos numéricos)
    status          TEXT NOT NULL DEFAULT 'pending',
                                                -- pending, validated, sent, accepted, rejected, error
    error_message   TEXT,
    error_code      TEXT,                       -- código da crítica BACEN que rejeitou
    sent_at         DATETIME,
    confirmed_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_envios_if ON envios(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_envios_cadoc ON envios(cadoc_code, data_base);
CREATE INDEX IF NOT EXISTS idx_envios_status ON envios(status);

-- ============================================================
-- AUDIT LOG (tamper-evident com hash chain)
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    if_id           TEXT,                       -- null para ações globais (admin)
    actor           TEXT NOT NULL,              -- user email, IP, system
    action          TEXT NOT NULL,              -- 'cadoc.validated', 'cadoc.sent', 'rule.executed', etc
    target          TEXT,                       -- recurso afetado (cadoc, envio, schema, etc)
    payload_hash    TEXT NOT NULL,              -- SHA-256 do payload
    prev_hash       TEXT NOT NULL,              -- hash da entrada anterior (chain)
    entry_hash      TEXT NOT NULL,              -- SHA-256(prev_hash + payload + metadata)
    metadata        TEXT,                       -- JSON com contexto adicional
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_if ON audit_log(if_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_chain ON audit_log(id);

-- ============================================================
-- RADAR ALERTAS (mudanças de leiaute detectadas)
-- ============================================================
CREATE TABLE IF NOT EXISTS radar_alerts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cadoc_code      TEXT NOT NULL,
    alert_type      TEXT NOT NULL,              -- 'leiaute_changed', 'new_critica', 'normativo_published'
    severity        TEXT NOT NULL DEFAULT 'info', -- info, warn, critical
    title           TEXT NOT NULL,
    description     TEXT,
    source_url      TEXT,
    detected_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at     DATETIME,
    UNIQUE(cadoc_code, alert_type, detected_at, title)
);

CREATE INDEX IF NOT EXISTS idx_radar_cadoc ON radar_alerts(cadoc_code, detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_radar_unresolved ON radar_alerts(resolved_at) WHERE resolved_at IS NULL;