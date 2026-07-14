// Package l4 implementa detecção de mudança vs envios anteriores (Layer 4).
//
// Filosofia: Radiant Norma valida o ecossistema inteiro de CADOCs. O L4
// compara o envio corrente com o anterior (mesma IF + CADOC + data-base
// ou remessa anterior) para detectar mudanças significativas.
//
// Sprint 55: implementação inicial.
//
// Arquitetura:
//
//	l4.Engine — ponto de entrada, recebe currentEnvioID
//	l4.Comparison — resultado da comparação (new_failures, fixed_rules, changed_fields)
//	l4.FieldExtractor — extrai campos agregados de qualquer CADOC via parsers existentes
package l4

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
)

// Engine compara envios atuais vs anteriores para detecção de mudança.
type Engine struct {
	db *sql.DB
}

// NewEngine cria um L4 Engine com a conexão de DB fornecida.
func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// SubmissionSnapshot representa o estado de um envio em um ponto no tempo.
type SubmissionSnapshot struct {
	EnvioID     string
	IfID        string
	CadocCode   string
	DataBase    string
	Remessa     int
	XMLContent  string
	Hash        string
	RulesPassed int
	RulesFailed int
	FailedRules []FailedRule
	CreatedAt   string
}

// FailedRule é uma regra que falhou neste envio.
type FailedRule struct {
	Code     string
	Severity string // E, A, I
	Message  string
}

// Comparison é o resultado da comparação entre dois envios.
type Comparison struct {
	Current  *SubmissionSnapshot
	Previous *SubmissionSnapshot

	// Alterações detectadas
	NewFailures   []FailedRule  // regras que agora falham (não falhavam antes)
	FixedRules    []FailedRule  // regras que antes falhavam e agora passam
	ChangedFields []FieldChange // mudanças em campos agregados (COSIF, etc.)
	Alerts        []Alert       // alertas processáveis
}

// FieldChange representa mudança em campo numérico.
type FieldChange struct {
	CadocCode string // qual CADOC
	Field     string // ex: "800.01", "Patrimonio", "RWACAM"
	Previous  float64
	Current   float64
	DeltaPct  float64 // variação percentual
}

// Alert é um alerta gerado pela análise de mudança.
type Alert struct {
	Type     string // "L4-NEW-FAILURE", "L4-FIXED", "L4-VARIATION"
	Code     string // código da regra ou campo
	Severity string
	Message  string
}

// Compare obtém o envio anterior (mesma IF + CADOC) e compara com o atual.
func (e *Engine) Compare(ctx context.Context, currentEnvioID string) (*Comparison, error) {
	// 1. Busca envio atual
	current, err := e.getEnvio(ctx, currentEnvioID)
	if err != nil {
		return nil, fmt.Errorf("envio atual: %w", err)
	}

	// 2. Busca envio anterior (mesma IF + CADOC + status accepted, criado antes)
	previous, err := e.getPreviousEnvio(ctx, current.IfID, current.CadocCode, currentEnvioID)
	if err != nil {
		return nil, fmt.Errorf("envio anterior: %w", err)
	}

	comp := &Comparison{
		Current:  current,
		Previous: previous,
	}

	// Se não há envio anterior, a comparação é nula (primeiro envio)
	if previous == nil {
		return comp, nil
	}

	// 3. Carrega falhas dos envios (rule_failures table)
	currentFailed, err := e.getFailedRules(ctx, currentEnvioID)
	if err != nil {
		return nil, fmt.Errorf("failed rules current: %w", err)
	}
	current.FailedRules = currentFailed

	prevFailed, err := e.getFailedRules(ctx, previous.EnvioID)
	if err != nil {
		return nil, fmt.Errorf("failed rules previous: %w", err)
	}
	previous.FailedRules = prevFailed

	// 4. Detecta new failures e fixed rules
	prevFailedSet := make(map[string]FailedRule)
	for _, f := range prevFailed {
		prevFailedSet[f.Code] = f
	}
	currFailedSet := make(map[string]FailedRule)
	for _, f := range currentFailed {
		currFailedSet[f.Code] = f
	}

	// New failures: no current but not in previous
	for code, rule := range currFailedSet {
		if _, existed := prevFailedSet[code]; !existed {
			comp.NewFailures = append(comp.NewFailures, rule)
		}
	}

	// Fixed rules: in previous but not in current
	for code, rule := range prevFailedSet {
		if _, exists := currFailedSet[code]; !exists {
			comp.FixedRules = append(comp.FixedRules, rule)
		}
	}

	// 5. Extrai e compara campos agregados (COSIF, etc.)
	fieldChanges, err := e.extractFieldChanges(ctx, previous, current)
	if err != nil {
		return nil, fmt.Errorf("field changes: %w", err)
	}
	comp.ChangedFields = fieldChanges

	// 6. Gera alertas
	comp.Alerts = GenerateAlerts(comp)

	return comp, nil
}

// getEnvio busca um envio pelo ID.
func (e *Engine) getEnvio(ctx context.Context, envioID string) (*SubmissionSnapshot, error) {
	query := `
		SELECT id, if_id, cadoc_code, data_base, remessa,
		       xml_hash, rules_passed, rules_failed, created_at
		FROM envios
		WHERE id = $1`

	row := e.db.QueryRowContext(ctx, query, envioID)
	var s SubmissionSnapshot
	err := row.Scan(&s.EnvioID, &s.IfID, &s.CadocCode, &s.DataBase,
		&s.Remessa, &s.Hash, &s.RulesPassed, &s.RulesFailed, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("envio %s não encontrado", envioID)
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// getPreviousEnvio busca o último envio aceito da mesma IF + CADOC, anterior ao atual.
func (e *Engine) getPreviousEnvio(ctx context.Context, ifID, cadocCode, currentEnvioID string) (*SubmissionSnapshot, error) {
	query := `
		SELECT e.id, e.if_id, e.cadoc_code, e.data_base, e.remessa,
		       e.xml_hash, e.rules_passed, e.rules_failed, e.created_at
		FROM envios e
		JOIN (SELECT created_at FROM envios WHERE id = $1) AS curr
		  ON e.created_at < curr.created_at
		WHERE e.if_id = $2
		  AND e.cadoc_code = $3
		  AND e.status = 'accepted'
		ORDER BY e.created_at DESC
		LIMIT 1`

	row := e.db.QueryRowContext(ctx, query, currentEnvioID, ifID, cadocCode)
	var s SubmissionSnapshot
	err := row.Scan(&s.EnvioID, &s.IfID, &s.CadocCode, &s.DataBase,
		&s.Remessa, &s.Hash, &s.RulesPassed, &s.RulesFailed, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // sem envio anterior — OK (primeiro envio)
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// getFailedRules busca todas as regras que falharam para um envio.
func (e *Engine) getFailedRules(ctx context.Context, envioID string) ([]FailedRule, error) {
	query := `
		SELECT rf.rule_code, rf.rule_severity, e.message
		FROM rule_failures rf
		LEFT JOIN envios env ON rf.envio_id = env.id
		LEFT JOIN LATERAL (
			SELECT message FROM rule_failures
			WHERE envio_id = $1 AND rule_code = rf.rule_code
			LIMIT 1
		) e ON true
		WHERE rf.envio_id = $1`

	rows, err := e.db.QueryContext(ctx, query, envioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var failures []FailedRule
	seen := make(map[string]bool)
	for rows.Next() {
		var f FailedRule
		if err := rows.Scan(&f.Code, &f.Severity, &f.Message); err != nil {
			return nil, err
		}
		if !seen[f.Code] {
			failures = append(failures, f)
			seen[f.Code] = true
		}
	}
	return failures, rows.Err()
}

// extractFieldChanges extrai e compara campos agregados entre dois envios.
func (e *Engine) extractFieldChanges(ctx context.Context, prev, curr *SubmissionSnapshot) ([]FieldChange, error) {
	switch curr.CadocCode {
	case "2061":
		return e.extractDLOChanges(prev, curr)
	case "2062":
		return e.extractDLIChanges(prev, curr)
	case "2160":
		return e.extractDRLChanges(prev, curr)
	case "2170":
		return e.extractDLPChanges(prev, curr)
	case "3040":
		return e.extract3040Changes(prev, curr)
	case "3050":
		return e.extract3050Changes(prev, curr)
	case "3044":
		return e.extract3044Changes(prev, curr)
	case "2060":
		return e.extractDRMChanges(prev, curr)
	case "4111":
		return e.extract4111Changes(prev, curr)
	case "2030":
		return e.extractDRSACChanges(prev, curr)
	case "2070":
		return e.extract2070Changes(prev, curr)
	default:
		return nil, nil // CADOC sem diff implementado
	}
}

// HasChanges retorna true se a comparação detectou alterações significativas.
func (c *Comparison) HasChanges() bool {
	return len(c.NewFailures) > 0 || len(c.FixedRules) > 0 || len(c.ChangedFields) > 0
}

// Summary retorna um texto resumido da comparação.
func (c *Comparison) Summary() string {
	var parts []string
	if len(c.NewFailures) > 0 {
		parts = append(parts, fmt.Sprintf("%d new failures", len(c.NewFailures)))
	}
	if len(c.FixedRules) > 0 {
		parts = append(parts, fmt.Sprintf("%d fixed", len(c.FixedRules)))
	}
	if len(c.ChangedFields) > 0 {
		parts = append(parts, fmt.Sprintf("%d field changes", len(c.ChangedFields)))
	}
	if len(parts) == 0 {
		return "no significant changes"
	}
	return strings.Join(parts, ", ")
}

// deltaPercent calcula a variação percentual, tolerando divisão por zero.
func deltaPercent(prev, curr float64) float64 {
	if prev == 0 {
		if curr == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return ((curr - prev) / math.Abs(prev)) * 100
}
