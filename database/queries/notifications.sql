-- ============================================================================
-- notifications.sql — Queries for notifications table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateNotification :one
INSERT INTO notifications (
    id, recipient_id, channel, event_type,
    title, body, deal_module, deal_id,
    is_read, delivery_status, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7,
    false, 'PENDING', NOW()
)
RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM notifications WHERE id = $1;

-- name: ListUnreadByUser :many
-- Lấy thông báo chưa đọc — bell icon + dropdown
SELECT * FROM notifications
WHERE recipient_id = $1
  AND is_read = false
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnreadByUser :one
-- Đếm thông báo chưa đọc — badge count
SELECT COUNT(*) FROM notifications
WHERE recipient_id = $1 AND is_read = false;

-- name: MarkAsRead :exec
-- Đánh dấu đã đọc khi user click
UPDATE notifications SET
    is_read = true,
    read_at = NOW()
WHERE id = $1 AND recipient_id = $2 AND is_read = false;

-- name: MarkAllAsRead :exec
-- Đánh dấu tất cả đã đọc — "Mark all as read" button
UPDATE notifications SET
    is_read = true,
    read_at = NOW()
WHERE recipient_id = $1 AND is_read = false;

-- name: ListPendingDelivery :many
-- Lấy notifications chờ gửi — email worker
SELECT * FROM notifications
WHERE delivery_status IN ('PENDING', 'RETRYING')
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- name: UpdateDeliveryStatus :exec
-- Cập nhật trạng thái gửi — email worker callback
UPDATE notifications SET
    delivery_status = $2,
    sent_at = CASE WHEN $2 = 'SENT' THEN NOW() ELSE sent_at END,
    retry_count = CASE WHEN $2 = 'RETRYING' THEN retry_count + 1 ELSE retry_count END,
    last_error = $3,
    next_retry_at = $4
WHERE id = $1;

-- name: ListNotificationsByUser :many
-- Lấy tất cả thông báo (đã đọc + chưa đọc) — notification center
SELECT * FROM notifications
WHERE recipient_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListNotificationsByDeal :many
-- Lấy thông báo liên quan đến deal — deal detail page
SELECT
    n.*,
    u.full_name AS recipient_name
FROM notifications n
JOIN users u ON u.id = n.recipient_id
WHERE n.deal_module = $1 AND n.deal_id = $2
ORDER BY n.created_at DESC;

-- name: DeleteOldNotifications :exec
-- Xóa thông báo cũ hơn N ngày — cleanup job
DELETE FROM notifications
WHERE created_at < NOW() - INTERVAL '1 day' * sqlc.arg('retention_days')::INT
  AND is_read = true;
