// DocDRL — Documento DRL (Demonstrativo de Liquidez) — CADOC 2160.
//
// Sprint 51: parser completo para DRL (2160) extraindo todas as contas COSIF.
// Estrutura hierárquica: <DocDRL> → <Conta> (HQLA, Outflows, Inflows).
//
// Referência: BACEN Res. 4.605 (LCR — Liquidity Coverage Ratio).
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DocDRL é o documento DRL parseado com estrutura hierárquica COSIF.
type DocDRL struct {
	Root DocDRLRoot

	// Campos agregados (legacy — mantidos para compatibilidade).
	HQLA     float64 // High Quality Liquid Assets (10.x + 11.x)
	Outflows float64 // Saídas de caixa em 30 dias (20.x + 21.x)
	Inflows  float64 // Entradas de caixa em 30 dias (30.x + 31.x)
	LCRRatio float64 // LCR ratio (HQLA / (Outflows - Inflows))
	Cenario1 CenarioLCR
	Cenario2 CenarioLCR
	Cenario3 CenarioLCR

	// Contas COSIF — map[codigoConta]valor.
	// HQLA: 10.x, 11.x | Outflows: 20.x, 21.x | Inflows: 30.x, 31.x
	Accounts map[string]float64
}

// CenarioLCR representa um cenário de estresse para LCR.
type CenarioLCR struct {
	HQLA     float64
	Outflows float64
	Inflows  float64
	LCRRatio float64
}

// DocDRLRoot é o elemento raiz do DRL.
type DocDRLRoot struct {
	CNPJ         string
	DataBase     string // YYYY-MM-DD
	TpDocumento  string // F=full, S=substituição
	NumeroVersao string
}

// PartialParseErrorDRL indica parse parcial bem-sucedido.
type PartialParseErrorDRL struct {
	Err error
}

func (e *PartialParseErrorDRL) Error() string { return "parse DRL: " + e.Err.Error() }
func (e *PartialParseErrorDRL) Unwrap() error { return e.Err }

// ParseDocDRL faz parse completo do XML DRL.
//
// Estrutura COSIF:
//
//	<DocDRL cnpj="..." dataBase="...">
//	  <HQLA valor="1000.00"/>
//	  <Outflows valor="500.00"/>
//	  <Inflows valor="200.00"/>
//	  <LCRRatio valor="333.33"/>
//	  <Conta codigoConta="10.1" valor="100.00"/>
//	  <Conta codigoConta="10.3.1" valor="500.00"/>
//	  <Conta codigoConta="20.1" valor="200.00"/>
//	  <Conta codigoConta="30.1" valor="100.00"/>
//	  <Cenario id="1">
//	    <HQLA valor="900.00"/>
//	    <Outflows valor="550.00"/>
//	    <Inflows valor="180.00"/>
//	  </Cenario>
//	</DocDRL>
func ParseDocDRL(data []byte) (*DocDRL, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &DocDRL{Accounts: make(map[string]float64)}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDRL{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DocDRL":
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
			case "HQLA":
				doc.HQLA = parseAttrFloat(t.Attr, "valor")
			case "Outflows":
				doc.Outflows = parseAttrFloat(t.Attr, "valor")
			case "Inflows":
				doc.Inflows = parseAttrFloat(t.Attr, "valor")
			case "LCRRatio":
				doc.LCRRatio = parseAttrFloat(t.Attr, "valor")
			case "Conta":
				doc.parseConta(dec, t.Attr)
			case "Cenario":
				cenario := &CenarioLCR{}
				idAttr := ""
				for _, a := range t.Attr {
					if a.Name.Local == "id" {
						idAttr = a.Value
					}
				}
				switch idAttr {
				case "1":
					doc.Cenario1 = *cenario
					doc.parseCenario(dec, &doc.Cenario1)
				case "2":
					doc.Cenario2 = *cenario
					doc.parseCenario(dec, &doc.Cenario2)
				case "3":
					doc.Cenario3 = *cenario
					doc.parseCenario(dec, &doc.Cenario3)
				}
			}
		}
	}

	return doc, nil
}

// parseConta processa uma tag <Conta> aninhada.
func (doc *DocDRL) parseConta(dec *xml.Decoder, attrs []xml.Attr) {
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

// parseCenario processa cenários de estresse dentro de <Cenario>.
func (doc *DocDRL) parseCenario(dec *xml.Decoder, c *CenarioLCR) {
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
			case "HQLA":
				c.HQLA = parseAttrFloat(el.Attr, "valor")
			case "Outflows":
				c.Outflows = parseAttrFloat(el.Attr, "valor")
			case "Inflows":
				c.Inflows = parseAttrFloat(el.Attr, "valor")
			case "LCRRatio":
				c.LCRRatio = parseAttrFloat(el.Attr, "valor")
			}
		}
	}
}

// SaldoConta retorna o saldo de uma conta (ou 0 se não existir).
func (doc *DocDRL) SaldoConta(codigo string) float64 {
	if doc == nil || doc.Accounts == nil {
		return 0
	}
	return doc.Accounts[codigo]
}

// SomaHQLA retorna soma de HQLA (contas 10.x + 11.x).
func (doc *DocDRL) SomaHQLA() float64 {
	var total float64
	for acct, val := range doc.Accounts {
		if strings.HasPrefix(acct, "10.") || strings.HasPrefix(acct, "11.") {
			total += val
		}
	}
	return total
}

// SomaOutflows retorna soma de saídas (contas 20.x + 21.x).
func (doc *DocDRL) SomaOutflows() float64 {
	var total float64
	for acct, val := range doc.Accounts {
		if strings.HasPrefix(acct, "20.") || strings.HasPrefix(acct, "21.") {
			total += val
		}
	}
	return total
}

// SomaInflows retorna soma de entradas (contas 30.x + 31.x).
func (doc *DocDRL) SomaInflows() float64 {
	var total float64
	for acct, val := range doc.Accounts {
		if strings.HasPrefix(acct, "30.") || strings.HasPrefix(acct, "31.") {
			total += val
		}
	}
	return total
}

// CalcularLCRRatio calcula LCR ratio = HQLA / (Outflows - Inflows) * 100.
func CalcularLCRRatio(hqla, outflows, inflows float64) float64 {
	denom := outflows - inflows
	if denom <= 0 {
		return -1
	}
	return (hqla / denom) * 100
}

// ValidarLCRMinimo verifica se LCR >= 100% (mínimo regulatório BACEN).
func ValidarLCRMinimo(doc *DocDRL) error {
	if doc == nil {
		return fmt.Errorf("DRL nil")
	}
	if doc.LCRRatio < 100 && doc.LCRRatio >= 0 {
		return fmt.Errorf("LCR ratio=%v%% < 100%% (mínimo regulatório BACEN)", doc.LCRRatio)
	}
	return nil
}

// ValidarDRLBasico valida consistência interna do DRL.
func ValidarDRLBasico(doc *DocDRL) error {
	if doc == nil {
		return fmt.Errorf("DRL nil")
	}
	if doc.HQLA < 0 {
		return fmt.Errorf("HQLA=%v negativo", doc.HQLA)
	}
	if doc.Outflows < 0 {
		return fmt.Errorf("Outflows=%v negativo", doc.Outflows)
	}
	if doc.Inflows < 0 {
		return fmt.Errorf("Inflows=%v negativo", doc.Inflows)
	}
	if doc.Inflows > doc.Outflows && doc.Outflows > 0 {
		return fmt.Errorf("Inflows=%v > Outflows=%v (inconsistência)", doc.Inflows, doc.Outflows)
	}
	return nil
}

// ============================================================
// Regras DRL/2160 — LCR01 a LCR08 + novas
// Sprint 51: Regras usando estrutura hierárquica de contas COSIF.
// ============================================================

// LCR01 — LCR Ratio >= 100% (mínimo regulatório BACEN Res. 4.605).
type LCR01 struct{}

func (LCR01) Code() string     { return "2160-LCR01" }
func (LCR01) Sheet() string    { return "LCR" }
func (LCR01) Severity() string { return "E" }

func (LCR01) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.LCRRatio < 100 && doc.LCRRatio >= 0 {
		return fmt.Errorf("LCR Ratio=%v%% < 100%% (mínimo regulatório BACEN Res. 4.605)", doc.LCRRatio)
	}
	return nil
}

// LCR02 — HQLA >= 0.
type LCR02 struct{}

func (LCR02) Code() string     { return "2160-LCR02" }
func (LCR02) Sheet() string    { return "LCR" }
func (LCR02) Severity() string { return "E" }

func (LCR02) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.HQLA < 0 {
		return fmt.Errorf("HQLA=%v negativo", doc.HQLA)
	}
	return nil
}

// LCR03 — Outflows >= 0.
type LCR03 struct{}

func (LCR03) Code() string     { return "2160-LCR03" }
func (LCR03) Sheet() string    { return "LCR" }
func (LCR03) Severity() string { return "E" }

func (LCR03) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.Outflows < 0 {
		return fmt.Errorf("Outflows=%v negativo", doc.Outflows)
	}
	return nil
}

// LCR04 — Inflows >= 0 e Inflows <= Outflows.
type LCR04 struct{}

func (LCR04) Code() string     { return "2160-LCR04" }
func (LCR04) Sheet() string    { return "LCR" }
func (LCR04) Severity() string { return "E" }

func (LCR04) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil {
		return nil
	}
	if doc.Inflows < 0 {
		return fmt.Errorf("Inflows=%v negativo", doc.Inflows)
	}
	if doc.Inflows > doc.Outflows && doc.Outflows > 0 {
		return fmt.Errorf("Inflows=%v > Outflows=%v (inconsistência)", doc.Inflows, doc.Outflows)
	}
	return nil
}

// LCR05 — LCR Ratio calculado = LCR Ratio declarado (consistência).
type LCR05 struct{}

func (LCR05) Code() string     { return "2160-LCR05" }
func (LCR05) Sheet() string    { return "LCR" }
func (LCR05) Severity() string { return "A" }

func (LCR05) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil {
		return nil
	}
	// Usa contas agregadas se disponíveis
	hqla := doc.HQLA
	outflows := doc.Outflows
	inflows := doc.Inflows
	if hqla == 0 {
		hqla = doc.SomaHQLA()
	}
	if outflows == 0 {
		outflows = doc.SomaOutflows()
	}
	if inflows == 0 {
		inflows = doc.SomaInflows()
	}
	calc := CalcularLCRRatio(hqla, outflows, inflows)
	if calc < 0 {
		return nil
	}
	if calc < doc.LCRRatio*0.99 || calc > doc.LCRRatio*1.01 {
		return fmt.Errorf("LCR declarado=%v%% vs calculado=%v%% (discrepância > 1%%)", doc.LCRRatio, calc)
	}
	return nil
}

// LCR06 — Cenário 1 LCR >= 100%.
type LCR06 struct{}

func (LCR06) Code() string     { return "2160-LCR06" }
func (LCR06) Sheet() string    { return "LCR" }
func (LCR06) Severity() string { return "E" }

func (LCR06) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.Cenario1.LCRRatio < 100 && doc.Cenario1.LCRRatio >= 0 {
		return fmt.Errorf("Cenário 1 LCR=%v%% < 100%%", doc.Cenario1.LCRRatio)
	}
	return nil
}

// LCR07 — Cenário 2 LCR >= 100%.
type LCR07 struct{}

func (LCR07) Code() string     { return "2160-LCR07" }
func (LCR07) Sheet() string    { return "LCR" }
func (LCR07) Severity() string { return "E" }

func (LCR07) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.Cenario2.LCRRatio < 100 && doc.Cenario2.LCRRatio >= 0 {
		return fmt.Errorf("Cenário 2 (adverso) LCR=%v%% < 100%%", doc.Cenario2.LCRRatio)
	}
	return nil
}

// LCR08 — DtBase DRL formato YYYY-MM-DD válido.
type LCR08 struct{}

func (LCR08) Code() string     { return "2160-LCR08" }
func (LCR08) Sheet() string    { return "LCR" }
func (LCR08) Severity() string { return "A" }

func (LCR08) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || doc.Root.DataBase == "" {
		return fmt.Errorf("DRL DtBase vazia")
	}
	if len(doc.Root.DataBase) != 10 || doc.Root.DataBase[4] != '-' || doc.Root.DataBase[7] != '-' {
		return fmt.Errorf("DRL DtBase=%q não está em formato YYYY-MM-DD", doc.Root.DataBase)
	}
	return nil
}

// LCR09 — HQLA declarado vs soma das contas 10.x + 11.x (consistência).
type LCR09HQLAConsistente struct{}

func (LCR09HQLAConsistente) Code() string     { return "2160-LCR09" }
func (LCR09HQLAConsistente) Sheet() string    { return "LCR" }
func (LCR09HQLAConsistente) Severity() string { return "A" }

func (LCR09HQLAConsistente) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || len(doc.Accounts) == 0 {
		return nil
	}
	soma := doc.SomaHQLA()
	if doc.HQLA == 0 || soma == 0 {
		return nil
	}
	diff := doc.HQLA - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > doc.HQLA*0.01 {
		return fmt.Errorf("HQLA agregado=%v vs soma contas HQLA=%.2f (diff=%.2f > 1%%)", doc.HQLA, soma, diff)
	}
	return nil
}

// LCR10 — Outflows agregado vs soma das contas 20.x + 21.x.
type LCR10OutflowsConsistente struct{}

func (LCR10OutflowsConsistente) Code() string     { return "2160-LCR10" }
func (LCR10OutflowsConsistente) Sheet() string    { return "LCR" }
func (LCR10OutflowsConsistente) Severity() string { return "A" }

func (LCR10OutflowsConsistente) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || len(doc.Accounts) == 0 {
		return nil
	}
	soma := doc.SomaOutflows()
	if doc.Outflows == 0 || soma == 0 {
		return nil
	}
	diff := doc.Outflows - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > doc.Outflows*0.01 {
		return fmt.Errorf("Outflows agregado=%v vs soma contas=%.2f (diff=%.2f > 1%%)", doc.Outflows, soma, diff)
	}
	return nil
}

// LCR11 — Inflows agregado vs soma das contas 30.x + 31.x.
type LCR11InflowsConsistente struct{}

func (LCR11InflowsConsistente) Code() string     { return "2160-LCR11" }
func (LCR11InflowsConsistente) Sheet() string    { return "LCR" }
func (LCR11InflowsConsistente) Severity() string { return "A" }

func (LCR11InflowsConsistente) Apply(_ context.Context, doc *DocDRL) error {
	if doc == nil || len(doc.Accounts) == 0 {
		return nil
	}
	soma := doc.SomaInflows()
	if doc.Inflows == 0 || soma == 0 {
		return nil
	}
	diff := doc.Inflows - soma
	if diff < 0 {
		diff = -diff
	}
	if diff > doc.Inflows*0.01 {
		return fmt.Errorf("Inflows agregado=%v vs soma contas=%.2f (diff=%.2f > 1%%)", doc.Inflows, soma, diff)
	}
	return nil
}

// Rule2160 é a interface para regras do CADOC 2160 (DRL/LCR).
type Rule2160 interface {
	Code() string
	Sheet() string
	Severity() string
	Apply(ctx context.Context, doc *DocDRL) error
}

// BuiltinDRL registra todas as regras DRL/2160 no registry fornecido.
func BuiltinDRL(r *Registry) {
	rules := []Rule2160{
		LCR01{}, LCR02{}, LCR03{}, LCR04{}, LCR05{},
		LCR06{}, LCR07{}, LCR08{},
		LCR09HQLAConsistente{},
		LCR10OutflowsConsistente{},
		LCR11InflowsConsistente{},
	}
	for _, rule := range rules {
		r.Register2160(rule)
	}
}

// parsedDRL é variável global para validações cross-doc (set via SetDRL).
//
// V51: globais são aceitáveis para service layer cross-doc.
var parsedDRL *DocDRL

// SetDRL configura o DRL para validações cross-doc (chamado pelo service layer).
func SetDRL(doc *DocDRL) {
	parsedDRL = doc
}
