package fx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

type pgxRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new FX deal repository backed by pgx.
func NewRepository(pool *pgxpool.Pool) repository.FxDealRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) Create(ctx context.Context, deal *model.FxDeal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Generate deal number
	dealNumber, err := r.nextDealNumber(ctx, tx, deal.TradeDate)
	if err != nil {
		return err
	}

	// Auto-generate ticket number if not provided
	if deal.TicketNumber == nil || *deal.TicketNumber == "" {
		ticketNum, err := r.nextTicketNumber(ctx, tx, deal.DealType, deal.TradeDate)
		if err != nil {
			return err
		}
		deal.TicketNumber = &ticketNum
	}

	deal.ID = uuid.New()
	deal.Version = 1

	branchID, _ := uuid.Parse("a0000000-0000-0000-0000-000000000001")

	if err := tx.QueryRow(ctx, `
		INSERT INTO fx_deals (
			id, deal_number, ticket_number, counterparty_id, deal_type, direction,
			notional_amount, currency_code, pair_code, trade_date, branch_id,
			status, note, cloned_from_id, created_by, updated_by, version,
			execution_date, pay_code_klb, pay_code_counterparty, is_international,
			attachment_path, attachment_name, settlement_amount, settlement_currency
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24, $25)
		RETURNING created_at, updated_at`,
		deal.ID, dealNumber, deal.TicketNumber, deal.CounterpartyID,
		deal.DealType, deal.Direction, deal.NotionalAmount, deal.CurrencyCode,
		deal.PairCode, deal.TradeDate, branchID,
		deal.Status, deal.Note, nil, deal.CreatedBy, deal.CreatedBy, deal.Version,
		deal.ExecutionDate, deal.PayCodeKLB, deal.PayCodeCounterparty, deal.IsInternational,
		deal.AttachmentPath, deal.AttachmentName, deal.SettlementAmount, deal.SettlementCurrency,
	).Scan(&deal.CreatedAt, &deal.UpdatedAt); err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert fx_deal")
	}

	// Insert legs
	for i := range deal.Legs {
		leg := &deal.Legs[i]
		leg.ID = uuid.New()
		leg.FxDealID = deal.ID

		_, err := tx.Exec(ctx, `
			INSERT INTO fx_deal_legs (
				id, deal_id, leg_number, value_date, settlement_date,
				exchange_rate, converted_amount, converted_currency,
				internal_ssi_id, counterparty_ssi_id,
				requires_international_settlement,
				pay_code_klb, pay_code_counterparty, is_international,
				execution_date, settlement_amount, settlement_currency
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
				'20000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000003',
				$9, $10, $11, $12, $13, $14, $15)`,
			leg.ID, leg.FxDealID, leg.LegNumber, leg.ValueDate, leg.ValueDate,
			leg.ExchangeRate, leg.BuyAmount, leg.BuyCurrency,
			leg.IsInternational,
			leg.PayCodeKLB, leg.PayCodeCounterparty, leg.IsInternational,
			leg.ExecutionDate, leg.SettlementAmount, leg.SettlementCurrency,
		)
		if err != nil {
			return apperror.Wrap(err, apperror.ErrInternal, "failed to insert fx_deal_leg")
		}
	}

	return tx.Commit(ctx)
}

func (r *pgxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.FxDeal, error) {
	deal := &model.FxDeal{}
	var ticketNumber, note *string
	var counterpartyCode, counterpartyName *string

	err := r.pool.QueryRow(ctx, `
		SELECT d.id, d.ticket_number, d.counterparty_id, d.deal_type, d.direction,
			d.notional_amount, d.currency_code, d.pair_code, d.trade_date, d.branch_id,
			d.status, d.note,
			d.created_by, d.created_at, d.updated_at, d.version,
			d.counterparty_code, d.counterparty_name,
			d.execution_date, d.pay_code_klb, d.pay_code_counterparty, d.is_international,
			d.attachment_path, d.attachment_name, d.settlement_amount, d.settlement_currency
		FROM v_fx_deal_detail d
		WHERE d.id = $1`, id,
	).Scan(
		&deal.ID, &ticketNumber, &deal.CounterpartyID, &deal.DealType, &deal.Direction,
		&deal.NotionalAmount, &deal.CurrencyCode, &deal.PairCode, &deal.TradeDate, &deal.BranchID,
		&deal.Status, &note,
		&deal.CreatedBy, &deal.CreatedAt, &deal.UpdatedAt, &deal.Version,
		&counterpartyCode, &counterpartyName,
		&deal.ExecutionDate, &deal.PayCodeKLB, &deal.PayCodeCounterparty, &deal.IsInternational,
		&deal.AttachmentPath, &deal.AttachmentName, &deal.SettlementAmount, &deal.SettlementCurrency,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "fx deal not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query fx deal")
	}

	deal.TicketNumber = ticketNumber
	deal.Note = note
	if counterpartyName != nil {
		deal.CounterpartyName = *counterpartyName
	}
	if counterpartyCode != nil {
		deal.CounterpartyCode = *counterpartyCode
	}

	// Load legs
	rows, err := r.pool.Query(ctx, `
		SELECT id, deal_id, leg_number, value_date, exchange_rate,
			converted_currency, converted_amount, converted_currency,
			pay_code_klb, pay_code_counterparty, is_international,
			execution_date, settlement_amount, settlement_currency
		FROM fx_deal_legs
		WHERE deal_id = $1
		ORDER BY leg_number`, id)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query fx deal legs")
	}
	defer rows.Close()

	for rows.Next() {
		var leg model.FxDealLeg
		var convertedCurrency string
		err := rows.Scan(
			&leg.ID, &leg.FxDealID, &leg.LegNumber, &leg.ValueDate,
			&leg.ExchangeRate, &leg.BuyCurrency, &leg.BuyAmount, &convertedCurrency,
			&leg.PayCodeKLB, &leg.PayCodeCounterparty, &leg.IsInternational,
			&leg.ExecutionDate, &leg.SettlementAmount, &leg.SettlementCurrency,
		)
		if err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan fx deal leg")
		}
		leg.SellCurrency = deal.CurrencyCode
		leg.SellAmount = deal.NotionalAmount
		deal.Legs = append(deal.Legs, leg)
	}

	return deal, nil
}

func (r *pgxRepository) buildFilterConditions(filter repository.FxDealFilter) ([]string, []interface{}, int) {
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
	if filter.DealType != nil {
		conditions = append(conditions, fmt.Sprintf("d.deal_type = $%d", argIdx))
		args = append(args, *filter.DealType)
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
	if filter.TicketNumber != nil {
		conditions = append(conditions, fmt.Sprintf("d.ticket_number ILIKE $%d", argIdx))
		args = append(args, "%"+*filter.TicketNumber+"%")
		argIdx++
	}

	return conditions, args, argIdx
}

func (r *pgxRepository) List(ctx context.Context, filter repository.FxDealFilter, pag dto.PaginationRequest) ([]model.FxDeal, int64, error) {
	conditions, args, argIdx := r.buildFilterConditions(filter)

	whereClause := "TRUE"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count total (use fx_deals base table — view already filters deleted_at)
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM v_fx_deals_list d WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count fx deals")
	}

	// Check if cursor mode
	if pag.IsCursorMode() && pag.Cursor != "" {
		return r.listCursor(ctx, whereClause, args, argIdx, pag, total)
	}

	// Offset-based pagination (backward compat)
	return r.listOffset(ctx, whereClause, args, argIdx, pag, total)
}

func (r *pgxRepository) listOffset(ctx context.Context, whereClause string, args []interface{}, argIdx int, pag dto.PaginationRequest, total int64) ([]model.FxDeal, int64, error) {
	// Allowed sort columns
	sortCol := "d.created_at"
	allowedSorts := map[string]string{
		"created_at": "d.created_at", "trade_date": "d.trade_date",
		"status": "d.status", "notional_amount": "d.notional_amount",
	}
	if col, ok := allowedSorts[pag.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.ticket_number, d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.deal_type, d.direction,
			d.notional_amount, d.currency_code, d.trade_date, d.status, d.note,
			d.created_by, d.created_at, d.updated_at, d.version
		FROM v_fx_deals_list d
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pag.PageSize, pag.Offset())

	return r.scanDealRows(ctx, dataQuery, args, total)
}

func (r *pgxRepository) listCursor(ctx context.Context, whereClause string, args []interface{}, argIdx int, pag dto.PaginationRequest, total int64) ([]model.FxDeal, int64, error) {
	cursor, err := dto.DecodeCursor(pag.Cursor)
	if err != nil {
		return nil, 0, apperror.New(apperror.ErrValidation, "invalid cursor")
	}

	cursorTS, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
	if err != nil {
		return nil, 0, apperror.New(apperror.ErrValidation, "invalid cursor timestamp")
	}

	// Keyset: WHERE (created_at, id) < ($cursor_ts, $cursor_id) ORDER BY created_at DESC, id DESC
	whereClause = fmt.Sprintf("%s AND (d.created_at, d.id) < ($%d, $%d)", whereClause, argIdx, argIdx+1)
	args = append(args, cursorTS, cursor.ID)
	argIdx += 2

	limit := pag.EffectiveLimit()
	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.ticket_number, d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.deal_type, d.direction,
			d.notional_amount, d.currency_code, d.trade_date, d.status, d.note,
			d.created_by, d.created_at, d.updated_at, d.version
		FROM v_fx_deals_list d
		WHERE %s
		ORDER BY d.created_at DESC, d.id DESC
		LIMIT $%d`,
		whereClause, argIdx)

	args = append(args, limit+1) // fetch one extra to determine hasMore

	return r.scanDealRows(ctx, dataQuery, args, total)
}

func (r *pgxRepository) scanDealRows(ctx context.Context, query string, args []interface{}, total int64) ([]model.FxDeal, int64, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query fx deals")
	}
	defer rows.Close()

	var deals []model.FxDeal
	for rows.Next() {
		var d model.FxDeal
		var ticketNumber, note *string
		var counterpartyCode, counterpartyName *string
		err := rows.Scan(
			&d.ID, &ticketNumber, &d.CounterpartyID, &counterpartyCode, &counterpartyName,
			&d.DealType, &d.Direction,
			&d.NotionalAmount, &d.CurrencyCode, &d.TradeDate, &d.Status, &note,
			&d.CreatedBy, &d.CreatedAt, &d.UpdatedAt, &d.Version,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan fx deal")
		}
		d.TicketNumber = ticketNumber
		d.Note = note
		if counterpartyName != nil {
			d.CounterpartyName = *counterpartyName
		}
		if counterpartyCode != nil {
			d.CounterpartyCode = *counterpartyCode
		}
		deals = append(deals, d)
	}

	return deals, total, nil
}

func (r *pgxRepository) Update(ctx context.Context, deal *model.FxDeal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Optimistic locking: only update when OPEN/PENDING_TP_REVIEW and version matches
	tag, err := tx.Exec(ctx, `
		UPDATE fx_deals SET
			ticket_number = $1, counterparty_id = $2, deal_type = $3, direction = $4,
			notional_amount = $5, currency_code = $6, trade_date = $7, note = $8,
			updated_by = $9, version = version + 1,
			execution_date = $12, pay_code_klb = $13, pay_code_counterparty = $14,
			is_international = $15, attachment_path = $16, attachment_name = $17,
			settlement_amount = $18, settlement_currency = $19
		WHERE id = $10 AND status IN ('OPEN', 'PENDING_TP_REVIEW') AND version = $11 AND deleted_at IS NULL`,
		deal.TicketNumber, deal.CounterpartyID, deal.DealType, deal.Direction,
		deal.NotionalAmount, deal.CurrencyCode, deal.TradeDate, deal.Note,
		deal.CreatedBy, deal.ID, deal.Version,
		deal.ExecutionDate, deal.PayCodeKLB, deal.PayCodeCounterparty,
		deal.IsInternational, deal.AttachmentPath, deal.AttachmentName,
		deal.SettlementAmount, deal.SettlementCurrency,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update fx deal")
	}
	if tag.RowsAffected() == 0 {
		// Check if it exists but is locked
		var status string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM fx_deals WHERE id = $1 AND deleted_at IS NULL", deal.ID).Scan(&status)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "fx deal not found")
		}
		if status != "OPEN" && status != "PENDING_TP_REVIEW" {
			return apperror.New(apperror.ErrDealLocked, "deal cannot be edited in current status")
		}
		return apperror.New(apperror.ErrConflict, "deal was modified by another user")
	}

	// Delete old legs and re-insert
	_, err = tx.Exec(ctx, "DELETE FROM fx_deal_legs WHERE deal_id = $1", deal.ID)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to delete old legs")
	}

	for i := range deal.Legs {
		leg := &deal.Legs[i]
		leg.ID = uuid.New()
		leg.FxDealID = deal.ID
		_, err := tx.Exec(ctx, `
			INSERT INTO fx_deal_legs (
				id, deal_id, leg_number, value_date, settlement_date,
				exchange_rate, converted_amount, converted_currency,
				internal_ssi_id, counterparty_ssi_id,
				pay_code_klb, pay_code_counterparty, is_international,
				execution_date, settlement_amount, settlement_currency
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
				'20000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000003',
				$9, $10, $11, $12, $13, $14)`,
			leg.ID, leg.FxDealID, leg.LegNumber, leg.ValueDate, leg.ValueDate,
			leg.ExchangeRate, leg.BuyAmount, leg.BuyCurrency,
			leg.PayCodeKLB, leg.PayCodeCounterparty, leg.IsInternational,
			leg.ExecutionDate, leg.SettlementAmount, leg.SettlementCurrency,
		)
		if err != nil {
			return apperror.Wrap(err, apperror.ErrInternal, "failed to insert fx deal leg")
		}
	}

	deal.Version++
	return tx.Commit(ctx)
}

func (r *pgxRepository) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE fx_deals SET status = $1, updated_by = $2
		WHERE id = $3 AND status = $4 AND deleted_at IS NULL`,
		newStatus, updatedBy, id, oldStatus,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update deal status")
	}
	if tag.RowsAffected() == 0 {
		var currentStatus string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM fx_deals WHERE id = $1 AND deleted_at IS NULL", id).Scan(&currentStatus)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "fx deal not found")
		}
		return apperror.NewWithDetail(apperror.ErrInvalidTransition,
			"status transition failed",
			fmt.Sprintf("expected status %s but found %s", oldStatus, currentStatus))
	}
	return nil
}

func (r *pgxRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE fx_deals SET deleted_at = NOW(), updated_by = $1
		WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'`,
		deletedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to soft delete fx deal")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "fx deal not found or cannot be deleted")
	}
	return nil
}

// nextDealNumber generates a gapless deal number like FX-20260403-0001.
func (r *pgxRepository) nextDealNumber(ctx context.Context, tx pgx.Tx, tradeDate time.Time) (string, error) {
	dateStr := tradeDate.Format("20060102")
	datePart := tradeDate.Truncate(24 * time.Hour)

	var seq int64
	err := tx.QueryRow(ctx, `
		INSERT INTO deal_sequences (module, prefix, date_partition, last_sequence)
		VALUES ('FX', 'FX', $1, 1)
		ON CONFLICT (module, prefix, date_partition)
		DO UPDATE SET last_sequence = deal_sequences.last_sequence + 1
		RETURNING last_sequence`, datePart,
	).Scan(&seq)

	if err != nil {
		return "", apperror.Wrap(err, apperror.ErrInternal, "failed to generate deal number")
	}

	return fmt.Sprintf("FX-%s-%04d", dateStr, seq), nil
}

// nextTicketNumber generates a ticket number like FX-20260404-001.
// Uses deal_sequences table with a separate prefix 'FX-TKT' for ticket numbering.
func (r *pgxRepository) nextTicketNumber(ctx context.Context, tx pgx.Tx, dealType string, tradeDate time.Time) (string, error) {
	dateStr := tradeDate.Format("20060102")
	datePart := tradeDate.Truncate(24 * time.Hour)

	var seq int64
	err := tx.QueryRow(ctx, `
		INSERT INTO deal_sequences (module, prefix, date_partition, last_sequence)
		VALUES ('FX', 'FX-TKT', $1, 1)
		ON CONFLICT (module, prefix, date_partition)
		DO UPDATE SET last_sequence = deal_sequences.last_sequence + 1
		RETURNING last_sequence`, datePart,
	).Scan(&seq)
	if err != nil {
		return "", apperror.Wrap(err, apperror.ErrInternal, "failed to generate ticket number")
	}

	return fmt.Sprintf("FX-%s-%03d", dateStr, seq), nil
}

// SumOutstandingByCounterparty returns the total notional amount of active (non-terminal) deals
// for a counterparty. Terminal statuses (COMPLETED, CANCELLED, REJECTED, VOIDED_*) are excluded.
// If excludeDealID is provided, that deal is excluded from the sum (for update scenarios).
func (r *pgxRepository) SumOutstandingByCounterparty(ctx context.Context, counterpartyID uuid.UUID, excludeDealID *uuid.UUID) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(notional_amount), 0)
		FROM fx_deals
		WHERE counterparty_id = $1
		  AND deleted_at IS NULL
		  AND status NOT IN ('COMPLETED', 'CANCELLED', 'REJECTED',
		      'VOIDED_BY_ACCOUNTING', 'VOIDED_BY_SETTLEMENT', 'VOIDED_BY_RISK')`

	var args []interface{}
	args = append(args, counterpartyID)

	if excludeDealID != nil {
		query += " AND id != $2"
		args = append(args, *excludeDealID)
	}

	var total decimal.Decimal
	err := r.pool.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		return decimal.Zero, apperror.Wrap(err, apperror.ErrInternal, "failed to sum outstanding deals")
	}
	return total, nil
}

// Ensure interface compliance
var _ repository.FxDealRepository = (*pgxRepository)(nil)

// insertDealDirect is a test helper that creates deals directly via pool.
func insertDealDirect(ctx context.Context, pool *pgxpool.Pool, deal *model.FxDeal) error {
	deal.ID = uuid.New()
	deal.Version = 1
	dealNumber := fmt.Sprintf("FX-TEST-%s", deal.ID.String()[:8])

	_, err := pool.Exec(ctx, `
		INSERT INTO fx_deals (
			id, deal_number, ticket_number, counterparty_id, deal_type, direction,
			notional_amount, currency_code, pair_code, trade_date, branch_id,
			status, note, created_by, updated_by, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14, $15)`,
		deal.ID, dealNumber, deal.TicketNumber, deal.CounterpartyID,
		deal.DealType, deal.Direction, deal.NotionalAmount, deal.CurrencyCode,
		"USD/VND", deal.TradeDate,
		"a0000000-0000-0000-0000-000000000001", // branch_id
		deal.Status, deal.Note, deal.CreatedBy, deal.Version,
	)
	if err != nil {
		return fmt.Errorf("insertDealDirect: %w", err)
	}

	for i := range deal.Legs {
		leg := &deal.Legs[i]
		leg.ID = uuid.New()
		leg.FxDealID = deal.ID
		_, err := pool.Exec(ctx, `
			INSERT INTO fx_deal_legs (
				id, deal_id, leg_number, value_date, settlement_date,
				exchange_rate, converted_amount, converted_currency,
				internal_ssi_id, counterparty_ssi_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '20000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000003')`,
			leg.ID, leg.FxDealID, leg.LegNumber, leg.ValueDate, leg.ValueDate,
			leg.ExchangeRate, leg.BuyAmount, leg.BuyCurrency,
		)
		if err != nil {
			return fmt.Errorf("insertDealDirect leg: %w", err)
		}
	}

	return nil
}
