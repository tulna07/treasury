package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InternationalPayment represents a TTQT (international settlement) payment record.
type InternationalPayment struct {
	ID                 uuid.UUID
	SourceModule       string // FX or MM
	SourceDealID       uuid.UUID
	SourceLegNumber    *int16
	TicketDisplay      string
	CounterpartyID     uuid.UUID
	CounterpartyCode   string // denormalized from view
	CounterpartyName   string // denormalized from view
	DebitAccount       string
	BICCode            *string
	CurrencyCode       string
	Amount             decimal.Decimal
	TransferDate       time.Time
	CounterpartySSI    string
	OriginalTradeDate  time.Time
	ApprovedByDivision *string
	SettlementStatus   string // PENDING, APPROVED, REJECTED
	SettledBy          *uuid.UUID
	SettledByName      *string // denormalized from view
	SettledAt          *time.Time
	RejectionReason    *string
	CreatedAt          time.Time
}
