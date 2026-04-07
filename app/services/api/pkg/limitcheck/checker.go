// Package limitcheck provides credit limit validation for deal operations.
package limitcheck

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/repository"
)

// LimitCheckResult contains the outcome of a credit limit check.
type LimitCheckResult struct {
	Allowed            bool
	CounterpartyID     uuid.UUID
	CurrencyCode       string
	RequestedAmount    decimal.Decimal
	AvailableLimit     decimal.Decimal
	TotalLimit         decimal.Decimal
	UsedAmount         decimal.Decimal
	OverLimitAmount    decimal.Decimal // positive if over limit
	RequiresEscalation bool           // true if over limit → needs Division Head
}

// Checker validates deal amounts against counterparty credit limits.
type Checker struct {
	limitRepo repository.CreditLimitRepository
	fxRepo    repository.FxDealRepository
	logger    *zap.Logger
}

// NewChecker creates a new limit checker.
func NewChecker(limitRepo repository.CreditLimitRepository, fxRepo repository.FxDealRepository, logger *zap.Logger) *Checker {
	return &Checker{limitRepo: limitRepo, fxRepo: fxRepo, logger: logger}
}

// CheckFXDeal validates a deal against the counterparty's credit limit.
//
// Logic:
//  1. Get active credit limit for counterparty + currency (or "ALL" currency fallback)
//  2. Sum all outstanding FX deals for this counterparty (exclude terminal statuses)
//  3. Calculate: available = total_limit - used
//  4. If amount > available → RequiresEscalation = true
//  5. If no limit defined → Allowed = true (no restriction, log warning)
func (c *Checker) CheckFXDeal(ctx context.Context, counterpartyID uuid.UUID, currencyCode string, amount decimal.Decimal, excludeDealID *uuid.UUID) (*LimitCheckResult, error) {
	result := &LimitCheckResult{
		CounterpartyID:  counterpartyID,
		CurrencyCode:    currencyCode,
		RequestedAmount: amount,
	}

	// 1. Get active credit limit — try specific currency first, then "ALL"
	limit, err := c.limitRepo.GetActiveByCounterparty(ctx, counterpartyID, currencyCode)
	if err != nil {
		// Try "ALL" currency fallback
		limit, err = c.limitRepo.GetActiveByCounterparty(ctx, counterpartyID, "ALL")
		if err != nil {
			// No limit defined → allow (no restriction)
			c.logger.Warn("no credit limit defined for counterparty, allowing deal",
				zap.String("counterparty_id", counterpartyID.String()),
				zap.String("currency_code", currencyCode),
			)
			result.Allowed = true
			return result, nil
		}
	}

	// 2. Sum outstanding deals
	usedAmount, err := c.fxRepo.SumOutstandingByCounterparty(ctx, counterpartyID, excludeDealID)
	if err != nil {
		return nil, err
	}

	// 3. Calculate available limit
	totalLimit := limit.ApprovedAmount
	available := totalLimit.Sub(usedAmount)

	result.TotalLimit = totalLimit
	result.UsedAmount = usedAmount
	result.AvailableLimit = available

	// 4. Check if within limit
	if amount.LessThanOrEqual(available) {
		result.Allowed = true
		return result, nil
	}

	// Over limit
	result.Allowed = false
	result.OverLimitAmount = amount.Sub(available)
	result.RequiresEscalation = true

	c.logger.Warn("deal exceeds credit limit, escalation required",
		zap.String("counterparty_id", counterpartyID.String()),
		zap.String("currency_code", currencyCode),
		zap.String("requested", amount.String()),
		zap.String("available", available.String()),
		zap.String("over_limit", result.OverLimitAmount.String()),
	)

	return result, nil
}
