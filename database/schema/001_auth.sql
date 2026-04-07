-- ============================================================================
-- 001_auth.sql — Authentication & User Management (8 tables)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- Trigger function: auto-update updated_at column
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION trigger_set_updated_at() IS 'Tự động cập nhật cột updated_at khi record bị thay đổi';

-- ---------------------------------------------------------------------------
-- Table 1: users — Người dùng hệ thống
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     VARCHAR(255) NULL,
    username        VARCHAR(100) NOT NULL,
    password_hash   VARCHAR(255) NULL,
    full_name       VARCHAR(255) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    branch_id       UUID NULL,
    department      VARCHAR(100) NULL,
    position        VARCHAR(100) NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    last_login_at   TIMESTAMPTZ NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ NULL,

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_external_id UNIQUE (external_id)
);

CREATE INDEX idx_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_external_id ON users (external_id) WHERE external_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_users_department ON users (department) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_branch_id ON users (branch_id) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE users IS 'Người dùng hệ thống Treasury — hỗ trợ cả standalone auth và Zitadel OIDC';
COMMENT ON COLUMN users.id IS 'UUID v7 — ứng dụng tạo, gen_random_uuid() là fallback';
COMMENT ON COLUMN users.external_id IS 'ID từ Zitadel IdP khi AUTH_MODE=zitadel';
COMMENT ON COLUMN users.username IS 'Tên đăng nhập — duy nhất trong hệ thống';
COMMENT ON COLUMN users.password_hash IS 'BCrypt hash — chỉ dùng khi AUTH_MODE=standalone';
COMMENT ON COLUMN users.branch_id IS 'Chi nhánh/phòng giao dịch — FK tới branches';
COMMENT ON COLUMN users.department IS 'Phòng ban (K.NV, P.KTTC, BP.TTQT, QLRR...)';
COMMENT ON COLUMN users.is_active IS 'Trạng thái hoạt động — false = bị vô hiệu hóa';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete — không bao giờ xóa cứng';

-- ---------------------------------------------------------------------------
-- Table 2: roles — Vai trò hệ thống (10 roles theo BRD v3)
-- ---------------------------------------------------------------------------
CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT NULL,
    scope       VARCHAR(50) NOT NULL DEFAULT 'ALL',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_roles_code UNIQUE (code),
    CONSTRAINT chk_roles_scope CHECK (scope IN ('ALL', 'MODULE_SPECIFIC', 'STEP_SPECIFIC'))
);

COMMENT ON TABLE roles IS 'Vai trò hệ thống — 10 vai trò theo BRD v3 section 2.4';
COMMENT ON COLUMN roles.code IS 'Mã vai trò: DEALER, DESK_HEAD, CENTER_DIRECTOR, DIVISION_HEAD, RISK_OFFICER, RISK_HEAD, ACCOUNTANT, CHIEF_ACCOUNTANT, SETTLEMENT_OFFICER, ADMIN';
COMMENT ON COLUMN roles.scope IS 'Phạm vi dữ liệu: ALL = toàn bộ, MODULE_SPECIFIC = theo module, STEP_SPECIFIC = theo bước duyệt';

-- ---------------------------------------------------------------------------
-- Table 3: permissions — Quyền hạn chi tiết (resource + action)
-- ---------------------------------------------------------------------------
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) NOT NULL,
    resource    VARCHAR(50) NOT NULL,
    action      VARCHAR(30) NOT NULL,
    description TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permissions_code UNIQUE (code),
    CONSTRAINT chk_permissions_resource CHECK (resource IN (
        'FX_DEAL', 'BOND_DEAL', 'MM_INTERBANK_DEAL', 'MM_OMO_REPO_DEAL',
        'CREDIT_LIMIT', 'INTERNATIONAL_PAYMENT', 'MASTER_DATA', 'SYSTEM'
    )),
    CONSTRAINT chk_permissions_action CHECK (action IN (
        'VIEW', 'CREATE', 'EDIT', 'APPROVE_L1', 'APPROVE_L2',
        'APPROVE_RISK_L1', 'APPROVE_RISK_L2', 'BOOK_L1', 'BOOK_L2',
        'SETTLE', 'RECALL', 'CANCEL_REQUEST', 'CANCEL_APPROVE_L1',
        'CANCEL_APPROVE_L2', 'CLONE', 'EXPORT', 'MANAGE'
    ))
);

CREATE INDEX idx_permissions_resource ON permissions (resource);
CREATE INDEX idx_permissions_action ON permissions (action);

COMMENT ON TABLE permissions IS 'Quyền hạn chi tiết — mô hình resource + action cho RBAC';
COMMENT ON COLUMN permissions.code IS 'Mã quyền: FX_DEAL.CREATE, BOND_DEAL.APPROVE_L1...';
COMMENT ON COLUMN permissions.resource IS 'Tài nguyên: FX_DEAL, BOND_DEAL, MM_INTERBANK_DEAL...';
COMMENT ON COLUMN permissions.action IS 'Hành động: VIEW, CREATE, APPROVE_L1, BOOK_L1...';

-- ---------------------------------------------------------------------------
-- Table 4: role_permissions — Gán quyền cho vai trò (RBAC)
-- ---------------------------------------------------------------------------
CREATE TABLE role_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT uq_role_permissions UNIQUE (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions (role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);

COMMENT ON TABLE role_permissions IS 'Bảng gán quyền cho vai trò — many-to-many giữa roles và permissions';

-- ---------------------------------------------------------------------------
-- Table 5: user_roles — Gán vai trò cho người dùng
-- ---------------------------------------------------------------------------
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by  UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT uq_user_roles UNIQUE (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles (role_id);

COMMENT ON TABLE user_roles IS 'Gán vai trò cho người dùng — mỗi user có thể có nhiều vai trò';

-- ---------------------------------------------------------------------------
-- Table 6: auth_configs — Cấu hình xác thực (standalone / Zitadel)
-- ---------------------------------------------------------------------------
CREATE TABLE auth_configs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_mode               VARCHAR(20) NOT NULL DEFAULT 'standalone',
    issuer_url              VARCHAR(500) NULL,
    client_id               VARCHAR(255) NULL,
    client_secret_encrypted TEXT NULL,
    scopes                  VARCHAR(500) NULL,
    auto_create_user        BOOLEAN NOT NULL DEFAULT true,
    sync_user_info          BOOLEAN NOT NULL DEFAULT true,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by              UUID NULL REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT chk_auth_configs_mode CHECK (auth_mode IN ('standalone', 'zitadel'))
);

CREATE TRIGGER trg_auth_configs_updated_at
    BEFORE UPDATE ON auth_configs
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

COMMENT ON TABLE auth_configs IS 'Cấu hình xác thực — chuyển đổi standalone ↔ Zitadel không cần migration';
COMMENT ON COLUMN auth_configs.auth_mode IS 'Chế độ: standalone (tự quản lý) hoặc zitadel (OIDC)';
COMMENT ON COLUMN auth_configs.client_secret_encrypted IS 'Client secret đã mã hóa — KHÔNG lưu plain text';

-- ---------------------------------------------------------------------------
-- Table 7: external_role_mappings — Ánh xạ nhóm Zitadel → vai trò nội bộ
-- ---------------------------------------------------------------------------
CREATE TABLE external_role_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_group  VARCHAR(255) NOT NULL,
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_external_role_mappings UNIQUE (external_group, role_id)
);

COMMENT ON TABLE external_role_mappings IS 'Ánh xạ nhóm/vai trò từ Zitadel IdP sang vai trò nội bộ Treasury';
COMMENT ON COLUMN external_role_mappings.external_group IS 'Tên group/role trong Zitadel';

-- ---------------------------------------------------------------------------
-- Table 8: user_sessions — Quản lý phiên đăng nhập (standalone mode)
-- ---------------------------------------------------------------------------
CREATE TABLE user_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL,
    ip_address  INET NULL,
    user_agent  TEXT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_user_sessions_token_hash UNIQUE (token_hash)
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_token_hash ON user_sessions (token_hash);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions (expires_at);

COMMENT ON TABLE user_sessions IS 'Phiên đăng nhập — chỉ dùng khi AUTH_MODE=standalone, Zitadel dùng IdP session';
COMMENT ON COLUMN user_sessions.token_hash IS 'SHA-256 hash của session/refresh token — KHÔNG lưu token gốc';
COMMENT ON COLUMN user_sessions.revoked_at IS 'Thời điểm thu hồi phiên (logout/force-expire)';

-- ---------------------------------------------------------------------------
-- Add FK from users.branch_id after branches table is created (see 002_organization.sql)
-- ALTER TABLE users ADD CONSTRAINT fk_users_branch_id FOREIGN KEY (branch_id) REFERENCES branches(id);
-- ---------------------------------------------------------------------------
