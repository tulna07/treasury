-- ============================================================================
-- 010_credit_limits.down.sql — Rollback Credit Limit Module
-- ============================================================================

DROP VIEW IF EXISTS v_daily_limit_summary;
DROP TABLE IF EXISTS limit_approval_records;
DROP TABLE IF EXISTS limit_utilization_snapshots;
DROP TABLE IF EXISTS credit_limits;
