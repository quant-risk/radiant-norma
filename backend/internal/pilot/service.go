// Package pilot implementa o programa de onboarding de pilotos (Sprint 64).
//
// Gerencia programas piloto e participantes, incluindo onboarding steps
// para bancos S3/S4 no Pilot 4.
package pilot

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func newID() string { return uuid.New().String() }

// Service gerencia pilotos e onboarding.
type Service struct {
	db *sql.DB
}

// NewService creates a pilot service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// ============================================================
// Types
// ============================================================

// Program represents a pilot program.
type Program struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Participant represents a bank enrolled in a pilot program.
type Participant struct {
	ID        string    `json:"id"`
	ProgramID string    `json:"program_id"`
	IFID      string    `json:"if_id"`
	Status    string    `json:"status"` // onboarding|active|churned
	JoinedAt  time.Time `json:"joined_at"`
	Notes     string    `json:"notes,omitempty"`
}

// OnboardingStep represents a single onboarding step for a tenant.
type OnboardingStep struct {
	ID          string     `json:"id"`
	IFID        string     `json:"if_id"`
	StepKey     string     `json:"step_key"`
	Status      string     `json:"status"` // pending|in_progress|completed|skipped
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

// ============================================================
// Program operations
// ============================================================

// CreateProgram creates a new pilot program.
func (s *Service) CreateProgram(ctx context.Context, name, description string, startDate, endDate *time.Time) (*Program, error) {
	id := newID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pilot_programs (id, name, description, start_date, end_date, active)
		VALUES (?, ?, ?, ?, ?, 1)
	`, id, name, description, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("create program: %w", err)
	}
	return &Program{
		ID:          id,
		Name:        name,
		Description: description,
		StartDate:   startDate,
		EndDate:     endDate,
		Active:      true,
		CreatedAt:   time.Now(),
	}, nil
}

// ListPrograms returns all pilot programs.
func (s *Service) ListPrograms(ctx context.Context) ([]Program, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, start_date, end_date, active, created_at
		FROM pilot_programs ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Program
	for rows.Next() {
		var p Program
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.StartDate, &p.EndDate, &p.Active, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ============================================================
// Participant operations
// ============================================================

// Enroll enrolls a tenant in a pilot program.
func (s *Service) Enroll(ctx context.Context, programID, ifID string) error {
	id := newID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pilot_participants (id, program_id, if_id, status)
		VALUES (?, ?, ?, 'onboarding')
	`, id, programID, ifID)
	if err != nil {
		return fmt.Errorf("enroll: %w", err)
	}
	return nil
}

// GetParticipant returns a participant by program + if_id.
func (s *Service) GetParticipant(ctx context.Context, programID, ifID string) (*Participant, error) {
	var p Participant
	var notes sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, program_id, if_id, status, joined_at, notes
		FROM pilot_participants WHERE program_id = ? AND if_id = ?
	`, programID, ifID).Scan(&p.ID, &p.ProgramID, &p.IFID, &p.Status, &p.JoinedAt, &notes)
	if err != nil {
		return nil, err
	}
	p.Notes = notes.String
	return &p, nil
}

// UpdateParticipantStatus updates the status of a participant.
func (s *Service) UpdateParticipantStatus(ctx context.Context, programID, ifID, status string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pilot_participants SET status = ? WHERE program_id = ? AND if_id = ?
	`, status, programID, ifID)
	return err
}

// ListParticipants returns all participants of a program.
func (s *Service) ListParticipants(ctx context.Context, programID string) ([]Participant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, program_id, if_id, status, joined_at, notes
		FROM pilot_participants WHERE program_id = ? ORDER BY joined_at DESC
	`, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Participant
	for rows.Next() {
		var p Participant
		var notes sql.NullString
		if err := rows.Scan(&p.ID, &p.ProgramID, &p.IFID, &p.Status, &p.JoinedAt, &notes); err != nil {
			return nil, err
		}
		p.Notes = notes.String
		out = append(out, p)
	}
	return out, rows.Err()
}

// ============================================================
// Onboarding step operations
// ============================================================

// DefaultSteps são os passos padrão de onboarding.
var DefaultSteps = []string{
	"docs_submitted",
	"cadoc_tested",
	"integration_verified",
	"production_approved",
	"go_live",
}

// InitOnboarding cria todos os passos de onboarding para um tenant.
func (s *Service) InitOnboarding(ctx context.Context, ifID string) error {
	for _, key := range DefaultSteps {
		id := newID()
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO onboarding_steps (id, if_id, step_key, status)
			VALUES (?, ?, ?, 'pending')
		`, id, ifID, key)
		if err != nil {
			return fmt.Errorf("init onboarding: %w", err)
		}
	}
	return nil
}

// CompleteStep marca um passo como completado.
func (s *Service) CompleteStep(ctx context.Context, ifID, stepKey string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE onboarding_steps
		SET status = 'completed', completed_at = CURRENT_TIMESTAMP
		WHERE if_id = ? AND step_key = ?
	`, ifID, stepKey)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("step not found: %s", stepKey)
	}
	return nil
}

// GetOnboardingProgress returns all onboarding steps for a tenant.
func (s *Service) GetOnboardingProgress(ctx context.Context, ifID string) ([]OnboardingStep, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, if_id, step_key, status, completed_at, notes
		FROM onboarding_steps WHERE if_id = ? ORDER BY id
	`, ifID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OnboardingStep
	for rows.Next() {
		var step OnboardingStep
		var notes sql.NullString
		if err := rows.Scan(&step.ID, &step.IFID, &step.StepKey, &step.Status, &step.CompletedAt, &notes); err != nil {
			return nil, err
		}
		step.Notes = notes.String
		out = append(out, step)
	}
	return out, rows.Err()
}

// OnboardingProgress returns a summary of onboarding progress (0.0-1.0).
func (s *Service) OnboardingProgress(ctx context.Context, ifID string) (float64, error) {
	steps, err := s.GetOnboardingProgress(ctx, ifID)
	if err != nil {
		return 0, err
	}
	if len(steps) == 0 {
		return 0, nil
	}
	var completed int
	for _, st := range steps {
		if st.Status == "completed" {
			completed++
		}
	}
	return float64(completed) / float64(len(steps)), nil
}

// ============================================================
// Segment helpers
// ============================================================

// SetSegment updates a tenant's segment.
func (s *Service) SetSegment(ctx context.Context, ifID, segment string) error {
	if !ValidSegment(segment) {
		return fmt.Errorf("segment inválido: %s", segment)
	}
	_, err := s.db.ExecContext(ctx, "UPDATE ifs SET segment = ? WHERE id = ?", segment, ifID)
	return err
}

// GetSegment returns a tenant's segment.
func (s *Service) GetSegment(ctx context.Context, ifID string) (string, error) {
	var seg string
	err := s.db.QueryRowContext(ctx, "SELECT segment FROM ifs WHERE id = ?", ifID).Scan(&seg)
	return seg, err
}

// ValidSegment checks if a segment string is valid.
func ValidSegment(seg string) bool {
	switch seg {
	case "s1", "s2", "s3", "s4":
		return true
	default:
		return false
	}
}
