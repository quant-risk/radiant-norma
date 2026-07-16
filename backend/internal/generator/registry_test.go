// registry_test.go — Sprint 57 v3.36.3: valida auto-registro.
//
// Testa que RegisterDefaults popula o Registry com todos os generators
// que o caller passar, e que NewRegistry retorna vazio sem registration.
//
// Estes testes NÃO dependem de importar os subpacotes gen* (porque
// estamos no pacote generator que não pode importar eles por causa do
// ciclo). Em vez disso, usamos stubs via mocks.
package generator

import (
	"context"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
)

// stubGen é um CADOCGenerator fake para testes.
type stubGen struct {
	code string
}

func (s *stubGen) CadocCode() string { return s.code }
func (s *stubGen) RootTag() string    { return s.code } // Phase 1.2: unified parser+generator
func (s *stubGen) Generate(_ context.Context, _ *canonical.CanonicalDocument, _ time.Time) (*GeneratedDoc, error) {
	return &GeneratedDoc{CadocCode: s.code}, nil
}
func (s *stubGen) RequiredFields() []schema.Field { return nil }
func (s *stubGen) SupportedVersions() []string    { return []string{"1.0"} }
func (s *stubGen) EstimateComplexity(_ *canonical.CanonicalDocument) ComplexityScore {
	return ComplexityScore{Score: 0.1}
}

func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if len(r.List()) != 0 {
		t.Fatalf("NewRegistry should return empty, got %d generators", len(r.List()))
	}
	if r.IsRegistered("3040") {
		t.Fatal("registry should not have any generators by default")
	}
}

func TestRegister_AddsGenerator(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubGen{code: "3040"})

	if !r.IsRegistered("3040") {
		t.Fatal("registry should contain 3040 after Register")
	}
	if got := r.Get("3040"); got == nil {
		t.Fatal("Get(3040) returned nil")
	} else if got.CadocCode() != "3040" {
		t.Fatalf("Get(3040) returned %q, want %q", got.CadocCode(), "3040")
	}
}

func TestRegisterDefaults_PopulatesAll(t *testing.T) {
	r := NewRegistry()
	gens := []CADOCGenerator{
		&stubGen{code: "2030"},
		&stubGen{code: "2060"},
		&stubGen{code: "2061"},
		&stubGen{code: "2062"},
		&stubGen{code: "2070"},
		&stubGen{code: "2160"},
		&stubGen{code: "2170"},
		&stubGen{code: "3040"},
		&stubGen{code: "3050"},
		&stubGen{code: "4111"},
	}
	RegisterDefaults(r, gens)

	if len(r.List()) != 10 {
		t.Fatalf("RegisterDefaults should register 10, got %d", len(r.List()))
	}

	for _, code := range []string{"2030", "2060", "2061", "2062", "2070", "2160", "2170", "3040", "3050", "4111"} {
		if !r.IsRegistered(code) {
			t.Errorf("expected CADOC %s to be registered", code)
		}
	}
}

func TestRegisterDefaults_NilSafety(t *testing.T) {
	// Não deve panic com registry nil.
	RegisterDefaults(nil, []CADOCGenerator{&stubGen{code: "3040"}})

	// Não deve panic com generators nil dentro da slice.
	r := NewRegistry()
	RegisterDefaults(r, []CADOCGenerator{nil, &stubGen{code: "3040"}, nil})

	if !r.IsRegistered("3040") {
		t.Fatal("3040 should be registered despite nil entries")
	}
}

func TestRegister_Overwrite(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubGen{code: "3040"})
	r.Register(&stubGen{code: "4111"}) // wrong code

	// Re-register com código correto.
	r.Register(&stubGen{code: "3040"})
	if got := r.Get("3040"); got == nil {
		t.Fatal("3040 should still be registered")
	}
}

func TestGet_UnknownReturnsNil(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubGen{code: "3040"})

	if got := r.Get("9999"); got != nil {
		t.Fatalf("Get(9999) should return nil, got %v", got.CadocCode())
	}
}
