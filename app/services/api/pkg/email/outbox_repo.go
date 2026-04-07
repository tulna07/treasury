package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxRepository defines the interface for email outbox persistence.
type OutboxRepository interface {
	Insert(ctx context.Context, email *OutboxEmail) error
	FetchPending(ctx context.Context, limit int) ([]OutboxEmail, error)
	MarkSending(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkRetry(ctx context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	GetByID(ctx context.Context, id uuid.UUID) (*OutboxEmail, error)
	CountByStatus(ctx context.Context) (map[string]int, error)
	ListRecent(ctx context.Context, limit int) ([]OutboxEmail, error)
	CountSent24h(ctx context.Context) (int, error)
	CountFailed24h(ctx context.Context) (int, error)
	OldestPendingMinutes(ctx context.Context) (float64, error)
	ListByStatus(ctx context.Context, status string, limit int) ([]OutboxEmail, error)
}

// PgOutboxRepository implements OutboxRepository with PostgreSQL.
type PgOutboxRepository struct {
	pool *pgxpool.Pool
}

// NewPgOutboxRepository creates a new PostgreSQL outbox repository.
func NewPgOutboxRepository(pool *pgxpool.Pool) *PgOutboxRepository {
	return &PgOutboxRepository{pool: pool}
}

func (r *PgOutboxRepository) Insert(ctx context.Context, email *OutboxEmail) error {
	if email.ID == uuid.Nil {
		email.ID = uuid.New()
	}

	templateDataJSON, err := json.Marshal(email.TemplateData)
	if err != nil {
		return fmt.Errorf("marshal template_data: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO email_outbox (
			id, to_addresses, cc_addresses, from_address, subject,
			body_html, body_text, template_name, template_data,
			deal_module, deal_id, trigger_event, triggered_by,
			status, retry_count, max_retries, next_retry_at,
			last_error, idempotency_key, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20
		)`,
		email.ID, email.ToAddresses, email.CCAddresses, email.FromAddress, email.Subject,
		email.BodyHTML, email.BodyText, email.TemplateName, templateDataJSON,
		email.DealModule, email.DealID, email.TriggerEvent, email.TriggeredBy,
		email.Status, email.RetryCount, email.MaxRetries, email.NextRetryAt,
		email.LastError, email.IdempotencyKey, email.CreatedAt,
	)
	return err
}

func (r *PgOutboxRepository) FetchPending(ctx context.Context, limit int) ([]OutboxEmail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, to_addresses, cc_addresses, from_address, subject,
			body_html, body_text, template_name, template_data,
			deal_module, deal_id, trigger_event, triggered_by,
			status, retry_count, max_retries, next_retry_at,
			last_error, idempotency_key, created_at, sent_at, failed_at
		FROM email_outbox
		WHERE status IN ('PENDING', 'RETRY')
		  AND next_retry_at <= now()
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEmails(rows)
}

func (r *PgOutboxRepository) MarkSending(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_outbox SET status = 'SENDING' WHERE id = $1`, id)
	return err
}

func (r *PgOutboxRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_outbox SET status = 'SENT', sent_at = now() WHERE id = $1`, id)
	return err
}

func (r *PgOutboxRepository) MarkRetry(ctx context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_outbox SET status = 'RETRY', last_error = $2, next_retry_at = $3, retry_count = retry_count + 1 WHERE id = $1`,
		id, errMsg, nextRetryAt)
	return err
}

func (r *PgOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_outbox SET status = 'FAILED', last_error = $2, failed_at = now() WHERE id = $1`,
		id, errMsg)
	return err
}

func (r *PgOutboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*OutboxEmail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, to_addresses, cc_addresses, from_address, subject,
			body_html, body_text, template_name, template_data,
			deal_module, deal_id, trigger_event, triggered_by,
			status, retry_count, max_retries, next_retry_at,
			last_error, idempotency_key, created_at, sent_at, failed_at
		FROM email_outbox WHERE id = $1`, id)

	e, err := scanEmail(row)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PgOutboxRepository) CountByStatus(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM email_outbox GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, rows.Err()
}

func (r *PgOutboxRepository) CountSent24h(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_outbox WHERE status = 'SENT' AND sent_at >= now() - interval '24 hours'`).Scan(&count)
	return count, err
}

func (r *PgOutboxRepository) CountFailed24h(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_outbox WHERE status = 'FAILED' AND failed_at >= now() - interval '24 hours'`).Scan(&count)
	return count, err
}

func (r *PgOutboxRepository) OldestPendingMinutes(ctx context.Context) (float64, error) {
	var minutes *float64
	err := r.pool.QueryRow(ctx,
		`SELECT EXTRACT(EPOCH FROM (now() - MIN(created_at))) / 60
		 FROM email_outbox
		 WHERE status IN ('PENDING', 'RETRY')`).Scan(&minutes)
	if err != nil {
		return 0, err
	}
	if minutes == nil {
		return 0, nil
	}
	return *minutes, nil
}

func (r *PgOutboxRepository) ListRecent(ctx context.Context, limit int) ([]OutboxEmail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, to_addresses, cc_addresses, from_address, subject,
			body_html, body_text, template_name, template_data,
			deal_module, deal_id, trigger_event, triggered_by,
			status, retry_count, max_retries, next_retry_at,
			last_error, idempotency_key, created_at, sent_at, failed_at
		FROM email_outbox
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEmails(rows)
}

func (r *PgOutboxRepository) ListByStatus(ctx context.Context, status string, limit int) ([]OutboxEmail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, to_addresses, cc_addresses, from_address, subject,
			body_html, body_text, template_name, template_data,
			deal_module, deal_id, trigger_event, triggered_by,
			status, retry_count, max_retries, next_retry_at,
			last_error, idempotency_key, created_at, sent_at, failed_at
		FROM email_outbox
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEmails(rows)
}

func scanEmails(rows pgx.Rows) ([]OutboxEmail, error) {
	var emails []OutboxEmail
	for rows.Next() {
		e, err := scanEmailRow(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func scanEmail(row pgx.Row) (OutboxEmail, error) {
	var e OutboxEmail
	var templateDataJSON []byte
	err := row.Scan(
		&e.ID, &e.ToAddresses, &e.CCAddresses, &e.FromAddress, &e.Subject,
		&e.BodyHTML, &e.BodyText, &e.TemplateName, &templateDataJSON,
		&e.DealModule, &e.DealID, &e.TriggerEvent, &e.TriggeredBy,
		&e.Status, &e.RetryCount, &e.MaxRetries, &e.NextRetryAt,
		&e.LastError, &e.IdempotencyKey, &e.CreatedAt, &e.SentAt, &e.FailedAt,
	)
	if err != nil {
		return e, err
	}
	if len(templateDataJSON) > 0 {
		_ = json.Unmarshal(templateDataJSON, &e.TemplateData)
	}
	return e, nil
}

func scanEmailRow(rows pgx.Rows) (OutboxEmail, error) {
	var e OutboxEmail
	var templateDataJSON []byte
	err := rows.Scan(
		&e.ID, &e.ToAddresses, &e.CCAddresses, &e.FromAddress, &e.Subject,
		&e.BodyHTML, &e.BodyText, &e.TemplateName, &templateDataJSON,
		&e.DealModule, &e.DealID, &e.TriggerEvent, &e.TriggeredBy,
		&e.Status, &e.RetryCount, &e.MaxRetries, &e.NextRetryAt,
		&e.LastError, &e.IdempotencyKey, &e.CreatedAt, &e.SentAt, &e.FailedAt,
	)
	if err != nil {
		return e, err
	}
	if len(templateDataJSON) > 0 {
		_ = json.Unmarshal(templateDataJSON, &e.TemplateData)
	}
	return e, nil
}
