// Package gen2062 implementa o CADOCGenerator para o documento 2062 (DLI).
//
// O CADOC 2062 — DLI (Demonstrativo de Limites Operacionais) reporta
// limites operacionais por tipo ao BACEN.
//
// Estrutura:
//   - documentoDLI@cnpj, dataBase, codigoDocumento="2062", tipoEnvio
//   - limitesInformados / limite@codigoLimite, enviado (S/N)
//   - parametros / parametro@codigo (31, 32, 33)
//   - contas / conta@codigoConta, valor
//
// COSIF: 6.x (PLA), 8.x (Capital), 9.x (Captação), 20-22.x (Partes Relacionadas),
// 34-36.x (TVM), 38-39.x (Fomento), 56-58.x (SCM), 76-78.x (Cooperativas).
package gen2062

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

// Generator implements CADOCGenerator for CADOC 2062 (DLI).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2062" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "2.0", "3.0"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A7", Required: true, Desc: "Data-base (AAAA-MM)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz da IF (8 dígitos)"},
		{Tag: "codigoDocumento", Type: "A4", Required: true, Desc: "Código documento (2062)"},
		{Tag: "tipoEnvio", Type: "A1", Required: true, Desc: "Tipo envio (I=inclusão, S=substituição)"},
		{Tag: "limite.codigoLimite", Type: "A5", Required: false, Desc: "Código do limite (ex: 06.00, 20.00)"},
		{Tag: "limite.enviado", Type: "A1", Required: false, Desc: "Se limite foi informado (S/N)"},
		{Tag: "parametro.codigo", Type: "N2", Required: false, Desc: "Código parâmetro (31, 32, 33)"},
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

// DocumentoDLI is the root element.
type DocumentoDLI struct {
	XMLName    xml.Name    `xml:"documentoDLI"`
	CNPJ       string      `xml:"cnpj,attr"`
	DataBase   string      `xml:"dataBase,attr"`
	CodigoDoc  string      `xml:"codigoDocumento,attr"`
	TipoEnvio  string      `xml:"tipoEnvio,attr"`
	Limites    []Limite    `xml:"limitesInformados>limite,omitempty"`
	Parametros []Parametro `xml:"parametros>parametro,omitempty"`
	Contas     []Conta     `xml:"contas>conta,omitempty"`
}

// Limite represents a reported operational limit.
type Limite struct {
	Codigo  string `xml:"codigoLimite,attr"`
	Enviado string `xml:"enviado,attr"`
	Valor   string `xml:",chardata"`
}

// Parametro represents a parameter from responsible.
type Parametro struct {
	Codigo string `xml:"codigo,attr"`
	Valor  string `xml:",chardata"`
}

// Conta represents a COSIF account in DLI.
type Conta struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
}

// keyDLI identifies a DLI limit by its code.
type keyDLI struct {
	codigoLimite string
	tipo         string
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocumentoDLI {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	tipoEnvio := strVal(doc.Extra["tipoEnvio"], "I")

	model := DocumentoDLI{
		CNPJ:      cnpj,
		DataBase:  dataBase.Format("2006-01"),
		CodigoDoc: "2062",
		TipoEnvio: tipoEnvio,
	}

	// Build limites from Extra["limites"] map or default.
	limites := buildLimites(doc)
	model.Limites = limites

	// Build parametros.
	model.Parametros = buildParametros(doc)

	// Build contas from operations.
	acctMap := groupByConta(doc)
	for code, ops := range acctMap {
		model.Contas = append(model.Contas, buildConta(code, ops))
	}

	return model
}

func buildLimites(doc *canonical.CanonicalDocument) []Limite {
	// Standard DLI limit codes (Anexo 2 do leiaute).
	standardLimites := []string{
		"01.00", // Crédito PF
		"02.00", // Crédito PJ
		"03.00", // Crédito rural
		"04.00", // Crédito interfinanceiro
		"05.00", // Títulos privados
		"06.00", // TVM
		"07.00", // Derivativos
		"20.00", // Partes relacionadas
		"21.00", // Operações com partes relacionadas
		"34.00", // Empréstimo TVM
	}
	var limites []Limite
	for _, code := range standardLimites {
		valor := "0.00"
		enviado := "N"
		// Check Extra["limites"] map for this code.
		limitesMap, hasMap := doc.Extra["limites"].(map[string]any)
		if hasMap && limitesMap != nil {
			if v, ok := limitesMap[code]; ok {
				switch vv := v.(type) {
				case float64:
					valor = formatMoney(vv)
					enviado = "S"
				case int:
					valor = formatMoney(float64(vv))
					enviado = "S"
				case string:
					if vv != "" && vv != "0" {
						valor = vv
						enviado = "S"
					}
				}
			}
		}
		limites = append(limites, Limite{
			Codigo:  code,
			Enviado: enviado,
			Valor:   valor,
		})
	}
	return limites
}

func buildParametros(doc *canonical.CanonicalDocument) []Parametro {
	// DLI parameters (Anexo 3): codes 31, 32, 33.
	codes := []string{"31", "32", "33"}
	var params []Parametro
	for _, code := range codes {
		valor := strVal(doc.Extra["parametro_"+code], "0")
		params = append(params, Parametro{
			Codigo: code,
			Valor:  valor,
		})
	}
	return params
}

func groupByConta(doc *canonical.CanonicalDocument) map[string][]canonical.Operacao {
	m := make(map[string][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		code := strVal(op.Extra["contaCosif"], op.Modalidade)
		if code == "" {
			code = "6.00.00"
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
	add("dataBase", "dataBase", time.Time(doc.DataBase).Format("2006-01"), canonical.FontSourceManual)
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
