// Package repository defines the interfaces for data access.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// FxDealRepository defines the interface for FX deal data operations.
type FxDealRepository interface {
	Create(ctx context.Context, deal *model.FxDeal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.FxDeal, error)
	List(ctx context.Context, filter FxDealFilter, pag dto.PaginationRequest) ([]model.FxDeal, int64, error)
	Update(ctx context.Context, deal *model.FxDeal) error
	UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	SumOutstandingByCounterparty(ctx context.Context, counterpartyID uuid.UUID, excludeDealID *uuid.UUID) (decimal.Decimal, error)
}

// FxDealFilter holds filter criteria for listing FX deals.
type FxDealFilter struct {
	Status          *string    // single status filter
	Statuses        *[]string  // multiple statuses (for role scope)
	CounterpartyID  *uuid.UUID
	DealType        *string
	FromDate        *string
	ToDate          *string
	CreatedBy       *uuid.UUID // resource owner filter
	TicketNumber    *string    // search by ticket
	ExcludeStatuses *[]string  // hide cancelled by default
}

// BondDealRepository defines the interface for Bond/GTCG deal data operations.
type BondDealRepository interface {
	Create(ctx context.Context, deal *model.BondDeal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.BondDeal, error)
	List(ctx context.Context, filter dto.BondDealListFilter, pag dto.PaginationRequest) ([]model.BondDeal, int64, error)
	Update(ctx context.Context, deal *model.BondDeal) error
	UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	CheckInventory(ctx context.Context, bondCode, bondCategory, portfolioType string) (int64, error)
	GetInventory(ctx context.Context, bondCode, bondCategory, portfolioType string) (*model.BondInventory, error)
	ListInventory(ctx context.Context) ([]model.BondInventory, error)
	IncrementInventory(ctx context.Context, bondCode, bondCategory, portfolioType string, quantity int64, updatedBy uuid.UUID) error
	DecrementInventory(ctx context.Context, bondCode, bondCategory, portfolioType string, quantity int64, updatedBy uuid.UUID) error
	UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error
}

// MMInterbankRepository defines the interface for MM interbank deal data operations.
type MMInterbankRepository interface {
	Create(ctx context.Context, deal *model.MMInterbankDeal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MMInterbankDeal, error)
	List(ctx context.Context, filter dto.MMInterbankFilter, pag dto.PaginationRequest) ([]model.MMInterbankDeal, int64, error)
	Update(ctx context.Context, deal *model.MMInterbankDeal) error
	UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error
}

// MMOMORepoRepository defines the interface for MM OMO/Repo deal data operations.
type MMOMORepoRepository interface {
	Create(ctx context.Context, deal *model.MMOMORepoDeal) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.MMOMORepoDeal, error)
	List(ctx context.Context, filter dto.MMOMORepoFilter, pag dto.PaginationRequest) ([]model.MMOMORepoDeal, int64, error)
	Update(ctx context.Context, deal *model.MMOMORepoDeal) error
	UpdateStatus(ctx context.Context, id uuid.UUID, oldStatus, newStatus string, updatedBy uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	UpdateCancelFields(ctx context.Context, id uuid.UUID, reason string, requestedBy uuid.UUID) error
}

// CreditLimitRepository defines the interface for credit limit data operations.
type CreditLimitRepository interface {
	Create(ctx context.Context, limit *model.CreditLimit) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.CreditLimit, error)
	GetByCounterparty(ctx context.Context, counterpartyID uuid.UUID, limitType string) (*model.CreditLimit, error)
	GetActiveByCounterparty(ctx context.Context, counterpartyID uuid.UUID, currencyCode string) (*model.CreditLimit, error)
	List(ctx context.Context, filter CreditLimitFilter, pag dto.PaginationRequest) ([]model.CreditLimit, int64, error)
	Update(ctx context.Context, limit *model.CreditLimit) error
	UpdateUsedAmount(ctx context.Context, id uuid.UUID, change decimal.Decimal) error
}

// CreditLimitFilter holds filter criteria for listing credit limits.
type CreditLimitFilter struct {
	CounterpartyID *uuid.UUID
	Status         *string
	LimitType      *string
}

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
}

// AdminUserRepository defines the interface for admin user management operations.
type AdminUserRepository interface {
	Create(ctx context.Context, user *model.User, passwordHash string) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, filter dto.UserFilter, pag dto.PaginationRequest) ([]model.AdminUser, int64, error)
	Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) error
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	AssignRole(ctx context.Context, userID uuid.UUID, roleCode string, grantedBy uuid.UUID) error
	RemoveRole(ctx context.Context, userID uuid.UUID, roleCode string) error
	ListRoles(ctx context.Context) ([]model.Role, error)
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	GetRolePermissions(ctx context.Context, roleCode string) ([]string, string, error)
	UpdateRolePermissions(ctx context.Context, roleCode string, permCodes []string) error
}

// CounterpartyRepository defines the interface for counterparty data operations.
type CounterpartyRepository interface {
	Create(ctx context.Context, cp *model.Counterparty) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Counterparty, error)
	GetByCode(ctx context.Context, code string) (*model.Counterparty, error)
	List(ctx context.Context, filter dto.CounterpartyFilter, pag dto.PaginationRequest) ([]model.Counterparty, int64, error)
	ListActive(ctx context.Context) ([]model.Counterparty, error)
	Update(ctx context.Context, cp *model.Counterparty) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

// MasterDataRepository defines the interface for read-only master data operations.
type MasterDataRepository interface {
	ListCurrencies(ctx context.Context) ([]model.Currency, error)
	ListCurrencyPairs(ctx context.Context) ([]model.CurrencyPair, error)
	ListBranches(ctx context.Context) ([]model.Branch, error)
	GetBranchByID(ctx context.Context, id uuid.UUID) (*model.Branch, error)
	ListExchangeRates(ctx context.Context, filter dto.ExchangeRateFilter, pag dto.PaginationRequest) ([]model.ExchangeRate, int64, error)
	GetLatestRate(ctx context.Context, currencyCode string) (*model.ExchangeRate, error)
}

// AuditLogRepository defines the interface for audit log operations.
type AuditLogRepository interface {
	List(ctx context.Context, filter dto.AuditLogFilter, pag dto.PaginationRequest) ([]model.AuditLog, int64, error)
	Stats(ctx context.Context, dateFrom, dateTo string) ([]dto.AuditLogStatsResponse, error)
}

// InternationalPaymentRepository defines the interface for international payment data operations.
type InternationalPaymentRepository interface {
	Create(ctx context.Context, payment *model.InternationalPayment) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.InternationalPayment, error)
	List(ctx context.Context, filter dto.InternationalPaymentFilter, pag dto.PaginationRequest) ([]model.InternationalPayment, int64, error)
	Approve(ctx context.Context, id uuid.UUID, settledBy uuid.UUID) error
	Reject(ctx context.Context, id uuid.UUID, settledBy uuid.UUID, reason string) error
}

// AttachmentRepository defines the interface for deal attachment data operations.
type AttachmentRepository interface {
	Create(ctx context.Context, a *model.DealAttachment) error
	ListByDeal(ctx context.Context, module string, dealID uuid.UUID) ([]model.DealAttachment, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.DealAttachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountByDeal(ctx context.Context, module string, dealID uuid.UUID) (int, error)
}

// NotificationRepository defines the interface for notification data operations.
type NotificationRepository interface {
	Create(ctx context.Context, n *model.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)
	ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, offset, limit int) ([]model.Notification, int, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	DeleteOld(ctx context.Context, olderThan time.Time) (int, error)
	ListUserIDsByRole(ctx context.Context, role string) ([]uuid.UUID, error)
}
