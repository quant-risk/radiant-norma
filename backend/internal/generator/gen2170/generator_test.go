package gen2170

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
	if got := g.CadocCode(); got != "2170" {
		t.Errorf("CadocCode() = %q, want 2170", got)
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
	if result.CadocCode != "2170" {
		t.Errorf("CadocCode = %q, want 2170", result.CadocCode)
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
		CadocCode: "2170",
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

func TestGenerator_NSFRCalculation(t *testing.T) {
	tests := []struct {
		name string
		asf  float64
		rsf  float64
		want float64
	}{
		{"normal", 125, 100, 125.0},
		{"zero rsf", 100, 0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateNSFR(tt.asf, tt.rsf)
			if got != tt.want {
				t.Errorf("calculateNSFR = %v, want %v", got, tt.want)
			}
		})
	}
}

func newTestDoc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "2170",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Extra: map[string]any{
			"asfTotal": "5000000.00",
			"rsfTotal": "4000000.00",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-1",
				TipoOperacao:   "D",
				Modalidade:     "30.01",
				NumeroContrato: "CTR-001",
				ValorPrincipal: canonical.Money{Valor: decimalNew("2000000.00"), Moeda: "BRL"},
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
