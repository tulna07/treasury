package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
)

// NotificationRepo is the PostgreSQL implementation of NotificationRepository.
type NotificationRepo struct {
	pool *pgxpool.Pool
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

// Create inserts a new notification.
func (r *NotificationRepo) Create(ctx context.Context, n *model.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, user_id, title, message, category, deal_module, deal_id, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		n.ID, n.UserID, n.Title, n.Message, n.Category, n.DealModule, n.DealID, n.IsRead, n.CreatedAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to create notification")
	}
	return nil
}

// GetByID retrieves a notification by its ID.
func (r *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, message, category, deal_module, deal_id, is_read, read_at, created_at
		FROM notifications WHERE id = $1`, id)

	var n model.Notification
	err := row.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category, &n.DealModule, &n.DealID, &n.IsRead, &n.ReadAt, &n.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperror.New(apperror.ErrNotFound, "notification not found")
	}
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get notification")
	}
	return &n, nil
}

// ListByUser lists notifications for a user with optional unread filter.
func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, offset, limit int) ([]model.Notification, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	args := []interface{}{userID}
	if unreadOnly {
		countQuery += ` AND is_read = false`
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count notifications")
	}

	// Fetch rows
	query := `SELECT id, user_id, title, message, category, deal_module, deal_id, is_read, read_at, created_at
		FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list notifications")
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Category, &n.DealModule, &n.DealID, &n.IsRead, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan notification")
		}
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []model.Notification{}
	}

	return notifications, total, nil
}

// MarkRead marks a single notification as read for the given user.
func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = true, read_at = now()
		WHERE id = $1 AND user_id = $2 AND is_read = false`, id, userID)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to mark notification as read")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "notification not found or already read")
	}
	return nil
}

// MarkAllRead marks all unread notifications as read for the given user.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications SET is_read = true, read_at = now()
		WHERE user_id = $1 AND is_read = false`, userID)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to mark all notifications as read")
	}
	return nil
}

// CountUnread returns the number of unread notifications for the given user.
func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	if err != nil {
		return 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count unread notifications")
	}
	return count, nil
}

// DeleteOld deletes notifications older than the given time. Returns the number deleted.
func (r *NotificationRepo) DeleteOld(ctx context.Context, olderThan time.Time) (int, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM notifications WHERE created_at < $1`, olderThan)
	if err != nil {
		return 0, apperror.Wrap(err, apperror.ErrInternal, "failed to delete old notifications")
	}
	return int(tag.RowsAffected()), nil
}

// ListUserIDsByRole returns the IDs of all active users that have the given role.
func (r *NotificationRepo) ListUserIDsByRole(ctx context.Context, role string) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles ro ON ro.id = ur.role_id
		WHERE ro.code = $1 AND u.is_active = true AND u.deleted_at IS NULL`, role)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to list users by role")
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan user id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
