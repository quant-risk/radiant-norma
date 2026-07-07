// Package soc2 implementa tooling SOC 2 Type II.
//
// Fornece collection contínua de evidências, monitoring de controles,
// e immutable audit trail para auditoria SOC 2 Type II.
//
// Sprint 56: SOC 2 Type I readiness (readiness report, evidence collector)
// Sprint 65: SOC 2 Type II (continuous evidence collection, control monitoring)
package soc2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TypeIICollector collects evidence for SOC 2 Type II continuously.
type TypeIICollector struct {
	db *sql.DB
}

// NewTypeIICollector creates a new Type II evidence collector.
func NewTypeIICollector(db *sql.DB) *TypeIICollector {
	return &TypeIICollector{db: db}
}

// EvidenceRecord represents a single evidence record.
type EvidenceRecord struct {
	ID           string    `json:"id"`
	ControlID    string    `json:"control_id"`
	Criterion    string    `json:"criterion"`
	EvidenceType string    `json:"evidence_type"`
	Evidence     string    `json:"evidence"`
	Result       string    `json:"result"`
	Metadata     string    `json:"metadata,omitempty"`
	CollectedAt  time.Time `json:"collected_at"`
	CollectedBy  string    `json:"collected_by"`
}

// Collect records a new evidence entry.
func (e *TypeIICollector) Collect(ctx context.Context, req CollectEvidenceRequest) error {
	id := uuid.New().String()
	metadata := ""
	if req.Metadata != nil {
		b, _ := json.Marshal(req.Metadata)
		metadata = string(b)
	}

	_, err := e.db.ExecContext(ctx, `
		INSERT INTO soc2_evidence_log
			(id, control_id, criterion, evidence_type, evidence, result, metadata, collected_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.ControlID, req.Criterion, req.EvidenceType, req.Evidence, req.Result, metadata, req.CollectedBy)
	if err != nil {
		return fmt.Errorf("collect evidence: %w", err)
	}
	return nil
}

// CollectEvidenceRequest is the request to collect evidence.
type CollectEvidenceRequest struct {
	ControlID    string         `json:"control_id"`
	Criterion    string         `json:"criterion"`
	EvidenceType string         `json:"evidence_type"` // automated_check | manual_review | system_log | document
	Evidence     string         `json:"evidence"`      // JSON string with evidence details
	Result       string         `json:"result"`        // pass | fail | warning | not_applicable
	Metadata     map[string]any `json:"metadata,omitempty"`
	CollectedBy  string         `json:"collected_by"` // system | auditor_id
}

// Query returns evidence records for a control within a time range.
func (e *TypeIICollector) Query(ctx context.Context, controlID string, from, to time.Time) ([]EvidenceRecord, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id, control_id, criterion, evidence_type, evidence, result, COALESCE(metadata,''), collected_at, collected_by
		FROM soc2_evidence_log
		WHERE control_id = ? AND collected_at >= ? AND collected_at <= ?
		ORDER BY collected_at DESC
	`, controlID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []EvidenceRecord
	for rows.Next() {
		var r EvidenceRecord
		if err := rows.Scan(&r.ID, &r.ControlID, &r.Criterion, &r.EvidenceType,
			&r.Evidence, &r.Result, &r.Metadata, &r.CollectedAt, &r.CollectedBy); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ControlPeriodStatus represents the status of a control over a period.
type ControlPeriodStatus struct {
	ID             string     `json:"id"`
	ControlID      string     `json:"control_id"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Status         string     `json:"status"` // compliant | non_compliant | in_progress | not_applicable
	Findings       int        `json:"findings"`
	LastEvidenceAt *time.Time `json:"last_evidence_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// UpdateControlStatus updates or inserts the status of a control for a period.
func (e *TypeIICollector) UpdateControlStatus(ctx context.Context, req UpdateControlPeriodRequest) error {
	id := uuid.New().String()
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO soc2_control_status
			(id, control_id, period_start, period_end, status, findings, last_evidence_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(control_id, period_start, period_end) DO UPDATE SET
			status = excluded.status,
			findings = excluded.findings,
			last_evidence_at = excluded.last_evidence_at,
			updated_at = CURRENT_TIMESTAMP
	`, id, req.ControlID, req.PeriodStart, req.PeriodEnd, req.Status, req.Findings, req.LastEvidenceAt)
	return err
}

// UpdateControlPeriodRequest is the request to update control status.
type UpdateControlPeriodRequest struct {
	ControlID      string     `json:"control_id"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	Status         string     `json:"status"`
	Findings       int        `json:"findings"`
	LastEvidenceAt *time.Time `json:"last_evidence_at,omitempty"`
}

// Finding represents a SOC 2 audit finding.
type Finding struct {
	ID           string     `json:"id"`
	ControlID    string     `json:"control_id"`
	FindingID    string     `json:"finding_id"`
	Severity     string     `json:"severity"` // critical | high | medium | low
	Description  string     `json:"description"`
	EvidenceRef  string     `json:"evidence_ref,omitempty"`
	Status       string     `json:"status"` // open | in_resolution | closed | accepted_risk
	DiscoveredAt time.Time  `json:"discovered_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy   string     `json:"resolved_by,omitempty"`
	Notes        string     `json:"notes,omitempty"`
}

// CreateFinding creates a new finding.
func (e *TypeIICollector) CreateFinding(ctx context.Context, req CreateFindingRequest) (*Finding, error) {
	id := uuid.New().String()
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO soc2_findings
			(id, control_id, finding_id, severity, description, evidence_ref, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, req.ControlID, req.FindingID, req.Severity, req.Description, req.EvidenceRef, "open")
	if err != nil {
		return nil, err
	}
	return &Finding{
		ID:          id,
		ControlID:   req.ControlID,
		FindingID:   req.FindingID,
		Severity:    req.Severity,
		Description: req.Description,
		EvidenceRef: req.EvidenceRef,
		Status:      "open",
	}, nil
}

// CreateFindingRequest is the request to create a finding.
type CreateFindingRequest struct {
	ControlID   string `json:"control_id"`
	FindingID   string `json:"finding_id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// ResolveFinding marks a finding as resolved.
func (e *TypeIICollector) ResolveFinding(ctx context.Context, id, resolvedBy, notes string) error {
	_, err := e.db.ExecContext(ctx, `
		UPDATE soc2_findings
		SET status='closed', resolved_at=CURRENT_TIMESTAMP, resolved_by=?, notes=?
		WHERE id=?
	`, resolvedBy, notes, id)
	return err
}

// ListFindings returns all findings, optionally filtered.
func (e *TypeIICollector) ListFindings(ctx context.Context, controlID, status string) ([]Finding, error) {
	query := `SELECT id, control_id, finding_id, severity, description, COALESCE(evidence_ref,''),
		status, discovered_at, COALESCE(resolved_at,'1970-01-01'), COALESCE(resolved_by,''), COALESCE(notes,'')
		FROM soc2_findings WHERE 1=1`
	args := []any{}

	if controlID != "" {
		query += " AND control_id = ?"
		args = append(args, controlID)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY discovered_at DESC"

	rows, err := e.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Finding
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.ID, &f.ControlID, &f.FindingID, &f.Severity, &f.Description,
			&f.EvidenceRef, &f.Status, &f.DiscoveredAt, &f.ResolvedAt, &f.ResolvedBy, &f.Notes); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
