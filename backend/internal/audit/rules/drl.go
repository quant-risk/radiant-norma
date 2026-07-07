// DocDRL — Documento DRL (Demonstrativo de Liquidez) — CADOC 2160.
//
// Sprint 40: parser best-effort para DRL (subset de 2160 focado em LCR
// — Liquidity Coverage Ratio). Suporta HQLA, Outflows, Inflows, LCR ratio.
//
// Referência: BACEN Res. 4.605 (LCR — Liquidity Coverage Ratio).
package rules

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DocDRL é o documento DRL parseado (subset LCR).
type DocDRL struct {
	Root     DocDRLRoot
	HQLA     float64 // High Quality Liquid Assets
	Outflows float64 // Saídas de caixa em 30 dias
	Inflows  float64 // Entradas de caixa em 30 dias
	LCRRatio float64 // LCR ratio (HQLA / (Outflows - Inflows))
	// Cenários de estresse
	Cenario1 CenarioLCR // Cenário base
	Cenario2 CenarioLCR // Cenário adverso
	Cenario3 CenarioLCR // Cenário idêntico ao LCR original
}

// DocDRLRoot é o elemento raiz do DRL.
type DocDRLRoot struct {
	CNPJ     string
	DataBase string // YYYY-MM-DD
}

// CenarioLCR representa um cenário de estresse para LCR.
type CenarioLCR struct {
	HQLA     float64
	Outflows float64
	Inflows  float64
	LCRRatio float64
}

// PartialParseErrorDRL indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDRL struct {
	Err error
}

func (e *PartialParseErrorDRL) Error() string { return "parse DRL: " + e.Err.Error() }
func (e *PartialParseErrorDRL) Unwrap() error { return e.Err }

// ParseDocDRL faz parse best-effort do XML DRL.
//
// Estrutura esperada (XSD simplificado):
//
//	<DocDRL cnpj="..." dataBase="...">
//	  <HQLA valor="1000.00"/>
//	  <Outflows valor="500.00"/>
//	  <Inflows valor="200.00"/>
//	  <LCRRatio valor="333.33"/>
//	  <Cenario id="1">
//	    <HQLA valor="900.00"/>
//	    <Outflows valor="550.00"/>
//	    <Inflows valor="180.00"/>
//	  </Cenario>
//	</DocDRL>
func ParseDocDRL(data []byte) (*DocDRL, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	doc := &DocDRL{}
	var cenarioAtual *CenarioLCR

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDRL{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDRL":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					}
				}
			case "Cenario":
				c := CenarioLCR{}
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
				case "3":
					cenarioAtual = &doc.Cenario3
				default:
					continue
				}
				*cenarioAtual = c
			case "HQLA":
				if cenarioAtual != nil {
					cenarioAtual.HQLA = parseAttrFloat(t.Attr, "valor")
				} else {
					doc.HQLA = parseAttrFloat(t.Attr, "valor")
				}
			case "Outflows":
				if cenarioAtual != nil {
					cenarioAtual.Outflows = parseAttrFloat(t.Attr, "valor")
				} else {
					doc.Outflows = parseAttrFloat(t.Attr, "valor")
				}
			case "Inflows":
				if cenarioAtual != nil {
					cenarioAtual.Inflows = parseAttrFloat(t.Attr, "valor")
				} else {
					doc.Inflows = parseAttrFloat(t.Attr, "valor")
				}
			case "LCRRatio":
				if cenarioAtual != nil {
					cenarioAtual.LCRRatio = parseAttrFloat(t.Attr, "valor")
				} else {
					doc.LCRRatio = parseAttrFloat(t.Attr, "valor")
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

// CalcularLCRRatio calcula LCR ratio = HQLA / (Outflows - Inflows) * 100.
//
// Retorna -1 se denominador <= 0.
func CalcularLCRRatio(hqla, outflows, inflows float64) float64 {
	denominador := outflows - inflows
	if denominador <= 0 {
		return -1
	}
	return (hqla / denominador) * 100
}

// ValidarLCRMinimo verifica se LCR >= 100% (mínimo regulatório BACEN).
func ValidarLCRMinimo(doc *DocDRL) error {
	if doc == nil {
		return fmt.Errorf("DRL nil")
	}
	if doc.LCRRatio < 100 && doc.LCRRatio >= 0 {
		return fmt.Errorf("LCR ratio=%v%% < 100%% (mínimo regulatório BACEN)", doc.LCRRatio)
	}
	return nil
}

// ValidarDRLBasico valida consistência interna do DRL.
func ValidarDRLBasico(doc *DocDRL) error {
	if doc == nil {
		return fmt.Errorf("DRL nil")
	}
	if doc.HQLA < 0 {
		return fmt.Errorf("HQLA=%v negativo", doc.HQLA)
	}
	if doc.Outflows < 0 {
		return fmt.Errorf("Outflows=%v negativo", doc.Outflows)
	}
	if doc.Inflows < 0 {
		return fmt.Errorf("Inflows=%v negativo", doc.Inflows)
	}
	if doc.Inflows > doc.Outflows && doc.Outflows > 0 {
		return fmt.Errorf("Inflows=%v > Outflows=%v (inconsistência)", doc.Inflows, doc.Outflows)
	}
	return nil
}
