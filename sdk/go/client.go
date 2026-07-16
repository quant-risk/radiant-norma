// Package radiant — SDK Go para Radiant Norma API.
//
// Cliente Go oficial para a Radiant Norma API REST.
//
// Instalação:
//
//	go get github.com/fortvna/radiant-norma-go
//
// Uso:
//
//	cfg := radiant.Config{
//		APIKey:  "sk-...",
//		BaseURL: "https://api.radiantnorma.com",
//		IFID:    "00000",
//	}
//	c := radiant.New(cfg)
//
//	// Validar um documento CADOC
//	result, err := c.Cadocs.Validate(ctx, "3040", xmlData)
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
	"net/url"
	"time"
)

// Config configura o cliente SDK.
type Config struct {
	APIKey  string // Bearer token (do dashboard radiantnorma.com)
	BaseURL string // default "https://api.radiantnorma.com"
	IFID    string // IF-ID do tenant (substitui header X-IF-ID em algumas chamadas)
}

// Client é o cliente principal da API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	ifID       string

	// Sub-recursos
	Cadocs    *CadocsService
	Audit     *AuditService
	Radar     *RadarService
	Insights  *InsightsService
	Schemas   *SchemasService
	Envios    *EnviosService    // Phase 4: STA submission history + DLQ
	Webhooks  *WebhooksService  // Phase 5: outbound webhook management
	Wizard    *WizardService    // Phase 2: wizard session management
	Generate  *GenerateService   // CADOC generation
	STA       *STAService       // STA endpoints
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
	c.Envios = &EnviosService{client: c}
	c.Webhooks = &WebhooksService{client: c}
	c.Wizard = &WizardService{client: c}
	c.Generate = &GenerateService{client: c}
	c.STA = &STAService{client: c}

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

// CadocsService — validação e envio de documentos.
type CadocsService struct {
	client *Client
}

// Validate valida um documento CADOC (não envia ao BACEN).
//
//	cadoc: código do CADOC (ex: "3040", "3050")
//	xmlData: conteúdo XML do documento
//
// Resposta: ValidationResult com passed/errors/warnings.
func (s *CadocsService) Validate(ctx context.Context, cadoc string, xmlData []byte) (*ValidationResult, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/validate", map[string]any{
		"cadoc_code": cadoc,
		"xml":        string(xmlData),
	})
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

// EnviosService — Phase 4: STA submission history, DLQ management.
type EnviosService struct {
	client *Client
}

// List returns paginated STA submissions.
func (s *EnviosService) List(ctx context.Context, cadoc, status string, limit int, period string) (*EnviosResponse, error) {
	path := "/v1/envios"
	q := url.Values{}
	if cadoc != "" {
		q.Set("cadoc", cadoc)
	}
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if period != "" {
		q.Set("period", period)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	resp, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[EnviosResponse](resp)
}

// Stats returns submission KPIs.
func (s *EnviosService) Stats(ctx context.Context) (map[string]any, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/envios/stats", nil)
	if err != nil {
		return nil, err
	}
	out, err := parseResponse[map[string]any](resp)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ListDLQ returns dead letter submissions (admin only).
func (s *EnviosService) ListDLQ(ctx context.Context, limit int) (*EnviosResponse, error) {
	path := "/v1/envios/dlq"
	if limit > 0 {
		path += "?limit=" + fmt.Sprintf("%d", limit)
	}
	resp, err := s.client.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[EnviosResponse](resp)
}

// Retry retries a dead letter submission (admin only).
func (s *EnviosService) Retry(ctx context.Context, id string) error {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/envios/"+id+"/retry", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("retry failed: %s", resp.Status)
	}
	return nil
}

// WebhooksService — Phase 5: outbound webhook management.
type WebhooksService struct {
	client *Client
}

// List returns registered webhooks.
func (s *WebhooksService) List(ctx context.Context) ([]Webhook, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/webhooks", nil)
	if err != nil {
		return nil, err
	}
	out, err := parseResponse[[]Webhook](resp)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// Create registers a new webhook.
func (s *WebhooksService) Create(ctx context.Context, req *WebhookCreate) (*Webhook, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/webhooks", req)
	if err != nil {
		return nil, err
	}
	return parseResponse[Webhook](resp)
}

// Delete removes a webhook.
func (s *WebhooksService) Delete(ctx context.Context, id string) error {
	resp, err := s.client.do(ctx, http.MethodDelete, "/v1/webhooks/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete failed: %s", resp.Status)
	}
	return nil
}

// ListDeliveries returns delivery attempts for a webhook.
func (s *WebhooksService) ListDeliveries(ctx context.Context, webhookID string) ([]WebhookDelivery, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/webhooks/"+webhookID+"/deliveries", nil)
	if err != nil {
		return nil, err
	}
	out, err := parseResponse[[]WebhookDelivery](resp)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// RetryDelivery retries a failed delivery.
func (s *WebhooksService) RetryDelivery(ctx context.Context, webhookID, deliveryID string) error {
	resp, err := s.client.do(ctx, http.MethodPost,
		"/v1/webhooks/"+webhookID+"/deliveries/"+deliveryID+"/retry", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("retry failed: %s", resp.Status)
	}
	return nil
}

// WizardService — Phase 2: wizard session management.
type WizardService struct {
	client *Client
}

// Start initiates a wizard session.
func (s *WizardService) Start(ctx context.Context) (*WizardSession, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/generate/wizard/start", nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[WizardSession](resp)
}

// Get retrieves a wizard session state.
func (s *WizardService) Get(ctx context.Context, id string) (*WizardSession, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/generate/wizard/"+id, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[WizardSession](resp)
}

// Advance moves the wizard to the next step.
func (s *WizardService) Advance(ctx context.Context, id string, input map[string]any) (*WizardSession, error) {
	resp, err := s.client.do(ctx, http.MethodPut, "/v1/generate/wizard/"+id, input)
	if err != nil {
		return nil, err
	}
	return parseResponse[WizardSession](resp)
}

// GetXML returns the generated XML from a wizard session.
func (s *WizardService) GetXML(ctx context.Context, id string) (string, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/generate/wizard/"+id+"/xml", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("get xml failed: %s", resp.Status)
	}
	data, _ := io.ReadAll(resp.Body)
	return string(data), nil
}

// GenerateService — CADOC generation.
type GenerateService struct {
	client *Client
}

// Single generates a single CADOC XML.
func (s *GenerateService) Single(ctx context.Context, cadoc string, req *GenerateRequest) (*GenerateResponse, error) {
	path := "/v1/generate/" + cadoc
	resp, err := s.client.do(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	return parseResponse[GenerateResponse](resp)
}

// Batch generates multiple CADOCs in one call.
func (s *GenerateService) Batch(ctx context.Context, req *BatchGenerateRequest) (*BatchGenerateResponse, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/generate/batch", req)
	if err != nil {
		return nil, err
	}
	return parseResponse[BatchGenerateResponse](resp)
}

// STAService — STA submission endpoints.
type STAService struct {
	client *Client
}

// Submit sends a CADOC to BACEN via STA (Phase 4: supports idempotency_key + returns dedup).
func (s *STAService) Submit(ctx context.Context, cadoc, dataBase, xml string, idempotencyKey string) (*SubmissionResponse, error) {
	body := map[string]any{
		"cadoc_code": cadoc,
		"data_base":  dataBase,
		"xml":        xml,
	}
	if idempotencyKey != "" {
		body["idempotency_key"] = idempotencyKey
	}
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/sta/submit", body)
	if err != nil {
		return nil, err
	}
	return parseResponse[SubmissionResponse](resp)
}

// AvailableFiles returns BACEN-available STA files.
func (s *STAService) AvailableFiles(ctx context.Context, dataHoraInicio string) (any, error) {
	resp, err := s.client.do(ctx, http.MethodGet,
		"/v1/sta/disponiveis?dataHoraInicio="+url.QueryEscape(dataHoraInicio), nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[any](resp)
}

// UpdateStatus updates STA file status (A_REC/REC).
func (s *STAService) UpdateStatus(ctx context.Context, req map[string]any) error {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/sta/situacao", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("update status failed: %s", resp.Status)
	}
	return nil
}

// InitChunked initiates a chunked upload session.
func (s *STAService) InitChunked(ctx context.Context, req map[string]any) (any, error) {
	resp, err := s.client.do(ctx, http.MethodPost, "/v1/sta/range/init", req)
	if err != nil {
		return nil, err
	}
	return parseResponse[any](resp)
}

// UploadChunk uploads a chunk for a given protocol.
func (s *STAService) UploadChunk(ctx context.Context, protocolo string, chunk []byte) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut,
		s.client.baseURL+"/v1/sta/range/"+protocolo, bytes.NewReader(chunk))
	req.Header.Set("Content-Type", "application/octet-stream")
	if s.client.ifID != "" {
		req.Header.Set("X-IF-ID", s.client.ifID)
	}
	_, err := s.client.httpClient.Do(req)
	return err
}

// ChunkStatus returns the chunk upload status for a protocol.
func (s *STAService) ChunkStatus(ctx context.Context, protocolo string) (any, error) {
	resp, err := s.client.do(ctx, http.MethodGet, "/v1/sta/range/"+protocolo, nil)
	if err != nil {
		return nil, err
	}
	return parseResponse[any](resp)
}

// InsightsService — KPIs, heatmap, recommendations (Phase 7 aligned).
// NOTE: Ask method already defined above; these are Phase 7 additions.
var _ = []KPIEntry{} // compile-time check for Phase 7 types
