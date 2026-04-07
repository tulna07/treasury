// Package dashboard provides the Dashboard aggregation layer.
package dashboard

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles dashboard view queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new dashboard repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ---------------------------------------------------------------------------
// SummaryToday — v_dashboard_summary_today (single row)
// ---------------------------------------------------------------------------

// SummaryToday holds today's aggregated KPIs.
type SummaryToday struct {
	TotalDealsToday          int     `json:"total_deals_today"          db:"total_deals_today"`
	CompletedToday           int     `json:"completed_today"            db:"completed_today"`
	PendingToday             int     `json:"pending_today"              db:"pending_today"`
	FxTotalNotional          float64 `json:"fx_total_notional"          db:"fx_total_notional"`
	FxBuyCount               int     `json:"fx_buy_count"               db:"fx_buy_count"`
	FxSellCount              int     `json:"fx_sell_count"              db:"fx_sell_count"`
	MmOutstanding            float64 `json:"mm_outstanding"             db:"mm_outstanding"`
	MmActiveCount            int     `json:"mm_active_count"            db:"mm_active_count"`
	BondPortfolioValue       float64 `json:"bond_portfolio_value"       db:"bond_portfolio_value"`
	TtqtPendingCount         int     `json:"ttqt_pending_count"         db:"ttqt_pending_count"`
	CounterpartiesWithLimits int     `json:"counterparties_with_limits" db:"counterparties_with_limits"`
}

// GetSummaryToday returns today's dashboard summary.
func (r *Repository) GetSummaryToday(ctx context.Context) (*SummaryToday, error) {
	rows, _ := r.pool.Query(ctx, "SELECT * FROM v_dashboard_summary_today")
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[SummaryToday])
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ---------------------------------------------------------------------------
// DailyVolume — v_dashboard_daily_volume (7 rows)
// ---------------------------------------------------------------------------

// DailyVolume holds per-day trading volume by module.
type DailyVolume struct {
	TradeDate  time.Time  `json:"trade_date"  db:"trade_date"`
	FxCount    int     `json:"fx_count"    db:"fx_count"`
	FxVolume   float64 `json:"fx_volume"   db:"fx_volume"`
	MmCount    int     `json:"mm_count"    db:"mm_count"`
	MmVolume   float64 `json:"mm_volume"   db:"mm_volume"`
	BondCount  int     `json:"bond_count"  db:"bond_count"`
	BondVolume float64 `json:"bond_volume" db:"bond_volume"`
}

// GetDailyVolume returns the last 7 days of trading volume.
func (r *Repository) GetDailyVolume(ctx context.Context) ([]DailyVolume, error) {
	rows, _ := r.pool.Query(ctx, "SELECT * FROM v_dashboard_daily_volume")
	return pgx.CollectRows(rows, pgx.RowToStructByName[DailyVolume])
}

// ---------------------------------------------------------------------------
// ModuleDistribution — v_dashboard_module_distribution (single row)
// ---------------------------------------------------------------------------

// ModuleDistribution holds deal counts per module.
type ModuleDistribution struct {
	FxCount   int `json:"fx_count"   db:"fx_count"`
	BondCount int `json:"bond_count" db:"bond_count"`
	MmCount   int `json:"mm_count"   db:"mm_count"`
	TtqtCount int `json:"ttqt_count" db:"ttqt_count"`
}

// GetModuleDistribution returns the current module distribution.
func (r *Repository) GetModuleDistribution(ctx context.Context) (*ModuleDistribution, error) {
	rows, _ := r.pool.Query(ctx, "SELECT * FROM v_dashboard_module_distribution")
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[ModuleDistribution])
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ---------------------------------------------------------------------------
// StatusDaily — v_dashboard_status_daily (5 rows)
// ---------------------------------------------------------------------------

// StatusDaily holds per-day deal status breakdown.
type StatusDaily struct {
	TradeDate      time.Time `json:"trade_date"      db:"trade_date"`
	OpenCount      int    `json:"open_count"       db:"open_count"`
	PendingCount   int    `json:"pending_count"    db:"pending_count"`
	CompletedCount int    `json:"completed_count"  db:"completed_count"`
	CancelledCount int    `json:"cancelled_count"  db:"cancelled_count"`
}

// GetStatusDaily returns status breakdown for the last 5 days.
func (r *Repository) GetStatusDaily(ctx context.Context) ([]StatusDaily, error) {
	rows, _ := r.pool.Query(ctx, "SELECT * FROM v_dashboard_status_daily")
	return pgx.CollectRows(rows, pgx.RowToStructByName[StatusDaily])
}

// ---------------------------------------------------------------------------
// RecentTransactions — v_dashboard_recent_transactions (10 rows)
// ---------------------------------------------------------------------------

// RecentTransaction represents a single recent deal.
type RecentTransaction struct {
	ID        string  `json:"id"         db:"id"`
	Ticket    string  `json:"ticket"     db:"ticket"`
	Module    string  `json:"module"     db:"module"`
	DealType  string  `json:"deal_type"  db:"deal_type"`
	Amount    float64 `json:"amount"     db:"amount"`
	Currency  *string `json:"currency"   db:"currency"`
	Status    string  `json:"status"     db:"status"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	CreatedBy *string        `json:"created_by,omitempty" db:"created_by"`
}

// GetRecentTransactions returns the 10 most recent transactions.
func (r *Repository) GetRecentTransactions(ctx context.Context) ([]RecentTransaction, error) {
	rows, _ := r.pool.Query(ctx, "SELECT * FROM v_dashboard_recent_transactions")
	return pgx.CollectRows(rows, pgx.RowToStructByName[RecentTransaction])
}
