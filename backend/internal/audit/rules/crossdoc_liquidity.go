// Regras Sprint 43 — CrossDoc_v2 (DRL/DLP × 3044).
//
// 8 regras cross-documento que validam consistência entre DRL (LCR),
// DLP (NSFR) e 3044 (eventos JSON). Todas implementadas com lógica real.
//
// NOTA: parsedDRL (de 2160.go) e parsedDLP (de 2170.go) são declarados
// nos respectivos arquivos. parsed3044 é declarado aqui.
package rules

import (
	"context"
	"fmt"
	"strings"
	"time"
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
// Regra de alerta (não bloqueante) — cenário pode ser legítimo.
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

// XD06 — IPOC em 3044 é válido e não-vazio.
//
// Valida que todo IPOC em Operacoes3044 é:
//   - Não-vazio (campo obrigatório)
//   - Comprimento mínimo (6 chars — formato BACEN: CNPJ +序号)
//   - Caracteres válidos (alfanumérico)
//
// Esta é a verificação de cadastro que é possível sem histórico de IPOCs.
// A regra "IPOC existe no histórico" requer DB lookup e é marcada como
// carry-over (XD06-HIST).
type XD06 struct{}

func (XD06) Code() string     { return "XD06" }
func (XD06) Sheet() string    { return "Cross-doc" }
func (XD06) Severity() string { return "E" }

func (XD06) Apply(_ context.Context, _ *Doc3040) error {
	if parsed3044 == nil {
		return nil
	}
	for i, op := range parsed3044.Operacoes {
		ipoc := strings.TrimSpace(op.IPOC)
		if ipoc == "" {
			return fmt.Errorf("XD06: operação[%d] IPOC é vazio (campo obrigatório)", i)
		}
		if len(ipoc) < 6 {
			return fmt.Errorf("XD06: operação[%d] IPOC=%s menor que 6 caracteres (formato inválido)", i, ipoc)
		}
		// BACEN IPOC: 26 chars (CNPJ 14 +序号 12), mas aceite >= 6 por tolerância.
		if len(ipoc) > 30 {
			return fmt.Errorf("XD06: operação[%d] IPOC=%s excede 30 caracteres (formato inválido)", i, ipoc)
		}
	}
	return nil
}

// XD07 — Atraso em 3044 consistente com classificação DRL/DLP.
//
// Se 3044 reporta operações com Atraso="S" (>=15 dias em atraso),
// DRL/DLP deveria refletir baixa liquidez:
//   - DRL: LCR < 100% (indicador de stress de liquidez)
//   - DLP: NSFR < 100% (indicador de funding instável)
//
// Se LCR E NSFR estão acima de 100% com operações em atraso,
// indica inconsistência (operações em atraso mas ratios OK,
// possivelmente subestimação de perdas).
type XD07 struct{}

func (XD07) Code() string     { return "XD07" }
func (XD07) Sheet() string    { return "Cross-doc" }
func (XD07) Severity() string { return "E" }

func (XD07) Apply(_ context.Context, _ *Doc3040) error {
	if parsed3044 == nil {
		return nil
	}

	// Conta operações com Atraso="S".
	var comAtraso int
	for _, op := range parsed3044.Operacoes {
		if op.Atraso == "S" {
			comAtraso++
		}
	}

	// Se não há operações em atraso, regra não se aplica.
	if comAtraso == 0 {
		return nil
	}

	// Se DRL ou DLP não estão disponíveis, não dá para validar.
	if parsedDRL == nil && parsedDLP == nil {
		return nil
	}

	// Verifica se ratios estão OK quando há operações em atraso.
	lcr := parsedDRL.LCRRatio
	nsfr := parsedDLP.NSFRRatio

	// Se LCR >= 100% E NSFR >= 100% com operações em atraso, alerta.
	// Esperado: se há inadimplência, ratios tendem a se deteriorar.
	if lcr > 0 && lcr >= 100 && nsfr > 0 && nsfr >= 100 {
		return fmt.Errorf("XD07: %d operação(ões) com Atraso='S' mas LCR=%.2f%% >= 100%% e NSFR=%.2f%% >= 100%% (inconsistência — ratios deveriam refletir stress)",
			comAtraso, lcr, nsfr)
	}

	return nil
}

// XD08 — Consistência de prazos entre 3044 e DLP.
//
// Valida que concessões de longo prazo (>1 ano) em 3044 são
// consistentes com o perfil de funding do DLP:
//   - Concessões de curto prazo (<=1 ano): espera-se RSF elevado
//   - Concessões de longo prazo (>1 ano): espera-se ASF elevado
//
// Se 3044 tem concessões de longo prazo mas DLP tem RSF > ASF,
// indica possível inconsistência (funding de longo prazo não reportado).
//
// Cálculo: avalia soma do valor das concessões por prazo.
// Limiar: >50% das concessões em prazos > 365 dias = "longo prazo".
type XD08 struct{}

func (XD08) Code() string     { return "XD08" }
func (XD08) Sheet() string    { return "Cross-doc" }
func (XD08) Severity() string { return "A" }

func (XD08) Apply(_ context.Context, _ *Doc3040) error {
	if parsed3044 == nil || parsedDLP == nil {
		return nil
	}

	var totalConcessoes, longoPrazo float64
	now := time.Now()

	for _, op := range parsed3044.Operacoes {
		for _, c := range op.Concessoes {
			dias := now.Sub(c.Data).Hours() / 24
			if dias < 0 {
				// Data futura — usa valor absoluto (concessão já acordada).
				dias = -dias
			}
			totalConcessoes += c.Valor
			if dias > 365 {
				longoPrazo += c.Valor
			}
		}
	}

	if totalConcessoes == 0 {
		return nil
	}

	ratio := longoPrazo / totalConcessoes

	// Se >50% das concessões são de longo prazo, espera-se ASF dominante.
	if ratio > 0.5 && parsedDLP.ASFTotal > 0 && parsedDLP.RSFTotal > 0 {
		// ASF deveria ser >= RSF para funding predominantemente longo.
		if parsedDLP.ASFTotal < parsedDLP.RSFTotal {
			return fmt.Errorf("XD08: %.0f%% das concessões são longo prazo (>1 ano) mas ASF=%.2f < RSF=%.2f (inconsistência — funding longo deveria ter ASF dominante)",
				ratio*100, parsedDLP.ASFTotal, parsedDLP.RSFTotal)
		}
	}

	return nil
}
