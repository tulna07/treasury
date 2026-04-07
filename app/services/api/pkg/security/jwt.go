// Package security provides JWT, password hashing, and RBAC for the Treasury API.
package security

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// LoadJWTConfig reads JWT configuration from environment variables.
func LoadJWTConfig() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "treasury-dev-secret-change-in-production"
	}

	accessTTL := 15 * time.Minute
	if v := os.Getenv("JWT_ACCESS_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			accessTTL = d
		}
	}
	// Also support minutes as integer
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if m, err := strconv.Atoi(v); err == nil {
			accessTTL = time.Duration(m) * time.Minute
		}
	}

	refreshTTL := 7 * 24 * time.Hour
	if v := os.Getenv("JWT_REFRESH_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			refreshTTL = d
		}
	}

	return JWTConfig{
		Secret:     secret,
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}
}

// JWTManager handles token generation and validation.
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWTManager with the given config.
func NewJWTManager(config JWTConfig) *JWTManager {
	return &JWTManager{config: config}
}

// GenerateAccessToken creates a signed JWT access token.
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, roles []string, branchID string) (string, error) {
	now := time.Now()
	claims := dto.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTTL)),
			Issuer:    "treasury-api",
		},
		UserID:   userID,
		Roles:    roles,
		BranchID: branchID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// GenerateRefreshToken creates a signed JWT refresh token.
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.config.RefreshTTL)),
		Issuer:    "treasury-api",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// ValidateToken parses and validates a JWT token, returning the claims.
func (m *JWTManager) ValidateToken(tokenString string) (*dto.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &dto.TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*dto.TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ExtractClaims parses a token without full validation (for expired token inspection).
func (m *JWTManager) ExtractClaims(tokenString string) (*dto.TokenClaims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &dto.TokenClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*dto.TokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims type")
	}

	return claims, nil
}
