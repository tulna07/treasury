-- ============================================================================
-- fx_deal_legs.sql — Queries for fx_deal_legs table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateFxDealLeg :one
INSERT INTO fx_deal_legs (
    id, deal_id, leg_number, value_date, settlement_date,
    exchange_rate, converted_amount, converted_currency,
    internal_ssi_id, counterparty_ssi_id,
    requires_international_settlement,
    created_at, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    NOW(), NOW(), $11
)
RETURNING *;

-- name: GetFxDealLegByID :one
SELECT * FROM fx_deal_legs WHERE id = $1;

-- name: GetFxDealLegsByDealID :many
-- Lấy tất cả legs của một deal (1 for Spot/Fwd, 2 for Swap)
SELECT * FROM fx_deal_legs
WHERE deal_id = $1
ORDER BY leg_number ASC;

-- name: GetFxDealLegByDealIDAndLeg :one
-- Lấy leg cụ thể của deal (near=1, far=2)
SELECT * FROM fx_deal_legs
WHERE deal_id = $1 AND leg_number = $2;

-- name: UpdateFxDealLeg :one
UPDATE fx_deal_legs SET
    value_date = COALESCE(sqlc.narg('value_date'), value_date),
    settlement_date = COALESCE(sqlc.narg('settlement_date'), settlement_date),
    exchange_rate = COALESCE(sqlc.narg('exchange_rate'), exchange_rate),
    converted_amount = COALESCE(sqlc.narg('converted_amount'), converted_amount),
    converted_currency = COALESCE(sqlc.narg('converted_currency'), converted_currency),
    internal_ssi_id = COALESCE(sqlc.narg('internal_ssi_id'), internal_ssi_id),
    counterparty_ssi_id = COALESCE(sqlc.narg('counterparty_ssi_id'), counterparty_ssi_id),
    requires_international_settlement = COALESCE(sqlc.narg('requires_international_settlement'), requires_international_settlement),
    updated_by = $1
WHERE id = $2
RETURNING *;

-- name: DeleteFxDealLegsByDealID :exec
-- Xóa tất cả legs — dùng khi thay đổi deal type (e.g. Swap → Spot)
DELETE FROM fx_deal_legs WHERE deal_id = $1;

-- name: ListLegsRequiringSettlement :many
-- Legs cần thanh toán quốc tế theo ngày settlement — BP.TTQT xử lý
SELECT
    l.*,
    d.deal_number,
    d.deal_type,
    d.direction,
    d.counterparty_id,
    d.notional_amount,
    d.currency_code
FROM fx_deal_legs l
JOIN fx_deals d ON d.id = l.deal_id
WHERE l.requires_international_settlement = true
  AND l.settlement_date = $1
  AND d.status = 'PENDING_SETTLEMENT'
  AND d.deleted_at IS NULL
ORDER BY d.created_at ASC;
