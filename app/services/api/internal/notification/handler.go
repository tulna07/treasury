package notification

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
	"github.com/kienlongbank/treasury-api/pkg/sse"
)

// Handler handles HTTP requests for notifications.
type Handler struct {
	service *Service
	broker  *sse.Broker
	logger  *zap.Logger
}

// NewHandler creates a new notification handler.
func NewHandler(service *Service, broker *sse.Broker, logger *zap.Logger) *Handler {
	return &Handler{service: service, broker: broker, logger: logger}
}

// List returns paginated notifications for the authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	pag := httputil.ParsePagination(r)

	unreadOnly := false
	if v := r.URL.Query().Get("unread_only"); v == "true" {
		unreadOnly = true
	}

	notifications, total, err := h.service.ListByUser(r.Context(), userID, unreadOnly, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	items := make([]dto.NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = toResponse(&n)
	}

	httputil.Paginated(w, r, items, int64(total), pag.Page, pag.PageSize)
}

// UnreadCount returns the count of unread notifications.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	count, err := h.service.CountUnread(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, dto.UnreadCountResponse{Count: count})
}

// MarkRead marks a single notification as read.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid notification ID"))
		return
	}

	if err := h.service.MarkRead(r.Context(), id, userID); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "notification marked as read"})
}

// MarkAllRead marks all notifications as read.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "user not authenticated"))
		return
	}

	if err := h.service.MarkAllRead(r.Context(), userID); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "all notifications marked as read"})
}

// Stream is the SSE endpoint for real-time notification delivery.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	userID := ctxutil.GetUserUUID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// http.NewResponseController traverses Unwrap() to find the real Flusher
	rc := http.NewResponseController(w)
	flush := sseFlush(func() {
		if err := rc.Flush(); err != nil {
			h.logger.Debug("SSE flush error", zap.Error(err))
		}
	})

	ch := h.broker.Subscribe(userID)
	defer h.broker.Unsubscribe(userID, ch)

	// Send initial unread count — first write triggers headers
	count, _ := h.service.CountUnread(r.Context(), userID)
	fmt.Fprintf(w, "event: badge_update\ndata: {\"count\":%d}\n\n", count)
	flush()

	// Retrieve Last-Event-ID for reconnection support
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		h.logger.Debug("SSE reconnect", zap.String("last_event_id", lastEventID))
	}

	for {
		select {
		case event := <-ch:
			writeSSErc(w, flush, event.Type, event.ID, string(event.Data))
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			// Heartbeat
			fmt.Fprintf(w, ": heartbeat\n\n")
			flush()
		}
	}
}

// sseFlush is a flush callback type.
type sseFlush func()

func writeSSErc(w http.ResponseWriter, flush sseFlush, eventType, id, data string) {
	if id != "" {
		fmt.Fprintf(w, "id: %s\n", id)
	}
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flush()
}

// parseIntParam parses an int query parameter with a default fallback.
func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}
