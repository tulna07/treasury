package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// --- Counterparty DTOs ---

// CreateCounterpartyRequest is the payload for creating a counterparty.
type CreateCounterpartyRequest struct {
	Code        string  `json:"code" validate:"required,min=2,max=20"`
	FullName    string  `json:"full_name" validate:"required,min=1,max=500"`
	ShortName   *string `json:"short_name,omitempty"`
	CIF         string  `json:"cif" validate:"required,min=1,max=50"`
	SwiftCode   *string `json:"swift_code,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
	Address     *string `json:"address,omitempty"`
	FxUsesLimit bool    `json:"fx_uses_limit"`
}

// UpdateCounterpartyRequest is the payload for updating a counterparty.
type UpdateCounterpartyRequest struct {
	FullName    *string `json:"full_name,omitempty"`
	ShortName   *string `json:"short_name,omitempty"`
	SwiftCode   *string `json:"swift_code,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	TaxID       *string `json:"tax_id,omitempty"`
	Address     *string `json:"address,omitempty"`
	FxUsesLimit *bool   `json:"fx_uses_limit,omitempty"`
}

// CounterpartyResponse represents a counterparty.
type CounterpartyResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	FullName    string    `json:"full_name"`
	ShortName   *string   `json:"short_name,omitempty"`
	CIF         string    `json:"cif"`
	SwiftCode   *string   `json:"swift_code,omitempty"`
	CountryCode *string   `json:"country_code,omitempty"`
	TaxID       *string   `json:"tax_id,omitempty"`
	Address     *string   `json:"address,omitempty"`
	FxUsesLimit bool      `json:"fx_uses_limit"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CounterpartyFilter holds filter criteria for listing counterparties.
type CounterpartyFilter struct {
	Search   *string
	IsActive *bool
}

// --- Currency DTOs ---

// CurrencyResponse represents a currency.
type CurrencyResponse struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	NumericCode   *int      `json:"numeric_code,omitempty"`
	Name          string    `json:"name"`
	DecimalPlaces int       `json:"decimal_places"`
	IsActive      bool      `json:"is_active"`
}

// --- Currency Pair DTOs ---

// CurrencyPairResponse represents a currency pair.
type CurrencyPairResponse struct {
	ID                uuid.UUID `json:"id"`
	BaseCurrency      string    `json:"base_currency"`
	QuoteCurrency     string    `json:"quote_currency"`
	PairCode          string    `json:"pair_code"`
	RateDecimalPlaces int       `json:"rate_decimal_places"`
	CalculationRule   string    `json:"calculation_rule"`
	ResultCurrency    string    `json:"result_currency"`
	IsActive          bool      `json:"is_active"`
}

// --- Branch DTOs ---

// BranchResponse represents a branch.
type BranchResponse struct {
	ID               uuid.UUID  `json:"id"`
	Code             string     `json:"code"`
	Name             string     `json:"name"`
	BranchType       string     `json:"branch_type"`
	ParentBranchID   *uuid.UUID `json:"parent_branch_id,omitempty"`
	FlexcubeBranch   *string    `json:"flexcube_branch_code,omitempty"`
	SwiftBranchCode  *string    `json:"swift_branch_code,omitempty"`
	Address          *string    `json:"address,omitempty"`
	IsActive         bool       `json:"is_active"`
}

// --- Exchange Rate DTOs ---

// ExchangeRateResponse represents an exchange rate.
type ExchangeRateResponse struct {
	ID              uuid.UUID       `json:"id"`
	CurrencyCode    string          `json:"currency_code"`
	EffectiveDate   string          `json:"effective_date"`
	BuyTransferRate decimal.Decimal `json:"buy_transfer_rate"`
	SellTransferRate decimal.Decimal `json:"sell_transfer_rate"`
	MidRate         decimal.Decimal `json:"mid_rate"`
	Source          string          `json:"source"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ExchangeRateFilter holds filter criteria for listing exchange rates.
type ExchangeRateFilter struct {
	CurrencyCode *string
	FromDate     *string
	ToDate       *string
}
