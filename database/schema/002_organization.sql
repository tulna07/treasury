-- ============================================================================
-- 002_organization.sql — Tổ chức / Chi nhánh (1 table)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 9: branches — Chi nhánh / Phòng giao dịch
-- ---------------------------------------------------------------------------
CREATE TABLE branches (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  VARCHAR(20) NOT NULL,
    name                  VARCHAR(255) NOT NULL,
    branch_type           VARCHAR(20) NOT NULL,
    parent_branch_id      UUID NULL REFERENCES branches(id) ON DELETE RESTRICT,
    flexcube_branch_code  VARCHAR(20) NULL,
    swift_branch_code     VARCHAR(5) NULL,
    address               TEXT NULL,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by            UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT uq_branches_code UNIQUE (code),
    CONSTRAINT chk_branches_type CHECK (branch_type IN ('HEAD_OFFICE', 'BRANCH', 'SUB_BRANCH', 'TRANSACTION_OFFICE'))
);

CREATE INDEX idx_branches_code ON branches (code);
CREATE INDEX idx_branches_type ON branches (branch_type);
CREATE INDEX idx_branches_parent ON branches (parent_branch_id);

CREATE TRIGGER trg_branches_updated_at
    BEFORE UPDATE ON branches
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE branches IS 'Chi nhánh / Phòng giao dịch KienlongBank — Phase 1 chỉ có HO, sẵn sàng mở rộng';
COMMENT ON COLUMN branches.code IS 'Mã chi nhánh: HO (Hội sở), HCM01, HN01, KG01...';
COMMENT ON COLUMN branches.branch_type IS 'Loại: HEAD_OFFICE, BRANCH, SUB_BRANCH, TRANSACTION_OFFICE';
COMMENT ON COLUMN branches.parent_branch_id IS 'Chi nhánh cha — hỗ trợ cấu trúc phân cấp';
COMMENT ON COLUMN branches.flexcube_branch_code IS 'Mã chi nhánh Flexcube Core Banking — Phase 2 tích hợp';
COMMENT ON COLUMN branches.swift_branch_code IS 'Mã chi nhánh SWIFT (nếu có)';

-- ---------------------------------------------------------------------------
-- Add FK constraint from users.branch_id → branches
-- ---------------------------------------------------------------------------
ALTER TABLE users ADD CONSTRAINT fk_users_branch_id
    FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE SET NULL;
