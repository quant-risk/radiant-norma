package radiant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Healthz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/healthz", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "3.36.2"})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.Healthz(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "3.36.2", resp.Version)
}

func TestClient_ListSchemasV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/schema", r.URL.Path)
		json.NewEncoder(w).Encode(SchemaListResponse{
			Schemas: []SchemaInfo{
				{CadocCode: "3040", LatestVersion: "3.2", SupportedVersions: []string{"3.0", "3.1", "3.2"}},
				{CadocCode: "4111", LatestVersion: "3.10", SupportedVersions: []string{"3.8", "3.9", "3.10"}},
			},
			Total: 2,
		})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.ListSchemasV2(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)
	assert.Equal(t, "3040", resp.Schemas[0].CadocCode)
	assert.Equal(t, "3.2", resp.Schemas[0].LatestVersion)
	// Verify "3.10" > "3.9" is correctly returned (not "3.9")
	assert.Equal(t, "4111", resp.Schemas[1].CadocCode)
	assert.Equal(t, "3.10", resp.Schemas[1].LatestVersion)
}

func TestClient_Validate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/validate", r.URL.Path)
		var req ValidateRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "3040", req.CadocCode)

		json.NewEncoder(w).Encode(ValidateResponse{
			CadocCode: "3040",
			Passed:    true,
			Errors:    []ValidationError{},
		})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.Validate(context.Background(), &ValidateRequest{
		CadocCode: "3040",
		Xml:       "<SCRDocumento>...</SCRDocumento>",
	})
	require.NoError(t, err)
	assert.True(t, resp.Passed)
	assert.Equal(t, "3040", resp.CadocCode)
}

func TestClient_GenerateCadoc(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/generate/3040", r.URL.Path)
		var req GenerateRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "if_demo", req.IfID)

		json.NewEncoder(w).Encode(GenerateResponse{
			CadocCode: "3040",
			Status:    "generated",
			Generated: &GeneratedDoc{
				XML:     "<SCRDocumento>...generated...</SCRDocumento>",
				XMLHash: "a3f2c1b4d5e6",
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.GenerateCadoc(context.Background(), "3040", &GenerateRequest{
		IfID:     "if_demo",
		Cnpj:     "12.345.678/0001-90",
		DataBase: "2026-06-30",
		Participantes: []Participante{
			{Id: "P001", Tipo: "PF", Nome: "Joao Silva", Cpf: "123.456.789-00", Rating: "AA"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "generated", resp.Status)
	assert.NotEmpty(t, resp.Generated.XML)
}

func TestClient_ListCrossDocRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/crossdoc/rules", r.URL.Path)
		json.NewEncoder(w).Encode(CrossDocRulesResponse{
			Rules: []CrossDocRule{
				{Code: "XD-4111-3040", Description: "XDRR01IPOC must match 3040", Severity: "error"},
			},
			Total: 1,
		})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.ListCrossDocRules(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, "XD-4111-3040", resp.Rules[0].Code)
}

func TestClient_CrossDocValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/crossdoc/validate", r.URL.Path)
		var req CrossDocValidateRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Len(t, req.Documents, 2)

		json.NewEncoder(w).Encode(CrossDocValidateResponse{
			Passed:        true,
			Errors:        []CrossDocError{},
			RulesExecuted: 3,
		})
	}))
	defer srv.Close()

	c, err := NewClient(NewConfig(srv.URL, "test-token"))
	require.NoError(t, err)

	resp, err := c.CrossDocValidate(context.Background(), &CrossDocValidateRequest{
		Documents: []CrossDocInput{
			{CadocCode: "4111", Xml: "<XDRR01IPOC>...</XDRR01IPOC>"},
			{CadocCode: "3040", Xml: "<SCRDocumento>...</SCRDocumento>"},
		},
	})
	require.NoError(t, err)
	assert.True(t, resp.Passed)
	assert.Equal(t, 3, resp.RulesExecuted)
}

func TestHTTPError(t *testing.T) {
	e := &HTTPError{StatusCode: 404, Code: "NOT_FOUND", Message: "schema not found"}
	assert.Contains(t, e.Error(), "NOT_FOUND")
	assert.Contains(t, e.Error(), "schema not found")

	e2 := &HTTPError{StatusCode: 500, Message: "internal server error"}
	assert.Contains(t, e2.Error(), "500")
}

func TestConfig_Defaults(t *testing.T) {
	cfg := NewConfig("https://api.radiantrisk.com/v1", "token")
	assert.NotNil(t, cfg.HTTPClient)
	assert.Equal(t, "token", cfg.AuthToken)
}
