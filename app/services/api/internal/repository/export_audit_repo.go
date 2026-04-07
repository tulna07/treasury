package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/export"
)

// ExportAuditRepo implements export.ExportAuditRepository using PostgreSQL.
type ExportAuditRepo struct {
	pool *pgxpool.Pool
}

// NewExportAuditRepo creates a new ExportAuditRepo.
func NewExportAuditRepo(pool *pgxpool.Pool) *ExportAuditRepo {
	return &ExportAuditRepo{pool: pool}
}

// Create inserts a new export audit log entry.
func (r *ExportAuditRepo) Create(ctx context.Context, log *export.ExportAuditLog) error {
	if r.pool == nil {
		return nil
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO treasury.export_audit_logs (
			id, export_code, user_id, module, report_type,
			date_from, date_to, record_count,
			minio_bucket, minio_object_key,
			file_size_bytes, file_checksum,
			client_ip, user_agent, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		log.ID, log.ExportCode, log.UserID, log.Module, log.ReportType,
		log.DateFrom, log.DateTo, log.RecordCount,
		log.MinioBucket, log.MinioObjectKey,
		log.FileSizeBytes, log.FileChecksum,
		log.ClientIP, log.UserAgent, log.CreatedAt, log.ExpiresAt,
	)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to create export audit log")
	}
	return nil
}

// GetByCode retrieves an export audit log by export code.
func (r *ExportAuditRepo) GetByCode(ctx context.Context, exportCode string) (*export.ExportAuditLog, error) {
	if r.pool == nil {
		return nil, apperror.New(apperror.ErrNotFound, "export not found")
	}
	var log export.ExportAuditLog
	err := r.pool.QueryRow(ctx, `
		SELECT id, export_code, user_id, module, report_type,
			date_from, date_to, record_count,
			minio_bucket, minio_object_key,
			file_size_bytes, file_checksum,
			client_ip, user_agent, created_at, expires_at
		FROM treasury.export_audit_logs
		WHERE export_code = $1`, exportCode).Scan(
		&log.ID, &log.ExportCode, &log.UserID, &log.Module, &log.ReportType,
		&log.DateFrom, &log.DateTo, &log.RecordCount,
		&log.MinioBucket, &log.MinioObjectKey,
		&log.FileSizeBytes, &log.FileChecksum,
		&log.ClientIP, &log.UserAgent, &log.CreatedAt, &log.ExpiresAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "export not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get export audit log")
	}
	return &log, nil
}

// ListByUser lists export audit logs for a specific user.
func (r *ExportAuditRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]export.ExportAuditLog, int64, error) {
	if r.pool == nil {
		return nil, 0, nil
	}

	var total int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM treasury.export_audit_logs WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count export audit logs")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, export_code, user_id, module, report_type,
			date_from, date_to, record_count,
			minio_bucket, minio_object_key,
			file_size_bytes, file_checksum,
			client_ip, user_agent, created_at, expires_at
		FROM treasury.export_audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list export audit logs")
	}
	defer rows.Close()

	return scanExportAuditLogs(rows, total)
}

// ListAll lists all export audit logs.
func (r *ExportAuditRepo) ListAll(ctx context.Context, limit, offset int) ([]export.ExportAuditLog, int64, error) {
	if r.pool == nil {
		return nil, 0, nil
	}

	var total int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM treasury.export_audit_logs`).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count export audit logs")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, export_code, user_id, module, report_type,
			date_from, date_to, record_count,
			minio_bucket, minio_object_key,
			file_size_bytes, file_checksum,
			client_ip, user_agent, created_at, expires_at
		FROM treasury.export_audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to list export audit logs")
	}
	defer rows.Close()

	return scanExportAuditLogs(rows, total)
}

func scanExportAuditLogs(rows pgx.Rows, total int64) ([]export.ExportAuditLog, int64, error) {
	var logs []export.ExportAuditLog
	for rows.Next() {
		var log export.ExportAuditLog
		if err := rows.Scan(
			&log.ID, &log.ExportCode, &log.UserID, &log.Module, &log.ReportType,
			&log.DateFrom, &log.DateTo, &log.RecordCount,
			&log.MinioBucket, &log.MinioObjectKey,
			&log.FileSizeBytes, &log.FileChecksum,
			&log.ClientIP, &log.UserAgent, &log.CreatedAt, &log.ExpiresAt,
		); err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan export audit log")
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}
