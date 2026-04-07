package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// Engine orchestrates the export process: build Excel, encrypt, upload to MinIO, audit.
type Engine struct {
	minio    *minio.Client
	auditRepo ExportAuditRepository
	config   ExportConfig
	logger   *zap.Logger
}

// NewEngine creates a new export engine.
func NewEngine(auditRepo ExportAuditRepository, cfg ExportConfig, logger *zap.Logger) (*Engine, error) {
	var minioClient *minio.Client
	if cfg.MinioEndpoint != "" {
		var err error
		minioClient, err = newMinioClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create MinIO client: %w", err)
		}
	}

	return &Engine{
		minio:     minioClient,
		auditRepo: auditRepo,
		config:    cfg,
		logger:    logger,
	}, nil
}

// NewEngineWithClient creates an engine with pre-configured clients (for testing).
func NewEngineWithClient(minioClient *minio.Client, auditRepo ExportAuditRepository, cfg ExportConfig, logger *zap.Logger) *Engine {
	return &Engine{
		minio:     minioClient,
		auditRepo: auditRepo,
		config:    cfg,
		logger:    logger,
	}
}

// GenerateExportCode generates a unique export code: EXP-YYYYMMDD-HHMMSS-XXXX
func GenerateExportCode() string {
	now := time.Now()
	suffix := fmt.Sprintf("%04d", rand.Intn(10000))
	return fmt.Sprintf("EXP-%s-%s-%s",
		now.Format("20060102"),
		now.Format("150405"),
		suffix,
	)
}

// Execute runs the full export pipeline: build sheets → encrypt → upload → audit.
func (e *Engine) Execute(ctx context.Context, builder ReportBuilder, params ExportParams) (*ExportResult, []byte, error) {
	exportCode := GenerateExportCode()

	// Create workbook
	f := excelize.NewFile()
	defer f.Close()

	// Build disclaimer sheet
	if err := buildDisclaimer(f, params, exportCode); err != nil {
		return nil, nil, fmt.Errorf("failed to build disclaimer: %w", err)
	}

	// Build report sheets
	if err := builder.BuildSheets(f); err != nil {
		return nil, nil, fmt.Errorf("failed to build report sheets: %w", err)
	}

	// Set document properties
	if err := setDocProperties(f, params, builder); err != nil {
		return nil, nil, fmt.Errorf("failed to set doc properties: %w", err)
	}

	// Remove default "Sheet1" if it still exists
	_ = f.DeleteSheet("Sheet1")

	// Set active sheet to disclaimer
	if idx, err := f.GetSheetIndex("Tuyên bố miễn trừ"); err == nil && idx >= 0 {
		f.SetActiveSheet(idx)
	}

	// Write to buffer (with password protection if supported)
	var buf bytes.Buffer
	// TODO: excelize v2 doesn't support password-protected save natively.
	// Use officecrypto-go or a pluggable encrypt function when available.
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, nil, fmt.Errorf("failed to write Excel file: %w", err)
	}

	fileData := buf.Bytes()
	fileSize := int64(len(fileData))
	checksum := fmt.Sprintf("%x", sha256.Sum256(fileData))

	// Upload to MinIO
	objKey := objectKey(builder.Module(), exportCode)
	if e.minio != nil {
		if err := ensureBucket(ctx, e.minio, e.config.MinioBucket); err != nil {
			return nil, nil, fmt.Errorf("failed to ensure MinIO bucket: %w", err)
		}
		if err := uploadToMinIO(ctx, e.minio, e.config.MinioBucket, objKey, fileData); err != nil {
			return nil, nil, fmt.Errorf("failed to upload to MinIO: %w", err)
		}
	}

	// Create audit log
	if e.auditRepo != nil {
		retentionDays := e.config.RetentionDays
		if retentionDays == 0 {
			retentionDays = 90
		}
		auditLog := &ExportAuditLog{
			ID:             uuid.New(),
			ExportCode:     exportCode,
			UserID:         params.User.ID,
			Module:         builder.Module(),
			ReportType:     builder.ReportType(),
			DateFrom:       params.DateFrom,
			DateTo:         params.DateTo,
			RecordCount:    builder.RecordCount(),
			MinioBucket:    e.config.MinioBucket,
			MinioObjectKey: objKey,
			FileSizeBytes:  fileSize,
			FileChecksum:   checksum,
			ClientIP:       params.ClientIP,
			UserAgent:      params.UserAgent,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().AddDate(0, 0, retentionDays),
		}
		if err := e.auditRepo.Create(ctx, auditLog); err != nil {
			e.logger.Error("failed to create export audit log",
				zap.String("export_code", exportCode),
				zap.Error(err),
			)
		}
	}

	result := &ExportResult{
		ExportCode: exportCode,
		FileSize:   fileSize,
		Checksum:   checksum,
		MinIOKey:   objKey,
	}

	return result, fileData, nil
}

// GetExportFile downloads an export file from MinIO by export code.
// GetExportFile downloads export file as []byte (legacy, for small files).
func (e *Engine) GetExportFile(ctx context.Context, exportCode string) ([]byte, *ExportAuditLog, error) {
	if e.auditRepo == nil {
		return nil, nil, fmt.Errorf("audit repository not configured")
	}

	auditLog, err := e.auditRepo.GetByCode(ctx, exportCode)
	if err != nil {
		return nil, nil, err
	}

	if e.minio == nil {
		return nil, auditLog, fmt.Errorf("MinIO client not configured")
	}

	data, err := downloadFromMinIO(ctx, e.minio, auditLog.MinioBucket, auditLog.MinioObjectKey)
	if err != nil {
		return nil, auditLog, fmt.Errorf("failed to download from MinIO: %w", err)
	}

	return data, auditLog, nil
}

// StreamExportFile returns a streaming reader + metadata for the export file.
// Caller MUST close the returned ReadCloser.
func (e *Engine) StreamExportFile(ctx context.Context, exportCode string) (io.ReadCloser, *minio.ObjectInfo, *ExportAuditLog, error) {
	if e.auditRepo == nil {
		return nil, nil, nil, fmt.Errorf("audit repository not configured")
	}

	auditLog, err := e.auditRepo.GetByCode(ctx, exportCode)
	if err != nil {
		return nil, nil, nil, err
	}

	if e.minio == nil {
		return nil, nil, auditLog, fmt.Errorf("MinIO client not configured")
	}

	obj, err := e.minio.GetObject(ctx, auditLog.MinioBucket, auditLog.MinioObjectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, auditLog, fmt.Errorf("failed to get object from MinIO: %w", err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, auditLog, fmt.Errorf("failed to stat MinIO object: %w", err)
	}

	return obj, &stat, auditLog, nil
}

// ListExports lists export audit logs.
func (e *Engine) ListExports(ctx context.Context, userID *uuid.UUID, limit, offset int) ([]ExportAuditLog, int64, error) {
	if e.auditRepo == nil {
		return nil, 0, fmt.Errorf("audit repository not configured")
	}

	if userID != nil {
		return e.auditRepo.ListByUser(ctx, *userID, limit, offset)
	}
	return e.auditRepo.ListAll(ctx, limit, offset)
}
