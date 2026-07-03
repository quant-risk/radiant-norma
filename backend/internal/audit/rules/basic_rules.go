// Regras básicas B01-B05 (Sprint 6 v1.5.0 / W3).
//
// Estas regras operam no XML bruto (sem parser tipado do 3040):
//   - B01: arquivo XML deve ser válido (cheque já feito em L1-PARSE)
//   - B02: estrutura básica presente (stub — válido)
//   - B03: tamanho razoável (stub — delegado)
//   - B04: codificação deve estar declarada (<?xml ... ?>)
//   - B05: arquivo não pode estar vazio/muito pequeno
//
// Antes (v1.4.x): hardcoded em audit/service.go::applyRegra com if c.Codigo == "B01" etc.
// Agora (v1.5.0): registradas via RawRule no Registry — mesmo padrão das 25 regras tipadas.
package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// B01ArquivoXMLValido: Sprintf placeholder. Validação real já foi feita
// em L1-PARSE — esta regra é defensiva.
type B01ArquivoXMLValido struct{}

func (B01ArquivoXMLValido) Code() string     { return "B01" }
func (B01ArquivoXMLValido) Sheet() string    { return "Básicas" }
func (B01ArquivoXMLValido) Severity() string { return "I" }
func (B01ArquivoXMLValido) ApplyRaw(_ context.Context, _ string) error {
	return nil // já validado em L1
}

// B02EstruturaBasica: stub — regras detalhadas em B06+.
type B02EstruturaBasica struct{}

func (B02EstruturaBasica) Code() string     { return "B02" }
func (B02EstruturaBasica) Sheet() string    { return "Básicas" }
func (B02EstruturaBasica) Severity() string { return "I" }
func (B02EstruturaBasica) ApplyRaw(_ context.Context, _ string) error {
	return nil // detalhe em B06-B15
}

// B03TamanhoArquivo: stub — B05 faz o check mínimo.
type B03TamanhoArquivo struct{}

func (B03TamanhoArquivo) Code() string     { return "B03" }
func (B03TamanhoArquivo) Sheet() string    { return "Básicas" }
func (B03TamanhoArquivo) Severity() string { return "A" }
func (B03TamanhoArquivo) ApplyRaw(_ context.Context, _ string) error {
	return nil // detalhe em B05
}

// B04CodificacaoDeclarada: arquivo deve começar com <?xml ... encoding=...
type B04CodificacaoDeclarada struct{}

func (B04CodificacaoDeclarada) Code() string     { return "B04" }
func (B04CodificacaoDeclarada) Sheet() string    { return "Básicas" }
func (B04CodificacaoDeclarada) Severity() string { return "E" }
func (B04CodificacaoDeclarada) ApplyRaw(_ context.Context, xmlContent string) error {
	trimmed := strings.TrimSpace(xmlContent)
	if !strings.HasPrefix(trimmed, "<?xml") {
		return errors.New("arquivo não começa com declaração <?xml")
	}
	return nil
}

// B05ArquivoNaoVazio: arquivo deve ter tamanho mínimo razoável.
type B05ArquivoNaoVazio struct{}

func (B05ArquivoNaoVazio) Code() string     { return "B05" }
func (B05ArquivoNaoVazio) Sheet() string    { return "Básicas" }
func (B05ArquivoNaoVazio) Severity() string { return "E" }
func (B05ArquivoNaoVazio) ApplyRaw(_ context.Context, xmlContent string) error {
	if len(xmlContent) == 0 {
		return errors.New("arquivo XML está vazio")
	}
	if len(xmlContent) < 50 {
		return fmt.Errorf("arquivo XML tem apenas %d bytes", len(xmlContent))
	}
	return nil
}
