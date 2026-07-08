// Doc3026 — Documento 3026 (Estrutura Organizacional) — CADOC.
//
// Sprint 50: parser para 3026 — Estrutura Organizacional de Conglomerados.
//
// Estrutura:
//
//	<Doc3026 CNPJ="..." DtBase="YYYY-MM">
//	  <CongEcon Cd="1">
//	    <Part Cd="12345678" Tp="2"/>
//	  </CongEcon>
//	  <CongEcon Cd="2">
//	    <Part Tp="2" Cd="12345678"/>
//	  </CongEcon>
//	</Doc3026>
//
// Referência: BACEN — leiaute 3026.
package rules

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
)

// Doc3026 é o documento 3026 parseado (Estrutura Organizacional).
type Doc3026 struct {
	Root      Doc3026Root
	CongEcons []CongEcon
}

// Doc3026Root é o elemento raiz do 3026.
type Doc3026Root struct {
	CNPJ     string
	DataBase string // YYYY-MM
}

// CongEcon representa um conglomerado econômico no documento 3026.
type CongEcon struct {
	Cd    string // código do conglomerado
	Parts []Part // participantes do conglomerado
}

// Part representa um participante (instituição) em um conglomerado.
type Part struct {
	Cd string // CNPJ ou código da instituição
	Tp string // tipo de participante (1=líder, 2=membro, etc.)
}

// PartialParseError3026 indica erro de parse no 3026.
type PartialParseError3026 struct {
	Err error
}

func (e *PartialParseError3026) Error() string { return "parse 3026: " + e.Err.Error() }
func (e *PartialParseError3026) Unwrap() error { return e.Err }

// ParseDoc3026 faz parse do XML 3026.
func ParseDoc3026(data []byte) (*Doc3026, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	doc := &Doc3026{}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return doc, &PartialParseError3026{Err: fmt.Errorf("token: %w", err)}
		}

		switch t := tok.(type) {
		case xml.StartElement:
			tag := t.Name.Local
			switch tag {
			case "Doc3026":
				for _, a := range t.Attr {
					switch a.Name.Local {
					case "CNPJ":
						doc.Root.CNPJ = a.Value
					case "DtBase":
						doc.Root.DataBase = a.Value
					}
				}
			case "CongEcon":
				cong := CongEcon{}
				for _, a := range t.Attr {
					if a.Name.Local == "Cd" {
						cong.Cd = a.Value
					}
				}
				// Decode nested Part elements
				for {
					tok2, err := dec.Token()
					if err == io.EOF {
						break
					}
					if err != nil {
						return doc, &PartialParseError3026{Err: fmt.Errorf("CongEcon token: %w", err)}
					}
					if el, ok := tok2.(xml.EndElement); ok && el.Name.Local == "CongEcon" {
						break
					}
					if el, ok := tok2.(xml.StartElement); ok && el.Name.Local == "Part" {
						p := Part{}
						for _, a := range el.Attr {
							switch a.Name.Local {
							case "Cd":
								p.Cd = a.Value
							case "Tp":
								p.Tp = a.Value
							}
						}
						cong.Parts = append(cong.Parts, p)
					}
				}
				doc.CongEcons = append(doc.CongEcons, cong)
			}
		}
	}

	return doc, nil
}

// Validar3026Basico faz validação básica do 3026 (consistência interna).
func Validar3026Basico(doc *Doc3026) error {
	if doc == nil {
		return fmt.Errorf("3026 nil")
	}
	if doc.Root.CNPJ == "" {
		return fmt.Errorf("CNPJ vazio")
	}
	// Cada CongEcon precisa de pelo menos 1 Part
	for i, cong := range doc.CongEcons {
		if len(cong.Parts) == 0 {
			return fmt.Errorf("CongEcon[%d] Cd=%s sem participantes", i, cong.Cd)
		}
		// Pelo menos 1 part com Tp="1" (líder)
		hasLider := false
		for _, p := range cong.Parts {
			if p.Tp == "1" {
				hasLider = true
				break
			}
		}
		if !hasLider {
			return fmt.Errorf("CongEcon[%d] Cd=%s sem participante líder (Tp=1)", i, cong.Cd)
		}
	}
	return nil
}

// 3026-01 — CNPJ da instituição líder presente.
//
// Severidade: E
type O302601CNPJLider struct{}

func (O302601CNPJLider) Code() string     { return "3026-01" }
func (O302601CNPJLider) Sheet() string    { return "EstruturaOrganizacional" }
func (O302601CNPJLider) Severity() string { return "E" }

func (O302601CNPJLider) Apply(_ context.Context, doc *Doc3026) error {
	if doc == nil || doc.Root.CNPJ == "" {
		return fmt.Errorf("CNPJ da instituição vazio")
	}
	return nil
}

// 3026-02 — Data-base presente e em formato válido (YYYY-MM).
//
// Severidade: A
type O302602DataBase struct{}

func (O302602DataBase) Code() string     { return "3026-02" }
func (O302602DataBase) Sheet() string    { return "EstruturaOrganizacional" }
func (O302602DataBase) Severity() string { return "A" }

func (O302602DataBase) Apply(_ context.Context, doc *Doc3026) error {
	if doc == nil || doc.Root.DataBase == "" {
		return fmt.Errorf("DataBase vazia")
	}
	if len(doc.Root.DataBase) != 7 || doc.Root.DataBase[4] != '-' {
		return fmt.Errorf("DataBase=%q não está em formato YYYY-MM", doc.Root.DataBase)
	}
	return nil
}

// 3026-03 — Conglomerado tem ao menos um participante.
//
// Severidade: E
type O302603ConglomeradoVazio struct{}

func (O302603ConglomeradoVazio) Code() string     { return "3026-03" }
func (O302603ConglomeradoVazio) Sheet() string    { return "EstruturaOrganizacional" }
func (O302603ConglomeradoVazio) Severity() string { return "E" }

func (O302603ConglomeradoVazio) Apply(_ context.Context, doc *Doc3026) error {
	if doc == nil {
		return fmt.Errorf("documento nil")
	}
	if len(doc.CongEcons) == 0 {
		return fmt.Errorf("documento sem conglomerados econômicos")
	}
	return nil
}

// 3026-04 — CNPJ do participante não pode ser vazio.
//
// Severidade: E
type O302604ParticipanteCNPJVazio struct{}

func (O302604ParticipanteCNPJVazio) Code() string     { return "3026-04" }
func (O302604ParticipanteCNPJVazio) Sheet() string    { return "EstruturaOrganizacional" }
func (O302604ParticipanteCNPJVazio) Severity() string { return "E" }

func (O302604ParticipanteCNPJVazio) Apply(_ context.Context, doc *Doc3026) error {
	if doc == nil {
		return nil
	}
	for i, cong := range doc.CongEcons {
		for j, p := range cong.Parts {
			if p.Cd == "" {
				return fmt.Errorf("CongEcon[%d] Part[%d]: CNPJ/Cd vazio", i, j)
			}
		}
	}
	return nil
}
