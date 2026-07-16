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

// isRetryable returns true if the delivery should be retried.
// Phase 5: status parameter added — 429 and 5xx are retryable; 4xx (except 429) is not.
func isRetryable(err error, status int) bool {
	if err == nil {
		return false
	}
	// 429 Rate Limited — retry after backoff.
	if status == 429 {
		return true
	}
	// 5xx BACEN/internal errors — retry.
	if status >= 500 && status < 600 {
		return true
	}
	// 4xx client errors (except 429) — don't retry, caller must fix input.
	if status >= 400 && status < 500 {
		return false
	}
	// Network errors — retry.
	msg := err.Error()
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
