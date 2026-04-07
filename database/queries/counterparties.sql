-- ============================================================================
-- counterparties.sql — Queries for counterparties table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateCounterparty :one
INSERT INTO counterparties (
    id, code, full_name, short_name, cif, swift_code, country_code,
    tax_id, address, fx_uses_limit, is_active,
    created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, true,
    NOW(), $10, NOW(), $10
)
RETURNING *;

-- name: GetCounterpartyByID :one
SELECT * FROM counterparties
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCounterpartyByCode :one
SELECT * FROM counterparties
WHERE code = $1 AND deleted_at IS NULL;

-- name: UpdateCounterparty :one
UPDATE counterparties SET
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    short_name = COALESCE(sqlc.narg('short_name'), short_name),
    cif = COALESCE(sqlc.narg('cif'), cif),
    swift_code = COALESCE(sqlc.narg('swift_code'), swift_code),
    country_code = COALESCE(sqlc.narg('country_code'), country_code),
    tax_id = COALESCE(sqlc.narg('tax_id'), tax_id),
    address = COALESCE(sqlc.narg('address'), address),
    fx_uses_limit = COALESCE(sqlc.narg('fx_uses_limit'), fx_uses_limit),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCounterparty :exec
UPDATE counterparties SET deleted_at = NOW(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCounterparties :many
SELECT * FROM counterparties
WHERE deleted_at IS NULL
ORDER BY code ASC
LIMIT $1 OFFSET $2;

-- name: CountCounterparties :one
SELECT COUNT(*) FROM counterparties WHERE deleted_at IS NULL;

-- name: ListActiveCounterparties :many
SELECT * FROM counterparties
WHERE is_active = true AND deleted_at IS NULL
ORDER BY code ASC
LIMIT $1 OFFSET $2;

-- name: SearchCounterpartiesByName :many
-- Tìm kiếm đối tác theo tên (ILIKE — case insensitive)
SELECT * FROM counterparties
WHERE deleted_at IS NULL
  AND (full_name ILIKE '%' || $1 || '%' OR short_name ILIKE '%' || $1 || '%')
ORDER BY code ASC
LIMIT $2 OFFSET $3;

-- name: ListCounterpartiesWithLimits :many
-- Liệt kê đối tác kèm hạn mức hiện tại — dùng cho màn hình credit limit
SELECT
    c.*,
    cl_unc.limit_amount AS uncollateralized_limit,
    cl_unc.is_unlimited AS uncollateralized_unlimited,
    cl_col.limit_amount AS collateralized_limit,
    cl_col.is_unlimited AS collateralized_unlimited
FROM counterparties c
LEFT JOIN credit_limits cl_unc ON cl_unc.counterparty_id = c.id
    AND cl_unc.limit_type = 'UNCOLLATERALIZED'
    AND cl_unc.is_current = true
LEFT JOIN credit_limits cl_col ON cl_col.counterparty_id = c.id
    AND cl_col.limit_type = 'COLLATERALIZED'
    AND cl_col.is_current = true
WHERE c.deleted_at IS NULL AND c.is_active = true
ORDER BY c.code ASC
LIMIT $1 OFFSET $2;
