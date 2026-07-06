package secrets

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// AWSManager implementa Manager usando AWS Secrets Manager.
//
// Auth: IAM role (ECS task role, IRSA, ou AWS credentials padrão).
// Zero credenciais hardcoded — usa AWS SDK chain padrão.
//
// Operações:
//
//	Get → GetSecretValue
//	Put → PutSecretValue (cria versão nova)
//	Delete → DeleteSecret (com recovery window 7 dias por padrão)
type AWSManager struct {
	client             *secretsmanager.Client
	recoveryWindowDays int64
}

// NewAWSManagerFromEnv constrói AWSManager usando credenciais da AWS SDK chain.
//
// Region: AWS_REGION ou AWS_DEFAULT_REGION env var (padrão SDK).
// Endpoint: AWS Secrets Manager endpoint padrão (configurável via AWS_ENDPOINT_URL).
func NewAWSManagerFromEnv(ctx context.Context, logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}) (*AWSManager, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("aws config load failed: %w", err)
	}

	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region not configured (set AWS_REGION or AWS_DEFAULT_REGION)")
	}

	if logger != nil {
		logger.Info("AWS secrets manager inicializado", "region", cfg.Region)
	}

	return &AWSManager{
		client:             secretsmanager.NewFromConfig(cfg),
		recoveryWindowDays: 7, // AWS default
	}, nil
}

// NewAWSManagerWithClient cria AWSManager a partir de client injetado.
// Útil para tests com mock client.
func NewAWSManagerWithClient(client *secretsmanager.Client) *AWSManager {
	return &AWSManager{
		client:             client,
		recoveryWindowDays: 7,
	}
}

func (m *AWSManager) Get(ctx context.Context, name string) (*Secret, error) {
	if name == "" {
		return nil, &ValidationError{Name: name, Reason: "name cannot be empty"}
	}

	out, err := m.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})
	if err != nil {
		return nil, classifyAWSError(err, name)
	}

	if out.SecretString == nil {
		return nil, &ValidationError{
			Name:   name,
			Reason: "secret exists but SecretString is nil (binary secret?)",
		}
	}

	return &Secret{
		Name:      name,
		Value:     aws.ToString(out.SecretString),
		VersionID: aws.ToString(out.VersionId),
		CreatedAt: aws.ToTime(out.CreatedDate),
	}, nil
}

func (m *AWSManager) Put(ctx context.Context, name, value string) (*Secret, error) {
	if name == "" {
		return nil, &ValidationError{Name: name, Reason: "name cannot be empty"}
	}
	if value == "" {
		return nil, &ValidationError{Name: name, Reason: "value cannot be empty"}
	}

	out, err := m.client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(name),
		SecretString: aws.String(value),
	})
	if err != nil {
		return nil, classifyAWSError(err, name)
	}

	return &Secret{
		Name:      name,
		Value:     value,
		VersionID: aws.ToString(out.VersionId),
		CreatedAt: time.Now(), // local timestamp; GetSecretValue returns authoritative CreatedDate
	}, nil
}

func (m *AWSManager) Delete(ctx context.Context, name string) error {
	if name == "" {
		return &ValidationError{Name: name, Reason: "name cannot be empty"}
	}

	_, err := m.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:             aws.String(name),
		RecoveryWindowInDays: aws.Int64(m.recoveryWindowDays),
	})
	if err != nil {
		return classifyAWSError(err, name)
	}
	return nil
}

func (m *AWSManager) Backend() string { return BackendAWS }

// classifyAWSError mapeia erros AWS SDK para erros tipados do pacote.
//
// Decisão Sprint 28: usamos error message matching via reflection porque
// tipos concretos do AWS SDK mudam entre minor versions. Classificador
// baseado em nome do tipo + mensagem, robusto a mudanças de struct.
func classifyAWSError(err error, name string) error {
	if err == nil {
		return nil
	}

	errType := reflect.TypeOf(err)
	typeName := ""
	if errType != nil {
		typeName = errType.String()
	}
	errMsg := err.Error()

	// ResourceNotFoundException → NotFound
	if strings.Contains(typeName, "ResourceNotFoundException") ||
		strings.Contains(errMsg, "Secrets Manager can't find the specified secret") ||
		strings.Contains(errMsg, "ResourceNotFoundException") {
		return &NotFoundError{Name: name, Backend: BackendAWS, Cause: err}
	}

	// AccessDeniedException → AccessDenied
	if strings.Contains(typeName, "AccessDeniedException") ||
		strings.Contains(errMsg, "AccessDeniedException") ||
		strings.Contains(errMsg, "is not authorized to perform") {
		return &AccessDeniedError{Name: name, Backend: BackendAWS, Cause: err}
	}

	// InvalidParameterException → Validation
	if strings.Contains(typeName, "InvalidParameterException") ||
		strings.Contains(errMsg, "InvalidParameterException") {
		return &ValidationError{Name: name, Reason: extractErrMsg(errMsg), Cause: err}
	}

	// LimitExceededException → Validation
	if strings.Contains(typeName, "LimitExceededException") ||
		strings.Contains(errMsg, "LimitExceededException") {
		return &ValidationError{Name: name, Reason: "AWS limit exceeded: " + extractErrMsg(errMsg), Cause: err}
	}

	// Default: wrap with generic classification (caller can retry)
	return fmt.Errorf("AWS secrets manager error for %q: %w", name, err)
}

// extractErrMsg pulls message from AWS error format.
// AWS SDK error format: "operation: ErrorMessage"
// We strip the prefix to get the meaty part.
func extractErrMsg(s string) string {
	idx := strings.LastIndex(s, ": ")
	if idx >= 0 && idx+2 < len(s) {
		return s[idx+2:]
	}
	return s
}

// Compile-time guarantee: AWSManager implementa Manager.
var _ Manager = (*AWSManager)(nil)

// Reference errors.As to keep it in scope (avoid lint warning).
var _ = errors.As