package settlement

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
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

type pgxRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new international payment repository backed by pgx.
func NewRepository(pool *pgxpool.Pool) repository.InternationalPaymentRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) Create(ctx context.Context, payment *model.InternationalPayment) error {
	payment.ID = uuid.New()
	err := r.pool.QueryRow(ctx, `
		INSERT INTO international_payments (
			id, source_module, source_deal_id, source_leg_number,
			ticket_display, counterparty_id, debit_account, bic_code,
			currency_code, amount, transfer_date, counterparty_ssi,
			original_trade_date, approved_by_division, settlement_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING created_at`,
		payment.ID, payment.SourceModule, payment.SourceDealID, payment.SourceLegNumber,
		payment.TicketDisplay, payment.CounterpartyID, payment.DebitAccount, payment.BICCode,
		payment.CurrencyCode, payment.Amount, payment.TransferDate, payment.CounterpartySSI,
		payment.OriginalTradeDate, payment.ApprovedByDivision, payment.SettlementStatus,
	).Scan(&payment.CreatedAt)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to insert international_payment")
	}
	return nil
}

const listColumns = `
	v.id, v.source_module, v.source_deal_id, v.source_leg_number,
	v.ticket_display, v.counterparty_id, v.counterparty_code, v.counterparty_name,
	v.debit_account, v.bic_code, v.currency_code, v.amount,
	v.transfer_date, v.counterparty_ssi, v.original_trade_date,
	v.approved_by_division, v.settlement_status,
	v.settled_by, v.settled_by_name, v.settled_at,
	v.rejection_reason, v.created_at`

func scanPayment(row pgx.Row) (*model.InternationalPayment, error) {
	p := &model.InternationalPayment{}
	err := row.Scan(
		&p.ID, &p.SourceModule, &p.SourceDealID, &p.SourceLegNumber,
		&p.TicketDisplay, &p.CounterpartyID, &p.CounterpartyCode, &p.CounterpartyName,
		&p.DebitAccount, &p.BICCode, &p.CurrencyCode, &p.Amount,
		&p.TransferDate, &p.CounterpartySSI, &p.OriginalTradeDate,
		&p.ApprovedByDivision, &p.SettlementStatus,
		&p.SettledBy, &p.SettledByName, &p.SettledAt,
		&p.RejectionReason, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *pgxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.InternationalPayment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+listColumns+` FROM v_international_payments_list v WHERE v.id = $1`, id)
	p, err := scanPayment(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "international payment not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get international_payment")
	}
	return p, nil
}

func (r *pgxRepository) List(ctx context.Context, filter dto.InternationalPaymentFilter, pag dto.PaginationRequest) ([]model.InternationalPayment, int64, error) {
	where, args := buildListWhere(filter)

	// Count
	var total int64
	countQ := `SELECT COUNT(*) FROM v_international_payments_list v` + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count international_payments")
	}
	if total == 0 {
		return []model.InternationalPayment{}, 0, nil
	}

	// Data
	offset := (pag.Page - 1) * pag.PageSize
	sortBy := "created_at"
	if pag.SortBy != "" {
		allowed := map[string]bool{
			"transfer_date": true, "amount": true, "settlement_status": true,
			"created_at": true, "ticket_display": true, "counterparty_name": true,
		}
		if allowed[pag.SortBy] {
			sortBy = pag.SortBy
		}
	}
	sortDir := "DESC"
	if pag.SortDir == "asc" {
		sortDir = "ASC"
	}

	dataQ := fmt.Sprintf(`SELECT %s FROM v_international_payments_list v%s ORDER BY v.%s %s LIMIT %d OFFSET %d`,
		listColumns, where, sortBy, sortDir, pag.PageSize, offset)

	rows, err := r.pool.Query(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list international_payments")
	}
	defer rows.Close()

	var payments []model.InternationalPayment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan international_payment")
		}
		payments = append(payments, *p)
	}
	if payments == nil {
		payments = []model.InternationalPayment{}
	}
	return payments, total, nil
}

func (r *pgxRepository) Approve(ctx context.Context, id uuid.UUID, settledBy uuid.UUID) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx, `
		UPDATE international_payments
		SET settlement_status = 'APPROVED', settled_by = $2, settled_at = $3
		WHERE id = $1 AND settlement_status = 'PENDING'`,
		id, settledBy, now)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to approve international_payment")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrConflict, "payment is not in PENDING status")
	}
	return nil
}

func (r *pgxRepository) Reject(ctx context.Context, id uuid.UUID, settledBy uuid.UUID, reason string) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx, `
		UPDATE international_payments
		SET settlement_status = 'REJECTED', settled_by = $2, settled_at = $3, rejection_reason = $4
		WHERE id = $1 AND settlement_status = 'PENDING'`,
		id, settledBy, now, reason)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to reject international_payment")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrConflict, "payment is not in PENDING status")
	}
	return nil
}

func buildListWhere(f dto.InternationalPaymentFilter) (string, []interface{}) {
	var conds []string
	var args []interface{}
	idx := 1

	if f.SettlementStatus != nil {
		conds = append(conds, fmt.Sprintf("v.settlement_status = $%d", idx))
		args = append(args, *f.SettlementStatus)
		idx++
	}
	if f.SourceModule != nil {
		conds = append(conds, fmt.Sprintf("v.source_module = $%d", idx))
		args = append(args, *f.SourceModule)
		idx++
	}
	if f.TransferDateFrom != nil {
		conds = append(conds, fmt.Sprintf("v.transfer_date >= $%d", idx))
		args = append(args, *f.TransferDateFrom)
		idx++
	}
	if f.TransferDateTo != nil {
		conds = append(conds, fmt.Sprintf("v.transfer_date <= $%d", idx))
		args = append(args, *f.TransferDateTo)
		idx++
	}
	if f.CounterpartyID != nil {
		conds = append(conds, fmt.Sprintf("v.counterparty_id = $%d", idx))
		args = append(args, *f.CounterpartyID)
		idx++
	}
	if f.TicketDisplay != nil {
		conds = append(conds, fmt.Sprintf("v.ticket_display ILIKE $%d", idx))
		args = append(args, "%"+*f.TicketDisplay+"%")
		idx++
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}
