// Testes Sprint 41 — AuditDLP 2170 (NSFR).
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint41_ParseDocDLP(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocDLP cnpj="12345678" dataBase="2024-12-31">
  <ASFTotal valor="5000.00"/>
  <RSFTotal valor="4000.00"/>
  <NSFRRatio valor="125.00"/>
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
}

func TestSprint41_CalcularNSFRRatio(t *testing.T) {
	tests := []struct {
		asf, rsf, want float64
	}{
		{5000, 4000, 125.0},
		{4000, 4000, 100.0},
		{0, 0, -1},    // RSF zero
		{1000, 0, -1}, // RSF zero
		{6000, 5000, 120.0},
	}
	for _, tt := range tests {
		got := CalcularNSFRRatio(tt.asf, tt.rsf)
		// Tolerância 0.5 para floating point
		if got < 0 && tt.want >= 0 {
			t.Errorf("CalcularNSFRRatio(%v, %v)=%v, want %v", tt.asf, tt.rsf, got, tt.want)
		}
		if got >= 0 && (got < tt.want-0.5 || got > tt.want+0.5) {
			t.Errorf("CalcularNSFRRatio(%v, %v)=%v, want ~%v", tt.asf, tt.rsf, got, tt.want)
		}
	}
}

func TestSprint41_NSFRRegras(t *testing.T) {
	ctx := context.Background()

	t.Run("NSFR01_NSFRABAIXO", func(t *testing.T) {
		parsedDLP = &DocDLP{NSFRRatio: 80}
		err := NSFR01{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "< 100%") {
			t.Errorf("esperava erro NSFR < 100%%, got %v", err)
		}
	})

	t.Run("NSFR01_NSFRok", func(t *testing.T) {
		parsedDLP = &DocDLP{NSFRRatio: 150}
		err := NSFR01{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("NSFR=150 OK, got %v", err)
		}
	})

	t.Run("NSFR02_ASFNegativo", func(t *testing.T) {
		parsedDLP = &DocDLP{ASFTotal: -100}
		err := NSFR02{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro ASF negativo, got %v", err)
		}
	})

	t.Run("NSFR03_RSFNegativo", func(t *testing.T) {
		parsedDLP = &DocDLP{RSFTotal: -50}
		err := NSFR03{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro RSF negativo, got %v", err)
		}
	})

	t.Run("NSFR04_ASFMenorRSF", func(t *testing.T) {
		parsedDLP = &DocDLP{ASFTotal: 3000, RSFTotal: 4000}
		err := NSFR04{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "< RSF Total") {
			t.Errorf("esperava erro ASF < RSF, got %v", err)
		}
	})

	t.Run("NSFR04_ASFIgualRSF", func(t *testing.T) {
		parsedDLP = &DocDLP{ASFTotal: 4000, RSFTotal: 4000}
		err := NSFR04{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("ASF=RSF OK, got %v", err)
		}
	})

	t.Run("NSFR05_NSFRDiscrepancia", func(t *testing.T) {
		// ASF=5000, RSF=4000 → calculado = 125
		// Declarado 150 → discrepância > 1%
		parsedDLP = &DocDLP{ASFTotal: 5000, RSFTotal: 4000, NSFRRatio: 150}
		err := NSFR05{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "discrepância") {
			t.Errorf("esperava erro discrepância NSFR, got %v", err)
		}
	})

	t.Run("NSFR05_NSFRConsistente", func(t *testing.T) {
		parsedDLP = &DocDLP{ASFTotal: 5000, RSFTotal: 4000, NSFRRatio: 125.0}
		err := NSFR05{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("NSFR declarado=calculado OK, got %v", err)
		}
	})

	t.Run("NSFR06_Cenario1ASFNegativo", func(t *testing.T) {
		parsedDLP = &DocDLP{Cenario1: CenarioNSFR{ASF: -100}}
		err := NSFR06{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Cenário 1 ASF negativo, got %v", err)
		}
	})

	t.Run("NSFR07_Cenario1RSFNegativo", func(t *testing.T) {
		parsedDLP = &DocDLP{Cenario1: CenarioNSFR{RSF: -50}}
		err := NSFR07{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Cenário 1 RSF negativo, got %v", err)
		}
	})

	t.Run("NSFR08_DtBaseInvalida", func(t *testing.T) {
		parsedDLP = &DocDLP{Root: DocDLPRoot{DataBase: "2024-12"}}
		err := NSFR08{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Errorf("esperava erro DtBase formato, got %v", err)
		}
	})
}
