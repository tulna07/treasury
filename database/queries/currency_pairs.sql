-- ============================================================================
-- currency_pairs.sql — Queries for currency_pairs table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateCurrencyPair :one
INSERT INTO currency_pairs (
    id, base_currency, quote_currency, pair_code,
    rate_decimal_places, calculation_rule, result_currency, is_active
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, true
)
RETURNING *;

-- name: GetCurrencyPairByID :one
SELECT * FROM currency_pairs WHERE id = $1;

-- name: GetCurrencyPairByCode :one
SELECT * FROM currency_pairs WHERE pair_code = $1;

-- name: UpdateCurrencyPair :one
UPDATE currency_pairs SET
    rate_decimal_places = COALESCE(sqlc.narg('rate_decimal_places'), rate_decimal_places),
    calculation_rule = COALESCE(sqlc.narg('calculation_rule'), calculation_rule),
    result_currency = COALESCE(sqlc.narg('result_currency'), result_currency),
    is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = $1
RETURNING *;

-- name: ListCurrencyPairs :many
SELECT * FROM currency_pairs
ORDER BY pair_code ASC;

-- name: ListActiveCurrencyPairs :many
SELECT * FROM currency_pairs
WHERE is_active = true
ORDER BY pair_code ASC;

-- name: GetCurrencyPairByCurrencies :one
-- Tìm cặp tiền tệ theo base + quote currency
SELECT * FROM currency_pairs
WHERE base_currency = $1 AND quote_currency = $2 AND is_active = true;
