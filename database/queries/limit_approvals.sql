-- ============================================================================
-- limit_approvals.sql — Queries for limit_approval_records table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateLimitApproval :one
INSERT INTO limit_approval_records (
    id, deal_module, deal_id, counterparty_id, limit_type,
    deal_amount_vnd, limit_snapshot, approval_status,
    created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, 'PENDING', NOW()
)
RETURNING *;

-- name: GetLimitApprovalByID :one
SELECT * FROM limit_approval_records WHERE id = $1;

-- name: GetLimitApprovalByDealID :one
-- Lấy record phê duyệt hạn mức theo deal
SELECT * FROM limit_approval_records
WHERE deal_module = $1 AND deal_id = $2;

-- name: UpdateLimitApprovalStatus :one
-- Cập nhật trạng thái phê duyệt từ QLRR
UPDATE limit_approval_records SET
    risk_officer_approved_by = CASE
        WHEN sqlc.arg('approval_level')::TEXT = 'RISK_OFFICER' THEN sqlc.arg('user_id')::UUID
        ELSE risk_officer_approved_by
    END,
    risk_officer_approved_at = CASE
        WHEN sqlc.arg('approval_level')::TEXT = 'RISK_OFFICER' THEN NOW()
        ELSE risk_officer_approved_at
    END,
    risk_head_approved_by = CASE
        WHEN sqlc.arg('approval_level')::TEXT = 'RISK_HEAD' THEN sqlc.arg('user_id')::UUID
        ELSE risk_head_approved_by
    END,
    risk_head_approved_at = CASE
        WHEN sqlc.arg('approval_level')::TEXT = 'RISK_HEAD' THEN NOW()
        ELSE risk_head_approved_at
    END,
    approval_status = $1,
    rejection_reason = COALESCE(sqlc.narg('rejection_reason'), rejection_reason)
WHERE id = $2
RETURNING *;

-- name: ListPendingLimitApprovals :many
-- Lấy danh sách các deal chờ QLRR duyệt hạn mức
SELECT
    lar.*,
    u.full_name AS deal_creator,
    CASE lar.deal_module
        WHEN 'MM' THEN mmd.deal_number
        WHEN 'FX' THEN fxd.deal_number
    END AS deal_number
FROM limit_approval_records lar
JOIN users u ON
    CASE lar.deal_module
        WHEN 'MM' THEN (SELECT created_by FROM mm_interbank_deals WHERE id = lar.deal_id) = u.id
        WHEN 'FX' THEN (SELECT created_by FROM fx_deals WHERE id = lar.deal_id) = u.id
    END
LEFT JOIN mm_interbank_deals mmd ON lar.deal_module = 'MM' AND lar.deal_id = mmd.id
LEFT JOIN fx_deals fxd ON lar.deal_module = 'FX' AND lar.deal_id = fxd.id
WHERE lar.approval_status = 'PENDING'
ORDER BY lar.created_at ASC
LIMIT $1 OFFSET $2;
