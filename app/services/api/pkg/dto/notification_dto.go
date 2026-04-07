package dto

import (
	"time"

	"github.com/google/uuid"
)

// NotificationResponse is the API response for a single notification.
type NotificationResponse struct {
	ID         uuid.UUID  `json:"id"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	Category   string     `json:"category"`
	DealModule string     `json:"deal_module,omitempty"`
	DealID     *uuid.UUID `json:"deal_id,omitempty"`
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// UnreadCountResponse is the API response for the unread notification count.
type UnreadCountResponse struct {
	Count int `json:"count"`
}
