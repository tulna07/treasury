-- ============================================================================
-- sessions.sql — Queries for user_sessions table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateSession :one
INSERT INTO user_sessions (id, user_id, token_hash, ip_address, user_agent, expires_at, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, NOW())
RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM user_sessions WHERE id = $1;

-- name: GetSessionByTokenHash :one
-- Tìm session active theo token hash — dùng cho middleware authentication
SELECT s.*, u.username, u.full_name, u.is_active AS user_is_active
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
  AND s.expires_at > NOW()
  AND s.revoked_at IS NULL
  AND u.deleted_at IS NULL;

-- name: RevokeSession :exec
-- Thu hồi một session (logout)
UPDATE user_sessions SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
-- Thu hồi tất cả sessions của user (force logout all devices)
UPDATE user_sessions SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CleanupExpiredSessions :exec
-- Dọn dẹp sessions đã hết hạn quá 30 ngày
DELETE FROM user_sessions
WHERE expires_at < NOW() - INTERVAL '30 days';

-- name: ListActiveSessionsByUser :many
SELECT * FROM user_sessions
WHERE user_id = $1
  AND expires_at > NOW()
  AND revoked_at IS NULL
ORDER BY created_at DESC;
