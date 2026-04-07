-- 011: Module Money Market — mm_interbank_deals + mm_omo_repo_deals + views

-- ---------------------------------------------------------------------------
-- Table: mm_interbank_deals — Giao dịch liên ngân hàng
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mm_interbank_deals (
    id                                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_number                         VARCHAR(30) NOT NULL,
    ticket_number                       VARCHAR(20) NULL,
    counterparty_id                     UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    branch_id                           UUID NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    currency_code                       VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    internal_ssi_id                     UUID NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    counterparty_ssi_id                 UUID NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    counterparty_ssi_text               TEXT NULL,
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
    cloned_from_id                      UUID NULL,
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
        'OPEN', 'PENDING_TP_REVIEW', 'PENDING_L2_APPROVAL', 'REJECTED',
        'PENDING_RISK_APPROVAL', 'VOIDED_BY_RISK',
        'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
        'PENDING_SETTLEMENT', 'COMPLETED',
        'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT',
        'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'CANCELLED'
    ))
);

ALTER TABLE mm_interbank_deals ADD CONSTRAINT fk_mm_interbank_cloned
    FOREIGN KEY (cloned_from_id) REFERENCES mm_interbank_deals(id) ON DELETE SET NULL;

CREATE INDEX idx_mm_interbank_status ON mm_interbank_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_trade_date ON mm_interbank_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_counterparty ON mm_interbank_deals (counterparty_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_maturity ON mm_interbank_deals (maturity_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_interbank_direction ON mm_interbank_deals (direction) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_mm_interbank_deals_updated_at
    BEFORE UPDATE ON mm_interbank_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ---------------------------------------------------------------------------
-- Table: mm_omo_repo_deals — Giao dịch OMO / Repo KBNN
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS mm_omo_repo_deals (
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
    cloned_from_id        UUID NULL,
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
        'COMPLETED', 'VOIDED_BY_ACCOUNTING',
        'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'CANCELLED'
    ))
);

ALTER TABLE mm_omo_repo_deals ADD CONSTRAINT fk_mm_omo_repo_cloned
    FOREIGN KEY (cloned_from_id) REFERENCES mm_omo_repo_deals(id) ON DELETE SET NULL;

CREATE INDEX idx_mm_omo_repo_status ON mm_omo_repo_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_omo_repo_trade_date ON mm_omo_repo_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_mm_omo_repo_subtype ON mm_omo_repo_deals (deal_subtype) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_mm_omo_repo_deals_updated_at
    BEFORE UPDATE ON mm_omo_repo_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ---------------------------------------------------------------------------
-- Views
-- ---------------------------------------------------------------------------

CREATE OR REPLACE VIEW v_mm_interbank_deals_list AS
SELECT
    d.id, d.deal_number, d.ticket_number, d.trade_date, d.effective_date,
    d.direction, d.counterparty_id,
    cp.code AS counterparty_code, cp.full_name AS counterparty_name,
    d.currency_code, d.principal_amount, d.interest_rate,
    d.day_count_convention, d.tenor_days, d.maturity_date,
    d.interest_amount, d.maturity_amount,
    d.has_collateral, d.requires_international_settlement,
    d.status, d.note, d.cloned_from_id,
    d.cancel_reason, d.cancel_requested_at,
    d.created_by, u.full_name AS created_by_name, d.created_at,
    br.code AS branch_code, br.name AS branch_name
FROM mm_interbank_deals d
    JOIN counterparties cp ON cp.id = d.counterparty_id
    JOIN users u ON u.id = d.created_by
    JOIN branches br ON br.id = d.branch_id
WHERE d.deleted_at IS NULL;

CREATE OR REPLACE VIEW v_mm_omo_repo_deals_list AS
SELECT
    d.id, d.deal_number, d.deal_subtype, d.session_name, d.trade_date,
    d.counterparty_id,
    cp.code AS counterparty_code, cp.full_name AS counterparty_name,
    d.notional_amount, d.bond_catalog_id,
    bc.bond_code, bc.issuer AS bond_issuer, bc.coupon_rate AS bond_coupon_rate,
    bc.maturity_date AS bond_maturity_date,
    d.winning_rate, d.tenor_days, d.settlement_date_1, d.settlement_date_2,
    d.haircut_pct, d.status, d.note, d.cloned_from_id,
    d.cancel_reason, d.cancel_requested_at,
    d.created_by, u.full_name AS created_by_name, d.created_at,
    br.code AS branch_code, br.name AS branch_name
FROM mm_omo_repo_deals d
    JOIN counterparties cp ON cp.id = d.counterparty_id
    JOIN bond_catalog bc ON bc.id = d.bond_catalog_id
    JOIN users u ON u.id = d.created_by
    JOIN branches br ON br.id = d.branch_id
WHERE d.deleted_at IS NULL;
