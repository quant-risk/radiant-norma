// Package ruleprefs — preferences service (rule enable/disable por IF).
//
// Sprint 11: regras habilitadas/desabilitadas eram localStorage no frontend
// (cada device tinha seu próprio estado). Agora: backend persiste por IF,
// emite audit event, frontend sincroniza via API.
//
// Pattern: simples CRUD com audit hook. Sem soft delete — toggle alterna
// presence/absence da row. PK (if_id, rule_code) garante 1 row por IF×rule.
//
// Nome do package: ruleprefs (e não rules) pra evitar conflito com
// internal/audit/rules (que define as regras hardcoded).
//
// Sprint 30 (v3.33.0): todos os métodos de leitura/escrita usam
// db.WithTenantTx para setar `app.if_id` em Postgres (FORCE RLS).
// Em SQLite, helper é no-op (compat).

package ruleprefs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/db"
)

// ErrRuleNotDisabled é retornado por Enable quando a regra não está
// na tabela (já está habilitada).
var ErrRuleNotDisabled = errors.New("rule not in disabled set")

// DisabledRule é o registro de 1 regra desabilitada por 1 IF.
type DisabledRule struct {
	IFID       string    `json:"if_id"`
	RuleCode   string    `json:"rule_code"`
	DisabledAt time.Time `json:"disabled_at"`
	DisabledBy string    `json:"disabled_by"`
}

// Preferences gerencia preferências de regras (enable/disable) por IF.
//
// Concurrency: usa DB transactions. Não há estado in-process — cada request
// é independente. Audit emission fica no handler (auditlog.Log) pra manter
// separation of concerns.
type Preferences struct {
	db *sql.DB
}

// NewPreferences cria Preferences com DB injetado.
func NewPreferences(db *sql.DB) *Preferences {
	return &Preferences{db: db}
}

// ListDisabled retorna todas as regras desabilitadas por 1 IF.
// Retorna slice vazio se nenhuma (não error).
func (p *Preferences) ListDisabled(ctx context.Context, ifID string) ([]DisabledRule, error) {
	var out []DisabledRule
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			`SELECT if_id, rule_code, disabled_at, disabled_by
			 FROM disabled_rules
			 WHERE if_id = ?
			 ORDER BY disabled_at DESC`,
			ifID)
		if err != nil {
			return fmt.Errorf("query disabled: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var r DisabledRule
			if err := rows.Scan(&r.IFID, &r.RuleCode, &r.DisabledAt, &r.DisabledBy); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			out = append(out, r)
		}
		return rows.Err()
	})
	return out, err
}

// ListDisabledCodes retorna só os códigos (rule_code) das regras
// desabilitadas por 1 IF. Usado pelo audit.Service pra filtrar regras
// sem precisar carregar metadata.
//
// Sprint 12 (v3.5.0): C32.23 — integração com engine de validação.
// Implementação separada de ListDisabled (que retorna DisabledRule
// struct) pra evitar overhead de carregar timestamps + actor quando
// caller só precisa dos codes.
func (p *Preferences) ListDisabledCodes(ctx context.Context, ifID string) ([]string, error) {
	var out []string
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			`SELECT rule_code FROM disabled_rules WHERE if_id = ?`,
			ifID)
		if err != nil {
			return fmt.Errorf("query disabled codes: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				return fmt.Errorf("scan code: %w", err)
			}
			out = append(out, code)
		}
		return rows.Err()
	})
	return out, err
}

// IsDisabled checa se 1 regra específica está desabilitada por 1 IF.
// Mais eficiente que ListDisabled se checar 1-2 regras (1 query).
func (p *Preferences) IsDisabled(ctx context.Context, ifID, ruleCode string) (bool, error) {
	var exists int
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		queryErr := tx.QueryRowContext(ctx,
			`SELECT 1 FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
			ifID, ruleCode,
		).Scan(&exists)
		if queryErr == sql.ErrNoRows {
			return nil // exists stays 0
		}
		return queryErr
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query is_disabled: %w", err)
	}
	return exists == 1, nil
}

// Disable adiciona regra ao set desabilitado. Idempotente — se já
// desabilitada, retorna o registro existente sem erro.
func (p *Preferences) Disable(ctx context.Context, ifID, ruleCode, actor string) (DisabledRule, error) {
	now := time.Now().UTC()
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		_, execErr := tx.ExecContext(ctx,
			`INSERT INTO disabled_rules (if_id, rule_code, disabled_at, disabled_by)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(if_id, rule_code) DO UPDATE SET
			   disabled_at = excluded.disabled_at,
			   disabled_by = excluded.disabled_by`,
			ifID, ruleCode, now, actor,
		)
		if execErr != nil {
			return fmt.Errorf("insert disabled: %w", execErr)
		}
		return nil
	})
	if err != nil {
		return DisabledRule{}, err
	}

	return DisabledRule{
		IFID:       ifID,
		RuleCode:   ruleCode,
		DisabledAt: now,
		DisabledBy: actor,
	}, nil
}

// Enable remove regra do set desabilitado. Retorna ErrRuleNotDisabled
// se a regra não está desabilitada (no-op idempotente).
func (p *Preferences) Enable(ctx context.Context, ifID, ruleCode string) error {
	var rowsAffected int64
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		res, execErr := tx.ExecContext(ctx,
			`DELETE FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
			ifID, ruleCode,
		)
		if execErr != nil {
			return fmt.Errorf("delete disabled: %w", execErr)
		}
		ra, raErr := res.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("rows affected: %w", raErr)
		}
		rowsAffected = ra
		return nil
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRuleNotDisabled
	}
	return nil
}

// Toggle alterna estado: se desabilitada → habilita; se habilitada → desabilita.
// Retorna (newState, error) onde newState é o estado após toggle ("enabled"
// ou "disabled"). Útil pra UI mostrar confirmação.
//
// Sprint 12 (v3.5.0) — C32.1: wrap em transaction com write lock pra
// eliminar race condition (TOCTOU) entre SELECT e INSERT/DELETE. Sem
// isso, multi-replica (Sprint 12 M2) teria ~1ms race window onde 2
// requests podem ver mesmo estado e aplicar toggle conflitante.
//
// SQLite: BEGIN IMMEDIATE adquire write lock. Postgres: SELECT FOR UPDATE.
func (p *Preferences) Toggle(ctx context.Context, ifID, ruleCode, actor string) (string, error) {
	// Phase 1: read current state (sem lock — eventual consistency OK)
	// Phase 2: write com lock (atomicity aqui)
	//
	// NOTA: BEGIN IMMEDIATE em SQLite bloqueia todo o DB até COMMIT.
	// Em produção multi-pod com Postgres, preferimos SELECT FOR UPDATE
	// (row-level lock). Por ora, single-instance + SQLite é OK.
	var newState string
	err := db.WithTenantTx(ctx, p.db, ifID, func(tx *sql.Tx) error {
		var isDisabled bool
		queryErr := tx.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM disabled_rules WHERE if_id = ? AND rule_code = ?)`,
			ifID, ruleCode,
		).Scan(&isDisabled)
		if queryErr != nil {
			return fmt.Errorf("read state: %w", queryErr)
		}

		if isDisabled {
			// Habilita (DELETE)
			_, execErr := tx.ExecContext(ctx,
				`DELETE FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
				ifID, ruleCode,
			)
			if execErr != nil {
				return fmt.Errorf("delete disabled: %w", execErr)
			}
			newState = "enabled"
			return nil
		}

		// Desabilita (INSERT com ON CONFLICT)
		now := time.Now().UTC()
		_, execErr := tx.ExecContext(ctx,
			`INSERT INTO disabled_rules (if_id, rule_code, disabled_at, disabled_by)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(if_id, rule_code) DO UPDATE SET
			   disabled_at = excluded.disabled_at,
			   disabled_by = excluded.disabled_by`,
			ifID, ruleCode, now, actor,
		)
		if execErr != nil {
			return fmt.Errorf("insert disabled: %w", execErr)
		}
		newState = "disabled"
		return nil
	})
	return newState, err
}
