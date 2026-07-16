package webhook

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		url    string
		expect bool // true = should pass (nil error)
	}{
		{"https://example.com/webhook", true},
		{"http://localhost:8080/hook", true},
		{"http://a.co", true},
		{"", false},
		{"ftp://example.com", false},
		{"example.com", false},
		{string(make([]byte, 2050)), false}, // too long (>2048)
	}

	for _, tt := range tests {
		err := validateURL(tt.url)
		if (err == nil) != tt.expect {
			t.Errorf("validateURL(%q): got err=%v, expect pass=%v", tt.url, err, tt.expect)
		}
	}
}

func TestSignPayload(t *testing.T) {
	payload := `{"event":"test"}`
	secret := "my-secret"

	sig := SignPayload(payload, secret)
	if sig == "" {
		t.Error("expected non-empty signature")
	}

	// Same payload + same secret = same signature
	sig2 := SignPayload(payload, secret)
	if sig != sig2 {
		t.Error("same payload + secret should produce same signature")
	}

	// Different secret = different signature
	sig3 := SignPayload(payload, "other")
	if sig == sig3 {
		t.Error("different secret should produce different signature")
	}

	// Empty secret = empty signature
	sig4 := SignPayload(payload, "")
	if sig4 != "" {
		t.Error("empty secret should produce empty signature")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		errMsg  string
		status  int
		expect  bool
	}{
		// Network errors (status=0, no HTTP response) — retryable.
		{"context deadline exceeded", 0, true},
		{"connection refused", 0, true},
		{"no such host", 0, true},
		{"connection reset", 0, true},
		{"temporary failure", 0, true},
		// HTTP 429 — retryable.
		{"rate limited", 429, true},
		// HTTP 5xx — retryable.
		{"internal error", 500, true},
		{"bad gateway", 502, true},
		{"service unavailable", 503, true},
		// HTTP 4xx (non-429) — not retryable.
		{"unauthorized", 401, false},
		{"forbidden", 403, false},
		{"not found", 404, false},
		{"bad request", 400, false},
	}

	for _, tt := range tests {
		err := &testError{msg: tt.errMsg}
		got := isRetryable(err, tt.status)
		if got != tt.expect {
			t.Errorf("isRetryable(%q, %d): got %v, want %v", tt.errMsg, tt.status, got, tt.expect)
		}
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestNewID(t *testing.T) {
	id1 := newID()
	id2 := newID()
	if id1 == "" || len(id1) != 36 {
		t.Errorf("newID(): got %q, want 36-char UUID", id1)
	}
	if id1 == id2 {
		t.Error("two calls to newID() should produce different IDs")
	}
}

func TestContains(t *testing.T) {
	if !contains("hello world", "world") {
		t.Error("expected 'hello world' to contain 'world'")
	}
	if contains("hello", "world") {
		t.Error("expected 'hello' to not contain 'world'")
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("timeout error", "timeout", "deadline") {
		t.Error("should find 'timeout'")
	}
	if containsAny("ok error", "timeout", "deadline") {
		t.Error("should not find any")
	}
}

func TestContainsAny_NoMatch(t *testing.T) {
	if containsAny("hello", "world", "foo") {
		t.Error("should not find")
	}
}
