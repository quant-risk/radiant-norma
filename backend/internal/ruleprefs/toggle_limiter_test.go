// Package ruleprefs — tests do ToggleLimiter.

package ruleprefs

import (
	"testing"
	"time"
)

func TestToggleLimiter_AllowsUnderLimit(t *testing.T) {
	l := NewToggleLimiter(3, time.Minute)

	for i := 0; i < 3; i++ {
		ok, retry := l.Allow("demo")
		if !ok {
			t.Errorf("call %d: should be allowed, got blocked (retry=%v)", i, retry)
		}
	}
}

func TestToggleLimiter_BlocksOverLimit(t *testing.T) {
	l := NewToggleLimiter(3, time.Minute)

	// First 3 allowed
	for i := 0; i < 3; i++ {
		ok, _ := l.Allow("demo")
		if !ok {
			t.Fatalf("call %d should be allowed", i)
		}
	}

	// 4th should be blocked
	ok, retry := l.Allow("demo")
	if ok {
		t.Error("4th call should be blocked")
	}
	if retry <= 0 {
		t.Errorf("retry should be positive, got %v", retry)
	}
	if retry > time.Minute {
		t.Errorf("retry should be <= window, got %v", retry)
	}
}

func TestToggleLimiter_PerKeyIsolation(t *testing.T) {
	l := NewToggleLimiter(2, time.Minute)

	// demo: 2 allowed
	l.Allow("demo")
	l.Allow("demo")

	// other: still 0
	ok, _ := l.Allow("other")
	if !ok {
		t.Error("other should be allowed (independent counter)")
	}
}

func TestToggleLimiter_SlidingWindowExpiry(t *testing.T) {
	// Use small window for fast test
	l := NewToggleLimiter(2, 50*time.Millisecond)

	// Fill up
	l.Allow("demo")
	l.Allow("demo")

	// Blocked
	ok, _ := l.Allow("demo")
	if ok {
		t.Fatal("3rd call should be blocked")
	}

	// Wait for window to expire
	time.Sleep(80 * time.Millisecond)

	// Now allowed again
	ok, _ = l.Allow("demo")
	if !ok {
		t.Error("after window expiry, should be allowed")
	}
}

func TestToggleLimiter_Reset(t *testing.T) {
	l := NewToggleLimiter(1, time.Minute)

	l.Allow("demo")
	ok, _ := l.Allow("demo")
	if ok {
		t.Fatal("2nd call should be blocked")
	}

	l.Reset()

	ok, _ = l.Allow("demo")
	if !ok {
		t.Error("after reset, should be allowed")
	}
}