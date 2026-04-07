-- ============================================================================
-- 011_notification.sql — Notification System (1 table)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 31: notifications — Thông báo (in-app + email)
-- ---------------------------------------------------------------------------
CREATE TABLE notifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel             VARCHAR(10) NOT NULL,
    event_type          VARCHAR(50) NOT NULL,
    title               VARCHAR(500) NOT NULL,
    body                TEXT NOT NULL,
    deal_module         VARCHAR(20) NULL,
    deal_id             UUID NULL,
    is_read             BOOLEAN NOT NULL DEFAULT false,
    read_at             TIMESTAMPTZ NULL,
    delivery_status     VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    retry_count         SMALLINT NOT NULL DEFAULT 0,
    last_error          TEXT NULL,
    next_retry_at       TIMESTAMPTZ NULL,
    sent_at             TIMESTAMPTZ NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_notifications_channel CHECK (channel IN ('IN_APP', 'EMAIL')),
    CONSTRAINT chk_notifications_delivery CHECK (delivery_status IN ('PENDING', 'SENT', 'FAILED', 'RETRYING'))
);

CREATE INDEX idx_notifications_recipient_read ON notifications (recipient_id, is_read);
CREATE INDEX idx_notifications_created_at ON notifications (created_at);
CREATE INDEX idx_notifications_delivery ON notifications (delivery_status) WHERE delivery_status IN ('PENDING', 'RETRYING');
CREATE INDEX idx_notifications_next_retry ON notifications (next_retry_at) WHERE delivery_status = 'RETRYING';

COMMENT ON TABLE notifications IS 'Thông báo — in-app và email. Worker xử lý gửi email bất đồng bộ';
COMMENT ON COLUMN notifications.channel IS 'Kênh: IN_APP (hiển thị trên web) hoặc EMAIL';
COMMENT ON COLUMN notifications.event_type IS 'Loại sự kiện: DEAL_PENDING_APPROVAL, DEAL_REJECTED, DEAL_CANCELLED...';
COMMENT ON COLUMN notifications.delivery_status IS 'Trạng thái gửi: PENDING → SENT / FAILED / RETRYING';
COMMENT ON COLUMN notifications.retry_count IS 'Số lần retry — giới hạn tối đa trong application';
COMMENT ON COLUMN notifications.next_retry_at IS 'Thời điểm retry tiếp theo — exponential backoff';
