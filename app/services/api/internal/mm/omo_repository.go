package mm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

type omoRepoRepository struct {
	pool *pgxpool.Pool
}

// NewOMORepoRepository creates a new MM OMO/Repo deal repository backed by pgx.
func NewOMORepoRepository(pool *pgxpool.Pool) repository.MMOMORepoRepository {
	return &omoRepoRepository{pool: pool}
}

func (r *omoRepoRepository) Create(ctx context.Context, deal *model.MMOMORepoDeal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Generate deal number: OMO-YYYYMMDD-NNNN for OMO, RK-YYYYMMDD-NNNN for STATE_REPO
	dealNumber, err := r.nextDealNumber(ctx, tx, deal.DealSubtype, deal.TradeDate)
	if err != nil {
		return err
	}

	deal.ID = uuid.New()
	deal.DealNumber = dealNumber

	branchID, _ := uuid.Parse("a0000000-0000-0000-0000-000000000001")

	if err := tx.QueryRow(ctx, `
		INSERT INTO mm_omo_repo_deals (
			id, deal_number, deal_subtype, session_name, trade_date, branch_id,
			counterparty_id, notional_amount, bond_catalog_id,
			winning_rate, tenor_days, settlement_date_1, settlement_date_2,
			haircut_pct, status, note, cloned_from_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $18
		)
		RETURNING created_at, updated_at`,
		deal.ID, dealNumber, deal.DealSubtype, deal.SessionName, deal.TradeDate, branchID,
		deal.CounterpartyID, deal.NotionalAmount, deal.BondCatalogID,
		deal.WinningRate, deal.TenorDays, deal.SettlementDate1, deal.SettlementDate2,
		deal.HaircutPct, deal.Status, deal.Note, deal.ClonedFromID,
		deal.CreatedBy,
	).Scan(&deal.CreatedAt, &deal.UpdatedAt); err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert mm_omo_repo_deal")
	}

	deal.Version = 1
	return tx.Commit(ctx)
}

func (r *omoRepoRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MMOMORepoDeal, error) {
	deal := &model.MMOMORepoDeal{}
	var counterpartyCode, counterpartyName *string
	var createdByName *string
	var branchCode, branchName *string

	err := r.pool.QueryRow(ctx, `
		SELECT d.id, d.deal_number, d.deal_subtype, d.session_name, d.trade_date,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.notional_amount, d.bond_catalog_id,
			d.bond_code, d.bond_issuer, d.bond_coupon_rate, d.bond_maturity_date,
			d.winning_rate, d.tenor_days, d.settlement_date_1, d.settlement_date_2,
			d.haircut_pct, d.status, d.note, d.cloned_from_id,
			d.cancel_reason, d.cancel_requested_at,
			d.created_by, d.created_by_name, d.created_at, d.branch_code, d.branch_name
		FROM v_mm_omo_repo_deals_list d
		WHERE d.id = $1`, id,
	).Scan(
		&deal.ID, &deal.DealNumber, &deal.DealSubtype, &deal.SessionName, &deal.TradeDate,
		&deal.CounterpartyID, &counterpartyCode, &counterpartyName,
		&deal.NotionalAmount, &deal.BondCatalogID,
		&deal.BondCode, &deal.BondIssuer, &deal.BondCouponRate, &deal.BondMaturityDate,
		&deal.WinningRate, &deal.TenorDays, &deal.SettlementDate1, &deal.SettlementDate2,
		&deal.HaircutPct, &deal.Status, &deal.Note, &deal.ClonedFromID,
		&deal.CancelReason, &deal.CancelRequestedAt,
		&deal.CreatedBy, &createdByName, &deal.CreatedAt, &branchCode, &branchName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "OMO/Repo deal not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query OMO/Repo deal")
	}

	if counterpartyName != nil {
		deal.CounterpartyName = *counterpartyName
	}
	if counterpartyCode != nil {
		deal.CounterpartyCode = *counterpartyCode
	}
	if createdByName != nil {
		deal.CreatedByName = *createdByName
	}
	if branchCode != nil {
		deal.BranchCode = *branchCode
	}
	if branchName != nil {
		deal.BranchName = *branchName
	}
	deal.Version = 1

	// Read actual version from base table
	_ = r.pool.QueryRow(ctx, `SELECT updated_at FROM mm_omo_repo_deals WHERE id = $1`, id).Scan(&deal.UpdatedAt)

	return deal, nil
}

func (r *omoRepoRepository) buildFilterConditions(filter dto.MMOMORepoFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	// Always filter by deal_subtype
	conditions = append(conditions, fmt.Sprintf("d.deal_subtype = $%d", argIdx))
	args = append(args, filter.DealSubtype)
	argIdx++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.Statuses != nil && len(*filter.Statuses) > 0 {
		placeholders := make([]string, len(*filter.Statuses))
		for i, s := range *filter.Statuses {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("d.status IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.ExcludeStatuses != nil && len(*filter.ExcludeStatuses) > 0 {
		placeholders := make([]string, len(*filter.ExcludeStatuses))
		for i, s := range *filter.ExcludeStatuses {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("d.status NOT IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.CounterpartyID != nil {
		conditions = append(conditions, fmt.Sprintf("d.counterparty_id = $%d", argIdx))
		args = append(args, *filter.CounterpartyID)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.trade_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("d.trade_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}
	if filter.CreatedBy != nil {
		conditions = append(conditions, fmt.Sprintf("d.created_by = $%d", argIdx))
		args = append(args, *filter.CreatedBy)
		argIdx++
	}
	if filter.DealNumber != nil {
		conditions = append(conditions, fmt.Sprintf("d.deal_number ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.DealNumber+"%")
		argIdx++
	}

	return conditions, args, argIdx
}

func (r *omoRepoRepository) List(ctx context.Context, filter dto.MMOMORepoFilter, pag dto.PaginationRequest) ([]model.MMOMORepoDeal, int64, error) {
	conditions, args, argIdx := r.buildFilterConditions(filter)

	whereClause := "TRUE"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM v_mm_omo_repo_deals_list d WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count OMO/Repo deals")
	}

	// Offset-based pagination
	sortCol := "d.created_at"
	allowedSorts := map[string]string{
		"created_at":      "d.created_at",
		"trade_date":      "d.trade_date",
		"status":          "d.status",
		"notional_amount": "d.notional_amount",
	}
	if col, ok := allowedSorts[pag.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.deal_number, d.deal_subtype, d.session_name, d.trade_date,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.notional_amount, d.bond_catalog_id,
			d.bond_code, d.bond_issuer, d.bond_coupon_rate, d.bond_maturity_date,
			d.winning_rate, d.tenor_days, d.settlement_date_1, d.settlement_date_2,
			d.haircut_pct, d.status, d.note,
			d.created_by, d.created_by_name, d.created_at
		FROM v_mm_omo_repo_deals_list d
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query OMO/Repo deals")
	}
	defer rows.Close()

	var deals []model.MMOMORepoDeal
	for rows.Next() {
		var d model.MMOMORepoDeal
		var counterpartyCode, counterpartyName *string
		var createdByName *string
		err := rows.Scan(
			&d.ID, &d.DealNumber, &d.DealSubtype, &d.SessionName, &d.TradeDate,
			&d.CounterpartyID, &counterpartyCode, &counterpartyName,
			&d.NotionalAmount, &d.BondCatalogID,
			&d.BondCode, &d.BondIssuer, &d.BondCouponRate, &d.BondMaturityDate,
			&d.WinningRate, &d.TenorDays, &d.SettlementDate1, &d.SettlementDate2,
			&d.HaircutPct, &d.Status, &d.Note,
			&d.CreatedBy, &createdByName, &d.CreatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan OMO/Repo deal")
		}
		if counterpartyName != nil {
			d.CounterpartyName = *counterpartyName
		}
		if counterpartyCode != nil {
			d.CounterpartyCode = *counterpartyCode
		}
		if createdByName != nil {
			d.CreatedByName = *createdByName
		}
		d.Version = 1
		deals = append(deals, d)
	}

	return deals, total, nil
}

func (r *omoRepoRepository) Update(ctx context.Context, deal *model.MMOMORepoDeal) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_omo_repo_deals SET
			session_name = $1, trade_date = $2, counterparty_id = $3,
			notional_amount = $4, bond_catalog_id = $5,
			winning_rate = $6, tenor_days = $7,
			settlement_date_1 = $8, settlement_date_2 = $9,
			haircut_pct = $10, note = $11, updated_by = $12
		WHERE id = $13 AND status = 'OPEN' AND deleted_at IS NULL`,
		deal.SessionName, deal.TradeDate, deal.CounterpartyID,
		deal.NotionalAmount, deal.BondCatalogID,
		deal.WinningRate, deal.TenorDays,
		deal.SettlementDate1, deal.SettlementDate2,
		deal.HaircutPct, deal.Note, deal.CreatedBy,
		deal.ID,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update OMO/Repo deal")
	}
	if tag.RowsAffected() == 0 {
		var status string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM mm_omo_repo_deals WHERE id = $1 AND deleted_at IS NULL", deal.ID).Scan(&status)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "OMO/Repo deal not found")
		}
		if status != constants.StatusOpen {
			return apperror.New(apperror.ErrDealLocked, "deal cannot be edited in current status")
		}
		return apperror.New(apperror.ErrConflict, "deal was modified by another user")
	}
	return nil
}

func (r *omoRepoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_omo_repo_deals SET status = $1, updated_by = $2
		WHERE id = $3 AND status = $4 AND deleted_at IS NULL`,
		newStatus, updatedBy, id, oldStatus,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update OMO/Repo deal status")
	}
	if tag.RowsAffected() == 0 {
		var currentStatus string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM mm_omo_repo_deals WHERE id = $1 AND deleted_at IS NULL", id).Scan(&currentStatus)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "OMO/Repo deal not found")
		}
		return apperror.NewWithDetail(apperror.ErrInvalidTransition,
			"status transition failed",
			fmt.Sprintf("expected status %s but found %s", oldStatus, currentStatus))
	}
	return nil
}

func (r *omoRepoRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_omo_repo_deals SET deleted_at = NOW(), updated_by = $1
		WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'`,
		deletedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to soft delete OMO/Repo deal")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "OMO/Repo deal not found or cannot be deleted")
	}
	return nil
}

func (r *omoRepoRepository) UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE mm_omo_repo_deals SET cancel_reason = $1, cancel_requested_by = $2, cancel_requested_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL`,
		reason, requestedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update cancel fields")
	}
	return nil
}

// --- Deal Number Generation ---

// nextDealNumber generates a gapless deal number.
// OMO-YYYYMMDD-NNNN for OMO, RK-YYYYMMDD-NNNN for STATE_REPO.
func (r *omoRepoRepository) nextDealNumber(ctx context.Context, tx pgx.Tx, dealSubtype string, tradeDate time.Time) (string, error) {
	prefix := "OMO"
	if dealSubtype == constants.MMSubtypeStateRepo {
		prefix = "RK"
	}

	dateStr := tradeDate.Format("20060102")
	datePart := tradeDate.Truncate(24 * time.Hour)

	var seq int64
	err := tx.QueryRow(ctx, `
		INSERT INTO deal_sequences (module, prefix, date_partition, last_sequence)
		VALUES ('MM_OMO_REPO', $1, $2, 1)
		ON CONFLICT (module, prefix, date_partition)
		DO UPDATE SET last_sequence = deal_sequences.last_sequence + 1
		RETURNING last_sequence`, prefix, datePart,
	).Scan(&seq)

	if err != nil {
		return "", apperror.Wrap(err, apperror.ErrInternal, "failed to generate OMO/Repo deal number")
	}

	return fmt.Sprintf("%s-%s-%04d", prefix, dateStr, seq), nil
}

// Ensure interface compliance
var _ repository.MMOMORepoRepository = (*omoRepoRepository)(nil)

// insertOMORepoDealDirect is a test helper that creates deals directly via pool.
func insertOMORepoDealDirect(ctx context.Context, pool *pgxpool.Pool, deal *model.MMOMORepoDeal) error {
	deal.ID = uuid.New()
	deal.Version = 1
	dealNumber := fmt.Sprintf("OMO-TEST-%s", deal.ID.String()[:8])
	deal.DealNumber = dealNumber

	_, err := pool.Exec(ctx, `
		INSERT INTO mm_omo_repo_deals (
			id, deal_number, deal_subtype, session_name, trade_date, branch_id,
			counterparty_id, notional_amount, bond_catalog_id,
			winning_rate, tenor_days, settlement_date_1, settlement_date_2,
			haircut_pct, status, note, cloned_from_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $18
		)`,
		deal.ID, dealNumber, deal.DealSubtype, deal.SessionName, deal.TradeDate,
		"a0000000-0000-0000-0000-000000000001", // branch_id
		deal.CounterpartyID, deal.NotionalAmount, deal.BondCatalogID,
		deal.WinningRate, deal.TenorDays, deal.SettlementDate1, deal.SettlementDate2,
		deal.HaircutPct, deal.Status, deal.Note, deal.ClonedFromID,
		deal.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insertOMORepoDealDirect: %w", err)
	}
	return nil
}
