package multiregion

import (
	"testing"
)

func TestParseRegion(t *testing.T) {
	tests := []struct {
		input string
		want  Region
		err   bool
	}{
		{"br-sp1", RegionBRSP1, false},
		{"br-sp2", RegionBRSP2, false},
		{"BR-SP1", RegionBRSP1, false},
		{"BR-SP2", RegionBRSP2, false},
		{"brsp1", RegionBRSP1, false},
		{"brsp2", RegionBRSP2, false},
		{"invalid", "", true},
		{"", "", true},
		{"us-east-1", "", true},
	}

	for _, tt := range tests {
		got, err := ParseRegion(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("ParseRegion(%q): err=%v, want err=%v", tt.input, err, tt.err)
		}
		if !tt.err && got != tt.want {
			t.Errorf("ParseRegion(%q): got %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestValidRegion(t *testing.T) {
	valid := []string{"br-sp1", "br-sp2", "BR-SP1", "BR-SP2"}
	for _, r := range valid {
		if !ValidRegion(r) {
			t.Errorf("ValidRegion(%q): got false, want true", r)
		}
	}

	invalid := []string{"br-sp3", "us-east", "", "br_sp1"}
	for _, r := range invalid {
		if ValidRegion(r) {
			t.Errorf("ValidRegion(%q): got true, want false", r)
		}
	}
}

func TestNewService(t *testing.T) {
	s := NewService(nil, RegionBRSP1)
	if s.self != RegionBRSP1 {
		t.Errorf("self: got %v, want br-sp1", s.self)
	}
	if s.peer != RegionBRSP2 {
		t.Errorf("peer: got %v, want br-sp2", s.peer)
	}

	s2 := NewService(nil, RegionBRSP2)
	if s2.peer != RegionBRSP1 {
		t.Errorf("peer br-sp2: got %v, want br-sp1", s2.peer)
	}
}

func TestService_IsLocal(t *testing.T) {
	s := NewService(nil, RegionBRSP1)
	if !s.IsLocal(RegionBRSP1) {
		t.Error("br-sp1 should be local")
	}
	if s.IsLocal(RegionBRSP2) {
		t.Error("br-sp2 should not be local for br-sp1")
	}
}

func TestService_ShouldReplicate(t *testing.T) {
	s := NewService(nil, RegionBRSP1)
	// br-sp2 events should replicate to br-sp1 (peer)
	if !s.ShouldReplicate(RegionBRSP2) {
		t.Error("br-sp2 should replicate to br-sp1")
	}
	// br-sp1 events should NOT replicate to br-sp1 (self)
	if s.ShouldReplicate(RegionBRSP1) {
		t.Error("br-sp1 should not replicate to self")
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

func TestReplicationStatus(t *testing.T) {
	rs := &ReplicationStatus{
		Region:     RegionBRSP1,
		LagSeconds: 2,
		Status:     "healthy",
	}
	if rs.Region != RegionBRSP1 {
		t.Errorf("Region: got %v", rs.Region)
	}
	if rs.Status != "healthy" {
		t.Errorf("Status: got %v", rs.Status)
	}
}

func TestReplicationEvent(t *testing.T) {
	e := ReplicationEvent{
		ID:         "id-1",
		RegionFrom: RegionBRSP1,
		RegionTo:   RegionBRSP2,
		EventType:  "tenant.created",
		EntityType: "tenant",
		EntityID:   "12345",
		Payload:    `{"name":"test"}`,
	}
	if e.RegionFrom != RegionBRSP1 {
		t.Errorf("RegionFrom: got %v", e.RegionFrom)
	}
	if e.RegionTo != RegionBRSP2 {
		t.Errorf("RegionTo: got %v", e.RegionTo)
	}
}
