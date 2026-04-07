package email

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockSender is a configurable mock for testing.
type mockSender struct {
	mu       sync.Mutex
	calls    []*OutboxEmail
	err      error
	errOnce  bool // only fail once, then succeed
	failedAt int
}

func (m *mockSender) Send(_ context.Context, msg *OutboxEmail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, msg)
	if m.err != nil {
		if m.errOnce {
			m.failedAt = len(m.calls)
			m.err = nil // clear for next call
			return errors.New("smtp: temporary failure")
		}
		return m.err
	}
	return nil
}

func (m *mockSender) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func newTestWorker(repo OutboxRepository, sender Sender) *Worker {
	templates, _ := NewTemplateRenderer()
	return NewWorker(repo, sender, templates, zap.NewNop(), 100, 100) // high rate limit for tests
}

func TestWorkerProcessOne_Success(t *testing.T) {
	repo := NewMockOutboxRepo()
	sender := &mockSender{}
	w := newTestWorker(repo, sender)

	email := &OutboxEmail{
		ID:          uuid.New(),
		ToAddresses: []string{"test@example.com"},
		FromAddress: "sender@example.com",
		Subject:     "Test Subject",
		BodyHTML:    "<p>Hello</p>",
		BodyText:    "Hello",
		Status:      StatusPending,
		MaxRetries:  3,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		TriggeredBy: uuid.New(),
	}
	require.NoError(t, repo.Insert(context.Background(), email))

	w.ProcessOne(context.Background(), *email)

	assert.Equal(t, 1, sender.callCount())

	updated, _ := repo.GetByID(context.Background(), email.ID)
	assert.Equal(t, StatusSent, updated.Status)
	assert.NotNil(t, updated.SentAt)
}

func TestWorkerProcessOne_RetryOnFailure(t *testing.T) {
	repo := NewMockOutboxRepo()
	sender := &mockSender{err: errors.New("smtp: connection refused")}
	w := newTestWorker(repo, sender)

	email := &OutboxEmail{
		ID:          uuid.New(),
		ToAddresses: []string{"test@example.com"},
		FromAddress: "sender@example.com",
		Subject:     "Test",
		BodyHTML:    "<p>Hello</p>",
		Status:      StatusPending,
		RetryCount:  0,
		MaxRetries:  3,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		TriggeredBy: uuid.New(),
	}
	require.NoError(t, repo.Insert(context.Background(), email))

	w.ProcessOne(context.Background(), *email)

	updated, _ := repo.GetByID(context.Background(), email.ID)
	assert.Equal(t, StatusRetry, updated.Status)
	assert.Contains(t, updated.LastError, "smtp: connection refused")
	assert.True(t, updated.NextRetryAt.After(time.Now()))
}

func TestWorkerProcessOne_FailAfterMaxRetries(t *testing.T) {
	repo := NewMockOutboxRepo()
	sender := &mockSender{err: errors.New("smtp: permanent failure")}
	w := newTestWorker(repo, sender)

	email := &OutboxEmail{
		ID:          uuid.New(),
		ToAddresses: []string{"test@example.com"},
		FromAddress: "sender@example.com",
		Subject:     "Test",
		BodyHTML:    "<p>Hello</p>",
		Status:      StatusRetry,
		RetryCount:  2, // already retried twice, next will be 3 (== MaxRetries)
		MaxRetries:  3,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		TriggeredBy: uuid.New(),
	}
	require.NoError(t, repo.Insert(context.Background(), email))

	w.ProcessOne(context.Background(), *email)

	updated, _ := repo.GetByID(context.Background(), email.ID)
	assert.Equal(t, StatusFailed, updated.Status)
	assert.Contains(t, updated.LastError, "permanent failure")
	assert.NotNil(t, updated.FailedAt)
}

func TestWorkerProcessOne_TemplateRendering(t *testing.T) {
	repo := NewMockOutboxRepo()
	sender := &mockSender{}
	w := newTestWorker(repo, sender)

	email := &OutboxEmail{
		ID:          uuid.New(),
		ToAddresses: []string{"kttc@kienlongbank.com"},
		FromAddress: "treasury@kienlongbank.com",
		Subject:     "[Treasury] Hủy giao dịch FX",
		TemplateName: "deal_cancelled",
		TemplateData: map[string]string{
			"DealModule":       "FX",
			"TicketNumber":     "FX-001",
			"CounterpartyName": "BIDV",
			"Amount":           "100,000",
			"Currency":         "USD",
			"CancelReason":     "Hủy theo yêu cầu",
			"RequestedBy":      "User A",
			"ApprovedBy":       "User B",
			"CancelledAt":      "04/04/2026 10:00:00",
		},
		Status:      StatusPending,
		MaxRetries:  3,
		NextRetryAt: time.Now(),
		CreatedAt:   time.Now(),
		TriggeredBy: uuid.New(),
	}
	require.NoError(t, repo.Insert(context.Background(), email))

	w.ProcessOne(context.Background(), *email)

	assert.Equal(t, 1, sender.callCount())
	sent := sender.calls[0]
	assert.Contains(t, sent.BodyHTML, "Thông báo hủy giao dịch")
	assert.Contains(t, sent.BodyHTML, "FX-001")
	assert.NotEmpty(t, sent.BodyText)
}

func TestWorkerThrottling(t *testing.T) {
	repo := NewMockOutboxRepo()
	sender := &mockSender{}

	templates, _ := NewTemplateRenderer()
	// Very low rate limit: 2 per second, burst 2
	w := NewWorker(repo, sender, templates, zap.NewNop(), 2, 2)

	// Insert 5 emails
	for i := 0; i < 5; i++ {
		email := &OutboxEmail{
			ID:          uuid.New(),
			ToAddresses: []string{"test@example.com"},
			FromAddress: "sender@example.com",
			Subject:     "Throttle Test",
			BodyHTML:    "<p>Test</p>",
			Status:      StatusPending,
			MaxRetries:  3,
			NextRetryAt: time.Now(),
			CreatedAt:   time.Now(),
			TriggeredBy: uuid.New(),
		}
		require.NoError(t, repo.Insert(context.Background(), email))
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	w.processBatch(ctx)
	elapsed := time.Since(start)

	// With rate limit of 2/sec and 5 emails, should take at least 1 second
	// (burst handles first 2 instantly, then ~1.5s for remaining 3)
	assert.Equal(t, 5, sender.callCount())
	assert.True(t, elapsed > 500*time.Millisecond, "expected throttling delay, got %v", elapsed)
}

func TestWorkerIndependentFailure(t *testing.T) {
	repo := NewMockOutboxRepo()

	// Fail on second email only
	sender := &failNthSender{failOn: 2}

	w := newTestWorker(repo, sender)

	var ids [3]uuid.UUID
	for i := 0; i < 3; i++ {
		ids[i] = uuid.New()
		email := &OutboxEmail{
			ID:          ids[i],
			ToAddresses: []string{"test@example.com"},
			FromAddress: "sender@example.com",
			Subject:     "Test",
			BodyHTML:    "<p>Test</p>",
			Status:      StatusPending,
			MaxRetries:  3,
			NextRetryAt: time.Now(),
			CreatedAt:   time.Now(),
			TriggeredBy: uuid.New(),
		}
		require.NoError(t, repo.Insert(context.Background(), email))
	}

	// Process all
	emails, _ := repo.FetchPending(context.Background(), 10)
	for _, e := range emails {
		w.ProcessOne(context.Background(), e)
	}

	// First and third should succeed, second should be in retry
	sentCount := 0
	retryCount := 0
	for _, e := range repo.GetAll() {
		switch e.Status {
		case StatusSent:
			sentCount++
		case StatusRetry:
			retryCount++
		}
	}
	assert.Equal(t, 2, sentCount, "2 emails should be sent")
	assert.Equal(t, 1, retryCount, "1 email should be in retry")
}

// failNthSender fails on the Nth call.
type failNthSender struct {
	mu     sync.Mutex
	count  int
	failOn int
}

func (f *failNthSender) Send(_ context.Context, _ *OutboxEmail) error {
	f.mu.Lock()
	f.count++
	n := f.count
	f.mu.Unlock()
	if n == f.failOn {
		return errors.New("smtp: temporary failure")
	}
	return nil
}
