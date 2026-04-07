package bond

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
	testPool    *pgxpool.Pool
	testService *Service
	testLogger  *zap.Logger
)

// Seed user/role UUIDs (matching 001_seed.sql)
var (
	dealerUserID   = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	deskHeadUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000002")
	directorUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000003")
	branchID       = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	counterpartyID = uuid.MustParse("e0000000-0000-0000-0000-000000000001") // MSB
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	// Start embedded postgres
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-bond")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-bond-test")).
		Port(15433). // Different port from FX tests
		Database("treasury_bond_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15433/treasury_bond_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}
	testPool = pool

	// Run ALL migrations 001-008
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
	userRepo := &testUserRepository{pool: pool}
	rbacChecker := security.NewRBACChecker()
	auditLogger := audit.NewLogger(pool, testLogger)
	testService = NewService(repo, userRepo, rbacChecker, auditLogger, pool, testLogger)

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

func makeGoviBuyRequest() dto.CreateBondDealRequest {
	pt := constants.PortfolioHTM
	return dto.CreateBondDealRequest{
		BondCategory:       constants.BondCategoryGovernment,
		TradeDate:          today(),
		ValueDate:          today(),
		Direction:          constants.BondDirectionBuy,
		CounterpartyID:     counterpartyID,
		TransactionType:    constants.BondTxOutright,
		BondCodeManual:     ptrString("TD2326123"),
		Issuer:             "Kho bạc Nhà nước",
		CouponRate:         decimal.NewFromFloat(5.5),
		MaturityDate:       futureDate(365),
		Quantity:           100,
		FaceValue:          decimal.NewFromInt(100000),
		DiscountRate:       decimal.Zero,
		CleanPrice:         decimal.NewFromInt(98500),
		SettlementPrice:    decimal.NewFromInt(99000),
		TotalValue:         decimal.NewFromInt(9900000),
		PortfolioType:      &pt,
		PaymentDate:        today(),
		RemainingTenorDays: 365,
		ConfirmationMethod: constants.ConfirmEmail,
		ContractPreparedBy: constants.ContractInternal,
	}
}

func createTestBondDeal(t *testing.T) *model.BondDeal {
	t.Helper()
	pt := constants.PortfolioHTM
	deal := &model.BondDeal{
		BondCategory:       constants.BondCategoryGovernment,
		TradeDate:          today(),
		ValueDate:          today(),
		Direction:          constants.BondDirectionBuy,
		CounterpartyID:     counterpartyID,
		TransactionType:    constants.BondTxOutright,
		BondCodeManual:     ptrString("TD2326123"),
		Issuer:             "Kho bạc Nhà nước",
		CouponRate:         decimal.NewFromFloat(5.5),
		MaturityDate:       futureDate(365),
		Quantity:           100,
		FaceValue:          decimal.NewFromInt(100000),
		DiscountRate:       decimal.Zero,
		CleanPrice:         decimal.NewFromInt(98500),
		SettlementPrice:    decimal.NewFromInt(99000),
		TotalValue:         decimal.NewFromInt(9900000),
		PortfolioType:      &pt,
		PaymentDate:        today(),
		RemainingTenorDays: 365,
		ConfirmationMethod: constants.ConfirmEmail,
		ContractPreparedBy: constants.ContractInternal,
		Status:             constants.StatusOpen,
		CreatedBy:          dealerUserID,
	}
	ctx := context.Background()
	err := insertDealDirect(ctx, testPool, deal)
	if err != nil {
		t.Fatalf("createTestBondDeal failed: %v", err)
	}
	return deal
}

func advanceDealStatus(t *testing.T, dealID uuid.UUID, status string) {
	t.Helper()
	_, err := testPool.Exec(context.Background(),
		`UPDATE bond_deals SET status = $1 WHERE id = $2`, status, dealID)
	if err != nil {
		t.Fatalf("advanceDealStatus failed: %v", err)
	}
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
	// Assign role — lookup role_id from roles table
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

// createDealInStatus creates a test deal and advances it to the given status.
func createDealInStatus(t *testing.T, status string) *model.BondDeal {
	t.Helper()
	deal := createTestBondDeal(t)
	if status != constants.StatusOpen {
		advanceDealStatus(t, deal.ID, status)
	}
	return deal
}

// approveFullChain runs the full 4-step approval chain:
// DeskHead → Director → Accountant → ChiefAccountant → COMPLETED.
func approveFullChain(t *testing.T, dealID uuid.UUID) {
	t.Helper()

	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	if err := testService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("DeskHead approve: %v", err)
	}

	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	if err := testService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Director approve: %v", err)
	}

	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	if err := testService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}

	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	if err := testService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test"); err != nil {
		t.Fatalf("ChiefAccountant approve: %v", err)
	}
}

// --- Integration Tests ---

func TestCreateDeal(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeGoviBuyRequest()

	resp, err := testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	if resp.Status != constants.StatusOpen {
		t.Errorf("expected status OPEN, got %s", resp.Status)
	}
	if resp.BondCategory != constants.BondCategoryGovernment {
		t.Errorf("expected category GOVERNMENT, got %s", resp.BondCategory)
	}
	if resp.Direction != constants.BondDirectionBuy {
		t.Errorf("expected direction BUY, got %s", resp.Direction)
	}
	if resp.DealNumber == "" {
		t.Error("expected non-empty deal_number")
	}
	// Verify deal number format: G-YYYYMMDD-NNNN
	if resp.DealNumber[0] != 'G' {
		t.Errorf("expected deal number starting with G, got %s", resp.DealNumber)
	}
	if resp.Quantity != 100 {
		t.Errorf("expected quantity 100, got %d", resp.Quantity)
	}
}

func TestGetDeal(t *testing.T) {
	deal := createTestBondDeal(t)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testService.GetDeal(ctx, deal.ID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}

	if resp.ID != deal.ID {
		t.Errorf("expected ID %s, got %s", deal.ID, resp.ID)
	}
	if resp.Issuer != "Kho bạc Nhà nước" {
		t.Errorf("expected issuer 'Kho bạc Nhà nước', got %s", resp.Issuer)
	}
}

func TestListDeals(t *testing.T) {
	_ = createTestBondDeal(t)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	pag := dto.PaginationRequest{Page: 1, PageSize: 20}
	filter := dto.BondDealListFilter{}
	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	if result.Total == 0 {
		t.Error("expected at least 1 deal")
	}
}

func TestFullApprovalWorkflow(t *testing.T) {
	// Create deal
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeGoviBuyRequest()
	resp, err := testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}
	dealID := resp.ID

	// Step 1: DeskHead approve → PENDING_L2_APPROVAL
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testService.ApproveDeal(dhCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("DeskHead approve: %v", err)
	}

	// Verify status
	deal, _ := testService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingL2Approval {
		t.Errorf("expected PENDING_L2_APPROVAL, got %s", deal.Status)
	}

	// Step 2: Director approve → PENDING_BOOKING
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testService.ApproveDeal(dirCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director approve: %v", err)
	}

	deal, _ = testService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingBooking {
		t.Errorf("expected PENDING_BOOKING, got %s", deal.Status)
	}

	// Step 3: Accountant approve → PENDING_CHIEF_ACCOUNTANT
	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})
	err = testService.ApproveDeal(accCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Accountant approve: %v", err)
	}

	deal, _ = testService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusPendingChiefAccountant {
		t.Errorf("expected PENDING_CHIEF_ACCOUNTANT, got %s", deal.Status)
	}

	// Step 4: Chief Accountant approve → COMPLETED (no settlement for Bonds!)
	caID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	caCtx := makeAuthContext(t, caID, []string{constants.RoleChiefAccountant})
	err = testService.ApproveDeal(caCtx, dealID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Chief Accountant approve: %v", err)
	}

	deal, _ = testService.GetDeal(ctx, dealID)
	if deal.Status != constants.StatusCompleted {
		t.Errorf("expected COMPLETED, got %s", deal.Status)
	}
}

func TestSelfApprovalBlocked(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeGoviBuyRequest()
	resp, err := testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal: %v", err)
	}

	// Dealer tries to approve own deal
	err = testService.ApproveDeal(ctx, resp.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrSelfApproval) {
		t.Errorf("expected ErrSelfApproval, got %v", err)
	}
}

func TestRecallDeal(t *testing.T) {
	deal := createDealInStatus(t, constants.StatusPendingL2Approval)

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	err := testService.RecallDeal(ctx, deal.ID, "wrong counterparty", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("RecallDeal: %v", err)
	}

	resp, _ := testService.GetDeal(ctx, deal.ID)
	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN after recall, got %s", resp.Status)
	}
}

func TestCancelFlow(t *testing.T) {
	deal := createDealInStatus(t, constants.StatusCompleted)

	// Request cancel
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	err := testService.CancelDeal(ctx, deal.ID, "client withdrew", "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CancelDeal: %v", err)
	}

	resp, _ := testService.GetDeal(ctx, deal.ID)
	if resp.Status != constants.StatusPendingCancelL1 {
		t.Errorf("expected PENDING_CANCEL_L1, got %s", resp.Status)
	}

	// L1 approve
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testService.ApproveCancelDeal(dhCtx, deal.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL1: %v", err)
	}

	resp, _ = testService.GetDeal(ctx, deal.ID)
	if resp.Status != constants.StatusPendingCancelL2 {
		t.Errorf("expected PENDING_CANCEL_L2, got %s", resp.Status)
	}

	// L2 approve → CANCELLED
	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err = testService.ApproveCancelDeal(dirCtx, deal.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveCancelL2: %v", err)
	}

	resp, _ = testService.GetDeal(ctx, deal.ID)
	if resp.Status != constants.StatusCancelled {
		t.Errorf("expected CANCELLED, got %s", resp.Status)
	}
}

func TestCloneDeal(t *testing.T) {
	deal := createDealInStatus(t, constants.StatusRejected)

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, err := testService.CloneDeal(ctx, deal.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CloneDeal: %v", err)
	}

	if resp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN for clone, got %s", resp.Status)
	}
	if resp.ClonedFromID == nil || *resp.ClonedFromID != deal.ID {
		t.Error("expected clone to reference source deal")
	}
}

func TestSoftDelete(t *testing.T) {
	deal := createTestBondDeal(t)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testService.SoftDelete(ctx, deal.ID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Should not be found via view (deleted_at IS NOT NULL excluded)
	_, err = testService.GetDeal(ctx, deal.ID)
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}
}

func TestDirectorReject(t *testing.T) {
	deal := createDealInStatus(t, constants.StatusPendingL2Approval)

	dirCtx := makeAuthContext(t, directorUserID, []string{constants.RoleDivisionHead})
	err := testService.ApproveDeal(dirCtx, deal.ID, dto.ApprovalRequest{Action: "REJECT", Comment: ptrString("bad deal")}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("Director reject: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testService.GetDeal(ctx, deal.ID)
	if resp.Status != constants.StatusRejected {
		t.Errorf("expected REJECTED, got %s", resp.Status)
	}
}

func TestValidationErrors(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	// Zero quantity
	req := makeGoviBuyRequest()
	req.Quantity = 0
	_, err := testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for zero quantity, got %v", err)
	}

	// BUY without portfolio type
	req = makeGoviBuyRequest()
	req.PortfolioType = nil
	_, err = testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for missing portfolio_type on BUY, got %v", err)
	}

	// maturity_date before payment_date
	req = makeGoviBuyRequest()
	req.MaturityDate = today().Add(-24 * time.Hour)
	_, err = testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Errorf("expected validation error for maturity < payment, got %v", err)
	}
}

func TestInventoryBuyThenSell(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	bondCode := fmt.Sprintf("INV-TEST-%s", uuid.New().String()[:8])

	// First create a buy deal and complete it to get inventory
	pt := constants.PortfolioHTM
	buyReq := makeGoviBuyRequest()
	buyReq.BondCodeManual = &bondCode
	buyReq.PortfolioType = &pt
	buyReq.Quantity = 200
	buyReq.TotalValue = decimal.NewFromInt(19800000)

	buyResp, err := testService.CreateDeal(ctx, buyReq, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal buy: %v", err)
	}

	// Fast-track to COMPLETED and manually update inventory
	advanceDealStatus(t, buyResp.ID, constants.StatusCompleted)
	err = testService.repo.IncrementInventory(ctx, bondCode, constants.BondCategoryGovernment, constants.PortfolioHTM, 200, dealerUserID)
	if err != nil {
		t.Fatalf("IncrementInventory: %v", err)
	}

	// Verify inventory
	available, err := testService.repo.CheckInventory(ctx, bondCode, constants.BondCategoryGovernment, constants.PortfolioHTM)
	if err != nil {
		t.Fatalf("CheckInventory: %v", err)
	}
	if available != 200 {
		t.Errorf("expected inventory 200, got %d", available)
	}

	// Now sell 50
	sellReq := makeGoviBuyRequest()
	sellReq.Direction = constants.BondDirectionSell
	sellReq.BondCodeManual = &bondCode
	sellReq.PortfolioType = &pt
	sellReq.Quantity = 50
	sellReq.TotalValue = decimal.NewFromInt(4950000)

	sellResp, err := testService.CreateDeal(ctx, sellReq, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal sell: %v", err)
	}

	if sellResp.Status != constants.StatusOpen {
		t.Errorf("expected OPEN, got %s", sellResp.Status)
	}

	// Try to sell more than available
	oversellReq := makeGoviBuyRequest()
	oversellReq.Direction = constants.BondDirectionSell
	oversellReq.BondCodeManual = &bondCode
	oversellReq.PortfolioType = &pt
	oversellReq.Quantity = 999
	oversellReq.TotalValue = decimal.NewFromInt(98901000)

	_, err = testService.CreateDeal(ctx, oversellReq, "127.0.0.1", "test")
	if !apperror.Is(err, apperror.ErrInsufficientInventory) {
		t.Errorf("expected ErrInsufficientInventory for oversell, got %v", err)
	}
}

func TestApprovalHistory(t *testing.T) {
	deal := createTestBondDeal(t)

	// Approve to generate history
	dhCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err := testService.ApproveDeal(dhCtx, deal.ID, dto.ApprovalRequest{Action: "APPROVE"}, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("ApproveDeal: %v", err)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	entries, err := testService.GetApprovalHistory(ctx, deal.ID)
	if err != nil {
		t.Fatalf("GetApprovalHistory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].ActionType != "DESK_HEAD_APPROVE" {
		t.Errorf("expected DESK_HEAD_APPROVE, got %s", entries[0].ActionType)
	}
}

func TestFIDealNumberPrefix(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeGoviBuyRequest()
	req.BondCategory = constants.BondCategoryFinancialInstitution

	resp, err := testService.CreateDeal(ctx, req, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("CreateDeal FI: %v", err)
	}

	if resp.DealNumber[0] != 'F' {
		t.Errorf("expected deal number starting with F for FI, got %s", resp.DealNumber)
	}
}

func TestAccountantDataScope(t *testing.T) {
	// Accountant should only see PENDING_BOOKING and beyond
	_ = createTestBondDeal(t) // Creates an OPEN deal

	accID := createTestUser(t, testPool, constants.RoleAccountant)
	accCtx := makeAuthContext(t, accID, []string{constants.RoleAccountant})

	pag := dto.PaginationRequest{Page: 1, PageSize: 20}
	filter := dto.BondDealListFilter{}
	result, err := testService.ListDeals(accCtx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals: %v", err)
	}

	// OPEN deals should NOT be visible to accountant
	for _, d := range result.Data {
		if d.Status == constants.StatusOpen {
			t.Errorf("accountant should not see OPEN deals, found deal %s", d.ID)
		}
	}
}
