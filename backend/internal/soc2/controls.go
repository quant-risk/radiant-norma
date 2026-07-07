// Package soc2 — SOC 2 Type I compliance tooling (Sprint 56 v3.34.38).
//
// Ferramentas de suporte à auditoria SOC 2 Type I.
//
// SOC 2 Type I: avalia o DESIGN dos controles de segurança em um ponto
// no tempo. Não requer demonstração de eficácia operacional ao longo
// do tempo (isso é Type II).
//
// Este package fornece:
//   - Registry de controles implementados (mapeamento CC/CR/Critério)
//   - Coleta de evidências (evidence) por controle
//   - Readiness report (oque está implementado vs. gaps)
//
// NÃO é uma certificação. A auditoria real deve ser conduzida por
// um CPA/AICPA-certified auditor.
//
// Referências:
//   - AICPA SOC 2 2017 Trust Services Criteria (TSP Section 100)
//   - TSP Section 100 (2022 update)
package soc2

import (
	"context"
	"database/sql"
	"time"
)

// TrustServiceCriterion representa um critério do Trust Services.
type TrustServiceCriterion string

const (
	CC1 TrustServiceCriterion = "CC1" // Control Environment
	CC2 TrustServiceCriterion = "CC2" // Communication and Information
	CC3 TrustServiceCriterion = "CC3" // Risk Assessment
	CC4 TrustServiceCriterion = "CC4" // Monitoring Activities
	CC5 TrustServiceCriterion = "CC5" // Control Activities
	CC6 TrustServiceCriterion = "CC6" // Logical and Physical Access Controls
	CC7 TrustServiceCriterion = "CC7" // System Operations
	CC8 TrustServiceCriterion = "CC8" // Change Management
	CC9 TrustServiceCriterion = "CC9" // Risk Mitigation
	A1  TrustServiceCriterion = "A1"  // Availability
	PI1 TrustServiceCriterion = "PI1" // Processing Integrity
	C1  TrustServiceCriterion = "C1"  // Confidentiality
	P1  TrustServiceCriterion = "P1"  // Privacy
)

// ControlStatus indica o status de implementação de um controle.
type ControlStatus string

const (
	ControlStatusImplemented    ControlStatus = "implemented"
	ControlStatusPartially      ControlStatus = "partially_implemented"
	ControlStatusNotImplemented ControlStatus = "not_implemented"
	ControlStatusNotApplicable  ControlStatus = "not_applicable"
)

// Control representa um controle de segurança SOC 2.
type Control struct {
	ID          string                  `json:"id"`           // ex: "CC1.1"
	Name        string                  `json:"name"`         // ex: "COSO Principle 1"
	Description string                  `json:"description"`  // o que o controle faz
	Criteria    []TrustServiceCriterion `json:"criteria"`     // Trust Services Criteria
	Status      ControlStatus           `json:"status"`       // implementação atual
	EvidenceIDs []string                `json:"evidence_ids"` // IDs dos evidências coletados
	LastUpdated time.Time               `json:"last_updated"`
}

// Evidence representa uma evidência coletada para um controle.
type Evidence struct {
	ID          string    `json:"id"`          // UUID
	ControlID   string    `json:"control_id"`  // Control.ID que este evidence suporta
	Type        string    `json:"type"`        // "log", "config", "policy", "screenshot"
	Description string    `json:"description"` // o que este evidence demonstra
	CollectedAt time.Time `json:"collected_at"`
	CollectedBy string    `json:"collected_by"`   // sistema ou auditor
	Data        string    `json:"data,omitempty"` // caminho ou conteúdo (opaque)
}

// ReadinessReport é o relatório de readiness SOC 2.
type ReadinessReport struct {
	GeneratedAt         time.Time          `json:"generated_at"`
	AuditedEntity       string             `json:"audited_entity"` // ex: "Radiant Norma"
	AuditPeriod         string             `json:"audit_period"`   // ex: "2026-07-01"
	Controls            []Control          `json:"controls"`
	Total               int                `json:"total"`
	Implemented         int                `json:"implemented"`
	Partially           int                `json:"partially_implemented"`
	NotImplemented      int                `json:"not_implemented"`
	NotApplicable       int                `json:"not_applicable"`
	CoverageByCriterion map[string]float64 `json:"coverage_by_criterion"` // % implementado por critério
	Gaps                []ControlGap       `json:"gaps,omitempty"`
}

// ControlGap representa uma lacuna em controles de segurança.
type ControlGap struct {
	ControlID      string `json:"control_id"`
	Criterion      string `json:"criterion"`
	Description    string `json:"description"`
	Severity       string `json:"severity"` // "high", "medium", "low"
	Recommendation string `json:"recommendation"`
}

// Service de SOC 2 tooling.
type Service struct {
	db *sql.DB
}

// NewService cria o service SOC 2.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// DefaultControls retorna a lista de controles padrão do Radiant Norma.
// Cada controle é mapeado para os Trust Services Criteria aplicáveis.
//
//nolint:revive,stylecheck
func DefaultControls() []Control {
	return []Control{
		// CC1 — Control Environment
		{ID: "CC1.1", Name: "Integridade e Valores Éticos",
			Description: "A organização demonstra compromisso com integridade e valores éticos.",
			Criteria:    []TrustServiceCriterion{CC1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC1.2", Name: "Supervisão Responsável",
			Description: "O board e管理层 supervisionam o controle interno.",
			Criteria:    []TrustServiceCriterion{CC1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC1.3", Name: "Estrutura Organizacional",
			Description: "Organização define estrutura de autoridade e responsabilidade.",
			Criteria:    []TrustServiceCriterion{CC1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC1.4", Name: "Recrutamento e Retenção de Competentes",
			Description: "Políticas de recrutamento garantem competência necessária.",
			Criteria:    []TrustServiceCriterion{CC1}, Status: ControlStatusPartially,
			LastUpdated: time.Now()},

		// CC2 — Communication and Information
		{ID: "CC2.1", Name: "Informação Relevante Interna",
			Description: "Informação relevante é comunicada internamente.",
			Criteria:    []TrustServiceCriterion{CC2}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC2.2", Name: "Informação Relevante Externa",
			Description: "A organização comunica informações externas relevantes.",
			Criteria:    []TrustServiceCriterion{CC2}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC2.3", Name: "Comunicação de Informações Internas",
			Description: "Canais de comunicação internos permitem aliran de informações.",
			Criteria:    []TrustServiceCriterion{CC2}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC3 — Risk Assessment
		{ID: "CC3.1", Name: "Especificação de Objetivos",
			Description: "Objectivos de segurança e compliance são claramente definidos.",
			Criteria:    []TrustServiceCriterion{CC3}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC3.2", Name: "Identificação e Análise de Riscos",
			Description: "Riscos são identificados e analisados regularmente.",
			Criteria:    []TrustServiceCriterion{CC3}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC4 — Monitoring Activities
		{ID: "CC4.1", Name: "Avaliação e Comunicação de Deficiências",
			Description: "Deficiências são avaliadas e comunicadas em tempo hábil.",
			Criteria:    []TrustServiceCriterion{CC4}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC5 — Control Activities
		{ID: "CC5.1", Name: "Segregação de Funções",
			Description: "Segregação adequada de funções (segredo, approve, execute).",
			Criteria:    []TrustServiceCriterion{CC5}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC5.2", Name: "Controles de Acesso",
			Description: "Controles de acesso lógico e físico implementados.",
			Criteria:    []TrustServiceCriterion{CC5, CC6}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC6 — Logical and Physical Access
		{ID: "CC6.1", Name: "Controle de Acesso Lógico",
			Description: "Acesso lógico baseado em princípio de menor privilégio.",
			Criteria:    []TrustServiceCriterion{CC6}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC6.2", Name: "Autenticação Multifator",
			Description: "MFA enforced para todos os acessos privileged.",
			Criteria:    []TrustServiceCriterion{CC6}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC6.3", Name: "Gestão de Identidades",
			Description: "Processo de gestão de identidade (on/offboarding) documentado.",
			Criteria:    []TrustServiceCriterion{CC6}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC6.4", Name: "Proteção de Dados em Trânsito",
			Description: "TLS 1.2+ enforced em todas as comunicações.",
			Criteria:    []TrustServiceCriterion{CC6, C1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC6.5", Name: "Proteção de Dados em Repouso",
			Description: "Dados sensíveis criptografados at rest (AES-256).",
			Criteria:    []TrustServiceCriterion{CC6, C1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC6.6", Name: "Logs de Auditoria Imutáveis",
			Description: "Audit log com chain hash — impossível alterar entradas passadas.",
			Criteria:    []TrustServiceCriterion{CC6, CC4}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC7 — System Operations
		{ID: "CC7.1", Name: "Detecção de Eventos de Segurança",
			Description: "Monitoramento contínuo de eventos de segurança.",
			Criteria:    []TrustServiceCriterion{CC7}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC7.2", Name: "Plano de Resposta a Incidentes",
			Description: "IRP documentado e testado.",
			Criteria:    []TrustServiceCriterion{CC7}, Status: ControlStatusPartially,
			LastUpdated: time.Now()},

		// CC8 — Change Management
		{ID: "CC8.1", Name: "Gestão de Mudanças",
			Description: "Processo de mudança documentado com approval e rollback.",
			Criteria:    []TrustServiceCriterion{CC8}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
		{ID: "CC8.2", Name: "Controle de Versão",
			Description: "Código versionado em Git com branch protection.",
			Criteria:    []TrustServiceCriterion{CC8}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},

		// CC9 — Risk Mitigation
		{ID: "CC9.2", Name: "Avaliação de Fornecedores",
			Description: "Avaliação de fornecedores críticos (data centers, cloud).",
			Criteria:    []TrustServiceCriterion{CC9}, Status: ControlStatusPartially,
			LastUpdated: time.Now()},

		// A1 — Availability
		{ID: "A1.1", Name: "Acordo de Nível de Serviço (SLA)",
			Description: "SLA documentado com uptime commitments.",
			Criteria:    []TrustServiceCriterion{A1}, Status: ControlStatusNotApplicable,
			LastUpdated: time.Now()},
		{ID: "A1.2", Name: "Plano de Recuperação de Desastres",
			Description: "DRP documentado e testado anualmente.",
			Criteria:    []TrustServiceCriterion{A1}, Status: ControlStatusPartially,
			LastUpdated: time.Now()},

		// C1 — Confidentiality
		{ID: "C1.1", Name: "Classificação de Dados",
			Description: "Dados classificados e controles proporcionais aplicados.",
			Criteria:    []TrustServiceCriterion{C1}, Status: ControlStatusImplemented,
			LastUpdated: time.Now()},
	}
}

// GenerateReadinessReport gera relatório de readiness SOC 2.
func (s *Service) GenerateReadinessReport(ctx context.Context) (*ReadinessReport, error) {
	controls := DefaultControls()

	var total, impl, partial, notImpl, notAppl int
	criterionCount := make(map[string]int)
	criterionImpl := make(map[string]int)

	for _, c := range controls {
		total++
		switch c.Status {
		case ControlStatusImplemented:
			impl++
		case ControlStatusPartially:
			partial++
		case ControlStatusNotImplemented:
			notImpl++
		case ControlStatusNotApplicable:
			notAppl++
		}

		for _, crit := range c.Criteria {
			key := string(crit)
			criterionCount[key]++
			if c.Status == ControlStatusImplemented {
				criterionImpl[key]++
			}
		}
	}

	coverage := make(map[string]float64)
	for k, count := range criterionCount {
		if count > 0 {
			coverage[k] = float64(criterionImpl[k]) / float64(count) * 100
		}
	}

	// Identify gaps
	var gaps []ControlGap
	for _, c := range controls {
		if c.Status == ControlStatusNotImplemented {
			gaps = append(gaps, ControlGap{
				ControlID:      c.ID,
				Criterion:      string(c.Criteria[0]),
				Description:    c.Description,
				Severity:       "high",
				Recommendation: "Implementar controle " + c.ID + ": " + c.Name,
			})
		} else if c.Status == ControlStatusPartially {
			gaps = append(gaps, ControlGap{
				ControlID:      c.ID,
				Criterion:      string(c.Criteria[0]),
				Description:    c.Description,
				Severity:       "medium",
				Recommendation: "Completar implementação do controle " + c.ID,
			})
		}
	}

	return &ReadinessReport{
		GeneratedAt:         time.Now(),
		AuditedEntity:       "Radiant Norma — cadocs platform",
		AuditPeriod:         time.Now().Format("2006-01-02"),
		Controls:            controls,
		Total:               total,
		Implemented:         impl,
		Partially:           partial,
		NotImplemented:      notImpl,
		NotApplicable:       notAppl,
		CoverageByCriterion: coverage,
		Gaps:                gaps,
	}, nil
}
