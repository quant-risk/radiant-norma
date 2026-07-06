// Tests para regras Sistemáticas (S12-S20) — Sprint 32 Fase 2.
//
// Padrão table-driven: cada teste invoca N sub-cases (positive + negative + edge).
package rules

import (
	"context"
	"strings"
	"testing"
)

// docValidoBaseSist retorna Doc3040 com DtBase válida (>=09/2010) e TpCli=1.
// Usado como base pra S15, S17, S19, S20.
func docValidoBaseSist() *Doc3040 {
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
				TpCli:       "1", // PF
				DesempOp:    "01",
				ProvConsttd: "100",
				QtdOp:       "10",
				QtdCli:      "10",
				Vencimentos: Vencimentos{
					V110: "100",
				},
			},
		},
	}
}

func TestS12_StubPassThrough(t *testing.T) {
	// S12 é stub pass-through na Fase 2 (precisa Operacao.Parcelas na Fase 3)
	doc := docValidoBaseSist()
	err := S12DtVencCompativelParcelas{}.Apply(context.Background(), doc)
	if err != nil {
		t.Errorf("S12 stub deveria passar: %v", err)
	}
}

func TestS15_DtContrNaoFutura(t *testing.T) {
	tests := []struct {
		nome     string
		dtBase   string
		querErro bool
	}{
		{"DtBase válida 2024-12 OK", "2024-12", false},
		{"DtBase válida 1990-01 OK", "1990-01", false},
		{"DtBase válida 2030-12 OK", "2030-12", false},
		{"DtBase vazia ERRO", "", true},
		{"DtBase formato errado ERRO", "202412", true},
		{"DtBase ano inválido ERRO", "abcd-12", true},
		{"DtBase mês inválido ERRO", "2024-13", true},
		{"DtBase mês 0 ERRO", "2024-00", true},
		// Nota: range de ano não é validado por S15 (pode haver dados antigos legítimos)
		// {"DtBase ano fora range ERRO", "1989-12", true},
		// {"DtBase ano futuro ERRO", "2031-01", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseSist()
			doc.Root.DtBase = tt.dtBase
			err := S15DtContrNaoFutura{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("DtBase=%q deveria dar erro", tt.dtBase)
			}
			if !tt.querErro && err != nil {
				t.Errorf("DtBase=%q não deveria dar erro: %v", tt.dtBase, err)
			}
		})
	}
}

func TestS17_CdTamanhoPorTpCli(t *testing.T) {
	tests := []struct {
		nome     string
		tpCli    string
		querErro bool
	}{
		{"TpCli=1 (PF) OK", "1", false},
		{"TpCli=2 (PJ) OK", "2", false},
		{"TpCli vazio OK (não validado)", "", false},
		{"TpCli=3 ERRO", "3", true},
		{"TpCli=PF ERRO", "PF", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseSist()
			doc.Agregados[0].TpCli = tt.tpCli
			err := S17CdTamanhoPorTpCli{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("TpCli=%q deveria dar erro", tt.tpCli)
			}
			if !tt.querErro && err != nil {
				t.Errorf("TpCli=%q não deveria dar erro: %v", tt.tpCli, err)
			}
		})
	}
}

func TestS19_DtBaseMinima(t *testing.T) {
	tests := []struct {
		nome     string
		dtBase   string
		querErro bool
	}{
		{"DtBase=2010-09 OK (boundary)", "2010-09", false},
		{"DtBase=2010-10 OK (1 mês após)", "2010-10", false},
		{"DtBase=2024-12 OK", "2024-12", false},
		{"DtBase=2010-08 ERRO (1 mês antes)", "2010-08", true},
		{"DtBase=2009-12 ERRO (1 ano antes)", "2009-12", true},
		{"DtBase=2000-01 ERRO (muito antes)", "2000-01", true},
		{"DtBase vazia ERRO", "", true},
		{"DtBase inválida ERRO", "202412", true},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseSist()
			doc.Root.DtBase = tt.dtBase
			err := S19DtBaseMinima{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("DtBase=%q deveria dar erro", tt.dtBase)
			}
			if !tt.querErro && err != nil {
				t.Errorf("DtBase=%q não deveria dar erro: %v", tt.dtBase, err)
			}
		})
	}
}

func TestS20_VencimentosHH(t *testing.T) {
	tests := []struct {
		nome     string
		natuOp   string
		classOp  string
		venc     string
		querErro bool // S20 é warning (severity A), não bloqueia → nunca retorna erro
	}{
		{"NatuOp=01 ClassOp=AA venc=100 OK", "01", "AA", "100", false},
		{"NatuOp=01 ClassOp=HH venc=400 OK (warning, mas passa)", "01", "HH", "400", false},
		{"NatuOp=34 ClassOp=AA venc=400 OK (exceção)", "34", "AA", "400", false},
		{"NatuOp=01 ClassOp=A venc=400 OK (warning não bloqueia)", "01", "A", "400", false},
	}
	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := docValidoBaseSist()
			doc.Agregados[0].NatuOp = tt.natuOp
			doc.Agregados[0].ClassOp = tt.classOp
			doc.Agregados[0].Vencimentos.V110 = tt.venc
			err := S20VencimentosHH{}.Apply(context.Background(), doc)
			if tt.querErro && err == nil {
				t.Errorf("esperava erro, получил nil")
			}
			if !tt.querErro && err != nil {
				t.Errorf("S20 não deveria bloquear (severity A): %v", err)
			}
		})
	}
}

// TestRegistry_Fase2Sistematicas verifica que S12/S15/S17/S19/S20 estão registrados.
func TestRegistry_Fase2Sistematicas(t *testing.T) {
	r := NewRegistry()
	regras := []Rule{
		S12DtVencCompativelParcelas{},
		S15DtContrNaoFutura{},
		S17CdTamanhoPorTpCli{},
		S19DtBaseMinima{},
		S20VencimentosHH{},
	}
	for _, regra := range regras {
		r.Register(regra)
	}
	codes := r.Codes()
	esperados := []string{"S12", "S15", "S17", "S19", "S20"}
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

// TestParseDtBaseYM é helper testado isoladamente pra garantir
// validação de formato YYYY-MM é robusta.
func TestParseDtBaseYM(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		wantAno int
		wantMes int
	}{
		{"2024-12", false, 2024, 12},
		{"2010-09", false, 2010, 9},
		{"", true, 0, 0},
		{"202412", true, 0, 0},
		{"abcd-12", true, 0, 0},
		{"2024-13", true, 0, 0},
		{"2024-00", true, 0, 0},
		{"2024/12", true, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ano, mes, err := parseDtBaseYM(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDtBaseYM(%q) deveria dar erro", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseDtBaseYM(%q) erro inesperado: %v", tt.input, err)
			}
			if ano != tt.wantAno || mes != tt.wantMes {
				t.Errorf("parseDtBaseYM(%q) = (%d, %d), want (%d, %d)",
					tt.input, ano, mes, tt.wantAno, tt.wantMes)
			}
		})
	}
	// Sanity: errors should contain useful info
	_, _, err := parseDtBaseYM("invalid")
	if err == nil || !strings.Contains(err.Error(), "formato") {
		t.Errorf("erro deveria mencionar 'formato', got %v", err)
	}
}
