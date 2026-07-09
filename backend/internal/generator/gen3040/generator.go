// Package gen3040 implementa o CADOCGenerator para o documento 3040 (SCR).
//
// O CADOC 3040 — SCR (Risco de Crédito) é o documento mais crítico:
// reporta todas as operações de crédito ao BACEN com dados por faixa
// de vencimento, modalidade, UF e classificação de risco.
//
// O 3040 tem estrutura hierárquica:
//   - Doc3040 (header)
//     └─ Agreg[] (por natureza, modalidade, UF, tpCli, desemp, prov)
//     └─ Venc (faixas de vencimento: v110 a v165)
//
// Layout base (pode variar por versão):
//
//	Doc3040@dataBase, cnpj, remessa, parte, tpArq, nomeResp, emailResp,
//	                 telResp, totalCli
//	  Agreg@natuOp, mod, origemRec, vincME, classOp, faixaVlr, przProvm,
//	               localiz, tpCli, desempOp, provConsttd, qtdOp, qtdCli
//	    Venc@v110, v120, v130, v140, v150, v160, v165
package gen3040

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

// Generator implementa CADOCGenerator para o CADOC 3040 (SCR).
type Generator struct{}

// New cria um novo generator 3040.
func New() *Generator {
	return &Generator{}
}

// CadocCode retorna "3040".
func (g *Generator) CadocCode() string { return "3040" }

// SupportedVersions retorna as versões de leiaute suportadas.
func (g *Generator) SupportedVersions() []string {
	return []string{"3.0", "3.1", "3.2"}
}

// RequiredFields retorna os campos obrigatórios do 3040.
func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "N6", Required: true, Desc: "Data-base (AAAAMM)"},
		{Tag: "cnpj", Type: "N14", Required: true, Desc: "CNPJ da IF"},
		{Tag: "remessa", Type: "N3", Required: true, Desc: "Número da remessa"},
		{Tag: "parte", Type: "N3", Required: true, Desc: "Número da parte"},
		{Tag: "tpArq", Type: "A1", Required: true, Desc: "Tipo arquivo (F=full, S=substituição)"},
		{Tag: "nomeResp", Type: "A50", Required: true, Desc: "Nome do responsável"},
		{Tag: "emailResp", Type: "A60", Required: true, Desc: "E-mail do responsável"},
		{Tag: "telResp", Type: "A15", Required: true, Desc: "Telefone do responsável"},
		{Tag: "totalCli", Type: "N8", Required: true, Desc: "Total de clientes"},
		{Tag: "natuOp", Type: "N2", Required: true, Desc: "Natureza da operação"},
		{Tag: "mod", Type: "N4", Required: true, Desc: "Modalidade"},
		{Tag: "origemRec", Type: "N1", Required: true, Desc: "Origem do recurso"},
		{Tag: "vincME", Type: "A1", Required: true, Desc: "Vínculo com ME/EPP"},
		{Tag: "classOp", Type: "A1", Required: true, Desc: "Classificação operação (A-H)"},
		{Tag: "faixaVlr", Type: "A1", Required: true, Desc: "Faixa de valor"},
		{Tag: "przProvm", Type: "A1", Required: true, Desc: "Prazo provisão"},
		{Tag: "localiz", Type: "A2", Required: true, Desc: "UF de localização"},
		{Tag: "tpCli", Type: "A1", Required: true, Desc: "Tipo cliente (1=PF, 2=PJ)"},
		{Tag: "desempOp", Type: "N2", Required: true, Desc: "Desempenho da operação"},
		{Tag: "provConsttd", Type: "N15,2", Required: true, Desc: "Provisão constituída"},
		{Tag: "qtdOp", Type: "N8", Required: true, Desc: "Quantidade de operações"},
		{Tag: "qtdCli", Type: "N8", Required: true, Desc: "Quantidade de clientes"},
		{Tag: "v110", Type: "N15,2", Required: true, Desc: "Venc até 3 meses"},
		{Tag: "v120", Type: "N15,2", Required: true, Desc: "Venc 3-6 meses"},
		{Tag: "v130", Type: "N15,2", Required: true, Desc: "Venc 6-12 meses"},
		{Tag: "v140", Type: "N15,2", Required: true, Desc: "Venc 1-3 anos"},
		{Tag: "v150", Type: "N15,2", Required: true, Desc: "Venc 3-5 anos"},
		{Tag: "v160", Type: "N15,2", Required: true, Desc: "Venc 5-10 anos"},
		{Tag: "v165", Type: "N15,2", Required: true, Desc: "Venc mais 10 anos"},
	}
}

// EstimateComplexity avalia a complexidade do documento.
func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	numPart := len(doc.Participantes)

	score := 0.3
	if numOp > 100 {
		score += 0.2
	}
	if numOp > 1000 {
		score += 0.2
	}
	if numPart > 50 {
		score += 0.1
	}

	return generator.ComplexityScore{
		Score:             score,
		NumOperacoes:      numOp,
		NumParticipantes:  numPart,
		EstimatedAPICalls: 0,
		EstimatedTimeMs:   int64(50 + numOp/10),
	}
}

// Generate produz o XML do 3040 a partir do CanonicalDocument.
func (g *Generator) Generate(ctx context.Context, doc *canonical.CanonicalDocument, dataBase time.Time) (*generator.GeneratedDoc, error) {
	start := time.Now()

	if errs := doc.Validate(); len(errs) > 0 {
		return &generator.GeneratedDoc{
			CadocCode: g.CadocCode(),
			Errors:    mapErrors(errs),
			Metadata: generator.GenMetadata{
				GeneratorVersion: "1.0.0",
				GeneratedAt:      start,
				DurationMs:       time.Since(start).Milliseconds(),
			},
		}, nil
	}

	model := buildModel(doc, dataBase)

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(model); err != nil {
		return &generator.GeneratedDoc{
			CadocCode: g.CadocCode(),
			Errors: []generator.GenError{{
				Code:    "XML_ENCODE_ERROR",
				Message: fmt.Sprintf("falha ao serializar XML: %v", err),
			}},
			Metadata: generator.GenMetadata{
				GeneratorVersion: "1.0.0",
				GeneratedAt:      start,
				DurationMs:       time.Since(start).Milliseconds(),
			},
		}, nil
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

// Doc3040 é o modelo XML do 3040.
type Doc3040 struct {
	XMLName     xml.Name `xml:"Doc3040"`
	DataBase    string   `xml:"dataBase,attr"`
	CNPJ        string   `xml:"cnpj,attr"`
	Remessa     string   `xml:"remessa,attr"`
	Parte       string   `xml:"parte,attr"`
	TpArq       string   `xml:"tpArq,attr"`
	NomeResp    string   `xml:"nomeResp,attr"`
	EmailResp   string   `xml:"emailResp,attr"`
	TelResp     string   `xml:"telResp,attr"`
	TotalCli    string   `xml:"totalCli,attr"`
	Agregadas   []Agreg  `xml:"Agreg,omitempty"`
	Totalizador *Totaliz `xml:"Totalizador,omitempty"`
}

// Agreg representa um bloco Agreg do 3040.
// Unique por (natuOp, mod, localiz, tpCli, desempOp, classOp, faixaVlr, przProvm).
type Agreg struct {
	NatuOp      string `xml:"natuOp,attr"`
	Mod         string `xml:"mod,attr"`
	OrigemRec   string `xml:"origemRec,attr"`
	VincME      string `xml:"vincME,attr"`
	ClassOp     string `xml:"classOp,attr"`
	FaixaVlr    string `xml:"faixaVlr,attr"`
	PrzProvm    string `xml:"przProvm,attr"`
	Localiz     string `xml:"localiz,attr"`
	TpCli       string `xml:"tpCli,attr"`
	DesempOp    string `xml:"desempOp,attr"`
	ProvConsttd string `xml:"provConsttd,attr"`
	QtdOp       string `xml:"qtdOp,attr"`
	QtdCli      string `xml:"qtdCli,attr"`
	Venc        Venc   `xml:"Venc"`
}

// Venc representa as faixas de vencimento do 3040.
type Venc struct {
	V110 string `xml:"v110,attr"` // até 3 meses
	V120 string `xml:"v120,attr"` // 3-6 meses
	V130 string `xml:"v130,attr"` // 6-12 meses
	V140 string `xml:"v140,attr"` // 1-3 anos
	V150 string `xml:"v150,attr"` // 3-5 anos
	V160 string `xml:"v160,attr"` // 5-10 anos
	V165 string `xml:"v165,attr"` // mais 10 anos
}

// Totaliz é o bloco de totalização do 3040.
type Totaliz struct {
	ProvTotal   string `xml:"provTotal,attr"`
	QtdOpTotal  string `xml:"qtdOpTotal,attr"`
	QtdCliTotal string `xml:"qtdCliTotal,attr"`
	V110T       string `xml:"v110,attr"`
	V120T       string `xml:"v120,attr"`
	V130T       string `xml:"v130,attr"`
	V140T       string `xml:"v140,attr"`
	V150T       string `xml:"v150,attr"`
	V160T       string `xml:"v160,attr"`
	V165T       string `xml:"v165,attr"`
}

// buildModel transforma o CanonicalDocument no modelo XML 3040.
func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) Doc3040 {
	model := Doc3040{
		DataBase:  dataBase.Format("200601"),
		CNPJ:      doc.Header.CNPJ,
		Remessa:   fmt.Sprintf("%d", max(1, intVal(doc.Extra["remessa"]))),
		Parte:     fmt.Sprintf("%d", max(1, intVal(doc.Extra["parte"]))),
		TpArq:     strVal(doc.Extra["tpArq"], "F"),
		NomeResp:  strVal(doc.Extra["nomeResp"], "RESP"),
		EmailResp: strVal(doc.Extra["emailResp"], "resp@if.com.br"),
		TelResp:   strVal(doc.Extra["telResp"], "0000000000"),
		TotalCli:  fmt.Sprintf("%d", len(doc.Participantes)),
	}

	agregMap := groupByKey(doc)
	for _, key := range sortedKeys(agregMap) {
		model.Agregadas = append(model.Agregadas, buildAgreg(agregMap[key], key))
	}

	if len(model.Agregadas) > 0 {
		model.Totalizador = buildTotalizador(model.Agregadas)
	}

	return model
}

// key3040 identifica uma Agreg única.
type key3040 struct {
	natuOp, mod, origemRec, vincME, classOp, faixaVlr, przProvm, localiz, tpCli, desempOp string
}

func groupByKey(doc *canonical.CanonicalDocument) map[key3040][]canonical.Operacao {
	m := make(map[key3040][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		k := key3040{
			natuOp:    strVal(doc.Extra["natuOp"], "01"),
			mod:       op.Modalidade,
			origemRec: strVal(doc.Extra["origemRec"], "1"),
			vincME:    strVal(doc.Extra["vincME"], "N"),
			classOp:   op.ClassificacaoIF,
			faixaVlr:  faixaFromValor(op.ValorPrincipal.Valor),
			przProvm:  strVal(doc.Extra["przProvm"], "N"),
			localiz:   op.UF,
			tpCli:     classFromParticipante(op),
			desempOp:  strVal(doc.Extra["desempOp"], "01"),
		}
		m[k] = append(m[k], op)
	}
	return m
}

func buildAgreg(ops []canonical.Operacao, k key3040) Agreg {
	var provTotal float64
	var qtdOp, qtdCli int64
	var v110, v120, v130, v140, v150, v160, v165 float64
	seenClients := make(map[string]bool)

	for _, op := range ops {
		if op.UF == "" {
			op.UF = "SP"
		}
		qtdOp++
		// qtdCli: each contract = 1 client (unique by contract number)
		if op.NumeroContrato != "" && !seenClients[op.NumeroContrato] {
			seenClients[op.NumeroContrato] = true
			qtdCli++
		}

		val := decimalToFloat(op.ValorPrincipal.Valor)
		provTotal += val * 0.01

		faixa := k.faixaVlr
		if faixa == "" {
			faixa = faixaFromValor(op.ValorPrincipal.Valor)
		}
		switch faixa {
		case "1":
			v110 += val
		case "2":
			v120 += val
		case "3":
			v130 += val
		case "4":
			v140 += val
		case "5":
			v150 += val
		case "6":
			v160 += val
		case "7":
			v165 += val
		default:
			v110 += val
		}
	}

	return Agreg{
		NatuOp:      k.natuOp,
		Mod:         k.mod,
		OrigemRec:   k.origemRec,
		VincME:      k.vincME,
		ClassOp:     k.classOp,
		FaixaVlr:    k.faixaVlr,
		PrzProvm:    k.przProvm,
		Localiz:     k.localiz,
		TpCli:       k.tpCli,
		DesempOp:    k.desempOp,
		ProvConsttd: fmt.Sprintf("%.2f", provTotal),
		QtdOp:       fmt.Sprintf("%d", qtdOp),
		QtdCli:      fmt.Sprintf("%d", qtdCli),
		Venc: Venc{
			V110: fmt.Sprintf("%.2f", v110),
			V120: fmt.Sprintf("%.2f", v120),
			V130: fmt.Sprintf("%.2f", v130),
			V140: fmt.Sprintf("%.2f", v140),
			V150: fmt.Sprintf("%.2f", v150),
			V160: fmt.Sprintf("%.2f", v160),
			V165: fmt.Sprintf("%.2f", v165),
		},
	}
}

func buildTotalizador(agregadas []Agreg) *Totaliz {
	var provTotal float64
	var qtdOpT, qtdCliT int64
	var v110, v120, v130, v140, v150, v160, v165 float64

	for _, a := range agregadas {
		prov, _ := strconv.ParseFloat(a.ProvConsttd, 64)
		provTotal += prov
		qtdOp, _ := strconv.ParseInt(a.QtdOp, 10, 64)
		qtdCli, _ := strconv.ParseInt(a.QtdCli, 10, 64)
		qtdOpT += qtdOp
		qtdCliT += qtdCli
		v110 += strFloat(a.Venc.V110)
		v120 += strFloat(a.Venc.V120)
		v130 += strFloat(a.Venc.V130)
		v140 += strFloat(a.Venc.V140)
		v150 += strFloat(a.Venc.V150)
		v160 += strFloat(a.Venc.V160)
		v165 += strFloat(a.Venc.V165)
	}

	return &Totaliz{
		ProvTotal:   fmt.Sprintf("%.2f", provTotal),
		QtdOpTotal:  fmt.Sprintf("%d", qtdOpT),
		QtdCliTotal: fmt.Sprintf("%d", qtdCliT),
		V110T:       fmt.Sprintf("%.2f", v110),
		V120T:       fmt.Sprintf("%.2f", v120),
		V130T:       fmt.Sprintf("%.2f", v130),
		V140T:       fmt.Sprintf("%.2f", v140),
		V150T:       fmt.Sprintf("%.2f", v150),
		V160T:       fmt.Sprintf("%.2f", v160),
		V165T:       fmt.Sprintf("%.2f", v165),
	}
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

func strFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// faixaFromValor determina a faixa de valor (1-7) pelo valor principal.
func faixaFromValor(val any) string {
	f := decimalToFloat(val)
	switch {
	case f <= 10_000:
		return "1"
	case f <= 50_000:
		return "2"
	case f <= 200_000:
		return "3"
	case f <= 1_000_000:
		return "4"
	case f <= 10_000_000:
		return "5"
	case f <= 100_000_000:
		return "6"
	default:
		return "7"
	}
}

func classFromParticipante(op canonical.Operacao) string {
	// tpCli: 1=PF, 2=PJ. Default PF if unknown.
	switch op.TipoPessoa {
	case "PJ":
		return "2"
	case "PF", "":
		return "1"
	default:
		return "1"
	}
}

func sortedKeys(m map[key3040][]canonical.Operacao) []key3040 {
	var keys []key3040
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.mod != b.mod {
			return a.mod < b.mod
		}
		return a.localiz < b.localiz
	})
	return keys
}

func mapErrors(errs []string) []generator.GenError {
	out := make([]generator.GenError, len(errs))
	for i, e := range errs {
		out[i] = generator.GenError{Code: "MISSING_FIELD", Campo: e, Message: e}
	}
	return out
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
	add("dataBase", "dataBase", time.Time(doc.DataBase).Format("200601"), canonical.FontSourceManual)
	add("cnpj", "cnpj", doc.Header.CNPJ, canonical.FontSourceManual)
	for _, op := range doc.Operacoes {
		add("modalidade", "mod", op.Modalidade, canonical.FontSourceManual)
		add("classOp", "classOp", op.ClassificacaoIF, canonical.FontSourceManual)
	}
	return fm
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
