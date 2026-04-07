package dto

import (
	"time"

	"github.com/google/uuid"
)

// ExportRequest is the payload for requesting a data export.
type ExportRequest struct {
	From     string `json:"from" validate:"required"`
	To       string `json:"to" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ExportAuditResponse represents an export audit log entry.
type ExportAuditResponse struct {
	ID             uuid.UUID `json:"id"`
	ExportCode     string    `json:"export_code"`
	UserID         uuid.UUID `json:"user_id"`
	Module         string    `json:"module"`
	ReportType     string    `json:"report_type"`
	DateFrom       time.Time `json:"date_from"`
	DateTo         time.Time `json:"date_to"`
	RecordCount    int       `json:"record_count"`
	FileSizeBytes  int64     `json:"file_size_bytes"`
	FileChecksum   string    `json:"file_checksum"`
	ClientIP       string    `json:"client_ip"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// ExportListResponse represents a paginated list of exports.
type ExportListResponse struct {
	Data  []ExportAuditResponse `json:"data"`
	Total int64                 `json:"total"`
}
