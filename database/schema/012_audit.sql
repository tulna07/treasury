-- ============================================================================
-- 012_audit.sql — Audit Trail (1 table, partitioned)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 32: audit_logs — Nhật ký kiểm toán (append-only, partitioned by month)
-- ---------------------------------------------------------------------------
CREATE TABLE audit_logs (
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    user_full_name      VARCHAR(255) NOT NULL,
    user_department     VARCHAR(100) NULL,
    user_branch_code    VARCHAR(20) NULL,
    action              VARCHAR(50) NOT NULL,
    deal_module         VARCHAR(20) NOT NULL,
    deal_id             UUID NULL,
    status_before       VARCHAR(30) NULL,
    status_after        VARCHAR(30) NULL,
    old_values          JSONB NULL,
    new_values          JSONB NULL,
    reason              TEXT NULL,
    ip_address          INET NULL,
    user_agent          TEXT NULL,
    performed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_audit_logs_module CHECK (deal_module IN (
        'FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO',
        'CREDIT_LIMIT', 'INTERNATIONAL_PAYMENT', 'SYSTEM'
    ))
) PARTITION BY RANGE (performed_at);

-- Primary key must include partition key
ALTER TABLE audit_logs ADD CONSTRAINT pk_audit_logs PRIMARY KEY (id, performed_at);

-- Create partitions for 2026 (extend as needed)
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_logs_2026_02 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_logs_2026_03 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_logs_2026_05 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_logs_2026_06 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_logs_2026_07 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_logs_2026_08 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_logs_2026_09 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_logs_2026_10 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_logs_2026_11 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_logs_2026_12 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- Default partition for data outside defined ranges
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

-- Indexes on partitioned table (automatically created on each partition)
CREATE INDEX idx_audit_logs_deal ON audit_logs (deal_module, deal_id);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_performed_at ON audit_logs (performed_at);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);

COMMENT ON TABLE audit_logs IS 'Nhật ký kiểm toán — APPEND-ONLY, không sửa/xóa. Phân vùng theo tháng (BRD section 8)';
COMMENT ON COLUMN audit_logs.user_id IS 'ID người thực hiện — FK logic (không enforce để tránh ảnh hưởng partition)';
COMMENT ON COLUMN audit_logs.user_full_name IS 'Tên đầy đủ — snapshot tránh JOIN (dữ liệu kiểm toán phải tự đầy đủ)';
COMMENT ON COLUMN audit_logs.user_department IS 'Phòng ban — snapshot';
COMMENT ON COLUMN audit_logs.user_branch_code IS 'Mã chi nhánh — snapshot';
COMMENT ON COLUMN audit_logs.action IS '14 loại sự kiện theo BRD 8.1: CREATE, EDIT, APPROVE, REJECT, RECALL, CANCEL_REQUEST, CANCEL_APPROVE, CANCEL_REJECT, BOOK, SETTLE, LOGIN, LOGOUT, EXPORT, SYSTEM';
COMMENT ON COLUMN audit_logs.deal_module IS 'Module: FX, BOND, MM_INTERBANK, MM_OMO_REPO, CREDIT_LIMIT, INTERNATIONAL_PAYMENT, SYSTEM';
COMMENT ON COLUMN audit_logs.old_values IS 'Giá trị cũ (khi sửa) — JSONB diff';
COMMENT ON COLUMN audit_logs.new_values IS 'Giá trị mới (khi sửa) — JSONB diff';
COMMENT ON COLUMN audit_logs.reason IS 'Lý do — cho reject, recall, cancel';
COMMENT ON COLUMN audit_logs.ip_address IS 'IP client';
COMMENT ON COLUMN audit_logs.performed_at IS 'Thời điểm chính xác — partition key';
