-- ============================================================================
-- audit_logs.sql — Queries for audit_logs table (partitioned)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateAuditLog :one
-- Ghi nhật ký kiểm toán — APPEND-ONLY, không sửa/xóa
INSERT INTO audit_logs (
    id, user_id, user_full_name, user_department, user_branch_code,
    action, deal_module, deal_id,
    status_before, status_after,
    old_values, new_values, reason,
    ip_address, user_agent, performed_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, NOW()
)
RETURNING *;

-- name: ListAuditLogsByDeal :many
-- Lịch sử kiểm toán theo deal — audit trail trên deal detail
SELECT * FROM audit_logs
WHERE deal_module = $1 AND deal_id = $2
ORDER BY performed_at ASC;

-- name: ListAuditLogsByUser :many
-- Lịch sử thao tác của user — user activity report
SELECT * FROM audit_logs
WHERE user_id = $1
ORDER BY performed_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditLogsByDateRange :many
-- Nhật ký theo khoảng thời gian — báo cáo kiểm toán định kỳ
-- Partition pruning tự động khi filter theo performed_at
SELECT * FROM audit_logs
WHERE performed_at BETWEEN $1 AND $2
ORDER BY performed_at DESC
LIMIT $3 OFFSET $4;

-- name: SearchAuditLogsByAction :many
-- Tìm theo loại hành động — vd: tất cả REJECT trong tháng
SELECT * FROM audit_logs
WHERE action = $1
  AND performed_at BETWEEN $2 AND $3
ORDER BY performed_at DESC
LIMIT $4 OFFSET $5;

-- name: ListAuditLogsByModule :many
-- Lịch sử theo module — báo cáo theo nghiệp vụ
SELECT * FROM audit_logs
WHERE deal_module = $1
  AND performed_at BETWEEN $2 AND $3
ORDER BY performed_at DESC
LIMIT $4 OFFSET $5;

-- name: CountAuditLogsByAction :many
-- Thống kê số lượng theo action — dashboard chart
SELECT action, COUNT(*) AS total
FROM audit_logs
WHERE performed_at BETWEEN $1 AND $2
GROUP BY action
ORDER BY total DESC;

-- name: ListAuditLogsByUserAndModule :many
-- Lịch sử user trên module cụ thể — compliance check
SELECT * FROM audit_logs
WHERE user_id = $1
  AND deal_module = $2
  AND performed_at BETWEEN $3 AND $4
ORDER BY performed_at DESC
LIMIT $5 OFFSET $6;

-- name: GetAuditLogByID :one
SELECT * FROM audit_logs
WHERE id = $1 AND performed_at = $2;
