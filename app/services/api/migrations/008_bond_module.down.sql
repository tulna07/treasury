-- 008: Rollback Module GTCG (Bond)
DROP VIEW IF EXISTS v_bond_deals_with_approval_history;
DROP VIEW IF EXISTS v_bond_inventory_summary;
DROP VIEW IF EXISTS v_bond_deals_pending_booking;
DROP VIEW IF EXISTS v_bond_deals_list;
DROP TABLE IF EXISTS bond_inventory;
DROP TABLE IF EXISTS bond_deals;
