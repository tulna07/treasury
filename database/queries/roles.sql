-- ============================================================================
-- roles.sql — Queries for roles table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateRole :one
INSERT INTO roles (id, code, name, description, scope, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
RETURNING *;

-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: GetRoleByCode :one
SELECT * FROM roles WHERE code = $1;

-- name: UpdateRole :one
UPDATE roles SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    scope = COALESCE(sqlc.narg('scope'), scope)
WHERE id = $1
RETURNING *;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY code ASC;

-- name: ListRolesByUser :many
-- Lấy tất cả roles được gán cho user
SELECT r.* FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.code ASC;

-- name: AssignRoleToUser :one
INSERT INTO user_roles (id, user_id, role_id, granted_at, granted_by)
VALUES (gen_random_uuid(), $1, $2, NOW(), $3)
ON CONFLICT (user_id, role_id) DO NOTHING
RETURNING *;

-- name: RevokeRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = $2;
