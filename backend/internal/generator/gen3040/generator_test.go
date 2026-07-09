package gen3040

import (
	"context"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/shopspring/decimal"
)

func TestGenerator_CadocCode(t *testing.T) {
	g := New()
	if got := g.CadocCode(); got != "3040" {
		t.Errorf("CadocCode = %q, want %q", got, "3040")
	}
}

func TestGenerator_SupportedVersions(t *testing.T) {
	g := New()
	versions := g.SupportedVersions()
	if len(versions) == 0 {
		t.Error("SupportedVersions returned empty")
	}
}

func TestGenerator_RequiredFields(t *testing.T) {
	g := New()
	fields := g.RequiredFields()
	if len(fields) == 0 {
		t.Error("RequiredFields returned empty")
	}
	// Check that critical fields are present.
	tags := make(map[string]bool)
	for _, f := range fields {
		tags[f.Tag] = true
	}
	want := []string{"dataBase", "cnpj", "remessa", "parte", "tpArq",
		"nomeResp", "emailResp", "telResp", "totalCli",
		"natuOp", "mod", "classOp", "v110"}
	for _, w := range want {
		if !tags[w] {
			t.Errorf("RequiredFields missing tag %q", w)
		}
	}
}

func TestGenerator_Generate_ValidDoc(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:         "if-test",
		CadocCode:    "3040",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "3.2",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste S.A.",
		},
		Extra: map[string]any{
			"nomeResp":  "Joao Silva",
			"emailResp": "joao@teste.com",
			"telResp":   "11999999999",
			"tpArq":     "F",
			"remessa":   "1",
			"parte":     "1",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op1",
				Modalidade:      "1000",
				ValorPrincipal:  canonical.Money{Valor: mustDecimal(50000.0), Moeda: "BRL"},
				TipoPessoa:      "PJ",
				UF:              "SP",
				ClassificacaoIF: "A",
				NumeroContrato:  "C001",
			},
		},
	}

	got, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(got.XML) == 0 {
		t.Error("Generate() returned empty XML")
	}
	if got.SHA256 == "" {
		t.Error("Generate() returned empty SHA256")
	}
	if got.CadocCode != "3040" {
		t.Errorf("CadocCode = %q, want %q", got.CadocCode, "3040")
	}
}

func TestGenerator_Generate_ValidationFailure(t *testing.T) {
	g := New()
	// Missing required fields: IFID, CadocCode, DataBase, CNPJ, NomeIF.
	doc := &canonical.CanonicalDocument{}

	_, err := g.Generate(context.Background(), doc, time.Now())
	if err == nil {
		t.Fatal("Generate() expected error for invalid doc, got nil")
	}
}

func TestGenerator_EstimateComplexity(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		CadocCode: "3040",
		Operacoes: make([]canonical.Operacao, 200),
	}
	score := g.EstimateComplexity(doc)
	if score.NumOperacoes != 200 {
		t.Errorf("NumOperacoes = %d, want 200", score.NumOperacoes)
	}
	if score.Score < 0.3 || score.Score > 1.0 {
		t.Errorf("Score out of range: %f", score.Score)
	}
}

// mustDecimal is a test helper to create decimal from float.
func mustDecimal(v float64) decimal.Decimal {
	return decimal.NewFromFloat(v)
}
