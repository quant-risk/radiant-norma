package schema

import (
	"strings"
	"testing"
)

func TestComputeChangelog_Empty(t *testing.T) {
	if g := ComputeChangelog(nil, nil); g != "" {
		t.Errorf("nil/nil: want empty, got %q", g)
	}
	if g := ComputeChangelog([]Field{}, []Field{}); g != "" {
		t.Errorf("empty/empty: want empty, got %q", g)
	}
}

func TestComputeChangelog_InitialVersion(t *testing.T) {
	fields := []Field{
		{Tag: "CNPJ", Type: "A8", Required: true, Desc: "CNPJ do IF"},
		{Tag: "Nome", Type: "A100", Required: true, Desc: "Nome"},
	}
	got := ComputeChangelog(nil, fields)
	if !strings.Contains(got, "versão inicial") {
		t.Errorf("nil prev: expected 'versão inicial', got %q", got)
	}
	if !strings.Contains(got, "2 campos") {
		t.Errorf("nil prev: expected '2 campos', got %q", got)
	}
}

func TestComputeChangelog_Added(t *testing.T) {
	prev := []Field{{Tag: "CNPJ", Type: "A8", Required: true}}
	curr := []Field{
		{Tag: "CNPJ", Type: "A8", Required: true},
		{Tag: "Novo", Type: "A20", Required: false, Desc: "novo campo"},
	}
	got := ComputeChangelog(prev, curr)
	if !strings.Contains(got, "+CAMPO ADDED") {
		t.Errorf("expected +CAMPO ADDED, got %q", got)
	}
	if !strings.Contains(got, "tag=Novo") {
		t.Errorf("expected tag=Novo, got %q", got)
	}
}

func TestComputeChangelog_Removed(t *testing.T) {
	prev := []Field{
		{Tag: "CNPJ", Type: "A8", Required: true},
		{Tag: "Velho", Type: "A20"},
	}
	curr := []Field{{Tag: "CNPJ", Type: "A8", Required: true}}
	got := ComputeChangelog(prev, curr)
	if !strings.Contains(got, "-CAMPO REMOVED") {
		t.Errorf("expected -CAMPO REMOVED, got %q", got)
	}
	if !strings.Contains(got, "tag=Velho") {
		t.Errorf("expected tag=Velho, got %q", got)
	}
}

func TestComputeChangelog_ModifiedType(t *testing.T) {
	prev := []Field{{Tag: "Saldo", Type: "N13,2", Required: false}}
	curr := []Field{{Tag: "Saldo", Type: "N15,2", Required: false}}
	got := ComputeChangelog(prev, curr)
	if !strings.Contains(got, "~CAMPO MODIFIED") {
		t.Errorf("expected ~CAMPO MODIFIED, got %q", got)
	}
	if !strings.Contains(got, "type N13,2 → N15,2") {
		t.Errorf("expected type change, got %q", got)
	}
}

func TestComputeChangelog_ModifiedRequired(t *testing.T) {
	prev := []Field{{Tag: "CNPJ", Type: "A8", Required: false}}
	curr := []Field{{Tag: "CNPJ", Type: "A8", Required: true}}
	got := ComputeChangelog(prev, curr)
	if !strings.Contains(got, "required false → true") {
		t.Errorf("expected required change, got %q", got)
	}
}

func TestComputeChangelog_NoChanges(t *testing.T) {
	fields := []Field{
		{Tag: "CNPJ", Type: "A8", Required: true, Desc: "CNPJ", Group: "root"},
	}
	got := ComputeChangelog(fields, fields)
	if got != "" {
		t.Errorf("no changes: expected empty, got %q", got)
	}
}

func TestComputeChangelog_MultipleChanges(t *testing.T) {
	prev := []Field{
		{Tag: "CNPJ", Type: "A8", Required: false, Desc: "old"},
		{Tag: "Velho", Type: "N10"},
	}
	curr := []Field{
		{Tag: "CNPJ", Type: "A14", Required: true, Desc: "new"},
		{Tag: "Novo", Type: "N20"},
	}
	got := ComputeChangelog(prev, curr)
	if !strings.Contains(got, "+CAMPO ADDED") {
		t.Errorf("expected +CAMPO ADDED, got %q", got)
	}
	if !strings.Contains(got, "-CAMPO REMOVED") {
		t.Errorf("expected -CAMPO REMOVED, got %q", got)
	}
	if !strings.Contains(got, "~CAMPO MODIFIED") {
		t.Errorf("expected ~CAMPO MODIFIED, got %q", got)
	}
}

func TestComputeChangelog_Deterministic(t *testing.T) {
	prev := []Field{
		{Tag: "A", Type: "N1"}, {Tag: "B", Type: "N2"}, {Tag: "C", Type: "N3"},
	}
	curr := []Field{
		{Tag: "D", Type: "N4"}, {Tag: "E", Type: "N5"}, {Tag: "F", Type: "N6"},
	}
	// Run twice — should be identical
	a := ComputeChangelog(prev, curr)
	b := ComputeChangelog(prev, curr)
	if a != b {
		t.Errorf("non-deterministic: %q vs %q", a, b)
	}
}
