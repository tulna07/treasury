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

// OMOCounterpartyName is the required counterparty name for OMO deals.
const OMOCounterpartyName = "Sở giao dịch NHNN"

// OMORepoDealNotifier is an optional interface for sending notifications on OMO/Repo deal status changes.
type OMORepoDealNotifier interface {
	NotifyDealStatusChange(ctx context.Context, dealModule string, dealID uuid.UUID, ticketNumber, fromStatus, toStatus, actorName string)
	NotifyUser(ctx context.Context, userID uuid.UUID, title, message, category, dealModule string, dealID *uuid.UUID)
}

// OMORepoDealEmailer is an optional interface for sending email notifications on OMO/Repo deal events.
type OMORepoDealEmailer interface {
	SendDealCancelled(ctx context.Context, params email.DealCancelledParams) error
	SendDealVoided(ctx context.Context, params email.DealVoidedParams) error
}

// OMORepoService handles MM OMO/Repo KBNN deal business logic.
type OMORepoService struct {
	repo     repository.MMOMORepoRepository
	userRepo repository.UserRepository
	rbac     *security.RBACChecker
	audit    *audit.Logger
	pool     *pgxpool.Pool
	logger   *zap.Logger
	notifier OMORepoDealNotifier
	emailer  OMORepoDealEmailer
}

// NewOMORepoService creates a new OMO/Repo service.
func NewOMORepoService(repo repository.MMOMORepoRepository, userRepo repository.UserRepository, rbac *security.RBACChecker, auditLogger *audit.Logger, pool *pgxpool.Pool, logger *zap.Logger) *OMORepoService {
	return &OMORepoService{repo: repo, userRepo: userRepo, rbac: rbac, audit: auditLogger, pool: pool, logger: logger}
}

// SetNotifier sets an optional notification service.
func (s *OMORepoService) SetNotifier(n OMORepoDealNotifier) {
	s.notifier = n
}

// SetEmailer sets an optional email service. If nil, email notifications are skipped.
func (s *OMORepoService) SetEmailer(e OMORepoDealEmailer) {
	s.emailer = e
}

// CreateDeal creates a new OMO or Repo KBNN deal.
func (s *OMORepoService) CreateDeal(ctx context.Context, req dto.CreateMMOMORepoRequest, ipAddress, userAgent string) (*dto.MMOMORepoResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	if err := s.validateCreateRequest(&req); err != nil {
		return nil, err
	}

	deal := &model.MMOMORepoDeal{
		DealSubtype:    req.DealSubtype,
		SessionName:    req.SessionName,
		TradeDate:      req.TradeDate,
		CounterpartyID: req.CounterpartyID,
		NotionalAmount: req.NotionalAmount,
		BondCatalogID:  req.BondCatalogID,
		WinningRate:    req.WinningRate,
		TenorDays:      req.TenorDays,
		SettlementDate1: req.SettlementDate1,
		SettlementDate2: req.SettlementDate2,
		HaircutPct:     req.HaircutPct,
		Status:         constants.StatusOpen,
		Note:           req.Note,
		CreatedBy:      userID,
	}

	if err := s.repo.Create(ctx, deal); err != nil {
		s.logger.Error("failed to create OMO/Repo deal", zap.Error(err))
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:      userID,
		FullName:    fullName,
		Department:  department,
		BranchCode:  branchCode,
		Action:      "CREATE_MM_OMO_REPO_DEAL",
		DealModule:  constants.ModuleMMOMORepo,
		DealID:      &deal.ID,
		StatusAfter: constants.StatusOpen,
		NewValues:   s.dealToAuditMap(deal),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})

	s.logger.Info("OMO/Repo deal created",
		zap.String("deal_id", deal.ID.String()),
		zap.String("subtype", deal.DealSubtype),
		zap.String("created_by", userID.String()),
	)

	return s.dealToResponse(deal), nil
}

// GetDeal retrieves a single OMO/Repo deal by ID.
func (s *OMORepoService) GetDeal(ctx context.Context, id uuid.UUID) (*dto.MMOMORepoResponse, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.dealToResponse(deal), nil
}

// ListDeals lists OMO/Repo deals with filters and pagination.
func (s *OMORepoService) ListDeals(ctx context.Context, filter dto.MMOMORepoFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.MMOMORepoResponse], error) {
	roles := ctxutil.GetRoles(ctx)
	userID := ctxutil.GetUserUUID(ctx)

	// Apply exclude_cancelled default
	if filter.Status == nil && filter.Statuses == nil && filter.ExcludeStatuses == nil {
		filter.ExcludeStatuses = &constants.CancelledStatuses
	}

	// Apply role-based data scope
	scopedFilter := s.applyDataScope(roles, userID, filter)
	if scopedFilter == nil {
		result := dto.NewPaginationResponse([]dto.MMOMORepoResponse{}, 0, pag.Page, pag.PageSize)
		return &result, nil
	}

	deals, total, err := s.repo.List(ctx, *scopedFilter, pag)
	if err != nil {
		return nil, err
	}

	var items []dto.MMOMORepoResponse
	for _, d := range deals {
		items = append(items, *s.dealToResponse(&d))
	}
	if items == nil {
		items = []dto.MMOMORepoResponse{}
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	return &result, nil
}

// applyDataScope restricts the filter based on the user's roles.
func (s *OMORepoService) applyDataScope(roles []string, _ uuid.UUID, filter dto.MMOMORepoFilter) *dto.MMOMORepoFilter {
	if len(roles) == 0 {
		return &filter
	}

	// K.NV roles (Dealer, DeskHead, CenterDirector, DivisionHead) → ALL
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

	// Others → no access
	return nil
}

// UpdateDeal updates an existing OMO/Repo deal (only when OPEN).
func (s *OMORepoService) UpdateDeal(ctx context.Context, id uuid.UUID, req dto.UpdateMMOMORepoRequest, ipAddress, userAgent string) (*dto.MMOMORepoResponse, error) {
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
	if req.SessionName != nil {
		existing.SessionName = *req.SessionName
	}
	if req.TradeDate != nil {
		existing.TradeDate = *req.TradeDate
	}
	if req.CounterpartyID != nil {
		existing.CounterpartyID = *req.CounterpartyID
	}
	if req.NotionalAmount != nil {
		existing.NotionalAmount = *req.NotionalAmount
	}
	if req.BondCatalogID != nil {
		existing.BondCatalogID = *req.BondCatalogID
	}
	if req.WinningRate != nil {
		existing.WinningRate = *req.WinningRate
	}
	if req.TenorDays != nil {
		existing.TenorDays = *req.TenorDays
	}
	if req.SettlementDate1 != nil {
		existing.SettlementDate1 = *req.SettlementDate1
	}
	if req.SettlementDate2 != nil {
		existing.SettlementDate2 = *req.SettlementDate2
	}
	if req.HaircutPct != nil {
		existing.HaircutPct = *req.HaircutPct
	}
	if req.Note != nil {
		existing.Note = req.Note
	}

	oldValues := s.dealToAuditMap(existing)

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update OMO/Repo deal", zap.Error(err))
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
		Action:     "UPDATE_MM_OMO_REPO_DEAL",
		DealModule: constants.ModuleMMOMORepo,
		DealID:     &id,
		OldValues:  oldValues,
		NewValues:  s.dealToAuditMap(updated),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return s.dealToResponse(updated), nil
}

// omoApproveTargetStatus determines the target status for OMO/Repo approval.
// Flow: OPEN → PENDING_L2_APPROVAL → PENDING_BOOKING → PENDING_CHIEF_ACCOUNTANT → COMPLETED
func omoApproveTargetStatus(currentStatus, action string) (string, string, error) {
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
			return constants.StatusCompleted, "CHIEF_ACCOUNTANT_APPROVE", nil
		}
		return constants.StatusVoidedByAccounting, "CHIEF_ACCOUNTANT_REJECT", nil

	default:
		return "", "", apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve deal in status %s", currentStatus))
	}
}

// ApproveDeal approves or rejects an OMO/Repo deal.
func (s *OMORepoService) ApproveDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
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

	newStatus, actionType, err := omoApproveTargetStatus(deal.Status, req.Action)
	if err != nil {
		return err
	}

	// Check permission via RBAC transition map
	requiredPerm := security.GetRequiredPermission(constants.ModuleMMOMORepo, deal.Status, newStatus)
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

	auditAction := "APPROVE_MM_OMO_REPO_DEAL"
	if req.Action == "REJECT" {
		auditAction = "REJECT_MM_OMO_REPO_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, omoStatusChangeInfo{
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
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMOMORepo, id, deal.DealNumber, deal.Status, newStatus, actorName)

		if isTerminalOrRejectStatus(newStatus) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Giao dịch OMO/Repo %s đã chuyển trạng thái: %s → %s bởi %s.", deal.DealNumber, deal.Status, newStatus, actorName),
				s.notificationCategory(newStatus),
				constants.ModuleMMOMORepo, &dealIDRef,
			)
		}
	}

	// Send email when accounting rejects (VOIDED_BY_ACCOUNTING) — notify dealer
	if s.emailer != nil && newStatus == constants.StatusVoidedByAccounting {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealVoided(context.Background(), email.DealVoidedParams{
				DealModule:       constants.ModuleMMOMORepo,
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.NotionalAmount.String(),
				Currency:         "VND",
				VoidReason:       derefStr(req.Comment),
				VoidedBy:         approverName,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// RecallDeal recalls an OMO/Repo deal back to OPEN.
func (s *OMORepoService) RecallDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
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
		// Dealer recall: only creator can recall
		if deal.CreatedBy != userID {
			return apperror.New(apperror.ErrForbidden, "only the deal creator can recall")
		}
	} else {
		actionType = "TP_RECALL"
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, targetStatus, userID); err != nil {
		return err
	}

	s.recordStatusChange(ctx, userID, omoStatusChangeInfo{
		dealID:      id,
		auditAction: "RECALL_MM_OMO_REPO_DEAL",
		actionType:  actionType,
		oldStatus:   deal.Status,
		newStatus:   targetStatus,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	return nil
}

// CancelDeal requests cancellation of a completed OMO/Repo deal.
func (s *OMORepoService) CancelDeal(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
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

	actorName := s.recordStatusChange(ctx, userID, omoStatusChangeInfo{
		dealID:      id,
		auditAction: "CANCEL_REQUEST_MM_OMO_REPO_DEAL",
		actionType:  "CANCEL_REQUEST",
		oldStatus:   originalStatus,
		newStatus:   constants.StatusPendingCancelL1,
		reason:      reason,
		ipAddress:   ipAddress,
		userAgent:   userAgent,
	})

	if s.notifier != nil {
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMOMORepo, id, deal.DealNumber, originalStatus, constants.StatusPendingCancelL1, actorName)
	}

	return nil
}

// cancelApproveTargetStatus determines the target status for cancel approval/rejection.
func (s *OMORepoService) cancelApproveTargetStatus(ctx context.Context, dealID uuid.UUID, currentStatus, action string) (newStatus, actionType, requiredPerm string, err error) {
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
			perm:          constants.PermMMOMORepoCancelApproveL1,
		},
		constants.StatusPendingCancelL2: {
			approveStatus: constants.StatusCancelled,
			approveAction: "CANCEL_APPROVE_L2",
			rejectAction:  "CANCEL_REJECT_L2",
			perm:          constants.PermMMOMORepoCancelApproveL2,
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

// ApproveCancelDeal handles L1/L2 cancel approval or rejection for OMO/Repo deals.
func (s *OMORepoService) ApproveCancelDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
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

	auditAction := "CANCEL_APPROVE_MM_OMO_REPO_DEAL"
	if req.Action == "REJECT" {
		auditAction = "CANCEL_REJECT_MM_OMO_REPO_DEAL"
	}

	actorName := s.recordStatusChange(ctx, userID, omoStatusChangeInfo{
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
		s.notifier.NotifyDealStatusChange(ctx, constants.ModuleMMOMORepo, id, deal.DealNumber, deal.Status, newStatus, actorName)

		if newStatus == constants.StatusCancelled || (req.Action == "REJECT" && newStatus != deal.Status) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Yêu cầu hủy giao dịch OMO/Repo %s: %s bởi %s.", deal.DealNumber, newStatus, actorName),
				s.notificationCategory(newStatus),
				constants.ModuleMMOMORepo, &dealIDRef,
			)
		}
	}

	// Send email notification when deal is fully cancelled
	if s.emailer != nil && newStatus == constants.StatusCancelled {
		approverName, _, _ := s.getActorInfo(ctx, userID)
		go func() {
			_ = s.emailer.SendDealCancelled(context.Background(), email.DealCancelledParams{
				DealModule:       constants.ModuleMMOMORepo,
				DealID:           id,
				TicketNumber:     deal.DealNumber,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.NotionalAmount.String(),
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

// GetApprovalHistory returns the approval actions for an OMO/Repo deal.
func (s *OMORepoService) GetApprovalHistory(ctx context.Context, dealID uuid.UUID) ([]dto.ApprovalHistoryEntry, error) {
	if _, err := s.repo.GetByID(ctx, dealID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.action_type, a.status_before, a.status_after,
			a.performed_by, COALESCE(u.full_name, 'Unknown') AS performer_name,
			a.performed_at, COALESCE(a.reason, '') AS reason
		FROM approval_actions a
		LEFT JOIN users u ON u.id = a.performed_by
		WHERE a.deal_module = $1 AND a.deal_id = $2
		ORDER BY a.performed_at ASC`, constants.ModuleMMOMORepo, dealID)
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

// CloneDeal clones a rejected/voided OMO/Repo deal.
func (s *OMORepoService) CloneDeal(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) (*dto.MMOMORepoResponse, error) {
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

	clone := &model.MMOMORepoDeal{
		DealSubtype:     source.DealSubtype,
		SessionName:     source.SessionName,
		TradeDate:       source.TradeDate,
		CounterpartyID:  source.CounterpartyID,
		NotionalAmount:  source.NotionalAmount,
		BondCatalogID:   source.BondCatalogID,
		WinningRate:     source.WinningRate,
		TenorDays:       source.TenorDays,
		SettlementDate1: source.SettlementDate1,
		SettlementDate2: source.SettlementDate2,
		HaircutPct:      source.HaircutPct,
		Status:          constants.StatusOpen,
		Note:            source.Note,
		ClonedFromID:    &id,
		CreatedBy:       userID,
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
		Action:     "CLONE_MM_OMO_REPO_DEAL",
		DealModule: constants.ModuleMMOMORepo,
		DealID:     &clone.ID,
		OldValues:  map[string]string{"source_id": id.String()},
		NewValues:  map[string]string{"clone_id": clone.ID.String()},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("OMO/Repo deal cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
		zap.String("by", userID.String()),
	)

	return s.dealToResponse(clone), nil
}

// SoftDelete soft-deletes an OMO/Repo deal.
func (s *OMORepoService) SoftDelete(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) error {
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
		Action:     "DELETE_MM_OMO_REPO_DEAL",
		DealModule: constants.ModuleMMOMORepo,
		DealID:     &id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return nil
}

// --- helpers ---

func (s *OMORepoService) validateCreateRequest(req *dto.CreateMMOMORepoRequest) error {
	if req.SessionName == "" {
		return apperror.New(apperror.ErrValidation, "session_name is required")
	}
	if req.NotionalAmount.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "notional_amount must be positive")
	}
	if req.TenorDays <= 0 {
		return apperror.New(apperror.ErrValidation, "tenor_days must be positive")
	}
	if !req.SettlementDate2.After(req.SettlementDate1) {
		return apperror.New(apperror.ErrValidation, "settlement_date_2 must be after settlement_date_1")
	}
	if req.CounterpartyID == uuid.Nil {
		return apperror.New(apperror.ErrValidation, "counterparty_id is required")
	}
	if req.BondCatalogID == uuid.Nil {
		return apperror.New(apperror.ErrValidation, "bond_catalog_id is required")
	}
	if req.WinningRate.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "winning_rate must be positive")
	}

	// For OMO subtype: counterparty should be "Sở giao dịch NHNN"
	// We cannot enforce UUID here, so we log a warning. The frontend should pre-select the correct counterparty.
	// Actual enforcement happens at the handler/UI level since we only have the UUID at this point.
	if req.DealSubtype == constants.MMSubtypeOMO {
		s.logger.Debug("OMO deal created — counterparty enforcement relies on frontend pre-selection",
			zap.String("counterparty_id", req.CounterpartyID.String()),
		)
	}

	return nil
}

// omoStatusChangeInfo captures the common parameters for recording a deal status change.
type omoStatusChangeInfo struct {
	dealID      uuid.UUID
	auditAction string
	actionType  string
	oldStatus   string
	newStatus   string
	reason      string
	ipAddress   string
	userAgent   string
}

// recordStatusChange logs an audit entry, inserts an approval action, and logs the event.
// Returns the actor's full name (useful for notifications).
func (s *OMORepoService) recordStatusChange(ctx context.Context, userID uuid.UUID, info omoStatusChangeInfo) string {
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       info.auditAction,
		DealModule:   constants.ModuleMMOMORepo,
		DealID:       &info.dealID,
		StatusBefore: info.oldStatus,
		StatusAfter:  info.newStatus,
		OldValues:    map[string]string{"status": info.oldStatus},
		NewValues:    map[string]string{"status": info.newStatus},
		Reason:       info.reason,
		IPAddress:    info.ipAddress,
		UserAgent:    info.userAgent,
	})
	s.insertApprovalAction(ctx, constants.ModuleMMOMORepo, info.dealID, info.actionType, info.oldStatus, info.newStatus, userID, info.reason)

	s.logger.Info("OMO/Repo deal status changed",
		zap.String("deal_id", info.dealID.String()),
		zap.String("action", info.actionType),
		zap.String("from", info.oldStatus),
		zap.String("to", info.newStatus),
		zap.String("by", userID.String()),
	)

	return fullName
}

func (s *OMORepoService) dealToResponse(deal *model.MMOMORepoDeal) *dto.MMOMORepoResponse {
	return &dto.MMOMORepoResponse{
		ID:               deal.ID,
		DealNumber:       deal.DealNumber,
		DealSubtype:      deal.DealSubtype,
		SessionName:      deal.SessionName,
		TradeDate:        deal.TradeDate,
		CounterpartyID:   deal.CounterpartyID,
		CounterpartyCode: deal.CounterpartyCode,
		CounterpartyName: deal.CounterpartyName,
		BranchCode:       deal.BranchCode,
		BranchName:       deal.BranchName,
		NotionalAmount:   deal.NotionalAmount,
		BondCatalogID:    deal.BondCatalogID,
		BondCode:         deal.BondCode,
		BondIssuer:       deal.BondIssuer,
		BondCouponRate:   deal.BondCouponRate,
		BondMaturityDate: deal.BondMaturityDate,
		WinningRate:      deal.WinningRate,
		TenorDays:        deal.TenorDays,
		SettlementDate1:  deal.SettlementDate1,
		SettlementDate2:  deal.SettlementDate2,
		HaircutPct:       deal.HaircutPct,
		Status:           deal.Status,
		Note:             deal.Note,
		ClonedFromID:     deal.ClonedFromID,
		CancelReason:     deal.CancelReason,
		CreatedBy:        deal.CreatedBy,
		CreatedByName:    deal.CreatedByName,
		CreatedAt:        deal.CreatedAt,
		UpdatedAt:        deal.UpdatedAt,
		Version:          deal.Version,
	}
}

func (s *OMORepoService) dealToAuditMap(deal *model.MMOMORepoDeal) map[string]interface{} {
	return map[string]interface{}{
		"id":               deal.ID.String(),
		"deal_number":      deal.DealNumber,
		"deal_subtype":     deal.DealSubtype,
		"session_name":     deal.SessionName,
		"counterparty_id":  deal.CounterpartyID.String(),
		"notional_amount":  deal.NotionalAmount.String(),
		"winning_rate":     deal.WinningRate.String(),
		"tenor_days":       deal.TenorDays,
		"settlement_date_1": deal.SettlementDate1,
		"settlement_date_2": deal.SettlementDate2,
		"trade_date":       deal.TradeDate,
		"status":           deal.Status,
	}
}

func (s *OMORepoService) getActorInfo(ctx context.Context, userID uuid.UUID) (fullName, department, branchCode string) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "unknown", "", ""
	}
	return user.FullName, user.Department, ""
}

func (s *OMORepoService) insertApprovalAction(ctx context.Context, dealModule string, dealID uuid.UUID, actionType, statusBefore, statusAfter string, performedBy uuid.UUID, reason string) {
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

func (s *OMORepoService) storeCancelMetadata(ctx context.Context, dealID uuid.UUID, originalStatus string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cancel_metadata (deal_id, deal_module, original_status)
		VALUES ($1, $2, $3)
		ON CONFLICT (deal_id, deal_module)
		DO UPDATE SET original_status = $3`,
		dealID, constants.ModuleMMOMORepo, originalStatus)
	if err != nil {
		s.logger.Error("failed to store cancel metadata",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
	}
}

func (s *OMORepoService) getCancelOriginalStatus(ctx context.Context, dealID uuid.UUID) string {
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT original_status FROM cancel_metadata
		WHERE deal_id = $1 AND deal_module = $2`, dealID, constants.ModuleMMOMORepo).Scan(&status)
	if err != nil {
		s.logger.Error("failed to get cancel original status",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
		return ""
	}
	return status
}

func (s *OMORepoService) notificationTitle(status string) string {
	switch status {
	case constants.StatusRejected:
		return "Giao dịch OMO/Repo bị từ chối"
	case constants.StatusCompleted:
		return "Giao dịch OMO/Repo hoàn thành"
	case constants.StatusCancelled:
		return "Giao dịch OMO/Repo đã hủy"
	case constants.StatusVoidedByAccounting:
		return "Giao dịch OMO/Repo bị trả lại từ kế toán"
	default:
		return "Cập nhật giao dịch OMO/Repo"
	}
}

func (s *OMORepoService) notificationCategory(status string) string {
	switch status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting:
		return "MM_OMO_REPO_REJECT"
	case constants.StatusCompleted:
		return "MM_OMO_REPO_COMPLETE"
	case constants.StatusCancelled:
		return "MM_OMO_REPO_CANCEL"
	default:
		return "MM_OMO_REPO_APPROVAL"
	}
}

// --- package-level helpers ---

