-- Migration 005: FX BRD v3 gaps — new fields, PENDING_TP_REVIEW status

-- 1. Add new columns to fx_deals
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS execution_date DATE NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS pay_code_klb TEXT NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS pay_code_counterparty TEXT NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS is_international BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS attachment_path TEXT NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS attachment_name TEXT NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS settlement_amount NUMERIC(20,2) NULL;
ALTER TABLE fx_deals ADD COLUMN IF NOT EXISTS settlement_currency VARCHAR(3) NULL;

-- 2. Add new columns to fx_deal_legs
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS pay_code_klb TEXT NULL;
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS pay_code_counterparty TEXT NULL;
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS is_international BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS execution_date DATE NULL;
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS settlement_amount NUMERIC(20,2) NULL;
ALTER TABLE fx_deal_legs ADD COLUMN IF NOT EXISTS settlement_currency VARCHAR(3) NULL;

-- 3. Extend status CHECK constraint to include PENDING_TP_REVIEW
ALTER TABLE fx_deals DROP CONSTRAINT IF EXISTS chk_fx_deals_status;
ALTER TABLE fx_deals ADD CONSTRAINT chk_fx_deals_status CHECK (status IN (
  'OPEN', 'PENDING_L2_APPROVAL', 'PENDING_TP_REVIEW', 'REJECTED',
  'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT', 'PENDING_SETTLEMENT',
  'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT', 'CANCELLED',
  'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2'
));

-- 4. Extend approval_actions CHECK to include new action types
ALTER TABLE approval_actions DROP CONSTRAINT IF EXISTS chk_approval_actions_type;
ALTER TABLE approval_actions ADD CONSTRAINT chk_approval_actions_type CHECK (action_type IN (
    'DESK_HEAD_APPROVE', 'DESK_HEAD_RETURN', 'DESK_HEAD_REAPPROVE', 'DESK_HEAD_RETURN_TO_CV',
    'DIRECTOR_APPROVE', 'DIRECTOR_REJECT',
    'RISK_OFFICER_APPROVE', 'RISK_OFFICER_REJECT',
    'RISK_HEAD_APPROVE', 'RISK_HEAD_REJECT',
    'ACCOUNTANT_APPROVE', 'ACCOUNTANT_REJECT',
    'CHIEF_ACCOUNTANT_APPROVE', 'CHIEF_ACCOUNTANT_REJECT',
    'SETTLEMENT_APPROVE', 'SETTLEMENT_REJECT',
    'DEALER_SUBMIT', 'DEALER_RECALL', 'DESK_HEAD_RECALL', 'TP_RECALL',
    'CANCEL_REQUEST', 'CANCEL_DESK_HEAD_APPROVE', 'CANCEL_DESK_HEAD_REJECT',
    'CANCEL_DIVISION_HEAD_APPROVE', 'CANCEL_DIVISION_HEAD_REJECT',
    'CANCEL_APPROVE_L1', 'CANCEL_REJECT_L1',
    'CANCEL_APPROVE_L2', 'CANCEL_REJECT_L2'
));

-- 5. Drop and recreate views (column order changed, CREATE OR REPLACE cannot reorder)
DROP VIEW IF EXISTS v_fx_deals_list;
CREATE VIEW v_fx_deals_list AS
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
    d.execution_date,
    d.branch_id,
    d.uses_credit_limit,
    d.is_international,
    d.settlement_amount,
    d.settlement_currency,
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
    l2.settlement_date AS leg2_settlement_date
FROM fx_deals d
JOIN counterparties c ON d.counterparty_id = c.id
LEFT JOIN fx_deal_legs l1 ON l1.deal_id = d.id AND l1.leg_number = 1
LEFT JOIN fx_deal_legs l2 ON l2.deal_id = d.id AND l2.leg_number = 2
WHERE d.deleted_at IS NULL;

DROP VIEW IF EXISTS v_fx_deal_detail;
CREATE VIEW v_fx_deal_detail AS
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

-- 5. Add TP review transition rules
INSERT INTO status_transition_rules (deal_module, from_status, to_status, required_role)
VALUES
    ('FX', 'PENDING_TP_REVIEW', 'PENDING_L2_APPROVAL', 'DESK_HEAD'),
    ('FX', 'PENDING_TP_REVIEW', 'OPEN', 'DESK_HEAD')
ON CONFLICT DO NOTHING;
