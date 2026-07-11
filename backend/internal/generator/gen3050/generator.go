// Package gen3050 implementa o CADOCGenerator para o documento 3050 (TXB).
//
// O CADOC 3050 — TXB (Taxas de Operações Bancárias) reporta taxas de juros
// e saldos por modalidade, tipo de pessoa e indexador ao BACEN.
//
// O 3050 tem estrutura hierárquica por agregação:
//   - DocTXB (header)
//     └─ Referencia
//     ├─ Diario
//     │   └─ CRDLivre → PesJuridica/PesFisica → Pre/Flu/Vc/Ind
//     │       └─ <sub-modalidade attrs.../>  (uma por grupo)
//     └─ Mensal
//     └─ CRDLivre → PesJuridica/PesFisica → Pre/Flu/Vc/Ind
//     └─ <sub-modalidade attrs.../>
//
// Layout TXB_V4 (BACEN):
//
//	DocTXB@cnpj, dataBase, indRemessa, nmContato, telContato
//	  Referencia@codigo
//	    Diario
//	      CRDLivre
//	        PesJuridica
//	          Pre | Flu | Vc | Ind
//	            <sub-modalidade txMedJuros="..." txMinima="..." .../>
package gen3050

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

// Generator implementa CADOCGenerator para o CADOC 3050 (TXB).
type Generator struct{}

// New cria um novo generator 3050.
func New() *Generator {
	return &Generator{}
}

// CadocCode retorna "3050".
func (g *Generator) CadocCode() string { return "3050" }

// SupportedVersions retorna as versões de leiaute suportadas.
func (g *Generator) SupportedVersions() []string {
	return []string{"4.0", "4.1", "4.2"}
}

// RequiredFields retorna os campos obrigatórios do 3050.
func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "N6", Required: true, Desc: "Data-base (AAAAMM)"},
		{Tag: "cnpj", Type: "N14", Required: true, Desc: "CNPJ da IF (8 dígitos BACEN)"},
		{Tag: "indRemessa", Type: "A1", Required: true, Desc: "Indicador remessa (I/A/S)"},
		{Tag: "nmContato", Type: "A50", Required: true, Desc: "Nome do contato responsável"},
		{Tag: "telContato", Type: "A15", Required: true, Desc: "Telefone do contato"},
		{Tag: "codigo", Type: "A6", Required: true, Desc: "Código referência (AAAAMM)"},
		{Tag: "tipoCli", Type: "A12", Required: true, Desc: "Tipo cliente (PesJuridica/PesFisica)"},
		{Tag: "encargo", Type: "A3", Required: true, Desc: "Tipo encargo (Pre/Flu/Vc/Ind)"},
		{Tag: "subModalidade", Type: "A30", Required: true, Desc: "Código da sub-modalidade"},
		{Tag: "txMedJuros", Type: "N9,6", Required: true, Desc: "Taxa média de juros (% a.m.)"},
		{Tag: "txMinima", Type: "N9,6", Required: true, Desc: "Taxa mínima (% a.m.)"},
		{Tag: "txMaxima", Type: "N9,6", Required: true, Desc: "Taxa máxima (% a.m.)"},
		{Tag: "vlrConcessoes", Type: "N15,2", Required: true, Desc: "Valor das concessões"},
		{Tag: "sldCarAtiva", Type: "N15,2", Required: true, Desc: "Saldo da carteira ativa"},
		{Tag: "qtdNovContratos", Type: "N8", Required: true, Desc: "Quantidade novos contratos"},
	}
}

// EstimateComplexity avalia a complexidade do documento.
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
		EstimatedTimeMs:   int64(30 + numOp/20),
	}
}

// Generate produz o XML do 3050 TXB a partir do CanonicalDocument.
// Retorna (nil, error) quando o documento não pode ser gerado.
// Retorna (GeneratedDoc, nil) em sucesso.
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

// DocTXB é o elemento raiz do XML 3050 TXB.
type DocTXB struct {
	XMLName    xml.Name   `xml:"DocTXB"`
	CNPJ       string     `xml:"cnpjInstituicao,attr"`
	DataBase   string     `xml:"dataBase,attr"`
	IndRemessa string     `xml:"indRemessa,attr"`
	NmContato  string     `xml:"nmContato,attr"`
	TelContato string     `xml:"telContato,attr"`
	Referencia Referencia `xml:"Referencia"`
}

// Referencia contém os blocos diário e mensal.
type Referencia struct {
	Codigo string `xml:"codigo,attr"`
	Diario Secao  `xml:"Diario"`
	Mensal Secao  `xml:"Mensal"`
}

// Secao representa uma seção (Diario ou Mensal) com blocos por tipo de pessoa.
type Secao struct {
	CRDLivre CRDLivre `xml:"CRDLivre"`
}

// CRDLivre contém os blocos para pessoa jurídica e física.
type CRDLivre struct {
	PesJuridica ClienteBloco `xml:"PesJuridica"`
	PesFisica   ClienteBloco `xml:"PesFisica"`
}

// ClienteBloco contém os quatro encargos (Pre/Flu/Vc/Ind).
type ClienteBloco struct {
	Pre []SubModalidade `xml:"Pre,omitempty"`
	Flu []SubModalidade `xml:"Flu,omitempty"`
	Vc  []SubModalidade `xml:"Vc,omitempty"`
	Ind []SubModalidade `xml:"Ind,omitempty"`
}

// SubModalidade representa uma sub-modalidade agregada.
type SubModalidade struct {
	Codigo               string `xml:"codigo,attr"`
	TxMedJuros           string `xml:"txMedJuros,attr"`
	TxMedJurosAjustada   string `xml:"txMedJurosAjustada,attr,omitempty"`
	TxMedEncFiscais      string `xml:"txMedEncFiscais,attr,omitempty"`
	TxMedEncOperacionais string `xml:"txMedEncOperacionais,attr,omitempty"`
	TxMinima             string `xml:"txMinima,attr"`
	TxMaxima             string `xml:"txMaxima,attr"`
	VlrConcessoes        string `xml:"vlrConcessoes,attr"`
	PrzDecMedConcessoes  string `xml:"przDecMedConcessoes,attr,omitempty"`
	QtdNovContratos      string `xml:"qtdNovContratos,attr"`
	SldCarAtiva          string `xml:"sldCarAtiva,attr"`
	SldCedido            string `xml:"sldCedido,attr,omitempty"`
	SldAdquirido         string `xml:"sldAdquirido,attr,omitempty"`
}

// Aggregation key for TXB grouping.
type txKey struct {
	tipoCli string // "PesJuridica" or "PesFisica"
	encargo string // "Pre", "Flu", "Vc", "Ind"
	mod     string // sub-modalidade code
}

// buildModel transforma o CanonicalDocument no modelo XML 3050.
func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) DocTXB {
	indRemessa := strVal(doc.Extra["indRemessa"], "I")
	nmContato := strVal(doc.Extra["nmContato"], "CONTATO")
	telContato := strVal(doc.Extra["telContato"], "0000000000")
	codigo := dataBase.Format("200601")

	// CNPJ: BACEN usa 8 dígitos (raiz do CNPJ).
	// Strip non-digits first to handle formatted inputs like "12.345.678/0001-23".
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) >= 8 {
		cnpj = cnpj[:8]
	}

	model := DocTXB{
		CNPJ:       cnpj,
		DataBase:   dataBase.Format("2006-01-02"),
		IndRemessa: indRemessa,
		NmContato:  nmContato,
		TelContato: telContato,
		Referencia: Referencia{
			Codigo: codigo,
		},
	}

	agregMap := groupByTXKey(doc)

	// Populate Diario and Mensal sections.
	// For now, all operations go to Diario (Mensal can be added separately).
	for k, ops := range agregMap {
		subMod := buildSubModalidade(ops, k)
		bloc := getBloco(&model.Referencia.Diario.CRDLivre, k.tipoCli)
		switch k.encargo {
		case "Pre":
			bloc.Pre = append(bloc.Pre, subMod)
		case "Flu":
			bloc.Flu = append(bloc.Flu, subMod)
		case "Vc":
			bloc.Vc = append(bloc.Vc, subMod)
		case "Ind":
			bloc.Ind = append(bloc.Ind, subMod)
		}
		setBloco(&model.Referencia.Diario.CRDLivre, k.tipoCli, bloc)
	}

	// mensal: copy same data (or could be different aggregation).
	// TXB typically has same daily and monthly sections with different periods.
	for k, ops := range agregMap {
		subMod := buildSubModalidade(ops, k)
		bloc := getBloco(&model.Referencia.Mensal.CRDLivre, k.tipoCli)
		switch k.encargo {
		case "Pre":
			bloc.Pre = append(bloc.Pre, subMod)
		case "Flu":
			bloc.Flu = append(bloc.Flu, subMod)
		case "Vc":
			bloc.Vc = append(bloc.Vc, subMod)
		case "Ind":
			bloc.Ind = append(bloc.Ind, subMod)
		}
		setBloco(&model.Referencia.Mensal.CRDLivre, k.tipoCli, bloc)
	}

	return model
}

func groupByTXKey(doc *canonical.CanonicalDocument) map[txKey][]canonical.Operacao {
	m := make(map[txKey][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		tipoCli := tipoCliMap(op.TipoPessoa)
		encargo := encargoMap(op.Indexador)
		mod := op.Modalidade
		if mod == "" {
			mod = strVal(doc.Extra["modalidade"], "desDuplicatas")
		}
		k := txKey{tipoCli: tipoCli, encargo: encargo, mod: mod}
		m[k] = append(m[k], op)
	}
	return m
}

func buildSubModalidade(ops []canonical.Operacao, k txKey) SubModalidade {
	var totalVal, minTaxa, maxTaxa float64
	var taxaSum float64
	var taxaCount int
	var qtdContratos int64

	minTaxa = 999
	maxTaxa = -999

	for _, op := range ops {
		val := decimalToFloat(op.ValorPrincipal.Valor)
		totalVal += val
		taxa := decimalToFloat(op.TaxaJuros)
		if taxa > 0 {
			taxaSum += taxa
			taxaCount++
			if taxa < minTaxa {
				minTaxa = taxa
			}
			if taxa > maxTaxa {
				maxTaxa = taxa
			}
		}
		qtdContratos++
	}

	var avgTaxa float64
	if taxaCount > 0 {
		avgTaxa = taxaSum / float64(taxaCount)
	}

	return SubModalidade{
		Codigo:          k.mod,
		TxMedJuros:      formatTaxa(avgTaxa),
		TxMinima:        formatTaxa(minTaxa),
		TxMaxima:        formatTaxa(maxTaxa),
		VlrConcessoes:   formatMoney(totalVal),
		QtdNovContratos: fmt.Sprintf("%d", qtdContratos),
		SldCarAtiva:     formatMoney(totalVal), // use same as proxy
	}
}

// tipoCliMap converts canonical TipoPessoa to TXB PesJuridica/PesFisica.
func tipoCliMap(tipoPessoa string) string {
	switch tipoPessoa {
	case "PJ", "pesJuridica", "juridica":
		return "PesJuridica"
	case "PF", "pesFisica", "fisica":
		return "PesFisica"
	default:
		return "PesJuridica"
	}
}

// encargoMap converts canonical Indexador to TXB encargo (Pre/Flu/Vc/Ind).
// PRE = pré-fixado, FLU = flutuante, VC = variação do câmbio, IND = índices.
func encargoMap(indexador string) string {
	switch indexador {
	case "PRE", "pre":
		return "Pre"
	case "FLU", "flu", "CDI", "IPCA", "TJLP", "TR", "IGP-M":
		return "Flu"
	case "VC", "vc", "câmbio", "cambio", "USD", "EUR":
		return "Vc"
	case "IND", "ind", "índice", "indice":
		return "Ind"
	default:
		return "Flu" // default to floating
	}
}

// getBloco returns the ClienteBloco for a given tipoCli.
func getBloco(crd *CRDLivre, tipoCli string) ClienteBloco {
	switch tipoCli {
	case "PesJuridica":
		return crd.PesJuridica
	default:
		return crd.PesFisica
	}
}

// setBloco sets the ClienteBloco for a given tipoCli.
func setBloco(crd *CRDLivre, tipoCli string, bloc ClienteBloco) {
	switch tipoCli {
	case "PesJuridica":
		crd.PesJuridica = bloc
	default:
		crd.PesFisica = bloc
	}
}

// formatTaxa formats a taxa (decimal, e.g. 0.015 = 1.5%) as percentage string.
// TXB uses percentage value (e.g. "1.5000" for 1.5% a.m.).
// Returns "0.0000" for zero, negative (sentinel -999), or >100 values.
func formatTaxa(v float64) string {
	if v <= 0 || v > 100 { // guard against zero, negative (sentinels), and overflow
		return "0.0000"
	}
	// Convert decimal to percentage (e.g. 0.015 → 1.5000).
	pct := v * 100
	return fmt.Sprintf("%.4f", pct)
}

// formatMoney formats a monetary value as N15,2 string.
func formatMoney(v float64) string {
	return fmt.Sprintf("%.2f", v)
}

// cleanDigits returns only digit characters from s.
func cleanDigits(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	return string(out)
}

// decimalToFloat converts a decimal.Decimal or float64 to float64.
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

func strVal(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// buildFieldMap creates audit trail of field mappings.
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
	add("dataBase", "dataBase", time.Time(doc.DataBase).Format("200601"), canonical.FontSourceManual)
	add("cnpj", "cnpj", doc.Header.CNPJ, canonical.FontSourceManual)
	return fm
}

// sha256Hex returns the hex string of SHA-256 of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
