package radiant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	c := New(Config{
		APIKey:  "test-key",
		IFID:    "12345",
		BaseURL: "https://test.local",
	})
	if c.baseURL != "https://test.local" {
		t.Errorf("baseURL: got %s, want https://test.local", c.baseURL)
	}
	if c.apiKey != "test-key" {
		t.Errorf("apiKey: got %s", c.apiKey)
	}
	if c.ifID != "12345" {
		t.Errorf("ifID: got %s", c.ifID)
	}
	if c.Cadocs == nil || c.Audit == nil || c.Radar == nil || c.Insights == nil || c.Schemas == nil {
		t.Error("all services should be initialized")
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	c := New(Config{APIKey: "test"})
	if c.baseURL != "https://api.radiantnorma.com" {
		t.Errorf("default baseURL: got %s", c.baseURL)
	}
}

func TestValidate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header: got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-IF-ID") != "12345" {
			t.Errorf("X-IF-ID header: got %s", r.Header.Get("X-IF-ID"))
		}
		resp := ValidationResult{Valid: true, Errors: []ValidationError{}, Warnings: []ValidationError{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", IFID: "12345", BaseURL: srv.URL})
	result, err := c.Cadocs.Validate(context.Background(), "3040", []byte(`<Doc3040/>`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestValidate_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Error: "bad_request", Message: "invalid xml"})
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	_, err := c.Cadocs.Validate(context.Background(), "3040", []byte(`bad`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"rules": []RuleDef{
				{Code: "F01", Severity: "E", Message: "data base inválida"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	rules, err := c.Audit.ListRules(context.Background(), "3040")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 || rules[0].Code != "F01" {
		t.Errorf("rules: got %+v", rules)
	}
}

func TestScan(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ScanResult{ID: "scan-1", Status: "clean", Changes: []Change{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	scan, err := c.Radar.Scan(context.Background(), "3040")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scan.Status != "clean" {
		t.Errorf("status: got %s", scan.Status)
	}
}

func TestAsk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := LLMAnswer{Answer: "resposta teste", Model: "minimax"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	ans, err := c.Insights.Ask(context.Background(), "o que é PLM?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ans.Answer != "resposta teste" {
		t.Errorf("answer: got %s", ans.Answer)
	}
}

func TestSchemas_ListVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"versions": []SchemaVersion{
				{ID: 1, CadocCode: "3040", EffectiveFrom: "2026-01"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	versions, err := c.Schemas.ListVersions(context.Background(), "3040")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("versions count: got %d", len(versions))
	}
}

func TestSchemas_GetChangelog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries": []SchemaVersion{
				{ID: 1, CadocCode: "3040", EffectiveFrom: "2026-01", Changelog: "+CAMPO ADDED"},
			},
		})
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	entries, err := c.Schemas.GetChangelog(context.Background(), "3040")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0].Changelog != "+CAMPO ADDED" {
		t.Errorf("entries: got %+v", entries)
	}
}

func TestValidateCrossDoc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CrossDocResult{Passed: true, RulesRun: []string{"XD4111CNPJConsistente"}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	result, err := c.Cadocs.ValidateCrossDoc(context.Background(), map[string][]byte{
		"3040": []byte(`<Doc3040/>`),
		"4111": []byte(`<Doc4111/>`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
}
