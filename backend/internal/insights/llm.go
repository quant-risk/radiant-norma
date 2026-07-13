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
	"sync"
	"time"
)

// ErrInsightsDisabled is returned when the tenant has not opted in.
var ErrInsightsDisabled = errors.New("insights: tenant has not enabled LLM insights")

// ErrRateLimited is returned when the tenant exceeds 5 req/min.
var ErrRateLimited = errors.New("insights: rate limit exceeded (5 req/min)")

// sanitizeForPrompt removes newlines and truncates to 200 chars to prevent
// prompt injection from attacker-controlled DB fields (audit_events, envios).
func sanitizeForPrompt(s string) string {
	const maxLen = 200
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	// Replace newlines/tabs to keep prompt structure intact.
	s = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(s)
	return s
}

// LLMClient is the interface for LLM providers.
type LLMClient interface {
	Chat(ctx context.Context, messages []Message) (string, error)
	// StreamChat calls the LLM and sends response chunks via the returned channel.
	// The channel is closed when streaming finishes or an error occurs.
	// If the context is cancelled, the channel is closed with the context error.
	StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
	Model() string
}

// StreamChunk is a chunk of streamed LLM response.
type StreamChunk struct {
	Text  string
	Done  bool // true when this is the final chunk
	Error error // non-nil if streaming failed
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

// StreamChat calls the LLM and streams the response in chunks via a channel.
// Since the standard MiniMax /text/chatcompletion_v2 endpoint is not a streaming
// endpoint, this implementation calls the regular Chat and then streams the
// response in word-sized chunks with small delays to simulate progressive output.
// This preserves the SSE streaming architecture; upgrading to a true streaming
// endpoint (e.g. MiniMax streaming API) requires only swapping the HTTP call.
func (m *MiniMaxChat) StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)

	// Run the LLM call and streaming in a goroutine so it doesn't block.
	go func() {
		defer close(ch)

		// 1. Call LLM to get the complete response.
		answer, err := m.Chat(ctx, messages)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		// 2. Stream answer in small chunks (simulated progressive output).
		// Split on spaces to get "word" granularity with natural pauses.
		words := strings.Split(answer, " ")
		for i, word := range words {
			chunk := word
			if i < len(words)-1 {
				chunk += " " // re-add space after all but last word
			}
			select {
			case ch <- StreamChunk{Text: chunk}:
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err()}
				return
			}
			// Small delay between chunks to simulate typing.
			if i < len(words)-1 {
				time.Sleep(20 * time.Millisecond)
			}
		}
		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

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

// StreamChat calls the LLM and streams the response in chunks via a channel.
// Same pattern as MiniMaxChat — non-streaming API call with simulated progressive output.
// Swap the HTTP call to a streaming endpoint (e.g. OpenAI /chat/completions with
// stream=true) to get true token streaming without other changes.
func (o *OpenAIChat) StreamChat(ctx context.Context, messages []Message) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)

	go func() {
		defer close(ch)

		answer, err := o.Chat(ctx, messages)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}

		words := strings.Split(answer, " ")
		for i, word := range words {
			chunk := word
			if i < len(words)-1 {
				chunk += " "
			}
			select {
			case ch <- StreamChunk{Text: chunk}:
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err()}
				return
			}
			if i < len(words)-1 {
				time.Sleep(20 * time.Millisecond)
			}
		}
		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

// ============================================================
// In-memory rate limiter — sliding window, 5 req/min/tenant
// ============================================================

type rateLimiter struct {
	mu        sync.Mutex
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

// Allow returns true if the request is allowed under the rate limit.
// Thread-safe using sync.Mutex.
func (r *rateLimiter) Allow(ifID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.windowLen)

	// Filter to only timestamps within the sliding window
	var valid []time.Time
	for _, t := range r.windows[ifID] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	// At limit → reject
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
	ConvStore       *ConversationStore // nil → histórico desabilitado
	RespCache       *ResponseCache     // nil → cache desabilitado
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
// If ConvStore is configured, conversation history is included in the prompt.
// If RespCache is configured, duplicate questions within 5min return cached answer.
// Thread-safe.
func (s *LLMService) Ask(ctx context.Context, ifID, question string) (*LLMAnswer, error) {
	if !s.rl.Allow(ifID) {
		return nil, ErrRateLimited
	}

	// 1. Check response cache.
	if s.cfg.RespCache != nil {
		if answer, model, ok := s.cfg.RespCache.Get(ifID, question); ok {
			return &LLMAnswer{Answer: answer, Model: model, Cached: true}, nil
		}
	}

	// 2. Build messages with conversation history + context.
	var history []Message
	if s.cfg.ConvStore != nil {
		// Fetch last MaxHistoryPairs pairs (user + assistant = 1 pair).
		n := s.cfg.MaxHistoryPairs * 2
		convMsgs, _ := s.cfg.ConvStore.GetHistory(ctx, ifID, n)
		// Convert ConversationMessage → Message.
		for _, m := range convMsgs {
			history = append(history, Message{Role: m.Role, Content: m.Content})
		}
	}

	events, _ := s.fetchRecentEvents(ctx, ifID)
	envios, _ := s.fetchRecentEnvios(ctx, ifID)

	messages := s.buildMessages(question, history, events, envios)

	// 3. Call LLM.
	answer, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	// 4. Persist conversation messages.
	if s.cfg.ConvStore != nil {
		// Save user message.
		um := ConversationMessage{IfID: ifID, Role: "user", Content: question}
		_, _ = s.cfg.ConvStore.SaveMessage(ctx, um)
		// Save assistant answer.
		am := ConversationMessage{IfID: ifID, Role: "assistant", Content: answer}
		_, _ = s.cfg.ConvStore.SaveMessage(ctx, am)
	}

	// 5. Cache the answer.
	if s.cfg.RespCache != nil {
		s.cfg.RespCache.Set(ifID, question, answer, s.llm.Model())
	}

	return &LLMAnswer{Answer: answer, Model: s.llm.Model()}, nil
}

// StreamAsk processes a natural-language question and streams the LLM response
// chunks via a channel. Conversation history is included in the prompt if
// ConvStore is configured. Rate limit is checked before streaming begins.
// The caller receives chunks via the returned channel and should range over it.
// After streaming finishes, the conversation messages are persisted.
func (s *LLMService) StreamAsk(ctx context.Context, ifID, question string) (<-chan LLMAnswerChunk, error) {
	if !s.rl.Allow(ifID) {
		return nil, ErrRateLimited
	}

	// Build messages with conversation history + context.
	var history []Message
	if s.cfg.ConvStore != nil {
		n := s.cfg.MaxHistoryPairs * 2
		convMsgs, _ := s.cfg.ConvStore.GetHistory(ctx, ifID, n)
		for _, m := range convMsgs {
			history = append(history, Message{Role: m.Role, Content: m.Content})
		}
	}

	events, _ := s.fetchRecentEvents(ctx, ifID)
	envios, _ := s.fetchRecentEnvios(ctx, ifID)
	messages := s.buildMessages(question, history, events, envios)

	// Create channel for streaming results.
	ch := make(chan LLMAnswerChunk, 1)

	go func() {
		defer close(ch)

		// Stream LLM response.
		streamCh, err := s.llm.StreamChat(ctx, messages)
		if err != nil {
			ch <- LLMAnswerChunk{Error: fmt.Errorf("llm stream: %w", err)}
			return
		}

		var fullAnswer strings.Builder
		modelName := s.llm.Model()

		for chunk := range streamCh {
			if chunk.Error != nil {
				ch <- LLMAnswerChunk{Error: chunk.Error}
				return
			}
			fullAnswer.WriteString(chunk.Text)
			ch <- LLMAnswerChunk{
				Text:  chunk.Text,
				Model: modelName,
				Done:  chunk.Done,
			}
			if chunk.Done {
				break
			}
		}

		// Persist conversation messages after streaming completes.
		if s.cfg.ConvStore != nil {
			um := ConversationMessage{IfID: ifID, Role: "user", Content: question}
			_, _ = s.cfg.ConvStore.SaveMessage(ctx, um)
			am := ConversationMessage{IfID: ifID, Role: "assistant", Content: fullAnswer.String()}
			_, _ = s.cfg.ConvStore.SaveMessage(ctx, am)
		}
	}()

	return ch, nil
}

// LLMAnswerChunk is a chunk of streamed LLM answer.
type LLMAnswerChunk struct {
	Text  string // incremental text chunk
	Model string // model name (set on first chunk)
	Error error  // non-nil on stream error
	Done  bool   // true when streaming is complete
}

// GetHistory returns the conversation history for a tenant.
func (s *LLMService) GetHistory(ctx context.Context, ifID string, limit int) ([]ConversationMessage, error) {
	if s.cfg.ConvStore == nil {
		return nil, nil
	}
	return s.cfg.ConvStore.GetHistory(ctx, ifID, limit)
}

// ClearHistory removes all conversation history for a tenant.
func (s *LLMService) ClearHistory(ctx context.Context, ifID string) error {
	if s.cfg.ConvStore == nil {
		return nil
	}
	return s.cfg.ConvStore.ClearHistory(ctx, ifID)
}


// LLMAnswer is the response from Ask.
type LLMAnswer struct {
	Answer string `json:"answer"`
	Model  string `json:"model"`
	Cached bool   `json:"cached,omitempty"` // true if served from cache
}

// fetchRecentEvents returns recent audit_events for the tenant.
func (s *LLMService) fetchRecentEvents(ctx context.Context, ifID string) ([]auditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT action, description, created_at
		FROM audit_events
		WHERE if_id = ? AND created_at > ?
		ORDER BY created_at DESC
		LIMIT ?`,
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
		WHERE if_id = ? AND deleted_at IS NULL
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

// buildMessages constructs the prompt with system context + optional history + user question.
func (s *LLMService) buildMessages(question string, history []Message, events []auditEvent, envios []envioSummary) []Message {
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
			period := sanitizeForPrompt(e.Period)
			status := sanitizeForPrompt(e.Status)
			sb.WriteString(fmt.Sprintf("- %s: %d%% regras passadas (%d/%d), status=%s\n",
				period, pct, e.RulesPassed, e.RulesFailed, status))
		}
	}

	if len(events) > 0 {
		sb.WriteString("\n## Eventos Recentes\n")
		for _, e := range events {
			action := sanitizeForPrompt(e.Action)
			desc := sanitizeForPrompt(e.Description)
			sb.WriteString(fmt.Sprintf("- %s: %s — %s\n",
				e.CreatedAt.Format("2006-01-02"), action, desc))
		}
	}

	systemContent := sb.String()

	// Build final message list: system → history (if any) → user.
	msgs := make([]Message, 0, 2+len(history))
	msgs = append(msgs, Message{Role: "system", Content: systemContent})
	msgs = append(msgs, history...)
	msgs = append(msgs, Message{Role: "user", Content: question})
	return msgs
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
