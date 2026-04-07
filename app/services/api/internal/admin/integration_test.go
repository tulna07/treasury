package admin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

var (
	testPool    *pgxpool.Pool
	testService *Service
	testLogger  *zap.Logger
)

// Seed user UUIDs (matching 001_seed.sql)
var (
	adminUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000010")
	branchID    = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-admin")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-admin-test")).
		Port(15433).
		Database("treasury_admin_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15433/treasury_admin_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}
	testPool = pool

	// Run migrations
	migrationsDir := filepath.Join("..", "..", "migrations")
	migrationUp, err := os.ReadFile(filepath.Join(migrationsDir, "001_initial.up.sql"))
	if err != nil {
		fmt.Printf("Failed to read migration: %v\n", err)
		pool.Close()
		pg.Stop()
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, string(migrationUp)); err != nil {
		fmt.Printf("Failed to run migration: %v\n", err)
		pool.Close()
		pg.Stop()
		os.Exit(1)
	}

	// Run views migration
	viewsMigration, err := os.ReadFile(filepath.Join(migrationsDir, "002_admin_views.up.sql"))
	if err == nil {
		pool.Exec(ctx, string(viewsMigration))
	}

	// Run seed
	seedFile, err := os.ReadFile(filepath.Join(migrationsDir, "seed", "001_seed.sql"))
	if err != nil {
		fmt.Printf("Failed to read seed: %v\n", err)
		pool.Close()
		pg.Stop()
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, string(seedFile)); err != nil {
		fmt.Printf("Failed to run seed: %v\n", err)
		pool.Close()
		pg.Stop()
		os.Exit(1)
	}

	// Create service
	repo := NewRepository(pool)
	rbacChecker := security.NewRBACChecker()
	auditLogger := audit.NewLogger(pool, testLogger)
	testService = NewService(repo, rbacChecker, auditLogger, testLogger)

	code := m.Run()

	pool.Close()
	pg.Stop()
	os.Exit(code)
}

func makeAdminContext(t *testing.T) context.Context {
	t.Helper()
	ctx := context.Background()
	ctx = ctxutil.WithUserID(ctx, adminUserID)
	ctx = ctxutil.WithRoles(ctx, []string{constants.RoleAdmin})
	ctx = ctxutil.WithBranchID(ctx, branchID.String())
	return ctx
}

func TestCreateUser(t *testing.T) {
	ctx := makeAdminContext(t)
	req := dto.CreateUserRequest{
		Username: "test_user_" + uuid.New().String()[:8],
		FullName: "Test User Create",
		Email:    "create_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}

	resp, err := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if resp.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if resp.Username != req.Username {
		t.Fatalf("expected username %s, got %s", req.Username, resp.Username)
	}
	if !resp.IsActive {
		t.Fatal("expected user to be active")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	ctx := makeAdminContext(t)
	username := "dup_user_" + uuid.New().String()[:8]
	req := dto.CreateUserRequest{
		Username: username,
		FullName: "Dup User",
		Email:    "dup_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}

	_, err := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	req.Email = "dup2_" + uuid.New().String()[:8] + "@test.com"
	_, err = testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected conflict error for duplicate username")
	}
	if !apperror.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	ctx := makeAdminContext(t)
	pag := dto.DefaultPagination()

	result, err := testService.ListUsers(ctx, dto.UserFilter{}, pag)
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if result.Total < 1 {
		t.Fatalf("expected at least 1 user, got %d", result.Total)
	}
}

func TestListUsers_FilterByDepartment(t *testing.T) {
	ctx := makeAdminContext(t)
	dept := "K.NV"
	filter := dto.UserFilter{Department: &dept}

	result, err := testService.ListUsers(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("ListUsers with dept filter failed: %v", err)
	}
	// Seed data has K.NV users
	if result.Total < 1 {
		t.Fatalf("expected at least 1 K.NV user, got %d", result.Total)
	}
}

func TestGetUser(t *testing.T) {
	ctx := makeAdminContext(t)

	// Create a user first
	req := dto.CreateUserRequest{
		Username: "get_user_" + uuid.New().String()[:8],
		FullName: "Get User Test",
		Email:    "get_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}
	created, _ := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")

	resp, err := testService.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if resp.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, resp.ID)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	ctx := makeAdminContext(t)
	_, err := testService.GetUser(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestUpdateUser(t *testing.T) {
	ctx := makeAdminContext(t)

	req := dto.CreateUserRequest{
		Username: "upd_user_" + uuid.New().String()[:8],
		FullName: "Before Update",
		Email:    "upd_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}
	created, _ := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")

	newName := "After Update"
	updReq := dto.UpdateUserRequest{FullName: &newName}
	resp, err := testService.UpdateUser(ctx, created.ID, updReq, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("UpdateUser failed: %v", err)
	}
	if resp.FullName != newName {
		t.Fatalf("expected name %s, got %s", newName, resp.FullName)
	}
}

func TestLockUnlockUser(t *testing.T) {
	ctx := makeAdminContext(t)

	req := dto.CreateUserRequest{
		Username: "lock_user_" + uuid.New().String()[:8],
		FullName: "Lock Test",
		Email:    "lock_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}
	created, _ := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")

	// Lock
	err := testService.LockUser(ctx, created.ID, "suspicious activity", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("LockUser failed: %v", err)
	}

	user, _ := testService.GetUser(ctx, created.ID)
	if user.IsActive {
		t.Fatal("expected user to be locked (inactive)")
	}

	// Unlock
	err = testService.UnlockUser(ctx, created.ID, "verified clean", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("UnlockUser failed: %v", err)
	}

	user, _ = testService.GetUser(ctx, created.ID)
	if !user.IsActive {
		t.Fatal("expected user to be unlocked (active)")
	}
}

func TestLockUser_SelfLock(t *testing.T) {
	ctx := makeAdminContext(t)
	err := testService.LockUser(ctx, adminUserID, "testing self-lock", "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error when locking self")
	}
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	ctx := makeAdminContext(t)

	req := dto.CreateUserRequest{
		Username: "pwd_user_" + uuid.New().String()[:8],
		FullName: "Reset Pwd Test",
		Email:    "pwd_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}
	created, _ := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")

	tempPwd, err := testService.ResetPassword(ctx, created.ID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}
	if tempPwd == "" {
		t.Fatal("expected non-empty temp password")
	}
	if len(tempPwd) < 10 {
		t.Fatalf("temp password too short: %d chars", len(tempPwd))
	}
}

func TestAssignAndRevokeRole(t *testing.T) {
	ctx := makeAdminContext(t)

	req := dto.CreateUserRequest{
		Username: "role_user_" + uuid.New().String()[:8],
		FullName: "Role Test",
		Email:    "role_" + uuid.New().String()[:8] + "@test.com",
		Password: "TestPass123!",
	}
	created, _ := testService.CreateUser(ctx, req, "127.0.0.1", "test-agent")

	// Assign role
	assignReq := dto.AssignRoleRequest{RoleCode: constants.RoleDealer, Reason: "new dealer"}
	err := testService.AssignRole(ctx, created.ID, assignReq, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	// Verify role assigned
	user, _ := testService.GetUser(ctx, created.ID)
	found := false
	for _, r := range user.Roles {
		if r == constants.RoleDealer {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DEALER role, got %v", user.Roles)
	}

	// Revoke role
	err = testService.RevokeRole(ctx, created.ID, constants.RoleDealer, "no longer needed", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("RevokeRole failed: %v", err)
	}

	user, _ = testService.GetUser(ctx, created.ID)
	for _, r := range user.Roles {
		if r == constants.RoleDealer {
			t.Fatal("expected DEALER role to be revoked")
		}
	}
}

func TestListRoles(t *testing.T) {
	ctx := makeAdminContext(t)

	roles, err := testService.ListRoles(ctx)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) < 5 {
		t.Fatalf("expected at least 5 roles, got %d", len(roles))
	}
}

func TestGetRolePermissions(t *testing.T) {
	ctx := makeAdminContext(t)

	result, err := testService.GetRolePermissions(ctx, constants.RoleDealer)
	if err != nil {
		t.Fatalf("GetRolePermissions failed: %v", err)
	}
	if result.RoleCode != constants.RoleDealer {
		t.Fatalf("expected role code %s, got %s", constants.RoleDealer, result.RoleCode)
	}
	if len(result.Permissions) < 3 {
		t.Fatalf("expected at least 3 permissions for DEALER, got %d", len(result.Permissions))
	}
}

func TestGetRolePermissions_NotFound(t *testing.T) {
	ctx := makeAdminContext(t)

	_, err := testService.GetRolePermissions(ctx, "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}
