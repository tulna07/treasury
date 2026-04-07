package creditlimit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

// Service handles credit limit business logic.
type Service struct {
	repo     Repository
	userRepo repository.UserRepository
	cpRepo   repository.CounterpartyRepository
	rbac     *security.RBACChecker
	audit    *audit.Logger
	logger   *zap.Logger
}

// NewService creates a new credit limit service.
func NewService(
	repo Repository,
	userRepo repository.UserRepository,
	cpRepo repository.CounterpartyRepository,
	rbac *security.RBACChecker,
	auditLogger *audit.Logger,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
		cpRepo:   cpRepo,
		rbac:     rbac,
		audit:    auditLogger,
		logger:   logger,
	}
}

// ─── Credit Limit CRUD ───

// SetLimit creates or updates a credit limit (SCD Type 2).
func (s *Service) SetLimit(ctx context.Context, req dto.SetCreditLimitRequest, ipAddress, userAgent string) (*dto.CreditLimitResponse, error) {
	userID := ctxutil.GetUserUUID(ctx)
	if userID == uuid.Nil {
		return nil, apperror.New(apperror.ErrUnauthorized, "user not authenticated")
	}

	// Validate: if not unlimited, must have amount
	if !req.IsUnlimited && (req.LimitAmount == nil || req.LimitAmount.LessThanOrEqual(decimal.Zero)) {
		return nil, apperror.New(apperror.ErrValidation, "limit_amount is required when is_unlimited is false")
	}

	// Verify counterparty exists
	cp, err := s.cpRepo.GetByID(ctx, req.CounterpartyID)
	if err != nil {
		return nil, apperror.New(apperror.ErrValidation, "counterparty not found")
	}

	limit := &model.CreditLimit{
		CounterpartyID:    req.CounterpartyID,
		LimitType:         req.LimitType,
		LimitAmount:       req.LimitAmount,
		IsUnlimited:       req.IsUnlimited,
		EffectiveFrom:     req.EffectiveFrom,
		ExpiryDate:        req.ExpiryDate,
		ApprovalReference: req.ApprovalReference,
		CreatedBy:         &userID,
		UpdatedBy:         &userID,
	}

	if req.IsUnlimited {
		limit.LimitAmount = nil
	}

	if err := s.repo.SetLimit(ctx, limit); err != nil {
		return nil, err
	}

	// Audit
	actor := s.getActorInfo(ctx, userID)
	s.audit.Log(ctx, audit.Entry{
		UserID:      userID,
		FullName:    actor.fullName,
		Department:  actor.department,
		BranchCode:  actor.branchCode,
		Action:      "SET_CREDIT_LIMIT",
		DealModule:  "CREDIT_LIMIT",
		DealID:      &limit.ID,
		StatusAfter: "ACTIVE",
		NewValues: map[string]interface{}{
			"counterparty_id":    req.CounterpartyID,
			"limit_type":         req.LimitType,
			"limit_amount":       req.LimitAmount,
			"is_unlimited":       req.IsUnlimited,
			"approval_reference": req.ApprovalReference,
		},
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	return s.buildLimitResponse(limit, cp), nil
}

// GetLimit returns the current credit limit for a counterparty and type.
func (s *Service) GetLimit(ctx context.Context, counterpartyID uuid.UUID, limitType string) (*dto.CreditLimitResponse, error) {
	limit, err := s.repo.GetCurrentLimit(ctx, counterpartyID, limitType)
	if err != nil {
		return nil, err
	}

	cp, _ := s.cpRepo.GetByID(ctx, counterpartyID)
	return s.buildLimitResponse(limit, cp), nil
}

// GetLimitByID returns a credit limit by ID.
func (s *Service) GetLimitByID(ctx context.Context, id uuid.UUID) (*dto.CreditLimitResponse, error) {
	limit, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	cp, _ := s.cpRepo.GetByID(ctx, limit.CounterpartyID)
	return s.buildLimitResponse(limit, cp), nil
}

// ListLimits returns paginated current limits.
func (s *Service) ListLimits(ctx context.Context, filter dto.CreditLimitListFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.CreditLimitResponse], error) {
	limits, total, err := s.repo.ListCurrentLimits(ctx, filter, pag)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.CreditLimitResponse, 0, len(limits))
	// Cache counterparty lookups
	cpCache := map[uuid.UUID]*model.Counterparty{}
	for _, l := range limits {
		cp, ok := cpCache[l.CounterpartyID]
		if !ok {
			cp, _ = s.cpRepo.GetByID(ctx, l.CounterpartyID)
			cpCache[l.CounterpartyID] = cp
		}
		responses = append(responses, *s.buildLimitResponse(&l, cp))
	}

	result := dto.NewPaginationResponse(responses, total, pag.Page, pag.PageSize)
	return &result, nil
}

// GetLimitsByCounterparty returns both COLLATERALIZED and UNCOLLATERALIZED limits for a counterparty.
func (s *Service) GetLimitsByCounterparty(ctx context.Context, counterpartyID uuid.UUID) ([]dto.CreditLimitResponse, error) {
	cp, err := s.cpRepo.GetByID(ctx, counterpartyID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "counterparty not found")
	}

	var responses []dto.CreditLimitResponse
	for _, lt := range []string{model.LimitTypeCollateralized, model.LimitTypeUncollateralized} {
		limit, err := s.repo.GetCurrentLimit(ctx, counterpartyID, lt)
		if err != nil {
			continue // no limit for this type
		}
		responses = append(responses, *s.buildLimitResponse(limit, cp))
	}
	return responses, nil
}

// ─── Utilization ───

// GetUtilization calculates current utilization for a counterparty and limit type.
func (s *Service) GetUtilization(ctx context.Context, counterpartyID uuid.UUID) ([]dto.UtilizationBreakdown, error) {
	cp, err := s.cpRepo.GetByID(ctx, counterpartyID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "counterparty not found")
	}

	today := time.Now()
	fxUtil, _ := s.repo.SumFXUtilization(ctx, counterpartyID, today)
	bondUtil, _ := s.repo.SumBondUtilization(ctx, counterpartyID, today)

	cpName := ""
	if cp.ShortName != nil {
		cpName = *cp.ShortName
	}

	var results []dto.UtilizationBreakdown

	for _, lt := range []string{model.LimitTypeCollateralized, model.LimitTypeUncollateralized} {
		limit, err := s.repo.GetCurrentLimit(ctx, counterpartyID, lt)
		if err != nil {
			continue
		}

		breakdown := dto.UtilizationBreakdown{
			CounterpartyID:   counterpartyID,
			CounterpartyName: cpName,
			LimitType:        lt,
			LimitAmount:      limit.LimitAmount,
			IsUnlimited:      limit.IsUnlimited,
			FXUtilized:       fxUtil,
		}

		// COLLATERALIZED: MM (collateral=true) + FX
		// UNCOLLATERALIZED: MM (no collateral) + Bond + FX
		// MM doesn't exist yet → 0
		if lt == model.LimitTypeUncollateralized {
			breakdown.BondUtilized = bondUtil
		}

		breakdown.TotalUtilized = breakdown.MMUtilized.Add(breakdown.BondUtilized).Add(breakdown.FXUtilized)
		breakdown.Remaining = limit.RemainingAmount(breakdown.TotalUtilized)

		results = append(results, breakdown)
	}

	return results, nil
}

// ─── Deal Approval (CV QLRR → TPB QLRR) ───

// ApproveDealRiskOfficer handles CV QLRR approving a deal.
func (s *Service) ApproveDealRiskOfficer(ctx context.Context, req dto.LimitApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	if !s.rbac.HasAnyPermission(roles, constants.PermCreditLimitApproveRiskL1) {
		return apperror.New(apperror.ErrForbidden, "insufficient permission for risk L1 approval")
	}

	rec, err := s.repo.GetApprovalByDeal(ctx, req.DealModule, req.DealID)
	if err != nil {
		return apperror.New(apperror.ErrNotFound, "no approval record found for this deal")
	}

	if rec.ApprovalStatus != model.LimitApprovalPending {
		return apperror.New(apperror.ErrInvalidTransition, "deal is not pending approval")
	}

	if req.Action == "APPROVE" {
		if err := s.repo.UpdateApprovalStatus(ctx, rec.ID, model.LimitApprovalRiskL1Done, &userID, nil, nil); err != nil {
			return err
		}
	} else {
		if err := s.repo.UpdateApprovalStatus(ctx, rec.ID, model.LimitApprovalRejected, &userID, nil, req.Comment); err != nil {
			return err
		}
	}

	// Save snapshot
	s.saveApprovalSnapshot(ctx, rec, &userID)

	// Audit
	actor := s.getActorInfo(ctx, userID)
	action := "APPROVE_LIMIT_RISK_L1"
	if req.Action == "REJECT" {
		action = "REJECT_LIMIT_RISK_L1"
	}
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     actor.fullName,
		Department:   actor.department,
		BranchCode:   actor.branchCode,
		Action:       action,
		DealModule:   rec.DealModule,
		DealID:       &rec.DealID,
		StatusBefore: model.LimitApprovalPending,
		StatusAfter:  req.Action,
		Reason:       ptrToStr(req.Comment),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	return nil
}

// ApproveDealRiskHead handles TPB QLRR approving a deal.
func (s *Service) ApproveDealRiskHead(ctx context.Context, req dto.LimitApprovalRequest, ipAddress, userAgent string) error {
	userID := ctxutil.GetUserUUID(ctx)
	roles := ctxutil.GetRoles(ctx)

	if !s.rbac.HasAnyPermission(roles, constants.PermCreditLimitApproveRiskL2) {
		return apperror.New(apperror.ErrForbidden, "insufficient permission for risk L2 approval")
	}

	rec, err := s.repo.GetApprovalByDeal(ctx, req.DealModule, req.DealID)
	if err != nil {
		return apperror.New(apperror.ErrNotFound, "no approval record found for this deal")
	}

	if rec.ApprovalStatus != model.LimitApprovalRiskL1Done {
		return apperror.New(apperror.ErrInvalidTransition, "deal has not been approved by risk officer yet")
	}

	if req.Action == "APPROVE" {
		if err := s.repo.UpdateApprovalStatus(ctx, rec.ID, model.LimitApprovalApproved, nil, &userID, nil); err != nil {
			return err
		}
	} else {
		if err := s.repo.UpdateApprovalStatus(ctx, rec.ID, model.LimitApprovalRejected, nil, &userID, req.Comment); err != nil {
			return err
		}
	}

	// Audit
	actor := s.getActorInfo(ctx, userID)
	action := "APPROVE_LIMIT_RISK_L2"
	if req.Action == "REJECT" {
		action = "REJECT_LIMIT_RISK_L2"
	}
	s.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     actor.fullName,
		Department:   actor.department,
		BranchCode:   actor.branchCode,
		Action:       action,
		DealModule:   rec.DealModule,
		DealID:       &rec.DealID,
		StatusBefore: model.LimitApprovalRiskL1Done,
		StatusAfter:  req.Action,
		Reason:       ptrToStr(req.Comment),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	return nil
}

// ListApprovals returns paginated approval records.
func (s *Service) ListApprovals(ctx context.Context, filter dto.LimitApprovalListFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.LimitApprovalResponse], error) {
	records, total, err := s.repo.ListApprovalRecords(ctx, filter, pag)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.LimitApprovalResponse, 0, len(records))
	cpCache := map[uuid.UUID]*model.Counterparty{}
	for _, rec := range records {
		cp, ok := cpCache[rec.CounterpartyID]
		if !ok {
			cp, _ = s.cpRepo.GetByID(ctx, rec.CounterpartyID)
			cpCache[rec.CounterpartyID] = cp
		}

		resp := dto.LimitApprovalResponse{
			ID:                    rec.ID,
			DealModule:            rec.DealModule,
			DealID:                rec.DealID,
			CounterpartyID:        rec.CounterpartyID,
			LimitType:             rec.LimitType,
			DealAmountVND:         rec.DealAmountVND,
			LimitSnapshot:         rec.LimitSnapshot,
			RiskOfficerApprovedBy: rec.RiskOfficerApprovedBy,
			RiskOfficerApprovedAt: rec.RiskOfficerApprovedAt,
			RiskHeadApprovedBy:    rec.RiskHeadApprovedBy,
			RiskHeadApprovedAt:    rec.RiskHeadApprovedAt,
			ApprovalStatus:        rec.ApprovalStatus,
			RejectionReason:       rec.RejectionReason,
			CreatedAt:             rec.CreatedAt,
		}
		if cp != nil {
			if cp.ShortName != nil {
				resp.CounterpartyName = *cp.ShortName
			}
		}
		responses = append(responses, resp)
	}

	result := dto.NewPaginationResponse(responses, total, pag.Page, pag.PageSize)
	return &result, nil
}

// ─── Daily Summary (BRD §3.4.4) ───

// GetDailySummary builds the 11-column daily summary table.
func (s *Service) GetDailySummary(ctx context.Context, date time.Time) (*dto.DailySummaryResponse, error) {
	bases, err := s.repo.GetDailySummaryCounterparties(ctx)
	if err != nil {
		return nil, err
	}

	var rows []dto.DailySummaryRow
	for _, base := range bases {
		row := dto.DailySummaryRow{
			CounterpartyID:              base.CounterpartyID,
			CIFCode:                     base.CIF,
			AllocatedCollateralized:     base.AllocatedCollateralized,
			IsUnlimitedCollateralized:   base.IsUnlimitedCollateralized,
			AllocatedUncollateralized:   base.AllocatedUncollateralized,
			IsUnlimitedUncollateralized: base.IsUnlimitedUncollateralized,
		}
		if base.CounterpartyName != nil {
			row.CounterpartyName = *base.CounterpartyName
		}

		// Get snapshots for opening utilization
		for _, lt := range []string{model.LimitTypeCollateralized, model.LimitTypeUncollateralized} {
			snap, _ := s.repo.GetLatestSnapshot(ctx, base.CounterpartyID, lt, date)

			// Calculate real-time intraday
			fxUtil, _ := s.repo.SumFXUtilization(ctx, base.CounterpartyID, date)
			bondUtil := decimal.Zero
			if lt == model.LimitTypeUncollateralized {
				bondUtil, _ = s.repo.SumBondUtilization(ctx, base.CounterpartyID, date)
			}

			opening := decimal.Zero
			if snap != nil {
				opening = snap.UtilizedTotal
			}

			intraday := fxUtil.Add(bondUtil).Sub(opening)
			if intraday.LessThan(decimal.Zero) {
				intraday = decimal.Zero
			}

			if lt == model.LimitTypeCollateralized {
				row.UsedOpeningCollateralized = opening
				row.UsedIntradayCollateralized = intraday
				if !base.IsUnlimitedCollateralized && base.AllocatedCollateralized != nil {
					rem := base.AllocatedCollateralized.Sub(opening).Sub(intraday)
					row.RemainingCollateralized = &rem
				}
			} else {
				row.UsedOpeningUncollateralized = opening
				row.UsedIntradayUncollateralized = intraday
				if !base.IsUnlimitedUncollateralized && base.AllocatedUncollateralized != nil {
					rem := base.AllocatedUncollateralized.Sub(opening).Sub(intraday)
					row.RemainingUncollateralized = &rem
				}
			}
		}

		rows = append(rows, row)
	}

	return &dto.DailySummaryResponse{
		Date: date.Format("2006-01-02"),
		Rows: rows,
	}, nil
}

// ─── Helpers ───

type actorInfo struct {
	fullName   string
	department string
	branchCode string
}

func (s *Service) getActorInfo(ctx context.Context, userID uuid.UUID) actorInfo {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return actorInfo{}
	}
	return actorInfo{
		fullName:   user.FullName,
		department: user.Department,
		branchCode: user.BranchID,
	}
}

func (s *Service) buildLimitResponse(limit *model.CreditLimit, cp *model.Counterparty) *dto.CreditLimitResponse {
	resp := &dto.CreditLimitResponse{
		ID:                limit.ID,
		CounterpartyID:    limit.CounterpartyID,
		LimitType:         limit.LimitType,
		LimitAmount:       limit.LimitAmount,
		IsUnlimited:       limit.IsUnlimited,
		EffectiveFrom:     limit.EffectiveFrom.Format("2006-01-02"),
		IsCurrent:         limit.IsCurrent,
		ApprovalReference: limit.ApprovalReference,
		CreatedAt:         limit.CreatedAt,
		CreatedBy:         limit.CreatedBy,
		UpdatedAt:         limit.UpdatedAt,
	}
	if limit.EffectiveTo != nil {
		s := limit.EffectiveTo.Format("2006-01-02")
		resp.EffectiveTo = &s
	}
	if limit.ExpiryDate != nil {
		s := limit.ExpiryDate.Format("2006-01-02")
		resp.ExpiryDate = &s
	}
	if cp != nil {
		if cp.ShortName != nil {
			resp.CounterpartyName = *cp.ShortName
		} else {
			resp.CounterpartyName = cp.FullName
		}
		resp.CIFCode = cp.CIF
	}
	return resp
}

func (s *Service) saveApprovalSnapshot(ctx context.Context, rec *model.LimitApprovalRecord, userID *uuid.UUID) {
	today := time.Now()
	limit, err := s.repo.GetCurrentLimit(ctx, rec.CounterpartyID, rec.LimitType)
	if err != nil {
		return
	}

	fxUtil, _ := s.repo.SumFXUtilization(ctx, rec.CounterpartyID, today)
	bondUtil := decimal.Zero
	if rec.LimitType == model.LimitTypeUncollateralized {
		bondUtil, _ = s.repo.SumBondUtilization(ctx, rec.CounterpartyID, today)
	}
	totalUtil := fxUtil.Add(bondUtil)

	snap := &model.LimitUtilizationSnapshot{
		CounterpartyID:   rec.CounterpartyID,
		SnapshotDate:     today,
		LimitType:        rec.LimitType,
		LimitGranted:     limit.LimitAmount,
		UtilizedOpening:  decimal.Zero,
		UtilizedIntraday: totalUtil,
		UtilizedTotal:    totalUtil,
		Remaining:        limit.RemainingAmount(totalUtil),
		BreakdownDetail: map[string]interface{}{
			"fx_utilized":   fxUtil.String(),
			"bond_utilized": bondUtil.String(),
			"mm_utilized":   "0",
		},
		CreatedBy: userID,
	}

	if err := s.repo.CreateSnapshot(ctx, snap); err != nil {
		s.logger.Error("failed to save utilization snapshot",
			zap.String("counterparty_id", rec.CounterpartyID.String()),
			zap.Error(err),
		)
	}
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─── Export ───

// ExportDailySummary generates the Excel export for daily summary.
// Returns column headers and row data for the export engine.
func (s *Service) ExportDailySummary(ctx context.Context, date time.Time) ([]string, [][]interface{}, error) {
	summary, err := s.GetDailySummary(ctx, date)
	if err != nil {
		return nil, nil, err
	}

	headers := []string{
		"Đối tác",
		"CIF",
		"Hạn mức có TSBĐ",
		"Đã SD đầu ngày (có TSBĐ)",
		"SD trong ngày (có TSBĐ)",
		"Còn lại (có TSBĐ)",
		"Hạn mức không TSBĐ",
		"Đã SD đầu ngày (không TSBĐ)",
		"SD trong ngày (không TSBĐ)",
		"Còn lại (không TSBĐ)",
	}

	var dataRows [][]interface{}
	for _, row := range summary.Rows {
		collAlloc := formatLimitAmount(row.AllocatedCollateralized, row.IsUnlimitedCollateralized)
		uncollAlloc := formatLimitAmount(row.AllocatedUncollateralized, row.IsUnlimitedUncollateralized)
		collRemain := formatRemaining(row.RemainingCollateralized, row.IsUnlimitedCollateralized)
		uncollRemain := formatRemaining(row.RemainingUncollateralized, row.IsUnlimitedUncollateralized)

		dataRows = append(dataRows, []interface{}{
			row.CounterpartyName,
			row.CIFCode,
			collAlloc,
			row.UsedOpeningCollateralized.StringFixed(2),
			row.UsedIntradayCollateralized.StringFixed(2),
			collRemain,
			uncollAlloc,
			row.UsedOpeningUncollateralized.StringFixed(2),
			row.UsedIntradayUncollateralized.StringFixed(2),
			uncollRemain,
		})
	}

	return headers, dataRows, nil
}

func formatLimitAmount(amount *decimal.Decimal, isUnlimited bool) string {
	if isUnlimited {
		return "Không giới hạn"
	}
	if amount == nil {
		return "—"
	}
	return amount.StringFixed(2)
}

func formatRemaining(remaining *decimal.Decimal, isUnlimited bool) string {
	if isUnlimited {
		return "Không giới hạn"
	}
	if remaining == nil {
		return "—"
	}
	return fmt.Sprintf("%s", remaining.StringFixed(2))
}
