// Package gen2160 implementa o CADOCGenerator para o documento 2160 (DRL).
//
// O CADOC 2160 — DRL (Demonstrativo de Liquidez) reporta o LCR
// (Liquidity Coverage Ratio) ao BACEN.
//
// Referência: BACEN Res. 4.605.
//
// Estrutura:
//   - DocDRL@cnpj, dataBase, tpDocumento, numeroVersao
//   - HQLA (High Quality Liquid Assets) — 10.x + 11.x
//   - Outflows — saídas em 30 dias — 20.x + 21.x
//   - Inflows — entradas em 30 dias — 30.x + 31.x
//   - LCRRatio — ratio HQLA / (Outflows - Inflows)
//   - Cenario (cenários de estresse 1, 2, 3)
package gen2160

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/shopspring/decimal"
)

// Generator implements CADOCGenerator for CADOC 2160 (DRL/LCR).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2160" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz (8 dígitos)"},
		{Tag: "tpDocumento", Type: "A1", Required: true, Desc: "Tipo documento (F/S)"},
		{Tag: "numeroVersao", Type: "N3", Required: true, Desc: "Número versão"},
		{Tag: "hqla", Type: "N15,2", Required: true, Desc: "High Quality Liquid Assets"},
		{Tag: "outflows", Type: "N15,2", Required: true, Desc: "Saídas de caixa 30 dias"},
		{Tag: "inflows", Type: "N15,2", Required: true, Desc: "Entradas de caixa 30 dias"},
		{Tag: "lcrRatio", Type: "N10,4", Required: true, Desc: "LCR Ratio (%)"},
		{Tag: "cenario.id", Type: "N1", Required: false, Desc: "ID do cenário (1/2/3)"},
	}
}

func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	score := 0.3
	if numOp > 50 {
		score += 0.2
	}
	if numOp > 200 {
		score += 0.2
	}
	return generator.ComplexityScore{
		Score:             score,
		NumOperacoes:      numOp,
		NumParticipantes:  len(doc.Participantes),
		EstimatedAPICalls: 0,
		EstimatedTimeMs:   int64(35 + numOp/20),
	}
}

// RootTag returns the canonical root tag for 2160.
func (g *Generator) RootTag() string {
	return "documentoDRL"
}

func (g *Generator) Generate(ctx context.Context, doc *canonical.CanonicalDocument, dataBase time.Time) (*generator.GeneratedDoc, error) {
	start := time.Now()
	if errs := doc.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("document validation failed: %v", errs)
	}
	model := buildModel(doc, dataBase)
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(model); err != nil {
		return nil, fmt.Errorf("XML encoding failed: %w", err)
	}
	return &generator.GeneratedDoc{
		XML:          buf.Bytes(),
		SHA256:       sha256Hex(buf.Bytes()),
		CadocCode:    g.CadocCode(),
		VersaoLayout: doc.VersaoLayout,
		DataBase:     dataBase,
		FieldMap:     buildFieldMap(doc),
		Metadata: generator.GenMetadata{
			GeneratorVersion: "1.0.0",
			GeneratedAt:      start,
			DurationMs:       time.Since(start).Milliseconds(),
			SourceAdapter:    doc.Metadata.SourceAdapter,
		},
	}, nil
}

// DocDRL is the root element of the DRL/LCR XML.
type DocDRL struct {
	XMLName      xml.Name     `xml:"DocDRL"`
	CNPJ         string       `xml:"cnpj,attr"`
	DataBase     string       `xml:"dataBase,attr"`
	TpDocumento  string       `xml:"tpDocumento,attr"`
	NumeroVersao string       `xml:"numeroVersao,attr"`
	HQLA         ValorSimples `xml:"HQLA,omitempty"`
	Outflows     ValorSimples `xml:"Outflows,omitempty"`
	Inflows      ValorSimples `xml:"Inflows,omitempty"`
	LCRRatio     ValorSimples `xml:"LCRRatio,omitempty"`
	Cenarios     []Cenario    `xml:"Cenario,omitempty"`
	Contas       []Conta      `xml:"Conta,omitempty"`
}

// ValorSimples is a simple element with a valor attribute: <HQLA valor="1000.00"/>
type ValorSimples struct {
	Valor string `xml:"valor,attr"`
}

// Conta represents a COSIF account line in DRL.
type Conta struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
}

// Cenario represents a stress scenario for LCR.
type Cenario struct {
	ID       string       `xml:"id,attr"`
	HQLA     ValorSimples `xml:"HQLA,omitempty"`
	Outflows ValorSimples `xml:"Outflows,omitempty"`
	Inflows  ValorSimples `xml:"Inflows,omitempty"`
	LCRRatio ValorSimples `xml:"LCRRatio,omitempty"`
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDRL {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	tpDoc := strVal(doc.Extra["tpDocumento"], "F")
	numVer := fmt.Sprintf("%d", max(1, intVal(doc.Extra["numeroVersao"])))

	hqla := parseExtraFloat(doc.Extra["hqla"])
	outflows := parseExtraFloat(doc.Extra["outflows"])
	inflows := parseExtraFloat(doc.Extra["inflows"])
	lcrRatio := calculateLCR(hqla, outflows, inflows)

	model := DocDRL{
		CNPJ:         cnpj,
		DataBase:     dataBase.Format("2006-01-02"),
		TpDocumento:  tpDoc,
		NumeroVersao: numVer,
		HQLA:         ValorSimples{Valor: formatMoney(hqla)},
		Outflows:     ValorSimples{Valor: formatMoney(outflows)},
		Inflows:      ValorSimples{Valor: formatMoney(inflows)},
		LCRRatio:     ValorSimples{Valor: formatRatio(lcrRatio)},
	}

	// Build contas from operations (COSIF accounts 10.x, 20.x, 30.x).
	acctMap := groupByConta(doc)
	for code, ops := range acctMap {
		model.Contas = append(model.Contas, buildConta(code, ops))
	}

	// Build cenários de estresse (1, 2, 3).
	model.Cenarios = buildCenarios(doc, hqla, outflows, inflows)

	return model
}

// calculateLCR computes LCR = HQLA / (Outflows - Inflows) * 100.
func calculateLCR(hqla, outflows, inflows float64) float64 {
	net := outflows - inflows
	if net <= 0 {
		return 0
	}
	return (hqla / net) * 100
}

func groupByConta(doc *canonical.CanonicalDocument) map[string][]canonical.Operacao {
	m := make(map[string][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		code := strVal(op.Extra["contaCosif"], op.Modalidade)
		if code == "" {
			// Infer from indexador or tipo de operação.
			switch op.Indexador {
			case "CDI", "IPCA", "PRE", "FLU":
				code = "10.01" // HQLA default
			default:
				code = "10.01"
			}
		}
		m[code] = append(m[code], op)
	}
	return m
}

func buildConta(code string, ops []canonical.Operacao) Conta {
	var total float64
	for _, op := range ops {
		total += decimalToFloat(op.ValorPrincipal.Valor)
	}
	return Conta{
		CodigoConta: code,
		Valor:       formatMoney(total),
	}
}

func buildCenarios(doc *canonical.CanonicalDocument, baseHQLA, baseOutflows, baseInflows float64) []Cenario {
	// Three standard scenarios with stress factors.
	// Cenario 1: mild stress (0.9 HQLA, 1.1 outflows)
	// Cenario 2: moderate stress (0.8 HQLA, 1.25 outflows)
	// Cenario 3: severe stress (0.7 HQLA, 1.5 outflows)
	scenarios := []struct {
		id         string
		hqlaFactor float64
		outFactor  float64
	}{
		{"1", 0.90, 1.10},
		{"2", 0.80, 1.25},
		{"3", 0.70, 1.50},
	}

	var casos []Cenario
	for _, s := range scenarios {
		scHQLA := baseHQLA * s.hqlaFactor
		scOut := baseOutflows * s.outFactor
		scIn := baseInflows * 0.9 // inflows reduce under stress
		casos = append(casos, Cenario{
			ID:       s.id,
			HQLA:     ValorSimples{Valor: formatMoney(scHQLA)},
			Outflows: ValorSimples{Valor: formatMoney(scOut)},
			Inflows:  ValorSimples{Valor: formatMoney(scIn)},
			LCRRatio: ValorSimples{Valor: formatRatio(calculateLCR(scHQLA, scOut, scIn))},
		})
	}
	return casos
}

func buildFieldMap(doc *canonical.CanonicalDocument) []canonical.FieldMapping {
	var fm []canonical.FieldMapping
	add := func(cosif, xmlTag, val string, fonte canonical.FieldSource) {
		fm = append(fm, canonical.FieldMapping{
			CampoCOSIF:     cosif,
			CampoXML:       xmlTag,
			ValorFormatado: val,
			Fonte:          fonte,
		})
	}
	add("dataBase", "dataBase", time.Time(doc.DataBase).Format("2006-01-02"), canonical.FontSourceManual)
	add("cnpj", "cnpj", doc.Header.CNPJ, canonical.FontSourceManual)
	return fm
}

// Helpers.
func strVal(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

func intVal(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case string:
		var i int
		fmt.Sscanf(n, "%d", &i)
		return i
	}
	return 0
}

func parseExtraFloat(v any) float64 {
	switch f := v.(type) {
	case float64:
		return f
	case int:
		return float64(f)
	case decimal.Decimal:
		f64, _ := strconv.ParseFloat(f.String(), 64)
		return f64
	case string:
		if f == "" {
			return 0
		}
		val, _ := strconv.ParseFloat(f, 64)
		return val
	}
	return 0
}

func decimalToFloat(d any) float64 {
	switch f := d.(type) {
	case float64:
		return f
	case decimal.Decimal:
		f64, _ := strconv.ParseFloat(f.String(), 64)
		return f64
	}
	return 0.0
}

func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

func formatRatio(v float64) string {
	return fmt.Sprintf("%.4f", v)
}

func cleanDigits(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	return string(out)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// Verify interface.
var _ generator.CADOCGenerator = (*Generator)(nil)

// Verify interface.
