// Package gen2030 implementa o CADOCGenerator para o documento 2030 (DRSAC).
//
// O CADOC 2030 — DRSAC (Demonstrativo de Risco de Crédito — Concentração)
// reporta concentração de crédito e subsegmentação de risco ESG ao BACEN.
//
// ATENÇÃO: O leiaute e XSD oficiais do DRSAC (2030) não estão disponíveis
// no repositório. Este generator implementa uma estrutura baseada em:
//   - crossdoc rules (XD-003): Subsegmento ESG (S1-S5)
//   - Padrão geral de documentos BACEN (CNPJ + DataBase + aggregate blocks)
//
// Estrutura proposta (sujeita a validação contra spec oficial):
//   - DocDRSAC@cnpj, dataBase, tpDocumento, subsegmento
//   - Concentracao[] (participações por faixa de risco)
//   - Ajustes[] (ajustes de rating por participante)
package gen2030

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

// Generator implements CADOCGenerator for CADOC 2030 (DRSAC/Concentração).
type Generator struct{}

func New() *Generator { return &Generator{} }

func (g *Generator) CadocCode() string { return "2030" }

func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1"}
}

func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A10", Required: true, Desc: "Data-base (AAAA-MM-DD)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz (8 dígitos)"},
		{Tag: "tpDocumento", Type: "A1", Required: true, Desc: "Tipo documento (F/S)"},
		{Tag: "subsegmento", Type: "A2", Required: true, Desc: "Subsegmento ESG (S1 a S5)"},
		{Tag: "concentracao.faixa", Type: "A10", Required: false, Desc: "Faixa de concentração"},
		{Tag: "concentracao.participantes", Type: "N5", Required: false, Desc: "Número de participantes na faixa"},
		{Tag: "concentracao.valor", Type: "N15,2", Required: false, Desc: "Valor total da faixa"},
		{Tag: "ajuste.participante", Type: "A14", Required: false, Desc: "CNPJ/CPF do participante com ajuste"},
		{Tag: "ajuste.ratingAnterior", Type: "A3", Required: false, Desc: "Rating antes do ajuste"},
		{Tag: "ajuste.ratingAtual", Type: "A3", Required: false, Desc: "Rating após o ajuste"},
	}
}

func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	numPart := len(doc.Participantes)
	score := 0.3
	if numPart > 20 {
		score += 0.2
	}
	if numPart > 100 {
		score += 0.2
	}
	return generator.ComplexityScore{
		Score:              score,
		NumOperacoes:       numOp,
		NumParticipantes:   numPart,
		EstimatedAPICalls:  0,
		EstimatedTimeMs:    int64(30 + numPart/10),
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

// DocDRSAC is the root element of the DRSAC XML.
type DocDRSAC struct {
	XMLName       xml.Name       `xml:"DocDRSAC"`
	CNPJ          string         `xml:"cnpj,attr"`
	DataBase      string         `xml:"dataBase,attr"`
	TpDocumento   string         `xml:"tpDocumento,attr"`
	Subsegmento   string         `xml:"subsegmento,attr"`
	Concentracoes []Concentracao `xml:"Concentracao,omitempty"`
	Ajustes       []Ajuste       `xml:"Ajustes>Ajuste,omitempty"`
}

// Concentracao represents a concentration band in DRSAC.
type Concentracao struct {
	Faixa         string `xml:"faixa,attr"`
	Participantes string `xml:"participantes,attr"`
	Valor         string `xml:"valor,attr"`
}

// Ajuste represents a rating adjustment for a participant.
type Ajuste struct {
	Participante    string `xml:"participante,attr"`
	RatingAnterior  string `xml:"ratingAnterior,attr"`
	RatingAtual     string `xml:"ratingAtual,attr"`
}

// faixaConcentracao determines concentration band from participation percentage.
type faixaConcentracao struct {
	faixa    string
	minPct   float64
	maxPct   float64
}

func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocDRSAC {
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}
	tpDoc := strVal(doc.Extra["tpDocumento"], "F")
	subsegmento := strVal(doc.Extra["subsegmento"], "S3") // default S3

	model := DocDRSAC{
		CNPJ:        cnpj,
		DataBase:    dataBase.Format("2006-01-02"),
		TpDocumento: tpDoc,
		Subsegmento: subsegmento,
	}

	// Build concentration bands from participants.
	model.Concentracoes = buildConcentracoes(doc)

	// Build rating adjustments.
	model.Ajustes = buildAjustes(doc)

	return model
}

// buildConcentracoes aggregates participants into concentration bands.
// Bands: "MEGA" (>10%), "GRANDE" (5-10%), "MEDIO" (1-5%), "PEQUENO" (<1%)
func buildConcentracoes(doc *canonical.CanonicalDocument) []Concentracao {
	// Calculate total exposure.
	var total float64
	participanteVals := make(map[string]float64)
	for _, op := range doc.Operacoes {
		val := decimalToFloat(op.ValorPrincipal.Valor)
		total += val
		// Group by participant CNPJ.
		pKey := op.Extra["participanteCnpj"]
		if pKey == nil {
			pKey = op.NumeroContrato
		}
		key := fmt.Sprintf("%v", pKey)
		participanteVals[key] += val
	}

	// Standard concentration bands.
	bands := []struct {
		name  string
		minPct float64
		maxPct float64
	}{
		{"MEGA", 10.0, 100.0},
		{"GRANDE", 5.0, 10.0},
		{"MEDIO", 1.0, 5.0},
		{"PEQUENO", 0.0, 1.0},
	}

	var concs []Concentracao
	for _, band := range bands {
		var count int
		var bandTotal float64
		for _, val := range participanteVals {
			if total > 0 {
				pct := (val / total) * 100
				if pct >= band.minPct && pct < band.maxPct {
					count++
					bandTotal += val
				}
			}
		}
		if count > 0 || band.minPct == 0 {
			concs = append(concs, Concentracao{
				Faixa:         band.name,
				Participantes: fmt.Sprintf("%d", count),
				Valor:         formatMoney(bandTotal),
			})
		}
	}

	return concs
}

func buildAjustes(doc *canonical.CanonicalDocument) []Ajuste {
	var ajustes []Ajuste
	for _, p := range doc.Participantes {
		if len(p.Ratings) == 0 {
			continue
		}
		for i, r := range p.Ratings {
			if i == 0 {
				continue // skip primary rating
			}
			if r.NomeAgencia == "" {
				continue
			}
			ajustes = append(ajustes, Ajuste{
				Participante:   p.CNPJ,
				RatingAnterior: strVal(p.Extra["ratingAnterior"], "BBB"),
				RatingAtual:    r.Nota,
			})
		}
	}
	return ajustes
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
