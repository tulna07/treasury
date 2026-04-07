-- ============================================================================
-- credit_limits.sql — Queries for credit_limits table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateCreditLimit :one
INSERT INTO credit_limits (
    id, counterparty_id, limit_type, limit_amount, is_unlimited,
    effective_from, effective_to, is_current, expiry_date,
    approval_reference, created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, NULL, true, $6, $7,
    NOW(), $8, NOW(), $8
)
RETURNING *;

-- name: GetCreditLimitByID :one
SELECT * FROM credit_limits WHERE id = $1;

-- name: GetCurrentLimit :one
-- Lấy hạn mức hiện tại cho đối tác + loại
SELECT * FROM credit_limits
WHERE counterparty_id = $1
  AND limit_type = $2
  AND is_current = true;

-- name: DeactivateCurrentLimit :one
-- SCD Type 2: đóng bản ghi hiện tại lại khi có update
UPDATE credit_limits SET
    effective_to = $3,
    is_current = false,
    updated_by = $1
WHERE id = $2
RETURNING *;

-- name: GetLimitHistory :many
-- Lấy lịch sử thay đổi hạn mức của đối tác
SELECT * FROM credit_limits
WHERE counterparty_id = $1
  AND limit_type = $2
ORDER BY effective_from DESC;

-- name: ListCurrentLimits :many
-- Liệt kê tất cả hạn mức hiện tại
SELECT * FROM credit_limits
WHERE is_current = true
ORDER BY counterparty_id, limit_type
LIMIT $1 OFFSET $2;

-- name: CalculateUtilization :one
-- Tính toán utilization tổng hợp (MM + FX + Bond) cho uncollateralized limit
-- Query phức tạp này join các deals đã được duyệt nhưng chưa đáo hạn/settle
WITH uncollateralized_deals AS (
    -- MM Interbank (LEND/PLACE không có TSBĐ)
    SELECT
        mm.counterparty_id,
        CASE
            WHEN mm.currency_code = 'VND' THEN mm.principal_amount
            ELSE mm.principal_amount * sqlc.arg('usd_mid_rate')::NUMERIC
        END AS utilized_amount
    FROM mm_interbank_deals mm
    WHERE mm.counterparty_id = $1
      AND mm.has_collateral = false
      AND mm.direction IN ('LEND', 'PLACE')
      AND mm.status IN ('COMPLETED', 'PENDING_SETTLEMENT')
      AND mm.maturity_date > NOW()::DATE
      AND mm.deleted_at IS NULL

    UNION ALL

    -- FX Deals (nếu có consume limit)
    SELECT
        fx.counterparty_id,
        CASE
            WHEN fx.currency_code = 'VND' THEN fx.notional_amount
            ELSE fx.notional_amount * sqlc.arg('usd_mid_rate')::NUMERIC
        END AS utilized_amount
    FROM fx_deals fx
    WHERE fx.counterparty_id = $1
      AND fx.uses_credit_limit = true
      AND fx.status IN ('COMPLETED', 'PENDING_SETTLEMENT')
      AND fx.deleted_at IS NULL
    -- NOTE: Cần logic nghiệp vụ để xác định leg nào của deal FX sẽ được tính vào utilization

    UNION ALL

    -- Bond Deals (FI/CD - bán Repo)
    SELECT
        bd.counterparty_id,
        bd.total_value AS utilized_amount
    FROM bond_deals bd
    WHERE bd.counterparty_id = $1
      AND bd.direction = 'SELL'
      AND bd.transaction_type = 'REPO'
      AND bd.bond_category IN ('FINANCIAL_INSTITUTION', 'CERTIFICATE_OF_DEPOSIT')
      AND bd.status IN ('COMPLETED')
      AND bd.payment_date > NOW()::DATE
      AND bd.deleted_at IS NULL
)
SELECT
    COALESCE(SUM(utilized_amount), 0)::NUMERIC(20, 2) AS total_utilization
FROM uncollateralized_deals;

-- name: GetDailyLimitSummary :many
-- Lấy summary 11 cột theo BRD
-- Query này rất phức tạp và cần snapshot để có performance tốt
-- Dưới đây là query mẫu, trong thực tế sẽ chạy trên bảng snapshot
SELECT
    c.code AS counterparty_code,
    c.full_name AS counterparty_name,
    cl.limit_amount AS granted_limit,
    COALESCE(ls.utilized_opening, 0) AS opening_utilization,
    COALESCE(ls.utilized_intraday, 0) AS intraday_utilization,
    COALESCE(ls.utilized_total, 0) AS total_utilization,
    COALESCE(ls.remaining, cl.limit_amount) AS remaining_limit,
    cl.expiry_date
FROM counterparties c
JOIN credit_limits cl ON c.id = cl.counterparty_id
LEFT JOIN limit_utilization_snapshots ls ON c.id = ls.counterparty_id
    AND ls.snapshot_date = sqlc.arg('snapshot_date')::DATE
WHERE c.deleted_at IS NULL
  AND c.is_active = true
  AND cl.is_current = true
  AND cl.limit_type = 'UNCOLLATERALIZED'
ORDER BY c.code
LIMIT $1 OFFSET $2;
