package decimal

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculateSettlementAmount(t *testing.T) {
	tests := []struct {
		name           string
		notional       decimal.Decimal
		rate           decimal.Decimal
		pair           string
		wantAmount     decimal.Decimal
		wantCurrency   string
		wantErr        bool
	}{
		{
			name:         "USD/VND multiply",
			notional:     decimal.NewFromInt(100000),
			rate:         decimal.NewFromFloat(25950),
			pair:         "USD/VND",
			wantAmount:   decimal.NewFromFloat(2595000000),
			wantCurrency: "VND",
		},
		{
			name:         "USD/EUR divide",
			notional:     decimal.NewFromInt(100000),
			rate:         decimal.NewFromFloat(1.08),
			pair:         "USD/EUR",
			wantAmount:   decimal.NewFromInt(100000).Div(decimal.NewFromFloat(1.08)),
			wantCurrency: "USD",
		},
		{
			name:         "EUR/USD multiply",
			notional:     decimal.NewFromInt(100000),
			rate:         decimal.NewFromFloat(1.08),
			pair:         "EUR/USD",
			wantAmount:   decimal.NewFromFloat(108000),
			wantCurrency: "USD",
		},
		{
			name:         "EUR/GBP cross pair multiply",
			notional:     decimal.NewFromInt(100000),
			rate:         decimal.NewFromFloat(0.86),
			pair:         "EUR/GBP",
			wantAmount:   decimal.NewFromFloat(86000),
			wantCurrency: "GBP",
		},
		{
			name:    "zero rate error",
			notional: decimal.NewFromInt(100000),
			rate:    decimal.Zero,
			pair:    "USD/VND",
			wantErr: true,
		},
		{
			name:    "invalid pair",
			notional: decimal.NewFromInt(100000),
			rate:    decimal.NewFromFloat(25950),
			pair:    "INVALID",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency, err := CalculateSettlementAmount(tt.notional, tt.rate, tt.pair)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !amount.Equal(tt.wantAmount) {
				t.Errorf("amount = %s, want %s", amount, tt.wantAmount)
			}
			if currency != tt.wantCurrency {
				t.Errorf("currency = %s, want %s", currency, tt.wantCurrency)
			}
		})
	}
}

func TestValidateRateDecimals(t *testing.T) {
	tests := []struct {
		name    string
		rate    decimal.Decimal
		pair    string
		wantErr bool
	}{
		{"USD/VND 2 decimals OK", decimal.NewFromFloat(25950.50), "USD/VND", false},
		{"USD/VND 0 decimals OK", decimal.NewFromFloat(25950), "USD/VND", false},
		{"USD/VND 3 decimals fail", decimal.NewFromFloat(25950.123), "USD/VND", true},
		{"USD/JPY 2 decimals OK", decimal.NewFromFloat(155.50), "USD/JPY", false},
		{"USD/JPY 3 decimals fail", decimal.NewFromFloat(155.123), "USD/JPY", true},
		{"USD/KRW 2 decimals OK", decimal.NewFromFloat(1350.00), "USD/KRW", false},
		{"EUR/USD 4 decimals OK", decimal.NewFromFloat(1.0856), "EUR/USD", false},
		{"EUR/USD 5 decimals fail", decimal.NewFromFloat(1.08567), "EUR/USD", true},
		{"EUR/GBP 4 decimals OK", decimal.NewFromFloat(0.8634), "EUR/GBP", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRateDecimals(tt.rate, tt.pair)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
