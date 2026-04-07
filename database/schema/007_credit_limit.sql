-- ============================================================================
-- 007_credit_limit.sql — Module Credit Limit: Hạn mức tín dụng (3 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 23: credit_limits — Hạn mức tín dụng (SCD Type 2)
-- ---------------------------------------------------------------------------
CREATE TABLE credit_limits (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    counterparty_id     UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    limit_type          VARCHAR(20) NOT NULL,
    limit_amount        NUMERIC(20,2) NULL,
    is_unlimited        BOOLEAN NOT NULL DEFAULT false,
    effective_from      DATE NOT NULL,
    effective_to        DATE NULL,
    is_current          BOOLEAN NOT NULL DEFAULT true,
    expiry_date         DATE NULL,
    approval_reference  TEXT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT chk_credit_limits_type CHECK (limit_type IN ('COLLATERALIZED', 'UNCOLLATERALIZED')),
    CONSTRAINT chk_credit_limits_amount CHECK (limit_amount IS NULL OR limit_amount >= 0),
    CONSTRAINT chk_credit_limits_dates CHECK (effective_to IS NULL OR effective_to >= effective_from)
);

CREATE INDEX idx_credit_limits_counterparty ON credit_limits (counterparty_id);
CREATE INDEX idx_credit_limits_type ON credit_limits (limit_type);
CREATE INDEX idx_credit_limits_effective ON credit_limits (effective_from, effective_to);
CREATE INDEX idx_credit_limits_current ON credit_limits (counterparty_id, limit_type) WHERE is_current = true;

CREATE TRIGGER trg_credit_limits_updated_at
    BEFORE UPDATE ON credit_limits
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE credit_limits IS 'Hạn mức tín dụng — SCD Type 2: lưu lịch sử thay đổi, is_current = true cho bản ghi hiện tại';
COMMENT ON COLUMN credit_limits.limit_type IS 'Loại: COLLATERALIZED (có TSBĐ) hoặc UNCOLLATERALIZED (không có TSBĐ)';
COMMENT ON COLUMN credit_limits.limit_amount IS 'Hạn mức VND — NULL khi is_unlimited = true';
COMMENT ON COLUMN credit_limits.is_unlimited IS 'Hạn mức không giới hạn (theo phê duyệt TGĐ)';
COMMENT ON COLUMN credit_limits.is_current IS 'Bản ghi hiện tại — false cho các phiên bản cũ';
COMMENT ON COLUMN credit_limits.effective_from IS 'Ngày bắt đầu hiệu lực phiên bản này';
COMMENT ON COLUMN credit_limits.effective_to IS 'Ngày kết thúc hiệu lực — NULL cho phiên bản hiện tại';
COMMENT ON COLUMN credit_limits.approval_reference IS 'Tham chiếu phê duyệt TGĐ/HĐQT';

-- ---------------------------------------------------------------------------
-- Table 24: limit_utilization_snapshots — Snapshot sử dụng hạn mức
-- ---------------------------------------------------------------------------
CREATE TABLE limit_utilization_snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    counterparty_id     UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    snapshot_date       DATE NOT NULL,
    limit_type          VARCHAR(20) NOT NULL,
    limit_granted       NUMERIC(20,2) NULL,
    utilized_opening    NUMERIC(20,2) NOT NULL DEFAULT 0,
    utilized_intraday   NUMERIC(20,2) NOT NULL DEFAULT 0,
    utilized_total      NUMERIC(20,2) NOT NULL DEFAULT 0,
    remaining           NUMERIC(20,2) NULL,
    fx_rate_applied     NUMERIC(20,4) NULL,
    breakdown_detail    JSONB NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT chk_limit_snapshots_type CHECK (limit_type IN ('COLLATERALIZED', 'UNCOLLATERALIZED'))
);

CREATE INDEX idx_limit_snapshots_counterparty ON limit_utilization_snapshots (counterparty_id);
CREATE INDEX idx_limit_snapshots_date ON limit_utilization_snapshots (snapshot_date);
CREATE INDEX idx_limit_snapshots_cp_date ON limit_utilization_snapshots (counterparty_id, snapshot_date DESC);

COMMENT ON TABLE limit_utilization_snapshots IS 'Snapshot sử dụng hạn mức — append-only, không sửa đổi';
COMMENT ON COLUMN limit_utilization_snapshots.snapshot_date IS 'Ngày chụp snapshot';
COMMENT ON COLUMN limit_utilization_snapshots.limit_granted IS 'Hạn mức cấp — NULL = unlimited';
COMMENT ON COLUMN limit_utilization_snapshots.utilized_opening IS 'Sử dụng đầu ngày (VND quy đổi)';
COMMENT ON COLUMN limit_utilization_snapshots.utilized_intraday IS 'Sử dụng trong ngày (VND quy đổi)';
COMMENT ON COLUMN limit_utilization_snapshots.utilized_total IS 'Tổng sử dụng = opening + intraday';
COMMENT ON COLUMN limit_utilization_snapshots.remaining IS 'Còn lại — NULL = unlimited';
COMMENT ON COLUMN limit_utilization_snapshots.fx_rate_applied IS 'Tỷ giá USD→VND mid rate dùng quy đổi';
COMMENT ON COLUMN limit_utilization_snapshots.breakdown_detail IS 'Chi tiết phân tách: MM deals + FX deals + FI Bond settlement prices';

-- ---------------------------------------------------------------------------
-- Table 25: limit_approval_records — Phê duyệt hạn mức theo từng deal
-- ---------------------------------------------------------------------------
CREATE TABLE limit_approval_records (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module                 VARCHAR(10) NOT NULL,
    deal_id                     UUID NOT NULL,
    counterparty_id             UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    limit_type                  VARCHAR(20) NOT NULL,
    deal_amount_vnd             NUMERIC(20,2) NOT NULL,
    limit_snapshot              JSONB NOT NULL,
    risk_officer_approved_by    UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    risk_officer_approved_at    TIMESTAMPTZ NULL,
    risk_head_approved_by       UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    risk_head_approved_at       TIMESTAMPTZ NULL,
    approval_status             VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    rejection_reason            TEXT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_limit_approval_module CHECK (deal_module IN ('FX', 'MM')),
    CONSTRAINT chk_limit_approval_type CHECK (limit_type IN ('COLLATERALIZED', 'UNCOLLATERALIZED')),
    CONSTRAINT chk_limit_approval_status CHECK (approval_status IN ('PENDING', 'APPROVED', 'REJECTED'))
);

CREATE INDEX idx_limit_approval_deal ON limit_approval_records (deal_module, deal_id);
CREATE INDEX idx_limit_approval_counterparty ON limit_approval_records (counterparty_id);
CREATE INDEX idx_limit_approval_status ON limit_approval_records (approval_status);

COMMENT ON TABLE limit_approval_records IS 'Phê duyệt hạn mức per-deal — snapshot tại thời điểm duyệt (BRD 8.3)';
COMMENT ON COLUMN limit_approval_records.deal_module IS 'Module nguồn: FX hoặc MM';
COMMENT ON COLUMN limit_approval_records.deal_id IS 'ID deal nguồn (fx_deals.id hoặc mm_interbank_deals.id)';
COMMENT ON COLUMN limit_approval_records.deal_amount_vnd IS 'Giá trị deal quy VND tại thời điểm duyệt';
COMMENT ON COLUMN limit_approval_records.limit_snapshot IS 'Snapshot hạn mức: granted, utilized, remaining tại thời điểm duyệt';
COMMENT ON COLUMN limit_approval_records.approval_status IS 'Trạng thái: PENDING, APPROVED, REJECTED';
