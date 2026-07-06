package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// EnvManager é implementação via environment variables.
//
// Naming convention:
//
//	Secret name: "bacen/senha/123450001.fulano"
//	Env var name: "RADIANT_SECRET_BACEN_SENHA_123450001_FULANO"
//	                 └── prefix ──┘ └──────── uppercase + / → _ ────────┘
//
// Segurança:
//
//	- Nenhum valor de env var é logado.
//	- Apenas o nome do secret aparece em logs/métricas.
//	- Put chama os.Setenv que afeta o PROCESS inteiro — callers devem
//	  assumir que outras partes do código podem ver o valor.
//	  Em produção, use AWSManager. EnvManager é dev/test fallback.
type EnvManager struct {
	prefix string
	mu     sync.Mutex
}

// NewEnvManager cria EnvManager com prefix "RADIANT_SECRET_".
func NewEnvManager() *EnvManager {
	return &EnvManager{prefix: "RADIANT_SECRET_"}
}

// NewEnvManagerWithPrefix cria EnvManager com prefix customizado.
// Útil para tests que querem isolar env vars.
func NewEnvManagerWithPrefix(prefix string) *EnvManager {
	return &EnvManager{prefix: prefix}
}

// envName converte nome de secret → nome de env var.
//
// Substitui caracteres não-alfanuméricos (exceto `_`) por `_` para garantir
// compatibilidade com shells. Uppercase para consistência com convenção Unix.
func (m *EnvManager) envName(name string) string {
	var sb strings.Builder
	sb.Grow(len(m.prefix) + len(name))
	sb.WriteString(m.prefix)
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			sb.WriteRune(r)
		case r >= 'a' && r <= 'z':
			sb.WriteRune(r - 'a' + 'A') // uppercase
		default:
			sb.WriteRune('_') // /, ., -, etc
		}
	}
	return sb.String()
}

func (m *EnvManager) Get(ctx context.Context, name string) (*Secret, error) {
	if name == "" {
		return nil, &ValidationError{Name: name, Reason: "name cannot be empty"}
	}

	val := os.Getenv(m.envName(name))
	if val == "" {
		return nil, &NotFoundError{Name: name, Backend: m.Backend()}
	}

	return &Secret{
		Name:      name,
		Value:     val,
		VersionID: "env", // static — env vars don't version
		CreatedAt: time.Now(),
	}, nil
}

func (m *EnvManager) Put(ctx context.Context, name, value string) (*Secret, error) {
	if name == "" {
		return nil, &ValidationError{Name: name, Reason: "name cannot be empty"}
	}
	if value == "" {
		return nil, &ValidationError{Name: name, Reason: "value cannot be empty"}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.Setenv(m.envName(name), value); err != nil {
		return nil, fmt.Errorf("os.Setenv failed: %w", err)
	}

	return &Secret{
		Name:      name,
		Value:     value,
		VersionID: "env",
		CreatedAt: time.Now(),
	}, nil
}

func (m *EnvManager) Delete(ctx context.Context, name string) error {
	if name == "" {
		return &ValidationError{Name: name, Reason: "name cannot be empty"}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	envName := m.envName(name)
	if os.Getenv(envName) == "" {
		return &NotFoundError{Name: name, Backend: m.Backend()}
	}

	if err := os.Unsetenv(envName); err != nil {
		return fmt.Errorf("os.Unsetenv failed: %w", err)
	}
	return nil
}

func (m *EnvManager) Backend() string { return BackendEnv }