-- ============================================================================
-- 002_admin_views.up.sql — Views for Admin, Master Data & Audit
-- ============================================================================

-- View: Users with aggregated roles (avoids N+1 when listing users)
CREATE OR REPLACE VIEW v_user_with_roles AS
SELECT u.id, u.username, u.full_name, u.email, u.department, u.position,
       u.branch_id, b.name as branch_name, u.is_active, u.last_login_at,
       u.created_at, u.updated_at,
       COALESCE(array_agg(r.code ORDER BY r.code) FILTER (WHERE r.code IS NOT NULL), '{}') AS role_codes,
       COALESCE(array_agg(r.name ORDER BY r.code) FILTER (WHERE r.name IS NOT NULL), '{}') AS role_names
FROM users u
LEFT JOIN branches b ON u.branch_id = b.id
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
WHERE u.deleted_at IS NULL
GROUP BY u.id, b.name;

-- View: Audit log summary for dashboard stats
CREATE OR REPLACE VIEW v_audit_log_summary AS
SELECT
    date_trunc('day', performed_at) AS log_date,
    deal_module,
    action,
    COUNT(*) AS action_count
FROM audit_logs
GROUP BY date_trunc('day', performed_at), deal_module, action;

-- Add MASTER_DATA and AUDIT_LOG resources to the permissions check constraint
-- (The audit_logs table uses deal_module CHECK, but SYSTEM covers admin operations)
-- No schema change needed — SYSTEM is already in the CHECK constraint
