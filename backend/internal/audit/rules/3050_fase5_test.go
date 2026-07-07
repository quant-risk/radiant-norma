// Tests Fase 5 — Sprint 33 Fase 5 (v3.34.8).
//
// Cobre:
// - I37-I50 Individuais (14 regras — sub-modalidades ≥ 0)
// - S39-S70 Sistema (32 stubs informativos matriz modalidade × encargo)
// - H21-H30 Header (10 regras — decimais + consolidações + caracteres)
// - Carry-over implícito (não tem stub separado)
//
// Total: ~25 funções de teste (smoke + casos reais pra regras com lógica).
package rules

import (
	"context"
	"testing"
)

// ========== I37-I50 — Individuais sub-modalidades restantes ==========

func TestI37_CredLivreVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "credLivre", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-100.0)}}}
	err := I37CredLivreVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI38_CredConsignadoVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "credConsignado", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-50.0)}}}
	err := I38CredConsignadoVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI39_CredDirecionadoVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "credDirecionado", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-200.0)}}}
	err := I39CredDirecionadoVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI40_ImobResidVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "imobResid", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-1000.0)}}}
	err := I40ImobResidVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI41_ImobComercVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "imobComerc", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-5000.0)}}}
	err := I41ImobComercVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI42_FinancMicroCredVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financMicroCred", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-10.0)}}}
	err := I42FinancMicroCredVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI43_FinancInfraVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financInfra", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-100000.0)}}}
	err := I43FinancInfraVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI44_FinancRuralCusteioVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financRuralCusteio", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-1000.0)}}}
	err := I44FinancRuralCusteioVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI45_FinancRuralInvestVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financRuralInvest", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-2000.0)}}}
	err := I45FinancRuralInvestVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI46_FinancRuralComercVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financRuralComerc", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-500.0)}}}
	err := I46FinancRuralComercVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI47_CoopCentraisVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "coopCentrais", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-100.0)}}}
	err := I47CoopCentraisVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI48_CoopSingularesVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "coopSingulares", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-50.0)}}}
	err := I48CoopSingularesVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI49_DescTitulosAdquiridosVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "descTitulosAdquiridos", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-1000.0)}}}
	err := I49DescTitulosAdquiridosVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI50_AntecipacaoFaturasVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "antecipacaoFaturas", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-500.0)}}}
	err := I50AntecipacaoFaturasVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

// ========== S39-S70 — Matriz modalidade × encargo (stubs informativos) ==========

// Verifica que cada stub tem Code/Sheet/Severity corretos e Apply retorna nil.
func TestS39_S70_MatrizStubs(t *testing.T) {
	rules := []struct {
		code  string
		apply func(context.Context, *Doc3050) error
		sev   string
		sheet string
	}{
		{"3050-S39", S39CapGirApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S40", S40ContaGarantidaApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S41", S41ChequeEspecialApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S42", S42DescontoDuplicatasApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S43", S43DescontoChequesApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S44", S44AntecipFaturaCartaoApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S45", S45AquisicaoVeiculosApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S46", S46ArrendMercantilApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S47", S47CapGirAte365BloqIPCA{}.Apply3050, "I", "Matriz"},
		{"3050-S48", S48CapGirSup365BloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S49", S49CapGirTetoRotBloqIPCA{}.Apply3050, "I", "Matriz"},
		{"3050-S50", S50ContaGarantidaBloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S51", S51ChequeEspecialBloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S52", S52AquisicaoVeiculosBloqPosFix{}.Apply3050, "I", "Matriz"},
		{"3050-S53", S53ArrendMercantilBloqPosFix{}.Apply3050, "I", "Matriz"},
		{"3050-S54", S54FinancBensBloqPosFix{}.Apply3050, "I", "Matriz"},
		{"3050-S55", S55FinancRuralApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S56", S56FinancImobApenasPref{}.Apply3050, "I", "Matriz"},
		{"3050-S57", S57DataBaseFimMesBACEN{}.Apply3050, "I", "Periodicidade"},
		{"3050-S58", S58PeriodicidadeDiariaBACEN{}.Apply3050, "I", "Periodicidade"},
		{"3050-S59", S59PeriodicidadeMensalBACEN{}.Apply3050, "I", "Periodicidade"},
		{"3050-S60", S60DataBaseJanelaUtilMes{}.Apply3050, "I", "Periodicidade"},
		{"3050-S61", S61DesDuplicatasConsolidado{}.Apply3050, "I", "Matriz"},
		{"3050-S62", S62DesChequesConsolidado{}.Apply3050, "I", "Matriz"},
		{"3050-S63", S63AntecipFaturaCartaoConsolidado{}.Apply3050, "I", "Matriz"},
		{"3050-S64", S64CapGirAte365BloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S65", S65CapGirSup365BloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S66", S66CapGirTetoRotBloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S67", S67CtgGtaBloqIPCA{}.Apply3050, "I", "Matriz"},
		{"3050-S68", S68ChqEspBloqMoedaEstrangeira{}.Apply3050, "I", "Matriz"},
		{"3050-S69", S69CcbConsolidado{}.Apply3050, "I", "Matriz"},
		{"3050-S70", S70FinancBensConsolidado{}.Apply3050, "I", "Matriz"},
	}

	for _, r := range rules {
		t.Run(r.code, func(t *testing.T) {
			doc := &Doc3050{}
			if err := r.apply(context.Background(), doc); err != nil {
				t.Errorf("stub %s deveria retornar nil, got %v", r.code, err)
			}
		})
	}
}

// ========== H21-H30 — Header adicionais ==========

func TestH25_NmContatoSemControle(t *testing.T) {
	tests := []struct {
		nome        string
		nm          string
		wantErrSubs string
	}{
		{"happy: nome normal", "João da Silva", ""},
		{"happy: tab permitido", "João\tSilva", ""},
		{"violação: newline", "João\nSilva", "caractere de controle"},
		{"violação: null", "João\x00Silva", "caractere de controle"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{NmContato: tt.nm}}
			err := H25NmContatoSemControle{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH30_CNPJSemZerosEsquerda(t *testing.T) {
	tests := []struct {
		nome        string
		cnpj        string
		wantErrSubs string
	}{
		{"happy: 12345678", "12345678", ""},
		{"violação: 01234567", "01234567", "zeros à esquerda"},
		{"happy: vazio", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{CNPJ: tt.cnpj}}
			err := H30CNPJSemZerosEsquerda{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== H21-H22 — Decimais (implementação real, Fase 5 validação) ==========

func TestH21_TxMedJurosMax4Decimals(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 15.5 (1 decimal)", ptrF(15.5), ""},
		{"happy: 15.1234 (4 decimais)", ptrF(15.1234), ""},
		{"violação: 15.12345 (5 decimais)", ptrF(15.12345), "5 decimais"},
		{"violação: 15.000001 (6 decimais)", ptrF(15.000001), "6 decimais"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "credLivre", Encargo: "pre", TipoCli: "pesJuridica", TxMedJuros: tt.val}}}
			err := H21TxMedJurosMax4Decimals{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH22_VlrConcessoesMax2Decimals(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 1000.00 (2 decimais)", ptrF(1000.00), ""},
		{"happy: 1000.5 (1 decimal)", ptrF(1000.5), ""},
		{"violação: 1000.123 (3 decimais)", ptrF(1000.123), "3 decimais"},
		{"violação: 100.0001 (4 decimais)", ptrF(100.0001), "4 decimais"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "credLivre", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: tt.val}}}
			err := H22VlrConcessoesMax2Decimals{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== H21-H30 stubs consolidações + decimais ==========

func TestH23_H29_StubsReasonable(t *testing.T) {
	// Verifica que H23-H29 (excluindo H21/H22 com lógica real) não panicam e retornam nil.
	rules := []func(context.Context, *Doc3050) error{
		H23QtdNovContratosInteiro{}.Apply3050,
		H24CNPJConsolidado{}.Apply3050,
		H26TelContatoConsolidado{}.Apply3050,
		H27DeclaracaoEncodingPresente{}.Apply3050,
		H28NamespaceXSD3050{}.Apply3050,
		H29IndRemessaConsolidado{}.Apply3050,
	}

	doc := &Doc3050{
		Root:   Doc3050Root{CNPJ: "12345678", DataBase: "2024-12-31", IndRemessa: "I", NmContato: "João", TelContato: "11999998888"},
		Diario: []Modalidade{{Codigo: "credLivre", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(1000.0), TxMedJuros: ptrF(15.5)}},
	}
	for i, apply := range rules {
		if err := apply(context.Background(), doc); err != nil {
			t.Errorf("rule[%d] deveria retornar nil para doc típico, got %v", i, err)
		}
	}
}

// ========== Integração ==========

func TestBuiltin3050_Fase5TotalRulesIs153(t *testing.T) {
	r := Builtin3050()
	got := len(r.All3050())
	if got != 153 {
		t.Fatalf("Builtin3050 deveria ter 153 regras após Fase 5, got %d", got)
	}
	// Confere Fase 5: I37-I50 (14) + S39-S70 (32) + H21-H30 (10) = 56.
	fase5Count := 0
	for _, code := range r.Codes3050() {
		if len(code) < 7 {
			continue
		}
		prefix := code[:6]
		num := code[6:]
		switch prefix {
		case "3050-H":
			if num >= "21" && num <= "30" {
				fase5Count++
			}
		case "3050-S":
			if num >= "39" && num <= "70" {
				fase5Count++
			}
		case "3050-I":
			if num >= "37" && num <= "50" {
				fase5Count++
			}
		}
	}
	if fase5Count != 56 {
		t.Errorf("esperado 56 regras Fase 5 (H21-H30 + S39-S70 + I37-I50), got %d", fase5Count)
	}
}
