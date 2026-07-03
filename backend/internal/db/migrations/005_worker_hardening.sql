-- Migration 005 — Worker hardening (Sprint 6 / W1 + W2)
--
-- Adiciona colunas necessárias para retry com backoff e lease sweeper.
--
-- W1 (retry/backoff):
--   - attempts: número de tentativas já feitas (0 = primeira)
--   - next_retry_at: timestamp em que worker deve reprocessar (NULL = ASAP)
--   - dead_letter: status terminal quando attempts >= max_attempts
--
-- W2 (lease timeout):
--   - processing_started_at: quando worker começou a processar
--   - Sweeper a cada 1min resseta para 'pending' se passou lease timeout (5min)
--
-- max_attempts é constante no código do worker (5), não coluna —
-- mudar limite = recompilar worker (simples, evita config-drift).

ALTER TABLE envios ADD COLUMN attempts INT NOT NULL DEFAULT 0;
ALTER TABLE envios ADD COLUMN next_retry_at DATETIME;
ALTER TABLE envios ADD COLUMN processing_started_at DATETIME;

CREATE INDEX IF NOT EXISTS idx_envios_pending_retry
    ON envios(status, next_retry_at)
    WHERE status = 'pending';
