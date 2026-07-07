// Package marketplace implementa o marketplace de regras customizadas.
//
// Permite que tenants publiquem regras de validação compartilháveis e que
// outros tenants as instalem no seu ambiente.
//
// Regras no marketplace são read-only no sentido de que o marketplace
// mantém o registro central — cada tenant tem sua própria instância
// da regra ao instalar.
package marketplace

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service gerencia o marketplace de regras.
type Service struct {
	db *sql.DB
}

// NewService creates a marketplace service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// Rule representa uma regra no marketplace.
type Rule struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Code         string    `json:"code"`  // ex: "CUSTOM_001"
	Cadoc        string    `json:"cadoc"` // ex: "3040"
	RuleType     string    `json:"rule_type"`
	Config       string    `json:"config,omitempty"` // JSON
	AuthorIFID   string    `json:"author_if_id"`
	AuthorName   string    `json:"author_name,omitempty"`
	Rating       float64   `json:"rating"`
	RatingCount  int       `json:"rating_count"`
	InstallCount int       `json:"install_count"`
	Tags         []string  `json:"tags,omitempty"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// List retorna regras do marketplace, filtráveis.
func (s *Service) List(ctx context.Context, cadoc, tag string, limit, offset int) ([]Rule, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	baseQuery := `FROM marketplace_rules WHERE active = 1`
	args := []any{}

	if cadoc != "" {
		baseQuery += " AND cadoc = ?"
		args = append(args, cadoc)
	}
	if tag != "" {
		baseQuery += " AND ',' || tags || ',' LIKE '%,' || ? || ',%'"
		args = append(args, tag)
	}

	// Count total.
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rules: %w", err)
	}

	// Fetch page.
	query := `SELECT id, name, description, code, cadoc, rule_type, config,
		author_if_id, author_name, rating, rating_count, install_count, tags,
		active, created_at, updated_at ` + baseQuery +
		` ORDER BY rating DESC, install_count DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var desc, config, authorName, tagsCSV sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &desc, &r.Code, &r.Cadoc, &r.RuleType,
			&config, &r.AuthorIFID, &authorName, &r.Rating, &r.RatingCount,
			&r.InstallCount, &tagsCSV, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan rule: %w", err)
		}
		r.Description = desc.String
		r.Config = config.String
		r.AuthorName = authorName.String
		if tagsCSV.Valid {
			r.Tags = strings.Split(tagsCSV.String, ",")
		}
		rules = append(rules, r)
	}
	return rules, total, rows.Err()
}

// Publish cria uma nova regra no marketplace.
func (s *Service) Publish(ctx context.Context, req PublishRuleRequest) (*Rule, error) {
	if err := validateRuleType(req.RuleType); err != nil {
		return nil, err
	}
	if err := validateCode(req.Code); err != nil {
		return nil, err
	}

	configJSON := ""
	if req.Config != nil {
		b, _ := json.Marshal(req.Config)
		configJSON = string(b)
	}

	id := uuid.New().String()
	now := time.Now()
	tags := strings.Join(req.Tags, ",")

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO marketplace_rules
			(id, name, description, code, cadoc, rule_type, config, author_if_id, author_name, tags, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, id, req.Name, req.Description, req.Code, req.Cadoc, req.RuleType, configJSON,
		req.AuthorIFID, req.AuthorName, tags, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert rule: %w", err)
	}

	return &Rule{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Cadoc:       req.Cadoc,
		RuleType:    req.RuleType,
		Config:      configJSON,
		AuthorIFID:  req.AuthorIFID,
		AuthorName:  req.AuthorName,
		Tags:        req.Tags,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Install instala uma regra para um tenant.
func (s *Service) Install(ctx context.Context, ruleID, ifID string) error {
	// Verify rule exists and is active.
	var exists int
	if err := s.db.QueryRowContext(ctx,
		"SELECT 1 FROM marketplace_rules WHERE id = ? AND active = 1", ruleID,
	).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("regra não encontrada")
		}
		return err
	}

	id := uuid.New().String()
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO marketplace_installs (id, rule_id, if_id) VALUES (?, ?, ?)`,
		id, ruleID, ifID)
	if err != nil {
		return fmt.Errorf("install rule: %w", err)
	}

	// Increment install count.
	_, _ = s.db.ExecContext(ctx,
		`UPDATE marketplace_rules SET install_count = install_count + 1 WHERE id = ?`, ruleID)

	return nil
}

// Rate avalia uma regra (1-5 estrelas).
func (s *Service) Rate(ctx context.Context, ruleID, ifID string, stars int) error {
	if stars < 1 || stars > 5 {
		return fmt.Errorf("rating deve ser entre 1 e 5")
	}

	// Upsert rating (one rating per user per rule).
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO marketplace_ratings (id, rule_id, if_id, stars)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(rule_id, if_id) DO UPDATE SET stars = excluded.stars`,
		uuid.New().String(), ruleID, ifID, stars)
	if err != nil {
		// Rating table may not exist yet; fall back to simple update.
		_, _ = s.db.ExecContext(ctx,
			`UPDATE marketplace_rules SET rating_count = rating_count + 1,
			 rating = (rating * rating_count + ?) / (rating_count + 1)
			 WHERE id = ?`, stars, ruleID)
		return nil
	}

	// Recalculate average.
	var avg float64
	var cnt int
	if err := s.db.QueryRowContext(ctx,
		`SELECT AVG(stars), COUNT(*) FROM marketplace_ratings WHERE rule_id = ?`, ruleID,
	).Scan(&avg, &cnt); err == nil {
		_, _ = s.db.ExecContext(ctx,
			`UPDATE marketplace_rules SET rating = ?, rating_count = ? WHERE id = ?`,
			avg, cnt, ruleID)
	}

	return nil
}

// GetInstalled retorna as regras instaladas por um tenant.
func (s *Service) GetInstalled(ctx context.Context, ifID string) ([]Rule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT mr.id, mr.name, mr.description, mr.code, mr.cadoc, mr.rule_type,
		       mr.config, mr.author_if_id, mr.author_name, mr.rating, mr.rating_count,
		       mr.install_count, mr.tags, mr.active, mr.created_at, mr.updated_at
		FROM marketplace_rules mr
		JOIN marketplace_installs mi ON mi.rule_id = mr.id
		WHERE mr.active = 1 AND mi.if_id = ?
		ORDER BY mr.rating DESC
	`, ifID)
	if err != nil {
		return nil, fmt.Errorf("get installed: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var r Rule
		var desc, config, authorName, tagsCSV sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &desc, &r.Code, &r.Cadoc, &r.RuleType,
			&config, &r.AuthorIFID, &authorName, &r.Rating, &r.RatingCount,
			&r.InstallCount, &tagsCSV, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}
		r.Description = desc.String
		r.Config = config.String
		r.AuthorName = authorName.String
		if tagsCSV.Valid {
			r.Tags = strings.Split(tagsCSV.String, ",")
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// PublishRuleRequest é o request para publicar uma regra.
type PublishRuleRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Code        string         `json:"code"` // ex: "CUSTOM_001"
	Cadoc       string         `json:"cadoc"`
	RuleType    string         `json:"rule_type"` // 'format' | 'semantic' | 'crossdoc'
	Config      map[string]any `json:"config,omitempty"`
	AuthorIFID  string         `json:"author_if_id"`
	AuthorName  string         `json:"author_name,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

// ============================================================
// Validation helpers
// ============================================================

var validRuleTypes = map[string]bool{
	"format":   true,
	"semantic": true,
	"crossdoc": true,
	"raw":      true,
}

func validateRuleType(rt string) error {
	if !validRuleTypes[rt] {
		return fmt.Errorf("rule_type deve ser: format, semantic, crossdoc ou raw")
	}
	return nil
}

var codePrefixes = []string{"CUSTOM_", "X_", "AUDIT_"}

func validateCode(code string) error {
	if len(code) < 4 || len(code) > 32 {
		return fmt.Errorf("code deve ter 4-32 caracteres")
	}
	validPrefix := false
	for _, p := range codePrefixes {
		if strings.HasPrefix(code, p) {
			validPrefix = true
			break
		}
	}
	if !validPrefix && !strings.HasPrefix(code, "CUSTOM_") {
		return fmt.Errorf("code deve começar com CUSTOM_, X_ ou AUDIT_")
	}
	if strings.ContainsAny(code, " <>&\"'") {
		return fmt.Errorf("code contém caracteres inválidos")
	}
	return nil
}
