// sentinel-audit: executa as primeiras 5 críticas do 3040 contra um XML de exemplo
// (Radiant Sentinel — Sprint 2 / T.2)
//
// Uso:
//   cd tools/
//   go run ./sentinel-audit -xml ../3040/exemploDesempenhoOperacao.xml
//
// Implementa regras B01-B05 da planilha SCR3040_Criticas.xls:
//   B01: arquivo XML deve ser válido contra XSD
//   B02: arquivo .ZIP deve ser gerado pelo aplicativo validador
//   B03: instituição remetente deve possuir autorização
//   B04: arquivo XML deve estar em codificação válida
//   B05: arquivo não pode estar vazio

package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// Critica representa uma regra de validação
type Critica struct {
	Cadoc      string `json:"cadoc"`
	Sheet      string `json:"sheet"`
	Codigo     string `json:"codigo"`
	Habilitado string `json:"habilitado?"`
	Regra      string `json:"regra"`
	Descricao  string `json:"descrição"`
}

// ValidationError representa um erro encontrado
type ValidationError struct {
	Critica   string
	Regra     string
	Severity  string
	Message   string
}

// Doc3040 é a estrutura mínima do XML do 3040
type Doc3040 struct {
	XMLName xml.Name `xml:"Doc3040"`
	CNPJ    string   `xml:"CNPJ,attr"`
	DtBase  string   `xml:"DtBase,attr"`
	// ... outros campos
}

// B01: arquivo XML deve ser válido (parsing OK)
func validarB01(xmlContent []byte) *ValidationError {
	var doc Doc3040
	if err := xml.Unmarshal(xmlContent, &doc); err != nil {
		return &ValidationError{
			Critica: "B01", Regra: "Erro XML",
			Severity: "ERRO",
			Message: fmt.Sprintf("XML não parseia: %v", err),
		}
	}
	return nil
}

// B04: arquivo XML deve estar em codificação válida
func validarB04(xmlContent []byte) *ValidationError {
	// Verifica declaração XML
	if !strings.HasPrefix(strings.TrimSpace(string(xmlContent)), "<?xml") {
		return &ValidationError{
			Critica: "B04", Regra: "Codificação inválida",
			Severity: "ERRO",
			Message: "Arquivo não começa com declaração <?xml",
		}
	}
	// Procura encoding válido
	content := string(xmlContent)
	if !strings.Contains(content, "encoding=") {
		return &ValidationError{
			Critica: "B04", Regra: "Codificação não declarada",
			Severity: "AVISO",
			Message: "Encoding não declarado — assumindo UTF-8",
		}
	}
	return nil
}

// B05: arquivo não pode estar vazio
func validarB05(xmlContent []byte) *ValidationError {
	if len(xmlContent) == 0 {
		return &ValidationError{
			Critica: "B05", Regra: "Arquivo vazio",
			Severity: "ERRO",
			Message: "Arquivo XML está vazio",
		}
	}
	if len(xmlContent) < 50 {
		return &ValidationError{
			Critica: "B05", Regra: "Arquivo muito pequeno",
			Severity: "ERRO",
			Message: fmt.Sprintf("Arquivo XML tem apenas %d bytes", len(xmlContent)),
		}
	}
	return nil
}

// Regra B01-B05 carrega críticas do JSON
func carregarCriticas(path string, codigos []string) []Critica {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("⚠ Não foi possível ler %s: %v", path, err)
		return nil
	}
	var cat struct {
		Criticas map[string][]Critica `json:"criticas"`
	}
	if err := json.Unmarshal(data, &cat); err != nil {
		log.Printf("⚠ Erro parseando %s: %v", path, err)
		return nil
	}
	all := cat.Criticas["3040"]
	var out []Critica
	for _, c := range all {
		for _, code := range codigos {
			if c.Codigo == code {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

func main() {
	xmlPath := flag.String("xml", "../3040/exemploDesempenhoOperacao.xml", "XML do 3040 pra validar")
	jsonPath := flag.String("json", "../_catalogos/criticas.json", "Catálogo de críticas")
	flag.Parse()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("  Radiant Sentinel Audit — Sprint 2 / T.2")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("XML: %s\n", *xmlPath)
	fmt.Printf("Catálogo: %s\n\n", *jsonPath)

	// Carrega XML
	xmlContent, err := os.ReadFile(*xmlPath)
	if err != nil {
		log.Fatalf("Erro lendo XML: %v", err)
	}
	fmt.Printf("✓ XML carregado: %d bytes\n\n", len(xmlContent))

	// Carrega regras
	codigos := []string{"B01", "B02", "B03", "B04", "B05"}
	criticas := carregarCriticas(*jsonPath, codigos)
	fmt.Printf("✓ Críticas carregadas: %d regras (B01-B05)\n\n", len(criticas))

	// Executa regras
	var erros []ValidationError

	// B01
	if err := validarB01(xmlContent); err != nil {
		erros = append(erros, *err)
	} else {
		fmt.Println("✓ B01 OK — XML parseia corretamente")
	}

	// B04
	if err := validarB04(xmlContent); err != nil {
		erros = append(erros, *err)
	} else {
		fmt.Println("✓ B04 OK — Codificação válida")
	}

	// B05
	if err := validarB05(xmlContent); err != nil {
		erros = append(erros, *err)
	} else {
		fmt.Println("✓ B05 OK — Arquivo não vazio")
	}

	// B02 e B03 — skip por enquanto (precisam STA stub)
	fmt.Println("⤵ B02 SKIP — verificação de ZIP gerado por validador (requer STA stub)")
	fmt.Println("⤵ B03 SKIP — verificação de autorização BACEN (requer BACEN API)")

	// Relatório
	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Printf("  Resumo: %d erros / %d regras executadas\n", len(erros), 3)
	fmt.Println("═══════════════════════════════════════════════════════")

	if len(erros) > 0 {
		fmt.Println("\nErros encontrados:")
		for _, e := range erros {
			fmt.Printf("  [%s] %s: %s\n", e.Severity, e.Critica, e.Message)
		}
		os.Exit(1)
	}

	fmt.Println("\n✓ Todas as regras passaram!")
}