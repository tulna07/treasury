package fx

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/export"
)

// mockFxDealRepo implements repository.FxDealRepository for handler tests.
type mockFxDealRepo struct {
	deals []model.FxDeal
}

func (m *mockFxDealRepo) Create(_ context.Context, _ *model.FxDeal) error     { return nil }
func (m *mockFxDealRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.FxDeal, error) {
	return nil, nil
}
func (m *mockFxDealRepo) List(_ context.Context, _ repository.FxDealFilter, _ dto.PaginationRequest) ([]model.FxDeal, int64, error) {
	return m.deals, int64(len(m.deals)), nil
}
func (m *mockFxDealRepo) Update(_ context.Context, _ *model.FxDeal) error     { return nil }
func (m *mockFxDealRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string, _ uuid.UUID) error {
	return nil
}
func (m *mockFxDealRepo) SoftDelete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil }
func (m *mockFxDealRepo) SumOutstandingByCounterparty(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// mockUserRepo implements repository.UserRepository for handler tests.
type mockUserRepo struct {
	user *model.User
}

func (m *mockUserRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.User, error) {
	return m.user, nil
}
func (m *mockUserRepo) GetByUsername(_ context.Context, _ string) (*model.User, error) {
	return m.user, nil
}

func newTestExportHandler() *ExportHandler {
	logger, _ := zap.NewDevelopment()

	userRepo := &mockUserRepo{
		user: &model.User{
			ID:       uuid.New(),
			Username: "dealer1",
			FullName: "Nguyễn Văn Dealer",
		},
	}

	service := &Service{
		repo:     &mockFxDealRepo{
			deals: []model.FxDeal{
				{
					ID: uuid.New(), DealType: "SPOT", Direction: "BUY",
					NotionalAmount: decimal.NewFromInt(100000), CurrencyCode: "USD",
					PairCode: "USD/VND", TradeDate: time.Now(), Status: "COMPLETED",
					CreatedBy: uuid.New(), CreatedAt: time.Now(),
					Legs: []model.FxDealLeg{
						{LegNumber: 1, ValueDate: time.Now(), ExchangeRate: decimal.NewFromFloat(24500),
							BuyCurrency: "USD", SellCurrency: "VND"},
					},
				},
			},
		},
		userRepo: userRepo,
		logger:   logger,
	}

	// Mock audit repo for export engine
	auditRepo := &mockExportAuditRepo{logs: make(map[string]*export.ExportAuditLog)}
	cfg := export.ExportConfig{MinioBucket: "test-bucket", RetentionDays: 30}
	engine := export.NewEngineWithClient(nil, auditRepo, cfg, logger)

	return NewExportHandler(service, engine, logger)
}

type mockExportAuditRepo struct {
	logs map[string]*export.ExportAuditLog
}

func (m *mockExportAuditRepo) Create(_ context.Context, log *export.ExportAuditLog) error {
	m.logs[log.ExportCode] = log
	return nil
}
func (m *mockExportAuditRepo) GetByCode(_ context.Context, code string) (*export.ExportAuditLog, error) {
	if l, ok := m.logs[code]; ok {
		return l, nil
	}
	return nil, nil
}
func (m *mockExportAuditRepo) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]export.ExportAuditLog, int64, error) {
	return nil, 0, nil
}
func (m *mockExportAuditRepo) ListAll(_ context.Context, _, _ int) ([]export.ExportAuditLog, int64, error) {
	return nil, 0, nil
}

func TestExportFXDeals_Success(t *testing.T) {
	handler := newTestExportHandler()

	body := dto.ExportRequest{
		From:     "2026-01-01",
		To:       "2026-04-04",
		Password: "Test1234",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/fx/deals/export", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	ctx := ctxutil.WithUserID(req.Context(), uuid.New())
	ctx = ctxutil.WithRoles(ctx, []string{"DEALER"})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ExportDeals(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Header().Get("X-Export-Code"))
	assert.NotEmpty(t, w.Header().Get("X-File-Checksum"))
	assert.Greater(t, w.Body.Len(), 0)
}

func TestExportFXDeals_InvalidDateRange(t *testing.T) {
	handler := newTestExportHandler()

	tests := []struct {
		name     string
		from     string
		to       string
		password string
		wantMsg  string
	}{
		{
			name:     "to before from",
			from:     "2026-04-04",
			to:       "2026-01-01",
			password: "Test1234",
			wantMsg:  "to date must be after from date",
		},
		{
			name:     "invalid from format",
			from:     "04/04/2026",
			to:       "2026-04-04",
			password: "Test1234",
			wantMsg:  "invalid from date format",
		},
		{
			name:     "invalid to format",
			from:     "2026-01-01",
			to:       "not-a-date",
			password: "Test1234",
			wantMsg:  "invalid to date format",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := dto.ExportRequest{From: tc.from, To: tc.to, Password: tc.password}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/fx/deals/export", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := ctxutil.WithUserID(req.Context(), uuid.New())
			ctx = ctxutil.WithRoles(ctx, []string{"DEALER"})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ExportDeals(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			errObj := resp["error"].(map[string]interface{})
			assert.Contains(t, errObj["message"], tc.wantMsg)
		})
	}
}

func TestExportFXDeals_WeakPassword(t *testing.T) {
	handler := newTestExportHandler()

	tests := []struct {
		name     string
		password string
		wantMsg  string
	}{
		{"too short", "Ab1", "at least 8 characters"},
		{"no uppercase", "testtest1", "uppercase letter"},
		{"no lowercase", "TESTTEST1", "lowercase letter"},
		{"no digit", "TestTestTest", "digit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := dto.ExportRequest{From: "2026-01-01", To: "2026-04-04", Password: tc.password}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/fx/deals/export", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			ctx := ctxutil.WithUserID(req.Context(), uuid.New())
			ctx = ctxutil.WithRoles(ctx, []string{"DEALER"})
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.ExportDeals(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			errObj := resp["error"].(map[string]interface{})
			assert.Contains(t, errObj["message"], tc.wantMsg)
		})
	}
}
