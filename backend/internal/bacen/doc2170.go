// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDLP is the root element of DLP (Net Stable Funding Ratio).
// CadocCode: 2170
//
// Structure:
// <DocDLP cnpj="..." dataBase="YYYY-MM-DD" tpDocumento="F" numeroVersao="N">
//   <ASFTotal valor="..."/>  <RSFTotal valor="..."/>
//   <NSFRRatio valor="..."/>
//   <Cenario id="N">
//     <ASF valor="..."/>  <RSF valor="..."/>  <NSFRRatio valor="..."/>
//   </Cenario>
//   <Conta codigoConta="..." valor="..."/>
// </DocDLP>
type DocDLP struct {
    XMLName      xml.Name      `xml:"DocDLP"`
    CNPJ         string        `xml:"cnpj,attr"`
    DataBase     string        `xml:"dataBase,attr"`
    TpDocumento  string        `xml:"tpDocumento,attr"`
    NumeroVersao string        `xml:"numeroVersao,attr"`
    ASFTotal     ValorSimples  `xml:"ASFTotal"`
    RSFTotal     ValorSimples  `xml:"RSFTotal"`
    NSFRRatio    ValorSimples  `xml:"NSFRRatio"`
    Cenarios     []CenarioDLP  `xml:"Cenario,omitempty"`
    Contas       []ContaDLP    `xml:"Conta,omitempty"`
}

type CenarioDLP struct {
    ID        string        `xml:"id,attr"`
    ASF       ValorSimples  `xml:"ASF"`
    RSF       ValorSimples  `xml:"RSF"`
    NSFRRatio ValorSimples  `xml:"NSFRRatio"`
}

type ContaDLP struct {
    CodigoConta string `xml:"codigoConta,attr"`
    Valor       string `xml:"valor,attr"`
}

func Parse2170(data []byte) (*DocDLP, error) {
    var doc DocDLP
    return &doc, xml.Unmarshal(data, &doc)
}
