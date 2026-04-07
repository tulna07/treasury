// Package notification provides notification management and SSE push for the Treasury system.
package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/sse"
)

// Service handles notification business logic.
type Service struct {
	repo   repository.NotificationRepository
	broker *sse.Broker
	logger *zap.Logger
}

// NewService creates a new notification service.
func NewService(repo repository.NotificationRepository, broker *sse.Broker, logger *zap.Logger) *Service {
	return &Service{repo: repo, broker: broker, logger: logger}
}

// CreateAndNotify creates a notification record and pushes an SSE event to the recipient.
func (s *Service) CreateAndNotify(ctx context.Context, n *model.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}

	if err := s.repo.Create(ctx, n); err != nil {
		s.logger.Error("failed to create notification", zap.Error(err))
		return err
	}

	// Push SSE notification event
	resp := toResponse(n)
	data, _ := json.Marshal(resp)
	s.broker.Publish(n.UserID, sse.Event{
		ID:   n.ID.String(),
		Type: "notification",
		Data: data,
	})

	// Also push updated badge count
	s.pushBadgeUpdate(ctx, n.UserID)

	return nil
}

// CountUnread returns the number of unread notifications for a user.
func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

// ListByUser lists notifications for a user.
func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, pag dto.PaginationRequest) ([]model.Notification, int, error) {
	return s.repo.ListByUser(ctx, userID, unreadOnly, pag.Offset(), pag.PageSize)
}

// MarkRead marks a notification as read and pushes a badge update.
func (s *Service) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.repo.MarkRead(ctx, id, userID); err != nil {
		return err
	}
	s.pushBadgeUpdate(ctx, userID)
	return nil
}

// MarkAllRead marks all notifications as read and pushes a badge update.
func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return err
	}
	s.pushBadgeUpdate(ctx, userID)
	return nil
}

// NotifyDealStatusChange sends notifications to the appropriate users after an FX deal status change.
func (s *Service) NotifyDealStatusChange(ctx context.Context, dealModule string, dealID uuid.UUID, ticketNumber, fromStatus, toStatus, actorName string) {
	var targetRoles []string
	var title, message string
	var creatorNotify bool

	ticket := ticketNumber
	if ticket == "" {
		ticket = dealID.String()[:8]
	}

	switch toStatus {
	case constants.StatusPendingL2Approval:
		// OPEN -> PENDING_L2: notify Center Director + Division Head
		targetRoles = []string{constants.RoleCenterDirector, constants.RoleDivisionHead}
		title = "Giao dịch chờ phê duyệt"
		message = fmt.Sprintf("Giao dịch %s %s đã được trình duyệt bởi %s, chờ phê duyệt cấp 2.", dealModule, ticket, actorName)

	case constants.StatusPendingBooking:
		// PENDING_L2 -> PENDING_BOOKING: notify Accountant
		targetRoles = []string{constants.RoleAccountant}
		title = "Giao dịch chờ hạch toán"
		message = fmt.Sprintf("Giao dịch %s %s đã được phê duyệt bởi %s, chờ hạch toán.", dealModule, ticket, actorName)

	case constants.StatusPendingChiefAccountant:
		// PENDING_BOOKING -> PENDING_CHIEF_ACCOUNTANT: notify Chief Accountant
		targetRoles = []string{constants.RoleChiefAccountant}
		title = "Giao dịch chờ kiểm soát kế toán"
		message = fmt.Sprintf("Giao dịch %s %s đã được hạch toán bởi %s, chờ kiểm soát.", dealModule, ticket, actorName)

	case constants.StatusPendingSettlement:
		// PENDING_CHIEF_ACCOUNTANT -> PENDING_SETTLEMENT: notify Settlement Officer
		targetRoles = []string{constants.RoleSettlementOfficer}
		title = "Giao dịch chờ thanh toán"
		message = fmt.Sprintf("Giao dịch %s %s đã được kiểm soát bởi %s, chờ thanh toán.", dealModule, ticket, actorName)

	case constants.StatusRejected:
		creatorNotify = true
		title = "Giao dịch bị từ chối"
		message = fmt.Sprintf("Giao dịch %s %s đã bị từ chối bởi %s.", dealModule, ticket, actorName)

	case constants.StatusCompleted:
		creatorNotify = true
		title = "Giao dịch hoàn thành"
		message = fmt.Sprintf("Giao dịch %s %s đã hoàn thành thanh toán bởi %s.", dealModule, ticket, actorName)

	case constants.StatusCancelled:
		creatorNotify = true
		title = "Giao dịch đã hủy"
		message = fmt.Sprintf("Giao dịch %s %s đã được hủy.", dealModule, ticket)

	case constants.StatusVoidedByAccounting:
		creatorNotify = true
		title = "Giao dịch bị trả lại từ kế toán"
		message = fmt.Sprintf("Giao dịch %s %s đã bị trả lại bởi kế toán %s.", dealModule, ticket, actorName)

	case constants.StatusVoidedBySettlement:
		creatorNotify = true
		title = "Giao dịch bị trả lại từ thanh toán"
		message = fmt.Sprintf("Giao dịch %s %s đã bị trả lại bởi thanh toán %s.", dealModule, ticket, actorName)

	case constants.StatusPendingCancelL1:
		// Cancel request — notify DeskHead
		targetRoles = []string{constants.RoleDeskHead}
		title = "Yêu cầu hủy giao dịch"
		message = fmt.Sprintf("Giao dịch %s %s có yêu cầu hủy từ %s, chờ phê duyệt.", dealModule, ticket, actorName)

	case constants.StatusPendingCancelL2:
		// Cancel L1 approved — notify Center Director + Division Head
		targetRoles = []string{constants.RoleCenterDirector, constants.RoleDivisionHead}
		title = "Yêu cầu hủy giao dịch chờ phê duyệt cấp 2"
		message = fmt.Sprintf("Yêu cầu hủy giao dịch %s %s đã được duyệt cấp 1 bởi %s.", dealModule, ticket, actorName)

	default:
		return
	}

	dealIDRef := dealID
	category := s.categoryForStatus(toStatus)

	if len(targetRoles) > 0 {
		s.notifyByRoles(ctx, targetRoles, title, message, category, dealModule, &dealIDRef)
	}
	if creatorNotify {
		// We don't have the creator ID in this function's parameters, so we skip
		// creator notification from here — the caller (FX service) should pass it separately.
		// This is handled via NotifyUser.
	}
}

// NotifyUser sends a notification to a specific user.
func (s *Service) NotifyUser(ctx context.Context, userID uuid.UUID, title, message, category, dealModule string, dealID *uuid.UUID) {
	n := &model.Notification{
		UserID:     userID,
		Title:      title,
		Message:    message,
		Category:   category,
		DealModule: dealModule,
		DealID:     dealID,
	}
	if err := s.CreateAndNotify(ctx, n); err != nil {
		s.logger.Error("failed to notify user",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) notifyByRoles(ctx context.Context, roles []string, title, message, category, dealModule string, dealID *uuid.UUID) {
	seen := make(map[uuid.UUID]bool)
	for _, role := range roles {
		userIDs, err := s.repo.ListUserIDsByRole(ctx, role)
		if err != nil {
			s.logger.Error("failed to list users by role",
				zap.String("role", role),
				zap.Error(err),
			)
			continue
		}
		for _, uid := range userIDs {
			if seen[uid] {
				continue
			}
			seen[uid] = true
			s.NotifyUser(ctx, uid, title, message, category, dealModule, dealID)
		}
	}
}

func (s *Service) pushBadgeUpdate(ctx context.Context, userID uuid.UUID) {
	count, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		s.logger.Error("failed to count unread for badge", zap.Error(err))
		return
	}
	data, _ := json.Marshal(dto.UnreadCountResponse{Count: count})
	s.broker.Publish(userID, sse.Event{
		Type: "badge_update",
		Data: data,
	})
}

func (s *Service) categoryForStatus(status string) string {
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

func toResponse(n *model.Notification) dto.NotificationResponse {
	return dto.NotificationResponse{
		ID:         n.ID,
		Title:      n.Title,
		Message:    n.Message,
		Category:   n.Category,
		DealModule: n.DealModule,
		DealID:     n.DealID,
		IsRead:     n.IsRead,
		ReadAt:     n.ReadAt,
		CreatedAt:  n.CreatedAt,
	}
}
