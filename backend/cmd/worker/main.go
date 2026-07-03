// Command worker é o processador assíncrono de envios STA.
//
// Por design: cmd/api recebe requests HTTP e enfileira envios. cmd/worker
// processa a fila: valida → submete STA → atualiza status. Isso desacopla
// latência HTTP da latência do BACEN (que pode levar minutos).
//
// Em Sprint 4 (v1.3.0) é stub: processa envios pending do DB chamando o
// STA client + audit. Em Sprint 5 vai usar fila real (asynq/machinery).
package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/auditlog"
	"github.com/fortvna/radiant-norma/backend/internal/db"
	"github.com/fortvna/radiant-norma/backend/internal/sta"
)

type envioRow struct {
	ID         string
	IFID       string
	CadocCode  string
	DataBase   string
	XMLHash    string
	ZipHash    string
	Status     string
	XMLContent string
}

func main() {
	var (
		dbPath   = flag.String("db", "radiant.db", "path to SQLite database")
		interval = flag.Duration("interval", 30*time.Second, "tick interval")
		batch    = flag.Int("batch", 10, "max envios per tick")
		once     = flag.Bool("once", false, "processa 1 batch e sai (útil pra teste)")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Open DB
	d, err := db.Open(*dbPath)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	// Migrations (worker pode rodar standalone antes da API ter criado schema)
	if err := db.Migrate(d); err != nil {
		logger.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	// Init services
	auditSvc := audit.New(d)
	auditLog := auditlog.New(d)
	staClient := sta.NewStubClient()

	logger.Info("worker started",
		"db", *dbPath,
		"interval", interval.String(),
		"batch", *batch,
	)

	// Graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Processa imediatamente no boot
	n, _ := processBatch(ctx, d, auditSvc, auditLog, staClient, *batch, logger)
	logger.Info("initial batch done", "processed", n)

	if *once {
		logger.Info("once mode: exiting")
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped", "reason", ctx.Err())
			return
		case <-ticker.C:
			n, err := processBatch(ctx, d, auditSvc, auditLog, staClient, *batch, logger)
			if err != nil {
				logger.Error("batch failed", "err", err)
				continue
			}
			if n > 0 {
				logger.Info("batch processed", "count", n)
			}
		}
	}
}

// processBatch processa até N envios com status pending.
//
// Concorrência: usa CLAIM atômico via UPDATE condicional. Sem isso,
// dois workers rodando simultaneamente poderiam pegar o mesmo envio.
//
// Pipeline por envio:
//  1. Claim: UPDATE ... SET status='processing' WHERE status='pending' RETURNING
//  2. Submete via STA client (stub: gera protocolo fake)
//  3. Atualiza status (processing → sent → accepted/rejected)
//
// Retorna quantos processou com sucesso.
func processBatch(
	ctx context.Context,
	d *sql.DB,
	auditSvc *audit.Service,
	auditLog *auditlog.Logger,
	staClient sta.Client,
	batch int,
	logger *slog.Logger,
) (int, error) {
	// Estratégia: claim um envio por vez dentro de um loop.
	// Em produção: usar SKIP LOCKED (Postgres) ou claim em batch.
	processed := 0
	for i := 0; i < batch; i++ {
		// 1. Claim atômico: 1 envio vira 'processing' (impede outro worker pegar)
		var e envioRow
		err := d.QueryRowContext(ctx, `
			UPDATE envios SET status='processing'
			WHERE id = (
				SELECT id FROM envios
				WHERE status IN ('pending', 'error')
				ORDER BY created_at ASC
				LIMIT 1
			)
			RETURNING id, if_id, cadoc_code, data_base, xml_hash, zip_hash, status, xml_content
		`).Scan(&e.ID, &e.IFID, &e.CadocCode, &e.DataBase,
			&e.XMLHash, &e.ZipHash, &e.Status, &e.XMLContent)
		if err == sql.ErrNoRows {
			break // sem mais envios
		}
		if err != nil {
			logger.Error("claim envio failed", "err", err)
			return processed, err
		}

		// 2. Submete via STA
		sub := &sta.Submission{
			CadocCode: e.CadocCode,
			DataBase:  e.DataBase,
			XML:       e.XMLContent,
			CNPJ:      e.IFID,
		}

		result, err := staClient.Submit(ctx, sub)
		if err != nil {
			logger.Error("sta submit failed", "envio_id", e.ID, "err", err)
			// Se este UPDATE falhar, envio fica em 'pending' e o loop claima
			// de novo — loop infinito de retries. Logamos pra investigar.
			if _, uerr := d.ExecContext(ctx,
				"UPDATE envios SET status='error', error_message=? WHERE id=?",
				err.Error(), e.ID); uerr != nil {
				logger.Error("failed to mark envio as error — possible loop",
					"envio_id", e.ID, "err", uerr)
			}
			continue
		}

		// 3. Atualiza status
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
			    error_code=?, error_message=?
			WHERE id=?
		`, newStatus, result.ProtocolSTA, newStatus,
			sqlNullString(result, true), sqlNullString(result, false), e.ID)
		if err != nil {
			logger.Error("update envio failed", "envio_id", e.ID, "err", err)
			continue
		}

		// 4. Audit log
		_, _ = auditLog.Log(e.IFID, "worker", "envio.processed", e.CadocCode, []byte(e.ID),
			map[string]any{
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
		processed++
	}

	return processed, nil
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
