// Package gen4111 implementa o CADOCGenerator para o documento 4111 (COSIF).
//
// O CADOC 4111 — COSIF (Plano Contábil das Instituições do SFN) reporta
// informações contábeis e estatísticas de clientes ao BACEN.
//
// ATENÇÃO: O XSD e críticas oficiais do 4111 NÃO estão disponíveis no
// repositório. Este generator implementa validações estruturais genéricas
// baseadas na interface observada nas regras cross-doc (3040_4111.go).
// As 30+ regras reais dependem do documento de críticas oficial do BACEN.
//
// Estrutura approximada baseada em cross-doc rules e parser:
//
//	Documento4111@cnpj, dataBase, codigoDocumento="4111"
//	  Cliente
//	    QtdCli         — quantidade de clientes neste registro
//	    CNPJ           — CNPJ do cliente (opcional)
//	    Modalidade@codigo, indicacao  — modalidades contratadas
//
// O 4111 funciona como extrato de modalidades: cada Cliente agrupa
// uma quantidade de clientes e suas modalidades. A indicação (S/N)
// marca inadimplência.
package gen4111

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

// Generator implementa CADOCGenerator para o CADOC 4111 (COSIF).
type Generator struct{}

// New cria um novo generator 4111.
func New() *Generator {
	return &Generator{}
}

// CadocCode retorna "4111".
func (g *Generator) CadocCode() string { return "4111" }

// SupportedVersions retorna as versões de leiaute suportadas.
func (g *Generator) SupportedVersions() []string {
	return []string{"1.0", "1.1", "1.2"}
}

// RequiredFields retorna os campos obrigatórios do 4111.
func (g *Generator) RequiredFields() []schema.Field {
	return []schema.Field{
		{Tag: "dataBase", Type: "A7", Required: true, Desc: "Data-base (AAAA-MM)"},
		{Tag: "cnpj", Type: "A8", Required: true, Desc: "CNPJ raiz da IF (8 dígitos)"},
		{Tag: "codigoDocumento", Type: "A4", Required: true, Desc: "Código do documento (4111)"},
		{Tag: "qtdCli", Type: "N8", Required: true, Desc: "Quantidade de clientes no registro"},
		{Tag: "modalidade.codigo", Type: "A10", Required: false, Desc: "Código da modalidade COSIF"},
		{Tag: "modalidade.indicacao", Type: "A1", Required: false, Desc: "Indicação de inadimplência (S/N)"},
	}
}

// EstimateComplexity avalia a complexidade do documento.
func (g *Generator) EstimateComplexity(doc *canonical.CanonicalDocument) generator.ComplexityScore {
	numOp := len(doc.Operacoes)
	numPart := len(doc.Participantes)

	score := 0.2
	if numPart > 100 {
		score += 0.2
	}
	if numPart > 1000 {
		score += 0.2
	}
	if numOp > 200 {
		score += 0.1
	}

	return generator.ComplexityScore{
		Score:              score,
		NumOperacoes:       numOp,
		NumParticipantes:   numPart,
		EstimatedAPICalls:  0,
		EstimatedTimeMs:    int64(30 + numPart/20),
	}
}

// Generate produz o XML do 4111 COSIF a partir do CanonicalDocument.
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

// Documento4111 é o elemento raiz do XML 4111.
type Documento4111 struct {
	XMLName     xml.Name  `xml:"Documento4111"`
	CNPJ        string   `xml:"cnpj,attr"`
	DataBase    string   `xml:"dataBase,attr"`
	CodigoDoc   string   `xml:"codigoDocumento,attr"`
	Clientes    []Cliente `xml:"Cliente,omitempty"`
	Totalizador *Totaliz `xml:"Totalizador,omitempty"`
}

// Cliente representa um bloco de cliente no 4111.
type Cliente struct {
	QtdCli     string       `xml:"QtdCli,omitempty"`
	CNPJ       string       `xml:"CNPJ,omitempty"`
	Modalidades []Modalidade `xml:"Modalidade,omitempty"`
}

// Modalidade representa uma modalidade de crédito no 4111.
type Modalidade struct {
	Codigo    string `xml:"codigo,attr"`
	Indicacao string `xml:"indicacao,attr"` // S = inadimplente, N = adimplente
}

// Totaliz é o bloco de totalização do 4111.
type Totaliz struct {
	QtdCliTotal string `xml:"qtdCliTotal,attr"`
	ModTotal    string `xml:"modTotal,attr"`
	QtdModTotal string `xml:"qtdModTotal,attr"`
}

// key4111 identifica um cliente único no 4111 pela raiz CNPJ.
type key4111 struct {
	cnpj       string
	tipoPessoa string
	uf         string
}

// buildModel transforma o CanonicalDocument no modelo XML 4111.
func buildModel(doc *canonical.CanonicalDocument, dataBase time.Time) Documento4111 {
	// CNPJ: BACEN usa 8 dígitos (raiz do CNPJ).
	cnpj := cleanDigits(doc.Header.CNPJ)
	if len(cnpj) > 8 {
		cnpj = cnpj[:8]
	}

	model := Documento4111{
		CNPJ:      cnpj,
		DataBase:  dataBase.Format("2006-01-02"), // YYYY-MM-DD para compatibilidade cross-doc com 3040
		CodigoDoc: "4111",
	}

	// Agrega operações por cliente (CNPJ raiz + tipo pessoa + UF).
	agregMap := groupByCliente(doc)
	for _, key := range sortedKeys(agregMap) {
		model.Clientes = append(model.Clientes, buildCliente(agregMap[key], key))
	}

	if len(model.Clientes) > 0 {
		model.Totalizador = buildTotalizador(model.Clientes)
	}

	return model
}

// groupByCliente agrega operações por cliente (raiz CNPJ + tipo + UF).
func groupByCliente(doc *canonical.CanonicalDocument) map[key4111][]canonical.Operacao {
	m := make(map[key4111][]canonical.Operacao)
	for _, op := range doc.Operacoes {
		// Tenta extrair CNPJ raiz do participante.
		cnpj := extractCNPJRaiz(op)
		if cnpj == "" {
			cnpj = "00000000"
		}
		k := key4111{
			cnpj:       cnpj,
			tipoPessoa: op.TipoPessoa,
			uf:         op.UF,
		}
		m[k] = append(m[k], op)
	}
	return m
}

// extractCNPJRaiz tenta extrair a raiz do CNPJ (8 dígitos) de uma operação.
// Olha no Extra, no NúmeroInscricao do primeiro participante, ou usa默认值.
func extractCNPJRaiz(op canonical.Operacao) string {
	// 1. Tenta Extra["cnpj"] ou Extra["cnpjRaiz"]
	if v, ok := op.Extra["cnpj"].(string); ok && v != "" {
		return cleanDigits(v)[:min(8, len(cleanDigits(v)))]
	}
	if v, ok := op.Extra["cnpjRaiz"].(string); ok && v != "" {
		return cleanDigits(v)[:min(8, len(cleanDigits(v)))]
	}

	// 2. Tenta Extra["numeroInscricao"]
	if v, ok := op.Extra["numeroInscricao"].(string); ok && v != "" {
		digits := cleanDigits(v)
		if len(digits) >= 8 {
			return digits[:8]
		}
	}

	return ""
}

// buildCliente constrói um bloco Cliente a partir das operações agregadas.
func buildCliente(ops []canonical.Operacao, k key4111) Cliente {
	// Conta clientes únicos por contrato.
	seenContracts := make(map[string]bool)
	for _, op := range ops {
		if op.NumeroContrato != "" {
			seenContracts[op.NumeroContrato] = true
		}
	}
	qtdCli := len(seenContracts)
	if qtdCli == 0 {
		qtdCli = len(ops) // fallback: 1 op = 1 cliente
	}

	cliente := Cliente{
		QtdCli:       fmt.Sprintf("%d", qtdCli),
		CNPJ:         k.cnpj,
		Modalidades:  buildModalidades(ops),
	}

	return cliente
}

// buildModalidades agrega operações por modalidade e determina indicação.
// Se qualquer operação do cliente tiver inadimplência (nível risco <= C),
// a modalidade é marcada com indicacao="S".
func buildModalidades(ops []canonical.Operacao) []Modalidade {
	// Agrega por modalidade.
	modMap := make(map[string][]canonical.Operacao)
	for _, op := range ops {
		mod := op.Modalidade
		if mod == "" {
			mod = "0000"
		}
		modMap[mod] = append(modMap[mod], op)
	}

	var mods []Modalidade
	for mod, opsMod := range modMap {
		indicacao := "N" // default: adimplente
		// Se qualquer operação tiver nível de risco elevado (E, F, G, H ou %provisao > 5%),
		// marca como inadimplente.
		for _, op := range opsMod {
			if isInadimplente(op) {
				indicacao = "S"
				break
			}
		}
		mods = append(mods, Modalidade{
			Codigo:    mod,
			Indicacao: indicacao,
		})
	}

	// Ordena por código de modalidade.
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].Codigo < mods[j].Codigo
	})

	return mods
}

// isInadimplente retorna true se a operação tem indicador de inadimplência.
func isInadimplente(op canonical.Operacao) bool {
	// 1. Nível de risco alto (E, F, G, H = Provisionamento 100%+).
	if op.NivelRisco == "E" || op.NivelRisco == "F" ||
		op.NivelRisco == "G" || op.NivelRisco == "H" {
		return true
	}
	// 2. Percentual de provisão > 5% (limiar de inadimplência).
	if !op.PercentualProvisao.IsZero() {
		pct, _ := op.PercentualProvisao.Float64()
		if pct > 0.05 {
			return true
		}
	}
	// 3. Extra["inadimplente"] = true.
	if v, ok := op.Extra["inadimplente"].(bool); ok && v {
		return true
	}
	if v, ok := op.Extra["inadimplente"].(string); ok && (v == "S" || v == "s" || v == "true") {
		return true
	}
	return false
}

// buildTotalizador calcula totais agregados do 4111.
func buildTotalizador(clientes []Cliente) *Totaliz {
	var totalCli int64
	var modSet = make(map[string]bool)

	for _, cl := range clientes {
		var qtd int64
		if qtd, _ = strconv.ParseInt(cl.QtdCli, 10, 64); qtd <= 0 {
			qtd = 1
		}
		totalCli += qtd
		for _, m := range cl.Modalidades {
			modSet[m.Codigo] = true
		}
	}

	return &Totaliz{
		QtdCliTotal: fmt.Sprintf("%d", totalCli),
		ModTotal:    fmt.Sprintf("%d", len(modSet)),
		QtdModTotal: fmt.Sprintf("%d", len(clientes)), // qtd de registros de cliente
	}
}

// sortedKeys retorna as chaves ordenadas do mapa.
func sortedKeys(m map[key4111][]canonical.Operacao) []key4111 {
	var keys []key4111
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].cnpj != keys[j].cnpj {
			return keys[i].cnpj < keys[j].cnpj
		}
		return keys[i].uf < keys[j].uf
	})
	return keys
}

// buildFieldMap cria o mapa de campos para auditoria.
func buildFieldMap(doc *canonical.CanonicalDocument) []canonical.FieldMapping {
	var fm []canonical.FieldMapping
	add := func(cosif, xmlTag, val string, fonte canonical.FieldSource) {
		fm = append(fm, canonical.FieldMapping{
			CampoCOSIF:       cosif,
			CampoXML:         xmlTag,
			ValorFormatado:   val,
			Fonte:            fonte,
		})
	}
	add("dataBase", "dataBase", time.Time(doc.DataBase).Format("2006-01"), canonical.FontSourceManual)
	add("cnpj", "cnpj", doc.Header.CNPJ, canonical.FontSourceManual)
	for _, op := range doc.Operacoes {
		add("modalidade", "Modalidade@codigo", op.Modalidade, canonical.FontSourceManual)
	}
	return fm
}

// sha256Hex retorna o hash SHA-256 em hex.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// cleanDigits retorna apenas os dígitos de uma string.
func cleanDigits(s string) string {
	var out []byte
	for _, c := range []byte(s) {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	return string(out)
}

// min retorna o menor inteiro.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decimalToFloat converte decimal.Decimal ou float64 para float64.
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

// Verify interface.
var _ generator.CADOCGenerator = (*Generator)(nil)
