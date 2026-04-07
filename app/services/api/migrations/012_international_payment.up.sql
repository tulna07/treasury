-- ============================================================================
-- 008_international_payment.sql — Module TTQT: Thanh toán quốc tế (1 table)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 26: international_payments — Thanh toán quốc tế
-- ---------------------------------------------------------------------------
CREATE TABLE international_payments (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_module           VARCHAR(10) NOT NULL,
    source_deal_id          UUID NOT NULL,
    source_leg_number       SMALLINT NULL,
    ticket_display          VARCHAR(25) NOT NULL,
    counterparty_id         UUID NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    debit_account           VARCHAR(100) NOT NULL,
    bic_code                VARCHAR(11) NULL,
    currency_code           VARCHAR(3) NOT NULL REFERENCES currencies(code) ON DELETE RESTRICT,
    amount                  NUMERIC(20,2) NOT NULL,
    transfer_date           DATE NOT NULL,
    counterparty_ssi        TEXT NOT NULL,
    original_trade_date     DATE NOT NULL,
    approved_by_division    VARCHAR(255) NULL,
    settlement_status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    settled_by              UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    settled_at              TIMESTAMPTZ NULL,
    rejection_reason        TEXT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_intl_payment_module CHECK (source_module IN ('FX', 'MM')),
    CONSTRAINT chk_intl_payment_amount CHECK (amount > 0),
    CONSTRAINT chk_intl_payment_status CHECK (settlement_status IN ('PENDING', 'APPROVED', 'REJECTED'))
);

CREATE INDEX idx_intl_payment_transfer_date ON international_payments (transfer_date);
CREATE INDEX idx_intl_payment_status ON international_payments (settlement_status);
CREATE INDEX idx_intl_payment_source ON international_payments (source_module, source_deal_id);
CREATE INDEX idx_intl_payment_counterparty ON international_payments (counterparty_id);

COMMENT ON TABLE international_payments IS 'Thanh toán quốc tế — tự động tạo khi deal chuyển sang Pending Settlement (BRD 3.5)';
COMMENT ON COLUMN international_payments.source_module IS 'Module nguồn: FX hoặc MM';
COMMENT ON COLUMN international_payments.source_deal_id IS 'ID deal nguồn';
COMMENT ON COLUMN international_payments.source_leg_number IS 'Swap: 1 (near) hoặc 2 (far); deal khác: NULL';
COMMENT ON COLUMN international_payments.ticket_display IS 'Mã hiển thị — Swap có suffix A/B cho 2 legs';
COMMENT ON COLUMN international_payments.debit_account IS 'Tài khoản nostro KLB (HABIB)';
COMMENT ON COLUMN international_payments.bic_code IS 'BIC code đối tác';
COMMENT ON COLUMN international_payments.amount IS 'Số tiền chuyển (FCY)';
COMMENT ON COLUMN international_payments.transfer_date IS 'Ngày thanh toán = settlement date';
COMMENT ON COLUMN international_payments.counterparty_ssi IS 'Thông tin SSI đối tác (snapshot text)';
COMMENT ON COLUMN international_payments.settlement_status IS 'Trạng thái: PENDING → APPROVED / REJECTED';

-- View: danh sách TTQT
CREATE OR REPLACE VIEW v_international_payments_list AS
SELECT
    ip.id, ip.source_module, ip.source_deal_id, ip.source_leg_number,
    ip.ticket_display, ip.counterparty_id,
    cp.code AS counterparty_code, cp.full_name AS counterparty_name,
    ip.debit_account, ip.bic_code, ip.currency_code,
    ip.amount, ip.transfer_date, ip.counterparty_ssi,
    ip.original_trade_date, ip.approved_by_division,
    ip.settlement_status, ip.settled_by,
    su.full_name AS settled_by_name,
    ip.settled_at, ip.rejection_reason, ip.created_at
FROM international_payments ip
    JOIN counterparties cp ON cp.id = ip.counterparty_id
    LEFT JOIN users su ON su.id = ip.settled_by;
