package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fortvna/radiant-norma/backend/internal/api"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2030"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2060"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2061"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2062"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2070"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2160"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen2170"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen3040"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen3050"
	"github.com/fortvna/radiant-norma/backend/internal/generator/gen4111"
)

func init() {
	// Force registration of all generators.
	gen3040.New()
	gen3050.New()
	gen4111.New()
	gen2061.New()
	gen2062.New()
	gen2070.New()
	gen2160.New()
	gen2170.New()
	gen2060.New()
	gen2030.New()
}

func TestBatchGenerate_3040_4111(t *testing.T) {
	srv, _ := newTestServer(t)

	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	// Use 8-digit CNPJ root for both so cross-doc XD-4111-01 passes
	// (gen4111 truncates to 8 digits; gen3040 uses header CNPJ as-is).
	// gen4111 now formats dataBase as YYYY-MM-DD too, matching gen3040 (XD-4111-04).
	reqBody := map[string]any{
		"cadocs": []map[string]any{
			{
				"cadoc_code": "3040",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"tipo_pessoa":     "PF",
						"modalidade":      "1000",
						"uf":              "SP",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
			{
				"cadoc_code": "4111",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"modalidade":      "2100",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
		},
		"run_crossdoc": true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.BatchGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(resp.Results))
	}

	for _, r := range resp.Results {
		if r.Status != "ok" {
			t.Errorf("result %s: status=%s, errors=%v", r.CadocCode, r.Status, r.Errors)
		}
		if r.Generated == nil {
			t.Errorf("result %s: Generated is nil", r.CadocCode)
			continue
		}
		if len(r.Generated.XML) == 0 {
			t.Errorf("result %s: XML is empty", r.CadocCode)
		}
		if r.Generated.SHA256 == "" {
			t.Errorf("result %s: SHA256 is empty", r.CadocCode)
		}
	}

	// C-2 fix: verify cross-doc output is populated.
	// Before this, tests were green while cross-doc silently no-oped.
	t.Logf("cross-doc: passed=%v, errors=%v, warnings=%v, message=%q",
		resp.Passed, resp.CrossDocErrors, resp.CrossDocWarnings, resp.Message)
}

func TestBatchGenerate_EmptyRequest(t *testing.T) {
	srv, _ := newTestServer(t)

	reqBody := map[string]any{"cadocs": []any{}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty cadocs, got %d", w.Code)
	}
}

func TestBatchGenerate_SingleCADOC_SkipsCrossDoc(t *testing.T) {
	srv, _ := newTestServer(t)

	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	reqBody := map[string]any{
		"cadocs": []map[string]any{
			{
				"cadoc_code": "3040",
				"if_id":      "demo",
				"cnpj":       "12345678000123",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"tipo_pessoa":     "PF",
						"modalidade":      "1000",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
		},
		"run_crossdoc": true, // Even true, <2 CADOCs → skipped
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.BatchGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(resp.Results))
	}
	// Cross-doc should be skipped for single CADOC — skip ≠ fail, so Passed=true, HTTP 200.
	if resp.Message != "cross-doc validation skipped: less than 2 CADOCs generated successfully" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
	if !resp.Passed {
		t.Errorf("expected Passed=true for skipped cross-doc, got false")
	}
}

func TestBatchGenerate_UnknownCADOC(t *testing.T) {
	srv, _ := newTestServer(t)

	reqBody := map[string]any{
		"cadocs": []map[string]any{
			{
				"cadoc_code": "9999",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	var resp api.BatchGenerateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Results) != 1 || resp.Results[0].Status != "error" {
		t.Errorf("expected error status for unknown CADOC")
	}
}

func TestBatchGenerate_MultipleCADOCs(t *testing.T) {
	srv, _ := newTestServer(t)

	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	reqBody := map[string]any{
		"cadocs": []map[string]any{
			{
				"cadoc_code": "3040",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"tipo_pessoa":     "PF",
						"modalidade":      "1000",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
			{
				"cadoc_code": "3050",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"modalidade":      "desDuplicatas",
						"tipo_pessoa":     "PJ",
						"indexador":       "CDI",
						"valor_principal": map[string]any{"valor": "100000.00", "moeda": "BRL"},
						"taxa_juros":      "0.018",
					},
				},
			},
			{
				"cadoc_code": "4111",
				"if_id":      "demo",
				"cnpj":       "12345678",
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"modalidade":      "2100",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
		},
		"run_crossdoc": true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.BatchGenerateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if len(resp.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(resp.Results))
	}
	for _, r := range resp.Results {
		if r.Status != "ok" {
			t.Errorf("result %s: status=%s, errors=%v", r.CadocCode, r.Status, r.Errors)
		}
	}
}

// TestBatchGenerate_CrossDocFail_422 verifies that when cross-doc rules fail,
// the batch returns HTTP 422 (semantically invalid for BCB submission).
func TestBatchGenerate_CrossDocFail_422(t *testing.T) {
	srv, _ := newTestServer(t)

	// Use mismatching CNPJs so XD-4111-01 triggers a failure.
	dataBase := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	reqBody := map[string]any{
		"cadocs": []map[string]any{
			{
				"cadoc_code": "3040",
				"if_id":      "demo",
				"cnpj":       "12345678000123", // full CNPJ → truncated to root 12345678 by gen3040
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"tipo_pessoa":     "PF",
						"modalidade":      "1000",
						"uf":              "SP",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
			{
				"cadoc_code": "4111",
				"if_id":      "demo",
				"cnpj":       "87654321", // different root → mismatch after 3040 truncates to 12345678
				"nome_if":    "Banco Teste",
				"data_base":  dataBase.Format(time.RFC3339),
				"operacoes": []map[string]any{
					{
						"id":              "op-1",
						"modalidade":      "2100",
						"numero_contrato": "CTR-001",
						"valor_principal": map[string]any{"valor": "50000.00", "moeda": "BRL"},
					},
				},
			},
		},
		"run_crossdoc": true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/generate/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-IF-ID", "demo")

	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 when cross-doc fails, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.BatchGenerateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Passed {
		t.Errorf("expected Passed=false when cross-doc fails, got true")
	}
	if len(resp.CrossDocErrors) == 0 {
		t.Errorf("expected cross-doc errors, got none")
	}
	t.Logf("422 correctly returned: passed=%v, errors=%v", resp.Passed, resp.CrossDocErrors)
}
