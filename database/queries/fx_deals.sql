-- ============================================================================
-- fx_deals.sql — Queries for fx_deals table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateFxDeal :one
INSERT INTO fx_deals (
    id, deal_number, ticket_number, counterparty_id,
    deal_type, direction, notional_amount, currency_code, pair_code,
    trade_date, branch_id, uses_credit_limit, status, note,
    cloned_from_id, created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, 'OPEN', $12,
    $13, NOW(), $14, NOW(), $14
)
RETURNING *;

-- name: GetFxDealByID :one
SELECT * FROM fx_deals
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateFxDeal :one
-- Chỉ cập nhật khi deal ở trạng thái OPEN
UPDATE fx_deals SET
    ticket_number = COALESCE(sqlc.narg('ticket_number'), ticket_number),
    counterparty_id = COALESCE(sqlc.narg('counterparty_id'), counterparty_id),
    deal_type = COALESCE(sqlc.narg('deal_type'), deal_type),
    direction = COALESCE(sqlc.narg('direction'), direction),
    notional_amount = COALESCE(sqlc.narg('notional_amount'), notional_amount),
    currency_code = COALESCE(sqlc.narg('currency_code'), currency_code),
    pair_code = COALESCE(sqlc.narg('pair_code'), pair_code),
    trade_date = COALESCE(sqlc.narg('trade_date'), trade_date),
    uses_credit_limit = COALESCE(sqlc.narg('uses_credit_limit'), uses_credit_limit),
    note = COALESCE(sqlc.narg('note'), note),
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'
RETURNING *;

-- name: SoftDeleteFxDeal :exec
UPDATE fx_deals SET deleted_at = NOW(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL AND status = 'OPEN';

-- name: UpdateFxDealStatus :one
-- Optimistic status transition: chỉ cập nhật nếu status hiện tại khớp
-- Application phải kiểm tra status_transition_rules trước khi gọi
UPDATE fx_deals SET
    status = $3,
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;

-- name: SetFxDealCancelRequest :one
-- Dealer yêu cầu hủy deal đã hoàn thành
UPDATE fx_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'CANCELLED',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;

-- name: ListFxDeals :many
SELECT * FROM fx_deals
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountFxDeals :one
SELECT COUNT(*) FROM fx_deals WHERE deleted_at IS NULL;

-- name: ListFxDealsByStatus :many
SELECT * FROM fx_deals
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListFxDealsByCounterpartyAndDateRange :many
SELECT * FROM fx_deals
WHERE counterparty_id = $1
  AND trade_date BETWEEN $2 AND $3
  AND deleted_at IS NULL
ORDER BY trade_date DESC, created_at DESC
LIMIT $4 OFFSET $5;

-- name: ListFxDealsByTradeDate :many
SELECT * FROM fx_deals
WHERE trade_date = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListFxDealsPendingApproval :many
-- Deals chờ phê duyệt — view cho Desk Head / Director / Accountant
SELECT * FROM fx_deals
WHERE status = $1 AND deleted_at IS NULL
ORDER BY trade_date ASC, created_at ASC
LIMIT $2 OFFSET $3;

-- name: GetFxDealWithLegs :many
-- Lấy deal header kèm tất cả legs — dùng cho chi tiết deal
SELECT
    d.*,
    l.id AS leg_id,
    l.leg_number,
    l.value_date,
    l.settlement_date,
    l.exchange_rate AS leg_exchange_rate,
    l.converted_amount,
    l.converted_currency,
    l.internal_ssi_id,
    l.counterparty_ssi_id,
    l.requires_international_settlement
FROM fx_deals d
LEFT JOIN fx_deal_legs l ON l.deal_id = d.id
WHERE d.id = $1 AND d.deleted_at IS NULL
ORDER BY l.leg_number ASC;

-- name: CountFxDealsByStatusAndDate :many
-- Dashboard: đếm deal theo status cho một ngày — hiển thị summary cards
SELECT status, COUNT(*) AS deal_count
FROM fx_deals
WHERE trade_date = $1 AND deleted_at IS NULL
GROUP BY status
ORDER BY status;

-- name: ListFxDealsForSettlement :many
-- Deals cần thanh toán quốc tế — chỉ deals đã qua phê duyệt booking
SELECT d.* FROM fx_deals d
JOIN fx_deal_legs l ON l.deal_id = d.id
WHERE d.status = 'PENDING_SETTLEMENT'
  AND d.deleted_at IS NULL
  AND l.requires_international_settlement = true
  AND l.settlement_date = $1
ORDER BY d.created_at ASC
LIMIT $2 OFFSET $3;
