-- ============================================================================
-- exchange_rates.sql — Queries for exchange_rates table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateExchangeRate :one
INSERT INTO exchange_rates (
    id, currency_code, effective_date,
    buy_transfer_rate, sell_transfer_rate, mid_rate,
    source, created_at, created_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), $7
)
RETURNING *;

-- name: GetExchangeRateByID :one
SELECT * FROM exchange_rates WHERE id = $1;

-- name: GetLatestRateByDate :one
-- Lấy tỷ giá mới nhất cho một currency tính đến ngày chỉ định
-- Logic "ngày làm việc trước đó": ứng dụng tính previous business day rồi truyền vào
SELECT * FROM exchange_rates
WHERE currency_code = $1 AND effective_date <= $2
ORDER BY effective_date DESC
LIMIT 1;

-- name: GetMidRate :one
-- Lấy mid rate cho quy đổi credit limit
SELECT mid_rate FROM exchange_rates
WHERE currency_code = $1 AND effective_date = $2;

-- name: ListRatesByDate :many
-- Lấy tất cả tỷ giá của một ngày
SELECT * FROM exchange_rates
WHERE effective_date = $1
ORDER BY currency_code ASC;

-- name: ListRatesByCurrencyAndDateRange :many
SELECT * FROM exchange_rates
WHERE currency_code = $1
  AND effective_date BETWEEN $2 AND $3
ORDER BY effective_date DESC
LIMIT $4 OFFSET $5;

-- name: UpsertExchangeRate :one
-- Insert or update tỷ giá — unique on (currency_code, effective_date)
INSERT INTO exchange_rates (
    id, currency_code, effective_date,
    buy_transfer_rate, sell_transfer_rate, mid_rate,
    source, created_at, created_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), $7
)
ON CONFLICT (currency_code, effective_date) DO UPDATE SET
    buy_transfer_rate = EXCLUDED.buy_transfer_rate,
    sell_transfer_rate = EXCLUDED.sell_transfer_rate,
    mid_rate = EXCLUDED.mid_rate,
    source = EXCLUDED.source,
    created_by = EXCLUDED.created_by
RETURNING *;
