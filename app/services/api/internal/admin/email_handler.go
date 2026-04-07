package admin

import (
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/email"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// EmailHealthHandler handles admin email health and outbox endpoints.
type EmailHealthHandler struct {
	repo   email.OutboxRepository
	logger *zap.Logger
}

// NewEmailHealthHandler creates a new email health handler.
func NewEmailHealthHandler(repo email.OutboxRepository, logger *zap.Logger) *EmailHealthHandler {
	return &EmailHealthHandler{repo: repo, logger: logger}
}

// Health returns email outbox health metrics.
func (h *EmailHealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	statusCounts, err := h.repo.CountByStatus(ctx)
	if err != nil {
		h.logger.Error("email health: count by status failed", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	sent24h, err := h.repo.CountSent24h(ctx)
	if err != nil {
		h.logger.Error("email health: count sent 24h failed", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	failed24h, err := h.repo.CountFailed24h(ctx)
	if err != nil {
		h.logger.Error("email health: count failed 24h failed", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	oldestPending, err := h.repo.OldestPendingMinutes(ctx)
	if err != nil {
		h.logger.Error("email health: oldest pending failed", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	result := map[string]interface{}{
		"pending":                statusCounts[email.StatusPending],
		"sending":               statusCounts[email.StatusSending],
		"retry":                 statusCounts[email.StatusRetry],
		"sent_24h":              sent24h,
		"failed_24h":            failed24h,
		"oldest_pending_minutes": oldestPending,
	}

	httputil.Success(w, r, result)
}

// ListOutbox returns recent outbox records, optionally filtered by status.
func (h *EmailHealthHandler) ListOutbox(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	status := r.URL.Query().Get("status")

	var (
		emails []email.OutboxEmail
		err    error
	)
	if status != "" {
		emails, err = h.repo.ListByStatus(ctx, status, limit)
	} else {
		emails, err = h.repo.ListRecent(ctx, limit)
	}

	if err != nil {
		h.logger.Error("email outbox list failed", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, emails)
}
