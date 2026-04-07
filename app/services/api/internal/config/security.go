// Package config provides application configuration from environment variables.
package config

import (
	"net/http"
	"strings"
	"time"
)

// SecurityConfig holds security-related configuration with environment-based presets.
type SecurityConfig struct {
	// Auth mode: "standalone" or "zitadel"
	AuthMode string

	// Token settings
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	JWTSecret       string

	// Cookie settings
	CookieDomain   string
	CookieSecure   bool
	CookieSameSite string
	CookiePath     string

	// Security level presets
	SecurityLevel string

	// Redis
	RedisURL string

	// Rate limiting
	LoginRateLimit    int
	LoginRateWindow   time.Duration
	APIRateLimit      int
	APIRateWindow     time.Duration
	RefreshRateLimit  int
	RefreshRateWindow time.Duration

	// Session
	MaxSessionsPerUser int

	// Password policy (standalone mode)
	MinPasswordLength int
	RequireUppercase  bool
	RequireNumbers    bool
	RequireSpecial    bool

	// CORS
	AllowedOrigins []string
}

// LoadSecurityConfig reads security configuration from environment variables
// and applies preset defaults based on SECURITY_LEVEL.
func LoadSecurityConfig() SecurityConfig {
	level := env("SECURITY_LEVEL", "development")
	authMode := env("AUTH_MODE", "standalone")
	jwtSecret := env("JWT_SECRET", "treasury-dev-secret-change-in-production")
	cookieDomain := env("COOKIE_DOMAIN", "")
	cookieSameSite := env("COOKIE_SAMESITE", "Lax")
	redisURL := env("REDIS_URL", "redis://localhost:6379")
	allowedOrigins := envSlice("CORS_ALLOWED_ORIGINS", []string{
		"http://localhost:34000",
		"http://localhost:3000",
	})

	var cfg SecurityConfig

	switch level {
	case "production":
		cfg = SecurityConfig{
			AccessTokenTTL:     15 * time.Minute,
			RefreshTokenTTL:    7 * 24 * time.Hour,
			CookieSecure:       true,
			CookieSameSite:     "Lax",
			LoginRateLimit:     5,
			LoginRateWindow:    15 * time.Minute,
			APIRateLimit:       100,
			APIRateWindow:      1 * time.Minute,
			RefreshRateLimit:   10,
			RefreshRateWindow:  1 * time.Minute,
			MaxSessionsPerUser: 5,
			MinPasswordLength:  12,
			RequireUppercase:   true,
			RequireNumbers:     true,
			RequireSpecial:     true,
		}
	case "staging":
		cfg = SecurityConfig{
			AccessTokenTTL:     30 * time.Minute,
			RefreshTokenTTL:    14 * 24 * time.Hour,
			CookieSecure:       true,
			CookieSameSite:     "Lax",
			LoginRateLimit:     20,
			LoginRateWindow:    5 * time.Minute,
			APIRateLimit:       500,
			APIRateWindow:      1 * time.Minute,
			RefreshRateLimit:   20,
			RefreshRateWindow:  1 * time.Minute,
			MaxSessionsPerUser: 10,
			MinPasswordLength:  8,
			RequireUppercase:   true,
			RequireNumbers:     true,
			RequireSpecial:     false,
		}
	default: // development
		cfg = SecurityConfig{
			AccessTokenTTL:     1 * time.Hour,
			RefreshTokenTTL:    30 * 24 * time.Hour,
			CookieSecure:       false,
			CookieSameSite:     "Lax",
			LoginRateLimit:     100,
			LoginRateWindow:    1 * time.Minute,
			APIRateLimit:       1000,
			APIRateWindow:      1 * time.Minute,
			RefreshRateLimit:   50,
			RefreshRateWindow:  1 * time.Minute,
			MaxSessionsPerUser: 0, // unlimited
			MinPasswordLength:  6,
			RequireUppercase:   false,
			RequireNumbers:     false,
			RequireSpecial:     false,
		}
	}

	// Override with env vars
	cfg.SecurityLevel = level
	cfg.AuthMode = authMode
	cfg.JWTSecret = jwtSecret
	cfg.RedisURL = redisURL
	cfg.CookieDomain = cookieDomain
	cfg.CookiePath = "/"
	cfg.AllowedOrigins = allowedOrigins

	// Override sameSite from env if explicitly set
	if cookieSameSite != "" {
		cfg.CookieSameSite = cookieSameSite
	}

	// Override TTLs from env if explicitly set
	if v := env("ACCESS_TOKEN_TTL", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.AccessTokenTTL = d
		}
	}
	if v := env("REFRESH_TOKEN_TTL", ""); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.RefreshTokenTTL = d
		}
	}

	return cfg
}

// ParseSameSite converts a string SameSite value to http.SameSite.
func ParseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}
