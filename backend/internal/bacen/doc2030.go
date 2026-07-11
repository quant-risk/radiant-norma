// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDRSAC is the root element of DRSAC (Credit Concentration Risk).
// CadocCode: 2030
//
// This type is defined directly in the bacen package to avoid import cycles
// with the drsac package. For full type definitions, see internal/drsac/types.go.
type DocDRSAC struct {
    XMLName      xml.Name        `xml:"Documento2030"`
    CNPJ         string          `xml:"cnpj,attr"`
    DataBase     string          `xml:"dataBase,attr"`
    Subsegmento  string          `xml:"subsegmento,attr"`
    Concentracao []Concentracao   `xml:"Concentracao,omitempty"`
    Ajustes      []Ajuste        `xml:"Ajustes,omitempty"`
}

type Concentracao struct {
    Faixa         string `xml:"faixa,attr"`
    Participantes string `xml:"participantes,attr"`
    Valor         string `xml:"valor,attr"`
}

type Ajuste struct {
    Codigo string `xml:"codigo,attr"`
    Valor  string `xml:"valor,attr"`
}

func Parse2030(data []byte) (*DocDRSAC, error) {
    var doc DocDRSAC
    return &doc, xml.Unmarshal(data, &doc)
}
