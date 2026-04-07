-- Create export_audit_logs table for tracking all data exports
CREATE SCHEMA IF NOT EXISTS treasury;

CREATE TABLE IF NOT EXISTS treasury.export_audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    export_code     VARCHAR(30) NOT NULL UNIQUE,
    user_id         UUID NOT NULL,
    module          VARCHAR(20) NOT NULL,
    report_type     VARCHAR(50) NOT NULL,
    date_from       TIMESTAMPTZ NOT NULL,
    date_to         TIMESTAMPTZ NOT NULL,
    record_count    INTEGER NOT NULL DEFAULT 0,
    minio_bucket    VARCHAR(100) NOT NULL,
    minio_object_key VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT NOT NULL DEFAULT 0,
    file_checksum   VARCHAR(128) NOT NULL,
    client_ip       VARCHAR(45),
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_export_audit_logs_code ON treasury.export_audit_logs (export_code);
CREATE INDEX IF NOT EXISTS idx_export_audit_logs_user ON treasury.export_audit_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_audit_logs_module ON treasury.export_audit_logs (module, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_audit_logs_expires ON treasury.export_audit_logs (expires_at);
