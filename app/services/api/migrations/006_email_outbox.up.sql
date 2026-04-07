-- 006: Email Outbox — banking-grade transactional email with retry + throttling
CREATE TABLE IF NOT EXISTS email_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    to_addresses    TEXT[] NOT NULL,
    cc_addresses    TEXT[] DEFAULT '{}',
    from_address    TEXT NOT NULL,
    subject         TEXT NOT NULL,
    body_html       TEXT DEFAULT '',
    body_text       TEXT DEFAULT '',
    template_name   TEXT DEFAULT '',
    template_data   JSONB DEFAULT '{}',
    deal_module     TEXT DEFAULT '',
    deal_id         UUID,
    trigger_event   TEXT DEFAULT '',
    triggered_by    UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    retry_count     INT NOT NULL DEFAULT 0,
    max_retries     INT NOT NULL DEFAULT 3,
    next_retry_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error      TEXT DEFAULT '',
    idempotency_key TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ,
    failed_at       TIMESTAMPTZ
);

-- Index for worker polling: fetch pending/retry emails efficiently
CREATE INDEX IF NOT EXISTS idx_email_outbox_pending
    ON email_outbox (status, next_retry_at)
    WHERE status IN ('PENDING', 'RETRY');

-- Idempotency: prevent duplicate emails for the same event
CREATE UNIQUE INDEX IF NOT EXISTS idx_email_outbox_idempotency
    ON email_outbox (idempotency_key)
    WHERE idempotency_key != '';

-- Index for admin queries by status
CREATE INDEX IF NOT EXISTS idx_email_outbox_status
    ON email_outbox (status, created_at DESC);

COMMENT ON TABLE email_outbox IS 'Transactional email outbox with retry and throttling support';
