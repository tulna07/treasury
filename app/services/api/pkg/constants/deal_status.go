// Package constants defines all business constants for the Treasury system.
package constants

// Deal status constants matching DB enum values.
const (
	StatusOpen                   = "OPEN"
	StatusPendingL2Approval      = "PENDING_L2_APPROVAL"
	StatusPendingTPReview        = "PENDING_TP_REVIEW"
	StatusRejected               = "REJECTED"
	StatusPendingBooking         = "PENDING_BOOKING"
	StatusPendingChiefAccountant = "PENDING_CHIEF_ACCOUNTANT"
	StatusPendingRiskApproval    = "PENDING_RISK_APPROVAL"
	StatusPendingSettlement      = "PENDING_SETTLEMENT"
	StatusCompleted              = "COMPLETED"
	StatusVoidedByAccounting     = "VOIDED_BY_ACCOUNTING"
	StatusVoidedBySettlement     = "VOIDED_BY_SETTLEMENT"
	StatusVoidedByRisk           = "VOIDED_BY_RISK"
	StatusCancelled              = "CANCELLED"
	StatusPendingCancelL1        = "PENDING_CANCEL_L1"
	StatusPendingCancelL2        = "PENDING_CANCEL_L2"
)

// CancelledStatuses contains all statuses that represent cancelled/voided deals.
var CancelledStatuses = []string{
	StatusCancelled, StatusVoidedByAccounting, StatusVoidedByRisk, StatusVoidedBySettlement,
}

// AllStatuses contains all valid deal statuses.
var AllStatuses = []string{
	StatusOpen, StatusPendingL2Approval, StatusPendingTPReview, StatusRejected,
	StatusPendingBooking, StatusPendingChiefAccountant,
	StatusPendingRiskApproval, StatusPendingSettlement,
	StatusCompleted, StatusVoidedByAccounting,
	StatusVoidedBySettlement, StatusVoidedByRisk,
	StatusCancelled,
	StatusPendingCancelL1, StatusPendingCancelL2,
}

// IsValidStatus checks if a status string is a valid deal status.
func IsValidStatus(status string) bool {
	for _, s := range AllStatuses {
		if s == status {
			return true
		}
	}
	return false
}
