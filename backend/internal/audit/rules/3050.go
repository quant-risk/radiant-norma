// Package rules — Sprint 33 / v3.34.0 (TXB_V11) Fase 1.
//
// Doc3050 é o documento CADOC 3050 (TXB - Taxas de Operações Bancárias)
// parseado conforme schema BACEN TXB_V4 e críticas TXB_V11.
//
// Decisões arquiteturais (ver SPRINT_33_RESEARCH.md §D-24/D-25/D-26/D-27):
//   - D-24: Rule3050 interface paralela a Rule (não quebrar 3040).
//   - D-25: Modalidade achatada (sem hierarquia pesJuridica→pre→desDuplicatas).
//   - D-26: Parser XML tolerante (best-effort, nil-safe).
//   - D-27: Stubs honestos com severity "I" (D-13 do Sprint 32 v3.30.0).
//
// Esta Fase 1 entrega:
//   - Doc3050 + Modalidade structs.
//   - ParseDoc3050 (parser XML best-effort).
//   - 14 regras Agregadas (A01-A14).
//   - 14 stubs (S01-S14).
//   - Total: 28 regras (16.5% cobertura do catálogo de 170).
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Doc3050 é o documento CADOC 3050 parseado.
//
// Estrutura:
//   - Root: atributos do cabeçalho (CNPJ, dataBase, indRemessa, etc).
//   - Diario: modalidades diárias achatadas (todas sub-modalidades × encargos × tipos cliente).
//   - Mensal: modalidades mensais achatadas.
type Doc3050 struct {
	Root   Doc3050Root
	Diario []Modalidade
	Mensal []Modalidade
}

// Doc3050Root é o elemento raiz <DocTXB>.
type Doc3050Root struct {
	CNPJ       string // cnpjInstituicao — 8 dígitos BACEN (raiz CNPJ)
	DataBase   string // dataBase — formato YYYY-MM-DD
	IndRemessa string // indRemessa — I (inclusão), A (alteração), S (substituição)
	NmContato  string // nmContato — obrigatório
	TelContato string // telContato — obrigatório
}

// Modalidade representa uma sub-modalidade (desDuplicatas, capGirPrzAte365, etc) do
// documento 3050 com seus atributos (taxas, valores, prazos, saldos) achatados.
//
// Codigo: nome do elemento XML (desDuplicatas, capGirPrzAte365, vendor, etc).
// Encargo: tipo de encargo financeiro (pre, flu, vc, ind).
// TipoCli: tipo de cliente (pesJuridica, pesFisica).
//
// Campos opcionais (nil = não-preenchido no XML):
//   - Modelo 1 (Diário): TxMedJuros, ..., PrzDecMedConcessoes, QtdNovContratos, SldCarAtiva, SldCedido, SldAdquirido.
//   - Modelo 2 (Diário): mesmo do Modelo 1 sem PrzDecMedConcessoes.
//   - Modelo 3 (Diário): só VlrConcessoes, PrzDecMedConcessoes, QtdNovContratos, SldCarAtiva (+ opcionais).
//   - Modelo 4 (Diário): só VlrConcessoes, QtdNovContratos, SldCarAtiva (+ opcionais).
//   - Mensais: SldBaiPrejuizo, SldCarAte14/60/90/Maior90, QtdConAte14/60/90/Maior90, PrzMedCarteira (+ mais).
//
// Uso de *float64 / *int: distinção entre "0" (preenchido com zero) e "ausente" (não veio no XML).
type Modalidade struct {
	Codigo  string // desDuplicatas, capGirPrzAte365, vendor, etc
	Encargo string // pre, flu, vc, ind
	TipoCli string // pesJuridica, pesFisica

	// Diários (modelos 1-4)
	TxMedJuros           *float64
	TxMedJurosAjustada   *float64 // txMedJurosAjustada — regra 3051 (S24 carry-over Fase 3)
	TxMedEncFiscais      *float64
	TxMedEncOperacionais *float64
	TxMinima             *float64
	TxMaxima             *float64
	VlrConcessoes        *float64
	PrzDecMedConcessoes  *int
	QtdNovContratos      *int
	SldCarAtiva          *float64
	SldCedido            *float64
	SldAdquirido         *float64

	// Mensais (modelos 1-4)
	SldBaiPrejuizo *float64
	SldCarAte14    *float64
	SldCarAte60    *float64
	SldCarAte90    *float64
	SldCarMaior90  *float64
	QtdConAte14    *int
	QtdConAte60    *int
	QtdConAte90    *int
	QtdConMaior90  *int
	PrzMedCarteira *int
}

// ============================================================================
// Parser XML
// ============================================================================

// ParseDoc3050 faz parse best-effort do XML CADOC 3050 conforme schema TXB_V4.
//
// Retorna (*Doc3050, nil) em sucesso completo ou (*Doc3050, *PartialParseError)
// em sucesso parcial (alguns elementos não reconhecidos). Erros fatais de XML
// malformado retornam (nil, err).
//
// Decisão D-26: best-effort é aceitável porque regras de Fase 1 cobrem subset.
// Stubs documentam carry-over para cobertura completa.
//
// Estratégia: como o XSD tem 4 modelos de atributos diferentes e cada
// sub-modalidade é um elemento XML distinto (desDuplicatas, capGirPrzAte365, etc),
// usamos `[]xml.Element` + reflection do nome via xml.Name.Local para descobrir
// o Codigo. Atributos comuns a todos os modelos são lidos em xml3050Attrs.
func ParseDoc3050(data []byte) (*Doc3050, error) {
	// 1ª passada: parse raw para extrair tokens de elementos + atributos
	type rawAttr struct {
		XMLName  xml.Name
		Attrs    []xml.Attr
		Children []rawAttr
	}

	// Decodifica como árvore genérica usando encoding/xml
	dec := xml.NewDecoder(bytes.NewReader(data))

	var (
		root          Doc3050Root
		stack         []rawAttr
		doc           = &Doc3050{Root: root}
		currentParent *rawAttr // ptr pro elemento-pai atual na stack
	)

	// Implementação manual: stream de StartElement/EndElement/CharData
	//
	// Estrutura do XSD:
	//   DocTXB
	//     referencia (1-5)
	//       diario
	//         crdLivre
	//           pesJuridica | pesFisica
	//             pre | flu | vc | ind
	//               <sub-modalidade attrs.../>   <-- aqui mora o Modalidade
	//       mensal
	//         crdLivre
	//           pesJuridica | pesFisica
	//             pre | flu | vc | ind
	//               <sub-modalidade attrs.../>
	//
	// Cada <sub-modalidade> tem o mesmo conjunto de atributos (união dos 4 modelos).
	// Quando abrimos um elemento "folha" (sem children relevantes), lemos seus atributos.

	_ = currentParent
	_ = stack

	// Mapemento de path → (period, encargo, tipoCli)
	var (
		currentPath    []string
		currentEncargo string
		currentTipoCli string
		currentPeriod  string // "diario" ou "mensal"
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse 3050: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			currentPath = append(currentPath, tag)

			// Header do documento (atributos do <DocTXB>)
			if tag == "DocTXB" {
				root = Doc3050Root{}
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "cnpjInstituicao":
						root.CNPJ = a.Value
					case "dataBase":
						root.DataBase = a.Value
					case "indRemessa":
						root.IndRemessa = a.Value
					case "nmContato":
						root.NmContato = a.Value
					case "telContato":
						root.TelContato = a.Value
					}
				}
				doc.Root = root
			}

			// Path tracking para contexto
			switch tag {
			case "diario":
				currentPeriod = "diario"
			case "mensal":
				currentPeriod = "mensal"
			case "pesJuridica":
				currentTipoCli = "pesJuridica"
			case "pesFisica":
				currentTipoCli = "pesFisica"
			case "pre":
				currentEncargo = "pre"
			case "flu":
				currentEncargo = "flu"
			case "vc":
				currentEncargo = "vc"
			case "ind":
				currentEncargo = "ind"
			}

			// Detecta sub-modalidade (filha direta de pre/flu/vc/ind):
			// a) tem parent na stack que é pre/flu/vc/ind
			// b) não tem children relevantes (folha)
			// c) tem pelo menos 1 atributo de modelo (txMedJuros, vlrConcessoes, etc)
			if len(currentPath) >= 5 && isEncargo(currentPath[len(currentPath)-2]) {
				attrs := parse3050Attrs(t.Attr)
				if isSubModalidade(attrs) {
					m := attrs.toModalidade(tag, currentEncargo, currentTipoCli)
					if currentPeriod == "diario" {
						doc.Diario = append(doc.Diario, m)
					} else if currentPeriod == "mensal" {
						doc.Mensal = append(doc.Mensal, m)
					}
				}
			}

		case xml.EndElement:
			if len(currentPath) > 0 {
				// Pop e ajusta contexto
				popped := currentPath[len(currentPath)-1]
				currentPath = currentPath[:len(currentPath)-1]
				switch popped {
				case "diario", "mensal":
					currentPeriod = ""
				case "pesJuridica", "pesFisica":
					currentTipoCli = ""
				case "pre", "flu", "vc", "ind":
					currentEncargo = ""
				}
			}
		}
	}

	if doc.Root.CNPJ == "" && len(doc.Diario) == 0 && len(doc.Mensal) == 0 {
		return nil, fmt.Errorf("parse 3050: documento vazio")
	}

	return doc, nil
}

// isEncargo retorna true se tag é um elemento de encargo (pre, flu, vc, ind).
func isEncargo(tag string) bool {
	switch tag {
	case "pre", "flu", "vc", "ind":
		return true
	}
	return false
}

// isSubModalidade detecta se os attrs preenchidos correspondem a uma
// sub-modalidade (vs. elemento vazio ou grupo).
func isSubModalidade(a xml3050Attrs) bool {
	return a.TxMedJuros != nil || a.TxMinima != nil || a.VlrConcessoes != nil ||
		a.SldCarAtiva != nil || a.SldCarAte14 != nil || a.PrzMedCarteira != nil
}

// parse3050Attrs converte []xml.Attr em xml3050Attrs usando parseFloatPtr.
func parse3050Attrs(attrs []xml.Attr) xml3050Attrs {
	a := xml3050Attrs{}
	for _, attr := range attrs {
		switch attr.Name.Local {
		case "txMedJuros":
			a.TxMedJuros = parseFloatPtr(attr.Value)
		case "txMedJurosAjustada":
			a.TxMedJurosAjustada = parseFloatPtr(attr.Value)
		case "txMedEncFiscais":
			a.TxMedEncFiscais = parseFloatPtr(attr.Value)
		case "txMedEncOperacionais":
			a.TxMedEncOperacionais = parseFloatPtr(attr.Value)
		case "txMinima":
			a.TxMinima = parseFloatPtr(attr.Value)
		case "txMaxima":
			a.TxMaxima = parseFloatPtr(attr.Value)
		case "vlrConcessoes":
			a.VlrConcessoes = parseFloatPtr(attr.Value)
		case "przDecMedConcessoes":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.PrzDecMedConcessoes = &v
			}
		case "qtdNovContratos":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.QtdNovContratos = &v
			}
		case "sldCarAtiva":
			a.SldCarAtiva = parseFloatPtr(attr.Value)
		case "sldCedido":
			a.SldCedido = parseFloatPtr(attr.Value)
		case "sldAdquirido":
			a.SldAdquirido = parseFloatPtr(attr.Value)
		case "sldBaiPrejuizo":
			a.SldBaiPrejuizo = parseFloatPtr(attr.Value)
		case "sldCarAte14":
			a.SldCarAte14 = parseFloatPtr(attr.Value)
		case "sldCarAte60":
			a.SldCarAte60 = parseFloatPtr(attr.Value)
		case "sldCarAte90":
			a.SldCarAte90 = parseFloatPtr(attr.Value)
		case "sldCarMaior90":
			a.SldCarMaior90 = parseFloatPtr(attr.Value)
		case "qtdConAte14":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.QtdConAte14 = &v
			}
		case "qtdConAte60":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.QtdConAte60 = &v
			}
		case "qtdConAte90":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.QtdConAte90 = &v
			}
		case "qtdConMaior90":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.QtdConMaior90 = &v
			}
		case "przMedCarteira":
			if v, err := strconv.Atoi(strings.TrimSpace(attr.Value)); err == nil {
				a.PrzMedCarteira = &v
			}
		}
	}
	return a
}

// xml3050Attrs é o conjunto união de todos os 4 modelos de atributos do XSD 3050.
// Campos são ponteiros para distinguir "0" vs "ausente".
type xml3050Attrs struct {
	TxMedJuros           *float64 `xml:"txMedJuros,attr,omitempty"`
	TxMedJurosAjustada   *float64 `xml:"txMedJurosAjustada,attr,omitempty"`
	TxMedEncFiscais      *float64 `xml:"txMedEncFiscais,attr,omitempty"`
	TxMedEncOperacionais *float64 `xml:"txMedEncOperacionais,attr,omitempty"`
	TxMinima             *float64 `xml:"txMinima,attr,omitempty"`
	TxMaxima             *float64 `xml:"txMaxima,attr,omitempty"`
	VlrConcessoes        *float64 `xml:"vlrConcessoes,attr,omitempty"`
	PrzDecMedConcessoes  *int     `xml:"przDecMedConcessoes,attr,omitempty"`
	QtdNovContratos      *int     `xml:"qtdNovContratos,attr,omitempty"`
	SldCarAtiva          *float64 `xml:"sldCarAtiva,attr,omitempty"`
	SldCedido            *float64 `xml:"sldCedido,attr,omitempty"`
	SldAdquirido         *float64 `xml:"sldAdquirido,attr,omitempty"`

	// Mensais
	SldBaiPrejuizo *float64 `xml:"sldBaiPrejuizo,attr,omitempty"`
	SldCarAte14    *float64 `xml:"sldCarAte14,attr,omitempty"`
	SldCarAte60    *float64 `xml:"sldCarAte60,attr,omitempty"`
	SldCarAte90    *float64 `xml:"sldCarAte90,attr,omitempty"`
	SldCarMaior90  *float64 `xml:"sldCarMaior90,attr,omitempty"`
	QtdConAte14    *int     `xml:"qtdConAte14,attr,omitempty"`
	QtdConAte60    *int     `xml:"qtdConAte60,attr,omitempty"`
	QtdConAte90    *int     `xml:"qtdConAte90,attr,omitempty"`
	QtdConMaior90  *int     `xml:"qtdConMaior90,attr,omitempty"`
	PrzMedCarteira *int     `xml:"przMedCarteira,attr,omitempty"`
}

func (a xml3050Attrs) toModalidade(codigo, encargo, tipoCli string) Modalidade {
	return Modalidade{
		Codigo:  codigo,
		Encargo: encargo,
		TipoCli: tipoCli,

		TxMedJuros:           a.TxMedJuros,
		TxMedJurosAjustada:   a.TxMedJurosAjustada,
		TxMedEncFiscais:      a.TxMedEncFiscais,
		TxMedEncOperacionais: a.TxMedEncOperacionais,
		TxMinima:             a.TxMinima,
		TxMaxima:             a.TxMaxima,
		VlrConcessoes:        a.VlrConcessoes,
		PrzDecMedConcessoes:  a.PrzDecMedConcessoes,
		QtdNovContratos:      a.QtdNovContratos,
		SldCarAtiva:          a.SldCarAtiva,
		SldCedido:            a.SldCedido,
		SldAdquirido:         a.SldAdquirido,

		SldBaiPrejuizo: a.SldBaiPrejuizo,
		SldCarAte14:    a.SldCarAte14,
		SldCarAte60:    a.SldCarAte60,
		SldCarAte90:    a.SldCarAte90,
		SldCarMaior90:  a.SldCarMaior90,
		QtdConAte14:    a.QtdConAte14,
		QtdConAte60:    a.QtdConAte60,
		QtdConAte90:    a.QtdConAte90,
		QtdConMaior90:  a.QtdConMaior90,
		PrzMedCarteira: a.PrzMedCarteira,
	}
}

// PartialParseError indica parse parcial bem-sucedido (D-26).
type PartialParseError struct {
	Reason string
}

func (e *PartialParseError) Error() string {
	return "3050 partial parse: " + e.Reason
}

// ============================================================================
// Interface Rule3050 (D-24)
// ============================================================================

// Rule3050 é a interface paralela a Rule, para regras do CADOC 3050.
//
// Decisão D-24: mantém compat com Rule (3040). Registry indexa ambos
// em maps separados.
type Rule3050 interface {
	Code() string
	Sheet() string
	Severity() string // E (Erro bloqueante), A (Aviso), I (Informativo)
	Apply3050(ctx context.Context, doc *Doc3050) error
}

// ============================================================================
// 14 Regras Agregadas A01-A14
// ============================================================================

// Helper: parseFloatPtr converte string em *float64. Nil se vazio ou inválido.
func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return nil
	}
	return &v
}

// A01 — Saldo da carteira ativa = soma dos saldos por faixa de atraso (regra 3018).
//
// Severidade: E (Erro bloqueante).
// Equivalente 3040: A01-A04 (somas Agregadas).
type A01SldCarSomaFaixas struct{}

func (A01SldCarSomaFaixas) Code() string     { return "3050-A01" }
func (A01SldCarSomaFaixas) Sheet() string    { return "Agregadas" }
func (A01SldCarSomaFaixas) Severity() string { return "E" }
func (A01SldCarSomaFaixas) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldCarAtiva == nil {
			continue
		}
		if m.SldCarAte14 == nil || m.SldCarAte60 == nil || m.SldCarAte90 == nil || m.SldCarMaior90 == nil {
			continue // incomplete — outras regras vão flagar
		}
		soma := *m.SldCarAte14 + *m.SldCarAte60 + *m.SldCarAte90 + *m.SldCarMaior90
		if diff := *m.SldCarAtiva - soma; abs(diff) > 0.01 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva=%.2f ≠ soma(sldCarAte14+Ate60+Ate90+Maior90)=%.2f (diff=%.2f)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCarAtiva, soma, diff)
		}
	}
	return nil
}

// A02 — Saldo cedido - saldo adquirido ≤ saldo carteira ativa (consistência interna, regra 3019 simplificada).
//
// Severidade: A (Aviso — pode ser legítimo em momentos específicos).
// Regra 3019 completa requer histórico de data-base anterior (carry-over Fase 2).
type A02SldCedidoMenosAdquirido struct{}

func (A02SldCedidoMenosAdquirido) Code() string     { return "3050-A02" }
func (A02SldCedidoMenosAdquirido) Sheet() string    { return "Agregadas" }
func (A02SldCedidoMenosAdquirido) Severity() string { return "A" }
func (A02SldCedidoMenosAdquirido) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldCedido == nil || m.SldAdquirido == nil {
			continue
		}
		if m.SldCarAtiva == nil {
			continue
		}
		if diff := *m.SldCedido - *m.SldAdquirido; diff > *m.SldCarAtiva+0.01 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCedido(%.2f) - sldAdquirido(%.2f) = %.2f > sldCarAtiva(%.2f)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCedido, *m.SldAdquirido, diff, *m.SldCarAtiva)
		}
	}
	return nil
}

// A03 — Saldo baixado para prejuízo ≤ saldo carteira ativa (regra 3020).
//
// Severidade: A (Aviso — depende de histórico, simplificação Fase 1).
type A03SldBaiPrejuizoLeSldCar struct{}

func (A03SldBaiPrejuizoLeSldCar) Code() string     { return "3050-A03" }
func (A03SldBaiPrejuizoLeSldCar) Sheet() string    { return "Agregadas" }
func (A03SldBaiPrejuizoLeSldCar) Severity() string { return "A" }
func (A03SldBaiPrejuizoLeSldCar) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldBaiPrejuizo == nil || m.SldCarAtiva == nil {
			continue
		}
		if *m.SldBaiPrejuizo > *m.SldCarAtiva+0.01 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldBaiPrejuizo(%.2f) > sldCarAtiva(%.2f)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldBaiPrejuizo, *m.SldCarAtiva)
		}
	}
	return nil
}

// A04 — Saldo cedido - saldo adquirido + concessões ≠ inventário impossível (regra 3021 simplificada).
//
// Severidade: A (Aviso).
// Verifica que sldCarAtiva + sldCedido ≥ sldAdquirido + vlrConcessoes
// (consistência: carteira + cedido não pode ser menor que adquirido + novo).
type A04SldCarMaisCedidoVsAdquirido struct{}

func (A04SldCarMaisCedidoVsAdquirido) Code() string     { return "3050-A04" }
func (A04SldCarMaisCedidoVsAdquirido) Sheet() string    { return "Agregadas" }
func (A04SldCarMaisCedidoVsAdquirido) Severity() string { return "A" }
func (A04SldCarMaisCedidoVsAdquirido) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.SldCarAtiva == nil || m.SldCedido == nil || m.SldAdquirido == nil || m.VlrConcessoes == nil {
			continue
		}
		esq := *m.SldCarAtiva + *m.SldCedido
		dir := *m.SldAdquirido + *m.VlrConcessoes
		if esq < dir-0.01 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva(%.2f) + sldCedido(%.2f) = %.2f < sldAdquirido(%.2f) + vlrConcessoes(%.2f) = %.2f",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCarAtiva, *m.SldCedido, esq, *m.SldAdquirido, *m.VlrConcessoes, dir)
		}
	}
	return nil
}

// A05 — CNPJ do cabeçalho é raiz válida (8 dígitos BACEN) (regra 2005 + 3021 formato).
//
// Severidade: E (Erro bloqueante).
// Equivalente 3040: H02 (CNPJ raiz).
type A05CNPJRaiz struct{}

func (A05CNPJRaiz) Code() string     { return "3050-A05" }
func (A05CNPJRaiz) Sheet() string    { return "Agregadas" }
func (A05CNPJRaiz) Severity() string { return "E" }
func (A05CNPJRaiz) Apply3050(_ context.Context, doc *Doc3050) error {
	cnpj := doc.Root.CNPJ
	if cnpj == "" {
		return fmt.Errorf("cnpjInstituicao vazio (obrigatório)")
	}
	if len(cnpj) != 8 {
		return fmt.Errorf("cnpjInstituicao=%q deve ter 8 dígitos (raiz BACEN), tem %d", cnpj, len(cnpj))
	}
	for _, c := range cnpj {
		if c < '0' || c > '9' {
			return fmt.Errorf("cnpjInstituicao=%q contém caractere não-numérico %q", cnpj, c)
		}
	}
	return nil
}

// A06 — Data-base formato YYYY-MM-DD (regra formato + 3001/3031-3033).
//
// Severidade: E.
type A06DataBaseFormato struct{}

func (A06DataBaseFormato) Code() string     { return "3050-A06" }
func (A06DataBaseFormato) Sheet() string    { return "Agregadas" }
func (A06DataBaseFormato) Severity() string { return "E" }
func (A06DataBaseFormato) Apply3050(_ context.Context, doc *Doc3050) error {
	db := doc.Root.DataBase
	if db == "" {
		return fmt.Errorf("dataBase vazio (obrigatório)")
	}
	// Validar formato YYYY-MM-DD (10 chars, posições 4 e 7 são '-')
	if len(db) != 10 || db[4] != '-' || db[7] != '-' {
		return fmt.Errorf("dataBase=%q deve estar no formato YYYY-MM-DD", db)
	}
	for i, c := range db {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return fmt.Errorf("dataBase=%q contém caractere não-numérico na posição %d", db, i)
		}
	}
	return nil
}

// A07 — Indicador de remessa ∈ {I, A, S} (regra 3052 + header).
//
// Severidade: E.
type A07IndRemessaValido struct{}

func (A07IndRemessaValido) Code() string     { return "3050-A07" }
func (A07IndRemessaValido) Sheet() string    { return "Agregadas" }
func (A07IndRemessaValido) Severity() string { return "E" }
func (A07IndRemessaValido) Apply3050(_ context.Context, doc *Doc3050) error {
	ir := doc.Root.IndRemessa
	switch ir {
	case "I", "A", "S":
		return nil
	case "":
		return fmt.Errorf("indRemessa vazio (obrigatório; esperado I=inclusão, A=alteração, S=substituição)")
	default:
		return fmt.Errorf("indRemessa=%q inválido (esperado I, A ou S)", ir)
	}
}

// A08 — Nome do contato não-vazio (regra 2005 header obrigatório).
//
// Severidade: E.
type A08NmContatoObrigatorio struct{}

func (A08NmContatoObrigatorio) Code() string     { return "3050-A08" }
func (A08NmContatoObrigatorio) Sheet() string    { return "Agregadas" }
func (A08NmContatoObrigatorio) Severity() string { return "E" }
func (A08NmContatoObrigatorio) Apply3050(_ context.Context, doc *Doc3050) error {
	if strings.TrimSpace(doc.Root.NmContato) == "" {
		return fmt.Errorf("nmContato vazio (obrigatório no cabeçalho)")
	}
	if strings.TrimSpace(doc.Root.TelContato) == "" {
		return fmt.Errorf("telContato vazio (obrigatório no cabeçalho)")
	}
	return nil
}

// A09 — Taxa média de juros ≤ 100% (regra 3026/3042 limites).
//
// Severidade: E (valores absurdos são bloqueantes — BACEN contacta).
type A09TxMedJurosLimite struct{}

func (A09TxMedJurosLimite) Code() string     { return "3050-A09" }
func (A09TxMedJurosLimite) Sheet() string    { return "Agregadas" }
func (A09TxMedJurosLimite) Severity() string { return "E" }
func (A09TxMedJurosLimite) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.TxMedJuros == nil {
			continue
		}
		if *m.TxMedJuros < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedJuros=%.4f < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedJuros)
		}
		if *m.TxMedJuros > 100 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedJuros=%.4f > 100%% (contatar BACEN 61 3414-3115)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedJuros)
		}
	}
	return nil
}

// A10 — Taxa média de encargos fiscais ≤ 100% (regra 3027/3043).
//
// Severidade: E.
type A10TxMedEncFiscaisLimite struct{}

func (A10TxMedEncFiscaisLimite) Code() string     { return "3050-A10" }
func (A10TxMedEncFiscaisLimite) Sheet() string    { return "Agregadas" }
func (A10TxMedEncFiscaisLimite) Severity() string { return "E" }
func (A10TxMedEncFiscaisLimite) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.TxMedEncFiscais == nil {
			continue
		}
		if *m.TxMedEncFiscais < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncFiscais=%.4f < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedEncFiscais)
		}
		if *m.TxMedEncFiscais > 100 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncFiscais=%.4f > 100%% (contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedEncFiscais)
		}
	}
	return nil
}

// A11 — Taxa média de encargos operacionais ≤ 100% (regra 3028/3044).
//
// Severidade: E.
type A11TxMedEncOperacionaisLimite struct{}

func (A11TxMedEncOperacionaisLimite) Code() string     { return "3050-A11" }
func (A11TxMedEncOperacionaisLimite) Sheet() string    { return "Agregadas" }
func (A11TxMedEncOperacionaisLimite) Severity() string { return "E" }
func (A11TxMedEncOperacionaisLimite) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.TxMedEncOperacionais == nil {
			continue
		}
		if *m.TxMedEncOperacionais < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncOperacionais=%.4f < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedEncOperacionais)
		}
		if *m.TxMedEncOperacionais > 100 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncOperacionais=%.4f > 100%% (contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedEncOperacionais)
		}
	}
	return nil
}

// A12 — Taxa mínima ≤ taxa máxima (consistência, regra 3051 base).
//
// Severidade: E.
type A12TxMinimaLeMaxima struct{}

func (A12TxMinimaLeMaxima) Code() string     { return "3050-A12" }
func (A12TxMinimaLeMaxima) Sheet() string    { return "Agregadas" }
func (A12TxMinimaLeMaxima) Severity() string { return "E" }
func (A12TxMinimaLeMaxima) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.TxMinima == nil || m.TxMaxima == nil {
			continue
		}
		if *m.TxMinima > *m.TxMaxima+0.0001 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMinima(%.4f) > txMaxima(%.4f)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMinima, *m.TxMaxima)
		}
	}
	return nil
}

// A13 — Prazo a decorrer médio das concessões ≥ 0 (regra 3036/3037 base).
//
// Severidade: E (prazo negativo é impossível).
type A13PrzDecMedConcessoesNaoNeg struct{}

func (A13PrzDecMedConcessoesNaoNeg) Code() string     { return "3050-A13" }
func (A13PrzDecMedConcessoesNaoNeg) Sheet() string    { return "Agregadas" }
func (A13PrzDecMedConcessoesNaoNeg) Severity() string { return "E" }
func (A13PrzDecMedConcessoesNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przDecMedConcessoes=%d < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// A14 — Prazo médio da carteira ≥ 0 (regra 3038/3039 base).
//
// Severidade: E.
type A14PrzMedCarteiraNaoNeg struct{}

func (A14PrzMedCarteiraNaoNeg) Code() string     { return "3050-A14" }
func (A14PrzMedCarteiraNaoNeg) Sheet() string    { return "Agregadas" }
func (A14PrzMedCarteiraNaoNeg) Severity() string { return "E" }
func (A14PrzMedCarteiraNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.PrzMedCarteira == nil {
			continue
		}
		if *m.PrzMedCarteira < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przMedCarteira=%d < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzMedCarteira)
		}
	}
	return nil
}

// ============================================================================
// 14 Stubs S01-S14 (severity I, D-27)
// ============================================================================

// S01 — Matriz modalidade × encargo × sub-modalidade (regra 2001 × 134).
//
// Stub: XSD valida combinações estruturais (2001.Tratado no XSD). Regra Go
// validaria regras de negócio específicas (ex: desDuplicatas apenas com
// encargo pre). Carry-over Fase 4 quando matriz estiver catalogada.
type S01MatrizEncargoModalidade struct{}

func (S01MatrizEncargoModalidade) Code() string     { return "3050-S01" }
func (S01MatrizEncargoModalidade) Sheet() string    { return "Stubs" }
func (S01MatrizEncargoModalidade) Severity() string { return "I" }
func (S01MatrizEncargoModalidade) Apply3050(_ context.Context, _ *Doc3050) error {
	// stub — carry-over Fase 4
	return nil
}

// S02 — Documento não esperado para a IF (regra 2002).
//
// Stub: depende de cadastro de quais IFs devem enviar 3050. Carry-over
// Sprint 33+ Fase 2 (requer lookup cadastro IFs).
type S02DocNaoEsperado struct{}

func (S02DocNaoEsperado) Code() string     { return "3050-S02" }
func (S02DocNaoEsperado) Sheet() string    { return "Stubs" }
func (S02DocNaoEsperado) Severity() string { return "I" }
func (S02DocNaoEsperado) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S03 — Arquivo dispensado de envio (regra 2003).
//
// Stub: depende de calendário BACEN (datas-base dispensadas).
type S03ArquivoDispensado struct{}

func (S03ArquivoDispensado) Code() string     { return "3050-S03" }
func (S03ArquivoDispensado) Sheet() string    { return "Stubs" }
func (S03ArquivoDispensado) Severity() string { return "I" }
func (S03ArquivoDispensado) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S04 — Erro no header (regra 2005) — parcialmente coberto por A05/A06/A07/A08.
//
// Stub: A05-A08 cobrem formato CNPJ/dataBase/indRemessa/contato. Carry-over
// para Fase 2 com validações adicionais (espaços, encoding, length max).
type S04HeaderDetalhe struct{}

func (S04HeaderDetalhe) Code() string     { return "3050-S04" }
func (S04HeaderDetalhe) Sheet() string    { return "Stubs" }
func (S04HeaderDetalhe) Severity() string { return "I" }
func (S04HeaderDetalhe) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S05 — Arquivo já processado (regra 2006) / em validação (regra 2009).
//
// Stub: depende de estado do BACEN (auditoria externa).
type S05ArquivoJaProcessado struct{}

func (S05ArquivoJaProcessado) Code() string     { return "3050-S05" }
func (S05ArquivoJaProcessado) Sheet() string    { return "Stubs" }
func (S05ArquivoJaProcessado) Severity() string { return "I" }
func (S05ArquivoJaProcessado) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S06 — Substituição sem recebimento original (regra 2007).
//
// Stub: depende de histórico de envios.
type S06SubstituicaoSemOriginal struct{}

func (S06SubstituicaoSemOriginal) Code() string     { return "3050-S06" }
func (S06SubstituicaoSemOriginal) Sheet() string    { return "Stubs" }
func (S06SubstituicaoSemOriginal) Severity() string { return "I" }
func (S06SubstituicaoSemOriginal) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S07 — Compactação diferente de GZ/ZIP (regra 2008).
//
// Stub: depende de validação no nível do protocolo de envio (não do XML).
type S07Compactacao struct{}

func (S07Compactacao) Code() string     { return "3050-S07" }
func (S07Compactacao) Sheet() string    { return "Stubs" }
func (S07Compactacao) Severity() string { return "I" }
func (S07Compactacao) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S08 — Data-base maior que dia corrente (regra 2010).
//
// Stub: requer comparação com data atual + lógica de tolerância (D-1).
type S08DataBaseFutura struct{}

func (S08DataBaseFutura) Code() string     { return "3050-S08" }
func (S08DataBaseFutura) Sheet() string    { return "Stubs" }
func (S08DataBaseFutura) Severity() string { return "I" }
func (S08DataBaseFutura) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S10 — Documento anterior não enviado (regra 3002).
//
// Stub: depende de histórico de envios.
type S10DocAnterior struct{}

func (S10DocAnterior) Code() string     { return "3050-S10" }
func (S10DocAnterior) Sheet() string    { return "Stubs" }
func (S10DocAnterior) Severity() string { return "I" }
func (S10DocAnterior) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S11 — Valor concessões vs taxa média de juros (regras 3003/3004/3007/3008/3009).
//
// Stub: 5 regras sobre coerência "se um é zero, o outro também deve ser".
// Carry-over Fase 2 com tabela de mapeamento modelo 1-4 × campos.
type S11VlrConcessoesVsTaxas struct{}

func (S11VlrConcessoesVsTaxas) Code() string     { return "3050-S11" }
func (S11VlrConcessoesVsTaxas) Sheet() string    { return "Stubs" }
func (S11VlrConcessoesVsTaxas) Severity() string { return "I" }
func (S11VlrConcessoesVsTaxas) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S12 — Prazo médio carteira se saldo != 0 (regras 3025/3034).
//
// Stub: A14 valida não-negatividade. Carry-over para Fase 2 com regra
// "se SldCarAtiva != 0, PrzMedCarteira obrigatório".
type S12PrzMedSeSld struct{}

func (S12PrzMedSeSld) Code() string     { return "3050-S12" }
func (S12PrzMedSeSld) Sheet() string    { return "Stubs" }
func (S12PrzMedSeSld) Severity() string { return "I" }
func (S12PrzMedSeSld) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// S14 — Cruzadas 3051/3054/3055/3056-3059.
//
// Stub: validações cruzadas complexas (ex: 3056 = saldo crédito pessoal
// não consignado = soma saldos sub-modalidades). Carry-over Fase 2.
type S14Cruzadas struct{}

func (S14Cruzadas) Code() string     { return "3050-S14" }
func (S14Cruzadas) Sheet() string    { return "Stubs" }
func (S14Cruzadas) Severity() string { return "I" }
func (S14Cruzadas) Apply3050(_ context.Context, _ *Doc3050) error {
	return nil
}

// ============================================================================
// 14 Sistemáticas S15-S28 (Fase 2)
// ============================================================================

// S15 — Data-base válida (regra 2010 — não-futura, não-anterior a 2009-09).
//
// Severidade: E.
type S15DataBaseValida struct{}

func (S15DataBaseValida) Code() string     { return "3050-S15" }
func (S15DataBaseValida) Sheet() string    { return "Sistemáticas" }
func (S15DataBaseValida) Severity() string { return "E" }
func (S15DataBaseValida) Apply3050(_ context.Context, doc *Doc3050) error {
	db := doc.Root.DataBase
	if db == "" || len(db) != 10 {
		return nil // formato já validado em A06
	}
	// Validar ano >= 2009 (TXB_V11 inicio) e ano <= ano corrente + 1 (tolerância)
	ano, err := strconv.Atoi(db[:4])
	if err != nil {
		return nil // formato já validado em A06
	}
	if ano < 2009 {
		return fmt.Errorf("dataBase=%q anterior a 2009-09 (início TXB_V11)", db)
	}
	if ano > 2030 {
		return fmt.Errorf("dataBase=%q muito futura (ano %d) — contatar BACEN", db, ano)
	}
	return nil
}

// S16 — Nome do contato length 1-100 caracteres (regra 2005 detalhe).
//
// Severidade: A.
type S16NmContatoLength struct{}

func (S16NmContatoLength) Code() string     { return "3050-S16" }
func (S16NmContatoLength) Sheet() string    { return "Sistemáticas" }
func (S16NmContatoLength) Severity() string { return "A" }
func (S16NmContatoLength) Apply3050(_ context.Context, doc *Doc3050) error {
	nm := strings.TrimSpace(doc.Root.NmContato)
	if nm == "" {
		return nil // A08 já cobre vazio
	}
	if len(nm) > 100 {
		return fmt.Errorf("nmContato length=%d > 100 (limite BACEN)", len(nm))
	}
	return nil
}

// S17 — Telefone do contato formato 10-11 dígitos (regra 2005 detalhe).
//
// Severidade: A.
type S17TelContatoFormato struct{}

func (S17TelContatoFormato) Code() string     { return "3050-S17" }
func (S17TelContatoFormato) Sheet() string    { return "Sistemáticas" }
func (S17TelContatoFormato) Severity() string { return "A" }
func (S17TelContatoFormato) Apply3050(_ context.Context, doc *Doc3050) error {
	tel := strings.TrimSpace(doc.Root.TelContato)
	if tel == "" {
		return nil // A08 já cobre vazio
	}
	digits := 0
	for _, c := range tel {
		if c >= '0' && c <= '9' {
			digits++
		}
	}
	if digits < 10 || digits > 11 {
		return fmt.Errorf("telContato=%q tem %d dígitos (esperado 10-11: DDD + número)", tel, digits)
	}
	return nil
}

// S18 — Valor concessões zero → txMedJuros deve ser zero (regra 3003).
//
// Severidade: E.
type S18VlrConcessoesZeroTxJurosZero struct{}

func (S18VlrConcessoesZeroTxJurosZero) Code() string     { return "3050-S18" }
func (S18VlrConcessoesZeroTxJurosZero) Sheet() string    { return "Sistemáticas" }
func (S18VlrConcessoesZeroTxJurosZero) Severity() string { return "E" }
func (S18VlrConcessoesZeroTxJurosZero) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil || m.TxMedJuros == nil {
			continue
		}
		if *m.VlrConcessoes == 0 && *m.TxMedJuros != 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): vlrConcessoes=0 mas txMedJuros=%.4f ≠ 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedJuros)
		}
	}
	return nil
}

// S19 — txMedJuros zero → vlrConcessoes deve ser > 0 (regra 3004).
//
// Severidade: E.
type S19TxJurosZeroVlrConcessoesPos struct{}

func (S19TxJurosZeroVlrConcessoesPos) Code() string     { return "3050-S19" }
func (S19TxJurosZeroVlrConcessoesPos) Sheet() string    { return "Sistemáticas" }
func (S19TxJurosZeroVlrConcessoesPos) Severity() string { return "E" }
func (S19TxJurosZeroVlrConcessoesPos) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil || m.TxMedJuros == nil {
			continue
		}
		if *m.TxMedJuros == 0 && *m.VlrConcessoes <= 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedJuros=0 mas vlrConcessoes=%.2f ≤ 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// S20 — txMedEncOperacionais zero → vlrConcessoes deve ser > 0 (regra 3007).
//
// Severidade: E.
type S20TxEncOperZeroVlrConcessoesPos struct{}

func (S20TxEncOperZeroVlrConcessoesPos) Code() string     { return "3050-S20" }
func (S20TxEncOperZeroVlrConcessoesPos) Sheet() string    { return "Sistemáticas" }
func (S20TxEncOperZeroVlrConcessoesPos) Severity() string { return "E" }
func (S20TxEncOperZeroVlrConcessoesPos) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil || m.TxMedEncOperacionais == nil {
			continue
		}
		if *m.TxMedEncOperacionais == 0 && *m.VlrConcessoes <= 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncOperacionais=0 mas vlrConcessoes=%.2f ≤ 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// S21 — przDecMedConcessoes zero → vlrConcessoes deve ser > 0 (regra 3008).
//
// Severidade: E.
type S21PrzDecZeroVlrConcessoesPos struct{}

func (S21PrzDecZeroVlrConcessoesPos) Code() string     { return "3050-S21" }
func (S21PrzDecZeroVlrConcessoesPos) Sheet() string    { return "Sistemáticas" }
func (S21PrzDecZeroVlrConcessoesPos) Severity() string { return "E" }
func (S21PrzDecZeroVlrConcessoesPos) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil || m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes == 0 && *m.VlrConcessoes <= 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przDecMedConcessoes=0 mas vlrConcessoes=%.2f ≤ 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// S22 — przDecMedConcessoes > 0 → vlrConcessoes deve ser > 0 (regra 3009).
//
// Severidade: E.
type S22PrzDecPosVlrConcessoesPos struct{}

func (S22PrzDecPosVlrConcessoesPos) Code() string     { return "3050-S22" }
func (S22PrzDecPosVlrConcessoesPos) Sheet() string    { return "Sistemáticas" }
func (S22PrzDecPosVlrConcessoesPos) Severity() string { return "E" }
func (S22PrzDecPosVlrConcessoesPos) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil || m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes > 0 && *m.VlrConcessoes <= 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przDecMedConcessoes=%d > 0 mas vlrConcessoes=%.2f ≤ 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes, *m.VlrConcessoes)
		}
	}
	return nil
}

// S23 — PrzMedCarteira condicional: se sldCarAtiva != 0, przMedCarteira obrigatório (regra 3025).
//
// Severidade: A (warning — pode ser edge case legítimo).
type S23PrzMedCondicional struct{}

func (S23PrzMedCondicional) Code() string     { return "3050-S23" }
func (S23PrzMedCondicional) Sheet() string    { return "Sistemáticas" }
func (S23PrzMedCondicional) Severity() string { return "A" }
func (S23PrzMedCondicional) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldCarAtiva == nil {
			continue
		}
		if m.PrzMedCarteira == nil && *m.SldCarAtiva != 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva=%.2f ≠ 0 mas przMedCarteira ausente",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCarAtiva)
		}
	}
	return nil
}

// S25 — cnpjInstituicao != zero prefix (regra formato BACEN).
//
// Severidade: A (cnpj "00000000" é placeholder, deve ser raiz real).
type S25CNPJNaoZero struct{}

func (S25CNPJNaoZero) Code() string     { return "3050-S25" }
func (S25CNPJNaoZero) Sheet() string    { return "Sistemáticas" }
func (S25CNPJNaoZero) Severity() string { return "A" }
func (S25CNPJNaoZero) Apply3050(_ context.Context, doc *Doc3050) error {
	cnpj := doc.Root.CNPJ
	if cnpj == "00000000" {
		return fmt.Errorf("cnpjInstituicao=%q é placeholder (deve ser raiz BACEN real)", cnpj)
	}
	return nil
}

// S26 — Data de referência duplicada não permitida no batch (regra 3054).
//
// Severidade: E.
// Verifica que não há 2+ Modalidades com mesma combinação Codigo+Encargo+TipoCli
// em cada período (Diario/Mensal). XSD permite múltiplas referências mas em
// prática IF envia 1 referência por arquivo.
type S26CodigoEncargoTipoCliUnico struct{}

func (S26CodigoEncargoTipoCliUnico) Code() string     { return "3050-S26" }
func (S26CodigoEncargoTipoCliUnico) Sheet() string    { return "Sistemáticas" }
func (S26CodigoEncargoTipoCliUnico) Severity() string { return "E" }
func (S26CodigoEncargoTipoCliUnico) Apply3050(_ context.Context, doc *Doc3050) error {
	seen := make(map[string]int)
	for i, m := range doc.Diario {
		key := m.Codigo + "|" + m.Encargo + "|" + m.TipoCli
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("modalidade %s [%d] (%s/%s) duplicada (primeira em [%d])",
				m.Codigo, i, m.Encargo, m.TipoCli, prev)
		}
		seen[key] = i
	}
	seenM := make(map[string]int)
	for i, m := range doc.Mensal {
		key := m.Codigo + "|" + m.Encargo + "|" + m.TipoCli
		if prev, ok := seenM[key]; ok {
			return fmt.Errorf("modalidade %s [%d] (%s/%s) duplicada em Mensal (primeira em [%d])",
				m.Codigo, i, m.Encargo, m.TipoCli, prev)
		}
		seenM[key] = i
	}
	return nil
}

// S27 — Periodicidade modalidades mensais: se Mensal presente, SldBaiPrejuizo deve ser >= 0 (regra formato).
//
// Severidade: E (negativo impossível).
type S27SldBaiPrejuizoNaoNeg struct{}

func (S27SldBaiPrejuizoNaoNeg) Code() string     { return "3050-S27" }
func (S27SldBaiPrejuizoNaoNeg) Sheet() string    { return "Sistemáticas" }
func (S27SldBaiPrejuizoNaoNeg) Severity() string { return "E" }
func (S27SldBaiPrejuizoNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldBaiPrejuizo == nil {
			continue
		}
		if *m.SldBaiPrejuizo < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldBaiPrejuizo=%.2f < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldBaiPrejuizo)
		}
	}
	return nil
}

// S28 — qtdNovContratos >= 0 (regra formato).
//
// Severidade: E.
type S28QtdNovContratosNaoNeg struct{}

func (S28QtdNovContratosNaoNeg) Code() string     { return "3050-S28" }
func (S28QtdNovContratosNaoNeg) Sheet() string    { return "Sistemáticas" }
func (S28QtdNovContratosNaoNeg) Severity() string { return "E" }
func (S28QtdNovContratosNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.QtdNovContratos == nil {
			continue
		}
		if *m.QtdNovContratos < 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): qtdNovContratos=%d < 0",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.QtdNovContratos)
		}
	}
	return nil
}

// ============================================================================
// 14 Individuais/Cruzadas I01-I14 (Fase 2)
// ============================================================================

// I01 — CapGirPrzAte365: przDecMedConcessoes ≤ 365 (regra 3036).
//
// Severidade: E.
type I01CapGirAte365 struct{}

func (I01CapGirAte365) Code() string     { return "3050-I01" }
func (I01CapGirAte365) Sheet() string    { return "Individuais" }
func (I01CapGirAte365) Severity() string { return "E" }
func (I01CapGirAte365) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "capGirPrzAte365" {
			continue
		}
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes > 365 {
			return fmt.Errorf("modalidade capGirPrzAte365 [%d] (%s/%s): przDecMedConcessoes=%d > 365 (regra 3036)",
				i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I02 — CapGirPrzSup365: przDecMedConcessoes > 365 (regra 3037).
//
// Severidade: E.
type I02CapGirSup365 struct{}

func (I02CapGirSup365) Code() string     { return "3050-I02" }
func (I02CapGirSup365) Sheet() string    { return "Individuais" }
func (I02CapGirSup365) Severity() string { return "E" }
func (I02CapGirSup365) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "capGirPrzSup365" {
			continue
		}
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes <= 365 {
			return fmt.Errorf("modalidade capGirPrzSup365 [%d] (%s/%s): przDecMedConcessoes=%d ≤ 365 (regra 3037)",
				i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I03 — Crédito Pessoal Não Consignado: sldCarAtiva = soma sub-modalidades (regra 3056).
//
// Severidade: E.
// Soma: aquVeiculos + aquOutBens + arrMerVeiculos + arrMerOutBens (4 sub-modalidades).
// crdPesNaoConsignado é a MODALIDADE AGREGADA, não se inclui na soma.
type I03CredPesNaoConsignadoSldCar struct{}

func (I03CredPesNaoConsignadoSldCar) Code() string     { return "3050-I03" }
func (I03CredPesNaoConsignadoSldCar) Sheet() string    { return "Individuais" }
func (I03CredPesNaoConsignadoSldCar) Severity() string { return "E" }
func (I03CredPesNaoConsignadoSldCar) Apply3050(_ context.Context, doc *Doc3050) error {
	subMods := []string{"aquVeiculos", "aquOutBens", "arrMerVeiculos", "arrMerOutBens"}
	for i, m := range doc.Mensal {
		if m.Codigo != "crdPesNaoConsignado" {
			continue
		}
		if m.SldCarAtiva == nil {
			continue
		}
		var soma float64
		for _, sub := range doc.Mensal {
			if sub.Encargo != m.Encargo || sub.TipoCli != m.TipoCli {
				continue
			}
			for _, s := range subMods {
				if sub.Codigo == s && sub.SldCarAtiva != nil {
					soma += *sub.SldCarAtiva
				}
			}
		}
		if diff := *m.SldCarAtiva - soma; abs(diff) > 0.01 {
			return fmt.Errorf("modalidade crdPesNaoConsignado [%d] (%s/%s): sldCarAtiva=%.2f ≠ soma(sub-modalidades)=%.2f (diff=%.2f)",
				i, m.Encargo, m.TipoCli, *m.SldCarAtiva, soma, diff)
		}
	}
	return nil
}

// I04 — Crédito Pessoal Não Consignado: vlrConcessoes = soma sub-modalidades (regra 3057).
//
// Severidade: E.
type I04CredPesNaoConsignadoVlrConcessoes struct{}

func (I04CredPesNaoConsignadoVlrConcessoes) Code() string     { return "3050-I04" }
func (I04CredPesNaoConsignadoVlrConcessoes) Sheet() string    { return "Individuais" }
func (I04CredPesNaoConsignadoVlrConcessoes) Severity() string { return "E" }
func (I04CredPesNaoConsignadoVlrConcessoes) Apply3050(_ context.Context, doc *Doc3050) error {
	subMods := []string{"aquVeiculos", "aquOutBens", "arrMerVeiculos", "arrMerOutBens"}
	for i, m := range doc.Diario {
		if m.Codigo != "crdPesNaoConsignado" {
			continue
		}
		if m.VlrConcessoes == nil {
			continue
		}
		var soma float64
		for _, sub := range doc.Diario {
			if sub.Encargo != m.Encargo || sub.TipoCli != m.TipoCli {
				continue
			}
			for _, s := range subMods {
				if sub.Codigo == s && sub.VlrConcessoes != nil {
					soma += *sub.VlrConcessoes
				}
			}
		}
		if diff := *m.VlrConcessoes - soma; abs(diff) > 0.01 {
			return fmt.Errorf("modalidade crdPesNaoConsignado [%d] (%s/%s): vlrConcessoes=%.2f ≠ soma(sub-modalidades)=%.2f (diff=%.2f)",
				i, m.Encargo, m.TipoCli, *m.VlrConcessoes, soma, diff)
		}
	}
	return nil
}

// I05 — Crédito Pessoal Não Consignado: sldAdquirido = soma sub-modalidades (regra 3058).
//
// Severidade: E.
type I05CredPesNaoConsignadoSldAdquirido struct{}

func (I05CredPesNaoConsignadoSldAdquirido) Code() string     { return "3050-I05" }
func (I05CredPesNaoConsignadoSldAdquirido) Sheet() string    { return "Individuais" }
func (I05CredPesNaoConsignadoSldAdquirido) Severity() string { return "E" }
func (I05CredPesNaoConsignadoSldAdquirido) Apply3050(_ context.Context, doc *Doc3050) error {
	subMods := []string{"aquVeiculos", "aquOutBens", "arrMerVeiculos", "arrMerOutBens"}
	for i, m := range doc.Mensal {
		if m.Codigo != "crdPesNaoConsignado" {
			continue
		}
		if m.SldAdquirido == nil {
			continue
		}
		var soma float64
		for _, sub := range doc.Mensal {
			if sub.Encargo != m.Encargo || sub.TipoCli != m.TipoCli {
				continue
			}
			for _, s := range subMods {
				if sub.Codigo == s && sub.SldAdquirido != nil {
					soma += *sub.SldAdquirido
				}
			}
		}
		if diff := *m.SldAdquirido - soma; abs(diff) > 0.01 {
			return fmt.Errorf("modalidade crdPesNaoConsignado [%d] (%s/%s): sldAdquirido=%.2f ≠ soma(sub-modalidades)=%.2f (diff=%.2f)",
				i, m.Encargo, m.TipoCli, *m.SldAdquirido, soma, diff)
		}
	}
	return nil
}

// I06 — Crédito Pessoal Não Consignado: sldCedido = soma sub-modalidades (regra 3059).
//
// Severidade: E.
type I06CredPesNaoConsignadoSldCedido struct{}

func (I06CredPesNaoConsignadoSldCedido) Code() string     { return "3050-I06" }
func (I06CredPesNaoConsignadoSldCedido) Sheet() string    { return "Individuais" }
func (I06CredPesNaoConsignadoSldCedido) Severity() string { return "E" }
func (I06CredPesNaoConsignadoSldCedido) Apply3050(_ context.Context, doc *Doc3050) error {
	subMods := []string{"aquVeiculos", "aquOutBens", "arrMerVeiculos", "arrMerOutBens"}
	for i, m := range doc.Mensal {
		if m.Codigo != "crdPesNaoConsignado" {
			continue
		}
		if m.SldCedido == nil {
			continue
		}
		var soma float64
		for _, sub := range doc.Mensal {
			if sub.Encargo != m.Encargo || sub.TipoCli != m.TipoCli {
				continue
			}
			for _, s := range subMods {
				if sub.Codigo == s && sub.SldCedido != nil {
					soma += *sub.SldCedido
				}
			}
		}
		if diff := *m.SldCedido - soma; abs(diff) > 0.01 {
			return fmt.Errorf("modalidade crdPesNaoConsignado [%d] (%s/%s): sldCedido=%.2f ≠ soma(sub-modalidades)=%.2f (diff=%.2f)",
				i, m.Encargo, m.TipoCli, *m.SldCedido, soma, diff)
		}
	}
	return nil
}

// I07 — PrzMedCarteira < 30 dias (limite baixo BACEN) — contatar (regra 3038).
//
// Severidade: A (warning heurístico — não bloqueia).
type I07PrzMedCarteiraBaixo struct{}

func (I07PrzMedCarteiraBaixo) Code() string     { return "3050-I07" }
func (I07PrzMedCarteiraBaixo) Sheet() string    { return "Individuais" }
func (I07PrzMedCarteiraBaixo) Severity() string { return "A" }
func (I07PrzMedCarteiraBaixo) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.PrzMedCarteira == nil {
			continue
		}
		if *m.PrzMedCarteira < 30 && *m.PrzMedCarteira >= 0 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przMedCarteira=%d < 30 dias (muito baixo — contatar BACEN 61 3414-3115)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzMedCarteira)
		}
	}
	return nil
}

// I08 — PrzMedCarteira > 5000 dias (limite alto BACEN) — contatar (regra 3039).
//
// Severidade: A.
type I08PrzMedCarteiraAlto struct{}

func (I08PrzMedCarteiraAlto) Code() string     { return "3050-I08" }
func (I08PrzMedCarteiraAlto) Sheet() string    { return "Individuais" }
func (I08PrzMedCarteiraAlto) Severity() string { return "A" }
func (I08PrzMedCarteiraAlto) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.PrzMedCarteira == nil {
			continue
		}
		if *m.PrzMedCarteira > 5000 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przMedCarteira=%d > 5000 dias (muito alto — contatar BACEN 61 3414-3115)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzMedCarteira)
		}
	}
	return nil
}

// I09 — PrzDecMedConcessoes < 1 dia (limite baixo) — contatar (regra 3040).
//
// Severidade: A.
type I09PrzDecMedConcessoesBaixo struct{}

func (I09PrzDecMedConcessoesBaixo) Code() string     { return "3050-I09" }
func (I09PrzDecMedConcessoesBaixo) Sheet() string    { return "Individuais" }
func (I09PrzDecMedConcessoesBaixo) Severity() string { return "A" }
func (I09PrzDecMedConcessoesBaixo) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes < 1 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przDecMedConcessoes=%d < 1 dia (muito baixo — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I10 — PrzDecMedConcessoes > 5000 dias (limite alto) — contatar (regra 3041).
//
// Severidade: A.
type I10PrzDecMedConcessoesAlto struct{}

func (I10PrzDecMedConcessoesAlto) Code() string     { return "3050-I10" }
func (I10PrzDecMedConcessoesAlto) Sheet() string    { return "Individuais" }
func (I10PrzDecMedConcessoesAlto) Severity() string { return "A" }
func (I10PrzDecMedConcessoesAlto) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes > 5000 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): przDecMedConcessoes=%d > 5000 dias (muito alto — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I11 — SldCarAtiva muito baixo (< R$ 1000) — contatar (regra 3045).
//
// Severidade: A.
type I11SldCarAtivaMuitoBaixo struct{}

func (I11SldCarAtivaMuitoBaixo) Code() string     { return "3050-I11" }
func (I11SldCarAtivaMuitoBaixo) Sheet() string    { return "Individuais" }
func (I11SldCarAtivaMuitoBaixo) Severity() string { return "A" }
func (I11SldCarAtivaMuitoBaixo) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldCarAtiva == nil {
			continue
		}
		if *m.SldCarAtiva < 1000 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva=%.2f < R$ 1000 (muito baixo — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCarAtiva)
		}
	}
	return nil
}

// I12 — SldCarAtiva muito alto (> R$ 1 trilhão) — contatar (regra 3046).
//
// Severidade: A.
type I12SldCarAtivaMuitoAlto struct{}

func (I12SldCarAtivaMuitoAlto) Code() string     { return "3050-I12" }
func (I12SldCarAtivaMuitoAlto) Sheet() string    { return "Individuais" }
func (I12SldCarAtivaMuitoAlto) Severity() string { return "A" }
func (I12SldCarAtivaMuitoAlto) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.SldCarAtiva == nil {
			continue
		}
		if *m.SldCarAtiva > 1e12 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva=%.2f > R$ 1 trilhão (muito alto — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCarAtiva)
		}
	}
	return nil
}

// I13 — VlrConcessoes muito baixo (< R$ 1000) — contatar (regra 3047).
//
// Severidade: A.
type I13VlrConcessoesMuitoBaixo struct{}

func (I13VlrConcessoesMuitoBaixo) Code() string     { return "3050-I13" }
func (I13VlrConcessoesMuitoBaixo) Sheet() string    { return "Individuais" }
func (I13VlrConcessoesMuitoBaixo) Severity() string { return "A" }
func (I13VlrConcessoesMuitoBaixo) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil {
			continue
		}
		if *m.VlrConcessoes < 1000 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): vlrConcessoes=%.2f < R$ 1000 (muito baixo — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// I14 — VlrConcessoes muito alto (> R$ 1 trilhão) — contatar (regra 3048).
//
// Severidade: A.
type I14VlrConcessoesMuitoAlto struct{}

func (I14VlrConcessoesMuitoAlto) Code() string     { return "3050-I14" }
func (I14VlrConcessoesMuitoAlto) Sheet() string    { return "Individuais" }
func (I14VlrConcessoesMuitoAlto) Severity() string { return "A" }
func (I14VlrConcessoesMuitoAlto) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.VlrConcessoes == nil {
			continue
		}
		if *m.VlrConcessoes > 1e12 {
			return fmt.Errorf("modalidade %s [%d] (%s/%s): vlrConcessoes=%.2f > R$ 1 trilhão (muito alto — contatar BACEN)",
				m.Codigo, i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// ============================================================================
// 6 Regras Header H10-H15 — Fase 3
// ============================================================================

// H10 — cnpjInstituicao length = 8 dígitos (raiz CNPJ BACEN, sem DV).
//
// Severidade: A (formato BACEN).
// Origem: 2005 (header obrigatório) + formato BACEN.
type H10CNPJLength struct{}

func (H10CNPJLength) Code() string     { return "3050-H10" }
func (H10CNPJLength) Sheet() string    { return "Header" }
func (H10CNPJLength) Severity() string { return "A" }
func (H10CNPJLength) Apply3050(_ context.Context, doc *Doc3050) error {
	cnpj := doc.Root.CNPJ
	if len(cnpj) != 8 {
		return fmt.Errorf("cnpjInstituicao=%q (length=%d, esperado 8 dígitos — raiz BACEN)", cnpj, len(cnpj))
	}
	return nil
}

// H11 — cnpjInstituicao all-digits (sem letras, símbolos ou espaços).
//
// Severidade: A.
type H11CNPJAllDigits struct{}

func (H11CNPJAllDigits) Code() string     { return "3050-H11" }
func (H11CNPJAllDigits) Sheet() string    { return "Header" }
func (H11CNPJAllDigits) Severity() string { return "A" }
func (H11CNPJAllDigits) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, c := range doc.Root.CNPJ {
		if c < '0' || c > '9' {
			return fmt.Errorf("cnpjInstituicao=%q contém caractere não-numérico na posição %d (%q)", doc.Root.CNPJ, i, c)
		}
	}
	return nil
}

// H12 — dataBase formato YYYY-MM-DD rigoroso (não 2024-1-1 ou 2024/01/01).
//
// Severidade: E.
type H12DataBaseFormatoRigoroso struct{}

func (H12DataBaseFormatoRigoroso) Code() string     { return "3050-H12" }
func (H12DataBaseFormatoRigoroso) Sheet() string    { return "Header" }
func (H12DataBaseFormatoRigoroso) Severity() string { return "E" }
func (H12DataBaseFormatoRigoroso) Apply3050(_ context.Context, doc *Doc3050) error {
	db := doc.Root.DataBase
	if len(db) != 10 {
		return fmt.Errorf("dataBase=%q (length=%d, esperado 10 caracteres formato YYYY-MM-DD)", db, len(db))
	}
	for i, c := range db {
		switch i {
		case 4, 7:
			if c != '-' {
				return fmt.Errorf("dataBase=%q: esperado '-' nas posições 4 e 7, encontrado %q na posição %d", db, c, i)
			}
		default:
			if c < '0' || c > '9' {
				return fmt.Errorf("dataBase=%q: caractere não-numérico %q na posição %d", db, c, i)
			}
		}
	}
	return nil
}

// H13 — indRemessa ∈ {I, A, S} case-sensitive (não 'i' ou 'a' minúsculos).
//
// Severidade: E.
type H13IndRemessaCaseSensitive struct{}

func (H13IndRemessaCaseSensitive) Code() string     { return "3050-H13" }
func (H13IndRemessaCaseSensitive) Sheet() string    { return "Header" }
func (H13IndRemessaCaseSensitive) Severity() string { return "E" }
func (H13IndRemessaCaseSensitive) Apply3050(_ context.Context, doc *Doc3050) error {
	ind := doc.Root.IndRemessa
	switch ind {
	case "I", "A", "S":
		return nil
	default:
		return fmt.Errorf("indRemessa=%q (esperado 'I' (inclusão), 'A' (alteração) ou 'S' (substituição) — case-sensitive)", ind)
	}
}

// H14 — nmContato sem espaços duplicados (sanity de trim).
//
// Severidade: A.
type H14NmContatoSemEspacosDuplicados struct{}

func (H14NmContatoSemEspacosDuplicados) Code() string     { return "3050-H14" }
func (H14NmContatoSemEspacosDuplicados) Sheet() string    { return "Header" }
func (H14NmContatoSemEspacosDuplicados) Severity() string { return "A" }
func (H14NmContatoSemEspacosDuplicados) Apply3050(_ context.Context, doc *Doc3050) error {
	nm := doc.Root.NmContato
	if strings.Contains(nm, "  ") {
		return fmt.Errorf("nmContato=%q contém espaços duplicados (esperado trim)", nm)
	}
	return nil
}

// H15 — telContato contém apenas dígitos após remover formatação.
//
// Severidade: A.
type H15TelContatoSemCaracteresResiduais struct{}

func (H15TelContatoSemCaracteresResiduais) Code() string     { return "3050-H15" }
func (H15TelContatoSemCaracteresResiduais) Sheet() string    { return "Header" }
func (H15TelContatoSemCaracteresResiduais) Severity() string { return "A" }
func (H15TelContatoSemCaracteresResiduais) Apply3050(_ context.Context, doc *Doc3050) error {
	tel := doc.Root.TelContato
	// Aceita (), -, espaços como formatação. Rejeita letras ou símbolos.
	for i, c := range tel {
		if c >= '0' && c <= '9' {
			continue
		}
		switch c {
		case '(', ')', '-', ' ', '+':
			continue
		}
		return fmt.Errorf("telContato=%q: caractere inválido %q na posição %d (esperado apenas dígitos + formatação ()- +espaço)", tel, c, i)
	}
	return nil
}

// ============================================================================
// 4 Regras Sistema S29-S32 — Fase 3
// ============================================================================

// S29 — dataBase deve estar entre 2009-01-01 e hoje+30 (não-futuro distante).
//
// Severidade: E (dataBase fora do range plausível = erro grave).
type S29DataBaseRangePlausivel struct{}

func (S29DataBaseRangePlausivel) Code() string     { return "3050-S29" }
func (S29DataBaseRangePlausivel) Sheet() string    { return "Sistemáticas" }
func (S29DataBaseRangePlausivel) Severity() string { return "E" }
func (S29DataBaseRangePlausivel) Apply3050(_ context.Context, doc *Doc3050) error {
	db, err := time.Parse("2006-01-02", doc.Root.DataBase)
	if err != nil {
		// Não-duplica erro de formato (H12 cobre).
		return nil
	}
	limiteInf := time.Date(2009, 1, 1, 0, 0, 0, 0, time.UTC)
	limiteSup := time.Now().UTC().AddDate(0, 0, 30)
	if db.Before(limiteInf) {
		return fmt.Errorf("dataBase=%s anterior a 2009-01-01 (CADOC 3050 não existia antes)", doc.Root.DataBase)
	}
	if db.After(limiteSup) {
		return fmt.Errorf("dataBase=%s > hoje+30 (futuro distante, provável erro de digitação)", doc.Root.DataBase)
	}
	return nil
}

// S30 — periodicidade: doc.Diario presente quando dataBase é dia útil (placeholder).
//
// Severidade: A — checa apenas que se há modelos 1-4 declarados, Diario tem dados.
// Carry-over para Fase 4: precisa de contexto de janela de envio BACEN.
type S30DiarioPresenteSeModelo1a4 struct{}

func (S30DiarioPresenteSeModelo1a4) Code() string     { return "3050-S30" }
func (S30DiarioPresenteSeModelo1a4) Sheet() string    { return "Sistemáticas" }
func (S30DiarioPresenteSeModelo1a4) Severity() string { return "A" }
func (S30DiarioPresenteSeModelo1a4) Apply3050(_ context.Context, doc *Doc3050) error {
	// Sem contexto de modelos declarados no XML (parser não distingue modelo 1/2/3/4
	// ainda), validamos apenas: se há vlrConcessoes em qualquer modalidade, Diario
	// ou Mensal deve ter entrada.
	hasData := false
	for _, m := range doc.Diario {
		if m.VlrConcessoes != nil {
			hasData = true
			break
		}
	}
	if !hasData {
		for _, m := range doc.Mensal {
			if m.VlrConcessoes != nil || m.SldCarAtiva != nil {
				hasData = true
				break
			}
		}
	}
	if !hasData && len(doc.Diario) == 0 && len(doc.Mensal) == 0 {
		return fmt.Errorf("doc vazio: nem Diario nem Mensal têm modalidades")
	}
	return nil
}

// S31 — indRemessa = "S" (substituição) → doc.AnteriorRef placeholder (carry-over).
//
// Severidade: I (placeholder — campo AnteriorRef não está no parser ainda).
type S31SubstituicaoSemAnteriorRef struct{}

func (S31SubstituicaoSemAnteriorRef) Code() string     { return "3050-S31" }
func (S31SubstituicaoSemAnteriorRef) Sheet() string    { return "Sistemáticas" }
func (S31SubstituicaoSemAnteriorRef) Severity() string { return "I" }
func (S31SubstituicaoSemAnteriorRef) Apply3050(_ context.Context, doc *Doc3050) error {
	// Carry-over: doc.AnteriorRef não está no Doc3050Root ainda (parser precisa
	// expor). Stub honesto: retorna nil e severity "I" sinaliza não-implementado.
	_ = doc
	return nil
}

// S32 — qtdTotal modalidades > 0 (sanity: doc não-vazio).
//
// Severidade: A.
type S32DocNaoVazio struct{}

func (S32DocNaoVazio) Code() string     { return "3050-S32" }
func (S32DocNaoVazio) Sheet() string    { return "Sistemáticas" }
func (S32DocNaoVazio) Severity() string { return "A" }
func (S32DocNaoVazio) Apply3050(_ context.Context, doc *Doc3050) error {
	if len(doc.Diario) == 0 && len(doc.Mensal) == 0 {
		return fmt.Errorf("doc sem modalidades (Diario vazio, Mensal vazio)")
	}
	return nil
}

// ============================================================================
// 14 Regras Individuais I15-I28 — Fase 3
// ============================================================================

// I15 — sldCarAtiva ≥ 0 em desDuplicatas.
//
// Severidade: E.
type I15DesDuplicatasSldCarNaoNeg struct{}

func (I15DesDuplicatasSldCarNaoNeg) Code() string     { return "3050-I15" }
func (I15DesDuplicatasSldCarNaoNeg) Sheet() string    { return "Individuais" }
func (I15DesDuplicatasSldCarNaoNeg) Severity() string { return "E" }
func (I15DesDuplicatasSldCarNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.Codigo != "desDuplicatas" || m.SldCarAtiva == nil {
			continue
		}
		if *m.SldCarAtiva < 0 {
			return fmt.Errorf("desDuplicatas [%d] (%s/%s): sldCarAtiva=%.2f < 0", i, m.Encargo, m.TipoCli, *m.SldCarAtiva)
		}
	}
	return nil
}

// I16 — vlrConcessoes ≥ 0 em desCheques.
//
// Severidade: E.
type I16DesChequesVlrConcessoesNaoNeg struct{}

func (I16DesChequesVlrConcessoesNaoNeg) Code() string     { return "3050-I16" }
func (I16DesChequesVlrConcessoesNaoNeg) Sheet() string    { return "Individuais" }
func (I16DesChequesVlrConcessoesNaoNeg) Severity() string { return "E" }
func (I16DesChequesVlrConcessoesNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "desCheques" || m.VlrConcessoes == nil {
			continue
		}
		if *m.VlrConcessoes < 0 {
			return fmt.Errorf("desCheques [%d] (%s/%s): vlrConcessoes=%.2f < 0", i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// I17 — txMedJuros ≥ 0 em vendor.
//
// Severidade: E.
type I17VendorTxMedJurosNaoNeg struct{}

func (I17VendorTxMedJurosNaoNeg) Code() string     { return "3050-I17" }
func (I17VendorTxMedJurosNaoNeg) Sheet() string    { return "Individuais" }
func (I17VendorTxMedJurosNaoNeg) Severity() string { return "E" }
func (I17VendorTxMedJurosNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "vendor" || m.TxMedJuros == nil {
			continue
		}
		if *m.TxMedJuros < 0 {
			return fmt.Errorf("vendor [%d] (%s/%s): txMedJuros=%.2f < 0", i, m.Encargo, m.TipoCli, *m.TxMedJuros)
		}
	}
	return nil
}

// I18 — przDecMedConcessoes ≥ 0 em compror.
//
// Severidade: E.
type I18ComprorPrzDecNaoNeg struct{}

func (I18ComprorPrzDecNaoNeg) Code() string     { return "3050-I18" }
func (I18ComprorPrzDecNaoNeg) Sheet() string    { return "Individuais" }
func (I18ComprorPrzDecNaoNeg) Severity() string { return "E" }
func (I18ComprorPrzDecNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "compror" || m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes < 0 {
			return fmt.Errorf("compror [%d] (%s/%s): przDecMedConcessoes=%d < 0", i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I19 — sldCarAtiva ≥ 0 em carCrd (cartão crédito).
//
// Severidade: E.
type I19CarCrdSldCarNaoNeg struct{}

func (I19CarCrdSldCarNaoNeg) Code() string     { return "3050-I19" }
func (I19CarCrdSldCarNaoNeg) Sheet() string    { return "Individuais" }
func (I19CarCrdSldCarNaoNeg) Severity() string { return "E" }
func (I19CarCrdSldCarNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Mensal {
		if m.Codigo != "carCrd" || m.SldCarAtiva == nil {
			continue
		}
		if *m.SldCarAtiva < 0 {
			return fmt.Errorf("carCrd [%d] (%s/%s): sldCarAtiva=%.2f < 0", i, m.Encargo, m.TipoCli, *m.SldCarAtiva)
		}
	}
	return nil
}

// I20 — vlrConcessoes ≥ 0 em carCrd.
//
// Severidade: E.
type I20CarCrdVlrConcessoesNaoNeg struct{}

func (I20CarCrdVlrConcessoesNaoNeg) Code() string     { return "3050-I20" }
func (I20CarCrdVlrConcessoesNaoNeg) Sheet() string    { return "Individuais" }
func (I20CarCrdVlrConcessoesNaoNeg) Severity() string { return "E" }
func (I20CarCrdVlrConcessoesNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "carCrd" || m.VlrConcessoes == nil {
			continue
		}
		if *m.VlrConcessoes < 0 {
			return fmt.Errorf("carCrd [%d] (%s/%s): vlrConcessoes=%.2f < 0", i, m.Encargo, m.TipoCli, *m.VlrConcessoes)
		}
	}
	return nil
}

// I21 — txMedJuros ≤ 100% em todas modalidades (sanity, regra 3026).
//
// Severidade: A.
type I21TxMedJurosMax100 struct{}

func (I21TxMedJurosMax100) Code() string     { return "3050-I21" }
func (I21TxMedJurosMax100) Sheet() string    { return "Individuais" }
func (I21TxMedJurosMax100) Severity() string { return "A" }
func (I21TxMedJurosMax100) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.TxMedJuros == nil {
				continue
			}
			if *m.TxMedJuros > 100 {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedJuros=%.2f%% > 100%% (limite BACEN)", m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedJuros)
			}
		}
	}
	return nil
}

// I22 — txMedEncOperacionais ≤ 50% em todas modalidades (regra 3027).
//
// Severidade: A.
type I22TxMedEncOperMax50 struct{}

func (I22TxMedEncOperMax50) Code() string     { return "3050-I22" }
func (I22TxMedEncOperMax50) Sheet() string    { return "Individuais" }
func (I22TxMedEncOperMax50) Severity() string { return "A" }
func (I22TxMedEncOperMax50) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.TxMedEncOperacionais == nil {
				continue
			}
			if *m.TxMedEncOperacionais > 50 {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedEncOperacionais=%.2f%% > 50%% (limite BACEN)", m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedEncOperacionais)
			}
		}
	}
	return nil
}

// I23 — przDecMedConcessoes ≤ 5000 dias em capGir (limite alto BACEN, regra 3041).
//
// Severidade: E.
type I23CapGirPrzDecMax5000 struct{}

func (I23CapGirPrzDecMax5000) Code() string     { return "3050-I23" }
func (I23CapGirPrzDecMax5000) Sheet() string    { return "Individuais" }
func (I23CapGirPrzDecMax5000) Severity() string { return "E" }
func (I23CapGirPrzDecMax5000) Apply3050(_ context.Context, doc *Doc3050) error {
	for i, m := range doc.Diario {
		if m.Codigo != "capGirPrzAte365" && m.Codigo != "capGirPrzSup365" {
			continue
		}
		if m.PrzDecMedConcessoes == nil {
			continue
		}
		if *m.PrzDecMedConcessoes > 5000 {
			return fmt.Errorf("%s [%d] (%s/%s): przDecMedConcessoes=%d > 5000 dias (limite BACEN)", m.Codigo, i, m.Encargo, m.TipoCli, *m.PrzDecMedConcessoes)
		}
	}
	return nil
}

// I24 — qtdNovContratos ≥ 0 em todas modalidades (cruzada, regra 3042).
//
// Severidade: E.
type I24QtdNovContratosNaoNeg struct{}

func (I24QtdNovContratosNaoNeg) Code() string     { return "3050-I24" }
func (I24QtdNovContratosNaoNeg) Sheet() string    { return "Individuais" }
func (I24QtdNovContratosNaoNeg) Severity() string { return "E" }
func (I24QtdNovContratosNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.QtdNovContratos == nil {
				continue
			}
			if *m.QtdNovContratos < 0 {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): qtdNovContratos=%d < 0", m.Codigo, i, m.Encargo, m.TipoCli, *m.QtdNovContratos)
			}
		}
	}
	return nil
}

// I25 — sldCedido ≥ 0 em todas modalidades (cruzada, regra 3043).
//
// Severidade: E.
type I25SldCedidoNaoNeg struct{}

func (I25SldCedidoNaoNeg) Code() string     { return "3050-I25" }
func (I25SldCedidoNaoNeg) Sheet() string    { return "Individuais" }
func (I25SldCedidoNaoNeg) Severity() string { return "E" }
func (I25SldCedidoNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.SldCedido == nil {
				continue
			}
			if *m.SldCedido < 0 {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCedido=%.2f < 0", m.Codigo, i, m.Encargo, m.TipoCli, *m.SldCedido)
			}
		}
	}
	return nil
}

// I26 — sldAdquirido ≥ 0 em todas modalidades (cruzada, regra 3044).
//
// Severidade: E.
type I26SldAdquiridoNaoNeg struct{}

func (I26SldAdquiridoNaoNeg) Code() string     { return "3050-I26" }
func (I26SldAdquiridoNaoNeg) Sheet() string    { return "Individuais" }
func (I26SldAdquiridoNaoNeg) Severity() string { return "E" }
func (I26SldAdquiridoNaoNeg) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.SldAdquirido == nil {
				continue
			}
			if *m.SldAdquirido < 0 {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): sldAdquirido=%.2f < 0", m.Codigo, i, m.Encargo, m.TipoCli, *m.SldAdquirido)
			}
		}
	}
	return nil
}

// I27 — sldCarAtiva > 0 → txMaxima > txMinima (consistência, regra 3029).
//
// Severidade: A.
type I27SldCarAtivaImpoeTxMaxGtMin struct{}

func (I27SldCarAtivaImpoeTxMaxGtMin) Code() string     { return "3050-I27" }
func (I27SldCarAtivaImpoeTxMaxGtMin) Sheet() string    { return "Individuais" }
func (I27SldCarAtivaImpoeTxMaxGtMin) Severity() string { return "A" }
func (I27SldCarAtivaImpoeTxMaxGtMin) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.SldCarAtiva == nil || *m.SldCarAtiva <= 0 {
				continue
			}
			if m.TxMaxima == nil || m.TxMinima == nil {
				continue
			}
			if *m.TxMaxima <= *m.TxMinima {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): sldCarAtiva>0 mas txMaxima=%.2f <= txMinima=%.2f (inconsistência)", m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMaxima, *m.TxMinima)
			}
		}
	}
	return nil
}

// I28 — indRemessa = "I" → qtdNovContratos ≥ 1 (cruzada, regra 2001).
//
// Severidade: A.
type I28IndRemessaIExigeNovContratos struct{}

func (I28IndRemessaIExigeNovContratos) Code() string     { return "3050-I28" }
func (I28IndRemessaIExigeNovContratos) Sheet() string    { return "Individuais" }
func (I28IndRemessaIExigeNovContratos) Severity() string { return "A" }
func (I28IndRemessaIExigeNovContratos) Apply3050(_ context.Context, doc *Doc3050) error {
	if doc.Root.IndRemessa != "I" {
		return nil
	}
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.QtdNovContratos != nil && *m.QtdNovContratos >= 1 {
				return nil // pelo menos 1 contrato novo encontrado
			}
			_ = i
		}
	}
	return fmt.Errorf("indRemessa=I (inclusão) mas nenhum qtdNovContratos ≥ 1 encontrado")
}

// ============================================================================
// Carry-over S09, S13, S24 — sair de stub para implementação real
// ============================================================================

// S09 — dataBase é dia útil BACEN (regra 2009/calendário).
//
// Severidade: E.
type S09DiasUteis struct{}

func (S09DiasUteis) Code() string     { return "3050-S09" }
func (S09DiasUteis) Sheet() string    { return "Sistemáticas" }
func (S09DiasUteis) Severity() string { return "E" }
func (S09DiasUteis) Apply3050(_ context.Context, doc *Doc3050) error {
	db, err := time.Parse("2006-01-02", doc.Root.DataBase)
	if err != nil {
		return nil // H12 cobre formato.
	}
	if !IsDiaUtilBACEN(db) {
		return fmt.Errorf("dataBase=%s não é dia útil BACEN (sábado/domingo/feriado)", doc.Root.DataBase)
	}
	return nil
}

// S13 — dataBase é último dia útil do mês BACEN (regras 3031-3035).
//
// Severidade: A.
type S13UltimoDiaUtil struct{}

func (S13UltimoDiaUtil) Code() string     { return "3050-S13" }
func (S13UltimoDiaUtil) Sheet() string    { return "Sistemáticas" }
func (S13UltimoDiaUtil) Severity() string { return "A" }
func (S13UltimoDiaUtil) Apply3050(_ context.Context, doc *Doc3050) error {
	db, err := time.Parse("2006-01-02", doc.Root.DataBase)
	if err != nil {
		return nil
	}
	if !IsUltimoDiaUtilMes(db) {
		return fmt.Errorf("dataBase=%s não é último dia útil do mês BACEN", doc.Root.DataBase)
	}
	return nil
}

// S24 — txMedJurosAjustada ≤ txMedJuros (regra 3051).
//
// Severidade: E.
// Carry-over Fase 3: parser agora expõe TxMedJurosAjustada (DT-29).
type S24TxJurosAjustadaLeTxJuros struct{}

func (S24TxJurosAjustadaLeTxJuros) Code() string     { return "3050-S24" }
func (S24TxJurosAjustadaLeTxJuros) Sheet() string    { return "Sistemáticas" }
func (S24TxJurosAjustadaLeTxJuros) Severity() string { return "E" }
func (S24TxJurosAjustadaLeTxJuros) Apply3050(_ context.Context, doc *Doc3050) error {
	for _, list := range [][]Modalidade{doc.Diario, doc.Mensal} {
		for i, m := range list {
			if m.TxMedJurosAjustada == nil || m.TxMedJuros == nil {
				continue
			}
			if *m.TxMedJurosAjustada > *m.TxMedJuros {
				return fmt.Errorf("modalidade %s [%d] (%s/%s): txMedJurosAjustada=%.2f > txMedJuros=%.2f (regra 3051)", m.Codigo, i, m.Encargo, m.TipoCli, *m.TxMedJurosAjustada, *m.TxMedJuros)
			}
		}
	}
	return nil
}

// ============================================================================
// Builtin3050 — Registry de regras 3050 (81 total após Fase 3)
// ============================================================================

// Builtin3050 retorna o registry com as 81 regras 3050 implementadas (Fases 1+2+3).
//
// Cobertura catálogo TXB_V11:
//   - 14 Agregadas A01-A14 (Fase 1 — severity E/A conforme regra).
//   - 14 Stubs S01-S14 (Fase 1 — severity I, honestos; S09/S13 saem de stub na Fase 3).
//   - 14 Sistemáticas S15-S28 (Fase 2 — severity E/A/I conforme regra; S24 sai de stub na Fase 3).
//   - 14 Individuais/Cruzadas I01-I14 (Fase 2 — severity E/A conforme regra).
//   - 6 Header H10-H15 (Fase 3 — severity E/A conforme regra).
//   - 4 Sistema S29-S32 (Fase 3 — severity A/I conforme regra).
//   - 14 Individuais I15-I28 (Fase 3 — severity E/A conforme regra).
//
// Total: 81/170 = 47.6% (Fases 1+2+3).
func Builtin3050() *Registry {
	r := NewRegistry()

	// 14 Agregadas (Fase 1)
	r.Register3050(A01SldCarSomaFaixas{})
	r.Register3050(A02SldCedidoMenosAdquirido{})
	r.Register3050(A03SldBaiPrejuizoLeSldCar{})
	r.Register3050(A04SldCarMaisCedidoVsAdquirido{})
	r.Register3050(A05CNPJRaiz{})
	r.Register3050(A06DataBaseFormato{})
	r.Register3050(A07IndRemessaValido{})
	r.Register3050(A08NmContatoObrigatorio{})
	r.Register3050(A09TxMedJurosLimite{})
	r.Register3050(A10TxMedEncFiscaisLimite{})
	r.Register3050(A11TxMedEncOperacionaisLimite{})
	r.Register3050(A12TxMinimaLeMaxima{})
	r.Register3050(A13PrzDecMedConcessoesNaoNeg{})
	r.Register3050(A14PrzMedCarteiraNaoNeg{})

	// 14 Stubs (Fase 1, severity I)
	r.Register3050(S01MatrizEncargoModalidade{})
	r.Register3050(S02DocNaoEsperado{})
	r.Register3050(S03ArquivoDispensado{})
	r.Register3050(S04HeaderDetalhe{})
	r.Register3050(S05ArquivoJaProcessado{})
	r.Register3050(S06SubstituicaoSemOriginal{})
	r.Register3050(S07Compactacao{})
	r.Register3050(S08DataBaseFutura{})
	r.Register3050(S09DiasUteis{})
	r.Register3050(S10DocAnterior{})
	r.Register3050(S11VlrConcessoesVsTaxas{})
	r.Register3050(S12PrzMedSeSld{})
	r.Register3050(S13UltimoDiaUtil{})
	r.Register3050(S14Cruzadas{})

	// 14 Sistemáticas (Fase 2)
	r.Register3050(S15DataBaseValida{})
	r.Register3050(S16NmContatoLength{})
	r.Register3050(S17TelContatoFormato{})
	r.Register3050(S18VlrConcessoesZeroTxJurosZero{})
	r.Register3050(S19TxJurosZeroVlrConcessoesPos{})
	r.Register3050(S20TxEncOperZeroVlrConcessoesPos{})
	r.Register3050(S21PrzDecZeroVlrConcessoesPos{})
	r.Register3050(S22PrzDecPosVlrConcessoesPos{})
	r.Register3050(S23PrzMedCondicional{})
	r.Register3050(S24TxJurosAjustadaLeTxJuros{})
	r.Register3050(S25CNPJNaoZero{})
	r.Register3050(S26CodigoEncargoTipoCliUnico{})
	r.Register3050(S27SldBaiPrejuizoNaoNeg{})
	r.Register3050(S28QtdNovContratosNaoNeg{})

	// 14 Individuais/Cruzadas (Fase 2)
	r.Register3050(I01CapGirAte365{})
	r.Register3050(I02CapGirSup365{})
	r.Register3050(I03CredPesNaoConsignadoSldCar{})
	r.Register3050(I04CredPesNaoConsignadoVlrConcessoes{})
	r.Register3050(I05CredPesNaoConsignadoSldAdquirido{})
	r.Register3050(I06CredPesNaoConsignadoSldCedido{})
	r.Register3050(I07PrzMedCarteiraBaixo{})
	r.Register3050(I08PrzMedCarteiraAlto{})
	r.Register3050(I09PrzDecMedConcessoesBaixo{})
	r.Register3050(I10PrzDecMedConcessoesAlto{})
	r.Register3050(I11SldCarAtivaMuitoBaixo{})
	r.Register3050(I12SldCarAtivaMuitoAlto{})
	r.Register3050(I13VlrConcessoesMuitoBaixo{})
	r.Register3050(I14VlrConcessoesMuitoAlto{})

	// 14 Individuais adicionais (Fase 3: I15-I28)
	r.Register3050(I15DesDuplicatasSldCarNaoNeg{})
	r.Register3050(I16DesChequesVlrConcessoesNaoNeg{})
	r.Register3050(I17VendorTxMedJurosNaoNeg{})
	r.Register3050(I18ComprorPrzDecNaoNeg{})
	r.Register3050(I19CarCrdSldCarNaoNeg{})
	r.Register3050(I20CarCrdVlrConcessoesNaoNeg{})
	r.Register3050(I21TxMedJurosMax100{})
	r.Register3050(I22TxMedEncOperMax50{})
	r.Register3050(I23CapGirPrzDecMax5000{})
	r.Register3050(I24QtdNovContratosNaoNeg{})
	r.Register3050(I25SldCedidoNaoNeg{})
	r.Register3050(I26SldAdquiridoNaoNeg{})
	r.Register3050(I27SldCarAtivaImpoeTxMaxGtMin{})
	r.Register3050(I28IndRemessaIExigeNovContratos{})

	// 4 Sistema adicionais (Fase 3: S29-S32)
	r.Register3050(S29DataBaseRangePlausivel{})
	r.Register3050(S30DiarioPresenteSeModelo1a4{})
	r.Register3050(S31SubstituicaoSemAnteriorRef{})
	r.Register3050(S32DocNaoVazio{})

	// 6 Header (Fase 3: H10-H15)
	r.Register3050(H10CNPJLength{})
	r.Register3050(H11CNPJAllDigits{})
	r.Register3050(H12DataBaseFormatoRigoroso{})
	r.Register3050(H13IndRemessaCaseSensitive{})
	r.Register3050(H14NmContatoSemEspacosDuplicados{})
	r.Register3050(H15TelContatoSemCaracteresResiduais{})

	return r
}

// abs helper (substitui math.Abs pra evitar import extra em testes).
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
