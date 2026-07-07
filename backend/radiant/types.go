// Package radiant — SDK Go para Radiant Norma API.
//
// Tipos compartilhados usados em todo o SDK.
package radiant

import "time"

// Config configura o cliente SDK.
type Config struct {
	APIKey  string // Bearer token (do dashboard radiantnorma.com)
	BaseURL string // default "https://api.radiantnorma.com"
	IFID    string // IF-ID do tenant (substitui header X-IF-ID em algumas chamadas)
}

// ValidationError representa erro de validação da API.
type ValidationError struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // E, A, I
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
}

// ValidationResult é o resultado de validação de um documento.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
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

// Envio representa um envio de documento ao BACEN.
type Envio struct {
	ID          int64     `json:"id"`
	IFID        string    `json:"if_id"`
	Cadoc       string    `json:"cadoc"`
	DataBase    string    `json:"data_base"`
	Status      string    `json:"status"`
	RulesPassed int       `json:"rules_passed"`
	RulesFailed int       `json:"rules_failed"`
	SubmittedAt time.Time `json:"submitted_at,omitempty"`
	ValidatedAt time.Time `json:"validated_at,omitempty"`
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
