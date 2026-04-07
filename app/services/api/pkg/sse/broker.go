// Package sse provides an in-memory pub/sub broker for Server-Sent Events.
package sse

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Event represents a single SSE event to be pushed to a client.
type Event struct {
	ID   string          `json:"id,omitempty"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Broker manages SSE subscriber channels per user.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[chan Event]struct{}
	logger      *zap.Logger
}

// NewBroker creates a new SSE Broker.
func NewBroker(logger *zap.Logger) *Broker {
	return &Broker{
		subscribers: make(map[uuid.UUID]map[chan Event]struct{}),
		logger:      logger,
	}
}

// Subscribe creates a new event channel for the given user.
func (b *Broker) Subscribe(userID uuid.UUID) chan Event {
	ch := make(chan Event, 64)
	b.mu.Lock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan Event]struct{})
	}
	b.subscribers[userID][ch] = struct{}{}
	b.mu.Unlock()

	b.logger.Debug("SSE subscriber added", zap.String("user_id", userID.String()))
	return ch
}

// Unsubscribe removes a channel for the given user and closes it.
func (b *Broker) Unsubscribe(userID uuid.UUID, ch chan Event) {
	b.mu.Lock()
	if subs, ok := b.subscribers[userID]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.subscribers, userID)
		}
	}
	b.mu.Unlock()
	close(ch)

	b.logger.Debug("SSE subscriber removed", zap.String("user_id", userID.String()))
}

// Publish sends an event to all subscribers of the given user.
func (b *Broker) Publish(userID uuid.UUID, event Event) {
	b.mu.RLock()
	subs := b.subscribers[userID]
	b.mu.RUnlock()

	for ch := range subs {
		select {
		case ch <- event:
		default:
			b.logger.Warn("SSE channel full, dropping event",
				zap.String("user_id", userID.String()),
				zap.String("event_type", event.Type),
			)
		}
	}
}

// PublishToMany sends an event to all subscribers of the given user IDs.
func (b *Broker) PublishToMany(userIDs []uuid.UUID, event Event) {
	for _, uid := range userIDs {
		b.Publish(uid, event)
	}
}
