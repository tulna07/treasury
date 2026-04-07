package mm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/email"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// InterbankDealNotifier is an optional interface for sending notifications on deal status changes.
type InterbankDealNotifier interface {
	NotifyDealStatusChange(ctx context.Context, dealModule string, dealID uuid.UUID, ticketNumber, fromStatus, toStatus, actorName string)
	NotifyUser(ctx context.Context, userID uuid.UUID, title, message, category, dealModule string, dealID *uuid.UUID)
}

// InterbankDealEmailer is an optional interface for sending email notifications on deal events.
type InterbankDealEmailer interface {
	SendDealCancelled(ctx context.Context, params email.DealCancelledParams) error
	SendDealVoided(ctx context.Context, params email.DealVoidedParams) error
}

// InterbankService handles MM interbank deal business logic.
type InterbankService struct {
	serviceBase
	repo     repository.MMInterbankRepository
	notifier InterbankDealNotifier
	emailer  InterbankDealEmailer
}

// SetNotifier sets an optional notification service.
func (s *InterbankService) SetNotifier(n InterbankDealNotifier) {
	s.notifier = n
}

// SetEmailer sets an optional email service. If nil, email notifications are skipped.
func (s *InterbankService) SetEmailer(e InterbankDealEmailer) {
	s.emailer = e
}

// NewInterbankService creates a new MM interbank service.
func NewInterbankService(repo repository.MMInterbankRepository, userRepo repository.UserRepository, rbac *security.RBACChecker, auditLogger *audit.Logger, pool *pgxpool.Pool, logger *zap.Logger) *InterbankService {
	return &InterbankService{
		serviceBase: serviceBase{userRepo: userRepo, rbac: rbac, audit: auditLogger, pool: pool, logger: logger},
		repo:        repo,
	}
}

// CreateDeal creates a new MM interbank deal.
func (s *InterbankService) CreateDeal(ctx context.Context, req dto.CreateMMInterbankRequest, ipAddress, userAgent string) (*dto.MMInterbankResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	if err := s.validateCreateRequest(&req); err != nil {
		return nil, err
	}

	// Calculate tenor, interest, maturity
	tenorDays := int(req.MaturityDate.Sub(req.EffectiveDate).Hours() / 24)

	deal := &model.MMInterbankDeal{
		TicketNumber:                    req.TicketNumber,
		CounterpartyID:                  req.CounterpartyID,
		CurrencyCode:                    req.CurrencyCode,
		InternalSSIID:                   req.InternalSSIID,
		CounterpartySSIID:               req.CounterpartySSIID,
		CounterpartySSIText:             req.CounterpartySSIText,
		Direction:                       req.Direction,
		PrincipalAmount:                 req.PrincipalAmount,
		InterestRate:                    req.InterestRate,
		DayCountConvention:              req.DayCountConvention,
		TradeDate:                       req.TradeDate,
		EffectiveDate:                   req.EffectiveDate,
		TenorDays:                       tenorDays,
		MaturityDate:                    req.MaturityDate,
		HasCollateral:                   req.HasCollateral,
		CollateralCurrency:              req.CollateralCurrency,
		CollateralDescription:           req.CollateralDescription,
		RequiresInternationalSettlement: req.RequiresInternationalSettlement,
		Status:                          constants.StatusOpen,
		Note:                            req.Note,
		CreatedBy:                       userID,
	}

	// Calculate interest and maturity amounts using model methods
	deal.InterestAmount = deal.CalculateInterest()
	deal.MaturityAmount = deal.PrincipalAmount.Add(deal.InterestAmount)

	if err := s.repo.Create(ctx, deal); err != nil {
		s.logger.Error("failed to create MM interbank deal", zap.Error(err))
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:      userID,
		FullName:    fullName,
		Department:  department,
		BranchCode:  branchCode,
		Action:      "CREATE_MM_INTERBANK_DEAL",
		DealModule:  constants.ModuleMMInterbank,
		DealID:      &deal.ID,
		StatusAfter: constants.StatusOpen,
		NewValues:   s.dealToAuditMap(deal),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})

	s.logger.Info("MM interbank deal created",
		zap.String("deal_id", deal.ID.String()),
		zap.String("created_by", userID.String()),
	)

	return s.dealToResponse(deal), nil
}

// GetDeal retrieves a single MM interbank deal by ID.
func (s *InterbankService) GetDeal(ctx context.Context, id uuid.UUID) (*dto.MMInterbankResponse, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.dealToResponse(deal), nil
}

// ListDeals lists MM interbank deals with filters and pagination.
func (s *InterbankService) ListDeals(ctx context.Context, filter dto.MMInterbankFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.MMInterbankResponse], error) {
	roles := ctxutil.GetRoles(ctx)
	userID := ctxutil.GetUserUUID(ctx)

	// Apply exclude_cancelled default
	if filter.Status == nil && filter.Statuses == nil && filter.ExcludeStatuses == nil {
		filter.ExcludeStatuses = &constants.CancelledStatuses
	}

	// Apply role-based data scope
	scopedFilter := s.applyDataScope(roles, userID, filter)
	if scopedFilter == nil {
		result := dto.NewPaginationResponse([]dto.MMInterbankResponse{}, 0, pag.Page, pag.PageSize)
		return &result, nil
	}

	deals, total, err := s.repo.List(ctx, *scopedFilter, pag)
	if err != nil {
		return nil, err
	}

	var items []dto.MMInterbankResponse
	for _, d := range deals {
		items = append(items, *s.dealToResponse(&d))
	}
	if items == nil {
		items = []dto.MMInterbankResponse{}
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	return &result, nil
}

// applyDataScope restricts the filter based on the user's roles per BRD v3 §6.2.
func (s *InterbankService) applyDataScope(roles []string, userID uuid.UUID, filter dto.MMInterbankFilter) *dto.MMInterbankFilter {
	if len(roles) == 0 {
		return &filter
	}

	// K.NV roles (Dealer, DeskHead, CenterDirector, DivisionHead) → ALL MM interbank deals
	if hasAnyRole(roles, constants.RoleDealer, constants.RoleDeskHead, constants.RoleCenterDirector, constants.RoleDivisionHead) {
		return &filter
	}

	// Admin → ALL
	if hasRole(roles, constants.RoleAdmin) {
		return &filter
	}

	// RiskOfficer/RiskHead → only PENDING_RISK_APPROVAL + VOIDED_BY_RISK
	if hasAnyRole(roles, constants.RoleRiskOfficer, constants.RoleRiskHead) {
		allowedStatuses := []string{
			constants.StatusPendingRiskApproval,
			constants.StatusVoidedByRisk,
		}
		filter.Statuses = &allowedStatuses
		return &filter
	}

	// Accountant/ChiefAccountant → PENDING_BOOKING and beyond
	if hasAnyRole(roles, constants.RoleAccountant, constants.RoleChiefAccountant) {
		allowedStatuses := []string{
			constants.StatusPendingBooking,
			constants.StatusPendingChiefAccountant,
			constants.StatusPendingSettlement,
			constants.StatusCompleted,
			constants.StatusVoidedByAccounting,
			constants.StatusCancelled,
			constants.StatusPendingCancelL1,
			constants.StatusPendingCancelL2,
		}
		filter.Statuses = &allowedStatuses
		return &filter
	}

	// SettlementOfficer → only PENDING_SETTLEMENT
	if hasRole(roles, constants.RoleSettlementOfficer) {
		allowedStatuses := []string{
			constants.StatusPendingSettlement,
		}
		filter.Statuses = &allowedStatuses
		return &filter
	}

	// Unknown role → no access
	return nil
}

// UpdateDeal updates an existing MM interbank deal (only when OPEN).
func (s *InterbankService) UpdateDeal(ctx context.Context, id uuid.UUID, req dto.UpdateMMInterbankRequest, ipAddress, userAgent string) (*dto.MMInterbankResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !existing.CanEdit() {
		return nil, apperror.New(apperror.ErrDealLocked, "deal cannot be edited in current status")
	}

	if existing.CreatedBy != userID {
		return nil, apperror.New(apperror.ErrForbidden, "only the deal creator can edit")
	}

	// Apply partial updates
	if req.TicketNumber != nil {
		existing.TicketNumber = req.TicketNumber
	}
	if req.CounterpartyID != nil {
		existing.CounterpartyID = *req.CounterpartyID
	}
	if req.CurrencyCode != nil {
		existing.CurrencyCode = *req.CurrencyCode
	}
	if req.InternalSSIID != nil {
		existing.InternalSSIID = req.InternalSSIID
	}
	if req.CounterpartySSIID != nil {
		existing.CounterpartySSIID = req.CounterpartySSIID
	}
	if req.CounterpartySSIText != nil {
		existing.CounterpartySSIText = req.CounterpartySSIText
	}
	if req.Direction != nil {
		existing.Direction = *req.Direction
	}
	if req.PrincipalAmount != nil {
		existing.PrincipalAmount = *req.PrincipalAmount
	}
	if req.InterestRate != nil {
		existing.InterestRate = *req.InterestRate
	}
	if req.DayCountConvention != nil {
		existing.DayCountConvention = *req.DayCountConvention
	}
	if req.TradeDate != nil {
		existing.TradeDate = *req.TradeDate
	}
	if req.EffectiveDate != nil {
		existing.EffectiveDate = *req.EffectiveDate
	}
	if req.MaturityDate != nil {
		existing.MaturityDate = *req.MaturityDate
	}
	if req.HasCollateral != nil {
		existing.HasCollateral = *req.HasCollateral
	}
	if req.CollateralCurrency != nil {
		existing.CollateralCurrency = req.CollateralCurrency
	}
	if req.CollateralDescription != nil {
		existing.CollateralDescription = req.CollateralDescription
	}
	if req.RequiresInternationalSettlement != nil {
		existing.RequiresInternationalSettlement = *req.RequiresInternationalSettlement
	}
	if req.Note != nil {
		existing.Note = req.Note
	}

	// Recalculate derived fields
	existing.TenorDays = existing.CalculateTenorDays()
	existing.InterestAmount = existing.CalculateInterest()
	existing.MaturityAmount = existing.PrincipalAmount.Add(existing.InterestAmount)

	oldValues := s.dealToAuditMap(existing)

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update MM interbank deal", zap.Error(err))
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "UPDATE_MM_INTERBANK_DEAL",
		DealModule: constants.ModuleMMInterbank,
		DealID:     &id,
		OldValues:  oldValues,
		NewValues:  s.dealToAuditMap(updated),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return s.dealToResponse(updated), nil
}

// interbankApproveTargetStatus determines the target status for MM interbank approval.
// Interbank flow: OPEN → PENDING_TP_REVIEW → PENDING_L2_APPROVAL → PENDING_RISK_APPROVAL
//
//	→ PENDING_BOOKING → PENDING_CHIEF_ACCOUNTANT → [PENDING_SETTLEMENT if TTQT] → COMPLETED
func interbankApproveTargetStatus(currentStatus, action string, requiresTTQT bool) (string, string, error) {
	switch currentStatus {
	case constants.StatusOpen:
		if action == "APPROVE" {
			return constants.StatusPendingTPReview, "DEALER_SUBMIT", nil
		}
		return "", "", apperror.New(apperror.ErrValidation, "can only approve OPEN deals at this stage")

	case constants.StatusPendingTPReview:
		if action == "APPROVE" {
			return constants.StatusPendingL2Approval, "DESK_HEAD_APPROVE", nil
		}
		return constants.StatusOpen, "DESK_HEAD_REJECT", nil

	case constants.StatusPendingL2Approval:
		if action == "APPROVE" {
			return constants.StatusPendingRiskApproval, "DIRECTOR_APPROVE", nil
		}
		return constants.StatusRejected, "DIRECTOR_REJECT", nil

	case constants.StatusPendingRiskApproval:
		if action == "APPROVE" {
			return constants.StatusPendingBooking, "RISK_APPROVE", nil
		}
		return constants.StatusVoidedByRisk, "RISK_REJECT", nil

	case constants.StatusPendingBooking:
		if action == "APPROVE" {
			return constants.StatusPendingChiefAccountant, "ACCOUNTANT_APPROVE", nil
		}
		return constants.StatusVoidedByAccounting, "ACCOUNTANT_REJECT", nil

	case constants.StatusPendingChiefAccountant:
		if action == "APPROVE" {
			if requiresTTQT {
				return constants.StatusPendingSettlement, "CHIEF_ACCOUNTANT_APPROVE", nil
			}
			return constants.StatusCompleted, "CHIEF_ACCOUNTANT_APPROVE", nil
		}
		return constants.StatusVoidedByAccounting, "CHIEF_ACCOUNTANT_REJECT", nil

	case constants.StatusPendingSettlement:
		if action == "APPROVE" {
			return constants.StatusCompleted, "SETTLEMENT_APPROVE", nil
		}
		return constants.StatusVoidedBySettlement, "SETTLEMENT_REJECT", nil

	default:
		return "", "", apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve deal in status %s", currentStatus))
	}
}

// interbankCancelLevels returns the cancel approval level config for interbank deals.
func interbankCancelLevels() map[string]cancelLevelConfig {
	return map[string]cancelLevelConfig{
		constants.StatusPendingCancelL1: {
			approveStatus: constants.StatusPendingCancelL2,
			approveAction: "CANCEL_APPROVE_L1",
			rejectAction:  "CANCEL_REJECT_L1",
			perm:          constants.PermMMInterbankCancelApproveL1,
		},
		constants.StatusPendingCancelL2: {
			approveStatus: constants.StatusCancelled,
			approveAction: "CANCEL_APPROVE_L2",
			rejectAction:  "CANCEL_REJECT_L2",
			perm:          constants.PermMMInterbankCancelApproveL2,
		},
	}
}

// ApproveDeal approves or rejects an MM interbank deal.
func (s *InterbankService) ApproveDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Prevent self-approval
	if deal.CreatedBy == userID {
		return apperror.New(apperror.ErrSelfApproval, "cannot approve your own deal")
	}

	newStatus, actionType, err := interbankApproveTargetStatus(deal.Status, req.Action, deal.RequiresInternationalSettlement)
	if err != nil {
		return err
	}

	// Check permission
	requiredPerm := security.GetRequiredPermission(constants.ModuleMMInterbank, deal.Status, newStatus)
	if requiredPerm == "" {
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("no permission defined for transition %s → %s", deal.Status, newStatus))
	}
	if !s.rbac.HasAnyPermission(roles, requiredPerm) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions for this approval step")
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, newStatus, userID); err != nil {
		return err
	}

	auditAction := "APPROVE_MM_INTERBANK_DEAL"
	if req.Action == "REJECT" {
		auditAction = "REJECT_MM_INTERBANK_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		dealModule:  constants.ModuleMMInterbank,
		auditAction: auditAction,
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   newStatus,
		reason:      derefStr(req.Comment),
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	// Notify
	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMInterbank, id, deal.DealNumber, deal.Status, newStatus, actorName)

		if isTerminalOrRejectStatus(newStatus) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Giao dịch MM Interbank %s đã chuyển trạng thái: %s → %s bởi %s.", deal.DealNumber, deal.Status, newStatus, actorName),
				s.notificationCategory(newStatus),
				constants.ModuleMMInterbank, &dealIDRef,
			)
		}
	}

	// Send email when deal is voided (accounting, risk, or settlement rejection)
	if s.emailer != nil && isVoidedStatus(newStatus) {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealVoided(context.Background(), email.DealVoidedParams{
				DealModule:       constants.ModuleMMInterbank,
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.PrincipalAmount.String(),
				Currency:         deal.CurrencyCode,
				VoidReason:       derefStr(req.Comment),
				VoidedBy:         approverName,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// RecallDeal recalls an MM interbank deal.
// CV recall → OPEN, TP recall from PENDING_L2 → OPEN.
func (s *InterbankService) RecallDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	if reason == "" {
		return apperror.New(apperror.ErrValidation, "reason is required for recall")
	}

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !deal.CanRecall() {
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot recall deal in status %s", deal.Status))
	}

	targetStatus := constants.StatusOpen
	actionType := "DEALER_RECALL"

	isTPRole := hasAnyRole(roles, constants.RoleDeskHead, constants.RoleCenterDirector, constants.RoleDivisionHead)
	if !isTPRole {
		// CV recall: only creator can recall
		if deal.CreatedBy != userID {
			return apperror.New(apperror.ErrForbidden, "only the deal creator can recall")
		}
	} else {
		actionType = "TP_RECALL"
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, targetStatus, userID); err != nil {
		return err
	}

	s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		dealModule:  constants.ModuleMMInterbank,
		auditAction: "RECALL_MM_INTERBANK_DEAL",
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   targetStatus,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	return nil
}

// CancelDeal requests cancellation of a completed MM interbank deal (2-level cancel approval flow).
func (s *InterbankService) CancelDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)

	if reason == "" {
		return apperror.New(apperror.ErrValidation, "reason is required for cancellation")
	}

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !deal.CanCancel() {
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot cancel deal in status %s", deal.Status))
	}

	originalStatus := deal.Status

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, constants.StatusPendingCancelL1, userID); err != nil {
		return err
	}

	// Store cancel fields on the deal
	if err := s.repo.UpdateCancelFields(ctx, id, reason, userID); err != nil {
		s.logger.Error("failed to update cancel fields", zap.Error(err))
	}

	// Store original status for revert on reject
	s.storeCancelMetadata(ctx, constants.ModuleMMInterbank, id, originalStatus)

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		dealModule:  constants.ModuleMMInterbank,
		auditAction: "CANCEL_REQUEST_MM_INTERBANK_DEAL",
		actionType:  "CANCEL_REQUEST",
		oldStatus:   originalStatus,
		newStatus:   constants.StatusPendingCancelL1,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMInterbank, id, deal.DealNumber, originalStatus, constants.StatusPendingCancelL1, actorName)
	}

	return nil
}

// ApproveCancelDeal handles L1/L2 cancel approval or rejection for MM interbank deals.
func (s *InterbankService) ApproveCancelDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	newStatus, actionType, requiredPerm, err := s.cancelApproveTargetStatus(ctx, constants.ModuleMMInterbank, id, deal.Status, req.Action, interbankCancelLevels())
	if err != nil {
		return err
	}

	if !s.rbac.HasAnyPermission(roles, requiredPerm) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions for this cancel approval step")
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, newStatus, userID); err != nil {
		return err
	}

	auditAction := "CANCEL_APPROVE_MM_INTERBANK_DEAL"
	if req.Action == "REJECT" {
		auditAction = "CANCEL_REJECT_MM_INTERBANK_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		dealModule:  constants.ModuleMMInterbank,
		auditAction: auditAction,
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   newStatus,
		reason:      derefStr(req.Comment),
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMInterbank, id, deal.DealNumber, deal.Status, newStatus, actorName)

		if newStatus == constants.StatusCancelled || (req.Action == "REJECT" && newStatus != deal.Status) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Yêu cầu hủy giao dịch MM Interbank %s: %s bởi %s.", deal.DealNumber, newStatus, actorName),
				s.notificationCategory(newStatus),
				constants.ModuleMMInterbank, &dealIDRef,
			)
		}
	}

	// Send email notification when deal is fully cancelled — notify P.KTTC
	if s.emailer != nil && newStatus == constants.StatusCancelled {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealCancelled(context.Background(), email.DealCancelledParams{
				DealModule:       constants.ModuleMMInterbank,
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.PrincipalAmount.String(),
				Currency:         deal.CurrencyCode,
				CancelReason:     derefStr(req.Comment),
				RequestedBy:      "",
				ApprovedBy:       approverName,
				IsInternational:  deal.RequiresInternationalSettlement,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// GetApprovalHistory returns the approval actions for an MM interbank deal.
func (s *InterbankService) GetApprovalHistory(ctx context.Context, dealID uuid.UUID) ([]dto.ApprovalHistoryEntry, error) {
	if _, err := s.repo.GetByID(ctx, dealID); err != nil {
		return nil, err
	}
	return s.getApprovalHistory(ctx, constants.ModuleMMInterbank, dealID)
}

// CloneDeal clones a rejected/voided MM interbank deal.
func (s *InterbankService) CloneDeal(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) (*dto.MMInterbankResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !source.CanClone() {
		return nil, apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot clone deal in status %s", source.Status))
	}

	clone := &model.MMInterbankDeal{
		TicketNumber:                    source.TicketNumber,
		CounterpartyID:                  source.CounterpartyID,
		CurrencyCode:                    source.CurrencyCode,
		InternalSSIID:                   source.InternalSSIID,
		CounterpartySSIID:               source.CounterpartySSIID,
		CounterpartySSIText:             source.CounterpartySSIText,
		Direction:                       source.Direction,
		PrincipalAmount:                 source.PrincipalAmount,
		InterestRate:                    source.InterestRate,
		DayCountConvention:              source.DayCountConvention,
		TradeDate:                       source.TradeDate,
		EffectiveDate:                   source.EffectiveDate,
		TenorDays:                       source.TenorDays,
		MaturityDate:                    source.MaturityDate,
		InterestAmount:                  source.InterestAmount,
		MaturityAmount:                  source.MaturityAmount,
		HasCollateral:                   source.HasCollateral,
		CollateralCurrency:              source.CollateralCurrency,
		CollateralDescription:           source.CollateralDescription,
		RequiresInternationalSettlement: source.RequiresInternationalSettlement,
		Status:                          constants.StatusOpen,
		Note:                            source.Note,
		ClonedFromID:                    &id,
		CreatedBy:                       userID,
	}

	if err := s.repo.Create(ctx, clone); err != nil {
		return nil, err
	}

	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "CLONE_MM_INTERBANK_DEAL",
		DealModule: constants.ModuleMMInterbank,
		DealID:     &clone.ID,
		OldValues:  map[string]string{"source_id": id.String()},
		NewValues:  map[string]string{"clone_id": clone.ID.String()},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("MM interbank deal cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
		zap.String("by", userID.String()),
	)

	return s.dealToResponse(clone), nil
}

// SoftDelete soft-deletes an MM interbank deal.
func (s *InterbankService) SoftDelete(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	if err := s.repo.SoftDelete(ctx, id, userID); err != nil {
		return err
	}

	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "DELETE_MM_INTERBANK_DEAL",
		DealModule: constants.ModuleMMInterbank,
		DealID:     &id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return nil
}

// --- helpers ---

func (s *InterbankService) validateCreateRequest(req *dto.CreateMMInterbankRequest) error {
	if req.CounterpartyID == uuid.Nil {
		return apperror.New(apperror.ErrValidation, "counterparty_id is required")
	}
	if req.PrincipalAmount.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "principal_amount must be positive")
	}
	if req.InterestRate.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "interest_rate must be positive")
	}
	if !req.MaturityDate.After(req.EffectiveDate) {
		return apperror.New(apperror.ErrValidation, "maturity_date must be after effective_date")
	}
	validDirection := false
	for _, d := range constants.AllMMInterbankDirections {
		if req.Direction == d {
			validDirection = true
			break
		}
	}
	if !validDirection {
		return apperror.New(apperror.ErrValidation, "direction must be one of: PLACE, TAKE, LEND, BORROW")
	}
	validDayCount := false
	for _, dc := range constants.AllDayCountConventions {
		if req.DayCountConvention == dc {
			validDayCount = true
			break
		}
	}
	if !validDayCount {
		return apperror.New(apperror.ErrValidation, "day_count_convention must be one of: ACT_365, ACT_360, ACT_ACT")
	}
	return nil
}

func (s *InterbankService) dealToResponse(deal *model.MMInterbankDeal) *dto.MMInterbankResponse {
	return &dto.MMInterbankResponse{
		ID:                              deal.ID,
		DealNumber:                      deal.DealNumber,
		TicketNumber:                    deal.TicketNumber,
		CounterpartyID:                  deal.CounterpartyID,
		CounterpartyCode:                deal.CounterpartyCode,
		CounterpartyName:                deal.CounterpartyName,
		BranchCode:                      deal.BranchCode,
		BranchName:                      deal.BranchName,
		CurrencyCode:                    deal.CurrencyCode,
		InternalSSIID:                   deal.InternalSSIID,
		CounterpartySSIID:               deal.CounterpartySSIID,
		CounterpartySSIText:             deal.CounterpartySSIText,
		Direction:                       deal.Direction,
		PrincipalAmount:                 deal.PrincipalAmount,
		InterestRate:                    deal.InterestRate,
		DayCountConvention:              deal.DayCountConvention,
		TradeDate:                       deal.TradeDate,
		EffectiveDate:                   deal.EffectiveDate,
		TenorDays:                       deal.TenorDays,
		MaturityDate:                    deal.MaturityDate,
		InterestAmount:                  deal.InterestAmount,
		MaturityAmount:                  deal.MaturityAmount,
		HasCollateral:                   deal.HasCollateral,
		CollateralCurrency:              deal.CollateralCurrency,
		CollateralDescription:           deal.CollateralDescription,
		RequiresInternationalSettlement: deal.RequiresInternationalSettlement,
		Status:                          deal.Status,
		Note:                            deal.Note,
		ClonedFromID:                    deal.ClonedFromID,
		CancelReason:                    deal.CancelReason,
		CreatedBy:                       deal.CreatedBy,
		CreatedByName:                   deal.CreatedByName,
		CreatedAt:                       deal.CreatedAt,
		UpdatedAt:                       deal.UpdatedAt,
		Version:                         deal.Version,
	}
}

func (s *InterbankService) dealToAuditMap(deal *model.MMInterbankDeal) map[string]interface{} {
	m := map[string]interface{}{
		"id":               deal.ID.String(),
		"deal_number":      deal.DealNumber,
		"direction":        deal.Direction,
		"counterparty_id":  deal.CounterpartyID.String(),
		"currency_code":    deal.CurrencyCode,
		"principal_amount": deal.PrincipalAmount.String(),
		"interest_rate":    deal.InterestRate.String(),
		"interest_amount":  deal.InterestAmount.String(),
		"maturity_amount":  deal.MaturityAmount.String(),
		"tenor_days":       deal.TenorDays,
		"effective_date":   deal.EffectiveDate,
		"maturity_date":    deal.MaturityDate,
		"status":           deal.Status,
	}
	return m
}

func (s *InterbankService) notificationTitle(status string) string {
	switch status {
	case constants.StatusRejected:
		return "Giao dịch MM Interbank bị từ chối"
	case constants.StatusCompleted:
		return "Giao dịch MM Interbank hoàn thành"
	case constants.StatusCancelled:
		return "Giao dịch MM Interbank đã hủy"
	case constants.StatusVoidedByAccounting:
		return "Giao dịch MM Interbank bị trả lại từ kế toán"
	case constants.StatusVoidedByRisk:
		return "Giao dịch MM Interbank bị trả lại từ quản lý rủi ro"
	case constants.StatusVoidedBySettlement:
		return "Giao dịch MM Interbank bị trả lại từ thanh toán quốc tế"
	default:
		return "Cập nhật giao dịch MM Interbank"
	}
}

func (s *InterbankService) notificationCategory(status string) string {
	switch status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting, constants.StatusVoidedByRisk, constants.StatusVoidedBySettlement:
		return "MM_INTERBANK_REJECT"
	case constants.StatusCompleted:
		return "MM_INTERBANK_COMPLETE"
	case constants.StatusCancelled:
		return "MM_INTERBANK_CANCEL"
	default:
		return "MM_INTERBANK_APPROVAL"
	}
}

// isVoidedStatus returns true for any voided-by-* status.
func isVoidedStatus(status string) bool {
	switch status {
	case constants.StatusVoidedByAccounting,
		constants.StatusVoidedByRisk,
		constants.StatusVoidedBySettlement:
		return true
	}
	return false
}
