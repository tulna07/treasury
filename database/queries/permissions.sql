-- ============================================================================
-- permissions.sql — Queries for permissions + role_permissions tables
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreatePermission :one
INSERT INTO permissions (id, code, resource, action, description, created_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW())
RETURNING *;

-- name: GetPermissionByID :one
SELECT * FROM permissions WHERE id = $1;

-- name: GetPermissionByCode :one
SELECT * FROM permissions WHERE code = $1;

-- name: ListPermissions :many
SELECT * FROM permissions
ORDER BY resource ASC, action ASC
LIMIT $1 OFFSET $2;

-- name: ListPermissionsByResource :many
SELECT * FROM permissions
WHERE resource = $1
ORDER BY action ASC;

-- name: ListPermissionsByRole :many
-- Lấy tất cả permissions của một role
SELECT p.* FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1
ORDER BY p.resource ASC, p.action ASC;

-- name: ListPermissionsByUser :many
-- Lấy tất cả permissions của user (qua user_roles + role_permissions)
SELECT DISTINCT p.* FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1
ORDER BY p.resource ASC, p.action ASC;

-- name: CheckUserPermission :one
-- Kiểm tra user có permission cụ thể không — dùng cho authorization middleware
SELECT EXISTS(
    SELECT 1 FROM permissions p
    JOIN role_permissions rp ON rp.permission_id = p.id
    JOIN user_roles ur ON ur.role_id = rp.role_id
    WHERE ur.user_id = $1 AND p.code = $2
) AS has_permission;

-- name: AssignPermissionToRole :one
INSERT INTO role_permissions (id, role_id, permission_id, created_at, created_by)
VALUES (gen_random_uuid(), $1, $2, NOW(), $3)
ON CONFLICT (role_id, permission_id) DO NOTHING
RETURNING *;

-- name: RevokePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;
