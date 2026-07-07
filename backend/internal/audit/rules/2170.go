// Regras Sprint 41 — AuditDLP 2170 (NSFR — Net Stable Funding Ratio).
//
// 8 regras que validam NSFR Ratio (ASF / RSF >= 100%).
// Filosofia V67/V68/V69/V70/V71: implementar lógica real, não stubs disfarçados.
package rules

import (
	"context"
	"fmt"
)

// NSFR01 — NSFR Ratio >= 100% (mínimo regulatório BACEN Res. 4.542).
//
// IMPLEMENTAÇÃO REAL — verifica NSFR >= 100%.
type NSFR01 struct{}

func (NSFR01) Code() string     { return "NSFR01" }
func (NSFR01) Sheet() string    { return "NSFR" }
func (NSFR01) Severity() string { return "E" }

func (NSFR01) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.NSFRRatio < 100 && parsedDLP.NSFRRatio >= 0 {
		return fmt.Errorf("NSFR Ratio=%v%% < 100%% (mínimo regulatório BACEN Res. 4.542)", parsedDLP.NSFRRatio)
	}
	return nil
}

// NSFR02 — ASF Total (Available Stable Funding) >= 0.
//
// IMPLEMENTAÇÃO REAL — ASF não pode ser negativo.
type NSFR02 struct{}

func (NSFR02) Code() string     { return "NSFR02" }
func (NSFR02) Sheet() string    { return "NSFR" }
func (NSFR02) Severity() string { return "E" }

func (NSFR02) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.ASFTotal < 0 {
		return fmt.Errorf("ASF Total=%v negativo", parsedDLP.ASFTotal)
	}
	return nil
}

// NSFR03 — RSF Total (Required Stable Funding) >= 0.
//
// IMPLEMENTAÇÃO REAL — RSF não pode ser negativo.
type NSFR03 struct{}

func (NSFR03) Code() string     { return "NSFR03" }
func (NSFR03) Sheet() string    { return "NSFR" }
func (NSFR03) Severity() string { return "E" }

func (NSFR03) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.RSFTotal < 0 {
		return fmt.Errorf("RSF Total=%v negativo", parsedDLP.RSFTotal)
	}
	return nil
}

// NSFR04 — ASF >= RSF (equivalente a NSFR >= 100%).
//
// IMPLEMENTAÇÃO REAL — ASF não pode ser menor que RSF.
type NSFR04 struct{}

func (NSFR04) Code() string     { return "NSFR04" }
func (NSFR04) Sheet() string    { return "NSFR" }
func (NSFR04) Severity() string { return "E" }

func (NSFR04) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.ASFTotal < parsedDLP.RSFTotal && parsedDLP.RSFTotal > 0 {
		return fmt.Errorf("ASF Total=%v < RSF Total=%v (NSFR < 100%%)", parsedDLP.ASFTotal, parsedDLP.RSFTotal)
	}
	return nil
}

// NSFR05 — NSFR Ratio calculado = NSFR Ratio declarado (consistência).
//
// IMPLEMENTAÇÃO REAL — verifica se NSFR declarado = calculado.
type NSFR05 struct{}

func (NSFR05) Code() string     { return "NSFR05" }
func (NSFR05) Sheet() string    { return "NSFR" }
func (NSFR05) Severity() string { return "A" }

func (NSFR05) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	calc := CalcularNSFRRatio(parsedDLP.ASFTotal, parsedDLP.RSFTotal)
	if calc < 0 {
		return nil // RSF <= 0, não calcula
	}
	// Tolerância 1% para discrepância (rounding).
	if (calc < parsedDLP.NSFRRatio*0.99) || (calc > parsedDLP.NSFRRatio*1.01) {
		return fmt.Errorf("NSFR declarado=%v%% vs calculado=%v%% (discrepância > 1%%)", parsedDLP.NSFRRatio, calc)
	}
	return nil
}

// NSFR06 — Cenário 1 (base) ASF >= 0.
//
// IMPLEMENTAÇÃO REAL.
type NSFR06 struct{}

func (NSFR06) Code() string     { return "NSFR06" }
func (NSFR06) Sheet() string    { return "NSFR" }
func (NSFR06) Severity() string { return "E" }

func (NSFR06) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.Cenario1.ASF < 0 {
		return fmt.Errorf("Cenário 1 ASF=%v negativo", parsedDLP.Cenario1.ASF)
	}
	return nil
}

// NSFR07 — Cenário 1 (base) RSF >= 0.
//
// IMPLEMENTAÇÃO REAL.
type NSFR07 struct{}

func (NSFR07) Code() string     { return "NSFR07" }
func (NSFR07) Sheet() string    { return "NSFR" }
func (NSFR07) Severity() string { return "E" }

func (NSFR07) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	if parsedDLP.Cenario1.RSF < 0 {
		return fmt.Errorf("Cenário 1 RSF=%v negativo", parsedDLP.Cenario1.RSF)
	}
	return nil
}

// NSFR08 — DtBase DLP dentro do período (formato YYYY-MM-DD).
//
// IMPLEMENTAÇÃO REAL — usa parsedDLP.Root.DataBase (configurado via SetDLP).
type NSFR08 struct{}

func (NSFR08) Code() string     { return "NSFR08" }
func (NSFR08) Sheet() string    { return "NSFR" }
func (NSFR08) Severity() string { return "A" }

func (NSFR08) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil {
		return nil
	}
	dt := parsedDLP.Root.DataBase
	if dt == "" {
		return fmt.Errorf("DLP DtBase vazia")
	}
	if len(dt) != 10 || dt[4] != '-' || dt[7] != '-' {
		return fmt.Errorf("DLP DtBase=%q não está em formato YYYY-MM-DD", dt)
	}
	return nil
}

// parsedDLP é variável global (configurada via SetDLP).
//
// V71-style: globais são aceitáveis para service layer cross-doc, mas
// precisam ser documentadas como tal.
var parsedDLP *DocDLP

// SetDLP configura o DLP para validações cross-doc (chamado pelo service layer).
func SetDLP(doc *DocDLP) {
	parsedDLP = doc
}
