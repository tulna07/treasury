package masterdata

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// --- Counterparty Repository ---

type counterpartyRepo struct {
	pool *pgxpool.Pool
}

// NewCounterpartyRepository creates a new counterparty repository.
func NewCounterpartyRepository(pool *pgxpool.Pool) repository.CounterpartyRepository {
	return &counterpartyRepo{pool: pool}
}

func (r *counterpartyRepo) Create(ctx context.Context, cp *model.Counterparty) error {
	cp.ID = uuid.New()
	err := r.pool.QueryRow(ctx, `
		INSERT INTO counterparties (id, code, full_name, short_name, cif, swift_code, country_code, tax_id, address, fx_uses_limit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at`,
		cp.ID, cp.Code, cp.FullName, cp.ShortName, cp.CIF,
		cp.SwiftCode, cp.CountryCode, cp.TaxID, cp.Address, cp.FxUsesLimit,
	).Scan(&cp.CreatedAt, &cp.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_counterparties_code") {
			return apperror.New(apperror.ErrConflict, "counterparty code already exists")
		}
		return apperror.Wrap(err, apperror.ErrInternal, "failed to create counterparty")
	}
	cp.IsActive = true
	return nil
}

func (r *counterpartyRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Counterparty, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, full_name, short_name, cif, swift_code, country_code,
		       tax_id, address, fx_uses_limit, is_active, created_at, updated_at
		FROM counterparties WHERE id = $1 AND deleted_at IS NULL`, id)

	cp := &model.Counterparty{}
	err := row.Scan(
		&cp.ID, &cp.Code, &cp.FullName, &cp.ShortName, &cp.CIF,
		&cp.SwiftCode, &cp.CountryCode, &cp.TaxID, &cp.Address,
		&cp.FxUsesLimit, &cp.IsActive, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "counterparty not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query counterparty")
	}
	return cp, nil
}

func (r *counterpartyRepo) GetByCode(ctx context.Context, code string) (*model.Counterparty, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, full_name, short_name, cif, swift_code, country_code,
		       tax_id, address, fx_uses_limit, is_active, created_at, updated_at
		FROM counterparties WHERE code = $1 AND deleted_at IS NULL`, code)

	cp := &model.Counterparty{}
	err := row.Scan(
		&cp.ID, &cp.Code, &cp.FullName, &cp.ShortName, &cp.CIF,
		&cp.SwiftCode, &cp.CountryCode, &cp.TaxID, &cp.Address,
		&cp.FxUsesLimit, &cp.IsActive, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query counterparty")
	}
	return cp, nil
}

func (r *counterpartyRepo) List(ctx context.Context, filter dto.CounterpartyFilter, pag dto.PaginationRequest) ([]model.Counterparty, int64, error) {
	conditions := []string{"deleted_at IS NULL"}
	var args []interface{}
	argIdx := 1

	if filter.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(code ILIKE $%d OR full_name ILIKE $%d OR short_name ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	var total int64
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM counterparties WHERE %s", whereClause), args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count counterparties")
	}

	query := fmt.Sprintf(`
		SELECT id, code, full_name, short_name, cif, swift_code, country_code,
		       tax_id, address, fx_uses_limit, is_active, created_at, updated_at
		FROM counterparties WHERE %s
		ORDER BY code ASC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query counterparties")
	}
	defer rows.Close()

	var result []model.Counterparty
	for rows.Next() {
		var cp model.Counterparty
		if err := rows.Scan(
			&cp.ID, &cp.Code, &cp.FullName, &cp.ShortName, &cp.CIF,
			&cp.SwiftCode, &cp.CountryCode, &cp.TaxID, &cp.Address,
			&cp.FxUsesLimit, &cp.IsActive, &cp.CreatedAt, &cp.UpdatedAt,
		); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan counterparty")
		}
		result = append(result, cp)
	}
	return result, total, nil
}

func (r *counterpartyRepo) ListActive(ctx context.Context) ([]model.Counterparty, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, full_name, short_name, cif, swift_code, country_code,
		       tax_id, address, fx_uses_limit, is_active, created_at, updated_at
		FROM counterparties WHERE deleted_at IS NULL AND is_active = true
		ORDER BY code ASC`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query active counterparties")
	}
	defer rows.Close()

	var result []model.Counterparty
	for rows.Next() {
		var cp model.Counterparty
		if err := rows.Scan(
			&cp.ID, &cp.Code, &cp.FullName, &cp.ShortName, &cp.CIF,
			&cp.SwiftCode, &cp.CountryCode, &cp.TaxID, &cp.Address,
			&cp.FxUsesLimit, &cp.IsActive, &cp.CreatedAt, &cp.UpdatedAt,
		); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan counterparty")
		}
		result = append(result, cp)
	}
	return result, nil
}

func (r *counterpartyRepo) Update(ctx context.Context, cp *model.Counterparty) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE counterparties SET
			full_name = $1, short_name = $2, swift_code = $3, country_code = $4,
			tax_id = $5, address = $6, fx_uses_limit = $7
		WHERE id = $8 AND deleted_at IS NULL`,
		cp.FullName, cp.ShortName, cp.SwiftCode, cp.CountryCode,
		cp.TaxID, cp.Address, cp.FxUsesLimit, cp.ID,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update counterparty")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "counterparty not found")
	}
	return nil
}

func (r *counterpartyRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE counterparties SET deleted_at = NOW(), updated_by = $1 WHERE id = $2 AND deleted_at IS NULL",
		deletedBy, id)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to soft delete counterparty")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "counterparty not found")
	}
	return nil
}

var _ repository.CounterpartyRepository = (*counterpartyRepo)(nil)

// --- Master Data Repository ---

type masterDataRepo struct {
	pool *pgxpool.Pool
}

// NewMasterDataRepository creates a new master data repository.
func NewMasterDataRepository(pool *pgxpool.Pool) repository.MasterDataRepository {
	return &masterDataRepo{pool: pool}
}

func (r *masterDataRepo) ListCurrencies(ctx context.Context) ([]model.Currency, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, numeric_code, name, decimal_places, is_active
		FROM currencies WHERE is_active = true ORDER BY code`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query currencies")
	}
	defer rows.Close()

	var result []model.Currency
	for rows.Next() {
		var c model.Currency
		if err := rows.Scan(&c.ID, &c.Code, &c.NumericCode, &c.Name, &c.DecimalPlaces, &c.IsActive); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan currency")
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *masterDataRepo) ListCurrencyPairs(ctx context.Context) ([]model.CurrencyPair, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, base_currency, quote_currency, pair_code,
		       rate_decimal_places, calculation_rule, result_currency, is_active
		FROM currency_pairs WHERE is_active = true ORDER BY pair_code`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query currency pairs")
	}
	defer rows.Close()

	var result []model.CurrencyPair
	for rows.Next() {
		var cp model.CurrencyPair
		if err := rows.Scan(&cp.ID, &cp.BaseCurrency, &cp.QuoteCurrency, &cp.PairCode,
			&cp.RateDecimalPlaces, &cp.CalculationRule, &cp.ResultCurrency, &cp.IsActive); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan currency pair")
		}
		result = append(result, cp)
	}
	return result, nil
}

func (r *masterDataRepo) ListBranches(ctx context.Context) ([]model.Branch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, branch_type, parent_branch_id,
		       flexcube_branch_code, swift_branch_code, address, is_active
		FROM branches WHERE is_active = true ORDER BY code`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query branches")
	}
	defer rows.Close()

	var result []model.Branch
	for rows.Next() {
		var b model.Branch
		if err := rows.Scan(&b.ID, &b.Code, &b.Name, &b.BranchType, &b.ParentBranchID,
			&b.FlexcubeBranch, &b.SwiftBranchCode, &b.Address, &b.IsActive); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan branch")
		}
		result = append(result, b)
	}
	return result, nil
}

func (r *masterDataRepo) GetBranchByID(ctx context.Context, id uuid.UUID) (*model.Branch, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, code, name, branch_type, parent_branch_id,
		       flexcube_branch_code, swift_branch_code, address, is_active
		FROM branches WHERE id = $1`, id)

	b := &model.Branch{}
	err := row.Scan(&b.ID, &b.Code, &b.Name, &b.BranchType, &b.ParentBranchID,
		&b.FlexcubeBranch, &b.SwiftBranchCode, &b.Address, &b.IsActive)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "branch not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query branch")
	}
	return b, nil
}

func (r *masterDataRepo) ListExchangeRates(ctx context.Context, filter dto.ExchangeRateFilter, pag dto.PaginationRequest) ([]model.ExchangeRate, int64, error) {
	conditions := []string{"TRUE"}
	var args []interface{}
	argIdx := 1

	if filter.CurrencyCode != nil {
		conditions = append(conditions, fmt.Sprintf("currency_code = $%d", argIdx))
		args = append(args, *filter.CurrencyCode)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("effective_date >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("effective_date <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	var total int64
	err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM exchange_rates WHERE %s", whereClause), args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count exchange rates")
	}

	query := fmt.Sprintf(`
		SELECT id, currency_code, effective_date, buy_transfer_rate, sell_transfer_rate, mid_rate, source, created_at
		FROM exchange_rates WHERE %s
		ORDER BY effective_date DESC, currency_code ASC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query exchange rates")
	}
	defer rows.Close()

	var result []model.ExchangeRate
	for rows.Next() {
		var er model.ExchangeRate
		if err := rows.Scan(&er.ID, &er.CurrencyCode, &er.EffectiveDate,
			&er.BuyTransferRate, &er.SellTransferRate, &er.MidRate, &er.Source, &er.CreatedAt); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan exchange rate")
		}
		result = append(result, er)
	}
	return result, total, nil
}

func (r *masterDataRepo) GetLatestRate(ctx context.Context, currencyCode string) (*model.ExchangeRate, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, currency_code, effective_date, buy_transfer_rate, sell_transfer_rate, mid_rate, source, created_at
		FROM exchange_rates WHERE currency_code = $1
		ORDER BY effective_date DESC LIMIT 1`, currencyCode)

	er := &model.ExchangeRate{}
	err := row.Scan(&er.ID, &er.CurrencyCode, &er.EffectiveDate,
		&er.BuyTransferRate, &er.SellTransferRate, &er.MidRate, &er.Source, &er.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "no exchange rate found for "+currencyCode)
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query latest exchange rate")
	}
	return er, nil
}

var _ repository.MasterDataRepository = (*masterDataRepo)(nil)
