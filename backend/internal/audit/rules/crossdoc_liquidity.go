// Regras Sprint 43 — CrossDoc_v2 (DRL/DLP × 3044).
//
// 8 regras cross-documento que validam consistência entre DRL (LCR),
// DLP (NSFR) e 3044 (eventos JSON). XD01-XD05 com lógica real;
// XD06-XD08 carry-over (requerem DB lookup ou parser mais profundo).
//
// Filosofia V73/V74: implementar lógica real, não stubs disfarçados.
//
// NOTA: parsedDRL (de 2160.go) e parsedDLP (de 2170.go) são declarados
// nos respectivos arquivos. parsed3044 é declarado aqui.
package rules

import (
	"context"
	"fmt"
)

// parsed3044 é global para validações cross-doc (set via Set3044).
// parsedDRL e parsedDLP já existem em 2160.go e 2170.go respectivamente.
var parsed3044 *Doc3044

// Set3044 configura o 3044 para validações cross-doc.
func Set3044(doc *Doc3044) { parsed3044 = doc }

// XD01 — CNPJ DRL == CNPJ DLP == CNPJ 3044.
//
// Verifica consistência de CNPJ entre os três documentos de liquidez.
type XD01 struct{}

func (XD01) Code() string     { return "XD01" }
func (XD01) Sheet() string    { return "Cross-doc" }
func (XD01) Severity() string { return "E" }

func (XD01) Apply(_ context.Context, _ *Doc3040) error {
	// Verifica DRL vs DLP.
	if parsedDRL != nil && parsedDLP != nil {
		if parsedDRL.Root.CNPJ != "" && parsedDLP.Root.CNPJ != "" {
			if parsedDRL.Root.CNPJ != parsedDLP.Root.CNPJ {
				return fmt.Errorf("XD01: CNPJ DRL=%s != CNPJ DLP=%s",
					parsedDRL.Root.CNPJ, parsedDLP.Root.CNPJ)
			}
		}
	}
	// Verifica DRL vs 3044.
	if parsedDRL != nil && parsed3044 != nil {
		if parsedDRL.Root.CNPJ != "" && parsed3044.CNPJ != "" {
			if parsedDRL.Root.CNPJ != parsed3044.CNPJ {
				return fmt.Errorf("XD01: CNPJ DRL=%s != CNPJ 3044=%s",
					parsedDRL.Root.CNPJ, parsed3044.CNPJ)
			}
		}
	}
	// Verifica DLP vs 3044.
	if parsedDLP != nil && parsed3044 != nil {
		if parsedDLP.Root.CNPJ != "" && parsed3044.CNPJ != "" {
			if parsedDLP.Root.CNPJ != parsed3044.CNPJ {
				return fmt.Errorf("XD01: CNPJ DLP=%s != CNPJ 3044=%s",
					parsedDLP.Root.CNPJ, parsed3044.CNPJ)
			}
		}
	}
	return nil
}

// XD02 — DtBase DRL == DtBase DLP == dataSaldoDevedor 3044.
//
// Verifica consistência temporal entre os três documentos.
// Tolerância: mesmo dia (YYYY-MM-DD) para DRL/DLP vs 3044.
type XD02 struct{}

func (XD02) Code() string     { return "XD02" }
func (XD02) Sheet() string    { return "Cross-doc" }
func (XD02) Severity() string { return "E" }

func (XD02) Apply(_ context.Context, _ *Doc3040) error {
	// Extrai data do DRL (YYYY-MM-DD).
	var drlDate, dlpDate string
	if parsedDRL != nil {
		drlDate = parsedDRL.Root.DataBase
	}
	if parsedDLP != nil {
		dlpDate = parsedDLP.Root.DataBase
	}

	// Verifica DRL vs DLP.
	if drlDate != "" && dlpDate != "" && drlDate != dlpDate {
		return fmt.Errorf("XD02: DtBase DRL=%s != DtBase DLP=%s", drlDate, dlpDate)
	}

	// Verifica DRL vs 3044.
	if parsedDRL != nil && parsed3044 != nil && len(parsed3044.Operacoes) > 0 {
		dtr := parsedDRL.Root.DataBase
		for _, op := range parsed3044.Operacoes {
			opDate := op.DataSaldoDevedor.Format("2006-01-02")
			if dtr != "" && opDate != dtr {
				return fmt.Errorf("XD02: DtBase DRL=%s != dataSaldoDevedor 3044 IPOC %s=%s",
					dtr, op.IPOC, opDate)
			}
		}
	}

	// Verifica DLP vs 3044.
	if parsedDLP != nil && parsed3044 != nil && len(parsed3044.Operacoes) > 0 {
		dlpDt := parsedDLP.Root.DataBase
		for _, op := range parsed3044.Operacoes {
			opDate := op.DataSaldoDevedor.Format("2006-01-02")
			if dlpDt != "" && opDate != dlpDt {
				return fmt.Errorf("XD02: DtBase DLP=%s != dataSaldoDevedor 3044 IPOC %s=%s",
					dlpDt, op.IPOC, opDate)
			}
		}
	}
	return nil
}

// XD03 — Soma saldoDevedor 3044 >= HQLA DRL.
//
// HQLA (High Quality Liquid Assets) deve ser suficiente para cobrir
// os saldos devedores das operações reportadas em 3044.
type XD03 struct{}

func (XD03) Code() string     { return "XD03" }
func (XD03) Sheet() string    { return "Cross-doc" }
func (XD03) Severity() string { return "A" }

func (XD03) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDRL == nil || parsed3044 == nil {
		return nil // sem contexto cross-doc
	}
	if parsedDRL.HQLA <= 0 {
		return nil // HQLA não configurado ou zero
	}

	somaSaldos := 0.0
	for _, op := range parsed3044.Operacoes {
		somaSaldos += op.SaldoDevedor
	}

	// HQLA deve ser >= soma dos saldos (regra de cobertura conservadora).
	// Na prática, HQLA é numerador do LCR e saldos são parcialmente cobertos.
	// Regra avisa se soma é maior que HQLA (indica possível subcobertura).
	if somaSaldos > parsedDRL.HQLA {
		return fmt.Errorf("XD03: soma saldoDevedor 3044=%.2f > HQLA DRL=%.2f",
			somaSaldos, parsedDRL.HQLA)
	}
	return nil
}

// XD04 — NSFR Ratio e LCR Ratio consistentes.
//
// Se LCR < 100% E NSFR >= 100%, pode indicar inconsistência (instituição
// com funding estável mas liquidez de curto prazo comprometida).
// Regra de警告 (não bloqueante) — cenário pode ser legítimo.
type XD04 struct{}

func (XD04) Code() string     { return "XD04" }
func (XD04) Sheet() string    { return "Cross-doc" }
func (XD04) Severity() string { return "A" }

func (XD04) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDRL == nil || parsedDLP == nil {
		return nil // sem contexto cross-doc
	}

	lcr := parsedDRL.LCRRatio
	nsfr := parsedDLP.NSFRRatio

	// Regra: se LCR < 80% E NSFR > 120%, alerta de inconsistência.
	// Esperado: se funding de longo prazo está OK (NSFR alto),
	// liquidity de curto prazo (LCR) tende a ser similarmente OK.
	if lcr > 0 && nsfr > 0 && lcr < 80 && nsfr > 120 {
		return fmt.Errorf("XD04: LCR=%v%% < 80%% mas NSFR=%v%% > 120%% (inconsistência)",
			lcr, nsfr)
	}
	return nil
}

// XD05 — Soma pagamentos 3044 consistente com Outflows DRL.
//
// A soma dos pagamentos em 3044 (eventos que reduzem saldos) deve ser
// <= Outflows reportados no DRL.
type XD05 struct{}

func (XD05) Code() string     { return "XD05" }
func (XD05) Sheet() string    { return "Cross-doc" }
func (XD05) Severity() string { return "A" }

func (XD05) Apply(_ context.Context, _ *Doc3040) error {
	if parsedDRL == nil || parsed3044 == nil {
		return nil
	}
	if parsedDRL.Outflows <= 0 {
		return nil
	}

	somaPagamentos := 0.0
	for _, op := range parsed3044.Operacoes {
		for _, p := range op.Pagamentos {
			somaPagamentos += p.Valor
		}
	}

	// Outflows DRL inclui pagamentos + outras saídas.
	// Se soma pagamentos > Outflows, pode indicar subestimação de outflows.
	if somaPagamentos > parsedDRL.Outflows {
		return fmt.Errorf("XD05: soma pagamentos 3044=%.2f > Outflows DRL=%.2f",
			somaPagamentos, parsedDRL.Outflows)
	}
	return nil
}

// XD06 — IPOC em 3044 existe no histórico DDR/DLO.
//
// CARRY-OVER: requer DB lookup do histórico de IPOCs.
// Implementação real: consultar tabela de IPOCs conhecidos.
type XD06 struct{}

func (XD06) Code() string     { return "XD06" }
func (XD06) Sheet() string    { return "Cross-doc" }
func (XD06) Severity() string { return "E" }

func (XD06) Apply(_ context.Context, _ *Doc3040) error {
	// CARRY-OVER: não pode validar sem acesso ao histórico de IPOCs.
	// Documentado para implementar quando DB layer estiver pronto.
	return nil
}

// XD07 — Atraso em 3044 consistente com classificação DRL/DLP.
//
// Se 3044 reporta atraso='S' (>=15 dias), DRL/DLP deveria refletir
// baixa liquidez (LCR/NSFR sob pressão).
// CARRY-OVER: requer validação de consistência de classificação.
type XD07 struct{}

func (XD07) Code() string     { return "XD07" }
func (XD07) Sheet() string    { return "Cross-doc" }
func (XD07) Severity() string { return "E" }

func (XD07) Apply(_ context.Context, _ *Doc3040) error {
	// CARRY-OVER: requer parser cross-doc que associe IPOC a classificação.
	// Documentado para implementar quando service layer suportar join.
	return nil
}

// XD08 — Consistência de prazos entre 3044 e DRL/DLP.
//
// Data de concessão em 3044 deve ser consistente com prazos de funding
// reportados em DLP (ASF/RSF).
// CARRY-OVER: requer análise temporal mais profunda.
type XD08 struct{}

func (XD08) Code() string     { return "XD08" }
func (XD08) Sheet() string    { return "Cross-doc" }
func (XD08) Severity() string { return "A" }

func (XD08) Apply(_ context.Context, _ *Doc3040) error {
	// CARRY-OVER: requer análise temporal de concessões vs ASF/RSF.
	// Documentado para implementar em sprint futura.
	return nil
}
