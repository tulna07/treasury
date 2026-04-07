-- ============================================================================
-- deal_sequences.sql — Queries for deal_sequences table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: GetNextSequence :one
-- Lấy và tăng sequence number — PHẢI chạy trong transaction
-- Pattern: SELECT FOR UPDATE → UPDATE → return new value
-- Tạo row mới nếu chưa tồn tại (first deal of the day)
INSERT INTO deal_sequences (id, module, prefix, date_partition, last_sequence, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, 1, NOW())
ON CONFLICT (module, prefix, date_partition)
DO UPDATE SET
    last_sequence = deal_sequences.last_sequence + 1,
    updated_at = NOW()
RETURNING *;

-- name: GetCurrentSequence :one
-- Xem sequence hiện tại mà không tăng — dùng cho báo cáo/monitoring
SELECT * FROM deal_sequences
WHERE module = $1
  AND prefix = $2
  AND date_partition = $3;

-- name: ListSequencesByDate :many
-- Liệt kê tất cả sequence trong ngày — monitoring dashboard
SELECT * FROM deal_sequences
WHERE date_partition = $1
ORDER BY module ASC, prefix ASC;

-- name: ListSequencesByModule :many
-- Lịch sử sequence theo module — dùng cho báo cáo
SELECT * FROM deal_sequences
WHERE module = $1
ORDER BY date_partition DESC
LIMIT $2 OFFSET $3;
