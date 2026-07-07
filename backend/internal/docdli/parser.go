// Package docdli implements parsing and validation for CADOC 2062 — DLI
// (Demonstrativo de Limites Operacionais).
//
// ATENÇÃO: O XSD e críticas oficiais do 2062 NÃO estão disponíveis no repositório.
// Este package implementa apenas validações estruturais genéricas baseadas no
// layout Excel (2062_DLI_Leiaute.xlsx) e nas instruções (DLI_2062_InstrucoesPreenchimento_v3.pdf).
//
//nolint:revive
package docdli

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// DocumentoDLI é o root element do documento 2062.
type DocumentoDLI struct {
	CNPJ      string `xml:"cnpj,attr"`            // 8 dígitos
	DataBase  string `xml:"dataBase,attr"`        // AAAA-MM
	CodigoDoc string `xml:"codigoDocumento,attr"` // "2062"
	TipoEnvio string `xml:"tipoEnvio,attr"`       // "I" ou "S"

	// Seções principais
	Limites     []Limite    `xml:"Limites>Limite"`
	Indicadores []Indicador `xml:"Indicadores>Indicador"`
	Parametros  []Parametro `xml:"Parametros>Parametro"`
	Contas      []Conta     `xml:"Contas>Conta"`

	// Cross-ref para validações
	LimitesOmit     bool `xml:"-"`
	IndicadoresOmit bool `xml:"-"`
	ParametrosOmit  bool `xml:"-"`
	ContasOmit      bool `xml:"-"`
}

// Limite representa um limite operacional reportado.
type Limite struct {
	Codigo string `xml:"codigo,attr"` // ex: "06.00", "20.00"
	Valor  string `xml:"valor,attr"`  // N15,2
}

// Indicador representa indicador de envio (S/N).
type Indicador struct {
	Codigo string `xml:"codigo,attr"` // TABELA 002
	Valor  string `xml:"valor,attr"`  // "S" ou "N"
}

// Parametro representa parâmetro do documento.
type Parametro struct {
	Codigo string `xml:"codigo,attr"` // TABELA 004
	Valor  string `xml:"valor,attr"`
}

// Conta representa uma conta COSIF.
type Conta struct {
	Codigo string `xml:"codigo,attr"` // ex: "6.10.01"
	Valor  string `xml:"valor,attr"`  // N15,2
}

// ============================================================
// Parsing
// ============================================================

// ParseError represents a parsing error.
type ParseError struct{ Inner error }

func (e *ParseError) Error() string { return fmt.Sprintf("parse DLI: %v", e.Inner) }
func (e *ParseError) Unwrap() error { return e.Inner }

var errEmptyDocument = fmt.Errorf("documento DLI vazio")

// Parse reads a DLI document from r.
func Parse(r io.Reader) (*DocumentoDLI, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Inner: err}
	}
	if len(data) == 0 {
		return nil, errEmptyDocument
	}

	var doc DocumentoDLI
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return nil, &ParseError{Inner: err}
	}

	if doc.CodigoDoc != "2062" {
		return nil, fmt.Errorf("codigoDocumento=%q, esperado 2062", doc.CodigoDoc)
	}

	return &doc, nil
}

// ParseFromBytes is a shortcut for Parse(bytes.NewReader(data)).
func ParseFromBytes(data []byte) (*DocumentoDLI, error) {
	return Parse(bytes.NewReader(data))
}

// ============================================================
// Validation errors
// ============================================================

// ValidationError represents a structural validation error.
type ValidationError struct {
	Path    string
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Message
	}
	return e.Message
}

// ============================================================
// Structural validation
// ============================================================

// ValidInstitutionTypes lists valid institution types (reference).
var ValidInstitutionTypes = map[string]bool{
	"BANCO_COMERCIAL":       true,
	"BANCO_MULTIPLO":        true,
	"BANCO_INVESTIMENTO":    true,
	"CTVM":                  true,
	"DTVM":                  true,
	"SOCIEDADE_CORRETORA":   true,
	"SCFI":                  true,
	"SCI":                   true,
	"SAM":                   true,
	"COMPANHIA_HIPOTECARIA": true,
	"AGENCIA_FOMENTO":       true,
	"COOPERATIVA_CREDITO":   true,
	"SCD":                   true,
	"SEP":                   true,
	"INSTITUICAO_PAGAMENTO": true,
}

// Validate executes structural validations over the DLI document.
func Validate(doc *DocumentoDLI) []error {
	var errs []error

	// DLI-01: CNPJ — 8 dígitos
	if !regexp.MustCompile(`^\d{8}$`).MatchString(doc.CNPJ) {
		errs = append(errs, &ValidationError{
			Path: "/DocumentoDLI", Field: "cnpj",
			Message: "CNPJ deve ter exatamente 8 dígitos",
		})
	}

	// DLI-02: DataBase — AAAA-MM
	if !regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(doc.DataBase) {
		errs = append(errs, &ValidationError{
			Path: "/DocumentoDLI", Field: "dataBase",
			Message: "dataBase deve ter formato AAAA-MM",
		})
	}

	// DLI-03: tipoEnvio — I ou S
	if doc.TipoEnvio != "I" && doc.TipoEnvio != "S" {
		errs = append(errs, &ValidationError{
			Path: "/DocumentoDLI", Field: "tipoEnvio",
			Message: "tipoEnvio deve ser I (inclusão) ou S (substituição)",
		})
	}

	// DLI-04: codigoDocumento — 2062
	if doc.CodigoDoc != "2062" {
		errs = append(errs, &ValidationError{
			Path: "/DocumentoDLI", Field: "codigoDocumento",
			Message: "codigoDocumento deve ser 2062",
		})
	}

	// DLI-05: pelo menos uma seção deve estar presente
	hasContent := len(doc.Limites) > 0 || len(doc.Indicadores) > 0 ||
		len(doc.Parametros) > 0 || len(doc.Contas) > 0
	if !hasContent {
		errs = append(errs, &ValidationError{
			Path:    "/DocumentoDLI",
			Message: "documento não tem conteúdo (sem Limites, Indicadores, Parametros ou Contas)",
		})
	}

	// DLI-06: Limite.codigo — formato N.NN.NN (ex: 06.00, 20.00)
	for i, l := range doc.Limites {
		path := fmt.Sprintf("/DocumentoDLI/Limites[%d]", i+1)
		if !regexp.MustCompile(`^\d{2}\.\d{2}$`).MatchString(l.Codigo) {
			errs = append(errs, &ValidationError{
				Path: path, Field: "codigo",
				Message: fmt.Sprintf("código de limite %q inválido (esperado NN.NN)", l.Codigo),
			})
		}
		// Valor deve ser numérico (N15,2)
		if l.Valor != "" && !regexp.MustCompile(`^-?\d{1,13}(\.\d{2})?$`).MatchString(l.Valor) {
			errs = append(errs, &ValidationError{
				Path: path, Field: "valor",
				Message: fmt.Sprintf("valor %q inválido (esperado N13,2)", l.Valor),
			})
		}
	}

	// DLI-07: Indicador.valor — S ou N
	for i, ind := range doc.Indicadores {
		path := fmt.Sprintf("/DocumentoDLI/Indicadores[%d]", i+1)
		if ind.Valor != "S" && ind.Valor != "N" {
			errs = append(errs, &ValidationError{
				Path: path, Field: "valor",
				Message: fmt.Sprintf("indicador %q deve ser S ou N", ind.Valor),
			})
		}
	}

	// DLI-08: Conta.codigo — formato N.NN.NN.NN (COSIF)
	for i, c := range doc.Contas {
		path := fmt.Sprintf("/DocumentoDLI/Contas[%d]", i+1)
		if !regexp.MustCompile(`^\d\.\d{2}\.\d{2}(\.\d{2})?$`).MatchString(c.Codigo) {
			errs = append(errs, &ValidationError{
				Path: path, Field: "codigo",
				Message: fmt.Sprintf("código de conta COSIF %q inválido", c.Codigo),
			})
		}
		// Valor deve ser numérico (N15,2)
		if c.Valor != "" && !regexp.MustCompile(`^-?\d{1,13}(\.\d{2})?$`).MatchString(c.Valor) {
			errs = append(errs, &ValidationError{
				Path: path, Field: "valor",
				Message: fmt.Sprintf("valor %q inválido (esperado N13,2)", c.Valor),
			})
		}
	}

	return errs
}

// ============================================================
// Result
// ============================================================

// Result is the result of a DLI validation.
type Result struct {
	Valid    bool
	Criticas []Critica
	Doc      *DocumentoDLI
}

// Critica is a single validation critique.
type Critica struct {
	Code     string
	Severity string // E, A, I
	Message  string
}

// ValidateDocument validates a DLI document from bytes.
func ValidateDocument(ctx context.Context, data []byte) (*Result, error) {
	doc, err := ParseFromBytes(data)
	if err != nil {
		return nil, err
	}

	errs := Validate(doc)
	var criticas []Critica
	for _, e := range errs {
		if ve, ok := e.(*ValidationError); ok {
			criticas = append(criticas, Critica{
				Code:     "DLI-01",
				Severity: "E",
				Message:  ve.Error(),
			})
		}
	}

	return &Result{
		Valid:    len(criticas) == 0,
		Criticas: criticas,
		Doc:      doc,
	}, nil
}

// ============================================================
// Helpers
// ============================================================

// ExtractLimite retrieves a Limite by its code.
func ExtractLimite(doc *DocumentoDLI, code string) *Limite {
	for _, l := range doc.Limites {
		if l.Codigo == code {
			return &l
		}
	}
	return nil
}

// ExtractConta retrieves a Conta by its COSIF code.
func ExtractConta(doc *DocumentoDLI, codigo string) *Conta {
	for _, c := range doc.Contas {
		if c.Codigo == codigo {
			return &c
		}
	}
	return nil
}

// HasIndicador returns true if the given indicador code has valor S.
func HasIndicador(doc *DocumentoDLI, code string) bool {
	for _, ind := range doc.Indicadores {
		if ind.Codigo == code && ind.Valor == "S" {
			return true
		}
	}
	return false
}

// LimiteMaximo returns the maximum limit for a given limit code based on PLA.
// Uses the regulatory formulas from the PDF instructions.
func LimiteMaximo(doc *DocumentoDLI, limitCode string, pla float64) float64 {
	switch limitCode {
	case "20.00": // Partes relacionadas: 10% do PLA
		return pla * 0.10
	case "21.00": // PN: 1% do PLA
		return pla * 0.01
	case "22.00": // PJ: 5% do PLA
		return pla * 0.05
	case "34.00": // Empréstimo de TVM: 5x PLA
		return pla * 5.0
	default:
		return 0
	}
}

// Balance returns a numeric value from an account or limit value string.
func Balance(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

var _ = strings.TrimSpace // lint
