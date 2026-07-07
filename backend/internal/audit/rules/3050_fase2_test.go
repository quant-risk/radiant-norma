// Tests para regras 3050 Fase 2 — Sprint 33.
//
// Adiciona testes para S15-S28 (Sistemáticas) + I01-I14 (Individuais/Cruzadas).
// Continua padrão table-driven da Fase 1 (3050_test.go).
package rules

import (
	"context"
	"testing"
)

// ========== S15 — Data-base válida (2009 ≤ ano ≤ 2030) ==========

func TestS15_DataBaseValida(t *testing.T) {
	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: 2024-12-31", "2024-12-31", ""},
		{"happy: 2009-09-01 (limite mínimo)", "2009-09-01", ""},
		{"happy: 2030-06-15 (limite máximo tolerância)", "2030-06-15", ""},
		{"muito antiga: 2008-12-31", "2008-12-31", "anterior a 2009-09"},
		{"muito futura: 2050-01-01", "2050-01-01", "muito futura"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.DataBase = tt.dataBase
			err := S15DataBaseValida{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S16 — nmContato length 1-100 ==========

func TestS16_NmContatoLength(t *testing.T) {
	tests := []struct {
		nome        string
		nm          string
		wantErrSubs string
	}{
		{"happy: 10 chars", "João Silva", ""},
		{"happy: 100 chars (limite)", string(make([]byte, 100)), ""},
		{"muito longo: 101 chars", "a" + string(make([]byte, 100)), "length=101"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.NmContato = tt.nm
			err := S16NmContatoLength{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S17 — telContato formato 10-11 dígitos ==========

func TestS17_TelContatoFormato(t *testing.T) {
	tests := []struct {
		nome        string
		tel         string
		wantErrSubs string
	}{
		{"happy: 11 dígitos (celular)", "11999998888", ""},
		{"happy: 10 dígitos (fixo)", "1133334444", ""},
		{"happy: formatado (11)99999-8888", "(11)99999-8888", ""},
		{"poucos dígitos", "1234", "4 dígitos"},
		{"muitos dígitos", "123456789012", "12 dígitos"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Root.TelContato = tt.tel
			err := S17TelContatoFormato{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S18 — vlrConcessoes=0 → txMedJuros=0 ==========

func TestS18_VlrConcessoesZeroTxJurosZero(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy: ambos zero",
			mutate: func(d *Doc3050) { zero := 0.0; d.Diario[0].TxMedJuros = &zero },
		},
		{
			nome:   "happy: ambos preenchidos",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: vlrConc=0 mas txMedJuros≠0",
			mutate: func(d *Doc3050) {
				zero := 0.0
				d.Diario[0].VlrConcessoes = &zero
			},
			wantErrSubs: "vlrConcessoes=0 mas txMedJuros",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := S18VlrConcessoesZeroTxJurosZero{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S19-S22 — txMedJuros/EncOper/przDec coerência ==========

func TestS19_TxJurosZeroVlrConcessoesPos(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy: txMedJuros > 0",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: txMedJuros=0 e vlrConc=0",
			mutate: func(d *Doc3050) {
				zero := 0.0
				d.Diario[0].TxMedJuros = &zero
				d.Diario[0].VlrConcessoes = &zero
			},
			wantErrSubs: "txMedJuros=0 mas vlrConcessoes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := S19TxJurosZeroVlrConcessoesPos{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestS20_TxEncOperZeroVlrConcessoesPos(t *testing.T) {
	doc := doc3050ValidoBase()
	zero := 0.0
	doc.Diario[0].TxMedEncOperacionais = &zero
	doc.Diario[0].VlrConcessoes = &zero
	err := S20TxEncOperZeroVlrConcessoesPos{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "txMedEncOperacionais=0 mas vlrConcessoes")
}

func TestS21_PrzDecZeroVlrConcessoesPos(t *testing.T) {
	doc := doc3050ValidoBase()
	zeroF := 0.0
	zeroI := 0
	doc.Diario[0].VlrConcessoes = &zeroF
	doc.Diario[0].PrzDecMedConcessoes = &zeroI
	err := S21PrzDecZeroVlrConcessoesPos{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "przDecMedConcessoes=0 mas vlrConcessoes")
}

func TestS22_PrzDecPosVlrConcessoesPos(t *testing.T) {
	doc := doc3050ValidoBase()
	zero := 0.0
	doc.Diario[0].VlrConcessoes = &zero
	// przDecMedConcessoes = 100 (default), então > 0 e vlrConc=0 viola
	doc.Diario[0].VlrConcessoes = &zero
	err := S22PrzDecPosVlrConcessoesPos{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "przDecMedConcessoes=180 > 0 mas vlrConcessoes")
}

// ========== S23 — PrzMedCarteira condicional ==========

func TestS23_PrzMedCondicional(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy: sldCar > 0 e przMed presente",
			mutate: func(d *Doc3050) {},
		},
		{
			nome:   "happy: sldCar = 0 sem przMed",
			mutate: func(d *Doc3050) { d.Mensal[0].SldCarAtiva = ptrF(0.0); d.Mensal[0].PrzMedCarteira = nil },
		},
		{
			nome: "violação: sldCar != 0 mas przMed ausente",
			mutate: func(d *Doc3050) {
				d.Mensal[0].PrzMedCarteira = nil
			},
			wantErrSubs: "przMedCarteira ausente",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := S23PrzMedCondicional{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S25 — CNPJ não-zero ==========

func TestS25_CNPJNaoZero(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Root.CNPJ = "00000000"
	err := S25CNPJNaoZero{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "00000000")
}

// ========== S26 — Codigo+Encargo+TipoCli únicos ==========

func TestS26_CodigoEncargoTipoCliUnico(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{
			nome:   "happy: 1 modalidade única",
			mutate: func(d *Doc3050) {},
		},
		{
			nome: "violação: 2 modalidades idênticas em Diario",
			mutate: func(d *Doc3050) {
				m := d.Diario[0]
				d.Diario = append(d.Diario, m)
			},
			wantErrSubs: "duplicada",
		},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			tt.mutate(doc)
			err := S26CodigoEncargoTipoCliUnico{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== S27 — SldBaiPrejuizo >= 0 ==========

func TestS27_SldBaiPrejuizoNaoNeg(t *testing.T) {
	doc := doc3050ValidoBase()
	neg := -1.0
	doc.Mensal[0].SldBaiPrejuizo = &neg
	err := S27SldBaiPrejuizoNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "sldBaiPrejuizo=-1.00 < 0")
}

// ========== S28 — qtdNovContratos >= 0 ==========

func TestS28_QtdNovContratosNaoNeg(t *testing.T) {
	doc := doc3050ValidoBase()
	neg := -5
	doc.Diario[0].QtdNovContratos = &neg
	err := S28QtdNovContratosNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "qtdNovContratos=-5 < 0")
}

// ========== I01-I02 — CapGirPrzAte365 / Sup365 ==========

func TestI01_CapGirAte365(t *testing.T) {
	tests := []struct {
		nome        string
		prz         int
		wantErrSubs string
	}{
		{"happy: 180 dias", 180, ""},
		{"happy: 365 (limite)", 365, ""},
		{"violação: 366", 366, "> 365"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Diario[0].Codigo = "capGirPrzAte365"
			doc.Diario[0].PrzDecMedConcessoes = &tt.prz
			err := I01CapGirAte365{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI02_CapGirSup365(t *testing.T) {
	tests := []struct {
		nome        string
		prz         int
		wantErrSubs string
	}{
		{"happy: 400 dias", 400, ""},
		{"violação: 365", 365, "≤ 365"},
		{"violação: 180", 180, "≤ 365"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ValidoBase()
			doc.Diario[0].Codigo = "capGirPrzSup365"
			doc.Diario[0].PrzDecMedConcessoes = &tt.prz
			err := I02CapGirSup365{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== I03-I06 — CredPesNaoConsignado soma sub-modalidades ==========

func doc3050ComCredPes() *Doc3050 {
	doc := doc3050ValidoBase()
	// Adicionar sub-modalidades em Diario com mesma encargo/tipoCli
	doc.Diario = append(doc.Diario,
		Modalidade{Codigo: "aquVeiculos", Encargo: "pre", TipoCli: "pesJuridica",
			VlrConcessoes: ptrF(100000.0), SldCarAtiva: ptrF(500000.0), SldCedido: ptrF(0.0), SldAdquirido: ptrF(0.0)},
		Modalidade{Codigo: "arrMerVeiculos", Encargo: "pre", TipoCli: "pesJuridica",
			VlrConcessoes: ptrF(50000.0), SldCarAtiva: ptrF(200000.0), SldCedido: ptrF(0.0), SldAdquirido: ptrF(0.0)},
	)
	// Mudar modalidade[0] para crdPesNaoConsignado com soma esperada
	doc.Diario[0].Codigo = "crdPesNaoConsignado"
	doc.Diario[0].VlrConcessoes = ptrF(150000.0) // 150k = 100k (aquVeiculos) + 50k (arrMerVeiculos)
	doc.Diario[0].SldCarAtiva = ptrF(700000.0)   // 700k = 500k + 200k
	// Mensal: similar
	doc.Mensal = append(doc.Mensal,
		Modalidade{Codigo: "aquVeiculos", Encargo: "pre", TipoCli: "pesJuridica",
			SldCarAtiva: ptrF(500000.0), SldCedido: ptrF(0.0), SldAdquirido: ptrF(0.0)},
		Modalidade{Codigo: "arrMerVeiculos", Encargo: "pre", TipoCli: "pesJuridica",
			SldCarAtiva: ptrF(200000.0), SldCedido: ptrF(0.0), SldAdquirido: ptrF(0.0)},
	)
	doc.Mensal[0].Codigo = "crdPesNaoConsignado"
	doc.Mensal[0].SldCarAtiva = ptrF(700000.0) // 700k = 500k + 200k
	doc.Mensal[0].SldAdquirido = ptrF(0.0)
	doc.Mensal[0].SldCedido = ptrF(0.0)
	return doc
}

func TestI03_CredPesNaoConsignadoSldCar(t *testing.T) {
	tests := []struct {
		nome        string
		mutate      func(*Doc3050)
		wantErrSubs string
	}{
		{"happy: soma confere", func(d *Doc3050) {}, ""},
		{"violação: sldCarAtiva != soma", func(d *Doc3050) {
			d.Mensal[0].SldCarAtiva = ptrF(800000.0)
		}, "sldCarAtiva=800000"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := doc3050ComCredPes()
			tt.mutate(doc)
			err := I03CredPesNaoConsignadoSldCar{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI04_CredPesNaoConsignadoVlrConcessoes(t *testing.T) {
	doc := doc3050ComCredPes()
	err := I04CredPesNaoConsignadoVlrConcessoes{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestI05_CredPesNaoConsignadoSldAdquirido(t *testing.T) {
	doc := doc3050ComCredPes()
	err := I05CredPesNaoConsignadoSldAdquirido{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestI06_CredPesNaoConsignadoSldCedido(t *testing.T) {
	doc := doc3050ComCredPes()
	err := I06CredPesNaoConsignadoSldCedido{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

// ========== I07-I10 — Limites BACEN (przMed, przDec) ==========

func TestI07_PrzMedCarteiraBaixo(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Mensal[0].PrzMedCarteira = ptrI(15)
	err := I07PrzMedCarteiraBaixo{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 30 dias")
}

func TestI08_PrzMedCarteiraAlto(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Mensal[0].PrzMedCarteira = ptrI(6000)
	err := I08PrzMedCarteiraAlto{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "> 5000 dias")
}

func TestI09_PrzDecMedConcessoesBaixo(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Diario[0].PrzDecMedConcessoes = ptrI(0)
	err := I09PrzDecMedConcessoesBaixo{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 1 dia")
}

func TestI10_PrzDecMedConcessoesAlto(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Diario[0].PrzDecMedConcessoes = ptrI(6000)
	err := I10PrzDecMedConcessoesAlto{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "> 5000 dias")
}

// ========== I11-I14 — Limites BACEN (saldos, concessões) ==========

func TestI11_SldCarAtivaMuitoBaixo(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Mensal[0].SldCarAtiva = ptrF(500.0)
	err := I11SldCarAtivaMuitoBaixo{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< R$ 1000")
}

func TestI12_SldCarAtivaMuitoAlto(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Mensal[0].SldCarAtiva = ptrF(2e12)
	err := I12SldCarAtivaMuitoAlto{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "> R$ 1 trilhão")
}

func TestI13_VlrConcessoesMuitoBaixo(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Diario[0].VlrConcessoes = ptrF(500.0)
	err := I13VlrConcessoesMuitoBaixo{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< R$ 1000")
}

func TestI14_VlrConcessoesMuitoAlto(t *testing.T) {
	doc := doc3050ValidoBase()
	doc.Diario[0].VlrConcessoes = ptrF(2e12)
	err := I14VlrConcessoesMuitoAlto{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "> R$ 1 trilhão")
}

// ========== S24 — Stub (txMedJurosAjustada carry-over) ==========

func TestS24_StubReturnsNil(t *testing.T) {
	doc := doc3050ValidoBase()
	rule := S24TxJurosAjustadaLeTxJuros{}
	if err := rule.Apply3050(context.Background(), doc); err != nil {
		t.Errorf("stub S24 deveria retornar nil, got %v", err)
	}
	if rule.Severity() != "I" {
		t.Errorf("S24 severity deveria ser I, got %s", rule.Severity())
	}
}

// ========== Builtin3050 — Fase 2 total 56 regras ==========

func TestBuiltin3050_Fase2TotalRulesIs(t *testing.T) {
	r := Builtin3050()
	if got := len(r.All3050()); got != 56 {
		t.Fatalf("Builtin3050 deveria ter 56 regras (14 A + 14 S Fase 1 + 14 S + 14 I Fase 2), got %d", got)
	}

	// Spot-check: S15-S28 + I01-I14
	for _, code := range []string{"3050-S15", "3050-S16", "3050-S17", "3050-S18", "3050-S19",
		"3050-S20", "3050-S21", "3050-S22", "3050-S23", "3050-S24", "3050-S25", "3050-S26",
		"3050-S27", "3050-S28"} {
		if r.Get3050(code) == nil {
			t.Errorf("Fase 2: regra %s não encontrada", code)
		}
	}
	for i := 1; i <= 14; i++ {
		code := "3050-I" + itoaPad2(i)
		if r.Get3050(code) == nil {
			t.Errorf("Fase 2: regra %s não encontrada", code)
		}
	}
}

// ========== Helpers ==========

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }
