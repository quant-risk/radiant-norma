package main

import (
	"os"
	"strings"
	"testing"
)

// =============================================================================
// Helpers tests
// =============================================================================

func TestLooksLikeSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
		why      string
	}{
		{"short", false, "too short (<8)"},
		{"longerlownodigit", false, "all lowercase, no digits (looks like plain text)"},
		{"lowerandDIGIT123", true, "mixed case + digits"},
		{"MixedCaseABC", true, "mixed case, no digits"},
		{"alllowernodig", false, "lowercase only, no digits (looks like plain text)"},
		{"12345678", true, "digits only (still >8 chars)"},
		{"abc123def", true, "lowercase + digits (>8)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeSecret(tt.input)
			if got != tt.expected {
				t.Errorf("looksLikeSecret(%q) = %v, want %v (%s)", tt.input, got, tt.expected, tt.why)
			}
		})
	}
}

func TestHasMixedCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc", false},
		{"ABC", false},
		{"Abc", true},
		{"aBc", true},
		{"123", false},
		{"Abc1", true},  // mixed case + digit
		{"1a2b", false}, // only digits + lowercase, no upper
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasMixedCase(tt.input); got != tt.expected {
				t.Errorf("hasMixedCase(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestContainsDigit(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abc", false},
		{"123", true},
		{"abc1", true},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := containsDigit(tt.input); got != tt.expected {
				t.Errorf("containsDigit(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Validation error tests
// =============================================================================

func TestValidationErr_Is(t *testing.T) {
	e1 := &validationErr{msg: "test"}
	e2 := &validationErr{msg: "other"}
	if !e1.Is(e2) {
		t.Error("validationErr should match other validationErr instances")
	}
	if e1.Error() != "test" {
		t.Errorf("Error() = %q, want %q", e1.Error(), "test")
	}
}

// TestBackendErr_Is — Validação 50: backendErr type usado por runList.
func TestBackendErr_Is(t *testing.T) {
	e1 := &backendErr{msg: "feature unavailable"}
	e2 := &backendErr{msg: "other reason"}
	if !e1.Is(e2) {
		t.Error("backendErr should match other backendErr instances")
	}
	if e1.Error() != "feature unavailable" {
		t.Errorf("Error() = %q, want %q", e1.Error(), "feature unavailable")
	}
	// Deve ser diferente de validationErr
	if e1.Is(&validationErr{msg: "x"}) {
		t.Error("backendErr should NOT match validationErr")
	}
}

// =============================================================================
// runMigrateBatch smoke (Validação 50 — coverage)
// =============================================================================

func TestRunMigrateBatch_ReadsJSON(t *testing.T) {
	tmpFile := t.TempDir() + "/secrets.json"
	jsonContent := `[{"from_env": "FAKE_ENV_1", "to_name": "fake/path/1"}, {"from_env": "FAKE_ENV_2", "to_name": "fake/path/2"}]`
	if err := os.WriteFile(tmpFile, []byte(jsonContent), 0600); err != nil {
		t.Fatalf("write tmp file: %v", err)
	}

	t.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	// FAKE_ENV_1 e FAKE_ENV_2 não estão setadas → SKIP em ambos → success_count=0
	// mas sem erro fatal
	err := runMigrateBatch([]string{"--file=" + tmpFile}, nil)
	if err != nil {
		t.Fatalf("runMigrateBatch should succeed even with missing env vars (SKIP): %v", err)
	}
}

func TestRunMigrateBatch_MissingFile(t *testing.T) {
	err := runMigrateBatch([]string{"--file=/nonexistent/path/secrets.json"}, nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if exitCode(err) != 1 {
		t.Errorf("exitCode = %d, want 1 (generic error for missing file)", exitCode(err))
	}
}

// =============================================================================
// exitCode tests
// =============================================================================

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"validationErr", &validationErr{msg: "x"}, 2},
		{"generic", os.ErrNotExist, 1},
		{"nil", nil, 1}, // nil falls through to default
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exitCode(tt.err); got != tt.want {
				t.Errorf("exitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

// =============================================================================
// runList tests (Validação 50 — exit 3 honesto em backend não-AWS)
// =============================================================================

// TestRunList_NonAWSReturnsError verifica que list em backend não-AWS retorna
// erro explícito (não exit 0 silencioso). Validação 50 F-S28-50-A: hollow stub
// removido — caller agora distingue "feature não suportada" de "lista vazia".
func TestRunList_NonAWSReturnsError(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	err := runList([]string{"--prefix=bacen/"}, nil)
	if err == nil {
		t.Fatal("runList should error on non-AWS backend (memory/env)")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got %v", err)
	}
	if !strings.Contains(err.Error(), "memory") {
		t.Errorf("error should mention backend name, got %v", err)
	}
	if exitCode(err) != 3 {
		t.Errorf("exitCode = %d, want 3 (backend error)", exitCode(err))
	}
}

// =============================================================================
// Smoke: full migrate dry-run path
// =============================================================================

func TestRunMigrate_DryRun(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	// Set a fake env var to migrate
	envKey := "TEST_MIGRATE_SOURCE_VAR"
	os.Setenv(envKey, "test-value-1234")
	defer os.Unsetenv(envKey)

	err := runMigrate([]string{
		"--from-env=" + envKey,
		"--to=test/migrated",
		"--dry-run",
	}, nil)
	if err != nil {
		t.Fatalf("runMigrate dry-run failed: %v", err)
	}

	// Verify dry-run did NOT actually migrate
	if os.Getenv(envKey) == "" {
		t.Error("dry-run should not have modified env var")
	}
}

func TestRunMigrate_MissingFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no from-env", []string{"--to=x"}, "--from-env required"},
		{"no to", []string{"--from-env=X"}, "--to required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runMigrate(tt.args, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.want)
			}
			if exitCode(err) != 2 {
				t.Errorf("exitCode = %d, want 2", exitCode(err))
			}
		})
	}
}

func TestRunMigrate_EmptyEnv(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	envKey := "TEST_NONEXISTENT_VAR"
	os.Unsetenv(envKey)

	err := runMigrate([]string{
		"--from-env=" + envKey,
		"--to=test/dest",
	}, nil)
	if err == nil {
		t.Fatal("expected error for empty env var")
	}
	if exitCode(err) != 2 {
		t.Errorf("exitCode = %d, want 2", exitCode(err))
	}
}
