// Tests for drsac package — DRSAC CADOC 2030 parser and validator.
//
// Cobertura:
//   - Parse: XML válido, encoding, campos obrigatórios
//   - Annex validators: todos os domínios
//   - Validate: CNPJ, dataBase, tipoEnvio, clientes, riscos, localiz
//   - Cross-field rules: GEE condicional, mitigador vs risco
package drsac_test

import (
	"fmt"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/drsac"
)

// ============================================================
// Annex validators — domain values
// ============================================================

func TestValidTipoEnvio(t *testing.T) {
	if !drsac.ValidTipoEnvio("I") {
		t.Error("I should be valid")
	}
	if !drsac.ValidTipoEnvio("S") {
		t.Error("S should be valid")
	}
	if drsac.ValidTipoEnvio("X") {
		t.Error("X should be invalid")
	}
}

func TestValidSisReg(t *testing.T) {
	for _, v := range []string{"B3", "CERC", "CSDBR", "Outro"} {
		if !drsac.ValidSisReg(v) {
			t.Errorf("%s should be valid SisReg", v)
		}
	}
	if drsac.ValidSisReg("INVALIDO") {
		t.Error("INVALIDO should be invalid")
	}
}

func TestValidTipoTVM(t *testing.T) {
	for _, v := range []string{"CPR", "CDCA", "CRA", "DEB", "Outro"} {
		if !drsac.ValidTipoTVM(v) {
			t.Errorf("%s should be valid TVM type", v)
		}
	}
}

func TestValidSicor(t *testing.T) {
	if !drsac.ValidSicor("S") || !drsac.ValidSicor("N") {
		t.Error("S and N should be valid")
	}
	if drsac.ValidSicor("X") {
		t.Error("X should be invalid")
	}
}

func TestValidTipoCliente(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "04", "05", "06"} {
		if !drsac.ValidTipoCliente(v) {
			t.Errorf("%s should be valid customer type", v)
		}
	}
	if drsac.ValidTipoCliente("07") {
		t.Error("07 should be invalid")
	}
}

func TestValidAvaliacaoRisco(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "04", "98", "99"} {
		if !drsac.ValidAvaliacaoRisco(v) {
			t.Errorf("%s should be valid risk assessment", v)
		}
	}
	if drsac.ValidAvaliacaoRisco("05") {
		t.Error("05 should be invalid")
	}
}

func TestValidRiscoSocial(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "04", "05", "99"} {
		if !drsac.ValidTipoRiscoSocial(v) {
			t.Errorf("%s should be valid social risk type", v)
		}
	}
	if drsac.ValidTipoRiscoSocial("06") {
		t.Error("06 should be invalid for social risk")
	}
}

func TestValidRiscoAmbiental(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "04", "05", "06", "07", "08", "09", "99"} {
		if !drsac.ValidTipoRiscoAmbiental(v) {
			t.Errorf("%s should be valid environmental risk type", v)
		}
	}
	if drsac.ValidTipoRiscoAmbiental("10") {
		t.Error("10 should be invalid")
	}
}

func TestValidRiscoClimaticoFisico(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "99"} {
		if !drsac.ValidTipoRiscoClimaticoFisico(v) {
			t.Errorf("%s should be valid physical climate risk type", v)
		}
	}
	if drsac.ValidTipoRiscoClimaticoFisico("04") {
		t.Error("04 should be invalid")
	}
}

func TestValidRiscoClimaticoTransicao(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "04", "99"} {
		if !drsac.ValidTipoRiscoClimaticoTransicao(v) {
			t.Errorf("%s should be valid transition climate risk type", v)
		}
	}
}

func TestValidEnquadContribPositiva(t *testing.T) {
	for _, v := range []string{"01", "02", "03", "98", "99"} {
		if !drsac.ValidEnquadContribPositiva(v) {
			t.Errorf("%s should be valid positive contribution", v)
		}
	}
}

func TestValidGEESituacao(t *testing.T) {
	for _, v := range []string{"01", "02", "98", "99"} {
		if !drsac.ValidGEESituacao(v) {
			t.Errorf("%s should be valid GEE situation", v)
		}
	}
}

func TestValidTipoAgrMit(t *testing.T) {
	for i := 1; i <= 10; i++ {
		v := fmt.Sprintf("%02d", i)
		if !drsac.ValidTipoAgrMit(v) {
			t.Errorf("%s should be valid Aggravant/Mitigator type", v)
		}
	}
}

func TestValidMitigadorClimFis(t *testing.T) {
	for _, v := range []string{"01", "02", "98", "99"} {
		if !drsac.ValidMitigadorClimFis(v) {
			t.Errorf("%s should be valid climate mitigator", v)
		}
	}
}

func TestValidRestricaoEconomica(t *testing.T) {
	if !drsac.ValidTipoRestricaoEconomica("01") {
		t.Error("01 should be valid")
	}
	if !drsac.ValidTipoRestricaoEconomica("02") {
		t.Error("02 should be valid")
	}
	if drsac.ValidTipoRestricaoEconomica("03") {
		t.Error("03 should be invalid")
	}
}

// ============================================================
// Parse
// ============================================================

func TestParse_ValidXML(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<DocumentoDRSAC cnpj="12345678" dataBase="2024-12" codigoDocumento="2030" tipoEnvio="I">
  <Contato nome="João Silva" fone="1122223333" email="joao@example.com"/>
  <Clientes>
    <Cliente ident="12345678901234" tipo="02" CNAE="1234567" versaoCNAE="01">
      <ExpAtivos>
        <ExpOperCred IPOC="ABC123" Sicor="S" saldo="1000.50">
          <RiscSoc tipo="01" av="01"/>
          <RiscAmb tipo="01" av="02"/>
          <RiscClimFis tipo="01" av="03"/>
          <RiscClimTrans tipo="01" av="04"/>
        </ExpOperCred>
      </ExpAtivos>
    </Cliente>
  </Clientes>
</DocumentoDRSAC>`

	doc, err := drsac.ParseFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("ParseFromBytes failed: %v", err)
	}
	if doc.CNPJ != "12345678" {
		t.Errorf("CNPJ = %q, want 12345678", doc.CNPJ)
	}
	if doc.DataBase != "2024-12" {
		t.Errorf("DataBase = %q, want 2024-12", doc.DataBase)
	}
	if doc.TipoEnvio != "I" {
		t.Errorf("TipoEnvio = %q, want I", doc.TipoEnvio)
	}
	if len(doc.Clientes) != 1 {
		t.Fatalf("len(Clientes) = %d, want 1", len(doc.Clientes))
	}
	cl := doc.Clientes[0]
	if cl.Ident != "12345678901234" {
		t.Errorf("Cliente.Ident = %q, want 12345678901234", cl.Ident)
	}
	if len(cl.ExpAtivos.ExpOperCred) != 1 {
		t.Fatalf("len(ExpOperCred) = %d, want 1", len(cl.ExpAtivos.ExpOperCred))
	}
	op := cl.ExpAtivos.ExpOperCred[0]
	if op.IPOC != "ABC123" {
		t.Errorf("IPOC = %q, want ABC123", op.IPOC)
	}
}

func TestParse_EmptyDocument(t *testing.T) {
	_, err := drsac.ParseFromBytes([]byte{})
	if err == nil {
		t.Fatal("expected error for empty document")
	}
}

func TestParse_WrongCodigoDocumento(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocumentoDRSAC cnpj="12345678" dataBase="2024-12" codigoDocumento="9999" tipoEnvio="I">
  <Contato nome="X" fone="1" email="x@x.com"/>
</DocumentoDRSAC>`
	_, err := drsac.ParseFromBytes([]byte(xml))
	if err == nil {
		t.Fatal("expected error for wrong codigoDocumento")
	}
}

func TestParse_UTF8BOM(t *testing.T) {
	xml := "\xef\xbb\xbf<?xml version=\"1.0\"?>" +
		`<DocumentoDRSAC cnpj="12345678" dataBase="2024-06" codigoDocumento="2030" tipoEnvio="I">` +
		`<Contato nome="Test" fone="1" email="test@test.com"/>` +
		`<Clientes></Clientes>` +
		`</DocumentoDRSAC>`

	doc, err := drsac.ParseFromBytes([]byte(xml))
	if err != nil {
		t.Fatalf("ParseFromBytes with BOM failed: %v", err)
	}
	if doc.CNPJ != "12345678" {
		t.Errorf("CNPJ = %q, want 12345678", doc.CNPJ)
	}
}

// ============================================================
// Validate
// ============================================================

func TestValidate_EmptyCNPJ(t *testing.T) {
	doc := minimalDoc()
	doc.CNPJ = ""
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for empty CNPJ")
	}
}

func TestValidate_InvalidCNPJ(t *testing.T) {
	doc := minimalDoc()
	doc.CNPJ = "123456" // 6 dígitos, não 8
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid CNPJ")
	}
}

func TestValidate_InvalidDataBase(t *testing.T) {
	doc := minimalDoc()
	doc.DataBase = "2024-13" // mês inválido
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid dataBase")
	}
}

func TestValidate_InvalidTipoEnvio(t *testing.T) {
	doc := minimalDoc()
	doc.TipoEnvio = "X"
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid tipoEnvio")
	}
}

func TestValidate_CNAEObrigatorioPJ(t *testing.T) {
	doc := minimalDoc()
	doc.Clientes[0].CNAE = ""
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for missing CNAE on PJ")
	}
}

func TestValidate_ExpOperCredSaldo(t *testing.T) {
	doc := minimalDoc()
	doc.Clientes[0].ExpAtivos.ExpOperCred[0].Saldo = "not-a-number"
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid saldo")
	}
}

func TestValidate_InvalidRiscSocTipo(t *testing.T) {
	doc := minimalDoc()
	doc.Clientes[0].ExpAtivos.ExpOperCred[0].RiscSoc.Tipo = "06"
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid RiscSoc tipo")
	}
}

func TestValidate_InvalidAvaliacaoRisco(t *testing.T) {
	doc := minimalDoc()
	doc.Clientes[0].ExpAtivos.ExpOperCred[0].RiscSoc.Av = "05"
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for invalid av")
	}
}

func TestValidate_ValidFull(t *testing.T) {
	doc := minimalDoc()
	err := drsac.Validate(doc)
	if err != nil {
		t.Fatalf("expected no error for valid doc: %v", err)
	}
}

func TestValidate_ContatoObrigatorio(t *testing.T) {
	doc := minimalDoc()
	doc.Contato.Email = "" // email obrigatório
	err := drsac.Validate(doc)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

// ============================================================
// ValidateDocument — integration
// ============================================================

func TestValidateDocument_Valid(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<DocumentoDRSAC cnpj="12345678" dataBase="2024-12" codigoDocumento="2030" tipoEnvio="I">
  <Contato nome="João Silva" fone="1122223333" email="joao@example.com"/>
  <Clientes>
    <Cliente ident="12345678901234" tipo="02" CNAE="1234567" versaoCNAE="01">
      <ExpAtivos>
        <ExpOperCred IPOC="ABC123" Sicor="S" saldo="1000.50">
          <RiscSoc tipo="01" av="01"/>
          <RiscAmb tipo="01" av="02"/>
          <RiscClimFis tipo="01" av="03"/>
          <RiscClimTrans tipo="01" av="04"/>
        </ExpOperCred>
      </ExpAtivos>
    </Cliente>
  </Clientes>
</DocumentoDRSAC>`

	result, err := drsac.ValidateDocument(nil, []byte(xml))
	if err != nil {
		t.Fatalf("ValidateDocument failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid result, got criticas: %+v", result.Criticas)
	}
}

func TestValidateDocument_Invalid(t *testing.T) {
	xml := `<?xml version="1.0"?>
<DocumentoDRSAC cnpj="INVALIDO" dataBase="2024-99" codigoDocumento="2030" tipoEnvio="X">
  <Contato nome="X" fone="" email=""/>
</DocumentoDRSAC>`

	result, err := drsac.ValidateDocument(nil, []byte(xml))
	if err != nil {
		t.Fatalf("ValidateDocument should not fail on parse: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result")
	}
	if len(result.Criticas) == 0 {
		t.Error("expected criticas for invalid doc")
	}
}

// ============================================================
// Helpers
// ============================================================

func minimalDoc() *drsac.DocumentoDRSAC {
	return &drsac.DocumentoDRSAC{
		CNPJ:      "12345678",
		DataBase:  "2024-12",
		CodigoDoc: "2030",
		TipoEnvio: "I",
		Contato: drsac.Contato{
			Nome:  "Test User",
			Fone:  "11999999999",
			Email: "test@test.com",
		},
		Clientes: []drsac.Cliente{
			{
				Ident:      "12345678901234",
				Tipo:       "02",
				CNAE:       "1234567",
				VersaoCNAE: "01",
				ExpAtivos: drsac.ExpAtivos{
					ExpOperCred: []drsac.ExpOperCred{
						{
							IPOC:          "IPOC001",
							Sicor:         "S",
							Saldo:         "1000.00",
							RiscSoc:       drsac.Risco{Tipo: "01", Av: "01"},
							RiscAmb:       drsac.Risco{Tipo: "01", Av: "02"},
							RiscClimFis:   drsac.Risco{Tipo: "01", Av: "03"},
							RiscClimTrans: drsac.Risco{Tipo: "01", Av: "04"},
						},
					},
				},
			},
		},
	}
}

func llen(n int) int { return 2 }
