// Package export provides a shared export engine for generating encrypted Excel reports.
package export

import (
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// ReportBuilder defines the interface for building Excel report sheets.
type ReportBuilder interface {
	Module() string
	ReportType() string
	BuildSheets(f *excelize.File) error
	RecordCount() int
}

// ExportParams holds the parameters for an export operation.
type ExportParams struct {
	User      UserInfo
	DateFrom  time.Time
	DateTo    time.Time
	Password  string
	Filters   map[string]any
	ClientIP  string
	UserAgent string
}

// UserInfo holds the exporting user's identity.
type UserInfo struct {
	ID       uuid.UUID
	Username string
	FullName string
	Role     string
}

// ExportResult holds the result of a successful export.
type ExportResult struct {
	ExportCode string `json:"export_code"`
	FileSize   int64  `json:"file_size"`
	Checksum   string `json:"checksum"`
	MinIOKey   string `json:"minio_key"`
}

// ExportConfig holds configuration for the export engine.
type ExportConfig struct {
	RetentionDays  int
	MinioBucket    string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
}
