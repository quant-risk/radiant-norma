// Testes Sprint 36 — Audit3040 Fase 2 — stubs + regras reais.
//
// Os stubs (severity "I") retornam nil por design (D-26: stub honesto).
// Testes confirmam que (a) retornam nil em docs vazios/ok, (b) regras
// reais detectam violações reais.
package rules

import (
	"context"
	"strings"
	"testing"
)

// sampleDoc3040V2 retorna um Doc3040 mínimo válido para testes Sprint 36.
func sampleDoc3040V2() *Doc3040 {
	return &Doc3040{
		Root: Doc3040Root{
			DtBase:    "2024-12",
			CNPJ:      "12345678",
			Remessa:   "1",
			Parte:     "1",
			TpArq:     "F",
			NomeResp:  "João da Silva",
			EmailResp: "[email protected]",
			TelResp:   "11999998888",
			TotalCli:  "10",
		},
		Agregados: []Agregado{
			{
				NatuOp:    "01",
				Mod:       "0201",
				OrigemRec: "01",
				VincME:    "N",
				ClassOp:   "A",
				FaixaVlr:  "01",
				PrzProvm:  "N",
				Localiz:   "SP",
				TpCli:     "1",
				DesempOp:  "01",
				QtdOp:     "5",
				QtdCli:    "10",
				Vencimentos: Vencimentos{
					V110: "0", V120: "0", V150: "0", V160: "0", V165: "0",
				},
			},
		},
		Operacoes: []Operacao{
			{
				Inf:      "0101",
				Contrt:   "12345",
				IPOC:     "12345678",
				Valor:    "10000",
				Perc:     "100",
				DtContr:  "2024-01-15",
				DtVencOp: "2025-01-15",
				ClassOp:  "A",
				Cli: &Cli{
					Cd:    "12345678901",
					TpCli: "1",
					IPOC:  "12345678",
				},
				Parcelas: []Parcela{
					{Num: 1, DtVenc: "2024-12-15"},
				},
			},
		},
	}
}

func TestSprint36_StubsReturnNil(t *testing.T) {
	doc := sampleDoc3040V2()
	ctx := context.Background()

	// Lista completa de stubs Sprint 36 (severity "I" — retornam nil).
	stubs := []Rule{
		C21Inf0101NatuOp01{},
		C22Inf0308Garantia{},
		C23Inf0313Perc{},
		C24Inf0501Reneg{},
		C25Inf0703DtLib{},
		C26Inf0704Refin{},
		C27Inf0801Vinculo{},
		C28Inf0901Rural{},
		C29Inf1001Habit{},
		C30Inf1101Leasing{},
		C44LocalizPF{},
		C45VincMEMod{},
		C46OrigemRecBNDES{},
		C56Inf0213Rel0307{},
		C57Inf0307Rel1201{},
		C61DtVencPosContr{},
		C62ClassOpIndAg{},
		C63ProvIndAg{},
		C65QtdCliIndAg{},
		C66CliObrigInfI03{},
		C68CliIPOCEqual{},
		C70GarantidorFidej{},
		N02CliMesmoClassOp{},
		N03LimitePorCli{},
		N04ConcentracaoMod{},
		N05LimiteBasileia{},
		N07PrazoMax{},
		N08CarenciaMin{},
		N09IdadeCli{},
		N10ConsolidacaoConglomerado{},
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

func TestSprint36_ReaisDetectamViolacoes(t *testing.T) {
	ctx := context.Background()

	t.Run("C58_IPOCDuplicado", func(t *testing.T) {
		doc := sampleDoc3040V2()
		// Duplicar IPOC
		doc.Operacoes = append(doc.Operacoes, Operacao{
			IPOC:    "12345678",
			DtContr: "2024-02-01",
		})
		err := C58IPOCUnicoRemessa{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "IPOC duplicado") {
			t.Errorf("esperava erro de IPOC duplicado, got %v", err)
		}
	})

	t.Run("C58_IPOCUnico", func(t *testing.T) {
		doc := sampleDoc3040V2()
		err := C58IPOCUnicoRemessa{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("IPOC único não deveria falhar: %v", err)
		}
	})

	t.Run("C59_ContratoUnicoIPOCDt", func(t *testing.T) {
		doc := sampleDoc3040V2()
		// Duplicar IPOC+DtContr
		doc.Operacoes = append(doc.Operacoes, Operacao{
			IPOC:    "12345678",
			DtContr: "2024-01-15",
		})
		err := C59ContratoUnicoIPOCDt{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "duplicado") {
			t.Errorf("esperava erro de duplicação, got %v", err)
		}
	})

	t.Run("C60_DtContrSaneamento", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].DtContr = "1899-01-01"
		err := C60DtContrSaneamento{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "1900") {
			t.Errorf("esperava erro de saneamento, got %v", err)
		}
	})

	t.Run("C67_CliCdFormatoPF", func(t *testing.T) {
		doc := sampleDoc3040V2()
		// PF deve ter 11 dígitos
		doc.Operacoes[0].Cli.Cd = "123"
		err := C67CliCdFormato{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "11 dígitos") {
			t.Errorf("esperava erro de CPF, got %v", err)
		}
	})

	t.Run("C67_CliCdFormatoPJ", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Cli.TpCli = "2"
		doc.Operacoes[0].Cli.Cd = "12345678" // 8 dígitos OK
		err := C67CliCdFormato{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("PJ com 8 dígitos OK, got %v", err)
		}
	})

	t.Run("C69_ParcelaDtVencOp", func(t *testing.T) {
		doc := sampleDoc3040V2()
		// DtVencOp = 2025-01-15, parcela com DtVenc = 2026-01-01 → falha
		doc.Operacoes[0].Parcelas[0].DtVenc = "2026-01-01"
		err := C69ParcelaDtVencOp{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "DtVencOp") {
			t.Errorf("esperava erro de parcela > DtVencOp, got %v", err)
		}
	})

	t.Run("H05_CNPJRaiz8Dig", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.CNPJ = "123"
		err := H05CNPJRaiz8Dig{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "8 dígitos") {
			t.Errorf("esperava erro de CNPJ raiz, got %v", err)
		}
	})

	t.Run("H06_RemessaNumerica", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.Remessa = "abc"
		err := H06RemessaNumerica{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "não-numérico") {
			t.Errorf("esperava erro de remessa, got %v", err)
		}
	})

	t.Run("H08_TpArqHeader", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.TpArq = "X"
		err := H08TpArqHeader{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "F ou S") {
			t.Errorf("esperava erro de TpArq, got %v", err)
		}
	})

	t.Run("N01_CliUnicoRemessa", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes = append(doc.Operacoes, Operacao{
			Cli: &Cli{Cd: "12345678901", TpCli: "1", IPOC: "99999"},
		})
		err := N01CliUnicoRemessa{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "cliente") {
			t.Errorf("esperava erro de cliente duplicado, got %v", err)
		}
	})

	t.Run("N06_ProvMinClassOp_H", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].ClassOp = "H"
		doc.Agregados[0].ProvConsttd = "0"
		err := N06ProvMinClassOp{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "ClassOp=H") {
			t.Errorf("esperava erro de provisão mínima, got %v", err)
		}
	})

	t.Run("C41_ClassOpPorMod_OK", func(t *testing.T) {
		doc := sampleDoc3040V2()
		err := C41ClassOpPorMod{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("Mod 02XX + ClassOp A deve passar, got %v", err)
		}
	})

	t.Run("C41_ClassOpPorMod_Fail", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].ClassOp = ""
		err := C41ClassOpPorMod{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "exige ClassOp") {
			t.Errorf("esperava erro de ClassOp, got %v", err)
		}
	})
}

func TestSprint36_SeverityCorrectness(t *testing.T) {
	// Regras reais devem ter severity "E" (erro) ou "A" (aviso).
	// Stubs devem ter "I" (info).
	stubs := []Rule{
		C21Inf0101NatuOp01{}, C22Inf0308Garantia{}, C23Inf0313Perc{},
		C44LocalizPF{}, C45VincMEMod{}, C56Inf0213Rel0307{},
		N02CliMesmoClassOp{}, N07PrazoMax{}, N09IdadeCli{},
	}
	for _, s := range stubs {
		if s.Severity() != "I" {
			t.Errorf("%s: stub deveria ter severity I, tem %q", s.Code(), s.Severity())
		}
	}

	reais := []Rule{
		C58IPOCUnicoRemessa{}, C59ContratoUnicoIPOCDt{}, C60DtContrSaneamento{},
		C67CliCdFormato{}, C69ParcelaDtVencOp{}, H05CNPJRaiz8Dig{},
		H06RemessaNumerica{}, H07ParteNumerica{}, H08TpArqHeader{},
		N01CliUnicoRemessa{},
	}
	for _, r := range reais {
		sev := r.Severity()
		if sev != "E" && sev != "A" {
			t.Errorf("%s: regra real deveria ter severity E/A, tem %q", r.Code(), sev)
		}
	}
}

func TestSprint36_SheetAtribuida(t *testing.T) {
	// Cada regra deve ter Sheet() não-vazia para reportagem cross-doc.
	all := []Rule{
		C21Inf0101NatuOp01{}, C41ClassOpPorMod{}, C58IPOCUnicoRemessa{},
		H04DtBasePeriodo{}, N01CliUnicoRemessa{}, N06ProvMinClassOp{},
	}
	for _, r := range all {
		if r.Sheet() == "" {
			t.Errorf("%s: Sheet() vazia", r.Code())
		}
	}
}
