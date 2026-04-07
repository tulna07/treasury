package model

import (
	"time"

	"github.com/google/uuid"
)

// AdminUser represents a user with roles info for admin views.
type AdminUser struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	BranchID    string     `json:"branch_id"`
	BranchName  string     `json:"branch_name"`
	Department  string     `json:"department"`
	Position    string     `json:"position"`
	IsActive    bool       `json:"is_active"`
	RoleCodes   []string   `json:"role_codes"`
	RoleNames   []string   `json:"role_names"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Role represents a system role.
type Role struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
}

// Permission represents a system permission.
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}
