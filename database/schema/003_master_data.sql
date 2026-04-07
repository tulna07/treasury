-- ============================================================================
-- 003_master_data.sql — Dữ liệu danh mục (7 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 10: counterparties — Đối tác giao dịch
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE counterparties IS 'Đối tác giao dịch — ngân hàng, tổ chức tài chính, KBNN, SBV';
COMMENT ON COLUMN counterparties.code IS 'Mã nội bộ đối tác: MSBI, ACB, VCB...';
COMMENT ON COLUMN counterparties.cif IS 'Mã khách hàng CIF từ Core Banking';
COMMENT ON COLUMN counterparties.swift_code IS 'SWIFT/BIC code — không bắt buộc với mọi đối tác';
COMMENT ON COLUMN counterparties.fx_uses_limit IS 'FX deals có tiêu thụ credit limit không (v3 feature)';
COMMENT ON COLUMN counterparties.deleted_at IS 'Soft delete — đối tác đã xóa vẫn giữ để tra cứu lịch sử';

-- ---------------------------------------------------------------------------
-- Table 11: currencies — Danh mục tiền tệ
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE currencies IS 'Danh mục tiền tệ theo ISO 4217';
COMMENT ON COLUMN currencies.code IS 'Mã tiền tệ ISO 4217: USD, VND, EUR, AUD, GBP, JPY, KRW...';
COMMENT ON COLUMN currencies.numeric_code IS 'Mã số ISO 4217: USD=840, VND=704';
COMMENT ON COLUMN currencies.decimal_places IS 'Số chữ số thập phân: VND=0, USD=2, JPY=0';

-- ---------------------------------------------------------------------------
-- Table 12: currency_pairs — Cặp tiền tệ và quy tắc tính toán
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE currency_pairs IS 'Cặp tiền tệ — quyết định công thức tính FX (MULTIPLY/DIVIDE)';
COMMENT ON COLUMN currency_pairs.pair_code IS 'Mã cặp: USD/VND, EUR/USD, EUR/GBP';
COMMENT ON COLUMN currency_pairs.calculation_rule IS 'Quy tắc tính: MULTIPLY (USD/VND) hoặc DIVIDE (USD/JPY)';
COMMENT ON COLUMN currency_pairs.result_currency IS 'Tiền tệ kết quả sau tính toán';
COMMENT ON COLUMN currency_pairs.rate_decimal_places IS 'Độ chính xác tỷ giá: 2 cho USD/VND, 4 cho EUR/USD';

-- ---------------------------------------------------------------------------
-- Table 13: bond_catalog — Danh mục trái phiếu / giấy tờ có giá
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE bond_catalog IS 'Danh mục trái phiếu / giấy tờ có giá — TPCP, TCTC, GTCG';
COMMENT ON COLUMN bond_catalog.bond_code IS 'Mã trái phiếu: TD2135068';
COMMENT ON COLUMN bond_catalog.coupon_rate IS 'Lãi suất coupon (%/năm)';
COMMENT ON COLUMN bond_catalog.bond_type IS 'Loại: GOVERNMENT (TPCP), FINANCIAL_INSTITUTION (TCTC), CERTIFICATE_OF_DEPOSIT (GTCG)';
COMMENT ON COLUMN bond_catalog.face_value IS 'Mệnh giá (VND) — số nguyên';
COMMENT ON COLUMN bond_catalog.payment_frequency IS 'Kỳ trả coupon: ANNUAL, SEMI_ANNUAL, QUARTERLY, ZERO_COUPON (Phase 2)';

-- ---------------------------------------------------------------------------
-- Table 14: settlement_instructions — Chỉ dẫn thanh toán / Pay Code / SSI
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE settlement_instructions IS 'Chỉ dẫn thanh toán (SSI/Pay Code) — mỗi đối tác có thể có nhiều SSI cho cùng tiền tệ';
COMMENT ON COLUMN settlement_instructions.owner_type IS 'Loại: INTERNAL (KLB) hoặc COUNTERPARTY (đối tác)';
COMMENT ON COLUMN settlement_instructions.account_number IS 'Số tài khoản thanh toán';
COMMENT ON COLUMN settlement_instructions.swift_code IS 'SWIFT code ngân hàng đại lý';
COMMENT ON COLUMN settlement_instructions.citad_code IS 'Mã Citad (thanh toán nội địa)';
COMMENT ON COLUMN settlement_instructions.is_default IS 'SSI mặc định cho counterparty + currency';

-- ---------------------------------------------------------------------------
-- Table 15: exchange_rates — Tỷ giá hối đoái
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE exchange_rates IS 'Tỷ giá hối đoái — dùng cho quy đổi credit limit, công bố cuối ngày';
COMMENT ON COLUMN exchange_rates.effective_date IS 'Ngày hiệu lực (cuối ngày làm việc trước đó)';
COMMENT ON COLUMN exchange_rates.mid_rate IS 'Tỷ giá trung bình = (buy + sell) / 2 — tính sẵn';
COMMENT ON COLUMN exchange_rates.source IS 'Nguồn tỷ giá: KLB_DAILY, SBV';

-- ---------------------------------------------------------------------------
-- Table 16: business_calendar — Lịch ngày làm việc
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE business_calendar IS 'Lịch ngày làm việc — dùng cho T+1/T+2 settlement date và tra cứu tỷ giá';
COMMENT ON COLUMN business_calendar.calendar_date IS 'Ngày trong năm';
COMMENT ON COLUMN business_calendar.is_business_day IS 'true = ngày làm việc, false = nghỉ/lễ';
COMMENT ON COLUMN business_calendar.holiday_name IS 'Tên ngày nghỉ/lễ: Tết Nguyên đán, 30/4, 2/9...';
