// Testes Sprint 42 — Audit3044 (Engine JSON).
package rules

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSprint42_ParseDoc3044(t *testing.T) {
	json := `{
  "cnpjIF": "12345678",
  "dataHoraRemessa": "2026-07-03 14:30:00",
  "envia3050": "S",
  "operacoes": [
    {
      "acao": 1,
      "ipoc": "876543210216210020716C1234",
      "class3050": "112212101",
      "saldoDevedor": 5000.00,
      "dataSaldoDevedor": "2026-07-03",
      "atraso": "N",
      "pagamentos": [
        { "acao": 1, "data": "2026-06-15", "valor": 1000.00 }
      ],
      "concessoes": [],
      "cessoes": [],
      "aquisicoes": []
    }
  ]
}`

	doc, err := ParseDoc3044([]byte(json))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.CNPJ != "12345678" {
		t.Errorf("CNPJ errado: %v", doc.CNPJ)
	}
	if doc.Envia3050 != "S" {
		t.Errorf("Envia3050 errado: %v", doc.Envia3050)
	}
	if len(doc.Operacoes) != 1 {
		t.Fatalf("esperava 1 operacao, got %d", len(doc.Operacoes))
	}
	op := doc.Operacoes[0]
	if op.IPOC != "876543210216210020716C1234" {
		t.Errorf("IPOC errado: %v", op.IPOC)
	}
	if op.SaldoDevedor != 5000.00 {
		t.Errorf("SaldoDevedor errado: %v", op.SaldoDevedor)
	}
	if len(op.Pagamentos) != 1 {
		t.Errorf("esperava 1 pagamento, got %d", len(op.Pagamentos))
	}
	if op.Pagamentos[0].Valor != 1000.00 {
		t.Errorf("Pagamento valor errado: %v", op.Pagamentos[0].Valor)
	}
}

func TestSprint42_T01_DataHoraRemessaAntesSaldo(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-01"),
		Operacoes: []Operacao3044{
			{
				IPOC:             "87654321",
				DataSaldoDevedor: timeDate("2026-07-03"),
			},
		},
	}
	err := T01{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T01") {
		t.Errorf("esperava erro T01, got %v", err)
	}
}

func TestSprint42_T01_Ok(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC:             "87654321",
				DataSaldoDevedor: timeDate("2026-07-03"),
			},
		},
	}
	err := T01{}.Apply(ctx, doc)
	if err != nil {
		t.Errorf("T01 OK, got %v", err)
	}
}

func TestSprint42_T02_PagamentoFuturo(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC:             "IPOC1",
				DataSaldoDevedor: timeDate("2026-07-03"),
				Pagamentos: []Pagamento3044{
					{Data: timeDate("2026-07-10"), Valor: 100},
				},
			},
		},
	}
	err := T02{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T02") {
		t.Errorf("esperava erro T02, got %v", err)
	}
}

func TestSprint42_T03_ConcessaoFutura(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC:             "IPOC1",
				DataSaldoDevedor: timeDate("2026-07-03"),
				Concessoes: []Concessao3044{
					{Data: timeDate("2026-07-10"), Valor: 500},
				},
			},
		},
	}
	err := T03{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T03") {
		t.Errorf("esperava erro T03, got %v", err)
	}
}

func TestSprint42_T04_Futura(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: time.Now().AddDate(0, 0, 1), // amanhã
	}
	err := T04{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T04") {
		t.Errorf("esperava erro T04, got %v", err)
	}
}

func TestSprint42_T05_Duplicado(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC: "IPOC1",
				Pagamentos: []Pagamento3044{
					{Data: timeDate("2026-06-01"), Valor: 100},
					{Data: timeDate("2026-06-01"), Valor: 200},
				},
			},
		},
	}
	err := T05{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T05") {
		t.Errorf("esperava erro T05, got %v", err)
	}
}

func TestSprint42_T06_DuplicadoConcessao(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC: "IPOC1",
				Concessoes: []Concessao3044{
					{Data: timeDate("2026-06-01"), Valor: 500},
					{Data: timeDate("2026-06-01"), Valor: 600},
				},
			},
		},
	}
	err := T06{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T06") {
		t.Errorf("esperava erro T06, got %v", err)
	}
}

func TestSprint42_T07_Envia3050N(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Envia3050:       "N",
		Operacoes: []Operacao3044{
			{
				IPOC:      "IPOC1",
				Class3050: "112212101",
			},
		},
	}
	err := T07{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T07") {
		t.Errorf("esperava erro T07, got %v", err)
	}
}

func TestSprint42_T08_FormatoInvalido(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Envia3050:       "S",
		Operacoes: []Operacao3044{
			{
				IPOC:      "IPOC1",
				Class3050: "abc", // inválido
			},
		},
	}
	err := T08{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T08") {
		t.Errorf("esperava erro T08, got %v", err)
	}
}

func TestSprint42_T15_ValorNegativo(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC: "IPOC1",
				Pagamentos: []Pagamento3044{
					{Data: timeDate("2026-06-01"), Valor: -50},
				},
			},
		},
	}
	err := T15{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T15") {
		t.Errorf("esperava erro T15, got %v", err)
	}
}

func TestSprint42_T16_SaldoNegativo(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{
				IPOC:         "IPOC1",
				SaldoDevedor: -1000,
			},
		},
	}
	err := T16{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T16") {
		t.Errorf("esperava erro T16, got %v", err)
	}
}

func TestSprint42_T17_IPOCDuplicado(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{IPOC: "IPOC1", Acao: 1},
			{IPOC: "IPOC1", Acao: 2},
		},
	}
	err := T17{}.Apply(ctx, doc)
	if err == nil || !strings.Contains(err.Error(), "T17") {
		t.Errorf("esperava erro T17, got %v", err)
	}
}

func TestSprint42_T17_Ok(t *testing.T) {
	ctx := context.Background()
	doc := &Doc3044{
		DataHoraRemessa: timeDate("2026-07-05"),
		Operacoes: []Operacao3044{
			{IPOC: "IPOC1", Acao: 1},
			{IPOC: "IPOC2", Acao: 2},
		},
	}
	err := T17{}.Apply(ctx, doc)
	if err != nil {
		t.Errorf("T17 OK, got %v", err)
	}
}

// timeDate é helper para criar time.Time a partir de string YYYY-MM-DD 00:00:00.
func timeDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}
