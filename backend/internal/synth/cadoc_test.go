// cadoc_test.go — Sprint 57 v3.36.3: testes para CadocType expansion.
package synth

import "testing"

func TestAllCadocTypes(t *testing.T) {
	got := AllCadocTypes()
	if len(got) != 10 {
		t.Errorf("expected 10 CADOCs, got %d", len(got))
	}
	want := map[CadocType]bool{
		Cadoc2030: true, Cadoc2060: true, Cadoc2061: true, Cadoc2062: true,
		Cadoc2070: true, Cadoc2160: true, Cadoc2170: true,
		Cadoc3040: true, Cadoc3050: true, Cadoc4111: true,
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected CADOC %q in AllCadocTypes", c)
		}
	}
}

func TestValidCadocType(t *testing.T) {
	tests := []struct {
		c    CadocType
		want bool
	}{
		{Cadoc2030, true},
		{Cadoc2060, true},
		{Cadoc3040, true},
		{Cadoc3050, true},
		{Cadoc4111, true},
		{CadocType("9999"), false},
		{CadocType("invalid"), false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidCadocType(tt.c); got != tt.want {
			t.Errorf("ValidCadocType(%q)=%v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestPromptForCadoc_AllSupported(t *testing.T) {
	// Cada CADOC suportado deve ter prompt não-vazio e não-default.
	for _, c := range AllCadocTypes() {
		p := promptForCadoc(c)
		if p == "" {
			t.Errorf("empty prompt for %q", c)
		}
		// Não pode cair no default (que diz apenas "Gere XML para o documento X")
		if len(p) < 50 {
			t.Errorf("prompt for %q too short (likely default fallback): %q", c, p)
		}
	}
}

func TestPromptForCadoc_DefaultFallback(t *testing.T) {
	// CadocType desconhecido deve cair no default sem panic.
	p := promptForCadoc(CadocType("9999"))
	if p == "" {
		t.Error("default prompt should not be empty")
	}
	if !contains(p, "9999") {
		t.Errorf("default prompt should mention CADOC code: %q", p)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
