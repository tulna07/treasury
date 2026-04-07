package limitcheck

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// --- mock repos ---

type mockCreditLimitRepo struct {
	limits map[string]*model.CreditLimit // key: counterpartyID+currencyCode
	err    error
}

func (m *mockCreditLimitRepo) GetActiveByCounterparty(_ context.Context, counterpartyID uuid.UUID, currencyCode string) (*model.CreditLimit, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := counterpartyID.String() + ":" + currencyCode
	if limit, ok := m.limits[key]; ok {
		return limit, nil
	}
	return nil, errors.New("not found")
}

// Unused interface methods — stubs
func (m *mockCreditLimitRepo) Create(_ context.Context, _ *model.CreditLimit) error    { return nil }
func (m *mockCreditLimitRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.CreditLimit, error) {
	return nil, nil
}
func (m *mockCreditLimitRepo) GetByCounterparty(_ context.Context, _ uuid.UUID, _ string) (*model.CreditLimit, error) {
	return nil, nil
}
func (m *mockCreditLimitRepo) List(_ context.Context, _ repository.CreditLimitFilter, _ dto.PaginationRequest) ([]model.CreditLimit, int64, error) {
	return nil, 0, nil
}
func (m *mockCreditLimitRepo) Update(_ context.Context, _ *model.CreditLimit) error { return nil }
func (m *mockCreditLimitRepo) UpdateUsedAmount(_ context.Context, _ uuid.UUID, _ decimal.Decimal) error {
	return nil
}

type mockFxDealRepo struct {
	outstanding decimal.Decimal
	err         error
}

func (m *mockFxDealRepo) SumOutstandingByCounterparty(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (decimal.Decimal, error) {
	if m.err != nil {
		return decimal.Zero, m.err
	}
	return m.outstanding, nil
}

// Unused interface methods — stubs
func (m *mockFxDealRepo) Create(_ context.Context, _ *model.FxDeal) error { return nil }
func (m *mockFxDealRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.FxDeal, error) {
	return nil, nil
}
func (m *mockFxDealRepo) List(_ context.Context, _ repository.FxDealFilter, _ dto.PaginationRequest) ([]model.FxDeal, int64, error) {
	return nil, 0, nil
}
func (m *mockFxDealRepo) Update(_ context.Context, _ *model.FxDeal) error                    { return nil }
func (m *mockFxDealRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _, _ string, _ uuid.UUID) error { return nil }
func (m *mockFxDealRepo) SoftDelete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error               { return nil }

// --- tests ---

func TestCheckFXDeal(t *testing.T) {
	ctx := context.Background()
	cpID := uuid.New()
	logger := zap.NewNop()

	tests := []struct {
		name               string
		amount             decimal.Decimal
		limitAmount        decimal.Decimal
		outstanding        decimal.Decimal
		noLimit            bool
		excludeDealID      *uuid.UUID
		wantAllowed        bool
		wantEscalation     bool
		wantOverLimit      decimal.Decimal
	}{
		{
			name:           "within limit",
			amount:         decimal.NewFromInt(50000),
			limitAmount:    decimal.NewFromInt(1000000),
			outstanding:    decimal.NewFromInt(200000),
			wantAllowed:    true,
			wantEscalation: false,
		},
		{
			name:           "exact limit",
			amount:         decimal.NewFromInt(800000),
			limitAmount:    decimal.NewFromInt(1000000),
			outstanding:    decimal.NewFromInt(200000),
			wantAllowed:    true,
			wantEscalation: false,
		},
		{
			name:           "over limit",
			amount:         decimal.NewFromInt(900000),
			limitAmount:    decimal.NewFromInt(1000000),
			outstanding:    decimal.NewFromInt(200000),
			wantAllowed:    false,
			wantEscalation: true,
			wantOverLimit:  decimal.NewFromInt(100000),
		},
		{
			name:           "no limit defined",
			amount:         decimal.NewFromInt(999999999),
			noLimit:        true,
			wantAllowed:    true,
			wantEscalation: false,
		},
		{
			name:           "exclude deal on update",
			amount:         decimal.NewFromInt(800000),
			limitAmount:    decimal.NewFromInt(1000000),
			outstanding:    decimal.NewFromInt(200000),
			excludeDealID:  uuidPtr(uuid.New()),
			wantAllowed:    true,
			wantEscalation: false,
		},
		{
			name:           "zero outstanding",
			amount:         decimal.NewFromInt(500000),
			limitAmount:    decimal.NewFromInt(1000000),
			outstanding:    decimal.Zero,
			wantAllowed:    true,
			wantEscalation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limitRepo := &mockCreditLimitRepo{
				limits: make(map[string]*model.CreditLimit),
			}
			if tt.noLimit {
				limitRepo.err = errors.New("not found")
			} else {
				limitRepo.limits[cpID.String()+":USD"] = &model.CreditLimit{
					ID:             uuid.New(),
					CounterpartyID: cpID,
					CurrencyCode:   "USD",
					ApprovedAmount: tt.limitAmount,
					Status:         "ACTIVE",
				}
			}

			fxRepo := &mockFxDealRepo{outstanding: tt.outstanding}

			checker := NewChecker(limitRepo, fxRepo, logger)
			result, err := checker.CheckFXDeal(ctx, cpID, "USD", tt.amount, tt.excludeDealID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
			if result.RequiresEscalation != tt.wantEscalation {
				t.Errorf("RequiresEscalation = %v, want %v", result.RequiresEscalation, tt.wantEscalation)
			}
			if !tt.wantOverLimit.IsZero() && !result.OverLimitAmount.Equal(tt.wantOverLimit) {
				t.Errorf("OverLimitAmount = %s, want %s", result.OverLimitAmount.String(), tt.wantOverLimit.String())
			}
		})
	}
}

func TestCheckFXDeal_MultipleCurrencies(t *testing.T) {
	ctx := context.Background()
	cpID := uuid.New()
	logger := zap.NewNop()

	limitRepo := &mockCreditLimitRepo{
		limits: map[string]*model.CreditLimit{
			cpID.String() + ":USD": {
				ID:             uuid.New(),
				CounterpartyID: cpID,
				CurrencyCode:   "USD",
				ApprovedAmount: decimal.NewFromInt(1000000),
				Status:         "ACTIVE",
			},
			cpID.String() + ":EUR": {
				ID:             uuid.New(),
				CounterpartyID: cpID,
				CurrencyCode:   "EUR",
				ApprovedAmount: decimal.NewFromInt(500000),
				Status:         "ACTIVE",
			},
		},
	}

	fxRepo := &mockFxDealRepo{outstanding: decimal.NewFromInt(100000)}
	checker := NewChecker(limitRepo, fxRepo, logger)

	// USD deal within limit
	result, err := checker.CheckFXDeal(ctx, cpID, "USD", decimal.NewFromInt(800000), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("USD deal should be within limit")
	}

	// EUR deal over limit
	result, err = checker.CheckFXDeal(ctx, cpID, "EUR", decimal.NewFromInt(500000), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("EUR deal should exceed limit")
	}
	if !result.RequiresEscalation {
		t.Error("EUR deal should require escalation")
	}
}

func TestCheckFXDeal_ALLCurrencyFallback(t *testing.T) {
	ctx := context.Background()
	cpID := uuid.New()
	logger := zap.NewNop()

	// No specific JPY limit, but has "ALL" currency limit
	limitRepo := &mockCreditLimitRepo{
		limits: map[string]*model.CreditLimit{
			cpID.String() + ":ALL": {
				ID:             uuid.New(),
				CounterpartyID: cpID,
				CurrencyCode:   "ALL",
				ApprovedAmount: decimal.NewFromInt(2000000),
				Status:         "ACTIVE",
			},
		},
	}

	fxRepo := &mockFxDealRepo{outstanding: decimal.NewFromInt(500000)}
	checker := NewChecker(limitRepo, fxRepo, logger)

	result, err := checker.CheckFXDeal(ctx, cpID, "JPY", decimal.NewFromInt(1000000), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("JPY deal should be within ALL currency limit")
	}
	if !result.TotalLimit.Equal(decimal.NewFromInt(2000000)) {
		t.Errorf("TotalLimit = %s, want 2000000", result.TotalLimit.String())
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
