-- ============================================================================
-- 013_bond_views.sql — SQL Views cho Module GTCG (Bond)
-- Treasury Management System — KienlongBank
-- ============================================================================

-- ---------------------------------------------------------------------------
-- View 1: v_bond_deals_list — Danh sách giao dịch GTCG
-- Dùng cho: Màn hình danh sách giao dịch GTCG (BRD §3.2.5)
-- Join: counterparties (tên đối tác), bond_catalog (mã TP), users (người tạo)
-- ---------------------------------------------------------------------------
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
    -- Mã trái phiếu: Govi lấy từ catalog, FI/CCTG lấy từ bond_code_manual
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
    bd.cancel_requested_by,
    bd.cancel_requested_at,
    bd.created_at,
    bd.created_by,
    -- Đối tác
    cp.code              AS counterparty_code,
    cp.full_name         AS counterparty_name,
    cp.short_name        AS counterparty_short_name,
    -- Người tạo
    u.full_name          AS created_by_name,
    u.username           AS created_by_username,
    -- Branch
    br.code              AS branch_code,
    br.name              AS branch_name
FROM bond_deals bd
    JOIN counterparties cp ON cp.id = bd.counterparty_id
    JOIN users u           ON u.id  = bd.created_by
    JOIN branches br       ON br.id = bd.branch_id
    LEFT JOIN bond_catalog bc ON bc.id = bd.bond_catalog_id
WHERE bd.deleted_at IS NULL;

COMMENT ON VIEW v_bond_deals_list IS
    'Danh sách giao dịch GTCG — join đối tác, catalog, người tạo. Dùng cho màn hình list (BRD §3.2.5)';

-- ---------------------------------------------------------------------------
-- View 2: v_bond_deals_pending_booking — Giao dịch chờ hạch toán
-- Dùng cho: KTTC dashboard — CV KTTC cấp 1 + LĐ KTTC cấp 2
-- Filter: status IN (PENDING_BOOKING, PENDING_CHIEF_ACCOUNTANT)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_bond_deals_pending_booking AS
SELECT
    bd.id,
    bd.deal_number,
    bd.bond_category,
    bd.trade_date,
    bd.value_date,
    bd.direction,
    bd.transaction_type,
    COALESCE(bc.bond_code, bd.bond_code_manual) AS bond_code_display,
    bd.issuer,
    bd.coupon_rate,
    bd.maturity_date,
    bd.quantity,
    bd.face_value,
    bd.settlement_price,
    bd.total_value,
    bd.portfolio_type,
    bd.payment_date,
    bd.remaining_tenor_days,
    bd.status,
    bd.note,
    bd.created_at,
    -- Đối tác
    cp.code              AS counterparty_code,
    cp.full_name         AS counterparty_name,
    -- Người tạo
    u.full_name          AS created_by_name,
    -- Booking level indicator
    CASE bd.status
        WHEN 'PENDING_BOOKING'          THEN 1  -- Cấp 1: CV KTTC
        WHEN 'PENDING_CHIEF_ACCOUNTANT' THEN 2  -- Cấp 2: LĐ KTTC
    END AS booking_level
FROM bond_deals bd
    JOIN counterparties cp ON cp.id = bd.counterparty_id
    JOIN users u           ON u.id  = bd.created_by
    LEFT JOIN bond_catalog bc ON bc.id = bd.bond_catalog_id
WHERE bd.deleted_at IS NULL
  AND bd.status IN ('PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT')
ORDER BY bd.trade_date ASC, bd.created_at ASC;

COMMENT ON VIEW v_bond_deals_pending_booking IS
    'Giao dịch GTCG chờ hạch toán (cấp 1 + cấp 2) — dùng cho dashboard P.KTTC';

-- ---------------------------------------------------------------------------
-- View 3: v_bond_inventory_summary — Tổng hợp tồn kho GTCG
-- Dùng cho: Màn hình tồn kho (BRD §5.4), kiểm tra khi bán
-- Join: bond_catalog (issuer, maturity, coupon)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_bond_inventory_summary AS
SELECT
    bi.id,
    bi.bond_code,
    bi.bond_category,
    bi.portfolio_type,
    bi.available_quantity,
    bi.acquisition_date,
    bi.acquisition_price,
    bi.version,
    bi.updated_at,
    -- Thông tin từ catalog (Govi Bond)
    bc.id                AS catalog_id,
    bc.issuer            AS catalog_issuer,
    bc.coupon_rate       AS catalog_coupon_rate,
    bc.issue_date        AS catalog_issue_date,
    bc.maturity_date     AS catalog_maturity_date,
    bc.face_value        AS catalog_face_value,
    bc.payment_frequency AS catalog_payment_frequency,
    -- Computed: tồn kho × mệnh giá = giá trị danh nghĩa
    bi.available_quantity * COALESCE(bc.face_value, 0) AS nominal_value,
    -- Người cập nhật cuối
    u.full_name          AS updated_by_name
FROM bond_inventory bi
    LEFT JOIN bond_catalog bc ON bc.id = bi.bond_catalog_id
    LEFT JOIN users u         ON u.id  = bi.updated_by
WHERE bi.available_quantity > 0;

COMMENT ON VIEW v_bond_inventory_summary IS
    'Tổng hợp tồn kho GTCG — join catalog info. Dùng cho inventory management + sell validation';

-- ---------------------------------------------------------------------------
-- View 4: v_bond_deals_with_approval_history — GD + lịch sử phê duyệt
-- Dùng cho: Màn hình chi tiết giao dịch — tab "Lịch sử" (BRD §8)
-- Join: approval_actions (timeline actions)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_bond_deals_with_approval_history AS
SELECT
    bd.id              AS deal_id,
    bd.deal_number,
    bd.bond_category,
    bd.status          AS current_status,
    bd.trade_date,
    bd.created_at      AS deal_created_at,
    -- Approval action
    aa.id              AS action_id,
    aa.action_type,
    aa.status_before,
    aa.status_after,
    aa.performed_at,
    aa.reason          AS action_reason,
    aa.metadata        AS action_metadata,
    -- Người thực hiện action
    au.full_name       AS performed_by_name,
    au.username        AS performed_by_username,
    -- Đơn vị người thực hiện
    ar.name            AS performed_by_role
FROM bond_deals bd
    LEFT JOIN approval_actions aa ON aa.deal_module = 'BOND'
                                  AND aa.deal_id = bd.id
    LEFT JOIN users au            ON au.id = aa.performed_by
    LEFT JOIN user_roles ur       ON ur.user_id = au.id
    LEFT JOIN roles ar            ON ar.id = ur.role_id
WHERE bd.deleted_at IS NULL
ORDER BY bd.id, aa.performed_at ASC;

COMMENT ON VIEW v_bond_deals_with_approval_history IS
    'Giao dịch GTCG + timeline phê duyệt — dùng cho tab Lịch sử (BRD §8)';
