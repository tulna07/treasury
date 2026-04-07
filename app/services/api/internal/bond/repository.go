package bond

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

type pgxRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Bond deal repository backed by pgx.
func NewRepository(pool *pgxpool.Pool) repository.BondDealRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) Create(ctx context.Context, deal *model.BondDeal) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Generate deal number: G-YYYYMMDD-NNNN (Govi) or F-YYYYMMDD-NNNN (FI/CCTG)
	dealNumber, err := r.nextDealNumber(ctx, tx, deal.BondCategory, deal.TradeDate)
	if err != nil {
		return err
	}

	deal.ID = uuid.New()
	deal.DealNumber = dealNumber

	branchID, _ := uuid.Parse("a0000000-0000-0000-0000-000000000001")

	if err := tx.QueryRow(ctx, `
		INSERT INTO bond_deals (
			id, deal_number, bond_category, trade_date, branch_id,
			order_date, value_date, direction, counterparty_id,
			transaction_type, transaction_type_other,
			bond_catalog_id, bond_code_manual, issuer,
			coupon_rate, issue_date, maturity_date,
			quantity, face_value, discount_rate,
			clean_price, settlement_price, total_value,
			portfolio_type, payment_date, remaining_tenor_days,
			confirmation_method, confirmation_other, contract_prepared_by,
			status, note, cloned_from_id,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23,
			$24, $25, $26,
			$27, $28, $29,
			$30, $31, $32,
			$33, $33
		)
		RETURNING created_at, updated_at`,
		deal.ID, dealNumber, deal.BondCategory, deal.TradeDate, branchID,
		deal.OrderDate, deal.ValueDate, deal.Direction, deal.CounterpartyID,
		deal.TransactionType, deal.TransactionTypeOther,
		deal.BondCatalogID, deal.BondCodeManual, deal.Issuer,
		deal.CouponRate, deal.IssueDate, deal.MaturityDate,
		deal.Quantity, deal.FaceValue, deal.DiscountRate,
		deal.CleanPrice, deal.SettlementPrice, deal.TotalValue,
		deal.PortfolioType, deal.PaymentDate, deal.RemainingTenorDays,
		deal.ConfirmationMethod, deal.ConfirmationOther, deal.ContractPreparedBy,
		deal.Status, deal.Note, deal.ClonedFromID,
		deal.CreatedBy,
	).Scan(&deal.CreatedAt, &deal.UpdatedAt); err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert bond_deal")
	}

	deal.Version = 1
	return tx.Commit(ctx)
}

func (r *pgxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.BondDeal, error) {
	deal := &model.BondDeal{}
	var counterpartyCode, counterpartyName *string
	var createdByName *string
	var branchCode, branchName *string

	err := r.pool.QueryRow(ctx, `
		SELECT d.id, d.deal_number, d.bond_category, d.trade_date,
			d.order_date, d.value_date, d.direction,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.transaction_type, d.transaction_type_other,
			d.bond_catalog_id, d.bond_code_manual, d.bond_code_display,
			d.issuer, d.coupon_rate, d.issue_date, d.maturity_date,
			d.quantity, d.face_value, d.discount_rate,
			d.clean_price, d.settlement_price, d.total_value,
			d.portfolio_type, d.payment_date, d.remaining_tenor_days,
			d.confirmation_method, d.contract_prepared_by,
			d.status, d.note, d.cloned_from_id,
			d.cancel_reason, d.cancel_requested_at,
			d.created_by, d.created_by_name, d.created_at, d.branch_code, d.branch_name
		FROM v_bond_deals_list d
		WHERE d.id = $1`, id,
	).Scan(
		&deal.ID, &deal.DealNumber, &deal.BondCategory, &deal.TradeDate,
		&deal.OrderDate, &deal.ValueDate, &deal.Direction,
		&deal.CounterpartyID, &counterpartyCode, &counterpartyName,
		&deal.TransactionType, &deal.TransactionTypeOther,
		&deal.BondCatalogID, &deal.BondCodeManual, &deal.BondCodeDisplay,
		&deal.Issuer, &deal.CouponRate, &deal.IssueDate, &deal.MaturityDate,
		&deal.Quantity, &deal.FaceValue, &deal.DiscountRate,
		&deal.CleanPrice, &deal.SettlementPrice, &deal.TotalValue,
		&deal.PortfolioType, &deal.PaymentDate, &deal.RemainingTenorDays,
		&deal.ConfirmationMethod, &deal.ContractPreparedBy,
		&deal.Status, &deal.Note, &deal.ClonedFromID,
		&deal.CancelReason, &deal.CancelRequestedAt,
		&deal.CreatedBy, &createdByName, &deal.CreatedAt, &branchCode, &branchName,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "bond deal not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query bond deal")
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
	deal.Version = 1 // Version tracked at service level via optimistic locking in Update

	// Read actual version from base table
	_ = r.pool.QueryRow(ctx, `SELECT updated_at FROM bond_deals WHERE id = $1`, id).Scan(&deal.UpdatedAt)

	return deal, nil
}

func (r *pgxRepository) buildFilterConditions(filter dto.BondDealListFilter) ([]string, []interface{}, int) {
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
	if filter.BondCategory != nil {
		conditions = append(conditions, fmt.Sprintf("d.bond_category = $%d", argIdx))
		args = append(args, *filter.BondCategory)
		argIdx++
	}
	if filter.Direction != nil {
		conditions = append(conditions, fmt.Sprintf("d.direction = $%d", argIdx))
		args = append(args, *filter.Direction)
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

func (r *pgxRepository) List(ctx context.Context, filter dto.BondDealListFilter, pag dto.PaginationRequest) ([]model.BondDeal, int64, error) {
	conditions, args, argIdx := r.buildFilterConditions(filter)

	whereClause := "TRUE"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM v_bond_deals_list d WHERE %s", whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count bond deals")
	}

	// Offset-based pagination
	sortCol := "d.created_at"
	allowedSorts := map[string]string{
		"created_at": "d.created_at", "trade_date": "d.trade_date",
		"status": "d.status", "total_value": "d.total_value",
	}
	if col, ok := allowedSorts[pag.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT d.id, d.deal_number, d.bond_category, d.trade_date,
			d.value_date, d.direction,
			d.counterparty_id, d.counterparty_code, d.counterparty_name,
			d.transaction_type, d.bond_code_display,
			d.issuer, d.quantity, d.face_value,
			d.settlement_price, d.total_value,
			d.portfolio_type, d.payment_date,
			d.status, d.note,
			d.created_by, d.created_by_name, d.created_at
		FROM v_bond_deals_list d
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query bond deals")
	}
	defer rows.Close()

	var deals []model.BondDeal
	for rows.Next() {
		var d model.BondDeal
		var counterpartyCode, counterpartyName *string
		var createdByName *string
		err := rows.Scan(
			&d.ID, &d.DealNumber, &d.BondCategory, &d.TradeDate,
			&d.ValueDate, &d.Direction,
			&d.CounterpartyID, &counterpartyCode, &counterpartyName,
			&d.TransactionType, &d.BondCodeDisplay,
			&d.Issuer, &d.Quantity, &d.FaceValue,
			&d.SettlementPrice, &d.TotalValue,
			&d.PortfolioType, &d.PaymentDate,
			&d.Status, &d.Note,
			&d.CreatedBy, &createdByName, &d.CreatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan bond deal")
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

func (r *pgxRepository) Update(ctx context.Context, deal *model.BondDeal) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bond_deals SET
			bond_category = $1, trade_date = $2, order_date = $3, value_date = $4,
			direction = $5, counterparty_id = $6,
			transaction_type = $7, transaction_type_other = $8,
			bond_catalog_id = $9, bond_code_manual = $10, issuer = $11,
			coupon_rate = $12, issue_date = $13, maturity_date = $14,
			quantity = $15, face_value = $16, discount_rate = $17,
			clean_price = $18, settlement_price = $19, total_value = $20,
			portfolio_type = $21, payment_date = $22, remaining_tenor_days = $23,
			confirmation_method = $24, confirmation_other = $25, contract_prepared_by = $26,
			note = $27, updated_by = $28
		WHERE id = $29 AND status = 'OPEN' AND deleted_at IS NULL`,
		deal.BondCategory, deal.TradeDate, deal.OrderDate, deal.ValueDate,
		deal.Direction, deal.CounterpartyID,
		deal.TransactionType, deal.TransactionTypeOther,
		deal.BondCatalogID, deal.BondCodeManual, deal.Issuer,
		deal.CouponRate, deal.IssueDate, deal.MaturityDate,
		deal.Quantity, deal.FaceValue, deal.DiscountRate,
		deal.CleanPrice, deal.SettlementPrice, deal.TotalValue,
		deal.PortfolioType, deal.PaymentDate, deal.RemainingTenorDays,
		deal.ConfirmationMethod, deal.ConfirmationOther, deal.ContractPreparedBy,
		deal.Note, deal.CreatedBy,
		deal.ID,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update bond deal")
	}
	if tag.RowsAffected() == 0 {
		var status string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM bond_deals WHERE id = $1 AND deleted_at IS NULL", deal.ID).Scan(&status)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "bond deal not found")
		}
		if status != constants.StatusOpen {
			return apperror.New(apperror.ErrDealLocked, "deal cannot be edited in current status")
		}
		return apperror.New(apperror.ErrConflict, "deal was modified by another user")
	}
	return nil
}

func (r *pgxRepository) UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bond_deals SET status = $1, updated_by = $2
		WHERE id = $3 AND status = $4 AND deleted_at IS NULL`,
		newStatus, updatedBy, id, oldStatus,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update bond deal status")
	}
	if tag.RowsAffected() == 0 {
		var currentStatus string
		checkErr := r.pool.QueryRow(ctx, "SELECT status FROM bond_deals WHERE id = $1 AND deleted_at IS NULL", id).Scan(&currentStatus)
		if checkErr == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "bond deal not found")
		}
		return apperror.NewWithDetail(apperror.ErrInvalidTransition,
			"status transition failed",
			fmt.Sprintf("expected status %s but found %s", oldStatus, currentStatus))
	}
	return nil
}

func (r *pgxRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bond_deals SET deleted_at = NOW(), updated_by = $1
		WHERE id = $2 AND deleted_at IS NULL AND status = 'OPEN'`,
		deletedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to soft delete bond deal")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "bond deal not found or cannot be deleted")
	}
	return nil
}

func (r *pgxRepository) UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE bond_deals SET cancel_reason = $1, cancel_requested_by = $2, cancel_requested_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL`,
		reason, requestedBy, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update cancel fields")
	}
	return nil
}

// --- Inventory ---

func (r *pgxRepository) CheckInventory(ctx context.Context, bondCode, bondCategory, portfolioType string) (int64, error) {
	var qty int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(available_quantity, 0)
		FROM bond_inventory
		WHERE bond_code = $1 AND bond_category = $2 AND portfolio_type = $3`,
		bondCode, bondCategory, portfolioType,
	).Scan(&qty)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, apperror.Wrap(err, apperror.ErrInternal, "failed to check inventory")
	}
	return qty, nil
}

func (r *pgxRepository) GetInventory(ctx context.Context, bondCode, bondCategory, portfolioType string) (*model.BondInventory, error) {
	inv := &model.BondInventory{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, bond_catalog_id, bond_code, bond_category, portfolio_type,
			available_quantity, acquisition_date, acquisition_price,
			version, updated_at, updated_by
		FROM bond_inventory
		WHERE bond_code = $1 AND bond_category = $2 AND portfolio_type = $3`,
		bondCode, bondCategory, portfolioType,
	).Scan(
		&inv.ID, &inv.BondCatalogID, &inv.BondCode, &inv.BondCategory, &inv.PortfolioType,
		&inv.AvailableQuantity, &inv.AcquisitionDate, &inv.AcquisitionPrice,
		&inv.Version, &inv.UpdatedAt, &inv.UpdatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get inventory")
	}
	return inv, nil
}

func (r *pgxRepository) ListInventory(ctx context.Context) ([]model.BondInventory, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, bond_code, bond_category, portfolio_type,
			available_quantity, acquisition_date, acquisition_price,
			version, updated_at,
			catalog_issuer, catalog_face_value, nominal_value, updated_by_name
		FROM v_bond_inventory_summary
		ORDER BY bond_code, portfolio_type`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to list inventory")
	}
	defer rows.Close()

	var items []model.BondInventory
	for rows.Next() {
		var inv model.BondInventory
		err := rows.Scan(
			&inv.ID, &inv.BondCode, &inv.BondCategory, &inv.PortfolioType,
			&inv.AvailableQuantity, &inv.AcquisitionDate, &inv.AcquisitionPrice,
			&inv.Version, &inv.UpdatedAt,
			&inv.CatalogIssuer, &inv.CatalogFaceValue, &inv.NominalValue, &inv.UpdatedByName,
		)
		if err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan inventory")
		}
		items = append(items, inv)
	}
	return items, nil
}

func (r *pgxRepository) IncrementInventory(ctx context.Context, bondCode, bondCategory, portfolioType string, quantity int64, updatedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bond_inventory (id, bond_code, bond_category, portfolio_type, available_quantity, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bond_code, bond_category, portfolio_type)
		DO UPDATE SET available_quantity = bond_inventory.available_quantity + $5,
			version = bond_inventory.version + 1, updated_by = $6`,
		uuid.New(), bondCode, bondCategory, portfolioType, quantity, updatedBy,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to increment inventory")
	}
	return nil
}

func (r *pgxRepository) DecrementInventory(ctx context.Context, bondCode, bondCategory, portfolioType string, quantity int64, updatedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bond_inventory
		SET available_quantity = available_quantity - $1,
			version = version + 1, updated_by = $2
		WHERE bond_code = $3 AND bond_category = $4 AND portfolio_type = $5
			AND available_quantity >= $1`,
		quantity, updatedBy, bondCode, bondCategory, portfolioType,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to decrement inventory")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrValidation, "insufficient inventory for sell operation")
	}
	return nil
}

// --- Deal Number Generation ---

// nextDealNumber generates a gapless deal number.
// G-YYYYMMDD-NNNN for GOVERNMENT, F-YYYYMMDD-NNNN for FINANCIAL_INSTITUTION / CERTIFICATE_OF_DEPOSIT.
func (r *pgxRepository) nextDealNumber(ctx context.Context, tx pgx.Tx, bondCategory string, tradeDate time.Time) (string, error) {
	prefix := "G"
	if bondCategory != constants.BondCategoryGovernment {
		prefix = "F"
	}

	dateStr := tradeDate.Format("20060102")
	datePart := tradeDate.Truncate(24 * time.Hour)

	seqPrefix := "BOND-" + prefix
	var seq int64
	err := tx.QueryRow(ctx, `
		INSERT INTO deal_sequences (module, prefix, date_partition, last_sequence)
		VALUES ('BOND', $1, $2, 1)
		ON CONFLICT (module, prefix, date_partition)
		DO UPDATE SET last_sequence = deal_sequences.last_sequence + 1
		RETURNING last_sequence`, seqPrefix, datePart,
	).Scan(&seq)

	if err != nil {
		return "", apperror.Wrap(err, apperror.ErrInternal, "failed to generate bond deal number")
	}

	return fmt.Sprintf("%s-%s-%04d", prefix, dateStr, seq), nil
}

// Ensure interface compliance
var _ repository.BondDealRepository = (*pgxRepository)(nil)

// insertDealDirect is a test helper that creates deals directly via pool.
func insertDealDirect(ctx context.Context, pool *pgxpool.Pool, deal *model.BondDeal) error {
	deal.ID = uuid.New()
	deal.Version = 1
	dealNumber := fmt.Sprintf("BOND-TEST-%s", deal.ID.String()[:8])
	deal.DealNumber = dealNumber

	_, err := pool.Exec(ctx, `
		INSERT INTO bond_deals (
			id, deal_number, bond_category, trade_date, branch_id,
			order_date, value_date, direction, counterparty_id,
			transaction_type, bond_catalog_id, bond_code_manual, issuer,
			coupon_rate, issue_date, maturity_date,
			quantity, face_value, discount_rate,
			clean_price, settlement_price, total_value,
			portfolio_type, payment_date, remaining_tenor_days,
			confirmation_method, contract_prepared_by,
			status, note, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19,
			$20, $21, $22,
			$23, $24, $25,
			$26, $27,
			$28, $29, $30, $30
		)`,
		deal.ID, dealNumber, deal.BondCategory, deal.TradeDate,
		"a0000000-0000-0000-0000-000000000001", // branch_id
		deal.OrderDate, deal.ValueDate, deal.Direction, deal.CounterpartyID,
		deal.TransactionType, deal.BondCatalogID, deal.BondCodeManual, deal.Issuer,
		deal.CouponRate, deal.IssueDate, deal.MaturityDate,
		deal.Quantity, deal.FaceValue, deal.DiscountRate,
		deal.CleanPrice, deal.SettlementPrice, deal.TotalValue,
		deal.PortfolioType, deal.PaymentDate, deal.RemainingTenorDays,
		deal.ConfirmationMethod, deal.ContractPreparedBy,
		deal.Status, deal.Note, deal.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insertDealDirect: %w", err)
	}
	return nil
}
