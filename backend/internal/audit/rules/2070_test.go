// Tests DDR 2070 — Sprint 35 Fase 1 (v3.34.12).
//
// Cobre:
// - C4693 Patrimônio Líquido Exterior (real)
// - C4751 Chaves Duplicadas (real)
// - C4678-C4686, C4763 (9 stubs cross-doc)
// - Integração Builtin2070 (11 regras)
//
// Total: ~10 funções de teste.
package rules

import (
	"context"
	"testing"
)

// ========== C4693 (Patrimônio Líquido Exterior — real) ==========

func TestC4693_PatrimonioLiquidoExterior_Real(t *testing.T) {
	tests := []struct {
		nome        string
		ddrs        []DDR
		wantErrSubs string
	}{
		{
			nome: "happy: 161000 >= 181000",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(1000.0)},
				{Codigo: "181000", Moeda: "USD", Valor: ptrF(500.0)},
			},
			wantErrSubs: "",
		},
		{
			nome: "happy: 161000 == 181000",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(500.0)},
				{Codigo: "181000", Moeda: "USD", Valor: ptrF(500.0)},
			},
			wantErrSubs: "",
		},
		{
			nome: "violação: 161000 < 181000",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(100.0)},
				{Codigo: "181000", Moeda: "USD", Valor: ptrF(500.0)},
			},
			wantErrSubs: "161000 (100.00) < soma 181000 (500.00)",
		},
		{
			nome: "skip: sem 181000",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(1000.0)},
			},
			wantErrSubs: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc2070{DDRs: tt.ddrs}
			err := C4693PatrimonioLiquidoExterior{}.Apply2070(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== C4751 (Chaves Duplicadas — real) ==========

func TestC4751_ChavesDuplicadas_Real(t *testing.T) {
	tests := []struct {
		nome        string
		ddrs        []DDR
		wantErrSubs string
	}{
		{
			nome: "happy: 1 chave única",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(100.0)},
			},
			wantErrSubs: "",
		},
		{
			nome: "happy: chaves distintas",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(100.0)},
				{Codigo: "161000", Moeda: "EUR", Valor: ptrF(200.0)},
				{Codigo: "181000", Moeda: "USD", Valor: ptrF(50.0)},
			},
			wantErrSubs: "",
		},
		{
			nome: "violação: 2x mesma chave",
			ddrs: []DDR{
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(100.0)},
				{Codigo: "161000", Moeda: "USD", Valor: ptrF(200.0)},
			},
			wantErrSubs: "chave duplicada 2 vezes: 161000|USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc2070{DDRs: tt.ddrs}
			err := C4751ChavesDuplicadas{}.Apply2070(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== C4678-C4686, C4763 (9 stubs cross-doc) ==========

func TestC4678_C4763_CrossDocStubs(t *testing.T) {
	rules := []struct {
		code  string
		apply func(context.Context, *Doc2070) error
		sev   string
	}{
		{"2070-4678", C4678ExposicaoLiquida{}.Apply2070, "I"},
		{"2070-4679", C4679DescasamentoVertical{}.Apply2070, "I"},
		{"2070-4680", C4680DescasamentoHorizontalDentroZona{}.Apply2070, "I"},
		{"2070-4681", C4681DescasamentoHorizontalEntreZonas{}.Apply2070, "I"},
		{"2070-4682", C4682ExposicaoBrutaRWACOM{}.Apply2070, "I"},
		{"2070-4684", C4684VaRRWAJUR1{}.Apply2070, "I"},
		{"2070-4685", C4685sVaRRWAJUR1{}.Apply2070, "I"},
		{"2070-4686", C4686PosicoesMoedas{}.Apply2070, "I"},
		{"2070-4763", C4763SaldoConta770DLO{}.Apply2070, "I"},
	}

	for _, r := range rules {
		t.Run(r.code, func(t *testing.T) {
			doc := &Doc2070{}
			if err := r.apply(context.Background(), doc); err != nil {
				t.Errorf("stub %s deveria retornar nil, got %v", r.code, err)
			}
		})
	}
}

// ========== Parser (DT-38) ==========

func TestParseDoc2070_Smoke(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<DocDDR cnpj="12345678" dataBase="2024-12-31" indRemessa="I" nmContato="João" telContato="11999998888">
  <DDR codigo="161000" moeda="USD" valor="1000.00"/>
  <DDR codigo="181000" moeda="USD" valor="500.00"/>
</DocDDR>`

	doc, err := ParseDoc2070([]byte(xmlInput))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.Root.CNPJ != "12345678" {
		t.Errorf("CNPJ=%q, want 12345678", doc.Root.CNPJ)
	}
	if len(doc.DDRs) != 2 {
		t.Errorf("DDRs count=%d, want 2", len(doc.DDRs))
	}
	if doc.DDRs[0].Codigo != "161000" || *doc.DDRs[0].Valor != 1000.0 {
		t.Errorf("DDR[0] = %+v, want {161000 USD 1000.0}", doc.DDRs[0])
	}
}

func TestParseDoc2070_DocumentoVazio(t *testing.T) {
	_, err := ParseDoc2070([]byte(""))
	if err == nil {
		t.Error("esperava erro para documento vazio")
	}
}

// ========== Integração ==========

func TestBuiltin2070_TotalRulesIs11(t *testing.T) {
	r := Builtin2070()
	got := len(r.All2070())
	if got != 11 {
		t.Fatalf("Builtin2070 deveria ter 11 regras, got %d", got)
	}

	codes := r.Codes2070()
	expectedCodes := []string{
		"2070-4678", "2070-4679", "2070-4680", "2070-4681", "2070-4682",
		"2070-4684", "2070-4685", "2070-4686", "2070-4763",
		"2070-4693", "2070-4751",
	}
	for _, code := range expectedCodes {
		if r.Get2070(code) == nil {
			t.Errorf("regra %s não encontrada no registry", code)
		}
	}
	_ = codes // silencia unused
}
