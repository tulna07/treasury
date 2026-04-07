-- ============================================================================
-- 005_bond.sql — Module Bond: Trái phiếu / GTCG (2 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 19: bond_deals — Giao dịch trái phiếu / GTCG
-- ---------------------------------------------------------------------------
CREATE TABLE bond_deals (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_number             VARCHAR(30) NOT NULL,
    bond_category           VARCHAR(30) NOT NULL,
    trade_date              DATE NOT NULL,
    branch_id               UUID NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    order_date              DATE NULL,
    value_date              DATE NOT NULL,
    direction               VARCHAR(5) NOT NULL,
    counterparty_id         UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    transaction_type        VARCHAR(20) NOT NULL,
    transaction_type_other  VARCHAR(255) NULL,
    bond_catalog_id         UUID NULL REFERENCES bond_catalog(id) ON DELETE RESTRICT,
    bond_code_manual        VARCHAR(50) NULL,
    issuer                  VARCHAR(500) NOT NULL,
    coupon_rate             NUMERIC(10,4) NOT NULL,
    issue_date              DATE NULL,
    maturity_date           DATE NOT NULL,
    quantity                BIGINT NOT NULL,
    face_value              NUMERIC(20,0) NOT NULL,
    discount_rate           NUMERIC(10,4) NOT NULL DEFAULT 0,
    clean_price             NUMERIC(20,0) NOT NULL,
    settlement_price        NUMERIC(20,0) NOT NULL,
    total_value             NUMERIC(20,0) NOT NULL,
    portfolio_type          VARCHAR(5) NULL,
    payment_date            DATE NOT NULL,
    remaining_tenor_days    INT NOT NULL,
    confirmation_method     VARCHAR(20) NOT NULL,
    confirmation_other      VARCHAR(255) NULL,
    contract_prepared_by    VARCHAR(15) NOT NULL,
    status                  VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    note                    TEXT NULL,
    cloned_from_id          UUID NULL REFERENCES bond_deals(id) ON DELETE SET NULL,
    cancel_reason           TEXT NULL,
    cancel_requested_by     UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    cancel_requested_at     TIMESTAMPTZ NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by              UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at              TIMESTAMPTZ NULL,

    CONSTRAINT uq_bond_deals_number UNIQUE (deal_number),
    CONSTRAINT chk_bond_deals_category CHECK (bond_category IN ('GOVERNMENT', 'FINANCIAL_INSTITUTION', 'CERTIFICATE_OF_DEPOSIT')),
    CONSTRAINT chk_bond_deals_direction CHECK (direction IN ('BUY', 'SELL')),
    CONSTRAINT chk_bond_deals_tx_type CHECK (transaction_type IN ('REPO', 'REVERSE_REPO', 'OUTRIGHT', 'OTHER')),
    CONSTRAINT chk_bond_deals_quantity CHECK (quantity > 0),
    CONSTRAINT chk_bond_deals_face_value CHECK (face_value > 0),
    CONSTRAINT chk_bond_deals_settlement_price CHECK (settlement_price > 0),
    CONSTRAINT chk_bond_deals_portfolio CHECK (portfolio_type IS NULL OR portfolio_type IN ('HTM', 'AFS', 'HFT')),
    CONSTRAINT chk_bond_deals_confirmation CHECK (confirmation_method IN ('EMAIL', 'REUTERS', 'OTHER')),
    CONSTRAINT chk_bond_deals_contract CHECK (contract_prepared_by IN ('INTERNAL', 'COUNTERPARTY')),
    CONSTRAINT chk_bond_deals_status CHECK (status IN (
        'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
        'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
        'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'CANCELLED'
    ))
);

CREATE INDEX idx_bond_deals_status ON bond_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_trade_date ON bond_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_counterparty ON bond_deals (counterparty_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_category ON bond_deals (bond_category) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_catalog ON bond_deals (bond_catalog_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_bond_deals_updated_at
    BEFORE UPDATE ON bond_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE bond_deals IS 'Giao dịch trái phiếu / GTCG — TPCP, TCTC, GTCG';
COMMENT ON COLUMN bond_deals.deal_number IS 'Mã giao dịch gapless: G-20260403-0001 (Govi), F-20260403-0001 (FI/CD)';
COMMENT ON COLUMN bond_deals.bond_category IS 'Phân loại: GOVERNMENT (TPCP), FINANCIAL_INSTITUTION (TCTC), CERTIFICATE_OF_DEPOSIT (GTCG)';
COMMENT ON COLUMN bond_deals.direction IS 'Hướng: BUY (mua), SELL (bán)';
COMMENT ON COLUMN bond_deals.transaction_type IS 'Loại giao dịch: REPO, REVERSE_REPO, OUTRIGHT, OTHER';
COMMENT ON COLUMN bond_deals.quantity IS 'Số lượng trái phiếu (integer)';
COMMENT ON COLUMN bond_deals.face_value IS 'Mệnh giá VND (NUMERIC(20,0))';
COMMENT ON COLUMN bond_deals.clean_price IS 'Giá sạch VND';
COMMENT ON COLUMN bond_deals.settlement_price IS 'Giá thanh toán/dirty price VND';
COMMENT ON COLUMN bond_deals.total_value IS 'Tổng giá trị = quantity × settlement_price';
COMMENT ON COLUMN bond_deals.portfolio_type IS 'Danh mục đầu tư: HTM, AFS, HFT — chỉ khi BUY';
COMMENT ON COLUMN bond_deals.remaining_tenor_days IS 'Thời hạn còn lại (ngày) = maturity_date − payment_date';

-- ---------------------------------------------------------------------------
-- Table 20: bond_inventory — Tồn kho trái phiếu
-- ---------------------------------------------------------------------------
CREATE TABLE bond_inventory (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bond_catalog_id     UUID NULL REFERENCES bond_catalog(id) ON DELETE RESTRICT,
    bond_code           VARCHAR(50) NOT NULL,
    bond_category       VARCHAR(30) NOT NULL,
    portfolio_type      VARCHAR(5) NOT NULL,
    available_quantity  BIGINT NOT NULL DEFAULT 0,
    acquisition_date    DATE NULL,
    acquisition_price   NUMERIC(20,0) NULL,
    version             INT NOT NULL DEFAULT 1,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT uq_bond_inventory UNIQUE (bond_code, bond_category, portfolio_type),
    CONSTRAINT chk_bond_inventory_quantity CHECK (available_quantity >= 0),
    CONSTRAINT chk_bond_inventory_category CHECK (bond_category IN ('GOVERNMENT', 'FINANCIAL_INSTITUTION', 'CERTIFICATE_OF_DEPOSIT')),
    CONSTRAINT chk_bond_inventory_portfolio CHECK (portfolio_type IN ('HTM', 'AFS', 'HFT'))
);

CREATE INDEX idx_bond_inventory_code ON bond_inventory (bond_code);
CREATE INDEX idx_bond_inventory_category ON bond_inventory (bond_category);

CREATE TRIGGER trg_bond_inventory_updated_at
    BEFORE UPDATE ON bond_inventory
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE bond_inventory IS 'Tồn kho trái phiếu — kiểm tra số lượng khả dụng khi bán';
COMMENT ON COLUMN bond_inventory.available_quantity IS 'Số lượng khả dụng — hard block khi bán vượt tồn kho';
COMMENT ON COLUMN bond_inventory.version IS 'Phiên bản — optimistic locking ngăn oversell';
COMMENT ON COLUMN bond_inventory.acquisition_date IS 'Ngày mua ban đầu';
COMMENT ON COLUMN bond_inventory.acquisition_price IS 'Giá mua ban đầu (VND)';
