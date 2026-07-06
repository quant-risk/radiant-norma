// Tests para Sprint 32 Fase 3 — Individuais/Campos/Header (19 regras)
//
// Pattern: table-driven com casos positivos (regra aceita) e negativos (regra rejeita).
package rules

import (
	"context"
	"strings"
	"testing"
)

// docValidoBaseOp retorna Doc3040 com 1 Agregado + 1 Operacao válidos.
// Base para testes de regras que operam em Operacao.
func docValidoBaseOp() *Doc3040 {
	return &Doc3040{
		Root: Doc3040Root{
			DtBase:   "2024-12",
			CNPJ:     "12345678",
			Remessa:  "1",
			Parte:    "1",
			TpArq:    "F",
			TotalCli: "10",
		},
		Agregados: []Agregado{
			{
				NatuOp:      "01",
				Mod:         "0210",
				OrigemRec:   "1",
				VincME:      "N",
				ClassOp:     "AA",
				FaixaVlr:    "1",
				PrzProvm:    "N",
				Localiz:     "SP",
				TpCli:       "1",
				DesempOp:    "01",
				ProvConsttd: "100",
				QtdOp:       "10",
				QtdCli:      "10",
				Vencimentos: Vencimentos{V110: "100"},
			},
		},
		Operacoes: []Operacao{
			{
				Inf:         "0101",
				Contrt:      "C1",
				IPOC:        "12345",
				Valor:       "10000",
				DtContr:     "2024-01-15",
				DtVencOp:    "2024-12-31",
				ClassOp:     "AA",
				ProvConsttd: "100",
				Vencimentos: Vencimentos{V110: "10000"},
				Cli:         &Cli{Cd: "12345678901", TpCli: "1", IPOC: "12345"},
			},
		},
	}
}

// ============================================================
// C11-C20 tests (Campos Obrigatórios por Inf)
// ============================================================

func TestC11_DtVencObrigatoria(t *testing.T) {
	tests := []struct {
		nome     string
		dtVenc   string
		querErro bool
	}{
		{"DtVencOp presente OK", "2024-12-31", false},
		{"DtVencOp vazia ERRO", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].DtVencOp = tt.dtVenc
			err := C11DtVencObrigatoria{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestC13_C14_C16_C17_C18_C19_C20_PorInf(t *testing.T) {
	// Testa todos juntos: pattern similar — exige campos baseado em Inf.
	tests := []struct {
		nome      string
		inf       string
		contrt    string
		ipoc      string
		valor     string
		perc      string
		querErro  bool
		rule      Rule
		expectMsg string
	}{
		// C13 — Inf 0303/0304
		{"C13 0303 OK", "0303", "C", "123", "100", "", false, C13Inf0303Cessao{}, ""},
		{"C13 0303 sem Contrt ERRO", "0303", "", "123", "100", "", true, C13Inf0303Cessao{}, "Cd"},
		{"C13 0304 sem Valor ERRO", "0304", "C", "123", "", "", true, C13Inf0303Cessao{}, "Valor"},
		{"C13 outro Inf não aplica", "0101", "", "", "", "", false, C13Inf0303Cessao{}, ""},
		// C14 — Inf 0305
		{"C14 0305 OK", "0305", "C", "123", "100", "", false, C14Inf0305Renegociacao{}, ""},
		{"C14 0305 sem Contrt ERRO", "0305", "", "123", "100", "", true, C14Inf0305Renegociacao{}, "obrigatórios"},
		// C16 — Inf 0307
		{"C16 0307 OK", "0307", "C", "123", "", "", false, C16Inf0307{}, ""},
		{"C16 0307 sem Contrt ERRO", "0307", "", "123", "", "", true, C16Inf0307{}, "obrigatórios"},
		// C17 — Inf 04XX
		{"C17 0401 OK", "0401", "C", "", "", "", false, C17Inf04XX{}, ""},
		{"C17 0401 sem Contrt ERRO", "0401", "", "", "", "", true, C17Inf04XX{}, "instrumento"},
		{"C17 outro Inf não aplica", "0101", "", "", "", "", false, C17Inf04XX{}, ""},
		// C18 — Inf 05XX
		{"C18 0501 OK", "0501", "C", "", "", "", false, C18Inf05XX{}, ""},
		{"C18 0501 sem Contrt ERRO", "0501", "", "", "", "", true, C18Inf05XX{}, "obrigatório"},
		// C19 — Inf 0701
		{"C19 0701 OK", "0701", "C", "", "100", "50", false, C19Inf0701{}, ""},
		{"C19 0701 sem Perc ERRO", "0701", "C", "", "100", "", true, C19Inf0701{}, "obrigatórios"},
		// C20 — Inf 0702/0703/0704
		{"C20 0702 OK", "0702", "C", "123", "100", "50", false, C20Inf0702{}, ""},
		{"C20 0703 sem Valor ERRO", "0703", "C", "123", "", "50", true, C20Inf0702{}, "obrigatórios"},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
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

// ============================================================
// S13, S14 tests (Sistemáticas individuais)
// ============================================================

func TestS13_GarantidorNaoCliente(t *testing.T) {
	tests := []struct {
		nome     string
		tpCli    string
		cliCd    string
		gars     []string
		querErro bool
	}{
		{"PF sem garantidor OK", "1", "12345678901", nil, false},
		{"PF garantidor ≠ cliente OK", "1", "12345678901", []string{"99999999999"}, false},
		{"PF garantidor = cliente ERRO", "1", "12345678901", []string{"12345678901"}, true},
		{"PJ não aplica constraint", "2", "12345678", []string{"12345678"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].Cli = &Cli{Cd: tt.cliCd, TpCli: tt.tpCli}
			doc.Operacoes[0].Garantidores = tt.gars
			err := S13GarantidorNaoCliente{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestS14_DtVencMaiorDtContr(t *testing.T) {
	tests := []struct {
		nome     string
		dtContr  string
		dtVenc   string
		querErro bool
	}{
		{"DtVenc > DtContr OK", "2024-01-15", "2024-12-31", false},
		{"DtVenc = DtContr OK (boundary)", "2024-01-15", "2024-01-15", false},
		{"DtVenc < DtContr ERRO", "2024-12-31", "2024-01-15", true},
		{"DtContr vazio skip", "", "2024-12-31", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].DtContr = tt.dtContr
			doc.Operacoes[0].DtVencOp = tt.dtVenc
			err := S14DtVencMaiorDtContr{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

// ============================================================
// I01-I05 tests (Individualizadas)
// ============================================================

func TestI01_ClassOpProvisaoIndividual(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		prov     string
		venc     string
		inf      string
		querErro bool
	}{
		{"AA prov=49 venc=10000 ratio=0.0049 OK", "AA", "49", "10000", "0101", false},
		{"AA prov=50 venc=10000 ratio=0.005 boundary ERRO (provMax exclusive)", "AA", "50", "10000", "0101", true},
		{"AA prov=51 venc=10000 ratio=0.0051 ERRO", "AA", "51", "10000", "0101", true},
		{"Modalidade 19XX isenta", "AA", "1000000", "10000", "1901", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].ClassOp = tt.classOp
			doc.Operacoes[0].ProvConsttd = tt.prov
			doc.Operacoes[0].Vencimentos.V110 = tt.venc
			doc.Operacoes[0].Inf = tt.inf
			err := I01ClassOpProvisaoIndividual{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestI02_ClassOpVencIndividual(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		venc     string
		querErro bool
	}{
		{"AA venc=200 (210 max) OK", "AA", "200", false},
		{"AA venc=210 boundary ERRO", "AA", "210", true},
		{"D venc=359 (360 max) OK", "D", "359", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].ClassOp = tt.classOp
			doc.Operacoes[0].Vencimentos.V110 = tt.venc
			err := I02ClassOpVencIndividual{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestI03_CliTpCliUnico(t *testing.T) {
	doc := docValidoBaseOp()
	// Adicionar 2ª operação com mesmo Cli
	doc.Operacoes = append(doc.Operacoes, Operacao{
		Inf:    "0101",
		Contrt: "C2",
		Cli:    &Cli{Cd: "12345678901", TpCli: "1", IPOC: "99999"},
	})
	err := I03CliTpCliUnico{}.Apply(context.Background(), doc)
	if err == nil {
		t.Fatal("esperava erro de duplicata")
	}
	if !strings.Contains(err.Error(), "duplicata") {
		t.Errorf("erro deve mencionar 'duplicata': %v", err)
	}

	// Sem duplicata OK
	doc2 := docValidoBaseOp()
	doc2.Operacoes = append(doc2.Operacoes, Operacao{
		Inf:    "0101",
		Contrt: "C2",
		Cli:    &Cli{Cd: "99999999999", TpCli: "1", IPOC: "99999"},
	})
	err = I03CliTpCliUnico{}.Apply(context.Background(), doc2)
	if err != nil {
		t.Errorf("Clis diferentes não devem colidir: %v", err)
	}
}

func TestI04_ContratoModalidadeUnico(t *testing.T) {
	doc := docValidoBaseOp()
	doc.Operacoes = append(doc.Operacoes, Operacao{
		Inf:    "0101",
		Contrt: "C1",
		IPOC:   "12345", // mesmo Contrt+IPOC
	})
	err := I04ContratoModalidadeUnico{}.Apply(context.Background(), doc)
	if err == nil {
		t.Fatal("esperava erro de duplicata")
	}
}

func TestI05_VencimentosUnicos(t *testing.T) {
	tests := []struct {
		nome     string
		v110     string
		v120     string
		querErro bool
	}{
		{"V110=100 V120=200 OK", "100", "200", false},
		{"V110=100 V120=100 ERRO", "100", "100", true},
		{"V110=0 V120=0 OK (zero é vazio)", "0", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Operacoes[0].Vencimentos.V110 = tt.v110
			doc.Operacoes[0].Vencimentos.V120 = tt.v120
			err := I05VencimentosUnicos{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestI11_StubPassThrough(t *testing.T) {
	// I11 é stub — Fase 4 precisa NatuOp em Operacao
	doc := docValidoBaseOp()
	err := I11CliNaoNatuOp32{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("I11 stub deveria passar: %v", err)
	}
}

// ============================================================
// H01-H03 tests (Header)
// ============================================================

func TestH01_TpArqValido(t *testing.T) {
	tests := []struct {
		nome     string
		tpArq    string
		querErro bool
	}{
		{"TpArq=F OK", "F", false},
		{"TpArq=S OK", "S", false},
		{"TpArq=X ERRO", "X", true},
		{"TpArq vazio ERRO", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Root.TpArq = tt.tpArq
			err := H01TpArqValido{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestH02_CNPJRaiz(t *testing.T) {
	tests := []struct {
		nome     string
		cnpj     string
		querErro bool
	}{
		{"CNPJ=12345678 OK", "12345678", false},
		{"CNPJ=1234 (curto) ERRO", "1234", true},
		{"CNPJ=123456789 (longo) ERRO", "123456789", true},
		{"CNPJ=1234abcd (não-num) ERRO", "1234abcd", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Root.CNPJ = tt.cnpj
			err := H02CNPJRaiz{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestH03_TotalCliPositivo(t *testing.T) {
	tests := []struct {
		nome     string
		totalCli string
		querErro bool
	}{
		{"TotalCli=10 OK", "10", false},
		{"TotalCli=0 ERRO", "0", true},
		{"TotalCli=-1 ERRO", "-1", true},
		{"TotalCli vazio ERRO (parse)", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseOp()
			doc.Root.TotalCli = tt.totalCli
			err := H03TotalCliPositivo{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

// TestRegistry_Fase3 verifica que todas as 19 regras estão registradas.
func TestRegistry_Fase3(t *testing.T) {
	r := NewRegistry()
	regras := []Rule{
		C11DtVencObrigatoria{}, C13Inf0303Cessao{}, C14Inf0305Renegociacao{},
		C16Inf0307{}, C17Inf04XX{}, C18Inf05XX{}, C19Inf0701{}, C20Inf0702{},
		S13GarantidorNaoCliente{}, S14DtVencMaiorDtContr{},
		I01ClassOpProvisaoIndividual{}, I02ClassOpVencIndividual{},
		I03CliTpCliUnico{}, I04ContratoModalidadeUnico{}, I05VencimentosUnicos{},
		I11CliNaoNatuOp32{},
		H01TpArqValido{}, H02CNPJRaiz{}, H03TotalCliPositivo{},
	}
	for _, regra := range regras {
		r.Register(regra)
	}
	codes := r.Codes()
	esperados := []string{
		"C11", "C13", "C14", "C16", "C17", "C18", "C19", "C20",
		"S13", "S14",
		"I01", "I02", "I03", "I04", "I05", "I11",
		"H01", "H02", "H03",
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
