// Package crossdoc implementa validação cross-document (Layer 3).
//
// Diferencial proprietário Radiant Norma: BCValidador valida UM CADOC por
// vez. Radiant Norma valida o ecossistema inteiro — checa consistência
// entre 3040, 4111, DRSAC etc.
//
// Sprint 6 v1.5.0: implementação inicial com 3 regras cross-doc.
//
// Arquitetura:
//   - CrossDocRule: interface comum (similar a rules.Rule do package audit)
//   - registry: indexa regras por código
//   - engine: orquestra (carrega múltiplos docs em paralelo, executa regras)
//   - rules/3040_4111.go: regras iniciais (3)
//
// API: POST /v1/crossdoc/validate recebe `{cadocs: {3040: xml, 4111: xml}}`
// e retorna erros/lista de regras passadas.
package crossdoc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// DocSet agrupa os documentos parseados (1 por CADOC).
//
// Cada campo é o XML raw — campos parseados são derivados lazy na regra.
// Cross-doc rules acessam campos específicos conforme necessário.
//
// Mapa genérico (não tipado por CADOC) porque diferentes regras precisam
// de diferentes CADOCs. Performance: parsear sob demanda, não sempre.
type DocSet struct {
	Cadocs map[string]string // cadoc_code → XML raw
}

// Has retorna true se CADOC foi fornecido.
func (d *DocSet) Has(cadoc string) bool {
	_, ok := d.Cadocs[cadoc]
	return ok
}

// Get retorna XML raw de um CADOC ou vazio.
func (d *DocSet) Get(cadoc string) string {
	return d.Cadocs[cadoc]
}

// CrossDocRule é a interface de regra cross-document.
type CrossDocRule interface {
	// Code identifica a regra (ex: "XD-001").
	Code() string

	// Description é a descrição human-readable.
	Description() string

	// Severity: "E" (Erro bloqueante), "A" (Aviso), "I" (Informativo).
	Severity() string

	// RequiredDocs retorna os CADOCs que devem estar presentes no DocSet.
	// Se algum estiver ausente, regra retorna erro descritivo.
	RequiredDocs() []string

	// Apply valida cruzando os docs em docs.
	// Retorna nil se OK, ou erro descritivo.
	Apply(ctx context.Context, docs *DocSet) error
}

// Error é o tipo de erro retornado por uma regra cross-doc.
//
// Implementa erro padrão + código + severidade pra metadata.
type Error struct {
	Code     string // código da regra (ex: "XD-001")
	Severity string // E/A/I
	Message  string // descrição do erro
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s/%s] %s", e.Code, e.Severity, e.Message)
}

// NewError cria um erro cross-doc.
func NewError(code, severity, message string) error {
	return &Error{Code: code, Severity: severity, Message: message}
}

// ============================
// XML helpers — extract fields por tag/text
// ============================

// ExtractTextBetween extrai texto entre 2 tags XML. Útil para campos simples.
//
// Ex: ExtractTextBetween("<Doc3040><CNPJ>123</CNPJ></Doc3040>", "CNPJ")
// retorna "123".
//
// Best-effort: usa parser XML padrão. Para casos mais complexos (com
// namespaces, atributos), refatorar para usar encoding/xml Decoder.
//
// Exportado (Sprint 6 v1.5.0) para uso por rules/3040_4111.go.
func ExtractTextBetween(xmlContent, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	openIdx := strings.Index(xmlContent, openTag)
	if openIdx == -1 {
		return ""
	}
	closeIdx := strings.Index(xmlContent[openIdx+len(openTag):], closeTag)
	if closeIdx == -1 {
		return ""
	}
	return strings.TrimSpace(xmlContent[openIdx+len(openTag) : openIdx+len(openTag)+closeIdx])
}

// ExtractSumOfTag soma valores numéricos em tags repetidas.
//
// Útil pra totalizar QtdOp em todos os <Agreg> do 3040, ou QtdCli em
// todos os clientes do 4111.
//
// Retorna 0 se nenhum match. Tolera vírgula/ponto como decimal.
//
// Exportado (Sprint 6 v1.5.0).
func ExtractSumOfTag(xmlContent, parentTag, childTag string) float64 {
	var total float64
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))
	for {
		t, err := decoder.Token()
		if err != nil || t == nil {
			break
		}
		if se, ok := t.(xml.StartElement); ok && se.Name.Local == parentTag {
			// Dentro de parentTag, ler todos os childTag
			for {
				inner, err := decoder.Token()
				if err != nil || inner == nil {
					break
				}
				if innerSE, ok := inner.(xml.StartElement); ok && innerSE.Name.Local == childTag {
					var content strings.Builder
					for {
						text, _ := decoder.Token()
						if text == nil {
							break
						}
						if ch, ok := text.(xml.CharData); ok {
							content.Write(ch)
						}
						if se, ok := text.(xml.EndElement); ok && se.Name.Local == childTag {
							val, _ := parseNum(strings.TrimSpace(content.String()))
							total += val
							break
						}
						if _, ok := text.(xml.StartElement); ok {
							// Pula tags aninhadas complexas
							for {
								if _, err := decoder.Token(); err != nil {
									break
								}
							}
						}
					}
				}
				if se, ok := inner.(xml.EndElement); ok && se.Name.Local == parentTag {
					break
				}
			}
		}
	}
	return total
}

// CountTag conta ocorrências de uma tag. Suporta self-closing (`<A/>`).
func CountTag(xmlContent, tag string) int {
	count := 0
	openTag := "<" + tag
	close := ">"
	// Aceita "<tag>" ou "<tag ...>" ou "<tag ... />"
	i := 0
	for i < len(xmlContent) {
		idx := strings.Index(xmlContent[i:], openTag)
		if idx == -1 {
			break
		}
		realIdx := i + idx
		// Confirma que é a tag (não "tagX" etc)
		afterTag := realIdx + len(openTag)
		if afterTag < len(xmlContent) {
			ch := xmlContent[afterTag]
			if ch == '>' || ch == ' ' || ch == '/' || ch == '\t' || ch == '\n' {
				count++
			}
		}
		// Avança pelo nome da tag (até próximo >)
		next := strings.Index(xmlContent[realIdx:], close)
		if next == -1 {
			break
		}
		i = realIdx + next + 1
	}
	return count
}

// parseNum converte string numérica BR/EN para float64.
// Tolera "1.234,56" (BR) e "1,234.56" (EN).
func parseNum(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	hasDot := strings.Contains(s, ".")
	hasComma := strings.Contains(s, ",")
	switch {
	case hasDot && hasComma:
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	case hasComma && !hasDot:
		s = strings.ReplaceAll(s, ",", ".")
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
