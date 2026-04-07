// Package auth provides authentication and session management.
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
)

// Repository handles user and session data access.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new auth Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── User Operations ───

const userSelectBase = `
	SELECT u.id, u.username, u.email, u.full_name, u.password_hash,
	       u.branch_id, u.is_active, u.last_login_at, u.created_at, u.updated_at,
	       COALESCE(b.name, '') AS branch_name,
	       COALESCE(u.department, '') AS department,
	       COALESCE(u.position, '') AS position
	FROM users u
	LEFT JOIN branches b ON u.branch_id = b.id`

// GetByID retrieves a user by their UUID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.scanUser(ctx, userSelectBase+" WHERE u.id = $1 AND u.deleted_at IS NULL", id)
}

// GetByUsername retrieves a user by their username.
func (r *Repository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.scanUser(ctx, userSelectBase+" WHERE u.username = $1 AND u.deleted_at IS NULL", username)
}

// GetByEmail retrieves a user by their email.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.scanUser(ctx, userSelectBase+" WHERE u.email = $1 AND u.deleted_at IS NULL", email)
}

// scanUser scans a single user row from the given query.
func (r *Repository) scanUser(ctx context.Context, query string, args ...interface{}) (*model.User, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	var u model.User
	var branchID *uuid.UUID

	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.PasswordHash,
		&branchID, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&u.BranchName, &u.Department, &u.Position,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}

	if branchID != nil {
		u.BranchID = branchID.String()
	}

	return &u, nil
}

// UpdateLastLogin updates the user's last_login_at timestamp.
func (r *Repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID, loginAt time.Time) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, loginAt, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

// GetUserRoles retrieves the role codes assigned to a user.
func (r *Repository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.code
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.code`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, code)
	}

	return roles, rows.Err()
}

// AssignRole assigns a role to a user.
func (r *Repository) AssignRole(ctx context.Context, userID uuid.UUID, roleCode string, grantedBy uuid.UUID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, granted_by)
		SELECT $1, r.id, $3
		FROM roles r WHERE r.code = $2
		ON CONFLICT (user_id, role_id) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, userID, roleCode, grantedBy)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	return nil
}

// RemoveRole removes a role from a user.
func (r *Repository) RemoveRole(ctx context.Context, userID uuid.UUID, roleCode string) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE code = $2)`

	_, err := r.pool.Exec(ctx, query, userID, roleCode)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	return nil
}

// UpdatePassword updates the user's password hash.
func (r *Repository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// ─── Session Operations ───

// CreateSession inserts a new session record.
func (r *Repository) CreateSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO user_sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.TokenHash,
		session.IPAddress, session.UserAgent, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSessionByTokenHash retrieves a session by the SHA-256 hash of its refresh token.
func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	query := `
		SELECT id, user_id, token_hash, COALESCE(host(ip_address), ''), COALESCE(user_agent, ''),
		       expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE token_hash = $1`

	row := r.pool.QueryRow(ctx, query, tokenHash)
	var s model.Session
	err := row.Scan(
		&s.ID, &s.UserID, &s.TokenHash,
		&s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get session by token hash: %w", err)
	}
	return &s, nil
}

// RevokeSession marks a session as revoked.
func (r *Repository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// RevokeAllUserSessions revokes all active sessions for a user.
func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("revoke all user sessions: %w", err)
	}
	return nil
}

// RevokeOtherSessions revokes all sessions for a user except the given session.
func (r *Repository) RevokeOtherSessions(ctx context.Context, userID uuid.UUID, exceptSessionID uuid.UUID) error {
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND id != $2 AND revoked_at IS NULL`
	_, err := r.pool.Exec(ctx, query, userID, exceptSessionID)
	if err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}
	return nil
}

// CleanupExpiredSessions deletes expired and revoked sessions older than 30 days.
func (r *Repository) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM user_sessions
		WHERE (expires_at < NOW() OR revoked_at IS NOT NULL)
		AND created_at < NOW() - INTERVAL '30 days'`

	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CountActiveSessions returns the number of active (non-revoked, non-expired) sessions for a user.
func (r *Repository) CountActiveSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`

	var count int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active sessions: %w", err)
	}
	return count, nil
}

// GetOldestActiveSession returns the oldest active session for a user (for eviction).
func (r *Repository) GetOldestActiveSession(ctx context.Context, userID uuid.UUID) (*model.Session, error) {
	query := `
		SELECT id, user_id, token_hash, COALESCE(host(ip_address), ''), COALESCE(user_agent, ''),
		       expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at ASC
		LIMIT 1`

	row := r.pool.QueryRow(ctx, query, userID)
	var s model.Session
	err := row.Scan(
		&s.ID, &s.UserID, &s.TokenHash,
		&s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get oldest session: %w", err)
	}
	return &s, nil
}

// ListActiveSessions returns all active sessions for a user.
func (r *Repository) ListActiveSessions(ctx context.Context, userID uuid.UUID) ([]model.Session, error) {
	query := `
		SELECT id, user_id, token_hash, COALESCE(host(ip_address), ''), COALESCE(user_agent, ''),
		       expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.TokenHash,
			&s.IPAddress, &s.UserAgent,
			&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// GetSessionByID retrieves a session by ID.
func (r *Repository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	query := `
		SELECT id, user_id, token_hash, COALESCE(host(ip_address), ''), COALESCE(user_agent, ''),
		       expires_at, revoked_at, created_at
		FROM user_sessions
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	var s model.Session
	err := row.Scan(
		&s.ID, &s.UserID, &s.TokenHash,
		&s.IPAddress, &s.UserAgent,
		&s.ExpiresAt, &s.RevokedAt, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get session by id: %w", err)
	}
	return &s, nil
}
