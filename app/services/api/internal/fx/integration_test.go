package fx

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
	"github.com/kienlongbank/treasury-api/internal/repository"
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
		CachePath(filepath.Join(os.TempDir(), "epg-cache-fx")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-fx-test")).
		Port(15432).
		Database("treasury_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	// Connect
	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15432/treasury_test?sslmode=disable"
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

	// Run additional migrations
	for _, migFile := range []string{"003_cancel_flow.up.sql", "005_fx_brd_gaps.up.sql"} {
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

	// Run tests
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

func createTestDeal(t *testing.T, pool *pgxpool.Pool) *model.FxDeal {
	t.Helper()
	deal := &model.FxDeal{
		CounterpartyID: counterpartyID,
		DealType:       constants.FxTypeSpot,
		Direction:      constants.DirectionBuy,
		NotionalAmount: decimal.NewFromInt(100000),
		CurrencyCode:   "USD",
		TradeDate:      time.Now().Truncate(24 * time.Hour),
		Status:         constants.StatusOpen,
		CreatedBy:      dealerUserID,
		Legs: []model.FxDealLeg{
			{
				LegNumber:    1,
				ValueDate:    time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour),
				ExchangeRate: decimal.NewFromFloat(25950.00),
				BuyCurrency:  "VND",
				SellCurrency: "USD",
				BuyAmount:    decimal.NewFromFloat(2595000000),
				SellAmount:   decimal.NewFromInt(100000),
			},
		},
	}
	ctx := context.Background()
	err := insertDealDirect(ctx, pool, deal)
	if err != nil {
		t.Fatalf("createTestDeal failed: %v", err)
	}
	return deal
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, role string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, username, full_name, email, branch_id, department)
		VALUES ($1, $2, $3, $4, $5, 'K.NV')`,
		userID, "test_"+userID.String()[:8], "Test User", "test@klb.com", branchID)
	if err != nil {
		t.Fatalf("createTestUser failed: %v", err)
	}
	// Assign role
	var roleID string
	err = pool.QueryRow(ctx, "SELECT id FROM roles WHERE code = $1", role).Scan(&roleID)
	if err != nil {
		t.Fatalf("role not found: %s: %v", role, err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)", userID, roleID)
	if err != nil {
		t.Fatalf("assign role failed: %v", err)
	}
	return userID
}

func makeCreateRequest() dto.CreateFxDealRequest {
	return dto.CreateFxDealRequest{
		CounterpartyID: counterpartyID,
		DealType:       constants.FxTypeSpot,
		Direction:      constants.DirectionBuy,
		NotionalAmount: decimal.NewFromInt(50000),
		CurrencyCode:   "USD",
		TradeDate:      time.Now().Truncate(24 * time.Hour),
		Legs: []dto.FxDealLegDTO{
			{
				LegNumber:    1,
				ValueDate:    time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour),
				ExchangeRate: decimal.NewFromFloat(25900.00),
				BuyCurrency:  "VND",
				SellCurrency: "USD",
				BuyAmount:    decimal.NewFromFloat(1295000000),
				SellAmount:   decimal.NewFromInt(50000),
			},
		},
	}
}

// --- Tests ---

func TestCreateFxDeal_Spot(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	if resp.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if resp.Status != constants.StatusOpen {
		t.Fatalf("expected status OPEN, got %s", resp.Status)
	}
	if resp.DealType != constants.FxTypeSpot {
		t.Fatalf("expected SPOT, got %s", resp.DealType)
	}
	if len(resp.Legs) != 1 {
		t.Fatalf("expected 1 leg, got %d", len(resp.Legs))
	}
}

func TestCreateFxDeal_Swap(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()
	req.DealType = constants.FxTypeSwap
	req.Direction = constants.DirectionSellBuy
	// Add second leg for swap
	req.Legs = append(req.Legs, dto.FxDealLegDTO{
		LegNumber:    2,
		ValueDate:    time.Now().Add(720 * time.Hour).Truncate(24 * time.Hour),
		ExchangeRate: decimal.NewFromFloat(26100.00),
		BuyCurrency:  "VND",
		SellCurrency: "USD",
		BuyAmount:    decimal.NewFromFloat(1305000000),
		SellAmount:   decimal.NewFromInt(50000),
	})

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal SWAP failed: %v", err)
	}

	if resp.DealType != constants.FxTypeSwap {
		t.Fatalf("expected SWAP, got %s", resp.DealType)
	}
	if len(resp.Legs) != 2 {
		t.Fatalf("expected 2 legs, got %d", len(resp.Legs))
	}
}

func TestCreateFxDeal_ValidationError(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := dto.CreateFxDealRequest{} // missing required fields

	_, err := testService.CreateDeal(ctx, req, "", "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestCreateFxDeal_NegativeAmount(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()
	req.NotionalAmount = decimal.NewFromFloat(-1000)

	_, err := testService.CreateDeal(ctx, req, "", "")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestGetFxDeal(t *testing.T) {
	deal := createTestDeal(t, testPool)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testService.GetDeal(ctx, deal.ID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}
	if resp.ID != deal.ID {
		t.Fatalf("expected ID %s, got %s", deal.ID, resp.ID)
	}
}

func TestGetFxDeal_NotFound(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	_, err := testService.GetDeal(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestListFxDeals_WithFilters(t *testing.T) {
	// Create a few deals
	createTestDeal(t, testPool)
	createTestDeal(t, testPool)

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	status := constants.StatusOpen
	filter := repository.FxDealFilter{Status: &status}
	pag := dto.DefaultPagination()

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}
	if result.Total < 2 {
		t.Fatalf("expected at least 2 deals, got %d", result.Total)
	}
	for _, d := range result.Data {
		if d.Status != constants.StatusOpen {
			t.Fatalf("expected OPEN status, got %s", d.Status)
		}
	}
}

func TestListFxDeals_Pagination(t *testing.T) {
	// Ensure enough deals exist
	for i := 0; i < 5; i++ {
		createTestDeal(t, testPool)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}

	// Page 1
	pag1 := dto.PaginationRequest{Page: 1, PageSize: 2, SortBy: "created_at", SortDir: "desc"}
	r1, err := testService.ListDeals(ctx, filter, pag1)
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}
	if len(r1.Data) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(r1.Data))
	}

	// Page 2
	pag2 := dto.PaginationRequest{Page: 2, PageSize: 2, SortBy: "created_at", SortDir: "desc"}
	r2, err := testService.ListDeals(ctx, filter, pag2)
	if err != nil {
		t.Fatalf("page 2 failed: %v", err)
	}
	if len(r2.Data) < 1 {
		t.Fatalf("expected at least 1 item on page 2, got %d", len(r2.Data))
	}

	// Verify no overlap
	if r1.Data[0].ID == r2.Data[0].ID {
		t.Fatal("page 1 and page 2 returned same first item")
	}
}

func TestUpdateFxDeal_WhenOpen(t *testing.T) {
	deal := createTestDeal(t, testPool)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	newAmount := decimal.NewFromInt(200000)
	req := dto.UpdateFxDealRequest{
		NotionalAmount: &newAmount,
		Version:        deal.Version,
		Legs: []dto.FxDealLegDTO{
			{
				LegNumber:    1,
				ValueDate:    time.Now().Add(48 * time.Hour).Truncate(24 * time.Hour),
				ExchangeRate: decimal.NewFromFloat(25950.00),
				BuyCurrency:  "VND",
				SellCurrency: "USD",
				BuyAmount:    decimal.NewFromFloat(5190000000),
				SellAmount:   decimal.NewFromInt(200000),
			},
		},
	}

	resp, err := testService.UpdateDeal(ctx, deal.ID, req, "", "")
	if err != nil {
		t.Fatalf("UpdateDeal failed: %v", err)
	}
	if !resp.NotionalAmount.Equal(newAmount) {
		t.Fatalf("expected amount %s, got %s", newAmount, resp.NotionalAmount)
	}
}

func TestUpdateFxDeal_WhenApproved(t *testing.T) {
	deal := createTestDeal(t, testPool)
	ctx := context.Background()

	// Move to non-OPEN status
	_, err := testPool.Exec(ctx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingL2Approval, deal.ID)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := dto.UpdateFxDealRequest{Version: deal.Version}

	_, err = testService.UpdateDeal(dealerCtx, deal.ID, req, "", "")
	if err == nil {
		t.Fatal("expected error when editing approved deal")
	}
	if !apperror.Is(err, apperror.ErrDealLocked) {
		t.Fatalf("expected DEAL_LOCKED, got %v", err)
	}
}

func TestApproveDeal_L1(t *testing.T) {
	deal := createTestDeal(t, testPool)
	deskHeadCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})

	req := dto.ApprovalRequest{Action: "APPROVE", Version: 1}
	err := testService.ApproveDeal(deskHeadCtx, deal.ID, req, "", "")
	if err != nil {
		t.Fatalf("ApproveDeal L1 failed: %v", err)
	}

	// Verify status changed
	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testService.GetDeal(dealerCtx, deal.ID)
	if resp.Status != constants.StatusPendingL2Approval {
		t.Fatalf("expected PENDING_L2_APPROVAL, got %s", resp.Status)
	}
}

func TestApproveDeal_SelfApproval(t *testing.T) {
	deal := createTestDeal(t, testPool)

	// dealer tries to approve own deal (even if they had desk_head role)
	selfCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDeskHead})

	req := dto.ApprovalRequest{Action: "APPROVE", Version: 1}
	err := testService.ApproveDeal(selfCtx, deal.ID, req, "", "")
	if err == nil {
		t.Fatal("expected self-approval error")
	}
	if !apperror.Is(err, apperror.ErrSelfApproval) {
		t.Fatalf("expected SELF_APPROVAL, got %v", err)
	}
}

func TestRecallDeal(t *testing.T) {
	deal := createTestDeal(t, testPool)
	ctx := context.Background()

	// Move to PENDING_L2_APPROVAL
	_, err := testPool.Exec(ctx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingL2Approval, deal.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	err = testService.RecallDeal(dealerCtx, deal.ID, "Need to correct rate", "", "")
	if err != nil {
		t.Fatalf("RecallDeal failed: %v", err)
	}

	resp, _ := testService.GetDeal(dealerCtx, deal.ID)
	if resp.Status != constants.StatusOpen {
		t.Fatalf("expected OPEN after recall, got %s", resp.Status)
	}
}

func TestRecallDeal_WrongStatus(t *testing.T) {
	deal := createTestDeal(t, testPool)
	// Deal is OPEN — cannot recall from OPEN
	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	err := testService.RecallDeal(dealerCtx, deal.ID, "some reason", "", "")
	if err == nil {
		t.Fatal("expected error when recalling from OPEN status")
	}
	if !apperror.Is(err, apperror.ErrInvalidTransition) {
		t.Fatalf("expected INVALID_TRANSITION, got %v", err)
	}
}

func TestCloneDeal(t *testing.T) {
	deal := createTestDeal(t, testPool)
	ctx := context.Background()

	// Set to REJECTED so it can be cloned
	_, err := testPool.Exec(ctx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusRejected, deal.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, err := testService.CloneDeal(dealerCtx, deal.ID, "", "")
	if err != nil {
		t.Fatalf("CloneDeal failed: %v", err)
	}
	if resp.ID == deal.ID {
		t.Fatal("clone should have new ID")
	}
	if resp.Status != constants.StatusOpen {
		t.Fatalf("clone should be OPEN, got %s", resp.Status)
	}
}

func TestSoftDelete(t *testing.T) {
	deal := createTestDeal(t, testPool)
	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	err := testService.SoftDelete(dealerCtx, deal.ID, "", "")
	if err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	// Should not be found anymore
	_, err = testService.GetDeal(dealerCtx, deal.ID)
	if err == nil {
		t.Fatal("expected not found after soft delete")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}

	// Should not appear in list
	filter := repository.FxDealFilter{}
	pag := dto.DefaultPagination()
	result, _ := testService.ListDeals(dealerCtx, filter, pag)
	for _, d := range result.Data {
		if d.ID == deal.ID {
			t.Fatal("deleted deal should not appear in list")
		}
	}
}

// === BRD v3 Gap Tests ===

func TestSettlementAmountCalculation(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	// USD/VND pair: settlement = notional × rate = 50000 × 25900 = 1,295,000,000
	if resp.SettlementAmount == nil {
		t.Fatal("expected settlement_amount to be calculated")
	}
	expected := decimal.NewFromInt(50000).Mul(decimal.NewFromFloat(25900))
	if !resp.SettlementAmount.Equal(expected) {
		t.Errorf("settlement_amount = %s, want %s", resp.SettlementAmount, expected)
	}
	if resp.SettlementCurrency == nil || *resp.SettlementCurrency != "VND" {
		t.Errorf("settlement_currency = %v, want VND", resp.SettlementCurrency)
	}
}

func TestTTQTBranching_InternationalGoesToPendingSettlement(t *testing.T) {
	// Create deal with is_international=true (via pay_code_counterparty)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()
	payCode := "SWIFT-INTL-001"
	req.PayCodeCounterparty = &payCode

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}
	if !resp.IsInternational {
		t.Fatal("expected is_international=true when pay_code_counterparty is set")
	}

	// Advance to PENDING_CHIEF_ACCOUNTANT via DB
	bgCtx := context.Background()
	_, err = testPool.Exec(bgCtx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingChiefAccountant, resp.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Chief accountant approves → should go to PENDING_SETTLEMENT (international)
	chiefUserID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	chiefCtx := makeAuthContext(t, chiefUserID, []string{constants.RoleChiefAccountant})
	approveReq := dto.ApprovalRequest{Action: "APPROVE"}
	err = testService.ApproveDeal(chiefCtx, resp.ID, approveReq, "", "")
	if err != nil {
		t.Fatalf("ApproveDeal failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	updated, _ := testService.GetDeal(dealerCtx, resp.ID)
	if updated.Status != constants.StatusPendingSettlement {
		t.Fatalf("expected PENDING_SETTLEMENT for international deal, got %s", updated.Status)
	}
}

func TestTTQTBranching_DomesticGoesToCompleted(t *testing.T) {
	// Create deal with is_international=false (no pay_code_counterparty)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}
	if resp.IsInternational {
		t.Fatal("expected is_international=false when no pay_code_counterparty")
	}

	// Advance to PENDING_CHIEF_ACCOUNTANT via DB
	bgCtx := context.Background()
	_, err = testPool.Exec(bgCtx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingChiefAccountant, resp.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Chief accountant approves → should go to COMPLETED (domestic, skip TTQT)
	chiefUserID := createTestUser(t, testPool, constants.RoleChiefAccountant)
	chiefCtx := makeAuthContext(t, chiefUserID, []string{constants.RoleChiefAccountant})
	approveReq := dto.ApprovalRequest{Action: "APPROVE"}
	err = testService.ApproveDeal(chiefCtx, resp.ID, approveReq, "", "")
	if err != nil {
		t.Fatalf("ApproveDeal failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	updated, _ := testService.GetDeal(dealerCtx, resp.ID)
	if updated.Status != constants.StatusCompleted {
		t.Fatalf("expected COMPLETED for domestic deal, got %s", updated.Status)
	}
}

func TestTPRecall_vs_CVRecall(t *testing.T) {
	// Create deal and advance to PENDING_L2_APPROVAL
	deal := createTestDeal(t, testPool)
	bgCtx := context.Background()
	_, err := testPool.Exec(bgCtx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingL2Approval, deal.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// TP (DeskHead) recall → PENDING_TP_REVIEW
	tpCtx := makeAuthContext(t, deskHeadUserID, []string{constants.RoleDeskHead})
	err = testService.RecallDeal(tpCtx, deal.ID, "Need TP review", "", "")
	if err != nil {
		t.Fatalf("TP RecallDeal failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	resp, _ := testService.GetDeal(dealerCtx, deal.ID)
	if resp.Status != constants.StatusPendingTPReview {
		t.Fatalf("expected PENDING_TP_REVIEW after TP recall, got %s", resp.Status)
	}

	// Now test CV recall: create another deal at PENDING_L2
	deal2 := createTestDeal(t, testPool)
	_, err = testPool.Exec(bgCtx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusPendingL2Approval, deal2.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// CV (Dealer/creator) recall → OPEN
	cvCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	err = testService.RecallDeal(cvCtx, deal2.ID, "CV correction needed", "", "")
	if err != nil {
		t.Fatalf("CV RecallDeal failed: %v", err)
	}

	resp2, _ := testService.GetDeal(dealerCtx, deal2.ID)
	if resp2.Status != constants.StatusOpen {
		t.Fatalf("expected OPEN after CV recall, got %s", resp2.Status)
	}
}

func TestExcludeCancelled(t *testing.T) {
	// Create a deal and cancel it
	deal := createTestDeal(t, testPool)
	bgCtx := context.Background()
	_, err := testPool.Exec(bgCtx, "UPDATE fx_deals SET status = $1 WHERE id = $2",
		constants.StatusCancelled, deal.ID)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	dealerCtx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	pag := dto.DefaultPagination()

	// Default listing (exclude_cancelled=true by default) → cancelled deal hidden
	filter := repository.FxDealFilter{}
	result, err := testService.ListDeals(dealerCtx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}
	for _, d := range result.Data {
		if d.ID == deal.ID {
			t.Fatal("cancelled deal should not appear in default listing")
		}
	}

	// Explicit include: ExcludeStatuses empty → show all
	empty := []string{}
	filterAll := repository.FxDealFilter{ExcludeStatuses: &empty}
	result2, err := testService.ListDeals(dealerCtx, filterAll, pag)
	if err != nil {
		t.Fatalf("ListDeals with all statuses failed: %v", err)
	}
	found := false
	for _, d := range result2.Data {
		if d.ID == deal.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("cancelled deal should appear when exclude_cancelled=false")
	}
}

func TestRateDecimalValidation(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	// USD/VND rate with 3 decimals → should fail
	req := makeCreateRequest()
	req.Legs[0].ExchangeRate = decimal.NewFromFloat(25900.123)

	_, err := testService.CreateDeal(ctx, req, "", "")
	if err == nil {
		t.Fatal("expected validation error for 3-decimal USD/VND rate")
	}
	if !apperror.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}

	// USD/VND rate with 2 decimals → should succeed
	req2 := makeCreateRequest()
	req2.Legs[0].ExchangeRate = decimal.NewFromFloat(25900.50)
	_, err = testService.CreateDeal(ctx, req2, "", "")
	if err != nil {
		t.Fatalf("CreateDeal with valid 2-decimal rate failed: %v", err)
	}
}

func TestNewFieldsRoundTrip(t *testing.T) {
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	req := makeCreateRequest()
	execDate := time.Now().Add(72 * time.Hour).Truncate(24 * time.Hour)
	payKLB := "PAY-KLB-001"
	payCpty := "PAY-CPTY-001"
	req.ExecutionDate = &execDate
	req.PayCodeKLB = &payKLB
	req.PayCodeCounterparty = &payCpty

	resp, err := testService.CreateDeal(ctx, req, "", "")
	if err != nil {
		t.Fatalf("CreateDeal failed: %v", err)
	}

	// Verify fields round-trip
	got, _ := testService.GetDeal(ctx, resp.ID)
	if got.ExecutionDate == nil || got.ExecutionDate.UTC().Truncate(24*time.Hour) != execDate.UTC().Truncate(24*time.Hour) {
		t.Errorf("execution_date mismatch: got %v, want %v", got.ExecutionDate, execDate)
	}
	if got.PayCodeKLB == nil || *got.PayCodeKLB != payKLB {
		t.Errorf("pay_code_klb = %v, want %s", got.PayCodeKLB, payKLB)
	}
	if got.PayCodeCounterparty == nil || *got.PayCodeCounterparty != payCpty {
		t.Errorf("pay_code_counterparty = %v, want %s", got.PayCodeCounterparty, payCpty)
	}
	if !got.IsInternational {
		t.Error("expected is_international=true")
	}
}
