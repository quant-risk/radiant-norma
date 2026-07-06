// Tests para Sprint 32 Fase 4 — 28 regras finais (14 completas + 14 stubs)
package rules

import (
	"context"
	"strings"
	"testing"
)

// docValidoBaseFase4 retorna Doc3040 com Operacao válida e DtBase >= 07/2011.
// Base para testes C31-C55, S21-S46, S69-S70.
func docValidoBaseFase4() *Doc3040 {
	return &Doc3040{
		Root: Doc3040Root{
			DtBase:   "2024-12",
			CNPJ:     "12345678",
			Remessa:  "1",
			Parte:    "1",
			TpArq:    "F",
			TotalCli: "10",
		},
		Operacoes: []Operacao{
			{
				Inf:         "0101",
				Contrt:      "2024-01-15",
				IPOC:        "12345678", // CNPJ 8 dígitos
				Valor:       "10000",
				Perc:        "50",
				DtContr:     "2024-01-15",
				DtVencOp:    "2024-12-31",
				ClassOp:     "AA",
				ProvConsttd: "100",
				Vencimentos: Vencimentos{V110: "10000"},
				Cli:         &Cli{Cd: "12345678", TpCli: "2", IPOC: "12345678"},
			},
		},
	}
}

// ============================================================
// C31-C40 tests
// ============================================================

func TestC31_FaturamentoObrigatorio(t *testing.T) {
	tests := []struct {
		nome     string
		dtBase   string
		valor    string
		querErro bool
	}{
		{"DtBase 2024-12 com Valor OK", "2024-12", "10000", false},
		{"DtBase 2024-12 sem Valor ERRO", "2024-12", "", true},
		{"DtBase 2010-01 (pré-07/2011) sem Valor OK (regra não aplica)", "2010-01", "", false},
		{"DtBase 2011-06 (boundary <07) sem Valor OK", "2011-06", "", false},
		{"DtBase 2011-07 (boundary =07) sem Valor ERRO", "2011-07", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Root.DtBase = tt.dtBase
			doc.Operacoes[0].Valor = tt.valor
			err := C31FaturamentoObrigatorio{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestC32_PercIndexador(t *testing.T) {
	tests := []struct {
		nome     string
		dtBase   string
		perc     string
		querErro bool
	}{
		{"2024-12 sem Perc ERRO", "2024-12", "", true},
		{"2011-08 (pré-09) sem Perc OK", "2011-08", "", false},
		{"2011-09 (boundary =09) sem Perc ERRO", "2011-09", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Root.DtBase = tt.dtBase
			doc.Operacoes[0].Perc = tt.perc
			err := C32PercIndexadorObrigatorio{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestC33_StubPassThrough(t *testing.T) {
	doc := docValidoBaseFase4()
	err := C33DiasAtrasoObrigatorio{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("C33 stub deveria passar: %v", err)
	}
}

func TestC34_C37_C39_C40_PorInf(t *testing.T) {
	// Reune testes C34, C37, C39, C40 — pattern similar (Inf → campo)
	tests := []struct {
		nome      string
		inf       string
		contrt    string
		ipoc      string
		valor     string
		perc      string
		rule      Rule
		expectMsg string
		querErro  bool
	}{
		// C34 — Inf=1201
		{"C34 1201 com Valor+Perc OK", "1201", "C", "12345", "100", "50", C34Inf1201Coobrigacao{}, "", false},
		{"C34 1201 sem Perc ERRO", "1201", "C", "12345", "100", "", C34Inf1201Coobrigacao{}, "obrigatórios", true},
		// C37 — Inf=1202
		{"C37 1202 com Contrt OK", "1202", "C", "", "", "", C37Inf1202{}, "", false},
		{"C37 1202 sem Contrt ERRO", "1202", "", "", "", "", C37Inf1202{}, "obrigatório", true},
		// C39 — Inf=1203
		{"C39 1203 com IPOC OK", "1203", "", "12345678", "", "", C39Inf1203{}, "", false},
		{"C39 1203 sem IPOC ERRO", "1203", "", "", "", "", C39Inf1203{}, "obrigatório", true},
		// C40 — Inf=1201
		{"C40 1201 com Contrt+IPOC OK", "1201", "C", "12345", "100", "50", C40Inf1201CdIdent{}, "", false},
		{"C40 1201 sem Contrt ERRO", "1201", "", "12345", "100", "50", C40Inf1201CdIdent{}, "obrigatórios", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].Contrt = tt.contrt
			doc.Operacoes[0].IPOC = tt.ipoc
			doc.Operacoes[0].Valor = tt.valor
			doc.Operacoes[0].Perc = tt.perc
			err := tt.rule.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
			if tt.querErro && tt.expectMsg != "" && err != nil &&
				!strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("erro deveria mencionar %q, got %v", tt.expectMsg, err)
			}
		})
	}
}

func TestC35_Inf1201Obrigatorio(t *testing.T) {
	tests := []struct {
		nome     string
		inf      string
		ipoc     string
		querErro bool
	}{
		{"Inf=1201 OK (já tem Inf correto)", "1201", "12345678", false},
		{"Inf vazio, Mod=1511 → deveria ter Inf=1201 ERRO", "", "15110000", true},
		{"Inf=0101, Mod=0210 OK (não requer 1201)", "0101", "02100000", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].IPOC = tt.ipoc
			err := C35Inf1201Obrigatorio{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestC36_IdentCedente(t *testing.T) {
	tests := []struct {
		nome     string
		dtBase   string
		inf      string
		ipoc     string
		querErro bool
	}{
		{"2024-12 Inf=0101 com IPOC OK", "2024-12", "0101", "12345678", false},
		{"2024-12 Inf=0101 sem IPOC ERRO", "2024-12", "0101", "", true},
		{"2012-02 (pré-03) Inf=0101 sem IPOC OK", "2012-02", "0101", "", false},
		{"2012-03 (boundary) sem IPOC ERRO", "2012-03", "0101", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Root.DtBase = tt.dtBase
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].IPOC = tt.ipoc
			err := C36IdentCedenteObrigatorio{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestC38_StubPassThrough(t *testing.T) {
	doc := docValidoBaseFase4()
	err := C38Pacote1512{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("C38 stub deveria passar: %v", err)
	}
}

// ============================================================
// C51-C55 tests
// ============================================================

func TestC51_C52_C54_C55_PorInf(t *testing.T) {
	tests := []struct {
		nome      string
		inf       string
		contrt    string
		tpCli     string
		rule      Rule
		expectMsg string
		querErro  bool
	}{
		// C51 — Inf=0313
		{"C51 0313 com Contrt+TpCli=2 OK", "0313", "C", "2", C51Inf0313{}, "", false},
		{"C51 0313 com TpCli=7 ERRO", "0313", "C", "7", C51Inf0313{}, "1-6", true},
		// C52 — Inf=04XX exceto 0406
		{"C52 0401 com Contrt OK", "0401", "C", "2", C52Inf04Excluindo0406{}, "", false},
		{"C52 0406 sem Contrt OK (exceção)", "0406", "", "2", C52Inf04Excluindo0406{}, "", false},
		// C54 — Inf=18XX
		{"C54 1801 com Contrt OK", "1801", "C", "2", C54Inf18XX{}, "", false},
		{"C54 1801 sem Contrt ERRO", "1801", "", "2", C54Inf18XX{}, "obrigatório", true},
		// C55 — Inf=1999
		{"C55 1999 com Contrt OK", "1999", "C", "2", C55Inf1999{}, "", false},
		{"C55 1999 sem Contrt ERRO", "1999", "", "2", C55Inf1999{}, "obrigatório", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].Contrt = tt.contrt
			doc.Operacoes[0].Cli.TpCli = tt.tpCli
			err := tt.rule.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
			if tt.querErro && tt.expectMsg != "" && err != nil &&
				!strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("erro deveria mencionar %q, got %v", tt.expectMsg, err)
			}
		})
	}
}

// ============================================================
// S21-S46 tests
// ============================================================

func TestS21_Mod15SemVenc310(t *testing.T) {
	tests := []struct {
		nome     string
		ipoc     string
		venc     string
		querErro bool
	}{
		{"Mod=1511 venc=100 OK", "15110000", "100", false},
		{"Mod=1511 venc=300 (>200) ERRO", "15110000", "300", true},
		{"Mod=0210 venc=300 OK (Mod 02 não tem restrição)", "02100000", "300", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].IPOC = tt.ipoc
			doc.Operacoes[0].Vencimentos.V110 = tt.venc
			err := S21Mod15SemVenc310{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS22_Mod1511NaoPF(t *testing.T) {
	tests := []struct {
		nome     string
		ipoc     string
		tpCli    string
		querErro bool
	}{
		{"Mod=1511 PJ OK", "1511", "2", false},
		{"Mod=1511 PF ERRO", "1511", "1", true},
		{"Mod=0210 PF OK (não é 1511)", "0210", "1", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].IPOC = tt.ipoc
			doc.Operacoes[0].Cli.TpCli = tt.tpCli
			err := S22Mod1511NaoPF{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS25_CNPJCabecalhoDiferente(t *testing.T) {
	doc := docValidoBaseFase4()
	// IPOC = CNPJ (auto-cessão)
	doc.Operacoes[0].Inf = "0303"
	doc.Operacoes[0].IPOC = "12345678" // mesmo CNPJ do cabeçalho
	err := S25CNPJCabecalhoDiferente{}.Apply(context.Background(), doc)
	if err == nil {
		t.Fatal("esperava erro de auto-cessão")
	}
	if !strings.Contains(err.Error(), "auto-cessão") {
		t.Errorf("erro deve mencionar 'auto-cessão': %v", err)
	}

	// IPOC diferente OK
	doc2 := docValidoBaseFase4()
	doc2.Operacoes[0].Inf = "0303"
	doc2.Operacoes[0].IPOC = "99999999"
	err = S25CNPJCabecalhoDiferente{}.Apply(context.Background(), doc2)
	if err != nil {
		t.Errorf("IPOC diferente do cabeçalho não deve dar erro: %v", err)
	}
}

func TestS26_S33_S34_S44_S70_Stubs(t *testing.T) {
	// Stubs pass-through
	doc := docValidoBaseFase4()
	stubs := []Rule{
		S26NatuOp02TemInf{},
		S33Inf0101Natureza{},
		S34CdCessao{},
		S44CaractEsp35{},
		S70IntramesDtContr{},
	}
	for _, stub := range stubs {
		err := stub.Apply(context.Background(), doc)
		if err != nil {
			t.Errorf("stub %s deveria passar: %v", stub.Code(), err)
		}
	}
}

func TestS41_IdentCNPJ8Digitos(t *testing.T) {
	tests := []struct {
		nome     string
		inf      string
		ipoc     string
		querErro bool
	}{
		{"0101 com 8 dígitos OK", "0101", "12345678", false},
		{"0303 com 8 dígitos OK", "0303", "12345678", false},
		{"1001 com 8 dígitos OK", "1001", "12345678", false},
		{"0101 com 11 dígitos ERRO (deveria ser 8)", "0101", "12345678901", true},
		{"0101 sem Ident ERRO", "0101", "", true},
		{"0101 com letras ERRO", "0101", "abcd1234", true},
		{"0105 com 8 dígitos OK (0105 é exceção, não exige)", "0105", "12345678", false},
		{"0201 com 8 dígitos OK (não está na lista)", "0201", "12345678", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].IPOC = tt.ipoc
			err := S41IdentCNPJ8Digitos{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS42_CedenteIgualCabecalho(t *testing.T) {
	doc := docValidoBaseFase4()
	doc.Operacoes[0].Inf = "1203"
	doc.Operacoes[0].IPOC = "12345678" // mesmo CNPJ
	err := S42CedenteIgualCabecalho{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("cedente = cabeçalho OK: %v", err)
	}

	doc2 := docValidoBaseFase4()
	doc2.Operacoes[0].Inf = "1203"
	doc2.Operacoes[0].IPOC = "99999999"
	err = S42CedenteIgualCabecalho{}.Apply(context.Background(), doc2)
	if err == nil {
		t.Fatal("esperava erro (cedente ≠ cabeçalho)")
	}
}

func TestS43_CedenteIgualCliente(t *testing.T) {
	doc := docValidoBaseFase4()
	doc.Operacoes[0].Inf = "0101"
	doc.Operacoes[0].IPOC = "12345678" // mesmo Cd do Cli
	err := S43CedenteIgualCliente{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("cedente = cliente OK: %v", err)
	}

	doc2 := docValidoBaseFase4()
	doc2.Operacoes[0].Inf = "0101"
	doc2.Operacoes[0].IPOC = "99999999"
	err = S43CedenteIgualCliente{}.Apply(context.Background(), doc2)
	if err == nil {
		t.Fatal("esperava erro (cedente ≠ cliente)")
	}
}

func TestS45_IdentCPFouCNPJ(t *testing.T) {
	tests := []struct {
		nome     string
		inf      string
		ipoc     string
		querErro bool
	}{
		{"0304 com 11 dígitos OK (CPF)", "0304", "12345678901", false},
		{"0304 com 8 dígitos OK (CNPJ)", "0304", "12345678", false},
		{"0701 com 11 OK", "0701", "12345678901", false},
		{"1003 com 8 OK", "1003", "12345678", false},
		{"0304 com 10 dígitos ERRO", "0304", "1234567890", true},
		{"0304 com letras ERRO", "0304", "abcdefghijk", true},
		{"0101 (não na lista) sem validação", "0101", "abc", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].IPOC = tt.ipoc
			err := S45IdentCPFouCNPJ{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS46_CdFormatoData(t *testing.T) {
	tests := []struct {
		nome     string
		inf      string
		contrt   string
		querErro bool
	}{
		{"0101 com 2024-01-15 OK", "0101", "2024-01-15", false},
		{"0303 com 2024-01-15 OK", "0303", "2024-01-15", false},
		{"0101 com 20240115 ERRO", "0101", "20240115", true},
		{"0101 com 2024/01/15 ERRO", "0101", "2024/01/15", true},
		{"0101 com vazio OK (skip)", "0101", "", false},
		{"0201 (não na lista) com data malformada OK (não valida)", "0201", "qualquer", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].Inf = tt.inf
			doc.Operacoes[0].Contrt = tt.contrt
			err := S46CdFormatoData{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS69_ClassOpHHProvZero(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		prov     string
		querErro bool
	}{
		{"ClassOp=HH prov=0 OK", "HH", "0", false},
		{"ClassOp=HH prov=100 ERRO (deveria ser 0)", "HH", "100", true},
		{"ClassOp=AA prov=100 OK (HH é que tem restrição)", "AA", "100", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseFase4()
			doc.Operacoes[0].ClassOp = tt.classOp
			doc.Operacoes[0].ProvConsttd = tt.prov
			err := S69ClassOpHHProvZero{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

// TestRegistry_Fase4 verifica que as 28 regras estão registradas.
func TestRegistry_Fase4(t *testing.T) {
	r := NewRegistry()
	regras := []Rule{
		C31FaturamentoObrigatorio{}, C32PercIndexadorObrigatorio{}, C33DiasAtrasoObrigatorio{},
		C34Inf1201Coobrigacao{}, C35Inf1201Obrigatorio{}, C36IdentCedenteObrigatorio{},
		C37Inf1202{}, C38Pacote1512{}, C39Inf1203{}, C40Inf1201CdIdent{},
		C51Inf0313{}, C52Inf04Excluindo0406{}, C54Inf18XX{}, C55Inf1999{},
		S21Mod15SemVenc310{}, S22Mod1511NaoPF{}, S25CNPJCabecalhoDiferente{},
		S26NatuOp02TemInf{}, S33Inf0101Natureza{}, S34CdCessao{},
		S41IdentCNPJ8Digitos{}, S42CedenteIgualCabecalho{}, S43CedenteIgualCliente{},
		S44CaractEsp35{}, S45IdentCPFouCNPJ{}, S46CdFormatoData{},
		S69ClassOpHHProvZero{}, S70IntramesDtContr{},
	}
	for _, regra := range regras {
		r.Register(regra)
	}
	codes := r.Codes()
	esperados := []string{
		"C31", "C32", "C33", "C34", "C35", "C36", "C37", "C38", "C39", "C40",
		"C51", "C52", "C54", "C55",
		"S21", "S22", "S25", "S26", "S33", "S34",
		"S41", "S42", "S43", "S44", "S45", "S46",
		"S69", "S70",
	}
	for _, esp := range esperados {
		found := false
		for _, c := range codes {
			if c == esp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("regra %s não foi registrada", esp)
		}
	}
}
