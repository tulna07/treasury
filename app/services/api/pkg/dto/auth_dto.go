package dto

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// LoginRequest is the payload for user login.
// @Description Login credentials.
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

// LoginResponse is the response after successful login.
// Tokens are set in HTTP-only cookies — never exposed in JSON.
// @Description Login response with user profile. Tokens are in HTTP-only cookies.
type LoginResponse struct {
	User UserProfile `json:"user"`
}

// UserProfile represents a user's public profile.
// @Description User profile information.
type UserProfile struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"full_name"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	BranchID    string    `json:"branch_id"`
	BranchName  string    `json:"branch_name"`
	Department  string    `json:"department"`
	Position    string    `json:"position"`
	IsActive    bool      `json:"is_active"`
}

// ChangePasswordRequest is the payload for changing password.
// @Description Password change request.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=128"`
}

// TokenPair holds access and refresh tokens (set via cookies, never in JSON).
type TokenPair struct {
	AccessToken  string `json:"-"`
	RefreshToken string `json:"-"`
}

// SessionInfo represents an active user session.
// @Description Active session information.
type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsCurrent bool      `json:"is_current"`
}

// TokenClaims holds the claims embedded in a JWT token.
type TokenClaims struct {
	jwt.RegisteredClaims
	UserID   uuid.UUID `json:"user_id"`
	Roles    []string  `json:"roles"`
	BranchID string    `json:"branch_id"`
}
