package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Credit limit types per BRD §3.4
const (
	LimitTypeCollateralized   = "COLLATERALIZED"  // Có TSBĐ
	LimitTypeUncollateralized = "UNCOLLATERALIZED" // Không TSBĐ
)

// Limit approval statuses
const (
	LimitApprovalPending    = "PENDING"
	LimitApprovalRiskL1Done = "RISK_L1_APPROVED"
	LimitApprovalApproved   = "APPROVED"
	LimitApprovalRejected   = "REJECTED"
)

// CreditLimit represents a credit limit record (SCD Type 2).
// Legacy fields (ApprovedAmount, UsedAmount, etc.) are retained for backward
// compatibility with pkg/limitcheck — they will be removed once that package
// is migrated to use the new schema.
type CreditLimit struct {
	ID             uuid.UUID
	CounterpartyID uuid.UUID
	LimitType      string // COLLATERALIZED, UNCOLLATERALIZED (new) or FX/MM/BOND/TOTAL (legacy)

	// ── New BRD §3.4 fields ──
	LimitAmount       *decimal.Decimal
	IsUnlimited       bool
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time
	IsCurrent         bool
	ApprovalReference *string
	CreatedBy         *uuid.UUID
	UpdatedBy         *uuid.UUID

	// ── Legacy fields (for pkg/limitcheck compat) ──
	ApprovedAmount decimal.Decimal
	UsedAmount     decimal.Decimal
	CurrencyCode   string
	EffectiveDate  time.Time
	ExpiryDate     *time.Time
	Status         string // ACTIVE, EXPIRED, SUSPENDED
	Note           *string
	Version        int

	// ── Shared timestamps ──
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AvailableAmount calculates the remaining available limit (legacy).
func (l *CreditLimit) AvailableAmount() decimal.Decimal {
	return l.ApprovedAmount.Sub(l.UsedAmount)
}

// CalculateUtilization calculates the utilization percentage (legacy).
func (l *CreditLimit) CalculateUtilization() decimal.Decimal {
	if l.ApprovedAmount.IsZero() {
		return decimal.Zero
	}
	return l.UsedAmount.Div(l.ApprovedAmount).Mul(decimal.NewFromInt(100))
}

// IsExceeded checks if the limit is exceeded by a given amount (legacy).
func (l *CreditLimit) IsExceeded(dealAmount decimal.Decimal) bool {
	return l.UsedAmount.Add(dealAmount).GreaterThan(l.ApprovedAmount)
}

// IsActive checks if the limit is currently active (legacy).
func (l *CreditLimit) IsActive() bool {
	now := time.Now()
	if l.ExpiryDate == nil {
		return l.Status == "ACTIVE" && now.After(l.EffectiveDate)
	}
	return l.Status == "ACTIVE" && now.After(l.EffectiveDate) && now.Before(*l.ExpiryDate)
}

// RemainingAmount returns remaining for new schema (unlimited → nil sentinel).
func (l *CreditLimit) RemainingAmount(used decimal.Decimal) *decimal.Decimal {
	if l.IsUnlimited || l.LimitAmount == nil {
		return nil // unlimited
	}
	rem := l.LimitAmount.Sub(used)
	return &rem
}

// LimitUtilizationSnapshot is an append-only snapshot of limit usage on a given date.
type LimitUtilizationSnapshot struct {
	ID               uuid.UUID
	CounterpartyID   uuid.UUID
	SnapshotDate     time.Time
	LimitType        string
	LimitGranted     *decimal.Decimal
	UtilizedOpening  decimal.Decimal
	UtilizedIntraday decimal.Decimal
	UtilizedTotal    decimal.Decimal
	Remaining        *decimal.Decimal
	FxRateApplied    *decimal.Decimal
	BreakdownDetail  map[string]interface{}
	CreatedAt        time.Time
	CreatedBy        *uuid.UUID
}

// LimitApprovalRecord tracks per-deal credit limit approval (CV QLRR → TPB QLRR).
type LimitApprovalRecord struct {
	ID                    uuid.UUID
	DealModule            string // FX, MM, BOND
	DealID                uuid.UUID
	CounterpartyID        uuid.UUID
	LimitType             string
	DealAmountVND         decimal.Decimal
	LimitSnapshot         map[string]interface{}
	RiskOfficerApprovedBy *uuid.UUID
	RiskOfficerApprovedAt *time.Time
	RiskHeadApprovedBy    *uuid.UUID
	RiskHeadApprovedAt    *time.Time
	ApprovalStatus        string
	RejectionReason       *string
	CreatedAt             time.Time
}

// DailySummaryRow represents one row in the 11-column daily limit summary (BRD §3.4.4).
type DailySummaryRow struct {
	CounterpartyID   uuid.UUID `json:"counterparty_id"`
	CounterpartyName string    `json:"counterparty_name"`
	CIFCode          string    `json:"cif_code"`

	// Collateralized (có TSBĐ)
	AllocatedCollateralized    *decimal.Decimal `json:"allocated_collateralized"`
	IsUnlimitedCollateralized  bool             `json:"is_unlimited_collateralized"`
	UsedOpeningCollateralized  decimal.Decimal  `json:"used_opening_collateralized"`
	UsedIntradayCollateralized decimal.Decimal  `json:"used_intraday_collateralized"`
	RemainingCollateralized    *decimal.Decimal `json:"remaining_collateralized"`

	// Uncollateralized (không TSBĐ)
	AllocatedUncollateralized    *decimal.Decimal `json:"allocated_uncollateralized"`
	IsUnlimitedUncollateralized  bool             `json:"is_unlimited_uncollateralized"`
	UsedOpeningUncollateralized  decimal.Decimal  `json:"used_opening_uncollateralized"`
	UsedIntradayUncollateralized decimal.Decimal  `json:"used_intraday_uncollateralized"`
	RemainingUncollateralized    *decimal.Decimal `json:"remaining_uncollateralized"`
}
