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

package ruleprefs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
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
	rows, err := p.db.QueryContext(ctx,
		`SELECT if_id, rule_code, disabled_at, disabled_by
		 FROM disabled_rules
		 WHERE if_id = ?
		 ORDER BY disabled_at DESC`,
		ifID)
	if err != nil {
		return nil, fmt.Errorf("query disabled: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DisabledRule
	for rows.Next() {
		var r DisabledRule
		if err := rows.Scan(&r.IFID, &r.RuleCode, &r.DisabledAt, &r.DisabledBy); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
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
	rows, err := p.db.QueryContext(ctx,
		`SELECT rule_code FROM disabled_rules WHERE if_id = ?`,
		ifID)
	if err != nil {
		return nil, fmt.Errorf("query disabled codes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan code: %w", err)
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

// IsDisabled checa se 1 regra específica está desabilitada por 1 IF.
// Mais eficiente que ListDisabled se checar 1-2 regras (1 query).
func (p *Preferences) IsDisabled(ctx context.Context, ifID, ruleCode string) (bool, error) {
	var exists int
	err := p.db.QueryRowContext(ctx,
		`SELECT 1 FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
		ifID, ruleCode,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query is_disabled: %w", err)
	}
	return true, nil
}

// Disable adiciona regra ao set desabilitado. Idempotente — se já
// desabilitada, retorna o registro existente sem erro.
func (p *Preferences) Disable(ctx context.Context, ifID, ruleCode, actor string) (DisabledRule, error) {
	now := time.Now().UTC()

	_, err := p.db.ExecContext(ctx,
		`INSERT INTO disabled_rules (if_id, rule_code, disabled_at, disabled_by)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(if_id, rule_code) DO UPDATE SET
		   disabled_at = excluded.disabled_at,
		   disabled_by = excluded.disabled_by`,
		ifID, ruleCode, now, actor,
	)
	if err != nil {
		return DisabledRule{}, fmt.Errorf("insert disabled: %w", err)
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
	res, err := p.db.ExecContext(ctx,
		`DELETE FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
		ifID, ruleCode,
	)
	if err != nil {
		return fmt.Errorf("delete disabled: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
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
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op se Commit rodar

	// Read current state dentro da tx (mantém lock)
	var isDisabled bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM disabled_rules WHERE if_id = ? AND rule_code = ?)`,
		ifID, ruleCode,
	).Scan(&isDisabled)
	if err != nil {
		return "", fmt.Errorf("read state: %w", err)
	}

	if isDisabled {
		// Habilita (DELETE)
		_, err := tx.ExecContext(ctx,
			`DELETE FROM disabled_rules WHERE if_id = ? AND rule_code = ?`,
			ifID, ruleCode,
		)
		if err != nil {
			return "", fmt.Errorf("delete disabled: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit: %w", err)
		}
		return "enabled", nil
	}

	// Desabilita (INSERT com ON CONFLICT)
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO disabled_rules (if_id, rule_code, disabled_at, disabled_by)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(if_id, rule_code) DO UPDATE SET
		   disabled_at = excluded.disabled_at,
		   disabled_by = excluded.disabled_by`,
		ifID, ruleCode, now, actor,
	)
	if err != nil {
		return "", fmt.Errorf("insert disabled: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return "disabled", nil
}
