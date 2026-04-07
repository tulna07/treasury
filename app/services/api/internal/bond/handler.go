// Package bond handles Bond/Bond (Giấy tờ có giá) deal operations.
package bond

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for Bond deals.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new Bond handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// parseID extracts and validates the deal UUID from the URL path.
func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return uuid.Nil, false
	}
	return id, true
}

// parseIDAndReason extracts the deal UUID and a reason from the request body.
func (h *Handler) parseIDAndReason(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	id, ok := h.parseID(w, r)
	if !ok {
		return uuid.Nil, "", false
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := httputil.ParseBody(r, &body); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return uuid.Nil, "", false
	}
	return id, body.Reason, true
}

// List godoc
// @Summary      Danh sách giao dịch Bond
// @Description  Lấy danh sách giao dịch Bond với filter và phân trang.
// @Tags         Bond
// @Accept       json
// @Produce      json
// @Param        status          query    string  false  "Filter by status"
// @Param        bond_category   query    string  false  "Filter by category (GOVERNMENT, FINANCIAL_INSTITUTION, CERTIFICATE_OF_DEPOSIT)"
// @Param        direction       query    string  false  "Filter by direction (BUY, SELL)"
// @Param        counterparty_id query    string  false  "Filter by counterparty UUID"
// @Param        from_date       query    string  false  "From date (YYYY-MM-DD)"
// @Param        to_date         query    string  false  "To date (YYYY-MM-DD)"
// @Param        deal_number     query    string  false  "Search by deal number"
// @Param        page            query    int     false  "Page number" default(1)
// @Param        page_size       query    int     false  "Items per page" default(20)
// @Param        sort_by         query    string  false  "Sort field" default(created_at)
// @Param        sort_dir        query    string  false  "Sort direction" default(desc)
// @Success      200  {object}  dto.APIResponse
// @Router       /bond [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.BondDealListFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}
	if v := r.URL.Query().Get("bond_category"); v != "" {
		filter.BondCategory = &v
	}
	if v := r.URL.Query().Get("direction"); v != "" {
		filter.Direction = &v
	}
	if v := r.URL.Query().Get("from_date"); v != "" {
		filter.FromDate = &v
	}
	if v := r.URL.Query().Get("to_date"); v != "" {
		filter.ToDate = &v
	}
	if v := r.URL.Query().Get("deal_number"); v != "" {
		filter.DealNumber = &v
	}
	if v := r.URL.Query().Get("exclude_cancelled"); v == "false" {
		empty := []string{}
		filter.ExcludeStatuses = &empty
	}

	result, err := h.service.ListDeals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// Create godoc
// @Summary      Tạo giao dịch Bond mới
// @Description  Tạo một giao dịch Bond mới. Giao dịch được tạo ở trạng thái OPEN.
// @Tags         Bond
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateBondDealRequest  true  "Bond deal creation payload"
// @Success      201   {object}  dto.APIResponse
// @Router       /bond [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateBondDealRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.service.CreateDeal(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// Get godoc
// @Summary      Chi tiết giao dịch Bond
// @Description  Lấy thông tin chi tiết một giao dịch Bond theo ID.
// @Tags         Bond
// @Param        id   path      string  true  "Bond Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /bond/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.service.GetDeal(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// Update godoc
// @Summary      Cập nhật giao dịch Bond
// @Description  Cập nhật thông tin giao dịch Bond. Chỉ giao dịch ở trạng thái OPEN mới được sửa.
// @Tags         Bond
// @Param        id    path      string                     true  "Bond Deal ID (UUID)"
// @Param        body  body      dto.UpdateBondDealRequest  true  "Bond deal update payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /bond/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateBondDealRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.service.UpdateDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// Approve godoc
// @Summary      Phê duyệt / Từ chối giao dịch Bond
// @Description  Phê duyệt (APPROVE) hoặc từ chối (REJECT) giao dịch Bond.
// @Tags         Bond
// @Param        id    path      string               true  "Bond Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /bond/{id}/approve [post]
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.ApproveDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal status updated"})
}

// Recall godoc
// @Summary      Thu hồi giao dịch Bond
// @Description  Thu hồi giao dịch Bond về trạng thái OPEN.
// @Tags         Bond
// @Param        id    path      string  true  "Bond Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Recall reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /bond/{id}/recall [post]
func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.service.RecallDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal recalled"})
}

// Cancel godoc
// @Summary      Yêu cầu hủy giao dịch Bond
// @Description  Yêu cầu hủy giao dịch Bond đã hoàn thành. Cần duyệt 2 cấp.
// @Tags         Bond
// @Param        id    path      string  true  "Bond Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Cancellation reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /bond/{id}/cancel [post]
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.service.CancelDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal cancel requested"})
}

// CancelApprove godoc
// @Summary      Phê duyệt / Từ chối yêu cầu hủy giao dịch Bond
// @Description  Phê duyệt hoặc từ chối yêu cầu hủy giao dịch. 2 cấp: DeskHead (L1), Director (L2).
// @Tags         Bond
// @Param        id    path      string               true  "Bond Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Cancel approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /bond/{id}/cancel-approve [post]
func (h *Handler) CancelApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.ApproveCancelDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "cancel action processed"})
}

// History godoc
// @Summary      Lịch sử phê duyệt giao dịch Bond
// @Description  Lấy lịch sử phê duyệt của một giao dịch Bond.
// @Tags         Bond
// @Param        id   path      string  true  "Bond Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /bond/{id}/history [get]
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	entries, err := h.service.GetApprovalHistory(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, entries)
}

// Clone godoc
// @Summary      Sao chép giao dịch Bond
// @Description  Tạo bản sao của giao dịch Bond bị từ chối hoặc trả lại.
// @Tags         Bond
// @Param        id   path      string  true  "Bond Deal ID to clone (UUID)"
// @Success      201  {object}  dto.APIResponse
// @Router       /bond/{id}/clone [post]
func (h *Handler) Clone(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.service.CloneDeal(r.Context(), id, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// Delete godoc
// @Summary      Xóa mềm giao dịch Bond
// @Description  Xóa mềm giao dịch Bond ở trạng thái OPEN.
// @Tags         Bond
// @Param        id   path      string  true  "Bond Deal ID (UUID)"
// @Success      204  "No Content"
// @Router       /bond/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.SoftDelete(r.Context(), id, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}

// Inventory godoc
// @Summary      Danh sách tồn kho trái phiếu
// @Description  Lấy danh sách tồn kho trái phiếu hiện tại.
// @Tags         Bond
// @Success      200  {object}  dto.APIResponse
// @Router       /bond/inventory [get]
func (h *Handler) Inventory(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListInventory(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, items)
}
