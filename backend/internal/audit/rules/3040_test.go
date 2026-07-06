// Tests for 3040 regras semânticas.
//
// Cobertura:
//   - parseNum helper (5 edge cases)
//   - B06 Remessa, B07 Parte
//   - F02 Datas (formato)
//   - C01 PJ obrigatórios
//   - S01 Detalhamento Cliente, S04 Crédito a Liberar, S05 Limite Crédito
package rules_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
)

// Helper: parse XML curto
func mustParse(t *testing.T, xml string) *rules.Doc3040 {
	t.Helper()
	doc, err := rules.ParseDoc3040([]byte(xml))
	if err != nil {
		t.Fatalf("ParseDoc3040: %v", err)
	}
	return doc
}

// ============================================================
// parseNum (via regras que o usam: S04, S01, S05)
// ============================================================

func TestS04_CreditoALiberar_Zero(t *testing.T) {
	// Modalidades onde v150/v160 devem ser zero
	modalidades := []string{"0204", "0210", "1304", "0201", "0213", "0214"}
	for _, mod := range modalidades {
		mod := mod
		t.Run(mod, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="` + mod + `" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="0" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
			doc := mustParse(t, xml)
			err := rules.S04CreditoALiberar{}.Apply(context.Background(), doc)
			if err != nil {
				t.Errorf("S04 com v150=0 v160=0 deveria passar, got: %v", err)
			}
		})
	}
}

func TestS04_CreditoALiberar_Preenchido(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="500" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S04CreditoALiberar{}.Apply(context.Background(), doc)
	if err == nil {
		t.Fatal("S04 com v150=500 deveria detectar erro")
	}
	if !strings.Contains(err.Error(), "0213") {
		t.Errorf("erro deveria mencionar modalidade 0213, got: %v", err)
	}
}

func TestS04_CreditoALiberar_OutrasModalidades(t *testing.T) {
	// Modalidades FORA da lista devem ser ignoradas mesmo com v150>0
	outras := []string{"0202", "0215", "0501", "19", "0900"}
	for _, mod := range outras {
		mod := mod
		t.Run(mod, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="` + mod + `" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="500" v160="500" v165="0"/>
  </Agreg>
</Doc3040>`
			doc := mustParse(t, xml)
			err := rules.S04CreditoALiberar{}.Apply(context.Background(), doc)
			if err != nil {
				t.Errorf("S04 não deveria aplicar a modalidade %s, got: %v", mod, err)
			}
		})
	}
}

// TestS04_ParseNumVariations testa que parseNum aceita várias representações de zero
// (helper introduzido em v1.3.6).
func TestS04_ParseNumVariations(t *testing.T) {
	cases := []struct {
		name  string
		v150  string
		v160  string
		valid bool
	}{
		{"inteiro zero", "0", "0", true},
		{"decimal zero", "0.0", "0.0", true},
		{"decimal zero negativo", "-0", "0.0", true},
		{"whitespace zero", "  0  ", "  0  ", true},
		{"vazio (Venc ausente)", "", "", true},
		{"decimal positivo", "500.50", "0", false},
		{"inteiro positivo", "500", "0", false},
		{"v160 positivo", "0", "500", false},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="` + c.v150 + `" v160="` + c.v160 + `" v165="0"/>
  </Agreg>
</Doc3040>`
			doc := mustParse(t, xml)
			err := rules.S04CreditoALiberar{}.Apply(context.Background(), doc)
			if c.valid && err != nil {
				t.Errorf("esperado válido, got: %v", err)
			}
			if !c.valid && err == nil {
				t.Errorf("esperado erro (v150=%q v160=%q), got nil", c.v150, c.v160)
			}
		})
	}
}

// ============================================================
// F02 Datas
// ============================================================

func TestF02_Datas_Validas(t *testing.T) {
	cases := []string{"2020-08", "2020-08-15", "2024-12-31"}
	for _, dt := range cases {
		dt := dt
		t.Run(dt, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Doc3040 DtBase="` + dt + `" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`
			doc := mustParse(t, xml)
			err := rules.F02Datas{}.Apply(context.Background(), doc)
			if err != nil {
				t.Errorf("DtBase=%q deveria passar, got: %v", dt, err)
			}
		})
	}
}

func TestF02_Datas_Invalidas(t *testing.T) {
	cases := []string{"", "20-08", "2020/08", "2020-13", "abc"}
	for _, dt := range cases {
		dt := dt
		t.Run(dt, func(t *testing.T) {
			xml := `<?xml version="1.0"?>
<Doc3040 DtBase="` + dt + `" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`
			doc := mustParse(t, xml)
			err := rules.F02Datas{}.Apply(context.Background(), doc)
			if err == nil {
				t.Errorf("DtBase=%q deveria falhar", dt)
			}
		})
	}
}

// ============================================================
// B06 Remessa
// ============================================================

func TestB06_Remessa_Valida(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`
	doc := mustParse(t, xml)
	err := rules.B06RemessaIncompativel{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("Remessa=1 deveria passar, got: %v", err)
	}
}

func TestB06_Remessa_Invalida(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="0" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`
	doc := mustParse(t, xml)
	err := rules.B06RemessaIncompativel{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("Remessa=0 deveria falhar")
	}
}

func TestB06_Remessa_MalFormada(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="abc" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`
	doc := mustParse(t, xml)
	err := rules.B06RemessaIncompativel{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("Remessa='abc' deveria falhar (não numérico)")
	}
}

// ============================================================
// C01 PJ obrigatórios
// ============================================================

func TestC01_PJ_QtdCliZero(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="2" DesempOp="01" ProvConsttd="0" QtdOp="5" QtdCli="0"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.C01CamposObrigatoriosPJ{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("TpCli=2 (PJ) com QtdCli=0 deveria falhar")
	}
}

func TestC01_PJ_QtdCliValido(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="5">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="2" DesempOp="01" ProvConsttd="0" QtdOp="5" QtdCli="5"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.C01CamposObrigatoriosPJ{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("TpCli=2 com QtdCli=5 deveria passar, got: %v", err)
	}
}

func TestC01_PF_Skip(t *testing.T) {
	// TpCli=1 (PF) — C01 não aplica
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="5" QtdCli="0"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.C01CamposObrigatoriosPJ{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("TpCli=1 (PF) C01 não deve aplicar, got: %v", err)
	}
}

// ============================================================
// S01 Detalhamento Cliente
// ============================================================

func TestS01_QtdCli1_QtdOp10(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="10" QtdCli="1"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S01DetalhamentoCliente{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("QtdCli=1 + QtdOp=10 deveria falhar (operação deveria ser individualizada)")
	}
}

func TestS01_QtdCli1_QtdOp1(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S01DetalhamentoCliente{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("QtdCli=1 + QtdOp=1 deveria passar, got: %v", err)
	}
}

// TestS01_QtdCliMalFormado valida o fix v1.3.6: parseNum aceita QtdCli="abc"
// (parseNum retorna 0, regra pula silenciosamente — comportamento defensivo).
func TestS01_QtdCliMalFormado(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="5">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="10" QtdCli="abc"></Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S01DetalhamentoCliente{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("QtdCli='abc' deve ser tratado como 0 (parseNum defensivo), got: %v", err)
	}
}

// ============================================================
// S05 Limite de Crédito
// ============================================================

func TestS05_Mod19_VencimentosLimite(t *testing.T) {
	// Mod=19 só aceita v110/v120. v150=0 v160=0 v165=0 deve passar.
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="19" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="0" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S05LimiteCredito{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("Mod=19 com v150=0 deveria passar, got: %v", err)
	}
}

func TestS05_Mod19_VencimentoIndevido(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="19" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="500" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S05LimiteCredito{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("Mod=19 com v150=500 deveria falhar")
	}
}

// TestS05_Mod19_DecimalInteiro valida o fix v1.3.6: parseNum detecta decimais
// (regressão: strconv.Atoi ignoraria "500.50" como zero).
func TestS05_Mod19_DecimalInteiro(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="19" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="500.50" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S05LimiteCredito{}.Apply(context.Background(), doc)
	if err == nil {
		t.Error("Mod=19 com v150='500.50' (decimal) deveria falhar — parseNum detecta")
	}
}

func TestS05_ModDiferente19_Skip(t *testing.T) {
	// Mod != 19 — regra não aplica
	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="500" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`
	doc := mustParse(t, xml)
	err := rules.S05LimiteCredito{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("S05 não deve aplicar a Mod=0213, got: %v", err)
	}
}

// ============================================================
// Registry: todas as 25 regras registradas
// ============================================================

func TestBuiltin3040_RegistryCompleto(t *testing.T) {
	r := rules.Builtin3040()
	codes := r.Codes()

	// Sprint 7b / v1.7.0: 60 regras — 5 raw + 55 tipadas.
	// Sprint 32 Fase 1: +14 Agregadas → 74 regras.
	// Sprint 32 Fase 2: +5 Sistemáticas → 79 regras.
	// Sprint 32 Fase 3: +19 Individuais/Campos/Header → 98 regras.
	expectedCodigos := []string{
		// Básicas raw (Sprint 6 v1.5.0 / W3)
		"B01", "B02", "B03", "B04", "B05",
		// Básicas tipadas (Sprint 4)
		"B06", "B07", "B08", "B09", "B10",
		"B11", "B12", "B13", "B14", "B15",
		// Básicas expandidas (Sprint 7b / v1.7.0)
		"B16", "B17", "B18", "B19", "B20",
		"B21", "B22", "B23", "B24", "B25",
		// Formato (Sprint 4)
		"F01", "F02", "F03", "F04", "F05",
		// Formato expandido (Sprint 7b)
		"F06", "F07", "F08", "F09", "F10",
		"F11", "F12", "F13", "F14", "F15",
		// Campos Obrigatórios (Sprint 4)
		"C01", "C02", "C03", "C04", "C05",
		// Campos Obrigatórios expandidos (Sprint 7b)
		"C06", "C07", "C08", "C09", "C10",
		// Semântica (Sprint 4)
		"S01", "S02", "S03", "S04", "S05",
		// Semântica expandida (Sprint 7b)
		"S06", "S07", "S08", "S09", "S10",
		// Agregadas (Sprint 32 Fase 1)
		// A08 não consta no catálogo BACEN scr3040_criticas
		"A01", "A02", "A03", "A04", "A05", "A06", "A07",
		"A09", "A10", "A11", "A12", "A13", "A14", "A15",
		// Sistemáticas (Sprint 32 Fase 2)
		// S11, S16, S18 não implementadas (gaps no catálogo)
		"S12", "S15", "S17", "S19", "S20",
		// Sprint 32 Fase 3 — Individuais/Campos/Header (19 regras)
		// C21, C23-C29 carry-over (precisam Garantidores/Parcelas completos)
		// I06-I10, I12-I15 carry-over (precisam Cli.IPOC, somatórios)
		// H04-H09 carry-over (histórico de envios)
		"C11", "C13", "C14", "C16", "C17", "C18", "C19", "C20",
		"S13", "S14",
		"I01", "I02", "I03", "I04", "I05", "I11",
		"H01", "H02", "H03",
	}

	if len(codes) != len(expectedCodigos) {
		t.Errorf("registry tem %d regras, esperado %d", len(codes), len(expectedCodigos))
	}

	codeSet := make(map[string]bool)
	for _, c := range codes {
		codeSet[c] = true
	}

	for _, exp := range expectedCodigos {
		if !codeSet[exp] {
			t.Errorf("regra %s não está no registry", exp)
		}
	}
}

func TestBuiltin3040_Severity(t *testing.T) {
	r := rules.Builtin3040()

	// S04, S05, F02, B06, C01 — severidade E (Erro)
	// C02 — severidade A (Aviso)
	// B09, B10 — severidade I (Informativo)
	testCases := []struct {
		code     string
		severity string
	}{
		{"S04", "E"},
		{"S05", "E"},
		{"F02", "E"},
		{"B06", "E"},
		{"C01", "E"},
		{"C02", "A"},
		{"B09", "I"},
		{"B10", "I"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.code, func(t *testing.T) {
			rule := r.Get(tc.code)
			if rule == nil {
				t.Fatalf("regra %s não encontrada", tc.code)
			}
			if rule.Severity() != tc.severity {
				t.Errorf("%s.Severity() = %q, want %q", tc.code, rule.Severity(), tc.severity)
			}
		})
	}
}
