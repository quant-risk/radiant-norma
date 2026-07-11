// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// DocDRM is the root element of DRM (Market Risk).
// CadocCode: 2060
//
// Structure:
// <DocDRM cnpj="..." dataBase="YYYY-MM-DD">
//
//	<RWAJUR1 valor="..."/>  <RWAJUR2 valor="..."/>
//	<RWAJUR3 valor="..."/>  <RWAJUR4 valor="..."/>
//	<VaR valor="..."/>  <sVaR valor="..."/>
//	<RWACOM valor="..."/>
//	<Posicao codigo="..." moeda="..." valor="..."/>
//
// </DocDRM>
type DocDRM struct {
	XMLName  xml.Name     `xml:"DocDRM"`
	CNPJ     string       `xml:"cnpj,attr"`
	DataBase string       `xml:"dataBase,attr"`
	RWAJUR1  ValorSimples `xml:"RWAJUR1"`
	RWAJUR2  ValorSimples `xml:"RWAJUR2"`
	RWAJUR3  ValorSimples `xml:"RWAJUR3"`
	RWAJUR4  ValorSimples `xml:"RWAJUR4"`
	VaR      ValorSimples `xml:"VaR"`
	SVaR     ValorSimples `xml:"sVaR"`
	RWACOM   ValorSimples `xml:"RWACOM"`
	Posicoes []Posicao    `xml:"Posicao,omitempty"`
}

type Posicao struct {
	Codigo string `xml:"codigo,attr"`
	Moeda  string `xml:"moeda,attr"`
	Valor  string `xml:"valor,attr"`
}

func Parse2060(data []byte) (*DocDRM, error) {
	var doc DocDRM
	return &doc, xml.Unmarshal(data, &doc)
}
