package mm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

// testUserRepository is a minimal implementation for tests.
type testUserRepository struct {
	pool *pgxpool.Pool
}

func (r *testUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `SELECT id, username, COALESCE(email,''), full_name, COALESCE(department,''), COALESCE(branch_id::text,'') FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.Email, &u.FullName, &u.Department, &u.BranchID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *testUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `SELECT id, username, COALESCE(email,''), full_name FROM users WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.Email, &u.FullName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

var (
	testPool             *pgxpool.Pool
	testInterbankService *InterbankService
	testOMORepoService   *OMORepoService
	testLogger           *zap.Logger
)

// Seed user/role UUIDs (matching 001_seed.sql)
var (
	dealerUserID   = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	deskHeadUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000002")
	directorUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000003")
	branchID       = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	counterpartyID = uuid.MustParse("e0000000-0000-0000-0000-000000000001") // MSB
	bondCatalogID  = uuid.MustParse("bc000000-0000-0000-0000-000000000001") // TD2125068
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	// Start embedded postgres — unique port for MM tests
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-mm")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-mm-test")).
		Port(15435). // Different port from FX (15432), Bond (15433), Limit (15434)
		Database("treasury_mm_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15435/treasury_mm_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}
	testPool = pool

	// Run ALL migrations 001-011
	migrationsDir := filepath.Join("..", "..", "migrations")
	migrationFiles := []string{
		"001_initial.up.sql",
		"002_admin_views.up.sql",
		"003_cancel_flow.up.sql",
		"004_notifications.up.sql",
		"005_fx_brd_gaps.up.sql",
		"006_email_outbox.up.sql",
		"007_attachments.up.sql",
		"008_bond_module.up.sql",
		"009_bond_view_add_counterparty_id.up.sql",
		"010_credit_limits.up.sql",
		"011_money_market.up.sql",
	}

	for _, migFile := range migrationFiles {
		migData, migErr := os.ReadFile(filepath.Join(migrationsDir, migFile))
		if migErr != nil {
			fmt.Printf("Failed to read migration %s: %v\n", migFile, migErr)
			pool.Close()
			pg.Stop()
			os.Exit(1)
		}
		if _, migErr = pool.Exec(ctx, string(migData)); migErr != nil {
			fmt.Printf("Failed to run migration %s: %v\n", migFile, migErr)
			pool.Close()
			pg.Stop()
			os.Exit(1)
		}
	}

	// Run seed data
	seedFiles := []string{
		"seed/001_seed.sql",
		"seed/003_bond_deals_seed.sql", // needed for bond_catalog FK
	}
	for _, sf := range seedFiles {
		seedData, seedErr := os.ReadFile(filepath.Join(migrationsDir, sf))
		if seedErr != nil {
			fmt.Printf("Failed to read seed %s: %v\n", sf, seedErr)
			pool.Close()
			pg.Stop()
			os.Exit(1)
		}
		if _, seedErr = pool.Exec(ctx, string(seedData)); seedErr != nil {
			fmt.Printf("Failed to run seed %s: %v\n", sf, seedErr)
			pool.Close()
			pg.Stop()
			os.Exit(1)
		}
	}

	// Create services
	userRepo := &testUserRepository{pool: pool}
	rbacChecker := security.NewRBACChecker()
	auditLogger := audit.NewLogger(pool, testLogger)

	interbankRepo := NewInterbankRepository(pool)
	testInterbankService = NewInterbankService(interbankRepo, userRepo, rbacChecker, auditLogger, pool, testLogger)

	omoRepoRepo := NewOMORepoRepository(pool)
	testOMORepoService = NewOMORepoService(omoRepoRepo, userRepo, rbacChecker, auditLogger, pool, testLogger)

	code := m.Run()

	pool.Close()
	pg.Stop()
	os.Exit(code)
}

// --- Test Helpers ---

func makeAuthContext(t *testing.T, userID uuid.UUID, roles []string) context.Context {
	t.Helper()
	ctx := context.Background()
	ctx = ctxutil.WithUserID(ctx, userID)
	ctx = ctxutil.WithRoles(ctx, roles)
	ctx = ctxutil.WithBranchID(ctx, branchID.String())
	return ctx
}

func ptrString(s string) *string { return &s }

func today() time.Time {
	return time.Now().Truncate(24 * time.Hour)
}

func futureDate(days int) time.Time {
	return today().Add(time.Duration(days) * 24 * time.Hour)
}

// createTestUser creates a test user and assigns a role.
func createTestUser(t *testing.T, pool *pgxpool.Pool, role string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	username := fmt.Sprintf("test_%s_%s", role, userID.String()[:8])
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, username, password_hash, full_name, email, department, branch_id, is_active)
		VALUES ($1, $2, '$2a$10$dummy', $3, $4, 'Test', $5, true)`,
		userID, username, "Test "+role, username+"@test.com", branchID)
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	var roleID uuid.UUID
	err = pool.QueryRow(context.Background(), "SELECT id FROM roles WHERE code = $1", role).Scan(&roleID)
	if err != nil {
		t.Fatalf("createTestUser role not found %s: %v", role, err)
	}
	_, err = pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`,
		userID, roleID)
	if err != nil {
		t.Fatalf("createTestUser role assign: %v", err)
	}
	return userID
}

// --- Interbank Helpers ---

func makeInterbankRequest() dto.CreateMMInterbankRequest {
	return dto.CreateMMInterbankRequest{
		CounterpartyID:     counterpartyID,
		CurrencyCode:       "VND",
		Direction:          constants.MMDirectionPlace,
		PrincipalAmount:    decimal.NewFromInt(10_000_000_000), // 10 billion VND
		InterestRate:       decimal.NewFromFloat(6.5),
		DayCountConvention: constants.DayCountACT365,
		TradeDate:          today(),
		EffectiveDate:      today(),
		MaturityDate:       futureDate(90),
		Note:               ptrString("Integration test deal"),
	}
}

func makeInterbankRequestUSD() dto.CreateMMInterbankRequest {
	req := makeInterbankRequest()
	req.CurrencyCode = "USD"
	req.PrincipalAmount = decimal.NewFromInt(1_000_000) // 1M USD
	req.InterestRate = decimal.NewFromFloat(5.25)
	return req
}

func makeInterbankRequestTTQT() dto.CreateMMInterbankRequest {
	req := makeInterbankRequestUSD()
	req.RequiresInternationalSettlement = true
	return req
}

func advanceInterbankStatus(t *testing.T, dealID uuid.UUID, status string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`UPDATE mm_interbank_deals SET status = $1 WHERE id = $2`, status, dealID)
	if err != nil {
		t.Fatalf("advanceInterbankStatus failed: %v", err)
	}
}

func createInterbankInStatus(t *testing.T, status string) uuid.UUID {
	t.Helper()
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, err := testInterbankService.CreateDeal(ctx, makeInterbankRequest(), "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("createInterbankInStatus: %v", err)
	}
	if status != constants.StatusOpen {
		advanceInterbankStatus(t, resp.ID, status)
	}
	return resp.ID
}

// approveInterbankFullChain runs the full interbank approval chain (no TTQT):
// DeskHead (L1) → Director (L2) → RiskOfficer (QLRR L1) → RiskHead (QLRR L2) → Accountant (KTTC L1) → ChiefAccountant (KTTC L2) → COMPLETED
func approveInterbankFullChain(t *testing.T, dealID uuid.UUID) {
	t.Helper()

	// Step 1: DeskHead approve → PENDING_TP_REVIEW → PENDING_L2_APPROVAL
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	if err := testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("DeskHead approve (OPEN→PENDING_TP_REVIEW): %v", err)
	}
	// The first approval from OPEN goes to PENDING_TP_REVIEW, need TP to approve again → PENDING_L2
	if err := testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("DeskHead approve (TP_REVIEW→L2): %v", err)
	}

	// Step 2: Director approve → PENDING_RISK_APPROVAL
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	if err := testInterbankService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Director approve: %v", err)
	}

	// Step 3: RiskOfficer approve → PENDING_BOOKING
	riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	riskCtx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
	if err := testInterbankService.ApproveDeal(riskCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("RiskOfficer approve: %v", err)
	}

	// Step 4: Accountant approve → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	if err := testInterbankService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}

	// Step 5: ChiefAccountant approve → COMPLETED (no TTQT)
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	if err := testInterbankService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
}

// --- OMO/Repo Helpers ---

func makeOMORequest() dto.CreateMMOMORepoRequest {
	return dto.CreateMMOMORepoRequest{
		DealSubtype:     constants.MMSubtypeOMO,
		SessionName:     "OMO Session 2026-04-05 #1",
		TradeDate:       today(),
		CounterpartyID:  counterpartyID,
		NotionalAmount:  decimal.NewFromInt(50_000_000_000), // 50 billion VND
		BondCatalogID:   bondCatalogID,
		WinningRate:     decimal.NewFromFloat(4.5),
		TenorDays:       14,
		SettlementDate1: today(),
		SettlementDate2: futureDate(14),
		HaircutPct:      decimal.NewFromFloat(5.0),
		Note:            ptrString("OMO test deal"),
	}
}

func makeRepoKBNNRequest() dto.CreateMMOMORepoRequest {
	req := makeOMORequest()
	req.DealSubtype = constants.MMSubtypeStateRepo
	req.SessionName = "Repo KBNN Session 2026-04-05 #1"
	req.TenorDays = 7
	req.SettlementDate2 = futureDate(7)
	req.Note = ptrString("Repo KBNN test deal")
	return req
}

func advanceOMOStatus(t *testing.T, dealID uuid.UUID, status string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`UPDATE mm_omo_repo_deals SET status = $1 WHERE id = $2`, status, dealID)
	if err != nil {
		t.Fatalf("advanceOMOStatus failed: %v", err)
	}
}

func createOMOInStatus(t *testing.T, status string) uuid.UUID {
	t.Helper()
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, err := testOMORepoService.CreateDeal(ctx, makeOMORequest(), "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("createOMOInStatus: %v", err)
	}
	if status != constants.StatusOpen {
		advanceOMOStatus(t, resp.ID, status)
	}
	return resp.ID
}

// approveOMOFullChain runs the full OMO/Repo approval chain:
// DeskHead (L1) → Director (L2) → Accountant (KTTC L1) → ChiefAccountant (KTTC L2) → COMPLETED
func approveOMOFullChain(t *testing.T, dealID uuid.UUID) {
	t.Helper()

	// Step 1: DeskHead approve → PENDING_L2_APPROVAL
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	if err := testOMORepoService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("DeskHead approve: %v", err)
	}

	// Step 2: Director approve → PENDING_BOOKING
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	if err := testOMORepoService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Director approve: %v", err)
	}

	// Step 3: Accountant approve → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	if err := testOMORepoService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}

	// Step 4: ChiefAccountant approve → COMPLETED
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	if err := testOMORepoService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
}

// ============================================================================
// INTERBANK TESTS
// ============================================================================

func TestInterbankCreateDeal(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeInterbankRequest()

	resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	if resp.Status != constants.StatusOpen {
		t.Errorf("expected status OPEN, got %s", resp.Status)
	}
	if resp.Direction != constants.MMDirectionPlace {
		t.Errorf("expected direction PLACE, got %s", resp.Direction)
	}
	if resp.DealNumber == "" {
		t.Error("expected non-empty deal_number")
	}
	// Verify deal number format: MM-YYYYMMDD-NNNN
	if len(resp.DealNumber) < 3 || resp.DealNumber[:3] != "MM-" {
		t.Errorf("expected deal number starting with MM-, got %s", resp.DealNumber)
	}
	if resp.CurrencyCode != "VND" {
		t.Errorf("expected currency VND, got %s", resp.CurrencyCode)
	}
	if resp.TenorDays != 90 {
		t.Errorf("expected tenor 90 days, got %d", resp.TenorDays)
	}
	// Verify interest calculation
	if resp.InterestAmount.IsZero() {
		t.Error("expected non-zero interest amount")
	}
	if resp.MaturityAmount.LessThanOrEqual(resp.PrincipalAmount) {
		t.Error("expected maturity > principal")
	}
}

func TestInterbankGetDeal(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testInterbankService.GetDeal(ctx, dealID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}
	if resp.ID != dealID {
		t.Errorf("expected ID %s, got %s", dealID, resp.ID)
	}
}

func TestInterbankListDeals(t *testing.T) {
	_ = createInterbankInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	pag := dto.PaginationRequest{Page: 1, PageSize: 20}
	filter := dto.MMInterbankFilter{}
	result, err := testInterbankService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}
	if result.Total == 0 {
		t.Error("expected at least 1 deal")
	}
}

func TestInterbankFullApprovalWorkflow(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeInterbankRequest()
	resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}
	dealID := resp.ID

	// Step 1: DeskHead approve OPEN → PENDING_TP_REVIEW
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("DeskHead approve (OPEN→TP): %v", err)
	}
	deal, _ := testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingTPReview {
		t.Errorf("expected PENDING_TP_REVIEW, got %s", deal.Status)
	}

	// Step 2: DeskHead approve PENDING_TP_REVIEW → PENDING_L2_APPROVAL
	err = testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("DeskHead approve (TP→L2): %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingL2Approval {
		t.Errorf("expected PENDING_L2_APPROVAL, got %s", deal.Status)
	}

	// Step 3: Director approve → PENDING_RISK_APPROVAL
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testInterbankService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director approve: %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingRiskApproval {
		t.Errorf("expected PENDING_RISK_APPROVAL, got %s", deal.Status)
	}

	// Step 4: RiskOfficer approve → PENDING_BOOKING
	riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	riskCtx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
	err = testInterbankService.ApproveDeal(riskCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("RiskOfficer approve: %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingBooking {
		t.Errorf("expected PENDING_BOOKING, got %s", deal.Status)
	}

	// Step 5: Accountant approve → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	err = testInterbankService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingChiefAccountant {
		t.Errorf("expected PENDING_CHIEF_ACCOUNTANT, got %s", deal.Status)
	}

	// Step 6: ChiefAccountant approve → COMPLETED (no TTQT)
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	err = testInterbankService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", deal.Status)
	}
}

func TestInterbankFullApprovalWithTTQT(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeInterbankRequestTTQT()
	resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal TTQT: %v", err)
	}
	dealID := resp.ID

	// DeskHead 2x approve → PENDING_L2
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	// Director → PENDING_RISK
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	testInterbankService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	// Risk → PENDING_BOOKING
	riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	riskCtx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
	testInterbankService.ApproveDeal(riskCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	// Accountant → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	testInterbankService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	// ChiefAccountant → PENDING_SETTLEMENT (because TTQT)
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	err = testInterbankService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
	deal, _ := testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingSettlement {
		t.Errorf("expected PENDING_SETTLEMENT for TTQT deal, got %s", deal.Status)
	}

	// Settlement → COMPLETED
	settleID := createTestUser(t, testPool, constants.RoleSettlementOfficer)
	settleCtx := makeAuthContext(t, settleID, []string{constants.RoleSettlementOfficer})
	err = testInterbankService.ApproveDeal(settleCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Settlement approve: %v", err)
	}
	deal, _ = testInterbankService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusCompleted {
		t.Errorf("expected COMPLETED after settlement, got %s", deal.Status)
	}
}

func TestInterbankSelfApprovalBlocked(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeInterbankRequest()
	resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	err = testInterbankService.ApproveDeal(ctx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrSelfApproval) {
		t.Errorf("expected ErrSelfApproval, got %v", err)
	}
}

func TestInterbankRecallDeal(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusPendingL2Approval)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testInterbankService.RecallDeal(ctx, dealID, "wrong counterparty", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("RecallDeal: %v", err)
	}

	resp, _ := testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN after recall, got %s", resp.Status)
	}
}

func TestInterbankRecallFromAllPendingStates(t *testing.T) {
	pendingStates := []string{
		constants.StatusPendingTPReview,
		constants.StatusPendingL2Approval,
		constants.StatusPendingRiskApproval,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant,
		constants.StatusPendingSettlement,
	}

	for _, status := range pendingStates {
		t.Run(status, func(t *testing.T) {
			dealID := createInterbankInStatus(t, status)
			ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
			err := testInterbankService.RecallDeal(ctx, dealID, "test recall from "+status, "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("RecallDeal from %s: %v", status, err)
			}
			resp, _ := testInterbankService.GetDeal(ctx, dealID)
			if resp.Status != constants.StatusOpen {
				t.Errorf("expected OPEN after recall from %s, got %s", status, resp.Status)
			}
		})
	}
}

func TestInterbankCancelFlow(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusCompleted)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	// Request cancel
	err := testInterbankService.CancelDeal(ctx, dealID, "client withdrew", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CancelDeal: %v", err)
	}
	resp, _ := testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusPendingCancelL1 {
		t.Errorf("expected PENDING_CANCEL_L1, got %s", resp.Status)
	}

	// L1 approve
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testInterbankService.ApproveCancelDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL1: %v", err)
	}
	resp, _ = testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusPendingCancelL2 {
		t.Errorf("expected PENDING_CANCEL_L2, got %s", resp.Status)
	}

	// L2 approve → CANCELLED
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testInterbankService.ApproveCancelDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL2: %v", err)
	}
	resp, _ = testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", resp.Status)
	}
}

func TestInterbankCloneDeal(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusRejected)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testInterbankService.CloneDeal(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal: %v", err)
	}
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN for clone, got %s", resp.Status)
	}
	if resp.ClonedFromID == nil || *resp.ClonedFromID != dealID {
		t.Error("expected clone to reference source deal")
	}
}

func TestInterbankCloneFromVoidedByRisk(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusVoidedByRisk)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testInterbankService.CloneDeal(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal from VOIDED_BY_RISK: %v", err)
	}
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN, got %s", resp.Status)
	}
}

func TestInterbankSoftDelete(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testInterbankService.SoftDelete(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = testInterbankService.GetDeal(ctx, dealID)
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}
}

func TestInterbankDirectorReject(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusPendingL2Approval)

	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err := testInterbankService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("bad deal")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusRejected {
		t.Errorf("expected REJECTED, got %s", resp.Status)
	}
}

func TestInterbankRiskReject(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusPendingRiskApproval)

	riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	riskCtx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})
	err := testInterbankService.ApproveDeal(riskCtx, dealID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("risk too high")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Risk reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusVoidedByRisk {
		t.Errorf("expected VOIDED_BY_RISK, got %s", resp.Status)
	}
}

func TestInterbankValidationErrors(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	// Zero principal
	req := makeInterbankRequest()
	req.PrincipalAmount = decimal.Zero
	_, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero principal, got %v", err)
	}

	// Zero interest rate
	req = makeInterbankRequest()
	req.InterestRate = decimal.Zero
	_, err = testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero interest_rate, got %v", err)
	}

	// Maturity before effective
	req = makeInterbankRequest()
	req.MaturityDate = today().Add(-24 * time.Hour)
	_, err = testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for maturity < effective, got %v", err)
	}

	// Invalid direction
	req = makeInterbankRequest()
	req.Direction = "INVALID"
	_, err = testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for invalid direction, got %v", err)
	}

	// Invalid day count convention
	req = makeInterbankRequest()
	req.DayCountConvention = "INVALID"
	_, err = testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for invalid day_count, got %v", err)
	}
}

func TestInterbankInterestCalculation(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	tests := []struct {
		name       string
		dayCount   string
		currency   string
		principal  int64
		rate       float64
		tenorDays  int
		wantZeroDp bool // VND = 0 decimals
	}{
		{"ACT_365 VND", constants.DayCountACT365, "VND", 10_000_000_000, 6.5, 90, true},
		{"ACT_360 VND", constants.DayCountACT360, "VND", 10_000_000_000, 6.5, 90, true},
		{"ACT_ACT VND", constants.DayCountACTACT, "VND", 10_000_000_000, 6.5, 90, true},
		{"ACT_365 USD", constants.DayCountACT365, "USD", 1_000_000, 5.25, 90, false},
		{"ACT_360 USD", constants.DayCountACT360, "USD", 1_000_000, 5.25, 90, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := makeInterbankRequest()
			req.DayCountConvention = tc.dayCount
			req.CurrencyCode = tc.currency
			req.PrincipalAmount = decimal.NewFromInt(tc.principal)
			req.InterestRate = decimal.NewFromFloat(tc.rate)
			req.MaturityDate = futureDate(tc.tenorDays)

			resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("CreateDeal: %v", err)
			}

			if resp.InterestAmount.IsZero() {
				t.Error("expected non-zero interest")
			}
			if resp.MaturityAmount.LessThanOrEqual(resp.PrincipalAmount) {
				t.Error("expected maturity > principal")
			}

			// VND should have no decimal places
			if tc.wantZeroDp {
				if resp.InterestAmount.Exponent() < 0 {
					t.Errorf("VND interest should be whole number, got %s", resp.InterestAmount.String())
				}
			}
		})
	}
}

func TestInterbankApprovalHistory(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeInterbankRequest()
	resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	// DeskHead approve
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	testInterbankService.ApproveDeal(dhCtx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	entries, err := testInterbankService.GetApprovalHistory(ctx, resp.ID)
	if err != nil {
		t.Fatalf("GetApprovalHistory: %v", err)
	}
	if len(entries) < 1 {
		t.Errorf("expected at least 1 history entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ActionType != "DEALER_SUBMIT" {
		t.Errorf("expected DEALER_SUBMIT, got %s", entries[0].ActionType)
	}
}

func TestInterbankRiskOfficerDataScope(t *testing.T) {
	// Create an OPEN deal (should NOT be visible to risk)
	_ = createInterbankInStatus(t, constants.StatusOpen)

	// Create a PENDING_RISK deal (should be visible)
	_ = createInterbankInStatus(t, constants.StatusPendingRiskApproval)

	riskID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	riskCtx := makeAuthContext(t, riskID, []string{constants.RoleRiskOfficer})

	pag := dto.PaginationRequest{Page: 1, PageSize: 100}
	filter := dto.MMInterbankFilter{}
	result, err := testInterbankService.ListDeals(riskCtx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals: %v", err)
	}

	for _, d := range result.Data {
		if d.Status != constants.StatusPendingRiskApproval && d.Status != constants.StatusVoidedByRisk {
			t.Errorf("risk officer should only see PENDING_RISK_APPROVAL|VOIDED_BY_RISK, got %s in deal %s", d.Status, d.ID)
		}
	}
}

func TestInterbankAccountantDataScope(t *testing.T) {
	_ = createInterbankInStatus(t, constants.StatusOpen)

	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})

	pag := dto.PaginationRequest{Page: 1, PageSize: 100}
	filter := dto.MMInterbankFilter{}
	result, err := testInterbankService.ListDeals(accCtx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals: %v", err)
	}

	for _, d := range result.Data {
		if d.Status == constants.StatusOpen || d.Status == constants.StatusPendingTPReview ||
			d.Status == constants.StatusPendingL2Approval || d.Status == constants.StatusPendingRiskApproval {
			t.Errorf("accountant should not see %s deals, found deal %s", d.Status, d.ID)
		}
	}
}

func TestInterbankAllDirections(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	for _, dir := range constants.AllMMInterbankDirections {
		t.Run(dir, func(t *testing.T) {
			req := makeInterbankRequest()
			req.Direction = dir
			resp, err := testInterbankService.CreateDeal(ctx, req, "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("CreateDeal direction %s: %v", dir, err)
			}
			if resp.Direction != dir {
				t.Errorf("expected direction %s, got %s", dir, resp.Direction)
			}
		})
	}
}

func TestInterbankDeskHeadRejectToOpen(t *testing.T) {
	dealID := createInterbankInStatus(t, constants.StatusPendingTPReview)

	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err := testInterbankService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("not ready")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("DeskHead reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testInterbankService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN after TP reject, got %s", resp.Status)
	}
}

// ============================================================================
// OMO/REPO TESTS
// ============================================================================

func TestOMOCreateDeal(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeOMORequest()

	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal OMO failed: %v", err)
	}

	if resp.Status != constants.StatusOpen {
		t.Errorf("expected status OPEN, got %s", resp.Status)
	}
	if resp.DealSubtype != constants.MMSubtypeOMO {
		t.Errorf("expected subtype OMO, got %s", resp.DealSubtype)
	}
	if resp.DealNumber == "" {
		t.Error("expected non-empty deal_number")
	}
	// OMO deals start with OMO-
	if len(resp.DealNumber) < 4 || resp.DealNumber[:4] != "OMO-" {
		t.Errorf("expected deal number starting with OMO-, got %s", resp.DealNumber)
	}
	if resp.TenorDays != 14 {
		t.Errorf("expected tenor 14, got %d", resp.TenorDays)
	}
}

func TestRepoKBNNCreateDeal(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeRepoKBNNRequest()

	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal Repo KBNN failed: %v", err)
	}

	if resp.DealSubtype != constants.MMSubtypeStateRepo {
		t.Errorf("expected subtype STATE_REPO, got %s", resp.DealSubtype)
	}
	// Repo KBNN deals start with RK-
	if len(resp.DealNumber) < 3 || resp.DealNumber[:3] != "RK-" {
		t.Errorf("expected deal number starting with RK-, got %s", resp.DealNumber)
	}
}

func TestOMOGetDeal(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testOMORepoService.GetDeal(ctx, dealID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}
	if resp.ID != dealID {
		t.Errorf("expected ID %s, got %s", dealID, resp.ID)
	}
}

func TestOMOListDeals(t *testing.T) {
	_ = createOMOInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	pag := dto.PaginationRequest{Page: 1, PageSize: 20}
	filter := dto.MMOMORepoFilter{DealSubtype: constants.MMSubtypeOMO}
	result, err := testOMORepoService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}
	if result.Total == 0 {
		t.Error("expected at least 1 deal")
	}
}

func TestOMOFullApprovalWorkflow(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeOMORequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}
	dealID := resp.ID

	// Step 1: DeskHead approve OPEN → PENDING_L2_APPROVAL
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testOMORepoService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("DeskHead approve: %v", err)
	}
	deal, _ := testOMORepoService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingL2Approval {
		t.Errorf("expected PENDING_L2_APPROVAL, got %s", deal.Status)
	}

	// Step 2: Director approve → PENDING_BOOKING (no risk for OMO)
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testOMORepoService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director approve: %v", err)
	}
	deal, _ = testOMORepoService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingBooking {
		t.Errorf("expected PENDING_BOOKING, got %s", deal.Status)
	}

	// Step 3: Accountant approve → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	err = testOMORepoService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}
	deal, _ = testOMORepoService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingChiefAccountant {
		t.Errorf("expected PENDING_CHIEF_ACCOUNTANT, got %s", deal.Status)
	}

	// Step 4: ChiefAccountant approve → COMPLETED
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	err = testOMORepoService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
	deal, _ = testOMORepoService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", deal.Status)
	}
}

func TestOMOSelfApprovalBlocked(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeOMORequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	err = testOMORepoService.ApproveDeal(ctx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrSelfApproval) {
		t.Errorf("expected ErrSelfApproval, got %v", err)
	}
}

func TestOMORecallDeal(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusPendingL2Approval)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testOMORepoService.RecallDeal(ctx, dealID, "wrong bond catalog", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("RecallDeal: %v", err)
	}
	resp, _ := testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN after recall, got %s", resp.Status)
	}
}

func TestOMORecallFromAllPendingStates(t *testing.T) {
	pendingStates := []string{
		constants.StatusPendingL2Approval,
		constants.StatusPendingBooking,
		constants.StatusPendingChiefAccountant,
	}

	for _, status := range pendingStates {
		t.Run(status, func(t *testing.T) {
			dealID := createOMOInStatus(t, status)
			ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
			err := testOMORepoService.RecallDeal(ctx, dealID, "test recall", "127.0.0.1", "test")
			if err != nil {
				t.Fatalf("RecallDeal from %s: %v", status, err)
			}
			resp, _ := testOMORepoService.GetDeal(ctx, dealID)
			if resp.Status != constants.StatusOpen {
				t.Errorf("expected OPEN after recall from %s, got %s", status, resp.Status)
			}
		})
	}
}

func TestOMOCancelFlow(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusCompleted)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testOMORepoService.CancelDeal(ctx, dealID, "session cancelled", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CancelDeal: %v", err)
	}
	resp, _ := testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusPendingCancelL1 {
		t.Errorf("expected PENDING_CANCEL_L1, got %s", resp.Status)
	}

	// L1 approve
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testOMORepoService.ApproveCancelDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL1: %v", err)
	}
	resp, _ = testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusPendingCancelL2 {
		t.Errorf("expected PENDING_CANCEL_L2, got %s", resp.Status)
	}

	// L2 approve → CANCELLED
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testOMORepoService.ApproveCancelDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL2: %v", err)
	}
	resp, _ = testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", resp.Status)
	}
}

func TestOMOCloneDeal(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusRejected)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testOMORepoService.CloneDeal(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal: %v", err)
	}
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN for clone, got %s", resp.Status)
	}
	if resp.ClonedFromID == nil || *resp.ClonedFromID != dealID {
		t.Error("expected clone to reference source deal")
	}
}

func TestOMOSoftDelete(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusOpen)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testOMORepoService.SoftDelete(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = testOMORepoService.GetDeal(ctx, dealID)
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}
}

func TestOMODirectorReject(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusPendingL2Approval)

	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err := testOMORepoService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("bad session")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusRejected {
		t.Errorf("expected REJECTED, got %s", resp.Status)
	}
}

func TestOMOValidationErrors(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	// Zero notional
	req := makeOMORequest()
	req.NotionalAmount = decimal.Zero
	_, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero notional, got %v", err)
	}

	// Zero tenor
	req = makeOMORequest()
	req.TenorDays = 0
	_, err = testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero tenor, got %v", err)
	}

	// settlement_date_2 before settlement_date_1
	req = makeOMORequest()
	req.SettlementDate2 = today().Add(-24 * time.Hour)
	_, err = testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for dates, got %v", err)
	}

	// Empty session name
	req = makeOMORequest()
	req.SessionName = ""
	_, err = testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for empty session_name, got %v", err)
	}

	// Zero winning rate
	req = makeOMORequest()
	req.WinningRate = decimal.Zero
	_, err = testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero winning_rate, got %v", err)
	}
}

func TestOMOAccountantDataScope(t *testing.T) {
	_ = createOMOInStatus(t, constants.StatusOpen)

	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})

	pag := dto.PaginationRequest{Page: 1, PageSize: 100}
	filter := dto.MMOMORepoFilter{}
	result, err := testOMORepoService.ListDeals(accCtx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals: %v", err)
	}

	for _, d := range result.Data {
		if d.Status == constants.StatusOpen || d.Status == constants.StatusPendingL2Approval {
			t.Errorf("accountant should not see %s deals, found deal %s", d.Status, d.ID)
		}
	}
}

func TestOMOApprovalHistory(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeOMORequest()
	resp, err := testOMORepoService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	// DeskHead approve
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	testOMORepoService.ApproveDeal(dhCtx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")

	entries, err := testOMORepoService.GetApprovalHistory(ctx, resp.ID)
	if err != nil {
		t.Fatalf("GetApprovalHistory: %v", err)
	}
	if len(entries) < 1 {
		t.Errorf("expected at least 1 history entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ActionType != "DESK_HEAD_APPROVE" {
		t.Errorf("expected DESK_HEAD_APPROVE, got %s", entries[0].ActionType)
	}
}

func TestOMOAccountantReject(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusPendingBooking)

	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	err := testOMORepoService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("docs missing")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Accountant reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testOMORepoService.GetDeal(ctx, dealID)
	if resp.Status != constants.StatusVoidedByAccounting {
		t.Errorf("expected VOIDED_BY_ACCOUNTING, got %s", resp.Status)
	}
}

func TestOMOCloneFromVoidedByAccounting(t *testing.T) {
	dealID := createOMOInStatus(t, constants.StatusVoidedByAccounting)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testOMORepoService.CloneDeal(ctx, dealID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal from VOIDED_BY_ACCOUNTING: %v", err)
	}
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN, got %s", resp.Status)
	}
}
