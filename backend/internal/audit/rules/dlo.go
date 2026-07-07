// DocDLO — Documento DLO (Demonstrativo de Limites Operacionais) — subset para cross-doc 2070.
//
// Sprint 39 Fase 2: parser best-effort para DLO (subset de 2061 focado em
// limites operacionais). Suporta conta 770 DLO/2061.
package rules

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DocDLO é o documento DLO parseado (subset limites operacionais).
type DocDLO struct {
	Root        DocDLORoot
	Conta770    float64 // saldo conta 770 (limites)
	LimiteTotal float64 // limite operacional total
	Patrimonio  float64 // patrimônio de referência
}

// DocDLORoot é o elemento raiz do DLO.
type DocDLORoot struct {
	CNPJ     string
	DataBase string // YYYY-MM-DD
}

// PartialParseErrorDLO indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDLO struct {
	Err error
}

func (e *PartialParseErrorDLO) Error() string { return "parse DLO: " + e.Err.Error() }
func (e *PartialParseErrorDLO) Unwrap() error { return e.Err }

// ParseDocDLO faz parse best-effort do XML DLO.
//
// Estrutura esperada (XSD simplificado):
//
//	<DocDLO cnpj="..." dataBase="...">
//	  <Conta770 valor="1000.00"/>
//	  <LimiteTotal valor="5000.00"/>
//	  <Patrimonio valor="3000.00"/>
//	</DocDLO>
func ParseDocDLO(data []byte) (*DocDLO, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	doc := &DocDLO{}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDLO{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDLO":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					}
				}
			case "Conta770":
				doc.Conta770 = parseAttrFloat(t.Attr, "valor")
			case "LimiteTotal":
				doc.LimiteTotal = parseAttrFloat(t.Attr, "valor")
			case "Patrimonio":
				doc.Patrimonio = parseAttrFloat(t.Attr, "valor")
			}
		}
	}

	return doc, nil
}

// parseAttrFloat helper extrai float de atributos XML.
func parseAttrFloat(attrs []xml.Attr, name string) float64 {
	for _, a := range attrs {
		if a.Name.Local == name {
			return parseNum(a.Value)
		}
	}
	return 0
}

// ValidarDLOBasico faz validação básica do DLO (consistência interna).
//
// Função helper usada por regras 2070 cross-doc DLO.
func ValidarDLOBasico(doc *DocDLO) error {
	if doc == nil {
		return fmt.Errorf("DLO nil")
	}
	if doc.Conta770 < 0 {
		return fmt.Errorf("Conta770=%v negativo", doc.Conta770)
	}
	if doc.LimiteTotal < 0 {
		return fmt.Errorf("LimiteTotal=%v negativo", doc.LimiteTotal)
	}
	if doc.Patrimonio < 0 {
		return fmt.Errorf("Patrimonio=%v negativo", doc.Patrimonio)
	}
	// Conta770 <= LimiteTotal (saneamento)
	if doc.Conta770 > doc.LimiteTotal && doc.LimiteTotal > 0 {
		return fmt.Errorf("Conta770=%v > LimiteTotal=%v", doc.Conta770, doc.LimiteTotal)
	}
	return nil
}

// ValidarDRMBasico faz validação básica do DRM (consistência interna).
func ValidarDRMBasico(doc *DocDRM) error {
	if doc == nil {
		return fmt.Errorf("DRM nil")
	}
	if doc.VaR < 0 {
		return fmt.Errorf("VaR=%v negativo", doc.VaR)
	}
	if doc.sVaR < 0 {
		return fmt.Errorf("sVaR=%v negativo", doc.sVaR)
	}
	if doc.RWACOM < 0 {
		return fmt.Errorf("RWACOM=%v negativo", doc.RWACOM)
	}
	// VaR <= sVaR (sanity — VaR é valor atual, sVaR é stress test, geralmente maior)
	if doc.VaR > 0 && doc.sVaR > 0 && doc.VaR > doc.sVaR {
		return fmt.Errorf("VaR=%v > sVaR=%v (esperado VaR <= sVaR em stress)", doc.VaR, doc.sVaR)
	}
	return nil
}

// _ = "context"  // context será usado por regras 2070 cross-doc DLO/DRM
// Placeholder removido em V69 — context usado em regras cross-doc.
