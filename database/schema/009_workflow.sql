-- ============================================================================
-- 009_workflow.sql — Workflow Engine (3 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Table 27: deal_sequences — Sinh mã giao dịch gapless
-- ---------------------------------------------------------------------------
CREATE TABLE deal_sequences (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module          VARCHAR(20) NOT NULL,
    prefix          VARCHAR(10) NOT NULL,
    date_partition  DATE NOT NULL,
    last_sequence   BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_deal_sequences UNIQUE (module, prefix, date_partition),
    CONSTRAINT chk_deal_sequences_module CHECK (module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO')),
    CONSTRAINT chk_deal_sequences_seq CHECK (last_sequence >= 0)
);

CREATE TRIGGER trg_deal_sequences_updated_at
    BEFORE UPDATE ON deal_sequences
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE deal_sequences IS 'Sinh mã giao dịch gapless — advisory lock per module+date';
COMMENT ON COLUMN deal_sequences.module IS 'Module: FX, BOND, MM_INTERBANK, MM_OMO_REPO';
COMMENT ON COLUMN deal_sequences.prefix IS 'Tiền tố: FX, G, F, MM, OMO, RK';
COMMENT ON COLUMN deal_sequences.date_partition IS 'Ngày làm việc — reset sequence mỗi ngày';
COMMENT ON COLUMN deal_sequences.last_sequence IS 'Số thứ tự cuối cùng — SELECT FOR UPDATE để tăng';

-- ---------------------------------------------------------------------------
-- Table 28: approval_actions — Lịch sử phê duyệt (append-only)
-- ---------------------------------------------------------------------------
CREATE TABLE approval_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module     VARCHAR(10) NOT NULL,
    deal_id         UUID NOT NULL,
    action_type     VARCHAR(30) NOT NULL,
    status_before   VARCHAR(30) NOT NULL,
    status_after    VARCHAR(30) NOT NULL,
    performed_by    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    performed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reason          TEXT NULL,
    metadata        JSONB NULL,

    CONSTRAINT chk_approval_actions_module CHECK (deal_module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO')),
    CONSTRAINT chk_approval_actions_type CHECK (action_type IN (
        'DESK_HEAD_APPROVE', 'DESK_HEAD_RETURN',
        'DIRECTOR_APPROVE', 'DIRECTOR_REJECT',
        'RISK_OFFICER_APPROVE', 'RISK_OFFICER_REJECT',
        'RISK_HEAD_APPROVE', 'RISK_HEAD_REJECT',
        'ACCOUNTANT_APPROVE', 'ACCOUNTANT_REJECT',
        'CHIEF_ACCOUNTANT_APPROVE', 'CHIEF_ACCOUNTANT_REJECT',
        'SETTLEMENT_APPROVE', 'SETTLEMENT_REJECT',
        'DEALER_RECALL', 'DESK_HEAD_RECALL',
        'CANCEL_REQUEST', 'CANCEL_DESK_HEAD_APPROVE', 'CANCEL_DESK_HEAD_REJECT',
        'CANCEL_DIVISION_HEAD_APPROVE', 'CANCEL_DIVISION_HEAD_REJECT'
    ))
);

CREATE INDEX idx_approval_actions_deal ON approval_actions (deal_module, deal_id);
CREATE INDEX idx_approval_actions_performed_by ON approval_actions (performed_by);
CREATE INDEX idx_approval_actions_performed_at ON approval_actions (performed_at);

COMMENT ON TABLE approval_actions IS 'Lịch sử phê duyệt — append-only, không sửa/xóa. Phục vụ workflow engine';
COMMENT ON COLUMN approval_actions.deal_module IS 'Module: FX, BOND, MM_INTERBANK, MM_OMO_REPO';
COMMENT ON COLUMN approval_actions.action_type IS 'Loại hành động: DESK_HEAD_APPROVE, DIRECTOR_REJECT, DEALER_RECALL...';
COMMENT ON COLUMN approval_actions.status_before IS 'Trạng thái trước hành động';
COMMENT ON COLUMN approval_actions.status_after IS 'Trạng thái sau hành động';
COMMENT ON COLUMN approval_actions.reason IS 'Lý do — bắt buộc cho reject, recall, cancel';
COMMENT ON COLUMN approval_actions.metadata IS 'Dữ liệu bổ sung (vd: snapshot hạn mức)';

-- ---------------------------------------------------------------------------
-- Table 29: status_transition_rules — Quy tắc chuyển trạng thái (state machine)
-- ---------------------------------------------------------------------------
CREATE TABLE status_transition_rules (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deal_module             VARCHAR(20) NOT NULL,
    from_status             VARCHAR(30) NOT NULL,
    to_status               VARCHAR(30) NOT NULL,
    required_role           VARCHAR(50) NOT NULL,
    requires_reason         BOOLEAN NOT NULL DEFAULT false,
    requires_confirmation   BOOLEAN NOT NULL DEFAULT false,
    is_active               BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT uq_status_transitions UNIQUE (deal_module, from_status, to_status, required_role),
    CONSTRAINT chk_status_transitions_module CHECK (deal_module IN ('FX', 'BOND', 'MM_INTERBANK', 'MM_OMO_REPO'))
);

CREATE INDEX idx_status_transitions_module ON status_transition_rules (deal_module);
CREATE INDEX idx_status_transitions_from ON status_transition_rules (deal_module, from_status) WHERE is_active = true;

COMMENT ON TABLE status_transition_rules IS 'Quy tắc state machine — config-driven, thay đổi flow bằng data không cần code';
COMMENT ON COLUMN status_transition_rules.deal_module IS 'Module: FX, BOND, MM_INTERBANK, MM_OMO_REPO';
COMMENT ON COLUMN status_transition_rules.from_status IS 'Trạng thái hiện tại';
COMMENT ON COLUMN status_transition_rules.to_status IS 'Trạng thái đích';
COMMENT ON COLUMN status_transition_rules.required_role IS 'Vai trò yêu cầu (FK logic tới roles.code)';
COMMENT ON COLUMN status_transition_rules.requires_reason IS 'Bắt buộc nhập lý do (reject, recall, cancel)';
COMMENT ON COLUMN status_transition_rules.requires_confirmation IS 'Hiển thị popup xác nhận trước khi thực hiện';
