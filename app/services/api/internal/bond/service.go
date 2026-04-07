package bond

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

// DealNotifier is an optional interface for sending notifications on deal status changes.
type DealNotifier interface {
	NotifyDealStatusChange(ctx context.Context, dealModule string, dealID uuid.UUID, ticketNumber, fromStatus, toStatus, actorName string)
	NotifyUser(ctx context.Context, userID uuid.UUID, title, message, category, dealModule string, dealID *uuid.UUID)
}

// DealEmailer is an optional interface for sending email notifications on deal events.
type DealEmailer interface {
	SendDealCancelled(ctx context.Context, params email.DealCancelledParams) error
	SendDealVoided(ctx context.Context, params email.DealVoidedParams) error
}

// Service handles Bond deal business logic.
type Service struct {
	repo     repository.BondDealRepository
	userRepo repository.UserRepository
	rbac     *security.RBACChecker
	audit    *audit.Logger
	pool     *pgxpool.Pool
	logger   *zap.Logger
	notifier DealNotifier
	emailer  DealEmailer
}

// SetNotifier sets an optional notification service.
func (s *Service) SetNotifier(n DealNotifier) {
	s.notifier = n
}

// SetEmailer sets an optional email service. If nil, email notifications are skipped.
func (s *Service) SetEmailer(e DealEmailer) {
	s.emailer = e
}

// NewService creates a new Bond service.
func NewService(repo repository.BondDealRepository, userRepo repository.UserRepository, rbac *security.RBACChecker, auditLogger *audit.Logger, pool *pgxpool.Pool, logger *zap.Logger) *Service {
	return &Service{repo: repo, userRepo: userRepo, rbac: rbac, audit: auditLogger, pool: pool, logger: logger}
}

// CreateDeal creates a new Bond deal.
func (s *Service) CreateDeal(ctx context.Context, req dto.CreateBondDealRequest, ipAddress, userAgent string) (*dto.BondDealResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	if err := s.validateCreateRequest(&req); err != nil {
		return nil, err
	}

	// Determine bond code for inventory check
	bondCode := ""
	if req.BondCodeManual != nil {
		bondCode = *req.BondCodeManual
	}
	// For SELL: check inventory before creating
	if req.Direction == constants.BondDirectionSell && bondCode != "" {
		portfolioType := ""
		if req.PortfolioType != nil {
			portfolioType = *req.PortfolioType
		}
		if portfolioType == "" {
			return nil, apperror.New(apperror.ErrValidation, "portfolio_type is required for sell deals")
		}
		available, err := s.repo.CheckInventory(ctx, bondCode, req.BondCategory, portfolioType)
		if err != nil {
			return nil, err
		}
		if req.Quantity > available {
			return nil, apperror.New(apperror.ErrInsufficientInventory,
				fmt.Sprintf("Số lượng tồn kho không đủ. Khả dụng: %d, Yêu cầu: %d", available, req.Quantity))
		}
	}

	// Calculate remaining tenor days
	remainingDays := int(req.MaturityDate.Sub(req.PaymentDate).Hours() / 24)
	if req.RemainingTenorDays > 0 {
		remainingDays = req.RemainingTenorDays
	}

	deal := &model.BondDeal{
		BondCategory:       req.BondCategory,
		TradeDate:          req.TradeDate,
		OrderDate:          req.OrderDate,
		ValueDate:          req.ValueDate,
		Direction:          req.Direction,
		CounterpartyID:     req.CounterpartyID,
		TransactionType:    req.TransactionType,
		TransactionTypeOther: req.TransactionTypeOther,
		BondCatalogID:      req.BondCatalogID,
		BondCodeManual:     req.BondCodeManual,
		Issuer:             req.Issuer,
		CouponRate:         req.CouponRate,
		IssueDate:          req.IssueDate,
		MaturityDate:       req.MaturityDate,
		Quantity:           req.Quantity,
		FaceValue:          req.FaceValue,
		DiscountRate:       req.DiscountRate,
		CleanPrice:         req.CleanPrice,
		SettlementPrice:    req.SettlementPrice,
		TotalValue:         req.TotalValue,
		PortfolioType:      req.PortfolioType,
		PaymentDate:        req.PaymentDate,
		RemainingTenorDays: remainingDays,
		ConfirmationMethod: req.ConfirmationMethod,
		ConfirmationOther:  req.ConfirmationOther,
		ContractPreparedBy: req.ContractPreparedBy,
		Status:             constants.StatusOpen,
		Note:               req.Note,
		CreatedBy:          userID,
	}

	if err := s.repo.Create(ctx, deal); err != nil {
		s.logger.Error("failed to create bond deal", zap.Error(err))
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:      userID,
		FullName:    fullName,
		Department:  department,
		BranchCode:  branchCode,
		Action:      "CREATE_BOND_DEAL",
		DealModule:  "BOND",
		DealID:      &deal.ID,
		StatusAfter: constants.StatusOpen,
		NewValues:   s.dealToAuditMap(deal),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})

	s.logger.Info("bond deal created",
		zap.String("deal_id", deal.ID.String()),
		zap.String("created_by", userID.String()),
	)

	return s.dealToResponse(deal), nil
}

// GetDeal retrieves a single Bond deal by ID.
func (s *Service) GetDeal(ctx context.Context, id uuid.UUID) (*dto.BondDealResponse, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.dealToResponse(deal), nil
}

// ListDeals lists Bond deals with filters and pagination.
func (s *Service) ListDeals(ctx context.Context, filter dto.BondDealListFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.BondDealResponse], error) {
	roles := ctxutil.GetRoles(ctx)
	userID := ctxutil.GetUserUUID(ctx)

	// Apply exclude_cancelled default
	if filter.Status == nil && filter.Statuses == nil && filter.ExcludeStatuses == nil {
		filter.ExcludeStatuses = &constants.CancelledStatuses
	}

	// Apply role-based data scope
	scopedFilter := s.applyDataScope(roles, userID, filter)
	if scopedFilter == nil {
		result := dto.NewPaginationResponse([]dto.BondDealResponse{}, 0, pag.Page, pag.PageSize)
		return &result, nil
	}

	deals, total, err := s.repo.List(ctx, *scopedFilter, pag)
	if err != nil {
		return nil, err
	}

	var items []dto.BondDealResponse
	for _, d := range deals {
		items = append(items, *s.dealToResponse(&d))
	}
	if items == nil {
		items = []dto.BondDealResponse{}
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	return &result, nil
}

// applyDataScope restricts the filter based on the user's roles per BRD v3 §6.2.
func (s *Service) applyDataScope(roles []string, userID uuid.UUID, filter dto.BondDealListFilter) *dto.BondDealListFilter {
	if len(roles) == 0 {
		return &filter
	}

	// K.NV roles (Dealer, DeskHead, CenterDirector, DivisionHead) → ALL Bond deals
	if hasAnyRole(roles, constants.RoleDealer, constants.RoleDeskHead, constants.RoleCenterDirector, constants.RoleDivisionHead) {
		return &filter
	}

	// Admin → ALL
	if hasRole(roles, constants.RoleAdmin) {
		return &filter
	}

	// Accountant/ChiefAccountant → only PENDING_BOOKING and beyond
	if hasAnyRole(roles, constants.RoleAccountant, constants.RoleChiefAccountant) {
		allowedStatuses := []string{
			constants.StatusPendingBooking,
			constants.StatusPendingChiefAccountant,
			constants.StatusCompleted,
			constants.StatusVoidedByAccounting,
			constants.StatusCancelled,
			constants.StatusPendingCancelL1,
			constants.StatusPendingCancelL2,
		}
		filter.Statuses = &allowedStatuses
		return &filter
	}

	// Risk/Settlement → NO access to Bond module
	return nil
}

// UpdateDeal updates an existing Bond deal (only when OPEN).
func (s *Service) UpdateDeal(ctx context.Context, id uuid.UUID, req dto.UpdateBondDealRequest, ipAddress, userAgent string) (*dto.BondDealResponse, error) {
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
	if req.BondCategory != nil {
		existing.BondCategory = *req.BondCategory
	}
	if req.TradeDate != nil {
		existing.TradeDate = *req.TradeDate
	}
	if req.OrderDate != nil {
		existing.OrderDate = req.OrderDate
	}
	if req.ValueDate != nil {
		existing.ValueDate = *req.ValueDate
	}
	if req.Direction != nil {
		existing.Direction = *req.Direction
	}
	if req.CounterpartyID != nil {
		existing.CounterpartyID = *req.CounterpartyID
	}
	if req.TransactionType != nil {
		existing.TransactionType = *req.TransactionType
	}
	if req.TransactionTypeOther != nil {
		existing.TransactionTypeOther = req.TransactionTypeOther
	}
	if req.BondCatalogID != nil {
		existing.BondCatalogID = req.BondCatalogID
	}
	if req.BondCodeManual != nil {
		existing.BondCodeManual = req.BondCodeManual
	}
	if req.Issuer != nil {
		existing.Issuer = *req.Issuer
	}
	if req.CouponRate != nil {
		existing.CouponRate = *req.CouponRate
	}
	if req.IssueDate != nil {
		existing.IssueDate = req.IssueDate
	}
	if req.MaturityDate != nil {
		existing.MaturityDate = *req.MaturityDate
	}
	if req.Quantity != nil {
		existing.Quantity = *req.Quantity
	}
	if req.FaceValue != nil {
		existing.FaceValue = *req.FaceValue
	}
	if req.DiscountRate != nil {
		existing.DiscountRate = *req.DiscountRate
	}
	if req.CleanPrice != nil {
		existing.CleanPrice = *req.CleanPrice
	}
	if req.SettlementPrice != nil {
		existing.SettlementPrice = *req.SettlementPrice
	}
	if req.TotalValue != nil {
		existing.TotalValue = *req.TotalValue
	}
	if req.PortfolioType != nil {
		existing.PortfolioType = req.PortfolioType
	}
	if req.PaymentDate != nil {
		existing.PaymentDate = *req.PaymentDate
	}
	if req.RemainingTenorDays != nil {
		existing.RemainingTenorDays = *req.RemainingTenorDays
	}
	if req.ConfirmationMethod != nil {
		existing.ConfirmationMethod = *req.ConfirmationMethod
	}
	if req.ConfirmationOther != nil {
		existing.ConfirmationOther = req.ConfirmationOther
	}
	if req.ContractPreparedBy != nil {
		existing.ContractPreparedBy = *req.ContractPreparedBy
	}
	if req.Note != nil {
		existing.Note = req.Note
	}

	oldValues := s.dealToAuditMap(existing)

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update bond deal", zap.Error(err))
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
		Action:     "UPDATE_BOND_DEAL",
		DealModule: "BOND",
		DealID:     &id,
		OldValues:  oldValues,
		NewValues:  s.dealToAuditMap(updated),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return s.dealToResponse(updated), nil
}

// approveTargetStatus determines the target status for Bond approval.
// Bond flow: OPEN → PENDING_L2 → PENDING_BOOKING → PENDING_CHIEF_ACCOUNTANT → COMPLETED
// NO risk check, NO settlement step.
func approveTargetStatus(currentStatus, action string) (string, string, error) {
	switch currentStatus {
	case constants.StatusOpen:
		if action == "APPROVE" {
			return constants.StatusPendingL2Approval, "DESK_HEAD_APPROVE", nil
		}
		return "", "", apperror.New(apperror.ErrValidation, "can only approve OPEN deals at this stage")

	case constants.StatusPendingL2Approval:
		if action == "APPROVE" {
			return constants.StatusPendingBooking, "DIRECTOR_APPROVE", nil
		}
		return constants.StatusRejected, "DIRECTOR_REJECT", nil

	case constants.StatusPendingBooking:
		if action == "APPROVE" {
			return constants.StatusPendingChiefAccountant, "ACCOUNTANT_APPROVE", nil
		}
		return constants.StatusVoidedByAccounting, "ACCOUNTANT_REJECT", nil

	case constants.StatusPendingChiefAccountant:
		if action == "APPROVE" {
			// Bond: always COMPLETED (no settlement step)
			return constants.StatusCompleted, "CHIEF_ACCOUNTANT_APPROVE", nil
		}
		return constants.StatusVoidedByAccounting, "CHIEF_ACCOUNTANT_REJECT", nil

	default:
		return "", "", apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve deal in status %s", currentStatus))
	}
}

// cancelApproveTargetStatus determines the target status for cancel approval/rejection.
func (s *Service) cancelApproveTargetStatus(ctx context.Context, dealID uuid.UUID, currentStatus, action string) (newStatus, actionType, requiredPerm string, err error) {
	type levelConfig struct {
		approveStatus string
		approveAction string
		rejectAction  string
		perm          string
	}

	levels := map[string]levelConfig{
		constants.StatusPendingCancelL1: {
			approveStatus: constants.StatusPendingCancelL2,
			approveAction: "CANCEL_APPROVE_L1",
			rejectAction:  "CANCEL_REJECT_L1",
			perm:          constants.PermBondCancelApproveL1,
		},
		constants.StatusPendingCancelL2: {
			approveStatus: constants.StatusCancelled,
			approveAction: "CANCEL_APPROVE_L2",
			rejectAction:  "CANCEL_REJECT_L2",
			perm:          constants.PermBondCancelApproveL2,
		},
	}

	cfg, ok := levels[currentStatus]
	if !ok {
		return "", "", "", apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve/reject cancel for deal in status %s", currentStatus))
	}

	switch action {
	case "APPROVE":
		return cfg.approveStatus, cfg.approveAction, cfg.perm, nil
	case "REJECT":
		orig := s.getCancelOriginalStatus(ctx, dealID)
		if orig == "" {
			orig = constants.StatusCompleted
		}
		return orig, cfg.rejectAction, cfg.perm, nil
	default:
		return "", "", "", apperror.New(apperror.ErrValidation, "action must be APPROVE or REJECT")
	}
}

// statusChangeInfo captures the common parameters for recording a deal status change.
type statusChangeInfo struct {
	dealID      uuid.UUID
	auditAction string
	actionType  string // for approval_actions table
	oldStatus   string
	newStatus   string
	reason      string
	ipAddress   string
	userAgent   string
}

// recordStatusChange logs an audit entry, inserts an approval action, and logs the event.
// Returns the actor's full name (useful for notifications).
func (s *Service) recordStatusChange(ctx context.Context, userID uuid.UUID, info statusChangeInfo) string {
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       info.auditAction,
		DealModule:   "BOND",
		DealID:       &info.dealID,
		StatusBefore: info.oldStatus,
		StatusAfter:  info.newStatus,
		OldValues:    map[string]string{"status": info.oldStatus},
		NewValues:    map[string]string{"status": info.newStatus},
		Reason:       info.reason,
		IPAddress:    info.ipAddress,
		UserAgent:    info.userAgent,
	})
	s.insertApprovalAction(ctx, "BOND", info.dealID, info.actionType, info.oldStatus, info.newStatus, userID, info.reason)

	s.logger.Info("bond deal status changed",
		zap.String("deal_id", info.dealID.String()),
		zap.String("action", info.actionType),
		zap.String("from", info.oldStatus),
		zap.String("to", info.newStatus),
		zap.String("by", userID.String()),
	)

	return fullName
}

// adjustInventory updates bond inventory. When reverse=false (completion), BUY increments
// and SELL decrements. When reverse=true (cancellation), the opposite occurs.
func (s *Service) adjustInventory(ctx context.Context, deal *model.BondDeal, reverse bool, updatedBy uuid.UUID) error {
	bondCode := deal.BondCode()
	if bondCode == "" {
		return nil
	}
	portfolioType := derefStr(deal.PortfolioType)
	if portfolioType == "" {
		return nil
	}

	shouldIncrement := deal.Direction == constants.BondDirectionBuy
	if reverse {
		shouldIncrement = !shouldIncrement
	}

	if shouldIncrement {
		return s.repo.IncrementInventory(ctx, bondCode, deal.BondCategory, portfolioType, deal.Quantity, updatedBy)
	}
	return s.repo.DecrementInventory(ctx, bondCode, deal.BondCategory, portfolioType, deal.Quantity, updatedBy)
}

// ApproveDeal approves or rejects a Bond deal.
func (s *Service) ApproveDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
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

	newStatus, actionType, err := approveTargetStatus(deal.Status, req.Action)
	if err != nil {
		return err
	}

	// Check permission
	requiredPerm := security.GetRequiredPermission(constants.ModuleBond, deal.Status, newStatus)
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

	// On COMPLETED: update inventory
	if newStatus == constants.StatusCompleted {
		if err := s.adjustInventory(ctx, deal, false, userID); err != nil {
			s.logger.Error("failed to update inventory on completion", zap.Error(err))
		}
	}

	auditAction := "APPROVE_BOND_DEAL"
	if req.Action == "REJECT" {
		auditAction = "REJECT_BOND_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
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
		s.notifier.NotifyDealStatusChange(ctx, "BOND", id, deal.DealNumber, deal.Status, newStatus, actorName)

		if isTerminalOrRejectStatus(newStatus) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Giao dịch Bond %s đã chuyển trạng thái: %s → %s bởi %s.", deal.DealNumber, deal.Status, newStatus, actorName),
				s.notificationCategory(newStatus),
				"BOND", &dealIDRef,
			)
		}
	}

	// Send email when KTTC rejects (VOIDED_BY_ACCOUNTING) — notify CV
	if s.emailer != nil && newStatus == constants.StatusVoidedByAccounting {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealVoided(context.Background(), email.DealVoidedParams{
				DealModule:       "BOND",
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.TotalValue.String(),
				Currency:         "VND",
				VoidReason:       derefStr(req.Comment),
				VoidedBy:         approverName,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// RecallDeal recalls a Bond deal.
// CV recall → OPEN, TP recall from PENDING_L2 → back to TP (but bond has no PENDING_TP_REVIEW, so just OPEN).
func (s *Service) RecallDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
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
		auditAction: "RECALL_BOND_DEAL",
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   targetStatus,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	return nil
}

// CancelDeal requests cancellation of a completed Bond deal (2-level cancel approval flow).
func (s *Service) CancelDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
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
	s.storeCancelMetadata(ctx, id, originalStatus)

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		auditAction: "CANCEL_REQUEST_BOND_DEAL",
		actionType:  "CANCEL_REQUEST",
		oldStatus:   originalStatus,
		newStatus:   constants.StatusPendingCancelL1,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, "BOND", id, deal.DealNumber, originalStatus, constants.StatusPendingCancelL1, actorName)
	}

	return nil
}

// ApproveCancelDeal handles L1/L2 cancel approval or rejection for Bond deals.
func (s *Service) ApproveCancelDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	newStatus, actionType, requiredPerm, err := s.cancelApproveTargetStatus(ctx, id, deal.Status, req.Action)
	if err != nil {
		return err
	}

	if !s.rbac.HasAnyPermission(roles, requiredPerm) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions for this cancel approval step")
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, newStatus, userID); err != nil {
		return err
	}

	// If cancelled, reverse inventory
	if newStatus == constants.StatusCancelled {
		if err := s.adjustInventory(ctx, deal, true, userID); err != nil {
			s.logger.Error("failed to reverse inventory on cancel", zap.Error(err))
		}
	}

	auditAction := "CANCEL_APPROVE_BOND_DEAL"
	if req.Action == "REJECT" {
		auditAction = "CANCEL_REJECT_BOND_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, statusChangeInfo{
		dealID:      id,
		auditAction: auditAction,
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   newStatus,
		reason:      derefStr(req.Comment),
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, "BOND", id, deal.DealNumber, deal.Status, newStatus, actorName)

		if newStatus == constants.StatusCancelled || (req.Action == "REJECT" && newStatus != deal.Status) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Yêu cầu hủy giao dịch Bond %s: %s bởi %s.", deal.DealNumber, newStatus, actorName),
				s.notificationCategory(newStatus),
				"BOND", &dealIDRef,
			)
		}
	}

	// Send email notification when deal is fully cancelled — notify P.KTTC
	if s.emailer != nil && newStatus == constants.StatusCancelled {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealCancelled(context.Background(), email.DealCancelledParams{
				DealModule:       "BOND",
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.TotalValue.String(),
				Currency:         "VND",
				CancelReason:     derefStr(req.Comment),
				RequestedBy:      "",
				ApprovedBy:       approverName,
				IsInternational:  false,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// GetApprovalHistory returns the approval actions for a deal.
func (s *Service) GetApprovalHistory(ctx context.Context, dealID uuid.UUID) ([]dto.ApprovalHistoryEntry, error) {
	if _, err := s.repo.GetByID(ctx, dealID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.action_type, a.status_before, a.status_after,
			a.performed_by, COALESCE(u.full_name, 'Unknown') AS performer_name,
			a.performed_at, COALESCE(a.reason, '') AS reason
		FROM approval_actions a
		LEFT JOIN users u ON u.id = a.performed_by
		WHERE a.deal_module = 'BOND' AND a.deal_id = $1
		ORDER BY a.performed_at ASC`, dealID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query approval history")
	}
	defer rows.Close()

	var entries []dto.ApprovalHistoryEntry
	for rows.Next() {
		var e dto.ApprovalHistoryEntry
		if err := rows.Scan(&e.ID, &e.ActionType, &e.StatusBefore, &e.StatusAfter,
			&e.PerformedBy, &e.PerformerName, &e.PerformedAt, &e.Reason); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan approval history")
		}
		entries = append(entries, e)
	}

	if entries == nil {
		entries = []dto.ApprovalHistoryEntry{}
	}

	return entries, nil
}

// CloneDeal clones a rejected/voided Bond deal.
func (s *Service) CloneDeal(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) (*dto.BondDealResponse, error) {
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

	clone := &model.BondDeal{
		BondCategory:       source.BondCategory,
		TradeDate:          source.TradeDate,
		OrderDate:          source.OrderDate,
		ValueDate:          source.ValueDate,
		Direction:          source.Direction,
		CounterpartyID:     source.CounterpartyID,
		TransactionType:    source.TransactionType,
		TransactionTypeOther: source.TransactionTypeOther,
		BondCatalogID:      source.BondCatalogID,
		BondCodeManual:     source.BondCodeManual,
		Issuer:             source.Issuer,
		CouponRate:         source.CouponRate,
		IssueDate:          source.IssueDate,
		MaturityDate:       source.MaturityDate,
		Quantity:           source.Quantity,
		FaceValue:          source.FaceValue,
		DiscountRate:       source.DiscountRate,
		CleanPrice:         source.CleanPrice,
		SettlementPrice:    source.SettlementPrice,
		TotalValue:         source.TotalValue,
		PortfolioType:      source.PortfolioType,
		PaymentDate:        source.PaymentDate,
		RemainingTenorDays: source.RemainingTenorDays,
		ConfirmationMethod: source.ConfirmationMethod,
		ConfirmationOther:  source.ConfirmationOther,
		ContractPreparedBy: source.ContractPreparedBy,
		Status:             constants.StatusOpen,
		Note:               source.Note,
		ClonedFromID:       &id,
		CreatedBy:          userID,
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
		Action:     "CLONE_BOND_DEAL",
		DealModule: "BOND",
		DealID:     &clone.ID,
		OldValues:  map[string]string{"source_id": id.String()},
		NewValues:  map[string]string{"clone_id": clone.ID.String()},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("bond deal cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
		zap.String("by", userID.String()),
	)

	return s.dealToResponse(clone), nil
}

// SoftDelete soft-deletes a Bond deal.
func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) error {
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
		Action:     "DELETE_BOND_DEAL",
		DealModule: "BOND",
		DealID:     &id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return nil
}

// ListInventory returns all inventory items.
func (s *Service) ListInventory(ctx context.Context) ([]dto.BondInventoryResponse, error) {
	items, err := s.repo.ListInventory(ctx)
	if err != nil {
		return nil, err
	}

	var result []dto.BondInventoryResponse
	for _, inv := range items {
		result = append(result, dto.BondInventoryResponse{
			ID:                inv.ID,
			BondCode:          inv.BondCode,
			BondCategory:      inv.BondCategory,
			PortfolioType:     inv.PortfolioType,
			AvailableQuantity: inv.AvailableQuantity,
			AcquisitionDate:   inv.AcquisitionDate,
			AcquisitionPrice:  inv.AcquisitionPrice,
			Version:           inv.Version,
			UpdatedAt:         inv.UpdatedAt,
			CatalogIssuer:     inv.CatalogIssuer,
			CatalogFaceValue:  inv.CatalogFaceValue,
			NominalValue:      inv.NominalValue,
			UpdatedByName:     inv.UpdatedByName,
		})
	}
	if result == nil {
		result = []dto.BondInventoryResponse{}
	}
	return result, nil
}

// --- helpers ---

func (s *Service) validateCreateRequest(req *dto.CreateBondDealRequest) error {
	if req.Quantity <= 0 {
		return apperror.New(apperror.ErrValidation, "quantity must be positive")
	}
	if req.FaceValue.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "face_value must be positive")
	}
	if req.SettlementPrice.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "settlement_price must be positive")
	}
	if req.TotalValue.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "total_value must be positive")
	}
	if req.CounterpartyID == uuid.Nil {
		return apperror.New(apperror.ErrValidation, "counterparty_id is required")
	}
	if req.Issuer == "" {
		return apperror.New(apperror.ErrValidation, "issuer is required")
	}
	// PortfolioType required when BUY
	if req.Direction == constants.BondDirectionBuy && (req.PortfolioType == nil || *req.PortfolioType == "") {
		return apperror.New(apperror.ErrValidation, "portfolio_type is required for BUY deals")
	}
	// MaturityDate must be after PaymentDate
	if !req.MaturityDate.After(req.PaymentDate) {
		return apperror.New(apperror.ErrValidation, "maturity_date must be after payment_date")
	}
	return nil
}

func (s *Service) dealToResponse(deal *model.BondDeal) *dto.BondDealResponse {
	return &dto.BondDealResponse{
		ID:                   deal.ID,
		DealNumber:           deal.DealNumber,
		BondCategory:         deal.BondCategory,
		TradeDate:            deal.TradeDate,
		OrderDate:            deal.OrderDate,
		ValueDate:            deal.ValueDate,
		Direction:            deal.Direction,
		CounterpartyID:       deal.CounterpartyID,
		CounterpartyCode:     deal.CounterpartyCode,
		CounterpartyName:     deal.CounterpartyName,
		TransactionType:      deal.TransactionType,
		TransactionTypeOther: deal.TransactionTypeOther,
		BondCatalogID:        deal.BondCatalogID,
		BondCodeManual:       deal.BondCodeManual,
		BondCodeDisplay:      deal.BondCodeDisplay,
		Issuer:               deal.Issuer,
		CouponRate:           deal.CouponRate,
		IssueDate:            deal.IssueDate,
		MaturityDate:         deal.MaturityDate,
		Quantity:             deal.Quantity,
		FaceValue:            deal.FaceValue,
		DiscountRate:         deal.DiscountRate,
		CleanPrice:           deal.CleanPrice,
		SettlementPrice:      deal.SettlementPrice,
		TotalValue:           deal.TotalValue,
		PortfolioType:        deal.PortfolioType,
		PaymentDate:          deal.PaymentDate,
		RemainingTenorDays:   deal.RemainingTenorDays,
		ConfirmationMethod:   deal.ConfirmationMethod,
		ConfirmationOther:    deal.ConfirmationOther,
		ContractPreparedBy:   deal.ContractPreparedBy,
		Status:               deal.Status,
		Note:                 deal.Note,
		ClonedFromID:         deal.ClonedFromID,
		CancelReason:         deal.CancelReason,
		CreatedBy:            deal.CreatedBy,
		CreatedByName:        deal.CreatedByName,
		CreatedAt:            deal.CreatedAt,
		UpdatedAt:            deal.UpdatedAt,
		Version:              deal.Version,
	}
}

func (s *Service) dealToAuditMap(deal *model.BondDeal) map[string]interface{} {
	m := map[string]interface{}{
		"id":               deal.ID.String(),
		"deal_number":      deal.DealNumber,
		"bond_category":    deal.BondCategory,
		"direction":        deal.Direction,
		"counterparty_id":  deal.CounterpartyID.String(),
		"quantity":         deal.Quantity,
		"settlement_price": deal.SettlementPrice.String(),
		"total_value":      deal.TotalValue.String(),
		"trade_date":       deal.TradeDate,
		"status":           deal.Status,
	}
	return m
}

func (s *Service) getActorInfo(ctx context.Context, userID uuid.UUID) (fullName, department, branchCode string) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "unknown", "", ""
	}
	return user.FullName, user.Department, ""
}

func (s *Service) insertApprovalAction(ctx context.Context, dealModule string, dealID uuid.UUID, actionType, statusBefore, statusAfter string, performedBy uuid.UUID, reason string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO approval_actions (id, deal_module, deal_id, action_type, status_before, status_after, performed_by, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New(), dealModule, dealID, actionType, statusBefore, statusAfter, performedBy, nullableStr(reason),
	)
	if err != nil {
		s.logger.Error("failed to insert approval action",
			zap.String("deal_id", dealID.String()),
			zap.String("action_type", actionType),
			zap.Error(err),
		)
	}
}

func (s *Service) storeCancelMetadata(ctx context.Context, dealID uuid.UUID, originalStatus string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cancel_metadata (deal_id, deal_module, original_status)
		VALUES ($1, 'BOND', $2)
		ON CONFLICT (deal_id, deal_module)
		DO UPDATE SET original_status = $2`,
		dealID, originalStatus)
	if err != nil {
		s.logger.Error("failed to store cancel metadata",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
	}
}

func (s *Service) getCancelOriginalStatus(ctx context.Context, dealID uuid.UUID) string {
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT original_status FROM cancel_metadata
		WHERE deal_id = $1 AND deal_module = 'BOND'`, dealID).Scan(&status)
	if err != nil {
		s.logger.Error("failed to get cancel original status",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
		return ""
	}
	return status
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func hasAnyRole(roles []string, targets ...string) bool {
	for _, r := range roles {
		for _, t := range targets {
			if r == t {
				return true
			}
		}
	}
	return false
}

func isTerminalOrRejectStatus(status string) bool {
	switch status {
	case constants.StatusRejected,
		constants.StatusCompleted,
		constants.StatusCancelled,
		constants.StatusVoidedByAccounting:
		return true
	}
	return false
}

func (s *Service) notificationTitle(status string) string {
	switch status {
	case constants.StatusRejected:
		return "Giao dịch Bond bị từ chối"
	case constants.StatusCompleted:
		return "Giao dịch Bond hoàn thành"
	case constants.StatusCancelled:
		return "Giao dịch Bond đã hủy"
	case constants.StatusVoidedByAccounting:
		return "Giao dịch Bond bị trả lại từ kế toán"
	default:
		return "Cập nhật giao dịch Bond"
	}
}

func (s *Service) notificationCategory(status string) string {
	switch status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting:
		return "BOND_REJECT"
	case constants.StatusCompleted:
		return "BOND_COMPLETE"
	case constants.StatusCancelled:
		return "BOND_CANCEL"
	default:
		return "BOND_APPROVAL"
	}
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

