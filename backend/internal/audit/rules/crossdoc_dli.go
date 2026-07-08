// Regras Sprint 53 — CrossDoc DLI × DRL × DLP.
//
// Cross-doc de liquidez: valida consistência entre DLI (2062 — Limites
// Operacionais), DRL (2160 — LCR) e DLP (2170 — NSFR).
//
// Regras XD-DLI-01 a XD-DLI-06:
//
//   - XD-DLI-01: CNPJ DLI == CNPJ DRL == CNPJ DLP
//   - XD-DLI-02: DataBase DLI == DataBase DRL == DataBase DLP
//   - XD-DLI-03: PLA (6.10.01 DLI) >= Capital mínimo para LCR
//   - XD-DLI-04: Margem PLA (6.00.00 DLI) >= 0
//   - XD-DLI-05: Limite 20.00 (PR) * 5 >= operações PR no DRL (se disponível)
//   - XD-DLI-06: NSFR Ratio (DLP) >= 100% OR LCR Ratio (DRL) >= 80% (consistência)
//
// Filosofia: DLI é o documento de limites; DRL/DLP medem compliance.
// Se DLI está ok mas DRL/DLP está em problema, é flag de alerta.
// Se DLI está ok E DRL/DLP estão ok, confidence aumenta.
package rules

import (
	"context"
	"fmt"
)

// XD-DLI-01 — CNPJ DLI == CNPJ DRL == CNPJ DLP.
type XDDLI01CNPJConsistente struct{}

func (XDDLI01CNPJConsistente) Code() string     { return "XD-DLI-01" }
func (XDDLI01CNPJConsistente) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI01CNPJConsistente) Severity() string { return "E" }

func (XDDLI01CNPJConsistente) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLI == nil {
		return nil // sem DLI, não aplica
	}
	dliCNPJ := parsedDLI.Root.CNPJ

	// DLI vs DRL.
	if parsedDRL != nil && parsedDRL.Root.CNPJ != "" && dliCNPJ != "" {
		if dliCNPJ != parsedDRL.Root.CNPJ {
			return fmt.Errorf("XD-DLI-01: CNPJ DLI=%s != CNPJ DRL=%s",
				dliCNPJ, parsedDRL.Root.CNPJ)
		}
	}
	// DLI vs DLP.
	if parsedDLP != nil && parsedDLP.Root.CNPJ != "" && dliCNPJ != "" {
		if dliCNPJ != parsedDLP.Root.CNPJ {
			return fmt.Errorf("XD-DLI-01: CNPJ DLI=%s != CNPJ DLP=%s",
				dliCNPJ, parsedDLP.Root.CNPJ)
		}
	}
	return nil
}

// XD-DLI-02 — DataBase DLI == DataBase DRL == DataBase DLP.
type XDDLI02DataBaseConsistente struct{}

func (XDDLI02DataBaseConsistente) Code() string     { return "XD-DLI-02" }
func (XDDLI02DataBaseConsistente) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI02DataBaseConsistente) Severity() string { return "E" }

func (XDDLI02DataBaseConsistente) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLI == nil {
		return nil
	}
	dliDB := parsedDLI.Root.DataBase

	// DLI vs DRL.
	if parsedDRL != nil && parsedDRL.Root.DataBase != "" && dliDB != "" {
		if dliDB != parsedDRL.Root.DataBase {
			return fmt.Errorf("XD-DLI-02: DataBase DLI=%s != DataBase DRL=%s",
				dliDB, parsedDRL.Root.DataBase)
		}
	}
	// DLI vs DLP.
	if parsedDLP != nil && parsedDLP.Root.DataBase != "" && dliDB != "" {
		if dliDB != parsedDLP.Root.DataBase {
			return fmt.Errorf("XD-DLI-02: DataBase DLI=%s != DataBase DLP=%s",
				dliDB, parsedDLP.Root.DataBase)
		}
	}
	return nil
}

// XD-DLI-03 — PLA (6.10.01 DLI) >= capital mínimo para operar (regra Prudencial).
//
// PLA deve ser positivo para que LCR/NSFR sejam significativos.
// Se PLA <= 0 E LCR/NSFR reportados como OK, é inconsistência.
type XDDLI03PLAPositivo struct{}

func (XDDLI03PLAPositivo) Code() string     { return "XD-DLI-03" }
func (XDDLI03PLAPositivo) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI03PLAPositivo) Severity() string { return "E" }

func (XDDLI03PLAPositivo) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLI == nil {
		return nil
	}
	pla := SomaPLA(parsedDLI.Accounts)
	if pla <= 0 {
		return fmt.Errorf("XD-DLI-03: PLA=%v deve ser > 0 (documento DLI)", pla)
	}
	return nil
}

// XD-DLI-04 — Margem PLA (6.00.00 DLI) >= 0.
//
// A margem PLA (ativo líquido após dedução do requerimento) não pode ser
// negativa — indicaria que o requerimento de capital excede o PLA.
type XDDLI04MargemPLANaoNegativa struct{}

func (XDDLI04MargemPLANaoNegativa) Code() string     { return "XD-DLI-04" }
func (XDDLI04MargemPLANaoNegativa) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI04MargemPLANaoNegativa) Severity() string { return "E" }

func (XDDLI04MargemPLANaoNegativa) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLI == nil {
		return nil
	}
	margem := parsedDLI.Accounts["6.00.00"]
	if margem < 0 {
		return fmt.Errorf("XD-DLI-04: Margem PLA (6.00.00)=%v não pode ser negativa", margem)
	}
	return nil
}

// XD-DLI-05 — Capital Realizado (8.10.01 DLI) >= 10M (mínimo prudencial para bancos).
//
// Instituições com PLA positivo mas Capital Realizado muito baixo indicam
// estrutura de capital atípica — alerta para validação manual.
type XDDLI05CapitalRealizadoMinimo struct{}

func (XDDLI05CapitalRealizadoMinimo) Code() string     { return "XD-DLI-05" }
func (XDDLI05CapitalRealizadoMinimo) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI05CapitalRealizadoMinimo) Severity() string { return "A" }

func (XDDLI05CapitalRealizadoMinimo) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLI == nil {
		return nil
	}
	cap := SomaCapitalRealizado(parsedDLI.Accounts)
	if cap > 0 && cap < 10_000_000 {
		return fmt.Errorf("XD-DLI-05: Capital Realizado=%v < 10M (mínimo prudencial)", cap)
	}
	return nil
}

// XD-DLI-06 — Se NSFR >= 100% E LCR < 80%, alerta (possível inconsistência).
//
// Funding estável de longo prazo (NSFR) deveria normalmente acompanhar
// liquidez de curto prazo (LCR). Se NSFR está OK mas LCR está crítico,
// indica mismatch de prazos.
type XDDLI06NSFRxLCRConsistente struct{}

func (XDDLI06NSFRxLCRConsistente) Code() string     { return "XD-DLI-06" }
func (XDDLI06NSFRxLCRConsistente) Sheet() string    { return "Cross-doc-DLI" }
func (XDDLI06NSFRxLCRConsistente) Severity() string { return "A" }

func (XDDLI06NSFRxLCRConsistente) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDLP == nil || parsedDRL == nil {
		return nil
	}
	nsfr := parsedDLP.NSFRRatio
	lcr := parsedDRL.LCRRatio

	if nsfr >= 100 && lcr < 80 {
		return fmt.Errorf("XD-DLI-06: NSFR=%v%% >= 100%% mas LCR=%v%% < 80%% (inconsistência)",
			nsfr, lcr)
	}
	return nil
}
