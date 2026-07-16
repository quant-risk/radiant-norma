// Package radiant — SDK Go para Radiant Norma API.
//
// Tipos compartilhados usados em todo o SDK.
package radiant

import "time"

// ValidationError representa erro de validação da API.
type ValidationError struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // E, A, I
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// ValidationResult é o resultado de validação de um documento.
type ValidationResult struct {
	Passed     bool              `json:"passed"`
	DataBase   string            `json:"data_base"`
	XMLHash    string            `json:"xml_hash"`
	Errors     []ValidationError `json:"errors"`
	Warnings   []ValidationError `json:"warnings"`
	DurationMs int64             `json:"duration_ms"`
}

// ErrorResponse é a resposta de erro padrão da API.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// CrossDocResult é o resultado de validação cross-document.
type CrossDocResult struct {
	Passed     bool              `json:"passed"`
	Errors     []ValidationError `json:"errors"`
	Warnings   []ValidationError `json:"warnings"`
	RulesRun   []string          `json:"rules_run"`
	RulesSkip  []string          `json:"rules_skipped"`
	DurationMs int64             `json:"duration_ms"`
}

// RuleDef define uma regra de auditoria.
type RuleDef struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // E, A, I
	Message  string `json:"message"`
}

// Envio representa um envio de documento ao BACEN.
// Phase 4: added Period, CadocCode, ProtocolSTA, ErrorCode, ErrorMessage, DurationMs, Attempts.
type Envio struct {
	ID           int64     `json:"id"`
	IFID         string    `json:"if_id"`
	CadocCode    string    `json:"cadoc_code"`
	Period       string    `json:"period"`
	DataBase     string    `json:"data_base"`
	Status       string    `json:"status"` // pending, accepted, rejected, error, dead_letter
	RulesPassed  int       `json:"rules_passed"`
	RulesFailed  int       `json:"rules_failed"`
	DurationMs   int64     `json:"duration_ms"`
	ProtocolSTA  string    `json:"protocol_sta,omitempty"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	SentAt       time.Time `json:"sent_at,omitempty"`
	ConfirmedAt  time.Time `json:"confirmed_at,omitempty"`
	Attempts     int       `json:"attempts"` // Phase 4: retry count
}

// ScanResult é o resultado de um radar scan.
type ScanResult struct {
	ID        string    `json:"id"`
	IFID      string    `json:"if_id"`
	Cadoc     string    `json:"cadoc"`
	Status    string    `json:"status"` // clean, changed, error
	Changes   []Change  `json:"changes,omitempty"`
	ScannedAt time.Time `json:"scanned_at"`
}

// Change representa uma mudança detectada no layout.
type Change struct {
	Tag      string `json:"tag"`
	Attr     string `json:"attr,omitempty"`
	Kind     string `json:"kind"` // added, removed, modified
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// SchemaVersion representa uma versão do schema de um CADOC.
type SchemaVersion struct {
	ID            int64     `json:"id"`
	CadocCode     string    `json:"cadoc_code"`
	EffectiveFrom string    `json:"effective_from"`
	SourceURI     string    `json:"source_uri,omitempty"`
	Changelog     string    `json:"changelog,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LLMAnswer é a resposta do insights LLM.
type LLMAnswer struct {
	Answer string `json:"answer"`
	Model  string `json:"model"`
}

// WizardSession representa uma sessão do wizard de geração.
type WizardSession struct {
	ID           string   `json:"id"`
	Step         string   `json:"step"`
	CadocCode    string   `json:"cadoc_code"`
	SourceType   string   `json:"source_type"`
	GeneratedXML string   `json:"generated_xml"`
	Errors       []string `json:"errors"`
}

// EnviosResponse é a resposta de listagem de envios.
type EnviosResponse struct {
	Envios []Envio `json:"envios"`
	Total  int     `json:"total"`
}

// GenerateRequest é o request para geração de um CADOC.
type GenerateRequest struct {
	CNPJ        string            `json:"cnpj"`
	NomeIF      string            `json:"nome_if"`
	DataBase    string            `json:"data_base"`
	VersaoLayout string           `json:"versao_layout,omitempty"`
	Campos      map[string]any    `json:"campos,omitempty"`
}

// GenerateResponse é a resposta de geração de um CADOC.
type GenerateResponse struct {
	XML          string `json:"xml"`
	XMLHash      string `json:"xml_hash"`
	CadocCode    string `json:"cadoc_code"`
	DataBase     string `json:"data_base"`
	VersaoLayout string `json:"versao_layout"`
	DurationMs   int64  `json:"duration_ms"`
}

// BatchGenerateRequest é o request para geração em batch.
type BatchGenerateRequest struct {
	Requests []GenerateRequest `json:"requests"`
}

// BatchGenerateResult é o resultado de um item no batch.
type BatchGenerateResult struct {
	CadocCode string            `json:"cadoc_code"`
	Status    string            `json:"status"` // ok, error
	XML       string            `json:"xml,omitempty"`
	Errors    []ValidationError `json:"errors,omitempty"`
}

// BatchGenerateResponse é a resposta de geração em batch.
type BatchGenerateResponse struct {
	Results []BatchGenerateResult `json:"results"`
}

// SubmissionResponse é a resposta de envio STA (Phase 4: added dedup field).
type SubmissionResponse struct {
	ProtocolSTA string           `json:"protocol_sta"`
	Accepted    bool             `json:"accepted"`
	Rejection   *Rejection       `json:"rejection,omitempty"`
	EnvioID     string           `json:"envio_id"`
	Dedup       string           `json:"dedup,omitempty"` // idempotency_key, xml_hash (Phase 4)
	Warning     string           `json:"warning,omitempty"`
}

// Rejection representa rejeição de um envio.
type Rejection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Webhook representa um webhook registrado (Phase 5).
type Webhook struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Events      []string  `json:"events"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

// WebhookCreate é o request para criar um webhook.
type WebhookCreate struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description,omitempty"`
	Secret      string   `json:"secret,omitempty"` // HMAC-SHA256 signing secret
}

// WebhookDelivery representa uma tentativa de entrega de webhook.
type WebhookDelivery struct {
	ID          string    `json:"id"`
	WebhookID   string    `json:"webhook_id"`
	Event       string    `json:"event"`
	Payload     string    `json:"payload"`
	Status      string    `json:"status"` // pending, success, failed, retrying
	HTTPStatus  int       `json:"http_status"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"created_at"`
	DeliveredAt time.Time `json:"delivered_at,omitempty"`
}

// KPIEntry representa um entry nos KPIs temporais.
type KPIEntry struct {
	Period     string  `json:"period"`
	Total      int     `json:"total"`
	Accepted   int     `json:"accepted"`
	Rejected   int     `json:"rejected"`
	Error      int     `json:"error"`
	DeadLetter int     `json:"dead_letter"`
	AvgLatMs   float64 `json:"avg_lat_ms"`
}

// HeatmapEntry representa uma célula do heatmap.
type HeatmapEntry struct {
	Cadoc   string `json:"cadoc"`
	Weekday int    `json:"weekday"` // 0=Mon, 6=Sun
	Count   int    `json:"count"`
	Errors  int    `json:"errors"`
}

// Recommendation representa uma recomendação heurística.
type Recommendation struct {
	Rule      string `json:"rule"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // E, A, I
	Impact    string `json:"impact,omitempty"`
}

// TopFailingRule representa uma regra que mais falha.
type TopFailingRule struct {
	Rule     string  `json:"rule"`
	Count    int     `json:"count"`
	Rate     float64 `json:"rate"`
	Severity string  `json:"severity"`
}
