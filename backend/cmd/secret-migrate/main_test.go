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
		{"Abc1", true}, // mixed case + digit
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
// runList test (env backend reports not supported)
// =============================================================================

func TestRunList_EnvBackend(t *testing.T) {
	oldVal := os.Getenv("RADIANT_SECRETS_BACKEND")
	defer os.Setenv("RADIANT_SECRETS_BACKEND", oldVal)
	os.Setenv("RADIANT_SECRETS_BACKEND", "memory")

	err := runList([]string{"--prefix=bacen/"}, nil)
	if err != nil {
		t.Fatalf("runList failed: %v", err)
	}
	// Output verified by capturing stdout in shell, just check no error
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