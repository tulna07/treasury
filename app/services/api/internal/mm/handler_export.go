package mm

import (
	"fmt"
	"net/http"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/export"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// ExportHandler handles MM Interbank export HTTP requests.
type ExportHandler struct {
	service *InterbankService
	engine  *export.Engine
	logger  *zap.Logger
}

// NewExportHandler creates a new MM Interbank export handler.
func NewExportHandler(service *InterbankService, engine *export.Engine, logger *zap.Logger) *ExportHandler {
	return &ExportHandler{service: service, engine: engine, logger: logger}
}

// ExportDeals handles POST /mm/interbank/export
func (h *ExportHandler) ExportDeals(w http.ResponseWriter, r *http.Request) {
	var req dto.ExportRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	// Parse dates
	dateFrom, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid from date format, expected YYYY-MM-DD"))
		return
	}
	dateTo, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid to date format, expected YYYY-MM-DD"))
		return
	}

	if dateTo.Before(dateFrom) {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "to date must be after from date"))
		return
	}

	// Validate password strength
	if err := validateMMExportPassword(req.Password); err != nil {
		httputil.Error(w, r, err)
		return
	}

	// Get user info from context
	userID := ctxutil.GetUserUUID(r.Context())
	roles := ctxutil.GetRoles(r.Context())

	// Fetch user details
	user, err := h.service.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to get user info"))
		return
	}

	roleName := ""
	if len(roles) > 0 {
		roleName = roles[0]
	}

	// Query deals with date filter
	fromStr := req.From
	toStr := req.To
	filter := dto.MMInterbankFilter{
		FromDate: &fromStr,
		ToDate:   &toStr,
	}

	result, err := h.service.ListDeals(r.Context(), filter, dto.PaginationRequest{
		Page:     1,
		PageSize: 100000, // export all matching deals
	})
	if err != nil {
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "failed to query deals for export"))
		return
	}

	// Build and execute export
	builder := NewInterbankReportBuilder(result.Data)
	params := export.ExportParams{
		User: export.UserInfo{
			ID:       userID,
			Username: user.Username,
			FullName: user.FullName,
			Role:     roleName,
		},
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		Password:  req.Password,
		ClientIP:  audit.ExtractIP(r),
		UserAgent: r.UserAgent(),
	}

	exportResult, fileData, err := h.engine.Execute(r.Context(), builder, params)
	if err != nil {
		h.logger.Error("MM interbank export failed", zap.Error(err))
		httputil.Error(w, r, apperror.Wrap(err, apperror.ErrInternal, "export failed"))
		return
	}

	// Stream file back to client
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, exportResult.ExportCode))
	w.Header().Set("X-Export-Code", exportResult.ExportCode)
	w.Header().Set("X-File-Checksum", exportResult.Checksum)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fileData)
}

// validateMMExportPassword checks password strength: min 8 chars, upper + lower + digit.
func validateMMExportPassword(password string) error {
	if len(password) < 8 {
		return apperror.New(apperror.ErrValidation, "password must be at least 8 characters")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper {
		return apperror.New(apperror.ErrValidation, "password must contain at least one uppercase letter")
	}
	if !hasLower {
		return apperror.New(apperror.ErrValidation, "password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return apperror.New(apperror.ErrValidation, "password must contain at least one digit")
	}

	return nil
}
