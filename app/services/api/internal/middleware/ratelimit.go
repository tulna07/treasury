package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/ratelimit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// RateLimit returns a Chi middleware that applies rate limiting using the given key function.
// It sets standard rate limit headers and returns 429 when the limit is exceeded.
func RateLimit(limiter *ratelimit.Limiter, limit int, window time.Duration, keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			allowed, remaining, retryAfter, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil {
				// Redis error — let the request through (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(retryAfter).Unix(), 10))

			if !allowed {
				retrySeconds := int(retryAfter.Seconds())
				if retrySeconds < 1 {
					retrySeconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
				httputil.WriteJSON(w, http.StatusTooManyRequests, dto.APIResponse{
					Success: false,
					Error: &dto.APIError{
						Code:    "RATE_LIMITED",
						Message: fmt.Sprintf("Too many requests. Try again in %d seconds.", retrySeconds),
					},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LoginRateLimit returns a middleware that rate-limits login attempts.
// The key is based on client IP + username from the request body.
// The body is peeked without consuming it.
func LoginRateLimit(limiter *ratelimit.Limiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(limiter, limit, window, func(r *http.Request) string {
		ip := extractClientIP(r)
		username := peekUsername(r)
		return ratelimit.LoginKey(ip, username)
	})
}

// APIRateLimit returns a middleware that rate-limits authenticated API calls per user.
// The user ID is extracted from the request context (set by Auth middleware).
func APIRateLimit(limiter *ratelimit.Limiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(limiter, limit, window, func(r *http.Request) string {
		userID := ctxutil.GetUserID(r.Context())
		if userID == "" {
			return ratelimit.IPKey(extractClientIP(r))
		}
		return ratelimit.APIKey(userID)
	})
}

// extractClientIP gets the client IP from X-Forwarded-For, X-Real-Ip, or RemoteAddr.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// peekUsername reads the request body to extract the username field,
// then restores the body so downstream handlers can read it normally.
func peekUsername(r *http.Request) string {
	if r.Body == nil {
		return "_unknown_"
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "_unknown_"
	}
	// Restore the body
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Username == "" {
		return "_unknown_"
	}
	return payload.Username
}
