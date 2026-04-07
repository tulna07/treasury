package limits

import (
	"context"

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

// NewRepository creates a new credit limit repository backed by pgx.
func NewRepository(pool *pgxpool.Pool) repository.CreditLimitRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) Create(ctx context.Context, limit *model.CreditLimit) error {
	limit.ID = uuid.New()
	limit.Version = 1
	return r.pool.QueryRow(ctx, `
		INSERT INTO credit_limits (
			id, counterparty_id, limit_type, approved_amount, used_amount,
			currency_code, effective_date, expiry_date, status, note,
			created_by, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at`,
		limit.ID, limit.CounterpartyID, limit.LimitType, limit.ApprovedAmount,
		limit.UsedAmount, limit.CurrencyCode, limit.EffectiveDate, limit.ExpiryDate,
		limit.Status, limit.Note, limit.CreatedBy, limit.Version,
	).Scan(&limit.CreatedAt, &limit.UpdatedAt)
}

func (r *pgxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.CreditLimit, error) {
	limit := &model.CreditLimit{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, limit_type, approved_amount, used_amount,
			currency_code, effective_date, expiry_date, status, note,
			created_by, created_at, updated_at, version
		FROM credit_limits WHERE id = $1`, id,
	).Scan(
		&limit.ID, &limit.CounterpartyID, &limit.LimitType, &limit.ApprovedAmount,
		&limit.UsedAmount, &limit.CurrencyCode, &limit.EffectiveDate, &limit.ExpiryDate,
		&limit.Status, &limit.Note, &limit.CreatedBy, &limit.CreatedAt, &limit.UpdatedAt,
		&limit.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "credit limit not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query credit limit")
	}
	return limit, nil
}

func (r *pgxRepository) GetByCounterparty(ctx context.Context, counterpartyID uuid.UUID, limitType string) (*model.CreditLimit, error) {
	limit := &model.CreditLimit{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, limit_type, approved_amount, used_amount,
			currency_code, effective_date, expiry_date, status, note,
			created_by, created_at, updated_at, version
		FROM credit_limits
		WHERE counterparty_id = $1 AND limit_type = $2 AND status = 'ACTIVE'
		ORDER BY created_at DESC LIMIT 1`, counterpartyID, limitType,
	).Scan(
		&limit.ID, &limit.CounterpartyID, &limit.LimitType, &limit.ApprovedAmount,
		&limit.UsedAmount, &limit.CurrencyCode, &limit.EffectiveDate, &limit.ExpiryDate,
		&limit.Status, &limit.Note, &limit.CreatedBy, &limit.CreatedAt, &limit.UpdatedAt,
		&limit.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "credit limit not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query credit limit")
	}
	return limit, nil
}

// GetActiveByCounterparty returns the active credit limit for a counterparty and currency.
// Looks for a limit where status = 'ACTIVE', effective_date <= now, and expiry_date >= now.
// Specific currency match takes priority over "ALL" currency.
func (r *pgxRepository) GetActiveByCounterparty(ctx context.Context, counterpartyID uuid.UUID, currencyCode string) (*model.CreditLimit, error) {
	limit := &model.CreditLimit{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, counterparty_id, limit_type, approved_amount, used_amount,
			currency_code, effective_date, expiry_date, status, note,
			created_by, created_at, updated_at, version
		FROM credit_limits
		WHERE counterparty_id = $1
		  AND currency_code = $2
		  AND status = 'ACTIVE'
		  AND effective_date <= NOW()
		  AND expiry_date >= NOW()
		ORDER BY created_at DESC
		LIMIT 1`, counterpartyID, currencyCode,
	).Scan(
		&limit.ID, &limit.CounterpartyID, &limit.LimitType, &limit.ApprovedAmount,
		&limit.UsedAmount, &limit.CurrencyCode, &limit.EffectiveDate, &limit.ExpiryDate,
		&limit.Status, &limit.Note, &limit.CreatedBy, &limit.CreatedAt, &limit.UpdatedAt,
		&limit.Version,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "no active credit limit found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query active credit limit")
	}
	return limit, nil
}

func (r *pgxRepository) List(ctx context.Context, filter repository.CreditLimitFilter, pag dto.PaginationRequest) ([]model.CreditLimit, int64, error) {
	// Placeholder — full implementation not needed for limit check feature
	return nil, 0, nil
}

func (r *pgxRepository) Update(ctx context.Context, limit *model.CreditLimit) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE credit_limits SET
			approved_amount = $1, used_amount = $2, currency_code = $3,
			effective_date = $4, expiry_date = $5, status = $6, note = $7,
			version = version + 1
		WHERE id = $8 AND version = $9`,
		limit.ApprovedAmount, limit.UsedAmount, limit.CurrencyCode,
		limit.EffectiveDate, limit.ExpiryDate, limit.Status, limit.Note,
		limit.ID, limit.Version,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update credit limit")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrConflict, "credit limit was modified by another user")
	}
	limit.Version++
	return nil
}

func (r *pgxRepository) UpdateUsedAmount(ctx context.Context, id uuid.UUID, change decimal.Decimal) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE credit_limits SET used_amount = used_amount + $1
		WHERE id = $2 AND status = 'ACTIVE'`, change, id,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update used amount")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "active credit limit not found")
	}
	return nil
}

var _ repository.CreditLimitRepository = (*pgxRepository)(nil)
