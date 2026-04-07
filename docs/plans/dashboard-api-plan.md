# Dashboard API Plan — Treasury Management System

> Created: 05/04/2026 | Status: DRAFT | Author: KAI

## 1. SQL Views (Migration 013)

### View 1: `v_dashboard_summary_today`
Tổng hợp số liệu hôm nay — 1 row duy nhất, hiệu năng cao.

```sql
CREATE OR REPLACE VIEW v_dashboard_summary_today AS
SELECT
  -- Card 1: Tổng quan
  (SELECT COUNT(*) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE) +
  (SELECT COUNT(*) FROM bond_deals WHERE DATE(created_at) = CURRENT_DATE) +
  (SELECT COUNT(*) FROM mm_interbank_deals WHERE DATE(created_at) = CURRENT_DATE) +
  (SELECT COUNT(*) FROM mm_omo_repo_deals WHERE DATE(created_at) = CURRENT_DATE)
    AS total_deals_today,

  (SELECT COUNT(*) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE AND status = 'COMPLETED') +
  (SELECT COUNT(*) FROM bond_deals WHERE DATE(created_at) = CURRENT_DATE AND status = 'COMPLETED') +
  (SELECT COUNT(*) FROM mm_interbank_deals WHERE DATE(created_at) = CURRENT_DATE AND status = 'COMPLETED') +
  (SELECT COUNT(*) FROM mm_omo_repo_deals WHERE DATE(created_at) = CURRENT_DATE AND status = 'COMPLETED')
    AS completed_today,

  (SELECT COUNT(*) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE AND status LIKE 'PENDING_%') +
  (SELECT COUNT(*) FROM bond_deals WHERE DATE(created_at) = CURRENT_DATE AND status LIKE 'PENDING_%') +
  (SELECT COUNT(*) FROM mm_interbank_deals WHERE DATE(created_at) = CURRENT_DATE AND status LIKE 'PENDING_%') +
  (SELECT COUNT(*) FROM mm_omo_repo_deals WHERE DATE(created_at) = CURRENT_DATE AND status LIKE 'PENDING_%')
    AS pending_today,

  -- Card 2: FX
  COALESCE((SELECT SUM(
    CASE WHEN buy_currency = 'VND' THEN buy_amount ELSE sell_amount END
  ) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE AND status NOT IN ('CANCELLED')), 0)
    AS fx_total_vnd,
  (SELECT COUNT(*) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE AND direction = 'BUY')
    AS fx_buy_count,
  (SELECT COUNT(*) FROM fx_deals WHERE DATE(created_at) = CURRENT_DATE AND direction = 'SELL')
    AS fx_sell_count,

  -- Card 3: MM & GTCG
  COALESCE((SELECT SUM(principal_amount) FROM mm_interbank_deals
    WHERE status NOT IN ('CANCELLED','REJECTED') AND maturity_date >= CURRENT_DATE), 0)
    AS mm_outstanding,
  (SELECT COUNT(*) FROM mm_interbank_deals
    WHERE status NOT IN ('CANCELLED','REJECTED','COMPLETED') AND maturity_date >= CURRENT_DATE)
    AS mm_active_count,
  COALESCE((SELECT SUM(par_value) FROM bond_deals
    WHERE status NOT IN ('CANCELLED','REJECTED') AND maturity_date >= CURRENT_DATE), 0)
    AS bond_portfolio_value,

  -- Card 4: Hạn mức & TTQT
  COALESCE((SELECT AVG(
    CASE WHEN approved_limit > 0
      THEN ((approved_limit - COALESCE(utilized_amount, 0)) / approved_limit) * 100
      ELSE 100
    END
  ) FROM credit_limits WHERE is_active = true), 0)
    AS avg_limit_remaining_pct,
  (SELECT COUNT(DISTINCT counterparty_id) FROM credit_limits
    WHERE is_active = true AND approved_limit > 0
    AND (COALESCE(utilized_amount, 0) / approved_limit) > 0.8)
    AS counterparties_over_80pct,
  (SELECT COUNT(*) FROM international_payments WHERE settlement_status = 'PENDING')
    AS ttqt_pending_count;
```

### View 2: `v_dashboard_daily_volume`
7-ngày volume theo module — cho Area Chart.

```sql
CREATE OR REPLACE VIEW v_dashboard_daily_volume AS
SELECT
  d.dt AS trade_date,
  COALESCE(fx.cnt, 0) AS fx_count,
  COALESCE(fx.vol, 0) AS fx_volume_vnd,
  COALESCE(mm.cnt, 0) AS mm_count,
  COALESCE(mm.vol, 0) AS mm_volume,
  COALESCE(bond.cnt, 0) AS bond_count,
  COALESCE(bond.vol, 0) AS bond_volume
FROM generate_series(
  CURRENT_DATE - INTERVAL '6 days', CURRENT_DATE, '1 day'
) AS d(dt)
LEFT JOIN (
  SELECT DATE(created_at) AS dt, COUNT(*) AS cnt,
    SUM(CASE WHEN buy_currency='VND' THEN buy_amount ELSE sell_amount END) AS vol
  FROM fx_deals WHERE status NOT IN ('CANCELLED')
  GROUP BY DATE(created_at)
) fx ON fx.dt = d.dt
LEFT JOIN (
  SELECT DATE(created_at) AS dt, COUNT(*) AS cnt, SUM(principal_amount) AS vol
  FROM mm_interbank_deals WHERE status NOT IN ('CANCELLED')
  GROUP BY DATE(created_at)
) mm ON mm.dt = d.dt
LEFT JOIN (
  SELECT DATE(created_at) AS dt, COUNT(*) AS cnt, SUM(par_value) AS vol
  FROM bond_deals WHERE status NOT IN ('CANCELLED')
  GROUP BY DATE(created_at)
) bond ON bond.dt = d.dt;
```

### View 3: `v_dashboard_module_distribution`
Phân bổ theo loại GD — cho Pie Chart.

```sql
CREATE OR REPLACE VIEW v_dashboard_module_distribution AS
SELECT
  (SELECT COUNT(*) FROM fx_deals WHERE status NOT IN ('CANCELLED')) AS fx_count,
  (SELECT COUNT(*) FROM bond_deals WHERE status NOT IN ('CANCELLED')) AS bond_count,
  (SELECT COUNT(*) FROM mm_interbank_deals WHERE status NOT IN ('CANCELLED')) +
  (SELECT COUNT(*) FROM mm_omo_repo_deals WHERE status NOT IN ('CANCELLED')) AS mm_count,
  (SELECT COUNT(*) FROM international_payments) AS ttqt_count;
```

### View 4: `v_dashboard_status_daily`
Trạng thái GD 5 ngày — cho Stacked Bar Chart.

```sql
CREATE OR REPLACE VIEW v_dashboard_status_daily AS
SELECT d.dt AS trade_date,
  COALESCE(SUM(CASE WHEN s.status = 'OPEN' THEN 1 ELSE 0 END), 0) AS open_count,
  COALESCE(SUM(CASE WHEN s.status LIKE 'PENDING_%' THEN 1 ELSE 0 END), 0) AS pending_count,
  COALESCE(SUM(CASE WHEN s.status = 'COMPLETED' THEN 1 ELSE 0 END), 0) AS completed_count,
  COALESCE(SUM(CASE WHEN s.status = 'CANCELLED' THEN 1 ELSE 0 END), 0) AS cancelled_count
FROM generate_series(CURRENT_DATE - INTERVAL '4 days', CURRENT_DATE, '1 day') AS d(dt)
LEFT JOIN (
  SELECT DATE(created_at) AS dt, status FROM fx_deals
  UNION ALL
  SELECT DATE(created_at), status FROM bond_deals
  UNION ALL
  SELECT DATE(created_at), status FROM mm_interbank_deals
  UNION ALL
  SELECT DATE(created_at), status FROM mm_omo_repo_deals
) s ON s.dt = d.dt
GROUP BY d.dt ORDER BY d.dt;
```

### View 5: `v_dashboard_recent_transactions`
5 giao dịch gần nhất — cho Table.

```sql
CREATE OR REPLACE VIEW v_dashboard_recent_transactions AS
(
  SELECT id, deal_number AS ticket, 'FX' AS module,
    direction || ' ' || buy_currency AS deal_type,
    CASE WHEN buy_currency='VND' THEN buy_amount ELSE sell_amount END AS amount_vnd,
    buy_currency AS currency, status, created_at, created_by
  FROM fx_deals ORDER BY created_at DESC LIMIT 5
)
UNION ALL
(
  SELECT id, deal_number, 'BOND',
    'Mua ' || bond_code, par_value, 'VND', status, created_at, created_by
  FROM bond_deals ORDER BY created_at DESC LIMIT 5
)
UNION ALL
(
  SELECT id, deal_number, 'MM',
    direction || ' ' || currency_code, principal_amount, currency_code, status, created_at, created_by
  FROM mm_interbank_deals ORDER BY created_at DESC LIMIT 5
)
ORDER BY created_at DESC LIMIT 5;
```

## 2. Indexes (nếu chưa có)

```sql
-- Tối ưu query theo ngày
CREATE INDEX IF NOT EXISTS idx_fx_deals_created_date ON fx_deals (DATE(created_at));
CREATE INDEX IF NOT EXISTS idx_mm_interbank_created_date ON mm_interbank_deals (DATE(created_at));
CREATE INDEX IF NOT EXISTS idx_bond_deals_created_date ON bond_deals (DATE(created_at));
CREATE INDEX IF NOT EXISTS idx_mm_omo_repo_created_date ON mm_omo_repo_deals (DATE(created_at));

-- MM maturity filter
CREATE INDEX IF NOT EXISTS idx_mm_interbank_maturity ON mm_interbank_deals (maturity_date) WHERE status NOT IN ('CANCELLED','REJECTED');
```

## 3. API Design

### Endpoint: `GET /api/v1/dashboard`

**Response shape:**
```json
{
  "success": true,
  "data": {
    "summary": {
      "total_deals_today": 24,
      "completed_today": 17,
      "pending_today": 7,
      "fx_total_vnd": 1250000000000,
      "fx_buy_count": 8,
      "fx_sell_count": 4,
      "mm_outstanding": 850000000000,
      "mm_active_count": 6,
      "bond_portfolio_value": 350000000000,
      "avg_limit_remaining_pct": 68.5,
      "counterparties_over_80pct": 2,
      "ttqt_pending_count": 1
    },
    "daily_volume": [
      { "trade_date": "2026-03-30", "fx_count": 5, "fx_volume_vnd": 450000000000, "mm_count": 3, "mm_volume": 320000000000 }
    ],
    "module_distribution": {
      "fx_count": 42, "bond_count": 18, "mm_count": 28, "ttqt_count": 12
    },
    "status_daily": [
      { "trade_date": "2026-04-01", "open": 3, "pending": 5, "completed": 12, "cancelled": 1 }
    ],
    "recent_transactions": [
      { "id": "...", "ticket": "FX-20260405-0001", "module": "FX", "deal_type": "BUY USD", "amount_vnd": 500000000, "status": "COMPLETED", "created_at": "..." }
    ]
  }
}
```

### Permission & Data Scope

```
Permission check: Mỗi phần chỉ trả nếu user có quyền VIEW tương ứng
┌─────────────────────┬──────────────────────────────┐
│ Dashboard section    │ Required permission          │
├─────────────────────┼──────────────────────────────┤
│ FX data             │ FX_DEAL.VIEW                 │
│ Bond data           │ BOND_DEAL.VIEW               │
│ MM data             │ MM_INTERBANK_DEAL.VIEW       │
│ TTQT data           │ INTERNATIONAL_PAYMENT.VIEW   │
│ Credit Limit data   │ CREDIT_LIMIT.VIEW            │
│ Summary (aggregate) │ Bất kỳ 1 VIEW permission     │
└─────────────────────┴──────────────────────────────┘

Data Scope (BRD §6.2):
- DEALER:     chỉ deal mình tạo (created_by = current_user_id)
- DESK_HEAD:  deal trong team (created_by IN team_members)  
- DIRECTOR+:  tất cả deals
- ADMIN:      tất cả (view only)
```

**Implementation:**
- Service nhận roles[] + userID từ context
- Xây WHERE clause dựa trên role → truyền vào view query
- Views dùng regular (non-materialized) vì data real-time
- Nếu cần tối ưu hơn → chuyển sang materialized view + refresh cron

## 4. Go Structure

```
internal/dashboard/
├── handler.go      (~80 lines)  — GET /dashboard, permission check
├── service.go      (~150 lines) — orchestrate queries, apply data scope
└── repository.go   (~200 lines) — 5 queries tương ứng 5 views
```

**Handler:** Check user có ít nhất 1 VIEW permission → gọi service → trả JSON
**Service:** Query từng section song song (goroutine) → merge → trả response
**Repository:** 5 methods, mỗi method query 1 view + optional WHERE cho data scope

## 5. Frontend Hook

```ts
// hooks/use-dashboard.ts
export function useDashboard() {
  return useQuery({
    queryKey: ["dashboard"],
    queryFn: () => api.get("/dashboard"),
    refetchInterval: 60_000, // auto-refresh mỗi 60s
  });
}
```

## 6. Effort Estimate

| Task | Effort |
|------|--------|
| Migration 013 (5 views + indexes) | ~30 phút |
| Go backend (handler + service + repo) | ~1 giờ |
| Frontend hook + update page.tsx | ~30 phút |
| Test + verify | ~30 phút |
| **Tổng** | **~2.5 giờ** |

## 7. Lưu ý

- **KHÔNG dùng materialized view** cho MVP — data cần real-time, volume chưa lớn
- Nếu performance chậm sau này → chuyển v_dashboard_summary_today sang MATERIALIZED + refresh mỗi 5 phút
- Recent transactions view dùng UNION ALL + LIMIT → cần test performance với data lớn
- Data scope filter áp dụng ở repository level, KHÔNG hardcode trong view (vì view không biết current user)
