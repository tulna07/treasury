package settlement

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for international settlements.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new settlement handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// List godoc
// @Summary      Danh sách thanh toán quốc tế
// @Description  Lấy danh sách các giao dịch TTQT với filter và phân trang.
// @Tags         Settlements
// @Produce      json
// @Param        status          query   string  false  "Filter by status (PENDING, APPROVED, REJECTED)"
// @Param        source_module   query   string  false  "Filter by source (FX, MM)"
// @Param        transfer_date   query   string  false  "Filter by transfer date (YYYY-MM-DD)"
// @Param        page            query   int     false  "Page number" default(1)
// @Param        page_size       query   int     false  "Items per page" default(20)
// @Success      200  {object}  dto.APIResponse
// @Router       /settlements [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.InternationalPaymentFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.SettlementStatus = &v
	}
	if v := r.URL.Query().Get("source_module"); v != "" {
		filter.SourceModule = &v
	}
	if v := r.URL.Query().Get("transfer_date_from"); v != "" {
		filter.TransferDateFrom = &v
	}
	if v := r.URL.Query().Get("transfer_date_to"); v != "" {
		filter.TransferDateTo = &v
	}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}

	result, err := h.service.ListPayments(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}

// Get godoc
// @Summary      Chi tiết thanh toán quốc tế
// @Tags         Settlements
// @Param        id   path      string  true  "Payment ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /settlements/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid ID"))
		return
	}

	resp, err := h.service.GetPayment(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, resp)
}

// Approve godoc
// @Summary      Duyệt thanh toán quốc tế
// @Description  BP.TTQT duyệt chuyển tiền quốc tế → GD chuyển trạng thái Hoàn thành.
// @Tags         Settlements
// @Param        id   path      string  true  "Payment ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /settlements/{id}/approve [post]
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid ID"))
		return
	}

	if err := h.service.ApprovePayment(r.Context(), id, r.RemoteAddr, r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "payment approved"})
}

// Reject godoc
// @Summary      Từ chối thanh toán quốc tế
// @Description  BP.TTQT từ chối → nhập lý do → GD chuyển trạng thái Hủy giao dịch.
// @Tags         Settlements
// @Param        id    path  string                       true  "Payment ID (UUID)"
// @Param        body  body  dto.SettlementApprovalRequest true  "Rejection reason"
// @Success      200  {object}  dto.APIResponse
// @Router       /settlements/{id}/reject [post]
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid ID"))
		return
	}

	var req dto.SettlementApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid request body"))
		return
	}

	if err := h.service.RejectPayment(r.Context(), id, req.Reason, r.RemoteAddr, r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"message": "payment rejected"})
}
