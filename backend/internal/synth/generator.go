// Package synth implementa o AuditForge — geração de envios CADOC sintéticos
// para teste massivo de regras (Sprint 34-T).
//
// Arquitetura Autodata-style (Challenger + Weak + Strong + Judge):
//   - Challenger: LLM gera XML variando features (valor, ClassOp, vencimento)
//   - Weak Solver: audit.Service determinístico valida o XML gerado
//   - Judge: LLM avalia realismo e dificuldade do caso gerado
//
// Output: dataset de edge cases que separam happy path vs regras-falhando.
package synth

import (
	"context"
	"fmt"

	"github.com/fortvna/radiant-norma/backend/internal/insights"
)

// CadocType é o tipo de documento CADOC a gerar.
type CadocType string

const (
	Cadoc3040 CadocType = "3040"
	Cadoc3050 CadocType = "3050"
	Cadoc4111 CadocType = "4111"
)

// LLMClient re-exporta a interface do package insights para convenience.
// Usa insights.Message como o tipo de mensagem padrão.
type LLMClient = insights.LLMClient

// Message re-exporta insights.Message.
type Message = insights.Message

// GeneratorConfig configura o gerador.
type GeneratorConfig struct {
	Cadoc       CadocType
	LLM         LLMClient
	Count       int     // número de casos a gerar
	FailureRate float64 // fração de casos que devem falhar regras (0.0-1.0)
}

// Case representa um caso sintético gerado.
type Case struct {
	ID          string `json:"id"`
	Cadoc       string `json:"cadoc"`
	XML         string `json:"xml"`
	RuleCode    string `json:"rule_code,omitempty"`  // regra que falha (se aplicável)
	ErrorMsg    string `json:"error_msg,omitempty"`  // mensagem de erro
	Realism     string `json:"realism,omitempty"`    // julgamento do judge: high/medium/low
	Difficulty  string `json:"difficulty,omitempty"` // easy/medium/hard
	IsFailure   bool   `json:"is_failure"`           // true se o caso viola uma regra
	GeneratedBy string `json:"generated_by"`         // challenger model
	JudgedBy    string `json:"judged_by,omitempty"`  // judge model
}

// Generator gera casos sintéticos usando LLM Challenger.
type Generator struct {
	cfg GeneratorConfig
}

// NewGenerator cria um novo gerador.
func NewGenerator(cfg GeneratorConfig) *Generator {
	return &Generator{cfg: cfg}
}

// promptForCadoc retorna o prompt de grounding para o tipo de CADOC.
func promptForCadoc(cadoc CadocType) string {
	switch cadoc {
	case Cadoc3040:
		return `Você é um gerador de dados sintéticos para validação de regras regulatórias bancárias brasileiras (BACEN).
Gere um XML válido para o documento CADOC 3040 (SCR — Risco de Crédito).
O XML deve seguir esta estrutura EXATA:
<Doc3040 dataBase="YYYY-MM" cnpj="XXXXXXXX" remessa="N" parte="N" tpArq="F" nomeResp="..." emailResp="..." telResp="..." totalCli="N">
  <Agreg natuOp="01" mod="1000" origemRec="1" vincME="N" classOp="A" faixaVlr="1" przProvm="N" localiz="SP" tpCli="1" desempOp="01" provConsttd="0" qtdOp="0" qtdCli="0">
    <Venc v110="0.00" v120="0.00" v150="0.00" v160="0.00" v165="0.00"/>
  </Agreg>
</Doc3040>

Regras que devem ser testadas (não precisam falhar todas, apenas variar os valores):
- B06: remessa >= 1
- B07: parte >= 1
- C01-C05: campos obrigatórios condicionais
- S01-S05: regras semânticas
- ClassOp: deve ser A, B, C, D, E, F, G ou H
- Vencimentos: valores monetários N15,2 (até 13 dígitos + 2 decimais)
- tpArq: deve ser F (full) ou S (substituição)

Gere APENAS o XML, sem markdown, sem explicação.`
	case Cadoc3050:
		return `Você é um gerador de dados sintéticos para validação de regras regulatórias bancárias brasileiras (BACEN).
Gera um XML válido para o documento CADOC 3050 (TXB — Taxa de Câmbio).
Estrutura: <Doc3050 dataBase="YYYY-MM" cnpj="XXXXXXXX" remessa="N" parte="N" tpArq="F">...</Doc3050>

Gere APENAS o XML, sem markdown, sem explicação.`
	default:
		return "Gere XML para o documento " + string(cadoc) + ". APENAS o XML, sem markdown."
	}
}

// Generate gera N casos sintéticos.
func (g *Generator) Generate(ctx context.Context) ([]Case, error) {
	var cases []Case
	for i := 0; i < g.cfg.Count; i++ {
		// Pergunta ao Challenger LLM para gerar um caso
		prompt := promptForCadoc(g.cfg.Cadoc)
		// Adiciona instrução de falha se necessário
		if g.cfg.FailureRate > 0 && i%int(1/g.cfg.FailureRate) == 0 {
			prompt += "\n\nIMPORTANT: Make this case violate AT LEAST ONE rule (e.g., invalid remessa, missing required field, out-of-range value)."
		}

		resp, err := g.cfg.LLM.Chat(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			return nil, fmt.Errorf("challenger chat: %w", err)
		}

		// Extrai XML da resposta (pode vir com markdown code blocks)
		xml := extractXML(resp)

		cases = append(cases, Case{
			ID:          fmt.Sprintf("synth-%s-%03d", g.cfg.Cadoc, i+1),
			Cadoc:       string(g.cfg.Cadoc),
			XML:         xml,
			GeneratedBy: g.cfg.LLM.Model(),
		})
	}
	return cases, nil
}

// extractXML removes markdown code fences from LLM output.
func extractXML(resp string) string {
	// Remove common markdown wrappers
	resp = trimPrefix(resp, "```xml")
	resp = trimPrefix(resp, "```XML")
	resp = trimPrefix(resp, "```")
	// Remove trailing markdown
	for len(resp) > 0 && (resp[len(resp)-1] == '`' || resp[len(resp)-1] == '\n' || resp[len(resp)-1] == ' ') {
		resp = resp[:len(resp)-1]
	}
	return resp
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
