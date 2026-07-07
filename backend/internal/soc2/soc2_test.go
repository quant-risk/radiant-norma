package soc2

import (
	"testing"
	"time"
)

func TestDefaultControls_AllHaveIDs(t *testing.T) {
	controls := DefaultControls()
	seen := make(map[string]bool)
	for _, c := range controls {
		if c.ID == "" {
			t.Errorf("Control with empty ID: %s", c.Name)
		}
		if seen[c.ID] {
			t.Errorf("Duplicate control ID: %s", c.ID)
		}
		seen[c.ID] = true
		if len(c.Criteria) == 0 {
			t.Errorf("Control %s has no criteria", c.ID)
		}
		if c.Status == "" {
			t.Errorf("Control %s has no status", c.ID)
		}
	}
}

func TestControlStatus_Constants(t *testing.T) {
	tests := []struct {
		s    ControlStatus
		want string
	}{
		{ControlStatusImplemented, "implemented"},
		{ControlStatusPartially, "partially_implemented"},
		{ControlStatusNotImplemented, "not_implemented"},
		{ControlStatusNotApplicable, "not_applicable"},
	}
	for _, tt := range tests {
		if string(tt.s) != tt.want {
			t.Errorf("ControlStatus %v = %q, want %q", tt.s, string(tt.s), tt.want)
		}
	}
}

func TestReadinessReport_Statistics(t *testing.T) {
	controls := []Control{
		{ID: "1", Status: ControlStatusImplemented, Criteria: []TrustServiceCriterion{CC1}},
		{ID: "2", Status: ControlStatusImplemented, Criteria: []TrustServiceCriterion{CC1}},
		{ID: "3", Status: ControlStatusPartially, Criteria: []TrustServiceCriterion{CC2}},
		{ID: "4", Status: ControlStatusNotImplemented, Criteria: []TrustServiceCriterion{CC3}},
		{ID: "5", Status: ControlStatusNotApplicable, Criteria: []TrustServiceCriterion{A1}},
	}
	// Verify statistics calculation
	var total, impl, partial, notImpl, notAppl int
	for _, c := range controls {
		total++
		switch c.Status {
		case ControlStatusImplemented:
			impl++
		case ControlStatusPartially:
			partial++
		case ControlStatusNotImplemented:
			notImpl++
		case ControlStatusNotApplicable:
			notAppl++
		}
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if impl != 2 {
		t.Errorf("impl = %d, want 2", impl)
	}
	if partial != 1 {
		t.Errorf("partial = %d, want 1", partial)
	}
	if notImpl != 1 {
		t.Errorf("notImpl = %d, want 1", notImpl)
	}
	if notAppl != 1 {
		t.Errorf("notAppl = %d, want 1", notAppl)
	}
}

func TestTrustServiceCriterion_Constants(t *testing.T) {
	criterions := []TrustServiceCriterion{
		CC1, CC2, CC3, CC4, CC5, CC6, CC7, CC8, CC9, A1, PI1, C1, P1,
	}
	for _, c := range criterions {
		if string(c) == "" {
			t.Errorf("Empty criterion")
		}
	}
}

func TestControlGap_Severity(t *testing.T) {
	gaps := []ControlGap{
		{ControlID: "1", Severity: "high"},
		{ControlID: "2", Severity: "medium"},
		{ControlID: "3", Severity: "low"},
	}
	for _, g := range gaps {
		if g.Severity != "high" && g.Severity != "medium" && g.Severity != "low" {
			t.Errorf("Invalid severity: %s", g.Severity)
		}
	}
}

func TestEvidence_Type(t *testing.T) {
	e := Evidence{
		ID:          "test-id",
		ControlID:   "CC6.6",
		Type:        "log",
		Description: "test",
		CollectedAt: time.Now(),
		CollectedBy: "test",
	}
	if e.Type != "log" {
		t.Errorf("Type = %q, want log", e.Type)
	}
}
