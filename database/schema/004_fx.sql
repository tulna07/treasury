-- ============================================================================
-- 004_fx.sql — Module FX: Ngoại hối (2 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 17: fx_deals — Giao dịch ngoại hối (header)
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE fx_deals IS 'Giao dịch ngoại hối — header. Spot/Forward = 1 leg, Swap = 2 legs';
COMMENT ON COLUMN fx_deals.deal_number IS 'Mã giao dịch gapless: FX-20260403-0001 — sinh từ deal_sequences';
COMMENT ON COLUMN fx_deals.deal_type IS 'Loại: SPOT, FORWARD, SWAP';
COMMENT ON COLUMN fx_deals.direction IS 'Hướng: SELL, BUY (Spot/Fwd); SELL_BUY, BUY_SELL (Swap)';
COMMENT ON COLUMN fx_deals.notional_amount IS 'Số tiền giao dịch (FCY, 2 chữ số thập phân)';
COMMENT ON COLUMN fx_deals.pair_code IS 'Cặp tiền tệ: USD/VND, EUR/USD...';
COMMENT ON COLUMN fx_deals.uses_credit_limit IS 'Giao dịch có tiêu thụ credit limit không';
COMMENT ON COLUMN fx_deals.status IS 'Trạng thái: OPEN → ... → COMPLETED / CANCELLED';
COMMENT ON COLUMN fx_deals.cloned_from_id IS 'ID giao dịch gốc khi clone';

-- ---------------------------------------------------------------------------
-- Table 18: fx_deal_legs — Legs giao dịch ngoại hối
-- ---------------------------------------------------------------------------
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

COMMENT ON TABLE fx_deal_legs IS 'Legs giao dịch FX — Spot/Forward = 1 leg, Swap = 2 legs (near + far)';
COMMENT ON COLUMN fx_deal_legs.leg_number IS '1 = near leg (Spot/Forward/Swap), 2 = far leg (Swap only)';
COMMENT ON COLUMN fx_deal_legs.exchange_rate IS 'Tỷ giá giao dịch (6 chữ số thập phân)';
COMMENT ON COLUMN fx_deal_legs.converted_amount IS 'Số tiền quy đổi = notional × rate hoặc notional / rate';
COMMENT ON COLUMN fx_deal_legs.requires_international_settlement IS 'Cần thanh toán quốc tế (tạo bản ghi international_payments)';
COMMENT ON COLUMN fx_deal_legs.internal_ssi_id IS 'Pay code KLB';
COMMENT ON COLUMN fx_deal_legs.counterparty_ssi_id IS 'Pay code đối tác';
