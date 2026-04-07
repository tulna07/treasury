-- ============================================================================
-- bond_catalog.sql — Queries for bond_catalog table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateBondCatalog :one
INSERT INTO bond_catalog (
    id, bond_code, issuer, coupon_rate, payment_frequency,
    issue_date, maturity_date, face_value, bond_type,
    is_active, created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8,
    true, NOW(), $9, NOW(), $9
)
RETURNING *;

-- name: GetBondCatalogByID :one
SELECT * FROM bond_catalog WHERE id = $1;

-- name: GetBondCatalogByCode :one
SELECT * FROM bond_catalog WHERE bond_code = $1;

-- name: UpdateBondCatalog :one
UPDATE bond_catalog SET
    issuer = COALESCE(sqlc.narg('issuer'), issuer),
    coupon_rate = COALESCE(sqlc.narg('coupon_rate'), coupon_rate),
    payment_frequency = COALESCE(sqlc.narg('payment_frequency'), payment_frequency),
    face_value = COALESCE(sqlc.narg('face_value'), face_value),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_by = $1
WHERE id = $2
RETURNING *;

-- name: ListBondCatalog :many
SELECT * FROM bond_catalog
WHERE is_active = true
ORDER BY bond_code ASC
LIMIT $1 OFFSET $2;

-- name: ListBondCatalogByType :many
SELECT * FROM bond_catalog
WHERE bond_type = $1 AND is_active = true
ORDER BY bond_code ASC
LIMIT $2 OFFSET $3;

-- name: ListBondCatalogByMaturityRange :many
-- Tìm trái phiếu theo khoảng ngày đáo hạn
SELECT * FROM bond_catalog
WHERE maturity_date BETWEEN $1 AND $2
  AND is_active = true
ORDER BY maturity_date ASC
LIMIT $3 OFFSET $4;

-- name: SearchBondCatalog :many
-- Tìm kiếm theo mã hoặc tổ chức phát hành
SELECT * FROM bond_catalog
WHERE is_active = true
  AND (bond_code ILIKE '%' || $1 || '%' OR issuer ILIKE '%' || $1 || '%')
ORDER BY bond_code ASC
LIMIT $2 OFFSET $3;
