package creditlimit

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for credit limits.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new credit limit handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// List godoc
// @Summary      Danh sách hạn mức tín dụng
// @Description  Lấy danh sách hạn mức tín dụng hiện tại với filter và phân trang.
// @Tags         Limits
// @Success      200  {object}  dto.APIResponse
// @Router       /limits [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.CreditLimitListFilter{}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}
	if v := r.URL.Query().Get("limit_type"); v != "" {
		filter.LimitType = &v
	}

	result, err := h.service.ListLimits(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// GetByCounterparty godoc
// @Summary      Hạn mức theo đối tác
// @Description  Lấy cả 2 loại hạn mức (có TSBĐ và không TSBĐ) cho một đối tác.
// @Tags         Limits
// @Param        counterpartyId  path  string  true  "Counterparty ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/{counterpartyId} [get]
func (h *Handler) GetByCounterparty(w http.ResponseWriter, r *http.Request) {
	cpID, err := httputil.ParseUUID(chi.URLParam(r, "counterpartyId"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	limits, err := h.service.GetLimitsByCounterparty(r.Context(), cpID)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, limits)
}

// SetLimit godoc
// @Summary      Cấp / cập nhật hạn mức
// @Description  Tạo hoặc cập nhật hạn mức tín dụng cho đối tác (SCD Type 2).
// @Tags         Limits
// @Param        counterpartyId  path  string                    true  "Counterparty ID (UUID)"
// @Param        body            body  dto.SetCreditLimitRequest true  "Limit data"
// @Success      201  {object}  dto.APIResponse
// @Router       /limits/{counterpartyId} [put]
func (h *Handler) SetLimit(w http.ResponseWriter, r *http.Request) {
	cpID, err := httputil.ParseUUID(chi.URLParam(r, "counterpartyId"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	var req dto.SetCreditLimitRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}
	req.CounterpartyID = cpID

	resp, err := h.service.SetLimit(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// GetUtilization godoc
// @Summary      Chi tiết sử dụng hạn mức
// @Description  Lấy chi tiết sử dụng hạn mức cho đối tác (MM + Bond + FX breakdown).
// @Tags         Limits
// @Param        counterpartyId  path  string  true  "Counterparty ID (UUID)"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/utilization/{counterpartyId} [get]
func (h *Handler) GetUtilization(w http.ResponseWriter, r *http.Request) {
	cpID, err := httputil.ParseUUID(chi.URLParam(r, "counterpartyId"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid counterparty ID"))
		return
	}

	result, err := h.service.GetUtilization(r.Context(), cpID)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// DailySummary godoc
// @Summary      Bảng tổng hợp hạn mức hàng ngày
// @Description  Lấy bảng tổng hợp 11 cột (BRD §3.4.4) cho một ngày.
// @Tags         Limits
// @Param        date  query  string  false  "Date (YYYY-MM-DD), default today"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/daily-summary [get]
func (h *Handler) DailySummary(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date := time.Now()
	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid date format, expected YYYY-MM-DD"))
			return
		}
		date = parsed
	}

	result, err := h.service.GetDailySummary(r.Context(), date)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// ExportDailySummary godoc
// @Summary      Xuất Excel bảng tổng hợp hạn mức
// @Description  Xuất file Excel bảng tổng hợp hạn mức hàng ngày.
// @Tags         Limits
// @Param        date  query  string  false  "Date (YYYY-MM-DD), default today"
// @Success      200  "Excel file"
// @Router       /limits/daily-summary/export [post]
func (h *Handler) ExportDailySummary(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date := time.Now()
	if dateStr != "" {
		parsed, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid date format"))
			return
		}
		date = parsed
	}

	headers, dataRows, err := h.service.ExportDailySummary(r.Context(), date)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	// Simple CSV-style response for now; can integrate with export engine later
	httputil.Success(w, r, map[string]interface{}{
		"date":    date.Format("2006-01-02"),
		"headers": headers,
		"rows":    dataRows,
	})
}

// ApproveRiskOfficer godoc
// @Summary      CV QLRR phê duyệt hạn mức deal
// @Description  Chuyên viên Quản lý Rủi ro phê duyệt / từ chối deal.
// @Tags         Limits
// @Param        body  body  dto.LimitApprovalRequest  true  "Approval payload"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/approve [post]
func (h *Handler) ApproveRiskOfficer(w http.ResponseWriter, r *http.Request) {
	var req dto.LimitApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.ApproveDealRiskOfficer(r.Context(), req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "limit approval updated"})
}

// RejectRiskOfficer godoc
// @Summary      CV QLRR từ chối hạn mức deal
// @Tags         Limits
// @Param        body  body  dto.LimitApprovalRequest  true  "Rejection payload"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/reject [post]
func (h *Handler) RejectRiskOfficer(w http.ResponseWriter, r *http.Request) {
	var req dto.LimitApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}
	req.Action = "REJECT"

	if err := h.service.ApproveDealRiskOfficer(r.Context(), req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal rejected by risk officer"})
}

// ApproveRiskHead godoc
// @Summary      TPB QLRR phê duyệt hạn mức deal
// @Description  Trưởng phòng Ban QLRR phê duyệt / từ chối deal.
// @Tags         Limits
// @Param        body  body  dto.LimitApprovalRequest  true  "Approval payload"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/approve-head [post]
func (h *Handler) ApproveRiskHead(w http.ResponseWriter, r *http.Request) {
	var req dto.LimitApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.ApproveDealRiskHead(r.Context(), req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "limit approval updated by risk head"})
}

// RejectRiskHead godoc
// @Summary      TPB QLRR từ chối hạn mức deal
// @Tags         Limits
// @Param        body  body  dto.LimitApprovalRequest  true  "Rejection payload"
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/reject-head [post]
func (h *Handler) RejectRiskHead(w http.ResponseWriter, r *http.Request) {
	var req dto.LimitApprovalRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}
	req.Action = "REJECT"

	if err := h.service.ApproveDealRiskHead(r.Context(), req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "deal rejected by risk head"})
}

// ListApprovals godoc
// @Summary      Danh sách phê duyệt hạn mức
// @Description  Lấy danh sách các bản ghi phê duyệt hạn mức.
// @Tags         Limits
// @Success      200  {object}  dto.APIResponse
// @Router       /limits/approvals [get]
func (h *Handler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.LimitApprovalListFilter{}
	if v := r.URL.Query().Get("counterparty_id"); v != "" {
		id, err := uuid.Parse(v)
		if err == nil {
			filter.CounterpartyID = &id
		}
	}
	if v := r.URL.Query().Get("deal_module"); v != "" {
		filter.DealModule = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		filter.Status = &v
	}

	result, err := h.service.ListApprovals(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}
