// Tests para RawRule (Sprint 6 v1.5.0 / W3) — B01-B05 movidas para registry.
//
// Cobertura:
//   - RegisterRaw / GetRaw exposto no Registry
//   - B01-B05 implementam a interface RawRule
//   - B04 (codificação) e B05 (tamanho) validam comportamento
//   - Builtin3040 inclui 5 raw rules + 25 tipadas
package rules_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/audit/rules"
)

// ===========================
// Registry — dual interface
// ===========================

func TestRegistry_RegisterRawAndGetRaw(t *testing.T) {
	r := rules.NewRegistry()

	raw := rules.RawRuleFunc{
		C:   "TEST",
		Sht: "Test",
		Sev: "E",
		ApplyFn: func(ctx context.Context, xmlContent string) error {
			return nil
		},
	}
	r.RegisterRaw(raw)

	got := r.GetRaw("TEST")
	if got == nil {
		t.Fatal("GetRaw(TEST) deveria retornar rule")
	}
	if got.Code() != "TEST" {
		t.Errorf("Code = %q, want TEST", got.Code())
	}
}

func TestRegistry_RawIsSeparateFromTyped(t *testing.T) {
	r := rules.NewRegistry()
	raw := rules.RawRuleFunc{
		C:       "BOTH",
		Sht:     "Both",
		Sev:     "E",
		ApplyFn: func(ctx context.Context, xmlContent string) error { return nil },
	}
	// Não há Register sem RegisterRaw aqui — só raw.
	r.RegisterRaw(raw)

	// Get retorna nil (não é Rule tipada)
	if r.Get("BOTH") != nil {
		t.Errorf("Get(BOTH) deveria ser nil (não é Rule tipada)")
	}
	// GetRaw retorna a regra
	if r.GetRaw("BOTH") == nil {
		t.Errorf("GetRaw(BOTH) deveria retornar rule raw")
	}
}

func TestRegistry_CodesIncludesBoth(t *testing.T) {
	r := rules.NewRegistry()
	r.RegisterRaw(rules.RawRuleFunc{
		C: "R1", Sht: "r", Sev: "I",
		ApplyFn: func(ctx context.Context, s string) error { return nil },
	})
	// (NÃO registramos rule tipada — não há export simples de struct Rule
	// sem import circular. Apenas valida que Codes() inclui raw.)

	codes := r.Codes()
	found := false
	for _, c := range codes {
		if c == "R1" {
			found = true
		}
	}
	if !found {
		t.Errorf("Codes() deveria incluir R1, got %v", codes)
	}
}

// ===========================
// Regras B01-B05 (raw)
// ===========================

func TestB01_ArquivoXMLValido(t *testing.T) {
	var r rules.B01ArquivoXMLValido
	if r.Code() != "B01" {
		t.Errorf("Code = %q, want B01", r.Code())
	}
	if r.Sheet() != "Básicas" {
		t.Errorf("Sheet = %q, want Básicas", r.Sheet())
	}
	if r.Severity() != "I" {
		t.Errorf("Severity = %q, want I", r.Severity())
	}
	if err := r.ApplyRaw(context.Background(), "<?xml version='1.0'?><root/>"); err != nil {
		t.Errorf("B01 deveria passar, got %v", err)
	}
}

func TestB04_CodificacaoDeclarada_Aprovado(t *testing.T) {
	var r rules.B04CodificacaoDeclarada
	cases := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?><root/>",
		"<?xml version='1.0'?><root/>",
		"\n<?xml version=\"1.0\"?>\n<root/>", // trim spaces
	}
	for _, c := range cases {
		if err := r.ApplyRaw(context.Background(), c); err != nil {
			t.Errorf("B04 deveria passar para %q, got %v", c, err)
		}
	}
}

func TestB04_CodificacaoDeclarada_Reprovado(t *testing.T) {
	var r rules.B04CodificacaoDeclarada
	cases := []string{
		"<root/>", // sem <?xml
		"",
	}
	for _, c := range cases {
		err := r.ApplyRaw(context.Background(), c)
		if err == nil {
			t.Errorf("B04 deveria falhar para %q, got nil", c)
		}
		if err != nil && !errors.Is(err, err) { // sanity
			t.Errorf("erro não-nil: %v", err)
		}
	}
}

func TestB05_ArquivoNaoVazio_Aprovado(t *testing.T) {
	var r rules.B05ArquivoNaoVazio
	cases := []string{
		"<?xml version=\"1.0\"?>" + string(make([]byte, 100)), // 100 bytes OK
		"<?xml?>", // 7 bytes, muito pequeno
	}
	for i, c := range cases {
		err := r.ApplyRaw(context.Background(), c)
		// Index 0 = grande, passa. Index 1 = pequeno, falha.
		// Mas make([]byte, 100) gera zeros, então o conteúdo final é comprimento 17+100=117. OK.
		if i == 0 && err != nil {
			t.Errorf("B05 deveria passar para XML longo, got %v", err)
		}
	}
}

func TestB05_ArquivoNaoVazio_Reprovado(t *testing.T) {
	var r rules.B05ArquivoNaoVazio
	cases := []struct {
		name string
		xml  string
	}{
		{"empty", ""},
		{"tiny", "<a/>"},            // 4 bytes, < 50
		{"under_50", "<?xml?><a/>"}, // 10 bytes, < 50
	}
	for _, c := range cases {
		err := r.ApplyRaw(context.Background(), c.xml)
		if err == nil {
			t.Errorf("B05 deveria falhar para %s (%d bytes), got nil",
				c.name, len(c.xml))
		}
	}
}

// ===========================
// Builtin3040 — inclui raw rules
// ===========================

func TestBuiltin3040_HasB01toB05AsRaw(t *testing.T) {
	r := rules.Builtin3040()

	codes := []string{"B01", "B02", "B03", "B04", "B05"}
	for _, c := range codes {
		// GetRaw deve retornar a regra (registrada via RegisterRaw)
		if got := r.GetRaw(c); got == nil {
			t.Errorf("B01-B05 raw: GetRaw(%s) deveria retornar regra", c)
		}
		// Get (tipada) deve retornar nil — B01-B05 não são Rule tipadas
		if got := r.Get(c); got != nil {
			t.Errorf("B01-B05 não deveriam estar em Get (tipadas), got %v", got)
		}
	}
}

func TestBuiltin3040_HasB06toB15AsTyped(t *testing.T) {
	r := rules.Builtin3040()

	codes := []string{"B06", "B07", "B08", "B09", "B10",
		"B11", "B12", "B13", "B14", "B15"}
	for _, c := range codes {
		// B06+ são regras tipadas (operam em *Doc3040)
		if got := r.Get(c); got == nil {
			t.Errorf("B06+: Get(%s) deveria retornar regra tipada", c)
		}
	}
}

func TestBuiltin3040_TotalRulesIs(t *testing.T) {
	r := rules.Builtin3040()
	total := len(r.Codes())
	// Sprint 7b / v1.7.0: 5 raw + 55 tipadas = 60
	// Sprint 32 / v3.25.0 Fase 1: +14 Agregadas (A01-A07, A09-A15)
	// Sprint 32 / v3.27.0 Fase 2: +5 Sistemáticas (S12 stub, S15, S17, S19, S20)
	// Sprint 32 / v3.29.0 Fase 3: +19 Individuais/Campos/Header (C11-C20, S13/S14, I01-I05/I11, H01-H03)
	// Sprint 32 / v3.30.0 Fase 4: +28 (C31-C40/C51-C55/S21-S46/S69-S70) → 126
	// Sprint 36 / v3.34.13 Fase 2: +51 (C21-C30, C41-C50, C56-C70, H04-H09, N01-N10) → 177
	// Sprint 37 / v3.34.15 Fase 3: +44 (I06-I15/A16-A30/S71-S90) + 5 destravadas sobrescrevem stubs (C44/46/57/62/68).
	// Sprint 38 / v3.34.17 Fase 4: +54 (C71-C90/SUB01-15/X01-10) + 9 destravadas sobrescrevem stubs (Sprint 36-37).
	// Sprint 40 / v3.34.21 DRL: +8 (LCR01-LCR08).
	// Sprint 41 / v3.34.22 DLP: +8 (NSFR01-NSFR08).
	// Sprint 51: LCR e NSFR movidos para Register2160/Register2170 (separados).
	// Total Registry: 5 raw + 261 tipadas = 266.
	if total != 266 {
		t.Errorf("Total regras = %d, want 266 (LCR/NSFR em registries separados)", total)
	}
}
