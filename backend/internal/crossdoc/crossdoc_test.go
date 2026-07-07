// Tests para Cross-Doc engine + regras iniciais.
//
// Cobertura:
//   - Engine.Validate: básico, regras faltantes são skipped, regras
//     aplicáveis rodam, erros/warnings categorizados corretamente
//   - TotalOperacoes3040Consistente4111: passa, tol. 5%, falha > 5%
//   - Modalidade0213FlagChequeEspecial: N/A sem 0213, mismatch
//   - DRSACSubsegmentoClassificacaoRisco: S4/S5 com score baixo
//   - DocSet helpers
//   - extractTextBetween / CountTag / parseNum
package crossdoc_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
	crossrules "github.com/fortvna/radiant-norma/backend/internal/crossdoc/rules"
)

// ============================
// Helpers
// ============================

// valid3040XML retorna um XML 3040 com QtdOp=10 em 1 Agreg.
func valid3040XML(qtdOp float64) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Doc3040>
  <Agreg>
    <Mod>0213</Mod>
    <QtdOp>` + numStr(qtdOp) + `</QtdOp>
  </Agreg>
  <Agreg>
    <Mod>0201</Mod>
    <QtdOp>5</QtdOp>
  </Agreg>
</Doc3040>`
}

// valid4111XML retorna um XML 4111 com N clientes, total QtdCli.
func valid4111XML(qtdCli float64) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Doc4111>
  <Cliente><QtdCli>` + numStr(qtdCli) + `</QtdCli></Cliente>
</Doc4111>`
}

func numStr(n float64) string {
	return strings.TrimRight(strings.TrimRight(
		// converte float → string sem expoente
		formatFloat(n), "0"), ".")
}

func formatFloat(n float64) string {
	// Para simplificar: inteiro ou .X
	if n == float64(int(n)) {
		return intStr(int(n))
	}
	return intStr(int(n)) + "." + intStr(int((n-float64(int(n)))*100))
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// ============================
// Engine
// ============================

func TestEngine_Validate_Basic(t *testing.T) {
	engine := crossdoc.NewEngine(crossrules.BuiltinRegistry())
	req := &crossdoc.ValidationRequest{
		Cadocs: map[string]string{
			"3040": valid3040XML(10),
			"4111": valid4111XML(11),
		},
	}
	resp := engine.Validate(context.Background(), req)

	// 3040+4111: totalOps 15 vs qtdCli 11 — diff 4 (> 5% de 11) → warning XD-001
	// XD-002 (Mod 0213) também roda mas matches (Mod existe no valid3040XML).
	// Sem 2030, regras DRSAC (XD-DR01~08) ficam skipped.
	// 4111 rules (XD-4111-01~05) rodam (RequiredDocs 3040+4111).
	// Rules run: XD-001, XD-002, XD-4111-01~05 → 7
	// Rules skip: XD-003, XD-DR01~08 → 9
	if !resp.Passed {
		t.Errorf("sem erros 'E', deveria passar. Warnings: %d", len(resp.Warnings))
	}
	if len(resp.RulesRun) != 7 {
		t.Errorf("Esperado 7 regras run (XD-001, XD-002, XD-4111-01~05), got %d (%v)",
			len(resp.RulesRun), resp.RulesRun)
	}
	if len(resp.RulesSkip) != 9 {
		t.Errorf("Esperado 9 regras skip (XD-003 + XD-DR01~08), got %d", len(resp.RulesSkip))
	}
}

func TestEngine_Validate_RequiredDocsMissing(t *testing.T) {
	engine := crossdoc.NewEngine(crossrules.BuiltinRegistry())
	// Com apenas 3040 — TODAS as 16 regras devem ser skipped:
	//   - XD-001, XD-002 precisam 3040+4111
	//   - XD-003 precisa 3040+2030
	//   - XD-DR01~08 precisam 2030+3040
	//   - XD-4111-01~05 precisam 4111+3040
	req := &crossdoc.ValidationRequest{
		Cadocs: map[string]string{
			"3040": valid3040XML(10),
		},
	}
	resp := engine.Validate(context.Background(), req)

	if len(resp.RulesRun) != 0 {
		t.Errorf("Nenhuma regra deveria rodar (faltam docs obrigatórios), got %d run",
			len(resp.RulesRun))
	}
	if len(resp.RulesSkip) != 16 {
		t.Errorf("16 regras deveriam ser skipped, got %d", len(resp.RulesSkip))
	}
}

func TestEngine_Validate_PassedClean(t *testing.T) {
	engine := crossdoc.NewEngine(crossrules.BuiltinRegistry())
	// 3040 ops = 15, 4111 clients = 15 → diff 0, ratio 0, passa
	req := &crossdoc.ValidationRequest{
		Cadocs: map[string]string{
			"3040": valid3040XML(10), // +5 do outro agreg = 15 total
			"4111": valid4111XML(15),
		},
	}
	resp := engine.Validate(context.Background(), req)
	if len(resp.Errors) != 0 {
		t.Errorf("Esperado 0 erros, got %v", resp.Errors)
	}
}

func TestEngine_Validate_HighDiscrepancy(t *testing.T) {
	engine := crossdoc.NewEngine(crossrules.BuiltinRegistry())
	// 3040 ops = 15, 4111 clients = 5 → diff 10 (66% > 5%) → warning XD-001
	req := &crossdoc.ValidationRequest{
		Cadocs: map[string]string{
			"3040": valid3040XML(10),
			"4111": valid4111XML(5),
		},
	}
	resp := engine.Validate(context.Background(), req)
	if len(resp.Warnings) == 0 {
		t.Errorf("Esperado warning XD-001, got 0")
	}
}

func TestEngine_Validate_EmptyRequest(t *testing.T) {
	engine := crossdoc.NewEngine(crossrules.BuiltinRegistry())
	req := &crossdoc.ValidationRequest{Cadocs: nil}
	resp := engine.Validate(context.Background(), req)
	if resp.Passed {
		t.Errorf("Deveria falhar com cadocs vazio")
	}
}

// ============================
// DocSet helpers
// ============================

func TestDocSet_HasAndGet(t *testing.T) {
	d := &crossdoc.DocSet{
		Cadocs: map[string]string{"3040": "xml1", "4111": "xml2"},
	}
	if !d.Has("3040") {
		t.Errorf("Deveria ter 3040")
	}
	if d.Has("9999") {
		t.Errorf("Não deveria ter 9999")
	}
	if d.Get("4111") != "xml2" {
		t.Errorf("Get 4111 = %q, want xml2", d.Get("4111"))
	}
}

// ============================
// Helpers públicos
// ============================

func TestExtractTextBetween(t *testing.T) {
	xml := "<Root><CNPJ>12345</CNPJ></Root>"
	if got := crossdoc.ExtractTextBetween(xml, "CNPJ"); got != "12345" {
		t.Errorf("ExtractTextBetween = %q, want 12345", got)
	}
	if got := crossdoc.ExtractTextBetween(xml, "INEXISTENTE"); got != "" {
		t.Errorf("Inexistente deveria retornar vazio, got %q", got)
	}
}

func TestCountTag(t *testing.T) {
	xml := "<Root><A/><A/><A/><B/></Root>"
	if got := crossdoc.CountTag(xml, "A"); got != 3 {
		t.Errorf("CountTag A = %d, want 3", got)
	}
	if got := crossdoc.CountTag(xml, "B"); got != 1 {
		t.Errorf("CountTag B = %d, want 1", got)
	}
}

func TestExtractSumOfTag(t *testing.T) {
	xml := `<?xml version="1.0"?>
<Root>
  <Item><Qtd>10</Qtd></Item>
  <Item><Qtd>20</Qtd></Item>
  <Item><Qtd>30</Qtd></Item>
</Root>`
	got := crossdoc.ExtractSumOfTag(xml, "Item", "Qtd")
	if got != 60 {
		t.Errorf("ExtractSumOfTag = %v, want 60", got)
	}
}

// ============================
// Registry builtin
// ============================

func TestBuiltinRegistry(t *testing.T) {
	r := crossrules.BuiltinRegistry()
	codes := r.Codes()
	// Sprint 52 v3.34.33: 3 originais + 8 DRSAC + 5 4111 = 16
	if len(codes) != 16 {
		t.Errorf("Builtin deveria ter 16 regras, got %d", len(codes))
	}

	// Confirma que todas as 16 regras estão lá
	// 3 originais + 8 DRSAC + 5 4111
	expected := map[string]bool{
		"XD-001": true, "XD-002": true, "XD-003": true, // originais
		"XD-DR01": true, "XD-DR02": true, "XD-DR03": true, // DRSAC 1-3
		"XD-DR04": true, "XD-DR05": true, "XD-DR06": true, // DRSAC 4-6
		"XD-DR07": true, "XD-DR08": true, // DRSAC 7-8
		"XD-4111-01": true, "XD-4111-02": true, "XD-4111-03": true, // 4111 1-3
		"XD-4111-04": true, "XD-4111-05": true, // 4111 4-5
	}
	for _, c := range codes {
		if !expected[c] {
			t.Errorf("Código inesperado: %s", c)
		}
		delete(expected, c)
	}
	if len(expected) > 0 {
		t.Errorf("Códigos faltando: %v", expected)
	}
}

func TestError_Error(t *testing.T) {
	e := crossdoc.NewError("XD-TEST", "E", "test message")
	if e.Error() != "[XD-TEST/E] test message" {
		t.Errorf("Error() formato errado: %q", e.Error())
	}
}
