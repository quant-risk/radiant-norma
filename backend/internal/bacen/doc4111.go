// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// Documento4111 is the root element of COSIF (Credit Portfolio).
// CadocCode: 4111
//
// Structure:
// <Documento4111 cnpj="..." dataBase="YYYY-MM-DD" codigoDocumento="4111">
//   <Cliente>
//     <QtdCli>N</QtdCli>
//     <CNPJ>NNNNNNNN</CNPJ>
//     <Modalidade codigo="NNNN" indicacao="S|N"/>
//   </Cliente>
//   <Totalizador qtdCliTotal="N" modTotal="N" qtdModTotal="N"/>
// </Documento4111>
type Documento4111 struct {
    XMLName   xml.Name    `xml:"Documento4111"`
    CNPJ      string      `xml:"cnpj,attr"`
    DataBase  string      `xml:"dataBase,attr"`
    CodigoDoc string      `xml:"codigoDocumento,attr"`
    Clientes  []Cliente   `xml:"Cliente,omitempty"`
    Totaliz   *Totaliz4111 `xml:"Totalizador,omitempty"`
}

type Totaliz4111 struct {
    QtdCliTotal string `xml:"qtdCliTotal,attr"`
    ModTotal    string `xml:"modTotal,attr"`
    QtdModTotal string `xml:"qtdModTotal,attr"`
}

type Cliente struct {
    QtdCli     string       `xml:"QtdCli"`
    CNPJ       string       `xml:"CNPJ,omitempty"`
    Modalidade []Modalidade `xml:"Modalidade,omitempty"`
}

type Modalidade struct {
    Codigo    string `xml:"codigo,attr"`
    Indicacao string `xml:"indicacao,attr"`
}

func Parse4111(data []byte) (*Documento4111, error) {
    var doc Documento4111
    return &doc, xml.Unmarshal(data, &doc)
}
