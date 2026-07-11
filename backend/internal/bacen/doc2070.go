// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDDR is the root element of DDR (Daily Capital Requirement).
// CadocCode: 2070
//
// Structure:
// <DocDDR cnpj="..." dataBase="YYYY-MM-DD" indRemessa="I"
//          nmContato="..." telContato="...">
//   <DDR codigo="NNNNNN" moeda="XXX" valor="..."/>
//   <DDR codigo="NNNNNN" moeda="XXX" valor="..."/>
// </DocDDR>
type DocDDR struct {
    XMLName    xml.Name `xml:"DocDDR"`
    CNPJ       string   `xml:"cnpj,attr"`
    DataBase   string   `xml:"dataBase,attr"`
    IndRemessa string   `xml:"indRemessa,attr"`
    NmContato  string   `xml:"nmContato,attr"`
    TelContato string   `xml:"telContato,attr"`
    DDRs       []DDR    `xml:"DDR,omitempty"`
}

type DDR struct {
    Codigo string `xml:"codigo,attr"`
    Moeda  string `xml:"moeda,attr"`
    Valor  string `xml:"valor,attr"`
}

func Parse2070(data []byte) (*DocDDR, error) {
    var doc DocDDR
    return &doc, xml.Unmarshal(data, &doc)
}
