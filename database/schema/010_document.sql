-- ============================================================================
-- 010_document.sql — Document Management (1 table)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 30: documents — Quản lý tài liệu (MinIO/S3 backend)
-- ---------------------------------------------------------------------------
CREATE TABLE documents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module         VARCHAR(20) NOT NULL,
    deal_id             UUID NOT NULL,
    document_type       VARCHAR(30) NOT NULL,
    file_name           VARCHAR(500) NOT NULL,
    storage_bucket      VARCHAR(100) NOT NULL,
    storage_key         VARCHAR(1000) NOT NULL,
    file_size           BIGINT NOT NULL,
    mime_type           VARCHAR(100) NOT NULL,
    checksum_sha256     VARCHAR(64) NOT NULL,
    scan_status         VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    scanned_at          TIMESTAMPTZ NULL,
    version             INT NOT NULL DEFAULT 1,
    is_current          BOOLEAN NOT NULL DEFAULT true,
    uploaded_by         UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    uploaded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ NULL,

    CONSTRAINT chk_documents_module CHECK (deal_module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO')),
    CONSTRAINT chk_documents_type CHECK (document_type IN ('TICKET', 'CONTRACT', 'CONFIRMATION', 'SUPPORTING', 'OTHER')),
    CONSTRAINT chk_documents_scan CHECK (scan_status IN ('PENDING', 'CLEAN', 'INFECTED', 'SKIPPED')),
    CONSTRAINT chk_documents_size CHECK (file_size > 0),
    CONSTRAINT chk_documents_version CHECK (version >= 1)
);

CREATE INDEX idx_documents_deal ON documents (deal_module, deal_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_documents_storage_key ON documents (storage_key);
CREATE INDEX idx_documents_scan ON documents (scan_status) WHERE scan_status = 'PENDING';

COMMENT ON TABLE documents IS 'Tài liệu đính kèm — lưu trữ MinIO/S3, bảng này chỉ chứa metadata';
COMMENT ON COLUMN documents.deal_module IS 'Module: FX, BOND, MM_INTERBANK, MM_OMO_REPO';
COMMENT ON COLUMN documents.document_type IS 'Loại: TICKET, CONTRACT, CONFIRMATION, SUPPORTING, OTHER';
COMMENT ON COLUMN documents.storage_bucket IS 'Tên bucket S3/MinIO: treasury-docs';
COMMENT ON COLUMN documents.storage_key IS 'S3 object key: {module}/{deal_id}/{uuid}_{filename}';
COMMENT ON COLUMN documents.checksum_sha256 IS 'SHA-256 hash — kiểm tra tính toàn vẹn';
COMMENT ON COLUMN documents.scan_status IS 'Trạng thái quét virus: PENDING, CLEAN, INFECTED, SKIPPED';
COMMENT ON COLUMN documents.version IS 'Phiên bản tài liệu — overwrite tạo version mới';
COMMENT ON COLUMN documents.is_current IS 'Phiên bản mới nhất — false cho các phiên bản cũ';
