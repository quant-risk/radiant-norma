// xd_rules_test.go — Sprint 57 v3.36.3: testes para XD02-XD12.
//
// Foco em:
//   - Code(), Description(), Severity(), RequiredDocs() de cada regra
//   - Apply() com DocSets mínimos (XMLs válidos vs inválidos)
//   - Engine registry carrega todas as 9 regras
package rules

import (
	"context"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/crossdoc"
)

// =======================================================================
// Meta tests (Code, Description, Severity, RequiredDocs)
// =======================================================================

func TestXD02_Meta(t *testing.T) {
	r := XD02TotalOperacoes3040vs3050{}
	if r.Code() != "XD02" {
		t.Errorf("Code()=%q, want XD02", r.Code())
	}
	if r.Severity() != "A" {
		t.Errorf("Severity()=%q, want A", r.Severity())
	}
	reqs := r.RequiredDocs()
	if len(reqs) != 2 || reqs[0] != "3040" || reqs[1] != "3050" {
		t.Errorf("RequiredDocs()=%v, want [3040, 3050]", reqs)
	}
}

func TestXD03_Meta(t *testing.T) {
	r := XD03LCRvsNSFR{}
	if r.Code() != "XD03" {
		t.Errorf("Code()=%q, want XD03", r.Code())
	}
	reqs := r.RequiredDocs()
	if len(reqs) != 2 {
		t.Errorf("RequiredDocs() len=%d, want 2", len(reqs))
	}
}

func TestXD06_Meta(t *testing.T) {
	r := XD06APRemLCR{}
	if r.Code() != "XD06" {
		t.Errorf("Code()=%q, want XD06", r.Code())
	}
}

func TestXD07_Meta(t *testing.T) {
	r := XD07Triangulacao304041113050{}
	if r.Code() != "XD07" {
		t.Errorf("Code()=%q, want XD07", r.Code())
	}
	reqs := r.RequiredDocs()
	if len(reqs) != 3 {
		t.Errorf("RequiredDocs() len=%d, want 3", len(reqs))
	}
}

func TestXD08_Meta(t *testing.T) {
	r := XD08LimitesvsCapital{}
	if r.Code() != "XD08" {
		t.Errorf("Code()=%q, want XD08", r.Code())
	}
}

func TestXD09_Meta(t *testing.T) {
	r := XD09LiquidezvsRisco{}
	if r.Code() != "XD09" {
		t.Errorf("Code()=%q, want XD09", r.Code())
	}
}

func TestXD10_Meta(t *testing.T) {
	r := XD10ESGvsInadimplencia{}
	if r.Code() != "XD10" {
		t.Errorf("Code()=%q, want XD10", r.Code())
	}
}

func TestXD11_Meta(t *testing.T) {
	r := XD11ESGvs4111{}
	if r.Code() != "XD11" {
		t.Errorf("Code()=%q, want XD11", r.Code())
	}
}

func TestXD12_Meta(t *testing.T) {
	r := XD12DataBaseConsistente{}
	if r.Code() != "XD12" {
		t.Errorf("Code()=%q, want XD12", r.Code())
	}
}

// =======================================================================
// Apply tests — comportamento de cada regra
// =======================================================================

func TestXD02_Apply_MissingDocs_NoError(t *testing.T) {
	r := XD02TotalOperacoes3040vs3050{}
	docs := &crossdoc.DocSet{Cadocs: map[string]string{}}
	if err := r.Apply(context.Background(), docs); err != nil {
		t.Errorf("Apply with empty docs: %v (should skip)", err)
	}
}

func TestXD02_Apply_InvalidXML_NoError(t *testing.T) {
	r := XD02TotalOperacoes3040vs3050{}
	docs := &crossdoc.DocSet{Cadocs: map[string]string{
		"3040": "<invalid",
		"3050": "<also invalid",
	}}
	// Parser retorna erro → regra silently skip (defesa contra XML malformado).
	if err := r.Apply(context.Background(), docs); err != nil {
		t.Errorf("Apply with invalid XML should skip: %v", err)
	}
}

func TestXD12_Apply_DataBaseConsistency_NoPanic(t *testing.T) {
	r := XD12DataBaseConsistente{}

	// DocSets com stubs mínimos — só verificamos que não panica.
	docsOK := &crossdoc.DocSet{Cadocs: map[string]string{
		"3040": "<Doc3040 dataBase=\"2026-06\" cnpj=\"123\"/>",
		"3050": "<DocTXB cnpj=\"123\" dataBase=\"2026-06\"/>",
	}}
	_ = r.Apply(context.Background(), docsOK)
}

// =======================================================================
// Registry — todas as 9 regras registradas
// =======================================================================

func TestBuiltinRegistry_ContainsAllXDRules(t *testing.T) {
	reg := BuiltinRegistry()

	expectedCodes := []string{"XD02", "XD03", "XD06", "XD07", "XD08", "XD09", "XD10", "XD11", "XD12"}
	for _, code := range expectedCodes {
		rule := reg.Get(code)
		if rule == nil {
			t.Errorf("registry missing %s", code)
			continue
		}
		if rule.Code() != code {
			t.Errorf("rule at code %s returns %s", code, rule.Code())
		}
	}
}

func TestBuiltinRegistry_AllRulesHaveRequiredDocs(t *testing.T) {
	reg := BuiltinRegistry()
	rules := reg.All()
	if len(rules) == 0 {
		t.Fatal("registry empty")
	}
	for _, r := range rules {
		reqs := r.RequiredDocs()
		if len(reqs) == 0 {
			t.Errorf("rule %s has no RequiredDocs", r.Code())
		}
		for _, code := range reqs {
			if len(code) != 4 {
				t.Errorf("rule %s has malformed RequiredDoc %q (expected 4-digit CADOC code)", r.Code(), code)
			}
		}
	}
}

func TestBuiltinRegistry_AllRulesHaveValidSeverity(t *testing.T) {
	reg := BuiltinRegistry()
	rules := reg.All()
	validSeverities := map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true, "I": true, "": true}
	for _, r := range rules {
		sev := r.Severity()
		if !validSeverities[sev] {
			t.Errorf("rule %s has invalid severity %q", r.Code(), sev)
		}
	}
}
