// Package radiant — SDK Go para Radiant Norma API.
//
// Cliente Go para a Radiant Norma API REST.
//
//nolint:revive,stylecheck
package radiant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client é o cliente principal da API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	ifID       string

	// Sub-recursos
	Cadocs   *CadocsService
	Audit    *AuditService
	Radar    *RadarService
	Insights *InsightsService
	Schemas  *SchemasService
}

// New cria um novo cliente SDK.
func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.radiantnorma.com"
	}

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		ifID:       cfg.IFID,
	}

	c.Cadocs = &CadocsService{client: c}
	c.Audit = &AuditService{client: c}
	c.Radar = &RadarService{client: c}
	c.Insights = &InsightsService{client: c}
	c.Schemas = &SchemasService{client: c}

	return c
}

// do executa request HTTP com auth e error handling padrão.
func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.ifID != "" {
		req.Header.Set("X-IF-ID", c.ifID)
	}

	return c.httpClient.Do(req)
}

// parseResponse lê response body em JSON, retorna erro se status != 2xx.
func parseResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, resp.Status)
		}
		return nil, fmt.Errorf("[%s] %s", errResp.Error, errResp.Message)
	}

	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// parseResponseRaw lê response body sem parsear JSON (para bytes).
func parseResponseRaw(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// CadocsService — validação e envio de documentos.
type CadocsService struct {
	client *Client
}

// Validate valida um documento CADOC (não envia ao BACEN).
//
//	xmlData: conteúdo XML do documento
//
// Resposta: ValidationResult com errors/warnings.
func (s *CadocsService) Validate(ctx context.Context, cadoc string, xmlData []byte) (*ValidationResult, error) {
	// POST /v1/validate/{cadoc}
	resp, err := s.client.do(ctx, http.MethodPost,
		fmt.Sprintf("/v1/validate/%s", cadoc), map[string]any{"xml": string(xmlData)})
	if err != nil {
		return nil, err
	}
	return parseResponse[ValidationResult](resp)
}

// ValidateCrossDoc valida consistência entre múltiplos documentos.
//
//	docs: map de cadoc_code → xml_bytes
//
// Resposta: CrossDocResult com resultados por regra.
func (s *CadocsService) ValidateCrossDoc(ctx context.Context, docs map[string][]byte) (*CrossDocResult, error) {
	// POST /v1/crossdoc/validate
	rawDocs := make(map[string]string)
	for k, v := range docs {
		rawDocs[k] = string(v)
	}
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/crossdoc/validate", map[string]any{"cadocs": rawDocs})
	if err != nil {
		return nil, err
	}
	return parseResponse[CrossDocResult](resp)
}

// AuditService — regras de auditoria.
type AuditService struct {
	client *Client
}

// ListRules retorna todas as regras de auditoria para um CADOC.
func (s *AuditService) ListRules(ctx context.Context, cadoc string) ([]RuleDef, error) {
	resp, err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/v1/rules/%s", cadoc), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Rules []RuleDef `json:"rules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return out.Rules, nil
}

// RuleDef define uma regra de auditoria.
type RuleDef struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // E, A, I
	Message  string `json:"message"`
}

// RadarService — detecção de mudanças de layout.
type RadarService struct {
	client *Client
}

// Scan executa um radar scan para um CADOC.
func (s *RadarService) Scan(ctx context.Context, cadoc string) (*ScanResult, error) {
	resp, err := s.client.do(ctx, http.MethodPost,
		fmt.Sprintf("/v1/radar/scan?cadoc=%s", cadoc), nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[ScanResult](resp)
}

// InsightsService — insights LLM.
type InsightsService struct {
	client *Client
}

// Ask faz uma pergunta em linguagem natural sobre o ambiente do tenant.
func (s *InsightsService) Ask(ctx context.Context, question string) (*LLMAnswer, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/insights/ask",
		map[string]string{"question": question})
	if err != nil {
		return nil, err
	}
	return parseResponse[LLMAnswer](resp)
}

// SchemasService — schema registry.
type SchemasService struct {
	client *Client
}

// ListVersions retorna histórico de versões de um CADOC.
func (s *SchemasService) ListVersions(ctx context.Context, cadoc string) ([]SchemaVersion, error) {
	resp, err := s.client.do(ctx, http.MethodGet,
		fmt.Sprintf("/v1/schemas/%s/versions", cadoc), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Versions []SchemaVersion `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return out.Versions, nil
}

// GetChangelog retorna timeline de changelogs de um CADOC.
func (s *SchemasService) GetChangelog(ctx context.Context, cadoc string) ([]SchemaVersion, error) {
	resp, err := s.client.do(ctx, http.MethodGet,
		fmt.Sprintf("/v1/schemas/%s/changelog", cadoc), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Entries []SchemaVersion `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return out.Entries, nil
}
