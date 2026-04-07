package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents an in-app notification for a user.
type Notification struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Category   string     `json:"category"`
	DealModule string     `json:"deal_module"`
	DealID     *uuid.UUID `json:"deal_id,omitempty"`
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
