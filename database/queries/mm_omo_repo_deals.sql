-- ============================================================================
-- mm_omo_repo_deals.sql — Queries for mm_omo_repo_deals table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateMMOmoRepoDeal :one
INSERT INTO mm_omo_repo_deals (
    id, deal_number, deal_subtype, session_name, trade_date, branch_id,
    counterparty_id, notional_amount, bond_catalog_id,
    winning_rate, tenor_days, settlement_date_1, settlement_date_2,
    haircut_pct, status, note, cloned_from_id,
    created_at, created_by, updated_at, updated_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
    $13, 'OPEN', $14, $15,
    NOW(), $16, NOW(), $16
)
RETURNING *;

-- name: GetMMOmoRepoDealByID :one
SELECT * FROM mm_omo_repo_deals
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateMMOmoRepoDeal :one
UPDATE mm_omo_repo_deals SET
    session_name = COALESCE(sqlc.narg('session_name'), session_name),
    counterparty_id = COALESCE(sqlc.narg('counterparty_id'), counterparty_id),
    notional_amount = COALESCE(sqlc.narg('notional_amount'), notional_amount),
    bond_catalog_id = COALESCE(sqlc.narg('bond_catalog_id'), bond_catalog_id),
    winning_rate = COALESCE(sqlc.narg('winning_rate'), winning_rate),
    tenor_days = COALESCE(sqlc.narg('tenor_days'), tenor_days),
    settlement_date_1 = COALESCE(sqlc.narg('settlement_date_1'), settlement_date_1),
    settlement_date_2 = COALESCE(sqlc.narg('settlement_date_2'), settlement_date_2),
    haircut_pct = COALESCE(sqlc.narg('haircut_pct'), haircut_pct),
    note = COALESCE(sqlc.narg('note'), note),
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'
RETURNING *;

-- name: SoftDeleteMMOmoRepoDeal :exec
UPDATE mm_omo_repo_deals SET deleted_at = NOW(), updated_by = $2
WHERE id = $1 AND deleted_at IS NULL AND status = 'OPEN';

-- name: UpdateMMOmoRepoDealStatus :one
UPDATE mm_omo_repo_deals SET
    status = $3,
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;

-- name: ListMMOmoRepoDeals :many
SELECT * FROM mm_omo_repo_deals
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountMMOmoRepoDeals :one
SELECT COUNT(*) FROM mm_omo_repo_deals WHERE deleted_at IS NULL;

-- name: ListMMOmoRepoDealsBySubtype :many
-- Lọc theo loại: OMO (với NHNN) hoặc STATE_REPO (với KBNN)
SELECT * FROM mm_omo_repo_deals
WHERE deal_subtype = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListMMOmoRepoDealsByStatus :many
SELECT * FROM mm_omo_repo_deals
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListMMOmoRepoPendingBooking :many
-- View cho KTTC
SELECT * FROM mm_omo_repo_deals
WHERE status IN ('PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT')
  AND deleted_at IS NULL
ORDER BY trade_date ASC, created_at ASC
LIMIT $1 OFFSET $2;

-- name: SetMMOmoRepoCancelRequest :one
UPDATE mm_omo_repo_deals SET
    cancel_reason = $3,
    cancel_requested_by = $1,
    cancel_requested_at = NOW(),
    status = 'CANCELLED',
    updated_by = $1
WHERE id = $2 AND deleted_at IS NULL AND status = $4
RETURNING *;
