package docdli

import (
	"context"
	"testing"
)

func TestParse_Valid(t *testing.T) {
	xml := `<documentoDLI cnpj="12345678" dataBase="2026-06" codigoDocumento="2062" tipoEnvio="I">
		<Limites>
			<Limite codigo="06.00" valor="1000000.00"/>
			<Limite codigo="20.00" valor="500000.00"/>
		</Limites>
		<Indicadores>
			<Indicador codigo="IND001" valor="S"/>
		</Indicadores>
		<Parametros>
			<Parametro codigo="P001" valor="abc"/>
		</Parametros>
		<Contas>
			<Conta codigo="1.10.00" valor="200000.00"/>
		</Contas>
	</documentoDLI>`

	doc, err := ParseFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.CNPJ != "12345678" {
		t.Errorf("CNPJ: got %q, want 12345678", doc.CNPJ)
	}
	if doc.DataBase != "2026-06" {
		t.Errorf("DataBase: got %q, want 2026-06", doc.DataBase)
	}
	if doc.TipoEnvio != "I" {
		t.Errorf("TipoEnvio: got %q, want I", doc.TipoEnvio)
	}
	if len(doc.Limites) != 2 {
		t.Errorf("Limites count: got %d, want 2", len(doc.Limites))
	}
	if len(doc.Indicadores) != 1 {
		t.Errorf("Indicadores count: got %d, want 1", len(doc.Indicadores))
	}
	if len(doc.Contas) != 1 {
		t.Errorf("Contas count: got %d, want 1", len(doc.Contas))
	}
}

func TestParse_EmptyDocument(t *testing.T) {
	_, err := ParseFromBytes([]byte{})
	if err != errEmptyDocument {
		t.Errorf("got %v, want errEmptyDocument", err)
	}
}

func TestParse_InvalidXML(t *testing.T) {
	_, err := ParseFromBytes([]byte("not xml"))
	if err == nil {
		t.Error("expected error on invalid XML")
	}
}

func TestParse_WrongCodigo(t *testing.T) {
	xml := `<documentoDLI cnpj="12345678" dataBase="2026-06" codigoDocumento="9999" tipoEnvio="I"/>`
	_, err := ParseFromBytes([]byte(xml))
	if err == nil {
		t.Error("expected error on wrong codigoDocumento")
	}
}

func TestValidate_AllErrors(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "123",     // invalid (DLI-01)
		DataBase:  "2026-13", // invalid month (DLI-02)
		TipoEnvio: "X",       // invalid (DLI-03)
		CodigoDoc: "9999",    // invalid (DLI-04)
		// no content → DLI-05
	}

	errs := Validate(doc)
	if len(errs) != 5 {
		t.Errorf("expected 5 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_LimiteErrors(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "12345678",
		DataBase:  "2026-06",
		TipoEnvio: "I",
		CodigoDoc: "2062",
		Limites: []Limite{
			{Codigo: "BAD", Valor: "100"},                 // invalid code
			{Codigo: "06.00", Valor: "not-a-number"},      // invalid value
			{Codigo: "20.00", Valor: "12345678901234.56"}, // > N13,2
		},
	}

	errs := Validate(doc)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_IndicadorErrors(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "12345678",
		DataBase:  "2026-06",
		TipoEnvio: "I",
		CodigoDoc: "2062",
		Indicadores: []Indicador{
			{Codigo: "IND001", Valor: "X"}, // must be S or N
		},
	}

	errs := Validate(doc)
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestValidate_ContaErrors(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "12345678",
		DataBase:  "2026-06",
		TipoEnvio: "I",
		CodigoDoc: "2062",
		Contas: []Conta{
			{Codigo: "BAD", Valor: "100"},             // invalid code
			{Codigo: "1.10.00", Valor: "not-numeric"}, // invalid value
		},
	}

	errs := Validate(doc)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_EmptyDocument(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "12345678",
		DataBase:  "2026-06",
		TipoEnvio: "I",
		CodigoDoc: "2062",
	}

	errs := Validate(doc)
	if len(errs) != 1 {
		t.Errorf("expected 1 error (no content), got %d: %v", len(errs), errs)
	}
}

func TestValidate_Valid(t *testing.T) {
	doc := &DocumentoDLI{
		CNPJ:      "12345678",
		DataBase:  "2026-06",
		TipoEnvio: "S",
		CodigoDoc: "2062",
		Limites:   []Limite{{Codigo: "06.00", Valor: "1000000.00"}},
	}

	errs := Validate(doc)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateDocument(t *testing.T) {
	xml := `<documentoDLI cnpj="12345678" dataBase="2026-06" codigoDocumento="2062" tipoEnvio="I">
		<Limites><Limite codigo="06.00" valor="1000000.00"/></Limites>
	</documentoDLI>`

	res, err := ValidateDocument(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Valid {
		t.Errorf("expected valid, got criticas: %v", res.Criticas)
	}
}

func TestValidateDocument_Invalid(t *testing.T) {
	xml := `<documentoDLI cnpj="BAD" dataBase="2026-06" codigoDocumento="2062" tipoEnvio="I">
		<Limites><Limite codigo="BAD" valor="x"/></Limites>
	</documentoDLI>`

	res, err := ValidateDocument(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Valid {
		t.Error("expected invalid")
	}
	if len(res.Criticas) == 0 {
		t.Error("expected criticas")
	}
}

func TestExtractLimite(t *testing.T) {
	doc := &DocumentoDLI{
		Limites: []Limite{
			{Codigo: "06.00", Valor: "100"},
			{Codigo: "20.00", Valor: "200"},
		},
	}

	l := ExtractLimite(doc, "20.00")
	if l == nil {
		t.Fatal("expected to find 20.00")
	}
	if l.Valor != "200" {
		t.Errorf("valor: got %s", l.Valor)
	}

	l = ExtractLimite(doc, "99.00")
	if l != nil {
		t.Error("should not find 99.00")
	}
}

func TestExtractConta(t *testing.T) {
	doc := &DocumentoDLI{
		Contas: []Conta{
			{Codigo: "1.10.00", Valor: "1000"},
			{Codigo: "2.20.00", Valor: "2000"},
		},
	}

	c := ExtractConta(doc, "2.20.00")
	if c == nil {
		t.Fatal("expected to find 2.20.00")
	}
	if c.Valor != "2000" {
		t.Errorf("valor: got %s", c.Valor)
	}
}

func TestHasIndicador(t *testing.T) {
	doc := &DocumentoDLI{
		Indicadores: []Indicador{
			{Codigo: "IND001", Valor: "S"},
			{Codigo: "IND002", Valor: "N"},
		},
	}

	if !HasIndicador(doc, "IND001") {
		t.Error("expected IND001=S to be true")
	}
	if HasIndicador(doc, "IND002") {
		t.Error("expected IND002=N to be false")
	}
	if HasIndicador(doc, "IND999") {
		t.Error("expected IND999 to be false")
	}
}

func TestLimiteMaximo(t *testing.T) {
	doc := &DocumentoDLI{}
	pla := 10_000_000.0

	tests := []struct {
		code   string
		pla    float64
		expect float64
	}{
		{"20.00", pla, pla * 0.10},
		{"21.00", pla, pla * 0.01},
		{"22.00", pla, pla * 0.05},
		{"34.00", pla, pla * 5.0},
		{"99.00", pla, 0},
	}

	for _, tt := range tests {
		got := LimiteMaximo(doc, tt.code, tt.pla)
		if got != tt.expect {
			t.Errorf("LimiteMaximo(%s): got %.2f, want %.2f", tt.code, got, tt.expect)
		}
	}
}

func TestBalance(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"100", 100},
		{"1234567890123.45", 1234567890123.45},
		{"-500.00", -500},
	}

	for _, tt := range tests {
		got := Balance(tt.input)
		if got != tt.want {
			t.Errorf("Balance(%q): got %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestValidInstitutionTypes(t *testing.T) {
	if !ValidInstitutionTypes["BANCO_COMERCIAL"] {
		t.Error("BANCO_COMERCIAL should be valid")
	}
	if !ValidInstitutionTypes["SCD"] {
		t.Error("SCD should be valid")
	}
	if ValidInstitutionTypes["INVALID"] {
		t.Error("INVALID should not be valid")
	}
}
