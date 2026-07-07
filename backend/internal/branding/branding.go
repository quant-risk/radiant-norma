// Package branding implementa gestão de branding WhiteLabel por tenant.
//
// Permite customização de logo, cores e domínio para Fintechs BaaS
// que revendem o Radiant Norma com sua própria marca.
package branding

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

// BrandingService gerencia branding de tenants.
type BrandingService struct {
	db *sql.DB
}

// NewBrandingService cria um novo BrandingService.
func NewBrandingService(db *sql.DB) *BrandingService {
	return &BrandingService{db: db}
}

// Branding representa o branding de um tenant.
type Branding struct {
	TenantID       string
	TenantName     string
	LogoURL        string
	PrimaryColor   string
	SecondaryColor string
	CustomDomain   string
	TenantSlug     string
}

// GetBranding retorna o branding de um tenant pelo ID.
func (s *BrandingService) GetBranding(ctx context.Context, tenantID string) (*Branding, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id é obrigatório")
	}

	var b Branding
	var logoURL, primaryColor, secondaryColor, customDomain, tenantSlug sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, nome,
			COALESCE(logo_url, ''),
			COALESCE(primary_color, '#3b6ef5'),
			COALESCE(secondary_color, '#1a2a5e'),
			COALESCE(custom_domain, ''),
			COALESCE(tenant_slug, '')
		FROM ifs
		WHERE id = ? AND deleted_at IS NULL
	`, tenantID).Scan(
		&b.TenantID,
		&b.TenantName,
		&logoURL,
		&primaryColor,
		&secondaryColor,
		&customDomain,
		&tenantSlug,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant %s não encontrado", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	b.LogoURL = logoURL.String
	b.PrimaryColor = primaryColor.String
	b.SecondaryColor = secondaryColor.String
	b.CustomDomain = customDomain.String
	b.TenantSlug = tenantSlug.String

	return &b, nil
}

// GetBrandingBySlug retorna o branding pelo tenant_slug (para rotas públicas).
func (s *BrandingService) GetBrandingBySlug(ctx context.Context, slug string) (*Branding, error) {
	if slug == "" {
		return nil, errors.New("slug é obrigatório")
	}

	var b Branding
	var logoURL, primaryColor, secondaryColor, customDomain, tenantID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, nome,
			COALESCE(logo_url, ''),
			COALESCE(primary_color, '#3b6ef5'),
			COALESCE(secondary_color, '#1a2a5e'),
			COALESCE(custom_domain, ''),
			COALESCE(tenant_slug, '')
		FROM ifs
		WHERE tenant_slug = ? AND deleted_at IS NULL
	`, slug).Scan(
		&tenantID,
		&b.TenantName,
		&logoURL,
		&primaryColor,
		&secondaryColor,
		&customDomain,
		&b.TenantSlug,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tenant com slug %s não encontrado", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	b.TenantID = tenantID.String
	b.LogoURL = logoURL.String
	b.PrimaryColor = primaryColor.String
	b.SecondaryColor = secondaryColor.String
	b.CustomDomain = customDomain.String

	return &b, nil
}

// UpdateBrandingRequest é o input para atualizar branding.
type UpdateBrandingRequest struct {
	LogoURL        *string `json:"logo_url"`
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
	CustomDomain   *string `json:"custom_domain"`
	TenantSlug     *string `json:"tenant_slug"`
}

// UpdateBranding atualiza o branding de um tenant.
func (s *BrandingService) UpdateBranding(ctx context.Context, tenantID string, req UpdateBrandingRequest) (*Branding, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_id é obrigatório")
	}

	// Validações antes de atualizar.
	if req.LogoURL != nil {
		if err := validateLogoURL(*req.LogoURL); err != nil {
			return nil, fmt.Errorf("logo_url: %w", err)
		}
	}
	if req.PrimaryColor != nil {
		if err := validateHexColor(*req.PrimaryColor); err != nil {
			return nil, fmt.Errorf("primary_color: %w", err)
		}
	}
	if req.SecondaryColor != nil {
		if err := validateHexColor(*req.SecondaryColor); err != nil {
			return nil, fmt.Errorf("secondary_color: %w", err)
		}
	}
	if req.TenantSlug != nil {
		if err := validateSlug(*req.TenantSlug); err != nil {
			return nil, fmt.Errorf("tenant_slug: %w", err)
		}
		// Verifica se slug já existe em outro tenant.
		exists, err := s.slugExistsOtherTenant(ctx, *req.TenantSlug, tenantID)
		if err != nil {
			return nil, fmt.Errorf("verificar slug: %w", err)
		}
		if exists {
			return nil, errors.New("slug já está em uso por outro tenant")
		}
	}

	// Executa update dinâmico (só campos não-nulos).
	if err := s.updateBrandingFields(ctx, tenantID, req); err != nil {
		return nil, err
	}

	return s.GetBranding(ctx, tenantID)
}

// updateBrandingFields atualiza campos de branding via UPDATE dinâmico.
func (s *BrandingService) updateBrandingFields(ctx context.Context, tenantID string, req UpdateBrandingRequest) error {
	// nolint: errcheck // contexto de query
	query := "UPDATE ifs SET updated_at = CURRENT_TIMESTAMP"
	args := []any{}

	if req.LogoURL != nil {
		query += ", logo_url = ?"
		args = append(args, *req.LogoURL)
	}
	if req.PrimaryColor != nil {
		query += ", primary_color = ?"
		args = append(args, *req.PrimaryColor)
	}
	if req.SecondaryColor != nil {
		query += ", secondary_color = ?"
		args = append(args, *req.SecondaryColor)
	}
	if req.CustomDomain != nil {
		query += ", custom_domain = ?"
		args = append(args, *req.CustomDomain)
	}
	if req.TenantSlug != nil {
		query += ", tenant_slug = ?"
		args = append(args, *req.TenantSlug)
	}

	query += " WHERE id = ? AND deleted_at IS NULL"
	args = append(args, tenantID)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update branding: %w", err)
	}
	return nil
}

// slugExistsOtherTenant verifica se slug existe em outro tenant.
func (s *BrandingService) slugExistsOtherTenant(ctx context.Context, slug string, tenantID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM ifs
		WHERE tenant_slug = ? AND id != ? AND deleted_at IS NULL
	`, slug, tenantID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Validações

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func validateHexColor(color string) error {
	if !hexColorRegex.MatchString(color) {
		return errors.New("deve ser hex color (ex: #3b6ef5)")
	}
	return nil
}

var logoURLRegex = regexp.MustCompile(`^https?://`)

func validateLogoURL(url string) error {
	if url == "" {
		return nil // opcional
	}
	if !logoURLRegex.MatchString(url) {
		return errors.New("deve ser URL válida (http:// ou https://)")
	}
	return nil
}

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

func validateSlug(slug string) error {
	if len(slug) < 2 || len(slug) > 63 {
		return errors.New("slug deve ter entre 2 e 63 caracteres")
	}
	if !slugRegex.MatchString(slug) {
		return errors.New("slug deve ser URL-safe (letras minúsculas, números, hífens)")
	}
	return nil
}
