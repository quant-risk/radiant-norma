// Testes Sprint 43 — CrossDoc_v2 (DRL/DLP × 3044).
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint43_XD01_CNPJMismatch(t *testing.T) {
	ctx := context.Background()

	// Reset globals.
	parsedDRL = nil
	parsedDLP = nil
	parsed3044 = nil

	t.Run("CNPJ_DRL_vs_3044_mismatch", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{CNPJ: "12345678"}}
		parsed3044 = &Doc3044{CNPJ: "87654321"}
		err := XD01{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD01") {
			t.Errorf("esperava erro XD01, got %v", err)
		}
	})

	t.Run("CNPJ_DRL_vs_DLP_mismatch", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{CNPJ: "12345678"}}
		parsedDLP = &DocDLP{Root: DocDLPRoot{CNPJ: "87654321"}}
		parsed3044 = nil
		err := XD01{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD01") {
			t.Errorf("esperava erro XD01, got %v", err)
		}
	})

	t.Run("CNPJ_OK", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{CNPJ: "12345678"}}
		parsedDLP = &DocDLP{Root: DocDLPRoot{CNPJ: "12345678"}}
		parsed3044 = &Doc3044{CNPJ: "12345678"}
		err := XD01{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD01 OK, got %v", err)
		}
	})
}

func TestSprint43_XD02_DataMismatch(t *testing.T) {
	ctx := context.Background()

	parsedDRL = nil
	parsedDLP = nil
	parsed3044 = nil

	t.Run("DtBase_DRL_vs_DLP_mismatch", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{DataBase: "2026-07-01"}}
		parsedDLP = &DocDLP{Root: DocDLPRoot{DataBase: "2026-07-03"}}
		err := XD02{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD02") {
			t.Errorf("esperava erro XD02, got %v", err)
		}
	})

	t.Run("DtBase_DRL_vs_3044_mismatch", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{DataBase: "2026-07-01"}}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{
					IPOC:             "IPOC1",
					DataSaldoDevedor: timeDate("2026-07-03"),
				},
			},
		}
		err := XD02{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD02") {
			t.Errorf("esperava erro XD02, got %v", err)
		}
	})

	t.Run("DtBase_OK", func(t *testing.T) {
		parsedDRL = &DocDRL{Root: DocDRLRoot{DataBase: "2026-07-03"}}
		parsedDLP = &DocDLP{Root: DocDLPRoot{DataBase: "2026-07-03"}}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{
					IPOC:             "IPOC1",
					DataSaldoDevedor: timeDate("2026-07-03"),
				},
			},
		}
		err := XD02{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD02 OK, got %v", err)
		}
	})
}

func TestSprint43_XD03_SaldoDevedorMaiorHQLA(t *testing.T) {
	ctx := context.Background()

	parsedDRL = nil
	parsedDLP = nil
	parsed3044 = nil

	t.Run("somaMaiorHQLA", func(t *testing.T) {
		parsedDRL = &DocDRL{HQLA: 1000}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{SaldoDevedor: 600},
				{SaldoDevedor: 600},
			},
		}
		err := XD03{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD03") {
			t.Errorf("esperava erro XD03, got %v", err)
		}
	})

	t.Run("somaMenorHQLA_ok", func(t *testing.T) {
		parsedDRL = &DocDRL{HQLA: 5000}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{SaldoDevedor: 600},
				{SaldoDevedor: 400},
			},
		}
		err := XD03{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD03 OK, got %v", err)
		}
	})
}

func TestSprint43_XD04_InconsistenciaLCRNSFR(t *testing.T) {
	ctx := context.Background()

	parsedDRL = nil
	parsedDLP = nil
	parsed3044 = nil

	t.Run("LCR_baixo_NSFR_alto", func(t *testing.T) {
		parsedDRL = &DocDRL{LCRRatio: 70}
		parsedDLP = &DocDLP{NSFRRatio: 130}
		err := XD04{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD04") {
			t.Errorf("esperava erro XD04, got %v", err)
		}
	})

	t.Run("LCR_baixo_NSFR_baixo_OK", func(t *testing.T) {
		parsedDRL = &DocDRL{LCRRatio: 70}
		parsedDLP = &DocDLP{NSFRRatio: 90}
		err := XD04{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD04 OK (ambos baixos), got %v", err)
		}
	})

	t.Run("LCR_OK_NSFR_OK", func(t *testing.T) {
		parsedDRL = &DocDRL{LCRRatio: 120}
		parsedDLP = &DocDLP{NSFRRatio: 130}
		err := XD04{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD04 OK, got %v", err)
		}
	})
}

func TestSprint43_XD05_PagamentosMaiorOutflows(t *testing.T) {
	ctx := context.Background()

	parsedDRL = nil
	parsedDLP = nil
	parsed3044 = nil

	t.Run("pagamentosMaiorOutflows", func(t *testing.T) {
		parsedDRL = &DocDRL{Outflows: 500}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{
					IPOC: "IPOC1",
					Pagamentos: []Pagamento3044{
						{Valor: 300},
						{Valor: 300},
					},
				},
			},
		}
		err := XD05{}.Apply(ctx, &Doc3040{})
		if err == nil || !strings.Contains(err.Error(), "XD05") {
			t.Errorf("esperava erro XD05, got %v", err)
		}
	})

	t.Run("pagamentosMenorOutflows_OK", func(t *testing.T) {
		parsedDRL = &DocDRL{Outflows: 1000}
		parsed3044 = &Doc3044{
			CNPJ: "12345678",
			Operacoes: []Operacao3044{
				{
					IPOC: "IPOC1",
					Pagamentos: []Pagamento3044{
						{Valor: 300},
						{Valor: 200},
					},
				},
			},
		}
		err := XD05{}.Apply(ctx, &Doc3040{})
		if err != nil {
			t.Errorf("XD05 OK, got %v", err)
		}
	})
}

func TestSprint43_XD06_CarryOver(t *testing.T) {
	ctx := context.Background()
	// XD06 é carry-over — deve retornar nil.
	err := XD06{}.Apply(ctx, &Doc3040{})
	if err != nil {
		t.Errorf("XD06 carry-over, got %v", err)
	}
}

func TestSprint43_XD07_CarryOver(t *testing.T) {
	ctx := context.Background()
	err := XD07{}.Apply(ctx, &Doc3040{})
	if err != nil {
		t.Errorf("XD07 carry-over, got %v", err)
	}
}

func TestSprint43_XD08_CarryOver(t *testing.T) {
	ctx := context.Background()
	err := XD08{}.Apply(ctx, &Doc3040{})
	if err != nil {
		t.Errorf("XD08 carry-over, got %v", err)
	}
}
