package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// newID returns a new UUID v4 string.
func newID() string {
	return uuid.New().String()
}

// contextWithTimeout is a shorthand for context.WithTimeout.
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// deliver attempts to deliver a webhook payload to a URL.
func deliver(ctx context.Context, url, event, payload, secret string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte(payload)))
	if err != nil {
		return 0, "", fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Radiant-Event", event)
	req.Header.Set("X-Radiant-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))

	if secret != "" {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(payload))
		sig := hex.EncodeToString(h.Sum(nil))
		req.Header.Set("X-Radiant-Signature", "sha256="+sig)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body), nil
}

// isRetryable returns true if the error should trigger a retry.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	// Retry on network errors and 5xx (but not 429 Rate Limited).
	// Note: we don't have access to status code here directly,
	// so we retry based on error message patterns.
	msg := err.Error()
	// "context deadline exceeded" = timeout → retry
	// "connection refused" → retry
	// "no such host" → retry
	retryable := containsAny(msg, "timeout", "deadline", "refused", "no such host",
		"connection reset", "temporary failure")
	return retryable
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) && contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
