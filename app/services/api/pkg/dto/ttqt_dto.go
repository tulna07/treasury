package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// InternationalPaymentResponse is the API response for a single international payment.
type InternationalPaymentResponse struct {
	ID                 uuid.UUID       `json:"id"`
	SourceModule       string          `json:"source_module"`
	SourceDealID       uuid.UUID       `json:"source_deal_id"`
	SourceLegNumber    *int16          `json:"source_leg_number,omitempty"`
	TicketDisplay      string          `json:"ticket_display"`
	CounterpartyID     uuid.UUID       `json:"counterparty_id"`
	CounterpartyCode   string          `json:"counterparty_code"`
	CounterpartyName   string          `json:"counterparty_name"`
	DebitAccount       string          `json:"debit_account"`
	BICCode            *string         `json:"bic_code,omitempty"`
	CurrencyCode       string          `json:"currency_code"`
	Amount             decimal.Decimal `json:"amount"`
	TransferDate       string          `json:"transfer_date"`
	CounterpartySSI    string          `json:"counterparty_ssi"`
	OriginalTradeDate  string          `json:"original_trade_date"`
	ApprovedByDivision *string         `json:"approved_by_division,omitempty"`
	SettlementStatus   string          `json:"settlement_status"`
	SettledBy          *uuid.UUID      `json:"settled_by,omitempty"`
	SettledByName      *string         `json:"settled_by_name,omitempty"`
	SettledAt          *time.Time      `json:"settled_at,omitempty"`
	RejectionReason    *string         `json:"rejection_reason,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// SettlementApprovalRequest is the request body for approving or rejecting a payment.
type SettlementApprovalRequest struct {
	Reason string `json:"reason"`
}

// InternationalPaymentFilter holds query filters for listing international payments.
type InternationalPaymentFilter struct {
	SettlementStatus *string
	SourceModule     *string
	TransferDateFrom *string
	TransferDateTo   *string
	CounterpartyID   *uuid.UUID
	TicketDisplay    *string
}
