package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateBondDealRequest is the payload for creating a bond (Fixed Income) deal.
type CreateBondDealRequest struct {
	BondCategory         string          `json:"bond_category" validate:"required,oneof=GOVERNMENT FINANCIAL_INSTITUTION CERTIFICATE_OF_DEPOSIT"`
	TradeDate            time.Time       `json:"trade_date" validate:"required"`
	OrderDate            *time.Time      `json:"order_date"`
	ValueDate            time.Time       `json:"value_date" validate:"required"`
	Direction            string          `json:"direction" validate:"required,oneof=BUY SELL"`
	CounterpartyID       uuid.UUID       `json:"counterparty_id" validate:"required"`
	TransactionType      string          `json:"transaction_type" validate:"required,oneof=REPO REVERSE_REPO OUTRIGHT OTHER"`
	TransactionTypeOther *string         `json:"transaction_type_other"`
	BondCatalogID        *uuid.UUID      `json:"bond_catalog_id"`
	BondCodeManual       *string         `json:"bond_code_manual" validate:"omitempty,max=50"`
	Issuer               string          `json:"issuer" validate:"required,max=500"`
	CouponRate           decimal.Decimal `json:"coupon_rate" validate:"gte=0"`
	IssueDate            *time.Time      `json:"issue_date"`
	MaturityDate         time.Time       `json:"maturity_date" validate:"required"`
	Quantity             int64           `json:"quantity" validate:"required,min=1"`
	FaceValue            decimal.Decimal `json:"face_value" validate:"required,gt=0"`
	DiscountRate         decimal.Decimal `json:"discount_rate"`
	CleanPrice           decimal.Decimal `json:"clean_price" validate:"required,gt=0"`
	SettlementPrice      decimal.Decimal `json:"settlement_price" validate:"required,gt=0"`
	TotalValue           decimal.Decimal `json:"total_value" validate:"required,gt=0"`
	PortfolioType        *string         `json:"portfolio_type" validate:"omitempty,oneof=HTM AFS HFT"`
	PaymentDate          time.Time       `json:"payment_date" validate:"required"`
	RemainingTenorDays   int             `json:"remaining_tenor_days"`
	ConfirmationMethod   string          `json:"confirmation_method" validate:"required,oneof=EMAIL REUTERS OTHER"`
	ConfirmationOther    *string         `json:"confirmation_other" validate:"omitempty,max=255"`
	ContractPreparedBy   string          `json:"contract_prepared_by" validate:"required,oneof=INTERNAL COUNTERPARTY"`
	Note                 *string         `json:"note" validate:"omitempty,max=2000"`
}

// UpdateBondDealRequest is the payload for updating a bond deal.
type UpdateBondDealRequest struct {
	Version              int              `json:"version" validate:"required,min=1"`
	BondCategory         *string          `json:"bond_category" validate:"omitempty,oneof=GOVERNMENT FINANCIAL_INSTITUTION CERTIFICATE_OF_DEPOSIT"`
	TradeDate            *time.Time       `json:"trade_date"`
	OrderDate            *time.Time       `json:"order_date"`
	ValueDate            *time.Time       `json:"value_date"`
	Direction            *string          `json:"direction" validate:"omitempty,oneof=BUY SELL"`
	CounterpartyID       *uuid.UUID       `json:"counterparty_id"`
	TransactionType      *string          `json:"transaction_type" validate:"omitempty,oneof=REPO REVERSE_REPO OUTRIGHT OTHER"`
	TransactionTypeOther *string          `json:"transaction_type_other"`
	BondCatalogID        *uuid.UUID       `json:"bond_catalog_id"`
	BondCodeManual       *string          `json:"bond_code_manual" validate:"omitempty,max=50"`
	Issuer               *string          `json:"issuer" validate:"omitempty,max=500"`
	CouponRate           *decimal.Decimal `json:"coupon_rate"`
	IssueDate            *time.Time       `json:"issue_date"`
	MaturityDate         *time.Time       `json:"maturity_date"`
	Quantity             *int64           `json:"quantity" validate:"omitempty,min=1"`
	FaceValue            *decimal.Decimal `json:"face_value"`
	DiscountRate         *decimal.Decimal `json:"discount_rate"`
	CleanPrice           *decimal.Decimal `json:"clean_price"`
	SettlementPrice      *decimal.Decimal `json:"settlement_price"`
	TotalValue           *decimal.Decimal `json:"total_value"`
	PortfolioType        *string          `json:"portfolio_type" validate:"omitempty,oneof=HTM AFS HFT"`
	PaymentDate          *time.Time       `json:"payment_date"`
	RemainingTenorDays   *int             `json:"remaining_tenor_days"`
	ConfirmationMethod   *string          `json:"confirmation_method" validate:"omitempty,oneof=EMAIL REUTERS OTHER"`
	ConfirmationOther    *string          `json:"confirmation_other" validate:"omitempty,max=255"`
	ContractPreparedBy   *string          `json:"contract_prepared_by" validate:"omitempty,oneof=INTERNAL COUNTERPARTY"`
	Note                 *string          `json:"note" validate:"omitempty,max=2000"`
}

// BondDealResponse is the response for a bond deal.
type BondDealResponse struct {
	ID                   uuid.UUID       `json:"id"`
	DealNumber           string          `json:"deal_number"`
	BondCategory         string          `json:"bond_category"`
	TradeDate            time.Time       `json:"trade_date"`
	OrderDate            *time.Time      `json:"order_date,omitempty"`
	ValueDate            time.Time       `json:"value_date"`
	Direction            string          `json:"direction"`
	CounterpartyID       uuid.UUID       `json:"counterparty_id"`
	CounterpartyCode     string          `json:"counterparty_code,omitempty"`
	CounterpartyName     string          `json:"counterparty_name,omitempty"`
	TransactionType      string          `json:"transaction_type"`
	TransactionTypeOther *string         `json:"transaction_type_other,omitempty"`
	BondCatalogID        *uuid.UUID      `json:"bond_catalog_id,omitempty"`
	BondCodeManual       *string         `json:"bond_code_manual,omitempty"`
	BondCodeDisplay      string          `json:"bond_code_display"`
	Issuer               string          `json:"issuer"`
	CouponRate           decimal.Decimal `json:"coupon_rate"`
	IssueDate            *time.Time      `json:"issue_date,omitempty"`
	MaturityDate         time.Time       `json:"maturity_date"`
	Quantity             int64           `json:"quantity"`
	FaceValue            decimal.Decimal `json:"face_value"`
	DiscountRate         decimal.Decimal `json:"discount_rate"`
	CleanPrice           decimal.Decimal `json:"clean_price"`
	SettlementPrice      decimal.Decimal `json:"settlement_price"`
	TotalValue           decimal.Decimal `json:"total_value"`
	PortfolioType        *string         `json:"portfolio_type,omitempty"`
	PaymentDate          time.Time       `json:"payment_date"`
	RemainingTenorDays   int             `json:"remaining_tenor_days"`
	ConfirmationMethod   string          `json:"confirmation_method"`
	ConfirmationOther    *string         `json:"confirmation_other,omitempty"`
	ContractPreparedBy   string          `json:"contract_prepared_by"`
	Status               string          `json:"status"`
	Note                 *string         `json:"note,omitempty"`
	ClonedFromID         *uuid.UUID      `json:"cloned_from_id,omitempty"`
	CancelReason         *string         `json:"cancel_reason,omitempty"`
	CreatedBy            uuid.UUID       `json:"created_by"`
	CreatedByName        string          `json:"created_by_name,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	Version              int             `json:"version"`
}

// BondDealListFilter holds filter criteria for listing Bond deals.
type BondDealListFilter struct {
	Status          *string
	Statuses        *[]string
	ExcludeStatuses *[]string
	CounterpartyID  *uuid.UUID
	BondCategory    *string
	Direction       *string
	FromDate        *string
	ToDate          *string
	CreatedBy       *uuid.UUID
	DealNumber      *string
}

// BondInventoryResponse is the response for a bond inventory record.
type BondInventoryResponse struct {
	ID                uuid.UUID        `json:"id"`
	BondCode          string           `json:"bond_code"`
	BondCategory      string           `json:"bond_category"`
	PortfolioType     string           `json:"portfolio_type"`
	AvailableQuantity int64            `json:"available_quantity"`
	AcquisitionDate   *time.Time       `json:"acquisition_date,omitempty"`
	AcquisitionPrice  *decimal.Decimal `json:"acquisition_price,omitempty"`
	Version           int              `json:"version"`
	UpdatedAt         time.Time        `json:"updated_at"`
	CatalogIssuer     *string          `json:"catalog_issuer,omitempty"`
	CatalogFaceValue  *decimal.Decimal `json:"catalog_face_value,omitempty"`
	NominalValue      *decimal.Decimal `json:"nominal_value,omitempty"`
	UpdatedByName     *string          `json:"updated_by_name,omitempty"`
}
