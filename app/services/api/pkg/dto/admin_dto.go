package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- User Management DTOs ---

// CreateUserRequest is the payload for creating a new user.
type CreateUserRequest struct {
	Username   string  `json:"username" validate:"required,min=3,max=100"`
	FullName   string  `json:"full_name" validate:"required,min=1,max=255"`
	Email      string  `json:"email" validate:"required,email,max=255"`
	Password   string  `json:"password" validate:"required,min=8,max=128"`
	BranchID   *string `json:"branch_id,omitempty"`
	Department *string `json:"department,omitempty"`
	Position   *string `json:"position,omitempty"`
}

// UpdateUserRequest is the payload for updating a user.
type UpdateUserRequest struct {
	FullName   *string `json:"full_name,omitempty" validate:"omitempty,min=1,max=255"`
	Email      *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	BranchID   *string `json:"branch_id,omitempty"`
	Department *string `json:"department,omitempty"`
	Position   *string `json:"position,omitempty"`
}

// AdminUserResponse represents a user in admin context (with roles, no password).
type AdminUserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Username    string     `json:"username"`
	FullName    string     `json:"full_name"`
	Email       string     `json:"email"`
	BranchID    string     `json:"branch_id"`
	BranchName  string     `json:"branch_name"`
	Department  string     `json:"department"`
	Position    string     `json:"position"`
	IsActive    bool       `json:"is_active"`
	Roles       []string   `json:"roles"`
	RoleNames   []string   `json:"role_names"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// LockUnlockRequest is the payload for lock/unlock reason.
type LockUnlockRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// ResetPasswordResponse contains the temporary password after admin reset.
type ResetPasswordResponse struct {
	TempPassword string `json:"temp_password"`
}

// --- Role Management DTOs ---

// RoleResponse represents a role.
type RoleResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
}

// RolePermissionResponse represents permissions for a role.
type RolePermissionResponse struct {
	RoleCode    string   `json:"role_code"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}

// PermissionResponse represents a permission in the system.
type PermissionResponse struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// UpdateRolePermissionsRequest is the payload for updating a role's permissions.
type UpdateRolePermissionsRequest struct {
	Permissions []string `json:"permissions" validate:"required"`
}

// AssignRoleRequest is the payload for assigning a role to a user.
type AssignRoleRequest struct {
	RoleCode string `json:"role_code" validate:"required"`
	Reason   string `json:"reason" validate:"required,min=1,max=500"`
}

// RevokeRoleRequest is the payload for revoking a role from a user.
type RevokeRoleRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// UserFilter holds filter criteria for listing users.
type UserFilter struct {
	Department *string
	BranchID   *string
	Role       *string
	IsActive   *bool
	Search     *string
}
