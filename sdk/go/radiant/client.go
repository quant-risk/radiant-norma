// Package radiant — Go client for Radiant Norma API.
//
// Auto-generated types from OpenAPI v3.36.2 (docs/openapi/v1.yaml).
//
// Usage:
//
//	cfg := radiant.NewConfig("https://api.radiantrisk.com/v1", "your-jwt-token")
//	c, err := radiant.NewClient(cfg)
//	schemas, err := c.ListSchemasV2(ctx)
package radiant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Config holds client configuration.
type Config struct {
	BaseURL   string // e.g. "https://api.radiantrisk.com/v1"
	AuthToken string // JWT Bearer token
	IFID      string // optional X-IF-ID header
	HTTPClient *http.Client
}

// NewConfig creates a Config with defaults.
func NewConfig(baseURL, authToken string) *Config {
	return &Config{
		BaseURL:    baseURL,
		AuthToken:  authToken,
		HTTPClient: http.DefaultClient,
	}
}

// Client is the Radiant Norma API client.
type Client struct {
	cfg  *Config
	http *http.Client
	base *url.URL
}

// NewClient creates a new API client.
func NewClient(cfg *Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.radiantrisk.com/v1"
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient, base: base}, nil
}

func (c *Client) auth(req *http.Request) {
	if c.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AuthToken)
	}
	if c.cfg.IFID != "" {
		req.Header.Set("X-IF-ID", c.cfg.IFID)
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	u := c.base.ResolveReference(&url.URL{Path: path})
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("json marshal: %w", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.auth(req)
	return c.http.Do(req)
}

func (c *Client) doOK(ctx context.Context, method, path string, body any, out any) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return readError(resp)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Healthz calls GET /healthz.
func (c *Client) Healthz(ctx context.Context) (*HealthResponse, error) {
	var out HealthResponse
	err := c.doOK(ctx, http.MethodGet, "/healthz", nil, &out)
	return &out, err
}

// Readyz calls GET /readyz.
func (c *Client) Readyz(ctx context.Context) (*ReadyResponse, error) {
	var out ReadyResponse
	err := c.doOK(ctx, http.MethodGet, "/readyz", nil, &out)
	return &out, err
}

// ListSchemas calls GET /schemas — returns basic CADOC list.
func (c *Client) ListSchemas(ctx context.Context) (*SchemasResponse, error) {
	var out SchemasResponse
	err := c.doOK(ctx, http.MethodGet, "/schemas", nil, &out)
	return &out, err
}

// ListSchemasV2 calls GET /schema — returns enriched schema list with complexity.
func (c *Client) ListSchemasV2(ctx context.Context) (*SchemaListResponse, error) {
	var out SchemaListResponse
	err := c.doOK(ctx, http.MethodGet, "/schema", nil, &out)
	return &out, err
}

// GetSchema calls GET /schemas/{cadoc}.
func (c *Client) GetSchema(ctx context.Context, cadoc string) (*SchemaVersion, error) {
	var out SchemaVersion
	err := c.doOK(ctx, http.MethodGet, "/schemas/"+cadoc, nil, &out)
	return &out, err
}

// ListVersions calls GET /schemas/{cadoc}/versions.
func (c *Client) ListVersions(ctx context.Context, cadoc string) (*VersionsResponse, error) {
	var out VersionsResponse
	err := c.doOK(ctx, http.MethodGet, "/schemas/"+cadoc+"/versions", nil, &out)
	return &out, err
}

// ListRules calls GET /rules — returns CADOCs with rules available.
func (c *Client) ListRules(ctx context.Context) (*RulesListResponse, error) {
	var out RulesListResponse
	err := c.doOK(ctx, http.MethodGet, "/rules", nil, &out)
	return &out, err
}

// ListRulesByCadoc calls GET /rules/{cadoc}.
func (c *Client) ListRulesByCadoc(ctx context.Context, cadoc string) (*RuleListResponse, error) {
	var out RuleListResponse
	err := c.doOK(ctx, http.MethodGet, "/rules/"+cadoc, nil, &out)
	return &out, err
}

// Validate calls POST /validate.
func (c *Client) Validate(ctx context.Context, req *ValidateRequest) (*ValidateResponse, error) {
	var out ValidateResponse
	err := c.doOK(ctx, http.MethodPost, "/validate", req, &out)
	return &out, err
}

// GenerateCadoc calls POST /generate/{cadoc}.
func (c *Client) GenerateCadoc(ctx context.Context, cadoc string, req *GenerateRequest) (*GenerateResponse, error) {
	var out GenerateResponse
	err := c.doOK(ctx, http.MethodPost, "/generate/"+cadoc, req, &out)
	return &out, err
}

// ListGenerateFields calls GET /generate/{cadoc}/fields.
func (c *Client) ListGenerateFields(ctx context.Context, cadoc string) (*FieldsResponse, error) {
	var out FieldsResponse
	err := c.doOK(ctx, http.MethodGet, "/generate/"+cadoc+"/fields", nil, &out)
	return &out, err
}

// ListSourceAdapters calls GET /generate/adapters.
func (c *Client) ListSourceAdapters(ctx context.Context) (*AdaptersResponse, error) {
	var out AdaptersResponse
	err := c.doOK(ctx, http.MethodGet, "/generate/adapters", nil, &out)
	return &out, err
}

// GenerateBatch calls POST /generate/batch.
func (c *Client) GenerateBatch(ctx context.Context, req *BatchGenerateRequest) (*BatchGenerateResponse, error) {
	var out BatchGenerateResponse
	err := c.doOK(ctx, http.MethodPost, "/generate/batch", req, &out)
	return &out, err
}

// ListGenerateHistory calls GET /generate/history.
func (c *Client) ListGenerateHistory(ctx context.Context, page, perPage int) (*GenerationHistoryResponse, error) {
	u := c.base.ResolveReference(&url.URL{
		Path: "/generate/history",
		RawQuery: fmt.Sprintf("page=%d&per_page=%d", page, perPage),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.auth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, readError(resp)
	}
	var out GenerationHistoryResponse
	err = json.NewDecoder(resp.Body).Decode(&out)
	return &out, err
}

// ListCrossDocRules calls GET /crossdoc/rules.
func (c *Client) ListCrossDocRules(ctx context.Context) (*CrossDocRulesResponse, error) {
	var out CrossDocRulesResponse
	err := c.doOK(ctx, http.MethodGet, "/crossdoc/rules", nil, &out)
	return &out, err
}

// CrossDocValidate calls POST /crossdoc/validate.
func (c *Client) CrossDocValidate(ctx context.Context, req *CrossDocValidateRequest) (*CrossDocValidateResponse, error) {
	var out CrossDocValidateResponse
	err := c.doOK(ctx, http.MethodPost, "/crossdoc/validate", req, &out)
	return &out, err
}

// =====================================================================
// Types
// =====================================================================

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type ReadyResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

type SchemasResponse struct {
	Cadocs []string `json:"cadocs"`
	Total  int      `json:"total"`
}

type VersionsResponse struct {
	Cadoc    string   `json:"cadoc"`
	Versions []string `json:"versions"`
	Total    int      `json:"total"`
}

type RulesListResponse struct {
	Cadocs []string `json:"cadocs"`
	Total  int      `json:"total"`
}

type ValidateRequest struct {
	CadocCode   string `json:"cadoc_code"`
	DataBase    string `json:"data_base,omitempty"`
	Xml         string `json:"xml"`
	ContentType string `json:"content_type,omitempty"`
}

type ValidateResponse struct {
	CadocCode     string            `json:"cadoc_code"`
	DataBase      string            `json:"data_base"`
	XmlHash       string            `json:"xml_hash"`
	Passed        bool              `json:"passed"`
	Errors        []ValidationError `json:"errors"`
	Warnings      []ValidationError `json:"warnings"`
	ExecutedAt    time.Time         `json:"executed_at"`
	DurationMs    int64             `json:"duration_ms"`
	DisabledRules []string          `json:"disabled_rules,omitempty"`
}

type ValidationError struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Position string `json:"position,omitempty"`
}

type SchemaVersion struct {
	Cadoc         string `json:"cadoc"`
	Version       string `json:"version"`
	EffectiveFrom string `json:"effective_from,omitempty"`
	SourceURI     string `json:"source_uri,omitempty"`
}

type ComplexityScore struct {
	Score              float64 `json:"score"`
	NumOperacoes       int     `json:"num_operacoes"`
	NumParticipantes   int     `json:"num_participantes"`
	EstimatedAPICalls  int     `json:"estimated_api_calls"`
	EstimatedTimeMs    int     `json:"estimated_time_ms"`
}

type SchemaInfo struct {
	CadocCode         string             `json:"cadoc_code"`
	LatestVersion     string             `json:"latest_version,omitempty"`
	EffectiveFrom     string             `json:"effective_from,omitempty"`
	SourceURI         string             `json:"source_uri,omitempty"`
	SupportedVersions []string           `json:"supported_versions"`
	FieldCount        int                `json:"field_count"`
	Complexity        ComplexityScore    `json:"complexity"`
}

type SchemaListResponse struct {
	Schemas []SchemaInfo `json:"schemas"`
	Total   int          `json:"total"`
}

type RuleListResponse struct {
	Cadoc  string `json:"cadoc"`
	Rules  []Rule `json:"rules"`
	Total  int    `json:"total"`
}

type Rule struct {
	Code         string   `json:"code"`
	Description  string   `json:"description"`
	Severity     string   `json:"severity"`
	Category     string   `json:"category,omitempty"`
	RequiredDocs []string `json:"required_docs,omitempty"`
}

type GenerateRequest struct {
	CadocCode     string                   `json:"cadoc_code"`
	IfID          string                   `json:"if_id"`
	Cnpj          string                   `json:"cnpj,omitempty"`
	NomeIF        string                   `json:"nome_if,omitempty"`
	VersaoLayout  string                   `json:"versao_layout,omitempty"`
	DataBase      string                   `json:"data_base,omitempty"`
	Extra         map[string]any           `json:"extra,omitempty"`
	Participantes []Participante            `json:"participantes,omitempty"`
	Operacoes     []Operacao               `json:"operacoes,omitempty"`
	Source        string                   `json:"source,omitempty"`
}

type Participante struct {
	Id     string `json:"id"`
	Tipo   string `json:"tipo,omitempty"`
	Nome   string `json:"nome,omitempty"`
	Cpf    string `json:"cpf,omitempty"`
	Cnpj   string `json:"cnpj,omitempty"`
	Rating string `json:"rating,omitempty"`
}

type Operacao struct {
	Id     string  `json:"id"`
	Tipo   string  `json:"tipo,omitempty"`
	Valor  float64 `json:"valor,omitempty"`
	Prazo  int     `json:"prazo,omitempty"`
	Taxa   float64 `json:"taxa,omitempty"`
}

type GenerateResponse struct {
	CadocCode string           `json:"cadoc_code"`
	DataBase  string           `json:"data_base"`
	Generated *GeneratedDoc    `json:"generated,omitempty"`
	Status    string           `json:"status"`
	Message   string           `json:"message,omitempty"`
}

type GeneratedDoc struct {
	XML     string         `json:"xml,omitempty"`
	XMLHash string         `json:"xml_hash,omitempty"`
	Explain map[string]any `json:"explain,omitempty"`
}

type FieldsResponse struct {
	CadocCode  string             `json:"cadoc_code"`
	Fields     []Field            `json:"fields"`
	Versions   []string           `json:"versions"`
	Complexity ComplexityScore    `json:"complexity"`
}

type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Example     any    `json:"example,omitempty"`
}

type AdaptersResponse struct {
	Adapters []AdapterInfo `json:"adapters"`
}

type AdapterInfo struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type BatchGenerateRequest struct {
	Cadocs      []GenerateRequest `json:"cadocs"`
	RunCrossDoc bool             `json:"run_crossdoc,omitempty"`
}

type BatchGenerateResponse struct {
	Results          []BatchResult    `json:"results"`
	CrossDocErrors   []CrossDocError  `json:"crossdoc_errors,omitempty"`
	CrossDocWarnings []CrossDocError  `json:"crossdoc_warnings,omitempty"`
	Passed           bool             `json:"passed"`
	Message          string           `json:"message,omitempty"`
}

type BatchResult struct {
	CadocCode string `json:"cadoc_code"`
	Status    string `json:"status"`
	XMLHash   string `json:"xml_hash,omitempty"`
	Message   string `json:"message,omitempty"`
}

type GenerationHistoryResponse struct {
	Items   []GenerationHistoryItem `json:"items"`
	Page    int                     `json:"page"`
	PerPage int                     `json:"per_page"`
	Total   int                     `json:"total"`
}

type GenerationHistoryItem struct {
	ID          string    `json:"id"`
	CadocCode   string    `json:"cadoc_code"`
	DataBase    string    `json:"data_base"`
	GeneratedAt time.Time `json:"generated_at"`
	SHA256      string    `json:"sha256,omitempty"`
	Status      string    `json:"status"`
	Passed      bool      `json:"passed"`
}

type CrossDocInput struct {
	CadocCode string `json:"cadoc_code"`
	Xml       string `json:"xml"`
	DataBase  string `json:"data_base,omitempty"`
}

type CrossDocError struct {
	Code         string   `json:"code"`
	Severity     string   `json:"severity"`
	Message      string   `json:"message"`
	InvolvedDocs []string `json:"involved_docs,omitempty"`
}

type CrossDocValidateRequest struct {
	Documents []CrossDocInput `json:"documents"`
}

type CrossDocValidateResponse struct {
	Passed        bool             `json:"passed"`
	Errors        []CrossDocError  `json:"errors"`
	Warnings      []CrossDocError  `json:"warnings"`
	RulesExecuted int              `json:"rules_executed"`
	DurationMs    int64            `json:"duration_ms"`
}

type CrossDocRulesResponse struct {
	Rules []CrossDocRule `json:"rules"`
	Total int            `json:"total"`
}

type CrossDocRule struct {
	Code         string   `json:"code"`
	Description  string   `json:"description"`
	Severity     string   `json:"severity"`
	RequiredDocs []string `json:"required_docs,omitempty"`
}

// =====================================================================
// Error handling
// =====================================================================

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type HTTPError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *HTTPError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func readError(resp *http.Response) error {
	var e apiError
	if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
		return &HTTPError{StatusCode: resp.StatusCode, Message: resp.Status}
	}
	return &HTTPError{StatusCode: resp.StatusCode, Code: e.Error, Message: e.Message}
}
