// Package insights — AI Insights service (Sprint 53 v3.34.35).
//
// Oferece respostas em linguagem natural sobre o ambiente CADOC/SCR/RADAR
// do tenant, fundadas nos dados reais (audit_log, envios, events).
//
// Arquitetura:
//   - LLMService: orchestration (busca dados, compila prompt, chama LLM)
//   - MiniMaxChat / OpenAIChat: implementação da interface LLMClient
//   - Rate limiter: sliding window in-memory, 5 req/min/tenant
//
// Auth: JWT standard. Feature flag: ifs.llm_insights_enabled.
package insights

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ErrInsightsDisabled is returned when the tenant has not opted in.
var ErrInsightsDisabled = errors.New("insights: tenant has not enabled LLM insights")

// ErrRateLimited is returned when the tenant exceeds 5 req/min.
var ErrRateLimited = errors.New("insights: rate limit exceeded (5 req/min)")

// LLMClient is the interface for LLM providers.
type LLMClient interface {
	Chat(ctx context.Context, messages []Message) (string, error)
	Model() string
}

// Message is an OpenAI-compatible chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ============================================================
// MiniMax Chat — implementation using OpenAI-compatible endpoint
// ============================================================

// MiniMaxConfig configures the MiniMax client.
type MiniMaxConfig struct {
	APIKey  string
	Model   string // default "MiniMax-Text-01"
	BaseURL string // default "https://api.minimax.chat/v1"
}

// MiniMaxChat calls the MiniMax chat API via HTTP (OpenAI-compatible).
type MiniMaxChat struct {
	cfg    MiniMaxConfig
	client *http.Client
}

// NewMiniMaxChat creates a MiniMax client.
func NewMiniMaxChat(cfg MiniMaxConfig) *MiniMaxChat {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.minimax.chat/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "MiniMax-Text-01"
	}
	return &MiniMaxChat{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Chat calls the MiniMax chat API.
func (m *MiniMaxChat) Chat(ctx context.Context, messages []Message) (string, error) {
	payload := map[string]any{"model": m.cfg.Model, "messages": messages}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		m.cfg.BaseURL+"/text/chatcompletion_v2", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.cfg.APIKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("minimax %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("minimax: empty choices")
	}
	return result.Choices[0].Message.Content, nil
}

func (m *MiniMaxChat) Model() string { return m.cfg.Model }

// ============================================================
// OpenAI-compatible client
// ============================================================

// OpenAIChat calls any OpenAI-compatible endpoint.
type OpenAIChat struct {
	APIKey    string
	modelName string
	BaseURL   string
	client    *http.Client
}

// NewOpenAIChat creates an OpenAI-compatible client.
func NewOpenAIChat(apiKey, model, baseURL string) *OpenAIChat {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIChat{
		APIKey:    apiKey,
		modelName: model,
		BaseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OpenAIChat) Chat(ctx context.Context, messages []Message) (string, error) {
	payload := map[string]any{"model": o.modelName, "messages": messages}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		o.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("openai: empty choices")
	}
	return result.Choices[0].Message.Content, nil
}

func (o *OpenAIChat) Model() string { return o.modelName }

// ============================================================
// In-memory rate limiter — sliding window, 5 req/min/tenant
// ============================================================

type rateLimiter struct {
	mu        int
	windows   map[string][]time.Time
	limit     int
	windowLen time.Duration
}

func newRateLimiter(limit int, windowLen time.Duration) *rateLimiter {
	return &rateLimiter{
		windows:   make(map[string][]time.Time),
		limit:     limit,
		windowLen: windowLen,
	}
}

func (r *rateLimiter) Allow(ifID string) bool {
	r.mu++
	defer func() { r.mu-- }()

	now := time.Now()
	cutoff := now.Add(-r.windowLen)

	var valid []time.Time
	for _, t := range r.windows[ifID] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		return false
	}
	valid = append(valid, now)
	r.windows[ifID] = valid
	return true
}

// ============================================================
// LLM Service — main orchestrator
// ============================================================

// LLMConfig configures the LLM service.
type LLMConfig struct {
	LLMClient       LLMClient
	DB              *sql.DB
	Logger          *slog.Logger
	RateLimit       int
	RateWindow      time.Duration
	MaxHistoryPairs int // conversation pairs sent in prompt (default 5)
	MaxEvents       int // recent events in context (default 50)
}

// LLMService handles AI-powered insights.
type LLMService struct {
	cfg    LLMConfig
	llm    LLMClient
	db     *sql.DB
	logger *slog.Logger
	rl     *rateLimiter
}

// NewLLMService creates the LLM service.
func NewLLMService(cfg LLMConfig) *LLMService {
	if cfg.RateLimit == 0 {
		cfg.RateLimit = 5
	}
	if cfg.RateWindow == 0 {
		cfg.RateWindow = time.Minute
	}
	if cfg.MaxHistoryPairs == 0 {
		cfg.MaxHistoryPairs = 5
	}
	if cfg.MaxEvents == 0 {
		cfg.MaxEvents = 50
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &LLMService{
		cfg:    cfg,
		llm:    cfg.LLMClient,
		db:     cfg.DB,
		logger: cfg.Logger,
		rl:     newRateLimiter(cfg.RateLimit, cfg.RateWindow),
	}
}

// Ask processes a natural-language question and returns an LLM answer.
func (s *LLMService) Ask(ctx context.Context, ifID, question string) (*LLMAnswer, error) {
	if !s.rl.Allow(ifID) {
		return nil, ErrRateLimited
	}

	events, _ := s.fetchRecentEvents(ctx, ifID)
	envios, _ := s.fetchRecentEnvios(ctx, ifID)

	messages := s.buildMessages(question, events, envios)

	answer, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	return &LLMAnswer{Answer: answer, Model: s.llm.Model()}, nil
}

// LLMAnswer is the response from Ask.
type LLMAnswer struct {
	Answer string `json:"answer"`
	Model  string `json:"model"`
}

// fetchRecentEvents returns recent audit_events for the tenant.
func (s *LLMService) fetchRecentEvents(ctx context.Context, ifID string) ([]auditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT action, description, created_at
		FROM audit_events
		WHERE if_id = $1 AND created_at > $2
		ORDER BY created_at DESC
		LIMIT $3`,
		ifID, time.Now().Add(-30*24*time.Hour), s.cfg.MaxEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []auditEvent
	for rows.Next() {
		var e auditEvent
		if err := rows.Scan(&e.Action, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// fetchRecentEnvios returns recent envios with rule pass/fail summary.
func (s *LLMService) fetchRecentEnvios(ctx context.Context, ifID string) ([]envioSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT period, rules_passed, rules_failed, status
		FROM envios
		WHERE if_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 12`,
		ifID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []envioSummary
	for rows.Next() {
		var s envioSummary
		if err := rows.Scan(&s.Period, &s.RulesPassed, &s.RulesFailed, &s.Status); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// buildMessages constructs the prompt with system context + user question.
func (s *LLMService) buildMessages(question string, events []auditEvent, envios []envioSummary) []Message {
	var sb strings.Builder
	sb.WriteString("Você é um assistente especializado em compliance regulatório bancário brasileiro (BACEN).\n")
	sb.WriteString("Responda em português brasileiro. Use apenas os dados fornecidos.\n")
	sb.WriteString("Se não houver informação suficiente, diga que não sabe — não invente dados.\n")
	sb.WriteString("Seja conciso e objetivo.\n\n")

	if len(envios) > 0 {
		sb.WriteString("## Últimos Envios\n")
		for _, e := range envios {
			pct := 0
			total := e.RulesPassed + e.RulesFailed
			if total > 0 {
				pct = e.RulesPassed * 100 / total
			}
			sb.WriteString(fmt.Sprintf("- %s: %d%% regras passadas (%d/%d), status=%s\n",
				e.Period, pct, e.RulesPassed, e.RulesFailed, e.Status))
		}
	}

	if len(events) > 0 {
		sb.WriteString("\n## Eventos Recentes\n")
		for _, e := range events {
			sb.WriteString(fmt.Sprintf("- %s: %s — %s\n",
				e.CreatedAt.Format("2006-01-02"), e.Action, e.Description))
		}
	}

	return []Message{
		{Role: "system", Content: sb.String()},
		{Role: "user", Content: question},
	}
}

// auditEvent is a row from audit_events.
type auditEvent struct {
	Action      string
	Description string
	CreatedAt   time.Time
}

// envioSummary is a summary row from envios.
type envioSummary struct {
	Period      string
	RulesPassed int
	RulesFailed int
	Status      string
}
