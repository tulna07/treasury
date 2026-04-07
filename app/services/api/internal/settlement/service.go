package settlement

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// Service handles international settlement business logic.
type Service struct {
	repo     repository.InternationalPaymentRepository
	userRepo repository.UserRepository
	rbac     *security.RBACChecker
	audit    *audit.Logger
	logger   *zap.Logger
}

// NewService creates a new settlement service.
func NewService(
	repo repository.InternationalPaymentRepository,
	userRepo repository.UserRepository,
	rbac *security.RBACChecker,
	auditLogger *audit.Logger,
	logger *zap.Logger,
) *Service {
	return &Service{repo: repo, userRepo: userRepo, rbac: rbac, audit: auditLogger, logger: logger}
}

// ListPayments returns a filtered, paginated list of international payments.
func (s *Service) ListPayments(ctx context.Context, filter dto.InternationalPaymentFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.InternationalPaymentResponse], error) {
	roles := ctxutil.GetRoles(ctx)

	// RBAC: only settlement officers + admin can view
	if !s.rbac.HasAnyPermission(roles, constants.PermIntlPaymentView) {
		return nil, apperror.New(apperror.ErrForbidden, "insufficient permissions")
	}

	payments, total, err := s.repo.List(ctx, filter, pag)
	if err != nil {
		return nil, err
	}

	items := make([]dto.InternationalPaymentResponse, len(payments))
	for i, p := range payments {
		items[i] = toPaymentResponse(p)
	}

	totalPages := int(total) / pag.PageSize
	if int(total)%pag.PageSize > 0 {
		totalPages++
	}

	return &dto.PaginationResponse[dto.InternationalPaymentResponse]{
		Data:       items,
		Total:      total,
		Page:       pag.Page,
		PageSize:   pag.PageSize,
		TotalPages: totalPages,
		HasMore:    pag.Page < totalPages,
	}, nil
}

// GetPayment returns a single international payment by ID.
func (s *Service) GetPayment(ctx context.Context, id uuid.UUID) (*dto.InternationalPaymentResponse, error) {
	roles := ctxutil.GetRoles(ctx)
	if !s.rbac.HasAnyPermission(roles, constants.PermIntlPaymentView) {
		return nil, apperror.New(apperror.ErrForbidden, "insufficient permissions")
	}

	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := toPaymentResponse(*payment)
	return &resp, nil
}

// ApprovePayment approves an international payment (BP.TTQT duyệt).
func (s *Service) ApprovePayment(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	if !s.rbac.HasAnyPermission(roles, constants.PermIntlPaymentSettle) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions to approve settlement")
	}

	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if payment.SettlementStatus != "PENDING" {
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve payment in status %s", payment.SettlementStatus))
	}

	if err := s.repo.Approve(ctx, id, userID); err != nil {
		return err
	}

	// Audit log
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		Action:     "SETTLEMENT_APPROVE",
		DealModule: "SETTLEMENT",
		DealID:     &id,
		NewValues:  map[string]interface{}{"settlement_status": "APPROVED"},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("International payment approved",
		zap.String("payment_id", id.String()),
		zap.String("approved_by", userID.String()),
	)

	return nil
}

// RejectPayment rejects an international payment with reason (BP.TTQT từ chối).
func (s *Service) RejectPayment(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	if !s.rbac.HasAnyPermission(roles, constants.PermIntlPaymentSettle) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions to reject settlement")
	}

	if reason == "" {
		return apperror.New(apperror.ErrValidation, "rejection reason is required")
	}

	payment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if payment.SettlementStatus != "PENDING" {
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot reject payment in status %s", payment.SettlementStatus))
	}

	if err := s.repo.Reject(ctx, id, userID, reason); err != nil {
		return err
	}

	// Audit log
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		Action:     "SETTLEMENT_REJECT",
		DealModule: "SETTLEMENT",
		DealID:     &id,
		NewValues:  map[string]interface{}{"settlement_status": "REJECTED", "reason": reason},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("International payment rejected",
		zap.String("payment_id", id.String()),
		zap.String("rejected_by", userID.String()),
		zap.String("reason", reason),
	)

	return nil
}

// toPaymentResponse converts model to DTO response.
func toPaymentResponse(p model.InternationalPayment) dto.InternationalPaymentResponse {
	return dto.InternationalPaymentResponse{
		ID:                 p.ID,
		SourceModule:       p.SourceModule,
		SourceDealID:       p.SourceDealID,
		SourceLegNumber:    p.SourceLegNumber,
		TicketDisplay:      p.TicketDisplay,
		CounterpartyID:     p.CounterpartyID,
		CounterpartyCode:   p.CounterpartyCode,
		CounterpartyName:   p.CounterpartyName,
		DebitAccount:       p.DebitAccount,
		BICCode:            p.BICCode,
		CurrencyCode:       p.CurrencyCode,
		Amount:             p.Amount,
		TransferDate:       p.TransferDate.Format("2006-01-02"),
		CounterpartySSI:    p.CounterpartySSI,
		OriginalTradeDate:  p.OriginalTradeDate.Format("2006-01-02"),
		ApprovedByDivision: p.ApprovedByDivision,
		SettlementStatus:   p.SettlementStatus,
		SettledBy:          p.SettledBy,
		SettledByName:      p.SettledByName,
		SettledAt:          p.SettledAt,
		RejectionReason:    p.RejectionReason,
		CreatedAt:          p.CreatedAt,
	}
}
