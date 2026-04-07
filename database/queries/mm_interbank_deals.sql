-- ============================================================================
-- mm_interbank_deals.sql — Queries for mm_interbank_deals table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateMMInterbankDeal :one
INSERT INTO mm_interbank_deals (
    id, deal_number, ticket_number, counterparty_id, branch_id,
    currency_code, internal_ssi_id, counterparty_ssi_id,
    direction, principal_amount, interest_rate, day_count_convention,
    trade_date, effective_date, tenor_days, maturity_date,
    interest_amount, maturity_amount,
    has_collateral, collateral_currency, collateral_description,
    requires_international_settlement,
    status, note, cloned_from_id,
    created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,
    'OPEN', $22, $23,
    NOW(), $24, NOW(), $24
)
RETURNING *;

-- name: GetMMInterbankDealByID :one
SELECT * FROM mm_interbank_deals
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMMInterbankDeal :one
-- Chỉ cập nhật khi OPEN
UPDATE mm_interbank_deals SET
    ticket_number = COALESCE(sqlc.narg('ticket_number'), ticket_number),
    counterparty_id = COALESCE(sqlc.narg('counterparty_id'), counterparty_id),
    currency_code = COALESCE(sqlc.narg('currency_code'), currency_code),
    internal_ssi_id = COALESCE(sqlc.narg('internal_ssi_id'), internal_ssi_id),
    counterparty_ssi_id = COALESCE(sqlc.narg('counterparty_ssi_id'), counterparty_ssi_id),
    direction = COALESCE(sqlc.narg('direction'), direction),
    principal_amount = COALESCE(sqlc.narg('principal_amount'), principal_amount),
    interest_rate = COALESCE(sqlc.narg('interest_rate'), interest_rate),
    day_count_convention = COALESCE(sqlc.narg('day_count_convention'), day_count_convention),
    effective_date = COALESCE(sqlc.narg('effective_date'), effective_date),
    tenor_days = COALESCE(sqlc.narg('tenor_days'), tenor_days),
    maturity_date = COALESCE(sqlc.narg('maturity_date'), maturity_date),
    interest_amount = COALESCE(sqlc.narg('interest_amount'), interest_amount),
    maturity_amount = COALESCE(sqlc.narg('maturity_amount'), maturity_amount),
    has_collateral = COALESCE(sqlc.narg('has_collateral'), has_collateral),
    collateral_currency = COALESCE(sqlc.narg('collateral_currency'), collateral_currency),
    collateral_description = COALESCE(sqlc.narg('collateral_description'), collateral_description),
    requires_international_settlement = COALESCE(sqlc.narg('requires_international_settlement'), requires_international_settlement),
    note = COALESCE(sqlc.narg('note'), note),
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'
RETURNING *;

-- name: SoftDeleteMMInterbankDeal :exec
UPDATE mm_interbank_deals SET deleted_at = NOW(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL AND status = 'OPEN';

-- name: UpdateMMInterbankDealStatus :one
-- Optimistic status transition
UPDATE mm_interbank_deals SET
    status = $3,
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;

-- name: ListMMInterbankDeals :many
SELECT * FROM mm_interbank_deals
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountMMInterbankDeals :one
SELECT COUNT(*) FROM mm_interbank_deals WHERE deleted_at IS NULL;

-- name: ListMMInterbankDealsByStatus :many
SELECT * FROM mm_interbank_deals
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListMMInterbankDealsByMaturityDate :many
-- Giám sát đáo hạn — deal sắp đáo hạn trong khoảng ngày
SELECT * FROM mm_interbank_deals
WHERE maturity_date BETWEEN $1 AND $2
  AND deleted_at IS NULL
  AND status NOT IN ('CANCELLED', 'VOIDED_BY_RISK', 'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT')
ORDER BY maturity_date ASC
LIMIT $3 OFFSET $4;

-- name: ListMMInterbankPendingRiskApproval :many
-- View cho QLRR: deals chờ phê duyệt rủi ro
SELECT d.*, c.code AS counterparty_code, c.full_name AS counterparty_name
FROM mm_interbank_deals d
JOIN counterparties c ON c.id = d.counterparty_id
WHERE d.status = 'PENDING_RISK_APPROVAL'
  AND d.deleted_at IS NULL
ORDER BY d.trade_date ASC, d.created_at ASC
LIMIT $1 OFFSET $2;

-- name: ListMMInterbankPendingSettlement :many
-- View cho TTQT: deals cần thanh toán quốc tế
-- Logic: direction (PLACE/LEND → KLB chuyển tiền ra; TAKE/BORROW → nhận tiền vào)
-- Chỉ hiển thị deals có requires_international_settlement = true
SELECT d.*, c.code AS counterparty_code, c.full_name AS counterparty_name
FROM mm_interbank_deals d
JOIN counterparties c ON c.id = d.counterparty_id
WHERE d.status = 'PENDING_SETTLEMENT'
  AND d.requires_international_settlement = true
  AND d.deleted_at IS NULL
ORDER BY d.effective_date ASC
LIMIT $1 OFFSET $2;

-- Interest calculation formula (for reference — computed in application):
-- ACT_365: interest = principal × rate/100 × tenor_days / 365
-- ACT_360: interest = principal × rate/100 × tenor_days / 360
-- ACT_ACT: interest = principal × rate/100 × tenor_days / year_basis
--          (year_basis = 365 or 366, depending on whether period spans Feb 29)

-- name: SetMMInterbankCancelRequest :one
UPDATE mm_interbank_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'CANCELLED',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;
