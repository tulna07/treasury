package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─── Credit Limit CRUD ───

// SetCreditLimitRequest is the payload for creating or updating a credit limit.
type SetCreditLimitRequest struct {
	CounterpartyID    uuid.UUID        `json:"counterparty_id" validate:"required"`
	LimitType         string           `json:"limit_type" validate:"required,oneof=COLLATERALIZED UNCOLLATERALIZED"`
	LimitAmount       *decimal.Decimal `json:"limit_amount" validate:"omitempty,gte=0"`
	IsUnlimited       bool             `json:"is_unlimited"`
	EffectiveFrom     time.Time        `json:"effective_from" validate:"required"`
	ExpiryDate        *time.Time       `json:"expiry_date"`
	ApprovalReference *string          `json:"approval_reference" validate:"omitempty,max=500"`
	Note              *string          `json:"note" validate:"omitempty,max=2000"`
}

// CreditLimitResponse is the API response for a single credit limit.
type CreditLimitResponse struct {
	ID                uuid.UUID        `json:"id"`
	CounterpartyID    uuid.UUID        `json:"counterparty_id"`
	CounterpartyName  string           `json:"counterparty_name"`
	CIFCode           string           `json:"cif_code"`
	LimitType         string           `json:"limit_type"`
	LimitAmount       *decimal.Decimal `json:"limit_amount"`
	IsUnlimited       bool             `json:"is_unlimited"`
	EffectiveFrom     string           `json:"effective_from"`
	EffectiveTo       *string          `json:"effective_to,omitempty"`
	IsCurrent         bool             `json:"is_current"`
	ExpiryDate        *string          `json:"expiry_date,omitempty"`
	ApprovalReference *string          `json:"approval_reference,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	CreatedBy         *uuid.UUID       `json:"created_by,omitempty"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// CreditLimitListFilter holds filter criteria for listing credit limits.
type CreditLimitListFilter struct {
	CounterpartyID *uuid.UUID
	LimitType      *string
	IsCurrent      *bool
}

// ─── Utilization ───

// UtilizationBreakdown shows how a limit is being utilized.
type UtilizationBreakdown struct {
	CounterpartyID   uuid.UUID       `json:"counterparty_id"`
	CounterpartyName string          `json:"counterparty_name"`
	LimitType        string          `json:"limit_type"`
	LimitAmount      *decimal.Decimal `json:"limit_amount"`
	IsUnlimited      bool            `json:"is_unlimited"`

	// Breakdown
	MMUtilized   decimal.Decimal `json:"mm_utilized"`
	BondUtilized decimal.Decimal `json:"bond_utilized"`
	FXUtilized   decimal.Decimal `json:"fx_utilized"`
	TotalUtilized decimal.Decimal `json:"total_utilized"`
	Remaining     *decimal.Decimal `json:"remaining"` // nil = unlimited
	FxRateApplied *decimal.Decimal `json:"fx_rate_applied,omitempty"`
}

// ─── Deal Approval (CV QLRR / TPB QLRR) ───

// LimitApprovalRequest is the payload for approving/rejecting a deal's credit limit.
type LimitApprovalRequest struct {
	DealModule string    `json:"deal_module" validate:"required,oneof=FX MM BOND"`
	DealID     uuid.UUID `json:"deal_id" validate:"required"`
	Action     string    `json:"action" validate:"required,oneof=APPROVE REJECT"`
	Comment    *string   `json:"comment" validate:"omitempty,max=2000"`
}

// LimitApprovalResponse is the API response for a limit approval record.
type LimitApprovalResponse struct {
	ID                    uuid.UUID        `json:"id"`
	DealModule            string           `json:"deal_module"`
	DealID                uuid.UUID        `json:"deal_id"`
	CounterpartyID        uuid.UUID        `json:"counterparty_id"`
	CounterpartyName      string           `json:"counterparty_name"`
	LimitType             string           `json:"limit_type"`
	DealAmountVND         decimal.Decimal  `json:"deal_amount_vnd"`
	LimitSnapshot         map[string]interface{} `json:"limit_snapshot"`
	RiskOfficerApprovedBy *uuid.UUID       `json:"risk_officer_approved_by,omitempty"`
	RiskOfficerApprovedAt *time.Time       `json:"risk_officer_approved_at,omitempty"`
	RiskOfficerName       string           `json:"risk_officer_name,omitempty"`
	RiskHeadApprovedBy    *uuid.UUID       `json:"risk_head_approved_by,omitempty"`
	RiskHeadApprovedAt    *time.Time       `json:"risk_head_approved_at,omitempty"`
	RiskHeadName          string           `json:"risk_head_name,omitempty"`
	ApprovalStatus        string           `json:"approval_status"`
	RejectionReason       *string          `json:"rejection_reason,omitempty"`
	CreatedAt             time.Time        `json:"created_at"`
}

// LimitApprovalListFilter holds filter criteria for listing approval records.
type LimitApprovalListFilter struct {
	CounterpartyID *uuid.UUID
	DealModule     *string
	Status         *string
}

// ─── Daily Summary (BRD §3.4.4) ───

// DailySummaryRequest is the query params for the daily summary endpoint.
type DailySummaryRequest struct {
	Date string `json:"date" validate:"required"` // YYYY-MM-DD
}

// DailySummaryRow is a single row in the 11-column daily summary table.
type DailySummaryRow struct {
	CounterpartyID   uuid.UUID `json:"counterparty_id"`
	CounterpartyName string    `json:"counterparty_name"`
	CIFCode          string    `json:"cif_code"`

	// Cột 1: Hạn mức cấp có TSBĐ
	AllocatedCollateralized   *decimal.Decimal `json:"allocated_collateralized"`
	IsUnlimitedCollateralized bool             `json:"is_unlimited_collateralized"`

	// Cột 2: Đã sử dụng đầu ngày có TSBĐ
	UsedOpeningCollateralized decimal.Decimal `json:"used_opening_collateralized"`

	// Cột 3: Sử dụng trong ngày có TSBĐ
	UsedIntradayCollateralized decimal.Decimal `json:"used_intraday_collateralized"`

	// Cột 4: Còn lại có TSBĐ = (1) - (2) - (3)
	RemainingCollateralized *decimal.Decimal `json:"remaining_collateralized"`

	// Cột 5: Hạn mức cấp không TSBĐ
	AllocatedUncollateralized   *decimal.Decimal `json:"allocated_uncollateralized"`
	IsUnlimitedUncollateralized bool             `json:"is_unlimited_uncollateralized"`

	// Cột 6: Đã sử dụng đầu ngày không TSBĐ
	UsedOpeningUncollateralized decimal.Decimal `json:"used_opening_uncollateralized"`

	// Cột 7: Sử dụng trong ngày không TSBĐ
	UsedIntradayUncollateralized decimal.Decimal `json:"used_intraday_uncollateralized"`

	// Cột 8: Còn lại không TSBĐ = (5) - (6) - (7)
	RemainingUncollateralized *decimal.Decimal `json:"remaining_uncollateralized"`
}

// DailySummaryResponse wraps the daily summary table data.
type DailySummaryResponse struct {
	Date string            `json:"date"`
	Rows []DailySummaryRow `json:"rows"`
}
