-- 008: Module GTCG (Bond) — bond_deals + bond_inventory + views
-- Thêm bảng giao dịch GTCG và tồn kho trái phiếu
-- Status constraint đã bao gồm PENDING_CANCEL_L1/L2 (cancel 2 cấp BRD v3)

-- ---------------------------------------------------------------------------
-- Table: bond_deals — Giao dịch trái phiếu / GTCG
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS bond_deals (
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
    cloned_from_id          UUID NULL,
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
        'COMPLETED', 'VOIDED_BY_ACCOUNTING',
        'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'CANCELLED'
    ))
);

-- Self-referencing FK cho clone (sau khi bảng đã tạo)
ALTER TABLE bond_deals ADD CONSTRAINT fk_bond_deals_cloned_from
    FOREIGN KEY (cloned_from_id) REFERENCES bond_deals(id) ON DELETE SET NULL;

CREATE INDEX idx_bond_deals_status ON bond_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_trade_date ON bond_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_counterparty ON bond_deals (counterparty_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_category ON bond_deals (bond_category) WHERE deleted_at IS NULL;
CREATE INDEX idx_bond_deals_catalog ON bond_deals (bond_catalog_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_bond_deals_updated_at
    BEFORE UPDATE ON bond_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE bond_deals IS 'Giao dịch trái phiếu / GTCG — TPCP, TCTC, CCTG';
COMMENT ON COLUMN bond_deals.deal_number IS 'Mã giao dịch: G-YYYYMMDD-NNNN (Govi), F-YYYYMMDD-NNNN (FI/CCTG)';
COMMENT ON COLUMN bond_deals.bond_category IS 'Phân loại: GOVERNMENT, FINANCIAL_INSTITUTION, CERTIFICATE_OF_DEPOSIT';
COMMENT ON COLUMN bond_deals.direction IS 'Hướng: BUY (mua), SELL (bán)';
COMMENT ON COLUMN bond_deals.transaction_type IS 'Loại GD: REPO, REVERSE_REPO, OUTRIGHT, OTHER';
COMMENT ON COLUMN bond_deals.portfolio_type IS 'Danh mục: HTM, AFS, HFT — chỉ khi BUY';
COMMENT ON COLUMN bond_deals.remaining_tenor_days IS 'Kỳ hạn còn lại (ngày) = maturity_date − payment_date';

-- ---------------------------------------------------------------------------
-- Table: bond_inventory — Tồn kho trái phiếu
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS bond_inventory (
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

COMMENT ON TABLE bond_inventory IS 'Tồn kho trái phiếu — hard block khi bán vượt số lượng khả dụng';
COMMENT ON COLUMN bond_inventory.version IS 'Optimistic locking — ngăn oversell';

-- ---------------------------------------------------------------------------
-- Views cho Module GTCG
-- ---------------------------------------------------------------------------

-- View 1: Danh sách giao dịch GTCG (BRD §3.2.5)
CREATE OR REPLACE VIEW v_bond_deals_list AS
SELECT
    bd.id,
    bd.deal_number,
    bd.bond_category,
    bd.trade_date,
    bd.order_date,
    bd.value_date,
    bd.direction,
    bd.transaction_type,
    bd.transaction_type_other,
    bd.bond_catalog_id,
    bd.bond_code_manual,
    COALESCE(bc.bond_code, bd.bond_code_manual) AS bond_code_display,
    bd.issuer,
    bd.coupon_rate,
    bd.issue_date,
    bd.maturity_date,
    bd.quantity,
    bd.face_value,
    bd.discount_rate,
    bd.clean_price,
    bd.settlement_price,
    bd.total_value,
    bd.portfolio_type,
    bd.payment_date,
    bd.remaining_tenor_days,
    bd.confirmation_method,
    bd.contract_prepared_by,
    bd.status,
    bd.note,
    bd.cloned_from_id,
    bd.cancel_reason,
    bd.cancel_requested_at,
    bd.created_at,
    bd.created_by,
    cp.code              AS counterparty_code,
    cp.full_name         AS counterparty_name,
    cp.short_name        AS counterparty_short_name,
    u.full_name          AS created_by_name,
    u.username           AS created_by_username,
    br.code              AS branch_code,
    br.name              AS branch_name
FROM bond_deals bd
    JOIN counterparties cp ON cp.id = bd.counterparty_id
    JOIN users u           ON u.id  = bd.created_by
    JOIN branches br       ON br.id = bd.branch_id
    LEFT JOIN bond_catalog bc ON bc.id = bd.bond_catalog_id
WHERE bd.deleted_at IS NULL;

-- View 2: Dashboard KTTC — deals chờ hạch toán (BRD §3.2.3)
CREATE OR REPLACE VIEW v_bond_deals_pending_booking AS
SELECT
    bd.id,
    bd.deal_number,
    bd.bond_category,
    bd.trade_date,
    bd.value_date,
    bd.direction,
    bd.transaction_type,
    COALESCE(bc.bond_code, bd.bond_code_manual) AS bond_code_display,
    bd.issuer,
    bd.quantity,
    bd.settlement_price,
    bd.total_value,
    bd.portfolio_type,
    bd.status,
    bd.created_at,
    cp.code              AS counterparty_code,
    cp.full_name         AS counterparty_name,
    u.full_name          AS created_by_name,
    CASE bd.status
        WHEN 'PENDING_BOOKING'          THEN 1
        WHEN 'PENDING_CHIEF_ACCOUNTANT' THEN 2
    END AS booking_level
FROM bond_deals bd
    JOIN counterparties cp ON cp.id = bd.counterparty_id
    JOIN users u           ON u.id  = bd.created_by
    LEFT JOIN bond_catalog bc ON bc.id = bd.bond_catalog_id
WHERE bd.deleted_at IS NULL
  AND bd.status IN ('PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT')
ORDER BY bd.trade_date ASC, bd.created_at ASC;

-- View 3: Tồn kho GTCG + catalog info (BRD §5.4)
CREATE OR REPLACE VIEW v_bond_inventory_summary AS
SELECT
    bi.id,
    bi.bond_code,
    bi.bond_category,
    bi.portfolio_type,
    bi.available_quantity,
    bi.acquisition_date,
    bi.acquisition_price,
    bi.version,
    bi.updated_at,
    bc.id                AS catalog_id,
    bc.issuer            AS catalog_issuer,
    bc.coupon_rate       AS catalog_coupon_rate,
    bc.issue_date        AS catalog_issue_date,
    bc.maturity_date     AS catalog_maturity_date,
    bc.face_value        AS catalog_face_value,
    bi.available_quantity * COALESCE(bc.face_value, 0) AS nominal_value,
    u.full_name          AS updated_by_name
FROM bond_inventory bi
    LEFT JOIN bond_catalog bc ON bc.id = bi.bond_catalog_id
    LEFT JOIN users u         ON u.id  = bi.updated_by
WHERE bi.available_quantity > 0;

-- View 4: GD + lịch sử phê duyệt (BRD §8)
CREATE OR REPLACE VIEW v_bond_deals_with_approval_history AS
SELECT
    bd.id              AS deal_id,
    bd.deal_number,
    bd.bond_category,
    bd.status          AS current_status,
    bd.trade_date,
    bd.created_at      AS deal_created_at,
    aa.id              AS action_id,
    aa.action_type,
    aa.status_before,
    aa.status_after,
    aa.performed_at,
    aa.reason          AS action_reason,
    aa.metadata        AS action_metadata,
    au.full_name       AS performed_by_name,
    au.username        AS performed_by_username
FROM bond_deals bd
    LEFT JOIN approval_actions aa ON aa.deal_module = 'BOND' AND aa.deal_id = bd.id
    LEFT JOIN users au            ON au.id = aa.performed_by
WHERE bd.deleted_at IS NULL
ORDER BY bd.id, aa.performed_at ASC;
