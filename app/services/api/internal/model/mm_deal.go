package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/pkg/constants"
	klbdecimal "github.com/kienlongbank/treasury-api/pkg/decimal"
)

// MMInterbankDeal represents a Money Market interbank deal (Giao dịch liên ngân hàng).
type MMInterbankDeal struct {
	ID                              uuid.UUID
	DealNumber                      string
	TicketNumber                    *string
	CounterpartyID                  uuid.UUID
	CounterpartyCode                string // denormalized from view
	CounterpartyName                string // denormalized from view
	BranchID                        uuid.UUID
	BranchCode                      string // denormalized from view
	BranchName                      string // denormalized from view
	CurrencyCode                    string
	InternalSSIID                   *uuid.UUID
	CounterpartySSIID               *uuid.UUID
	CounterpartySSIText             *string
	Direction                       string // PLACE, TAKE, LEND, BORROW
	PrincipalAmount                 decimal.Decimal
	InterestRate                    decimal.Decimal
	DayCountConvention              string // ACT_365, ACT_360, ACT_ACT
	TradeDate                       time.Time
	EffectiveDate                   time.Time
	TenorDays                       int
	MaturityDate                    time.Time
	InterestAmount                  decimal.Decimal
	MaturityAmount                  decimal.Decimal
	HasCollateral                   bool
	CollateralCurrency              *string
	CollateralDescription           *string
	RequiresInternationalSettlement bool
	Status                          string
	Note                            *string
	ClonedFromID                    *uuid.UUID
	CancelReason                    *string
	CancelRequestedBy               *uuid.UUID
	CancelRequestedAt               *time.Time
	CreatedBy                       uuid.UUID
	CreatedByName                   string // denormalized from view
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
	UpdatedBy                       uuid.UUID
	DeletedAt                       *time.Time
	Version                         int
}

// CalculateInterest calculates interest = principal x (rate/100) x (actual_days / day_count_base).
func (d *MMInterbankDeal) CalculateInterest() decimal.Decimal {
	days := decimal.NewFromInt(int64(d.TenorDays))
	rate := d.InterestRate.Div(decimal.NewFromInt(100))

	var base decimal.Decimal
	switch d.DayCountConvention {
	case constants.DayCountACT360:
		base = decimal.NewFromInt(360)
	case constants.DayCountACT365:
		base = decimal.NewFromInt(365)
	case constants.DayCountACTACT:
		base = decimal.NewFromInt(int64(daysInYear(d.EffectiveDate.Year())))
	default:
		return decimal.Zero
	}

	interest := d.PrincipalAmount.Mul(rate).Mul(days).Div(base)
	return d.RoundAmount(interest)
}

// CalculateMaturityAmount returns principal + interest.
func (d *MMInterbankDeal) CalculateMaturityAmount() decimal.Decimal {
	return d.PrincipalAmount.Add(d.CalculateInterest())
}

// CalculateInterestACTACT calculates interest with precise ACT/ACT across year boundaries.
func (d *MMInterbankDeal) CalculateInterestACTACT() decimal.Decimal {
	rate := d.InterestRate.Div(decimal.NewFromInt(100))
	totalInterest := decimal.Zero

	if d.EffectiveDate.After(d.MaturityDate) || d.EffectiveDate.Equal(d.MaturityDate) {
		return decimal.Zero
	}

	currentDate := d.EffectiveDate
	for currentDate.Year() < d.MaturityDate.Year() {
		yearEnd := time.Date(currentDate.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		periodDays := decimal.NewFromFloat(yearEnd.Sub(currentDate).Hours()/24 + 1)
		yearDays := decimal.NewFromInt(int64(daysInYear(currentDate.Year())))
		interestForPeriod := d.PrincipalAmount.Mul(rate).Mul(periodDays).Div(yearDays)
		totalInterest = totalInterest.Add(interestForPeriod)
		currentDate = yearEnd.AddDate(0, 0, 1)
	}

	// Final period in the maturity year
	finalDays := decimal.NewFromFloat(d.MaturityDate.Sub(currentDate).Hours() / 24)
	finalYearDays := decimal.NewFromInt(int64(daysInYear(d.MaturityDate.Year())))
	interestForFinalPeriod := d.PrincipalAmount.Mul(rate).Mul(finalDays).Div(finalYearDays)
	totalInterest = totalInterest.Add(interestForFinalPeriod)

	return d.RoundAmount(totalInterest)
}

// CalculateTenorDays computes maturity_date - effective_date in days.
func (d *MMInterbankDeal) CalculateTenorDays() int {
	return int(d.MaturityDate.Sub(d.EffectiveDate).Hours() / 24)
}

// RoundAmount rounds a decimal amount based on currency (VND=0, USD=2).
func (d *MMInterbankDeal) RoundAmount(amount decimal.Decimal) decimal.Decimal {
	return amount.Round(klbdecimal.DecimalsForCurrency(d.CurrencyCode))
}

// CanEdit checks if the deal can be edited (only OPEN).
func (d *MMInterbankDeal) CanEdit() bool {
	return d.Status == constants.StatusOpen
}

// CanRecall checks if the deal can be recalled from a pending state.
// Interbank flow: PENDING_TP_REVIEW, PENDING_L2, PENDING_RISK, PENDING_BOOKING, PENDING_CHIEF_ACCOUNTANT, PENDING_SETTLEMENT
func (d *MMInterbankDeal) CanRecall() bool {
	switch d.Status {
	case constants.StatusPendingTPReview,
		constants.StatusPendingL2Approval,
		constants.StatusPendingRiskApproval,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant,
		constants.StatusPendingSettlement:
		return true
	}
	return false
}

// CanCancel checks if the deal can be cancelled (only COMPLETED).
func (d *MMInterbankDeal) CanCancel() bool {
	return d.Status == constants.StatusCompleted
}

// CanClone checks if the deal can be cloned.
func (d *MMInterbankDeal) CanClone() bool {
	switch d.Status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting, constants.StatusVoidedByRisk:
		return true
	}
	return false
}

// --- MMOMORepoDeal ---

// MMOMORepoDeal represents an OMO or Repo KBNN deal.
type MMOMORepoDeal struct {
	ID                  uuid.UUID
	DealNumber          string
	DealSubtype         string // OMO or STATE_REPO
	SessionName         string
	TradeDate           time.Time
	BranchID            uuid.UUID
	BranchCode          string // denormalized from view
	BranchName          string // denormalized from view
	CounterpartyID      uuid.UUID
	CounterpartyCode    string // denormalized from view
	CounterpartyName    string // denormalized from view
	NotionalAmount      decimal.Decimal
	BondCatalogID       uuid.UUID
	BondCode            string          // denormalized from view
	BondIssuer          string          // denormalized from view
	BondCouponRate      decimal.Decimal // denormalized from view
	BondMaturityDate    *time.Time      // denormalized from view
	WinningRate         decimal.Decimal
	TenorDays           int
	SettlementDate1     time.Time
	SettlementDate2     time.Time
	HaircutPct          decimal.Decimal
	Status              string
	Note                *string
	ClonedFromID        *uuid.UUID
	CancelReason        *string
	CancelRequestedBy   *uuid.UUID
	CancelRequestedAt   *time.Time
	CreatedBy           uuid.UUID
	CreatedByName       string // denormalized from view
	CreatedAt           time.Time
	UpdatedAt           time.Time
	UpdatedBy           uuid.UUID
	DeletedAt           *time.Time
	Version             int
}

// IsOMO returns true if the deal subtype is OMO.
func (d *MMOMORepoDeal) IsOMO() bool {
	return d.DealSubtype == constants.MMSubtypeOMO
}

// IsStateRepo returns true if the deal subtype is STATE_REPO.
func (d *MMOMORepoDeal) IsStateRepo() bool {
	return d.DealSubtype == constants.MMSubtypeStateRepo
}

// CanEdit checks if the deal can be edited (only OPEN).
func (d *MMOMORepoDeal) CanEdit() bool {
	return d.Status == constants.StatusOpen
}

// CanRecall checks if the deal can be recalled.
// OMO/Repo flow: PENDING_L2, PENDING_BOOKING, PENDING_CHIEF_ACCOUNTANT
func (d *MMOMORepoDeal) CanRecall() bool {
	switch d.Status {
	case constants.StatusPendingL2Approval,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant:
		return true
	}
	return false
}

// CanCancel checks if the deal can be cancelled (only COMPLETED).
func (d *MMOMORepoDeal) CanCancel() bool {
	return d.Status == constants.StatusCompleted
}

// CanClone checks if the deal can be cloned.
func (d *MMOMORepoDeal) CanClone() bool {
	switch d.Status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting:
		return true
	}
	return false
}

// --- Helpers ---

// isLeap checks if a year is a leap year.
func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// daysInYear returns 366 for leap years, 365 otherwise.
func daysInYear(year int) int {
	if isLeap(year) {
		return 366
	}
	return 365
}
