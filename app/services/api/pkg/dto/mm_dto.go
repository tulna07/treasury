package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── MM Interbank DTOs ───

// CreateMMInterbankRequest is the payload for creating a MM interbank deal.
type CreateMMInterbankRequest struct {
	TicketNumber                    *string         `json:"ticket_number" validate:"omitempty,max=20"`
	CounterpartyID                  uuid.UUID       `json:"counterparty_id" validate:"required"`
	CurrencyCode                    string          `json:"currency_code" validate:"required,len=3"`
	InternalSSIID                   *uuid.UUID      `json:"internal_ssi_id"`
	CounterpartySSIID               *uuid.UUID      `json:"counterparty_ssi_id"`
	CounterpartySSIText             *string         `json:"counterparty_ssi_text" validate:"omitempty,max=2000"`
	Direction                       string          `json:"direction" validate:"required,oneof=PLACE TAKE LEND BORROW"`
	PrincipalAmount                 decimal.Decimal `json:"principal_amount" validate:"required,gt=0"`
	InterestRate                    decimal.Decimal `json:"interest_rate" validate:"required,gt=0"`
	DayCountConvention              string          `json:"day_count_convention" validate:"required,oneof=ACT_365 ACT_360 ACT_ACT"`
	TradeDate                       time.Time       `json:"trade_date" validate:"required"`
	EffectiveDate                   time.Time       `json:"effective_date" validate:"required"`
	MaturityDate                    time.Time       `json:"maturity_date" validate:"required"`
	HasCollateral                   bool            `json:"has_collateral"`
	CollateralCurrency              *string         `json:"collateral_currency" validate:"omitempty,len=3"`
	CollateralDescription           *string         `json:"collateral_description" validate:"omitempty,max=2000"`
	RequiresInternationalSettlement bool            `json:"requires_international_settlement"`
	Note                            *string         `json:"note" validate:"omitempty,max=2000"`
}

// UpdateMMInterbankRequest is the payload for updating a MM interbank deal.
type UpdateMMInterbankRequest struct {
	Version                         int              `json:"version" validate:"required,min=1"`
	TicketNumber                    *string          `json:"ticket_number" validate:"omitempty,max=20"`
	CounterpartyID                  *uuid.UUID       `json:"counterparty_id"`
	CurrencyCode                    *string          `json:"currency_code" validate:"omitempty,len=3"`
	InternalSSIID                   *uuid.UUID       `json:"internal_ssi_id"`
	CounterpartySSIID               *uuid.UUID       `json:"counterparty_ssi_id"`
	CounterpartySSIText             *string          `json:"counterparty_ssi_text" validate:"omitempty,max=2000"`
	Direction                       *string          `json:"direction" validate:"omitempty,oneof=PLACE TAKE LEND BORROW"`
	PrincipalAmount                 *decimal.Decimal `json:"principal_amount"`
	InterestRate                    *decimal.Decimal `json:"interest_rate"`
	DayCountConvention              *string          `json:"day_count_convention" validate:"omitempty,oneof=ACT_365 ACT_360 ACT_ACT"`
	TradeDate                       *time.Time       `json:"trade_date"`
	EffectiveDate                   *time.Time       `json:"effective_date"`
	MaturityDate                    *time.Time       `json:"maturity_date"`
	HasCollateral                   *bool            `json:"has_collateral"`
	CollateralCurrency              *string          `json:"collateral_currency" validate:"omitempty,len=3"`
	CollateralDescription           *string          `json:"collateral_description" validate:"omitempty,max=2000"`
	RequiresInternationalSettlement *bool            `json:"requires_international_settlement"`
	Note                            *string          `json:"note" validate:"omitempty,max=2000"`
}

// MMInterbankResponse is the response for a MM interbank deal.
type MMInterbankResponse struct {
	ID                              uuid.UUID       `json:"id"`
	DealNumber                      string          `json:"deal_number"`
	TicketNumber                    *string         `json:"ticket_number,omitempty"`
	CounterpartyID                  uuid.UUID       `json:"counterparty_id"`
	CounterpartyCode                string          `json:"counterparty_code,omitempty"`
	CounterpartyName                string          `json:"counterparty_name,omitempty"`
	BranchCode                      string          `json:"branch_code,omitempty"`
	BranchName                      string          `json:"branch_name,omitempty"`
	CurrencyCode                    string          `json:"currency_code"`
	InternalSSIID                   *uuid.UUID      `json:"internal_ssi_id,omitempty"`
	CounterpartySSIID               *uuid.UUID      `json:"counterparty_ssi_id,omitempty"`
	CounterpartySSIText             *string         `json:"counterparty_ssi_text,omitempty"`
	Direction                       string          `json:"direction"`
	PrincipalAmount                 decimal.Decimal `json:"principal_amount"`
	InterestRate                    decimal.Decimal `json:"interest_rate"`
	DayCountConvention              string          `json:"day_count_convention"`
	TradeDate                       time.Time       `json:"trade_date"`
	EffectiveDate                   time.Time       `json:"effective_date"`
	TenorDays                       int             `json:"tenor_days"`
	MaturityDate                    time.Time       `json:"maturity_date"`
	InterestAmount                  decimal.Decimal `json:"interest_amount"`
	MaturityAmount                  decimal.Decimal `json:"maturity_amount"`
	HasCollateral                   bool            `json:"has_collateral"`
	CollateralCurrency              *string         `json:"collateral_currency,omitempty"`
	CollateralDescription           *string         `json:"collateral_description,omitempty"`
	RequiresInternationalSettlement bool            `json:"requires_international_settlement"`
	Status                          string          `json:"status"`
	Note                            *string         `json:"note,omitempty"`
	ClonedFromID                    *uuid.UUID      `json:"cloned_from_id,omitempty"`
	CancelReason                    *string         `json:"cancel_reason,omitempty"`
	CreatedBy                       uuid.UUID       `json:"created_by"`
	CreatedByName                   string          `json:"created_by_name,omitempty"`
	CreatedAt                       time.Time       `json:"created_at"`
	UpdatedAt                       time.Time       `json:"updated_at"`
	Version                         int             `json:"version"`
}

// MMInterbankFilter holds filter criteria for listing MM interbank deals.
type MMInterbankFilter struct {
	Status          *string
	Statuses        *[]string
	ExcludeStatuses *[]string
	CounterpartyID  *uuid.UUID
	Direction       *string
	CurrencyCode    *string
	FromDate        *string
	ToDate          *string
	CreatedBy       *uuid.UUID
	DealNumber      *string
}

// ──��� MM OMO/Repo DTOs ───

// CreateMMOMORepoRequest is the payload for creating an OMO or Repo KBNN deal.
type CreateMMOMORepoRequest struct {
	DealSubtype     string          `json:"deal_subtype" validate:"required,oneof=OMO STATE_REPO"`
	SessionName     string          `json:"session_name" validate:"required,max=100"`
	TradeDate       time.Time       `json:"trade_date" validate:"required"`
	CounterpartyID  uuid.UUID       `json:"counterparty_id" validate:"required"`
	NotionalAmount  decimal.Decimal `json:"notional_amount" validate:"required,gt=0"`
	BondCatalogID   uuid.UUID       `json:"bond_catalog_id" validate:"required"`
	WinningRate     decimal.Decimal `json:"winning_rate" validate:"required,gt=0"`
	TenorDays       int             `json:"tenor_days" validate:"required,min=1"`
	SettlementDate1 time.Time       `json:"settlement_date_1" validate:"required"`
	SettlementDate2 time.Time       `json:"settlement_date_2" validate:"required"`
	HaircutPct      decimal.Decimal `json:"haircut_pct" validate:"gte=0"`
	Note            *string         `json:"note" validate:"omitempty,max=2000"`
}

// UpdateMMOMORepoRequest is the payload for updating an OMO or Repo KBNN deal.
type UpdateMMOMORepoRequest struct {
	Version         int              `json:"version" validate:"required,min=1"`
	SessionName     *string          `json:"session_name" validate:"omitempty,max=100"`
	TradeDate       *time.Time       `json:"trade_date"`
	CounterpartyID  *uuid.UUID       `json:"counterparty_id"`
	NotionalAmount  *decimal.Decimal `json:"notional_amount"`
	BondCatalogID   *uuid.UUID       `json:"bond_catalog_id"`
	WinningRate     *decimal.Decimal `json:"winning_rate"`
	TenorDays       *int             `json:"tenor_days" validate:"omitempty,min=1"`
	SettlementDate1 *time.Time       `json:"settlement_date_1"`
	SettlementDate2 *time.Time       `json:"settlement_date_2"`
	HaircutPct      *decimal.Decimal `json:"haircut_pct"`
	Note            *string          `json:"note" validate:"omitempty,max=2000"`
}

// MMOMORepoResponse is the response for an OMO or Repo KBNN deal.
type MMOMORepoResponse struct {
	ID               uuid.UUID       `json:"id"`
	DealNumber       string          `json:"deal_number"`
	DealSubtype      string          `json:"deal_subtype"`
	SessionName      string          `json:"session_name"`
	TradeDate        time.Time       `json:"trade_date"`
	CounterpartyID   uuid.UUID       `json:"counterparty_id"`
	CounterpartyCode string          `json:"counterparty_code,omitempty"`
	CounterpartyName string          `json:"counterparty_name,omitempty"`
	BranchCode       string          `json:"branch_code,omitempty"`
	BranchName       string          `json:"branch_name,omitempty"`
	NotionalAmount   decimal.Decimal `json:"notional_amount"`
	BondCatalogID    uuid.UUID       `json:"bond_catalog_id"`
	BondCode         string          `json:"bond_code,omitempty"`
	BondIssuer       string          `json:"bond_issuer,omitempty"`
	BondCouponRate   decimal.Decimal `json:"bond_coupon_rate"`
	BondMaturityDate *time.Time      `json:"bond_maturity_date,omitempty"`
	WinningRate      decimal.Decimal `json:"winning_rate"`
	TenorDays        int             `json:"tenor_days"`
	SettlementDate1  time.Time       `json:"settlement_date_1"`
	SettlementDate2  time.Time       `json:"settlement_date_2"`
	HaircutPct       decimal.Decimal `json:"haircut_pct"`
	Status           string          `json:"status"`
	Note             *string         `json:"note,omitempty"`
	ClonedFromID     *uuid.UUID      `json:"cloned_from_id,omitempty"`
	CancelReason     *string         `json:"cancel_reason,omitempty"`
	CreatedBy        uuid.UUID       `json:"created_by"`
	CreatedByName    string          `json:"created_by_name,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	Version          int             `json:"version"`
}

// MMOMORepoFilter holds filter criteria for listing MM OMO/Repo deals.
type MMOMORepoFilter struct {
	DealSubtype     string // OMO or STATE_REPO — always set by handler
	Status          *string
	Statuses        *[]string
	ExcludeStatuses *[]string
	CounterpartyID  *uuid.UUID
	FromDate        *string
	ToDate          *string
	CreatedBy       *uuid.UUID
	DealNumber      *string
}
