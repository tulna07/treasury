-- ============================================================================
-- documents.sql — Queries for documents table
-- Treasury Management System — KienlongBank
-- ============================================================================

-- name: CreateDocument :one
INSERT INTO documents (
    id, deal_module, deal_id, document_type,
    file_name, storage_bucket, storage_key,
    file_size, mime_type, checksum_sha256,
    scan_status, version, is_current,
    uploaded_by, uploaded_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9,
    'PENDING', $10, true, $11, NOW()
)
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListDocumentsByDeal :many
-- Lấy tất cả tài liệu của deal — chỉ version hiện tại
SELECT
    d.*,
    u.full_name AS uploader_name
FROM documents d
JOIN users u ON u.id = d.uploaded_by
WHERE d.deal_module = $1
  AND d.deal_id = $2
  AND d.is_current = true
  AND d.deleted_at IS NULL
ORDER BY d.document_type ASC, d.uploaded_at DESC;

-- name: GetLatestDocumentVersion :one
-- Lấy phiên bản mới nhất của tài liệu theo tên file
SELECT * FROM documents
WHERE deal_module = $1
  AND deal_id = $2
  AND file_name = $3
  AND is_current = true
  AND deleted_at IS NULL;

-- name: MarkDocumentScanned :one
-- Cập nhật kết quả quét virus
UPDATE documents SET
    scan_status = $2,
    scanned_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDocument :exec
-- Xóa mềm tài liệu
UPDATE documents SET
    deleted_at = NOW(),
    is_current = false
WHERE id = $1 AND deleted_at IS NULL;

-- name: MarkOldVersions :exec
-- Đánh dấu các version cũ khi upload version mới
UPDATE documents SET is_current = false
WHERE deal_module = $1
  AND deal_id = $2
  AND file_name = $3
  AND is_current = true
  AND deleted_at IS NULL
  AND id != $4;

-- name: ListPendingScanDocuments :many
-- Lấy tài liệu chờ quét virus — antivirus worker
SELECT * FROM documents
WHERE scan_status = 'PENDING'
  AND deleted_at IS NULL
ORDER BY uploaded_at ASC
LIMIT $1;

-- name: ListDocumentVersions :many
-- Lấy tất cả version của một file — version history
SELECT
    d.*,
    u.full_name AS uploader_name
FROM documents d
JOIN users u ON u.id = d.uploaded_by
WHERE d.deal_module = $1
  AND d.deal_id = $2
  AND d.file_name = $3
  AND d.deleted_at IS NULL
ORDER BY d.version DESC;

-- name: GetDocumentByStorageKey :one
-- Tìm document theo storage key — dùng khi cần verify
SELECT * FROM documents
WHERE storage_key = $1 AND deleted_at IS NULL;
