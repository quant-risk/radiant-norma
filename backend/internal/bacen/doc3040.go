// Package bacen define os tipos XML para todos os CADOCs do BACEN.
//
// Estes tipos são o contrato compartilhado entre o motor de geração
// (CADOCGenerator) e as regras de validação cross-doc.
//
// Ao usar encoding/xml com estes tipos, eliminamos:
//   - String-scraping com regex frágil
//   - Descasamento entre o XML que o gerador emite e o que as regras leem
//   - Inconsistências de caixa (CNPJ vs cnpj vs Cnpj)
//
// Layouts baseados nos Leiautes Oficiais do BACEN.
// Onde o XSD oficial não está disponível, a estrutura segue o parser
// cross-doc (internal/crossdoc/rules/) e referências públicas BACEN.
package bacen

import (
	"encoding/xml"
	"fmt"
)

// Doc3040 é o root element do SCR (Risco de Crédito).
// CadocCode: 3040
//
// Estrutura (child-element style para compatibilidade com cross-doc rules):
//
//	<Doc3040 cnpj="..." dataBase="YYYY-MM-DD" remessa="N" parte="N"
//	          tpArq="F" nomeResp="..." emailResp="..." telResp="..."
//	          totalCli="N">
//	  <Agreg>
//	    <NatuOp>N</NatuOp>          <Mod>N</Mod>
//	    <OrigemRec>N</OrigemRec>    <VincME>C</VincME>
//	    <ClassOp>C</ClassOp>         <FaixaVlr>N</FaixaVlr>
//	    <PrzProvm>C</PrzProvm>       <Localiz>FF</Localiz>
//	    <TpCli>N</TpCli>            <DesempOp>N</DesempOp>
//	    <ProvConsttd>NNNNN.NN</ProvConsttd>
//	    <QtdOp>N</QtdOp>            <QtdCli>N</QtdCli>
//	    <Venc>
//	      <V110>NNNNN.NN</V110>    <V120>NNNNN.NN</V120>
//	      <V130>NNNNN.NN</V130>    <V140>NNNNN.NN</V140>
//	      <V150>NNNNN.NN</V150>    <V160>NNNNN.NN</V160>
//	      <V165>NNNNN.NN</V165>
//	    </Venc>
//	  </Agreg>
//	  <Totalizador provTotal="..." qtdOpTotal="N" qtdCliTotal="N"
//	               V110="..." V120="..." V130="..." V140="..."
//	               V150="..." V160="..." V165="..."/>
//	</Doc3040>
type Doc3040 struct {
	XMLName   xml.Name  `xml:"Doc3040"`
	CNPJ      string   `xml:"cnpj,attr"`
	DataBase  string   `xml:"dataBase,attr"`
	Remessa   string   `xml:"remessa,attr"`
	Parte     string   `xml:"parte,attr"`
	TpArq     string   `xml:"tpArq,attr"`
	NomeResp  string   `xml:"nomeResp,attr"`
	EmailResp string   `xml:"emailResp,attr"`
	TelResp   string   `xml:"telResp,attr"`
	TotalCli  string   `xml:"totalCli,attr"`
	Agregadas []Agreg `xml:"Agreg,omitempty"`
	Totaliz   *Totaliz `xml:"Totalizador,omitempty"`
}

// Agreg representa um bloco de operações agregadas no SCR.
// Unique por (NatuOp, Mod, Localiz, TpCli, DesempOp, ClassOp, FaixaVlr, PrzProvm).
// Os campos IPOC, CNPJ e Saldo existem apenas para compatibilidade com o
// DRSAC (2030) que cruza dados com o SCR — no SCR padrão não são usados.
type Agreg struct {
	NatuOp      string `xml:"NatuOp,omitempty"`
	Mod         string `xml:"Mod,omitempty"`
	OrigemRec   string `xml:"OrigemRec,omitempty"`
	VincME      string `xml:"VincME,omitempty"`
	ClassOp     string `xml:"ClassOp,omitempty"`
	FaixaVlr    string `xml:"FaixaVlr,omitempty"`
	PrzProvm    string `xml:"PrzProvm,omitempty"`
	Localiz     string `xml:"Localiz,omitempty"`
	TpCli       string `xml:"TpCli,omitempty"`
	DesempOp    string `xml:"DesempOp,omitempty"`
	ProvConsttd string `xml:"ProvConsttd,omitempty"`
	QtdOp       string `xml:"QtdOp,omitempty"`
	QtdCli      string `xml:"QtdCli,omitempty"`
	IPOC        string `xml:"IPOC,omitempty"` // DRSAC cross-ref
	CNPJSCR     string `xml:"CNPJ,omitempty"` // DRSAC cross-ref
	Saldo       string `xml:"Saldo,omitempty"` // DRSAC cross-ref
	Venc        Venc   `xml:"Venc"`
}

// Venc contém as 7 faixas de vencimento (N1 a N7).
type Venc struct {
	V110 string `xml:"V110,omitempty"` // vencidos até 3 meses
	V120 string `xml:"V120,omitempty"` // vencidos 3-6 meses
	V130 string `xml:"V130,omitempty"` // vencidos 6-12 meses
	V140 string `xml:"V140,omitempty"` // vencidos 1-3 anos
	V150 string `xml:"V150,omitempty"` // vencidos 3-5 anos (inadimplência >90d)
	V160 string `xml:"V160,omitempty"` // vencidos 5-10 anos
	V165 string `xml:"V165,omitempty"` // vencidos mais de 10 anos
}

// Totaliz é o bloco de totalização do SCR.
type Totaliz struct {
	ProvTotal   string `xml:"provTotal,attr"`
	QtdOpTotal  string `xml:"qtdOpTotal,attr"`
	QtdCliTotal string `xml:"qtdCliTotal,attr"`
	V110        string `xml:"V110,attr"`
	V120        string `xml:"V120,attr"`
	V130        string `xml:"V130,attr"`
	V140        string `xml:"V140,attr"`
	V150        string `xml:"V150,attr"`
	V160        string `xml:"V160,attr"`
	V165        string `xml:"V165,attr"`
}

// Parse3040 unmarshals XML bytes into a Doc3040.
func Parse3040(data []byte) (*Doc3040, error) {
	var doc Doc3040
	return &doc, xml.Unmarshal(data, &doc)
}

// QtdOpTotal returns the sum of all QtdOp values across all Agreg blocks.
func (d *Doc3040) QtdOpTotal() float64 {
	var total float64
	for _, a := range d.Agregadas {
		var q int
		fmt.Sscanf(a.QtdOp, "%d", &q)
		total += float64(q)
	}
	return total
}

// CountV150Gt0 returns the number of Agreg blocks where V150 > 0.
// Used by cross-doc rule XD-4111-03.
func (d *Doc3040) CountV150Gt0() int {
	count := 0
	for _, a := range d.Agregadas {
		var v float64
		fmt.Sscanf(a.Venc.V150, "%f", &v)
		if v > 0 {
			count++
		}
	}
	return count
}

// HasMod0213 returns true if any Agreg has Mod == "0213".
// Used by cross-doc rule XD-002.
func (d *Doc3040) HasMod0213() bool {
	for _, a := range d.Agregadas {
		if a.Mod == "0213" {
			return true
		}
	}
	return false
}

// SCRData carries the subset of 3040 data needed by DRSAC cross-doc rules.
// Mirrors drsac.SCRData but lives here to avoid import cycle.
type SCRData struct {
	Saldo             string
	CNAE              string
	HasCliente        bool
	HasHighRiskFlag   bool
	HasCollateral     bool
	IsGreenInstrument bool
}

// ScrData returns a map of IPOC → SCRData for DRSAC cross-reference validation.
func (d *Doc3040) ScrData() map[string]SCRData {
	result := make(map[string]SCRData)
	for _, a := range d.Agregadas {
		if a.IPOC != "" {
			result[a.IPOC] = SCRData{
				Saldo:      a.Saldo,
				CNAE:       "",
				HasCliente: a.CNPJSCR != "",
			}
		}
	}
	return result
}
