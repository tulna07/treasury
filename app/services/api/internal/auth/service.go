package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// Service provides authentication business logic.
type Service struct {
	repo     *Repository
	jwt      *security.JWTManager
	rbac     *security.RBACChecker
	security *config.SecurityConfig
	logger   *zap.Logger
}

// NewService creates a new auth Service.
func NewService(repo *Repository, jwt *security.JWTManager, rbac *security.RBACChecker, securityCfg *config.SecurityConfig, logger *zap.Logger) *Service {
	return &Service{
		repo:     repo,
		jwt:      jwt,
		rbac:     rbac,
		security: securityCfg,
		logger:   logger,
	}
}

// LoginResult contains the login response data including tokens (set via cookies by the handler).
type LoginResult struct {
	User         dto.UserProfile
	AccessToken  string
	RefreshToken string
	CSRFToken    string
}

// Login authenticates a user with username/password (standalone mode).
func (s *Service) Login(ctx context.Context, req dto.LoginRequest, ipAddress, userAgent string) (*LoginResult, error) {
	// 1. Find user by username
	user, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Error("failed to find user", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}
	if user == nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "invalid username or password")
	}

	// 2. Verify password
	if !security.VerifyPassword(req.Password, user.PasswordHash) {
		return nil, apperror.New(apperror.ErrUnauthorized, "invalid username or password")
	}

	// 3. Check user is_active
	if !user.IsActive {
		return nil, apperror.New(apperror.ErrForbidden, "account is deactivated")
	}

	// 4. Load roles
	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to get user roles", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	// 5. Check max sessions (evict oldest if exceeded)
	if s.security.MaxSessionsPerUser > 0 {
		count, err := s.repo.CountActiveSessions(ctx, user.ID)
		if err != nil {
			s.logger.Error("failed to count sessions", zap.Error(err))
			return nil, apperror.New(apperror.ErrInternal, "internal error")
		}
		for count >= s.security.MaxSessionsPerUser {
			oldest, err := s.repo.GetOldestActiveSession(ctx, user.ID)
			if err != nil || oldest == nil {
				break
			}
			if err := s.repo.RevokeSession(ctx, oldest.ID); err != nil {
				s.logger.Warn("failed to evict session", zap.Error(err))
				break
			}
			count--
		}
	}

	// 6. Generate access + refresh tokens
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, roles, user.BranchID)
	if err != nil {
		s.logger.Error("failed to generate access token", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	// 7. Create session record (store refresh token hash)
	tokenHash := hashToken(refreshToken)
	session := &model.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		IPAddress: ipAddress,
		UserAgent: truncateString(userAgent, 512),
		ExpiresAt: time.Now().Add(s.security.RefreshTokenTTL),
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		s.logger.Error("failed to create session", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	// 8. Update last_login_at
	now := time.Now()
	if err := s.repo.UpdateLastLogin(ctx, user.ID, now); err != nil {
		s.logger.Warn("failed to update last login", zap.Error(err))
	}

	// 9. Generate CSRF token
	csrfToken, err := s.generateCSRFToken()
	if err != nil {
		s.logger.Error("failed to generate CSRF token", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	return &LoginResult{
		User:         s.buildUserProfile(user, roles),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
	}, nil
}

// RefreshToken generates a new access token using a valid refresh token.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	// 1. Hash the refresh token
	tokenHash := hashToken(refreshToken)

	// 2. Find session by hash
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("failed to find session", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}
	if session == nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "invalid refresh token")
	}

	// 3. Check session valid
	if !session.IsValid() {
		return nil, apperror.New(apperror.ErrUnauthorized, "session expired or revoked")
	}

	// 4. Load user + roles
	user, err := s.repo.GetByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not found")
	}
	if !user.IsActive {
		return nil, apperror.New(apperror.ErrForbidden, "account is deactivated")
	}

	roles, err := s.repo.GetUserRoles(ctx, user.ID)
	if err != nil {
		s.logger.Error("failed to get user roles", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	// 5. Generate new access token (keep same refresh token)
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, roles, user.BranchID)
	if err != nil {
		s.logger.Error("failed to generate access token", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	// 6. Generate CSRF token
	csrfToken, err := s.generateCSRFToken()
	if err != nil {
		s.logger.Error("failed to generate CSRF token", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	return &LoginResult{
		User:         s.buildUserProfile(user, roles),
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // same refresh token
		CSRFToken:    csrfToken,
	}, nil
}

// Logout revokes the session associated with the given refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil // nothing to revoke
	}

	tokenHash := hashToken(refreshToken)
	session, err := s.repo.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		s.logger.Error("failed to find session for logout", zap.Error(err))
		return apperror.New(apperror.ErrInternal, "internal error")
	}
	if session == nil {
		return nil // already revoked or doesn't exist
	}

	return s.repo.RevokeSession(ctx, session.ID)
}

// LogoutAll revokes all sessions for the current user.
func (s *Service) LogoutAll(ctx context.Context) error {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return apperror.New(apperror.ErrUnauthorized, "not authenticated")
	}

	if err := s.repo.RevokeAllUserSessions(ctx, userID); err != nil {
		s.logger.Error("failed to revoke all sessions", zap.Error(err))
		return apperror.New(apperror.ErrInternal, "internal error")
	}
	return nil
}

// GetCurrentUser returns the profile of the currently authenticated user.
func (s *Service) GetCurrentUser(ctx context.Context) (*dto.UserProfile, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "not authenticated")
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}
	if user == nil {
		return nil, apperror.New(apperror.ErrNotFound, "user not found")
	}

	roles, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get roles", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	profile := s.buildUserProfile(user, roles)
	return &profile, nil
}

// ChangePassword changes the password for the current user (standalone mode).
func (s *Service) ChangePassword(ctx context.Context, req dto.ChangePasswordRequest) error {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return apperror.New(apperror.ErrUnauthorized, "not authenticated")
	}

	// 1. Load user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return apperror.New(apperror.ErrNotFound, "user not found")
	}

	// 2. Verify old password
	if !security.VerifyPassword(req.OldPassword, user.PasswordHash) {
		return apperror.New(apperror.ErrUnauthorized, "incorrect current password")
	}

	// 3. Validate new password (policy)
	if err := s.validatePasswordPolicy(req.NewPassword); err != nil {
		return err
	}

	// 4. Hash new password
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return apperror.New(apperror.ErrInternal, "internal error")
	}

	// 5. Update password
	if err := s.repo.UpdatePassword(ctx, userID, hash); err != nil {
		s.logger.Error("failed to update password", zap.Error(err))
		return apperror.New(apperror.ErrInternal, "internal error")
	}

	// 6. Revoke all sessions (force re-login)
	if err := s.repo.RevokeAllUserSessions(ctx, userID); err != nil {
		s.logger.Warn("failed to revoke sessions after password change", zap.Error(err))
	}

	return nil
}

// ListSessions returns all active sessions for the current user.
func (s *Service) ListSessions(ctx context.Context, currentRefreshToken string) ([]dto.SessionInfo, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "not authenticated")
	}

	sessions, err := s.repo.ListActiveSessions(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list sessions", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	currentHash := ""
	if currentRefreshToken != "" {
		currentHash = hashToken(currentRefreshToken)
	}

	result := make([]dto.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		result = append(result, dto.SessionInfo{
			ID:        sess.ID,
			IPAddress: sess.IPAddress,
			UserAgent: sess.UserAgent,
			CreatedAt: sess.CreatedAt,
			ExpiresAt: sess.ExpiresAt,
			IsCurrent: sess.TokenHash == currentHash,
		})
	}
	return result, nil
}

// RevokeSession revokes a specific session by ID (must belong to current user).
func (s *Service) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return apperror.New(apperror.ErrUnauthorized, "not authenticated")
	}

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		s.logger.Error("failed to get session", zap.Error(err))
		return apperror.New(apperror.ErrInternal, "internal error")
	}
	if session == nil {
		return apperror.New(apperror.ErrNotFound, "session not found")
	}
	if session.UserID != userID {
		return apperror.New(apperror.ErrForbidden, "cannot revoke another user's session")
	}

	return s.repo.RevokeSession(ctx, sessionID)
}

// buildUserProfile constructs a UserProfile DTO from a user model and roles.
func (s *Service) buildUserProfile(user *model.User, roles []string) dto.UserProfile {
	return dto.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		FullName:    user.FullName,
		Email:       user.Email,
		Roles:       roles,
		Permissions: s.rbac.GetPermissionsForRoles(roles),
		BranchID:    user.BranchID,
		BranchName:  user.BranchName,
		Department:  user.Department,
		Position:    user.Position,
		IsActive:    user.IsActive,
	}
}

// ─── Helpers ───

// hashToken returns the SHA-256 hex hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateRefreshToken generates a cryptographically random refresh token.
func (s *Service) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// generateCSRFToken generates a CSRF token for Double Submit Cookie pattern.
func (s *Service) generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate csrf token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// validatePasswordPolicy checks if a password meets the configured policy.
func (s *Service) validatePasswordPolicy(password string) error {
	if len(password) < s.security.MinPasswordLength {
		return apperror.NewWithDetail(apperror.ErrValidation, "password too short",
			fmt.Sprintf("minimum %d characters required", s.security.MinPasswordLength))
	}

	if s.security.RequireUppercase {
		hasUpper := false
		for _, c := range password {
			if unicode.IsUpper(c) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			return apperror.NewWithDetail(apperror.ErrValidation, "password requires uppercase", "at least one uppercase letter required")
		}
	}

	if s.security.RequireNumbers {
		hasDigit := false
		for _, c := range password {
			if unicode.IsDigit(c) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			return apperror.NewWithDetail(apperror.ErrValidation, "password requires number", "at least one digit required")
		}
	}

	if s.security.RequireSpecial {
		hasSpecial := false
		for _, c := range password {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			return apperror.NewWithDetail(apperror.ErrValidation, "password requires special character", "at least one special character required")
		}
	}

	return nil
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}


