-- ============================================================================
-- limit_snapshots.sql — Queries for limit_utilization_snapshots table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateLimitSnapshot :one
INSERT INTO limit_utilization_snapshots (
    id, counterparty_id, snapshot_date, limit_type,
    limit_granted, utilized_opening, utilized_intraday, utilized_total,
    remaining, fx_rate_applied, breakdown_detail,
    created_at, created_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    NOW(), $11
)
RETURNING *;

-- name: GetLatestSnapshotByCounterparty :one
-- Lấy snapshot mới nhất của đối tác
SELECT * FROM limit_utilization_snapshots
WHERE counterparty_id = $1 AND limit_type = $2
ORDER BY snapshot_date DESC
LIMIT 1;

-- name: GetSnapshotByCounterpartyAndDate :one
SELECT * FROM limit_utilization_snapshots
WHERE counterparty_id = $1
  AND limit_type = $2
  AND snapshot_date = $3;

-- name: UpsertLimitSnapshot :one
-- Tạo hoặc cập nhật snapshot cho ngày
INSERT INTO limit_utilization_snapshots (
    id, counterparty_id, snapshot_date, limit_type,
    limit_granted, utilized_opening, utilized_intraday, utilized_total,
    remaining, fx_rate_applied, breakdown_detail,
    created_at, created_by
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    NOW(), $11
)
ON CONFLICT (counterparty_id, snapshot_date, limit_type) DO UPDATE SET
    limit_granted = EXCLUDED.limit_granted,
    utilized_opening = EXCLUDED.utilized_opening,
    utilized_intraday = EXCLUDED.utilized_intraday,
    utilized_total = EXCLUDED.utilized_total,
    remaining = EXCLUDED.remaining,
    fx_rate_applied = EXCLUDED.fx_rate_applied,
    breakdown_detail = EXCLUDED.breakdown_detail,
    created_by = EXCLUDED.created_by
RETURNING *;

-- name: ListSnapshotsByCounterparty :many
-- Lấy lịch sử snapshot của đối tác
SELECT * FROM limit_utilization_snapshots
WHERE counterparty_id = $1
ORDER BY snapshot_date DESC
LIMIT $2 OFFSET $3;
