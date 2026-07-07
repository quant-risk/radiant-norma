// Tests Fase 4 — Sprint 33 Fase 4 (v3.34.6).
//
// Cobre:
// - H16-H20 (5 regras Header)
// - S33, S34, S36, S38 (4 regras Sistema)
// - I29-I36 (8 regras Individuais — sub-modalidades específicas)
// - Edge case fix IsUltimoDiaUtilMes (sábado último dia)
//
// Total: 18 funções de teste + 1 edge case test.
package rules

import (
	"context"
	"testing"
	"time"
)

// ========== H16-H20 — Header avançado ==========

func TestH16_EncodingUTF8(t *testing.T) {
	tests := []struct {
		nome        string
		encoding    string
		wantErrSubs string
	}{
		{"happy: UTF-8", "UTF-8", ""},
		{"happy: utf-8 (case-insensitive)", "utf-8", ""},
		{"happy: vazio (sem declaração)", "", ""},
		{"violação: ISO-8859-1", "ISO-8859-1", "esperado UTF-8"},
		{"violação: ASCII", "ASCII", "esperado UTF-8"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{Encoding: tt.encoding}}
			err := H16EncodingUTF8{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH17_SemBOMUTF8(t *testing.T) {
	tests := []struct {
		nome        string
		bomPresent  bool
		wantErrSubs string
	}{
		{"happy: sem BOM", false, ""},
		{"violação: com BOM", true, "BOM UTF-8"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{BomPresent: tt.bomPresent}}
			err := H17SemBOMUTF8{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestH18_RaizDocTXB(t *testing.T) {
	doc := &Doc3050{} // vazio
	err := H18RaizDocTXB{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "vazio")

	doc = &Doc3050{Root: Doc3050Root{CNPJ: "12345678", DataBase: "2024-12-31"}}
	err = H18RaizDocTXB{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestH19_ApenasUmaReferencia(t *testing.T) {
	// Carry-over (parser change necessário para contar elementos).
	doc := &Doc3050{}
	err := H19ApenasUmaReferencia{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "") // stub atual: sempre passa
}

func TestH20_ApenasUmDiarioUmMensal(t *testing.T) {
	// Carry-over (parser change necessário).
	doc := &Doc3050{}
	err := H20ApenasUmDiarioUmMensal{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

// ========== S33, S34, S36, S38 — Sistema ==========

func TestS33_DataBaseMax1YearOld(t *testing.T) {
	hoje := time.Now().UTC()
	ontem := hoje.AddDate(0, 0, -1).Format("2006-01-02")
	seisMesesAtras := hoje.AddDate(0, -6, 0).Format("2006-01-02")
	doisAnosAtras := hoje.AddDate(-2, 0, 0).Format("2006-01-02")

	tests := []struct {
		nome        string
		dataBase    string
		wantErrSubs string
	}{
		{"happy: ontem", ontem, ""},
		{"happy: 6 meses atrás", seisMesesAtras, ""},
		{"violação: 2 anos atrás", doisAnosAtras, "> 1 ano atrás"},
		{"skip: formato inválido", "abc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Root: Doc3050Root{DataBase: tt.dataBase}}
			err := S33DataBaseMax1YearOld{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestS34_DataBaseConsistente(t *testing.T) {
	doc := &Doc3050{Root: Doc3050Root{DataBase: ""}}
	err := S34DataBaseConsistente{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "ausente")

	doc = &Doc3050{Root: Doc3050Root{DataBase: "2024-12-31"}}
	err = S34DataBaseConsistente{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestS36_IndRemessaIApenasPrimeiraVez_StubReturnsNil(t *testing.T) {
	// Stub honesto: precisa histórico de envios.
	doc := &Doc3050{Root: Doc3050Root{IndRemessa: "I"}}
	err := S36IndRemessaIApenasPrimeiraVez{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

func TestS38_DocUnicoPorCNPJDataBase_StubReturnsNil(t *testing.T) {
	// Stub: validação real requer histórico.
	doc := &Doc3050{Root: Doc3050Root{CNPJ: "12345678", DataBase: "2024-12-31"}}
	err := S38DocUnicoPorCNPJDataBase{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "")
}

// ========== I29-I36 — Sub-modalidades específicas ==========

func TestI29_AquVeiculosVlrConcNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         *float64
		wantErrSubs string
	}{
		{"happy: 100000.0", ptrF(100000.0), ""},
		{"violação: -50.0", ptrF(-50.0), "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "aquVeiculos", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: tt.val}}}
			err := I29AquVeiculosVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

func TestI30_ArrMerVeiculosVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "arrMerVeiculos", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-10.0)}}}
	err := I30ArrMerVeiculosVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI31_ArrMerOutrosVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "arrMerOutros", Encargo: "pre", TipoCli: "pesJuridica", VlrConcessoes: ptrF(-5.0)}}}
	err := I31ArrMerOutrosVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI32_CapGirTetoRotSldCarNaoNeg(t *testing.T) {
	doc := &Doc3050{Mensal: []Modalidade{{Codigo: "capGirTetoRot", Encargo: "pre", TipoCli: "pesJuridica", SldCarAtiva: ptrF(-1000.0)}}}
	err := I32CapGirTetoRotSldCarNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI33_ChqEspSldCarNaoNeg(t *testing.T) {
	doc := &Doc3050{Mensal: []Modalidade{{Codigo: "chqEsp", Encargo: "pre", TipoCli: "pesFisica", SldCarAtiva: ptrF(-500.0)}}}
	err := I33ChqEspSldCarNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI34_CtgGtaSldCarNaoNeg(t *testing.T) {
	doc := &Doc3050{Mensal: []Modalidade{{Codigo: "ctgGta", Encargo: "pre", TipoCli: "pesJuridica", SldCarAtiva: ptrF(-200.0)}}}
	err := I34CtgGtaSldCarNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI35_FinancBensVlrConcNaoNeg(t *testing.T) {
	doc := &Doc3050{Diario: []Modalidade{{Codigo: "financBens", Encargo: "pre", TipoCli: "pesFisica", VlrConcessoes: ptrF(-1000.0)}}}
	err := I35FinancBensVlrConcNaoNeg{}.Apply3050(context.Background(), doc)
	checkErr(t, err, "< 0")
}

func TestI36_CcbPrzDecNaoNeg(t *testing.T) {
	tests := []struct {
		nome        string
		val         int
		wantErrSubs string
	}{
		{"happy: 180 dias", 180, ""},
		{"violação: -1 dia", -1, "< 0"},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc := &Doc3050{Diario: []Modalidade{{Codigo: "ccb", Encargo: "pre", TipoCli: "pesFisica", PrzDecMedConcessoes: &tt.val}}}
			err := I36CcbPrzDecNaoNeg{}.Apply3050(context.Background(), doc)
			checkErr(t, err, tt.wantErrSubs)
		})
	}
}

// ========== Parser — H16/H17 (DT-31) ==========

func TestParseDoc3050_DetectaEncoding(t *testing.T) {
	tests := []struct {
		nome         string
		xmlInput     string
		wantEncoding string
		wantBom      bool
	}{
		{"happy: encoding minúsculo", `<?xml version="1.0" encoding="utf-8"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia></referencia></DocTXB>`, "UTF-8", false},
		{"happy: sem declaração encoding", `<?xml version="1.0"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia></referencia></DocTXB>`, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.nome, func(t *testing.T) {
			doc, _ := ParseDoc3050([]byte(tt.xmlInput))
			if doc == nil {
				t.Fatalf("parse retornou nil")
			}
			if doc.Root.Encoding != tt.wantEncoding {
				t.Errorf("Encoding=%q, want %q", doc.Root.Encoding, tt.wantEncoding)
			}
			if doc.Root.BomPresent != tt.wantBom {
				t.Errorf("BomPresent=%v, want %v", doc.Root.BomPresent, tt.wantBom)
			}
		})
	}
}

func TestParseDoc3050_DetectaBOM(t *testing.T) {
	xmlValido := `<?xml version="1.0" encoding="UTF-8"?><DocTXB cnpjInstituicao="12345678" dataBase="2024-12-31"><referencia></referencia></DocTXB>`
	xmlComBom := append([]byte{0xEF, 0xBB, 0xBF}, []byte(xmlValido)...)
	doc, _ := ParseDoc3050(xmlComBom)
	if doc == nil {
		t.Fatalf("parse retornou nil")
	}
	if !doc.Root.BomPresent {
		t.Errorf("BOM presente nos primeiros 3 bytes deveria ser detectado")
	}
	if doc.Root.Encoding != "UTF-8" {
		t.Errorf("Encoding=%q, want UTF-8", doc.Root.Encoding)
	}
}

// ========== Integração ==========

func TestBuiltin3050_Fase4TotalRulesIs97(t *testing.T) {
	// REMOVIDO na Fase 5: contagem mudou de 97 (Fase 4) para 153 (Fase 5).
	// Teste obsoleto — deixado apenas como sentinel. Verificação real em
	// TestBuiltin3050_TotalRulesIs (3050_test.go).
	t.Skip("contagem mudou pós-Fase 5 (153 regras); ver TestBuiltin3050_TotalRulesIs em 3050_test.go")
}
