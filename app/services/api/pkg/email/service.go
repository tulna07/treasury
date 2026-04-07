package email

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
)

// Service is the high-level email facade for business logic.
type Service struct {
	repo      OutboxRepository
	worker    *Worker
	templates *TemplateRenderer
	config    config.EmailConfig
	logger    *zap.Logger
}

// NewService creates a new email service.
func NewService(repo OutboxRepository, worker *Worker, templates *TemplateRenderer, cfg config.EmailConfig, logger *zap.Logger) *Service {
	return &Service{
		repo:      repo,
		worker:    worker,
		templates: templates,
		config:    cfg,
		logger:    logger,
	}
}

// SendDealCancelled enqueues a deal cancellation notification email.
func (s *Service) SendDealCancelled(ctx context.Context, params DealCancelledParams) error {
	subject := fmt.Sprintf("[Treasury] Hủy giao dịch %s — %s", params.DealModule, params.TicketNumber)

	templateData := map[string]string{
		"DealModule":       params.DealModule,
		"TicketNumber":     params.TicketNumber,
		"CounterpartyName": params.CounterpartyName,
		"Amount":           params.Amount,
		"Currency":         params.Currency,
		"CancelReason":     params.CancelReason,
		"RequestedBy":      params.RequestedBy,
		"ApprovedBy":       params.ApprovedBy,
		"CancelledAt":      time.Now().Format("02/01/2006 15:04:05"),
	}

	// Build recipient list: P.KTTC emails (configurable via env)
	to := []string{"kttc@kienlongbank.com"}
	var cc []string
	if params.IsInternational {
		cc = []string{"ttqt@kienlongbank.com"}
	}

	idempotencyKey := fmt.Sprintf("cancel:%s:%s", params.DealID.String(), "kttc")
	dealID := params.DealID

	email := &OutboxEmail{
		ID:             uuid.New(),
		ToAddresses:    to,
		CCAddresses:    cc,
		FromAddress:    s.config.FromAddress,
		Subject:        subject,
		TemplateName:   "deal_cancelled",
		TemplateData:   templateData,
		DealModule:     params.DealModule,
		DealID:         &dealID,
		TriggerEvent:   "DEAL_CANCELLED",
		TriggeredBy:    params.TriggeredBy,
		Status:         StatusPending,
		RetryCount:     0,
		MaxRetries:     s.config.MaxRetries,
		NextRetryAt:    time.Now(),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.Insert(ctx, email); err != nil {
		s.logger.Error("failed to enqueue cancel email",
			zap.String("deal_id", params.DealID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("cancel email enqueued",
		zap.String("deal_id", params.DealID.String()),
		zap.String("ticket", params.TicketNumber))

	s.worker.Enqueue()
	return nil
}

// SendDealVoided enqueues a deal void notification email (KTTC void).
func (s *Service) SendDealVoided(ctx context.Context, params DealVoidedParams) error {
	subject := fmt.Sprintf("[Treasury] Void giao dịch %s — %s", params.DealModule, params.TicketNumber)

	templateData := map[string]string{
		"DealModule":       params.DealModule,
		"TicketNumber":     params.TicketNumber,
		"CounterpartyName": params.CounterpartyName,
		"Amount":           params.Amount,
		"Currency":         params.Currency,
		"VoidReason":       params.VoidReason,
		"VoidedBy":         params.VoidedBy,
		"VoidedAt":         time.Now().Format("02/01/2006 15:04:05"),
	}

	idempotencyKey := fmt.Sprintf("void:%s:%s", params.DealID.String(), "kttc")
	dealID := params.DealID

	email := &OutboxEmail{
		ID:             uuid.New(),
		ToAddresses:    []string{"kttc@kienlongbank.com"},
		FromAddress:    s.config.FromAddress,
		Subject:        subject,
		TemplateName:   "deal_voided",
		TemplateData:   templateData,
		DealModule:     params.DealModule,
		DealID:         &dealID,
		TriggerEvent:   "DEAL_VOIDED",
		TriggeredBy:    params.TriggeredBy,
		Status:         StatusPending,
		RetryCount:     0,
		MaxRetries:     s.config.MaxRetries,
		NextRetryAt:    time.Now(),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.Insert(ctx, email); err != nil {
		s.logger.Error("failed to enqueue void email",
			zap.String("deal_id", params.DealID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("void email enqueued",
		zap.String("deal_id", params.DealID.String()),
		zap.String("ticket", params.TicketNumber))

	s.worker.Enqueue()
	return nil
}

// SendGeneric enqueues a generic notification email.
func (s *Service) SendGeneric(ctx context.Context, to []string, subject, templateName string, data map[string]string, event string, triggeredBy uuid.UUID) error {
	idempotencyKey := ""
	if event != "" {
		idempotencyKey = fmt.Sprintf("generic:%s:%s:%d", event, triggeredBy.String(), time.Now().UnixNano())
	}

	email := &OutboxEmail{
		ID:             uuid.New(),
		ToAddresses:    to,
		FromAddress:    s.config.FromAddress,
		Subject:        subject,
		TemplateName:   templateName,
		TemplateData:   data,
		TriggerEvent:   event,
		TriggeredBy:    triggeredBy,
		Status:         StatusPending,
		RetryCount:     0,
		MaxRetries:     s.config.MaxRetries,
		NextRetryAt:    time.Now(),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.Insert(ctx, email); err != nil {
		s.logger.Error("failed to enqueue generic email", zap.Error(err))
		return err
	}

	s.worker.Enqueue()
	return nil
}

// Repo returns the outbox repository (for admin health checks).
func (s *Service) Repo() OutboxRepository {
	return s.repo
}
