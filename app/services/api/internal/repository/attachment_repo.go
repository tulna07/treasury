package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
)

// AttachmentRepo is the PostgreSQL implementation of AttachmentRepository.
type AttachmentRepo struct {
	pool *pgxpool.Pool
}

// NewAttachmentRepo creates a new AttachmentRepo.
func NewAttachmentRepo(pool *pgxpool.Pool) *AttachmentRepo {
	return &AttachmentRepo{pool: pool}
}

func (r *AttachmentRepo) Create(ctx context.Context, a *model.DealAttachment) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO deal_attachments (id, deal_module, deal_id, file_name, file_size, content_type, minio_bucket, minio_key, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.DealModule, a.DealID, a.FileName, a.FileSize, a.ContentType, a.MinioBucket, a.MinioKey, a.UploadedBy,
	)
	return err
}

func (r *AttachmentRepo) ListByDeal(ctx context.Context, module string, dealID uuid.UUID) ([]model.DealAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, deal_module, deal_id, file_name, file_size, content_type, minio_bucket, minio_key, uploaded_by, created_at
		FROM deal_attachments
		WHERE deal_module = $1 AND deal_id = $2
		ORDER BY created_at ASC`, module, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.DealAttachment
	for rows.Next() {
		var a model.DealAttachment
		if err := rows.Scan(&a.ID, &a.DealModule, &a.DealID, &a.FileName, &a.FileSize, &a.ContentType, &a.MinioBucket, &a.MinioKey, &a.UploadedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *AttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.DealAttachment, error) {
	var a model.DealAttachment
	err := r.pool.QueryRow(ctx, `
		SELECT id, deal_module, deal_id, file_name, file_size, content_type, minio_bucket, minio_key, uploaded_by, created_at
		FROM deal_attachments
		WHERE id = $1`, id).
		Scan(&a.ID, &a.DealModule, &a.DealID, &a.FileName, &a.FileSize, &a.ContentType, &a.MinioBucket, &a.MinioKey, &a.UploadedBy, &a.CreatedAt)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "attachment not found")
	}
	return &a, nil
}

func (r *AttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM deal_attachments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "attachment not found")
	}
	return nil
}

func (r *AttachmentRepo) CountByDeal(ctx context.Context, module string, dealID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM deal_attachments WHERE deal_module = $1 AND deal_id = $2`,
		module, dealID).Scan(&count)
	return count, err
}
