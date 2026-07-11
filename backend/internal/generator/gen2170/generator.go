// Package gen2170 implementa o CADOCGenerator para o documento 2170 (DLP).
//
// O CADOC 2170 — DLP (Demonstrativo de Liquidez de Longo Prazo) reporta
// o NSFR (Net Stable Funding Ratio) ao BACEN.
//
// Referência: BACEN Res. 4.542.
//
// Estrutura:
//   - DocDLP@cnpj, dataBase, tpDocumento, numeroVersao
//   - ASFTotal — Available Stable Funding total
//   - RSFTotal — Required Stable Funding total
//   - NSFRRatio — ratio ASF / RSF (%)
//   - Cenario (cenários de estresse 1, 2)
package gen2170

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

// Generator implements CADOCGenerator for CADOC 2170 (DLP/NSFR).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2170" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz (8 dígitos)"},
		{Tag: "tpDocumento", Type: "A1", Required: true, Desc: "Tipo documento (F/S)"},
		{Tag: "numeroVersao", Type: "N3", Required: true, Desc: "Número versão"},
		{Tag: "asfTotal", Type: "N15,2", Required: true, Desc: "Available Stable Funding total"},
		{Tag: "rsfTotal", Type: "N15,2", Required: true, Desc: "Required Stable Funding total"},
		{Tag: "nsfrRatio", Type: "N10,4", Required: true, Desc: "NSFR Ratio (%)"},
		{Tag: "cenario.id", Type: "N1", Required: false, Desc: "ID do cenário (1/2)"},
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

// DocDLP is the root element of the DLP/NSFR XML.
type DocDLP struct {
	XMLName      xml.Name     `xml:"DocDLP"`
	CNPJ         string       `xml:"cnpj,attr"`
	DataBase     string       `xml:"dataBase,attr"`
	TpDocumento  string       `xml:"tpDocumento,attr"`
	NumeroVersao string       `xml:"numeroVersao,attr"`
	ASFTotal     ValorSimples `xml:"ASFTotal,omitempty"`
	RSFTotal     ValorSimples `xml:"RSFTotal,omitempty"`
	NSFRRatio    ValorSimples `xml:"NSFRRatio,omitempty"`
	Cenarios     []Cenario    `xml:"Cenario,omitempty"`
	Contas       []Conta      `xml:"Conta,omitempty"`
}

// ValorSimples is a simple element with a valor attribute: <ASFTotal valor="5000.00"/>
type ValorSimples struct {
	Valor string `xml:"valor,attr"`
}

// Conta represents a COSIF account line in DLP.
type Conta struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
}

// Cenario represents a stress scenario for NSFR.
type Cenario struct {
	ID        string       `xml:"id,attr"`
	ASF       ValorSimples `xml:"ASF,omitempty"`
	RSF       ValorSimples `xml:"RSF,omitempty"`
	NSFRRatio ValorSimples `xml:"NSFRRatio,omitempty"`
}

// calculateNSFR computes NSFR = ASF / RSF * 100.
func calculateNSFR(asf, rsf float64) float64 {
	if rsf <= 0 {
		return 0
	}
	return (asf / rsf) * 100
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDLP {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	tpDoc := strVal(doc.Extra["tpDocumento"], "F")
	numVer := fmt.Sprintf("%d", max(1, intVal(doc.Extra["numeroVersao"])))

	asfTotal := parseExtraFloat(doc.Extra["asfTotal"])
	rsfTotal := parseExtraFloat(doc.Extra["rsfTotal"])
	nsfrRatio := calculateNSFR(asfTotal, rsfTotal)

	model := DocDLP{
		CNPJ:         cnpj,
		DataBase:     dataBase.Format("2006-01-02"),
		TpDocumento:  tpDoc,
		NumeroVersao: numVer,
		ASFTotal:     ValorSimples{Valor: formatMoney(asfTotal)},
		RSFTotal:     ValorSimples{Valor: formatMoney(rsfTotal)},
		NSFRRatio:    ValorSimples{Valor: formatRatio(nsfrRatio)},
	}

	// Build contas from operations (COSIF accounts 30.x = ASF, 40.x = RSF).
	acctMap := groupByConta(doc)
	for code, ops := range acctMap {
		model.Contas = append(model.Contas, buildConta(code, ops))
	}

	// Build cenários de estresse (2 standard scenarios).
	model.Cenarios = buildCenarios(doc, asfTotal, rsfTotal)

	return model
}

func groupByConta(doc *canonical.CanonicalDocument) map[string][]canonical.Operacao {
	m := make(map[string][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		code := strVal(op.Extra["contaCosif"], op.Modalidade)
		if code == "" {
			// ASF = 30.x (funding), RSF = 40.x (assets)
			switch op.TipoOperacao {
			case "D", "DEP":
				code = "30.01" // ASF: depósitos
			default:
				code = "40.01" // RSF: operações ativas
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

func buildCenarios(doc *canonical.CanonicalDocument, baseASF, baseRSF float64) []Cenario {
	// Two standard NSFR stress scenarios.
	scenarios := []struct {
		id        string
		asfFactor float64
		rsfFactor float64
	}{
		{"1", 0.85, 1.15}, // mild stress
		{"2", 0.70, 1.30}, // severe stress
	}

	var casos []Cenario
	for _, s := range scenarios {
		scASF := baseASF * s.asfFactor
		scRSF := baseRSF * s.rsfFactor
		casos = append(casos, Cenario{
			ID:        s.id,
			ASF:       ValorSimples{Valor: formatMoney(scASF)},
			RSF:       ValorSimples{Valor: formatMoney(scRSF)},
			NSFRRatio: ValorSimples{Valor: formatRatio(calculateNSFR(scASF, scRSF))},
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
