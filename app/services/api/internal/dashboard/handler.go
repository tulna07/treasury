package dashboard

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for the dashboard.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new dashboard handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// Get godoc
// @Summary      Dashboard tổng quan
// @Description  Lấy dữ liệu tổng quan dashboard: summary, daily volume, module distribution, status daily, recent transactions.
// @Tags         Dashboard
// @Produce      json
// @Success      200  {object}  dto.APIResponse
// @Router       /dashboard [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetDashboard(r.Context())
	if err != nil {
		h.logger.Error("failed to get dashboard", zap.Error(err))
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}
