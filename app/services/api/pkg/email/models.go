package email

import (
	"time"

	"github.com/google/uuid"
)

// OutboxEmail represents a single email in the outbox queue.
type OutboxEmail struct {
	ID             uuid.UUID
	ToAddresses    []string
	CCAddresses    []string
	FromAddress    string
	Subject        string
	BodyHTML       string
	BodyText       string
	TemplateName   string
	TemplateData   map[string]string
	DealModule     string
	DealID         *uuid.UUID
	TriggerEvent   string
	TriggeredBy    uuid.UUID
	Status         string
	RetryCount     int
	MaxRetries     int
	NextRetryAt    time.Time
	LastError      string
	IdempotencyKey string
	CreatedAt      time.Time
	SentAt         *time.Time
	FailedAt       *time.Time
}

// Email status constants.
const (
	StatusPending = "PENDING"
	StatusSending = "SENDING"
	StatusSent    = "SENT"
	StatusRetry   = "RETRY"
	StatusFailed  = "FAILED"
)

// DealCancelledParams contains data for the deal_cancelled email template.
type DealCancelledParams struct {
	DealModule       string
	DealID           uuid.UUID
	TicketNumber     string
	CounterpartyName string
	Amount           string
	Currency         string
	CancelReason     string
	RequestedBy      string
	ApprovedBy       string
	IsInternational  bool
	TriggeredBy      uuid.UUID
}

// DealVoidedParams contains data for the deal_voided email template.
type DealVoidedParams struct {
	DealModule       string
	DealID           uuid.UUID
	TicketNumber     string
	CounterpartyName string
	Amount           string
	Currency         string
	VoidReason       string
	VoidedBy         string
	TriggeredBy      uuid.UUID
}
