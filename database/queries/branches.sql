-- ============================================================================
-- branches.sql — Queries for branches table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateBranch :one
INSERT INTO branches (
    id, code, name, branch_type, parent_branch_id,
    flexcube_branch_code, swift_branch_code, address,
    is_active, created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7,
    true, NOW(), $8, NOW(), $8
)
RETURNING *;

-- name: GetBranchByID :one
SELECT * FROM branches WHERE id = $1;

-- name: GetBranchByCode :one
SELECT * FROM branches WHERE code = $1;

-- name: UpdateBranch :one
UPDATE branches SET
    name = COALESCE(sqlc.narg('name'), name),
    branch_type = COALESCE(sqlc.narg('branch_type'), branch_type),
    parent_branch_id = COALESCE(sqlc.narg('parent_branch_id'), parent_branch_id),
    flexcube_branch_code = COALESCE(sqlc.narg('flexcube_branch_code'), flexcube_branch_code),
    swift_branch_code = COALESCE(sqlc.narg('swift_branch_code'), swift_branch_code),
    address = COALESCE(sqlc.narg('address'), address),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_by = $1
WHERE id = $2
RETURNING *;

-- name: ListBranches :many
SELECT * FROM branches
WHERE is_active = true
ORDER BY code ASC
LIMIT $1 OFFSET $2;

-- name: ListBranchesByType :many
SELECT * FROM branches
WHERE branch_type = $1 AND is_active = true
ORDER BY code ASC;

-- name: ListChildBranches :many
-- Lấy danh sách chi nhánh con trực tiếp
SELECT * FROM branches
WHERE parent_branch_id = $1 AND is_active = true
ORDER BY code ASC;
