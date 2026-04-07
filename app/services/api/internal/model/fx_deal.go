package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/pkg/constants"
)

// FxDeal represents an FX deal domain model.
type FxDeal struct {
	ID                  uuid.UUID
	TicketNumber        *string
	CounterpartyID      uuid.UUID
	CounterpartyCode    string // populated from view
	CounterpartyName    string // populated from view
	DealType            string
	Direction           string
	NotionalAmount      decimal.Decimal
	CurrencyCode        string
	PairCode            string
	BranchID            uuid.UUID
	TradeDate           time.Time
	ExecutionDate       *time.Time
	PayCodeKLB          *string
	PayCodeCounterparty *string
	IsInternational     bool
	AttachmentPath      *string
	AttachmentName      *string
	SettlementAmount    *decimal.Decimal
	SettlementCurrency  *string
	Status              string
	Note                *string
	Legs                []FxDealLeg
	CreatedBy           uuid.UUID
	ApprovedBy          *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Version             int
}

// FxDealLeg represents a single leg of an FX deal.
type FxDealLeg struct {
	ID                  uuid.UUID
	FxDealID            uuid.UUID
	LegNumber           int
	ValueDate           time.Time
	ExecutionDate       *time.Time
	ExchangeRate        decimal.Decimal
	BuyCurrency         string
	SellCurrency        string
	BuyAmount           decimal.Decimal
	SellAmount          decimal.Decimal
	PayCodeKLB          *string
	PayCodeCounterparty *string
	IsInternational     bool
	SettlementAmount    *decimal.Decimal
	SettlementCurrency  *string
}

// CanEdit checks if the deal can be edited. OPEN and PENDING_TP_REVIEW deals can be edited.
func (d *FxDeal) CanEdit() bool {
	return d.Status == constants.StatusOpen || d.Status == constants.StatusPendingTPReview
}

// CanRecall checks if the deal can be recalled. Only deals pending L2 approval can be.
func (d *FxDeal) CanRecall() bool {
	// BRD: Recall allowed from any pending approval/booking state
	// CV/TP can recall while deal is awaiting next approval step
	switch d.Status {
	case constants.StatusPendingL2Approval,
		constants.StatusPendingTPReview,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant,
		constants.StatusPendingRiskApproval,
		constants.StatusPendingSettlement:
		return true
	default:
		return false
	}
}

// CanCancel checks if a completed or pending settlement deal can be requested for cancellation.
func (d *FxDeal) CanCancel() bool {
	return d.Status == constants.StatusCompleted || d.Status == constants.StatusPendingSettlement
}

// CanClone checks if a rejected or voided deal can be cloned.
func (d *FxDeal) CanClone() bool {
	switch d.Status {
	case constants.StatusRejected,
		constants.StatusVoidedByAccounting,
		constants.StatusVoidedByRisk,
		constants.StatusVoidedBySettlement:
		return true
	default:
		return false
	}
}

// CalculateConvertedAmount calculates the equivalent amount based on rate and rule.
// Rule can be "MULTIPLY" or "DIVIDE".
func (d *FxDeal) CalculateConvertedAmount(rate decimal.Decimal, rule string) (decimal.Decimal, error) {
	if rate.IsNegative() || rate.IsZero() {
		return decimal.Zero, fmt.Errorf("exchange rate must be positive")
	}
	switch rule {
	case "MULTIPLY":
		return d.NotionalAmount.Mul(rate), nil
	case "DIVIDE":
		return d.NotionalAmount.Div(rate), nil
	default:
		return decimal.Zero, fmt.Errorf("invalid conversion rule: %s", rule)
	}
}
