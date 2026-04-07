-- ============================================================================
-- 006_money_market.sql — Module Money Market: Thị trường tiền tệ (2 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 21: mm_interbank_deals — Giao dịch liên ngân hàng
-- ---------------------------------------------------------------------------
CREATE TABLE mm_interbank_deals (
    id                                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_number                         VARCHAR(30) NOT NULL,
    ticket_number                       VARCHAR(20) NULL,
    counterparty_id                     UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    branch_id                           UUID NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    currency_code                       VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    internal_ssi_id                     UUID NOT NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    counterparty_ssi_id                 UUID NOT NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    direction                           VARCHAR(20) NOT NULL,
    principal_amount                    NUMERIC(20,2) NOT NULL,
    interest_rate                       NUMERIC(10,6) NOT NULL,
    day_count_convention                VARCHAR(15) NOT NULL,
    trade_date                          DATE NOT NULL,
    effective_date                      DATE NOT NULL,
    tenor_days                          INT NOT NULL,
    maturity_date                       DATE NOT NULL,
    interest_amount                     NUMERIC(20,2) NOT NULL,
    maturity_amount                     NUMERIC(20,2) NOT NULL,
    has_collateral                      BOOLEAN NOT NULL DEFAULT false,
    collateral_currency                 VARCHAR(3) NULL,
    collateral_description              TEXT NULL,
    requires_international_settlement   BOOLEAN NOT NULL DEFAULT false,
    status                              VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    note                                TEXT NULL,
    cloned_from_id                      UUID NULL REFERENCES mm_interbank_deals(id) ON DELETE SET NULL,
    cancel_reason                       TEXT NULL,
    cancel_requested_by                 UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    cancel_requested_at                 TIMESTAMPTZ NULL,
    created_at                          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                          UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at                          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                          UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at                          TIMESTAMPTZ NULL,

    CONSTRAINT uq_mm_interbank_deals_number UNIQUE (deal_number),
    CONSTRAINT chk_mm_interbank_direction CHECK (direction IN ('PLACE', 'TAKE', 'LEND', 'BORROW')),
    CONSTRAINT chk_mm_interbank_principal CHECK (principal_amount > 0),
    CONSTRAINT chk_mm_interbank_rate CHECK (interest_rate > 0),
    CONSTRAINT chk_mm_interbank_tenor CHECK (tenor_days > 0),
    CONSTRAINT chk_mm_interbank_day_count CHECK (day_count_convention IN ('ACT_365', 'ACT_360', 'ACT_ACT')),
    CONSTRAINT chk_mm_interbank_status CHECK (status IN (
        'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
        'PENDING_RISK_APPROVAL',
        'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
        'PENDING_SETTLEMENT', 'COMPLETED',
        'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT', 'VOIDED_BY_RISK',
        'CANCELLED'
    ))
);

CREATE INDEX idx_mm_interbank_status ON mm_interbank_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_trade_date ON mm_interbank_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_counterparty ON mm_interbank_deals (counterparty_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_maturity ON mm_interbank_deals (maturity_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_direction ON mm_interbank_deals (direction) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_mm_interbank_deals_updated_at
    BEFORE UPDATE ON mm_interbank_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE mm_interbank_deals IS 'Giao dịch liên ngân hàng — gửi/nhận tiền gửi, cho vay/đi vay';
COMMENT ON COLUMN mm_interbank_deals.deal_number IS 'Mã giao dịch gapless: MM-20260403-0001';
COMMENT ON COLUMN mm_interbank_deals.direction IS 'Hướng: PLACE (Gửi TG), TAKE (Nhận TG), LEND (Cho vay), BORROW (Đi vay)';
COMMENT ON COLUMN mm_interbank_deals.principal_amount IS 'Số tiền gốc (FCY: 2 chữ số)';
COMMENT ON COLUMN mm_interbank_deals.interest_rate IS 'Lãi suất (%/năm, 6 chữ số thập phân)';
COMMENT ON COLUMN mm_interbank_deals.day_count_convention IS 'Quy ước tính ngày: ACT_365, ACT_360, ACT_ACT';
COMMENT ON COLUMN mm_interbank_deals.interest_amount IS 'Tiền lãi = principal × rate × tenor / day_basis';
COMMENT ON COLUMN mm_interbank_deals.maturity_amount IS 'Số tiền đáo hạn = principal + interest';
COMMENT ON COLUMN mm_interbank_deals.has_collateral IS 'Có tài sản đảm bảo không — ảnh hưởng tính credit limit';
COMMENT ON COLUMN mm_interbank_deals.requires_international_settlement IS 'Cần thanh toán quốc tế (tạo bản ghi international_payments)';
COMMENT ON COLUMN mm_interbank_deals.status IS 'Trạng thái — bao gồm PENDING_RISK_APPROVAL cho QLRR';

-- ---------------------------------------------------------------------------
-- Table 22: mm_omo_repo_deals — Giao dịch OMO / Repo Kho bạc
-- ---------------------------------------------------------------------------
CREATE TABLE mm_omo_repo_deals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_number           VARCHAR(30) NOT NULL,
    deal_subtype          VARCHAR(15) NOT NULL,
    session_name          VARCHAR(100) NOT NULL,
    trade_date            DATE NOT NULL,
    branch_id             UUID NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    counterparty_id       UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    notional_amount       NUMERIC(20,0) NOT NULL,
    bond_catalog_id       UUID NOT NULL REFERENCES bond_catalog(id) ON DELETE RESTRICT,
    winning_rate          NUMERIC(10,6) NOT NULL,
    tenor_days            INT NOT NULL,
    settlement_date_1     DATE NOT NULL,
    settlement_date_2     DATE NOT NULL,
    haircut_pct           NUMERIC(5,2) NOT NULL DEFAULT 0,
    status                VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    note                  TEXT NULL,
    cloned_from_id        UUID NULL REFERENCES mm_omo_repo_deals(id) ON DELETE SET NULL,
    cancel_reason         TEXT NULL,
    cancel_requested_by   UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    cancel_requested_at   TIMESTAMPTZ NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at            TIMESTAMPTZ NULL,

    CONSTRAINT uq_mm_omo_repo_number UNIQUE (deal_number),
    CONSTRAINT chk_mm_omo_repo_subtype CHECK (deal_subtype IN ('OMO', 'STATE_REPO')),
    CONSTRAINT chk_mm_omo_repo_tenor CHECK (tenor_days > 0),
    CONSTRAINT chk_mm_omo_repo_haircut CHECK (haircut_pct >= 0),
    CONSTRAINT chk_mm_omo_repo_amount CHECK (notional_amount > 0),
    CONSTRAINT chk_mm_omo_repo_status CHECK (status IN (
        'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
        'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
        'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'CANCELLED'
    ))
);

CREATE INDEX idx_mm_omo_repo_status ON mm_omo_repo_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_omo_repo_trade_date ON mm_omo_repo_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_omo_repo_subtype ON mm_omo_repo_deals (deal_subtype) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_mm_omo_repo_deals_updated_at
    BEFORE UPDATE ON mm_omo_repo_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE mm_omo_repo_deals IS 'Giao dịch OMO (Ngân hàng Nhà nước) và Repo Kho bạc Nhà nước';
COMMENT ON COLUMN mm_omo_repo_deals.deal_subtype IS 'Loại: OMO (với NHNN) hoặc STATE_REPO (với KBNN)';
COMMENT ON COLUMN mm_omo_repo_deals.session_name IS 'Tên phiên giao dịch: Session 1, Session 2...';
COMMENT ON COLUMN mm_omo_repo_deals.notional_amount IS 'Mệnh giá giao dịch VND (NUMERIC(20,0))';
COMMENT ON COLUMN mm_omo_repo_deals.winning_rate IS 'Lãi suất trúng thầu (%/năm)';
COMMENT ON COLUMN mm_omo_repo_deals.settlement_date_1 IS 'Ngày thanh toán lần 1 (mua/bán)';
COMMENT ON COLUMN mm_omo_repo_deals.settlement_date_2 IS 'Ngày thanh toán lần 2 (mua lại/bán lại)';
COMMENT ON COLUMN mm_omo_repo_deals.haircut_pct IS 'Tỷ lệ chiết khấu (%)';
