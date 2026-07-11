package gen4111

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
	if got := g.CadocCode(); got != "4111" {
		t.Errorf("CadocCode() = %q, want 4111", got)
	}
}

func TestGenerator_SupportedVersions(t *testing.T) {
	g := New()
	versions := g.SupportedVersions()
	if len(versions) == 0 {
		t.Error("SupportedVersions() returned empty")
	}
	if versions[0] != "1.0" {
		t.Errorf("SupportedVersions()[0] = %q, want 1.0", versions[0])
	}
}

func TestGenerator_RequiredFields(t *testing.T) {
	g := New()
	fields := g.RequiredFields()
	if len(fields) == 0 {
		t.Error("RequiredFields() returned empty")
	}
	// Verifica campos obrigatórios conhecidos.
	wantTags := []string{"dataBase", "cnpj", "codigoDocumento", "qtdCli"}
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

func TestGenerator_EstimateComplexity(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		Operacoes: make([]canonical.Operacao, 50),
	}
	score := g.EstimateComplexity(doc)
	if score.Score < 0.2 {
		t.Errorf("EstimateComplexity score too low: %v", score.Score)
	}
	if score.NumOperacoes != 50 {
		t.Errorf("EstimateComplexity NumOperacoes = %d, want 50", score.NumOperacoes)
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
	if result.CadocCode != "4111" {
		t.Errorf("CadocCode = %q, want 4111", result.CadocCode)
	}
	if len(result.XML) == 0 {
		t.Error("Generate() returned empty XML")
	}
	if result.SHA256 == "" {
		t.Error("Generate() returned empty SHA256")
	}
	if result.DataBase != dataBase {
		t.Errorf("DataBase = %v, want %v", result.DataBase, dataBase)
	}
}

func TestGenerator_Generate_EmptyOperations(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "4111",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Operacoes:    nil, // sem operações
	}
	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	result, err := g.Generate(context.Background(), doc, dataBase)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.XML) == 0 {
		t.Error("Generate() returned empty XML for doc without operations")
	}
	// Sem operações → sem Clientes (omitempty).
}

func TestGenerator_Generate_InvalidDoc(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID: "", // IFID obrigatório
	}
	_, err := g.Generate(context.Background(), doc, time.Now())
	if err == nil {
		t.Error("Generate() expected error for invalid doc")
	}
}

func TestGenerator_Generate_WithInadimplente(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "4111",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000199",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-1",
				TipoPessoa:     "PF",
				UF:             "SP",
				NumeroContrato: "CONTRATO-001",
				Modalidade:     "0213",
				NivelRisco:     "H", // alto risco → inadimplente
				ValorPrincipal: canonical.Money{Valor: decimalZero(), Moeda: "BRL"},
			},
			{
				ID:             "op-2",
				TipoPessoa:     "PJ",
				UF:             "RJ",
				NumeroContrato: "CONTRATO-002",
				Modalidade:     "0213",
				NivelRisco:     "A", // adimplente
				ValorPrincipal: canonical.Money{Valor: decimalZero(), Moeda: "BRL"},
			},
		},
	}
	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	result, err := g.Generate(context.Background(), doc, dataBase)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	xmlStr := string(result.XML)
	// Deve conter indicacao="S" para modalidade 0213 (inadimplente).
	if !containsString(xmlStr, `indicacao="S"`) {
		t.Errorf("Generate() XML should contain indicacao=S for high-risk operation\ngot: %s", xmlStr)
	}
}

// TestGenerator_GroupByCliente verifica a agregação por cliente.
func TestGenerator_GroupByCliente(t *testing.T) {
	doc := &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "4111",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		Operacoes: []canonical.Operacao{
			{ID: "op-1", TipoPessoa: "PF", UF: "SP", NumeroContrato: "C1", Modalidade: "0213"},
			{ID: "op-2", TipoPessoa: "PF", UF: "SP", NumeroContrato: "C1", Modalidade: "0213"}, // mesmo cliente
			{ID: "op-3", TipoPessoa: "PJ", UF: "RJ", NumeroContrato: "C2", Modalidade: "0450"},
		},
	}
	agregMap := groupByCliente(doc)
	if len(agregMap) != 2 { // 2 clientes únicos (C1=PF/SP, C2=PJ/RJ)
		t.Errorf("groupByCliente len = %d, want 2", len(agregMap))
	}
}

// TestGenerator_Totalizador verifica totalização.
func TestGenerator_Totalizador(t *testing.T) {
	clientes := []Cliente{
		{QtdCli: "10", CNPJ: "12345678", Modalidades: []Modalidade{
			{Codigo: "0213", Indicacao: "N"},
		}},
		{QtdCli: "5", CNPJ: "87654321", Modalidades: []Modalidade{
			{Codigo: "0450", Indicacao: "S"},
		}},
	}
	total := buildTotalizador(clientes)
	if total.QtdCliTotal != "15" {
		t.Errorf("QtdCliTotal = %q, want 15", total.QtdCliTotal)
	}
	if total.ModTotal != "2" {
		t.Errorf("ModTotal = %q, want 2", total.ModTotal)
	}
}

// TestGenerator_IsInadimplente testa detecção de inadimplência.
func TestGenerator_IsInadimplente(t *testing.T) {
	tests := []struct {
		name  string
		op    canonical.Operacao
		want  bool
	}{
		{"nivel risco H", canonical.Operacao{NivelRisco: "H"}, true},
		{"nivel risco A", canonical.Operacao{NivelRisco: "A"}, false},
		{"provisao > 5%", canonical.Operacao{PercentualProvisao: decimalNew("0.10")}, true},
		{"provisao < 5%", canonical.Operacao{PercentualProvisao: decimalNew("0.02")}, false},
		{"extra inadimplente S", canonical.Operacao{Extra: map[string]any{"inadimplente": "S"}}, true},
		{"extra inadimplente true", canonical.Operacao{Extra: map[string]any{"inadimplente": true}}, true},
		{"normal", canonical.Operacao{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInadimplente(tt.op)
			if got != tt.want {
				t.Errorf("isInadimplente() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerator_CleanDigits.
func TestGenerator_CleanDigits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"12345678", "12345678"},
		{"12.345.678/0001-99", "12345678000199"},
		{"abc123def", "123"},
		{"", ""},
	}
	for _, tt := range tests {
		got := cleanDigits(tt.input)
		if got != tt.want {
			t.Errorf("cleanDigits(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// helpers.

func newTestDoc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:      "if-001",
		DataBase:  canonical.DataBase(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)),
		CadocCode: "4111",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678",
			NomeIF: "Banco Teste",
		},
		VersaoLayout: "1.0",
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-1",
				TipoPessoa:     "PF",
				UF:             "SP",
				NumeroContrato: "CTR-001",
				Modalidade:     "0213",
				NivelRisco:     "B",
				ValorPrincipal: canonical.Money{Valor: decimalNew("50000.00"), Moeda: "BRL"},
			},
			{
				ID:             "op-2",
				TipoPessoa:     "PJ",
				UF:             "RJ",
				NumeroContrato: "CTR-002",
				Modalidade:     "0450",
				NivelRisco:     "A",
				ValorPrincipal: canonical.Money{Valor: decimalNew("200000.00"), Moeda: "BRL"},
			},
		},
	}
}

func decimalNew(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

var decimalZero = func() decimal.Decimal {
	d, _ := decimal.NewFromString("0")
	return d
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Verifica interface.
var _ generator.CADOCGenerator = (*Generator)(nil)
