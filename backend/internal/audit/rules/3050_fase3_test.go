// Tests Fase 3 — Sprint 33 Fase 3 (v3.34.4).
//
// Cobre:
// - H10-H15 (6 regras Header)
// - I15-I28 (14 regras Individuais)
// - S29-S32 (4 regras Sistema)
// - S09/S13/S24 (carry-over — saem de stub)
//
// Total: 28 funções de teste table-driven + 1 integração.
package rules

import (
	"context"
	"testing"
	"time"
)

// ========== H10-H15 — Header ==========

func TestH10_CNPJLength(t *testing.T) {
	tests := []struct {
		nome        string
		cnpj        string
		wantErrSubs string
	}{
		{"happy: 8 dígitos", "12345678", ""},
		{"violação: 7 dígitos", "1234567", "length=7"},
		{"violação: 9 dígitos (com DV)", "123456789", "length=9"},
		{"violação: vazio", "", "length=0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{CNPJ: tt.cnpj, DataBase: "2024-12-31"}}
			err := H10CNPJLength{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH11_CNPJAllDigits(t *testing.T) {
	tests := []struct {
		nome        string
		cnpj        string
		wantErrSubs string
	}{
		{"happy: 8 dígitos", "12345678", ""},
		{"violação: letra", "1234567A", "não-numérico"},
		{"violação: símbolo", "12345-78", "não-numérico"},
		{"violação: espaço", "12345 78", "não-numérico"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{CNPJ: tt.cnpj, DataBase: "2024-12-31"}}
			err := H11CNPJAllDigits{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH12_DataBaseFormatoRigoroso(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: YYYY-MM-DD", "2024-12-31", ""},
		{"violação: length errado (8 chars, sem sep)", "20241231", "length=8"},
		{"violação: length errado (11 chars)", "2024-12-311", "length=11"},
		{"violação: separador /", "2024/12/31", "'-'"},
		{"violação: separador . (em length 10)", "2024.12.31", "'-'"},
		{"violação: char não-numérico (em length 10)", "2024-1A-31", "não-numérico"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{DataBase: tt.dataBase}}
			err := H12DataBaseFormatoRigoroso{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH13_IndRemessaCaseSensitive(t *testing.T) {
	tests := []struct {
		nome        string
		ind         string
		wantErrSubs string
	}{
		{"happy: I", "I", ""},
		{"happy: A", "A", ""},
		{"happy: S", "S", ""},
		{"violação: i minúsculo", "i", "case-sensitive"},
		{"violação: X inválido", "X", "case-sensitive"},
		{"violação: vazio", "", "case-sensitive"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{IndRemessa: tt.ind}}
			err := H13IndRemessaCaseSensitive{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH14_NmContatoSemEspacosDuplicados(t *testing.T) {
	tests := []struct {
		nome        string
		nm          string
		wantErrSubs string
	}{
		{"happy: nome normal", "João da Silva", ""},
		{"happy: 1 palavra", "João", ""},
		{"violação: 2 espaços", "João  Silva", "espaços duplicados"},
		{"violação: 3 espaços", "João   Silva", "espaços duplicados"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{NmContato: tt.nm}}
			err := H14NmContatoSemEspacosDuplicados{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH15_TelContatoSemCaracteresResiduais(t *testing.T) {
	tests := []struct {
		nome        string
		tel         string
		wantErrSubs string
	}{
		{"happy: 11 dígitos", "11999998888", ""},
		{"happy: formatado", "(11)99999-8888", ""},
		{"happy: com +", "+5511999998888", ""},
		{"violação: letras", "1199999abcd", "caractere inválido"},
		{"violação: símbolo #", "11999#8888", "caractere inválido"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{TelContato: tt.tel}}
			err := H15TelContatoSemCaracteresResiduais{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== I15-I20 — Não-negatividade por sub-modalidade ==========

func TestI15_DesDuplicatasSldCarNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 1000.0", ptrF(1000.0), ""},
		{"happy: 0.0", ptrF(0.0), ""},
		{"violação: -1.0", ptrF(-1.0), "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Mensal: []Modalidade{{Codigo: "desDuplicatas", Encargo: "pre", TipoCli: "pesJuridica", SldCarAtiva: tt.val}}}
			err := I15DesDuplicatasSldCarNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI16_DesChequesVlrConcessoesNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 50000.0", ptrF(50000.0), ""},
		{"violação: -100.0", ptrF(-100.0), "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "desCheques", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: tt.val}}}
			err := I16DesChequesVlrConcessoesNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI17_VendorTxMedJurosNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 25.5", ptrF(25.5), ""},
		{"violação: -0.5", ptrF(-0.5), "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "vendor", Encargo: "pre", TipoCli: "pesJuridica", TxMedJuros: tt.val}}}
			err := I17VendorTxMedJurosNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI18_ComprorPrzDecNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         int
		wantErrSubs string
	}{
		{"happy: 90 dias", 90, ""},
		{"violação: -5 dias", -5, "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "compror", Encargo: "pre", TipoCli: "pesJuridica", PrzDecMedConcessoes: &tt.val}}}
			err := I18ComprorPrzDecNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI19_CarCrdSldCarNaoNeg(t *testing.T) {
	doc := &Doc3050{Mensal: []Modalidade{{Codigo: "carCrd", Encargo: "pre", TipoCli: "pesFisica", SldCarAtiva: ptrF(-100.0)}}}
	err := I19CarCrdSldCarNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI20_CarCrdVlrConcessoesNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "carCrd", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-50.0)}}}
	err := I20CarCrdVlrConcessoesNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

// ========== I21-I26 — Cruzadas (todas modalidades) ==========

func TestI21_TxMedJurosMax100(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 50.0%", ptrF(50.0), ""},
		{"happy: 100.0% (limite)", ptrF(100.0), ""},
		{"violação: 100.5%", ptrF(100.5), "> 100%"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", TxMedJuros: tt.val}}}
			err := I21TxMedJurosMax100{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI22_TxMedEncOperMax50(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 25.0%", ptrF(25.0), ""},
		{"violação: 55.0%", ptrF(55.0), "> 50%"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", TxMedEncOperacionais: tt.val}}}
			err := I22TxMedEncOperMax50{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI23_CapGirPrzDecMax5000(t *testing.T) {
	tests := []struct {
		nome        string
		val         int
		wantErrSubs string
	}{
		{"happy: 365 dias", 365, ""},
		{"violação: 5500 dias", 5500, "> 5000 dias"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzSup365", Encargo: "pre", TipoCli: "pesJuridica", PrzDecMedConcessoes: &tt.val}}}
			err := I23CapGirPrzDecMax5000{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI24_QtdNovContratosNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         int
		wantErrSubs string
	}{
		{"happy: 50", 50, ""},
		{"violação: -1", -1, "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", QtdNovContratos: &tt.val}}}
			err := I24QtdNovContratosNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI25_SldCedidoNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", SldCedido: ptrF(-10.0)}}}
	err := I25SldCedidoNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI26_SldAdquiridoNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", SldAdquirido: ptrF(-5.0)}}}
	err := I26SldAdquiridoNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

// ========== I27-I28 — Inconsistência composta ==========

func TestI27_SldCarAtivaImpoeTxMaxGtMin(t *testing.T) {
	tests := []struct {
		nome        string
		sldCar      *float64
		txMin       *float64
		txMax       *float64
		wantErrSubs string
	}{
		{"happy: sldCar=0 ignora tx", ptrF(0.0), ptrF(10.0), ptrF(5.0), ""},
		{"happy: txMax > txMin", ptrF(1000.0), ptrF(10.0), ptrF(25.0), ""},
		{"violação: txMax <= txMin com sldCar>0", ptrF(1000.0), ptrF(20.0), ptrF(15.0), "inconsistência"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica",
				SldCarAtiva: tt.sldCar, TxMinima: tt.txMin, TxMaxima: tt.txMax}}}
			err := I27SldCarAtivaImpoeTxMaxGtMin{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI28_IndRemessaIExigeNovContratos(t *testing.T) {
	tests := []struct {
		nome        string
		ind         string
		qtdNov      *int
		wantErrSubs string
	}{
		{"happy: indRemessa=I + qtdNov=50", "I", ptrI(50), ""},
		{"happy: indRemessa=A sem exigir", "A", nil, ""},
		{"violação: indRemessa=I sem qtdNov", "I", nil, "qtdNovContratos ≥ 1"},
		{"violação: indRemessa=I com qtdNov=0", "I", ptrI(0), "qtdNovContratos ≥ 1"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{
				Root:   Doc3050Root{IndRemessa: tt.ind},
				Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica", QtdNovContratos: tt.qtdNov}},
			}
			err := I28IndRemessaIExigeNovContratos{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S29-S32 — Sistema ==========

func TestS29_DataBaseRangePlausivel(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: 2024-12-31", "2024-12-31", ""},
		{"violação: 2008-01-01 (anterior)", "2008-01-01", "anterior a 2009-01-01"},
		{"violação: 2099-12-31 (muito futuro)", "2099-12-31", "futuro distante"},
		{"skip: formato inválido (H12 cobre)", "abc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{DataBase: tt.dataBase}}
			err := S29DataBaseRangePlausivel{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestS30_DiarioPresenteSeModelo1a4(t *testing.T) {
	doc := &Doc3050{} // vazio
	err := S30DiarioPresenteSeModelo1a4{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "vazio")

	doc = &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", VlrConcessoes: ptrF(100.0)}}}
	err = S30DiarioPresenteSeModelo1a4{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestS31_SubstituicaoSemAnteriorRef_StubReturnsNil(t *testing.T) {
	doc := &Doc3050{Root: Doc3050Root{IndRemessa: "S"}}
	err := S31SubstituicaoSemAnteriorRef{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "") // stub honesto
}

func TestS32_DocNaoVazio(t *testing.T) {
	doc := &Doc3050{}
	err := S32DocNaoVazio{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "vazio")

	doc = &Doc3050{Mensal: []Modalidade{{Codigo: "capGirPrzAte365"}}}
	err = S32DocNaoVazio{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

// ========== S09/S13/S24 — Carry-over (saem de stub) ==========

func TestS09_DiasUteis_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: 2024-12-30 (segunda útil)", "2024-12-30", ""},
		{"violação: 2024-12-25 (Natal)", "2024-12-25", "não é dia útil BACEN"},
		{"violação: 2024-12-21 (sábado)", "2024-12-21", "não é dia útil BACEN"},
		{"violação: 2024-12-22 (domingo)", "2024-12-22", "não é dia útil BACEN"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{DataBase: tt.dataBase}}
			err := S09DiasUteis{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestS13_UltimoDiaUtil_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: 2024-04-30 (terça, último útil de abril/2024)", "2024-04-30", ""},
		{"violação: 2024-04-29 (segunda, mas 30 também é útil)", "2024-04-29", "não é último dia útil"},
		{"violação: 2024-12-31 (terça, último útil de dez/2024)", "2024-12-31", ""},
		{"violação: 2024-12-25 (Natal, não é último útil)", "2024-12-25", "não é último dia útil"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{DataBase: tt.dataBase}}
			err := S13UltimoDiaUtil{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestS24_TxJurosAjustadaLeTxJuros_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		txMed       *float64
		txAjustada  *float64
		wantErrSubs string
	}{
		{"happy: ajustada < txMed", ptrF(15.0), ptrF(14.5), ""},
		{"happy: ajustada == txMed", ptrF(15.0), ptrF(15.0), ""},
		{"violação: ajustada > txMed", ptrF(15.0), ptrF(16.0), "> txMedJuros"},
		{"skip: ajustada nil", ptrF(15.0), nil, ""},
		{"skip: txMed nil", nil, ptrF(15.0), ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica",
				TxMedJuros: tt.txMed, TxMedJurosAjustada: tt.txAjustada}}}
			err := S24TxJurosAjustadaLeTxJuros{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== Helpers (IsDiaUtilBACEN, IsUltimoDiaUtilMes) ==========

func TestIsDiaUtilBACEN(t *testing.T) {
	tests := []struct {
		data string
		util bool
		desc string
	}{
		{"2024-12-25", false, "Natal"},
		{"2024-01-01", false, "Confraternização"},
		{"2024-12-21", false, "Sábado"},
		{"2024-12-22", false, "Domingo"},
		{"2024-12-23", true, "Segunda comum"},
		{"2024-12-30", true, "Segunda comum"},
		{"2024-04-21", false, "Tiradentes"},
		{"2024-05-01", false, "Dia do Trabalho"},
		{"2024-09-07", false, "Independência"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			data, _ := time.Parse("2006-01-02", tt.data)
			if got := IsDiaUtilBACEN(data); got != tt.util {
				t.Errorf("IsDiaUtilBACEN(%s)=%v, want %v", tt.data, got, tt.util)
			}
		})
	}
}

func TestIsUltimoDiaUtilMes(t *testing.T) {
	tests := []struct {
		data   string
		ultimo bool
		desc   string
	}{
		{"2024-04-30", true, "Abril/2024 termina terça 30"},
		{"2024-12-31", true, "Dez/2024 termina terça 31"},
		{"2024-04-29", false, "29/abr não é último (30 também é útil)"},
		{"2024-04-26", false, "26/abr (sexta) não é último"},
		{"2024-02-29", true, "Fev/2024 bissexto, último dia 29"},
		{"2023-02-28", true, "Fev/2023 não-bissexto, último dia 28"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			data, _ := time.Parse("2006-01-02", tt.data)
			if got := IsUltimoDiaUtilMes(data); got != tt.ultimo {
				t.Errorf("IsUltimoDiaUtilMes(%s)=%v, want %v", tt.data, got, tt.ultimo)
			}
		})
	}
}

// ========== Integração ==========

func TestBuiltin3050_Fase3TotalRulesIs80(t *testing.T) {
	r := Builtin3050()
	got := len(r.All3050())
	if got != 80 {
		t.Fatalf("Builtin3050 deveria ter 80 regras após Fase 3, got %d", got)
	}
	// Confere Fase 3: H10-H15 (6) + S29-S32 (4) + I15-I28 (14) = 24.
	fase3Count := 0
	for _, code := range r.Codes3050() {
		if len(code) < 7 {
			continue
		}
		prefix := code[:6]
		num := code[6:]
		switch prefix {
		case "3050-H":
			if num >= "10" && num <= "15" {
				fase3Count++
			}
		case "3050-S":
			if num >= "29" && num <= "32" {
				fase3Count++
			}
		case "3050-I":
			if num >= "15" && num <= "28" {
				fase3Count++
			}
		}
	}
	if fase3Count != 24 {
		t.Errorf("esperado 24 regras Fase 3 (H10-H15 + S29-S32 + I15-I28), got %d", fase3Count)
	}
}
