// Testes Sprint 37 — Audit3040 Fase 3 — I06-I15, A16-A30, S71-S90 + destravadas.
package rules

import (
	"context"
	"strings"
	"testing"
)

func TestSprint37_ReaisDetectamViolacoes(t *testing.T) {
	ctx := context.Background()

	t.Run("I06_ModRuralPJ_Fail", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Cli.TpCli = "2" // PJ
		doc.Operacoes[0].Contrt = "05RURAL"
		err := I06ContratoModPJ{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "PJ") {
			t.Errorf("esperava erro PJ rural, got %v", err)
		}
	})

	t.Run("I07_IPOCCliUnico_Duplicado", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes = append(doc.Operacoes, Operacao{
			IPOC: "12345678",
			Cli:  &Cli{Cd: "12345678901", TpCli: "1", IPOC: "12345678"},
		})
		err := I07IPOCCliUnico{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "duplicado") {
			t.Errorf("esperava erro IPOC+Cli duplicado, got %v", err)
		}
	})

	t.Run("I08_ProvNegativa", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].ProvConsttd = "-100"
		err := I08ProvIndPositiva{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro provisão negativa, got %v", err)
		}
	})

	t.Run("I10_IPOCInvalido", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].IPOC = "abc@123" // contém @
		err := I10IPOCFormato{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "formato") {
			t.Errorf("esperava erro IPOC formato, got %v", err)
		}
	})

	t.Run("I12_CliIPOC_Diferente", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Cli.IPOC = "99999"
		err := I12CliIPOCIgualOpIPOC{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Cli.IPOC") {
			t.Errorf("esperava erro Cli.IPOC diferente, got %v", err)
		}
	})

	t.Run("A19_ModNatuOp_Invalido", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Mod = "9999"
		doc.Agregados[0].NatuOp = "99"
		err := A19ModNatuOpValido{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "regulamentar") {
			t.Errorf("esperava erro Mod×NatuOp, got %v", err)
		}
	})

	t.Run("A21_LocalizInvalida", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Localiz = "XX"
		err := A21LocalizUFValida{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "UF") {
			t.Errorf("esperava erro UF inválida, got %v", err)
		}
	})

	t.Run("A21_LocalizValida_SP", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Localiz = "sp" // minúsculo
		err := A21LocalizUFValida{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("UF=SP (minúsculo) deve normalizar para SP OK, got %v", err)
		}
	})

	t.Run("A29_QtdCliExigeCampos", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].QtdCli = "10"
		doc.Agregados[0].ClassOp = ""
		err := A29QtdCliExigeCampos{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "ClassOp") {
			t.Errorf("esperava erro ClassOp required, got %v", err)
		}
	})

	t.Run("S72_PercForaRange", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Perc = "150"
		err := S72PercRange{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "[0, 100]") {
			t.Errorf("esperava erro Perc fora range, got %v", err)
		}
	})

	t.Run("S76_ParteNaoNumerica", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.Parte = "abc"
		err := S76ParteNumericaSeq{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "não-numérico") {
			t.Errorf("esperava erro Parte não-numérica, got %v", err)
		}
	})

	t.Run("S81_VencimentosForaOrdem", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Vencimentos = Vencimentos{
			V110: "100", V120: "50", // V110 > V120 (invertido)
		}
		err := S81VencimentosOrdem{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "ordem") {
			t.Errorf("esperava erro ordem cronológica, got %v", err)
		}
	})

	t.Run("S83_QtdCliNaoInteiro", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].QtdCli = "10.5"
		err := S83QtdCliInteiro{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "não-inteiro") {
			t.Errorf("esperava erro QtdCli não-inteiro, got %v", err)
		}
	})

	t.Run("S89_ClassOpVincME_A", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].VincME = "S"
		doc.Agregados[0].ClassOp = "A"
		err := S89ClassOpVincME{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "VincME=S") {
			t.Errorf("esperava erro VincME=S com ClassOp A, got %v", err)
		}
	})

	// Carry-over destravadas
	t.Run("C44Destravada_LocalizPF", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].NatuOp = "02"
		doc.Agregados[0].TpCli = "1"
		doc.Agregados[0].Localiz = ""
		err := C44LocalizPFDestravada{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "Localiz") {
			t.Errorf("esperava erro Localiz required, got %v", err)
		}
	})

	t.Run("C46Destravada_BNDES_OrigemRec", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].Mod = "0271"
		doc.Agregados[0].OrigemRec = ""
		err := C46OrigemRecBNDESDestravada{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "BNDES") {
			t.Errorf("esperava erro BNDES sem OrigemRec, got %v", err)
		}
	})
}

func TestSprint37_StubsReturnNil(t *testing.T) {
	doc := sampleDoc3040V2()
	ctx := context.Background()

	stubs := []Rule{
		I15LimitePF{},
		S78ClassOpPorModValido{},
		S79DtBaseAtual{},
		S84CNPJCliConsolidado{},
		S85CessaoCedente{},
		S86DtVencCalc{},
		S90RemessaUnicaDtBase{},
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

	t.Run("I13_DtVenc5Anos", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].DtVencOp = "2099-01-01" // 75 anos no futuro
		err := I13DtVencJanela5Anos{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "5anos") {
			t.Errorf("esperava erro DtVenc > 5anos, got %v", err)
		}
	})

	t.Run("I14_IPOCZeros", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].IPOC = "00000000"
		err := I14IPOCBemFormado{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "zeros") {
			t.Errorf("esperava erro IPOC zeros, got %v", err)
		}
	})

	t.Run("A16_FaixaVlrInvalida", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].FaixaVlr = "99"
		err := A16ClassOpFaixaVlr{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "FaixaVlr") {
			t.Errorf("esperava erro FaixaVlr inválida, got %v", err)
		}
	})

	t.Run("A20_PrzProvmS_ClassA", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].PrzProvm = "S"
		doc.Agregados[0].ClassOp = "A"
		err := A20PrzProvmClassOp{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "PrzProvm=S") {
			t.Errorf("esperava erro PrzProvm=S com ClassOp A, got %v", err)
		}
	})

	t.Run("A24_DesempOpInvalido", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].DesempOp = "99"
		err := A24DesempOpValido{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "DesempOp") {
			t.Errorf("esperava erro DesempOp inválido, got %v", err)
		}
	})

	t.Run("A26_NatuOp02SemOrigemRec", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].NatuOp = "02"
		doc.Agregados[0].OrigemRec = ""
		err := A26NatuOp02OrigemRec{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "OrigemRec") {
			t.Errorf("esperava erro OrigemRec required, got %v", err)
		}
	})

	t.Run("A27_VincME_ModNaoME", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].VincME = "S"
		doc.Agregados[0].Mod = "0201" // não é ME
		err := A27VincMEModME{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "VincME=S") {
			t.Errorf("esperava erro VincME=S sem Mod ME, got %v", err)
		}
	})

	t.Run("S71_ValorNegativo", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Valor = "-100"
		err := S71ValorPositivoQtdOp{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro Valor negativo, got %v", err)
		}
	})

	t.Run("S75_TotalCliBate", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Root.TotalCli = "10"
		doc.Agregados[0].QtdCli = "10"
		err := S75TotalCliConsistente{}.Apply(ctx, doc)
		if err != nil {
			t.Errorf("TotalCli=10 soma=10 OK, got %v", err)
		}
	})

	t.Run("S80_QtdOpNegativo", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].QtdOp = "-5"
		err := S80QtdOpNaoNegativo{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "negativo") {
			t.Errorf("esperava erro QtdOp negativo, got %v", err)
		}
	})

	t.Run("S82_ValorMenorVenc", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Operacoes[0].Valor = "100"
		doc.Operacoes[0].Vencimentos = Vencimentos{
			V110: "50", V120: "100", // soma = 150 > Valor = 100
		}
		err := S82ValorMaiorVencimentos{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "< soma vencimentos") {
			t.Errorf("esperava erro Valor < soma, got %v", err)
		}
	})

	t.Run("S87_QtdOpNaoInteiro", func(t *testing.T) {
		doc := sampleDoc3040V2()
		doc.Agregados[0].QtdOp = "5.5"
		err := S87QtdOpInteiro{}.Apply(ctx, doc)
		if err == nil || !strings.Contains(err.Error(), "não-inteiro") {
			t.Errorf("esperava erro QtdOp não-inteiro, got %v", err)
		}
	})
}

func TestSprint37_Helpers(t *testing.T) {
	t.Run("validarUF", func(t *testing.T) {
		if !validarUF("SP") {
			t.Errorf("SP deve ser UF válida")
		}
		if !validarUF("EX") {
			t.Errorf("EX (exterior) deve ser válido")
		}
		if validarUF("XX") {
			t.Errorf("XX não deve ser UF válida")
		}
	})

	t.Run("validarIPOC", func(t *testing.T) {
		if !validarIPOC("ABC12345") {
			t.Errorf("ABC12345 deve ser IPOC válido")
		}
		if validarIPOC("abc@123") {
			t.Errorf("abc@123 não deve ser IPOC válido (contém @)")
		}
		if validarIPOC("1234567") {
			t.Errorf("1234567 (7 chars) não deve ser IPOC válido")
		}
	})

	t.Run("validarModNatuOp", func(t *testing.T) {
		if !validarModNatuOp("0201", "01") {
			t.Errorf("0201|01 deve ser válido")
		}
		if validarModNatuOp("9999", "99") {
			t.Errorf("9999|99 não deve ser válido")
		}
	})

	t.Run("validarPerc", func(t *testing.T) {
		if !validarPerc(50) {
			t.Errorf("50 deve ser válido")
		}
		if validarPerc(-1) || validarPerc(101) {
			t.Errorf("-1 ou 101 não deve ser válido")
		}
	})

	t.Run("addAnos", func(t *testing.T) {
		if got := addAnos("2024", 5); got != "2029" {
			t.Errorf("addAnos(2024, 5)=%s, want 2029", got)
		}
		if got := addAnos("2024", 0); got != "2024" {
			t.Errorf("addAnos(2024, 0)=%s, want 2024", got)
		}
	})

	t.Run("isVencimentoOrdemCronologica", func(t *testing.T) {
		v := Vencimentos{V110: "10", V120: "20", V150: "30", V160: "40", V165: "50"}
		if !isVencimentoOrdemCronologica(v) {
			t.Errorf("V110<V120<V150<V160<V165 deve ser OK")
		}
		v2 := Vencimentos{V110: "50", V120: "10"}
		if isVencimentoOrdemCronologica(v2) {
			t.Errorf("V110>V120 não deve ser ordem cronológica")
		}
	})

	t.Run("isFaixaVlrValida", func(t *testing.T) {
		if !isFaixaVlrValida("01") || !isFaixaVlrValida("13") {
			t.Errorf("01 e 13 devem ser válidos")
		}
		if isFaixaVlrValida("14") || isFaixaVlrValida("00") {
			t.Errorf("14 e 00 não devem ser válidos")
		}
	})
}
