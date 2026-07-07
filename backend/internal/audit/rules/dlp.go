// DocDLP — Documento DLP (Demonstrativo de Liquidez de Longo Prazo) — CADOC 2170.
//
// Sprint 41: parser best-effort para DLP (subset de 2170 focado em NSFR
// — Net Stable Funding Ratio). Suporta ASF Total, RSF Total, NSFR ratio.
//
// Referência: BACEN Res. 4.542 (NSFR — Net Stable Funding Ratio).
package rules

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DocDLP é o documento DLP parseado (subset NSFR).
type DocDLP struct {
	Root      DocDLPRoot
	ASFTotal  float64 // Available Stable Funding Total
	RSFTotal  float64 // Required Stable Funding Total
	NSFRRatio float64 // NSFR ratio (ASF / RSF × 100)
	// Cenários de estresse
	Cenario1 CenarioNSFR // Cenário base
	Cenario2 CenarioNSFR // Cenário adverso
}

// DocDLPRoot é o elemento raiz do DLP.
type DocDLPRoot struct {
	CNPJ     string
	DataBase string // YYYY-MM-DD
}

// CenarioNSFR representa um cenário de estresse para NSFR.
type CenarioNSFR struct {
	ASF       float64
	RSF       float64
	NSFRRatio float64
}

// ASFItem representa um item componente do ASF (Available Stable Funding).
type ASFItem struct {
	Codigo string
	Desc   string
	Valor  float64
	Fator  float64 // peso de estabilidade (0.0-1.0)
}

// RSFItem representa um item componente do RSF (Required Stable Funding).
type RSFItem struct {
	Codigo string
	Desc   string
	Valor  float64
	Fator  float64 // peso de requerimento (0.0-1.0)
}

// PartialParseErrorDLP indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDLP struct {
	Err error
}

func (e *PartialParseErrorDLP) Error() string { return "parse DLP: " + e.Err.Error() }
func (e *PartialParseErrorDLP) Unwrap() error { return e.Err }

// ParseDocDLP faz parse best-effort do XML DLP.
//
// Estrutura esperada (XSD simplificado):
//
//	<DocDLP cnpj="..." dataBase="...">
//	  <ASFTotal valor="5000.00"/>
//	  <RSFTotal valor="4000.00"/>
//	  <NSFRRatio valor="125.00"/>
//	  <ASFItens>
//	    <ASFItem codigo="1" descricao="Capital" valor="1000.00" fator="1.00"/>
//	    <ASFItem codigo="2" descricao="FinanciamentosLongoPrazo" valor="2000.00" fator="0.80"/>
//	  </ASFItens>
//	  <RSFItens>
//	    <RSFItem codigo="1" descricao="AtivosLiquidos" valor="500.00" fator="0.00"/>
//	    <RSFItem codigo="2" descricao="EmprestimosLongoPrazo" valor="2000.00" fator="0.50"/>
//	  </RSFItens>
//	  <Cenario id="1">
//	    <ASF valor="4800.00"/>
//	    <RSF valor="4000.00"/>
//	    <NSFRRatioCenario valor="120.00"/>
//	  </Cenario>
//	</DocDLP>
func ParseDocDLP(data []byte) (*DocDLP, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	doc := &DocDLP{}
	var cenarioAtual *CenarioNSFR

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDLP{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDLP":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					}
				}
			case "ASFTotal":
				doc.ASFTotal = parseAttrFloat(t.Attr, "valor")
			case "RSFTotal":
				doc.RSFTotal = parseAttrFloat(t.Attr, "valor")
			case "NSFRRatio":
				doc.NSFRRatio = parseAttrFloat(t.Attr, "valor")
			case "Cenario":
				c := CenarioNSFR{}
				idAttr := ""
				for _, a := range t.Attr {
					if a.Name.Local == "id" {
						idAttr = a.Value
					}
				}
				switch idAttr {
				case "1":
					cenarioAtual = &doc.Cenario1
				case "2":
					cenarioAtual = &doc.Cenario2
				default:
					continue
				}
				*cenarioAtual = c
			case "ASF":
				if cenarioAtual != nil {
					cenarioAtual.ASF = parseAttrFloat(t.Attr, "valor")
				}
			case "RSF":
				if cenarioAtual != nil {
					cenarioAtual.RSF = parseAttrFloat(t.Attr, "valor")
				}
			case "NSFRRatioCenario":
				if cenarioAtual != nil {
					cenarioAtual.NSFRRatio = parseAttrFloat(t.Attr, "valor")
				}
			}
		case xml.EndElement:
			if t.Name.Local == "Cenario" {
				cenarioAtual = nil
			}
		}
	}

	return doc, nil
}

// CalcularNSFRRatio calcula NSFR ratio = ASF / RSF × 100.
//
// Retorna -1 se RSF <= 0.
func CalcularNSFRRatio(asf, rsf float64) float64 {
	if rsf <= 0 {
		return -1
	}
	return (asf / rsf) * 100
}

// ValidarNSFRMinimo verifica se NSFR >= 100% (mínimo regulatório BACEN).
func ValidarNSFRMinimo(doc *DocDLP) error {
	if doc == nil {
		return fmt.Errorf("DLP nil")
	}
	if doc.NSFRRatio < 100 && doc.NSFRRatio >= 0 {
		return fmt.Errorf("NSFR ratio=%v%% < 100%% (mínimo regulatório BACEN Res. 4.542)", doc.NSFRRatio)
	}
	return nil
}

// ValidarDLPBasico valida consistência interna do DLP.
func ValidarDLPBasico(doc *DocDLP) error {
	if doc == nil {
		return fmt.Errorf("DLP nil")
	}
	if doc.ASFTotal < 0 {
		return fmt.Errorf("ASFTotal=%v negativo", doc.ASFTotal)
	}
	if doc.RSFTotal < 0 {
		return fmt.Errorf("RSFTotal=%v negativo", doc.RSFTotal)
	}
	// ASF >= RSF é equivalente a NSFR >= 100%
	if doc.ASFTotal < doc.RSFTotal && doc.RSFTotal > 0 {
		return fmt.Errorf("ASFTotal=%v < RSFTotal=%v (NSFR < 100%%)", doc.ASFTotal, doc.RSFTotal)
	}
	return nil
}
