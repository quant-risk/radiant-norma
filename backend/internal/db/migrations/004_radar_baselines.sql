-- Migration 004 — Radar Baselines (Sprint 6 F3)
--
-- Sprint 6 v1.5.0 (F3): tabela dedicada radar_baselines com PK composta
-- (cadoc_code, alert_type). Resolve race window entre concurrent recordBaseline
-- + lastKnownHash que existia em v1.4.x (ambos UPDATE/INSERT em radar_alerts
-- sem constraint UNIQUE).
--
-- Benefícios vs usar radar_alerts com _baseline_* (v1.4.x):
--   - Atomicidade: INSERT ... ON CONFLICT (cadoc_code, alert_type) DO UPDATE
--     em uma única operação atômica (SQLite 3.24+)
--   - Sem race window: UNIQUE constraint garante que 2 goroutines
--     gravando o mesmo baseline são serializadas pelo DB
--   - Tabela dedicada: queries de baseline não competem com queries de
--     alertas reais (last_known_hash vs select_alerts)
--   - Sem poluição: ListAlerts filtra WHERE alert_type NOT LIKE '_baseline_%'
--     hoje; com tabela nova, ListAlerts não precisa desse filtro
--
-- NOTA: dados antigos (radar_alerts WHERE alert_type LIKE '_baseline_%')
-- SÃO migrados. Em produção real isso seria problemático; em dev é OK
-- porque dados existentes são ephemeral.

CREATE TABLE IF NOT EXISTS radar_baselines (
    cadoc_code    TEXT NOT NULL,
    alert_type    TEXT NOT NULL,
    hash          TEXT NOT NULL,
    source_url    TEXT,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (cadoc_code, alert_type)
);

-- Migra baselines existentes de radar_alerts para radar_baselines.
-- ON CONFLICT IGNORE: se já houver dado na nova tabela, mantém (idempotente).
INSERT OR IGNORE INTO radar_baselines (cadoc_code, alert_type, hash, source_url, updated_at)
SELECT cadoc_code, alert_type, description, source_url, detected_at
FROM radar_alerts
WHERE alert_type LIKE '_baseline_%';

-- Opcional: limpar baselines migrados de radar_alerts após migração.
-- Comentado para preservar audit trail até validação final.
-- DELETE FROM radar_alerts WHERE alert_type LIKE '_baseline_%';
