package audit

import (
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for audit log operations.
type Handler struct {
	repo   repository.AuditLogRepository
	logger *zap.Logger
}

// NewHandler creates a new audit Handler.
func NewHandler(repo repository.AuditLogRepository, logger *zap.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// List lists audit logs with filters and pagination.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.AuditLogFilter{}
	if v := r.URL.Query().Get("user_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.UserID = &id
		}
	}
	if v := r.URL.Query().Get("deal_module"); v != "" {
		filter.DealModule = &v
	}
	if v := r.URL.Query().Get("deal_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.DealID = &id
		}
	}
	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = &v
	}
	if v := r.URL.Query().Get("date_from"); v != "" {
		filter.DateFrom = &v
	}
	if v := r.URL.Query().Get("date_to"); v != "" {
		filter.DateTo = &v
	}

	logs, total, err := h.repo.List(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.AuditLogResponse, 0, len(logs))
	for _, l := range logs {
		items = append(items, AuditLogToResponse(&l))
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	httputil.Success(w, r, result)
}

// Stats returns audit log action counts for a date range.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if dateFrom == "" || dateTo == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "date_from and date_to are required"))
		return
	}

	stats, err := h.repo.Stats(r.Context(), dateFrom, dateTo)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, stats)
}
