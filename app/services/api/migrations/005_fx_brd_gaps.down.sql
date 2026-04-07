-- Rollback migration 005

ALTER TABLE fx_deals DROP COLUMN IF EXISTS execution_date;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS pay_code_klb;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS pay_code_counterparty;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS is_international;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS attachment_path;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS attachment_name;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS settlement_amount;
ALTER TABLE fx_deals DROP COLUMN IF EXISTS settlement_currency;

ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS pay_code_klb;
ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS pay_code_counterparty;
ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS is_international;
ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS execution_date;
ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS settlement_amount;
ALTER TABLE fx_deal_legs DROP COLUMN IF EXISTS settlement_currency;

ALTER TABLE fx_deals DROP CONSTRAINT IF EXISTS chk_fx_deals_status;
ALTER TABLE fx_deals ADD CONSTRAINT chk_fx_deals_status CHECK (status IN (
  'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
  'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT', 'PENDING_SETTLEMENT',
  'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT', 'CANCELLED',
  'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2'
));

DELETE FROM status_transition_rules WHERE from_status = 'PENDING_TP_REVIEW';
