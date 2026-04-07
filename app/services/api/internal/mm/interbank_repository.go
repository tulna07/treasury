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

type interbankRepository struct {
	pool *pgxpool.Pool
}

// NewInterbankRepository creates a new MM interbank deal repository backed by pgx.
func NewInterbankRepository(pool *pgxpool.Pool) repository.MMInterbankRepository {
	return &interbankRepository{pool: pool}
}

func (r *interbankRepository) Create(ctx context.Context, deal *model.MMInterbankDeal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Generate deal number: MM-YYYYMMDD-NNNN
	dealNumber, err := r.nextDealNumber(ctx, tx, deal.TradeDate)
	if err != nil {
		return err
	}

	deal.ID = uuid.New()
	deal.DealNumber = dealNumber

	branchID, _ := uuid.Parse("a0000000-0000-0000-0000-000000000001")

	if err := tx.QueryRow(ctx, `
		INSERT INTO mm_interbank_deals (
			id, deal_number, ticket_number, counterparty_id, branch_id,
			currency_code, internal_ssi_id, counterparty_ssi_id, counterparty_ssi_text,
			direction, principal_amount, interest_rate, day_count_convention,
			trade_date, effective_date, tenor_days, maturity_date,
			interest_amount, maturity_amount,
			has_collateral, collateral_currency, collateral_description,
			requires_international_settlement,
			status, note, cloned_from_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19,
			$20, $21, $22,
			$23,
			$24, $25, $26,
			$27, $27
		)
		RETURNING created_at, updated_at`,
		deal.ID, dealNumber, deal.TicketNumber, deal.CounterpartyID, branchID,
		deal.CurrencyCode, deal.InternalSSIID, deal.CounterpartySSIID, deal.CounterpartySSIText,
		deal.Direction, deal.PrincipalAmount, deal.InterestRate, deal.DayCountConvention,
		deal.TradeDate, deal.EffectiveDate, deal.TenorDays, deal.MaturityDate,
		deal.InterestAmount, deal.MaturityAmount,
		deal.HasCollateral, deal.CollateralCurrency, deal.CollateralDescription,
		deal.RequiresInternationalSettlement,
		deal.Status, deal.Note, deal.ClonedFromID,
		deal.CreatedBy,
	).Scan(&deal.CreatedAt, &deal.UpdatedAt); err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert mm_interbank_deal")
	}

	deal.Version = 1
	return tx.Commit(ctx)
}

func (r *interbankRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MMInterbankDeal, error) {
	deal := &model.MMInterbankDeal{}
	var counterpartyCode, counterpartyName *string
	var createdByName *string
	var branchCode, branchName *string

	err := r.pool.QueryRow(ctx, `
		SELECT d.id, d.deal_number, d.ticket_number,
			d.trade_date, d.effective_date, d.direction,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.currency_code, d.principal_amount, d.interest_rate,
			d.day_count_convention, d.tenor_days, d.maturity_date,
			d.interest_amount, d.maturity_amount,
			d.has_collateral, d.requires_international_settlement,
			d.status, d.note, d.cloned_from_id,
			d.cancel_reason, d.cancel_requested_at,
			d.created_by, d.created_by_name, d.created_at,
			d.branch_code, d.branch_name
		FROM v_mm_interbank_deals_list d
		WHERE d.id = $1`, id,
	).Scan(
		&deal.ID, &deal.DealNumber, &deal.TicketNumber,
		&deal.TradeDate, &deal.EffectiveDate, &deal.Direction,
		&deal.CounterpartyID, &counterpartyCode, &counterpartyName,
		&deal.CurrencyCode, &deal.PrincipalAmount, &deal.InterestRate,
		&deal.DayCountConvention, &deal.TenorDays, &deal.MaturityDate,
		&deal.InterestAmount, &deal.MaturityAmount,
		&deal.HasCollateral, &deal.RequiresInternationalSettlement,
		&deal.Status, &deal.Note, &deal.ClonedFromID,
		&deal.CancelReason, &deal.CancelRequestedAt,
		&deal.CreatedBy, &createdByName, &deal.CreatedAt,
		&branchCode, &branchName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "interbank deal not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query interbank deal")
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
	deal.Version = 1 // Version tracked at service level via optimistic locking in Update

	// Read actual version from base table
	_ = r.pool.QueryRow(ctx, `SELECT updated_at FROM mm_interbank_deals WHERE id = $1`, id).Scan(&deal.UpdatedAt)

	return deal, nil
}

func (r *interbankRepository) buildFilterConditions(filter dto.MMInterbankFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

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
	if filter.Direction != nil {
		conditions = append(conditions, fmt.Sprintf("d.direction = $%d", argIdx))
		args = append(args, *filter.Direction)
		argIdx++
	}
	if filter.CurrencyCode != nil {
		conditions = append(conditions, fmt.Sprintf("d.currency_code = $%d", argIdx))
		args = append(args, *filter.CurrencyCode)
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

func (r *interbankRepository) List(ctx context.Context, filter dto.MMInterbankFilter, pag dto.PaginationRequest) ([]model.MMInterbankDeal, int64, error) {
	conditions, args, argIdx := r.buildFilterConditions(filter)

	whereClause := "TRUE"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM v_mm_interbank_deals_list d WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count interbank deals")
	}

	// Offset-based pagination
	sortCol := "d.created_at"
	allowedSorts := map[string]string{
		"created_at":       "d.created_at",
		"trade_date":       "d.trade_date",
		"status":           "d.status",
		"principal_amount": "d.principal_amount",
		"maturity_date":    "d.maturity_date",
	}
	if col, ok := allowedSorts[pag.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.deal_number, d.ticket_number,
			d.trade_date, d.effective_date, d.direction,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.currency_code, d.principal_amount, d.interest_rate,
			d.day_count_convention, d.tenor_days, d.maturity_date,
			d.interest_amount, d.maturity_amount,
			d.has_collateral, d.requires_international_settlement,
			d.status, d.note,
			d.created_by, d.created_by_name, d.created_at,
			d.branch_code, d.branch_name
		FROM v_mm_interbank_deals_list d
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query interbank deals")
	}
	defer rows.Close()

	var deals []model.MMInterbankDeal
	for rows.Next() {
		var d model.MMInterbankDeal
		var counterpartyCode, counterpartyName *string
		var createdByName *string
		var branchCode, branchName *string
		err := rows.Scan(
			&d.ID, &d.DealNumber, &d.TicketNumber,
			&d.TradeDate, &d.EffectiveDate, &d.Direction,
			&d.CounterpartyID, &counterpartyCode, &counterpartyName,
			&d.CurrencyCode, &d.PrincipalAmount, &d.InterestRate,
			&d.DayCountConvention, &d.TenorDays, &d.MaturityDate,
			&d.InterestAmount, &d.MaturityAmount,
			&d.HasCollateral, &d.RequiresInternationalSettlement,
			&d.Status, &d.Note,
			&d.CreatedBy, &createdByName, &d.CreatedAt,
			&branchCode, &branchName,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan interbank deal")
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
		if branchCode != nil {
			d.BranchCode = *branchCode
		}
		if branchName != nil {
			d.BranchName = *branchName
		}
		d.Version = 1
		deals = append(deals, d)
	}

	return deals, total, nil
}

func (r *interbankRepository) Update(ctx context.Context, deal *model.MMInterbankDeal) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_interbank_deals SET
			ticket_number = $1, counterparty_id = $2,
			currency_code = $3, internal_ssi_id = $4,
			counterparty_ssi_id = $5, counterparty_ssi_text = $6,
			direction = $7, principal_amount = $8,
			interest_rate = $9, day_count_convention = $10,
			trade_date = $11, effective_date = $12,
			tenor_days = $13, maturity_date = $14,
			interest_amount = $15, maturity_amount = $16,
			has_collateral = $17, collateral_currency = $18,
			collateral_description = $19, requires_international_settlement = $20,
			note = $21, updated_by = $22
		WHERE id = $23 AND status = 'OPEN' AND deleted_at IS NULL`,
		deal.TicketNumber, deal.CounterpartyID,
		deal.CurrencyCode, deal.InternalSSIID,
		deal.CounterpartySSIID, deal.CounterpartySSIText,
		deal.Direction, deal.PrincipalAmount,
		deal.InterestRate, deal.DayCountConvention,
		deal.TradeDate, deal.EffectiveDate,
		deal.TenorDays, deal.MaturityDate,
		deal.InterestAmount, deal.MaturityAmount,
		deal.HasCollateral, deal.CollateralCurrency,
		deal.CollateralDescription, deal.RequiresInternationalSettlement,
		deal.Note, deal.CreatedBy,
		deal.ID,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update interbank deal")
	}
	if tag.RowsAffected() == 0 {
		var status string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM mm_interbank_deals WHERE id = $1 AND deleted_at IS NULL", deal.ID).Scan(&status)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "interbank deal not found")
		}
		if status != constants.StatusOpen {
			return apperror.New(apperror.ErrDealLocked, "deal cannot be edited in current status")
		}
		return apperror.New(apperror.ErrConflict, "deal was modified by another user")
	}
	return nil
}

func (r *interbankRepository) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_interbank_deals SET status = $1, updated_by = $2
		WHERE id = $3 AND status = $4 AND deleted_at IS NULL`,
		newStatus, updatedBy, id, oldStatus,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update interbank deal status")
	}
	if tag.RowsAffected() == 0 {
		var currentStatus string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM mm_interbank_deals WHERE id = $1 AND deleted_at IS NULL", id).Scan(&currentStatus)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "interbank deal not found")
		}
		return apperror.NewWithDetail(apperror.ErrInvalidTransition,
			"status transition failed",
			fmt.Sprintf("expected status %s but found %s", oldStatus, currentStatus))
	}
	return nil
}

func (r *interbankRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE mm_interbank_deals SET deleted_at = NOW(), updated_by = $1
		WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'`,
		deletedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to soft delete interbank deal")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "interbank deal not found or cannot be deleted")
	}
	return nil
}

func (r *interbankRepository) UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE mm_interbank_deals SET cancel_reason = $1, cancel_requested_by = $2, cancel_requested_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL`,
		reason, requestedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update cancel fields")
	}
	return nil
}

// --- Deal Number Generation ---

// nextDealNumber generates a gapless deal number: MM-YYYYMMDD-NNNN.
func (r *interbankRepository) nextDealNumber(ctx context.Context, tx pgx.Tx, tradeDate time.Time) (string, error) {
	dateStr := tradeDate.Format("20060102")
	datePart := tradeDate.Truncate(24 * time.Hour)

	var seq int64
	err := tx.QueryRow(ctx, `
		INSERT INTO deal_sequences (module, prefix, date_partition, last_sequence)
		VALUES ('MM_INTERBANK', $1, $2, 1)
		ON CONFLICT (module, prefix, date_partition)
		DO UPDATE SET last_sequence = deal_sequences.last_sequence + 1
		RETURNING last_sequence`, "MM", datePart,
	).Scan(&seq)

	if err != nil {
		return "", apperror.Wrap(err, apperror.ErrInternal, "failed to generate interbank deal number")
	}

	return fmt.Sprintf("MM-%s-%04d", dateStr, seq), nil
}

// Ensure interface compliance
var _ repository.MMInterbankRepository = (*interbankRepository)(nil)

// insertInterbankDealDirect is a test helper that creates deals directly via pool.
func insertInterbankDealDirect(ctx context.Context, pool *pgxpool.Pool, deal *model.MMInterbankDeal) error {
	deal.ID = uuid.New()
	deal.Version = 1
	dealNumber := fmt.Sprintf("MM-TEST-%s", deal.ID.String()[:8])
	deal.DealNumber = dealNumber

	_, err := pool.Exec(ctx, `
		INSERT INTO mm_interbank_deals (
			id, deal_number, ticket_number, counterparty_id, branch_id,
			currency_code, internal_ssi_id, counterparty_ssi_id, counterparty_ssi_text,
			direction, principal_amount, interest_rate, day_count_convention,
			trade_date, effective_date, tenor_days, maturity_date,
			interest_amount, maturity_amount,
			has_collateral, collateral_currency, collateral_description,
			requires_international_settlement,
			status, note, cloned_from_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19,
			$20, $21, $22,
			$23,
			$24, $25, $26,
			$27, $27
		)`,
		deal.ID, dealNumber, deal.TicketNumber, deal.CounterpartyID,
		"a0000000-0000-0000-0000-000000000001", // branch_id
		deal.CurrencyCode, deal.InternalSSIID, deal.CounterpartySSIID, deal.CounterpartySSIText,
		deal.Direction, deal.PrincipalAmount, deal.InterestRate, deal.DayCountConvention,
		deal.TradeDate, deal.EffectiveDate, deal.TenorDays, deal.MaturityDate,
		deal.InterestAmount, deal.MaturityAmount,
		deal.HasCollateral, deal.CollateralCurrency, deal.CollateralDescription,
		deal.RequiresInternationalSettlement,
		deal.Status, deal.Note, deal.ClonedFromID,
		deal.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insertInterbankDealDirect: %w", err)
	}
	return nil
}
