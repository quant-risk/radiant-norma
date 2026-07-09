package gen3050

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/shopspring/decimal"
)

func TestGenerator_CadocCode(t *testing.T) {
	g := New()
	if got := g.CadocCode(); got != "3050" {
		t.Errorf("CadocCode() = %q, want %q", got, "3050")
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
	// Check some known fields.
	var tags []string
	for _, f := range fields {
		tags = append(tags, f.Tag)
	}
	for _, want := range []string{"dataBase", "cnpj", "indRemessa", "txMedJuros"} {
		found := false
		for _, tag := range tags {
			if tag == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RequiredFields() missing field %q", want)
		}
	}
}

func TestGenerator_Generate_HappyPath(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-test",
		CadocCode: "3050",
		DataBase:  canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "João Silva",
			"telContato": "1199999999",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op1",
				TipoPessoa:     "PJ",
				Indexador:      "CDI",
				Modalidade:     "desDuplicatas",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(50000.0), Moeda: "BRL"},
				TaxaJuros:      decimal.NewFromFloat(0.015), // 1.5% a.m.
			},
			{
				ID:             "op2",
				TipoPessoa:     "PF",
				Indexador:      "PRE",
				Modalidade:     "capGirPrzAte365",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(30000.0), Moeda: "BRL"},
				TaxaJuros:      decimal.NewFromFloat(0.012), // 1.2% a.m.
			},
		},
	}

	result, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.XML) == 0 {
		t.Fatal("Generate() returned empty XML")
	}
	if result.SHA256 == "" {
		t.Error("Generate() returned empty SHA256")
	}
	if result.CadocCode != "3050" {
		t.Errorf("CadocCode = %q, want %q", result.CadocCode, "3050")
	}
}

func TestGenerator_Generate_ValidXMLStructure(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-test",
		CadocCode: "3050",
		DataBase:  canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "João Silva",
			"telContato": "1199999999",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op1",
				TipoPessoa:     "PJ",
				Indexador:      "CDI",
				Modalidade:     "desDuplicatas",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(100000.0), Moeda: "BRL"},
				TaxaJuros:      decimal.NewFromFloat(0.018),
			},
		},
	}

	result, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Parse XML back to verify structure.
	var docTXB DocTXB
	if err := xml.Unmarshal(result.XML, &docTXB); err != nil {
		t.Fatalf("XML unmarshal failed: %v", err)
	}

	if docTXB.CNPJ != "12345678" {
		t.Errorf("CNPJ = %q, want 8-digit root %q", docTXB.CNPJ, "12345678")
	}
	if docTXB.IndRemessa != "I" {
		t.Errorf("IndRemessa = %q, want %q", docTXB.IndRemessa, "I")
	}
	if docTXB.NmContato != "João Silva" {
		t.Errorf("NmContato = %q, want %q", docTXB.NmContato, "João Silva")
	}
	if docTXB.Referencia.Codigo != "202607" {
		t.Errorf("Referencia.Codigo = %q, want %q", docTXB.Referencia.Codigo, "202607")
	}

	// Check that PesJuridica/Flu has a sub-modality.
	pj := docTXB.Referencia.Diario.CRDLivre.PesJuridica
	if len(pj.Flu) == 0 {
		t.Error("PesJuridica.Flu is empty, want at least 1 sub-modalidade")
	}
}

func TestGenerator_Generate_Aggregation(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-test",
		CadocCode: "3050",
		DataBase:  canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "Teste",
			"telContato": "0000000000",
		},
		// 3 operations with same (PJ, CDI) key → should aggregate to 1.
		Operacoes: []canonical.Operacao{
			{TipoPessoa: "PJ", Indexador: "CDI", Modalidade: "desDuplicatas", ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(10000.0), Moeda: "BRL"}, TaxaJuros: decimal.NewFromFloat(0.010)},
			{TipoPessoa: "PJ", Indexador: "CDI", Modalidade: "desDuplicatas", ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(20000.0), Moeda: "BRL"}, TaxaJuros: decimal.NewFromFloat(0.020)},
			{TipoPessoa: "PJ", Indexador: "CDI", Modalidade: "desDuplicatas", ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(30000.0), Moeda: "BRL"}, TaxaJuros: decimal.NewFromFloat(0.030)},
		},
	}

	result, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	var docTXB DocTXB
	if err := xml.Unmarshal(result.XML, &docTXB); err != nil {
		t.Fatalf("XML unmarshal failed: %v", err)
	}

	pj := docTXB.Referencia.Diario.CRDLivre.PesJuridica
	if len(pj.Flu) != 1 {
		t.Errorf("len(PesJuridica.Flu) = %d, want 1 (3 ops should aggregate to 1)", len(pj.Flu))
	}
}

func TestGenerator_Generate_ValidationFailure(t *testing.T) {
	g := New()
	// Missing CNPJ → validation should fail.
	doc := &canonical.CanonicalDocument{
		IFID:      "if-test",
		CadocCode: "3050",
		DataBase:  canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		Header:    canonical.DocumentHeader{CNPJ: ""}, // missing
	}

	_, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Error("Generate() expected error for missing CNPJ, got nil")
	}
}

func TestGenerator_EstimateComplexity(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		Operacoes: []canonical.Operacao{
			{ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(1000.0)}},
			{ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(2000.0)}},
		},
		Participantes: []canonical.Participante{
			{Tipo: "IF", CNPJ: "12345678000123", Nome: "Teste"},
		},
	}

	score := g.EstimateComplexity(doc)
	if score.NumOperacoes != 2 {
		t.Errorf("NumOperacoes = %d, want 2", score.NumOperacoes)
	}
	if score.NumParticipantes != 1 {
		t.Errorf("NumParticipantes = %d, want 1", score.NumParticipantes)
	}
}

func TestTipoCliMap(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"PJ", "PesJuridica"},
		{"PF", "PesFisica"},
		{"pesJuridica", "PesJuridica"},
		{"pesFisica", "PesFisica"},
		{"juridica", "PesJuridica"},
		{"fisica", "PesFisica"},
		{"unknown", "PesJuridica"}, // default
	}
	for _, c := range cases {
		got := tipoCliMap(c.input)
		if got != c.want {
			t.Errorf("tipoCliMap(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestEncargoMap(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"PRE", "Pre"},
		{"CDI", "Flu"},
		{"IPCA", "Flu"},
		{"TJLP", "Flu"},
		{"VC", "Vc"},
		{"câmbio", "Vc"},
		{"IND", "Ind"},
		{"índice", "Ind"},
		{"unknown", "Flu"}, // default
	}
	for _, c := range cases {
		got := encargoMap(c.input)
		if got != c.want {
			t.Errorf("encargoMap(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatTaxa(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0.015, "1.5000"},
		{0.001, "0.1000"},
		{0.0, "0.0000"},
		{0.999, "99.9000"},
		{0, "0.0000"}, // zero guard
	}
	for _, c := range cases {
		got := formatTaxa(c.input)
		if got != c.want {
			t.Errorf("formatTaxa(%f) = %q, want %q", c.input, got, c.want)
		}
	}
}

// TestGenerator_Generate_XMLContainsElement verifies the XML output contains expected elements.
func TestGenerator_Generate_XMLContainsElements(t *testing.T) {
	g := New()
	doc := &canonical.CanonicalDocument{
		IFID:      "if-test",
		CadocCode: "3050",
		DataBase:  canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "Teste",
			"telContato": "0000000000",
		},
		Operacoes: []canonical.Operacao{
			{TipoPessoa: "PJ", Indexador: "PRE", Modalidade: "vendor", ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(50000.0), Moeda: "BRL"}, TaxaJuros: decimal.NewFromFloat(0.025)},
		},
	}

	result, err := g.Generate(context.Background(), doc, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	xmlStr := string(result.XML)
	for _, want := range []string{"DocTXB", "Referencia", "Diario", "PesJuridica", "Pre", "vendor"} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("XML missing expected element/text %q", want)
		}
	}
}
