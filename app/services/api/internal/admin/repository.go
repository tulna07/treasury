package admin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

type pgxRepository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new admin user repository backed by pgx.
func NewRepository(pool *pgxpool.Pool) repository.AdminUserRepository {
	return &pgxRepository{pool: pool}
}

func (r *pgxRepository) Create(ctx context.Context, user *model.User, passwordHash string) error {
	user.ID = uuid.New()
	var branchID *uuid.UUID
	if user.BranchID != "" {
		id, err := uuid.Parse(user.BranchID)
		if err == nil {
			branchID = &id
		}
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, username, full_name, email, password_hash, branch_id, department, position)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`,
		user.ID, user.Username, user.FullName, user.Email, passwordHash,
		branchID, nullStr(user.Department), nullStr(user.Position),
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "uq_users_username") {
			return apperror.New(apperror.ErrConflict, "username already exists")
		}
		return apperror.Wrap(err, apperror.ErrInternal, "failed to create user")
	}
	return nil
}

func (r *pgxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AdminUser, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.full_name, u.email,
		       COALESCE(u.branch_id::text, '') AS branch_id,
		       COALESCE(b.name, '') AS branch_name,
		       COALESCE(u.department, '') AS department,
		       COALESCE(u.position, '') AS position,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN branches b ON u.branch_id = b.id
		WHERE u.id = $1 AND u.deleted_at IS NULL`, id)

	var user model.AdminUser
	err := row.Scan(
		&user.ID, &user.Username, &user.FullName, &user.Email,
		&user.BranchID, &user.BranchName,
		&user.Department, &user.Position,
		&user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperror.New(apperror.ErrNotFound, "user not found")
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query user")
	}

	// Load roles
	roles, roleNames, err := r.getUserRolesDetailed(ctx, id)
	if err != nil {
		return nil, err
	}
	user.RoleCodes = roles
	user.RoleNames = roleNames

	return &user, nil
}

func (r *pgxRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.full_name, u.password_hash,
		       COALESCE(u.branch_id::text, '') AS branch_id,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at,
		       COALESCE(b.name, '') AS branch_name,
		       COALESCE(u.department, '') AS department,
		       COALESCE(u.position, '') AS position
		FROM users u
		LEFT JOIN branches b ON u.branch_id = b.id
		WHERE u.username = $1 AND u.deleted_at IS NULL`, username)

	var u model.User
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.PasswordHash,
		&u.BranchID, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&u.BranchName, &u.Department, &u.Position,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query user")
	}
	return &u, nil
}

func (r *pgxRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.full_name, '',
		       COALESCE(u.branch_id::text, '') AS branch_id,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at,
		       '', '', ''
		FROM users u
		WHERE u.email = $1 AND u.deleted_at IS NULL`, email)

	var u model.User
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.FullName, &u.PasswordHash,
		&u.BranchID, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		&u.BranchName, &u.Department, &u.Position,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to query user by email")
	}
	return &u, nil
}

func (r *pgxRepository) List(ctx context.Context, filter dto.UserFilter, pag dto.PaginationRequest) ([]model.AdminUser, int64, error) {
	conditions, args, argIdx := r.buildUserFilterConditions(filter)

	whereClause := "u.deleted_at IS NULL"
	if len(conditions) > 0 {
		whereClause += " AND " + strings.Join(conditions, " AND ")
	}

	// Count
	var total int64
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM users u
		LEFT JOIN branches b ON u.branch_id = b.id
		WHERE %s`, whereClause)
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to count users")
	}

	// Data
	sortCol := "u.created_at"
	allowedSorts := map[string]string{
		"created_at": "u.created_at", "username": "u.username",
		"full_name": "u.full_name", "department": "u.department",
	}
	if col, ok := allowedSorts[pag.SortBy]; ok {
		sortCol = col
	}
	sortDir := "DESC"
	if strings.EqualFold(pag.SortDir, "asc") {
		sortDir = "ASC"
	}

	dataQuery := fmt.Sprintf(`
		SELECT u.id, u.username, u.full_name, u.email,
		       COALESCE(u.branch_id::text, '') AS branch_id,
		       COALESCE(b.name, '') AS branch_name,
		       COALESCE(u.department, '') AS department,
		       COALESCE(u.position, '') AS position,
		       u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN branches b ON u.branch_id = b.id
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause, sortCol, sortDir, argIdx, argIdx+1)

	args = append(args, pag.PageSize, pag.Offset())

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to query users")
	}
	defer rows.Close()

	var users []model.AdminUser
	for rows.Next() {
		var u model.AdminUser
		err := rows.Scan(
			&u.ID, &u.Username, &u.FullName, &u.Email,
			&u.BranchID, &u.BranchName,
			&u.Department, &u.Position,
			&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, apperror.Wrap(err, apperror.ErrInternal, "failed to scan user")
		}

		roles, roleNames, _ := r.getUserRolesDetailed(ctx, u.ID)
		u.RoleCodes = roles
		u.RoleNames = roleNames
		if u.RoleCodes == nil {
			u.RoleCodes = []string{}
		}
		if u.RoleNames == nil {
			u.RoleNames = []string{}
		}

		users = append(users, u)
	}

	return users, total, nil
}

func (r *pgxRepository) Update(ctx context.Context, id uuid.UUID, req dto.UpdateUserRequest) error {
	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.FullName != nil {
		sets = append(sets, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, *req.FullName)
		argIdx++
	}
	if req.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *req.Email)
		argIdx++
	}
	if req.BranchID != nil {
		if *req.BranchID == "" {
			sets = append(sets, "branch_id = NULL")
		} else {
			sets = append(sets, fmt.Sprintf("branch_id = $%d", argIdx))
			branchUUID, _ := uuid.Parse(*req.BranchID)
			args = append(args, branchUUID)
			argIdx++
		}
	}
	if req.Department != nil {
		sets = append(sets, fmt.Sprintf("department = $%d", argIdx))
		args = append(args, *req.Department)
		argIdx++
	}
	if req.Position != nil {
		sets = append(sets, fmt.Sprintf("position = $%d", argIdx))
		args = append(args, *req.Position)
		argIdx++
	}

	if len(sets) == 0 {
		return apperror.New(apperror.ErrValidation, "no fields to update")
	}

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL",
		strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "uq_users") {
			return apperror.New(apperror.ErrConflict, "email already exists")
		}
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update user")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "user not found")
	}
	return nil
}

func (r *pgxRepository) SetActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET is_active = $1 WHERE id = $2 AND deleted_at IS NULL",
		isActive, id)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update user active status")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "user not found")
	}
	return nil
}

func (r *pgxRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET password_hash = $1 WHERE id = $2 AND deleted_at IS NULL",
		passwordHash, id)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to update password")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "user not found")
	}
	return nil
}

func (r *pgxRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.code FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 ORDER BY r.code`, userID)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get user roles")
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan role")
		}
		roles = append(roles, code)
	}
	return roles, nil
}

func (r *pgxRepository) AssignRole(ctx context.Context, userID uuid.UUID, roleCode string, grantedBy uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, granted_by)
		SELECT $1, r.id, $3
		FROM roles r WHERE r.code = $2
		ON CONFLICT (user_id, role_id) DO NOTHING`, userID, roleCode, grantedBy)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to assign role")
	}
	if tag.RowsAffected() == 0 {
		// Check if role exists
		var exists bool
		_ = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM roles WHERE code = $1)", roleCode).Scan(&exists)
		if !exists {
			return apperror.New(apperror.ErrNotFound, "role not found: "+roleCode)
		}
		return apperror.New(apperror.ErrConflict, "role already assigned")
	}
	return nil
}

func (r *pgxRepository) RemoveRole(ctx context.Context, userID uuid.UUID, roleCode string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE code = $2)`,
		userID, roleCode)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to remove role")
	}
	if tag.RowsAffected() == 0 {
		return apperror.New(apperror.ErrNotFound, "role not assigned to user")
	}
	return nil
}

func (r *pgxRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, COALESCE(description, ''), scope
		FROM roles ORDER BY code`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to list roles")
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.Scope); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan role")
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *pgxRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, COALESCE(description, '')
		FROM permissions ORDER BY code`)
	if err != nil {
		return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to list permissions")
	}
	defer rows.Close()

	var perms []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description); err != nil {
			return nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan permission")
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *pgxRepository) GetRolePermissions(ctx context.Context, roleCode string) ([]string, string, error) {
	// Get role name
	var roleName string
	err := r.pool.QueryRow(ctx, "SELECT name FROM roles WHERE code = $1", roleCode).Scan(&roleName)
	if err != nil {
		return nil, "", apperror.Wrap(err, apperror.ErrNotFound, "role not found: "+roleCode)
	}

	// Get permission codes
	rows, err := r.pool.Query(ctx, `
		SELECT p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN roles r ON r.id = rp.role_id
		WHERE r.code = $1
		ORDER BY p.code`, roleCode)
	if err != nil {
		return nil, "", apperror.Wrap(err, apperror.ErrInternal, "failed to get role permissions")
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, "", apperror.Wrap(err, apperror.ErrInternal, "failed to scan permission")
		}
		perms = append(perms, code)
	}
	if perms == nil {
		perms = []string{}
	}
	return perms, roleName, nil
}

func (r *pgxRepository) UpdateRolePermissions(ctx context.Context, roleCode string, permCodes []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	// Get role ID from code
	var roleID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT id FROM roles WHERE code = $1", roleCode).Scan(&roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return apperror.New(apperror.ErrNotFound, "role not found: "+roleCode)
		}
		return apperror.Wrap(err, apperror.ErrInternal, "failed to get role")
	}

	// Delete existing permissions
	_, err = tx.Exec(ctx, "DELETE FROM role_permissions WHERE role_id = $1", roleID)
	if err != nil {
		return apperror.Wrap(err, apperror.ErrInternal, "failed to delete existing permissions")
	}

	// Insert new permissions
	for _, code := range permCodes {
		tag, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT $1, id FROM permissions WHERE code = $2`,
			roleID, code)
		if err != nil {
			return apperror.Wrap(err, apperror.ErrInternal, "failed to insert permission: "+code)
		}
		if tag.RowsAffected() == 0 {
			return apperror.New(apperror.ErrValidation, "invalid permission code: "+code)
		}
	}

	return tx.Commit(ctx)
}

// --- helpers ---

func (r *pgxRepository) getUserRolesDetailed(ctx context.Context, userID uuid.UUID) ([]string, []string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.code, r.name FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1 ORDER BY r.code`, userID)
	if err != nil {
		return nil, nil, apperror.Wrap(err, apperror.ErrInternal, "failed to get user roles")
	}
	defer rows.Close()

	var codes, names []string
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, nil, apperror.Wrap(err, apperror.ErrInternal, "failed to scan role")
		}
		codes = append(codes, code)
		names = append(names, name)
	}
	return codes, names, nil
}

func (r *pgxRepository) buildUserFilterConditions(filter dto.UserFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Department != nil {
		conditions = append(conditions, fmt.Sprintf("u.department = $%d", argIdx))
		args = append(args, *filter.Department)
		argIdx++
	}
	if filter.BranchID != nil {
		branchUUID, err := uuid.Parse(*filter.BranchID)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("u.branch_id = $%d", argIdx))
			args = append(args, branchUUID)
			argIdx++
		}
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("u.is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}
	if filter.Role != nil {
		conditions = append(conditions, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM user_roles ur JOIN roles rl ON ur.role_id = rl.id
			WHERE ur.user_id = u.id AND rl.code = $%d)`, argIdx))
		args = append(args, *filter.Role)
		argIdx++
	}
	if filter.Search != nil {
		conditions = append(conditions, fmt.Sprintf(
			"(u.username ILIKE $%d OR u.full_name ILIKE $%d OR u.email ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	return conditions, args, argIdx
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ repository.AdminUserRepository = (*pgxRepository)(nil)
