package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/kienlongbank/treasury-api/internal/ratelimit"
)

func setupLimiter(t *testing.T) (*miniredis.Miniredis, *ratelimit.Limiter) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := ratelimit.NewFromClient(client)
	return mr, limiter
}

func TestRateLimitMiddleware_Returns429(t *testing.T) {
	_, limiter := setupLimiter(t)
	defer limiter.Close()

	mw := RateLimit(limiter, 1, time.Minute, func(r *http.Request) string {
		return "test:mw:429"
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — allowed
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first request: got %d, want 200", rec.Code)
	}

	// Second request — denied
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second request: got %d, want 429", rec.Code)
	}

	// Verify body contains error info
	body := rec.Body.String()
	if !strings.Contains(body, "RATE_LIMITED") {
		t.Errorf("body should contain RATE_LIMITED, got: %s", body)
	}
}

func TestRateLimitMiddleware_SetsHeaders(t *testing.T) {
	_, limiter := setupLimiter(t)
	defer limiter.Close()

	mw := RateLimit(limiter, 5, time.Minute, func(r *http.Request) string {
		return "test:mw:headers"
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("X-RateLimit-Limit=%q, want 5", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "4" {
		t.Errorf("X-RateLimit-Remaining=%q, want 4", rec.Header().Get("X-RateLimit-Remaining"))
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("X-RateLimit-Reset should be set")
	}
}

func TestRateLimitMiddleware_RetryAfterHeader(t *testing.T) {
	_, limiter := setupLimiter(t)
	defer limiter.Close()

	mw := RateLimit(limiter, 1, time.Minute, func(r *http.Request) string {
		return "test:mw:retry"
	})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Second request — should have Retry-After
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") == "" {
		t.Error("Retry-After should be set when rate limited")
	}
}

func TestLoginRateLimit_PeeksBody(t *testing.T) {
	_, limiter := setupLimiter(t)
	defer limiter.Close()

	mw := LoginRateLimit(limiter, 10, time.Minute)

	var bodyReceived string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		bodyReceived = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"username":"testuser","password":"secret"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rec.Code)
	}
	if !strings.Contains(bodyReceived, "testuser") {
		t.Error("body should still contain username after middleware peek")
	}
}

func TestAPIRateLimit_UsesIPWhenNoUser(t *testing.T) {
	_, limiter := setupLimiter(t)
	defer limiter.Close()

	// With limit=2, unauthenticated requests fall back to IP key
	mw := APIRateLimit(limiter, 2, time.Minute)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests allowed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: got %d, want 200", i+1, rec.Code)
		}
	}

	// Third request denied
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("third request: got %d, want 429", rec.Code)
	}
}
