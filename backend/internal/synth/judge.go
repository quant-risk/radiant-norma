package synth

import (
	"context"
	"encoding/json"
	"fmt"
)

// JudgeResult é o veredito do Judge sobre um caso.
type JudgeResult struct {
	Accept      bool    // true se caso deve ser aceito no dataset
	Realism     string  // high/medium/low
	Difficulty  string  // easy/medium/hard
	WeakScore   float64 // 0.0-1.0 (1.0 = weak succeed, 0.0 = weak falha muito)
	StrongScore float64 // 0.0-1.0 (1.0 = strong succeed)
	Gap         float64 // strong - weak (positivo = bom, caso separa os dois)
	Feedback    string  // feedback concreto para o Challenger
}

// Judge avalia a qualidade (realismo e dificuldade) de um caso gerado.
// Implementa o "Verifier/Judge" do Agentic Self-Instruct (Autodata paper).
//
// Critério de aceitação:
//   - Gap > 0 (strong succeed while weak fails)
//   - Realism >= medium
//   - Difficulty >= easy
//
// Se não aceito, feedback indica o que melhorar.
type Judge struct {
	llm LLMClient
}

// NewJudge cria um novo judge.
func NewJudge(llm LLMClient) *Judge {
	return &Judge{llm: llm}
}

// JudgeCase avalia um caso e retorna JudgeResult.
func (j *Judge) JudgeCase(ctx context.Context, c *Case) (*JudgeResult, error) {
	prompt := fmt.Sprintf(`Você é um judge de dados regulatórios sintéticos (Autodata Framework).
Avalie o seguinte caso CADOC %s gerado:

XML:
%s

Métricas do weak solver (audit.Service com regras estruturais):
- IsFailure: %v
- RuleCode: %s
- ErrorMsg: %s

Métricas do strong solver (verificação semântica):
- StrongScore: %.2f

Responda APENAS com JSON válido (sem markdown):
{"accept": true/false, "realism": "high|medium|low", "difficulty": "easy|medium|hard",
 "weak_score": 0.0-1.0, "strong_score": 0.0-1.0, "gap": -1.0-1.0,
 "feedback": "curto, concreto, para o gerador (em português)"}

Critérios de aceitação (Autodata):
- accept=true se: gap > 0.2 (strong >> weak) E realism != low E difficulty != easy
- accept=false se: weak succeeded (gap <= 0), ou realism=low, ou caso muito fácil
- realism: high = documento real plausível, medium = razoável, low = óbio ou fake
- difficulty: easy = viola regra óbvia, medium = violação sutil, hard = edge case realista
- feedback: diga O QUE CORRIGIR (ex: "aumentar range de vencimentos para ser mais realista")`, c.Cadoc, c.XML, c.IsFailure, c.RuleCode, c.ErrorMsg, c.StrongScore)

	resp, err := j.llm.Chat(ctx, []Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, fmt.Errorf("judge chat: %w", err)
	}

	var result struct {
		Accept      bool    `json:"accept"`
		Realism     string  `json:"realism"`
		Difficulty  string  `json:"difficulty"`
		WeakScore   float64 `json:"weak_score"`
		StrongScore float64 `json:"strong_score"`
		Gap         float64 `json:"gap"`
		Feedback    string  `json:"feedback"`
	}
	if err := json.Unmarshal([]byte(extractXML(resp)), &result); err != nil {
		// Fallback: aceita se weak falhou (gap > 0 implícito)
		return &JudgeResult{
			Accept:     c.IsFailure,
			Realism:    "medium",
			Difficulty: "medium",
			Feedback:   "judge parse error — aceitando por default",
		}, nil
	}

	return &JudgeResult{
		Accept:      result.Accept,
		Realism:     result.Realism,
		Difficulty:  result.Difficulty,
		WeakScore:   result.WeakScore,
		StrongScore: result.StrongScore,
		Gap:         result.Gap,
		Feedback:    result.Feedback,
	}, nil
}

// ruleContext retorna contexto de regras para o tipo de CADOC.
func ruleContext(cadoc string) string {
	switch cadoc {
	case "3040":
		return `Regras known neste documento:
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
