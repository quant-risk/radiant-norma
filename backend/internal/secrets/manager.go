// Package secrets implementa abstração para secret managers.
//
// Suporta múltiplos backends via interface Manager:
//   - AWS Secrets Manager (default prod)
//   - Env vars (fallback dev/test)
//   - In-memory (tests)
//
// Decisão arquitetural (Sprint 28 / v3.23.0 / Plano Ouro §3.2):
//
//	Antes (Sprint 23-27): senha Sisbacen ficava em env var. Vetores:
//	  - ps aux leak
//	  - log aggregator leak
//	  - rotação manual (caller precisa aws secretsmanager update-secret)
//	Depois (Sprint 28+): interface Manager abstrai backend, cmd/senhaws-rotate
//	  tem subcomando `apply` que rotaciona BACEN + atualiza secret manager
//	  em uma única operação atômica-ish (audit log registra ambos).
//
// Naming convention:
//
//	Secret names: "bacen/senha/{user}", "bacen/sta/{if_id}/user"
//	Env vars: "RADIANT_SECRET_" + uppercase(name with / → _)
//	Exemplo: "bacen/senha/123450001.fulano" → "RADIANT_SECRET_BACEN_SENHA_123450001_FULANO"
//
// Segurança:
//
//   - Nenhum valor de secret é logado em NENHUM nível (incluindo Debug).
//     Apenas o nome do secret aparece em logs.
//   - Erros tipados (NotFoundError, AccessDeniedError, ValidationError)
//     permitem caller classificar sem expor values.
//   - AWS auth via IAM role (zero credenciais em código).
package secrets

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Secret representa um valor armazenado num backend.
type Secret struct {
	Name      string    // identificador único do secret
	Value     string    // valor (sensitive — nunca logar)
	VersionID string    // versão (AWS-specific; vazio em outros backends)
	CreatedAt time.Time // quando foi criado/updated
}

// Manager é a interface comum a todos os backends.
//
// Decisão (Sprint 28 RESEARCH D-1): interface mínima (3 métodos).
// Adicionar capability depois (Rotate, List, etc) só se virar requisito.
type Manager interface {
	// Get retorna o secret atual. Retorna *NotFoundError se não existir.
	Get(ctx context.Context, name string) (*Secret, error)

	// Put cria ou atualiza um secret. Retorna metadata do write.
	Put(ctx context.Context, name, value string) (*Secret, error)

	// Delete remove um secret. Retorna *NotFoundError se já não existir.
	Delete(ctx context.Context, name string) error

	// Backend retorna identificador curto do tipo de manager.
	// Útil para logs, métricas e healthchecks.
	// Valores: "aws" | "env" | "memory".
	Backend() string
}

// Backend type constants — exported para callers validarem.
const (
	BackendAWS    = "aws"
	BackendEnv    = "env"
	BackendMemory = "memory"
)

// NewManagerFromEnv constrói o Manager baseado em env var RADIANT_SECRETS_BACKEND.
//
// Valores aceitos:
//
//	"" (default) — EnvManager (back-compat com Sprint 23-27)
//	"env"        — EnvManager (idêntico ao default)
//	"aws"        — AWSManager (Sprint 28+, default prod)
//	"memory"     — MemoryManager (tests, dev local sem env)
//
// Falha fast se backend inválido ou se config do backend escolhido falhar
// (ex: AWS sem IAM role). Caller decide retry policy.
func NewManagerFromEnv(ctx context.Context, logger *slog.Logger) (Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	backend := strings.ToLower(strings.TrimSpace(os.Getenv("RADIANT_SECRETS_BACKEND")))

	switch backend {
	case "", BackendEnv:
		logger.Info("secrets manager: env (fallback dev/test — não recomendado em prod)")
		return NewEnvManager(), nil

	case BackendMemory:
		logger.Info("secrets manager: memory (test/dev — secrets não persistem entre restarts)")
		return NewMemoryManager(), nil

	case BackendAWS:
		mgr, err := NewAWSManagerFromEnv(ctx, logger)
		if err != nil {
			return nil, fmt.Errorf("AWS secrets manager init failed: %w", err)
		}
		logger.Info("secrets manager: AWS Secrets Manager ativo")
		return mgr, nil

	default:
		return nil, fmt.Errorf("RADIANT_SECRETS_BACKEND=%q inválido (aceito: env|aws|memory)", backend)
	}
}
