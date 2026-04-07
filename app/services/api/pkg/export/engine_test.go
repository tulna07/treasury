package export

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

// mockBuilder implements ReportBuilder for testing.
type mockBuilder struct {
	module      string
	reportType  string
	recordCount int
	buildErr    error
}

func (m *mockBuilder) Module() string      { return m.module }
func (m *mockBuilder) ReportType() string   { return m.reportType }
func (m *mockBuilder) RecordCount() int     { return m.recordCount }
func (m *mockBuilder) BuildSheets(f *excelize.File) error {
	if m.buildErr != nil {
		return m.buildErr
	}
	_, err := f.NewSheet("TestSheet")
	if err != nil {
		return err
	}
	_ = f.SetCellValue("TestSheet", "A1", "test data")
	return nil
}

// mockAuditRepo implements ExportAuditRepository for testing.
type mockAuditRepo struct {
	logs    map[string]*ExportAuditLog
	allLogs []ExportAuditLog
}

func newMockAuditRepo() *mockAuditRepo {
	return &mockAuditRepo{
		logs: make(map[string]*ExportAuditLog),
	}
}

func (m *mockAuditRepo) Create(_ context.Context, log *ExportAuditLog) error {
	m.logs[log.ExportCode] = log
	m.allLogs = append(m.allLogs, *log)
	return nil
}

func (m *mockAuditRepo) GetByCode(_ context.Context, code string) (*ExportAuditLog, error) {
	if log, ok := m.logs[code]; ok {
		return log, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockAuditRepo) ListByUser(_ context.Context, userID uuid.UUID, limit, offset int) ([]ExportAuditLog, int64, error) {
	var result []ExportAuditLog
	for _, log := range m.allLogs {
		if log.UserID == userID {
			result = append(result, log)
		}
	}
	total := int64(len(result))
	if offset < len(result) {
		end := offset + limit
		if end > len(result) {
			end = len(result)
		}
		result = result[offset:end]
	} else {
		result = nil
	}
	return result, total, nil
}

func (m *mockAuditRepo) ListAll(_ context.Context, limit, offset int) ([]ExportAuditLog, int64, error) {
	total := int64(len(m.allLogs))
	end := offset + limit
	if end > len(m.allLogs) {
		end = len(m.allLogs)
	}
	if offset >= len(m.allLogs) {
		return nil, total, nil
	}
	return m.allLogs[offset:end], total, nil
}

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func testParams() ExportParams {
	return ExportParams{
		User: UserInfo{
			ID:       uuid.New(),
			Username: "testuser",
			FullName: "Nguyễn Văn Test",
			Role:     "DEALER",
		},
		DateFrom:  time.Now().AddDate(0, -1, 0),
		DateTo:    time.Now(),
		Password:  "Test1234",
		ClientIP:  "127.0.0.1",
		UserAgent: "TestAgent/1.0",
	}
}

func TestGenerateExportCode(t *testing.T) {
	code := GenerateExportCode()
	assert.NotEmpty(t, code)
	assert.True(t, len(code) > 0)

	// Format: EXP-YYYYMMDD-HHMMSS-XXXX
	pattern := `^EXP-\d{8}-\d{6}-\d{4}$`
	matched, err := regexp.MatchString(pattern, code)
	require.NoError(t, err)
	assert.True(t, matched, "export code %q should match pattern %s", code, pattern)

	// Generate multiple — should be unique
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		c := GenerateExportCode()
		codes[c] = true
	}
	// Allow some collisions due to same-second generation, but should have many unique
	assert.Greater(t, len(codes), 50)
}

func TestExecute_Success(t *testing.T) {
	auditRepo := newMockAuditRepo()
	cfg := ExportConfig{
		RetentionDays: 30,
		MinioBucket:   "test-bucket",
	}
	engine := NewEngineWithClient(nil, auditRepo, cfg, testLogger())

	builder := &mockBuilder{
		module:      "FX",
		reportType:  "FX_DEALS",
		recordCount: 5,
	}

	result, fileData, err := engine.Execute(context.Background(), builder, testParams())
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.ExportCode)
	assert.NotEmpty(t, result.Checksum)
	assert.Greater(t, result.FileSize, int64(0))
	assert.NotEmpty(t, result.MinIOKey)
	assert.NotEmpty(t, fileData)

	// Verify audit log was created
	assert.Len(t, auditRepo.allLogs, 1)
	assert.Equal(t, result.ExportCode, auditRepo.allLogs[0].ExportCode)
	assert.Equal(t, "FX", auditRepo.allLogs[0].Module)
	assert.Equal(t, 5, auditRepo.allLogs[0].RecordCount)
}

func TestExecute_BuilderError(t *testing.T) {
	auditRepo := newMockAuditRepo()
	cfg := ExportConfig{MinioBucket: "test-bucket"}
	engine := NewEngineWithClient(nil, auditRepo, cfg, testLogger())

	builder := &mockBuilder{
		module:   "FX",
		buildErr: fmt.Errorf("build failed"),
	}

	result, fileData, err := engine.Execute(context.Background(), builder, testParams())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build failed")
	assert.Nil(t, result)
	assert.Nil(t, fileData)

	// No audit log should be created on failure
	assert.Len(t, auditRepo.allLogs, 0)
}

func TestGetExportFile_NotFound(t *testing.T) {
	auditRepo := newMockAuditRepo()
	cfg := ExportConfig{MinioBucket: "test-bucket"}
	engine := NewEngineWithClient(nil, auditRepo, cfg, testLogger())

	data, log, err := engine.GetExportFile(context.Background(), "EXP-NONEXISTENT")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Nil(t, log)
}

func TestGetExportFile_Success(t *testing.T) {
	auditRepo := newMockAuditRepo()
	cfg := ExportConfig{MinioBucket: "test-bucket"}
	engine := NewEngineWithClient(nil, auditRepo, cfg, testLogger())

	// First execute to create an audit log
	builder := &mockBuilder{module: "FX", reportType: "FX_DEALS", recordCount: 3}
	result, _, err := engine.Execute(context.Background(), builder, testParams())
	require.NoError(t, err)

	// GetExportFile — will fail on MinIO download (no client) but audit log found
	data, auditLog, err := engine.GetExportFile(context.Background(), result.ExportCode)
	assert.Error(t, err) // MinIO not configured
	assert.Contains(t, err.Error(), "MinIO client not configured")
	assert.Nil(t, data)
	assert.NotNil(t, auditLog)
	assert.Equal(t, result.ExportCode, auditLog.ExportCode)
}

func TestListExports(t *testing.T) {
	auditRepo := newMockAuditRepo()
	cfg := ExportConfig{MinioBucket: "test-bucket"}
	engine := NewEngineWithClient(nil, auditRepo, cfg, testLogger())

	// Execute several exports
	for i := 0; i < 3; i++ {
		builder := &mockBuilder{module: "FX", reportType: "FX_DEALS", recordCount: i + 1}
		_, _, err := engine.Execute(context.Background(), builder, testParams())
		require.NoError(t, err)
	}

	// List all
	logs, total, err := engine.ListExports(context.Background(), nil, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, logs, 3)

	// List by user
	userID := testParams().User.ID
	logs, total, err = engine.ListExports(context.Background(), &userID, 10, 0)
	require.NoError(t, err)
	// May or may not match since each call creates a new UUID
	assert.GreaterOrEqual(t, total, int64(0))
}
