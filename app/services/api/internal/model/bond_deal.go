package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/pkg/constants"
)

// BondDeal represents a bond/GTCG transaction matching migration 008.
type BondDeal struct {
	ID                 uuid.UUID
	DealNumber         string
	BondCategory       string // GOVERNMENT, FINANCIAL_INSTITUTION, CERTIFICATE_OF_DEPOSIT
	TradeDate          time.Time
	BranchID           uuid.UUID
	OrderDate          *time.Time
	ValueDate          time.Time
	Direction          string // BUY or SELL
	CounterpartyID     uuid.UUID
	CounterpartyCode   string // denormalized from view
	CounterpartyName   string // denormalized from view
	TransactionType    string // REPO, REVERSE_REPO, OUTRIGHT, OTHER
	TransactionTypeOther *string
	BondCatalogID      *uuid.UUID
	BondCodeManual     *string
	BondCodeDisplay    string // COALESCE(catalog.bond_code, bond_code_manual)
	Issuer             string
	CouponRate         decimal.Decimal
	IssueDate          *time.Time
	MaturityDate       time.Time
	Quantity           int64
	FaceValue          decimal.Decimal
	DiscountRate       decimal.Decimal
	CleanPrice         decimal.Decimal
	SettlementPrice    decimal.Decimal
	TotalValue         decimal.Decimal
	PortfolioType      *string // HTM, AFS, HFT — only when BUY
	PaymentDate        time.Time
	RemainingTenorDays int
	ConfirmationMethod string // EMAIL, REUTERS, OTHER
	ConfirmationOther  *string
	ContractPreparedBy string // INTERNAL or COUNTERPARTY
	Status             string
	Note               *string
	ClonedFromID       *uuid.UUID
	CancelReason       *string
	CancelRequestedBy  *uuid.UUID
	CancelRequestedAt  *time.Time
	CreatedBy          uuid.UUID
	CreatedByName      string // denormalized from view
	CreatedAt          time.Time
	UpdatedAt          time.Time
	UpdatedBy          uuid.UUID
	DeletedAt          *time.Time
	Version            int // not in DB — set to 1 for optimistic locking at service level
}

// BondInventory represents a bond inventory record.
type BondInventory struct {
	ID                uuid.UUID
	BondCatalogID     *uuid.UUID
	BondCode          string
	BondCategory      string
	PortfolioType     string
	AvailableQuantity int64
	AcquisitionDate   *time.Time
	AcquisitionPrice  *decimal.Decimal
	Version           int
	UpdatedAt         time.Time
	UpdatedBy         *uuid.UUID
	// denormalized from view
	CatalogIssuer      *string
	CatalogCouponRate  *decimal.Decimal
	CatalogIssueDate   *time.Time
	CatalogMaturityDate *time.Time
	CatalogFaceValue   *decimal.Decimal
	NominalValue       *decimal.Decimal
	UpdatedByName      *string
}

// CalculateTotalValue calculates the total transaction amount (quantity × settlement_price).
func (d *BondDeal) CalculateTotalValue() decimal.Decimal {
	return d.SettlementPrice.Mul(decimal.NewFromInt(d.Quantity))
}

// CalculateRemainingTenorDays calculates maturity_date − payment_date in days.
func (d *BondDeal) CalculateRemainingTenorDays() int {
	return int(d.MaturityDate.Sub(d.PaymentDate).Hours() / 24)
}

// CanEdit checks if the deal can be edited (only OPEN).
func (d *BondDeal) CanEdit() bool {
	return d.Status == constants.StatusOpen
}

// CanRecall checks if the deal can be recalled from a pending state.
// Bond flow: PENDING_L2_APPROVAL, PENDING_BOOKING, PENDING_CHIEF_ACCOUNTANT
func (d *BondDeal) CanRecall() bool {
	switch d.Status {
	case constants.StatusPendingL2Approval,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant:
		return true
	}
	return false
}

// CanCancel checks if the deal can be cancelled (only COMPLETED for bond).
func (d *BondDeal) CanCancel() bool {
	return d.Status == constants.StatusCompleted
}

// CanClone checks if the deal can be cloned.
func (d *BondDeal) CanClone() bool {
	switch d.Status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting:
		return true
	}
	return false
}

// BondCode returns the effective bond code (catalog or manual).
func (d *BondDeal) BondCode() string {
	if d.BondCodeDisplay != "" {
		return d.BondCodeDisplay
	}
	if d.BondCodeManual != nil {
		return *d.BondCodeManual
	}
	return ""
}

// IsGovernment returns true if the bond category is GOVERNMENT.
func (d *BondDeal) IsGovernment() bool {
	return d.BondCategory == constants.BondCategoryGovernment
}
