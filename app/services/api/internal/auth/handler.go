package auth

import (
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	service  *Service
	security *config.SecurityConfig
	logger   *zap.Logger
}

// NewHandler creates a new auth Handler.
func NewHandler(service *Service, securityCfg *config.SecurityConfig, logger *zap.Logger) *Handler {
	return &Handler{
		service:  service,
		security: securityCfg,
		logger:   logger,
	}
}

// Login godoc
// @Summary      Đăng nhập hệ thống
// @Description  Xác thực bằng username/password. Token được trả về qua HTTP-only cookies.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "Login credentials"
// @Success      200   {object}  dto.APIResponse{data=dto.LoginResponse}
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Router       /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.Username == "" || req.Password == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "username and password are required"))
		return
	}

	ipAddress := extractIP(r)
	userAgent := r.UserAgent()

	result, err := h.service.Login(r.Context(), req, ipAddress, userAgent)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	// Set tokens as HTTP-only cookies
	h.setAccessTokenCookie(w, result.AccessToken)
	h.setRefreshTokenCookie(w, result.RefreshToken)
	h.setCSRFCookie(w, result.CSRFToken)

	httputil.Success(w, r, dto.LoginResponse{
		User: result.User,
	})
}

// Refresh godoc
// @Summary      Làm mới access token
// @Description  Tạo access token mới từ refresh token cookie. Refresh token cookie tự động gửi theo request.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  dto.APIResponse{data=dto.LoginResponse}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("treasury_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if refreshToken == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrUnauthorized, "refresh token required"))
		return
	}

	result, err := h.service.RefreshToken(r.Context(), refreshToken)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	h.setAccessTokenCookie(w, result.AccessToken)
	h.setCSRFCookie(w, result.CSRFToken)

	httputil.Success(w, r, dto.LoginResponse{
		User: result.User,
	})
}

// Logout godoc
// @Summary      Đăng xuất
// @Description  Thu hồi phiên hiện tại và xóa cookies.
// @Tags         Auth
// @Produce      json
// @Success      204  "No Content"
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("treasury_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		h.logger.Warn("logout error", zap.Error(err))
	}

	h.clearCookies(w)
	httputil.NoContent(w)
}

// Me godoc
// @Summary      Thông tin người dùng hiện tại
// @Description  Lấy profile của user đang đăng nhập.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  dto.APIResponse{data=dto.UserProfile}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.GetCurrentUser(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, profile)
}

// ChangePassword godoc
// @Summary      Đổi mật khẩu
// @Description  Đổi mật khẩu cho user hiện tại. Yêu cầu nhập mật khẩu cũ.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.ChangePasswordRequest  true  "Password change request"
// @Success      204   "No Content"
// @Failure      400   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401   {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500   {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /auth/password [post]
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ChangePasswordRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "old_password and new_password are required"))
		return
	}

	if err := h.service.ChangePassword(r.Context(), req); err != nil {
		httputil.Error(w, r, err)
		return
	}

	h.clearCookies(w)
	httputil.NoContent(w)
}

// LogoutAll godoc
// @Summary      Đăng xuất tất cả thiết bị
// @Description  Thu hồi tất cả phiên đang hoạt động của user hiện tại.
// @Tags         Auth
// @Produce      json
// @Success      204  "No Content"
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /auth/logout-all [post]
func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	if err := h.service.LogoutAll(r.Context()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	h.clearCookies(w)
	httputil.NoContent(w)
}

// ListSessions godoc
// @Summary      Danh sách phiên đăng nhập
// @Description  Lấy danh sách tất cả phiên đang hoạt động của user hiện tại.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  dto.APIResponse{data=[]dto.SessionInfo}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /auth/sessions [get]
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("treasury_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	sessions, err := h.service.ListSessions(r.Context(), refreshToken)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, sessions)
}

// RevokeSession godoc
// @Summary      Thu hồi phiên cụ thể
// @Description  Thu hồi một phiên đăng nhập cụ thể theo ID.
// @Tags         Auth
// @Produce      json
// @Param        id   path      string  true  "Session UUID"
// @Success      204  "No Content"
// @Failure      400  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      401  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      403  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      404  {object}  dto.APIResponse{error=dto.APIError}
// @Failure      500  {object}  dto.APIResponse{error=dto.APIError}
// @Security     BearerAuth
// @Router       /auth/sessions/{id} [delete]
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	sessionID, err := uuid.Parse(idStr)
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid session ID"))
		return
	}

	if err := h.service.RevokeSession(r.Context(), sessionID); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}

// ─── Cookie Helpers ───

func (h *Handler) setAccessTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_access_token",
		Value:    token,
		Path:     "/",
		Domain:   h.security.CookieDomain,
		MaxAge:   int(h.security.AccessTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
}

func (h *Handler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		Domain:   h.security.CookieDomain,
		MaxAge:   int(h.security.RefreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
}

func (h *Handler) setCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_csrf_token",
		Value:    token,
		Path:     "/",
		Domain:   h.security.CookieDomain,
		MaxAge:   int(h.security.AccessTokenTTL.Seconds()),
		HttpOnly: false, // JS needs to read this for X-CSRF-Token header
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
}

func (h *Handler) clearCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_access_token",
		Value:    "",
		Path:     "/",
		Domain:   h.security.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Domain:   h.security.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "treasury_csrf_token",
		Value:    "",
		Path:     "/",
		Domain:   h.security.CookieDomain,
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   h.security.CookieSecure,
		SameSite: config.ParseSameSite(h.security.CookieSameSite),
	})
}

// extractIP gets the client IP from X-Forwarded-For or RemoteAddr.
func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xff := r.Header.Get("X-Real-Ip"); xff != "" {
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
