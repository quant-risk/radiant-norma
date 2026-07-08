// Alert generation for L4 comparisons.
//
// Thresholds:
//
//	New failure (E severity): → "L4-NEW-FAILURE" alert
//	Fixed rule (any): → "L4-FIXED" alert
//	Field variation > 20% (or crossed zero): → "L4-VARIATION" alert
//
// Thresholds are configurable via AlertConfig.
package l4

import (
	"fmt"
	"math"
	"strings"
)

// DefaultThresholds são os thresholds padrão de alerta.
var DefaultThresholds = Thresholds{
	VariationPct:  20.0, // 20% de variação mínima para alertar
	NewFailureSev: "E",  // só alertas de nova falha com severity E (bloqueante)
}

// Thresholds configura os limites de sensibilidade do alerting.
type Thresholds struct {
	VariationPct  float64 // variação mínima em % para alertar (default 20%)
	NewFailureSev string  // severity mínima para new failure alert (default "E")
}

// GenerateAlerts gera alertas processáveis a partir de uma Comparison.
func GenerateAlerts(c *Comparison, opts ...Thresholds) []Alert {
	var t Thresholds
	if len(opts) > 0 {
		t = opts[0]
	} else {
		t = DefaultThresholds
	}

	var alerts []Alert

	// L4-NEW-FAILURE: regras que agora falham
	for _, f := range c.NewFailures {
		// Só alerta para severity >= threshold
		if severityScore(f.Severity) >= severityScore(t.NewFailureSev) {
			alerts = append(alerts, Alert{
				Type:     "L4-NEW-FAILURE",
				Code:     f.Code,
				Severity: f.Severity,
				Message:  fmt.Sprintf("regra %s começou a falhar (gravidade %s)", f.Code, f.Severity),
			})
		}
	}

	// L4-FIXED: regras que antes falhavam e agora passam
	for _, f := range c.FixedRules {
		alerts = append(alerts, Alert{
			Type:     "L4-FIXED",
			Code:     f.Code,
			Severity: f.Severity,
			Message:  fmt.Sprintf("regra %s que falhava anteriormente está OK agora", f.Code),
		})
	}

	// L4-VARIATION: mudanças significativas em campos agregados
	for _, ch := range c.ChangedFields {
		if math.Abs(ch.DeltaPct) >= t.VariationPct || ch.Previous == 0 || ch.Current == 0 {
			// Cruzou zero: alerta de cualquier variação percentual
			direction := "aumentou"
			if ch.Current < ch.Previous {
				direction = "diminuiu"
			}
			var msg string
			if ch.Previous == 0 {
				msg = fmt.Sprintf("campo %s.%s novo valor: %.2f (antes zero)",
					ch.CadocCode, ch.Field, ch.Current)
			} else if ch.Current == 0 {
				msg = fmt.Sprintf("campo %s.%s zerou (antes %.2f)",
					ch.CadocCode, ch.Field, ch.Previous)
			} else {
				msg = fmt.Sprintf("campo %s.%s %s %.1f%% (de %.2f para %.2f)",
					ch.CadocCode, ch.Field, direction, math.Abs(ch.DeltaPct), ch.Previous, ch.Current)
			}
			alerts = append(alerts, Alert{
				Type:     "L4-VARIATION",
				Code:     ch.Field,
				Severity: inferSeverityFromChange(ch),
				Message:  msg,
			})
		}
	}

	return alerts
}

// severityScore retorna um número para comparar severities: E=3, A=2, I=1.
func severityScore(s string) int {
	switch s {
	case "E":
		return 3
	case "A":
		return 2
	case "I":
		return 1
	default:
		return 0
	}
}

// inferSeverityFromChange infere a severity de um alerta de variação.
func inferSeverityFromChange(ch FieldChange) string {
	// Variação > 50% em campos de risco (HQLA, ASF, RSF, RWACAM) → A
	highRiskFields := map[string]bool{
		"HQLA": true, "ASFTotal": true, "RSFTotal": true,
		"LCRRatio": true, "NSFRRatio": true,
		"RWACAM": true,
	}
	if highRiskFields[ch.Field] && math.Abs(ch.DeltaPct) > 50 {
		return "A"
	}
	return "I"
}

// AlertSummary retorna um sumário curto para UI.
func AlertSummary(alerts []Alert) string {
	if len(alerts) == 0 {
		return "sem alertas"
	}
	var parts []string
	newFailures := countByType(alerts, "L4-NEW-FAILURE")
	fixed := countByType(alerts, "L4-FIXED")
	variations := countByType(alerts, "L4-VARIATION")
	if newFailures > 0 {
		parts = append(parts, fmt.Sprintf("%d novo", newFailures))
	}
	if fixed > 0 {
		parts = append(parts, fmt.Sprintf("%d corrigiu", fixed))
	}
	if variations > 0 {
		parts = append(parts, fmt.Sprintf("%d variação", variations))
	}
	return strings.Join(parts, ", ")
}

func countByType(alerts []Alert, typ string) int {
	n := 0
	for _, a := range alerts {
		if a.Type == typ {
			n++
		}
	}
	return n
}
