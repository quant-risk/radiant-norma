// Testes Sprint 38 — Audit3040 Fase 4 — FECHAMENTO 3040.
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint38_ReaisDetectamViolacoes(t *testing.T) {
	ctx := context.Background()

	t.Run("C81_DtContrFuturo", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].DtContr = "2099-12-01"
		err := C81DtContrNaoFuturo{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "futuro") {
			t.Errorf("esperava erro DtContr futuro, got %v", err)
		}
	})

	t.Run("C82_DtVencAntesContr", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].DtContr = "2024-12-15"
		doc.Operacoes[0].DtVencOp = "2024-01-01"
		err := C82DtVencAposContr{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "DtVencOp") {
			t.Errorf("esperava erro DtVencOp < DtContr, got %v", err)
		}
	})

	t.Run("C75_Inf0307_ValorZero", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Infos = []InfoAdicional{{Tp: "0307", Valor: "0"}}
		err := C75Inf1601CustoAquisicao{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "0307") {
			t.Errorf("esperava erro cessão sem custo, got %v", err)
		}
	})

	t.Run("C77_Inf18XX_PercZero", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Infos = []InfoAdicional{{Tp: "1801", Perc: "0"}}
		err := C77Inf18XXCoobrig{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "coobrigação") {
			t.Errorf("esperava erro coobrigação sem Perc, got %v", err)
		}
	})

	t.Run("C80_CrossRef_0307_Sem1201", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Infos = []InfoAdicional{{Tp: "0307"}}
		err := C80InfCrossRef03071201{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "0307/1201") {
			t.Errorf("esperava erro cross-ref incompleto, got %v", err)
		}
	})

	t.Run("C90_Cessao_SemCedente", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Infos = []InfoAdicional{{Tp: "0307"}}
		doc.Operacoes[0].Cli = nil
		err := C90CessaoCedenteCd{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "cedente") {
			t.Errorf("esperava erro cessão sem cedente, got %v", err)
		}
	})

	t.Run("SUB01_TpArqS_Remessa0", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.TpArq = "S"
		doc.Root.Remessa = "0"
		err := SUB01SubstituicaoRemessa{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Remessa=0") {
			t.Errorf("esperava erro TpArq=S Remessa=0, got %v", err)
		}
	})

	t.Run("SUB05_Substituicao_InfInvalido", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.TpArq = "S"
		doc.Operacoes[0].Infos = []InfoAdicional{{Tp: "0101"}} // não é I03XX
		err := SUB05SubstituicaoInf{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "I03XX") {
			t.Errorf("esperava erro Inf não-I03XX em substituição, got %v", err)
		}
	})

	t.Run("SUB06_SubstSemOps", func(t *testing.T) {
		doc := &Doc3040{Root: Doc3040Root{TpArq: "S"}}
		err := SUB06SubstituicaoMin1{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "substituição") {
			t.Errorf("esperava erro substituição sem ops, got %v", err)
		}
	})

	t.Run("I15_PF_AcimaLimite", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Cli.TpCli = "1"
		doc.Operacoes[0].Vencimentos = Vencimentos{V110: "600000"}
		err := I15LimitePFDestravada{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "limite") {
			t.Errorf("esperava erro PF > limite, got %v", err)
		}
	})

	t.Run("S78_ModRural_ClassH", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Mod = "0501"  // rural
		doc.Agregados[0].ClassOp = "H" // não permitido em rural (default A-D)
		err := S78ClassOpPorModDestravada{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Mod=0501") {
			t.Errorf("esperava erro Mod rural com ClassOp H, got %v", err)
		}
	})

	t.Run("N07_PrazoMax", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].DtContr = "2020-01-01"
		doc.Operacoes[0].DtVencOp = "2099-01-01" // 79 anos
		err := N07PrazoMaxDestravada{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "prazo") {
			t.Errorf("esperava erro prazo > max, got %v", err)
		}
	})

	// V69 — testes para regras consertadas (stubs disfarçados).
	t.Run("C84_PercForaRange", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Perc = "150"
		err := C84PercPropria{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "[0, 100]") {
			t.Errorf("esperava erro Perc fora de range, got %v", err)
		}
	})

	t.Run("C84_PercNegativo", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Perc = "-10"
		err := C84PercPropria{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "[0, 100]") {
			t.Errorf("esperava erro Perc negativo, got %v", err)
		}
	})

	t.Run("C84_PercOK", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Perc = "100"
		err := C84PercPropria{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("Perc=100 OK, got %v", err)
		}
	})

	t.Run("SUB07_TpArqS_Vazio", func(t *testing.T) {
		doc := &Doc3040{Root: Doc3040Root{TpArq: "S"}}
		err := SUB07SubstituicaoTotalF{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "substituição total") {
			t.Errorf("esperava erro TpArq=S vazio, got %v", err)
		}
	})

	t.Run("SUB07_TpArqS_ComOps_OK", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.TpArq = "S"
		err := SUB07SubstituicaoTotalF{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("TpArq=S com ops OK, got %v", err)
		}
	})
}

func TestSprint38_StubsReturnNil(t *testing.T) {
	doc := sampleDoc3040V2()
	ctx := context.Background()

	// Stubs Sprint 38 que continuam stubs (severity I, return nil em doc OK).
	stubs := []Rule{
		C71Inf1301Comissao{}, C72Inf1302Tarifa{}, C73Inf1401Seguro{},
		C74Inf1501IOF{}, C76Inf17XXGarantia{}, C78Inf19XXReestrut{},
		C79Inf20XXXNovos{}, C87DtVencCalc{}, C88ValorPrincipalJuros{},
		C89GarantiaFidej{},
		SUB02SubstituicaoParte{}, SUB03DocumentosReferenciados{},
		SUB04PreservaOperacoes{}, SUB08HistoricoSubstituicoes{},
		SUB11PreservaCli{}, SUB12SubstDataLimite{}, SUB14SubstAgregados{},
		SUB15SubstCrossIF{},
		X01CNPJCrossDoc{}, X03Ops30402042{}, X04Ops30402042Ag{},
		X05CliUnicoCross{}, X06IPOCUnicoCross{}, X07VencimentosCross{},
		X08ProvConsttdCross{}, X09Consolidacao3050{}, X10ModalidadeCross{},
		S86DtVencCalcDestravada{},
	}

	for _, s := range stubs {
		t.Run(s.Code(), func(t *testing.T) {
			err := s.Apply(ctx, doc)
			if err != nil {
				t.Errorf("%s stub retornou erro inesperado: %v", s.Code(), err)
			}
			if s.Severity() != "I" {
				t.Errorf("%s deveria ter severity \"I\", tem %q", s.Code(), s.Severity())
			}
		})
	}
}
