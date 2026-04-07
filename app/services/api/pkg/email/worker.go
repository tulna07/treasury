package email

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/google/uuid"
)

// Worker polls the outbox and sends emails with rate limiting.
type Worker struct {
	repo      OutboxRepository
	sender    Sender
	templates *TemplateRenderer
	logger    *zap.Logger
	limiter   *rate.Limiter
	notify    chan struct{}
	wg        sync.WaitGroup
	stopOnce  sync.Once
	cancel    context.CancelFunc
}

// NewWorker creates a new email outbox worker.
func NewWorker(repo OutboxRepository, sender Sender, templates *TemplateRenderer, logger *zap.Logger, rateLimit int, burstSize int) *Worker {
	return &Worker{
		repo:      repo,
		sender:    sender,
		templates: templates,
		logger:    logger,
		limiter:   rate.NewLimiter(rate.Limit(rateLimit), burstSize),
		notify:    make(chan struct{}, 1),
	}
}

// Start begins the worker loop. Call Stop() to shut down.
func (w *Worker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.loop(ctx)
	}()
	w.logger.Info("email worker started")
}

// Stop gracefully shuts down the worker.
func (w *Worker) Stop() {
	w.stopOnce.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		w.wg.Wait()
		w.logger.Info("email worker stopped")
	})
}

// Enqueue sends a non-blocking wake signal to process pending emails.
func (w *Worker) Enqueue() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *Worker) loop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.notify:
			w.processBatch(ctx)
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	emails, err := w.repo.FetchPending(ctx, 20)
	if err != nil {
		if ctx.Err() == nil {
			w.logger.Error("fetch pending emails failed", zap.Error(err))
		}
		return
	}

	for _, email := range emails {
		if err := w.limiter.Wait(ctx); err != nil {
			return // context cancelled
		}
		w.processOne(ctx, email)
	}
}

// ProcessOne processes a single outbox email. Exported for testing.
func (w *Worker) ProcessOne(ctx context.Context, email OutboxEmail) {
	w.processOne(ctx, email)
}

func (w *Worker) processOne(ctx context.Context, email OutboxEmail) {
	if err := w.repo.MarkSending(ctx, email.ID); err != nil {
		w.logger.Error("mark sending failed", zap.String("id", email.ID.String()), zap.Error(err))
		return
	}

	// Render template if needed
	if email.TemplateName != "" && email.BodyHTML == "" {
		html, text, err := w.templates.Render(email.TemplateName, email.TemplateData)
		if err != nil {
			w.logger.Error("template render failed",
				zap.String("template", email.TemplateName),
				zap.Error(err))
			_ = w.repo.MarkFailed(ctx, email.ID, "template error: "+err.Error())
			return
		}
		email.BodyHTML = html
		email.BodyText = text
	}

	// Send with timeout
	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := w.sender.Send(sendCtx, &email)
	if err != nil {
		email.RetryCount++
		if email.RetryCount >= email.MaxRetries {
			_ = w.repo.MarkFailed(ctx, email.ID, err.Error())
			w.logger.Error("email permanently failed",
				zap.String("to", strings.Join(email.ToAddresses, ",")),
				zap.String("subject", email.Subject),
				zap.Int("retries", email.RetryCount),
				zap.Error(err))
		} else {
			// Exponential backoff: 1min, 3min, 9min
			backoff := time.Duration(math.Pow(3, float64(email.RetryCount))) * time.Minute
			_ = w.repo.MarkRetry(ctx, email.ID, err.Error(), time.Now().Add(backoff))
			w.logger.Warn("email retry scheduled",
				zap.String("id", email.ID.String()),
				zap.Int("attempt", email.RetryCount),
				zap.Duration("next_retry_in", backoff))
		}
	} else {
		_ = w.repo.MarkSent(ctx, email.ID)
		w.logger.Info("email sent",
			zap.String("to", strings.Join(email.ToAddresses, ",")),
			zap.String("subject", email.Subject),
			zap.String("event", email.TriggerEvent))
	}
}

// --- Mock repo for testing ---

// MockOutboxRepo is a simple in-memory outbox for tests.
type MockOutboxRepo struct {
	mu     sync.Mutex
	emails map[uuid.UUID]*OutboxEmail
}

func NewMockOutboxRepo() *MockOutboxRepo {
	return &MockOutboxRepo{emails: make(map[uuid.UUID]*OutboxEmail)}
}

func (m *MockOutboxRepo) Insert(_ context.Context, email *OutboxEmail) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if email.ID == uuid.Nil {
		email.ID = uuid.New()
	}
	cp := *email
	m.emails[email.ID] = &cp
	return nil
}

func (m *MockOutboxRepo) FetchPending(_ context.Context, limit int) ([]OutboxEmail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []OutboxEmail
	now := time.Now()
	for _, e := range m.emails {
		if (e.Status == StatusPending || e.Status == StatusRetry) && !e.NextRetryAt.After(now) {
			result = append(result, *e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockOutboxRepo) MarkSending(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.emails[id]; ok {
		e.Status = StatusSending
	}
	return nil
}

func (m *MockOutboxRepo) MarkSent(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.emails[id]; ok {
		e.Status = StatusSent
		now := time.Now()
		e.SentAt = &now
	}
	return nil
}

func (m *MockOutboxRepo) MarkRetry(_ context.Context, id uuid.UUID, errMsg string, nextRetryAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.emails[id]; ok {
		e.Status = StatusRetry
		e.LastError = errMsg
		e.NextRetryAt = nextRetryAt
		e.RetryCount++
	}
	return nil
}

func (m *MockOutboxRepo) MarkFailed(_ context.Context, id uuid.UUID, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.emails[id]; ok {
		e.Status = StatusFailed
		e.LastError = errMsg
		now := time.Now()
		e.FailedAt = &now
	}
	return nil
}

func (m *MockOutboxRepo) GetByID(_ context.Context, id uuid.UUID) (*OutboxEmail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.emails[id]; ok {
		cp := *e
		return &cp, nil
	}
	return nil, nil
}

func (m *MockOutboxRepo) CountByStatus(_ context.Context) (map[string]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]int)
	for _, e := range m.emails {
		result[e.Status]++
	}
	return result, nil
}

func (m *MockOutboxRepo) ListRecent(_ context.Context, limit int) ([]OutboxEmail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []OutboxEmail
	for _, e := range m.emails {
		result = append(result, *e)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *MockOutboxRepo) CountSent24h(_ context.Context) (int, error)    { return 0, nil }
func (m *MockOutboxRepo) CountFailed24h(_ context.Context) (int, error)  { return 0, nil }
func (m *MockOutboxRepo) OldestPendingMinutes(_ context.Context) (float64, error) { return 0, nil }
func (m *MockOutboxRepo) ListByStatus(_ context.Context, status string, limit int) ([]OutboxEmail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []OutboxEmail
	for _, e := range m.emails {
		if e.Status == status {
			result = append(result, *e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetAll returns all emails in the mock repo (for test assertions).
func (m *MockOutboxRepo) GetAll() []*OutboxEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*OutboxEmail
	for _, e := range m.emails {
		result = append(result, e)
	}
	return result
}
