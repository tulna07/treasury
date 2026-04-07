package mm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// serviceBase holds dependencies and methods shared by InterbankService and OMORepoService.
type serviceBase struct {
	userRepo repository.UserRepository
	rbac     *security.RBACChecker
	audit    *audit.Logger
	pool     *pgxpool.Pool
	logger   *zap.Logger
}

// statusChangeInfo captures the common parameters for recording a deal status change.
type statusChangeInfo struct {
	dealID      uuid.UUID
	dealModule  string
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
func (b *serviceBase) recordStatusChange(ctx context.Context, userID uuid.UUID, info statusChangeInfo) string {
	fullName, department, branchCode := b.getActorInfo(ctx, userID)
	b.audit.Log(ctx, audit.Entry{
		UserID:       userID,
		FullName:     fullName,
		Department:   department,
		BranchCode:   branchCode,
		Action:       info.auditAction,
		DealModule:   info.dealModule,
		DealID:       &info.dealID,
		StatusBefore: info.oldStatus,
		StatusAfter:  info.newStatus,
		OldValues:    map[string]string{"status": info.oldStatus},
		NewValues:    map[string]string{"status": info.newStatus},
		Reason:       info.reason,
		IPAddress:    info.ipAddress,
		UserAgent:    info.userAgent,
	})
	b.insertApprovalAction(ctx, info.dealModule, info.dealID, info.actionType, info.oldStatus, info.newStatus, userID, info.reason)

	b.logger.Info("deal status changed",
		zap.String("module", info.dealModule),
		zap.String("deal_id", info.dealID.String()),
		zap.String("action", info.actionType),
		zap.String("from", info.oldStatus),
		zap.String("to", info.newStatus),
		zap.String("by", userID.String()),
	)

	return fullName
}

// getActorInfo retrieves the user's full name, department, and branch code.
func (b *serviceBase) getActorInfo(ctx context.Context, userID uuid.UUID) (fullName, department, branchCode string) {
	user, err := b.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "unknown", "", ""
	}
	return user.FullName, user.Department, ""
}

// insertApprovalAction records an approval action in the database.
func (b *serviceBase) insertApprovalAction(ctx context.Context, dealModule string, dealID uuid.UUID, actionType, statusBefore, statusAfter string, performedBy uuid.UUID, reason string) {
	_, err := b.pool.Exec(ctx, `
		INSERT INTO approval_actions (id, deal_module, deal_id, action_type, status_before, status_after, performed_by, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		uuid.New(), dealModule, dealID, actionType, statusBefore, statusAfter, performedBy, nullableStr(reason),
	)
	if err != nil {
		b.logger.Error("failed to insert approval action",
			zap.String("deal_id", dealID.String()),
			zap.String("action_type", actionType),
			zap.Error(err),
		)
	}
}

// storeCancelMetadata stores the original status before cancel was requested.
func (b *serviceBase) storeCancelMetadata(ctx context.Context, dealModule string, dealID uuid.UUID, originalStatus string) {
	_, err := b.pool.Exec(ctx, `
		INSERT INTO cancel_metadata (deal_id, deal_module, original_status)
		VALUES ($1, $2, $3)
		ON CONFLICT (deal_id, deal_module)
		DO UPDATE SET original_status = $3`,
		dealID, dealModule, originalStatus)
	if err != nil {
		b.logger.Error("failed to store cancel metadata",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
	}
}

// getCancelOriginalStatus retrieves the original status before cancel was requested.
func (b *serviceBase) getCancelOriginalStatus(ctx context.Context, dealModule string, dealID uuid.UUID) string {
	var status string
	err := b.pool.QueryRow(ctx, `
		SELECT original_status FROM cancel_metadata
		WHERE deal_id = $1 AND deal_module = $2`, dealID, dealModule).Scan(&status)
	if err != nil {
		b.logger.Error("failed to get cancel original status",
			zap.String("deal_id", dealID.String()),
			zap.Error(err))
		return ""
	}
	return status
}

// getApprovalHistory returns the approval actions for a deal.
func (b *serviceBase) getApprovalHistory(ctx context.Context, dealModule string, dealID uuid.UUID) ([]dto.ApprovalHistoryEntry, error) {
	rows, err := b.pool.Query(ctx, `
		SELECT a.id, a.action_type, a.status_before, a.status_after,
			a.performed_by, COALESCE(u.full_name, 'Unknown') AS performer_name,
			a.performed_at, COALESCE(a.reason, '') AS reason
		FROM approval_actions a
		LEFT JOIN users u ON u.id = a.performed_by
		WHERE a.deal_module = $1 AND a.deal_id = $2
		ORDER BY a.performed_at ASC`, dealModule, dealID)
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

// cancelLevelConfig defines a single level in the cancel approval flow.
type cancelLevelConfig struct {
	approveStatus string
	approveAction string
	rejectAction  string
	perm          string
}

// cancelApproveTargetStatus determines the target status for cancel approval/rejection.
func (b *serviceBase) cancelApproveTargetStatus(ctx context.Context, dealModule string, dealID uuid.UUID, currentStatus, action string, levels map[string]cancelLevelConfig) (newStatus, actionType, requiredPerm string, err error) {
	cfg, ok := levels[currentStatus]
	if !ok {
		return "", "", "", apperror.New(apperror.ErrInvalidTransition,
			fmt.Sprintf("cannot approve/reject cancel for deal in status %s", currentStatus))
	}

	switch action {
	case "APPROVE":
		return cfg.approveStatus, cfg.approveAction, cfg.perm, nil
	case "REJECT":
		orig := b.getCancelOriginalStatus(ctx, dealModule, dealID)
		if orig == "" {
			orig = constants.StatusCompleted
		}
		return orig, cfg.rejectAction, cfg.perm, nil
	default:
		return "", "", "", apperror.New(apperror.ErrValidation, "action must be APPROVE or REJECT")
	}
}

// --- package-level helpers ---

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
		constants.StatusVoidedByAccounting,
		constants.StatusVoidedByRisk,
		constants.StatusVoidedBySettlement:
		return true
	}
	return false
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
