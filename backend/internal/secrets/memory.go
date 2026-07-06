package secrets

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// MemoryManager é implementação in-memory para tests e dev local.
//
// Decisão (Sprint 28 RESEARCH D-2): EnvManager é fallback oficial, MemoryManager
// é para tests. Razão: tests não querem mockar env vars; MemoryManager permite
// fixture determinística.
type MemoryManager struct {
	mu      sync.RWMutex
	store   map[string]*Secret
	logger  *slogLogger
	logSink func(string, ...any)
}

// NewMemoryManager cria um MemoryManager vazio.
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		store: make(map[string]*Secret),
	}
}

// Set insere/sobrescreve um secret (helper para tests).
func (m *MemoryManager) Set(name, value, versionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[name] = &Secret{
		Name:      name,
		Value:     value,
		VersionID: versionID,
		CreatedAt: time.Now(),
	}
}

func (m *MemoryManager) Get(ctx context.Context, name string) (*Secret, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.store[name]
	if !ok {
		return nil, &NotFoundError{Name: name, Backend: m.Backend()}
	}
	// Return copy to prevent caller mutation of internal state
	return &Secret{
		Name:      s.Name,
		Value:     s.Value,
		VersionID: s.VersionID,
		CreatedAt: s.CreatedAt,
	}, nil
}

func (m *MemoryManager) Put(ctx context.Context, name, value string) (*Secret, error) {
	if name == "" {
		return nil, &ValidationError{Name: name, Reason: "name cannot be empty"}
	}
	if value == "" {
		return nil, &ValidationError{Name: name, Reason: "value cannot be empty"}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Auto-increment version for test determinism
	existing := m.store[name]
	versionID := "v1"
	if existing != nil && existing.VersionID != "" {
		// Strip "v" prefix and increment
		var n int
		if _, err := fmt.Sscanf(existing.VersionID, "v%d", &n); err == nil {
			versionID = fmt.Sprintf("v%d", n+1)
		}
	}

	now := time.Now()
	s := &Secret{
		Name:      name,
		Value:     value,
		VersionID: versionID,
		CreatedAt: now,
	}
	m.store[name] = s

	return &Secret{
		Name:      s.Name,
		Value:     s.Value,
		VersionID: s.VersionID,
		CreatedAt: s.CreatedAt,
	}, nil
}

func (m *MemoryManager) Delete(ctx context.Context, name string) error {
	if name == "" {
		return &ValidationError{Name: name, Reason: "name cannot be empty"}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.store[name]; !ok {
		return &NotFoundError{Name: name, Backend: m.Backend()}
	}
	delete(m.store, name)
	return nil
}

func (m *MemoryManager) Backend() string { return BackendMemory }

// slogLogger is an interface to allow optional slog integration without import cycle.
// Empty for now; can be wired in later sprint if needed.
type slogLogger interface{}
var _ slogLogger = (*string)(nil)

// Keep "strings" import used (helper for future string ops on names)
var _ = strings.ToUpper