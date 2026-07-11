package gen2060

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
	if got := g.CadocCode(); got != "2060" {
		t.Errorf("CadocCode() = %q, want 2060", got)
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
	if result.CadocCode != "2060" {
		t.Errorf("CadocCode = %q, want 2060", result.CadocCode)
	}
	if len(result.XML) == 0 {
		t.Error("Generate() returned empty XML")
	}
}

func TestGenerator_Generate_EmptyDoc(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "2060",
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
		CadocCode: "2060",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Extra: map[string]any{
			"rwajur1": "500000.00",
			"rwajur2": "300000.00",
			"rwajur3": "200000.00",
			"rwajur4": "100000.00",
			"vaR":     "50000.00",
			"svaR":    "75000.00",
			"rwacom":  "80000.00",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-1",
				Modalidade:     "CDI",
				Indexador:      "CDI",
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
