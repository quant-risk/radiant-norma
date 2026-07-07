// Testes Sprint 39 Fase 2 — AuditDDR cross-doc DRM/DLO.
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint39_ParseDocDRM(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocDRM cnpj="12345678" dataBase="2024-12-31">
  <RWAJUR1 valor="100.00"/>
  <RWAJUR2 valor="50.00"/>
  <RWAJUR3 valor="30.00"/>
  <RWAJUR4 valor="20.00"/>
  <VaR valor="40.00"/>
  <sVaR valor="60.00"/>
  <RWACOM valor="80.00"/>
  <Posicao codigo="161000" moeda="USD" valor="100.00"/>
  <Posicao codigo="181000" moeda="USD" valor="50.00"/>
</DocDRM>`

	doc, err := ParseDocDRM([]byte(xml))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.Root.CNPJ != "12345678" {
		t.Errorf("CNPJ errado: %s", doc.Root.CNPJ)
	}
	if doc.VaR != 40.0 {
		t.Errorf("VaR errado: %v", doc.VaR)
	}
	if doc.sVaR != 60.0 {
		t.Errorf("sVaR errado: %v", doc.sVaR)
	}
	if len(doc.Posicoes) != 2 {
		t.Errorf("esperado 2 posições, got %d", len(doc.Posicoes))
	}
}

func TestSprint39_ParseDocDLO(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocDLO cnpj="12345678" dataBase="2024-12-31">
  <Conta770 valor="1000.00"/>
  <LimiteTotal valor="5000.00"/>
  <Patrimonio valor="3000.00"/>
</DocDLO>`

	doc, err := ParseDocDLO([]byte(xml))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.Conta770 != 1000.0 {
		t.Errorf("Conta770 errado: %v", doc.Conta770)
	}
	if doc.Patrimonio != 3000.0 {
		t.Errorf("Patrimonio errado: %v", doc.Patrimonio)
	}
}

func TestSprint39_ValidarBasicos(t *testing.T) {
	t.Run("ValidarDRMBasico_OK", func(t *testing.T) {
		doc := &DocDRM{VaR: 40, sVaR: 60, RWACOM: 80}
		if err := ValidarDRMBasico(doc); err != nil {
			t.Errorf("esperava OK, got %v", err)
		}
	})

	t.Run("ValidarDRMBasico_VarMaiorSVaR", func(t *testing.T) {
		doc := &DocDRM{VaR: 100, sVaR: 50}
		err := ValidarDRMBasico(doc)
		if err == nil || !strings.Contains(err.Error(), "VaR=") {
			t.Errorf("esperava erro VaR > sVaR, got %v", err)
		}
	})

	t.Run("ValidarDLOBasico_OK", func(t *testing.T) {
		doc := &DocDLO{Conta770: 1000, LimiteTotal: 5000, Patrimonio: 3000}
		if err := ValidarDLOBasico(doc); err != nil {
			t.Errorf("esperava OK, got %v", err)
		}
	})

	t.Run("ValidarDLOBasico_ContaMaiorLimite", func(t *testing.T) {
		doc := &DocDLO{Conta770: 6000, LimiteTotal: 5000, Patrimonio: 3000}
		err := ValidarDLOBasico(doc)
		if err == nil || !strings.Contains(err.Error(), "> LimiteTotal") {
			t.Errorf("esperava erro Conta770 > LimiteTotal, got %v", err)
		}
	})
}

func TestSprint39_CrossDocRegras(t *testing.T) {
	ctx := context.Background()

	t.Run("C4693CrossDoc_DLOPresente", func(t *testing.T) {
		doc2070 := &Doc2070{
			DDRs: []DDR{
				{Codigo: "161000", Moeda: "BRL", Valor: floatPtr(1000)},
				{Codigo: "181000", Moeda: "BRL", Valor: floatPtr(500)},
			},
		}
		parsedDLO = &DocDLO{Patrimonio: 1500}
		err := C4693CrossDocPatrimonioLiquido{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR soma 1500 == DLO 1500 OK, got %v", err)
		}
	})

	t.Run("C4678CrossDoc_DRMPresente", func(t *testing.T) {
		doc2070 := &Doc2070{
			DDRs: []DDR{
				{Codigo: "467821", Moeda: "BRL", Valor: floatPtr(100)},
				{Codigo: "467822", Moeda: "BRL", Valor: floatPtr(50)},
			},
		}
		parsedDRM = &DocDRM{RWAJUR2: 80, RWAJUR3: 30, RWAJUR4: 40} // soma = 150
		err := C4678CrossDocExposicaoLiquida{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR 150 == DRM 150 OK, got %v", err)
		}
	})

	t.Run("C4686CrossDoc_PosicaoSemDDR", func(t *testing.T) {
		doc2070 := &Doc2070{
			DDRs: []DDR{},
		}
		parsedDRM = &DocDRM{
			Posicoes: []PosicaoMoeda{
				{Codigo: "161000", Moeda: "USD", Valor: 100},
			},
		}
		err := C4686CrossDocPosicoesMoedas{}.Apply2070(ctx, doc2070)
		if err == nil || !strings.Contains(err.Error(), "sem contraparte DDR") {
			t.Errorf("esperava erro posição sem DDR, got %v", err)
		}
	})

	t.Run("C4763CrossDoc_Saldo770_Discrepancia", func(t *testing.T) {
		doc2070 := &Doc2070{
			DDRs: []DDR{
				{Codigo: "770", Moeda: "BRL", Valor: floatPtr(1500)},
			},
		}
		parsedDLO = &DocDLO{Conta770: 1000}
		err := C4763CrossDocSaldo770DLO{}.Apply2070(ctx, doc2070)
		if err == nil || !strings.Contains(err.Error(), "discrepância") {
			t.Errorf("esperava erro discrepância 770, got %v", err)
		}
	})

	t.Run("C4763CrossDoc_Saldo770_OK", func(t *testing.T) {
		doc2070 := &Doc2070{
			DDRs: []DDR{
				{Codigo: "770", Moeda: "BRL", Valor: floatPtr(1000)},
			},
		}
		parsedDLO = &DocDLO{Conta770: 1000}
		err := C4763CrossDocSaldo770DLO{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR 770 = DLO 1000 OK, got %v", err)
		}
	})
}

func floatPtr(v float64) *float64 {
	return &v
}

// V70 — testes para regras Sprint 39 consertadas (stubs disfarçados).
func TestSprint39_V70_CrossDocReais(t *testing.T) {
	ctx := context.Background()

	t.Run("C4679_Descasamento_Fail", func(t *testing.T) {
		// DRM reporta RWAJUR1 mas DDR não tem descasamento vertical.
		doc2070 := &Doc2070{DDRs: []DDR{}}
		parsedDRM = &DocDRM{RWAJUR1: 100}
		err := C4679CrossDocDescasamentoVertical{}.Apply2070(ctx, doc2070)
		if err == nil || !strings.Contains(err.Error(), "descasamento vertical") {
			t.Errorf("esperava erro descasamento, got %v", err)
		}
	})

	t.Run("C4679_Descasamento_OK", func(t *testing.T) {
		// DDR tem 46791.
		doc2070 := &Doc2070{DDRs: []DDR{{Codigo: "46791", Moeda: "BRL", Valor: floatPtr(100)}}}
		parsedDRM = &DocDRM{RWAJUR1: 100}
		err := C4679CrossDocDescasamentoVertical{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR 46791 presente OK, got %v", err)
		}
	})

	t.Run("C4684_VaR_Fail", func(t *testing.T) {
		doc2070 := &Doc2070{DDRs: []DDR{}}
		parsedDRM = &DocDRM{VaR: 50}
		err := C4684CrossDocVaR{}.Apply2070(ctx, doc2070)
		if err == nil || !strings.Contains(err.Error(), "VaR") {
			t.Errorf("esperava erro VaR, got %v", err)
		}
	})

	t.Run("C4684_VaR_OK", func(t *testing.T) {
		doc2070 := &Doc2070{DDRs: []DDR{{Codigo: "46841", Moeda: "BRL", Valor: floatPtr(50)}}}
		parsedDRM = &DocDRM{VaR: 50}
		err := C4684CrossDocVaR{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR 46841 presente OK, got %v", err)
		}
	})

	t.Run("C4685_sVaR_Fail", func(t *testing.T) {
		doc2070 := &Doc2070{DDRs: []DDR{}}
		parsedDRM = &DocDRM{sVaR: 80}
		err := C4685CrossDocsVaR{}.Apply2070(ctx, doc2070)
		if err == nil || !strings.Contains(err.Error(), "sVaR") {
			t.Errorf("esperava erro sVaR, got %v", err)
		}
	})

	t.Run("C4685_sVaR_OK", func(t *testing.T) {
		doc2070 := &Doc2070{DDRs: []DDR{{Codigo: "46851", Moeda: "BRL", Valor: floatPtr(80)}}}
		parsedDRM = &DocDRM{sVaR: 80}
		err := C4685CrossDocsVaR{}.Apply2070(ctx, doc2070)
		if err != nil {
			t.Errorf("DDR 46851 presente OK, got %v", err)
		}
	})
}
