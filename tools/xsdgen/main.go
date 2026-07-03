// xsdgen: gera XSD do CADOC 3040 a partir de leiautes.json (Radiant Sentinel)
//
// Uso:
//   cd tools/
//   go run ./xsdgen -in ../_catalogos/leiautes.json -cadoc 3040 -out ../_catalogos/3040_generated.xsd
//
// Sprint 2 / T.1 — gera XSD básico (só elementos + atributos + obrigatoriedade).
// Tipos BACEN (A8, N19,2, etc) são mapeados para xs:string por enquanto;
// T.1.5 pode mapear para tipos XSD mais específicos.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type SheetRow struct {
	Sheet  string   `json:"sheet"`
	Values []string `json:"values"`
}

type Leiaute struct {
	Source    string     `json:"source"`
	TotalRows int        `json:"total_rows"`
	Rows      []SheetRow `json:"rows"`
}

type Catalog struct {
	Metadata map[string]any `json:"_metadata"`
	Leiautes map[string]Leiaute `json:"leiautes"`
}

// tipoBACENtoXSD faz o mapping básico de tipos BACEN para XSD
func tipoBACENtoXSD(tipo string) string {
	t := strings.TrimSpace(tipo)
	if t == "" || t == "-" {
		return "xs:string"
	}
	upper := strings.ToUpper(t)
	// Numéricos
	if strings.HasPrefix(upper, "N") {
		return "xs:decimal"
	}
	// Data A8, A10 → string formatado
	if strings.HasPrefix(upper, "A") {
		return "xs:string"
	}
	// Default
	return "xs:string"
}

func main() {
	in := flag.String("in", "../_catalogos/leiautes.json", "Caminho do leiautes.json")
	cadoc := flag.String("cadoc", "3040", "Código do CADOC (ex: 3040, 3050)")
	out := flag.String("out", "../_catalogos/3040_generated.xsd", "Arquivo XSD de saída")
	flag.Parse()

	// Carrega catálogo
	data, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalf("Erro lendo %s: %v", *in, err)
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		log.Fatalf("Erro parseando %s: %v", *in, err)
	}

	lei, ok := cat.Leiautes[*cadoc]
	if !ok {
		log.Fatalf("CADOC %s não encontrado no catálogo", *cadoc)
	}

	// Gera XSD básico
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString(fmt.Sprintf("<!-- XSD gerado automaticamente por Radiant Sentinel xsdgen a partir de leiautes.json (%s) -->\n", lei.Source))
	b.WriteString("<!-- NÃO É OFICIAL — BACEN só publica XSD de 3045 e 3026 -->\n")
	b.WriteString("<!-- Use com cautela: tipos BACEN foram mapeados genericamente para xs:string/decimal -->\n")
	b.WriteString("<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\" elementFormDefault=\"qualified\">\n\n")

	// Estado: detectar grupos hierárquicos
	inHeader := false
	rowCount := 0

	for _, row := range lei.Rows {
		if len(row.Values) < 1 {
			continue
		}
		first := strings.TrimSpace(row.Values[0])
		if first == "" {
			continue
		}

		// Detecta cabeçalho de seção "Para cada cliente (Cli)"
		if strings.HasPrefix(first, "Para cada") || strings.Contains(first, "(de cada") {
			// Ex: "Para cada cliente (Cli)" → tag = Cli
			parts := strings.Split(first, "(")
			if len(parts) >= 2 {
				tag := strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
				b.WriteString(fmt.Sprintf("\n<!-- %s -->\n", first))
				b.WriteString(fmt.Sprintf("<xs:complexType name=\"%s\">\n", tag))
				b.WriteString("  <xs:sequence>\n")
				b.WriteString(fmt.Sprintf("    <xs:element name=\"%s\" type=\"%s\" minOccurs=\"0\" maxOccurs=\"unbounded\"/>\n", tag, tag))
				b.WriteString("  </xs:sequence>\n")
				b.WriteString("</xs:complexType>\n\n")
			}
			continue
		}

		// Header row: "Campo | Formato | Obrigatório | Elemento (Tag) | Atributo"
		if first == "Campo" {
			inHeader = true
			continue
		}
		// Linha vazia depois do header
		if first == "" && inHeader {
			inHeader = false
			continue
		}

		// Linha de campo real
		if len(row.Values) >= 4 && inHeader {
			campo := strings.TrimSpace(row.Values[0])
			formato := strings.TrimSpace(safeGet(row.Values, 1))
			obrigatorio := strings.TrimSpace(safeGet(row.Values, 2))
			tag := strings.TrimSpace(safeGet(row.Values, 3))
			atributo := strings.TrimSpace(safeGet(row.Values, 4))

			if campo == "" || (tag == "" && atributo == "") {
				continue
			}

			// Linha de grupo (não campo): "Cabeçalho do documento XML", etc
			if tag == "" && atributo == "" {
				// É um header de seção, não campo
				continue
			}

			rowCount++
			minOcc := "0"
			if strings.Contains(strings.ToLower(obrigatorio), "sim") {
				minOcc = "1"
			}

			xsdType := tipoBACENtoXSD(formato)
			useAttr := "optional"
			if minOcc == "1" {
				useAttr = "required"
			}

			if tag != "" {
				// Elemento
				b.WriteString(fmt.Sprintf("  <!-- %s -->\n", campo))
				b.WriteString(fmt.Sprintf("  <xs:element name=\"%s\" type=\"%s\" minOccurs=\"%s\"/>\n",
					tag, xsdType, minOcc))
			} else if atributo != "" {
				// Atributo (pode ser múltiplos separados por vírgula)
				atrs := splitAtributos(atributo)
				for _, a := range atrs {
					b.WriteString(fmt.Sprintf("  <!-- %s --> %s\n", campo, a))
					b.WriteString(fmt.Sprintf("  <xs:attribute name=\"%s\" type=\"%s\" use=\"%s\"/>\n",
						a, xsdType, useAttr))
				}
			}
		}
	}

	b.WriteString("\n</xs:schema>\n")

	// Escreve
	if err := os.WriteFile(*out, []byte(b.String()), 0644); err != nil {
		log.Fatalf("Erro escrevendo %s: %v", *out, err)
	}

	fmt.Printf("✓ XSD gerado em %s\n", *out)
	fmt.Printf("  - CADOC: %s\n", *cadoc)
	fmt.Printf("  - Linhas processadas: %d\n", rowCount)
	fmt.Printf("  - Total rows na sheet: %d\n", lei.TotalRows)
	fmt.Printf("  - Fonte: %s\n", lei.Source)
}

func safeGet(arr []string, idx int) string {
	if idx >= len(arr) {
		return ""
	}
	return arr[idx]
}

func splitAtributos(s string) []string {
	// BACEN separa atributos por vírgula ou \n
	s = strings.ReplaceAll(s, "\n", ",")
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Remove anotações tipo "*não obrigatório..."
		if strings.Contains(p, "*") {
			parts2 := strings.Split(p, "*")
			p = strings.TrimSpace(parts2[0])
		}
		if p != "" && p != "-" {
			out = append(out, p)
		}
	}
	return out
}