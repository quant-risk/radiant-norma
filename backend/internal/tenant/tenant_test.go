// Tests for tenant package — tenant lifecycle management.
//
// Cobertura:
//   - Create: válido, CNPJ duplicado, validação de campos
//   - Get: existe, não existe
//   - GetByCNPJ: existe, não existe, CNPJ inválido
//   - Deactivate: ok, já desativado
//   - UpdatePlano: ok, plano inválido
//   - List: todos, filtro segmento
package tenant_test

import (
	"context"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/tenant"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

func TestCreate_Ok(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	tn, err := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ:     "12345678",
		Nome:     "IP Médio Teste",
		Tipo:     "IP",
		Segmento: "S3",
		Plano:    "pro",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if tn.CNPJ != "12345678" {
		t.Errorf("CNPJ = %q, want 12345678", tn.CNPJ)
	}
	if tn.Nome != "IP Médio Teste" {
		t.Errorf("Nome = %q, want 'IP Médio Teste'", tn.Nome)
	}
	if tn.Tipo != "IP" {
		t.Errorf("Tipo = %q, want IP", tn.Tipo)
	}
	if tn.Segmento != "S3" {
		t.Errorf("Segmento = %q, want S3", tn.Segmento)
	}
	if tn.Plano != "pro" {
		t.Errorf("Plano = %q, want pro", tn.Plano)
	}
	if !tn.Ativo {
		t.Error("Ativo should be true after creation")
	}
}

func TestCreate_DuplicateCNPJ(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "99999999", Nome: "Tenant 1", Tipo: "IP", Segmento: "S3", Plano: "lite",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "99999999", Nome: "Tenant 2", Tipo: "SCD", Segmento: "S4", Plano: "pro",
	})
	if err == nil {
		t.Fatal("Expected error for duplicate CNPJ")
	}
}

func TestCreate_InvalidCNPJ(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	tests := []struct {
		cnpj  string
		valid bool
	}{
		{"12345678", true},
		{"1234567", false},   // 7 digits
		{"123456789", false}, // 9 digits
		{"", false},
		{"abcdefgh", false},
	}

	for _, tc := range tests {
		_, err := svc.Create(context.Background(), tenant.CreateTenantInput{
			CNPJ: tc.cnpj, Nome: "Test", Tipo: "IP", Segmento: "S3", Plano: "lite",
		})
		if tc.valid && err != nil {
			t.Errorf("CNPJ %q should be valid, got err: %v", tc.cnpj, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("CNPJ %q should be invalid, but no error", tc.cnpj)
		}
	}
}

func TestCreate_InvalidTipo(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "12345678", Nome: "Test", Tipo: "INVALIDO", Segmento: "S3", Plano: "lite",
	})
	if err == nil {
		t.Fatal("Expected error for invalid tipo")
	}
}

func TestCreate_InvalidSegmento(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "12345678", Nome: "Test", Tipo: "IP", Segmento: "S99", Plano: "lite",
	})
	if err == nil {
		t.Fatal("Expected error for invalid segmento")
	}
}

func TestCreate_InvalidPlano(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "12345678", Nome: "Test", Tipo: "IP", Segmento: "S3", Plano: "invalid",
	})
	if err == nil {
		t.Fatal("Expected error for invalid plano")
	}
}

func TestGet_NotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.Get(context.Background(), "inexistente")
	if err == nil {
		t.Fatal("Expected error for non-existent tenant")
	}
}

func TestGet_Ok(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	created, _ := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "11111111", Nome: "Get Test", Tipo: "IP", Segmento: "S2", Plano: "scale",
	})

	tn, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if tn.ID != created.ID {
		t.Errorf("ID = %q, want %q", tn.ID, created.ID)
	}
}

func TestGetByCNPJ_NotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.GetByCNPJ(context.Background(), "00000000")
	if err == nil {
		t.Fatal("Expected error for non-existent CNPJ")
	}
}

func TestGetByCNPJ_InvalidCNPJ(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	_, err := svc.GetByCNPJ(context.Background(), "123")
	if err == nil {
		t.Fatal("Expected error for invalid CNPJ")
	}
}

func TestGetByCNPJ_Ok(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	created, _ := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "22222222", Nome: "CNPJ Test", Tipo: "SCD", Segmento: "S1", Plano: "enterprise",
	})

	tn, err := svc.GetByCNPJ(context.Background(), "22222222")
	if err != nil {
		t.Fatalf("GetByCNPJ failed: %v", err)
	}
	if tn.ID != created.ID {
		t.Errorf("ID = %q, want %q", tn.ID, created.ID)
	}
}

func TestDeactivate_Ok(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	created, _ := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "33333333", Nome: "Deactivate Test", Tipo: "IP", Segmento: "S3", Plano: "lite",
	})

	err := svc.Deactivate(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}

	// Get deve retornar erro
	_, err = svc.Get(context.Background(), created.ID)
	if err == nil {
		t.Error("Get should fail after deactivate")
	}
}

func TestDeactivate_NotFound(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	err := svc.Deactivate(context.Background(), "inexistente")
	if err == nil {
		t.Fatal("Expected error for non-existent tenant")
	}
}

func TestUpdatePlano_Ok(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	created, _ := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "44444444", Nome: "Update Plano Test", Tipo: "IP", Segmento: "S3", Plano: "lite",
	})

	err := svc.UpdatePlano(context.Background(), created.ID, "pro")
	if err != nil {
		t.Fatalf("UpdatePlano failed: %v", err)
	}

	tn, _ := svc.Get(context.Background(), created.ID)
	if tn.Plano != "pro" {
		t.Errorf("Plano = %q, want pro", tn.Plano)
	}
}

func TestUpdatePlano_Invalid(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	created, _ := svc.Create(context.Background(), tenant.CreateTenantInput{
		CNPJ: "55555555", Nome: "Test", Tipo: "IP", Segmento: "S3", Plano: "lite",
	})

	err := svc.UpdatePlano(context.Background(), created.ID, "invalidplan")
	if err == nil {
		t.Fatal("Expected error for invalid plano")
	}
}

func TestList_All(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	// Cria 3 tenants
	for i, cnpj := range []string{"60111111", "60222222", "60333333"} {
		_, _ = svc.Create(context.Background(), tenant.CreateTenantInput{
			CNPJ: cnpj, Nome: "List Test", Tipo: "IP", Segmento: "S3", Plano: "lite",
		})
		_ = i
	}

	tenants, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	// Os 3 novos + os seedados pelo NewTestDB
	if len(tenants) < 3 {
		t.Errorf("Expected at least 3 tenants, got %d", len(tenants))
	}
}

func TestList_FilterSegmento(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := tenant.NewService(d)

	// Cria tenants em segmentos diferentes
	for _, seg := range []string{"S1", "S1", "S3"} {
		cnpj := "70" + seg + "00000"
		_, _ = svc.Create(context.Background(), tenant.CreateTenantInput{
			CNPJ: cnpj, Nome: "Seg Test " + seg, Tipo: "IP", Segmento: seg, Plano: "lite",
		})
	}

	tenants, err := svc.List(context.Background(), "S1")
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}
	for _, tn := range tenants {
		if tn.Segmento != "S1" {
			t.Errorf("Got segmento %q, want S1", tn.Segmento)
		}
	}
}
