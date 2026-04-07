// Package model defines domain models with business logic for the Treasury system.
package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a system user.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name"`
	PasswordHash string     `json:"-"`
	Roles        []string   `json:"roles"`
	BranchID     string     `json:"branch_id"`
	BranchName   string     `json:"branch_name"`
	Department   string     `json:"department"`
	Position     string     `json:"position"`
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// HasRole checks if the user has a specific role.
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the given roles.
func (u *User) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}
