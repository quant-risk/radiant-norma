// Tests para regras Agregadas (A01-A15) — Sprint 32 Fase 1.
//
// Padrão table-driven: cada teste invoca N sub-cases (positive + negative + edge).
package rules

import (
	"context"
	"strings"
	"testing"
)

// docValidoBase retorna um Doc3040 com 1 agregado válido pronto para usar
// como base de testes. Cada teste copia e muta conforme o caso.
func docValidoBase() *Doc3040 {
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
				NatuOp:      "01", // própria
				Mod:         "0210",
				OrigemRec:   "1",
				VincME:      "N",
				ClassOp:     "AA",
				FaixaVlr:    "1",
				PrzProvm:    "N",
				Localiz:     "SP",
				TpCli:       "1", // PF
				DesempOp:    "01",
				ProvConsttd: "100",
				QtdOp:       "10",
				QtdCli:      "10",
				Vencimentos: Vencimentos{
					V110: "100", // até 14 dias
				},
			},
		},
	}
}

func TestA01_ClassOpProvisao(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		prov     string
		venc     string
		querErro bool
	}{
		{"AA prov=100, venc=100000 → ratio=0.001 (0..0.005) OK", "AA", "100", "100000", false},
		{"AA prov=499, venc=100000 → ratio=0.00499 OK", "AA", "499", "100000", false},
		{"AA prov=600, venc=100000 → ratio=0.006 > 0.005 ERRO", "AA", "600", "100000", true},
		{"A prov=999, venc=100000 → ratio=0.00999 OK", "A", "999", "100000", false},
		{"A prov=1500, venc=100000 → ratio=0.015 > 0.01 ERRO", "A", "1500", "100000", true},
		{"H prov=150000, venc=100000 → ratio=1.5 (>=1.0) OK", "H", "150000", "100000", false},
		{"H prov=50000, venc=100000 → ratio=0.5 ERRO (H exige >=1.0)", "H", "50000", "100000", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].ClassOp = tt.classOp
			doc.Agregados[0].ProvConsttd = tt.prov
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := A01ClassOpProvisao{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA02_ClassOpVencSemPrazo(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		venc     string
		querErro bool
	}{
		{"AA venc=100 (210 max) OK", "AA", "100", false},
		{"AA venc=210 (boundary >=210) ERRO", "AA", "210", true},
		{"B venc=239 (240 max) OK", "B", "239", false},
		{"B venc=240 (boundary) ERRO", "B", "240", true},
		{"D venc=359 (360 max) OK", "D", "359", false},
		{"D venc=360 ERRO", "D", "360", true},
		{"H venc=500 (360 max) ERRO", "H", "500", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].ClassOp = tt.classOp
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := A02ClassOpVencSemPrazo{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA03_ClassOpVencComPrazo(t *testing.T) {
	tests := []struct {
		nome     string
		classOp  string
		prz      string
		venc     string
		querErro bool
	}{
		{"AA PrzProvm=S venc=219 (220 max) OK", "AA", "S", "219", false},
		{"AA PrzProvm=S venc=220 ERRO", "AA", "S", "220", true},
		{"AA PrzProvm=N (sem prazo) venc=500 OK", "AA", "N", "500", false},
		{"B PrzProvm=S venc=249 (250 max) OK", "B", "S", "249", false},
		{"B PrzProvm=S venc=250 ERRO", "B", "S", "250", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].ClassOp = tt.classOp
			doc.Agregados[0].PrzProvm = tt.prz
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := A03ClassOpVencComPrazo{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA04_MinimoVencimento(t *testing.T) {
	doc := docValidoBase()
	// Base tem V110=100 → ok
	err := A04MinimoVencimento{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("base deveria passar: %v", err)
	}
	// Zerar todos vencimentos
	doc.Agregados[0].Vencimentos = Vencimentos{}
	err = A04MinimoVencimento{}.Apply(context.Background(), doc)
	if err == nil {
		t.Errorf("agregado sem vencimentos deveria falhar")
	}
}

func TestA05_NatuOpLocaliz(t *testing.T) {
	tests := []struct {
		nome     string
		natuOp   string
		localiz  string
		querErro bool
	}{
		{"NatuOp=01 Localiz=SP OK", "01", "SP", false},
		{"NatuOp=32 Localiz=10100 OK", "32", "10100", false},
		{"NatuOp=32 Localiz=SP ERRO (deveria ser 10100)", "32", "SP", true},
		{"NatuOp=01 Localiz=10100 OK (1a demais)", "01", "10100", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].NatuOp = tt.natuOp
			doc.Agregados[0].Localiz = tt.localiz
			err := A05NatuOpLocaliz{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA06_DesempOpVenc(t *testing.T) {
	tests := []struct {
		nome     string
		desempOp string
		venc     string
		querErro bool
	}{
		{"DesempOp=01 venc=200 (até 205) OK", "01", "200", false},
		{"DesempOp=01 venc=206 ERRO", "01", "206", true},
		{"DesempOp=02 venc=15 (>=15) OK", "02", "15", false},
		{"DesempOp=02 venc=10 (<15) ERRO", "02", "10", true},
		{"DesempOp=03 venc=500 OK (sem regra)", "03", "500", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].DesempOp = tt.desempOp
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := A06DesempOpVenc{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA07_AgregadoDuplicado(t *testing.T) {
	doc := docValidoBase()
	// Adicionar segundo agregado idêntico
	doc.Agregados = append(doc.Agregados, doc.Agregados[0])
	err := A07AgregadoDuplicado{}.Apply(context.Background(), doc)
	if err == nil {
		t.Fatal("esperava erro de duplicata")
	}
	if !strings.Contains(err.Error(), "duplicata") {
		t.Errorf("erro deve mencionar 'duplicata': %v", err)
	}

	// Sem duplicata OK
	doc2 := docValidoBase()
	doc2.Agregados[0].ClassOp = "AA"
	doc2.Agregados = append(doc2.Agregados, Agregado{
		NatuOp: "01", Mod: "0210", ClassOp: "B", Vencimentos: Vencimentos{V110: "100"},
	})
	err = A07AgregadoDuplicado{}.Apply(context.Background(), doc2)
	if err != nil {
		t.Errorf("agregados diferentes não devem colidir: %v", err)
	}
}

func TestA09_FaixaVlrMedia(t *testing.T) {
	tests := []struct {
		nome     string
		qtdOp    string
		venc     string
		faixaEsp string
		querErro bool
	}{
		{"10 ops, 1000 total → media 100 → faixa 1 OK", "10", "1000", "1", false},
		{"10 ops, 1000 total → media 100 → faixa 2 ERRO", "10", "1000", "2", true},
		{"10 ops, 100000 total → media 10000 → faixa 2 OK", "10", "100000", "2", false},
		{"10 ops, 1000000 total → media 100000 → faixa 3 OK", "10", "1000000", "3", false},
		{"10 ops, 10000000 total → media 1000000 → faixa 4 OK", "10", "10000000", "4", false},
		{"10 ops, 100000000 total → media 10000000 → faixa 5 OK", "10", "100000000", "5", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].QtdOp = tt.qtdOp
			doc.Agregados[0].QtdCli = tt.qtdOp // satisfazer A10
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			doc.Agregados[0].FaixaVlr = tt.faixaEsp
			err := A09FaixaVlrMedia{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA10_QtdOpMaiorQtdCli(t *testing.T) {
	tests := []struct {
		nome     string
		qtdOp    string
		qtdCli   string
		querErro bool
	}{
		{"10 ops 10 cli OK", "10", "10", false},
		{"10 ops 5 cli OK (cliente multi-op)", "10", "5", false},
		{"5 ops 10 cli ERRO", "5", "10", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].QtdOp = tt.qtdOp
			doc.Agregados[0].QtdCli = tt.qtdCli
			err := A10QtdOpMaiorQtdCli{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA11A12_FaixaAltVencMedio(t *testing.T) {
	tests := []struct {
		nome     string
		qtdOp    string
		venc     string
		faixa    string
		querErro bool
	}{
		{"faixa 4 com media 1M OK", "10", "10000000", "4", false},
		{"faixa 4 com media 100k ERRO", "10", "1000000", "4", true},
		{"faixa 1 com media 100 OK", "10", "1000", "1", false},
		{"faixa 5 com media 10M OK", "10", "100000000", "5", false},
		{"faixa 5 com media 1M ERRO", "10", "10000000", "5", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].QtdOp = tt.qtdOp
			doc.Agregados[0].QtdCli = tt.qtdOp
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			doc.Agregados[0].FaixaVlr = tt.faixa
			err := A11FaixaAltVencMedioBaixo{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA13_RiscoMedioMin(t *testing.T) {
	tests := []struct {
		nome     string
		natuOp   string
		qtdOp    string
		venc     string
		querErro bool
	}{
		{"risco 100 (PF, <200) aviso", "01", "10", "1000", true},
		{"risco 300 (PF, >=200) OK", "01", "10", "3000", false},
		{"risco 50 (exterior NatuOp=32) isento", "32", "10", "500", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].NatuOp = tt.natuOp
			doc.Agregados[0].QtdOp = tt.qtdOp
			doc.Agregados[0].QtdCli = tt.qtdOp
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := A13RiscoMedioMin{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA14_LocalizExterior(t *testing.T) {
	tests := []struct {
		nome     string
		natuOp   string
		localiz  string
		querErro bool
	}{
		{"NatuOp=01 Localiz=SP OK", "01", "SP", false},
		{"NatuOp=32 Localiz=10100 OK", "32", "10100", false},
		{"NatuOp=32 Localiz=SP05 ERRO (5 chars mas não numérico)", "32", "SP05", true},
		{"NatuOp=32 Localiz=1000 ERRO (4 chars)", "32", "1000", true},
		{"NatuOp=32 Localiz=20000 OK", "32", "20000", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].NatuOp = tt.natuOp
			doc.Agregados[0].Localiz = tt.localiz
			err := A14LocalizExterior{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("inesperado erro: %v", err)
			}
		})
	}
}

func TestA15_AgregadoDuplicadoCompleto(t *testing.T) {
	// A15 delega à A07 na Fase 1; teste mínimo só verifica que não panic
	doc := docValidoBase()
	_ = A15AgregadoDuplicadoCompleto{}.Apply(context.Background(), doc)
}

// TestRegistryAgregadas verifies that all A01-A15 rules can be added to registry.
func TestRegistry_Agregadas(t *testing.T) {
	r := NewRegistry()
	regras := []Rule{
		A01ClassOpProvisao{},
		A02ClassOpVencSemPrazo{},
		A03ClassOpVencComPrazo{},
		A04MinimoVencimento{},
		A05NatuOpLocaliz{},
		A06DesempOpVenc{},
		A07AgregadoDuplicado{},
		A09FaixaVlrMedia{},
		A10QtdOpMaiorQtdCli{},
		A11FaixaAltVencMedioBaixo{},
		A12FaixaAltRiscoMedio{},
		A13RiscoMedioMin{},
		A14LocalizExterior{},
		A15AgregadoDuplicadoCompleto{},
	}
	for _, regra := range regras {
		r.Register(regra)
	}
	// Verifica que todas foram registradas
	codes := r.Codes()
	esperados := []string{"A01", "A02", "A03", "A04", "A05", "A06", "A07", "A09", "A10", "A11", "A12", "A13", "A14", "A15"}
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

// TestClassOpInA01Range — Validação 51 (F-S28-51-C): helper exposto na
// Sprint 32 e agora reusado por F06ClassOpValido (single source of truth).
// Sprint 32 Fase 2: HH adicionado à tabela (S20 — Vencimentos HH).
func TestClassOpInA01Range(t *testing.T) {
	validas := []string{"AA", "A", "B", "C", "D", "E", "F", "G", "H", "HH"}
	for _, c := range validas {
		if !ClassOpInA01Range(c) {
			t.Errorf("ClassOp %q deveria estar na tabela A01", c)
		}
	}
	invalidas := []string{"", "X", "Z", "9", "AA1", " a", "AAA"}
	for _, c := range invalidas {
		if ClassOpInA01Range(c) {
			t.Errorf("ClassOp %q NÃO deveria estar na tabela A01", c)
		}
	}
}

// TestF06_ReusaClassOpInA01Range — Validação 51: F06 agora reusa tabela A01
// em vez de regex hardcoded. Sprint 32 Fase 2: HH agora aceito.
func TestF06_ReusaClassOpInA01Range(t *testing.T) {
	tests := []struct {
		classOp  string
		querErro bool
	}{
		{"A", false}, {"B", false}, {"H", false}, {"HH", false}, // HH adicionado
		{"X", true}, {"9", true}, {"", true},
	}
	for _, tt := range tests {
		t.Run(tt.classOp, func(t *testing.T) {
			doc := docValidoBase()
			doc.Agregados[0].ClassOp = tt.classOp
			err := F06ClassOpValido{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("ClassOp=%q deveria dar erro", tt.classOp)
			}
			if !tt.querErro && err != nil {
				t.Errorf("ClassOp=%q não deveria dar erro: %v", tt.classOp, err)
			}
		})
	}
}
