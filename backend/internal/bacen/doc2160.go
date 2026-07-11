// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDRL is the root element of DRL (Liquidity Coverage Ratio).
// CadocCode: 2160
//
// Structure:
// <DocDRL cnpj="..." dataBase="YYYY-MM-DD" tpDocumento="F" numeroVersao="N">
//
//	<HQLA valor="..."/>  <Outflows valor="..."/>  <Inflows valor="..."/>
//	<LCRRatio valor="..."/>
//	<Cenario id="N">
//	  <HQLA valor="..."/>  <Outflows valor="..."/>  <Inflows valor="..."/>
//	  <LCRRatio valor="..."/>
//	</Cenario>
//	<Conta codigoConta="..." valor="..."/>
//
// </DocDRL>
type DocDRL struct {
	XMLName      xml.Name     `xml:"DocDRL"`
	CNPJ         string       `xml:"cnpj,attr"`
	DataBase     string       `xml:"dataBase,attr"`
	TpDocumento  string       `xml:"tpDocumento,attr"`
	NumeroVersao string       `xml:"numeroVersao,attr"`
	HQLA         ValorSimples `xml:"HQLA"`
	Outflows     ValorSimples `xml:"Outflows"`
	Inflows      ValorSimples `xml:"Inflows"`
	LCRRatio     ValorSimples `xml:"LCRRatio"`
	Cenarios     []Cenario    `xml:"Cenario,omitempty"`
	Contas       []ContaDRL   `xml:"Conta,omitempty"`
}

type Cenario struct {
	ID       string       `xml:"id,attr"`
	HQLA     ValorSimples `xml:"HQLA"`
	Outflows ValorSimples `xml:"Outflows"`
	Inflows  ValorSimples `xml:"Inflows"`
	LCRRatio ValorSimples `xml:"LCRRatio"`
}

type ContaDRL struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
}

func Parse2160(data []byte) (*DocDRL, error) {
	var doc DocDRL
	return &doc, xml.Unmarshal(data, &doc)
}
