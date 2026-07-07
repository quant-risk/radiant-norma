// Package multiregion implementa replicação entre regiões BR-SP1 e BR-SP2.
//
// Permite que tenants sejam atribuídos a uma região e que dados sejam
// replicados entre as regiões para disaster recovery e baixa latência.
package multiregion

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service gerencia replicação entre regiões.
type Service struct {
	db   *sql.DB
	self Region // "br-sp1" ou "br-sp2"
	peer Region
}

// Region representa uma região geográfica.
type Region string

const (
	RegionBRSP1 Region = "br-sp1"
	RegionBRSP2 Region = "br-sp2"
)

// NewService cria um multiregion service.
func NewService(db *sql.DB, self Region) *Service {
	if self != RegionBRSP1 && self != RegionBRSP2 {
		self = RegionBRSP1
	}
	peer := RegionBRSP2
	if self == RegionBRSP2 {
		peer = RegionBRSP1
	}
	return &Service{db: db, self: self, peer: peer}
}

// AssignRegion atribui uma região a um tenant.
func (s *Service) AssignRegion(ctx context.Context, ifID string, region Region) error {
	if region != RegionBRSP1 && region != RegionBRSP2 {
		return fmt.Errorf("região inválida: %s", region)
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE ifs SET region = ? WHERE id = ?", string(region), ifID)
	return err
}

// GetRegion retorna a região de um tenant.
func (s *Service) GetRegion(ctx context.Context, ifID string) (Region, error) {
	var r Region
	err := s.db.QueryRowContext(ctx,
		"SELECT region FROM ifs WHERE id = ?", ifID).Scan(&r)
	return r, err
}

// IsLocal returns true se a região é local (não precisa replicar).
func (s *Service) IsLocal(region Region) bool {
	return region == s.self
}

// ShouldReplicate returns true se um evento deve ser replicado para a outra região.
func (s *Service) ShouldReplicate(region Region) bool {
	return region != s.self
}

// EmitEvent registra um evento de replicação.
func newID() string { return uuid.New().String() }

func (s *Service) EmitEvent(ctx context.Context, eventType, entityType, entityID string, payload string) error {
	id := newID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO region_events (id, region_from, region_to, event_type, entity_type, entity_id, payload, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')
	`, id, string(s.self), string(s.peer), eventType, entityType, entityID, payload)
	return err
}

// GetStatus retorna o status de replicação da região.
func (s *Service) GetStatus(ctx context.Context, region Region) (*ReplicationStatus, error) {
	var rs ReplicationStatus
	err := s.db.QueryRowContext(ctx,
		`SELECT region, last_sync_at, lag_seconds, status, updated_at
		 FROM replication_status WHERE region = ?`, string(region),
	).Scan(&rs.Region, &rs.LastSyncAt, &rs.LagSeconds, &rs.Status, &rs.UpdatedAt)
	if err == sql.ErrNoRows {
		return &ReplicationStatus{Region: region, Status: "unknown"}, nil
	}
	return &rs, err
}

// MarkSynced marca um evento de replicação como sincronizado.
func (s *Service) MarkSynced(ctx context.Context, eventID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE region_events SET status='synced', synced_at=CURRENT_TIMESTAMP WHERE id=?`,
		eventID)
	return err
}

// MarkFailed marca um evento de replicação como falhado.
func (s *Service) MarkFailed(ctx context.Context, eventID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE region_events SET status='failed' WHERE id=?`, eventID)
	return err
}

// PendingEvents retorna eventos pendentes de replicação.
func (s *Service) PendingEvents(ctx context.Context, limit int) ([]ReplicationEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, region_from, region_to, event_type, entity_type, entity_id, payload, created_at
		FROM region_events
		WHERE region_from = ? AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`, string(s.self), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []ReplicationEvent
	for rows.Next() {
		var e ReplicationEvent
		var payload sql.NullString
		if err := rows.Scan(&e.ID, &e.RegionFrom, &e.RegionTo, &e.EventType,
			&e.EntityType, &e.EntityID, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Payload = payload.String
		events = append(events, e)
	}
	return events, rows.Err()
}

// UpdateReplicationStatus atualiza o status de replicação da região.
func (s *Service) UpdateReplicationStatus(ctx context.Context, region Region, status string, lagSeconds int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO replication_status (region, last_sync_at, lag_seconds, status, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(region) DO UPDATE SET
			last_sync_at = CURRENT_TIMESTAMP,
			lag_seconds = excluded.lag_seconds,
			status = excluded.status,
			updated_at = CURRENT_TIMESTAMP
	`, string(region), lagSeconds, status)
	return err
}

// SetIFRegion é um alias para AssignRegion.
func SetIFRegion(ctx context.Context, db *sql.DB, ifID string, region Region) error {
	s := NewService(db, region)
	return s.AssignRegion(ctx, ifID, region)
}

// ============================================================
// Types
// ============================================================

// ReplicationStatus representa o status de replicação de uma região.
type ReplicationStatus struct {
	Region     Region    `json:"region"`
	LastSyncAt time.Time `json:"last_sync_at,omitempty"`
	LagSeconds int       `json:"lag_seconds"`
	Status     string    `json:"status"` // healthy|degraded|offline|unknown
	UpdatedAt  time.Time `json:"updated_at"`
}

// ReplicationEvent representa um evento pendente de replicação.
type ReplicationEvent struct {
	ID         string    `json:"id"`
	RegionFrom Region    `json:"region_from"`
	RegionTo   Region    `json:"region_to"`
	EventType  string    `json:"event_type"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Payload    string    `json:"payload,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ValidRegion checks if a string is a valid region.
func ValidRegion(r string) bool {
	r = strings.ToLower(r)
	return r == string(RegionBRSP1) || r == string(RegionBRSP2)
}

// ParseRegion parses a region string.
func ParseRegion(r string) (Region, error) {
	r = strings.ToLower(strings.TrimSpace(r))
	if r == "br-sp1" || r == "brsp1" {
		return RegionBRSP1, nil
	}
	if r == "br-sp2" || r == "brsp2" {
		return RegionBRSP2, nil
	}
	return "", fmt.Errorf("região inválida: %s", r)
}
