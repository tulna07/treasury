package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	fxcalc "github.com/kienlongbank/treasury-api/pkg/decimal"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/email"
	"github.com/kienlongbank/treasury-api/pkg/limitcheck"
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
}

// Service handles FX deal business logic.
type Service struct {
	repo           repository.FxDealRepository
	userRepo       repository.UserRepository
	attachmentRepo repository.AttachmentRepository
	rbac           *security.RBACChecker
	audit          *audit.Logger
	pool           *pgxpool.Pool
	logger         *zap.Logger
	limitChecker   *limitcheck.Checker
	notifier       DealNotifier
	emailer        DealEmailer
}

// SetLimitChecker sets an optional credit limit checker. If nil, limit checks are skipped.
func (s *Service) SetLimitChecker(lc *limitcheck.Checker) {
	s.limitChecker = lc
}

// SetNotifier sets an optional notification service. If nil, notifications are skipped.
func (s *Service) SetNotifier(n DealNotifier) {
	s.notifier = n
}

// SetEmailer sets an optional email service. If nil, email notifications are skipped.
func (s *Service) SetEmailer(e DealEmailer) {
	s.emailer = e
}

// SetAttachmentRepo sets an optional attachment repository. If nil, attachment loading is skipped.
func (s *Service) SetAttachmentRepo(r repository.AttachmentRepository) {
	s.attachmentRepo = r
}

// NewService creates a new FX service.
func NewService(repo repository.FxDealRepository, userRepo repository.UserRepository, rbac *security.RBACChecker, auditLogger *audit.Logger, pool *pgxpool.Pool, logger *zap.Logger) *Service {
	return &Service{repo: repo, userRepo: userRepo, rbac: rbac, audit: auditLogger, pool: pool, logger: logger}
}

// CreateDeal creates a new FX deal.
func (s *Service) CreateDeal(ctx context.Context, req dto.CreateFxDealRequest, ipAddress, userAgent string) (*dto.FxDealResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	// Validate
	if err := s.validateCreateRequest(&req); err != nil {
		return nil, err
	}

	// Credit limit check (optional)
	var limitInfo *dto.LimitCheckInfo
	if s.limitChecker != nil {
		result, err := s.limitChecker.CheckFXDeal(ctx, req.CounterpartyID, req.CurrencyCode, req.NotionalAmount, nil)
		if err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "limit check failed")
		}
		if !result.Allowed && !result.RequiresEscalation {
			return nil, apperror.New(apperror.ErrLimitExceeded, fmt.Sprintf(
				"Vượt hạn mức tín dụng đối tác. Hạn mức khả dụng: %s, Yêu cầu: %s",
				result.AvailableLimit.String(), result.RequestedAmount.String(),
			))
		}
		if !result.TotalLimit.IsZero() {
			limitInfo = &dto.LimitCheckInfo{
				TotalLimit:     result.TotalLimit.String(),
				UsedAmount:     result.UsedAmount.String(),
				AvailableLimit: result.AvailableLimit.String(),
				Escalated:      result.RequiresEscalation,
			}
		}
	}

	// Determine pair code and validate rate decimals
	pairCode := s.determinePairCode(req.CurrencyCode, req.Legs)
	if len(req.Legs) > 0 {
		for _, leg := range req.Legs {
			if err := fxcalc.ValidateRateDecimals(leg.ExchangeRate, pairCode); err != nil {
				return nil, apperror.New(apperror.ErrValidation, err.Error())
			}
		}
	}

	// Determine is_international from pay_code_counterparty
	isInternational := req.PayCodeCounterparty != nil && *req.PayCodeCounterparty != ""

	// Calculate settlement amount from first leg rate
	var settlementAmount *decimal.Decimal
	var settlementCurrency *string
	if len(req.Legs) > 0 && pairCode != "" {
		amt, cur, calcErr := fxcalc.CalculateSettlementAmount(req.NotionalAmount, req.Legs[0].ExchangeRate, pairCode)
		if calcErr == nil {
			settlementAmount = &amt
			settlementCurrency = &cur
		}
	}

	// Build domain model
	deal := &model.FxDeal{
		TicketNumber:        req.TicketNumber,
		CounterpartyID:      req.CounterpartyID,
		DealType:            req.DealType,
		Direction:           req.Direction,
		NotionalAmount:      req.NotionalAmount,
		CurrencyCode:        req.CurrencyCode,
		PairCode:            pairCode,
		TradeDate:           req.TradeDate,
		ExecutionDate:       req.ExecutionDate,
		PayCodeKLB:          req.PayCodeKLB,
		PayCodeCounterparty: req.PayCodeCounterparty,
		IsInternational:     isInternational,
		AttachmentPath:      req.AttachmentPath,
		AttachmentName:      req.AttachmentName,
		SettlementAmount:    settlementAmount,
		SettlementCurrency:  settlementCurrency,
		Status:              constants.StatusOpen,
		Note:                req.Note,
		CreatedBy:           userID,
	}

	// Build legs
	for _, legDTO := range req.Legs {
		leg := model.FxDealLeg{
			LegNumber:    legDTO.LegNumber,
			ValueDate:    legDTO.ValueDate,
			ExchangeRate: legDTO.ExchangeRate,
			BuyCurrency:  legDTO.BuyCurrency,
			SellCurrency: legDTO.SellCurrency,
			BuyAmount:    legDTO.BuyAmount,
			SellAmount:   legDTO.SellAmount,
		}
		if legDTO.ExecutionDate != nil {
			leg.ExecutionDate = legDTO.ExecutionDate
		}
		if legDTO.PayCodeKLB != nil {
			leg.PayCodeKLB = legDTO.PayCodeKLB
		}
		if legDTO.PayCodeCounterparty != nil {
			leg.PayCodeCounterparty = legDTO.PayCodeCounterparty
			leg.IsInternational = *legDTO.PayCodeCounterparty != ""
		}
		// Calculate per-leg settlement amount
		if pairCode != "" {
			amt, cur, calcErr := fxcalc.CalculateSettlementAmount(legDTO.SellAmount, legDTO.ExchangeRate, pairCode)
			if calcErr == nil {
				leg.SettlementAmount = &amt
				leg.SettlementCurrency = &cur
			}
		}
		deal.Legs = append(deal.Legs, leg)
	}

	if err := s.repo.Create(ctx, deal); err != nil {
		s.logger.Error("failed to create fx deal", zap.Error(err))
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:      userID,
		FullName:    fullName,
		Department:  department,
		BranchCode:  branchCode,
		Action:      "CREATE_FX_DEAL",
		DealModule:  "FX",
		DealID:      &deal.ID,
		StatusAfter: constants.StatusOpen,
		NewValues:   s.dealToAuditMap(deal),
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})

	s.logger.Info("fx deal created",
		zap.String("deal_id", deal.ID.String()),
		zap.String("created_by", userID.String()),
	)

	resp := s.dealToResponse(deal)
	resp.LimitCheck = limitInfo
	return resp, nil
}

// GetDeal retrieves a single FX deal by ID.
func (s *Service) GetDeal(ctx context.Context, id uuid.UUID) (*dto.FxDealResponse, error) {
	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := s.dealToResponse(deal)
	// Load full attachment list for single deal view
	if s.attachmentRepo != nil {
		atts, err := s.attachmentRepo.ListByDeal(ctx, "FX", id)
		if err == nil && len(atts) > 0 {
			for _, a := range atts {
				resp.Attachments = append(resp.Attachments, dto.AttachmentResponse{
					ID:          a.ID,
					DealModule:  a.DealModule,
					DealID:      a.DealID,
					FileName:    a.FileName,
					FileSize:    a.FileSize,
					ContentType: a.ContentType,
					UploadedBy:  a.UploadedBy,
					CreatedAt:   a.CreatedAt,
					DownloadURL: fmt.Sprintf("/api/v1/attachments/%s/download", a.ID.String()),
				})
			}
		}
	}
	return resp, nil
}

// ListDeals lists FX deals with filters and pagination.
func (s *Service) ListDeals(ctx context.Context, filter repository.FxDealFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.FxDealResponse], error) {
	roles := ctxutil.GetRoles(ctx)
	userID := ctxutil.GetUserUUID(ctx)

	// Apply exclude_cancelled default: if no explicit status filter and ExcludeStatuses not set,
	// hide cancelled/voided deals by default
	if filter.Status == nil && filter.Statuses == nil && filter.ExcludeStatuses == nil {
		filter.ExcludeStatuses = &constants.CancelledStatuses
	}

	// Apply role-based data scope
	scopedFilter := s.applyDataScope(roles, userID, filter)

	// If role has NO access to FX module → return empty
	if scopedFilter == nil {
		result := dto.NewPaginationResponse([]dto.FxDealResponse{}, 0, pag.Page, pag.PageSize)
		return &result, nil
	}

	deals, total, err := s.repo.List(ctx, *scopedFilter, pag)
	if err != nil {
		return nil, err
	}

	var items []dto.FxDealResponse
	for _, d := range deals {
		resp := s.dealToResponse(&d)
		// Include attachment count (not full list) for list view
		if s.attachmentRepo != nil {
			if count, err := s.attachmentRepo.CountByDeal(ctx, "FX", d.ID); err == nil && count > 0 {
				resp.AttachmentCount = &count
			}
		}
		items = append(items, *resp)
	}
	if items == nil {
		items = []dto.FxDealResponse{}
	}

	// Check if cursor mode
	if pag.IsCursorMode() {
		limit := pag.EffectiveLimit()
		hasMore := len(items) > limit
		if hasMore {
			items = items[:limit]
		}
		var nextCursor, prevCursor string
		if hasMore && len(items) > 0 {
			last := items[len(items)-1]
			nextCursor = dto.EncodeCursor(last.ID, last.CreatedAt)
		}
		if pag.Cursor != "" && len(items) > 0 {
			first := items[0]
			prevCursor = dto.EncodeCursor(first.ID, first.CreatedAt)
		}
		result := dto.NewCursorPaginationResponse(items, total, nextCursor, prevCursor, hasMore)
		return &result, nil
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	return &result, nil
}

// applyDataScope restricts the filter based on the user's roles per BRD v3 section 6.2.
func (s *Service) applyDataScope(roles []string, userID uuid.UUID, filter repository.FxDealFilter) *repository.FxDealFilter {
	// If no roles in context (e.g., internal calls, tests without role context), return unfiltered
	if len(roles) == 0 {
		return &filter
	}

	// K.NV roles (Dealer, DeskHead, CenterDirector, DivisionHead) → ALL FX deals
	if hasAnyRole(roles, constants.RoleDealer, constants.RoleDeskHead, constants.RoleCenterDirector, constants.RoleDivisionHead) {
		return &filter // no restriction
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

	// Settlement → only PENDING_SETTLEMENT, today only
	if hasRole(roles, constants.RoleSettlementOfficer) {
		status := constants.StatusPendingSettlement
		filter.Status = &status
		today := time.Now().Format("2006-01-02")
		filter.ToDate = &today
		filter.FromDate = &today
		return &filter
	}

	// Risk roles → NO access to FX (they only see MM)
	if hasAnyRole(roles, constants.RoleRiskOfficer, constants.RoleRiskHead) {
		return nil // no FX access
	}

	// Unknown role → no access
	return nil
}

// UpdateDeal updates an existing FX deal (only when OPEN).
func (s *Service) UpdateDeal(ctx context.Context, id uuid.UUID, req dto.UpdateFxDealRequest, ipAddress, userAgent string) (*dto.FxDealResponse, error) {
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

	// Check ownership
	if existing.CreatedBy != userID {
		return nil, apperror.New(apperror.ErrForbidden, "only the deal creator can edit")
	}

	// Check version for optimistic locking
	if req.Version != existing.Version {
		return nil, apperror.New(apperror.ErrConflict, "deal was modified by another user")
	}

	// Apply partial updates
	if req.TicketNumber != nil {
		existing.TicketNumber = req.TicketNumber
	}
	if req.CounterpartyID != nil {
		existing.CounterpartyID = *req.CounterpartyID
	}
	if req.DealType != nil {
		existing.DealType = *req.DealType
	}
	if req.Direction != nil {
		existing.Direction = *req.Direction
	}
	if req.NotionalAmount != nil {
		existing.NotionalAmount = *req.NotionalAmount
	}
	if req.CurrencyCode != nil {
		existing.CurrencyCode = *req.CurrencyCode
	}
	if req.TradeDate != nil {
		existing.TradeDate = *req.TradeDate
	}
	if req.Note != nil {
		existing.Note = req.Note
	}
	if req.ExecutionDate != nil {
		existing.ExecutionDate = req.ExecutionDate
	}
	if req.PayCodeKLB != nil {
		existing.PayCodeKLB = req.PayCodeKLB
	}
	if req.PayCodeCounterparty != nil {
		existing.PayCodeCounterparty = req.PayCodeCounterparty
		existing.IsInternational = *req.PayCodeCounterparty != ""
	}
	if req.AttachmentPath != nil {
		existing.AttachmentPath = req.AttachmentPath
	}
	if req.AttachmentName != nil {
		existing.AttachmentName = req.AttachmentName
	}

	// Validate rate decimals
	pairCode := existing.PairCode
	if len(req.Legs) > 0 {
		for _, leg := range req.Legs {
			if err := fxcalc.ValidateRateDecimals(leg.ExchangeRate, pairCode); err != nil {
				return nil, apperror.New(apperror.ErrValidation, err.Error())
			}
		}
	}

	// Replace legs if provided
	if len(req.Legs) > 0 {
		var legs []model.FxDealLeg
		for _, legDTO := range req.Legs {
			leg := model.FxDealLeg{
				LegNumber:    legDTO.LegNumber,
				ValueDate:    legDTO.ValueDate,
				ExchangeRate: legDTO.ExchangeRate,
				BuyCurrency:  legDTO.BuyCurrency,
				SellCurrency: legDTO.SellCurrency,
				BuyAmount:    legDTO.BuyAmount,
				SellAmount:   legDTO.SellAmount,
			}
			if legDTO.ExecutionDate != nil {
				leg.ExecutionDate = legDTO.ExecutionDate
			}
			if legDTO.PayCodeKLB != nil {
				leg.PayCodeKLB = legDTO.PayCodeKLB
			}
			if legDTO.PayCodeCounterparty != nil {
				leg.PayCodeCounterparty = legDTO.PayCodeCounterparty
				leg.IsInternational = *legDTO.PayCodeCounterparty != ""
			}
			// Per-leg settlement
			if pairCode != "" {
				amt, cur, calcErr := fxcalc.CalculateSettlementAmount(legDTO.SellAmount, legDTO.ExchangeRate, pairCode)
				if calcErr == nil {
					leg.SettlementAmount = &amt
					leg.SettlementCurrency = &cur
				}
			}
			legs = append(legs, leg)
		}
		existing.Legs = legs
	}

	// Recalculate settlement amount
	if pairCode != "" && len(existing.Legs) > 0 {
		amt, cur, calcErr := fxcalc.CalculateSettlementAmount(existing.NotionalAmount, existing.Legs[0].ExchangeRate, pairCode)
		if calcErr == nil {
			existing.SettlementAmount = &amt
			existing.SettlementCurrency = &cur
		}
	}

	// Credit limit check (optional)
	var limitInfo *dto.LimitCheckInfo
	if s.limitChecker != nil {
		result, err := s.limitChecker.CheckFXDeal(ctx, existing.CounterpartyID, existing.CurrencyCode, existing.NotionalAmount, &id)
		if err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "limit check failed")
		}
		if !result.Allowed && !result.RequiresEscalation {
			return nil, apperror.New(apperror.ErrLimitExceeded, fmt.Sprintf(
				"Vượt hạn mức tín dụng đối tác. Hạn mức khả dụng: %s, Yêu cầu: %s",
				result.AvailableLimit.String(), result.RequestedAmount.String(),
			))
		}
		if !result.TotalLimit.IsZero() {
			limitInfo = &dto.LimitCheckInfo{
				TotalLimit:     result.TotalLimit.String(),
				UsedAmount:     result.UsedAmount.String(),
				AvailableLimit: result.AvailableLimit.String(),
				Escalated:      result.RequiresEscalation,
			}
		}
	}

	// Capture old values for audit before update
	oldValues := s.dealToAuditMap(existing)

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update fx deal", zap.Error(err))
		return nil, err
	}

	// Re-fetch to get updated data
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "UPDATE_FX_DEAL",
		DealModule: "FX",
		DealID:     &id,
		OldValues:  oldValues,
		NewValues:  s.dealToAuditMap(updated),
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	resp := s.dealToResponse(updated)
	resp.LimitCheck = limitInfo
	return resp, nil
}

// approveTargetStatus determines the target status based on the current status and action.
// The deal parameter is needed for TTQT branching (is_international check).
// Returns (newStatus, actionType, error).
func approveTargetStatus(currentStatus, action string, deal *model.FxDeal) (string, string, error) {
	switch currentStatus {
	case constants.StatusOpen:
		if action == "APPROVE" {
			return constants.StatusPendingL2Approval, "DESK_HEAD_APPROVE", nil
		}
		return "", "", apperror.New(apperror.ErrValidation, "can only approve OPEN deals at this stage, not reject")

	case constants.StatusPendingTPReview:
		// TP re-approves after TP recall
		if action == "APPROVE" {
			return constants.StatusPendingL2Approval, "DESK_HEAD_REAPPROVE", nil
		}
		// TP returns to CV
		return constants.StatusOpen, "DESK_HEAD_RETURN_TO_CV", nil

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
			// TTQT branching: if international → PENDING_SETTLEMENT, else → COMPLETED directly
			if deal != nil && !deal.IsInternational {
				return constants.StatusCompleted, "CHIEF_ACCOUNTANT_APPROVE", nil
			}
			return constants.StatusPendingSettlement, "CHIEF_ACCOUNTANT_APPROVE", nil
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

// ApproveDeal approves or rejects a deal.
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

	// Determine target status based on current status and action
	newStatus, actionType, err := approveTargetStatus(deal.Status, req.Action, deal)
	if err != nil {
		return err
	}

	// Check permission for the transition using the centralized permission map
	requiredPerm := security.GetRequiredPermission(constants.ModuleFX, deal.Status, newStatus)
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

	// Determine audit action name
	auditAction := "APPROVE_FX_DEAL"
	if req.Action == "REJECT" {
		auditAction = "REJECT_FX_DEAL"
	}

	// Audit trail + approval action
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       auditAction,
		DealModule:   "FX",
		DealID:       &id,
		StatusBefore: deal.Status,
		StatusAfter:  newStatus,
		OldValues:    map[string]string{"status": deal.Status},
		NewValues:    map[string]string{"status": newStatus},
		Reason:       derefStr(req.Comment),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
	s.insertApprovalAction(ctx, "FX", id, actionType, deal.Status, newStatus, userID, derefStr(req.Comment))

	s.logger.Info("fx deal status changed",
		zap.String("deal_id", id.String()),
		zap.String("action", actionType),
		zap.String("from", deal.Status),
		zap.String("to", newStatus),
		zap.String("by", userID.String()),
		zap.String("required_permission", requiredPerm),
	)

	// Notify relevant users of the status change
	if s.notifier != nil {
		ticketStr := ""
		if deal.TicketNumber != nil {
			ticketStr = *deal.TicketNumber
		}
		actorName := fullName
		s.notifier.NotifyDealStatusChange(ctx, "FX", id, ticketStr, deal.Status, newStatus, actorName)

		// For reject/void/complete statuses, also notify the deal creator
		if isTerminalOrRejectStatus(newStatus) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Giao dịch FX %s đã chuyển trạng thái: %s → %s bởi %s.", ticketStr, deal.Status, newStatus, actorName),
				s.notificationCategory(newStatus),
				"FX", &dealIDRef,
			)
		}
	}

	return nil
}

// RecallDeal recalls a deal.
// BRD: CV recall → OPEN, TP recall from PENDING_L2 → PENDING_TP_REVIEW.
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

	// Determine recall target status: TP recall vs CV recall
	// TP (DeskHead) recalling from PENDING_L2 → PENDING_TP_REVIEW
	// CV (Dealer/creator) recall → OPEN
	targetStatus := constants.StatusOpen
	actionType := "DEALER_RECALL"

	isTPRole := hasAnyRole(roles, constants.RoleDeskHead, constants.RoleCenterDirector, constants.RoleDivisionHead)
	if isTPRole && deal.Status == constants.StatusPendingL2Approval {
		targetStatus = constants.StatusPendingTPReview
		actionType = "TP_RECALL"
	} else {
		// CV recall: only creator can recall
		if deal.CreatedBy != userID {
			return apperror.New(apperror.ErrForbidden, "only the deal creator can recall")
		}
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, targetStatus, userID); err != nil {
		return err
	}

	// Audit trail + approval action
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       "RECALL_FX_DEAL",
		DealModule:   "FX",
		DealID:       &id,
		StatusBefore: deal.Status,
		StatusAfter:  targetStatus,
		OldValues:    map[string]string{"status": deal.Status},
		NewValues:    map[string]string{"status": targetStatus},
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
	s.insertApprovalAction(ctx, "FX", id, actionType, deal.Status, targetStatus, userID, reason)

	s.logger.Info("fx deal recalled",
		zap.String("deal_id", id.String()),
		zap.String("target_status", targetStatus),
		zap.String("reason", reason),
		zap.String("by", userID.String()),
	)

	return nil
}

// CancelDeal requests cancellation of a deal (2-level cancel approval flow).
// Sets status to PENDING_CANCEL_L1, storing the original status in metadata.
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

	// Store original status so we can revert on reject
	s.storeCancelMetadata(ctx, id, originalStatus)

	// Audit trail + approval action
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       "CANCEL_REQUEST_FX_DEAL",
		DealModule:   "FX",
		DealID:       &id,
		StatusBefore: originalStatus,
		StatusAfter:  constants.StatusPendingCancelL1,
		OldValues:    map[string]string{"status": originalStatus},
		NewValues:    map[string]string{"status": constants.StatusPendingCancelL1},
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
	s.insertApprovalAction(ctx, "FX", id, "CANCEL_REQUEST", originalStatus, constants.StatusPendingCancelL1, userID, reason)

	s.logger.Info("fx deal cancel requested",
		zap.String("deal_id", id.String()),
		zap.String("original_status", originalStatus),
		zap.String("reason", reason),
		zap.String("by", userID.String()),
	)

	// Notify relevant approvers about cancel request
	if s.notifier != nil {
		ticketStr := ""
		if deal.TicketNumber != nil {
			ticketStr = *deal.TicketNumber
		}
		actorName, _, _ := s.getActorInfo(ctx, userID)
		s.notifier.NotifyDealStatusChange(ctx, "FX", id, ticketStr, originalStatus, constants.StatusPendingCancelL1, actorName)
	}

	return nil
}

// ApproveCancelDeal handles L1/L2 cancel approval or rejection.
func (s *Service) ApproveCancelDeal(ctx context.Context, id uuid.UUID, req dto.ApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	deal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Determine target status based on current cancel status and action
	var newStatus, actionType, requiredPerm string

	switch deal.Status {
	case constants.StatusPendingCancelL1:
		if req.Action == "APPROVE" {
			newStatus = constants.StatusPendingCancelL2
			actionType = "CANCEL_APPROVE_L1"
			requiredPerm = constants.PermFxCancelApproveL1
		} else if req.Action == "REJECT" {
			// Revert to original status before cancel was requested
			newStatus = s.getCancelOriginalStatus(ctx, id)
			if newStatus == "" {
				newStatus = constants.StatusCompleted // fallback
			}
			actionType = "CANCEL_REJECT_L1"
			requiredPerm = constants.PermFxCancelApproveL1
		} else {
			return apperror.New(apperror.ErrValidation, "action must be APPROVE or REJECT")
		}

	case constants.StatusPendingCancelL2:
		if req.Action == "APPROVE" {
			newStatus = constants.StatusCancelled
			actionType = "CANCEL_APPROVE_L2"
			requiredPerm = constants.PermFxCancelApproveL2
		} else if req.Action == "REJECT" {
			newStatus = s.getCancelOriginalStatus(ctx, id)
			if newStatus == "" {
				newStatus = constants.StatusCompleted
			}
			actionType = "CANCEL_REJECT_L2"
			requiredPerm = constants.PermFxCancelApproveL2
		} else {
			return apperror.New(apperror.ErrValidation, "action must be APPROVE or REJECT")
		}

	default:
		return apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve/reject cancel for deal in status %s", deal.Status))
	}

	// Check permission
	if !s.rbac.HasAnyPermission(roles, requiredPerm) {
		return apperror.New(apperror.ErrForbidden, "insufficient permissions for this cancel approval step")
	}

	if err := s.repo.UpdateStatus(ctx, id, deal.Status, newStatus, userID); err != nil {
		return err
	}

	// Audit trail
	auditAction := "CANCEL_APPROVE_FX_DEAL"
	if req.Action == "REJECT" {
		auditAction = "CANCEL_REJECT_FX_DEAL"
	}

	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       auditAction,
		DealModule:   "FX",
		DealID:       &id,
		StatusBefore: deal.Status,
		StatusAfter:  newStatus,
		OldValues:    map[string]string{"status": deal.Status},
		NewValues:    map[string]string{"status": newStatus},
		Reason:       derefStr(req.Comment),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})
	s.insertApprovalAction(ctx, "FX", id, actionType, deal.Status, newStatus, userID, derefStr(req.Comment))

	s.logger.Info("fx deal cancel action",
		zap.String("deal_id", id.String()),
		zap.String("action", actionType),
		zap.String("from", deal.Status),
		zap.String("to", newStatus),
		zap.String("by", userID.String()),
	)

	// Notify on cancel approval/rejection
	if s.notifier != nil {
		ticketStr := ""
		if deal.TicketNumber != nil {
			ticketStr = *deal.TicketNumber
		}
		actorName, _, _ := s.getActorInfo(ctx, userID)
		s.notifier.NotifyDealStatusChange(ctx, "FX", id, ticketStr, deal.Status, newStatus, actorName)

		// Notify deal creator about final cancel result
		if newStatus == constants.StatusCancelled || (req.Action == "REJECT" && newStatus != deal.Status) {
			dealIDRef := id
			s.notifier.NotifyUser(ctx, deal.CreatedBy,
				s.notificationTitle(newStatus),
				fmt.Sprintf("Yêu cầu hủy giao dịch FX %s: %s bởi %s.", ticketStr, newStatus, actorName),
				s.notificationCategory(newStatus),
				"FX", &dealIDRef,
			)
		}
	}

	// Send email notification when deal is fully cancelled
	if s.emailer != nil && newStatus == constants.StatusCancelled {
		ticketStr := ""
		if deal.TicketNumber != nil {
			ticketStr = *deal.TicketNumber
		}
		cancelRequester := "" // best-effort: use audit trail to find requester
		approverName, _, _ := s.getActorInfo(ctx, userID)

		go func() {
			_ = s.emailer.SendDealCancelled(context.Background(), email.DealCancelledParams{
				DealModule:       "FX",
				DealID:           id,
				TicketNumber:     ticketStr,
				CounterpartyName: deal.CounterpartyName,
				Amount:           deal.NotionalAmount.String(),
				Currency:         deal.CurrencyCode,
				CancelReason:     derefStr(req.Comment),
				RequestedBy:      cancelRequester,
				ApprovedBy:       approverName,
				IsInternational:  deal.IsInternational,
				TriggeredBy:      userID,
			})
		}()
	}

	return nil
}

// GetApprovalHistory returns the combined approval actions and audit log entries for a deal.
func (s *Service) GetApprovalHistory(ctx context.Context, dealID uuid.UUID) ([]dto.ApprovalHistoryEntry, error) {
	// Verify deal exists
	if _, err := s.repo.GetByID(ctx, dealID); err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.action_type, a.status_before, a.status_after,
			a.performed_by, COALESCE(u.full_name, 'Unknown') AS performer_name,
			a.performed_at, COALESCE(a.reason, '') AS reason
		FROM approval_actions a
		LEFT JOIN users u ON u.id = a.performed_by
		WHERE a.deal_module = 'FX' AND a.deal_id = $1
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

// storeCancelMetadata stores the original status before cancel was requested.
func (s *Service) storeCancelMetadata(ctx context.Context, dealID uuid.UUID, originalStatus string) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cancel_metadata (deal_id, deal_module, original_status)
		VALUES ($1, 'FX', $2)
		ON CONFLICT (deal_id, deal_module)
		DO UPDATE SET original_status = $2`,
		dealID, originalStatus)
	if err != nil {
		s.logger.Error("failed to store cancel metadata",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
	}
}

// getCancelOriginalStatus retrieves the original status before cancel was requested.
func (s *Service) getCancelOriginalStatus(ctx context.Context, dealID uuid.UUID) string {
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT original_status FROM cancel_metadata
		WHERE deal_id = $1 AND deal_module = 'FX'`, dealID).Scan(&status)
	if err != nil {
		s.logger.Error("failed to get cancel original status",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
		return ""
	}
	return status
}

// CloneDeal clones a rejected/voided deal.
func (s *Service) CloneDeal(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) (*dto.FxDealResponse, error) {
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

	// Create new deal from source — ticket number is auto-generated (not copied)
	clone := &model.FxDeal{
		TicketNumber:        nil, // auto-generate new ticket number
		CounterpartyID:      source.CounterpartyID,
		DealType:            source.DealType,
		Direction:           source.Direction,
		NotionalAmount:      source.NotionalAmount,
		CurrencyCode:        source.CurrencyCode,
		PairCode:            source.PairCode,
		TradeDate:           source.TradeDate,
		ExecutionDate:       source.ExecutionDate,
		PayCodeKLB:          source.PayCodeKLB,
		PayCodeCounterparty: source.PayCodeCounterparty,
		IsInternational:     source.IsInternational,
		SettlementAmount:    source.SettlementAmount,
		SettlementCurrency:  source.SettlementCurrency,
		Status:              constants.StatusOpen,
		Note:                source.Note,
		CreatedBy:           userID,
	}

	for _, leg := range source.Legs {
		clone.Legs = append(clone.Legs, model.FxDealLeg{
			LegNumber:           leg.LegNumber,
			ValueDate:           leg.ValueDate,
			ExecutionDate:       leg.ExecutionDate,
			ExchangeRate:        leg.ExchangeRate,
			BuyCurrency:         leg.BuyCurrency,
			SellCurrency:        leg.SellCurrency,
			BuyAmount:           leg.BuyAmount,
			SellAmount:          leg.SellAmount,
			PayCodeKLB:          leg.PayCodeKLB,
			PayCodeCounterparty: leg.PayCodeCounterparty,
			IsInternational:     leg.IsInternational,
			SettlementAmount:    leg.SettlementAmount,
			SettlementCurrency:  leg.SettlementCurrency,
		})
	}

	if err := s.repo.Create(ctx, clone); err != nil {
		return nil, err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "CLONE_FX_DEAL",
		DealModule: "FX",
		DealID:     &clone.ID,
		OldValues:  map[string]string{"source_id": id.String()},
		NewValues:  map[string]string{"clone_id": clone.ID.String()},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("fx deal cloned",
		zap.String("source_id", id.String()),
		zap.String("clone_id", clone.ID.String()),
		zap.String("by", userID.String()),
	)

	return s.dealToResponse(clone), nil
}

// SoftDelete soft-deletes a deal.
func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	if err := s.repo.SoftDelete(ctx, id, userID); err != nil {
		return err
	}

	// Audit trail
	fullName, department, branchCode := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:     userID,
		FullName:   fullName,
		Department: department,
		BranchCode: branchCode,
		Action:     "DELETE_FX_DEAL",
		DealModule: "FX",
		DealID:     &id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return nil
}

// --- helpers ---

func (s *Service) validateCreateRequest(req *dto.CreateFxDealRequest) error {
	if req.NotionalAmount.LessThanOrEqual(decimal.Zero) {
		return apperror.New(apperror.ErrValidation, "notional amount must be positive")
	}
	if req.CounterpartyID == uuid.Nil {
		return apperror.New(apperror.ErrValidation, "counterparty_id is required")
	}
	if req.CurrencyCode == "" {
		return apperror.New(apperror.ErrValidation, "currency_code is required")
	}
	if len(req.Legs) == 0 {
		return apperror.New(apperror.ErrValidation, "at least one leg is required")
	}

	// SWAP must have exactly 2 legs
	if req.DealType == constants.FxTypeSwap && len(req.Legs) != 2 {
		return apperror.New(apperror.ErrValidation, "SWAP deal must have exactly 2 legs")
	}
	// SPOT/FORWARD must have exactly 1 leg
	if (req.DealType == constants.FxTypeSpot || req.DealType == constants.FxTypeForward) && len(req.Legs) != 1 {
		return apperror.New(apperror.ErrValidation, "SPOT/FORWARD deal must have exactly 1 leg")
	}

	for _, leg := range req.Legs {
		if leg.ExchangeRate.LessThanOrEqual(decimal.Zero) {
			return apperror.New(apperror.ErrValidation, "exchange rate must be positive")
		}
		if leg.BuyAmount.LessThanOrEqual(decimal.Zero) || leg.SellAmount.LessThanOrEqual(decimal.Zero) {
			return apperror.New(apperror.ErrValidation, "buy and sell amounts must be positive")
		}
	}

	return nil
}

func (s *Service) dealToResponse(deal *model.FxDeal) *dto.FxDealResponse {
	resp := &dto.FxDealResponse{
		ID:                  deal.ID,
		TicketNumber:        deal.TicketNumber,
		CounterpartyID:      deal.CounterpartyID,
		CounterpartyCode:    deal.CounterpartyCode,
		CounterpartyName:    deal.CounterpartyName,
		DealType:            deal.DealType,
		Direction:           deal.Direction,
		NotionalAmount:      deal.NotionalAmount,
		CurrencyCode:        deal.CurrencyCode,
		TradeDate:           deal.TradeDate,
		ExecutionDate:       deal.ExecutionDate,
		PayCodeKLB:          deal.PayCodeKLB,
		PayCodeCounterparty: deal.PayCodeCounterparty,
		IsInternational:     deal.IsInternational,
		AttachmentPath:      deal.AttachmentPath,
		AttachmentName:      deal.AttachmentName,
		SettlementAmount:    deal.SettlementAmount,
		SettlementCurrency:  deal.SettlementCurrency,
		Status:              deal.Status,
		Note:                deal.Note,
		CreatedBy:           deal.CreatedBy,
		CreatedAt:           deal.CreatedAt,
		UpdatedAt:           deal.UpdatedAt,
		Version:             deal.Version,
	}

	for _, leg := range deal.Legs {
		legDTO := dto.FxDealLegDTO{
			LegNumber:           leg.LegNumber,
			ValueDate:           leg.ValueDate,
			ExecutionDate:       leg.ExecutionDate,
			ExchangeRate:        leg.ExchangeRate,
			BuyCurrency:         leg.BuyCurrency,
			SellCurrency:        leg.SellCurrency,
			BuyAmount:           leg.BuyAmount,
			SellAmount:          leg.SellAmount,
			PayCodeKLB:          leg.PayCodeKLB,
			PayCodeCounterparty: leg.PayCodeCounterparty,
			SettlementAmount:    leg.SettlementAmount,
			SettlementCurrency:  leg.SettlementCurrency,
		}
		if leg.IsInternational {
			b := true
			legDTO.IsInternational = &b
		}
		resp.Legs = append(resp.Legs, legDTO)
	}
	if resp.Legs == nil {
		resp.Legs = []dto.FxDealLegDTO{}
	}

	return resp
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

// getActorInfo fetches the user's full name and department for audit logging.
// BranchCode is left empty as the users table stores branch UUID, not the short code.
func (s *Service) getActorInfo(ctx context.Context, userID uuid.UUID) (fullName, department, branchCode string) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "unknown", "", ""
	}
	return user.FullName, user.Department, ""
}

// insertApprovalAction records an approval action for the deal's approval trail.
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

// dealToAuditMap converts a deal to a map suitable for audit log JSONB storage.
func (s *Service) dealToAuditMap(deal *model.FxDeal) map[string]interface{} {
	m := map[string]interface{}{
		"id":               deal.ID.String(),
		"deal_type":        deal.DealType,
		"direction":        deal.Direction,
		"notional_amount":  deal.NotionalAmount.String(),
		"currency_code":    deal.CurrencyCode,
		"pair_code":        deal.PairCode,
		"counterparty_id":  deal.CounterpartyID.String(),
		"trade_date":       deal.TradeDate,
		"status":           deal.Status,
	}
	if deal.TicketNumber != nil {
		m["ticket_number"] = *deal.TicketNumber
	}
	if len(deal.Legs) > 0 {
		legsJSON, _ := json.Marshal(deal.Legs)
		m["legs"] = json.RawMessage(legsJSON)
	}
	return m
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

func (s *Service) determinePairCode(notionalCurrency string, legs []dto.FxDealLegDTO) string {
	if len(legs) == 0 {
		return ""
	}
	// Determine the other currency from the first leg
	otherCurrency := legs[0].BuyCurrency
	if otherCurrency == notionalCurrency {
		otherCurrency = legs[0].SellCurrency
	}

	// Standard pair ordering: major currencies come first as base
	// USD is base against most currencies; EUR is base against USD
	base, quote := notionalCurrency, otherCurrency

	// Apply standard FX pair conventions
	majorOrder := map[string]int{"EUR": 1, "GBP": 2, "AUD": 3, "NZD": 4, "USD": 5, "CAD": 6, "CHF": 7, "JPY": 8, "KRW": 9, "VND": 10}

	baseRank, baseOK := majorOrder[base]
	quoteRank, quoteOK := majorOrder[quote]

	if baseOK && quoteOK && baseRank > quoteRank {
		base, quote = quote, base
	}

	return base + "/" + quote
}

// isTerminalOrRejectStatus returns true for statuses that represent a terminal/reject outcome.
func isTerminalOrRejectStatus(status string) bool {
	switch status {
	case constants.StatusRejected,
		constants.StatusCompleted,
		constants.StatusCancelled,
		constants.StatusVoidedByAccounting,
		constants.StatusVoidedBySettlement,
		constants.StatusVoidedByRisk:
		return true
	}
	return false
}

func (s *Service) notificationTitle(status string) string {
	switch status {
	case constants.StatusRejected:
		return "Giao dịch bị từ chối"
	case constants.StatusCompleted:
		return "Giao dịch hoàn thành"
	case constants.StatusCancelled:
		return "Giao dịch đã hủy"
	case constants.StatusVoidedByAccounting:
		return "Giao dịch bị trả lại từ kế toán"
	case constants.StatusVoidedBySettlement:
		return "Giao dịch bị trả lại từ thanh toán"
	case constants.StatusVoidedByRisk:
		return "Giao dịch bị trả lại từ quản lý rủi ro"
	default:
		return "Cập nhật giao dịch"
	}
}

func (s *Service) notificationCategory(status string) string {
	switch status {
	case constants.StatusRejected, constants.StatusVoidedByAccounting, constants.StatusVoidedBySettlement, constants.StatusVoidedByRisk:
		return "FX_REJECT"
	case constants.StatusCompleted:
		return "FX_SETTLEMENT"
	case constants.StatusCancelled:
		return "FX_CANCEL"
	default:
		return "FX_APPROVAL"
	}
}
