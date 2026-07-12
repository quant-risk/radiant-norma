// Sprint 57 v3.34.37: NormaGeneratorFoundation — wizard state machine.
package wizard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// Step representa o estado atual do wizard.
type Step string

const (
	StepSelectCadoc  Step = "select_cadoc"
	StepSelectSource Step = "select_source"
	StepMapFields    Step = "map_fields"
	StepPreview      Step = "preview"
	StepGenerate     Step = "generate"
)

// IsValid returns true if s is a known step.
func (s Step) IsValid() bool {
	switch s {
	case StepSelectCadoc, StepSelectSource, StepMapFields, StepPreview, StepGenerate:
		return true
	}
	return false
}

// Next returns the next step after s.
func (s Step) Next() Step {
	switch s {
	case StepSelectCadoc:
		return StepSelectSource
	case StepSelectSource:
		return StepMapFields
	case StepMapFields:
		return StepPreview
	case StepPreview:
		return StepGenerate
	default:
		return s
	}
}

// Session representa uma sessão ativa do wizard de geração.
type Session struct {
	ID            string          `json:"id"`
	IfID          string          `json:"if_id"`
	Step          Step            `json:"step"`
	CadocCode     string          `json:"cadoc_code,omitempty"`
	SourceType    string          `json:"source_type,omitempty"`
	CanonicalJSON string          `json:"canonical_json,omitempty"`
	GeneratedXML  string          `json:"generated_xml,omitempty"`
	FieldMapping  json.RawMessage `json:"field_mapping,omitempty"`
	Errors        json.RawMessage `json:"errors,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}

// Store gerencia sessões de wizard no DB.
type Store struct {
	db *sql.DB
}

// NewStore creates a wizard Store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ErrSessionNotFound returned when a session doesn't exist.
var ErrSessionNotFound = errors.New("wizard session not found")

// ErrInvalidTransition returned when a step transition is invalid.
var ErrInvalidTransition = errors.New("invalid wizard step transition")

// Create cria uma nova sessão para o tenant.
func (s *Store) Create(ctx context.Context, ifID string) (*Session, error) {
	id := generateSessionID()
	now := time.Now()
	session := &Session{
		ID:        id,
		IfID:      ifID,
		Step:      StepSelectCadoc,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO wizard_sessions (id, if_id, step, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, ifID, string(StepSelectCadoc), now, now)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// Get retorna uma sessão pelo ID.
func (s *Store) Get(ctx context.Context, id string) (*Session, error) {
	var step, cadocCode, sourceType, canonicalJSON, generatedXML string
	var fieldMapping, errorsJSON sql.NullString
	var createdAt, updatedAt time.Time
	var completedAt sql.NullTime
	var gotIfID string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, if_id, step, cadoc_code, source_type, canonical_json,
		       generated_xml, field_mapping, errors,
		       created_at, updated_at, completed_at
		FROM wizard_sessions WHERE id = ?
	`, id).Scan(
		&id, &gotIfID, &step, &cadocCode, &sourceType,
		&canonicalJSON, &generatedXML, &fieldMapping, &errorsJSON,
		&createdAt, &updatedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:            id,
		IfID:          gotIfID,
		Step:          Step(step),
		CadocCode:     cadocCode,
		SourceType:    sourceType,
		CanonicalJSON: canonicalJSON,
		GeneratedXML:  generatedXML,
		FieldMapping:  nullToRaw(fieldMapping),
		Errors:        nullToRaw(errorsJSON),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CompletedAt:   nullTime(completedAt),
	}, nil
}

// Advance avança a sessão para o próximo step com dados atualizados.
func (s *Store) Advance(ctx context.Context, id string, data map[string]any) (*Session, error) {
	session, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	next := session.Step.Next()
	now := time.Now()

	var canonicalJSON, fieldMapping string
	if b, err := json.Marshal(data); err == nil {
		canonicalJSON = string(b)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE wizard_sessions SET
			step = ?,
			cadoc_code = COALESCE(NULLIF(?, ''), cadoc_code),
			source_type = COALESCE(NULLIF(?, ''), source_type),
			canonical_json = COALESCE(NULLIF(?, ''), canonical_json),
			field_mapping = COALESCE(NULLIF(?, ''), field_mapping),
			updated_at = ?
		WHERE id = ?
	`, string(next), data["cadoc_code"], data["source_type"], canonicalJSON, fieldMapping, now, id)
	if err != nil {
		return nil, err
	}

	session.Step = next
	session.UpdatedAt = now
	return session, nil
}

// SetGeneratedXML salva o XML gerado e marca como complete.
func (s *Store) SetGeneratedXML(ctx context.Context, id, xml string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE wizard_sessions SET
			generated_xml = ?,
			step = ?,
			completed_at = ?,
			updated_at = ?
		WHERE id = ?
	`, xml, string(StepGenerate), now, now, id)
	return err
}

// SetError registra erros no wizard.
func (s *Store) SetError(ctx context.Context, id string, errs []string) error {
	b, _ := json.Marshal(errs)
	_, err := s.db.ExecContext(ctx, `
		UPDATE wizard_sessions SET errors = ?, updated_at = ? WHERE id = ?
	`, string(b), time.Now(), id)
	return err
}

// ListActive retorna sessões ativas para o tenant (não completadas).
func (s *Store) ListActive(ctx context.Context, ifID string) ([]*Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, if_id, step, cadoc_code, source_type, created_at, updated_at
		FROM wizard_sessions
		WHERE if_id = ? AND completed_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 10
	`, ifID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.IfID, &sess.Step, &sess.CadocCode, &sess.SourceType, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, &sess)
	}
	return sessions, rows.Err()
}

// Prune remove sessões abandonadas (>2h since updated_at) e completadas (>24h).
func (s *Store) Prune(ctx context.Context) (int, error) {
	cutoff := time.Now().Add(-2 * time.Hour)
	cutoffOld := time.Now().Add(-24 * time.Hour)
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM wizard_sessions
		WHERE (completed_at IS NULL AND updated_at < ?)
		   OR (completed_at IS NOT NULL AND completed_at < ?)
	`, cutoff, cutoffOld)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Helpers.

func nullToRaw(n sql.NullString) json.RawMessage {
	if n.Valid {
		return json.RawMessage(n.String)
	}
	return nil
}

func nullTime(n sql.NullTime) *time.Time {
	if n.Valid {
		return &n.Time
	}
	return nil
}

func generateSessionID() string {
	b := make([]byte, 8)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(now >> (i * 8))
	}
	const hex = "0123456789abcdef"
	r := make([]byte, len(b)*2)
	for i, v := range b {
		r[i*2] = hex[v>>4]
		r[i*2+1] = hex[v&0xf]
	}
	return "wiz_" + string(r)
}
