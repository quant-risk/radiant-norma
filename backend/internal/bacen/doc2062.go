// Package bacen — see doc3040.go for package docs.
package bacen

import "encoding/xml"

// documentoDLI is the root element of DLI (Individual Limits).
// CadocCode: 2062
//
// Structure:
// <documentoDLI cnpj="..." dataBase="YYYY-MM" codigoDocumento="2062" tipoEnvio="I">
//
//	<limitesInformados>
//	  <limite codigoLimite="NN.NN" enviado="S|N">VALOR</limite>
//	  ...
//	</limitesInformados>
//	<parametros>
//	  <parametro codigo="NN">VALOR</parametro>
//	</parametros>
//	<contas>
//	  <conta codigoConta="X.XX.XX" valor="..."/>
//	</contas>
//
// </documentoDLI>
type documentoDLI struct {
	XMLName         xml.Name   `xml:"documentoDLI"`
	CNPJ            string     `xml:"cnpj,attr"`
	DataBase        string     `xml:"dataBase,attr"`
	CodigoDocumento string     `xml:"codigoDocumento,attr"`
	TipoEnvio       string     `xml:"tipoEnvio,attr"`
	Limites         Limites    `xml:"limitesInformados"`
	Parametros      Parametros `xml:"parametros"`
	Contas          Contas     `xml:"contas"`
}

type Limites struct {
	Limite []Limite `xml:"limite"`
}

type Limite struct {
	Codigo  string `xml:"codigoLimite,attr"`
	Enviado string `xml:"enviado,attr"` // "S" or "N"
	Value   string `xml:",innerxml"`    // the text content (the value)
}

type Parametros struct {
	Parametro []Parametro `xml:"parametro"`
}

type Parametro struct {
	Codigo string `xml:"codigo,attr"`
	Value  string `xml:",innerxml"`
}

type Contas struct {
	Conta []ContaDLI `xml:"conta"`
}

type ContaDLI struct {
	CodigoConta string `xml:"codigoConta,attr"`
	Valor       string `xml:"valor,attr"`
}

func Parse2062(data []byte) (*documentoDLI, error) {
	var doc documentoDLI
	return &doc, xml.Unmarshal(data, &doc)
}
