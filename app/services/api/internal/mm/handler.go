// Package mm xử lý nghiệp vụ Thị trường Tiền tệ (Money Market).
package mm

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

// Handler xử lý các HTTP request cho module Money Market (Liên ngân hàng, OMO, Government Repo).
type Handler struct {
	interbankService *InterbankService
	omoRepoService   *OMORepoService
	logger           *zap.Logger
}

// NewHandler tạo handler mới cho module Money Market.
func NewHandler(interbankService *InterbankService, omoRepoService *OMORepoService, logger *zap.Logger) *Handler {
	return &Handler{interbankService: interbankService, omoRepoService: omoRepoService, logger: logger}
}

// parseID trích xuất và validate UUID của deal từ URL path.
func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid deal ID"))
		return uuid.Nil, false
	}
	return id, true
}

// parseIDAndReason trích xuất UUID của deal và lý do từ request body.
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

// ─── INTERBANK (Liên ngân hàng) ────────────────────────────────────────────────

// InterbankList godoc
// @Summary      Danh sách giao dịch Liên ngân hàng
// @Description  Lấy danh sách giao dịch Liên ngân hàng với filter và phân trang.
// @Tags         MM Interbank
// @Accept       json
// @Produce      json
// @Param        status          query    string  false  "Filter by status"
// @Param        counterparty_id query    string  false  "Filter by counterparty UUID"
// @Param        direction       query    string  false  "Filter by direction (PLACE, TAKE, LEND, BORROW)"
// @Param        currency_code   query    string  false  "Filter by currency code"
// @Param        from_date       query    string  false  "From date (YYYY-MM-DD)"
// @Param        to_date         query    string  false  "To date (YYYY-MM-DD)"
// @Param        deal_number     query    string  false  "Search by deal number"
// @Param        page            query    int     false  "Page number" default(1)
// @Param        page_size       query    int     false  "Items per page" default(20)
// @Param        sort_by         query    string  false  "Sort field" default(created_at)
// @Param        sort_dir        query    string  false  "Sort direction" default(desc)
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/interbank [get]
func (h *Handler) InterbankList(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.MMInterbankFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}
	if v := r.URL.Query().Get("direction"); v != "" {
		filter.Direction = &v
	}
	if v := r.URL.Query().Get("currency_code"); v != "" {
		filter.CurrencyCode = &v
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

	result, err := h.interbankService.ListDeals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// InterbankCreate godoc
// @Summary      Tạo giao dịch Liên ngân hàng mới
// @Description  Tạo một giao dịch Liên ngân hàng mới. Giao dịch được tạo ở trạng thái OPEN.
// @Tags         MM Interbank
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateMMInterbankRequest  true  "Interbank deal creation payload"
// @Success      201   {object}  dto.APIResponse
// @Router       /mm/interbank [post]
func (h *Handler) InterbankCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMMInterbankRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.interbankService.CreateDeal(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// InterbankGet godoc
// @Summary      Chi tiết giao dịch Liên ngân hàng
// @Description  Lấy thông tin chi tiết một giao dịch Liên ngân hàng theo ID.
// @Tags         MM Interbank
// @Param        id   path      string  true  "Interbank Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/interbank/{id} [get]
func (h *Handler) InterbankGet(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.interbankService.GetDeal(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// InterbankUpdate godoc
// @Summary      Cập nhật giao dịch Liên ngân hàng
// @Description  Cập nhật thông tin giao dịch Liên ngân hàng. Chỉ giao dịch ở trạng thái OPEN mới được sửa.
// @Tags         MM Interbank
// @Param        id    path      string                        true  "Interbank Deal ID (UUID)"
// @Param        body  body      dto.UpdateMMInterbankRequest  true  "Interbank deal update payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/interbank/{id} [put]
func (h *Handler) InterbankUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateMMInterbankRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.interbankService.UpdateDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// InterbankApprove godoc
// @Summary      Phê duyệt / Từ chối giao dịch Liên ngân hàng
// @Description  Phê duyệt (APPROVE) hoặc từ chối (REJECT) giao dịch Liên ngân hàng.
// @Tags         MM Interbank
// @Param        id    path      string               true  "Interbank Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/approve [post]
func (h *Handler) InterbankApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.interbankService.ApproveDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal status updated"})
}

// InterbankRecall godoc
// @Summary      Thu hồi giao dịch Liên ngân hàng
// @Description  Thu hồi giao dịch Liên ngân hàng về trạng thái OPEN.
// @Tags         MM Interbank
// @Param        id    path      string  true  "Interbank Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Recall reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/recall [post]
func (h *Handler) InterbankRecall(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.interbankService.RecallDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal recalled"})
}

// InterbankCancel godoc
// @Summary      Yêu cầu hủy giao dịch Liên ngân hàng
// @Description  Yêu cầu hủy giao dịch Liên ngân hàng đã hoàn thành. Cần duyệt 2 cấp.
// @Tags         MM Interbank
// @Param        id    path      string  true  "Interbank Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Cancellation reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/cancel [post]
func (h *Handler) InterbankCancel(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.interbankService.CancelDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal cancel requested"})
}

// InterbankCancelApprove godoc
// @Summary      Phê duyệt / Từ chối yêu cầu hủy giao dịch Liên ngân hàng
// @Description  Phê duyệt hoặc từ chối yêu cầu hủy giao dịch. 2 cấp: DeskHead (L1), Director (L2).
// @Tags         MM Interbank
// @Param        id    path      string               true  "Interbank Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Cancel approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/cancel-approve [post]
func (h *Handler) InterbankCancelApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.interbankService.ApproveCancelDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "cancel action processed"})
}

// InterbankHistory godoc
// @Summary      Lịch sử phê duyệt giao dịch Liên ngân hàng
// @Description  Lấy lịch sử phê duyệt của một giao dịch Liên ngân hàng.
// @Tags         MM Interbank
// @Param        id   path      string  true  "Interbank Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/history [get]
func (h *Handler) InterbankHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	entries, err := h.interbankService.GetApprovalHistory(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, entries)
}

// InterbankClone godoc
// @Summary      Sao chép giao dịch Liên ngân hàng
// @Description  Tạo bản sao của giao dịch Liên ngân hàng bị từ chối hoặc trả lại.
// @Tags         MM Interbank
// @Param        id   path      string  true  "Interbank Deal ID to clone (UUID)"
// @Success      201  {object}  dto.APIResponse
// @Router       /mm/interbank/{id}/clone [post]
func (h *Handler) InterbankClone(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.interbankService.CloneDeal(r.Context(), id, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// InterbankDelete godoc
// @Summary      Xóa mềm giao dịch Liên ngân hàng
// @Description  Xóa mềm giao dịch Liên ngân hàng ở trạng thái OPEN.
// @Tags         MM Interbank
// @Param        id   path      string  true  "Interbank Deal ID (UUID)"
// @Success      204  "No Content"
// @Router       /mm/interbank/{id} [delete]
func (h *Handler) InterbankDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.interbankService.SoftDelete(r.Context(), id, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}

// ─── OMO (Nghiệp vụ thị trường mở) ────────────────────────────────────────────

// omoRepoFilter builds a MMOMORepoFilter from query params with the given deal subtype.
func (h *Handler) omoRepoFilter(r *http.Request, subtype string) dto.MMOMORepoFilter {
	filter := dto.MMOMORepoFilter{DealSubtype: subtype}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
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
	return filter
}

// OMOList godoc
// @Summary      Danh sách giao dịch OMO
// @Description  Lấy danh sách giao dịch OMO (Nghiệp vụ thị trường mở) với filter và phân trang.
// @Tags         MM OMO
// @Accept       json
// @Produce      json
// @Param        status          query    string  false  "Filter by status"
// @Param        counterparty_id query    string  false  "Filter by counterparty UUID"
// @Param        from_date       query    string  false  "From date (YYYY-MM-DD)"
// @Param        to_date         query    string  false  "To date (YYYY-MM-DD)"
// @Param        deal_number     query    string  false  "Search by deal number"
// @Param        page            query    int     false  "Page number" default(1)
// @Param        page_size       query    int     false  "Items per page" default(20)
// @Param        sort_by         query    string  false  "Sort field" default(created_at)
// @Param        sort_dir        query    string  false  "Sort direction" default(desc)
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/omo [get]
func (h *Handler) OMOList(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)
	filter := h.omoRepoFilter(r, "OMO")

	result, err := h.omoRepoService.ListDeals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// OMOCreate godoc
// @Summary      Tạo giao dịch OMO mới
// @Description  Tạo một giao dịch OMO mới. Giao dịch được tạo ở trạng thái OPEN.
// @Tags         MM OMO
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateMMOMORepoRequest  true  "OMO deal creation payload"
// @Success      201   {object}  dto.APIResponse
// @Router       /mm/omo [post]
func (h *Handler) OMOCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMMOMORepoRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}
	req.DealSubtype = "OMO"

	resp, err := h.omoRepoService.CreateDeal(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// OMOGet godoc
// @Summary      Chi tiết giao dịch OMO
// @Description  Lấy thông tin chi tiết một giao dịch OMO theo ID.
// @Tags         MM OMO
// @Param        id   path      string  true  "OMO Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/omo/{id} [get]
func (h *Handler) OMOGet(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.omoRepoService.GetDeal(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// OMOUpdate godoc
// @Summary      Cập nhật giao dịch OMO
// @Description  Cập nhật thông tin giao dịch OMO. Chỉ giao dịch ở trạng thái OPEN mới được sửa.
// @Tags         MM OMO
// @Param        id    path      string                      true  "OMO Deal ID (UUID)"
// @Param        body  body      dto.UpdateMMOMORepoRequest  true  "OMO deal update payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/omo/{id} [put]
func (h *Handler) OMOUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateMMOMORepoRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.omoRepoService.UpdateDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// OMOApprove godoc
// @Summary      Phê duyệt / Từ chối giao dịch OMO
// @Description  Phê duyệt (APPROVE) hoặc từ chối (REJECT) giao dịch OMO.
// @Tags         MM OMO
// @Param        id    path      string               true  "OMO Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/omo/{id}/approve [post]
func (h *Handler) OMOApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.omoRepoService.ApproveDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal status updated"})
}

// OMORecall godoc
// @Summary      Thu hồi giao dịch OMO
// @Description  Thu hồi giao dịch OMO về trạng thái OPEN.
// @Tags         MM OMO
// @Param        id    path      string  true  "OMO Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Recall reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/omo/{id}/recall [post]
func (h *Handler) OMORecall(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.RecallDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal recalled"})
}

// OMOCancel godoc
// @Summary      Yêu cầu hủy giao dịch OMO
// @Description  Yêu cầu hủy giao dịch OMO đã hoàn thành. Cần duyệt 2 cấp.
// @Tags         MM OMO
// @Param        id    path      string  true  "OMO Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Cancellation reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/omo/{id}/cancel [post]
func (h *Handler) OMOCancel(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.CancelDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal cancel requested"})
}

// OMOCancelApprove godoc
// @Summary      Phê duyệt / Từ chối yêu cầu hủy giao dịch OMO
// @Description  Phê duyệt hoặc từ chối yêu cầu hủy giao dịch OMO. 2 cấp: DeskHead (L1), Director (L2).
// @Tags         MM OMO
// @Param        id    path      string               true  "OMO Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Cancel approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/omo/{id}/cancel-approve [post]
func (h *Handler) OMOCancelApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.omoRepoService.ApproveCancelDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "cancel action processed"})
}

// OMOHistory godoc
// @Summary      Lịch sử phê duyệt giao dịch OMO
// @Description  Lấy lịch sử phê duyệt của một giao dịch OMO.
// @Tags         MM OMO
// @Param        id   path      string  true  "OMO Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/omo/{id}/history [get]
func (h *Handler) OMOHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	entries, err := h.omoRepoService.GetApprovalHistory(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, entries)
}

// OMOClone godoc
// @Summary      Sao chép giao dịch OMO
// @Description  Tạo bản sao của giao dịch OMO bị từ chối hoặc trả lại.
// @Tags         MM OMO
// @Param        id   path      string  true  "OMO Deal ID to clone (UUID)"
// @Success      201  {object}  dto.APIResponse
// @Router       /mm/omo/{id}/clone [post]
func (h *Handler) OMOClone(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.omoRepoService.CloneDeal(r.Context(), id, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// OMODelete godoc
// @Summary      Xóa mềm giao dịch OMO
// @Description  Xóa mềm giao dịch OMO ở trạng thái OPEN.
// @Tags         MM OMO
// @Param        id   path      string  true  "OMO Deal ID (UUID)"
// @Success      204  "No Content"
// @Router       /mm/omo/{id} [delete]
func (h *Handler) OMODelete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.SoftDelete(r.Context(), id, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}

// ─── REPO Government Treasury (Government Securities Repo) ─────────────────────────────────────────

// RepoList godoc
// @Summary      Danh sách giao dịch Government Repo
// @Description  Lấy danh sách giao dịch Government Repo với filter và phân trang.
// @Tags         MM Government Repo
// @Accept       json
// @Produce      json
// @Param        status          query    string  false  "Filter by status"
// @Param        counterparty_id query    string  false  "Filter by counterparty UUID"
// @Param        from_date       query    string  false  "From date (YYYY-MM-DD)"
// @Param        to_date         query    string  false  "To date (YYYY-MM-DD)"
// @Param        deal_number     query    string  false  "Search by deal number"
// @Param        page            query    int     false  "Page number" default(1)
// @Param        page_size       query    int     false  "Items per page" default(20)
// @Param        sort_by         query    string  false  "Sort field" default(created_at)
// @Param        sort_dir        query    string  false  "Sort direction" default(desc)
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/repo [get]
func (h *Handler) RepoList(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)
	filter := h.omoRepoFilter(r, "STATE_REPO")

	result, err := h.omoRepoService.ListDeals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// RepoCreate godoc
// @Summary      Tạo giao dịch Government Repo mới
// @Description  Tạo một giao dịch Government Repo mới. Giao dịch được tạo ở trạng thái OPEN.
// @Tags         MM Government Repo
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateMMOMORepoRequest  true  "Government Repo deal creation payload"
// @Success      201   {object}  dto.APIResponse
// @Router       /mm/repo [post]
func (h *Handler) RepoCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateMMOMORepoRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}
	req.DealSubtype = "STATE_REPO"

	resp, err := h.omoRepoService.CreateDeal(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// RepoGet godoc
// @Summary      Chi tiết giao dịch Government Repo
// @Description  Lấy thông tin chi tiết một giao dịch Government Repo theo ID.
// @Tags         MM Government Repo
// @Param        id   path      string  true  "Government Repo Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/repo/{id} [get]
func (h *Handler) RepoGet(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.omoRepoService.GetDeal(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// RepoUpdate godoc
// @Summary      Cập nhật giao dịch Government Repo
// @Description  Cập nhật thông tin giao dịch Government Repo. Chỉ giao dịch ở trạng thái OPEN mới được sửa.
// @Tags         MM Government Repo
// @Param        id    path      string                      true  "Government Repo Deal ID (UUID)"
// @Param        body  body      dto.UpdateMMOMORepoRequest  true  "Government Repo deal update payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/repo/{id} [put]
func (h *Handler) RepoUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.UpdateMMOMORepoRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.omoRepoService.UpdateDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// RepoApprove godoc
// @Summary      Phê duyệt / Từ chối giao dịch Government Repo
// @Description  Phê duyệt (APPROVE) hoặc từ chối (REJECT) giao dịch Government Repo.
// @Tags         MM Government Repo
// @Param        id    path      string               true  "Government Repo Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/repo/{id}/approve [post]
func (h *Handler) RepoApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.omoRepoService.ApproveDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal status updated"})
}

// RepoRecall godoc
// @Summary      Thu hồi giao dịch Government Repo
// @Description  Thu hồi giao dịch Government Repo về trạng thái OPEN.
// @Tags         MM Government Repo
// @Param        id    path      string  true  "Government Repo Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Recall reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/repo/{id}/recall [post]
func (h *Handler) RepoRecall(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.RecallDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal recalled"})
}

// RepoCancel godoc
// @Summary      Yêu cầu hủy giao dịch Government Repo
// @Description  Yêu cầu hủy giao dịch Government Repo đã hoàn thành. Cần duyệt 2 cấp.
// @Tags         MM Government Repo
// @Param        id    path      string  true  "Government Repo Deal ID (UUID)"
// @Param        body  body      object{reason=string}  true  "Cancellation reason"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/repo/{id}/cancel [post]
func (h *Handler) RepoCancel(w http.ResponseWriter, r *http.Request) {
	id, reason, ok := h.parseIDAndReason(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.CancelDeal(r.Context(), id, reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal cancel requested"})
}

// RepoCancelApprove godoc
// @Summary      Phê duyệt / Từ chối yêu cầu hủy giao dịch Government Repo
// @Description  Phê duyệt hoặc từ chối yêu cầu hủy giao dịch Government Repo. 2 cấp: DeskHead (L1), Director (L2).
// @Tags         MM Government Repo
// @Param        id    path      string               true  "Government Repo Deal ID (UUID)"
// @Param        body  body      dto.ApprovalRequest  true  "Cancel approval payload"
// @Success      200   {object}  dto.APIResponse
// @Router       /mm/repo/{id}/cancel-approve [post]
func (h *Handler) RepoCancelApprove(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req dto.ApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.omoRepoService.ApproveCancelDeal(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "cancel action processed"})
}

// RepoHistory godoc
// @Summary      Lịch sử phê duyệt giao dịch Government Repo
// @Description  Lấy lịch sử phê duyệt của một giao dịch Government Repo.
// @Tags         MM Government Repo
// @Param        id   path      string  true  "Government Repo Deal ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /mm/repo/{id}/history [get]
func (h *Handler) RepoHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	entries, err := h.omoRepoService.GetApprovalHistory(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, entries)
}

// RepoClone godoc
// @Summary      Sao chép giao dịch Government Repo
// @Description  Tạo bản sao của giao dịch Government Repo bị từ chối hoặc trả lại.
// @Tags         MM Government Repo
// @Param        id   path      string  true  "Government Repo Deal ID to clone (UUID)"
// @Success      201  {object}  dto.APIResponse
// @Router       /mm/repo/{id}/clone [post]
func (h *Handler) RepoClone(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	resp, err := h.omoRepoService.CloneDeal(r.Context(), id, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// RepoDelete godoc
// @Summary      Xóa mềm giao dịch Government Repo
// @Description  Xóa mềm giao dịch Government Repo ở trạng thái OPEN.
// @Tags         MM Government Repo
// @Param        id   path      string  true  "Government Repo Deal ID (UUID)"
// @Success      204  "No Content"
// @Router       /mm/repo/{id} [delete]
func (h *Handler) RepoDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.omoRepoService.SoftDelete(r.Context(), id, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}
