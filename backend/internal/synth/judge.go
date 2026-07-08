package synth

import (
	"context"
	"encoding/json"
	"fmt"
)

// Judge avalia a qualidade (realismo e dificuldade) de um caso gerado.
type Judge struct {
	llm LLMClient
}

// NewJudge cria um novo judge.
func NewJudge(llm LLMClient) *Judge {
	return &Judge{llm: llm}
}

// JudgeCase avalia um caso e preenche os campos de julgamento.
func (j *Judge) JudgeCase(ctx context.Context, c *Case) error {
	prompt := fmt.Sprintf(`Você é um judge de dados regulatórios sintéticos.
Avalie o seguinte caso CADOC %s gerado:

XML:
%s

%s

Responda APENAS com JSON válido (sem markdown):
{"realism": "high|medium|low", "difficulty": "easy|medium|hard", "reasoning": "breve explicação"}

Regras de avaliação:
- realism: high = parece um documento real, medium = plausível, low = obviously fake
- difficulty: easy = viola regra óbvia (ex: remessa=0), medium = violação sutil, hard = edge case realista
- reasoning: 1-2 frases explicando o julgamento`, c.Cadoc, c.XML, ruleContext(c.Cadoc))

	resp, err := j.llm.Chat(ctx, []Message{{Role: "user", Content: prompt}})
	if err != nil {
		return fmt.Errorf("judge chat: %w", err)
	}

	// Parse JSON
	var result struct {
		Realism    string `json:"realism"`
		Difficulty string `json:"difficulty"`
		Reasoning  string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(extractXML(resp)), &result); err != nil {
		// Fallback: marca como medium/low em caso de parse error
		c.Realism = "medium"
		c.Difficulty = "medium"
		return nil
	}

	c.Realism = result.Realism
	c.Difficulty = result.Difficulty
	c.JudgedBy = j.llm.Model()
	return nil
}

// ruleContext retorna contexto de regras para o tipo de CADOC.
func ruleContext(cadoc string) string {
	switch cadoc {
	case "3040":
		return `Regras conhecidas neste documento:
- B06: remessa deve ser >= 1
- B07: parte deve ser >= 1
- B08: parte rejeitada (verifica histórico)
- C01-C05: campos obrigatórios condicionais
- S01-S05: regras semânticas
- ClassOp: A-H
- Vencimentos: N15,2 monetário`
	case "3050":
		return `Regras: TXB_v01 a TXB_v170 (170 regras de taxa de câmbio).`
	case "4111":
		return `Regras: 4111 D01-D30 (validação estrutural + domínio).`
	default:
		return ""
	}
}
