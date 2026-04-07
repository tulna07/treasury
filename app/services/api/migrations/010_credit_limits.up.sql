-- ============================================================================
-- 010_credit_limits.up.sql — Credit Limit Module (BRD §3.4)
-- Hạn mức liên ngân hàng: COLLATERALIZED (có TSBĐ) / UNCOLLATERALIZED (không TSBĐ)
-- SCD Type 2 for history, utilization snapshots, per-deal approval records
-- ============================================================================

-- Update permissions resource constraint to include AUDIT_LOG
ALTER TABLE permissions DROP CONSTRAINT IF EXISTS chk_permissions_resource;
ALTER TABLE permissions ADD CONSTRAINT chk_permissions_resource CHECK (resource IN (
    'FX_DEAL', 'BOND_DEAL', 'MM_INTERBANK_DEAL', 'MM_OMO_REPO_DEAL',
    'CREDIT_LIMIT', 'INTERNATIONAL_PAYMENT', 'MASTER_DATA', 'SYSTEM', 'AUDIT_LOG'
));

-- ---------------------------------------------------------------------------
-- Table: credit_limits — Hạn mức tín dụng (SCD Type 2)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS credit_limits (
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

CREATE INDEX IF NOT EXISTS idx_credit_limits_counterparty ON credit_limits (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_credit_limits_type ON credit_limits (limit_type);
CREATE INDEX IF NOT EXISTS idx_credit_limits_effective ON credit_limits (effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_credit_limits_current ON credit_limits (counterparty_id, limit_type) WHERE is_current = true;

CREATE TRIGGER trg_credit_limits_updated_at
    BEFORE UPDATE ON credit_limits
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE credit_limits IS 'Hạn mức tín dụng liên ngân hàng — SCD Type 2: is_current = true cho bản ghi hiện tại';
COMMENT ON COLUMN credit_limits.limit_type IS 'COLLATERALIZED (có TSBĐ) hoặc UNCOLLATERALIZED (không TSBĐ)';
COMMENT ON COLUMN credit_limits.limit_amount IS 'Hạn mức VND — NULL khi is_unlimited = true';
COMMENT ON COLUMN credit_limits.is_unlimited IS 'Hạn mức không giới hạn (theo phê duyệt TGĐ)';
COMMENT ON COLUMN credit_limits.is_current IS 'Bản ghi hiện tại — false cho phiên bản cũ';

-- ---------------------------------------------------------------------------
-- Table: limit_utilization_snapshots — Snapshot sử dụng hạn mức (append-only)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS limit_utilization_snapshots (
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

CREATE INDEX IF NOT EXISTS idx_limit_snapshots_counterparty ON limit_utilization_snapshots (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_limit_snapshots_date ON limit_utilization_snapshots (snapshot_date);
CREATE INDEX IF NOT EXISTS idx_limit_snapshots_cp_date ON limit_utilization_snapshots (counterparty_id, snapshot_date DESC);

COMMENT ON TABLE limit_utilization_snapshots IS 'Snapshot sử dụng hạn mức — append-only, không sửa đổi';

-- ---------------------------------------------------------------------------
-- Table: limit_approval_records — Phê duyệt hạn mức per-deal
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS limit_approval_records (
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

    CONSTRAINT chk_limit_approval_module CHECK (deal_module IN ('FX', 'MM', 'BOND')),
    CONSTRAINT chk_limit_approval_type CHECK (limit_type IN ('COLLATERALIZED', 'UNCOLLATERALIZED')),
    CONSTRAINT chk_limit_approval_status CHECK (approval_status IN ('PENDING', 'RISK_L1_APPROVED', 'APPROVED', 'REJECTED'))
);

CREATE INDEX IF NOT EXISTS idx_limit_approval_deal ON limit_approval_records (deal_module, deal_id);
CREATE INDEX IF NOT EXISTS idx_limit_approval_counterparty ON limit_approval_records (counterparty_id);
CREATE INDEX IF NOT EXISTS idx_limit_approval_status ON limit_approval_records (approval_status);

COMMENT ON TABLE limit_approval_records IS 'Phê duyệt hạn mức per-deal — CV QLRR → TPB QLRR (2-level)';

-- ---------------------------------------------------------------------------
-- View: v_daily_limit_summary — Bảng tổng hợp hạn mức hàng ngày (11 columns)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_daily_limit_summary AS
SELECT
    c.id AS counterparty_id,
    c.short_name AS counterparty_name,
    c.cif,
    -- (1) Hạn mức cấp có TSBĐ
    coll.limit_amount AS allocated_collateralized,
    coll.is_unlimited AS is_unlimited_collateralized,
    -- (2) Hạn mức cấp không TSBĐ
    uncoll.limit_amount AS allocated_uncollateralized,
    uncoll.is_unlimited AS is_unlimited_uncollateralized,
    -- Snapshot data will be joined at query time for:
    -- (3) Đã sử dụng đầu ngày có TSBĐ
    -- (4) Sử dụng trong ngày có TSBĐ
    -- (5) Còn lại có TSBĐ = (1) - (3) - (4)
    -- (6) Đã sử dụng đầu ngày không TSBĐ
    -- (7) Sử dụng trong ngày không TSBĐ
    -- (8) Còn lại không TSBĐ = (2) - (6) - (7)
    coll.effective_from AS coll_effective_from,
    uncoll.effective_from AS uncoll_effective_from,
    coll.approval_reference AS coll_approval_reference,
    uncoll.approval_reference AS uncoll_approval_reference
FROM counterparties c
LEFT JOIN credit_limits coll
    ON coll.counterparty_id = c.id
    AND coll.limit_type = 'COLLATERALIZED'
    AND coll.is_current = true
LEFT JOIN credit_limits uncoll
    ON uncoll.counterparty_id = c.id
    AND uncoll.limit_type = 'UNCOLLATERALIZED'
    AND uncoll.is_current = true
WHERE c.deleted_at IS NULL
    AND c.is_active = true
    AND (coll.id IS NOT NULL OR uncoll.id IS NOT NULL);
