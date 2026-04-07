package dto

import (
	"time"

	"github.com/google/uuid"
)

// AuditLogResponse represents an audit log entry.
type AuditLogResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	UserFullName   string                 `json:"user_full_name"`
	UserDepartment *string                `json:"user_department,omitempty"`
	UserBranchCode *string                `json:"user_branch_code,omitempty"`
	Action         string                 `json:"action"`
	DealModule     string                 `json:"deal_module"`
	DealID         *uuid.UUID             `json:"deal_id,omitempty"`
	StatusBefore   *string                `json:"status_before,omitempty"`
	StatusAfter    *string                `json:"status_after,omitempty"`
	OldValues      map[string]interface{} `json:"old_values,omitempty"`
	NewValues      map[string]interface{} `json:"new_values,omitempty"`
	Reason         *string                `json:"reason,omitempty"`
	IPAddress      *string                `json:"ip_address,omitempty"`
	PerformedAt    time.Time              `json:"performed_at"`
}

// AuditLogFilter holds filter criteria for listing audit logs.
type AuditLogFilter struct {
	UserID     *uuid.UUID
	DealModule *string
	DealID     *uuid.UUID
	Action     *string
	DateFrom   *string
	DateTo     *string
}

// AuditLogStatsResponse represents action count stats.
type AuditLogStatsResponse struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}
