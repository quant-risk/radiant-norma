// Tests for branding package — WhiteLabel branding service.
//
// Cobertura:
//   - GetBranding: tenant encontrado, não encontrado, tenant vazio
//   - GetBrandingBySlug: slug encontrado, não encontrado, slug vazio
//   - UpdateBranding: update simples, update parcial, campos todos
//   - UpdateBranding: validação hex color, logo URL, slug
//   - UpdateBranding: slug único entre tenants
//   - Slug uniqueness constraint: update mesmo slug p/ outro tenant
package branding_test

import (
	"context"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/branding"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

func TestGetBranding_TenantNotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	_, err := svc.GetBranding(context.Background(), "inexistente")
	if err == nil {
		t.Fatal("esperado erro para tenant inexistente")
	}
}

func TestGetBranding_TenantEmpty(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	_, err := svc.GetBranding(context.Background(), "")
	if err == nil {
		t.Fatal("esperado erro para tenant_id vazio")
	}
}

func TestGetBranding_OK(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	// seed if-test existe em NewTestDB
	b, err := svc.GetBranding(context.Background(), "if-test")
	if err != nil {
		t.Fatalf("GetBranding failed: %v", err)
	}
	if b.TenantID != "if-test" {
		t.Errorf("TenantID = %q, want if-test", b.TenantID)
	}
	// Defaults
	if b.PrimaryColor != "#3b6ef5" {
		t.Errorf("PrimaryColor = %q, want #3b6ef5", b.PrimaryColor)
	}
	if b.SecondaryColor != "#1a2a5e" {
		t.Errorf("SecondaryColor = %q, want #1a2a5e", b.SecondaryColor)
	}
}

func TestGetBrandingBySlug_NotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	_, err := svc.GetBrandingBySlug(context.Background(), "slug-inexistente")
	if err == nil {
		t.Fatal("esperado erro para slug inexistente")
	}
}

func TestGetBrandingBySlug_Empty(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	_, err := svc.GetBrandingBySlug(context.Background(), "")
	if err == nil {
		t.Fatal("esperado erro para slug vazio")
	}
}

func TestUpdateBranding_Simple(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	primary := "#ff0000"
	b, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		PrimaryColor: &primary,
	})
	if err != nil {
		t.Fatalf("UpdateBranding failed: %v", err)
	}
	if b.PrimaryColor != "#ff0000" {
		t.Errorf("PrimaryColor = %q, want #ff0000", b.PrimaryColor)
	}

	// Verifica persistência
	b2, err := svc.GetBranding(context.Background(), "if-test")
	if err != nil {
		t.Fatalf("GetBranding after update: %v", err)
	}
	if b2.PrimaryColor != "#ff0000" {
		t.Errorf("PrimaryColor após persistência = %q, want #ff0000", b2.PrimaryColor)
	}
}

func TestUpdateBranding_AllFields(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	logo := "https://example.com/logo.png"
	primary := "#aabbcc"
	secondary := "#ddeeaa"
	domain := "branding.example.com"
	slug := "tenant-x"

	b, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		LogoURL:        &logo,
		PrimaryColor:   &primary,
		SecondaryColor: &secondary,
		CustomDomain:   &domain,
		TenantSlug:     &slug,
	})
	if err != nil {
		t.Fatalf("UpdateBranding all fields: %v", err)
	}
	if b.LogoURL != logo {
		t.Errorf("LogoURL = %q, want %q", b.LogoURL, logo)
	}
	if b.PrimaryColor != primary {
		t.Errorf("PrimaryColor = %q, want %q", b.PrimaryColor, primary)
	}
	if b.SecondaryColor != secondary {
		t.Errorf("SecondaryColor = %q, want %q", b.SecondaryColor, secondary)
	}
	if b.CustomDomain != domain {
		t.Errorf("CustomDomain = %q, want %q", b.CustomDomain, domain)
	}
	if b.TenantSlug != slug {
		t.Errorf("TenantSlug = %q, want %q", b.TenantSlug, slug)
	}
}

func TestUpdateBranding_InvalidHexColor(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	invalid := "#xyz"
	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		PrimaryColor: &invalid,
	})
	if err == nil {
		t.Fatal("esperado erro para hex color inválido")
	}
}

func TestUpdateBranding_InvalidLogoURL(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	invalid := "not-a-url"
	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		LogoURL: &invalid,
	})
	if err == nil {
		t.Fatal("esperado erro para logo URL inválido")
	}
}

func TestUpdateBranding_EmptyLogoURLOk(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	empty := ""
	b, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		LogoURL: &empty,
	})
	if err != nil {
		t.Fatalf("UpdateBranding com logo_url vazio: %v", err)
	}
	if b.LogoURL != "" {
		t.Errorf("LogoURL = %q, want empty", b.LogoURL)
	}
}

func TestUpdateBranding_InvalidSlug(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	tests := []struct {
		slug  string
		valid bool
	}{
		{"abc", true},
		{"a", false}, // muito curto
		{"ab", true}, // min 2
		{"my-tenant", true},
		{"My-Tenant", false}, // maiúsculas
		{"my_tenant", false}, // underscore
		{"-tenant", false},   // starts with hyphen
		{"tenant-", false},   // ends with hyphen
		{"my tenant", false}, // espaço
		{"my@tenant", false}, // char especial
	}

	for _, tc := range tests {
		_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
			TenantSlug: &tc.slug,
		})
		if tc.valid && err != nil {
			t.Errorf("slug %q deveria ser válido, got err: %v", tc.slug, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("slug %q deveria ser inválido, mas não deu erro", tc.slug)
		}
	}
}

func TestUpdateBranding_SlugUniqueAcrossTenants(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	slug := "unique-slug"

	// if-test toma o slug primeiro
	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		TenantSlug: &slug,
	})
	if err != nil {
		t.Fatalf("if-test deveria conseguir usar slug: %v", err)
	}

	// if-demo tenta usar o mesmo slug
	_, err = svc.UpdateBranding(context.Background(), "if-demo", branding.UpdateBrandingRequest{
		TenantSlug: &slug,
	})
	if err == nil {
		t.Fatal("esperado erro para slug duplicado por outro tenant")
	}
}

func TestUpdateBranding_SlugSameTenantAllowed(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	slug := "same-tenant-slug"

	// if-test toma o slug
	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		TenantSlug: &slug,
	})
	if err != nil {
		t.Fatalf("if-test deveria conseguir usar slug: %v", err)
	}

	// if-test atualiza o mesmo slug (mesmo tenant = OK)
	_, err = svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		TenantSlug: &slug,
	})
	if err != nil {
		t.Fatalf("if-test atualizando próprio slug deveria funcionar: %v", err)
	}
}

func TestUpdateBranding_PartialUpdate(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	// Seta todos os campos primeiro
	logo := "https://example.com/logo.png"
	primary := "#ff0000"
	secondary := "#00ff00"
	domain := "domain.com"
	slug := "partial-test"

	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		LogoURL:        &logo,
		PrimaryColor:   &primary,
		SecondaryColor: &secondary,
		CustomDomain:   &domain,
		TenantSlug:     &slug,
	})
	if err != nil {
		t.Fatalf("setup completo: %v", err)
	}

	// Atualiza só primary_color — outros campos devem permanecer
	newPrimary := "#0000ff"
	b, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		PrimaryColor: &newPrimary,
	})
	if err != nil {
		t.Fatalf("partial update: %v", err)
	}
	if b.PrimaryColor != "#0000ff" {
		t.Errorf("PrimaryColor = %q, want #0000ff", b.PrimaryColor)
	}
	if b.LogoURL != logo {
		t.Errorf("LogoURL = %q, want %q (inalterado)", b.LogoURL, logo)
	}
	if b.SecondaryColor != secondary {
		t.Errorf("SecondaryColor = %q, want %q (inalterado)", b.SecondaryColor, secondary)
	}
	if b.CustomDomain != domain {
		t.Errorf("CustomDomain = %q, want %q (inalterado)", b.CustomDomain, domain)
	}
	if b.TenantSlug != slug {
		t.Errorf("TenantSlug = %q, want %q (inalterado)", b.TenantSlug, slug)
	}
}

func TestGetBrandingBySlug_AfterUpdate(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	slug := "get-by-slug-test"
	primary := "#112233"

	_, err := svc.UpdateBranding(context.Background(), "if-test", branding.UpdateBrandingRequest{
		TenantSlug:   &slug,
		PrimaryColor: &primary,
	})
	if err != nil {
		t.Fatalf("UpdateBranding: %v", err)
	}

	b, err := svc.GetBrandingBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("GetBrandingBySlug: %v", err)
	}
	if b.TenantID != "if-test" {
		t.Errorf("TenantID = %q, want if-test", b.TenantID)
	}
	if b.PrimaryColor != primary {
		t.Errorf("PrimaryColor = %q, want %q", b.PrimaryColor, primary)
	}
}

func TestUpdateBranding_TenantNotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	primary := "#111111"
	_, err := svc.UpdateBranding(context.Background(), "inexistente", branding.UpdateBrandingRequest{
		PrimaryColor: &primary,
	})
	if err == nil {
		t.Fatal("esperado erro para tenant inexistente")
	}
}

func TestUpdateBranding_TenantEmpty(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := branding.NewBrandingService(d)

	primary := "#111111"
	_, err := svc.UpdateBranding(context.Background(), "", branding.UpdateBrandingRequest{
		PrimaryColor: &primary,
	})
	if err == nil {
		t.Fatal("esperado erro para tenant_id vazio")
	}
}
