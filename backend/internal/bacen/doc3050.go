// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocTXB is the root element of TXB (Taxas de Operações Bancárias).
// CadocCode: 3050
//
// Structure:
// <DocTXB cnpjInstituicao="..." dataBase="YYYY-MM-DD" indRemessa="I"
//          nmContato="..." telContato="...">
//   <Referencia codigo="YYYYMM">
//     <Diario>
//       <CRDLivre>
//         <PesJuridica>
//           <Flu codigo="..." txMedJuros="..." txMinima="..." txMaxima="..."
//                vlrConcessoes="..." qtdNovContratos="..." sldCarAtiva="..."/>
//           <Pre .../>  <Vc .../>  <Ind .../>
//         </PesJuridica>
//         <PesFisica>
//           <Flu .../> <Pre .../> <Vc .../> <Ind .../>
//         </PesFisica>
//       </CRDLivre>
//     </Diario>
//     <Mensal> (same structure) </Mensal>
//   </Referencia>
// </DocTXB>
type DocTXB struct {
    XMLName    xml.Name   `xml:"DocTXB"`
    CNPJ       string     `xml:"cnpjInstituicao,attr"`
    DataBase   string     `xml:"dataBase,attr"`
    IndRemessa string     `xml:"indRemessa,attr"`
    NmContato  string     `xml:"nmContato,attr"`
    TelContato string     `xml:"telContato,attr"`
    Referencia Referencia `xml:"Referencia"`
}

type Referencia struct {
    Codigo string `xml:"codigo,attr"`
    Diario Secao  `xml:"Diario"`
    Mensal Secao  `xml:"Mensal"`
}

type Secao struct {
    CRDLivre CRDLivre `xml:"CRDLivre"`
}

type CRDLivre struct {
    PesJuridica ClienteBloco `xml:"PesJuridica"`
    PesFisica   ClienteBloco `xml:"PesFisica"`
}

type ClienteBloco struct {
    Pre []SubModalidade `xml:"Pre,omitempty"`
    Flu []SubModalidade `xml:"Flu,omitempty"`
    Vc  []SubModalidade `xml:"Vc,omitempty"`
    Ind []SubModalidade `xml:"Ind,omitempty"`
}

type SubModalidade struct {
    Codigo          string `xml:"codigo,attr"`
    TxMedJuros      string `xml:"txMedJuros,attr"`
    TxMedJurosAj    string `xml:"txMedJurosAjustada,attr,omitempty"`
    TxMedEncFiscais string `xml:"txMedEncFiscais,attr,omitempty"`
    TxMedEncOper    string `xml:"txMedEncOperacionais,attr,omitempty"`
    TxMinima        string `xml:"txMinima,attr"`
    TxMaxima        string `xml:"txMaxima,attr"`
    VlrConcessoes   string `xml:"vlrConcessoes,attr"`
    PrzDecMedConc   string `xml:"przDecMedConcessoes,attr,omitempty"`
    QtdNovContratos string `xml:"qtdNovContratos,attr"`
    SldCarAtiva     string `xml:"sldCarAtiva,attr"`
    SldCedido       string `xml:"sldCedido,attr,omitempty"`
    SldAdquirido    string `xml:"sldAdquirido,attr,omitempty"`
}

func Parse3050(data []byte) (*DocTXB, error) {
    var doc DocTXB
    return &doc, xml.Unmarshal(data, &doc)
}
