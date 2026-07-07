// Tests Fase 6 — Sprint 34 (v3.34.10) — fechar 3050 em 100%.
//
// Cobre:
// - S12 (PrzMedSeSld) — implementação real
// - S14 (Cruzadas 3055) — implementação real
// - H19 (ApenasUmaReferencia) — implementação real via RawXML
// - H20 (ApenasUmDiarioUmMensal) — implementação real via RawXML
// - S71-S87 (17 stubs matriz adicionais) — smoke tests
// - Integração Fase 6 (170 total)
//
// Total: ~10 funções de teste.
package rules

import (
	"context"
	"testing"
)

// ========== S12 (lógica pura) ==========

func TestS12_PrzMedSeSld_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		sldBai      *float64
		przMed      *int
		wantErrSubs string
	}{
		{"happy: sldBai=0 sem przMed", ptrF(0.0), nil, ""},
		{"happy: sldBai>0 com przMed", ptrF(100.0), ptrI(180), ""},
		{"violação: sldBai>0 sem przMed", ptrF(100.0), nil, "przMedCarteira ausente"},
		{"skip: sldBai nil", nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Mensal: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica",
				SldBaiPrejuizo: tt.sldBai, PrzMedCarteira: tt.przMed}}}
			err := S12PrzMedSeSld{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S14 (cruzadas 3055) ==========

func TestS14_Cruzadas_TxMaxGtTxMin_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		txMin       *float64
		txMax       *float64
		wantErrSubs string
	}{
		{"happy: txMax > txMin", ptrF(10.0), ptrF(20.0), ""},
		{"violação: txMax == txMin (inconsistência — taxa max deve ser > min)", ptrF(10.0), ptrF(10.0), "txMaxima=10.00 <= txMinima=10.00"},
		{"violação: txMax < txMin", ptrF(20.0), ptrF(10.0), "txMaxima=10.00 <= txMinima=20.00"},
		{"skip: txMin nil", nil, ptrF(20.0), ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "capGirPrzAte365", Encargo: "pre", TipoCli: "pesJuridica",
				TxMinima: tt.txMin, TxMaxima: tt.txMax}}}
			err := S14Cruzadas{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== H19/H20 (parser RawXML) ==========

func TestH19_ApenasUmaReferencia_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		xmlInput    string
		wantErrSubs string
	}{
		{"happy: 1 referencia", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia></referencia></DocTXB>`, ""},
		{"violação: 2 referencias", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia/><referencia/></DocTXB>`, "2 elementos <referencia>"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc, err := ParseDoc3050([]byte(tt.xmlInput))
			if err != nil {
				t.Fatalf("parse falhou: %v", err)
			}
			applyErr := H19ApenasUmaReferencia{}.Apply3050(context.Background(), doc)
			checkErr(t, applyErr, tt.wantErrSubs)
		})
	}
}

func TestH20_ApenasUmDiarioUmMensal_RealImplementation(t *testing.T) {
	tests := []struct {
		nome        string
		xmlInput    string
		wantErrSubs string
	}{
		{"happy: 1 diario + 1 mensal", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia><diario/><mensal/></referencia></DocTXB>`, ""},
		{"violação: 2 diarios", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia><diario/><diario/></referencia></DocTXB>`, "2 elementos <diario>"},
		{"violação: 2 mensais", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia><diario/><mensal/><mensal/></referencia></DocTXB>`, "2 elementos <mensal>"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc, err := ParseDoc3050([]byte(tt.xmlInput))
			if err != nil {
				t.Fatalf("parse falhou: %v", err)
			}
			applyErr := H20ApenasUmDiarioUmMensal{}.Apply3050(context.Background(), doc)
			checkErr(t, applyErr, tt.wantErrSubs)
		})
	}
}

// ========== S71-S87 (17 stubs matriz adicionais) ==========

func TestS71_S87_MatrizStubsAdicionais(t *testing.T) {
	rules := []struct {
		code  string
		apply func(context.Context, *Doc3050) error
		sev   string
	}{
		{"3050-S71", S71FinancBensVeiculosApenasPref{}.Apply3050, "I"},
		{"3050-S72", S72ArrendamentoVeiculosApenasPref{}.Apply3050, "I"},
		{"3050-S73", S73LeasingVeiculosApenasPref{}.Apply3050, "I"},
		{"3050-S74", S74CredConsigFuncPublicoApenasPref{}.Apply3050, "I"},
		{"3050-S75", S75CredRuralConsolidado{}.Apply3050, "I"},
		{"3050-S76", S76MicroFinancConsolidado{}.Apply3050, "I"},
		{"3050-S77", S77CapGirTetoRotIPCA{}.Apply3050, "I"},
		{"3050-S78", S78ImobConsolidado{}.Apply3050, "I"},
		{"3050-S79", S79FinancImobReforma{}.Apply3050, "I"},
		{"3050-S80", S80ChqEspBloqDat{}.Apply3050, "I"},
		{"3050-S81", S81GarantiasConsolidado{}.Apply3050, "I"},
		{"3050-S82", S82MatrizConsolidadaFinal{}.Apply3050, "I"},
		{"3050-S83", S83PeriodicidadeAnualBACEN{}.Apply3050, "I"},
		{"3050-S84", S84MatrizEncargoConsolidado1{}.Apply3050, "I"},
		{"3050-S85", S85MatrizModalidadeConsolidado2{}.Apply3050, "I"},
		{"3050-S86", S86PeriodicidadeQuinzenalBACEN{}.Apply3050, "I"},
		{"3050-S87", S87MatrizFechamentoFinal{}.Apply3050, "I"},
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

// ========== Integração Fase 6 ==========

func TestBuiltin3050_Fase6TotalRulesIs170(t *testing.T) {
	r := Builtin3050()
	got := len(r.All3050())
	if got != 170 {
		t.Fatalf("Builtin3050 deveria ter 170 regras após Fase 6 (100%% catálogo), got %d", got)
	}
	// Confere Fase 6: S12, S14 (substituições — não somam), S71-S87 (17 novos) + H19, H20 (substituições — não somam).
	// Verifica apenas S71-S87 = 17.
	fase6Count := 0
	for _, code := range r.Codes3050() {
		if len(code) < 7 {
			continue
		}
		prefix := code[:6]
		num := code[6:]
		switch prefix {
		case "3050-S":
			if num >= "71" && num <= "87" {
				fase6Count++
			}
		}
	}
	if fase6Count != 17 {
		t.Errorf("esperado 17 regras novas Fase 6 (S71-S87), got %d", fase6Count)
	}
}
