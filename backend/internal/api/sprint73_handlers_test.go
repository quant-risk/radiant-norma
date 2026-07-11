// Sprint 74 — Tests para handlers do Sprint 73.
//
// Endpoints testados:
//   - GET /v1/crossdoc/rules     → listCrossDocRules
//   - GET /v1/schema             → listSchemasV2
//   - GET /v1/generate/history   → listGenerateHistory
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fortvna/radiant-norma/backend/internal/testutil"
)

// --- listCrossDocRules ---

func TestListCrossDocRules_HappyPath(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/v1/crossdoc/rules", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Rules []struct {
			Code         string   `json:"code"`
			Description  string   `json:"description"`
			Severity     string   `json:"severity"`
			RequiredDocs []string `json:"required_docs"`
		} `json:"rules"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least one rule (XD-001..XD-003 + XD-4111-01..05 + DRSAC)")
	}

	found := false
	for _, r := range resp.Rules {
		if r.Code == "XD-001" {
			found = true
			if r.Severity != "A" {
				t.Errorf("XD-001 severity: got %s, want A", r.Severity)
			}
			if len(r.RequiredDocs) != 2 || r.RequiredDocs[0] != "3040" || r.RequiredDocs[1] != "4111" {
				t.Errorf("XD-001 required_docs: got %v", r.RequiredDocs)
			}
		}
	}
	if !found {
		t.Error("XD-001 rule not found in response")
	}
}

func TestListCrossDocRules_EngineNil(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.CrossDoc = nil // engine desabilitado

	req := httptest.NewRequest("GET", "/v1/crossdoc/rules", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

// --- listSchemasV2 ---

func TestListSchemasV2_HappyPath(t *testing.T) {
	srv, d := newTestServer(t)

	// Seed schema_versions so ListCadocs returns CADOCs.
	d.Exec(`INSERT OR IGNORE INTO schema_versions
		(cadoc_code, effective_from, source_uri, fields_json, xsd, changelog, created_at)
		VALUES ('3040', '2025-01-01', 'https://example.com/3040.xsd',
			'[]', '', '', datetime('now')),
		       ('4111', '2025-01-01', 'https://example.com/4111.xsd',
			'[]', '', '', datetime('now'))`)

	req := httptest.NewRequest("GET", "/v1/schema", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Schemas []struct {
			CadocCode         string   `json:"cadoc_code"`
			SupportedVersions []string `json:"supported_versions"`
			FieldCount        int      `json:"field_count"`
			Complexity        struct {
				Score float64 `json:"score"`
			} `json:"complexity"`
		} `json:"schemas"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least one schema (generators auto-registered)")
	}

	for _, s := range resp.Schemas {
		if s.CadocCode == "3040" {
			if s.FieldCount == 0 {
				t.Error("3040 field_count should be > 0")
			}
			if len(s.SupportedVersions) == 0 {
				t.Error("3040 supported_versions should be non-empty")
			}
			if s.Complexity.Score < 0 || s.Complexity.Score > 1 {
				t.Errorf("3040 complexity score out of range: %v", s.Complexity.Score)
			}
		}
	}
}

// --- listGenerateHistory ---

func TestListGenerateHistory_HappyPath(t *testing.T) {
	srv, d := newTestServer(t)

	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Cadoc: "3040", Status: "accepted", DaysAgo: 0},
		{ID: "E2", Cadoc: "4111", Status: "validated", DaysAgo: 1},
		{ID: "E3", Cadoc: "3040", Status: "pending", DaysAgo: 2},
	})

	req := httptest.NewRequest("GET", "/v1/generate/history", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Items []struct {
			ID        string `json:"id"`
			CadocCode string `json:"cadoc_code"`
			Status    string `json:"status"`
			Passed    bool   `json:"passed"`
		} `json:"items"`
		Page    int `json:"page"`
		PerPage int `json:"per_page"`
		Total   int `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 items, got %d", resp.Total)
	}
	for _, item := range resp.Items {
		if item.Status == "accepted" || item.Status == "validated" {
			if !item.Passed {
				t.Errorf("status=%s should have Passed=true", item.Status)
			}
		}
	}
}

func TestListGenerateHistory_Empty(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/v1/generate/history", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Items []struct{ ID string `json:"id"` } `json:"items"`
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 0 {
		t.Errorf("expected 0 items, got %d", resp.Total)
	}
	if resp.Items == nil {
		t.Error("items should be [], not nil")
	}
}

func TestListGenerateHistory_Pagination(t *testing.T) {
	srv, d := newTestServer(t)

	for i := 0; i < 5; i++ {
		testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
			{ID: "E" + string(rune('A'+i)), Cadoc: "3040", Status: "accepted", DaysAgo: i},
		})
	}

	tests := []struct {
		page    string
		perPage string
		wantLen int
	}{
		{"1", "2", 2},
		{"2", "2", 2},
		{"3", "2", 1},
		{"10", "2", 0},
	}

	for _, tt := range tests {
		t.Run("page="+tt.page+"_perPage="+tt.perPage, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/generate/history?page="+tt.page+"&per_page="+tt.perPage, nil)
			req.Header.Set("X-IF-ID", "demo")
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, req)

			var resp struct {
				Items []struct{ ID string `json:"id"` } `json:"items"`
			}
			json.Unmarshal(w.Body.Bytes(), &resp)
			if len(resp.Items) != tt.wantLen {
				t.Errorf("page=%s,per_page=%s: got %d items, want %d", tt.page, tt.perPage, len(resp.Items), tt.wantLen)
			}
		})
	}
}

func TestListGenerateHistory_PageZeroClamp(t *testing.T) {
	srv, d := newTestServer(t)
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Cadoc: "3040", Status: "accepted"},
	})

	req := httptest.NewRequest("GET", "/v1/generate/history?page=0", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp struct{ Page int `json:"page"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Page != 1 {
		t.Errorf("page=0 should clamp to 1, got %d", resp.Page)
	}
}

func TestListGenerateHistory_PerPageMaxClamp(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/v1/generate/history?per_page=9999", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp struct{ PerPage int `json:"per_page"` }
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.PerPage != 100 {
		t.Errorf("per_page=9999 should clamp to 100, got %d", resp.PerPage)
	}
}

func TestListGenerateHistory_DBAvailable(t *testing.T) {
	srv, _ := newTestServer(t)

	req := httptest.NewRequest("GET", "/v1/generate/history", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when DB configured, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListGenerateHistory_FilterByCadoc(t *testing.T) {
	srv, d := newTestServer(t)

	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E1", Cadoc: "3040", Status: "accepted"},
		{ID: "E2", Cadoc: "4111", Status: "accepted"},
		{ID: "E3", Cadoc: "3040", Status: "accepted"},
	})

	req := httptest.NewRequest("GET", "/v1/generate/history?cadoc=3040", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp struct {
		Items []struct{ CadocCode string `json:"cadoc_code"` } `json:"items"`
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 2 {
		t.Errorf("expected 2 items for cadoc=3040, got %d", resp.Total)
	}
	for _, item := range resp.Items {
		if item.CadocCode != "3040" {
			t.Errorf("unexpected cadoc %s in filtered results", item.CadocCode)
		}
	}
}

func TestListGenerateHistory_CrossTenantIsolation(t *testing.T) {
	srv, d := newTestServer(t)

	testutil.SeedTestEnvios(t, d, "other-if", []testutil.EnvioFixture{
		{ID: "E-OTHER", Cadoc: "3040", Status: "accepted"},
	})
	testutil.SeedTestEnvios(t, d, "demo", []testutil.EnvioFixture{
		{ID: "E-DEMO", Cadoc: "3040", Status: "accepted"},
	})

	req := httptest.NewRequest("GET", "/v1/generate/history", nil)
	req.Header.Set("X-IF-ID", "demo")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp struct {
		Items []struct{ ID string `json:"id"` } `json:"items"`
		Total int `json:"total"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Total != 1 {
		t.Errorf("demo should only see 1 envio (its own), got %d", resp.Total)
	}
	if len(resp.Items) > 0 && resp.Items[0].ID != "E-DEMO" {
		t.Errorf("expected E-DEMO, got %s", resp.Items[0].ID)
	}
}
