// Package gen2061 implementa o CADOCGenerator para o documento 2061 (DLO).
//
// O CADOC 2061 — DLO (Demonstrativo de Limites Operacionais) reporta
// limites operacionais e composition patrimonial ao BACEN.
//
// Estrutura:
//   - DocDLO@cnpj, dataBase, tpDocumento, numeroVersao
//   - Conta770, LimiteTotal, Patrimonio
//   - Conta@codigoConta, valor (cada conta COSIF)
//   - Elem@codigoElem, descElem, valor, peso
//
// O DLO funciona como demonstrativo de limites: each Conta groups Elem elements
// with peso (ponderação) for RWACAM calculations.
package gen2061

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

// Generator implements CADOCGenerator for CADOC 2061 (DLO).
type Generator struct{}

// New creates a new DLO generator.
func New() *Generator { return &Generator{} }

// CadocCode returns "2061".
func (g *Generator) CadocCode() string { return "2061" }

// SupportedVersions returns supported layout versions.
func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "2.0"}
}

// RequiredFields returns the required fields for DLO.
func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz da IF (8 dígitos)"},
		{Tag: "tpDocumento", Type: "A1", Required: true, Desc: "Tipo documento (F=full, S=substituição)"},
		{Tag: "numeroVersao", Type: "N3", Required: true, Desc: "Número da versão"},
		{Tag: "conta770", Type: "N15,2", Required: true, Desc: "Conta 770 (RWACAM)"},
		{Tag: "limiteTotal", Type: "N15,2", Required: true, Desc: "Limite total operacional"},
		{Tag: "patrimonio", Type: "N15,2", Required: true, Desc: "Patrimônio"},
		{Tag: "conta.codigoConta", Type: "A15", Required: false, Desc: "Código conta COSIF"},
		{Tag: "conta.valor", Type: "N15,2", Required: false, Desc: "Valor da conta COSIF"},
	}
}

// EstimateComplexity evaluates document complexity.
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
		EstimatedTimeMs:   int64(40 + numOp/20),
	}
}

// Generate produces the DLO XML from the CanonicalDocument.
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

// DocDLO is the root element of the DLO XML.
type DocDLO struct {
	XMLName      xml.Name     `xml:"DocDLO"`
	CNPJ         string       `xml:"cnpj,attr"`
	DataBase     string       `xml:"dataBase,attr"`
	TpDocumento  string       `xml:"tpDocumento,attr"`
	NumeroVersao string       `xml:"numeroVersao,attr"`
	Conta770     ValorSimples `xml:"Conta770,omitempty"`
	LimiteTotal  ValorSimples `xml:"LimiteTotal,omitempty"`
	Patrimonio   ValorSimples `xml:"Patrimonio,omitempty"`
	Contas       []Conta      `xml:"Conta,omitempty"`
	Totalizador  *Totaliz     `xml:"Totalizador,omitempty"`
}

// ValorSimples is a simple element with a valor attribute: <Conta770 valor="1000.00"/>
type ValorSimples struct {
	Valor string `xml:"valor,attr"`
}

// Conta represents a COSIF account in DLO.
type Conta struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
	Elems       []Elem `xml:"Elem,omitempty"`
}

// Elem represents a detail element within a Conta.
type Elem struct {
	CodigoElem string `xml:"codigoElem,attr"`
	DescElem   string `xml:"descElem,attr"`
	Valor      string `xml:"valor,attr"`
	Peso       string `xml:"peso,attr,omitempty"`
}

// Totaliz is the DLO totals block.
type Totaliz struct {
	ContasTotal     string `xml:"contasTotal,attr"`
	RWACAMTotal     string `xml:"rwacamTotal,attr"`
	PatrimonioTotal string `xml:"patrimonioTotal,attr"`
}

// buildModel transforms the CanonicalDocument into DLO XML model.
func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDLO {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	tpDoc := strVal(doc.Extra["tpDocumento"], "F")
	numVer := fmt.Sprintf("%d", max(1, intVal(doc.Extra["numeroVersao"])))

	conta770 := strVal(doc.Extra["conta770"], "0.00")
	limiteTotal := strVal(doc.Extra["limiteTotal"], "0.00")
	patrimonio := strVal(doc.Extra["patrimonio"], "0.00")

	model := DocDLO{
		CNPJ:         cnpj,
		DataBase:     dataBase.Format("2006-01-02"),
		TpDocumento:  tpDoc,
		NumeroVersao: numVer,
		Conta770:     ValorSimples{Valor: conta770},
		LimiteTotal:  ValorSimples{Valor: limiteTotal},
		Patrimonio:   ValorSimples{Valor: patrimonio},
	}

	// Aggregate accounts from operations.
	acctMap := groupByConta(doc)
	for acctCode, ops := range acctMap {
		model.Contas = append(model.Contas, buildConta(acctCode, ops))
	}

	if len(model.Contas) > 0 {
		model.Totalizador = buildTotalizador(model.Contas, patrimonio)
	}

	return model
}

func groupByConta(doc *canonical.CanonicalDocument) map[string][]canonical.Operacao {
	m := make(map[string][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		// Use Extra["contaCosif"] or fallback to modality.
		code := strVal(op.Extra["contaCosif"], op.Modalidade)
		if code == "" {
			code = "000.00"
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
		Elems:       buildElems(ops),
	}
}

func buildElems(ops []canonical.Operacao) []Elem {
	// Build one Elem per unique contract or default.
	seen := make(map[string]bool)
	var elems []Elem
	for _, op := range ops {
		if op.NumeroContrato != "" && !seen[op.NumeroContrato] {
			seen[op.NumeroContrato] = true
			elems = append(elems, Elem{
				CodigoElem: "1",
				DescElem:   "Principal",
				Valor:      formatMoney(decimalToFloat(op.ValorPrincipal.Valor)),
				Peso:       "1.00",
			})
		}
	}
	if len(elems) == 0 {
		elems = append(elems, Elem{
			CodigoElem: "1",
			DescElem:   "Saldo",
			Valor:      "0.00",
			Peso:       "1.00",
		})
	}
	return elems
}

func buildTotalizador(contas []Conta, patrimonio string) *Totaliz {
	var total float64
	for _, c := range contas {
		v, _ := strconv.ParseFloat(c.Valor, 64)
		total += v
	}
	return &Totaliz{
		ContasTotal:     fmt.Sprintf("%d", len(contas)),
		RWACAMTotal:     formatMoney(total),
		PatrimonioTotal: patrimonio, // use explicit patrimonio, not sum of contas
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
	for _, op := range doc.Operacoes {
		add("contaCosif", "Conta@codigoConta", op.Modalidade, canonical.FontSourceManual)
	}
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
