// Package ratelimit provides Redis-based rate limiting using a sliding window counter.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter provides Redis-backed rate limiting.
type Limiter struct {
	client *redis.Client
}

// New creates a new Limiter connected to the given Redis URL.
// Returns an error if the connection or PING fails.
func New(redisURL string) (*Limiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("ratelimit: parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ratelimit: redis ping: %w", err)
	}

	return &Limiter{client: client}, nil
}

// NewFromClient creates a Limiter from an existing redis.Client (useful for testing).
func NewFromClient(client *redis.Client) *Limiter {
	return &Limiter{client: client}
}

// Allow checks whether the action identified by key is within the rate limit.
// It uses a fixed-window INCR + EXPIRE pattern.
// Returns:
//   - allowed: whether the request is permitted
//   - remaining: how many requests are left in the current window
//   - retryAfter: if not allowed, how long until the window resets
//   - err: any Redis error
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Duration, error) {
	pipe := l.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return false, 0, 0, fmt.Errorf("ratelimit: pipeline exec: %w", err)
	}

	count := int(incrCmd.Val())

	// If this is the first request in the window, set the expiry.
	if count == 1 {
		l.client.Expire(ctx, key, window)
	}

	// If no TTL set (key exists without expiry), set one as a safety net.
	ttl := ttlCmd.Val()
	if ttl < 0 {
		l.client.Expire(ctx, key, window)
		ttl = window
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	if count > limit {
		return false, remaining, ttl, nil
	}

	return true, remaining, 0, nil
}

// Reset clears the rate limit counter for a key.
func (l *Limiter) Reset(ctx context.Context, key string) error {
	return l.client.Del(ctx, key).Err()
}

// Close closes the Redis connection.
func (l *Limiter) Close() error {
	if l == nil || l.client == nil {
		return nil
	}
	return l.client.Close()
}

// ─── Key Builders ───

// LoginKey returns the rate limit key for login attempts by IP and username.
func LoginKey(ip, username string) string {
	return fmt.Sprintf("rl:login:%s:%s", ip, username)
}

// RefreshKey returns the rate limit key for token refresh by user ID.
func RefreshKey(userID string) string {
	return fmt.Sprintf("rl:refresh:%s", userID)
}

// APIKey returns the rate limit key for API calls by user ID.
func APIKey(userID string) string {
	return fmt.Sprintf("rl:api:%s", userID)
}

// IPKey returns the rate limit key for general IP-based limiting.
func IPKey(ip string) string {
	return fmt.Sprintf("rl:ip:%s", ip)
}
