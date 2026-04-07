// Package creditlimit handles Credit Limit management per BRD §3.4.
package creditlimit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// Repository defines data operations for credit limits.
type Repository interface {
	// Credit limits (SCD Type 2)
	SetLimit(ctx context.Context, limit *model.CreditLimit) error
	GetCurrentLimit(ctx context.Context, counterpartyID uuid.UUID, limitType string) (*model.CreditLimit, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.CreditLimit, error)
	ListCurrentLimits(ctx context.Context, filter dto.CreditLimitListFilter, pag dto.PaginationRequest) ([]model.CreditLimit, int64, error)
	ListLimitHistory(ctx context.Context, counterpartyID uuid.UUID, limitType string) ([]model.CreditLimit, error)

	// Utilization queries
	SumBondUtilization(ctx context.Context, counterpartyID uuid.UUID, asOfDate time.Time) (decimal.Decimal, error)
	SumFXUtilization(ctx context.Context, counterpartyID uuid.UUID, asOfDate time.Time) (decimal.Decimal, error)
	GetFxMidRate(ctx context.Context, currencyCode string, asOfDate time.Time) (decimal.Decimal, error)

	// Snapshots (append-only)
	CreateSnapshot(ctx context.Context, snap *model.LimitUtilizationSnapshot) error
	GetLatestSnapshot(ctx context.Context, counterpartyID uuid.UUID, limitType string, date time.Time) (*model.LimitUtilizationSnapshot, error)

	// Approval records
	CreateApprovalRecord(ctx context.Context, rec *model.LimitApprovalRecord) error
	GetApprovalRecord(ctx context.Context, id uuid.UUID) (*model.LimitApprovalRecord, error)
	GetApprovalByDeal(ctx context.Context, dealModule string, dealID uuid.UUID) (*model.LimitApprovalRecord, error)
	ListApprovalRecords(ctx context.Context, filter dto.LimitApprovalListFilter, pag dto.PaginationRequest) ([]model.LimitApprovalRecord, int64, error)
	UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status string, riskOfficerBy, riskHeadBy *uuid.UUID, reason *string) error

	// Daily summary
	GetDailySummaryCounterparties(ctx context.Context) ([]dailySummaryBase, error)
}

// dailySummaryBase holds the base data from the view join.
type dailySummaryBase struct {
	CounterpartyID              uuid.UUID
	CounterpartyName            *string
	CIF                         string
	AllocatedCollateralized     *decimal.Decimal
	IsUnlimitedCollateralized   bool
	AllocatedUncollateralized   *decimal.Decimal
	IsUnlimitedUncollateralized bool
}

type pgxRepo struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new credit limit repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgxRepo{pool: pool}
}

// ─── Credit Limits (SCD Type 2) ───

func (r *pgxRepo) SetLimit(ctx context.Context, limit *model.CreditLimit) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Expire the current version
	_, err = tx.Exec(ctx, `
		UPDATE credit_limits
		SET is_current = false, effective_to = $1, updated_by = $2
		WHERE counterparty_id = $3 AND limit_type = $4 AND is_current = true`,
		limit.EffectiveFrom, limit.CreatedBy, limit.CounterpartyID, limit.LimitType,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to expire old limit")
	}

	// Insert new version
	limit.ID = uuid.New()
	limit.IsCurrent = true
	err = tx.QueryRow(ctx, `
		INSERT INTO credit_limits (
			id, counterparty_id, limit_type, limit_amount, is_unlimited,
			effective_from, effective_to, is_current, expiry_date,
			approval_reference, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`,
		limit.ID, limit.CounterpartyID, limit.LimitType, limit.LimitAmount,
		limit.IsUnlimited, limit.EffectiveFrom, limit.EffectiveTo, limit.IsCurrent,
		limit.ExpiryDate, limit.ApprovalReference, limit.CreatedBy, limit.UpdatedBy,
	).Scan(&limit.CreatedAt, &limit.UpdatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert credit limit")
	}

	return tx.Commit(ctx)
}

func (r *pgxRepo) GetCurrentLimit(ctx context.Context, counterpartyID uuid.UUID, limitType string) (*model.CreditLimit, error) {
	return r.scanLimit(r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, limit_type, limit_amount, is_unlimited,
			effective_from, effective_to, is_current, expiry_date,
			approval_reference, created_by, updated_by, created_at, updated_at
		FROM credit_limits
		WHERE counterparty_id = $1 AND limit_type = $2 AND is_current = true
		LIMIT 1`, counterpartyID, limitType))
}

func (r *pgxRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.CreditLimit, error) {
	return r.scanLimit(r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, limit_type, limit_amount, is_unlimited,
			effective_from, effective_to, is_current, expiry_date,
			approval_reference, created_by, updated_by, created_at, updated_at
		FROM credit_limits
		WHERE id = $1`, id))
}

func (r *pgxRepo) scanLimit(row pgx.Row) (*model.CreditLimit, error) {
	l := &model.CreditLimit{}
	err := row.Scan(
		&l.ID, &l.CounterpartyID, &l.LimitType, &l.LimitAmount, &l.IsUnlimited,
		&l.EffectiveFrom, &l.EffectiveTo, &l.IsCurrent, &l.ExpiryDate,
		&l.ApprovalReference, &l.CreatedBy, &l.UpdatedBy, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "credit limit not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query credit limit")
	}
	return l, nil
}

func (r *pgxRepo) ListCurrentLimits(ctx context.Context, filter dto.CreditLimitListFilter, pag dto.PaginationRequest) ([]model.CreditLimit, int64, error) {
	where := []string{"cl.is_current = true"}
	args := []interface{}{}
	idx := 1

	if filter.CounterpartyID != nil {
		where = append(where, fmt.Sprintf("cl.counterparty_id = $%d", idx))
		args = append(args, *filter.CounterpartyID)
		idx++
	}
	if filter.LimitType != nil {
		where = append(where, fmt.Sprintf("cl.limit_type = $%d", idx))
		args = append(args, *filter.LimitType)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count
	var total int64
	countSQL := "SELECT COUNT(*) FROM credit_limits cl WHERE " + whereClause
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count limits")
	}

	// Data
	sortBy := "cl.created_at"
	if pag.SortBy != "" {
		allowed := map[string]string{
			"created_at":     "cl.created_at",
			"limit_type":     "cl.limit_type",
			"effective_from": "cl.effective_from",
		}
		if col, ok := allowed[pag.SortBy]; ok {
			sortBy = col
		}
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataSQL := fmt.Sprintf(`
		SELECT cl.id, cl.counterparty_id, cl.limit_type, cl.limit_amount, cl.is_unlimited,
			cl.effective_from, cl.effective_to, cl.is_current, cl.expiry_date,
			cl.approval_reference, cl.created_by, cl.updated_by, cl.created_at, cl.updated_at
		FROM credit_limits cl
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortBy, sortDir, idx, idx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list limits")
	}
	defer rows.Close()

	var limits []model.CreditLimit
	for rows.Next() {
		var l model.CreditLimit
		if err := rows.Scan(
			&l.ID, &l.CounterpartyID, &l.LimitType, &l.LimitAmount, &l.IsUnlimited,
			&l.EffectiveFrom, &l.EffectiveTo, &l.IsCurrent, &l.ExpiryDate,
			&l.ApprovalReference, &l.CreatedBy, &l.UpdatedBy, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan limit")
		}
		limits = append(limits, l)
	}

	return limits, total, nil
}

func (r *pgxRepo) ListLimitHistory(ctx context.Context, counterpartyID uuid.UUID, limitType string) ([]model.CreditLimit, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, counterparty_id, limit_type, limit_amount, is_unlimited,
			effective_from, effective_to, is_current, expiry_date,
			approval_reference, created_by, updated_by, created_at, updated_at
		FROM credit_limits
		WHERE counterparty_id = $1 AND limit_type = $2
		ORDER BY effective_from DESC`, counterpartyID, limitType)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to list limit history")
	}
	defer rows.Close()

	var limits []model.CreditLimit
	for rows.Next() {
		var l model.CreditLimit
		if err := rows.Scan(
			&l.ID, &l.CounterpartyID, &l.LimitType, &l.LimitAmount, &l.IsUnlimited,
			&l.EffectiveFrom, &l.EffectiveTo, &l.IsCurrent, &l.ExpiryDate,
			&l.ApprovalReference, &l.CreatedBy, &l.UpdatedBy, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan limit")
		}
		limits = append(limits, l)
	}
	return limits, nil
}

// ─── Utilization Queries ───

// SumBondUtilization sums settlement_price of completed, not-matured bond deals for counterparty.
// Used for UNCOLLATERALIZED calculation.
func (r *pgxRepo) SumBondUtilization(ctx context.Context, counterpartyID uuid.UUID, asOfDate time.Time) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(settlement_price), 0)
		FROM bond_deals
		WHERE counterparty_id = $1
			AND status = 'COMPLETED'
			AND maturity_date > $2
			AND deleted_at IS NULL`,
		counterpartyID, asOfDate,
	).Scan(&total)
	if err != nil {
		return decimal.Zero, apperror.Wrap(err, apperror.ErrInternal, "failed to sum bond utilization")
	}
	return total, nil
}

// SumFXUtilization sums outstanding FX deal amounts for counterparty.
// FX contributes to both COLLATERALIZED and UNCOLLATERALIZED.
func (r *pgxRepo) SumFXUtilization(ctx context.Context, counterpartyID uuid.UUID, asOfDate time.Time) (decimal.Decimal, error) {
	var total decimal.Decimal
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(settlement_amount), 0)
		FROM fx_deals
		WHERE counterparty_id = $1
			AND uses_credit_limit = true
			AND status NOT IN ('CANCELLED', 'VOIDED', 'REJECTED', 'OPEN')
			AND deleted_at IS NULL`,
		counterpartyID,
	).Scan(&total)
	if err != nil {
		return decimal.Zero, apperror.Wrap(err, apperror.ErrInternal, "failed to sum FX utilization")
	}
	return total, nil
}

// GetFxMidRate gets (buy_transfer + sell_transfer) / 2 from previous business day.
func (r *pgxRepo) GetFxMidRate(ctx context.Context, currencyCode string, asOfDate time.Time) (decimal.Decimal, error) {
	var midRate decimal.Decimal
	err := r.pool.QueryRow(ctx, `
		SELECT (buy_transfer + sell_transfer) / 2
		FROM exchange_rates
		WHERE currency_code = $1
			AND effective_date < $2
		ORDER BY effective_date DESC
		LIMIT 1`,
		currencyCode, asOfDate,
	).Scan(&midRate)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, apperror.New(apperror.ErrNotFound, "no exchange rate found for "+currencyCode)
		}
		return decimal.Zero, apperror.Wrap(err, apperror.ErrInternal, "failed to get FX mid rate")
	}
	return midRate, nil
}

// ─── Snapshots ───

func (r *pgxRepo) CreateSnapshot(ctx context.Context, snap *model.LimitUtilizationSnapshot) error {
	snap.ID = uuid.New()
	breakdownJSON, _ := json.Marshal(snap.BreakdownDetail)

	return r.pool.QueryRow(ctx, `
		INSERT INTO limit_utilization_snapshots (
			id, counterparty_id, snapshot_date, limit_type,
			limit_granted, utilized_opening, utilized_intraday, utilized_total,
			remaining, fx_rate_applied, breakdown_detail, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at`,
		snap.ID, snap.CounterpartyID, snap.SnapshotDate, snap.LimitType,
		snap.LimitGranted, snap.UtilizedOpening, snap.UtilizedIntraday, snap.UtilizedTotal,
		snap.Remaining, snap.FxRateApplied, breakdownJSON, snap.CreatedBy,
	).Scan(&snap.CreatedAt)
}

func (r *pgxRepo) GetLatestSnapshot(ctx context.Context, counterpartyID uuid.UUID, limitType string, date time.Time) (*model.LimitUtilizationSnapshot, error) {
	snap := &model.LimitUtilizationSnapshot{}
	var breakdownJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, snapshot_date, limit_type,
			limit_granted, utilized_opening, utilized_intraday, utilized_total,
			remaining, fx_rate_applied, breakdown_detail, created_by, created_at
		FROM limit_utilization_snapshots
		WHERE counterparty_id = $1 AND limit_type = $2 AND snapshot_date = $3
		ORDER BY created_at DESC
		LIMIT 1`,
		counterpartyID, limitType, date,
	).Scan(
		&snap.ID, &snap.CounterpartyID, &snap.SnapshotDate, &snap.LimitType,
		&snap.LimitGranted, &snap.UtilizedOpening, &snap.UtilizedIntraday, &snap.UtilizedTotal,
		&snap.Remaining, &snap.FxRateApplied, &breakdownJSON, &snap.CreatedBy, &snap.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // no snapshot yet — not an error
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get snapshot")
	}
	if breakdownJSON != nil {
		_ = json.Unmarshal(breakdownJSON, &snap.BreakdownDetail)
	}
	return snap, nil
}

// ─── Approval Records ───

func (r *pgxRepo) CreateApprovalRecord(ctx context.Context, rec *model.LimitApprovalRecord) error {
	rec.ID = uuid.New()
	snapshotJSON, _ := json.Marshal(rec.LimitSnapshot)

	return r.pool.QueryRow(ctx, `
		INSERT INTO limit_approval_records (
			id, deal_module, deal_id, counterparty_id, limit_type,
			deal_amount_vnd, limit_snapshot, approval_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`,
		rec.ID, rec.DealModule, rec.DealID, rec.CounterpartyID, rec.LimitType,
		rec.DealAmountVND, snapshotJSON, rec.ApprovalStatus,
	).Scan(&rec.CreatedAt)
}

func (r *pgxRepo) GetApprovalRecord(ctx context.Context, id uuid.UUID) (*model.LimitApprovalRecord, error) {
	return r.scanApproval(r.pool.QueryRow(ctx, `
		SELECT id, deal_module, deal_id, counterparty_id, limit_type,
			deal_amount_vnd, limit_snapshot,
			risk_officer_approved_by, risk_officer_approved_at,
			risk_head_approved_by, risk_head_approved_at,
			approval_status, rejection_reason, created_at
		FROM limit_approval_records
		WHERE id = $1`, id))
}

func (r *pgxRepo) GetApprovalByDeal(ctx context.Context, dealModule string, dealID uuid.UUID) (*model.LimitApprovalRecord, error) {
	return r.scanApproval(r.pool.QueryRow(ctx, `
		SELECT id, deal_module, deal_id, counterparty_id, limit_type,
			deal_amount_vnd, limit_snapshot,
			risk_officer_approved_by, risk_officer_approved_at,
			risk_head_approved_by, risk_head_approved_at,
			approval_status, rejection_reason, created_at
		FROM limit_approval_records
		WHERE deal_module = $1 AND deal_id = $2
		ORDER BY created_at DESC
		LIMIT 1`, dealModule, dealID))
}

func (r *pgxRepo) scanApproval(row pgx.Row) (*model.LimitApprovalRecord, error) {
	rec := &model.LimitApprovalRecord{}
	var snapshotJSON []byte
	err := row.Scan(
		&rec.ID, &rec.DealModule, &rec.DealID, &rec.CounterpartyID, &rec.LimitType,
		&rec.DealAmountVND, &snapshotJSON,
		&rec.RiskOfficerApprovedBy, &rec.RiskOfficerApprovedAt,
		&rec.RiskHeadApprovedBy, &rec.RiskHeadApprovedAt,
		&rec.ApprovalStatus, &rec.RejectionReason, &rec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "approval record not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query approval record")
	}
	if snapshotJSON != nil {
		_ = json.Unmarshal(snapshotJSON, &rec.LimitSnapshot)
	}
	return rec, nil
}

func (r *pgxRepo) ListApprovalRecords(ctx context.Context, filter dto.LimitApprovalListFilter, pag dto.PaginationRequest) ([]model.LimitApprovalRecord, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if filter.CounterpartyID != nil {
		where = append(where, fmt.Sprintf("counterparty_id = $%d", idx))
		args = append(args, *filter.CounterpartyID)
		idx++
	}
	if filter.DealModule != nil {
		where = append(where, fmt.Sprintf("deal_module = $%d", idx))
		args = append(args, *filter.DealModule)
		idx++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("approval_status = $%d", idx))
		args = append(args, *filter.Status)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int64
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM limit_approval_records WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count approvals")
	}

	dataSQL := fmt.Sprintf(`
		SELECT id, deal_module, deal_id, counterparty_id, limit_type,
			deal_amount_vnd, limit_snapshot,
			risk_officer_approved_by, risk_officer_approved_at,
			risk_head_approved_by, risk_head_approved_at,
			approval_status, rejection_reason, created_at
		FROM limit_approval_records
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, idx, idx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list approvals")
	}
	defer rows.Close()

	var records []model.LimitApprovalRecord
	for rows.Next() {
		var rec model.LimitApprovalRecord
		var snapshotJSON []byte
		if err := rows.Scan(
			&rec.ID, &rec.DealModule, &rec.DealID, &rec.CounterpartyID, &rec.LimitType,
			&rec.DealAmountVND, &snapshotJSON,
			&rec.RiskOfficerApprovedBy, &rec.RiskOfficerApprovedAt,
			&rec.RiskHeadApprovedBy, &rec.RiskHeadApprovedAt,
			&rec.ApprovalStatus, &rec.RejectionReason, &rec.CreatedAt,
		); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan approval")
		}
		if snapshotJSON != nil {
			_ = json.Unmarshal(snapshotJSON, &rec.LimitSnapshot)
		}
		records = append(records, rec)
	}

	return records, total, nil
}

func (r *pgxRepo) UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status string, riskOfficerBy, riskHeadBy *uuid.UUID, reason *string) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx, `
		UPDATE limit_approval_records SET
			approval_status = $1,
			risk_officer_approved_by = COALESCE($2, risk_officer_approved_by),
			risk_officer_approved_at = CASE WHEN $2 IS NOT NULL THEN $5 ELSE risk_officer_approved_at END,
			risk_head_approved_by = COALESCE($3, risk_head_approved_by),
			risk_head_approved_at = CASE WHEN $3 IS NOT NULL THEN $5 ELSE risk_head_approved_at END,
			rejection_reason = COALESCE($4, rejection_reason)
		WHERE id = $6`,
		status, riskOfficerBy, riskHeadBy, reason, now, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update approval status")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "approval record not found")
	}
	return nil
}

// ─── Daily Summary ───

func (r *pgxRepo) GetDailySummaryCounterparties(ctx context.Context) ([]dailySummaryBase, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			c.id,
			c.short_name,
			c.cif,
			coll.limit_amount,
			COALESCE(coll.is_unlimited, false),
			uncoll.limit_amount,
			COALESCE(uncoll.is_unlimited, false)
		FROM counterparties c
		LEFT JOIN credit_limits coll
			ON coll.counterparty_id = c.id
			AND coll.limit_type = 'COLLATERALIZED'
			AND coll.is_current = true
		LEFT JOIN credit_limits uncoll
			ON uncoll.counterparty_id = c.id
			AND uncoll.limit_type = 'UNCOLLATERALIZED'
			AND uncoll.is_current = true
		WHERE c.deleted_at IS NULL
			AND c.is_active = true
			AND (coll.id IS NOT NULL OR uncoll.id IS NOT NULL)
		ORDER BY c.short_name, c.cif`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query daily summary")
	}
	defer rows.Close()

	var results []dailySummaryBase
	for rows.Next() {
		var row dailySummaryBase
		if err := rows.Scan(
			&row.CounterpartyID,
			&row.CounterpartyName,
			&row.CIF,
			&row.AllocatedCollateralized,
			&row.IsUnlimitedCollateralized,
			&row.AllocatedUncollateralized,
			&row.IsUnlimitedUncollateralized,
		); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan daily summary row")
		}
		results = append(results, row)
	}
	return results, nil
}
