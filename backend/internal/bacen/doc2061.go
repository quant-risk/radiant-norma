// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDLO is the root element of DLO (Operational Limits).
// CadocCode: 2061
//
// Structure:
// <DocDLO cnpj="..." dataBase="YYYY-MM-DD" tpDocumento="F" numeroVersao="N">
//
//	<Conta770 valor="..."/>  <LimiteTotal valor="..."/>
//	<Patrimonio valor="..."/>
//	<Conta codigoConta="..." valor="...">
//	  <Elem codigoElem="N" descElem="..." valor="..." peso="..."/>
//	</Conta>
//	<Totalizador contasTotal="N" rwacamTotal="..." patrimonioTotal="..."/>
//
// </DocDLO>
type DocDLO struct {
	XMLName      xml.Name     `xml:"DocDLO"`
	CNPJ         string       `xml:"cnpj,attr"`
	DataBase     string       `xml:"dataBase,attr"`
	TpDocumento  string       `xml:"tpDocumento,attr"`
	NumeroVersao string       `xml:"numeroVersao,attr"`
	Conta770     ValorSimples `xml:"Conta770,omitempty"`
	LimiteTotal  ValorSimples `xml:"LimiteTotal,omitempty"`
	Patrimonio   ValorSimples `xml:"Patrimonio,omitempty"`
	Contas       []Conta      `xml:"Conta,omitempty"`
	Totaliz      *TotalizDLO  `xml:"Totalizador,omitempty"`
}

type TotalizDLO struct {
	ContasTotal     string `xml:"contasTotal,attr"`
	RWACAMTotal     string `xml:"rwacamTotal,attr"`
	PatrimonioTotal string `xml:"patrimonioTotal,attr"`
}

type Conta struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
	Elems       []Elem `xml:"Elem,omitempty"`
}

type Elem struct {
	CodigoElem string `xml:"codigoElem,attr"`
	DescElem   string `xml:"descElem,attr"`
	Valor      string `xml:"valor,attr"`
	Peso       string `xml:"peso,attr,omitempty"`
}

func Parse2061(data []byte) (*DocDLO, error) {
	var doc DocDLO
	return &doc, xml.Unmarshal(data, &doc)
}
