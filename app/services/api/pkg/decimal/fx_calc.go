package decimal

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// CalculateSettlementAmount computes the settlement amount from notional amount and rate,
// based on the currency pair convention.
//
// Rules:
//   - pair contains "/VND" → multiply, result in VND
//   - pair starts with "USD/" → divide, result in USD
//   - pair ends with "/USD" → multiply, result in USD
//   - cross pair → multiply, result in quote currency (part after /)
func CalculateSettlementAmount(notionalAmount, rate decimal.Decimal, pairCode string) (decimal.Decimal, string, error) {
	if rate.IsZero() || rate.IsNegative() {
		return decimal.Zero, "", fmt.Errorf("exchange rate must be positive")
	}

	parts := strings.SplitN(pairCode, "/", 2)
	if len(parts) != 2 {
		return decimal.Zero, "", fmt.Errorf("invalid pair code: %s", pairCode)
	}
	quote := parts[1]

	if quote == "VND" {
		// USD/VND → multiply, result in VND
		return notionalAmount.Mul(rate), "VND", nil
	}

	if parts[0] == "USD" {
		// USD/XXX (non-VND) → divide, result in USD
		return notionalAmount.Div(rate), "USD", nil
	}

	if quote == "USD" {
		// XXX/USD → multiply, result in USD
		return notionalAmount.Mul(rate), "USD", nil
	}

	// Cross pair → multiply, result in quote currency
	return notionalAmount.Mul(rate), quote, nil
}

// ValidateRateDecimals checks that the exchange rate doesn't exceed the allowed decimal places
// for the given pair.
//
// Rules:
//   - USD/VND, USD/JPY, USD/KRW → max 2 decimal places
//   - All other pairs → max 4 decimal places
func ValidateRateDecimals(rate decimal.Decimal, pairCode string) error {
	maxDecimals := int32(4) // default

	// Pairs with max 2 decimal places
	twoDecimalPairs := map[string]bool{
		"USD/VND": true,
		"USD/JPY": true,
		"USD/KRW": true,
	}

	if twoDecimalPairs[pairCode] {
		maxDecimals = 2
	}

	// Check decimal places by truncating and comparing
	truncated := rate.Truncate(maxDecimals)
	if !rate.Equal(truncated) {
		return fmt.Errorf("exchange rate for %s must have at most %d decimal places", pairCode, maxDecimals)
	}

	return nil
}
