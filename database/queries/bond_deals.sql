-- ============================================================================
-- bond_deals.sql — Queries for bond_deals table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateBondDeal :one
INSERT INTO bond_deals (
    id, deal_number, bond_category, trade_date, branch_id,
    order_date, value_date, direction, counterparty_id,
    transaction_type, transaction_type_other,
    bond_catalog_id, bond_code_manual, issuer, coupon_rate,
    issue_date, maturity_date, quantity, face_value,
    discount_rate, clean_price, settlement_price, total_value,
    portfolio_type, payment_date, remaining_tenor_days,
    confirmation_method, confirmation_other, contract_prepared_by,
    status, note, cloned_from_id,
    created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22,
    $23, $24, $25, $26, $27, $28,
    'OPEN', $29, $30,
    NOW(), $31, NOW(), $31
)
RETURNING *;

-- name: GetBondDealByID :one
SELECT * FROM bond_deals
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateBondDeal :one
-- Chỉ cập nhật khi OPEN
UPDATE bond_deals SET
    order_date = COALESCE(sqlc.narg('order_date'), order_date),
    value_date = COALESCE(sqlc.narg('value_date'), value_date),
    counterparty_id = COALESCE(sqlc.narg('counterparty_id'), counterparty_id),
    transaction_type = COALESCE(sqlc.narg('transaction_type'), transaction_type),
    transaction_type_other = COALESCE(sqlc.narg('transaction_type_other'), transaction_type_other),
    bond_catalog_id = COALESCE(sqlc.narg('bond_catalog_id'), bond_catalog_id),
    bond_code_manual = COALESCE(sqlc.narg('bond_code_manual'), bond_code_manual),
    issuer = COALESCE(sqlc.narg('issuer'), issuer),
    coupon_rate = COALESCE(sqlc.narg('coupon_rate'), coupon_rate),
    quantity = COALESCE(sqlc.narg('quantity'), quantity),
    face_value = COALESCE(sqlc.narg('face_value'), face_value),
    discount_rate = COALESCE(sqlc.narg('discount_rate'), discount_rate),
    clean_price = COALESCE(sqlc.narg('clean_price'), clean_price),
    settlement_price = COALESCE(sqlc.narg('settlement_price'), settlement_price),
    total_value = COALESCE(sqlc.narg('total_value'), total_value),
    portfolio_type = COALESCE(sqlc.narg('portfolio_type'), portfolio_type),
    payment_date = COALESCE(sqlc.narg('payment_date'), payment_date),
    remaining_tenor_days = COALESCE(sqlc.narg('remaining_tenor_days'), remaining_tenor_days),
    note = COALESCE(sqlc.narg('note'), note),
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'
RETURNING *;

-- name: SoftDeleteBondDeal :exec
UPDATE bond_deals SET deleted_at = NOW(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL AND status = 'OPEN';

-- name: UpdateBondDealStatus :one
-- Optimistic status update
UPDATE bond_deals SET
    status = $3,
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;

-- name: ListBondDeals :many
SELECT * FROM bond_deals
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBondDeals :one
SELECT COUNT(*) FROM bond_deals WHERE deleted_at IS NULL;

-- name: ListBondDealsByCategory :many
-- Lọc theo loại: GOVERNMENT (TPCP), FINANCIAL_INSTITUTION (TCTC), CERTIFICATE_OF_DEPOSIT (GTCG)
SELECT * FROM bond_deals
WHERE bond_category = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBondDealsByStatus :many
SELECT * FROM bond_deals
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListBondDealsPendingBooking :many
-- View cho KTTC: deals chờ hạch toán (L1 hoặc L2)
SELECT * FROM bond_deals
WHERE status IN ('PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT')
  AND deleted_at IS NULL
ORDER BY trade_date ASC, created_at ASC
LIMIT $1 OFFSET $2;

-- name: GetNextBondDealCode :one
-- Lấy deal code tiếp theo để sinh prefix: G (Govi) hoặc F (FI/CD)
-- Application logic: bond_category = 'GOVERNMENT' → prefix 'G', else 'F'
SELECT
    CASE
        WHEN $1 = 'GOVERNMENT' THEN 'G'
        ELSE 'F'
    END AS prefix;

-- name: CheckInventoryBeforeSell :one
-- Kiểm tra tồn kho trước khi bán — hard block nếu không đủ
-- Trả về available_quantity: 0 nếu không tìm thấy record
SELECT COALESCE(bi.available_quantity, 0) AS available_quantity
FROM bond_inventory bi
WHERE bi.bond_code = $1
  AND bi.bond_category = $2
  AND bi.portfolio_type = $3;

-- name: SetBondDealCancelRequest :one
UPDATE bond_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'CANCELLED',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;
