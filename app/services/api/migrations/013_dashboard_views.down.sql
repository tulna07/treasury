-- 013: Drop Dashboard Views
BEGIN;

DROP VIEW IF EXISTS v_dashboard_recent_transactions;
DROP VIEW IF EXISTS v_dashboard_status_daily;
DROP VIEW IF EXISTS v_dashboard_module_distribution;
DROP VIEW IF EXISTS v_dashboard_daily_volume;
DROP VIEW IF EXISTS v_dashboard_summary_today;

COMMIT;
