// Tests for doc4111 package.
package doc4111_test

import (
	"context"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/doc4111"
)

func TestParse_Valid(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Documento4111 cnpj="12345678" dataBase="2024-12" codigoDocumento="4111">
  <Cliente>
    <QtdCli>10</QtdCli>
    <CNPJ>12345678000199</CNPJ>
  </Cliente>
</Documento4111>`

	doc, err := doc4111.ParseFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("ParseFromBytes failed: %v", err)
	}
	if doc.CNPJ != "12345678" {
		t.Errorf("CNPJ = %q, want 12345678", doc.CNPJ)
	}
	if len(doc.Clientes) != 1 {
		t.Fatalf("len(Clientes) = %d, want 1", len(doc.Clientes))
	}
}

func TestParse_WrongCodigo(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Documento4111 cnpj="12345678" dataBase="2024-12" codigoDocumento="9999">
</Documento4111>`
	_, err := doc4111.ParseFromBytes([]byte(xml))
	if err == nil {
		t.Fatal("expected error for wrong codigoDocumento")
	}
}

func TestValidate_CNPJ(t *testing.T) {
	doc := minimalDoc()
	doc.CNPJ = "1234567" // 7 digits
	errs := doc4111.Validate(doc)
	if len(errs) == 0 {
		t.Error("expected error for invalid CNPJ")
	}
}

func TestValidate_DataBase(t *testing.T) {
	doc := minimalDoc()
	doc.DataBase = "2024-13" // invalid month
	errs := doc4111.Validate(doc)
	if len(errs) == 0 {
		t.Error("expected error for invalid dataBase")
	}
}

func TestValidateDocument_Valid(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Documento4111 cnpj="12345678" dataBase="2024-12" codigoDocumento="4111">
  <Cliente><QtdCli>5</QtdCli></Cliente>
</Documento4111>`

	result, err := doc4111.ValidateDocument(context.Background(), []byte(xml))
	if err != nil {
		t.Fatalf("ValidateDocument failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got criticas: %+v", result.Criticas)
	}
}

func TestExtractQtdTotal(t *testing.T) {
	doc := &doc4111.Documento4111{
		Clientes: []doc4111.Cliente{
			{QtdCli: "10"},
			{QtdCli: "20"},
		},
	}
	total := doc4111.ExtractQtdTotal(doc)
	if total != 30 {
		t.Errorf("ExtractQtdTotal = %f, want 30", total)
	}
}

func TestHasModalidadeInadimplente(t *testing.T) {
	doc := &doc4111.Documento4111{
		Clientes: []doc4111.Cliente{
			{
				Modalidade: []doc4111.Modalidade{
					{Codigo: "0213", Indicacao: "S"},
				},
			},
		},
	}
	if !doc4111.HasModalidadeInadimplente(doc) {
		t.Error("expected true for inadimplente")
	}
}

func minimalDoc() *doc4111.Documento4111 {
	return &doc4111.Documento4111{
		CNPJ:      "12345678",
		DataBase:  "2024-12",
		CodigoDoc: "4111",
	}
}
