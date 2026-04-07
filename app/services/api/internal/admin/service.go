package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// Service handles admin user management business logic.
type Service struct {
	repo   repository.AdminUserRepository
	rbac   *security.RBACChecker
	audit  *audit.Logger
	logger *zap.Logger
}

// NewService creates a new admin Service.
func NewService(repo repository.AdminUserRepository, rbac *security.RBACChecker, auditLogger *audit.Logger, logger *zap.Logger) *Service {
	return &Service{repo: repo, rbac: rbac, audit: auditLogger, logger: logger}
}

// ListUsers lists users with filters and pagination.
func (s *Service) ListUsers(ctx context.Context, filter dto.UserFilter, pag dto.PaginationRequest) (*dto.PaginationResponse[dto.AdminUserResponse], error) {
	users, total, err := s.repo.List(ctx, filter, pag)
	if err != nil {
		return nil, err
	}

	var items []dto.AdminUserResponse
	for _, u := range users {
		items = append(items, s.adminUserToResponse(&u))
	}
	if items == nil {
		items = []dto.AdminUserResponse{}
	}

	result := dto.NewPaginationResponse(items, total, pag.Page, pag.PageSize)
	return &result, nil
}

// CreateUser creates a new user.
func (s *Service) CreateUser(ctx context.Context, req dto.CreateUserRequest, ipAddress, userAgent string) (*dto.AdminUserResponse, error) {
	actorID := ctxutil.GetUserUUID(ctx)

	if err := s.validateCreateUser(ctx, req); err != nil {
		return nil, err
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return nil, apperror.New(apperror.ErrInternal, "internal error")
	}

	u := userFromCreateReq(req)
	user := &u
	if err := s.repo.Create(ctx, user, hash); err != nil {
		return nil, err
	}

	// Audit log
	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "CREATE_USER",
		DealModule: "SYSTEM",
		DealID:     &user.ID,
		NewValues: map[string]interface{}{
			"username": user.Username, "full_name": user.FullName,
			"email": user.Email, "department": user.Department,
		},
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	s.logger.Info("user created", zap.String("user_id", user.ID.String()), zap.String("by", actorID.String()))

	return s.getUserResponse(ctx, user.ID)
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*dto.AdminUserResponse, error) {
	return s.getUserResponse(ctx, id)
}

// UpdateUser updates a user's info.
func (s *Service) UpdateUser(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest, ipAddress, userAgent string) (*dto.AdminUserResponse, error) {
	actorID := ctxutil.GetUserUUID(ctx)

	// Get old values for audit
	old, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, id, req); err != nil {
		return nil, err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "UPDATE_USER",
		DealModule: "SYSTEM",
		DealID:     &id,
		OldValues: map[string]interface{}{
			"full_name": old.FullName, "email": old.Email,
			"department": old.Department, "position": old.Position,
		},
		NewValues:  req,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	return s.getUserResponse(ctx, id)
}

// LockUser deactivates a user.
func (s *Service) LockUser(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
	actorID := ctxutil.GetUserUUID(ctx)

	if actorID == id {
		return apperror.New(apperror.ErrValidation, "cannot lock your own account")
	}

	if err := s.repo.SetActive(ctx, id, false); err != nil {
		return err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:       actorID,
		FullName:     s.getActorName(ctx, actorID),
		Action:       "LOCK_USER",
		DealModule:   "SYSTEM",
		DealID:       &id,
		StatusBefore: "active",
		StatusAfter:  "locked",
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	s.logger.Info("user locked", zap.String("user_id", id.String()), zap.String("by", actorID.String()))
	return nil
}

// UnlockUser activates a user.
func (s *Service) UnlockUser(ctx context.Context, id uuid.UUID, reason, ipAddress, userAgent string) error {
	actorID := ctxutil.GetUserUUID(ctx)

	if err := s.repo.SetActive(ctx, id, true); err != nil {
		return err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:       actorID,
		FullName:     s.getActorName(ctx, actorID),
		Action:       "UNLOCK_USER",
		DealModule:   "SYSTEM",
		DealID:       &id,
		StatusBefore: "locked",
		StatusAfter:  "active",
		Reason:       reason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	s.logger.Info("user unlocked", zap.String("user_id", id.String()), zap.String("by", actorID.String()))
	return nil
}

// ResetPassword generates a temporary password for a user.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID, ipAddress, userAgent string) (string, error) {
	actorID := ctxutil.GetUserUUID(ctx)

	// Verify user exists
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return "", err
	}

	tempPassword := generateTempPassword()
	hash, err := security.HashPassword(tempPassword)
	if err != nil {
		return "", apperror.New(apperror.ErrInternal, "internal error")
	}

	if err := s.repo.UpdatePassword(ctx, id, hash); err != nil {
		return "", err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "RESET_PASSWORD",
		DealModule: "SYSTEM",
		DealID:     &id,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("user password reset", zap.String("user_id", id.String()), zap.String("by", actorID.String()))
	return tempPassword, nil
}

// ListRoles lists all roles.
func (s *Service) ListRoles(ctx context.Context) ([]dto.RoleResponse, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	var result []dto.RoleResponse
	for _, r := range roles {
		result = append(result, dto.RoleResponse{
			ID: r.ID, Code: r.Code, Name: r.Name,
			Description: r.Description, Scope: r.Scope,
		})
	}
	if result == nil {
		result = []dto.RoleResponse{}
	}
	return result, nil
}

// GetRolePermissions returns the permissions for a given role code.
func (s *Service) GetRolePermissions(ctx context.Context, roleCode string) (*dto.RolePermissionResponse, error) {
	// Read from DB (source of truth)
	perms, roleName, err := s.repo.GetRolePermissions(ctx, roleCode)
	if err != nil {
		// Fallback to constants if DB fails
		constPerms, ok := constants.RolePermissions[roleCode]
		if !ok {
			return nil, apperror.New(apperror.ErrNotFound, "role not found: "+roleCode)
		}
		return &dto.RolePermissionResponse{
			RoleCode:    roleCode,
			RoleName:    roleCode,
			Permissions: constPerms,
		}, nil
	}

	return &dto.RolePermissionResponse{
		RoleCode:    roleCode,
		RoleName:    roleName,
		Permissions: perms,
	}, nil
}

// AssignRole assigns a role to a user.
func (s *Service) AssignRole(ctx context.Context, userID uuid.UUID, req dto.AssignRoleRequest, ipAddress, userAgent string) error {
	actorID := ctxutil.GetUserUUID(ctx)

	// Verify user exists
	if _, err := s.repo.GetByID(ctx, userID); err != nil {
		return err
	}

	// Validate role code
	if _, ok := constants.RolePermissions[req.RoleCode]; !ok {
		return apperror.New(apperror.ErrValidation, "invalid role code: "+req.RoleCode)
	}

	if err := s.repo.AssignRole(ctx, userID, req.RoleCode, actorID); err != nil {
		return err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "ASSIGN_ROLE",
		DealModule: "SYSTEM",
		DealID:     &userID,
		NewValues:  map[string]interface{}{"role_code": req.RoleCode},
		Reason:     req.Reason,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("role assigned",
		zap.String("user_id", userID.String()),
		zap.String("role", req.RoleCode),
		zap.String("by", actorID.String()),
	)
	return nil
}

// RevokeRole revokes a role from a user.
func (s *Service) RevokeRole(ctx context.Context, userID uuid.UUID, roleCode, reason, ipAddress, userAgent string) error {
	actorID := ctxutil.GetUserUUID(ctx)

	if err := s.repo.RemoveRole(ctx, userID, roleCode); err != nil {
		return err
	}

	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "REVOKE_ROLE",
		DealModule: "SYSTEM",
		DealID:     &userID,
		OldValues:  map[string]interface{}{"role_code": roleCode},
		Reason:     reason,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("role revoked",
		zap.String("user_id", userID.String()),
		zap.String("role", roleCode),
		zap.String("by", actorID.String()),
	)
	return nil
}

// ListPermissions returns all available permissions in the system.
func (s *Service) ListPermissions(ctx context.Context) ([]dto.PermissionResponse, error) {
	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	var result []dto.PermissionResponse
	for _, p := range perms {
		result = append(result, dto.PermissionResponse{
			ID: p.ID, Code: p.Code, Name: p.Name, Description: p.Description,
		})
	}
	if result == nil {
		result = []dto.PermissionResponse{}
	}
	return result, nil
}

// UpdateRolePermissions replaces all permissions for a role.
func (s *Service) UpdateRolePermissions(ctx context.Context, roleCode string, permCodes []string, ipAddress, userAgent string) error {
	actorID := ctxutil.GetUserUUID(ctx)

	// Get old permissions for audit
	oldPerms, ok := constants.RolePermissions[roleCode]
	if !ok {
		return apperror.New(apperror.ErrNotFound, "role not found: "+roleCode)
	}

	if err := s.repo.UpdateRolePermissions(ctx, roleCode, permCodes); err != nil {
		return err
	}

	// Update in-memory constants so RBAC checker picks up changes
	constants.RolePermissions[roleCode] = permCodes
	s.rbac.Reload()

	s.audit.Log(ctx, audit.Entry{
		UserID:     actorID,
		FullName:   s.getActorName(ctx, actorID),
		Action:     "UPDATE_ROLE_PERMISSIONS",
		DealModule: "SYSTEM",
		OldValues:  map[string]interface{}{"permissions": oldPerms},
		NewValues:  map[string]interface{}{"permissions": permCodes, "role_code": roleCode},
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
	})

	s.logger.Info("role permissions updated",
		zap.String("role", roleCode),
		zap.Int("count", len(permCodes)),
		zap.String("by", actorID.String()),
	)
	return nil
}

// --- helpers ---

func (s *Service) validateCreateUser(ctx context.Context, req dto.CreateUserRequest) error {
	if req.Username == "" {
		return apperror.New(apperror.ErrValidation, "username is required")
	}
	if req.FullName == "" {
		return apperror.New(apperror.ErrValidation, "full_name is required")
	}
	if req.Email == "" {
		return apperror.New(apperror.ErrValidation, "email is required")
	}
	if len(req.Password) < 8 {
		return apperror.New(apperror.ErrValidation, "password must be at least 8 characters")
	}

	// Check username uniqueness
	existing, err := s.repo.GetByUsername(ctx, req.Username)
	if err != nil {
		return err
	}
	if existing != nil {
		return apperror.New(apperror.ErrConflict, "username already exists")
	}

	return nil
}

func (s *Service) getUserResponse(ctx context.Context, id uuid.UUID) (*dto.AdminUserResponse, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := s.adminUserToResponse(user)
	return &resp, nil
}

func (s *Service) adminUserToResponse(u *model.AdminUser) dto.AdminUserResponse {
	roles := u.RoleCodes
	if roles == nil {
		roles = []string{}
	}
	roleNames := u.RoleNames
	if roleNames == nil {
		roleNames = []string{}
	}
	return dto.AdminUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		FullName:    u.FullName,
		Email:       u.Email,
		BranchID:    u.BranchID,
		BranchName:  u.BranchName,
		Department:  u.Department,
		Position:    u.Position,
		IsActive:    u.IsActive,
		Roles:       roles,
		RoleNames:   roleNames,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (s *Service) getActorName(ctx context.Context, actorID uuid.UUID) string {
	user, err := s.repo.GetByID(ctx, actorID)
	if err != nil || user == nil {
		return "unknown"
	}
	return user.FullName
}

func userFromCreateReq(req dto.CreateUserRequest) model.User {
	u := model.User{
		Username: req.Username,
		FullName: req.FullName,
		Email:    req.Email,
		IsActive: true,
	}
	if req.BranchID != nil {
		u.BranchID = *req.BranchID
	}
	if req.Department != nil {
		u.Department = *req.Department
	}
	if req.Position != nil {
		u.Position = *req.Position
	}
	return u
}

func generateTempPassword() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("Tmp%s!", hex.EncodeToString(b)[:12])
}
