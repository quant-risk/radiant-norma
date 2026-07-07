package marketplace

import (
	"testing"
)

func TestValidateRuleType(t *testing.T) {
	valid := []string{"format", "semantic", "crossdoc", "raw"}
	for _, rt := range valid {
		if err := validateRuleType(rt); err != nil {
			t.Errorf("validateRuleType(%q): unexpected error %v", rt, err)
		}
	}

	invalid := []string{"invalid", "Format", "SEMANTIC", ""}
	for _, rt := range invalid {
		if err := validateRuleType(rt); err == nil {
			t.Errorf("validateRuleType(%q): expected error, got nil", rt)
		}
	}
}

func TestValidateCode(t *testing.T) {
	valid := []string{
		"CUSTOM_001",
		"CUSTOM_BANCO_TESTE",
		"X_MinhaRegra",
		"AUDIT_FOO",
		"CUSTOM_AB",
	}

	for _, code := range valid {
		if err := validateCode(code); err != nil {
			t.Errorf("validateCode(%q): unexpected error %v", code, err)
		}
	}

	invalid := []string{
		"",                                      // too short
		"ABC",                                   // too short
		"BAD_PREFIX",                            // wrong prefix
		"MY_CODE",                               // wrong prefix
		"CUSTOM_ABC" + string(make([]byte, 40)), // too long
		"CUSTOM_A B",                            // contains space
		"CUSTOM_A<B>",                           // contains <
	}

	for _, code := range invalid {
		if err := validateCode(code); err == nil {
			t.Errorf("validateCode(%q): expected error, got nil", code)
		}
	}
}

func TestRuleStruct(t *testing.T) {
	r := Rule{
		ID:           "id-1",
		Name:         "Test Rule",
		Description:  "A test rule",
		Code:         "CUSTOM_TEST",
		Cadoc:        "3040",
		RuleType:     "semantic",
		AuthorIFID:   "12345",
		Rating:       4.5,
		RatingCount:  10,
		InstallCount: 50,
		Tags:         []string{"credito", "limite"},
		Active:       true,
	}

	if r.Code != "CUSTOM_TEST" {
		t.Errorf("Code: got %s, want CUSTOM_TEST", r.Code)
	}
	if r.Rating != 4.5 {
		t.Errorf("Rating: got %f, want 4.5", r.Rating)
	}
	if len(r.Tags) != 2 {
		t.Errorf("Tags len: got %d, want 2", len(r.Tags))
	}
}

func TestPublishRuleRequest(t *testing.T) {
	req := PublishRuleRequest{
		Name:        "My Rule",
		Description: "A test rule",
		Code:        "CUSTOM_MINE",
		Cadoc:       "3040",
		RuleType:    "format",
		Config:      map[string]any{"threshold": 0.5},
		AuthorIFID:  "12345",
		AuthorName:  "Test Corp",
		Tags:        []string{"test"},
	}

	if req.Name != "My Rule" {
		t.Errorf("Name: got %s", req.Name)
	}
	if req.RuleType != "format" {
		t.Errorf("RuleType: got %s", req.RuleType)
	}
	if req.Config["threshold"] != 0.5 {
		t.Errorf("Config threshold: got %v", req.Config["threshold"])
	}
}
