# SPRINT 28 — RESEARCH: VaultIntegration (AWS Secrets Manager para rotação Sisbacen)

> **Sprint:** 28 (v3.23.0)
> **Quando:** 2026-07-06
> **Status:** 🔄 Draft → ✅ Pronto pra implementação
> **Plano:** [MASTER_PLAN.md](../../MASTER_PLAN.md) §3.2 Épico B (Norma Connect)
> **Trigger:** Plano Ouro aprovado (Sprint anterior v3.22.0) + gap operacional da Sprint 27

---

## TL;DR

Hoje a senha Sisbacen fica em **env var** (`SENHAWS_PASSWORD`). Isso é o vetor #1 de secret disclosure em qualquer auditoria SOC 2 / LGPD:

1. Aparece em `ps aux` se IF passar via flag CLI.
2. Aparece em logs agregados (Datadog/Loki/Better Stack).
3. Aparece em `.env` files commitados por engano.
4. Rotação é manual (`senhaws-rotate rotate > /tmp/newpass.txt && aws secretsmanager ...`).

**Decisão:** abstrair via interface `secrets.Manager`, com 2 implementações:
- `AWSSecretsManager` (prod) — usa AWS SDK v2, IAM role-based auth.
- `EnvManager` (dev/test) — fallback seguro para env vars (não loga valores).

`senhaws-rotate` ganha subcomando `apply` que atualiza AWS direto após rotação.

---

## Problema

### Estado atual (Sprint 23 + 27)

```
$ senhaws-rotate rotate --base-url=https://www9.bcb.gov.br/senhaws \
                       --user=123450001.fulano \
                       --password=$SENHAWS_PASSWORD
# Exit 0, senha alterada no BACEN.
# Mas: caller precisa manualmente:
aws secretsmanager update-secret --secret-id bacen/senha --secret-string file:///tmp/newpass.txt
rm /tmp/newpass.txt
```

### Vetores de risco

| Vetor | Severidade | Cenário |
|---|---|---|
| **ps aux leak** | Alto | Dev passa `--password=...` em flag, aparece em `ps aux` |
| **Log aggregator leak** | Crítico | Senha em env var é serializada em logs (stack traces) |
| **`.env` commit** | Alto | Dev commita `.env` sem querer (mesmo com `.gitignore`) |
| **Rotação manual** | Médio | IF esquece de atualizar secret manager, próxima call STA quebra |
| **Audit trail fraco** | Médio | Não há log centralizado de quem rotacionou quando |

### Compliance gap

- **SOC 2 CC6.1:** "Logical access controls" requer que secrets estejam em vault gerenciado.
- **LGPD Art. 46:** "Medidas de segurança adequadas" inclui controle de acesso a credenciais.
- **BACEN Res. 4.658:** Política de segurança cibernética exige vault pra credenciais.

---

## Pesquisa

### Alternativas consideradas

| Opção | Prós | Contras | Decisão |
|---|---|---|---|
| **AWS Secrets Manager** | Managed, IAM, audit via CloudTrail, auto-rotation | Vendor lock-in AWS | ✅ Default prod |
| **HashiCorp Vault** | Open source, self-hosted, dynamic secrets | Complexo de operar, exige infra extra | ❌ Roadmap (Sprint 35+) |
| **Kubernetes Secrets** | Nativo se for k8s | Não estamos em k8s (ECS Fargate) | ❌ YAGNI |
| **GCP Secret Manager** | Bom se multi-cloud | Stack é AWS-first | ❌ YAGNI |
| **Manter env var** | Simples | Vetores acima | ❌ Não-compliance |
| **SOPS + age** | GitOps-friendly | Não é runtime rotation | ❌ Complementar, não substitui |

### AWS SDK Go v2

- **Pacote:** `github.com/aws/aws-sdk-go-v2/service/secretsmanager` + `github.com/aws/aws-sdk-go-v2/config`
- **Auth:** IAM role (ECS task role), zero credentials em código.
- **Custo:** ~$0.40/secret/mês + $0.05/10k API calls.
- **Latência:** ~50ms GetSecretValue (cached em app por 5min).

### Padrão de uso

```go
// Hoje
senhawsClient.AlterarSenha(ctx, novaSenha)

// Sprint 28+
senhawsClient.AlterarSenha(ctx, novaSenha)
secretMgr.Update(ctx, "bacen/senha/123450001.fulano", novaSenha)
auditLog.Log("senhaws.rotate.completed", ...)
```

---

## Decisão arquitetural

### Interface `secrets.Manager`

```go
package secrets

type Secret struct {
    Name      string
    Value     string
    VersionID string  // AWS secret version
    CreatedAt time.Time
}

type Manager interface {
    Get(ctx context.Context, name string) (*Secret, error)
    Put(ctx context.Context, name, value string) (*Secret, error)
    Delete(ctx context.Context, name string) error
    Backend() string  // "aws" | "env" | "vault" | "memory"
}

// Factory
func NewManagerFromEnv(logger *slog.Logger) (Manager, error) {
    backend := strings.ToLower(os.Getenv("RADIANT_SECRETS_BACKEND"))
    switch backend {
    case "", "env":
        return NewEnvManager(), nil
    case "aws":
        return NewAWSManagerFromEnv(ctx, logger)
    case "memory":
        return NewMemoryManager(), nil
    default:
        return nil, fmt.Errorf("RADIANT_SECRETS_BACKEND=%q inválido", backend)
    }
}
```

### Implementação AWS

```go
type AWSManager struct {
    client *secretsmanager.Client
    logger *slog.Logger
}

func NewAWSManagerFromEnv(ctx context.Context, logger *slog.Logger) (*AWSManager, error) {
    cfg, err := config.LoadDefaultConfig(ctx,
        config.WithRegion(os.Getenv("AWS_REGION")),
    )
    if err != nil { return nil, err }

    return &AWSManager{
        client: secretsmanager.NewFromConfig(cfg),
        logger: logger,
    }, nil
}

func (m *AWSManager) Get(ctx context.Context, name string) (*Secret, error) {
    out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(name),
    })
    if err != nil { return nil, err }

    return &Secret{
        Name:      name,
        Value:     aws.ToString(out.SecretString),
        VersionID: aws.ToString(out.VersionId),
        CreatedAt: aws.ToTime(out.CreatedDate),
    }, nil
}

func (m *AWSManager) Put(ctx context.Context, name, value string) (*Secret, error) {
    out, err := m.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
        SecretId:     aws.String(name),
        SecretString: aws.String(value),
    })
    if err != nil { return nil, err }

    return &Secret{
        Name:      name,
        Value:     value,
        VersionID: aws.ToString(out.VersionId),
        CreatedAt: time.Now(),
    }, nil
}
```

### Implementação Env (fallback dev)

```go
type EnvManager struct {
    prefix string  // "RADIANT_SECRET_"
}

func (m *EnvManager) Get(ctx context.Context, name string) (*Secret, error) {
    envName := m.prefix + strings.ToUpper(strings.ReplaceAll(name, "/", "_"))
    val := os.Getenv(envName)
    if val == "" {
        return nil, &NotFoundError{Name: name}
    }
    return &Secret{Name: name, Value: val, CreatedAt: time.Now()}, nil
}

func (m *EnvManager) Put(ctx context.Context, name, value string) (*Secret, error) {
    envName := m.prefix + strings.ToUpper(strings.ReplaceAll(name, "/", "_"))
    if err := os.Setenv(envName, value); err != nil {
        return nil, err
    }
    return &Secret{Name: name, Value: value, CreatedAt: time.Now()}, nil
}
```

### Integração com senhaws-rotate

```go
// cmd/senhaws-rotate/main.go (modificado)
// Subcomando "apply" — após AlterarSenha, atualiza secret manager
func runApply(ctx context.Context, ...) error {
    cfg, err := loadConfig()
    if err != nil { return err }

    client, err := senhaws.NewSenhawsClient(cfg.senhaws)
    if err != nil { return err }

    secretMgr, err := secrets.NewManagerFromEnv(logger)
    if err != nil { return err }

    secretName := fmt.Sprintf("bacen/senha/%s", cfg.senhaws.User)
    current, err := secretMgr.Get(ctx, secretName)
    if err != nil { return err }

    novaSenha := senhaws.GerarSenhaRandom()
    if err := client.AlterarSenha(ctx, novaSenha); err != nil {
        return err
    }

    if _, err := secretMgr.Put(ctx, secretName, novaSenha); err != nil {
        // Falha crítica — BACEN aceitou senha nova mas secret manager falhou.
        // Caller precisa manualmente atualizar.
        return fmt.Errorf("senha alterada no BACEN mas falha ao persistir: %w", err)
    }

    auditLog.Log(ctx, "senhaws.rotate.applied", cfg.senhaws.User, ...)
    fmt.Printf("senha_alterada=true secret_updated=%s backend=%s\n", secretName, secretMgr.Backend())
    return nil
}
```

---

## Decisões YAGNI

- **Sem Vault integration agora** — Sprint 35+ roadmap. AWS cobre 90% do use case.
- **Sem in-memory cache** — app já é stateless, AWS SDK tem client-side caching.
- **Sem versionamento customizado** — AWS versiona automático (Stage labels: AWSCURRENT, AWSPREVIOUS).
- **Sem webhooks de rotação** — caller decide quando rotacionar (cron ou manual).
- **Sem cross-region replication** — single region (sa-east-1) por enquanto.
- **Sem diff de secret values** — values são strings opacas pra gente.

---

## Decisões de design não-óbvias

### 1. **Interface segregation antes de duplicação**

`Manager` interface é o mínimo (Get, Put, Delete). Adiciona-se capability depois se virar requisito (Rotate, List, etc).

### 2. **`EnvManager` é fallback, não substituto**

Pode parecer redundante ter env + AWS. Razões:
- Tests não querem mockar AWS SDK.
- Dev local pode rodar sem AWS (mesmo que com secret em env).
- Migration path: dev usa env, prod usa AWS. Sem breaking change.

### 3. **`Backend()` method para observabilidade**

Todo `Manager` retorna seu tipo via `Backend() string`. Usado em:
- Logs estruturados (`logger.Info("secret loaded", "backend", m.Backend())`)
- Métricas Prometheus (`radiant_secrets_backend{backend="aws"} 1`)
- Healthcheck (`/healthz` reporta qual backend está ativo)

### 4. **Audit log obrigatório**

Toda chamada `Put` e `Delete` DEVE emitir audit_log entry. Pattern consistente com codebase (Sprint 6+).

### 5. **Compile-time asserts em production source**

```go
var (
    _ Manager = (*AWSManager)(nil)
    _ Manager = (*EnvManager)(nil)
    _ Manager = (*MemoryManager)(nil)
)
```

### 6. **Naming convention**

Secret names em `kebab-case` ou `path-like`:
- `bacen/senha/{user}` — senha Sisbacen
- `bacen/sta/{if_id}/user` — usuário STA
- `radiant/jwt/dev-private-key` — chave privada JWT dev

Prefix `RADIANT_SECRET_` em env vars: `RADIANT_SECRET_BACEN_SENHA_12345_0001_FULANO`.

### 7. **Errors tipados**

```go
type NotFoundError struct{ Name string }
type AccessDeniedError struct{ Name string; Reason string }
type ValidationError struct{ Name, Reason string }
```

Caller usa `errors.As(err, &secrets.NotFoundError{})` para classificar.

---

## Entregas

### Arquivos novos

| Arquivo | LoC | Descrição |
|---|---|---|
| `internal/secrets/manager.go` | ~50 | Interface `Manager` + factory |
| `internal/secrets/aws.go` | ~150 | AWS SDK v2 implementation |
| `internal/secrets/env.go` | ~80 | Env fallback implementation |
| `internal/secrets/memory.go` | ~60 | In-memory (tests + dev local) |
| `internal/secrets/errors.go` | ~50 | Erros tipados (NotFound, AccessDenied, Validation) |
| `internal/secrets/manager_test.go` | ~200 | Tests table-driven |
| `internal/secrets/aws_test.go` | ~150 | Tests com AWS SDK mock |
| `cmd/secret-migrate/main.go` | ~250 | CLI que migra env → AWS (uma vez) |
| `cmd/secret-migrate/main_test.go` | ~150 | Tests |
| `SPRINT_28_RESEARCH.md` | (este) | Research |
| `SPRINT_28_RESULTS.md` | ~250 | Deliverables |
| `VALIDATION_49.md` | ~150 | Validação profunda |
| `CHANGELOG.md` | +50 | Entrada v3.23.0 |

### Arquivos modificados

| Arquivo | Mudança |
|---|---|
| `backend/go.mod` | + `aws-sdk-go-v2/service/secretsmanager` + `aws-sdk-go-v2/config` |
| `backend/go.sum` | checksum updates |
| `cmd/senhaws-rotate/main.go` | + subcomando `apply` que integra com Manager |
| `cmd/senhaws-rotate/main_test.go` | + tests do subcomando apply |

---

## Acceptance criteria

### Funcionais
- [ ] Interface `secrets.Manager` com 3 métodos (Get, Put, Delete) + Backend()
- [ ] `AWSManager` usando AWS SDK v2, com IAM role auth (zero credenciais hardcoded)
- [ ] `EnvManager` fallback seguro (não loga values, falha claro se ausente)
- [ ] `MemoryManager` para tests
- [ ] `cmd/secret-migrate` migra env vars → AWS Secrets Manager
- [ ] `cmd/senhaws-rotate apply` chama AlterarSenha + atualiza Manager + audit log
- [ ] Erros tipados (NotFoundError, AccessDeniedError, ValidationError)
- [ ] Compile-time asserts em production source
- [ ] Factory `NewManagerFromEnv` baseado em `RADIANT_SECRETS_BACKEND`

### Qualidade
- [ ] Coverage ≥ 85% no pacote `internal/secrets`
- [ ] Coverage ≥ 90% no `cmd/senhaws-rotate` (era 60.7%)
- [ ] Todos os testes passam com `-race`
- [ ] `gofmt -l ./backend` clean
- [ ] `go vet ./...` clean
- [ ] `lint-no-placeholder.sh` clean
- [ ] Smoke E2E contra secrets manager mock (httptest)
- [ ] Zero secrets em logs (regex scan + test que verifica)

### Compatibilidade
- [ ] Sem mudança em contratos de API REST
- [ ] Default behavior: se `RADIANT_SECRETS_BACKEND` não setado, usa `env` (back-compat)
- [ ] Sem mudança em senhaws-rotate subcomandos existentes (`check`, `rotate`, `info`)
- [ ] Audit log emite `senhaws.rotate.applied` (novo) + mantém `senhaws.rotate.completed` (existente)

### Documentação
- [ ] SPRINT_28_RESEARCH.md (este)
- [ ] SPRINT_28_RESULTS.md
- [ ] VALIDAÇÃO_49.md (validação profunda)
- [ ] CHANGELOG.md entrada v3.23.0
- [ ] godoc completo em todos os exports
- [ ] README atualizado com seção "Secrets Management"

---

## Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| **AWS SDK adiciona ~50MB ao binary** | Alta | Baixo | Já temos 25MB no binary; 50MB é aceitável. CGO_ENABLED=0 mantido. |
| **IAM role mal configurado** | Média | Alto | Documentação clara + smoke E2E + fail-fast no startup |
| **Race condition entre rotação BACEN e update AWS** | Baixa | Alto | Auditoria forte + DLQ-like pattern (se Put falha, deixa senha nova em memória + alerta) |
| **Env var ainda vaza em logs** | Média | Médio | EnvManager faz `os.Getenv` mas nunca loga value; só nome |
| **Migrar secrets existentes quebra deploy** | Baixa | Alto | `secret-migrate` roda dry-run primeiro; flag `--dry-run` |

---

## Próximos passos (Sprint 29+)

| Sprint | Escopo | Justificativa |
|---|---|---|
| **29** | BacenHomologSmoke | Smoke contra sta-h.bcb.gov.br/staws |
| **30** | PostgresRLS | Ativar migration 014_rls_enforce.sql |
| **35** | VaultIntegration | HashiCorp Vault se multi-cloud virar requisito |
| **47** | DRSACResearch | Solicitar acesso BACEN ao material 2030 |
| **45** | StripeBilling | Planos + self-service |

---

**Última atualização:** 2026-07-06 · Pronto pra implementação.