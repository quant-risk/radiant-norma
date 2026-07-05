// Package worker implementa o processador assíncrono de envios STA.
//
// Por design: cmd/api recebe requests HTTP e enfileira envios. cmd/worker
// processa a fila: valida → submete STA → atualiza status. Isso desacopla
// latência HTTP da latência do BACEN (que pode levar minutos).
//
// Em Sprint 4 (v1.3.0) é stub: processa envios pending do DB chamando o
// STA client + audit. Em Sprint 5+ usa fila real (asynq/machinery).
//
// Sprint 6 (v1.5.0) — hardening:
//   - W1: retry com exponential backoff (5 tentativas: 1m, 5m, 30m, 2h, 12h)
//   - W2: lease timeout sweeper (envios em processing > 5min → pending)
//   - Dead letter: status terminal quando attempts >= MaxAttempts
//
// Race safety: usa UPDATE condicional (claim atômico via subselect). Em
// produção com múltiplos workers, O DB serializa via _txlock=immediate
// (SQLite) ou SELECT FOR UPDATE (Postgres).
package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/loggerutil"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

// Constantes de hardening (Sprint 6 v1.5.0 / W1+W2).
const (
	// MaxAttempts limite antes de enviar pra dead letter.
	// 5 tentativas com backoff total ~15h36m.
	MaxAttempts = 5

	// LeaseTimeout tempo máximo que um envio pode ficar em 'processing'.
	// Acima disso, sweeper resseta para 'pending' (assume worker crash).
	LeaseTimeout = 5 * time.Minute

	// LeaseSweepInterval frequência do sweeper.
	LeaseSweepInterval = 1 * time.Minute

	// StatusDeadLetter status terminal (não tenta mais).
	StatusDeadLetter = "dead_letter"
)

// backoffDurations é o array de intervalos de retry (W1).
// Índice = número de tentativas JÁ feitas (1-indexed pra clareza).
//
// Tentativa 1 → 1min, 2 → 5min, 3 → 30min, 4 → 2h, 5 → 12h
// Total worst-case: ~15h36m pra dar dead letter.
var backoffDurations = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

// ComputeBackoff retorna o intervalo de retry baseado em attempts já feitas.
//
// attempts = 0 → 1min (primeira falha → primeira retry)
// attempts = 4 → 12h (última antes de dead letter)
//
// Se attempts >= len(backoffDurations), retorna 0 (não retry, dead letter).
//
// Exportado para testes em worker_test.go.
func ComputeBackoff(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	if attempts >= len(backoffDurations) {
		return 0
	}
	return backoffDurations[attempts]
}

// EnvioRow representa um envio do DB (campos mínimos pro worker).
//
// Reutilizado de v1.4.x mas agora com campos novos (attempts, next_retry_at).
type EnvioRow struct {
	ID                  string
	IFID                string
	CadocCode           string
	DataBase            string
	XMLHash             string
	ZipHash             string
	Status              string
	XMLContent          string
	Attempts            int
	NextRetryAt         sql.NullTime
	ProcessingStartedAt sql.NullTime
}

// ProcessBatch processa até N envios com status pending (devido retry).
//
// Mudanças Sprint 6 v1.5.0:
//   - Cada envio tem processing_started_at setado ao claim
//   - Falha de STA: incrementa attempts, se < MaxAttempts agenda retry com backoff,
//     senão marca dead_letter
//   - Sucesso: marca accepted/rejected normalmente
//
// Concorrência: usa claim atômico via UPDATE condicional. Sem isso,
// dois workers rodando simultaneamente poderiam pegar o mesmo envio.
func ProcessBatch(
	ctx context.Context,
	d *sql.DB,
	auditSvc *audit.Service,
	auditLog *auditlog.Logger,
	staClient sta.Client,
	batch int,
	logger *slog.Logger,
) (int, error) {
	processed := 0
	for i := 0; i < batch; i++ {
		e, ok, err := claimEnvio(ctx, d)
		if err != nil {
			// Validação 16 (F16.1): sanitizar err.
			logger.Error("claim envio failed", "err", loggerutil.SafeError(err))
			return processed, err
		}
		if !ok {
			break // sem mais envios pendentes
		}

		// Processa o envio (sucesso ou falha).
		if err := processEnvio(ctx, d, auditSvc, auditLog, staClient, e, logger); err != nil {
			// Validação 16 (F16.1): sanitizar err (process_envio id).
			logger.Error("process envio failed", "envio_id", e.ID, "err", loggerutil.SafeError(err))
			// Continua para o próximo — não para o batch inteiro.
			continue
		}
		processed++
	}
	return processed, nil
}

// claimEnvio pega atomicamente 1 envio pendente, marca como processing
// com processing_started_at.
//
// Filtra envios com next_retry_at <= NOW() (retry só ocorre quando due).
//
// Retorna ok=false quando não há mais envios (sql.ErrNoRows).
func claimEnvio(ctx context.Context, d *sql.DB) (EnvioRow, bool, error) {
	var e EnvioRow
	err := d.QueryRowContext(ctx, `
		UPDATE envios SET status='processing', processing_started_at = CURRENT_TIMESTAMP
		WHERE id = (
			SELECT id FROM envios
			WHERE status = 'pending'
			  AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
			ORDER BY COALESCE(next_retry_at, created_at) ASC
			LIMIT 1
		)
		RETURNING id, if_id, cadoc_code, data_base, xml_hash, zip_hash, status,
		          xml_content, attempts, next_retry_at, processing_started_at
	`).Scan(&e.ID, &e.IFID, &e.CadocCode, &e.DataBase,
		&e.XMLHash, &e.ZipHash, &e.Status, &e.XMLContent,
		&e.Attempts, &e.NextRetryAt, &e.ProcessingStartedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return EnvioRow{}, false, nil
	}
	if err != nil {
		return EnvioRow{}, false, err
	}
	return e, true, nil
}

// processEnvio processa 1 envio: submete via STA, atualiza status,
// gerencia retry/dead-letter.
//
// Sucesso: status = accepted/rejected/sent.
// Falha + attempts < MaxAttempts: status = pending, next_retry_at = NOW()+backoff.
// Falha + attempts >= MaxAttempts: status = dead_letter.
func processEnvio(
	ctx context.Context,
	d *sql.DB,
	auditSvc *audit.Service,
	auditLog *auditlog.Logger,
	staClient sta.Client,
	e EnvioRow,
	logger *slog.Logger,
) error {
	sub := &sta.Submission{
		CadocCode: e.CadocCode,
		DataBase:  e.DataBase,
		XML:       e.XMLContent,
		CNPJ:      e.IFID,
	}

	result, err := staClient.Submit(ctx, sub)
	if err != nil {
		// Falha: marca para retry ou dead letter.
		newAttempts := e.Attempts + 1
		backoff := ComputeBackoff(e.Attempts) // attempts ANTES do increment
		newStatus := "pending"
		var deadLetter bool
		if backoff == 0 || newAttempts >= MaxAttempts {
			newStatus = StatusDeadLetter
			deadLetter = true
		}

		// Construir SQL do update.
		// IMPORTANTE: usamos DATETIME(CURRENT_TIMESTAMP, '+N seconds') Go-side
		// NÃO funciona — driver modernc/sqlite formata time.Time como RFC3339
		// (com 'T' e timezone), CURRENT_TIMESTAMP usa formato SQLite ('YYYY-MM-DD HH:MM:SS').
		// Comparação textual resultaria em "2026-07-03T21:00:00Z" < "2026-07-03 21:00:00"
		// (T < espaço em ASCII). Solução: aritmética datetime dentro do próprio SQLite.
		// Validação 18 (F18.13) — Sprint 13 (v3.5.2) [HIGH C-N8/S13.4]:
		// sanitizar err.Error() ANTES de gravar em envios.error_message.
		// Coluna persiste em DB (LGPD/SOC2). Sem SafeError, DSN Postgres,
		// SQL fragments, tokens BACEN podem vazar para disco, backups,
		// e data lake de auditoria.
		safeErr := loggerutil.SafeError(err)
		var uerr error
		if deadLetter {
			uerr = execRetryOrDeadLetter(d, ctx, e.ID, newStatus, newAttempts, "", safeErr)
		} else {
			uerr = execRetryOrDeadLetter(d, ctx, e.ID, newStatus, newAttempts,
				fmt.Sprintf("+%d seconds", int(backoff.Seconds())), safeErr)
		}
		if uerr != nil {
			return fmt.Errorf("mark envio retry/dead-letter: %w", uerr)
		}

		// Audit
		action := "envio.retry.scheduled"
		if deadLetter {
			action = "envio.dead_letter"
		}
		_, _ = auditLog.Log(e.IFID, "worker", action, e.CadocCode, []byte(e.ID), map[string]any{
			"envio_id": e.ID,
			"attempts": newAttempts,
			"max":      MaxAttempts,
			// Validação 18 (F18.13): sanitizar err.Error() antes do AuditLog.
			// AuditLog persiste em disco (LGPD/SOC2) — vetor persistente.
			"err": loggerutil.SafeError(err),
		})

		logger.Error("envio submission failed",
			"envio_id", e.ID,
			"cadoc", e.CadocCode,
			"attempts", newAttempts,
			"status", newStatus,
			// Validação 16 (F16.1): sanitizar err.
			"err", loggerutil.SafeError(err),
		)
		return err
	}

	// Sucesso: atualiza status final.
	newStatus := "sent"
	if result.Accepted {
		newStatus = "accepted"
	} else {
		newStatus = "rejected"
	}

	_, err = d.ExecContext(ctx, `
		UPDATE envios
		SET status=?, protocol_sta=?, sent_at=CURRENT_TIMESTAMP,
		    confirmed_at=CASE WHEN ? IN ('accepted','rejected') THEN CURRENT_TIMESTAMP ELSE NULL END,
		    error_code=?, error_message=?,
		    next_retry_at = NULL,
		    processing_started_at = NULL
		WHERE id=?
	`, newStatus, result.ProtocolSTA, newStatus,
		sqlNullString(result, true), sqlNullString(result, false), e.ID)
	if err != nil {
		return fmt.Errorf("update envio success: %w", err)
	}

	_, _ = auditLog.Log(e.IFID, "worker", "envio.processed", e.CadocCode, []byte(e.ID), map[string]any{
		"envio_id": e.ID,
		"protocol": result.ProtocolSTA,
		"status":   newStatus,
	})

	logger.Info("envio processed",
		"envio_id", e.ID,
		"cadoc", e.CadocCode,
		"status", newStatus,
		"protocol", result.ProtocolSTA,
	)
	return nil
}

// RunLeaseSweeper resseta envios stuck em 'processing' há mais de
// LeaseTimeout. Assume que o worker crashed.
//
// Retorna quantos envios foram resetados.
//
// Idempotente: rodar várias vezes não causa efeitos colaterais.
func RunLeaseSweeper(ctx context.Context, d *sql.DB, logger *slog.Logger) (int, error) {
	res, err := d.ExecContext(ctx, `
		UPDATE envios
		SET status = 'pending',
		    processing_started_at = NULL,
		    error_message = 'lease timeout: worker crash assumed'
		WHERE status = 'processing'
		  AND processing_started_at IS NOT NULL
		  AND processing_started_at < DATETIME(CURRENT_TIMESTAMP, ?)
	`, fmt.Sprintf("-%d seconds", int(LeaseTimeout.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("lease sweeper update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		logger.Warn("lease sweeper reset stuck envios",
			"count", n,
			"lease_timeout", LeaseTimeout.String())
	}
	return int(n), nil
}

// execRetryOrDeadLetter centraliza o SQL do UPDATE pós-falha.
//
// Se retryOffsetSeconds for vazio, marca dead_letter (sem retry).
// Senão, agenda retry usando DATETIME(CURRENT_TIMESTAMP, '+N seconds')
// do SQLite (consistência textual entre next_retry_at e CURRENT_TIMESTAMP
// no claimEnvio).
func execRetryOrDeadLetter(d *sql.DB, ctx context.Context, id string, status string, attempts int, retryOffsetSeconds string, errMsg string) error {
	var nextRetryExpr string
	if retryOffsetSeconds == "" {
		nextRetryExpr = "NULL"
	} else {
		nextRetryExpr = fmt.Sprintf("DATETIME(CURRENT_TIMESTAMP, '%s')", retryOffsetSeconds)
	}
	query := fmt.Sprintf(`
		UPDATE envios
		SET status = ?,
		    attempts = ?,
		    next_retry_at = %s,
		    processing_started_at = NULL,
		    error_message = ?,
		    error_code = 'STA_SUBMIT_FAILED'
		WHERE id = ?
	`, nextRetryExpr)
	_, err := d.ExecContext(ctx, query, status, attempts, errMsg, id)
	return err
}

// sqlNullString extrai string opcional de um Result de STA submission.
func sqlNullString(r *sta.Result, isCode bool) sql.NullString {
	if r == nil || r.Rejection == nil {
		return sql.NullString{}
	}
	if isCode {
		return sql.NullString{String: r.Rejection.Code, Valid: true}
	}
	return sql.NullString{String: r.Rejection.Message, Valid: true}
}
