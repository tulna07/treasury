package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *Limiter) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewFromClient(client)
	return mr, limiter
}

func TestAllow_WithinLimit(t *testing.T) {
	_, limiter := setupMiniredis(t)
	defer limiter.Close()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		allowed, remaining, _, err := limiter.Allow(ctx, "test:key", 5, time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
		expected := 5 - (i + 1)
		if remaining != expected {
			t.Errorf("request %d: remaining=%d, want %d", i+1, remaining, expected)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	_, limiter := setupMiniredis(t)
	defer limiter.Close()

	ctx := context.Background()
	limit := 3

	for i := 0; i < limit; i++ {
		allowed, _, _, err := limiter.Allow(ctx, "test:exceed", limit, time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	allowed, remaining, retryAfter, err := limiter.Allow(ctx, "test:exceed", limit, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("request should be denied after exceeding limit")
	}
	if remaining != 0 {
		t.Errorf("remaining=%d, want 0", remaining)
	}
	if retryAfter <= 0 {
		t.Error("retryAfter should be positive when rate limited")
	}
}

func TestAllow_WindowExpiry(t *testing.T) {
	mr, limiter := setupMiniredis(t)
	defer limiter.Close()

	ctx := context.Background()
	limit := 2

	for i := 0; i < limit; i++ {
		limiter.Allow(ctx, "test:expiry", limit, 10*time.Second)
	}

	allowed, _, _, _ := limiter.Allow(ctx, "test:expiry", limit, 10*time.Second)
	if allowed {
		t.Fatal("should be denied")
	}

	mr.FastForward(11 * time.Second)

	allowed, _, _, err := limiter.Allow(ctx, "test:expiry", limit, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("should be allowed after window expiry")
	}
}

func TestReset(t *testing.T) {
	_, limiter := setupMiniredis(t)
	defer limiter.Close()

	ctx := context.Background()

	limiter.Allow(ctx, "test:reset", 2, time.Minute)
	limiter.Allow(ctx, "test:reset", 2, time.Minute)

	allowed, _, _, _ := limiter.Allow(ctx, "test:reset", 2, time.Minute)
	if allowed {
		t.Fatal("should be denied")
	}

	if err := limiter.Reset(ctx, "test:reset"); err != nil {
		t.Fatalf("reset error: %v", err)
	}

	allowed, remaining, _, err := limiter.Allow(ctx, "test:reset", 2, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatal("should be allowed after reset")
	}
	if remaining != 1 {
		t.Errorf("remaining=%d, want 1", remaining)
	}
}

func TestKeyBuilders(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"LoginKey", func() string { return LoginKey("1.2.3.4", "admin") }, "rl:login:1.2.3.4:admin"},
		{"RefreshKey", func() string { return RefreshKey("user-123") }, "rl:refresh:user-123"},
		{"APIKey", func() string { return APIKey("user-456") }, "rl:api:user-456"},
		{"IPKey", func() string { return IPKey("10.0.0.1") }, "rl:ip:10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestClose_Nil(t *testing.T) {
	var l *Limiter
	if err := l.Close(); err != nil {
		t.Errorf("Close on nil limiter should not error: %v", err)
	}
}
