-- ============================================================================
-- approval_actions.sql — Queries for approval_actions table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateApprovalAction :one
INSERT INTO approval_actions (
    id, deal_module, deal_id, action_type,
    status_before, status_after, performed_by,
    performed_at, reason, metadata
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, NOW(), $7, $8
)
RETURNING *;

-- name: ListApprovalActionsByDeal :many
-- Lịch sử phê duyệt theo deal — timeline hiển thị trên UI
SELECT
    aa.*,
    u.full_name AS performer_name,
    u.department AS performer_department
FROM approval_actions aa
JOIN users u ON u.id = aa.performed_by
WHERE aa.deal_module = $1 AND aa.deal_id = $2
ORDER BY aa.performed_at ASC;

-- name: GetLatestApprovalActionByDeal :one
-- Hành động phê duyệt gần nhất — xác định trạng thái hiện tại
SELECT
    aa.*,
    u.full_name AS performer_name
FROM approval_actions aa
JOIN users u ON u.id = aa.performed_by
WHERE aa.deal_module = $1 AND aa.deal_id = $2
ORDER BY aa.performed_at DESC
LIMIT 1;

-- name: ListPendingApprovalsByRole :many
-- Danh sách deal đang chờ role cụ thể phê duyệt
-- Join status_transition_rules để biết role nào cần duyệt tiếp
SELECT DISTINCT
    aa.deal_module,
    aa.deal_id,
    aa.status_after AS current_status,
    aa.performed_at AS last_action_at,
    str.to_status AS expected_next_status,
    str.required_role
FROM approval_actions aa
JOIN status_transition_rules str ON
    str.deal_module = aa.deal_module
    AND str.from_status = aa.status_after
    AND str.is_active = true
WHERE str.required_role = $1
  AND aa.performed_at = (
      SELECT MAX(aa2.performed_at)
      FROM approval_actions aa2
      WHERE aa2.deal_module = aa.deal_module
        AND aa2.deal_id = aa.deal_id
  )
ORDER BY aa.performed_at ASC
LIMIT $2 OFFSET $3;

-- name: CountApprovalActionsByDeal :one
-- Đếm số hành động phê duyệt theo deal
SELECT COUNT(*) FROM approval_actions
WHERE deal_module = $1 AND deal_id = $2;

-- name: ListApprovalActionsByUser :many
-- Lịch sử phê duyệt của user — báo cáo cá nhân
SELECT * FROM approval_actions
WHERE performed_by = $1
ORDER BY performed_at DESC
LIMIT $2 OFFSET $3;

-- name: ListApprovalActionsByDateRange :many
-- Lịch sử phê duyệt theo khoảng thời gian — báo cáo quản lý
SELECT
    aa.*,
    u.full_name AS performer_name
FROM approval_actions aa
JOIN users u ON u.id = aa.performed_by
WHERE aa.performed_at BETWEEN $1 AND $2
ORDER BY aa.performed_at DESC
LIMIT $3 OFFSET $4;
