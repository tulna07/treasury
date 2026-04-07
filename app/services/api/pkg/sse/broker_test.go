package sse

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testBroker() *Broker {
	return NewBroker(zap.NewNop())
}

func TestSubscribeUnsubscribe(t *testing.T) {
	b := testBroker()
	userID := uuid.New()

	ch := b.Subscribe(userID)
	assert.NotNil(t, ch)

	b.mu.RLock()
	assert.Len(t, b.subscribers[userID], 1)
	b.mu.RUnlock()

	b.Unsubscribe(userID, ch)

	b.mu.RLock()
	assert.Len(t, b.subscribers[userID], 0)
	b.mu.RUnlock()
}

func TestPublish(t *testing.T) {
	b := testBroker()
	userID := uuid.New()

	ch := b.Subscribe(userID)
	defer b.Unsubscribe(userID, ch)

	event := Event{
		ID:   "1",
		Type: "notification",
		Data: json.RawMessage(`{"title":"test"}`),
	}

	b.Publish(userID, event)

	select {
	case received := <-ch:
		assert.Equal(t, event.Type, received.Type)
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestPublishToMany(t *testing.T) {
	b := testBroker()
	user1 := uuid.New()
	user2 := uuid.New()

	ch1 := b.Subscribe(user1)
	defer b.Unsubscribe(user1, ch1)
	ch2 := b.Subscribe(user2)
	defer b.Unsubscribe(user2, ch2)

	event := Event{
		Type: "badge_update",
		Data: json.RawMessage(`{"count":5}`),
	}

	b.PublishToMany([]uuid.UUID{user1, user2}, event)

	for _, ch := range []chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			assert.Equal(t, "badge_update", received.Type)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	}
}

func TestPublishNoSubscribers(t *testing.T) {
	b := testBroker()
	// Should not panic
	b.Publish(uuid.New(), Event{Type: "test", Data: json.RawMessage(`{}`)})
}

func TestConcurrentAccess(t *testing.T) {
	b := testBroker()
	userID := uuid.New()

	var wg sync.WaitGroup
	channels := make([]chan Event, 10)

	// Subscribe concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			channels[idx] = b.Subscribe(userID)
		}(i)
	}
	wg.Wait()

	// Publish concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Publish(userID, Event{Type: "test", Data: json.RawMessage(`{}`)})
		}()
	}
	wg.Wait()

	// Unsubscribe concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b.Unsubscribe(userID, channels[idx])
		}(i)
	}
	wg.Wait()

	b.mu.RLock()
	assert.Empty(t, b.subscribers[userID])
	b.mu.RUnlock()
}

func TestChannelBufferFull(t *testing.T) {
	b := testBroker()
	userID := uuid.New()

	ch := b.Subscribe(userID)
	defer b.Unsubscribe(userID, ch)

	// Fill the buffer (64 events)
	for i := 0; i < 64; i++ {
		b.Publish(userID, Event{Type: "test", Data: json.RawMessage(`{}`)})
	}

	// 65th event should be dropped without blocking
	b.Publish(userID, Event{Type: "dropped", Data: json.RawMessage(`{}`)})

	// Drain and verify we got 64 events
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			require.Equal(t, 64, count)
			return
		}
	}
}
