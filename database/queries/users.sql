-- ============================================================================
-- users.sql — Queries for users table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateUser :one
INSERT INTO users (
    id, external_id, username, password_hash, full_name, email,
    branch_id, department, position, is_active, created_at, updated_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, true, NOW(), NOW()
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByExternalID :one
-- Dùng khi AUTH_MODE=zitadel: match user từ JWT claim
SELECT * FROM users
WHERE external_id = $1 AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users SET
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    email = COALESCE(sqlc.narg('email'), email),
    branch_id = COALESCE(sqlc.narg('branch_id'), branch_id),
    department = COALESCE(sqlc.narg('department'), department),
    position = COALESCE(sqlc.narg('position'), position),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
ORDER BY full_name ASC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL;

-- name: ListUsersByDepartment :many
SELECT * FROM users
WHERE department = $1 AND deleted_at IS NULL AND is_active = true
ORDER BY full_name ASC
LIMIT $2 OFFSET $3;

-- name: ListUsersByBranch :many
SELECT * FROM users
WHERE branch_id = $1 AND deleted_at IS NULL AND is_active = true
ORDER BY full_name ASC
LIMIT $2 OFFSET $3;

-- name: ListActiveUsersByRole :many
-- Liệt kê users đang hoạt động theo role code — dùng cho assign approval
SELECT u.* FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
WHERE r.code = $1
  AND u.is_active = true
  AND u.deleted_at IS NULL
ORDER BY u.full_name ASC
LIMIT $2 OFFSET $3;
