package pilot

import (
	"testing"
)

func TestValidSegment(t *testing.T) {
	valid := []string{"s1", "s2", "s3", "s4"}
	for _, seg := range valid {
		if !ValidSegment(seg) {
			t.Errorf("ValidSegment(%q): got false, want true", seg)
		}
	}

	invalid := []string{"S1", "s5", "bank", "", "S3"}
	for _, seg := range invalid {
		if ValidSegment(seg) {
			t.Errorf("ValidSegment(%q): got true, want false", seg)
		}
	}
}

func TestNewID(t *testing.T) {
	id := newID()
	if id == "" || len(id) != 36 {
		t.Errorf("newID(): got %q, want 36-char UUID", id)
	}
	id2 := newID()
	if id == id2 {
		t.Error("two calls to newID() should produce different IDs")
	}
}

func TestProgramStruct(t *testing.T) {
	p := Program{
		ID:          "prog-1",
		Name:        "Banco S3-S4 Pilot 4",
		Description: "Pilot for S3/S4 segment",
		Active:      true,
	}
	if p.Name != "Banco S3-S4 Pilot 4" {
		t.Errorf("Name: got %s", p.Name)
	}
	if !p.Active {
		t.Error("Active should be true")
	}
}

func TestParticipantStruct(t *testing.T) {
	p := Participant{
		ID:        "p-1",
		ProgramID: "prog-1",
		IFID:      "12345",
		Status:    "onboarding",
		Notes:     "test note",
	}
	if p.Status != "onboarding" {
		t.Errorf("Status: got %s", p.Status)
	}
}

func TestOnboardingStepStruct(t *testing.T) {
	step := OnboardingStep{
		ID:      "step-1",
		IFID:    "12345",
		StepKey: "docs_submitted",
		Status:  "completed",
	}
	if step.Status != "completed" {
		t.Errorf("Status: got %s", step.Status)
	}
	if step.StepKey != "docs_submitted" {
		t.Errorf("StepKey: got %s", step.StepKey)
	}
}

func TestDefaultSteps(t *testing.T) {
	expected := []string{
		"docs_submitted",
		"cadoc_tested",
		"integration_verified",
		"production_approved",
		"go_live",
	}
	if len(DefaultSteps) != len(expected) {
		t.Errorf("DefaultSteps len: got %d, want %d", len(DefaultSteps), len(expected))
	}
	for i, e := range expected {
		if DefaultSteps[i] != e {
			t.Errorf("DefaultSteps[%d]: got %q, want %q", i, DefaultSteps[i], e)
		}
	}
}

func TestESGSteps(t *testing.T) {
	expected := []string{
		"drsac_policy_configured",
		"drsac_first_submission",
		"crossdoc_drsac_verified",
		"esg_dashboard_configured",
		"esg_go_live",
	}
	if len(ESGSteps) != len(expected) {
		t.Errorf("ESGSteps len: got %d, want %d", len(ESGSteps), len(expected))
	}
	for i, e := range expected {
		if ESGSteps[i] != e {
			t.Errorf("ESGSteps[%d]: got %q, want %q", i, ESGSteps[i], e)
		}
	}
}

func TestPilot3Constants(t *testing.T) {
	if pilot3Name == "" {
		t.Error("pilot3Name should not be empty")
	}
	if pilot3Description == "" {
		t.Error("pilot3Description should not be empty")
	}
	if pilot3Name != "Pilot 3 — ESG-first" {
		t.Errorf("pilot3Name: got %q", pilot3Name)
	}
}
