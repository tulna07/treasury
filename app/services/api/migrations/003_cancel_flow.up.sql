-- Extend fx_deals status CHECK constraint to include cancel states
ALTER TABLE fx_deals DROP CONSTRAINT IF EXISTS chk_fx_deals_status;
ALTER TABLE fx_deals ADD CONSTRAINT chk_fx_deals_status CHECK (status IN (
  'OPEN', 'PENDING_L2_APPROVAL', 'REJECTED',
  'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT', 'PENDING_SETTLEMENT',
  'COMPLETED', 'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT', 'CANCELLED',
  'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2'
));

-- Cancel metadata table: stores original status before cancel request
-- so we can revert on rejection
CREATE TABLE IF NOT EXISTS cancel_metadata (
    deal_id UUID NOT NULL,
    deal_module VARCHAR(20) NOT NULL DEFAULT 'FX',
    original_status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (deal_id, deal_module)
);

-- Add new cancel statuses to the deal_status enum if it exists as a check constraint
-- Since we use VARCHAR for status, no ALTER TYPE needed — just ensure the views handle it

-- Insert cancel flow transition rules (if status_transition_rules table exists)
INSERT INTO status_transition_rules (deal_module, from_status, to_status, required_role)
VALUES
    ('FX', 'COMPLETED', 'PENDING_CANCEL_L1', 'DEALER'),
    ('FX', 'PENDING_SETTLEMENT', 'PENDING_CANCEL_L1', 'DEALER'),
    ('FX', 'PENDING_CANCEL_L1', 'PENDING_CANCEL_L2', 'DESK_HEAD'),
    ('FX', 'PENDING_CANCEL_L1', 'COMPLETED', 'DESK_HEAD'),
    ('FX', 'PENDING_CANCEL_L1', 'PENDING_SETTLEMENT', 'DESK_HEAD'),
    ('FX', 'PENDING_CANCEL_L2', 'CANCELLED', 'CENTER_DIRECTOR'),
    ('FX', 'PENDING_CANCEL_L2', 'CANCELLED', 'DIVISION_HEAD'),
    ('FX', 'PENDING_CANCEL_L2', 'COMPLETED', 'CENTER_DIRECTOR'),
    ('FX', 'PENDING_CANCEL_L2', 'COMPLETED', 'DIVISION_HEAD'),
    ('FX', 'PENDING_CANCEL_L2', 'PENDING_SETTLEMENT', 'CENTER_DIRECTOR'),
    ('FX', 'PENDING_CANCEL_L2', 'PENDING_SETTLEMENT', 'DIVISION_HEAD')
ON CONFLICT DO NOTHING;
