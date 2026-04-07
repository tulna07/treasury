// Package fx handles FX (foreign exchange) deal operations.
package fx

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for FX deals.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new FX handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// List godoc
// @Summary      Danh sách giao dịch FX
// @Description  Lấy danh sách giao dịch ngoại tệ với filter và phân trang. Dữ liệu được lọc theo quyền của người dùng (BRD v3 §6.2). Requires permission: FX_DEAL.VIEW
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        status          query    string  false  "Filter by status (OPEN, PENDING_L2_APPROVAL, ...)"
// @Param        deal_type       query    string  false  "Filter by type (SPOT, FORWARD, SWAP)"
// @Param        counterparty_id query    string  false  "Filter by counterparty UUID"
// @Param        from_date       query    string  false  "From date (YYYY-MM-DD)"
// @Param        to_date         query    string  false  "To date (YYYY-MM-DD)"
// @Param        ticket_number   query    string  false  "Filter by ticket number"
// @Param        page            query    int     false  "Page number (offset mode)" default(1)
// @Param        page_size       query    int     false  "Items per page" default(20)
// @Param        sort_by         query    string  false  "Sort field" default(created_at)
// @Param        sort_dir        query    string  false  "Sort direction (asc, desc)" default(desc)
// @Param        cursor          query    string  false  "Cursor for keyset pagination"
// @Param        limit           query    int     false  "Limit for cursor mode" default(20)
// @Param        direction       query    string  false  "Cursor direction (next, prev)"
// @Success      200  {object}  dto.APIResponse{data=[]dto.FxDealResponse,meta=dto.ResponseMeta}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	// Parse cursor params
	if v := r.URL.Query().Get("cursor"); v != "" {
		pag.Cursor = v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			pag.Limit = n
		}
	}
	if v := r.URL.Query().Get("direction"); v != "" {
		pag.Direction = v
	}

	filter := repository.FxDealFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}
	if v := r.URL.Query().Get("deal_type"); v != "" {
		filter.DealType = &v
	}
	if v := r.URL.Query().Get("from_date"); v != "" {
		filter.FromDate = &v
	}
	if v := r.URL.Query().Get("to_date"); v != "" {
		filter.ToDate = &v
	}
	if v := r.URL.Query().Get("ticket_number"); v != "" {
		filter.TicketNumber = &v
	}
	// exclude_cancelled=false shows all including cancelled (default is to exclude)
	if v := r.URL.Query().Get("exclude_cancelled"); v == "false" {
		empty := []string{}
		filter.ExcludeStatuses = &empty // override to show all
	}

	result, err := h.service.ListDeals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// Create godoc
// @Summary      Tạo giao dịch FX mới
// @Description  Tạo một giao dịch ngoại tệ mới (Spot, Forward, hoặc Swap). Giao dịch được tạo ở trạng thái OPEN. Requires permission: FX_DEAL.CREATE (Role: DEALER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateFxDealRequest  true  "FX deal creation payload"
// @Success      201   {object}  dto.APIResponse{data=dto.FxDealResponse,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateFxDealRequest
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
// @Summary      Chi tiết giao dịch FX
// @Description  Lấy thông tin chi tiết một giao dịch ngoại tệ theo ID. Requires permission: FX_DEAL.VIEW
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "FX Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse{data=dto.FxDealResponse,meta=dto.ResponseMeta}
// @Failure      400  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
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
// @Summary      Cập nhật giao dịch FX
// @Description  Cập nhật thông tin giao dịch ngoại tệ. Chỉ giao dịch ở trạng thái OPEN mới được sửa. Sử dụng optimistic locking (version). Requires permission: FX_DEAL.EDIT (Role: DEALER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "FX Deal ID (UUID)"
// @Param        body  body      dto.UpdateFxDealRequest  true  "FX deal update payload"
// @Success      200   {object}  dto.APIResponse{data=dto.FxDealResponse,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      409   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return
	}

	var req dto.UpdateFxDealRequest
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
// @Summary      Phê duyệt / Từ chối giao dịch FX
// @Description  Phê duyệt (APPROVE) hoặc từ chối (REJECT) giao dịch ngoại tệ. Requires permission: FX_DEAL.APPROVE_L1 (DESK_HEAD), FX_DEAL.APPROVE_L2 (CENTER_DIRECTOR/DIVISION_HEAD), FX_DEAL.BOOK_L1 (ACCOUNTANT), FX_DEAL.BOOK_L2 (CHIEF_ACCOUNTANT), FX_DEAL.SETTLE (SETTLEMENT_OFFICER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "FX Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Approval payload"
// @Success      200   {object}  dto.APIResponse{data=object,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      409   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/approve [post]
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
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
// @Summary      Thu hồi giao dịch FX
// @Description  Thu hồi giao dịch ngoại tệ về trạng thái OPEN (khi đang chờ duyệt). Requires permission: FX_DEAL.RECALL (Role: DEALER, DESK_HEAD)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "FX Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Recall reason"
// @Success      200   {object}  dto.APIResponse{data=object,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      409   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/recall [post]
func (h *Handler) Recall(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := httputil.ParseBody(r, &body); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.RecallDeal(r.Context(), id, body.Reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal recalled"})
}

// Cancel godoc
// @Summary      Hủy giao dịch FX
// @Description  Yêu cầu hủy giao dịch ngoại tệ. Giao dịch sẽ chuyển sang trạng thái CANCELLED. Requires permission: FX_DEAL.CANCEL_REQUEST (Role: DEALER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "FX Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Cancellation reason"
// @Success      200   {object}  dto.APIResponse{data=object,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      409   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/cancel [post]
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if err := httputil.ParseBody(r, &body); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.CancelDeal(r.Context(), id, body.Reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal cancel requested"})
}

// CancelApprove godoc
// @Summary      Phê duyệt / Từ chối yêu cầu hủy giao dịch FX
// @Description  Phê duyệt hoặc từ chối yêu cầu hủy giao dịch. 2 cấp: DeskHead (L1), Director (L2). Requires permission: FX_DEAL.CANCEL_APPROVE_L1 or FX_DEAL.CANCEL_APPROVE_L2
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "FX Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Cancel approval payload"
// @Success      200   {object}  dto.APIResponse{data=object,meta=dto.ResponseMeta}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      422   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/cancel-approve [post]
func (h *Handler) CancelApprove(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
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
// @Summary      Lịch sử phê duyệt giao dịch FX
// @Description  Lấy lịch sử phê duyệt của một giao dịch ngoại tệ. Requires permission: FX_DEAL.VIEW
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "FX Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse{data=[]dto.ApprovalHistoryEntry,meta=dto.ResponseMeta}
// @Failure      400  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/history [get]
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
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
// @Summary      Sao chép giao dịch FX
// @Description  Tạo bản sao của giao dịch ngoại tệ hiện có. Giao dịch mới ở trạng thái OPEN. Requires permission: FX_DEAL.CLONE (Role: DEALER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "FX Deal ID to clone (UUID)"
// @Success      201  {object}  dto.APIResponse{data=dto.FxDealResponse,meta=dto.ResponseMeta}
// @Failure      400  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id}/clone [post]
func (h *Handler) Clone(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
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
// @Summary      Xóa mềm giao dịch FX
// @Description  Xóa mềm (soft delete) giao dịch ngoại tệ. Giao dịch vẫn tồn tại trong DB nhưng không hiển thị. Requires permission: FX_DEAL.DELETE (Role: DEALER)
// @Tags         FX
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "FX Deal ID (UUID)"
// @Success      204  "No Content"
// @Failure      400  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /fx/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return
	}

	if err := h.service.SoftDelete(r.Context(), id, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}
