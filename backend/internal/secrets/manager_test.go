package secrets

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// Compile-time guarantees: todas as implementações satisfazem Manager.
var (
	_ Manager = (*MemoryManager)(nil)
	_ Manager = (*EnvManager)(nil)
)

// =============================================================================
// MemoryManager tests
// =============================================================================

func TestMemoryManager_GetPutDelete(t *testing.T) {
	m := NewMemoryManager()
	ctx := context.Background()

	// Get missing → NotFound
	_, err := m.Get(ctx, "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}

	// Put → success
	s1, err := m.Put(ctx, "test/secret", "value123")
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}
	if s1.Name != "test/secret" || s1.Value != "value123" {
		t.Fatalf("Put returned wrong values: %+v", s1)
	}
	if s1.VersionID != "v1" {
		t.Fatalf("expected version v1, got %q", s1.VersionID)
	}

	// Get → success
	s2, err := m.Get(ctx, "test/secret")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s2.Value != "value123" {
		t.Fatalf("Get returned wrong value: %q", s2.Value)
	}

	// Put again → version increments
	s3, err := m.Put(ctx, "test/secret", "value456")
	if err != nil {
		t.Fatalf("Put 2nd failed: %v", err)
	}
	if s3.VersionID != "v2" {
		t.Fatalf("expected version v2, got %q", s3.VersionID)
	}

	// Delete → success
	if err := m.Delete(ctx, "test/secret"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get after delete → NotFound
	_, err = m.Get(ctx, "test/secret")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound after delete, got %v", err)
	}

	// Delete missing → NotFound
	err = m.Delete(ctx, "missing")
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound on delete missing, got %v", err)
	}
}

func TestMemoryManager_ValidationErrors(t *testing.T) {
	m := NewMemoryManager()
	ctx := context.Background()

	// Empty name
	_, err := m.Put(ctx, "", "value")
	if !IsValidation(err) {
		t.Fatalf("expected ValidationError for empty name, got %v", err)
	}

	// Empty value
	_, err = m.Put(ctx, "name", "")
	if !IsValidation(err) {
		t.Fatalf("expected ValidationError for empty value, got %v", err)
	}

	// Delete empty name
	err = m.Delete(ctx, "")
	if !IsValidation(err) {
		t.Fatalf("expected ValidationError for empty delete name, got %v", err)
	}
}

func TestMemoryManager_GetReturnsCopy(t *testing.T) {
	m := NewMemoryManager()
	ctx := context.Background()

	m.Set("test", "original", "v1")

	s, err := m.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Caller mutates returned secret — should NOT affect internal store
	s.Value = "mutated"

	s2, err := m.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Get 2nd failed: %v", err)
	}
	if s2.Value != "original" {
		t.Fatalf("internal store was mutated: got %q", s2.Value)
	}
}

func TestMemoryManager_Backend(t *testing.T) {
	m := NewMemoryManager()
	if m.Backend() != BackendMemory {
		t.Fatalf("expected %q, got %q", BackendMemory, m.Backend())
	}
}

// =============================================================================
// EnvManager tests
// =============================================================================

func TestEnvManager_NamingConvention(t *testing.T) {
	tests := []struct {
		secretName string
		envVar     string
	}{
		{"bacen/senha/123450001.fulano", "RADIANT_SECRET_BACEN_SENHA_123450001_FULANO"},
		{"simple", "RADIANT_SECRET_SIMPLE"},
		{"nested/deep/path", "RADIANT_SECRET_NESTED_DEEP_PATH"},
		{"MixedCase/with-dash", "RADIANT_SECRET_MIXEDCASE_WITH_DASH"},
	}

	for _, tt := range tests {
		t.Run(tt.secretName, func(t *testing.T) {
			prefix := "RADIANT_SECRET_"
			em := NewEnvManagerWithPrefix(prefix)
			got := em.envName(tt.secretName)
			if got != tt.envVar {
				t.Errorf("envName(%q) = %q, want %q", tt.secretName, got, tt.envVar)
			}
		})
	}
}

func TestEnvManager_GetPutDelete(t *testing.T) {
	// Use unique prefix to avoid pollution from other tests
	prefix := "TEST_SECRET_" + strings.ToUpper(t.Name()) + "_"
	em := NewEnvManagerWithPrefix(prefix)
	ctx := context.Background()
	name := "test/key"

	// Get missing
	_, err := em.Get(ctx, name)
	if !IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}

	// Put
	if _, err := em.Put(ctx, name, "secret-value"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify env var is set
	envName := prefix + "TEST_KEY"
	if got := os.Getenv(envName); got != "secret-value" {
		t.Fatalf("env var not set: got %q, want %q", got, "secret-value")
	}

	// Get
	s, err := em.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s.Value != "secret-value" {
		t.Fatalf("Get returned wrong value: %q", s.Value)
	}
	if em.Backend() != BackendEnv {
		t.Fatalf("Backend() = %q, want %q", em.Backend(), BackendEnv)
	}

	// Delete
	if err := em.Delete(ctx, name); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if got := os.Getenv(envName); got != "" {
		t.Fatalf("env var not unset: got %q", got)
	}

	// Delete missing → NotFound
	if err := em.Delete(ctx, name); !IsNotFound(err) {
		t.Fatalf("expected NotFound on delete missing, got %v", err)
	}

	// Cleanup
	os.Unsetenv(envName)
}

func TestEnvManager_ValidationErrors(t *testing.T) {
	em := NewEnvManager()
	ctx := context.Background()

	_, err := em.Get(ctx, "")
	if !IsValidation(err) {
		t.Fatalf("Get empty name: expected ValidationError, got %v", err)
	}

	_, err = em.Put(ctx, "", "value")
	if !IsValidation(err) {
		t.Fatalf("Put empty name: expected ValidationError, got %v", err)
	}

	_, err = em.Put(ctx, "name", "")
	if !IsValidation(err) {
		t.Fatalf("Put empty value: expected ValidationError, got %v", err)
	}
}

// =============================================================================
// Error classification tests
// =============================================================================

func TestErrors_IsHelpers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantFn  func(error) bool
		wantHit bool
	}{
		{
			name:    "NotFoundError",
			err:     &NotFoundError{Name: "x"},
			wantFn:  IsNotFound,
			wantHit: true,
		},
		{
			name:    "AccessDeniedError",
			err:     &AccessDeniedError{Name: "x"},
			wantFn:  IsAccessDenied,
			wantHit: true,
		},
		{
			name:    "ValidationError",
			err:     &ValidationError{Name: "x", Reason: "y"},
			wantFn:  IsValidation,
			wantHit: true,
		},
		{
			name:    "wrapped NotFoundError",
			err:     errorWrap(&NotFoundError{Name: "x"}),
			wantFn:  IsNotFound,
			wantHit: true,
		},
		{
			name:    "generic error → no hit",
			err:     errors.New("generic"),
			wantFn:  IsNotFound,
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.wantFn(tt.err)
			if got != tt.wantHit {
				t.Errorf("got %v, want %v", got, tt.wantHit)
			}
		})
	}
}

// errorWrap is a helper to construct errors for testing.
func errorWrap(err error) error {
	if err == nil {
		return nil
	}
	return &wrappedErr{inner: err}
}

type wrappedErr struct {
	inner error
}

func (e *wrappedErr) Error() string { return "wrapped: " + e.inner.Error() }
func (e *wrappedErr) Unwrap() error { return e.inner }

// =============================================================================
// Factory test
// =============================================================================

func TestNewManagerFromEnv_DefaultIsEnv(t *testing.T) {
	// Clear env var
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Unsetenv("RADIANT_SECRETS_BACKEND")

	mgr, err := NewManagerFromEnv(context.Background(), nil)
	if err != nil {
		t.Fatalf("NewManagerFromEnv failed: %v", err)
	}
	if mgr.Backend() != BackendEnv {
		t.Errorf("default backend = %q, want %q", mgr.Backend(), BackendEnv)
	}
}

func TestNewManagerFromEnv_Memory(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	mgr, err := NewManagerFromEnv(context.Background(), nil)
	if err != nil {
		t.Fatalf("NewManagerFromEnv failed: %v", err)
	}
	if mgr.Backend() != BackendMemory {
		t.Errorf("backend = %q, want %q", mgr.Backend(), BackendMemory)
	}
}

func TestNewManagerFromEnv_InvalidBackend(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "vault-foo")

	_, err := NewManagerFromEnv(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for invalid backend")
	}
	if !strings.Contains(err.Error(), "vault-foo") {
		t.Errorf("error should mention invalid value: %v", err)
	}
}

func TestNewManagerFromEnv_AWSRequiresRegion(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	oldRegion := os.Getenv("AWS_REGION")
	oldDefault := os.Getenv("AWS_DEFAULT_REGION")
	defer func() {
		os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
		os.Setenv("AWS_REGION", oldRegion)
		os.Setenv("AWS_DEFAULT_REGION", oldDefault)
	}()
	os.Setenv("RADIANT_SECRETS_BACKEND", "aws")
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_DEFAULT_REGION")

	// Will fail at LoadDefaultConfig because no AWS creds/region in test env.
	// The error message depends on whether there's a default config file.
	// We just check that it returns an error, not a successful manager.
	_, err := NewManagerFromEnv(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for AWS without region")
	}
	// Either "region not configured" or "config load failed" is acceptable
	if !strings.Contains(err.Error(), "region") && !strings.Contains(err.Error(), "config") {
		t.Errorf("expected region/config error, got: %v", err)
	}
}
