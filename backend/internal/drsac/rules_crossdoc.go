// Crossdoc rules DRSAC (2030) ↔ SCR (3040).
//
// Regras que validam consistência entre o Documento de Riscos Social,
// Ambiental e Climático (DRSAC) e os dados do SCR (3040).
//
// Regras XD-DR01 a XD-DR08.
//
//nolint:revive,stylecheck
package drsac

import (
	"context"
	"fmt"
	"strings"
)

// ============================================================
// Cross-Doc DRSAC ↔ SCR (3040)
// ============================================================

// XD-DR01 — IPOC de operação no DRSAC deve existir no SCR (3040).
//
// Justificativa: IPOC é o ID que vincula uma operação de crédito
// entre os dois documentos. Se o DRSAC reporta uma operação com IPOC X,
// o SCR deve conter o mesmo IPOC X.
type XD01IPOCExistsInSCR struct{}

func (XD01IPOCExistsInSCR) Code() string     { return "XD-DR01" }
func (XD01IPOCExistsInSCR) Severity() string { return "E" }
func (XD01IPOCExistsInSCR) Message() string {
	return "IPOC de operação no DRSAC deve existir no SCR (3040)"
}

// XD-DR02 — Saldo reportado no DRSAC deve ser consistente com SCR.
//
// Para operações com mesmo IPOC, o saldo informado no DRSAC (tag <saldo>)
// deve ser próximo (±10%) ao saldo reportado no SCR.
// Tolerância de 10% cobre diferenças de汇率 e timing de corte.
type XD02SaldoConsistenteComSCR struct{}

func (XD02SaldoConsistenteComSCR) Code() string     { return "XD-DR02" }
func (XD02SaldoConsistenteComSCR) Severity() string { return "A" }
func (XD02SaldoConsistenteComSCR) Message() string {
	return "Saldo DRSAC diverge mais de 10% do saldo SCR para mesmo IPOC"
}

// XD-DR03 — CNPJ do cliente no DRSAC deve existir no SCR.
//
// O identificador do cliente (CNPJ ou CPF) no DRSAC deve ter um registro
// correspondente no SCR (3040) para a mesma data-base.
type XD03ClienteExisteNoSCR struct{}

func (XD03ClienteExisteNoSCR) Code() string     { return "XD-DR03" }
func (XD03ClienteExisteNoSCR) Severity() string { return "E" }
func (XD03ClienteExisteNoSCR) Message() string {
	return "Cliente do DRSAC não encontrado no SCR para a mesma data-base"
}

// XD-DR04 — Setor CNAE no DRSAC deve ser consistente com SCR.
//
// Para operações de crédito, o setor CNAE reportado no DRSAC (nível setor
// ou cliente) deve ser compatível com a classificação CNAE no SCR.
// Diferenças de versão CNAE (Ex: 1.0 vs 2.0) são aceitáveis se ambas
// referenciam a mesma atividade econômica.
type XD04SetorCNAEConsistente struct{}

func (XD04SetorCNAEConsistente) Code() string     { return "XD-DR04" }
func (XD04SetorCNAEConsistente) Severity() string { return "A" }
func (XD04SetorCNAEConsistente) Message() string {
	return "CNAE do setor DRSAC diverge da classificação no SCR"
}

// XD-DR05 — Alto risco social (av=01) no DRSAC deve ter corresponds no SCR.
//
// Quando o DRSAC reporta risco social "Alto" (av=01) para uma operação,
// espera-se que o SCR tenha flag de inadimplência ouProvisionamento
// correspondente para a mesma operação.
type XD05RiscoSocialAltoNoSCR struct{}

func (XD05RiscoSocialAltoNoSCR) Code() string     { return "XD-DR05" }
func (XD05RiscoSocialAltoNoSCR) Severity() string { return "A" }
func (XD05RiscoSocialAltoNoSCR) Message() string {
	return "Operação com risco social alto (av=01) no DRSAC sem flag correspondente no SCR"
}

// XD-DR06 — Risco ambiental (av=01 ou 02) no DRSAC deve constar no SCR.
//
// Operações com risco ambiental "Alto" ou "Médio" no DRSAC devem aparecer
// como operações com garantia real ou collateral no SCR, indicando que
// a instituição conhece e monitora o risco ambiental.
type XD06RiscoAmbientalNoSCR struct{}

func (XD06RiscoAmbientalNoSCR) Code() string     { return "XD-DR06" }
func (XD06RiscoAmbientalNoSCR) Severity() string { return "A" }
func (XD06RiscoAmbientalNoSCR) Message() string {
	return "Operação com risco ambiental no DRSAC sem menção no SCR"
}

// XD-DR07 — Total de exposição em TVM no DRSAC deve ser consistente com SCR.
//
// Soma dos valores de TVM no DRSAC deve estar dentro de ±15% do total
// reportado na seção de TVM do SCR (3040) para a mesma data-base.
type XD07TotalTVMConsistente struct{}

func (XD07TotalTVMConsistente) Code() string     { return "XD-DR07" }
func (XD07TotalTVMConsistente) Severity() string { return "A" }
func (XD07TotalTVMConsistente) Message() string {
	return "Total de exposição TVM no DRSAC diverge mais de 15% do SCR"
}

// XD-DR08 — Contribuição positiva no DRSAC deve ter对应的绿色金融工具 no SCR.
//
// Operações com contribuição positiva (enquadramento 01, 02 ou 03) devem
// ter registro de instrumento financeiro verde (green bond, sustainability-linked,
// etc) no SCR para a mesma operação.
type XD08ContribPositivaGreenInstrument struct{}

func (XD08ContribPositivaGreenInstrument) Code() string     { return "XD-DR08" }
func (XD08ContribPositivaGreenInstrument) Severity() string { return "I" }
func (XD08ContribPositivaGreenInstrument) Message() string {
	return "Operação com contribuição positiva sem instrumento verde registrado no SCR"
}

// ============================================================
// CrossRefResult — resultado da validação cross-doc
// ============================================================

// CrossRefResult representa o resultado de uma validação cross-doc.
type CrossRefResult struct {
	Code     string
	Severity string // E, A, I
	Message  string
	IPOC     string // IPOC da operação relacionada (se aplicável)
}

// ExtractIPOCs retorna todos os IPOCs únicos de um documento DRSAC.
func ExtractIPOCs(doc *DocumentoDRSAC) []string {
	seen := make(map[string]bool)
	var ipocs []string
	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC != "" && !seen[op.IPOC] {
				seen[op.IPOC] = true
				ipocs = append(ipocs, op.IPOC)
			}
		}
	}
	return ipocs
}

// ExtractClienteIDs retorna todos os IDs de clientes únicos.
func ExtractClienteIDs(doc *DocumentoDRSAC) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, cl := range doc.Clientes {
		if cl.Ident != "" && !seen[cl.Ident] {
			seen[cl.Ident] = true
			ids = append(ids, cl.Ident)
		}
	}
	return ids
}

// ValidateCrossRefs executa todas as regras cross-doc DRSAC↔SCR.
// scrData: map de IPOC → {saldo, cnpj, hasFlag, isGreen}
// Retorna lista de CrossRefResult.
func ValidateCrossRefs(doc *DocumentoDRSAC, scrData map[string]SCRData) []CrossRefResult {
	var results []CrossRefResult

	for _, cl := range doc.Clientes {
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC == "" {
				continue
			}
			scr, ok := scrData[op.IPOC]
			if !ok {
				results = append(results, CrossRefResult{
					Code:     "XD-DR01",
					Severity: "E",
					Message:  "IPOC " + op.IPOC + " não encontrado no SCR",
					IPOC:     op.IPOC,
				})
				continue
			}

			// XD-DR02: Saldo consistente
			if op.Saldo != "" && scr.Saldo != "" {
				if !saldoConsistente(op.Saldo, scr.Saldo, 0.10) {
					results = append(results, CrossRefResult{
						Code:     "XD-DR02",
						Severity: "A",
						Message:  fmt.Sprintf("Saldo DRSAC=%s diverge >10%% do SCR=%s para IPOC=%s", op.Saldo, scr.Saldo, op.IPOC),
						IPOC:     op.IPOC,
					})
				}
			}

			// XD-DR03: Cliente existe no SCR
			if cl.Ident != "" && len(cl.Ident) >= 8 {
				if !scr.HasCliente {
					results = append(results, CrossRefResult{
						Code:     "XD-DR03",
						Severity: "E",
						Message:  fmt.Sprintf("Cliente %s não encontrado no SCR para IPOC %s", cl.Ident, op.IPOC),
						IPOC:     op.IPOC,
					})
				}
			}

			// XD-DR05: Risco social alto → flag no SCR
			if op.RiscSoc.Av == AvAlto && !scr.HasHighRiskFlag {
				results = append(results, CrossRefResult{
					Code:     "XD-DR05",
					Severity: "A",
					Message:  fmt.Sprintf("IPOC %s: risco social alto sem flag no SCR", op.IPOC),
					IPOC:     op.IPOC,
				})
			}

			// XD-DR06: Risco ambiental → menção no SCR
			if (op.RiscAmb.Av == AvAlto || op.RiscAmb.Av == AvMedio) && !scr.HasCollateral {
				results = append(results, CrossRefResult{
					Code:     "XD-DR06",
					Severity: "A",
					Message:  fmt.Sprintf("IPOC %s: risco ambiental sem menção no SCR", op.IPOC),
					IPOC:     op.IPOC,
				})
			}

			// XD-DR08: Contribuição positiva → instrumento verde
			if op.ContribPositiva != nil && !scr.IsGreenInstrument {
				results = append(results, CrossRefResult{
					Code:     "XD-DR08",
					Severity: "I",
					Message:  fmt.Sprintf("IPOC %s: contribuição positiva sem instrumento verde no SCR", op.IPOC),
					IPOC:     op.IPOC,
				})
			}
		}

		// XD-DR04: Setor CNAE
		for _, op := range cl.ExpAtivos.ExpOperCred {
			if op.IPOC != "" && cl.CNAE != "" {
				scr, ok := scrData[op.IPOC]
				if ok && scr.CNAE != "" && !cnaeConsistente(cl.CNAE, scr.CNAE) {
					results = append(results, CrossRefResult{
						Code:     "XD-DR04",
						Severity: "A",
						Message:  fmt.Sprintf("CNAE %s no DRSAC diverge do SCR %s para IPOC %s", cl.CNAE, scr.CNAE, op.IPOC),
						IPOC:     op.IPOC,
					})
				}
			}
		}
	}

	// XD-DR07: Total TVM
	drsacTotal := totalTVM(doc)
	if scrData["_TVM_TOTAL"] != (SCRData{}) && scrData["_TVM_TOTAL"].Saldo != "" {
		scrTotal := scrData["_TVM_TOTAL"].Saldo
		if !saldoConsistente(drsacTotal, scrTotal, 0.15) {
			results = append(results, CrossRefResult{
				Code:     "XD-DR07",
				Severity: "A",
				Message:  fmt.Sprintf("Total TVM DRSAC=%s diverge >15%% do SCR=%s", drsacTotal, scrTotal),
			})
		}
	}

	return results
}

// SCRData representa dados relevantes do SCR para cross-reference.
type SCRData struct {
	Saldo             string // saldo da operação
	CNAE              string // CNAE do cliente
	HasCliente        bool   // cliente existe no SCR
	HasHighRiskFlag   bool   // tem flag de alto risco
	HasCollateral     bool   // tem collateral/garantia
	IsGreenInstrument bool   // é instrumento verde
}

// saldoConsistente verifica se dois saldos são consistentes dentro da tolerância.
func saldoConsistente(drsacSaldo, scrSaldo string, tolerancia float64) bool {
	var ds, ss float64
	if _, err := fmt.Sscanf(drsacSaldo, "%f", &ds); err != nil {
		return false
	}
	if _, err := fmt.Sscanf(scrSaldo, "%f", &ss); err != nil {
		return false
	}
	if ds == 0 && ss == 0 {
		return true
	}
	diff := ds - ss
	if diff < 0 {
		diff = -diff
	}
	maxAllowed := ss * tolerancia
	return diff <= maxAllowed
}

// cnaeConsistente verifica se dois CNAEs são consistentes.
// CNAEs são consistentes se são idênticos ou se a versão difere mas o código base é o mesmo.
func cnaeConsistente(a, b string) bool {
	if a == b {
		return true
	}
	// Handles minor version differences: 1234567-1 vs 1234567-2
	if len(a) >= 7 && len(b) >= 7 && a[:7] == b[:7] {
		return true
	}
	return false
}

// totalTVM calcula a soma dos valores de TVM no DRSAC.
func totalTVM(doc *DocumentoDRSAC) string {
	var total float64
	for _, cl := range doc.Clientes {
		for _, tv := range cl.ExpAtivos.ExpTVM {
			var v float64
			if _, err := fmt.Sscanf(tv.Valor, "%f", &v); err == nil {
				total += v
			}
		}
	}
	// Format as N13,2
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", total)
}

// Rule interface para cross-doc rules (para compatibilidade com ValidateRules pattern).
type CrossDocRule interface {
	Code() string
	Severity() string
	Message() string
	Apply(context.Context, *DocumentoDRSAC, map[string]SCRData) []CrossRefResult
}

// nolint
func applyCrossDocRules(ctx context.Context, doc *DocumentoDRSAC, scrData map[string]SCRData) []CrossRefResult {
	var results []CrossRefResult

	// All cross-doc rules are applied via ValidateCrossRefs
	// This function exists for future extensibility with individual rule objects
	results = append(results, ValidateCrossRefs(doc, scrData)...)

	// Remove duplicate results based on Code + IPOC
	seen := make(map[string]bool)
	var unique []CrossRefResult
	for _, r := range results {
		key := r.Code + ":" + r.IPOC
		if !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// FilterCriticasPorSeverity filtra críticas por severidade.
func FilterCriticasPorSeverity(criticas []Critica, severities []string) []Critica {
	if len(severities) == 0 {
		return criticas
	}
	sevMap := make(map[string]bool)
	for _, s := range severities {
		sevMap[s] = true
	}
	var filtered []Critica
	for _, c := range criticas {
		if sevMap[c.Severity] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// FormatResultsSummary formata um Result como string legível.
func FormatResultsSummary(result *Result) string {
	if result == nil {
		return "nil result"
	}
	if result.Valid {
		return fmt.Sprintf("válido (%d críticas)", len(result.Criticas))
	}
	var parts []string
	for _, c := range result.Criticas {
		parts = append(parts, fmt.Sprintf("%s[%s]: %s", c.Code, c.Severity, c.Message))
	}
	return strings.Join(parts, "; ")
}
