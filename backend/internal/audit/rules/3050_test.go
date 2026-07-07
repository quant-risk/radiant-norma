// Tests para regras 3050 (TXB_V11) — Sprint 33 Fase 1.
//
// Padrão table-driven: cada teste invoca N sub-cases (positive + negative + edge).
package rules

import (
	"context"
	"testing"
)

// doc3050ValidoBase retorna um Doc3050 com header válido + 1 modalidade diária
// (modelo 1) + 1 modalidade mensal (modelo 1). Cada teste copia e muta.
func doc3050ValidoBase() *Doc3050 {
	f := func(v float64) *float64 { return &v }
	i := func(v int) *int { return &v }

	return &Doc3050{
		Root: Doc3050Root{
			CNPJ:       "12345678",
			DataBase:   "2024-12-31",
			IndRemessa: "I",
			NmContato:  "João da Silva",
			TelContato: "11999998888",
		},
		Diario: []Modalidade{
			{
				Codigo:  "capGirPrzAte365",
				Encargo: "pre",
				TipoCli: "pesJuridica",
				// Modelo 1
				TxMedJuros:           f(15.5),
				TxMedEncFiscais:      f(0.5),
				TxMedEncOperacionais: f(1.0),
				TxMinima:             f(10.0),
				TxMaxima:             f(25.0),
				VlrConcessoes:        f(1000000.00),
				PrzDecMedConcessoes:  i(180),
				QtdNovContratos:      i(50),
				SldCarAtiva:          f(5000000.00),
				SldCedido:            f(100000.00),
				SldAdquirido:         f(50000.00),
			},
		},
		Mensal: []Modalidade{
			{
				Codigo:         "capGirPrzAte365",
				Encargo:        "pre",
				TipoCli:        "pesJuridica",
				SldCarAtiva:    f(5000000.00),
				SldCedido:      f(100000.00),
				SldAdquirido:   f(50000.00),
				SldBaiPrejuizo: f(200000.00),
				SldCarAte14:    f(3000000.00),
				SldCarAte60:    f(1500000.00),
				SldCarAte90:    f(400000.00),
				SldCarMaior90:  f(100000.00),
				QtdConAte14:    i(30),
				QtdConAte60:    i(15),
				QtdConAte90:    i(4),
				QtdConMaior90:  i(1),
				PrzMedCarteira: i(120),
			},
		},
	}
}

// ========== A01 — SldCarAtiva = soma faixas (regra 3018) ==========

func TestA01_SldCarSomaFaixas(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string // substring esperada na msg de erro; vazia = sem erro
	}{
		{
			nome:   "happy path: soma confere",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "diff > 0.01: erro E",
			mutate: func(d *Doc3050) {
				f := 100.0
				d.Mensal[0].SldCarAte14 = &f // era 3M, agora 100 — diff > 0.01
			},
			wantErrSubs: "sldCarAtiva=",
		},
		{
			nome: "campos nil: skip (outras regras flagam)",
			mutate: func(d *Doc3050) {
				d.Mensal[0].SldCarAte14 = nil
			},
		},
		{
			nome: "diff pequeno (0.005): aceito (tolerância)",
			mutate: func(d *Doc3050) {
				f := *d.Mensal[0].SldCarAte14 + 0.005
				d.Mensal[0].SldCarAte14 = &f
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := A01SldCarSomaFaixas{}.Apply3050(context.Background(), doc)
			if tt.wantErrSubs == "" {
				if err != nil {
					t.Fatalf("esperava nil, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("esperava erro contendo %q, got nil", tt.wantErrSubs)
				}
				if !contains(err.Error(), tt.wantErrSubs) {
					t.Fatalf("esperava erro contendo %q, got %q", tt.wantErrSubs, err.Error())
				}
			}
		})
	}
}

// ========== A02 — SldCedido - SldAdquirido ≤ SldCarAtiva (regra 3019) ==========

func TestA02_SldCedidoMenosAdquirido(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy path: cedido - adquirido < sldCarAtiva",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: diff > sldCarAtiva",
			mutate: func(d *Doc3050) {
				v := 100_000_000.0 // 100M >> sldCarAtiva 5M
				d.Mensal[0].SldCedido = &v
				zero := 0.0
				d.Mensal[0].SldAdquirido = &zero
			},
			wantErrSubs: "sldCedido(100000000",
		},
		{
			nome: "campos nil: skip",
			mutate: func(d *Doc3050) {
				d.Mensal[0].SldCedido = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := A02SldCedidoMenosAdquirido{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A03 — SldBaiPrejuizo ≤ SldCarAtiva (regra 3020) ==========

func TestA03_SldBaiPrejuizoLeSldCar(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy path: baixado < sldCarAtiva",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: baixado > sldCarAtiva",
			mutate: func(d *Doc3050) {
				v := *d.Mensal[0].SldCarAtiva + 1.0
				d.Mensal[0].SldBaiPrejuizo = &v
			},
			wantErrSubs: "sldBaiPrejuizo(5000001",
		},
		{
			nome: "campos nil: skip",
			mutate: func(d *Doc3050) {
				d.Mensal[0].SldBaiPrejuizo = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := A03SldBaiPrejuizoLeSldCar{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A04 — SldCarAtiva + SldCedido ≥ SldAdquirido + VlrConcessoes (regra 3021) ==========

func TestA04_SldCarMaisCedidoVsAdquirido(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy path: carteira + cedido > adquirido + concessoes",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: esquerda < direita",
			mutate: func(d *Doc3050) {
				v := 1.0 // era 5M, agora 1
				d.Diario[0].SldCarAtiva = &v
			},
			wantErrSubs: "sldCarAtiva(1",
		},
		{
			nome: "campos nil: skip",
			mutate: func(d *Doc3050) {
				d.Diario[0].SldCedido = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := A04SldCarMaisCedidoVsAdquirido{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A05 — CNPJ raiz válida ==========

func TestA05_CNPJRaiz(t *testing.T) {
	tests := []struct {
		nome        string
		cnpj        string
		wantErrSubs string
	}{
		{"happy: 8 dígitos", "12345678", ""},
		{"vazio", "", "vazio"},
		{"muito curto", "1234567", "8 dígitos"},
		{"muito longo", "123456789", "8 dígitos"},
		{"letra no meio", "1234567A", "não-numérico"},
		{"underscore", "1234_678", "não-numérico"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.CNPJ = tt.cnpj
			err := A05CNPJRaiz{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A06 — dataBase formato YYYY-MM-DD ==========

func TestA06_DataBaseFormato(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: 2024-12-31", "2024-12-31", ""},
		{"happy: 2025-01-01", "2025-01-01", ""},
		{"vazio", "", "vazio"},
		{"formato errado 1: YYYY/MM/DD", "2024/12/31", "formato YYYY-MM-DD"},
		{"formato errado 2: YYYY-MM", "2024-12", "formato YYYY-MM-DD"},
		{"formato errado 3: DD-MM-YYYY", "31-12-2024", "formato YYYY-MM-DD"},
		{"letra na posição 0", "A024-12-31", "não-numérico"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.DataBase = tt.dataBase
			err := A06DataBaseFormato{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A07 — indRemessa válido ==========

func TestA07_IndRemessaValido(t *testing.T) {
	tests := []struct {
		nome        string
		ind         string
		wantErrSubs string
	}{
		{"I (inclusão)", "I", ""},
		{"A (alteração)", "A", ""},
		{"S (substituição)", "S", ""},
		{"vazio", "", "vazio"},
		{"X inválido", "X", "inválido"},
		{"minúscula i", "i", "inválido"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.IndRemessa = tt.ind
			err := A07IndRemessaValido{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A08 — nmContato + telContato não-vazios ==========

func TestA08_NmContatoObrigatorio(t *testing.T) {
	tests := []struct {
		nome        string
		nm, tel     string
		wantErrSubs string
	}{
		{"happy", "João", "11999998888", ""},
		{"nm vazio", "", "11999998888", "nmContato vazio"},
		{"tel vazio", "João", "", "telContato vazio"},
		{"ambos whitespace", "   ", "\t", "nmContato vazio"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.NmContato = tt.nm
			doc.Root.TelContato = tt.tel
			err := A08NmContatoObrigatorio{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A09-A11 — Limites de taxas (regra 3026-3028/3042-3044) ==========

func TestA09_TxMedJurosLimite(t *testing.T) {
	tests := []struct {
		nome        string
		tx          float64
		wantErrSubs string
	}{
		{"happy: 15.5%", 15.5, ""},
		{"happy: 0%", 0.0, ""},
		{"happy: 100% (boundary)", 100.0, ""},
		{"negativo: -0.1", -0.1, "txMedJuros=-0.1000 < 0"},
		{"muito alto: 150%", 150.0, "> 100%"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Diario[0].TxMedJuros = &tt.tx
			err := A09TxMedJurosLimite{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestA10_TxMedEncFiscaisLimite(t *testing.T) {
	doc := doc3050ValidoBase()
	v := -1.0
	doc.Diario[0].TxMedEncFiscais = &v
	err := A10TxMedEncFiscaisLimite{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "txMedEncFiscais=-1")
}

func TestA11_TxMedEncOperacionaisLimite(t *testing.T) {
	doc := doc3050ValidoBase()
	v := -1.0
	doc.Diario[0].TxMedEncOperacionais = &v
	err := A11TxMedEncOperacionaisLimite{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "txMedEncOperacionais=-1")
}

// ========== A12 — TxMinima ≤ TxMaxima ==========

func TestA12_TxMinimaLeMaxima(t *testing.T) {
	tests := []struct {
		nome        string
		min, max    float64
		wantErrSubs string
	}{
		{"happy: 10 < 25", 10.0, 25.0, ""},
		{"happy: igual (boundary)", 20.0, 20.0, ""},
		{"violação: 30 > 25", 30.0, 25.0, "txMinima(30.0000) > txMaxima(25.0000)"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Diario[0].TxMinima = &tt.min
			doc.Diario[0].TxMaxima = &tt.max
			err := A12TxMinimaLeMaxima{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A13 — PrzDecMedConcessoes ≥ 0 ==========

func TestA13_PrzDecMedConcessoesNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		prz         int
		wantErrSubs string
	}{
		{"happy: 180", 180, ""},
		{"happy: 0", 0, ""},
		{"negativo: -1", -1, "przDecMedConcessoes=-1 < 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Diario[0].PrzDecMedConcessoes = &tt.prz
			err := A13PrzDecMedConcessoesNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== A14 — PrzMedCarteira ≥ 0 ==========

func TestA14_PrzMedCarteiraNaoNeg(t *testing.T) {
	doc := doc3050ValidoBase()
	v := -5
	doc.Mensal[0].PrzMedCarteira = &v
	err := A14PrzMedCarteiraNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "przMedCarteira=-5 < 0")
}

// ========== Stubs (S01-S14) — devem retornar nil sempre ==========

func TestS01_S14_StubsReturnNil(t *testing.T) {
	doc := doc3050ValidoBase()
	stubs := []struct {
		code string
		rule Rule3050
	}{
		{"3050-S01", S01MatrizEncargoModalidade{}},
		{"3050-S02", S02DocNaoEsperado{}},
		{"3050-S03", S03ArquivoDispensado{}},
		{"3050-S04", S04HeaderDetalhe{}},
		{"3050-S05", S05ArquivoJaProcessado{}},
		{"3050-S06", S06SubstituicaoSemOriginal{}},
		{"3050-S07", S07Compactacao{}},
		{"3050-S08", S08DataBaseFutura{}},
		{"3050-S09", S09DiasUteis{}},
		{"3050-S10", S10DocAnterior{}},
		{"3050-S11", S11VlrConcessoesVsTaxas{}},
		{"3050-S12", S12PrzMedSeSld{}},
		{"3050-S13", S13UltimoDiaUtil{}},
		{"3050-S14", S14Cruzadas{}},
	}

	for _, s := range stubs {
		t.Run(s.code, func(t *testing.T) {
			if s.rule.Severity() != "I" {
				t.Errorf("stub %s severity=%q, esperado I", s.code, s.rule.Severity())
			}
			if err := s.rule.Apply3050(context.Background(), doc); err != nil {
				t.Errorf("stub %s deveria retornar nil, got %v", s.code, err)
			}
		})
	}
}

// ========== Builtin3050 — registry integration ==========

func TestBuiltin3050_TotalRulesIs(t *testing.T) {
	r := Builtin3050()
	if got := len(r.All3050()); got != 56 {
		t.Fatalf("Builtin3050 deveria ter 56 regras (Fase 1 28 + Fase 2 28), got %d", got)
	}

	codes := r.Codes3050()
	// 14 Agregadas A01-A14
	for i := 1; i <= 14; i++ {
		code := "3050-A" + itoaPad2(i)
		if r.Get3050(code) == nil {
			t.Errorf("regra %s não encontrada no registry", code)
		}
	}
	// 14 Stubs S01-S14
	for i := 1; i <= 14; i++ {
		code := "3050-S" + itoaPad2(i)
		if r.Get3050(code) == nil {
			t.Errorf("regra %s não encontrada no registry", code)
		}
	}

	_ = codes // para silenciar unused warning
}

// ========== ParseDoc3050 — smoke test do parser ==========

func TestParseDoc3050_Smoke(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31" indRemessa="I" nmContato="João" telContato="11999998888">
  <referencia>
    <diario>
      <crdLivre>
        <pesJuridica>
          <pre>
            <capGirPrzAte365 txMedJuros="15.5" txMedEncFiscais="0.5" txMedEncOperacionais="1.0" txMinima="10.0" txMaxima="25.0" vlrConcessoes="1000000.00" przDecMedConcessoes="180" qtdNovContratos="50" sldCarAtiva="5000000.00" sldCedido="100000.00" sldAdquirido="50000.00"/>
          </pre>
        </pesJuridica>
      </crdLivre>
    </diario>
    <mensal>
      <crdLivre>
        <pesJuridica>
          <pre>
            <capGirPrzAte365 sldCarAtiva="5000000.00" sldCedido="100000.00" sldAdquirido="50000.00" sldBaiPrejuizo="200000.00" sldCarAte14="3000000.00" sldCarAte60="1500000.00" sldCarAte90="400000.00" sldCarMaior90="100000.00" qtdConAte14="30" qtdConAte60="15" qtdConAte90="4" qtdConMaior90="1" przMedCarteira="120"/>
          </pre>
        </pesJuridica>
      </crdLivre>
    </mensal>
  </referencia>
</DocTXB>`

	doc, err := ParseDoc3050([]byte(xmlInput))
	if err != nil {
		t.Fatalf("parse falhou: %v", err)
	}
	if doc.Root.CNPJ != "12345678" {
		t.Errorf("CNPJ=%q, esperado 12345678", doc.Root.CNPJ)
	}
	if doc.Root.DataBase != "2024-12-31" {
		t.Errorf("DataBase=%q, esperado 2024-12-31", doc.Root.DataBase)
	}
	if len(doc.Diario) != 1 {
		t.Errorf("len(Diario)=%d, esperado 1", len(doc.Diario))
	}
	if len(doc.Mensal) != 1 {
		t.Errorf("len(Mensal)=%d, esperado 1", len(doc.Mensal))
	}
	if doc.Diario[0].Codigo != "capGirPrzAte365" {
		t.Errorf("Codigo=%q, esperado capGirPrzAte365", doc.Diario[0].Codigo)
	}
	if doc.Diario[0].Encargo != "pre" {
		t.Errorf("Encargo=%q, esperado pre", doc.Diario[0].Encargo)
	}
	if doc.Diario[0].TipoCli != "pesJuridica" {
		t.Errorf("TipoCli=%q, esperado pesJuridica", doc.Diario[0].TipoCli)
	}
	if doc.Diario[0].TxMedJuros == nil || *doc.Diario[0].TxMedJuros != 15.5 {
		t.Errorf("TxMedJuros=%v, esperado 15.5", doc.Diario[0].TxMedJuros)
	}
	if doc.Mensal[0].SldCarAte14 == nil || *doc.Mensal[0].SldCarAte14 != 3000000.00 {
		t.Errorf("SldCarAte14=%v, esperado 3000000.00", doc.Mensal[0].SldCarAte14)
	}
}

// ========== Helpers ==========

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func itoaPad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func checkErr(t *testing.T, err error, wantSubs string) {
	t.Helper()
	if wantSubs == "" {
		if err != nil {
			t.Fatalf("esperava nil, got %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("esperava erro contendo %q, got nil", wantSubs)
	}
	if !contains(err.Error(), wantSubs) {
		t.Fatalf("esperava erro contendo %q, got %q", wantSubs, err.Error())
	}
}
