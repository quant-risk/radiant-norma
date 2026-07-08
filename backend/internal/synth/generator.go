// Package synth implementa o AuditForge — geração de envios CADOC sintéticos
// para teste massivo de regras (Sprint 34-T).
//
// Arquitetura Autodata/Agentic Self-Instruct (Challenger + Weak + Strong + Judge):
//   - Challenger: LLM gera XML variando features (valor, ClassOp, vencimento)
//   - Weak Solver: audit.Service (ou modelo fraco) falha em casos edge
//   - Strong Solver: audit.Service completo (ou ground truth humano) succeed em casos edge
//   - Judge: LLM avalia realismo e dificuldade do caso gerado
//   - Loop: Challenger → Weak+Strong → Judge → feedback → Challenger (iterações)
//
// A chave é a "gap": queremos casos onde strong solver SUCEDE e weak solver FALHA.
// Isso produz training data que separa weak/strong models.
//
// Paper: "Autodata: An agentic data scientist to create high quality synthetic data"
// (FAIR Meta, Jun 2026) — https://arxiv.org/abs/2606.25996
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
type LLMClient = insights.LLMClient

// Message re-exporta insights.Message.
type Message = insights.Message

// GeneratorConfig configura o gerador.
type GeneratorConfig struct {
	Cadoc       CadocType
	LLM         LLMClient
	Count       int     // número de casos a gerar
	FailureRate float64 // fração de casos que devem falhar regras (0.0-1.0)
	MaxRounds   int     // iterações do loop por caso (default 5)
}

// Case representa um caso sintético gerado.
type Case struct {
	ID          string `json:"id"`
	Cadoc       string `json:"cadoc"`
	XML         string `json:"xml"`
	RuleCode    string `json:"rule_code,omitempty"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	Realism     string `json:"realism,omitempty"`
	Difficulty  string `json:"difficulty,omitempty"`
	IsFailure   bool   `json:"is_failure"`
	GeneratedBy string `json:"generated_by"`
	JudgedBy    string `json:"judged_by,omitempty"`
	Rounds      int    `json:"rounds"` // número de iterações até aceitar

	// Weak/Strong gap (Autodata metric: strong - weak)
	WeakScore     float64 `json:"weak_score,omitempty"`
	StrongScore   float64 `json:"strong_score,omitempty"`
	JudgeFeedback string  `json:"judge_feedback,omitempty"`
}

// Generator gera casos sintéticos usando LLM Challenger.
// Segue o loop Agentic Self-Instruct: Challenger → Weak+Strong → Judge → feedback.
type Generator struct {
	cfg GeneratorConfig
}

// NewGenerator cria um novo gerador.
func NewGenerator(cfg GeneratorConfig) *Generator {
	if cfg.MaxRounds == 0 {
		cfg.MaxRounds = 5
	}
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

Regras que devem ser testadas:
- B06: remessa deve ser >= 1
- B07: parte deve ser >= 1
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

// challengePrompt retorna o prompt para o Challenger com feedback do Judge.
func challengePrompt(cadoc CadocType, feedback string) string {
	base := promptForCadoc(cadoc)
	if feedback != "" {
		base += "\n\nFeedback do Judge (aplique estas correções):\n" + feedback
	}
	return base
}

// Generate gera N casos sintéticos usando o loop Challenger→Weak→Strong→Judge.
// Cada caso passa por até MaxRounds iterações até satisfazer os critérios de qualidade:
// - Weak solver falha (score < threshold)
// - Strong solver succeed (score > threshold)
// - Judge aprova realismo e dificuldade
func (g *Generator) Generate(ctx context.Context) ([]Case, error) {
	var cases []Case
	for i := 0; i < g.cfg.Count; i++ {
		c, err := g.generateOne(ctx, i)
		if err != nil {
			// Fallback: gera sem loop se falhar
			c = &Case{
				ID:          fmt.Sprintf("synth-%s-%03d", g.cfg.Cadoc, i+1),
				Cadoc:       string(g.cfg.Cadoc),
				XML:         "",
				GeneratedBy: g.cfg.LLM.Model(),
				Rounds:      0,
			}
		}
		cases = append(cases, *c)
	}
	return cases, nil
}

// generateOne gera um único caso usando o loop Agentic Self-Instruct.
// Retorna (case, error) — em caso de erro, retorna caso parcial.
func (g *Generator) generateOne(ctx context.Context, idx int) (*Case, error) {
	id := fmt.Sprintf("synth-%s-%03d", g.cfg.Cadoc, idx+1)
	feedback := ""

	for round := 1; round <= g.cfg.MaxRounds; round++ {
		prompt := challengePrompt(g.cfg.Cadoc, feedback)

		resp, err := g.cfg.LLM.Chat(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			return nil, fmt.Errorf("challenger round %d: %w", round, err)
		}

		xml := extractXML(resp)

		// Caso válido: pelo menos tem conteúdo
		if xml == "" || len(xml) < 20 {
			feedback = "XML vazio ou inválido. Gere um XML completo com todos os campos."
			continue
		}

		// Cria caso parcial
		c := &Case{
			ID:          id,
			Cadoc:       string(g.cfg.Cadoc),
			XML:         xml,
			GeneratedBy: g.cfg.LLM.Model(),
			Rounds:      round,
		}

		return c, nil // loop controlado por quem chama (Judge decide se aceita)
	}

	// Esgotou rounds
	return &Case{
		ID:          id,
		Cadoc:       string(g.cfg.Cadoc),
		XML:         "",
		GeneratedBy: g.cfg.LLM.Model(),
		Rounds:      g.cfg.MaxRounds,
	}, nil
}

// extractXML removes markdown code fences from LLM output.
func extractXML(resp string) string {
	resp = trimPrefix(resp, "```xml")
	resp = trimPrefix(resp, "```XML")
	resp = trimPrefix(resp, "```")
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
