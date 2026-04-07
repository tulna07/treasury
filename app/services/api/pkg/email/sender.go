package email

import "context"

// Sender is the interface for sending emails via SMTP or other transports.
type Sender interface {
	Send(ctx context.Context, msg *OutboxEmail) error
}
