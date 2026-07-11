// Package gen2060 implementa o CADOCGenerator para o documento 2060 (DRM).
//
// O CADOC 2060 — DRM (Demonstrativo de Risco de Mercado) reporta
// o risco de mercado (VaR, sVaR, RWACOM) ao BACEN.
//
// Referência: BACEN Res. 4.557.
//
// Estrutura:
//   - DocDRM@cnpj, dataBase
//   - RWAJUR1, RWAJUR2, RWAJUR3, RWAJUR4 — RWA por tipo de risco
//   - VaR — Value at Risk
//   - sVaR — Stressed VaR
//   - RWACOM — RWA Commodity
//   - Posicoes (moeda: pares codigo + valor)
package gen2060

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

// Generator implements CADOCGenerator for CADOC 2060 (DRM/Market Risk).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2060" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz (8 dígitos)"},
		{Tag: "rwajur1", Type: "N15,2", Required: true, Desc: "RWA Jur 1 (VaR)"},
		{Tag: "rwajur2", Type: "N15,2", Required: true, Desc: "RWA Jur 2"},
		{Tag: "rwajur3", Type: "N15,2", Required: true, Desc: "RWA Jur 3"},
		{Tag: "rwajur4", Type: "N15,2", Required: true, Desc: "RWA Jur 4"},
		{Tag: "vaR", Type: "N15,2", Required: true, Desc: "Value at Risk"},
		{Tag: "sVaR", Type: "N15,2", Required: true, Desc: "Stressed VaR"},
		{Tag: "rwacom", Type: "N15,2", Required: true, Desc: "RWA Commodity"},
		{Tag: "posicao.moeda", Type: "A3", Required: false, Desc: "Código moeda (ex: USD, EUR)"},
		{Tag: "posicao.valor", Type: "N15,2", Required: false, Desc: "Valor da posição em moeda"},
	}
}

func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	score := 0.3
	if numOp > 30 {
		score += 0.2
	}
	if numOp > 100 {
		score += 0.2
	}
	return generator.ComplexityScore{
		Score:              score,
		NumOperacoes:       numOp,
		NumParticipantes:   len(doc.Participantes),
		EstimatedAPICalls:  0,
		EstimatedTimeMs:    int64(30 + numOp/20),
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

// DocDRM is the root element of the DRM XML.
type DocDRM struct {
	XMLName   xml.Name   `xml:"DocDRM"`
	CNPJ      string     `xml:"cnpj,attr"`
	DataBase  string     `xml:"dataBase,attr"`
	RWAJUR1   ValorSimples `xml:"RWAJUR1,omitempty"`
	RWAJUR2   ValorSimples `xml:"RWAJUR2,omitempty"`
	RWAJUR3   ValorSimples `xml:"RWAJUR3,omitempty"`
	RWAJUR4   ValorSimples `xml:"RWAJUR4,omitempty"`
	VaR       ValorSimples `xml:"VaR,omitempty"`
	SVaR      ValorSimples `xml:"sVaR,omitempty"`
	RWACOM    ValorSimples `xml:"RWACOM,omitempty"`
	Posicoes  []Posicao   `xml:"Posicao,omitempty"`
}

// ValorSimples is a simple element with a valor attribute: <RWAJUR1 valor="100.00"/>
type ValorSimples struct {
	Valor string `xml:"valor,attr"`
}

// Posicao represents a currency position in the DRM.
// Parser expects: <Posicao codigo="161000" moeda="USD" valor="100.00"/>
type Posicao struct {
	Codigo string `xml:"codigo,attr"`
	Moeda  string `xml:"moeda,attr"`
	Valor  string `xml:"valor,attr"`
}

// keyMoeda identifies a currency position.
type keyMoeda struct {
	codigo string
	moeda  string
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDRM {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}

	rwajur1 := parseExtraFloat(doc.Extra["rwajur1"])
	rwajur2 := parseExtraFloat(doc.Extra["rwajur2"])
	rwajur3 := parseExtraFloat(doc.Extra["rwajur3"])
	rwajur4 := parseExtraFloat(doc.Extra["rwajur4"])
	vaR := parseExtraFloat(doc.Extra["vaR"])
	sVaR := parseExtraFloat(doc.Extra["svaR"])
	rwacom := parseExtraFloat(doc.Extra["rwacom"])

	model := DocDRM{
		CNPJ:     cnpj,
		DataBase: dataBase.Format("2006-01-02"),
		RWAJUR1:  ValorSimples{Valor: formatMoney(rwajur1)},
		RWAJUR2:  ValorSimples{Valor: formatMoney(rwajur2)},
		RWAJUR3:  ValorSimples{Valor: formatMoney(rwajur3)},
		RWAJUR4:  ValorSimples{Valor: formatMoney(rwajur4)},
		VaR:      ValorSimples{Valor: formatMoney(vaR)},
		SVaR:     ValorSimples{Valor: formatMoney(sVaR)},
		RWACOM:   ValorSimples{Valor: formatMoney(rwacom)},
	}

	// Build currency positions from operations with currency exposure.
	posMap := groupByMoeda(doc)
	for k, ops := range posMap {
		model.Posicoes = append(model.Posicoes, buildPosicao(k, ops))
	}

	// Sort positions by currency code.
	sort.Slice(model.Posicoes, func(i, j int) bool {
		return model.Posicoes[i].Moeda < model.Posicoes[j].Moeda
	})

	return model
}

func groupByMoeda(doc *canonical.CanonicalDocument) map[keyMoeda][]canonical.Operacao {
	m := make(map[keyMoeda][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		moeda := strVal(op.Extra["moeda"], "BRL")
		codigo := strVal(op.Extra["posicaoCodigo"], op.Indexador)
		if codigo == "" {
			codigo = "1"
		}
		k := keyMoeda{codigo: codigo, moeda: moeda}
		m[k] = append(m[k], op)
	}
	return m
}

func buildPosicao(k keyMoeda, ops []canonical.Operacao) Posicao {
	var total float64
	for _, op := range ops {
		total += decimalToFloat(op.ValorPrincipal.Valor)
	}
	return Posicao{
		Codigo: k.codigo,
		Moeda:  k.moeda,
		Valor:  formatMoney(total),
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

var _ = strconv.ParseFloat
var _ = sort.Slice

// Verify interface.
var _ generator.CADOCGenerator = (*Generator)(nil)
