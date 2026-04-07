package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// FxDealLegDTO represents a single leg of an FX deal.
type FxDealLegDTO struct {
	LegNumber           int              `json:"leg_number" validate:"required,min=1,max=2"`
	ValueDate           time.Time        `json:"value_date" validate:"required"`
	ExecutionDate       *time.Time       `json:"execution_date,omitempty"`
	ExchangeRate        decimal.Decimal  `json:"exchange_rate" validate:"required,gt=0"`
	BuyCurrency         string           `json:"buy_currency" validate:"required,len=3"`
	SellCurrency        string           `json:"sell_currency" validate:"required,len=3"`
	BuyAmount           decimal.Decimal  `json:"buy_amount" validate:"required,gt=0"`
	SellAmount          decimal.Decimal  `json:"sell_amount" validate:"required,gt=0"`
	PayCodeKLB          *string          `json:"pay_code_klb,omitempty"`
	PayCodeCounterparty *string          `json:"pay_code_counterparty,omitempty"`
	IsInternational     *bool            `json:"is_international,omitempty"`
	SettlementAmount    *decimal.Decimal `json:"settlement_amount,omitempty"`
	SettlementCurrency  *string          `json:"settlement_currency,omitempty"`
}

// CreateFxDealRequest is the payload for creating a new FX deal.
type CreateFxDealRequest struct {
	TicketNumber        *string         `json:"ticket_number" validate:"omitempty,max=20"`
	CounterpartyID      uuid.UUID       `json:"counterparty_id" validate:"required"`
	DealType            string          `json:"deal_type" validate:"required,oneof=SPOT FORWARD SWAP"`
	Direction           string          `json:"direction" validate:"required,oneof=SELL BUY SELL_BUY BUY_SELL"`
	NotionalAmount      decimal.Decimal `json:"notional_amount" validate:"required,gt=0"`
	CurrencyCode        string          `json:"currency_code" validate:"required,len=3"`
	TradeDate           time.Time       `json:"trade_date" validate:"required"`
	ExecutionDate       *time.Time      `json:"execution_date,omitempty"`
	PayCodeKLB          *string         `json:"pay_code_klb,omitempty"`
	PayCodeCounterparty *string         `json:"pay_code_counterparty,omitempty"`
	AttachmentPath      *string         `json:"attachment_path,omitempty"`
	AttachmentName      *string         `json:"attachment_name,omitempty"`
	Note                *string         `json:"note" validate:"omitempty,max=2000"`
	Legs                []FxDealLegDTO  `json:"legs" validate:"required,min=1,max=2,dive"`
}

// UpdateFxDealRequest is the payload for updating an existing FX deal.
type UpdateFxDealRequest struct {
	TicketNumber        *string          `json:"ticket_number" validate:"omitempty,max=20"`
	CounterpartyID      *uuid.UUID       `json:"counterparty_id" validate:"omitempty"`
	DealType            *string          `json:"deal_type" validate:"omitempty,oneof=SPOT FORWARD SWAP"`
	Direction           *string          `json:"direction" validate:"omitempty,oneof=SELL BUY SELL_BUY BUY_SELL"`
	NotionalAmount      *decimal.Decimal `json:"notional_amount" validate:"omitempty,gt=0"`
	CurrencyCode        *string          `json:"currency_code" validate:"omitempty,len=3"`
	TradeDate           *time.Time       `json:"trade_date" validate:"omitempty"`
	ExecutionDate       *time.Time       `json:"execution_date,omitempty"`
	PayCodeKLB          *string          `json:"pay_code_klb,omitempty"`
	PayCodeCounterparty *string          `json:"pay_code_counterparty,omitempty"`
	AttachmentPath      *string          `json:"attachment_path,omitempty"`
	AttachmentName      *string          `json:"attachment_name,omitempty"`
	Note                *string          `json:"note" validate:"omitempty,max=2000"`
	Legs                []FxDealLegDTO   `json:"legs" validate:"omitempty,min=1,max=2,dive"`
	Version             int              `json:"version" validate:"required,min=1"` // optimistic locking
}

// ApprovalHistoryEntry represents a single entry in the deal's approval history.
type ApprovalHistoryEntry struct {
	ID            uuid.UUID `json:"id"`
	ActionType    string    `json:"action_type"`
	StatusBefore  string    `json:"status_before"`
	StatusAfter   string    `json:"status_after"`
	PerformedBy   uuid.UUID `json:"performed_by"`
	PerformerName string    `json:"performer_name"`
	PerformedAt   time.Time `json:"performed_at"`
	Reason        string    `json:"reason"`
}

// ApprovalRequest is the payload for approve/reject actions.
// (Already defined in auth_dto.go but duplicated here for FX cancel flow reference.)

// LimitCheckInfo contains credit limit check results returned with deal responses.
type LimitCheckInfo struct {
	TotalLimit     string `json:"total_limit"`
	UsedAmount     string `json:"used_amount"`
	AvailableLimit string `json:"available_limit"`
	Escalated      bool   `json:"escalated"`
}

// FxDealResponse is the response for an FX deal.
type FxDealResponse struct {
	ID                  uuid.UUID        `json:"id"`
	TicketNumber        *string          `json:"ticket_number,omitempty"`
	CounterpartyID      uuid.UUID        `json:"counterparty_id"`
	CounterpartyCode    string           `json:"counterparty_code,omitempty"`
	CounterpartyName    string           `json:"counterparty_name,omitempty"`
	DealType            string           `json:"deal_type"`
	Direction           string           `json:"direction"`
	NotionalAmount      decimal.Decimal  `json:"notional_amount"`
	CurrencyCode        string           `json:"currency_code"`
	TradeDate           time.Time        `json:"trade_date"`
	ExecutionDate       *time.Time       `json:"execution_date,omitempty"`
	PayCodeKLB          *string          `json:"pay_code_klb,omitempty"`
	PayCodeCounterparty *string          `json:"pay_code_counterparty,omitempty"`
	IsInternational     bool             `json:"is_international"`
	AttachmentPath      *string              `json:"attachment_path,omitempty"`
	AttachmentName      *string              `json:"attachment_name,omitempty"`
	Attachments         []AttachmentResponse `json:"attachments,omitempty"`
	AttachmentCount     *int                 `json:"attachment_count,omitempty"`
	SettlementAmount    *decimal.Decimal     `json:"settlement_amount,omitempty"`
	SettlementCurrency  *string              `json:"settlement_currency,omitempty"`
	Status              string               `json:"status"`
	Note                *string              `json:"note,omitempty"`
	Legs                []FxDealLegDTO       `json:"legs"`
	CreatedBy           uuid.UUID            `json:"created_by"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	Version             int                  `json:"version"`
	LimitCheck          *LimitCheckInfo      `json:"limit_check,omitempty"`
}
