-- 013: Dashboard Views — real-time aggregation views for Treasury dashboard
BEGIN;

-- ---------------------------------------------------------------------------
-- 1. v_dashboard_summary_today — Single-row summary of today's activity
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_dashboard_summary_today AS
WITH fx_today AS (
    SELECT *
    FROM fx_deals
    WHERE trade_date = CURRENT_DATE
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
),
bond_today AS (
    SELECT *
    FROM bond_deals
    WHERE trade_date = CURRENT_DATE
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
),
mm_today AS (
    SELECT *
    FROM mm_interbank_deals
    WHERE trade_date = CURRENT_DATE
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
),
omo_today AS (
    SELECT *
    FROM mm_omo_repo_deals
    WHERE trade_date = CURRENT_DATE
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
),
all_statuses AS (
    SELECT status FROM fx_today
    UNION ALL SELECT status FROM bond_today
    UNION ALL SELECT status FROM mm_today
    UNION ALL SELECT status FROM omo_today
),
fx_agg AS (
    SELECT
        COALESCE(COUNT(*), 0)                                                       AS cnt,
        COALESCE(SUM(notional_amount), 0)::float8 AS total_notional,
        COALESCE(COUNT(*) FILTER (WHERE direction IN ('BUY', 'BUY_SELL')), 0)       AS buy_count,
        COALESCE(COUNT(*) FILTER (WHERE direction IN ('SELL', 'SELL_BUY')), 0)      AS sell_count
    FROM fx_today
),
mm_outstanding AS (
    SELECT
        COALESCE(SUM(principal_amount), 0)::float8 AS outstanding_amount,
        COALESCE(COUNT(*), 0)               AS active_count
    FROM mm_interbank_deals
    WHERE maturity_date >= CURRENT_DATE
      AND status NOT IN ('CANCELLED', 'REJECTED')
      AND deleted_at IS NULL
),
bond_portfolio AS (
    SELECT COALESCE(SUM(total_value), 0)::float8 AS portfolio_value
    FROM bond_deals
    WHERE maturity_date >= CURRENT_DATE
      AND status NOT IN ('CANCELLED', 'REJECTED')
      AND deleted_at IS NULL
),
ttqt_agg AS (
    SELECT COALESCE(COUNT(*), 0) AS pending_count
    FROM international_payments
    WHERE settlement_status = 'PENDING'
),
credit_agg AS (
    SELECT COALESCE(COUNT(DISTINCT counterparty_id), 0) AS counterparties_with_limits
    FROM credit_limits
    WHERE is_current = true
)
SELECT
    -- Deal counts today
    COALESCE((SELECT COUNT(*) FROM all_statuses), 0)                                        AS total_deals_today,
    COALESCE((SELECT COUNT(*) FROM all_statuses WHERE status = 'COMPLETED'), 0)             AS completed_today,
    COALESCE((SELECT COUNT(*) FROM all_statuses WHERE status LIKE 'PENDING_%'), 0)          AS pending_today,

    -- FX
    fa.total_notional       AS fx_total_notional,
    fa.buy_count            AS fx_buy_count,
    fa.sell_count           AS fx_sell_count,

    -- Money Market
    mo.outstanding_amount   AS mm_outstanding,
    mo.active_count         AS mm_active_count,

    -- Bond
    bp.portfolio_value      AS bond_portfolio_value,

    -- TTQT
    ta.pending_count        AS ttqt_pending_count,

    -- Credit Limits
    ca.counterparties_with_limits

FROM fx_agg fa
CROSS JOIN mm_outstanding mo
CROSS JOIN bond_portfolio bp
CROSS JOIN ttqt_agg ta
CROSS JOIN credit_agg ca;


-- ---------------------------------------------------------------------------
-- 2. v_dashboard_daily_volume — 7-day rolling volume by module
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_dashboard_daily_volume AS
WITH date_series AS (
    SELECT d::date AS trade_date
    FROM generate_series(
        CURRENT_DATE - INTERVAL '6 days',
        CURRENT_DATE,
        INTERVAL '1 day'
    ) AS d
),
fx_daily AS (
    SELECT trade_date,
           COUNT(*)                         AS fx_count,
           COALESCE(SUM(notional_amount), 0)::float8 AS fx_volume
    FROM fx_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '6 days'
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
    GROUP BY trade_date
),
mm_daily AS (
    SELECT trade_date,
           COUNT(*)                          AS mm_count,
           COALESCE(SUM(principal_amount), 0)::float8 AS mm_volume
    FROM mm_interbank_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '6 days'
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
    GROUP BY trade_date
),
bond_daily AS (
    SELECT trade_date,
           COUNT(*)                        AS bond_count,
           COALESCE(SUM(total_value), 0)::float8 AS bond_volume
    FROM bond_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '6 days'
      AND status != 'CANCELLED'
      AND deleted_at IS NULL
    GROUP BY trade_date
)
SELECT
    ds.trade_date,
    COALESCE(fx.fx_count, 0)    AS fx_count,
    COALESCE(fx.fx_volume, 0)::float8 AS fx_volume,
    COALESCE(mm.mm_count, 0)    AS mm_count,
    COALESCE(mm.mm_volume, 0)::float8 AS mm_volume,
    COALESCE(bd.bond_count, 0)  AS bond_count,
    COALESCE(bd.bond_volume, 0)::float8 AS bond_volume
FROM date_series ds
LEFT JOIN fx_daily fx   ON fx.trade_date = ds.trade_date
LEFT JOIN mm_daily mm   ON mm.trade_date = ds.trade_date
LEFT JOIN bond_daily bd ON bd.trade_date = ds.trade_date
ORDER BY ds.trade_date;


-- ---------------------------------------------------------------------------
-- 3. v_dashboard_module_distribution — Deal count by module (all time)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_dashboard_module_distribution AS
SELECT
    COALESCE((SELECT COUNT(*) FROM fx_deals             WHERE status != 'CANCELLED' AND deleted_at IS NULL), 0)  AS fx_count,
    COALESCE((SELECT COUNT(*) FROM bond_deals           WHERE status != 'CANCELLED' AND deleted_at IS NULL), 0)  AS bond_count,
    COALESCE((SELECT COUNT(*) FROM mm_interbank_deals   WHERE status != 'CANCELLED' AND deleted_at IS NULL), 0)
  + COALESCE((SELECT COUNT(*) FROM mm_omo_repo_deals    WHERE status != 'CANCELLED' AND deleted_at IS NULL), 0)  AS mm_count,
    COALESCE((SELECT COUNT(*) FROM international_payments WHERE settlement_status != 'REJECTED'), 0)             AS ttqt_count;


-- ---------------------------------------------------------------------------
-- 4. v_dashboard_status_daily — 5-day status breakdown across all modules
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_dashboard_status_daily AS
WITH date_series AS (
    SELECT d::date AS trade_date
    FROM generate_series(
        CURRENT_DATE - INTERVAL '4 days',
        CURRENT_DATE,
        INTERVAL '1 day'
    ) AS d
),
all_deals AS (
    SELECT trade_date, status
    FROM fx_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '4 days'
      AND deleted_at IS NULL

    UNION ALL

    SELECT trade_date, status
    FROM bond_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '4 days'
      AND deleted_at IS NULL

    UNION ALL

    SELECT trade_date, status
    FROM mm_interbank_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '4 days'
      AND deleted_at IS NULL

    UNION ALL

    SELECT trade_date, status
    FROM mm_omo_repo_deals
    WHERE trade_date >= CURRENT_DATE - INTERVAL '4 days'
      AND deleted_at IS NULL
),
daily_agg AS (
    SELECT
        trade_date,
        COUNT(*) FILTER (WHERE status = 'OPEN')                AS open_count,
        COUNT(*) FILTER (WHERE status LIKE 'PENDING_%')        AS pending_count,
        COUNT(*) FILTER (WHERE status = 'COMPLETED')           AS completed_count,
        COUNT(*) FILTER (WHERE status = 'CANCELLED')           AS cancelled_count
    FROM all_deals
    GROUP BY trade_date
)
SELECT
    ds.trade_date,
    COALESCE(da.open_count, 0)      AS open_count,
    COALESCE(da.pending_count, 0)   AS pending_count,
    COALESCE(da.completed_count, 0) AS completed_count,
    COALESCE(da.cancelled_count, 0) AS cancelled_count
FROM date_series ds
LEFT JOIN daily_agg da ON da.trade_date = ds.trade_date
ORDER BY ds.trade_date;


-- ---------------------------------------------------------------------------
-- 5. v_dashboard_recent_transactions — Latest 10 across all modules
-- ---------------------------------------------------------------------------
CREATE OR REPLACE VIEW v_dashboard_recent_transactions AS
SELECT * FROM (
    (
        SELECT
            id,
            deal_number     AS ticket,
            'FX'            AS module,
            deal_type       AS deal_type,
            notional_amount::float8 AS amount,
            currency_code   AS currency,
            status,
            created_at,
            created_by
        FROM fx_deals
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT 10
    )
    UNION ALL
    (
        SELECT
            id,
            deal_number     AS ticket,
            'BOND'          AS module,
            bond_category   AS deal_type,
            total_value     AS amount,
            NULL            AS currency,
            status,
            created_at,
            created_by
        FROM bond_deals
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT 10
    )
    UNION ALL
    (
        SELECT
            id,
            deal_number     AS ticket,
            'MM'            AS module,
            direction       AS deal_type,
            principal_amount::float8 AS amount,
            currency_code   AS currency,
            status,
            created_at,
            created_by
        FROM mm_interbank_deals
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT 10
    )
    UNION ALL
    (
        SELECT
            id,
            deal_number     AS ticket,
            'MM'            AS module,
            deal_subtype    AS deal_type,
            notional_amount::float8 AS amount,
            NULL            AS currency,
            status,
            created_at,
            created_by
        FROM mm_omo_repo_deals
        WHERE deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT 10
    )
    UNION ALL
    (
        SELECT
            id,
            ticket_display  AS ticket,
            'TTQT'          AS module,
            source_module   AS deal_type,
            amount,
            currency_code   AS currency,
            settlement_status AS status,
            created_at,
            NULL::uuid      AS created_by
        FROM international_payments
        ORDER BY created_at DESC
        LIMIT 10
    )
) combined
ORDER BY created_at DESC
LIMIT 10;

COMMIT;
