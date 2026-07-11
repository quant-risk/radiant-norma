// Package validation provides integration tests for all CADOC generators.
// This package is separated to avoid import cycles with the generator package.
package validation

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2030"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2060"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2061"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2062"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2070"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2160"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2170"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen3050"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen4111"
	"github.com/shopspring/decimal"
)

// TestGenerators_All produces valid XML with correct root elements, SHA256, CadocCode,
// DataBase, and FieldMap for all registered generators.
func TestGenerators_All(t *testing.T) {
	testCases := []struct {
		name      string
		gen       generator.CADOCGenerator
		rootTag   string
		cadocCode string
		buildDoc  func() *canonical.CanonicalDocument
	}{
		{
			name:      "3040_SCR",
			gen:       gen3040.New(),
			rootTag:   "Doc3040",
			cadocCode: "3040",
			buildDoc:  build3040Doc,
		},
		{
			name:      "3050_TXB",
			gen:       gen3050.New(),
			rootTag:   "DocTXB",
			cadocCode: "3050",
			buildDoc:  build3050Doc,
		},
		{
			name:      "4111_COSIF",
			gen:       gen4111.New(),
			rootTag:   "Documento4111",
			cadocCode: "4111",
			buildDoc:  build4111Doc,
		},
		{
			name:      "2061_DLO",
			gen:       gen2061.New(),
			rootTag:   "DocDLO",
			cadocCode: "2061",
			buildDoc:  build2061Doc,
		},
		{
			name:      "2062_DLI",
			gen:       gen2062.New(),
			rootTag:   "documentoDLI",
			cadocCode: "2062",
			buildDoc:  build2062Doc,
		},
		{
			name:      "2070_DDR",
			gen:       gen2070.New(),
			rootTag:   "DocDDR",
			cadocCode: "2070",
			buildDoc:  build2070Doc,
		},
		{
			name:      "2160_DRL",
			gen:       gen2160.New(),
			rootTag:   "DocDRL",
			cadocCode: "2160",
			buildDoc:  build2160Doc,
		},
		{
			name:      "2170_DLP",
			gen:       gen2170.New(),
			rootTag:   "DocDLP",
			cadocCode: "2170",
			buildDoc:  build2170Doc,
		},
		{
			name:      "2060_DRM",
			gen:       gen2060.New(),
			rootTag:   "DocDRM",
			cadocCode: "2060",
			buildDoc:  build2060Doc,
		},
		{
			name:      "2030_DRSAC",
			gen:       gen2030.New(),
			rootTag:   "DocDRSAC",
			cadocCode: "2030",
			buildDoc:  build2030Doc,
		},
	}

	dataBase := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.buildDoc()

			// Generate the document.
			result, err := tc.gen.Generate(context.Background(), doc, dataBase)
			if err != nil {
				t.Fatalf("%s: Generate() error = %v", tc.name, err)
			}

			// 1. Verify XML is non-empty.
			if len(result.XML) == 0 {
				t.Errorf("%s: Generate() returned empty XML", tc.name)
				return
			}

			// 2. Verify XML is valid (parseable using a token-based decoder).
			decoder := xml.NewDecoder(bytes.NewReader(result.XML))
			foundRoot := false
			for {
				token, err := decoder.Token()
				if err != nil {
					break
				}
				if se, ok := token.(xml.StartElement); ok {
					if se.Name.Local == tc.rootTag {
						foundRoot = true
					}
				}
			}
			if !foundRoot {
				t.Errorf("%s: XML does not contain valid root element %q", tc.name, tc.rootTag)
				return
			}

			// 3. Verify the XML contains the expected root element.
			xmlStr := string(result.XML)
			if !strings.Contains(xmlStr, tc.rootTag) {
				t.Errorf("%s: XML does not contain expected root element %q.\nXML content:\n%s",
					tc.name, tc.rootTag, xmlStr)
			}

			// 4. Verify SHA256 is computed (64 hex characters).
			if result.SHA256 == "" {
				t.Errorf("%s: SHA256 is empty", tc.name)
			} else if len(result.SHA256) != 64 {
				t.Errorf("%s: SHA256 has invalid length %d, expected 64.\nSHA256: %s",
					tc.name, len(result.SHA256), result.SHA256)
			}

			// 5. Verify CadocCode matches.
			if result.CadocCode != tc.cadocCode {
				t.Errorf("%s: CadocCode = %q, want %q",
					tc.name, result.CadocCode, tc.cadocCode)
			}

			// 6. Verify DataBase is set.
			if result.DataBase.IsZero() {
				t.Errorf("%s: DataBase is zero", tc.name)
			}

			// 7. Verify FieldMap is populated.
			if len(result.FieldMap) == 0 {
				t.Errorf("%s: FieldMap is empty", tc.name)
			}

			// 8. Print the generated XML for inspection.
			t.Logf("%s generated XML:\n%s", tc.name, xmlStr)
			t.Logf("%s SHA256: %s", tc.name, result.SHA256)
			t.Logf("%s FieldMap entries: %d", tc.name, len(result.FieldMap))
		})
	}
}

// TestGenerators_SHA256Uniqueness verifies SHA256 changes when content changes.
func TestGenerators_SHA256Uniqueness(t *testing.T) {
	gen := gen3040.New()
	dataBase := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	doc1 := build3040Doc()
	result1, err := gen.Generate(context.Background(), doc1, dataBase)
	if err != nil {
		t.Fatalf("First Generate() failed: %v", err)
	}

	// Generate same doc again - should have same SHA256 (deterministic).
	result2, err := gen.Generate(context.Background(), doc1, dataBase)
	if err != nil {
		t.Fatalf("Second Generate() failed: %v", err)
	}

	if result1.SHA256 != result2.SHA256 {
		t.Errorf("SHA256 should be deterministic, but got different values:\nSHA256-1: %s\nSHA256-2: %s",
			result1.SHA256, result2.SHA256)
	}

	// Change a value - SHA256 should change.
	doc2 := build3040Doc()
	doc2.Header.CNPJ = "99999999000199" // Different CNPJ
	result3, err := gen.Generate(context.Background(), doc2, dataBase)
	if err != nil {
		t.Fatalf("Third Generate() failed: %v", err)
	}

	if result1.SHA256 == result3.SHA256 {
		t.Errorf("SHA256 should change when document content changes")
	}
}

// TestGenerators_ValidationErrors verifies generators return errors for invalid documents.
func TestGenerators_ValidationErrors(t *testing.T) {
	generators := []struct {
		name string
		gen  generator.CADOCGenerator
	}{
		{"3040", gen3040.New()},
		{"3050", gen3050.New()},
		{"4111", gen4111.New()},
		{"2061", gen2061.New()},
		{"2062", gen2062.New()},
		{"2070", gen2070.New()},
		{"2160", gen2160.New()},
		{"2170", gen2170.New()},
		{"2060", gen2060.New()},
		{"2030", gen2030.New()},
	}

	for _, g := range generators {
		t.Run(g.name, func(t *testing.T) {
			// Empty document should fail validation.
			emptyDoc := &canonical.CanonicalDocument{}
			_, err := g.gen.Generate(context.Background(), emptyDoc, time.Now())
			if err == nil {
				t.Errorf("%s: Generate() expected error for empty document, got nil", g.name)
			}
		})
	}
}

// TestRegistry_Integration tests that generators can be registered and retrieved.
func TestRegistry_Integration(t *testing.T) {
	r := generator.NewRegistry()

	// Register all generators.
	r.Register(gen3040.New())
	r.Register(gen3050.New())
	r.Register(gen4111.New())
	r.Register(gen2061.New())
	r.Register(gen2062.New())
	r.Register(gen2070.New())
	r.Register(gen2160.New())
	r.Register(gen2170.New())
	r.Register(gen2060.New())
	r.Register(gen2030.New())

	// Verify all are registered.
	expected := []string{"3040", "3050", "4111", "2061", "2062", "2070", "2160", "2170", "2060", "2030"}
	for _, code := range expected {
		if !r.IsRegistered(code) {
			t.Errorf("Generator %s not registered", code)
		}
		got := r.Get(code)
		if got == nil {
			t.Errorf("Get(%s) returned nil after registration", code)
		}
		if got.CadocCode() != code {
			t.Errorf("Get(%s).CadocCode() = %q", code, got.CadocCode())
		}
	}

	// Verify list returns all.
	list := r.List()
	if len(list) != len(expected) {
		t.Errorf("List() returned %d generators, want %d", len(list), len(expected))
	}
}

// ---------------------------------------------------------------------------
// Document builders for each generator.
// ---------------------------------------------------------------------------

func build3040Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-3040",
		CadocCode:    "3040",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "3.2",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste SCR S.A.",
		},
		Extra: map[string]any{
			"nomeResp":  "Joao Silva",
			"emailResp": "joao@teste.com.br",
			"telResp":   "11999999999",
			"tpArq":     "F",
			"remessa":   1,
			"parte":     1,
			"natuOp":    "01",
			"origemRec": "1",
			"vincME":    "N",
			"przProvm":  "N",
			"desempOp":  "01",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-3040-1",
				Modalidade:      "1000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(50000.0), Moeda: "BRL"},
				TipoPessoa:      "PJ",
				UF:              "SP",
				ClassificacaoIF: "A",
				NumeroContrato:  "CTR-001",
			},
			{
				ID:              "op-3040-2",
				Modalidade:      "1000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(30000.0), Moeda: "BRL"},
				TipoPessoa:      "PF",
				UF:              "RJ",
				ClassificacaoIF: "B",
				NumeroContrato:  "CTR-002",
			},
		},
		Participantes: []canonical.Participante{
			{
				Tipo: "CLIENTE",
				CNPJ: "12345678000123",
				Nome: "Cliente Teste PJ",
			},
			{
				Tipo: "CLIENTE",
				CNPJ: "12345678900",
				Nome: "Cliente Teste PF",
			},
		},
	}
}

func build3050Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-3050",
		CadocCode:    "3050",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "4.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste TXB S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "Maria Silva",
			"telContato": "21988888888",
			"modalidade": "desDuplicatas",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-3050-1",
				Modalidade:     "desDuplicatas",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(100000.0), Moeda: "BRL"},
				TipoPessoa:     "PJ",
				Indexador:      "CDI",
				TaxaJuros:      decimal.NewFromFloat(0.018), // 1.8% a.m.
			},
			{
				ID:             "op-3050-2",
				Modalidade:     "desDuplicatas",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(50000.0), Moeda: "BRL"},
				TipoPessoa:     "PF",
				Indexador:      "PRE",
				TaxaJuros:      decimal.NewFromFloat(0.015), // 1.5% a.m.
			},
		},
	}
}

func build4111Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-4111",
		CadocCode:    "4111",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste COSIF S.A.",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:             "op-4111-1",
				Modalidade:     "2100",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(200000.0), Moeda: "BRL"},
				TipoPessoa:     "PJ",
				UF:             "SP",
				NivelRisco:     "B",
				Extra: map[string]any{
					"cnpj": "12345678000123",
				},
			},
			{
				ID:             "op-4111-2",
				Modalidade:     "2100",
				ValorPrincipal: canonical.Money{Valor: decimal.NewFromFloat(50000.0), Moeda: "BRL"},
				TipoPessoa:     "PF",
				UF:             "RJ",
				NivelRisco:     "C",
			},
		},
	}
}

func build2061Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2061",
		CadocCode:    "2061",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DLO S.A.",
		},
		Extra: map[string]any{
			"conta770":    "1000000.00",
			"limiteTotal": "5000000.00",
			"patrimonio":  "3000000.00",
			"tpDocumento": "F",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2061-1",
				Modalidade:      "800.01",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(500000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DLO-001",
				Extra: map[string]any{
					"contaCosif": "800.01",
				},
			},
		},
	}
}

func build2062Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2062",
		CadocCode:    "2062",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DLI S.A.",
		},
		Extra: map[string]any{
			"tipoEnvio":    "I",
			"parametro_31": "100",
			"parametro_32": "200",
			"parametro_33": "300",
			"limites": map[string]any{
				"01.00": "500000.00",
				"02.00": "750000.00",
			},
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2062-1",
				Modalidade:      "6.00.00",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(100000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DLI-001",
				Extra: map[string]any{
					"contaCosif": "6.00.00",
				},
			},
		},
	}
}

func build2070Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2070",
		CadocCode:    "2070",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DDR S.A.",
		},
		Extra: map[string]any{
			"indRemessa": "I",
			"nmContato":  "Carlos Silva",
			"telContato": "31977777777",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2070-1",
				Modalidade:      "161000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(1000000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DDR-001",
				Extra: map[string]any{
					"ddrCodigo": "161000",
					"moeda":     "BRL",
				},
			},
			{
				ID:              "op-2070-2",
				Modalidade:      "181000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(500000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DDR-002",
				Extra: map[string]any{
					"ddrCodigo": "181000",
					"moeda":     "USD",
				},
			},
		},
	}
}

func build2160Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2160",
		CadocCode:    "2160",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DRL S.A.",
		},
		Extra: map[string]any{
			"tpDocumento":  "F",
			"numeroVersao": 1,
			"hqla":         5000000.00,
			"outflows":     3000000.00,
			"inflows":      1000000.00,
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2160-1",
				Modalidade:      "10.01",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(2000000.0), Moeda: "BRL"},
				Indexador:       "CDI",
				NumeroContrato:  "CTR-DRL-001",
				Extra: map[string]any{
					"contaCosif": "10.01",
				},
			},
		},
	}
}

func build2170Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2170",
		CadocCode:    "2170",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DLP S.A.",
		},
		Extra: map[string]any{
			"tpDocumento":  "F",
			"numeroVersao": 1,
			"asfTotal":     8000000.00,
			"rsfTotal":     6000000.00,
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2170-1",
				Modalidade:      "30.01",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(3000000.0), Moeda: "BRL"},
				TipoOperacao:    "DEP",
				NumeroContrato:  "CTR-DLP-001",
				Extra: map[string]any{
					"contaCosif": "30.01",
				},
			},
		},
	}
}

func build2060Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2060",
		CadocCode:    "2060",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DRM S.A.",
		},
		Extra: map[string]any{
			"rwajur1": 1000000.00,
			"rwajur2": 800000.00,
			"rwajur3": 500000.00,
			"rwajur4": 300000.00,
			"vaR":     200000.00,
			"svaR":    250000.00,
			"rwacom":  100000.00,
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2060-1",
				Modalidade:      "moeda",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(500000.0), Moeda: "BRL"},
				Indexador:       "USD",
				NumeroContrato:  "CTR-DRM-001",
				Extra: map[string]any{
					"moeda":         "USD",
					"posicaoCodigo": "1",
				},
			},
		},
	}
}

func build2030Doc() *canonical.CanonicalDocument {
	return &canonical.CanonicalDocument{
		IFID:         "if-test-2030",
		CadocCode:    "2030",
		DataBase:     canonical.DataBase(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
		VersaoLayout: "1.0",
		Header: canonical.DocumentHeader{
			CNPJ:   "12345678000123",
			NomeIF: "Banco Teste DRSAC S.A.",
		},
		Extra: map[string]any{
			"tpDocumento": "F",
			"subsegmento": "S3",
		},
		Operacoes: []canonical.Operacao{
			{
				ID:              "op-2030-1",
				Modalidade:      "1000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(10000000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DRSAC-001",
				Extra: map[string]any{
					"participanteCnpj": "98765432000199",
				},
			},
			{
				ID:              "op-2030-2",
				Modalidade:      "1000",
				ValorPrincipal:  canonical.Money{Valor: decimal.NewFromFloat(5000000.0), Moeda: "BRL"},
				NumeroContrato:  "CTR-DRSAC-002",
				Extra: map[string]any{
					"participanteCnpj": "11122233000155",
				},
			},
		},
		Participantes: []canonical.Participante{
			{
				Tipo: "CLIENTE",
				CNPJ: "98765432000199",
				Nome: "Cliente Grande",
				Ratings: []canonical.Rating{
					{NomeAgencia: "S&P", Nota: "AAA"},
				},
			},
			{
				Tipo: "CLIENTE",
				CNPJ: "11122233000155",
				Nome: "Cliente Medio",
			},
		},
	}
}
