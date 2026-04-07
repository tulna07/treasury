-- 009 down: Revert v_bond_deals_list to version without counterparty_id

CREATE OR REPLACE VIEW v_bond_deals_list AS
SELECT
    bd.id,
    bd.deal_number,
    bd.bond_category,
    bd.trade_date,
    bd.order_date,
    bd.value_date,
    bd.direction,
    bd.transaction_type,
    bd.transaction_type_other,
    bd.bond_catalog_id,
    bd.bond_code_manual,
    COALESCE(bc.bond_code, bd.bond_code_manual) AS bond_code_display,
    bd.issuer,
    bd.coupon_rate,
    bd.issue_date,
    bd.maturity_date,
    bd.quantity,
    bd.face_value,
    bd.discount_rate,
    bd.clean_price,
    bd.settlement_price,
    bd.total_value,
    bd.portfolio_type,
    bd.payment_date,
    bd.remaining_tenor_days,
    bd.confirmation_method,
    bd.contract_prepared_by,
    bd.status,
    bd.note,
    bd.cloned_from_id,
    bd.cancel_reason,
    bd.cancel_requested_at,
    bd.created_at,
    bd.created_by,
    cp.code              AS counterparty_code,
    cp.full_name         AS counterparty_name,
    cp.short_name        AS counterparty_short_name,
    u.full_name          AS created_by_name,
    u.username           AS created_by_username,
    br.code              AS branch_code,
    br.name              AS branch_name
FROM bond_deals bd
    JOIN counterparties cp ON cp.id = bd.counterparty_id
    JOIN users u           ON u.id  = bd.created_by
    JOIN branches br       ON br.id = bd.branch_id
    LEFT JOIN bond_catalog bc ON bc.id = bd.bond_catalog_id
WHERE bd.deleted_at IS NULL;
