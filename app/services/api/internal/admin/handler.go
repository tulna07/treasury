package admin

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/httputil"
)

// Handler handles HTTP requests for admin operations.
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new admin Handler.
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// ListUsers lists users with filters and pagination.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	pag := httputil.ParsePagination(r)

	filter := dto.UserFilter{}
	if v := r.URL.Query().Get("department"); v != "" {
		filter.Department = &v
	}
	if v := r.URL.Query().Get("branch_id"); v != "" {
		filter.BranchID = &v
	}
	if v := r.URL.Query().Get("role"); v != "" {
		filter.Role = &v
	}
	if v := r.URL.Query().Get("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			filter.IsActive = &b
		}
	}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = &v
	}

	result, err := h.service.ListUsers(r.Context(), filter, pag)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, result)
}

// CreateUser creates a new user.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateUserRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.service.CreateUser(r.Context(), req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Created(w, r, resp)
}

// GetUser gets a user by ID.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	resp, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// UpdateUser updates a user.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	var req dto.UpdateUserRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	resp, err := h.service.UpdateUser(r.Context(), id, req, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, resp)
}

// LockUser deactivates a user.
func (h *Handler) LockUser(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	var req dto.LockUnlockRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.Reason == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "reason is required"))
		return
	}

	if err := h.service.LockUser(r.Context(), id, req.Reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "user locked"})
}

// UnlockUser activates a user.
func (h *Handler) UnlockUser(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	var req dto.LockUnlockRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.Reason == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "reason is required"))
		return
	}

	if err := h.service.UnlockUser(r.Context(), id, req.Reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "user unlocked"})
}

// ResetPassword generates a temp password.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	tempPwd, err := h.service.ResetPassword(r.Context(), id, audit.ExtractIP(r), r.UserAgent())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, dto.ResetPasswordResponse{TempPassword: tempPwd})
}

// ListRoles lists all roles.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, roles)
}

// GetRolePermissions lists permissions for a role.
func (h *Handler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "role code is required"))
		return
	}

	result, err := h.service.GetRolePermissions(r.Context(), code)
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, result)
}

// ListPermissions lists all available permissions.
func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.service.ListPermissions(r.Context())
	if err != nil {
		httputil.Error(w, r, err)
		return
	}
	httputil.Success(w, r, perms)
}

// UpdateRolePermissions updates the permissions for a role.
func (h *Handler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	roleCode := chi.URLParam(r, "code")
	if roleCode == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "role code is required"))
		return
	}

	var req dto.UpdateRolePermissionsRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if err := h.service.UpdateRolePermissions(r.Context(), roleCode, req.Permissions, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}

// AssignRole assigns a role to a user.
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}

	var req dto.AssignRoleRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.RoleCode == "" || req.Reason == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "role_code and reason are required"))
		return
	}

	if err := h.service.AssignRole(r.Context(), userID, req, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.Success(w, r, map[string]string{"message": "role assigned"})
}

// RevokeRole revokes a role from a user.
func (h *Handler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	userID, err := httputil.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "invalid user ID"))
		return
	}
	roleCode := chi.URLParam(r, "code")
	if roleCode == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "role code is required"))
		return
	}

	var req dto.RevokeRoleRequest
	if err := httputil.ParseBody(r, &req); err != nil {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, err.Error()))
		return
	}

	if req.Reason == "" {
		httputil.Error(w, r, apperror.New(apperror.ErrValidation, "reason is required"))
		return
	}

	if err := h.service.RevokeRole(r.Context(), userID, roleCode, req.Reason, audit.ExtractIP(r), r.UserAgent()); err != nil {
		httputil.Error(w, r, err)
		return
	}

	httputil.NoContent(w)
}
