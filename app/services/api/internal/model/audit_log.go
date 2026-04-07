package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	UserFullName   string    `json:"user_full_name"`
	UserDepartment *string   `json:"user_department,omitempty"`
	UserBranchCode *string   `json:"user_branch_code,omitempty"`
	Action         string    `json:"action"`
	DealModule     string    `json:"deal_module"`
	DealID         *uuid.UUID `json:"deal_id,omitempty"`
	StatusBefore   *string   `json:"status_before,omitempty"`
	StatusAfter    *string   `json:"status_after,omitempty"`
	OldValues      []byte    `json:"old_values,omitempty"`
	NewValues      []byte    `json:"new_values,omitempty"`
	Reason         *string   `json:"reason,omitempty"`
	IPAddress      *string   `json:"ip_address,omitempty"`
	UserAgent      *string   `json:"user_agent,omitempty"`
	PerformedAt    time.Time `json:"performed_at"`
}
