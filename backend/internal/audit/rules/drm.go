// DocDRM — Documento DRM (Demonstrativo de Risco de Mercado) — subset para cross-doc 2070.
//
// Sprint 39 Fase 2: parser best-effort para DRM (subset de 3050 focado em
// risco mercado). Suporta RWAJUR1, RWAJUR2, RWAJUR3, RWAJUR4, VaR, sVaR, RWACOM.
//
// Referência: BACEN Res. 4.557 (Requerimento Mínimo de Patrimônio de Referência).
package rules

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
)

// DocDRM é o documento DRM parseado (subset risco mercado).
type DocDRM struct {
	Root     DocDRMRoot
	RWAJUR1  float64        // RWA Jur 1 — VaR
	RWAJUR2  float64        // RWA Jur 2 — RWAJUR2/3/4 vs DRM
	RWAJUR3  float64        // RWA Jur 3
	RWAJUR4  float64        // RWA Jur 4
	VaR      float64        // Value at Risk
	sVaR     float64        // Stressed VaR
	RWACOM   float64        // RWA Commodity
	Posicoes []PosicaoMoeda // posições moedas (pares Codigo + Moeda)
}

// DocDRMRoot é o elemento raiz do DRM.
type DocDRMRoot struct {
	CNPJ     string
	DataBase string // YYYY-MM-DD
}

// PosicaoMoeda representa uma posição em moeda no DRM.
type PosicaoMoeda struct {
	Codigo string
	Moeda  string
	Valor  float64
}

// PartialParseErrorDRM indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDRM struct {
	Err error
}

func (e *PartialParseErrorDRM) Error() string { return "parse DRM: " + e.Err.Error() }
func (e *PartialParseErrorDRM) Unwrap() error { return e.Err }

// ParseDocDRM faz parse best-effort do XML DRM.
//
// Estrutura esperada (XSD simplificado):
//
//	<DocDRM cnpj="..." dataBase="...">
//	  <RWAJUR1 valor="100.00"/>
//	  <RWAJUR2 valor="50.00"/>
//	  <RWAJUR3 valor="30.00"/>
//	  <RWAJUR4 valor="20.00"/>
//	  <VaR valor="40.00"/>
//	  <sVaR valor="60.00"/>
//	  <RWACOM valor="80.00"/>
//	  <Posicao codigo="161000" moeda="USD" valor="100.00"/>
//	</DocDRM>
func ParseDocDRM(data []byte) (*DocDRM, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	doc := &DocDRM{}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDRM{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDRM":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					}
				}
			case "RWAJUR1":
				doc.RWAJUR1 = parseAttrFloat(t.Attr, "valor")
			case "RWAJUR2":
				doc.RWAJUR2 = parseAttrFloat(t.Attr, "valor")
			case "RWAJUR3":
				doc.RWAJUR3 = parseAttrFloat(t.Attr, "valor")
			case "RWAJUR4":
				doc.RWAJUR4 = parseAttrFloat(t.Attr, "valor")
			case "VaR":
				doc.VaR = parseAttrFloat(t.Attr, "valor")
			case "sVaR":
				doc.sVaR = parseAttrFloat(t.Attr, "valor")
			case "RWACOM":
				doc.RWACOM = parseAttrFloat(t.Attr, "valor")
			case "Posicao":
				pos := PosicaoMoeda{}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "codigo":
						pos.Codigo = a.Value
					case "moeda":
						pos.Moeda = a.Value
					case "valor":
						pos.Valor = parseNum(a.Value)
					}
				}
				doc.Posicoes = append(doc.Posicoes, pos)
			}
		}
	}

	return doc, nil
}

// DocDLO está definido em dlo.go (separado por organização).
// (continua em dlo.go)
