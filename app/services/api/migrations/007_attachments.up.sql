-- 007: Multi-file deal attachments (replaces single attachment_path/attachment_name on fx_deals)
CREATE TABLE IF NOT EXISTS deal_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module     VARCHAR(20) NOT NULL,
    deal_id         UUID NOT NULL,
    file_name       VARCHAR(500) NOT NULL,
    file_size       BIGINT NOT NULL,
    content_type    VARCHAR(200) NOT NULL,
    minio_bucket    VARCHAR(100) NOT NULL DEFAULT 'treasury-attachments',
    minio_key       VARCHAR(500) NOT NULL,
    uploaded_by     UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_attachments_deal ON deal_attachments(deal_module, deal_id);
CREATE INDEX idx_attachments_user ON deal_attachments(uploaded_by);
