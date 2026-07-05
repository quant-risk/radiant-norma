// Package insights — service de acknowledgment de recommendations.
//
// Sprint 12 (opcional) — feature pequena: permite ao usuário marcar
// recomendações como "fechadas" sem removê-las. Audit trail via
// acknowledged_at + acknowledged_by.
//
// Semelhante ao ruleprefs mas pra recommendations (que são computadas,
// não regras hardcoded).

package insights

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrRecommendationNotAcknowledged é retornado por Unacknowledge quando
// a recommendation não foi marcada.
var ErrRecommendationNotAcknowledged = errors.New("recommendation not acknowledged")

// AcknowledgedRecommendation representa 1 acknowledgment.
type AcknowledgedRecommendation struct {
	IFID           string    `json:"if_id"`
	RecID          string    `json:"rec_id"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
	AcknowledgedBy string    `json:"acknowledged_by"`
}

// Acknowledgments gerencia acknowledgments de recommendations por IF.
type Acknowledgments struct {
	db *sql.DB
}

// NewAcknowledgments cria service com DB injetado.
func NewAcknowledgments(db *sql.DB) *Acknowledgments {
	return &Acknowledgments{db: db}
}

// Acknowledge marca recommendation como vista. Idempotente.
func (a *Acknowledgments) Acknowledge(ctx context.Context, ifID, recID, actor string) (AcknowledgedRecommendation, error) {
	now := time.Now().UTC()
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO acknowledged_recommendations (if_id, rec_id, acknowledged_at, acknowledged_by)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(if_id, rec_id) DO UPDATE SET
		   acknowledged_at = excluded.acknowledged_at,
		   acknowledged_by = excluded.acknowledged_by`,
		ifID, recID, now, actor,
	)
	if err != nil {
		return AcknowledgedRecommendation{}, fmt.Errorf("acknowledge: %w", err)
	}
	return AcknowledgedRecommendation{
		IFID:           ifID,
		RecID:          recID,
		AcknowledgedAt: now,
		AcknowledgedBy: actor,
	}, nil
}

// Unacknowledge remove acknowledgment. Retorna ErrRecommendationNotAcknowledged
// se não existe.
func (a *Acknowledgments) Unacknowledge(ctx context.Context, ifID, recID string) error {
	res, err := a.db.ExecContext(ctx,
		`DELETE FROM acknowledged_recommendations WHERE if_id = ? AND rec_id = ?`,
		ifID, recID,
	)
	if err != nil {
		return fmt.Errorf("unacknowledge: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrRecommendationNotAcknowledged
	}
	return nil
}

// IsAcknowledged checa se 1 recommendation específica foi acknowledge por 1 IF.
func (a *Acknowledgments) IsAcknowledged(ctx context.Context, ifID, recID string) (bool, error) {
	var exists int
	err := a.db.QueryRowContext(ctx,
		`SELECT 1 FROM acknowledged_recommendations WHERE if_id = ? AND rec_id = ?`,
		ifID, recID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query acknowledged: %w", err)
	}
	return true, nil
}

// ListAcknowledged retorna set de rec_ids acknowledged por 1 IF.
// Útil pra handler de listagem cruzar com recommendations computadas.
func (a *Acknowledgments) ListAcknowledged(ctx context.Context, ifID string) (map[string]time.Time, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT rec_id, acknowledged_at FROM acknowledged_recommendations WHERE if_id = ?`,
		ifID,
	)
	if err != nil {
		return nil, fmt.Errorf("list acknowledged: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]time.Time)
	for rows.Next() {
		var recID string
		var at time.Time
		if err := rows.Scan(&recID, &at); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out[recID] = at
	}
	return out, rows.Err()
}