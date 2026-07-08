// DocDLP — Documento DLP (Demonstrativo de Liquidez de Longo Prazo) — CADOC 2170.
//
// Sprint 51: parser completo para DLP (2170) extraindo todas as contas COSIF.
// Estrutura: <DocDLP> → <Conta> (ASF — Available Stable Funding, RSF — Required).
//
// Referência: BACEN Res. 4.542 (NSFR — Net Stable Funding Ratio).
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DocDLP é o documento DLP parseado com estrutura hierárquica COSIF.
type DocDLP struct {
	Root DocDLPRoot

	// Campos agregados (legacy).
	ASFTotal  float64
	RSFTotal  float64
	NSFRRatio float64
	Cenario1  CenarioNSFR
	Cenario2  CenarioNSFR

	// Contas COSIF — map[codigoConta]valor.
	// ASF: contas 30.x | RSF: contas 40.x
	Accounts map[string]float64
}

// DocDLPRoot é o elemento raiz do DLP.
type DocDLPRoot struct {
	CNPJ         string
	DataBase     string // YYYY-MM-DD
	TpDocumento  string
	NumeroVersao string
}

// CenarioNSFR representa um cenário de estresse para NSFR.
type CenarioNSFR struct {
	ASF       float64
	RSF       float64
	NSFRRatio float64
}

// PartialParseErrorDLP indica parse parcial bem-sucedido (D-26).
type PartialParseErrorDLP struct {
	Err error
}

func (e *PartialParseErrorDLP) Error() string { return "parse DLP: " + e.Err.Error() }
func (e *PartialParseErrorDLP) Unwrap() error { return e.Err }

// ParseDocDLP faz parse completo do XML DLP.
//
// Estrutura COSIF:
//
//	<DocDLP cnpj="..." dataBase="...">
//	  <ASFTotal valor="5000.00"/>
//	  <RSFTotal valor="4000.00"/>
//	  <NSFRRatio valor="125.00"/>
//	  <Conta codigoConta="30.01" valor="1000.00"/>
//	  <Conta codigoConta="30.02" valor="2000.00"/>
//	  <Conta codigoConta="40.01" valor="500.00"/>
//	  <Cenario id="1">
//	    <ASF valor="4800.00"/>
//	    <RSF valor="4000.00"/>
//	    <NSFRRatioCenario valor="120.00"/>
//	  </Cenario>
//	</DocDLP>
func ParseDocDLP(data []byte) (*DocDLP, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &DocDLP{Accounts: make(map[string]float64)}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDLP{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDLP":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpj":
						doc.Root.CNPJ = a.Value
					case "dataBase":
						doc.Root.DataBase = a.Value
					case "tpDocumento":
						doc.Root.TpDocumento = a.Value
					case "numeroVersao":
						doc.Root.NumeroVersao = a.Value
					}
				}
			case "ASFTotal":
				doc.ASFTotal = parseAttrFloat(t.Attr, "valor")
			case "RSFTotal":
				doc.RSFTotal = parseAttrFloat(t.Attr, "valor")
			case "NSFRRatio":
				doc.NSFRRatio = parseAttrFloat(t.Attr, "valor")
			case "Conta":
				doc.parseConta(dec, t.Attr)
			case "Cenario":
				c := CenarioNSFR{}
				idAttr := ""
				for _, a := range t.Attr {
					if a.Name.Local == "id" {
						idAttr = a.Value
					}
				}
				switch idAttr {
				case "1":
					doc.Cenario1 = c
					doc.parseCenario(dec, &doc.Cenario1)
				case "2":
					doc.Cenario2 = c
					doc.parseCenario(dec, &doc.Cenario2)
				}
			}
		}
	}

	return doc, nil
}

// parseConta processa <Conta> com seus elementos.
func (doc *DocDLP) parseConta(dec *xml.Decoder, attrs []xml.Attr) {
	var codigo string
	var valor float64
	for _, a := range attrs {
		if a.Name.Local == "codigoConta" {
			codigo = a.Value
		}
		if a.Name.Local == "valor" {
			valor = parseNum(a.Value)
		}
	}
	if codigo != "" {
		doc.Accounts[codigo] = valor
	}
}

// parseCenario processa cenários de estresse.
func (doc *DocDLP) parseCenario(dec *xml.Decoder, c *CenarioNSFR) {
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if el, ok := tok.(xml.EndElement); ok && el.Name.Local == "Cenario" {
			break
		}
		if el, ok := tok.(xml.StartElement); ok {
			switch el.Name.Local {
			case "ASF":
				c.ASF = parseAttrFloat(el.Attr, "valor")
			case "RSF":
				c.RSF = parseAttrFloat(el.Attr, "valor")
			case "NSFRRatioCenario":
				c.NSFRRatio = parseAttrFloat(el.Attr, "valor")
			}
		}
	}
}

// SaldoConta retorna saldo de uma conta (ou 0).
func (doc *DocDLP) SaldoConta(codigo string) float64 {
	if doc == nil || doc.Accounts == nil {
		return 0
	}
	return doc.Accounts[codigo]
}

// SomaASF retorna soma das contas ASF (30.x).
func (doc *DocDLP) SomaASF() float64 {
	var total float64
	for acct, val := range doc.Accounts {
		if strings.HasPrefix(acct, "30.") {
			total += val
		}
	}
	return total
}

// SomaRSF retorna soma das contas RSF (40.x).
func (doc *DocDLP) SomaRSF() float64 {
	var total float64
	for acct, val := range doc.Accounts {
		if strings.HasPrefix(acct, "40.") {
			total += val
		}
	}
	return total
}

// CalcularNSFRRatio calcula NSFR ratio = ASF / RSF × 100.
func CalcularNSFRRatio(asf, rsf float64) float64 {
	if rsf <= 0 {
		return -1
	}
	return (asf / rsf) * 100
}

// ValidarNSFRMinimo verifica se NSFR >= 100%.
func ValidarNSFRMinimo(doc *DocDLP) error {
	if doc == nil {
		return fmt.Errorf("DLP nil")
	}
	if doc.NSFRRatio < 100 && doc.NSFRRatio >= 0 {
		return fmt.Errorf("NSFR ratio=%v%% < 100%% (mínimo regulatório BACEN Res. 4.542)", doc.NSFRRatio)
	}
	return nil
}

// ValidarDLPBasico valida consistência interna do DLP.
func ValidarDLPBasico(doc *DocDLP) error {
	if doc == nil {
		return fmt.Errorf("DLP nil")
	}
	if doc.ASFTotal < 0 {
		return fmt.Errorf("ASFTotal=%v negativo", doc.ASFTotal)
	}
	if doc.RSFTotal < 0 {
		return fmt.Errorf("RSFTotal=%v negativo", doc.RSFTotal)
	}
	if doc.ASFTotal < doc.RSFTotal && doc.RSFTotal > 0 {
		return fmt.Errorf("ASFTotal=%v < RSFTotal=%v (NSFR < 100%%)", doc.ASFTotal, doc.RSFTotal)
	}
	return nil
}

// ============================================================
// Regras DLP/2170 — NSFR01 a NSFR08 + novas
// Sprint 51: Regras usando estrutura hierárquica de contas COSIF.
// ============================================================

// NSFR01 — NSFR Ratio >= 100% (mínimo regulatório BACEN Res. 4.542).
type NSFR01 struct{}

func (NSFR01) Code() string     { return "2170-NSFR01" }
func (NSFR01) Sheet() string    { return "NSFR" }
func (NSFR01) Severity() string { return "E" }

func (NSFR01) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.NSFRRatio < 100 && doc.NSFRRatio >= 0 {
		return fmt.Errorf("NSFR Ratio=%v%% < 100%% (mínimo regulatório BACEN Res. 4.542)", doc.NSFRRatio)
	}
	return nil
}

// NSFR02 — ASF Total >= 0.
type NSFR02 struct{}

func (NSFR02) Code() string     { return "2170-NSFR02" }
func (NSFR02) Sheet() string    { return "NSFR" }
func (NSFR02) Severity() string { return "E" }

func (NSFR02) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.ASFTotal < 0 {
		return fmt.Errorf("ASF Total=%v negativo", doc.ASFTotal)
	}
	return nil
}

// NSFR03 — RSF Total >= 0.
type NSFR03 struct{}

func (NSFR03) Code() string     { return "2170-NSFR03" }
func (NSFR03) Sheet() string    { return "NSFR" }
func (NSFR03) Severity() string { return "E" }

func (NSFR03) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.RSFTotal < 0 {
		return fmt.Errorf("RSF Total=%v negativo", doc.RSFTotal)
	}
	return nil
}

// NSFR04 — ASF >= RSF (equivalente a NSFR >= 100%).
type NSFR04 struct{}

func (NSFR04) Code() string     { return "2170-NSFR04" }
func (NSFR04) Sheet() string    { return "NSFR" }
func (NSFR04) Severity() string { return "E" }

func (NSFR04) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil {
		return nil
	}
	if doc.ASFTotal < doc.RSFTotal && doc.RSFTotal > 0 {
		return fmt.Errorf("ASF Total=%v < RSF Total=%v (NSFR < 100%%)", doc.ASFTotal, doc.RSFTotal)
	}
	return nil
}

// NSFR05 — NSFR Ratio calculado = NSFR Ratio declarado (consistência).
type NSFR05 struct{}

func (NSFR05) Code() string     { return "2170-NSFR05" }
func (NSFR05) Sheet() string    { return "NSFR" }
func (NSFR05) Severity() string { return "A" }

func (NSFR05) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil {
		return nil
	}
	asf := doc.ASFTotal
	rsf := doc.RSFTotal
	if asf == 0 {
		asf = doc.SomaASF()
	}
	if rsf == 0 {
		rsf = doc.SomaRSF()
	}
	calc := CalcularNSFRRatio(asf, rsf)
	if calc < 0 {
		return nil
	}
	if calc < doc.NSFRRatio*0.99 || calc > doc.NSFRRatio*1.01 {
		return fmt.Errorf("NSFR declarado=%v%% vs calculado=%v%% (discrepância > 1%%)", doc.NSFRRatio, calc)
	}
	return nil
}

// NSFR06 — Cenário 1 ASF >= 0.
type NSFR06 struct{}

func (NSFR06) Code() string     { return "2170-NSFR06" }
func (NSFR06) Sheet() string    { return "NSFR" }
func (NSFR06) Severity() string { return "E" }

func (NSFR06) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.Cenario1.ASF < 0 {
		return fmt.Errorf("Cenário 1 ASF=%v negativo", doc.Cenario1.ASF)
	}
	return nil
}

// NSFR07 — Cenário 1 RSF >= 0.
type NSFR07 struct{}

func (NSFR07) Code() string     { return "2170-NSFR07" }
func (NSFR07) Sheet() string    { return "NSFR" }
func (NSFR07) Severity() string { return "E" }

func (NSFR07) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.Cenario1.RSF < 0 {
		return fmt.Errorf("Cenário 1 RSF=%v negativo", doc.Cenario1.RSF)
	}
	return nil
}

// NSFR08 — DtBase DLP formato YYYY-MM-DD válido.
type NSFR08 struct{}

func (NSFR08) Code() string     { return "2170-NSFR08" }
func (NSFR08) Sheet() string    { return "NSFR" }
func (NSFR08) Severity() string { return "A" }

func (NSFR08) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || doc.Root.DataBase == "" {
		return fmt.Errorf("DLP DtBase vazia")
	}
	if len(doc.Root.DataBase) != 10 || doc.Root.DataBase[4] != '-' || doc.Root.DataBase[7] != '-' {
		return fmt.Errorf("DLP DtBase=%q não está em formato YYYY-MM-DD", doc.Root.DataBase)
	}
	return nil
}

// NSFR09 — ASF agregado vs soma das contas ASF (consistência).
type NSFR09ASFConsistente struct{}

func (NSFR09ASFConsistente) Code() string     { return "2170-NSFR09" }
func (NSFR09ASFConsistente) Sheet() string    { return "NSFR" }
func (NSFR09ASFConsistente) Severity() string { return "A" }

func (NSFR09ASFConsistente) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || len(doc.Accounts) == 0 {
		return nil
	}
	soma := doc.SomaASF()
	if doc.ASFTotal == 0 || soma == 0 {
		return nil
	}
	diff := doc.ASFTotal - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > doc.ASFTotal*0.01 {
		return fmt.Errorf("ASF agregado=%v vs soma contas ASF=%.2f (diff=%.2f > 1%%)", doc.ASFTotal, soma, diff)
	}
	return nil
}

// NSFR10 — RSF agregado vs soma das contas RSF (consistência).
type NSFR10RSFConsistente struct{}

func (NSFR10RSFConsistente) Code() string     { return "2170-NSFR10" }
func (NSFR10RSFConsistente) Sheet() string    { return "NSFR" }
func (NSFR10RSFConsistente) Severity() string { return "A" }

func (NSFR10RSFConsistente) Apply(_ context.Context, doc *DocDLP) error {
	if doc == nil || len(doc.Accounts) == 0 {
		return nil
	}
	soma := doc.SomaRSF()
	if doc.RSFTotal == 0 || soma == 0 {
		return nil
	}
	diff := doc.RSFTotal - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > doc.RSFTotal*0.01 {
		return fmt.Errorf("RSF agregado=%v vs soma contas RSF=%.2f (diff=%.2f > 1%%)", doc.RSFTotal, soma, diff)
	}
	return nil
}

// Rule2170 é a interface para regras do CADOC 2170 (DLP/NSFR).
type Rule2170 interface {
	Code() string
	Sheet() string
	Severity() string
	Apply(ctx context.Context, doc *DocDLP) error
}

// BuiltinDLP registra todas as regras DLP/2170 no registry fornecido.
func BuiltinDLP(r *Registry) {
	rules := []Rule2170{
		NSFR01{}, NSFR02{}, NSFR03{}, NSFR04{}, NSFR05{},
		NSFR06{}, NSFR07{}, NSFR08{},
		NSFR09ASFConsistente{},
		NSFR10RSFConsistente{},
	}
	for _, rule := range rules {
		r.Register2170(rule)
	}
}

// parsedDLP é variável global para validações cross-doc (set via SetDLP).
//
// V51: globais são aceitáveis para service layer cross-doc.
var parsedDLP *DocDLP

// SetDLP configura o DLP para validações cross-doc (chamado pelo service layer).
func SetDLP(doc *DocDLP) {
	parsedDLP = doc
}
