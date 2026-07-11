package gen2061

import (
	"context"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"github.com/shopspring/decimal"
)

func TestGenerator_CadocCode(t *testing.T) {
	g := New()
	if got := g.CadocCode(); got != "2061" {
		t.Errorf("CadocCode() = %q, want 2061", got)
	}
}

func TestGenerator_SupportedVersions(t *testing.T) {
	g := New()
	versions := g.SupportedVersions()
	if len(versions) == 0 {
		t.Error("SupportedVersions() returned empty")
	}
}

func TestGenerator_RequiredFields(t *testing.T) {
	g := New()
	fields := g.RequiredFields()
	if len(fields) == 0 {
		t.Error("RequiredFields() returned empty")
	}
	wantTags := []string{"dataBase", "cnpj", "tpDocumento", "patrimonio"}
	found := make(map[string]bool)
	for _, f := range fields {
		found[f.Tag] = true
	}
	for _, tag := range wantTags {
		if !found[tag] {
			t.Errorf("RequiredFields() missing tag %q", tag)
		}
	}
}

func TestGenerator_Generate(t *testing.T) {
	g := New()
	doc := newTestDoc()
	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	result, err := g.Generate(context.Background(), doc, dataBase)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result == nil {
		t.Fatal("Generate() returned nil")
	}
	if result.CadocCode != "2061" {
		t.Errorf("CadocCode = %q, want 2061", result.CadocCode)
	}
	if len(result.XML) == 0 {
		t.Error("Generate() returned empty XML")
	}
	if result.SHA256 == "" {
		t.Error("Generate() returned empty SHA256")
	}
}

func TestGenerator_Generate_EmptyDoc(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "2061",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
	}
	_, err := g.Generate(context.Background(), doc, time.Now())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func newTestDoc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "2061",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Extra: map[string]any{
			"conta770":    "1000000.00",
			"limiteTotal": "5000000.00",
			"patrimonio":  "3000000.00",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-1",
				Modalidade:     "800.01",
				NumeroContrato: "CTR-001",
				ValorPrincipal: canonical.Money{Valor: decimalNew("500000.00"), Moeda: "BRL"},
			},
		},
	}
}

func decimalNew(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

var _ = decimal.NewFromString

// Verify interface.
var _ generator.CADOCGenerator = (*Generator)(nil)
