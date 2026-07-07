// Package doc4111 implementa parsing e validação básica do CADOC 4111.
//
// ATENÇÃO: O XSD e críticas oficiais do 4111 NÃO estão disponíveis no repositório.
// Este package implementa apenas validações estruturais genéricas baseadas na
// interface observada nas regras cross-doc (internal/crossdoc/rules/3040_4111.go).
//
// As validações reais (30+ regras) dependem do documento de críticas oficial
// do BACEN, que deve ser solicitado formalmente.
//
//nolint:revive
package doc4111

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Documento4111 é o root element do documento 4111.
//
// Estrutura aproximada baseada em cross-doc rules:
// <Documento4111 cnpj="XXXXXXXX" dataBase="AAAA-MM" codigoDocumento="4111">
//
//	<Cliente>
//	  <QtdCli>...</QtdCli>
//	  ... (campos adicionais desconhecidos sem spec)
//	</Cliente>
//
// </Documento4111>
type Documento4111 struct {
	CNPJ      string    `xml:"cnpj,attr"`            // A8
	DataBase  string    `xml:"dataBase,attr"`        // AAAA-MM
	CodigoDoc string    `xml:"codigoDocumento,attr"` // "4111"
	Clientes  []Cliente `xml:"Cliente"`
}

// Cliente representa um cliente no 4111.
type Cliente struct {
	QtdCli     string       `xml:"QtdCli"`               // Quantidade de clientes neste registro
	CNPJ       string       `xml:"CNPJ,omitempty"`       // CNPJ do cliente (se disponível)
	Modalidade []Modalidade `xml:"Modalidade,omitempty"` // Modalidades contratadas
}

// Modalidade representa uma modalidade de crédito.
type Modalidade struct {
	Codigo    string `xml:"codigo,attr"`    // Código da modalidade
	Indicacao string `xml:"indicacao,attr"` // Indicação de inadimplência
}

// ParseError representa erro de parsing.
type ParseError struct{ Inner error }

func (e *ParseError) Error() string { return fmt.Sprintf("parse 4111: %v", e.Inner) }
func (e *ParseError) Unwrap() error { return e.Inner }

// ErrEmptyDocument.
var ErrEmptyDocument = errors.New("documento 4111 vazio")

// Parse lê um documento 4111 de r.
func Parse(r io.Reader) (*Documento4111, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, &ParseError{Inner: err}
	}
	if len(data) == 0 {
		return nil, ErrEmptyDocument
	}

	var doc Documento4111
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		return nil, &ParseError{Inner: err}
	}

	if doc.CodigoDoc != "4111" {
		return nil, fmt.Errorf("codigoDocumento=%q, esperado 4111", doc.CodigoDoc)
	}

	return &doc, nil
}

// ParseFromBytes é atalho para Parse(bytes.NewReader(data)).
func ParseFromBytes(data []byte) (*Documento4111, error) {
	return Parse(bytes.NewReader(data))
}

// ============================================================
// Validações estruturais (genéricas — requer spec oficial para 30+ regras reais)
// ============================================================

// ValidationError.
type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Message
	}
	return e.Message
}

// Validate executa validações estruturais sobre o documento.
// Retorna lista de ValidationError (vazia se válido).
func Validate(doc *Documento4111) []error {
	var errs []error

	// 4111-01: CNPJ: 8 dígitos
	if !regexp.MustCompile(`^\d{8}$`).MatchString(doc.CNPJ) {
		errs = append(errs, &ValidationError{
			Path: "/Documento4111", Message: "CNPJ deve ter 8 dígitos",
		})
	}

	// 4111-02: DataBase: AAAA-MM
	if !regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`).MatchString(doc.DataBase) {
		errs = append(errs, &ValidationError{
			Path: "/Documento4111", Message: "dataBase deve ter formato AAAA-MM",
		})
	}

	// 4111-03: pelo menos 1 cliente (se omitido é documento zerado — válido)
	// Regra: se há clientes, QtdCli deve ser > 0
	for i, cl := range doc.Clientes {
		path := fmt.Sprintf("/Documento4111/Cliente[%d]", i+1)
		if cl.QtdCli != "" {
			if !regexp.MustCompile(`^\d+$`).MatchString(cl.QtdCli) {
				errs = append(errs, &ValidationError{
					Path: path, Message: "QtdCli deve ser numérico",
				})
			}
		}
		// CNPJ do cliente, se presente, deve ser válido
		if cl.CNPJ != "" {
			if !isValidCNPJOrCPF(cl.CNPJ) {
				errs = append(errs, &ValidationError{
					Path: path, Message: fmt.Sprintf("CNPJ/CPF=%s inválido", cl.CNPJ),
				})
			}
		}
	}

	return errs
}

// isValidCNPJOrCPF aceita CNPJ (8 para raiz ou 14 para CNPJ completo)
// ou CPF (11 dígitos).
func isValidCNPJOrCPF(v string) bool {
	if regexp.MustCompile(`^\d{8}$`).MatchString(v) {
		return true // CNPJ raiz
	}
	if regexp.MustCompile(`^\d{14}$`).MatchString(v) {
		return true // CNPJ completo
	}
	if regexp.MustCompile(`^\d{11}$`).MatchString(v) {
		return true // CPF
	}
	return false
}

// ============================================================
// Result
// ============================================================

// Result é o resultado de validação do 4111.
type Result struct {
	Valid    bool
	Criticas []Critica
	Doc      *Documento4111
}

// Critica.
type Critica struct {
	Code     string
	Severity string // E, A, I
	Message  string
}

// ValidateDocument valida um documento 4111.
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
				Code:     "4111-01",
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

// ExtractQtdTotal retorna a soma de QtdCli de todos os clientes.
func ExtractQtdTotal(doc *Documento4111) float64 {
	var total float64
	for _, cl := range doc.Clientes {
		var qtd float64
		if _, err := fmt.Sscanf(cl.QtdCli, "%f", &qtd); err == nil {
			total += qtd
		}
	}
	return total
}

// ExtractCNPJs retorna todos os CNPJs únicos presentes no documento.
func ExtractCNPJs(doc *Documento4111) []string {
	seen := make(map[string]bool)
	var cnpjs []string
	for _, cl := range doc.Clientes {
		if cl.CNPJ != "" && !seen[cl.CNPJ] {
			seen[cl.CNPJ] = true
			cnpjs = append(cnpjs, cl.CNPJ)
		}
	}
	return cnpjs
}

// HasModalidadeInadimplente verifica se algum cliente tem modalidade com indicação.
func HasModalidadeInadimplente(doc *Documento4111) bool {
	for _, cl := range doc.Clientes {
		for _, m := range cl.Modalidade {
			if m.Indicacao == "S" || m.Indicacao == "s" {
				return true
			}
		}
	}
	return false
}

var _ = strings.TrimSpace // golint
var _ = regexp.MustCompile("")
