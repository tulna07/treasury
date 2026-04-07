package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Counterparty represents a counterparty entity.
type Counterparty struct {
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

// Currency represents a currency.
type Currency struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	NumericCode   *int      `json:"numeric_code,omitempty"`
	Name          string    `json:"name"`
	DecimalPlaces int       `json:"decimal_places"`
	IsActive      bool      `json:"is_active"`
}

// CurrencyPair represents a currency pair.
type CurrencyPair struct {
	ID                uuid.UUID `json:"id"`
	BaseCurrency      string    `json:"base_currency"`
	QuoteCurrency     string    `json:"quote_currency"`
	PairCode          string    `json:"pair_code"`
	RateDecimalPlaces int       `json:"rate_decimal_places"`
	CalculationRule   string    `json:"calculation_rule"`
	ResultCurrency    string    `json:"result_currency"`
	IsActive          bool      `json:"is_active"`
}

// Branch represents an organizational branch.
type Branch struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	BranchType      string     `json:"branch_type"`
	ParentBranchID  *uuid.UUID `json:"parent_branch_id,omitempty"`
	FlexcubeBranch  *string    `json:"flexcube_branch_code,omitempty"`
	SwiftBranchCode *string    `json:"swift_branch_code,omitempty"`
	Address         *string    `json:"address,omitempty"`
	IsActive        bool       `json:"is_active"`
}

// ExchangeRate represents a daily exchange rate.
type ExchangeRate struct {
	ID               uuid.UUID       `json:"id"`
	CurrencyCode     string          `json:"currency_code"`
	EffectiveDate    time.Time       `json:"effective_date"`
	BuyTransferRate  decimal.Decimal `json:"buy_transfer_rate"`
	SellTransferRate decimal.Decimal `json:"sell_transfer_rate"`
	MidRate          decimal.Decimal `json:"mid_rate"`
	Source           string          `json:"source"`
	CreatedAt        time.Time       `json:"created_at"`
}
