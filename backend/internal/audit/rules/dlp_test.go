// Testes Sprint 51 — AuditDLP 2170 (NSFR) com estrutura hierárquica COSIF.
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint51_ParseDocDLP(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocDLP cnpj="12345678" dataBase="2024-12-31">
  <ASFTotal valor="5000.00"/>
  <RSFTotal valor="4000.00"/>
  <NSFRRatio valor="125.00"/>
  <Conta codigoConta="30.01" valor="3000.00"/>
  <Conta codigoConta="30.02" valor="2000.00"/>
  <Conta codigoConta="40.01" valor="2000.00"/>
  <Cenario id="1">
    <ASF valor="4800.00"/>
    <RSF valor="4000.00"/>
    <NSFRRatioCenario valor="120.00"/>
  </Cenario>
  <Cenario id="2">
    <ASF valor="4500.00"/>
    <RSF valor="4000.00"/>
    <NSFRRatioCenario valor="112.50"/>
  </Cenario>
</DocDLP>`

	doc, err := ParseDocDLP([]byte(xml))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.ASFTotal != 5000.0 {
		t.Errorf("ASFTotal errado: %v", doc.ASFTotal)
	}
	if doc.RSFTotal != 4000.0 {
		t.Errorf("RSFTotal errado: %v", doc.RSFTotal)
	}
	if doc.NSFRRatio != 125.0 {
		t.Errorf("NSFRRatio errado: %v", doc.NSFRRatio)
	}
	if doc.Cenario1.ASF != 4800.0 {
		t.Errorf("Cenario1.ASF errado: %v", doc.Cenario1.ASF)
	}
	if doc.Cenario2.NSFRRatio != 112.5 {
		t.Errorf("Cenario2.NSFRRatio errado: %v", doc.Cenario2.NSFRRatio)
	}
	if doc.Accounts["30.01"] != 3000.0 {
		t.Errorf("Account 30.01 errado: %v", doc.Accounts["30.01"])
	}
	if doc.Accounts["40.01"] != 2000.0 {
		t.Errorf("Account 40.01 errado: %v", doc.Accounts["40.01"])
	}
}

func TestSprint51_SomaASF(t *testing.T) {
	doc := &DocDLP{}
	doc.Accounts = map[string]float64{
		"30.01": 3000,
		"30.02": 2000,
	}
	soma := doc.SomaASF()
	if soma != 5000 {
		t.Errorf("SomaASF: esperado 5000, got %v", soma)
	}
}

func TestSprint51_SomaRSF(t *testing.T) {
	doc := &DocDLP{}
	doc.Accounts = map[string]float64{
		"40.01": 2000,
		"40.02": 1500,
	}
	soma := doc.SomaRSF()
	if soma != 3500 {
		t.Errorf("SomaRSF: esperado 3500, got %v", soma)
	}
}

func TestSprint51_CalcularNSFRRatio(t *testing.T) {
	tests := []struct {
		asf, rsf, want float64
	}{
		{5000, 4000, 125.0},
		{4000, 4000, 100.0},
		{0, 0, -1},
		{1000, 0, -1},
		{6000, 5000, 120.0},
	}
	for _, tt := range tests {
		got := CalcularNSFRRatio(tt.asf, tt.rsf)
		if got < 0 && tt.want >= 0 {
			t.Errorf("CalcularNSFRRatio(%v,%v)=%v, want %v", tt.asf, tt.rsf, got, tt.want)
		}
		if got >= 0 && (got < tt.want-0.5 || got > tt.want+0.5) {
			t.Errorf("CalcularNSFRRatio(%v,%v)=%v, want ~%v", tt.asf, tt.rsf, got, tt.want)
		}
	}
}

func TestSprint51_NSFRRegras(t *testing.T) {
	ctx := context.Background()

	t.Run("NSFR01_NSFRABAIXO", func(t *testing.T) {
		doc := &DocDLP{NSFRRatio: 80}
		err := NSFR01{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "< 100%") {
			t.Errorf("esperava erro NSFR < 100%%, got %v", err)
		}
	})

	t.Run("NSFR01_nsfrOk", func(t *testing.T) {
		doc := &DocDLP{NSFRRatio: 150}
		err := NSFR01{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("NSFR=150 OK, got %v", err)
		}
	})

	t.Run("NSFR02_ASFNegativo", func(t *testing.T) {
		doc := &DocDLP{ASFTotal: -100}
		err := NSFR02{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro ASF negativo, got %v", err)
		}
	})

	t.Run("NSFR03_RSFNegativo", func(t *testing.T) {
		doc := &DocDLP{RSFTotal: -50}
		err := NSFR03{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro RSF negativo, got %v", err)
		}
	})

	t.Run("NSFR04_ASFMenorRSF", func(t *testing.T) {
		doc := &DocDLP{ASFTotal: 3000, RSFTotal: 4000}
		err := NSFR04{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "< RSF Total") {
			t.Errorf("esperava erro ASF < RSF, got %v", err)
		}
	})

	t.Run("NSFR04_ASFIgualRSF", func(t *testing.T) {
		doc := &DocDLP{ASFTotal: 4000, RSFTotal: 4000}
		err := NSFR04{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("ASF=RSF OK, got %v", err)
		}
	})

	t.Run("NSFR05_NSFRDiscrepancia", func(t *testing.T) {
		doc := &DocDLP{ASFTotal: 5000, RSFTotal: 4000, NSFRRatio: 150}
		err := NSFR05{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "discrepância") {
			t.Errorf("esperava erro discrepância NSFR, got %v", err)
		}
	})

	t.Run("NSFR05_NSFRConsistente", func(t *testing.T) {
		doc := &DocDLP{ASFTotal: 5000, RSFTotal: 4000, NSFRRatio: 125.0}
		err := NSFR05{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("NSFR declarado=calculado OK, got %v", err)
		}
	})

	t.Run("NSFR06_Cenario1ASFNegativo", func(t *testing.T) {
		doc := &DocDLP{Cenario1: CenarioNSFR{ASF: -100}}
		err := NSFR06{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Cenário 1 ASF negativo, got %v", err)
		}
	})

	t.Run("NSFR07_Cenario1RSFNegativo", func(t *testing.T) {
		doc := &DocDLP{Cenario1: CenarioNSFR{RSF: -50}}
		err := NSFR07{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Cenário 1 RSF negativo, got %v", err)
		}
	})

	t.Run("NSFR08_DtBaseInvalida", func(t *testing.T) {
		doc := &DocDLP{Root: DocDLPRoot{DataBase: "2024-12"}}
		err := NSFR08{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("esperava erro DtBase formato, got %v", err)
		}
	})

	t.Run("NSFR09_ASFConsistente", func(t *testing.T) {
		doc := &DocDLP{
			ASFTotal: 5000,
			Accounts: map[string]float64{
				"30.01": 3000,
				"30.02": 2000,
			},
		}
		err := NSFR09ASFConsistente{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("ASF consistente OK, got %v", err)
		}
	})

	t.Run("NSFR09_ASFDivergente", func(t *testing.T) {
		doc := &DocDLP{
			ASFTotal: 5000,
			Accounts: map[string]float64{
				"30.01": 100,
				"30.02": 100,
			},
		}
		err := NSFR09ASFConsistente{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "diff=") {
			t.Errorf("esperava erro ASF divergente, got %v", err)
		}
	})

	t.Run("NSFR10_RSFConsistente", func(t *testing.T) {
		doc := &DocDLP{
			RSFTotal: 4000,
			Accounts: map[string]float64{
				"40.01": 2500,
				"40.02": 1500,
			},
		}
		err := NSFR10RSFConsistente{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("RSF consistente OK, got %v", err)
		}
	})
}
