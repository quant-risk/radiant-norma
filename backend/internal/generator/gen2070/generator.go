// Package gen2070 implementa o CADOCGenerator para o documento 2070 (DDR).
//
// O CADOC 2070 — DDR (Requerimento de Capital Diário) reporta o capital
// regulatório diário ao BACEN.
//
// Estrutura:
//   - DocDDR@cnpj, dataBase, indRemessa, nmContato, telContato
//   - DDR@codigo, moeda, valor (cada posição)
//
// CadocCode: "2070" | COSIF accounts: 161000, 181000, 710000, etc.
package gen2070

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/canonical"
	"github.com/fortvna/radiant-norma/backend/internal/generator"
	"github.com/fortvna/radiant-norma/backend/internal/schema"
	"github.com/shopspring/decimal"
)

// Generator implements CADOCGenerator for CADOC 2070 (DDR).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2070" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz (8 dígitos)"},
		{Tag: "indRemessa", Type: "A1", Required: true, Desc: "Indicador remessa (I/A/S)"},
		{Tag: "nmContato", Type: "A50", Required: true, Desc: "Nome do contato responsável"},
		{Tag: "telContato", Type: "A15", Required: true, Desc: "Telefone do contato"},
		{Tag: "ddr.codigo", Type: "N6", Required: true, Desc: "Código exposição (ex: 161000)"},
		{Tag: "ddr.moeda", Type: "A3", Required: true, Desc: "Código moeda (ex: BRL, USD)"},
		{Tag: "ddr.valor", Type: "N15,2", Required: true, Desc: "Valor da posição"},
	}
}

func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	score := 0.3
	if numOp > 20 {
		score += 0.2
	}
	if numOp > 100 {
		score += 0.2
	}
	return generator.ComplexityScore{
		Score:             score,
		NumOperacoes:      numOp,
		NumParticipantes:  len(doc.Participantes),
		EstimatedAPICalls: 0,
		EstimatedTimeMs:   int64(25 + numOp/20),
	}
}

// RootTag returns the canonical root tag for 2070.
func (g *Generator) RootTag() string {
	return "documentoDDR"
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

// DocDDR is the root element of the DDR XML (CADOC 2070).
type DocDDR struct {
	XMLName    xml.Name `xml:"DocDDR"`
	CNPJ       string   `xml:"cnpj,attr"`
	DataBase   string   `xml:"dataBase,attr"`
	IndRemessa string   `xml:"indRemessa,attr"`
	NmContato  string   `xml:"nmContato,attr"`
	TelContato string   `xml:"telContato,attr"`
	DDRs       []DDR    `xml:"DDR,omitempty"`
}

// DDR represents a DDR (Requerimento de Capital) entry.
type DDR struct {
	Codigo string  `xml:"codigo,attr"`
	Moeda  string  `xml:"moeda,attr"`
	Valor  *string `xml:"valor,attr,omitempty"`
}

// keyDDR identifies a DDR entry by code + moeda.
type keyDDR struct {
	code  string
	moeda string
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDDR {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	indRemessa := strVal(doc.Extra["indRemessa"], "I")
	nmContato := strVal(doc.Extra["nmContato"], "CONTATO")
	telContato := strVal(doc.Extra["telContato"], "0000000000")

	model := DocDDR{
		CNPJ:       cnpj,
		DataBase:   dataBase.Format("2006-01-02"),
		IndRemessa: indRemessa,
		NmContato:  nmContato,
		TelContato: telContato,
	}

	// Aggregate DDRs by code + moeda.
	ddrMap := groupByDDR(doc)
	for k, ops := range ddrMap {
		model.DDRs = append(model.DDRs, buildDDR(k, ops))
	}

	// Sort DDRs by code for deterministic output.
	sort.Slice(model.DDRs, func(i, j int) bool {
		return model.DDRs[i].Codigo < model.DDRs[j].Codigo
	})

	return model
}

func groupByDDR(doc *canonical.CanonicalDocument) map[keyDDR][]canonical.Operacao {
	m := make(map[keyDDR][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		code := strVal(op.Extra["ddrCodigo"], op.Modalidade)
		if code == "" {
			code = "161000" // default: operações de crédito
		}
		moeda := extraStr(op.Extra, "moeda", extraStr(op.Extra, "ddrMoeda", "BRL"))
		k := keyDDR{code: code, moeda: moeda}
		m[k] = append(m[k], op)
	}
	return m
}

func buildDDR(k keyDDR, ops []canonical.Operacao) DDR {
	var total float64
	for _, op := range ops {
		total += decimalToFloat(op.ValorPrincipal.Valor)
	}
	valStr := formatMoney(total)
	return DDR{
		Codigo: k.code,
		Moeda:  k.moeda,
		Valor:  &valStr,
	}
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

// extraStr safely extracts a string from a map[string]any.
func extraStr(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
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
