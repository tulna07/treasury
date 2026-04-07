// Package decimal provides Money type with currency-aware rounding for banking operations.
package decimal

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// currencyDecimals maps currencies to their decimal places.
var currencyDecimals = map[string]int32{
	"VND": 0,
	"JPY": 0,
	"KRW": 0,
	"USD": 2,
	"EUR": 2,
	"GBP": 2,
	"CHF": 2,
	"AUD": 2,
	"CAD": 2,
	"SGD": 2,
	"HKD": 2,
	"CNY": 2,
	"THB": 2,
}

// DefaultDecimals is the default number of decimal places for unknown currencies.
const DefaultDecimals int32 = 2

// Money represents a monetary amount with a currency code.
type Money struct {
	Amount   decimal.Decimal
	Currency string
}

// NewMoney creates a new Money instance.
func NewMoney(amount decimal.Decimal, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: strings.ToUpper(currency),
	}
}

// Zero returns a zero Money for the given currency.
func Zero(currency string) Money {
	return NewMoney(decimal.Zero, currency)
}

// Add adds two Money values. Both must have the same currency.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: cannot add %s to %s", other.Currency, m.Currency)
	}
	return NewMoney(m.Amount.Add(other.Amount), m.Currency), nil
}

// Sub subtracts other from m. Both must have the same currency.
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: cannot subtract %s from %s", other.Currency, m.Currency)
	}
	return NewMoney(m.Amount.Sub(other.Amount), m.Currency), nil
}

// Multiply multiplies the amount by a rate.
func (m Money) Multiply(rate decimal.Decimal) Money {
	return NewMoney(m.Amount.Mul(rate), m.Currency)
}

// Divide divides the amount by a rate. Returns error if rate is zero.
func (m Money) Divide(rate decimal.Decimal) (Money, error) {
	if rate.IsZero() {
		return Money{}, fmt.Errorf("division by zero")
	}
	return NewMoney(m.Amount.Div(rate), m.Currency), nil
}

// Round rounds the amount to the correct number of decimal places for the currency.
func (m Money) Round() Money {
	decimals := DecimalsForCurrency(m.Currency)
	return NewMoney(m.Amount.Round(decimals), m.Currency)
}

// IsPositive returns true if the amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.Amount.IsPositive()
}

// IsNegative returns true if the amount is less than zero.
func (m Money) IsNegative() bool {
	return m.Amount.IsNegative()
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// String returns a formatted string representation (e.g., "1,000,000 VND" or "1,234.56 USD").
func (m Money) String() string {
	rounded := m.Round()
	decimals := DecimalsForCurrency(m.Currency)
	return rounded.Amount.StringFixed(decimals) + " " + m.Currency
}

// DecimalsForCurrency returns the number of decimal places for a currency.
func DecimalsForCurrency(currency string) int32 {
	if d, ok := currencyDecimals[strings.ToUpper(currency)]; ok {
		return d
	}
	return DefaultDecimals
}
