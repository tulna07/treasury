// Package config cung cấp cấu hình ứng dụng từ biến môi trường.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Config chứa toàn bộ cấu hình ứng dụng.
type Config struct {
	Server ServerConfig
	DB     DatabaseConfig
	Auth   AuthConfig
	CORS   CORSConfig
	Otel   OtelConfig
	Email  EmailConfig
}

// EmailConfig — cấu hình email SMTP + outbox worker.
type EmailConfig struct {
	Host        string // EMAIL_HOST (mặc định: localhost)
	Port        int    // EMAIL_PORT (mặc định: 1025 for dev)
	Username    string // EMAIL_USERNAME
	Password    string // EMAIL_PASSWORD
	FromAddress string // EMAIL_FROM (mặc định: treasury@kienlongbank.com)
	FromName    string // EMAIL_FROM_NAME (mặc định: KienlongBank Treasury)
	UseTLS      bool   // EMAIL_USE_TLS (mặc định: false for dev)
	MaxRetries  int    // EMAIL_MAX_RETRIES (mặc định: 3)
	RateLimit   int    // EMAIL_RATE_LIMIT (mặc định: 10 emails per second)
	BurstSize   int    // EMAIL_BURST_SIZE (mặc định: 20)
}

// OtelConfig -- cấu hình OpenTelemetry
type OtelConfig struct {
	Endpoint string // OTEL_EXPORTER_OTLP_ENDPOINT
}

// ServerConfig — cấu hình HTTP server.
type ServerConfig struct {
	Port string // APP_PORT (mặc định: 34080)
	Env  string // APP_ENV (mặc định: development)
}

// DatabaseConfig — cấu hình kết nối PostgreSQL.
type DatabaseConfig struct {
	URL             string // DATABASE_URL (bắt buộc)
	MaxConns        int32  // DATABASE_MAX_CONNS (mặc định: 25)
	MinConns        int32  // DATABASE_MIN_CONNS (mặc định: 5)
}

// AuthConfig — cấu hình xác thực OIDC.
type AuthConfig struct {
	IssuerURL string // AUTH_ISSUER_URL
}

// CORSConfig — cấu hình CORS.
type CORSConfig struct {
	AllowedOrigins []string // CORS_ALLOWED_ORIGINS (phân tách bằng dấu phẩy)
	MaxAge         int      // CORS_MAX_AGE (mặc định: 300)
}

// Load đọc cấu hình từ biến môi trường.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: env("APP_PORT", "34080"),
			Env:  env("APP_ENV", "development"),
		},
		DB: DatabaseConfig{
			URL:      env("DATABASE_URL", "postgres://localhost:5432/treasury?sslmode=disable"),
			MaxConns: int32(envInt("DATABASE_MAX_CONNS", 25)),
			MinConns: int32(envInt("DATABASE_MIN_CONNS", 5)),
		},
		Auth: AuthConfig{
			IssuerURL: env("AUTH_ISSUER_URL", "https://zitadel.xdigi.cloud"),
		},
		CORS: CORSConfig{
			AllowedOrigins: envSlice("CORS_ALLOWED_ORIGINS", []string{
				"http://localhost:34000",
				"http://localhost:3000",
			}),
			MaxAge: envInt("CORS_MAX_AGE", 300),
		},
		Otel: OtelConfig{
			Endpoint: env("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
		},
		Email: EmailConfig{
			Host:        env("EMAIL_HOST", "localhost"),
			Port:        envInt("EMAIL_PORT", 1025),
			Username:    env("EMAIL_USERNAME", ""),
			Password:    env("EMAIL_PASSWORD", ""),
			FromAddress: env("EMAIL_FROM", "treasury@kienlongbank.com"),
			FromName:    env("EMAIL_FROM_NAME", "KienlongBank Treasury"),
			UseTLS:      envBool("EMAIL_USE_TLS", false),
			MaxRetries:  envInt("EMAIL_MAX_RETRIES", 3),
			RateLimit:   envInt("EMAIL_RATE_LIMIT", 10),
			BurstSize:   envInt("EMAIL_BURST_SIZE", 20),
		},
	}
}

// --- Hàm hỗ trợ đọc biến môi trường ---

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func envSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return fallback
}
