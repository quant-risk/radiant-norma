// Sprint 53 v3.34.35: Conversation history + response cache.
//
// Package insights — AI Insights service.
//
// Adiciona:
//   - ConversationStore: persiste histórico user/assistant por tenant
//   - ResponseCache: cache de respostas por (if_id, question_hash) por 5min
//
// Conversations são automaticamente pruned após 30 dias.
package insights

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrConversationNotFound returned when a conversation has no messages.
var ErrConversationNotFound = errors.New("conversation not found")

// ConversationStore persiste mensagens de conversa por tenant no DB.
// Thread-safe via sync.Mutex (WriteMutex apenas — reads diretos no DB).
type ConversationStore struct {
	db       *sql.DB
	writeMu  sync.Mutex // serializa writes, reads são direct DB
	maxAge   time.Duration
	maxPairs int // máximo de pares user/assistant por sessão de prompt
}

// NewConversationStore cria um ConversationStore.
func NewConversationStore(db *sql.DB) *ConversationStore {
	return &ConversationStore{
		db:     db,
		maxAge: 30 * 24 * time.Hour,
	}
}

// SaveMessage persiste uma mensagem no DB e retorna o ID.
func (c *ConversationStore) SaveMessage(ctx context.Context, msg ConversationMessage) (string, error) {
	id := generateMsgID()
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO insights_conversations (id, if_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, msg.IfID, msg.Role, msg.Content, time.Now())
	if err != nil {
		return "", err
	}
	return id, nil
}

// ConversationMessage é uma mensagem de conversa armazenada no DB.
type ConversationMessage struct {
	IfID    string `json:"if_id"`
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// GetHistory retorna as últimas N mensagens para o tenant (mais recentes primeiro).
func (c *ConversationStore) GetHistory(ctx context.Context, ifID string, limit int) ([]ConversationMessage, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := c.db.QueryContext(ctx, `
		SELECT role, content FROM insights_conversations
		WHERE if_id = ? AND created_at > ?
		ORDER BY created_at DESC
		LIMIT ?
	`, ifID, time.Now().Add(-c.maxAge), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ConversationMessage
	for rows.Next() {
		var m ConversationMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, err
		}
		m.IfID = ifID
		msgs = append(msgs, m)
	}
	// Reverse to chronological order (oldest first).
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

// ClearHistory remove todas as mensagens do tenant.
func (c *ConversationStore) ClearHistory(ctx context.Context, ifID string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	_, err := c.db.ExecContext(ctx,
		"DELETE FROM insights_conversations WHERE if_id = ?", ifID)
	return err
}

// Prune remove mensagens com mais de maxAge.
func (c *ConversationStore) Prune(ctx context.Context) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	res, err := c.db.ExecContext(ctx, `
		DELETE FROM insights_conversations WHERE created_at < ?
	`, time.Now().Add(-c.maxAge))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// CountMessages retorna o número de mensagens stored for a tenant.
func (c *ConversationStore) CountMessages(ctx context.Context, ifID string) (int, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM insights_conversations WHERE if_id = ?
	`, ifID).Scan(&count)
	return count, err
}

// generateMsgID generates a unique message ID using crypto/rand.
// Uses time.Now().UnixNano() for uniqueness across calls plus
// crypto/rand bytes to avoid collision within the same nanosecond.
func generateMsgID() string {
	h := sha256.New()
	now := time.Now().UnixNano()
	b := make([]byte, 8)
	for i := range b {
		b[i] = byte(now >> (i * 8))
	}
	h.Write(b)
	// Add 8 bytes of crypto/rand to ensure uniqueness.
	randBytes := make([]byte, 8)
	cryptoRand.Read(randBytes) // ignoring error — best-effort
	h.Write(randBytes)
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// ============================================================
// ResponseCache — cache de respostas por (if_id, question_hash)
//
// Evita cobrar tokens para perguntas idênticas dentro de 5min.
// ============================================================

type cachedAnswer struct {
	answer  string
	model   string
	created time.Time
}

// ResponseCache stores LLM answers keyed by (if_id, question_hash).
type ResponseCache struct {
	mu       sync.Mutex
	entries  map[string]cachedAnswer // key: ifID|questionHash
	maxAge   time.Duration
	capacity int // max entries before eviction
}

// NewResponseCache creates a response cache.
func NewResponseCache(maxAge time.Duration, capacity int) *ResponseCache {
	if maxAge == 0 {
		maxAge = 5 * time.Minute
	}
	if capacity == 0 {
		capacity = 1000
	}
	return &ResponseCache{
		entries:  make(map[string]cachedAnswer),
		maxAge:   maxAge,
		capacity: capacity,
	}
}

// cacheKey builds a cache key from ifID and question.
func (c *ResponseCache) cacheKey(ifID, question string) string {
	h := sha256.New()
	h.Write([]byte(ifID + "|" + question))
	return hex.EncodeToString(h.Sum(nil))
}

// Get returns cached answer if fresh, otherwise ("", "", false).
func (c *ResponseCache) Get(ifID, question string) (answer, model string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(ifID, question)
	entry, found := c.entries[key]
	if !found {
		return "", "", false
	}
	if time.Since(entry.created) > c.maxAge {
		return "", "", false
	}
	return entry.answer, entry.model, true
}

// Set stores an answer in the cache.
func (c *ResponseCache) Set(ifID, question, answer, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := c.cacheKey(ifID, question)
	c.entries[key] = cachedAnswer{answer: answer, model: model, created: time.Now()}

	// Eviction: if over capacity, delete oldest 50% of entries.
	if len(c.entries) > c.capacity {
		type entryInfo struct {
			key     string
			created time.Time
		}
		entries := make([]entryInfo, 0, len(c.entries))
		for k, v := range c.entries {
			entries = append(entries, entryInfo{k, v.created})
		}
		// Sort by created ascending (oldest first).
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].created.Before(entries[j].created)
		})
		// Delete oldest half.
		half := len(entries) / 2
		for _, e := range entries[:half] {
			delete(c.entries, e.key)
		}
	}
}

// Invalidate removes all entries for a tenant.
func (c *ResponseCache) Invalidate(ifID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Clear all entries (can't selectively delete without storing ifID in key).
	c.entries = make(map[string]cachedAnswer)
}
