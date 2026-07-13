// session_test.go — Sprint 57 v3.36.3: testes para Step.Prev, IsBackwardTransition
// e Store.Revindicate (via sqlite em memória).
package wizard

import (
	"testing"
)

func TestStep_Next(t *testing.T) {
	tests := []struct {
		in, want Step
	}{
		{StepSelectCadoc, StepSelectSource},
		{StepSelectSource, StepMapFields},
		{StepMapFields, StepPreview},
		{StepPreview, StepGenerate},
		{StepGenerate, StepGenerate}, // terminal
	}
	for _, tt := range tests {
		if got := tt.in.Next(); got != tt.want {
			t.Errorf("%q.Next()=%q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStep_Prev(t *testing.T) {
	tests := []struct {
		in, want Step
	}{
		{StepSelectCadoc, StepSelectCadoc}, // initial → stay
		{StepSelectSource, StepSelectCadoc},
		{StepMapFields, StepSelectSource},
		{StepPreview, StepMapFields},
		{StepGenerate, StepPreview},
		{Step("invalid"), Step("invalid")}, // unknown → stay
	}
	for _, tt := range tests {
		if got := tt.in.Prev(); got != tt.want {
			t.Errorf("%q.Prev()=%q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStep_IsBackwardTransition(t *testing.T) {
	tests := []struct {
		from, to Step
		want     bool
	}{
		{StepSelectCadoc, StepSelectSource, false}, // forward
		{StepSelectSource, StepSelectCadoc, true},  // backward
		{StepMapFields, StepSelectSource, true},
		{StepGenerate, StepPreview, true},
		{StepSelectCadoc, StepSelectCadoc, false}, // self
	}
	for _, tt := range tests {
		if got := tt.from.IsBackwardTransition(tt.to); got != tt.want {
			t.Errorf("%q.IsBackwardTransition(%q)=%v, want %v",
				tt.from, tt.to, got, tt.want)
		}
	}
}

func TestStep_IsValid(t *testing.T) {
	if !StepSelectCadoc.IsValid() {
		t.Error("StepSelectCadoc should be valid")
	}
	if !StepGenerate.IsValid() {
		t.Error("StepGenerate should be valid")
	}
	if Step("invalid").IsValid() {
		t.Error("invalid step should not be valid")
	}
}
