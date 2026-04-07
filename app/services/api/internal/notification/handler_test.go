package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/sse"
)

// mockNotificationRepo implements repository.NotificationRepository for testing.
type mockNotificationRepo struct {
	notifications []model.Notification
	unreadCount   int
}

func (m *mockNotificationRepo) Create(_ context.Context, n *model.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	m.notifications = append(m.notifications, *n)
	return nil
}

func (m *mockNotificationRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Notification, error) {
	for _, n := range m.notifications {
		if n.ID == id {
			return &n, nil
		}
	}
	return nil, nil
}

func (m *mockNotificationRepo) ListByUser(_ context.Context, _ uuid.UUID, _ bool, _, limit int) ([]model.Notification, int, error) {
	if limit > len(m.notifications) {
		limit = len(m.notifications)
	}
	return m.notifications[:limit], len(m.notifications), nil
}

func (m *mockNotificationRepo) MarkRead(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockNotificationRepo) MarkAllRead(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockNotificationRepo) CountUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return m.unreadCount, nil
}

func (m *mockNotificationRepo) DeleteOld(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

func (m *mockNotificationRepo) ListUserIDsByRole(_ context.Context, _ string) ([]uuid.UUID, error) {
	return nil, nil
}

func setupTestHandler() (*Handler, *mockNotificationRepo) {
	logger := zap.NewNop()
	broker := sse.NewBroker(logger)
	repo := &mockNotificationRepo{
		unreadCount: 3,
		notifications: []model.Notification{
			{
				ID:         uuid.New(),
				UserID:     uuid.New(),
				Title:      "Test notification",
				Message:    "Test message",
				Category:   "FX_APPROVAL",
				DealModule: "FX",
				IsRead:     false,
				CreatedAt:  time.Now(),
			},
		},
	}
	svc := NewService(repo, broker, logger)
	handler := NewHandler(svc, broker, logger)
	return handler, repo
}

func authedContext(userID uuid.UUID) context.Context {
	ctx := context.Background()
	ctx = ctxutil.WithUserID(ctx, userID)
	ctx = ctxutil.WithRoles(ctx, []string{"DEALER"})
	return ctx
}

func TestListNotifications(t *testing.T) {
	handler, _ := setupTestHandler()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?page=1&page_size=20", nil)
	req = req.WithContext(authedContext(userID))
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestUnreadCount(t *testing.T) {
	handler, _ := setupTestHandler()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	req = req.WithContext(authedContext(userID))
	w := httptest.NewRecorder()

	handler.UnreadCount(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)

	data, _ := json.Marshal(resp.Data)
	var countResp dto.UnreadCountResponse
	json.Unmarshal(data, &countResp)
	assert.Equal(t, 3, countResp.Count)
}

func TestMarkRead(t *testing.T) {
	handler, repo := setupTestHandler()
	userID := uuid.New()
	notifID := repo.notifications[0].ID

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+notifID.String()+"/read", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", notifID.String())
	req = req.WithContext(context.WithValue(authedContext(userID), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.MarkRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMarkAllRead(t *testing.T) {
	handler, _ := setupTestHandler()
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/read-all", nil)
	req = req.WithContext(authedContext(userID))
	w := httptest.NewRecorder()

	handler.MarkAllRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSSEStream(t *testing.T) {
	logger := zap.NewNop()
	broker := sse.NewBroker(logger)
	repo := &mockNotificationRepo{unreadCount: 2}
	svc := NewService(repo, broker, logger)
	handler := NewHandler(svc, broker, logger)

	userID := uuid.New()

	// Use a context with cancel to simulate client disconnect
	ctx, cancel := context.WithCancel(authedContext(userID))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/stream", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.Stream(w, req)
		close(done)
	}()

	// Give the handler time to set up and send initial badge
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	broker.Publish(userID, sse.Event{
		ID:   "test-1",
		Type: "notification",
		Data: json.RawMessage(`{"title":"hello"}`),
	})

	time.Sleep(100 * time.Millisecond)

	// Cancel context to disconnect
	cancel()
	<-done

	body := w.Body.String()
	assert.Contains(t, body, "event: badge_update")
	assert.Contains(t, body, `"count":2`)
	assert.Contains(t, body, "event: notification")
	assert.Contains(t, body, "id: test-1")
}
