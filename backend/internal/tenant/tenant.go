// Package tenant implementa gestão de lifecycle de tenants no Radiant Norma.
//
// Funcionalidades:
//   - Onboarding: criar tenant com CNPJ, tipo, segmento
//   - Ativação: marcar tenant como ativo (após onboarding completo)
//   - Desativação: soft-delete (não remove dados)
//   - Query: buscar por ID, CNPJ, segmento
//
// Este package é o centro de gravidade para tudo que muda no lifecycle
// de um tenant (criação → produção → upgrade/downgrade → cancelamento).
package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

// Tenant representa uma instituição financeira no Radiant Norma.
type Tenant struct {
	ID           string
	CNPJ         string
	Nome         string
	Tipo         string // SCD, IP, SEP, BC, SCD_S3, IP_S3
	Segmento     string // S1, S2, S3, S4, S5
	Plano        string // lite, pro, scale, enterprise
	Ativo        bool
	StripeCustID string
	CreatedAt    string
	UpdatedAt    string
}

// Segmentos válidos.
var Segmentos = map[string]bool{
	"S1": true, "S2": true, "S3": true, "S4": true, "S5": true,
}

// Planos válidos.
var Planos = map[string]bool{
	"lite": true, "pro": true, "scale": true, "enterprise": true,
}

// Tipos válidos.
var Tipos = map[string]bool{
	"SCD": true, "IP": true, "SEP": true, "BC": true,
	"SCD_S3": true, "IP_S3": true,
}

// Service de tenants.
type Service struct {
	db *sql.DB
}

// NewService cria um novo TenantService.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateTenantInput é o input para criar um novo tenant.
type CreateTenantInput struct {
	ID       string // optional — se vazio, gera UUID
	CNPJ     string
	Nome     string
	Tipo     string
	Segmento string
	Plano    string
}

// Create cria um novo tenant no banco.
func (s *Service) Create(ctx context.Context, input CreateTenantInput) (*Tenant, error) {
	if err := validateCNPJ(input.CNPJ); err != nil {
		return nil, fmt.Errorf("cnpj: %w", err)
	}
	if !Tipos[input.Tipo] {
		return nil, fmt.Errorf("tipo %q inválido (SCD, IP, SEP, BC, SCD_S3, IP_S3)", input.Tipo)
	}
	if !Segmentos[input.Segmento] {
		return nil, fmt.Errorf("segmento %q inválido (S1-S5)", input.Segmento)
	}
	if !Planos[input.Plano] {
		return nil, fmt.Errorf("plano %q inválido (lite, pro, scale, enterprise)", input.Plano)
	}

	tenantID := input.ID
	if tenantID == "" {
		tenantID = generateID()
	}

	// Verifica CNPJ único
	var existing string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM ifs WHERE cnpj = ? AND deleted_at IS NULL`, input.CNPJ).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("cnpj %s já existe (tenant %s)", input.CNPJ, existing)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("verificar cnpj: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO ifs (id, cnpj, nome, tipo, segmento, plano, sta_service)
		VALUES (?, ?, ?, ?, ?, ?, 'PSTA300')
	`, tenantID, input.CNPJ, input.Nome, input.Tipo, input.Segmento, input.Plano)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}

	return s.Get(ctx, tenantID)
}

// Get retorna um tenant pelo ID.
func (s *Service) Get(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	var ativo int // SQLite não tem bool
	var plano, stripeCustID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, cnpj, nome, tipo, COALESCE(segmento, ''), plano,
		       COALESCE(stripe_customer_id, ''),
		       deleted_at IS NULL AS ativo,
		       created_at, updated_at
		FROM ifs
		WHERE id = ? AND deleted_at IS NULL
	`, id).Scan(
		&t.ID, &t.CNPJ, &t.Nome, &t.Tipo, &t.Segmento,
		&plano, &stripeCustID, &ativo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant %s não encontrado", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query tenant: %w", err)
	}

	t.Plano = plano.String
	t.StripeCustID = stripeCustID.String
	t.Ativo = ativo == 1

	return &t, nil
}

// GetByCNPJ retorna um tenant pelo CNPJ.
func (s *Service) GetByCNPJ(ctx context.Context, cnpj string) (*Tenant, error) {
	if err := validateCNPJ(cnpj); err != nil {
		return nil, fmt.Errorf("cnpj: %w", err)
	}
	var t Tenant
	var ativo int
	var plano, stripeCustID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, cnpj, nome, tipo, COALESCE(segmento, ''), plano,
		       COALESCE(stripe_customer_id, ''),
		       deleted_at IS NULL AS ativo,
		       created_at, updated_at
		FROM ifs
		WHERE cnpj = ? AND deleted_at IS NULL
	`, cnpj).Scan(
		&t.ID, &t.CNPJ, &t.Nome, &t.Tipo, &t.Segmento,
		&plano, &stripeCustID, &ativo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant com CNPJ %s não encontrado", cnpj)
	}
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	t.Plano = plano.String
	t.StripeCustID = stripeCustID.String
	t.Ativo = ativo == 1

	return &t, nil
}

// Deactivate desativa um tenant (soft-delete).
func (s *Service) Deactivate(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE ifs SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("deactivate: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("tenant não encontrado ou já desativado")
	}
	return nil
}

// UpdatePlano atualiza o plano de um tenant.
func (s *Service) UpdatePlano(ctx context.Context, id, plano string) error {
	if !Planos[plano] {
		return fmt.Errorf("plano %q inválido", plano)
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE ifs SET plano = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, plano, id)
	if err != nil {
		return fmt.Errorf("update plano: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("tenant não encontrado")
	}
	return nil
}

// List retorna todos os tenants ativos (com opção de filtro por segmento).
func (s *Service) List(ctx context.Context, segmento string) ([]Tenant, error) {
	query := `
		SELECT id, cnpj, nome, tipo, COALESCE(segmento, ''), plano,
		       COALESCE(stripe_customer_id, ''),
		       deleted_at IS NULL AS ativo,
		       created_at, updated_at
		FROM ifs
		WHERE deleted_at IS NULL
	`
	args := []any{}
	if segmento != "" {
		query += " AND segmento = ?"
		args = append(args, segmento)
	}
	query += " ORDER BY nome"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		var ativo int
		var plano, stripeCustID sql.NullString
		if err := rows.Scan(
			&t.ID, &t.CNPJ, &t.Nome, &t.Tipo, &t.Segmento,
			&plano, &stripeCustID, &ativo, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		t.Plano = plano.String
		t.StripeCustID = stripeCustID.String
		t.Ativo = ativo == 1
		tenants = append(tenants, t)
	}

	return tenants, rows.Err()
}

// Helpers

var cnpjRegex = regexp.MustCompile(`^\d{8}$`)

func validateCNPJ(cnpj string) error {
	if cnpj == "" {
		return errors.New("cnpj é obrigatório")
	}
	if !cnpjRegex.MatchString(cnpj) {
		return errors.New("cnpj deve ter exatamente 8 dígitos")
	}
	return nil
}

func generateID() string {
	return uuid.New().String()
}
