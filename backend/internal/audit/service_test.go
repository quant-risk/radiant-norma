// Tests for audit.Service (Validate + UnmarshalJSON + LoadCriticas).
package audit_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/audit"
	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// Helper: XML 3040 mínimo válido
const validXML = `<?xml version="1.0"?>
<Doc3040 DtBase="2020-08" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="1">
  <Agreg NatuOp="01" Mod="0213" OrigemRec="0100" VincME="N" ClassOp="A" FaixaVlr="2" PrzProvm="N" Localiz="10058" TpCli="1" DesempOp="01" ProvConsttd="0" QtdOp="1" QtdCli="1">
    <Venc v110="100" v120="0" v150="0" v160="0" v165="0"/>
  </Agreg>
</Doc3040>`

type criticaFixture struct {
	Codigo    string
	Sheet     string
	Gravidade string
	Enabled   bool
}

// seedCriticas insere críticas fake para teste.
func seedCriticas(t *testing.T, d *sql.DB, cs []criticaFixture) {
	t.Helper()
	for _, c := range cs {
		enabled := 0
		if c.Enabled {
			enabled = 1
		}
		_, err := d.ExecContext(context.Background(), `
			INSERT INTO criticas (cadoc_code, sheet, codigo, regra, descricao, gravidade, mensagem_erro, enabled)
			VALUES ('3040', ?, ?, 'regra', 'descrição', ?, 'msg', ?)
		`, c.Sheet, c.Codigo, c.Gravidade, enabled)
		if err != nil {
			t.Fatalf("seed %s: %v", c.Codigo, err)
		}
	}
}

// ============================================================
// UnmarshalJSON customizado: aceita "cadoc" ou "cadoc_code"
// ============================================================

func TestUnmarshalJSON_CadocCode(t *testing.T) {
	body := []byte(`{"cadoc_code":"3040","xml":"<x/>"}`)
	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.CadocCode != "3040" {
		t.Errorf("CadocCode = %q, want 3040", req.CadocCode)
	}
}

func TestUnmarshalJSON_CadocAlias(t *testing.T) {
	body := []byte(`{"cadoc":"3040","xml":"<x/>"}`)
	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.CadocCode != "3040" {
		t.Errorf("alias 'cadoc' não foi convertido, got CadocCode=%q", req.CadocCode)
	}
}

func TestUnmarshalJSON_CadocCodePrecedence(t *testing.T) {
	// Se ambos enviados, cadoc_code tem precedência
	body := []byte(`{"cadoc":"3050","cadoc_code":"3040","xml":"<x/>"}`)
	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.CadocCode != "3040" {
		t.Errorf("cadoc_code deve preceder, got %q", req.CadocCode)
	}
}

func TestUnmarshalJSON_Invalid(t *testing.T) {
	body := []byte(`{invalid}`)
	var req audit.ValidationRequest
	if err := json.Unmarshal(body, &req); err == nil {
		t.Error("esperado erro de parse")
	}
}

// ============================================================
// Validate — entrypoint principal
// ============================================================

func TestValidate_XMLValido_Passa(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	// Popula F02 habilitada (vai validar formato de DtBase)
	seedCriticas(t, d, []criticaFixture{
		{Codigo: "F02", Sheet: "Formato", Gravidade: "E", Enabled: true},
	})

	resp, err := svc.Validate(context.Background(), &audit.ValidationRequest{
		CadocCode: "3040",
		DataBase:  "2020-08",
		XML:       validXML,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !resp.Passed {
		t.Errorf("Validate com XML válido deveria passar, got errors=%v", resp.Errors)
	}
}

func TestValidate_XMLQuebrado_L1Parse(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	resp, err := svc.Validate(context.Background(), &audit.ValidationRequest{
		CadocCode: "3040",
		DataBase:  "2020-08",
		XML:       "<broken",
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if resp.Passed {
		t.Error("Validate com XML quebrado deveria falhar")
	}
	if len(resp.Errors) == 0 {
		t.Error("esperado pelo menos 1 erro (L1-PARSE)")
	}
	found := false
	for _, e := range resp.Errors {
		if e.Critica.Codigo == "L1-PARSE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperado erro L1-PARSE, got: %v", resp.Errors)
	}
}

func TestValidate_DtBaseInvalido_F02Detecta(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	seedCriticas(t, d, []criticaFixture{
		{Codigo: "F02", Sheet: "Formato", Gravidade: "E", Enabled: true},
	})

	xml := strings.Replace(validXML, `DtBase="2020-08"`, `DtBase="20-08"`, 1)
	resp, err := svc.Validate(context.Background(), &audit.ValidationRequest{
		CadocCode: "3040",
		XML:       xml,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	foundF02 := false
	for _, e := range resp.Errors {
		if e.Critica.Codigo == "F02" {
			foundF02 = true
		}
	}
	if !foundF02 {
		t.Error("esperado erro F02 para DtBase inválido")
	}
}

// TestValidate_RegrasDesabilitadas valida fix v1.3.3: regras com enabled=0
// são carregadas mas NÃO executadas.
func TestValidate_RegrasDesabilitadas(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	seedCriticas(t, d, []criticaFixture{
		{Codigo: "F02", Sheet: "Formato", Gravidade: "E", Enabled: false},
	})

	xml := strings.Replace(validXML, `DtBase="2020-08"`, `DtBase="20-08"`, 1)
	resp, err := svc.Validate(context.Background(), &audit.ValidationRequest{
		CadocCode: "3040",
		XML:       xml,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, e := range resp.Errors {
		if e.Critica.Codigo == "F02" {
			t.Errorf("F02 desabilitada não deveria aparecer nos errors")
		}
	}
}

// ============================================================
// LoadCriticas
// ============================================================

func TestLoadCriticas_RetornaTodas(t *testing.T) {
	// v1.3.3+: LoadCriticas retorna TODAS (habilitadas + desabilitadas)
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	seedCriticas(t, d, []criticaFixture{
		{Codigo: "F02", Sheet: "Formato", Gravidade: "E", Enabled: true},
		{Codigo: "S04", Sheet: "Semantica", Gravidade: "E", Enabled: false},
		{Codigo: "S05", Sheet: "Semantica", Gravidade: "E", Enabled: true},
	})

	criticas, err := svc.LoadCriticas(context.Background(), "3040")
	if err != nil {
		t.Fatalf("LoadCriticas: %v", err)
	}
	if len(criticas) != 3 {
		t.Errorf("esperado 3 críticas (todas), got %d", len(criticas))
	}
	enabledCount := 0
	for _, c := range criticas {
		if c.Enabled {
			enabledCount++
		}
	}
	if enabledCount != 2 {
		t.Errorf("esperado 2 habilitadas, got %d", enabledCount)
	}
}

func TestLoadCriticas_CadocInexistente(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	criticas, err := svc.LoadCriticas(context.Background(), "9999")
	if err != nil {
		t.Fatalf("LoadCriticas: %v", err)
	}
	if len(criticas) != 0 {
		t.Errorf("cadoc inexistente deveria retornar slice vazio, got %d", len(criticas))
	}
}

// TestLoadCriticas_MensagemErroNULL testa o fix v1.4.0: LoadCriticas
// não quebra quando `mensagem_erro` é NULL no DB (registros antigos
// ou INSERTs manuais sem a coluna).
//
// Antes do fix: Scan error "converting NULL to string is unsupported"
// → L2-LOAD error → validação inteira quebrava.
func TestLoadCriticas_MensagemErroNULL(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	// Insert com mensagem_erro NULL explícito
	_, err := d.ExecContext(context.Background(), `
		INSERT INTO criticas (cadoc_code, sheet, codigo, regra, descricao, gravidade, enabled, mensagem_erro)
		VALUES ('3040', 'Formato', 'F02', 'regra', 'desc', '', 1, NULL)
	`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	criticas, err := svc.LoadCriticas(context.Background(), "3040")
	if err != nil {
		t.Fatalf("LoadCriticas não deveria falhar com mensagem_erro NULL: %v", err)
	}
	if len(criticas) != 1 {
		t.Fatalf("esperado 1 crítica, got %d", len(criticas))
	}
	if criticas[0].MensagemErro != "" {
		t.Errorf("MensagemErro = %q, esperado string vazia (NULL convertido)", criticas[0].MensagemErro)
	}
}

// TestValidate_F02_MesInvalido valida o fix de F02 em v1.4.0: mês 13 detectado.
func TestValidate_F02_MesInvalido(t *testing.T) {
	d := testutil.NewTestDB(t)
	svc := audit.New(d)

	seedCriticas(t, d, []criticaFixture{
		{Codigo: "F02", Sheet: "Formato", Gravidade: "E", Enabled: true},
	})

	xml := `<?xml version="1.0"?>
<Doc3040 DtBase="2020-13" CNPJ="12345678" Remessa="1" Parte="1" TpArq="F" TotalCli="0"></Doc3040>`

	resp, err := svc.Validate(context.Background(), &audit.ValidationRequest{
		CadocCode: "3040",
		DataBase:  "2020-13",
		XML:       xml,
	})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	foundF02 := false
	for _, e := range resp.Errors {
		if e.Critica.Codigo == "F02" {
			foundF02 = true
			if !strings.Contains(e.Message, "13") {
				t.Errorf("erro F02 deveria mencionar mês 13, got: %s", e.Message)
			}
		}
	}
	if !foundF02 {
		t.Error("F02 habilitado deveria detectar mês 13 (DtBase='2020-13')")
	}
}
