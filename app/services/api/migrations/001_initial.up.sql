-- ============================================================================
-- 001_initial.up.sql — Full Treasury Management System Schema
-- Combined from database/schema/ files
-- ============================================================================

-- ===========================================================================
-- 001_auth.sql — Authentication & User Management
-- ===========================================================================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     VARCHAR(255) NULL,
    username        VARCHAR(100) NOT NULL,
    password_hash   VARCHAR(255) NULL,
    full_name       VARCHAR(255) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    branch_id       UUID NULL,
    department      VARCHAR(100) NULL,
    position        VARCHAR(100) NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ NULL,
    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_external_id UNIQUE (external_id)
);

CREATE INDEX idx_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_external_id ON users (external_id) WHERE external_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_department ON users (department) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_branch_id ON users (branch_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT NULL,
    scope       VARCHAR(50) NOT NULL DEFAULT 'ALL',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_roles_code UNIQUE (code),
    CONSTRAINT chk_roles_scope CHECK (scope IN ('ALL', 'MODULE_SPECIFIC', 'STEP_SPECIFIC'))
);

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) NOT NULL,
    resource    VARCHAR(50) NOT NULL,
    action      VARCHAR(30) NOT NULL,
    description TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_permissions_code UNIQUE (code),
    CONSTRAINT chk_permissions_resource CHECK (resource IN (
        'FX_DEAL', 'BOND_DEAL', 'MM_INTERBANK_DEAL', 'MM_OMO_REPO_DEAL',
        'CREDIT_LIMIT', 'INTERNATIONAL_PAYMENT', 'MASTER_DATA', 'SYSTEM'
    )),
    CONSTRAINT chk_permissions_action CHECK (action IN (
        'VIEW', 'CREATE', 'EDIT', 'DELETE', 'APPROVE_L1', 'APPROVE_L2',
        'APPROVE_RISK_L1', 'APPROVE_RISK_L2', 'BOOK_L1', 'BOOK_L2',
        'SETTLE', 'RECALL', 'CANCEL_REQUEST', 'CANCEL_APPROVE_L1',
        'CANCEL_APPROVE_L2', 'CLONE', 'EXPORT', 'MANAGE'
    ))
);

CREATE INDEX idx_permissions_resource ON permissions (resource);
CREATE INDEX idx_permissions_action ON permissions (action);

CREATE TABLE role_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_role_permissions UNIQUE (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions (role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);

CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by  UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_user_roles UNIQUE (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);

CREATE TABLE auth_configs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_mode               VARCHAR(20) NOT NULL DEFAULT 'standalone',
    issuer_url              VARCHAR(500) NULL,
    client_id               VARCHAR(255) NULL,
    client_secret_encrypted TEXT NULL,
    scopes                  VARCHAR(500) NULL,
    auto_create_user        BOOLEAN NOT NULL DEFAULT true,
    sync_user_info          BOOLEAN NOT NULL DEFAULT true,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by              UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_auth_configs_mode CHECK (auth_mode IN ('standalone', 'zitadel'))
);

CREATE TRIGGER trg_auth_configs_updated_at
    BEFORE UPDATE ON auth_configs
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE external_role_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_group  VARCHAR(255) NOT NULL,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_external_role_mappings UNIQUE (external_group, role_id)
);

CREATE TABLE user_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL,
    ip_address  INET NULL,
    user_agent  TEXT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_sessions_token_hash UNIQUE (token_hash)
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_token_hash ON user_sessions (token_hash);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);

-- ===========================================================================
-- 002_organization.sql — Branches
-- ===========================================================================

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

ALTER TABLE users ADD CONSTRAINT fk_users_branch_id
    FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE SET NULL;

-- ===========================================================================
-- 003_master_data.sql — Counterparties, Currencies, etc.
-- ===========================================================================

CREATE TABLE counterparties (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(20) NOT NULL,
    full_name       VARCHAR(500) NOT NULL,
    short_name      VARCHAR(255) NULL,
    cif             VARCHAR(50) NOT NULL,
    swift_code      VARCHAR(11) NULL,
    country_code    VARCHAR(2) NULL,
    tax_id          VARCHAR(20) NULL,
    address         TEXT NULL,
    fx_uses_limit   BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    deleted_at      TIMESTAMPTZ NULL,
    CONSTRAINT uq_counterparties_code UNIQUE (code)
);

CREATE INDEX idx_counterparties_code ON counterparties (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_counterparties_cif ON counterparties (cif) WHERE deleted_at IS NULL;
CREATE INDEX idx_counterparties_swift ON counterparties (swift_code) WHERE swift_code IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_counterparties_active ON counterparties (is_active) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_counterparties_updated_at
    BEFORE UPDATE ON counterparties
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE currencies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(3) NOT NULL,
    numeric_code    SMALLINT NULL,
    name            VARCHAR(100) NOT NULL,
    decimal_places  SMALLINT NOT NULL DEFAULT 2,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_currencies_code UNIQUE (code),
    CONSTRAINT chk_currencies_decimal CHECK (decimal_places >= 0 AND decimal_places <= 6)
);

CREATE TABLE currency_pairs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    base_currency       VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    quote_currency      VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    pair_code           VARCHAR(7) NOT NULL,
    rate_decimal_places SMALLINT NOT NULL DEFAULT 4,
    calculation_rule    VARCHAR(20) NOT NULL,
    result_currency     VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_currency_pairs_code UNIQUE (pair_code),
    CONSTRAINT chk_currency_pairs_rule CHECK (calculation_rule IN ('MULTIPLY', 'DIVIDE')),
    CONSTRAINT chk_currency_pairs_different CHECK (base_currency <> quote_currency)
);

CREATE TABLE bond_catalog (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bond_code           VARCHAR(50) NOT NULL,
    issuer              VARCHAR(500) NOT NULL,
    coupon_rate         NUMERIC(10,4) NOT NULL,
    payment_frequency   VARCHAR(20) NULL,
    issue_date          DATE NOT NULL,
    maturity_date       DATE NOT NULL,
    face_value          NUMERIC(20,0) NOT NULL,
    bond_type           VARCHAR(20) NOT NULL,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_bond_catalog_code UNIQUE (bond_code),
    CONSTRAINT chk_bond_catalog_type CHECK (bond_type IN ('GOVERNMENT', 'FINANCIAL_INSTITUTION', 'CERTIFICATE_OF_DEPOSIT')),
    CONSTRAINT chk_bond_catalog_frequency CHECK (payment_frequency IS NULL OR payment_frequency IN ('ANNUAL', 'SEMI_ANNUAL', 'QUARTERLY', 'ZERO_COUPON')),
    CONSTRAINT chk_bond_catalog_face_value CHECK (face_value > 0),
    CONSTRAINT chk_bond_catalog_dates CHECK (maturity_date > issue_date)
);

CREATE INDEX idx_bond_catalog_code ON bond_catalog (bond_code);
CREATE INDEX idx_bond_catalog_type ON bond_catalog (bond_type);
CREATE INDEX idx_bond_catalog_maturity ON bond_catalog (maturity_date);

CREATE TRIGGER trg_bond_catalog_updated_at
    BEFORE UPDATE ON bond_catalog
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE settlement_instructions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    counterparty_id     UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    currency_code       VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    owner_type          VARCHAR(15) NOT NULL,
    account_number      VARCHAR(100) NOT NULL,
    bank_name           VARCHAR(500) NOT NULL,
    swift_code          VARCHAR(11) NULL,
    citad_code          VARCHAR(20) NULL,
    description         TEXT NULL,
    is_default          BOOLEAN NOT NULL DEFAULT false,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_settlement_instructions_owner CHECK (owner_type IN ('INTERNAL', 'COUNTERPARTY'))
);

CREATE INDEX idx_settlement_instructions_counterparty ON settlement_instructions (counterparty_id);
CREATE INDEX idx_settlement_instructions_currency ON settlement_instructions (currency_code);
CREATE INDEX idx_settlement_instructions_owner ON settlement_instructions (owner_type);

CREATE TRIGGER trg_settlement_instructions_updated_at
    BEFORE UPDATE ON settlement_instructions
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE exchange_rates (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    currency_code       VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    effective_date      DATE NOT NULL,
    buy_transfer_rate   NUMERIC(20,4) NOT NULL,
    sell_transfer_rate  NUMERIC(20,4) NOT NULL,
    mid_rate            NUMERIC(20,4) NOT NULL,
    source              VARCHAR(50) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_exchange_rates UNIQUE (currency_code, effective_date),
    CONSTRAINT chk_exchange_rates_buy CHECK (buy_transfer_rate > 0),
    CONSTRAINT chk_exchange_rates_sell CHECK (sell_transfer_rate > 0),
    CONSTRAINT chk_exchange_rates_mid CHECK (mid_rate > 0)
);

CREATE INDEX idx_exchange_rates_date ON exchange_rates (effective_date);
CREATE INDEX idx_exchange_rates_currency_date ON exchange_rates (currency_code, effective_date DESC);

CREATE TABLE business_calendar (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_date   DATE NOT NULL,
    country_code    VARCHAR(2) NOT NULL DEFAULT 'VN',
    is_business_day BOOLEAN NOT NULL,
    holiday_name    VARCHAR(255) NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_business_calendar UNIQUE (calendar_date, country_code)
);

CREATE INDEX idx_business_calendar_date ON business_calendar (calendar_date);
CREATE INDEX idx_business_calendar_country_bizday ON business_calendar (country_code, is_business_day);

-- ===========================================================================
-- 004_fx.sql — FX Deals
-- ===========================================================================

CREATE TABLE fx_deals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_number           VARCHAR(30) NOT NULL,
    ticket_number         VARCHAR(20) NULL,
    counterparty_id       UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    deal_type             VARCHAR(10) NOT NULL,
    direction             VARCHAR(10) NOT NULL,
    notional_amount       NUMERIC(20,2) NOT NULL,
    currency_code         VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    pair_code             VARCHAR(7) NOT NULL REFERENCES currency_pairs(pair_code) ON DELETE RESTRICT,
    trade_date            DATE NOT NULL,
    branch_id             UUID NOT NULL REFERENCES branches(id) ON DELETE RESTRICT,
    uses_credit_limit     BOOLEAN NOT NULL DEFAULT false,
    status                VARCHAR(30) NOT NULL DEFAULT 'OPEN',
    note                  TEXT NULL,
    cloned_from_id        UUID NULL REFERENCES fx_deals(id) ON DELETE SET NULL,
    cancel_reason         TEXT NULL,
    cancel_requested_by   UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    cancel_requested_at   TIMESTAMPTZ NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by            UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    deleted_at            TIMESTAMPTZ NULL,
    version               INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT uq_fx_deals_number UNIQUE (deal_number),
    CONSTRAINT chk_fx_deals_type CHECK (deal_type IN ('SPOT', 'FORWARD', 'SWAP')),
    CONSTRAINT chk_fx_deals_direction CHECK (direction IN ('SELL', 'BUY', 'SELL_BUY', 'BUY_SELL')),
    CONSTRAINT chk_fx_deals_amount CHECK (notional_amount > 0),
    CONSTRAINT chk_fx_deals_status CHECK (status IN (
        'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
        'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT',
        'PENDING_SETTLEMENT', 'COMPLETED',
        'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT',
        'CANCELLED'
    ))
);

CREATE INDEX idx_fx_deals_status ON fx_deals (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_fx_deals_trade_date ON fx_deals (trade_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_fx_deals_counterparty ON fx_deals (counterparty_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_fx_deals_type ON fx_deals (deal_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_fx_deals_created_by ON fx_deals (created_by) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_fx_deals_updated_at
    BEFORE UPDATE ON fx_deals
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE fx_deal_legs (
    id                                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_id                             UUID NOT NULL REFERENCES fx_deals(id) ON DELETE CASCADE,
    leg_number                          SMALLINT NOT NULL,
    value_date                          DATE NOT NULL,
    settlement_date                     DATE NOT NULL,
    exchange_rate                       NUMERIC(20,6) NOT NULL,
    converted_amount                    NUMERIC(20,2) NOT NULL,
    converted_currency                  VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    internal_ssi_id                     UUID NOT NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    counterparty_ssi_id                 UUID NOT NULL REFERENCES settlement_instructions(id) ON DELETE RESTRICT,
    requires_international_settlement   BOOLEAN NOT NULL DEFAULT false,
    created_at                          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                          UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_fx_deal_legs UNIQUE (deal_id, leg_number),
    CONSTRAINT chk_fx_deal_legs_number CHECK (leg_number IN (1, 2)),
    CONSTRAINT chk_fx_deal_legs_rate CHECK (exchange_rate > 0)
);

CREATE INDEX idx_fx_deal_legs_deal_id ON fx_deal_legs (deal_id);
CREATE INDEX idx_fx_deal_legs_settlement ON fx_deal_legs (settlement_date);

CREATE TRIGGER trg_fx_deal_legs_updated_at
    BEFORE UPDATE ON fx_deal_legs
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ===========================================================================
-- 009_workflow.sql — Deal Sequences, Approvals, State Machine
-- ===========================================================================

CREATE TABLE deal_sequences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module          VARCHAR(20) NOT NULL,
    prefix          VARCHAR(10) NOT NULL,
    date_partition  DATE NOT NULL,
    last_sequence   BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_deal_sequences UNIQUE (module, prefix, date_partition),
    CONSTRAINT chk_deal_sequences_module CHECK (module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO')),
    CONSTRAINT chk_deal_sequences_seq CHECK (last_sequence >= 0)
);

CREATE TRIGGER trg_deal_sequences_updated_at
    BEFORE UPDATE ON deal_sequences
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TABLE approval_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module     VARCHAR(20) NOT NULL,
    deal_id         UUID NOT NULL,
    action_type     VARCHAR(30) NOT NULL,
    status_before   VARCHAR(30) NOT NULL,
    status_after    VARCHAR(30) NOT NULL,
    performed_by    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    performed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason          TEXT NULL,
    metadata        JSONB NULL,
    CONSTRAINT chk_approval_actions_module CHECK (deal_module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO')),
    CONSTRAINT chk_approval_actions_type CHECK (action_type IN (
        'DESK_HEAD_APPROVE', 'DESK_HEAD_RETURN',
        'DIRECTOR_APPROVE', 'DIRECTOR_REJECT',
        'RISK_OFFICER_APPROVE', 'RISK_OFFICER_REJECT',
        'RISK_HEAD_APPROVE', 'RISK_HEAD_REJECT',
        'ACCOUNTANT_APPROVE', 'ACCOUNTANT_REJECT',
        'CHIEF_ACCOUNTANT_APPROVE', 'CHIEF_ACCOUNTANT_REJECT',
        'SETTLEMENT_APPROVE', 'SETTLEMENT_REJECT',
        'DEALER_SUBMIT', 'DEALER_RECALL', 'DESK_HEAD_RECALL', 'TP_RECALL',
        'CANCEL_REQUEST', 'CANCEL_APPROVE_L1', 'CANCEL_APPROVE_L2',
        'CANCEL_DESK_HEAD_APPROVE', 'CANCEL_DESK_HEAD_REJECT',
        'CANCEL_DIVISION_HEAD_APPROVE', 'CANCEL_DIVISION_HEAD_REJECT'
    ))
);

CREATE INDEX idx_approval_actions_deal ON approval_actions (deal_module, deal_id);
CREATE INDEX idx_approval_actions_performed_by ON approval_actions (performed_by);
CREATE INDEX idx_approval_actions_performed_at ON approval_actions (performed_at);

CREATE TABLE status_transition_rules (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module             VARCHAR(20) NOT NULL,
    from_status             VARCHAR(30) NOT NULL,
    to_status               VARCHAR(30) NOT NULL,
    required_role           VARCHAR(50) NOT NULL,
    requires_reason         BOOLEAN NOT NULL DEFAULT false,
    requires_confirmation   BOOLEAN NOT NULL DEFAULT false,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT uq_status_transitions UNIQUE (deal_module, from_status, to_status, required_role),
    CONSTRAINT chk_status_transitions_module CHECK (deal_module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO'))
);

CREATE INDEX idx_status_transitions_module ON status_transition_rules (deal_module);
CREATE INDEX idx_status_transitions_from ON status_transition_rules (deal_module, from_status) WHERE is_active = true;

-- ===========================================================================
-- 012_audit.sql — Audit Logs (partitioned)
-- ===========================================================================

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

ALTER TABLE audit_logs ADD CONSTRAINT pk_audit_logs PRIMARY KEY (id, performed_at);

CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit_logs_2026_02 PARTITION OF audit_logs FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit_logs_2026_03 PARTITION OF audit_logs FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit_logs_2026_05 PARTITION OF audit_logs FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit_logs_2026_06 PARTITION OF audit_logs FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit_logs_2026_07 PARTITION OF audit_logs FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit_logs_2026_08 PARTITION OF audit_logs FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit_logs_2026_09 PARTITION OF audit_logs FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit_logs_2026_10 PARTITION OF audit_logs FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit_logs_2026_11 PARTITION OF audit_logs FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit_logs_2026_12 PARTITION OF audit_logs FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX idx_audit_logs_deal ON audit_logs (deal_module, deal_id);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_performed_at ON audit_logs (performed_at);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);

-- ===========================================================================
-- Views for high-performance FX deal listing/detail
-- ===========================================================================

-- View for FX deal listing — pre-joined with counterparty, includes leg summary
CREATE OR REPLACE VIEW v_fx_deals_list AS
SELECT
    d.id,
    d.deal_number,
    d.ticket_number,
    d.counterparty_id,
    c.code AS counterparty_code,
    c.full_name AS counterparty_name,
    d.deal_type,
    d.direction,
    d.notional_amount,
    d.currency_code,
    d.pair_code,
    d.trade_date,
    d.branch_id,
    d.uses_credit_limit,
    d.status,
    d.note,
    d.created_by,
    d.created_at,
    d.updated_at,
    d.version,
    -- Leg 1 summary
    l1.value_date AS leg1_value_date,
    l1.settlement_date AS leg1_settlement_date,
    l1.exchange_rate AS leg1_exchange_rate,
    l1.converted_amount AS leg1_converted_amount,
    l1.converted_currency AS leg1_converted_currency,
    -- Leg 2 summary (NULL for spot/forward)
    l2.value_date AS leg2_value_date,
    l2.settlement_date AS leg2_settlement_date,
    l2.exchange_rate AS leg2_exchange_rate,
    l2.converted_amount AS leg2_converted_amount,
    l2.converted_currency AS leg2_converted_currency,
    -- Computed
    CASE WHEN d.deal_type = 'SWAP' THEN 2 ELSE 1 END AS leg_count
FROM fx_deals d
JOIN counterparties c ON d.counterparty_id = c.id
LEFT JOIN fx_deal_legs l1 ON d.id = l1.deal_id AND l1.leg_number = 1
LEFT JOIN fx_deal_legs l2 ON d.id = l2.deal_id AND l2.leg_number = 2
WHERE d.deleted_at IS NULL;

-- View for FX deal detail — full info with all legs
CREATE OR REPLACE VIEW v_fx_deal_detail AS
SELECT
    d.*,
    c.code AS counterparty_code,
    c.full_name AS counterparty_name,
    c.swift_code AS counterparty_swift,
    u.full_name AS created_by_name,
    b.code AS branch_code,
    b.name AS branch_name
FROM fx_deals d
JOIN counterparties c ON d.counterparty_id = c.id
JOIN users u ON d.created_by = u.id
JOIN branches b ON d.branch_id = b.id
WHERE d.deleted_at IS NULL;

-- Indexes to support the views
CREATE INDEX IF NOT EXISTS idx_fx_deals_status_trade_date ON fx_deals(status, trade_date DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_fx_deals_counterparty_status ON fx_deals(counterparty_id, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_fx_deal_legs_deal_leg ON fx_deal_legs(deal_id, leg_number);
