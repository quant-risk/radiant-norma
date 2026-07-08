// DRM Leiaute — parser completo para CADOC 2060 (Demonstrativo de Risco de Mercado).
//
// Sprint 55: reescrito para matchear leiaute oficial (67 rows de
// 2060_DRM_Leiaute.xls). Estrutura:
//
//	<DRM>
//	  <IdDocto>2060</IdDocto>
//	  <IdDoctoVersao>v01</IdDoctoVersao>
//	  <DataBase>AAAA-MM</DataBase>
//	  <IdInstFinanc>CNPJ</IdInstFinanc>
//	  <TipoArq>I</TipoArq>  (I=inclusão, A=alteração)
//	  <NomeContato>...</NomeContato>
//	  <FoneContato>...</FoneContato>
//	  <Ativo>
//	    <ItemCarteira Item="..." FatorRisco="..." LocalRegistro="..." CarteiraNegoc="...">
//	      <FluxoVertice CodVertice="..." ValorAlocado="..." ValorMaM="..."/>
//	    </ItemCarteira>
//	  </Ativo>
//	  <Passivo>
//	    <ItemCarteira Item="..." FatorRisco="..." LocalRegistro="..." CarteiraNegoc="...">
//	      <FluxoVertice CodVertice="..." ValorAlocado="..." ValorMaM="..."/>
//	    </ItemCarteira>
//	  </Passivo>
//	  <Derivativo>
//	    <ItemCarteira Item="..." IdPosicao="..." FatorRisco="..." LocalRegistro="..." CarteiraNegoc="...">
//	      <FluxoVertice CodVertice="..." ValorAlocado="..." ValorMaM="..."/>
//	    </ItemCarteira>
//	  </Derivativo>
//	  <AtividadeFinanceira>
//	    <ItemCarteira Item="AEC" IdPosicao="..." FatorRisco="...">
//	      <FluxoVertice CodVertice="..." ValorAlocado="..."/>
//	    </ItemCarteira>
//	  </AtividadeFinanceira>
//	</DRM>
//
// Regras DRM-01 a DRM-07 implementadas. Parser completo.
//
// Referência: BACEN — 2060_DRM_Leiaute.xls (67 rows extraídas).
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DocDRMLeiaute é o documento DRM parseado com estrutura completa do leiaute.
type DocDRMLeiaute struct {
	Root DocDRMLeiauteRoot

	// Seções do documento
	Ativos                []ItemCarteiraLeiaute
	Passivos              []ItemCarteiraLeiaute
	Derivativos           []ItemCarteiraLeiaute
	AtividadesFinanceiras []ItemCarteiraLeiaute

	// Agregados
	TotalValorAlocado float64
	TotalValorMaM     float64
}

// DocDRMLeiauteRoot representa o header do DRM.
type DocDRMLeiauteRoot struct {
	IdDocto       string // "2060"
	IdDoctoVersao string // "v01" etc
	DataBase      string // AAAA-MM
	IdInstFinanc  string // CNPJ (8 dígitos)
	TipoArq       string // "I" ou "A"
	NomeContato   string
	FoneContato   string
}

// ItemCarteiraLeiaute representa um <ItemCarteira> com FluxoVertice children.
type ItemCarteiraLeiaute struct {
	Item          string // Table 016/017
	IdPosicao     string // "C" ou "V" — só em Derivativos
	FatorRisco    string // Table 011
	LocalRegistro string // Table 012
	CarteiraNegoc string // Table 013
	Fluxos        []FluxoVerticeLeiaute
}

// FluxoVerticeLeiaute representa um <FluxoVertice>.
type FluxoVerticeLeiaute struct {
	CodVertice   string
	ValorAlocado float64
	ValorMaM     float64 // só requerido quando CodVertice=12
}

// PartialParseErrorDRMLEIAUTE indica parse parcial bem-sucedido.
type PartialParseErrorDRMLEIAUTE struct{ Err error }

func (e *PartialParseErrorDRMLEIAUTE) Error() string { return "parse DRM leiaute: " + e.Err.Error() }
func (e *PartialParseErrorDRMLEIAUTE) Unwrap() error { return e.Err }

// ParseDocDRMLeiaute faz parse completo do XML DRM (leiaute oficial).
func ParseDocDRMLeiaute(data []byte) (*DocDRMLeiaute, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &DocDRMLeiaute{}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseErrorDRMLEIAUTE{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "DRM":
				// root element — no attributes needed
			case "IdDocto":
				doc.Root.IdDocto = strings.TrimSpace(readCharDataDRM(dec))
			case "IdDoctoVersao":
				doc.Root.IdDoctoVersao = strings.TrimSpace(readCharDataDRM(dec))
			case "DataBase":
				doc.Root.DataBase = strings.TrimSpace(readCharDataDRM(dec))
			case "IdInstFinanc":
				doc.Root.IdInstFinanc = strings.TrimSpace(readCharDataDRM(dec))
			case "TipoArq":
				doc.Root.TipoArq = strings.TrimSpace(readCharDataDRM(dec))
			case "NomeContato":
				doc.Root.NomeContato = strings.TrimSpace(readCharDataDRM(dec))
			case "FoneContato":
				doc.Root.FoneContato = strings.TrimSpace(readCharDataDRM(dec))
			case "Ativo":
				items := parseItemCarteiraListLeiaute(dec, "Ativo")
				doc.Ativos = append(doc.Ativos, items...)
			case "Passivo":
				items := parseItemCarteiraListLeiaute(dec, "Passivo")
				doc.Passivos = append(doc.Passivos, items...)
			case "Derivativo":
				items := parseItemCarteiraListLeiaute(dec, "Derivativo")
				doc.Derivativos = append(doc.Derivativos, items...)
			case "AtividadeFinanceira":
				items := parseItemCarteiraListLeiaute(dec, "AtividadeFinanceira")
				doc.AtividadesFinanceiras = append(doc.AtividadesFinanceiras, items...)
			}
		}
	}

	doc.computeAggregates()
	return doc, nil
}

func readCharDataDRM(dec *xml.Decoder) string {
	tok, err := dec.Token()
	if err != nil {
		return ""
	}
	if cd, ok := tok.(xml.CharData); ok {
		return string(cd)
	}
	return ""
}

func parseItemCarteiraListLeiaute(dec *xml.Decoder, container string) []ItemCarteiraLeiaute {
	var items []ItemCarteiraLeiaute
	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if t.Name.Local == "ItemCarteira" {
				item := parseItemCarteiraLeiaute(dec, t)
				items = append(items, item)
			}
		case xml.EndElement:
			depth--
			if t.Name.Local == container && depth == 0 {
				break
			}
		}
	}
	return items
}

func parseItemCarteiraLeiaute(dec *xml.Decoder, start xml.StartElement) ItemCarteiraLeiaute {
	item := ItemCarteiraLeiaute{}
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "Item":
			item.Item = a.Value
		case "IdPosicao":
			item.IdPosicao = a.Value
		case "FatorRisco":
			item.FatorRisco = a.Value
		case "LocalRegistro":
			item.LocalRegistro = a.Value
		case "CarteiraNegoc":
			item.CarteiraNegoc = a.Value
		}
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "FluxoVertice" {
				fv := FluxoVerticeLeiaute{}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "CodVertice":
						fv.CodVertice = a.Value
					case "ValorAlocado":
						fv.ValorAlocado = parseNumDRMLEIAUTE(a.Value)
					case "ValorMaM":
						fv.ValorMaM = parseNumDRMLEIAUTE(a.Value)
					}
				}
				item.Fluxos = append(item.Fluxos, fv)
			}
		case xml.EndElement:
			if t.Name.Local == "ItemCarteira" {
				goto done
			}
		}
	}
done:
	return item
}

func parseNumDRMLEIAUTE(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Detecta formato: se tem vírgula com dígitos depois, é brasileiro (1.234,56).
	// Caso contrário, é formato padrão US/ISO (1000.00 ou 1000).
	hasComma := strings.Contains(s, ",")
	hasDot := strings.Contains(s, ".")
	if hasComma && hasDot {
		// Brasileiro: 1.234,56 → remove pontos → 1234,56 → vírgula → 1234.56
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if hasComma {
		// Brasileiro sem ponto: 1234,56
		s = strings.ReplaceAll(s, ",", ".")
	}
	// Caso contrário: formato padrão (1000.00 ou 1000) — usa direto
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func (d *DocDRMLeiaute) computeAggregates() {
	appendAndSum := func(items []ItemCarteiraLeiaute) {
		for _, item := range items {
			for _, fv := range item.Fluxos {
				d.TotalValorAlocado += fv.ValorAlocado
				d.TotalValorMaM += fv.ValorMaM
			}
		}
	}
	appendAndSum(d.Ativos)
	appendAndSum(d.Passivos)
	appendAndSum(d.Derivativos)
	// AtividadesFinanceiras não têm ValorMaM
	for _, item := range d.AtividadesFinanceiras {
		for _, fv := range item.Fluxos {
			d.TotalValorAlocado += fv.ValorAlocado
		}
	}
}

// ============================================================
// RuleDRM — interface para regras de validação do CADOC 2060 (DRM).
// ============================================================

type RuleDRM interface {
	Code() string
	Sheet() string
	Severity() string // E, A, I
	Apply(ctx context.Context, doc *DocDRMLeiaute) error
}

// DRM01HeaderValido — campos obrigatórios do header.
type DRM01HeaderValido struct{}

func (DRM01HeaderValido) Code() string     { return "DRM-01" }
func (DRM01HeaderValido) Sheet() string    { return "Estrutura" }
func (DRM01HeaderValido) Severity() string { return "E" }

func (DRM01HeaderValido) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	if doc.Root.IdDocto != "2060" {
		return fmt.Errorf("IdDocto=%q, esperado 2060", doc.Root.IdDocto)
	}
	if doc.Root.DataBase == "" {
		return fmt.Errorf("DataBase ausente")
	}
	if doc.Root.IdInstFinanc == "" {
		return fmt.Errorf("IdInstFinanc (CNPJ) ausente")
	}
	if doc.Root.TipoArq != "I" && doc.Root.TipoArq != "A" {
		return fmt.Errorf("TipoArq=%q inválido (I ou A)", doc.Root.TipoArq)
	}
	return nil
}

// DRM02ItensObrigatorios — pelo menos uma seção tem itens.
type DRM02ItensObrigatorios struct{}

func (DRM02ItensObrigatorios) Code() string     { return "DRM-02" }
func (DRM02ItensObrigatorios) Sheet() string    { return "Estrutura" }
func (DRM02ItensObrigatorios) Severity() string { return "E" }

func (DRM02ItensObrigatorios) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	hasItems := len(doc.Ativos) > 0 || len(doc.Passivos) > 0 ||
		len(doc.Derivativos) > 0 || len(doc.AtividadesFinanceiras) > 0
	if !hasItems {
		return fmt.Errorf("documento DRM sem itens (Ativo, Passivo, Derivativo ou AtividadeFinanceira)")
	}
	return nil
}

// DRM03ItemFormatValid — Item e FatorRisco presentes.
type DRM03ItemFormatValid struct{}

func (DRM03ItemFormatValid) Code() string     { return "DRM-03" }
func (DRM03ItemFormatValid) Sheet() string    { return "Estrutura" }
func (DRM03ItemFormatValid) Severity() string { return "E" }

func (DRM03ItemFormatValid) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	allItems := append(append(append([]ItemCarteiraLeiaute{}, doc.Ativos...), doc.Passivos...), doc.Derivativos...)
	for i, item := range allItems {
		if item.Item == "" {
			return fmt.Errorf("ItemCarteira[%d]: Item ausente", i)
		}
		if item.FatorRisco == "" {
			return fmt.Errorf("ItemCarteira[%d]: FatorRisco ausente", i)
		}
	}
	return nil
}

// DRM04ValorMaMRequerido — ValorMaM requerido quando CodVertice=12.
type DRM04ValorMaMRequerido struct{}

func (DRM04ValorMaMRequerido) Code() string     { return "DRM-04" }
func (DRM04ValorMaMRequerido) Sheet() string    { return "Consistência" }
func (DRM04ValorMaMRequerido) Severity() string { return "E" }

func (DRM04ValorMaMRequerido) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	allItems := append(append(append([]ItemCarteiraLeiaute{}, doc.Ativos...), doc.Passivos...), doc.Derivativos...)
	for i, item := range allItems {
		for j, fv := range item.Fluxos {
			if fv.CodVertice == "12" && fv.ValorMaM == 0 {
				return fmt.Errorf("Item[%d] Fluxo[%d]: CodVertice=12 requer ValorMaM", i, j)
			}
		}
	}
	return nil
}

// DRM05ValorAlocadoPositivo — ValorAlocado não-negativo.
type DRM05ValorAlocadoPositivo struct{}

func (DRM05ValorAlocadoPositivo) Code() string     { return "DRM-05" }
func (DRM05ValorAlocadoPositivo) Sheet() string    { return "Consistência" }
func (DRM05ValorAlocadoPositivo) Severity() string { return "E" }

func (DRM05ValorAlocadoPositivo) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	allItems := append(append(append([]ItemCarteiraLeiaute{}, doc.Ativos...), doc.Passivos...), doc.Derivativos...)
	for i, item := range allItems {
		for j, fv := range item.Fluxos {
			if fv.ValorAlocado < 0 {
				return fmt.Errorf("Item[%d] Fluxo[%d]: ValorAlocado negativo=%v", i, j, fv.ValorAlocado)
			}
		}
	}
	return nil
}

// DRM06AtividadeFinanceiraSemMaM — AtividadeFinanceira não deve ter ValorMaM.
type DRM06AtividadeFinanceiraSemMaM struct{}

func (DRM06AtividadeFinanceiraSemMaM) Code() string     { return "DRM-06" }
func (DRM06AtividadeFinanceiraSemMaM) Sheet() string    { return "Consistência" }
func (DRM06AtividadeFinanceiraSemMaM) Severity() string { return "A" }

func (DRM06AtividadeFinanceiraSemMaM) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	for i, item := range doc.AtividadesFinanceiras {
		for j, fv := range item.Fluxos {
			if fv.ValorMaM != 0 {
				return fmt.Errorf("AtividadeFinanceira[%d] Fluxo[%d]: não deve ter ValorMaM (esperado 0)", i, j)
			}
		}
	}
	return nil
}

// DRM07FatorRiscoValido — FatorRisco em Table 011.
type DRM07FatorRiscoValido struct{}

func (DRM07FatorRiscoValido) Code() string     { return "DRM-07" }
func (DRM07FatorRiscoValido) Sheet() string    { return "Consistência" }
func (DRM07FatorRiscoValido) Severity() string { return "A" }

func (DRM07FatorRiscoValido) Apply(_ context.Context, doc *DocDRMLeiaute) error {
	validFactors := map[string]bool{
		"JM1": true, "JM2": true, "JM3": true, "JM4": true, "JM5": true,
		"JM7": true, "JM9": true,
		"JT1": true, "JT2": true, "JT3": true, "JT9": true,
		"JI1": true, "JI2": true, "JI9": true,
		"FF1": true,
	}
	allItems := append(append(append([]ItemCarteiraLeiaute{}, doc.Ativos...), doc.Passivos...), doc.Derivativos...)
	for i, item := range allItems {
		if !validFactors[item.FatorRisco] {
			return fmt.Errorf("Item[%d]: FatorRisco=%q desconhecido", i, item.FatorRisco)
		}
	}
	return nil
}

// BuiltinDRM registra as regras DRM no registry.
func BuiltinDRM(r *Registry) {
	rules := []RuleDRM{
		DRM01HeaderValido{},
		DRM02ItensObrigatorios{},
		DRM03ItemFormatValid{},
		DRM04ValorMaMRequerido{},
		DRM05ValorAlocadoPositivo{},
		DRM06AtividadeFinanceiraSemMaM{},
		DRM07FatorRiscoValido{},
	}
	for _, rule := range rules {
		r.RegisterDRM(rule)
	}
}
