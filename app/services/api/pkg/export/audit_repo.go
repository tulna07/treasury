package export

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ExportAuditLog represents an export audit trail entry.
type ExportAuditLog struct {
	ID             uuid.UUID `json:"id"`
	ExportCode     string    `json:"export_code"`
	UserID         uuid.UUID `json:"user_id"`
	Module         string    `json:"module"`
	ReportType     string    `json:"report_type"`
	DateFrom       time.Time `json:"date_from"`
	DateTo         time.Time `json:"date_to"`
	RecordCount    int       `json:"record_count"`
	MinioBucket    string    `json:"minio_bucket"`
	MinioObjectKey string    `json:"minio_object_key"`
	FileSizeBytes  int64     `json:"file_size_bytes"`
	FileChecksum   string    `json:"file_checksum"`
	ClientIP       string    `json:"client_ip"`
	UserAgent      string    `json:"user_agent"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ExportAuditRepository defines the interface for export audit log operations.
type ExportAuditRepository interface {
	Create(ctx context.Context, log *ExportAuditLog) error
	GetByCode(ctx context.Context, exportCode string) (*ExportAuditLog, error)
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]ExportAuditLog, int64, error)
	ListAll(ctx context.Context, limit, offset int) ([]ExportAuditLog, int64, error)
}
