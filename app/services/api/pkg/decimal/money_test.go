package decimal

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewMoney(t *testing.T) {
	m := NewMoney(decimal.NewFromInt(1000), "usd")
	if m.Currency != "USD" {
		t.Errorf("expected USD, got %s", m.Currency)
	}
	if !m.Amount.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("expected 1000, got %s", m.Amount)
	}
}

func TestMoney_Add(t *testing.T) {
	a := NewMoney(decimal.NewFromInt(100), "USD")
	b := NewMoney(decimal.NewFromFloat(50.25), "USD")

	result, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	expected := decimal.NewFromFloat(150.25)
	if !result.Amount.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, result.Amount)
	}
}

func TestMoney_Add_CurrencyMismatch(t *testing.T) {
	a := NewMoney(decimal.NewFromInt(100), "USD")
	b := NewMoney(decimal.NewFromInt(50), "VND")

	_, err := a.Add(b)
	if err == nil {
		t.Fatal("expected currency mismatch error")
	}
}

func TestMoney_Multiply(t *testing.T) {
	m := NewMoney(decimal.NewFromInt(1000), "USD")
	rate := decimal.NewFromFloat(23500.50)

	result := m.Multiply(rate)
	expected := decimal.NewFromInt(1000).Mul(decimal.NewFromFloat(23500.50))
	if !result.Amount.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, result.Amount)
	}
}

func TestMoney_Divide(t *testing.T) {
	m := NewMoney(decimal.NewFromInt(23500000), "VND")
	rate := decimal.NewFromFloat(23500)

	result, err := m.Divide(rate)
	if err != nil {
		t.Fatalf("Divide failed: %v", err)
	}
	expected := decimal.NewFromInt(1000)
	if !result.Amount.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, result.Amount)
	}
}

func TestMoney_Divide_ByZero(t *testing.T) {
	m := NewMoney(decimal.NewFromInt(1000), "USD")
	_, err := m.Divide(decimal.Zero)
	if err == nil {
		t.Fatal("expected division by zero error")
	}
}

func TestMoney_Round_VND(t *testing.T) {
	m := NewMoney(decimal.NewFromFloat(1000000.75), "VND")
	rounded := m.Round()
	expected := decimal.NewFromInt(1000001)
	if !rounded.Amount.Equal(expected) {
		t.Errorf("VND round: expected %s, got %s", expected, rounded.Amount)
	}
}

func TestMoney_Round_USD(t *testing.T) {
	m := NewMoney(decimal.NewFromFloat(1234.5678), "USD")
	rounded := m.Round()
	expected := decimal.NewFromFloat(1234.57)
	if !rounded.Amount.Equal(expected) {
		t.Errorf("USD round: expected %s, got %s", expected, rounded.Amount)
	}
}

func TestMoney_Round_JPY(t *testing.T) {
	m := NewMoney(decimal.NewFromFloat(1234.5), "JPY")
	rounded := m.Round()
	expected := decimal.NewFromInt(1235)
	if !rounded.Amount.Equal(expected) {
		t.Errorf("JPY round: expected %s, got %s", expected, rounded.Amount)
	}
}

func TestMoney_String(t *testing.T) {
	tests := []struct {
		amount   string
		currency string
		expected string
	}{
		{"1000000", "VND", "1000000 VND"},
		{"1234.56", "USD", "1234.56 USD"},
		{"9876", "JPY", "9876 JPY"},
		{"100.5", "USD", "100.50 USD"},
	}
	for _, tt := range tests {
		amt, _ := decimal.NewFromString(tt.amount)
		m := NewMoney(amt, tt.currency)
		if m.String() != tt.expected {
			t.Errorf("expected %q, got %q", tt.expected, m.String())
		}
	}
}

func TestMoney_IsPositive(t *testing.T) {
	pos := NewMoney(decimal.NewFromInt(100), "USD")
	if !pos.IsPositive() {
		t.Error("expected positive")
	}

	neg := NewMoney(decimal.NewFromInt(-100), "USD")
	if neg.IsPositive() {
		t.Error("expected not positive")
	}

	zero := Zero("USD")
	if zero.IsPositive() {
		t.Error("zero should not be positive")
	}
}

func TestDecimalsForCurrency(t *testing.T) {
	if DecimalsForCurrency("VND") != 0 {
		t.Error("VND should be 0 decimals")
	}
	if DecimalsForCurrency("USD") != 2 {
		t.Error("USD should be 2 decimals")
	}
	if DecimalsForCurrency("XYZ") != DefaultDecimals {
		t.Error("unknown currency should use default")
	}
}
