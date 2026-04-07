-- ============================================================================
-- currencies.sql — Queries for currencies table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateCurrency :one
INSERT INTO currencies (id, code, numeric_code, name, decimal_places, is_active)
VALUES (gen_random_uuid(), $1, $2, $3, $4, true)
RETURNING *;

-- name: GetCurrencyByID :one
SELECT * FROM currencies WHERE id = $1;

-- name: GetCurrencyByCode :one
SELECT * FROM currencies WHERE code = $1;

-- name: UpdateCurrency :one
UPDATE currencies SET
    name = COALESCE(sqlc.narg('name'), name),
    numeric_code = COALESCE(sqlc.narg('numeric_code'), numeric_code),
    decimal_places = COALESCE(sqlc.narg('decimal_places'), decimal_places),
    is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = $1
RETURNING *;

-- name: ListCurrencies :many
SELECT * FROM currencies
ORDER BY code ASC;

-- name: ListActiveCurrencies :many
SELECT * FROM currencies
WHERE is_active = true
ORDER BY code ASC;
