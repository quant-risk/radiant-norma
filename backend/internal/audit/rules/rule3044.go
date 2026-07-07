// Regras Sprint 42 — Audit3044 (Engine JSON — Eventos de Operações de Crédito).
//
// 17 regras T01-T19 (T18/T19 carry-over: dependem de DB lookup).
// Filosofia V71/V72: implementar lógica real, não stubs disfarçados.
package rules

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Rule3044 é a interface de uma regra de validação 3044.
type Rule3044 interface {
	Code() string
	Severity() string // E (Erro bloqueante), A (Aviso), I (Informativo)
	Apply(ctx context.Context, doc *Doc3044) error
}

// T01 — dataHoraRemessa >= dataSaldoDevedor (para cada operação).
type T01 struct{}

func (T01) Code() string     { return "T01" }
func (T01) Severity() string { return "E" }

func (T01) Apply(_ context.Context, doc *Doc3044) error {
	for _, op := range doc.Operacoes {
		if doc.DataHoraRemessa.Before(op.DataSaldoDevedor) {
			return fmt.Errorf("T01: dataHoraRemessa=%v < dataSaldoDevedor=%v (op IPOC %s)",
				doc.DataHoraRemessa.Format("2006-01-02 15:04:05"), op.DataSaldoDevedor.Format("2006-01-02"), op.IPOC)
		}
	}
	return nil
}

// T02 — Pagamentos: data <= dataSaldoDevedor.
type T02 struct{}

func (T02) Code() string     { return "T02" }
func (T02) Severity() string { return "E" }

func (T02) Apply(_ context.Context, doc *Doc3044) error {
	for _, op := range doc.Operacoes {
		for _, p := range op.Pagamentos {
			if p.Data.After(op.DataSaldoDevedor) {
				return fmt.Errorf("T02: pagamento data=%v > dataSaldoDevedor=%v (IPOC %s)",
					p.Data.Format("2006-01-02"), op.DataSaldoDevedor.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T03 — Concessões: data <= dataSaldoDevedor.
type T03 struct{}

func (T03) Code() string     { return "T03" }
func (T03) Severity() string { return "E" }

func (T03) Apply(_ context.Context, doc *Doc3044) error {
	for _, op := range doc.Operacoes {
		for _, c := range op.Concessoes {
			if c.Data.After(op.DataSaldoDevedor) {
				return fmt.Errorf("T03: concessao data=%v > dataSaldoDevedor=%v (IPOC %s)",
					c.Data.Format("2006-01-02"), op.DataSaldoDevedor.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T04 — dataHoraRemessa não futura, não >21 dias antiga.
//
// Rejeita se:
//   - dataHoraRemessa > agora (futura)
//   - dataHoraRemessa < agora - 21 dias (muito antiga)
type T04 struct{}

func (T04) Code() string     { return "T04" }
func (T04) Severity() string { return "E" }

func (T04) Apply(_ context.Context, doc *Doc3044) error {
	now := time.Now()
	if doc.DataHoraRemessa.After(now) {
		return fmt.Errorf("T04: dataHoraRemessa=%v é futura (agora=%v)",
			doc.DataHoraRemessa.Format("2006-01-02 15:04:05"), now.Format("2006-01-02 15:04:05"))
	}
	cutoff := now.AddDate(0, 0, -21)
	if doc.DataHoraRemessa.Before(cutoff) {
		return fmt.Errorf("T04: dataHoraRemessa=%v é anterior a 21 dias (cutoff=%v)",
			doc.DataHoraRemessa.Format("2006-01-02 15:04:05"), cutoff.Format("2006-01-02 15:04:05"))
	}
	return nil
}

// T05 — Sem pagamentos duplicados (mesmo IPOC + mesma data).
type T05 struct{}

func (T05) Code() string     { return "T05" }
func (T05) Severity() string { return "E" }

func (T05) Apply(_ context.Context, doc *Doc3044) error {
	seen := make(map[string]int) // key = "IPOC#YYYY-MM-DD" -> count
	for _, op := range doc.Operacoes {
		for _, p := range op.Pagamentos {
			key := fmt.Sprintf("%s#%s", op.IPOC, p.Data.Format("2006-01-02"))
			seen[key]++
			if seen[key] > 1 {
				return fmt.Errorf("T05: pagamento duplicado IPOC=%s data=%v (count=%d)",
					op.IPOC, p.Data.Format("2006-01-02"), seen[key])
			}
		}
	}
	return nil
}

// T06 — Sem concessões duplicadas (mesmo IPOC + mesma data).
type T06 struct{}

func (T06) Code() string     { return "T06" }
func (T06) Severity() string { return "E" }

func (T06) Apply(_ context.Context, doc *Doc3044) error {
	seen := make(map[string]int)
	for _, op := range doc.Operacoes {
		for _, c := range op.Concessoes {
			key := fmt.Sprintf("%s#%s", op.IPOC, c.Data.Format("2006-01-02"))
			seen[key]++
			if seen[key] > 1 {
				return fmt.Errorf("T06: concessao duplicada IPOC=%s data=%v (count=%d)",
					op.IPOC, c.Data.Format("2006-01-02"), seen[key])
			}
		}
	}
	return nil
}

// T07 — class3050 proibido se envia3050='N'.
type T07 struct{}

func (T07) Code() string     { return "T07" }
func (T07) Severity() string { return "E" }

func (T07) Apply(_ context.Context, doc *Doc3044) error {
	if doc.Envia3050 == "N" {
		for _, op := range doc.Operacoes {
			if op.Class3050 != "" {
				return fmt.Errorf("T07: class3050=%q presente mas envia3050='N' (IPOC %s)", op.Class3050, op.IPOC)
			}
		}
	}
	return nil
}

// T08 — class3050 deve pertencer ao domínio se envia3050='S'.
//
// Validação estrutural: class3050 deve ter formato 9 dígitos (prefixo
// DOC/DIR/DAC + código). Carry-over: validação plena requer domínio do BACEN.
type T08 struct{}

func (T08) Code() string     { return "T08" }
func (T08) Severity() string { return "A" }

func (T08) Apply(_ context.Context, doc *Doc3044) error {
	if doc.Envia3050 != "S" {
		return nil
	}
	for _, op := range doc.Operacoes {
		if op.Class3050 == "" {
			continue
		}
		// Formato esperado: 9 dígitos (código de classificação DOC/DIR/DAC).
		// Exemplo: "112212101"
		if len(op.Class3050) != 9 {
			return fmt.Errorf("T08: class3050=%q não tem 9 dígitos (IPOC %s)", op.Class3050, op.IPOC)
		}
		// Deve conter apenas dígitos.
		for _, c := range op.Class3050 {
			if c < '0' || c > '9' {
				return fmt.Errorf("T08: class3050=%q contém caractere não numérico (IPOC %s)", op.Class3050, op.IPOC)
			}
		}
	}
	return nil
}

// T11 — Data pagamento dentro dos últimos 6 meses.
type T11 struct{}

func (T11) Code() string     { return "T11" }
func (T11) Severity() string { return "E" }

func (T11) Apply(_ context.Context, doc *Doc3044) error {
	cutoff := time.Now().AddDate(0, -6, 0)
	for _, op := range doc.Operacoes {
		for _, p := range op.Pagamentos {
			if p.Data.Before(cutoff) {
				return fmt.Errorf("T11: pagamento data=%v fora dos últimos 6 meses (IPOC %s)",
					p.Data.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T12 — Data concessão dentro dos últimos 6 meses.
type T12 struct{}

func (T12) Code() string     { return "T12" }
func (T12) Severity() string { return "E" }

func (T12) Apply(_ context.Context, doc *Doc3044) error {
	cutoff := time.Now().AddDate(0, -6, 0)
	for _, op := range doc.Operacoes {
		for _, c := range op.Concessoes {
			if c.Data.Before(cutoff) {
				return fmt.Errorf("T12: concessao data=%v fora dos últimos 6 meses (IPOC %s)",
					c.Data.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T13 — Data cessão dentro dos últimos 6 meses.
type T13 struct{}

func (T13) Code() string     { return "T13" }
func (T13) Severity() string { return "E" }

func (T13) Apply(_ context.Context, doc *Doc3044) error {
	cutoff := time.Now().AddDate(0, -6, 0)
	for _, op := range doc.Operacoes {
		for _, cs := range op.Cessoes {
			if cs.Data.Before(cutoff) {
				return fmt.Errorf("T13: cessao data=%v fora dos últimos 6 meses (IPOC %s)",
					cs.Data.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T14 — Data aquisição dentro dos últimos 6 meses.
type T14 struct{}

func (T14) Code() string     { return "T14" }
func (T14) Severity() string { return "E" }

func (T14) Apply(_ context.Context, doc *Doc3044) error {
	cutoff := time.Now().AddDate(0, -6, 0)
	for _, op := range doc.Operacoes {
		for _, aq := range op.Aquisicoes {
			if aq.Data.Before(cutoff) {
				return fmt.Errorf("T14: aquisicao data=%v fora dos últimos 6 meses (IPOC %s)",
					aq.Data.Format("2006-01-02"), op.IPOC)
			}
		}
	}
	return nil
}

// T15 — Valores não podem ser negativos (pagamentos, concessões, cessões, aquisições).
type T15 struct{}

func (T15) Code() string     { return "T15" }
func (T15) Severity() string { return "E" }

func (T15) Apply(_ context.Context, doc *Doc3044) error {
	for _, op := range doc.Operacoes {
		for _, p := range op.Pagamentos {
			if p.Valor < 0 {
				return fmt.Errorf("T15: pagamento valor negativo=%.2f (IPOC %s)", p.Valor, op.IPOC)
			}
		}
		for _, c := range op.Concessoes {
			if c.Valor < 0 {
				return fmt.Errorf("T15: concessao valor negativo=%.2f (IPOC %s)", c.Valor, op.IPOC)
			}
		}
		for _, cs := range op.Cessoes {
			if cs.Valor < 0 {
				return fmt.Errorf("T15: cessao valor negativo=%.2f (IPOC %s)", cs.Valor, op.IPOC)
			}
		}
		for _, aq := range op.Aquisicoes {
			if aq.Valor < 0 {
				return fmt.Errorf("T15: aquisicao valor negativo=%.2f (IPOC %s)", aq.Valor, op.IPOC)
			}
		}
	}
	return nil
}

// T16 — saldoDevedor não negativo (exceto anuidade/cashback — não distingue).
type T16 struct{}

func (T16) Code() string     { return "T16" }
func (T16) Severity() string { return "E" }

func (T16) Apply(_ context.Context, doc *Doc3044) error {
	for _, op := range doc.Operacoes {
		if op.SaldoDevedor < 0 {
			return fmt.Errorf("T16: saldoDevedor negativo=%.2f (IPOC %s)", op.SaldoDevedor, op.IPOC)
		}
	}
	return nil
}

// T17 — IPOC não pode repetir no mesmo documento.
type T17 struct{}

func (T17) Code() string     { return "T17" }
func (T17) Severity() string { return "E" }

func (T17) Apply(_ context.Context, doc *Doc3044) error {
	seen := make(map[string]bool)
	for _, op := range doc.Operacoes {
		if seen[op.IPOC] {
			return fmt.Errorf("T17: IPOC duplicado no documento: %s", op.IPOC)
		}
		seen[op.IPOC] = true
	}
	return nil
}

// T18 — acao=2 (Excluir) requer IPOC existente na base.
//
// CARRY-OVER: requer DB lookup (não implementado nesta sprint).
// Implementação real: consultar tabela de IPOCs existentes.
type T18 struct{}

func (T18) Code() string     { return "T18" }
func (T18) Severity() string { return "E" }

func (T18) Apply(_ context.Context, doc *Doc3044) error {
	// Stub: não pode validar existência na base sem contexto de DB.
	// Carry-over documentado — implementar quando DB layer estiver pronto.
	_ = doc
	return nil
}

// T19 — acao=3 (Alterar) requer IPOC existente na base.
//
// CARRY-OVER: requer DB lookup (não implementado nesta sprint).
type T19 struct{}

func (T19) Code() string     { return "T19" }
func (T19) Severity() string { return "E" }

func (T19) Apply(_ context.Context, doc *Doc3044) error {
	// Stub: não pode validar existência na base sem contexto de DB.
	_ = doc
	return nil
}

// dominioClass3050 é o conjunto de códigos 3050 válidos para T08.
//
// Extraído do XSD BACEN 3050 (TXB_V11). Prefixos válidos:
// 1 = Crédito Pessoal, 2 = Habitacional, 3 = Rural, etc.
// Validado como string numérica de 9 dígitos.
var dominioClass3050Prefixes = []string{
	"1", "2", "3", "4", "5", "6", "7", "8", "9",
}

// Class3050Valido verifica se class3050 pertence ao domínio (prefixo válido).
//
// Validação T08: class3050 deve começar com dígito 1-9 e ter 9 dígitos.
// Carry-over: validação plena (código completo) fica para sprint futura.
func Class3050Valido(class3050 string) bool {
	if len(class3050) != 9 {
		return false
	}
	for _, c := range class3050 {
		if c < '0' || c > '9' {
			return false
		}
	}
	return strings.Contains("123456789", class3050[:1])
}
