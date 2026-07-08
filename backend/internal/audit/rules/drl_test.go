// Testes Sprint 51 — AuditDRL 2160 (LCR) com estrutura hierárquica COSIF.
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint51_ParseDocDRL(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocDRL cnpj="12345678" dataBase="2024-12-31">
  <HQLA valor="1000.00"/>
  <Outflows valor="500.00"/>
  <Inflows valor="200.00"/>
  <LCRRatio valor="333.33"/>
  <Conta codigoConta="10.1" valor="100.00"/>
  <Conta codigoConta="10.3.1" valor="500.00"/>
  <Conta codigoConta="20.1" valor="200.00"/>
  <Cenario id="1">
    <HQLA valor="900.00"/>
    <Outflows valor="550.00"/>
    <Inflows valor="180.00"/>
    <LCRRatio valor="243.24"/>
  </Cenario>
  <Cenario id="2">
    <HQLA valor="800.00"/>
    <Outflows valor="600.00"/>
    <Inflows valor="150.00"/>
    <LCRRatio valor="177.78"/>
  </Cenario>
</DocDRL>`

	doc, err := ParseDocDRL([]byte(xml))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.HQLA != 1000.0 {
		t.Errorf("HQLA errado: %v", doc.HQLA)
	}
	if doc.Outflows != 500.0 {
		t.Errorf("Outflows errado: %v", doc.Outflows)
	}
	if doc.LCRRatio != 333.33 {
		t.Errorf("LCRRatio errado: %v", doc.LCRRatio)
	}
	if doc.Cenario1.HQLA != 900.0 {
		t.Errorf("Cenario1.HQLA errado: %v", doc.Cenario1.HQLA)
	}
	if doc.Cenario2.LCRRatio != 177.78 {
		t.Errorf("Cenario2.LCRRatio errado: %v", doc.Cenario2.LCRRatio)
	}
	// Test hierarchical accounts
	if doc.Accounts["10.1"] != 100.0 {
		t.Errorf("Account 10.1 errado: %v", doc.Accounts["10.1"])
	}
	if doc.Accounts["20.1"] != 200.0 {
		t.Errorf("Account 20.1 errado: %v", doc.Accounts["20.1"])
	}
}

func TestSprint51_SomaHQLA(t *testing.T) {
	doc := &DocDRL{}
	doc.Accounts = map[string]float64{
		"10.1":   100,
		"10.3.1": 500,
		"11.1":   200,
	}
	soma := doc.SomaHQLA()
	if soma != 800 {
		t.Errorf("SomaHQLA: esperado 800, got %v", soma)
	}
}

func TestSprint51_SomaOutflows(t *testing.T) {
	doc := &DocDRL{}
	doc.Accounts = map[string]float64{
		"20.1": 300,
		"21.1": 150,
	}
	soma := doc.SomaOutflows()
	if soma != 450 {
		t.Errorf("SomaOutflows: esperado 450, got %v", soma)
	}
}

func TestSprint51_CalcularLCRRatio(t *testing.T) {
	tests := []struct {
		hqla, outflows, inflows, want float64
	}{
		{1000, 500, 200, 333.33},
		{500, 400, 100, 166.66},
		{0, 0, 0, -1},
		{1000, 500, 500, -1},
	}
	for _, tt := range tests {
		got := CalcularLCRRatio(tt.hqla, tt.outflows, tt.inflows)
		if got < 0 && tt.want >= 0 {
			t.Errorf("CalcularLCRRatio(%v,%v,%v)=%v, want %v", tt.hqla, tt.outflows, tt.inflows, got, tt.want)
		}
		if got >= 0 && (got < tt.want-0.5 || got > tt.want+0.5) {
			t.Errorf("CalcularLCRRatio(%v,%v,%v)=%v, want ~%v", tt.hqla, tt.outflows, tt.inflows, got, tt.want)
		}
	}
}

func TestSprint51_LCRRegras(t *testing.T) {
	ctx := context.Background()

	t.Run("LCR01_LCRAbaixoMinimo", func(t *testing.T) {
		doc := &DocDRL{LCRRatio: 80}
		err := LCR01{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "< 100%") {
			t.Errorf("esperava erro LCR < 100%%, got %v", err)
		}
	})

	t.Run("LCR01_LCROk", func(t *testing.T) {
		doc := &DocDRL{LCRRatio: 150}
		err := LCR01{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("LCR=150 OK, got %v", err)
		}
	})

	t.Run("LCR02_HQLANegativo", func(t *testing.T) {
		doc := &DocDRL{HQLA: -100}
		err := LCR02{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro HQLA negativo, got %v", err)
		}
	})

	t.Run("LCR03_OutflowsNegativo", func(t *testing.T) {
		doc := &DocDRL{Outflows: -50}
		err := LCR03{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Outflows negativo, got %v", err)
		}
	})

	t.Run("LCR04_InflowsMaiorOutflows", func(t *testing.T) {
		doc := &DocDRL{Inflows: 600, Outflows: 500}
		err := LCR04{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "inconsistência") {
			t.Errorf("esperava erro Inflows > Outflows, got %v", err)
		}
	})

	t.Run("LCR05_LCRDiscrepancia", func(t *testing.T) {
		doc := &DocDRL{HQLA: 1000, Outflows: 500, Inflows: 200, LCRRatio: 500}
		err := LCR05{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "discrepância") {
			t.Errorf("esperava erro discrepância LCR, got %v", err)
		}
	})

	t.Run("LCR05_LCRConsistente", func(t *testing.T) {
		doc := &DocDRL{HQLA: 1000, Outflows: 500, Inflows: 200, LCRRatio: 333.33}
		err := LCR05{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("LCR declarado=calculado OK, got %v", err)
		}
	})

	t.Run("LCR06_Cenario1Abaixo", func(t *testing.T) {
		doc := &DocDRL{Cenario1: CenarioLCR{LCRRatio: 80}}
		err := LCR06{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Cenário 1") {
			t.Errorf("esperava erro Cenário 1, got %v", err)
		}
	})

	t.Run("LCR07_Cenario2Abaixo", func(t *testing.T) {
		doc := &DocDRL{Cenario2: CenarioLCR{LCRRatio: 70}}
		err := LCR07{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Cenário 2") {
			t.Errorf("esperava erro Cenário 2, got %v", err)
		}
	})

	t.Run("LCR08_DtBaseInvalida", func(t *testing.T) {
		doc := &DocDRL{Root: DocDRLRoot{DataBase: "2024-12"}}
		err := LCR08{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("esperava erro DtBase formato, got %v", err)
		}
	})

	t.Run("LCR09_HQLAConsistente", func(t *testing.T) {
		doc := &DocDRL{
			HQLA: 1000,
			Accounts: map[string]float64{
				"10.1": 500,
				"10.2": 500,
			},
		}
		err := LCR09HQLAConsistente{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("HQLA consistente OK, got %v", err)
		}
	})

	t.Run("LCR09_HQLADivergente", func(t *testing.T) {
		doc := &DocDRL{
			HQLA: 1000,
			Accounts: map[string]float64{
				"10.1": 100,
				"10.2": 100,
			},
		}
		err := LCR09HQLAConsistente{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "diff=") {
			t.Errorf("esperava erro HQLA divergente, got %v", err)
		}
	})
}
