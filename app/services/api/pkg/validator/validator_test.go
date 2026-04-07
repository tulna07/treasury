package validator

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCurrencyValidator(t *testing.T) {
	v := New()

	type testStruct struct {
		Code string `validate:"currency"`
	}

	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{"valid USD", "USD", true},
		{"valid VND", "VND", true},
		{"valid JPY", "JPY", true},
		{"lowercase", "usd", false},
		{"too short", "US", false},
		{"too long", "USDD", false},
		{"with numbers", "US1", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(testStruct{Code: tt.code})
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid for %q", tt.code)
			}
		})
	}
}

func TestDealTypeValidator(t *testing.T) {
	v := New()

	type testStruct struct {
		DealType string `validate:"deal_type"`
	}

	tests := []struct {
		name  string
		dt    string
		valid bool
	}{
		{"SPOT", "SPOT", true},
		{"FORWARD", "FORWARD", true},
		{"SWAP", "SWAP", true},
		{"DEPOSIT", "DEPOSIT", true},
		{"GOVERNMENT_BOND", "GOVERNMENT_BOND", true},
		{"invalid", "INVALID_TYPE", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(testStruct{DealType: tt.dt})
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid for %q", tt.dt)
			}
		})
	}
}

func TestAmountVNDValidator(t *testing.T) {
	v := New()

	type testStruct struct {
		Amount decimal.Decimal `validate:"amount_vnd"`
	}

	tests := []struct {
		name  string
		amt   string
		valid bool
	}{
		{"whole number", "1000000", true},
		{"zero", "0", true},
		{"with decimals", "1000000.50", false},
		{"small decimal", "100.01", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amt, _ := decimal.NewFromString(tt.amt)
			err := v.Struct(testStruct{Amount: amt})
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid for %s", tt.amt)
			}
		})
	}
}

func TestAmountFCYValidator(t *testing.T) {
	v := New()

	type testStruct struct {
		Amount decimal.Decimal `validate:"amount_fcy"`
	}

	tests := []struct {
		name  string
		amt   string
		valid bool
	}{
		{"whole number", "1000", true},
		{"2 decimals", "1000.50", true},
		{"1 decimal", "1000.5", true},
		{"3 decimals", "1000.123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amt, _ := decimal.NewFromString(tt.amt)
			err := v.Struct(testStruct{Amount: amt})
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("expected invalid for %s", tt.amt)
			}
		})
	}
}

func TestValidateStruct(t *testing.T) {
	v := New()

	type testStruct struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required"`
	}

	errs := ValidateStruct(v, testStruct{Name: "ab", Email: ""})
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}

	errs = ValidateStruct(v, testStruct{Name: "John", Email: "john@test.com"})
	if errs != nil {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestIsZeroDecimalCurrency(t *testing.T) {
	if !IsZeroDecimalCurrency("VND") {
		t.Error("VND should be zero decimal")
	}
	if !IsZeroDecimalCurrency("JPY") {
		t.Error("JPY should be zero decimal")
	}
	if IsZeroDecimalCurrency("USD") {
		t.Error("USD should not be zero decimal")
	}
}
