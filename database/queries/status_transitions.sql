-- ============================================================================
-- status_transitions.sql — Queries for status_transition_rules table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: GetValidTransitions :many
-- Lấy danh sách trạng thái đích hợp lệ từ trạng thái hiện tại
-- Dùng để hiển thị các action buttons trên UI
SELECT * FROM status_transition_rules
WHERE deal_module = $1
  AND from_status = $2
  AND is_active = true
ORDER BY to_status ASC;

-- name: ValidateTransition :one
-- Kiểm tra transition có hợp lệ không (from_status + to_status + role)
-- Return NULL nếu không hợp lệ → application deny action
SELECT * FROM status_transition_rules
WHERE deal_module = $1
  AND from_status = $2
  AND to_status = $3
  AND required_role = $4
  AND is_active = true;

-- name: GetValidTransitionsForRole :many
-- Lấy transitions mà role cụ thể được phép thực hiện
-- Dùng để filter UI buttons theo quyền user
SELECT * FROM status_transition_rules
WHERE deal_module = $1
  AND from_status = $2
  AND required_role = $3
  AND is_active = true
ORDER BY to_status ASC;

-- name: ListAllTransitionRules :many
-- Liệt kê toàn bộ rules — trang quản trị config
SELECT * FROM status_transition_rules
WHERE deal_module = $1
ORDER BY from_status ASC, to_status ASC;

-- name: CreateTransitionRule :one
-- Tạo rule mới — admin config
INSERT INTO status_transition_rules (
    id, deal_module, from_status, to_status,
    required_role, requires_reason, requires_confirmation, is_active
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, true
)
RETURNING *;

-- name: UpdateTransitionRule :one
-- Cập nhật rule — admin config
UPDATE status_transition_rules SET
    requires_reason = COALESCE(sqlc.narg('requires_reason'), requires_reason),
    requires_confirmation = COALESCE(sqlc.narg('requires_confirmation'), requires_confirmation),
    is_active = COALESCE(sqlc.narg('is_active'), is_active)
WHERE id = $1
RETURNING *;

-- name: DeactivateTransitionRule :exec
-- Vô hiệu hóa rule — soft disable
UPDATE status_transition_rules SET is_active = false
WHERE id = $1;

-- name: ListTransitionRulesByRole :many
-- Tất cả transitions mà role được phép — authorization matrix
SELECT * FROM status_transition_rules
WHERE required_role = $1 AND is_active = true
ORDER BY deal_module ASC, from_status ASC;
