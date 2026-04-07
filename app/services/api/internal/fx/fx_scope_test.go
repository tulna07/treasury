package fx

// Tests for:
// - Resource Owner / Role-Based Data Scope (Issue 3)
// - Keyset Pagination (Issue 2)
// - View Performance (Issue 1)
//
// Shares TestMain from integration_test.go (embedded postgres).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/constants"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

// ============================================================================
// Helper: create deal with specific status (for scope tests)
// ============================================================================

func createDealWithStatus(t *testing.T, status string) *model.FxDeal {
	t.Helper()
	deal := createTestDeal(t, testPool)
	if status != constants.StatusOpen {
		advanceDealStatus(t, deal.ID, status)
	}
	return deal
}

func createDealWithStatusAndDate(t *testing.T, status string, tradeDate time.Time) *model.FxDeal {
	t.Helper()
	deal := &model.FxDeal{
		CounterpartyID: counterpartyID,
		DealType:       constants.FxTypeSpot,
		Direction:      constants.DirectionBuy,
		NotionalAmount: decimal.NewFromInt(100000),
		CurrencyCode:   "USD",
		TradeDate:      tradeDate,
		Status:         constants.StatusOpen,
		CreatedBy:      dealerUserID,
		Legs: []model.FxDealLeg{
			{
				LegNumber:    1,
				ValueDate:    tradeDate.Add(48 * time.Hour),
				ExchangeRate: decimal.NewFromFloat(25950.00),
				BuyCurrency:  "VND",
				SellCurrency: "USD",
				BuyAmount:    decimal.NewFromFloat(2595000000),
				SellAmount:   decimal.NewFromInt(100000),
			},
		},
	}
	ctx := context.Background()
	err := insertDealDirect(ctx, testPool, deal)
	if err != nil {
		t.Fatalf("createDealWithStatusAndDate failed: %v", err)
	}
	if status != constants.StatusOpen {
		advanceDealStatus(t, deal.ID, status)
	}
	return deal
}

// ============================================================================
// Resource Owner / Data Scope Tests
// ============================================================================

func TestListDeals_DealerSeesAll(t *testing.T) {
	// Create deals in various statuses
	createDealWithStatus(t, constants.StatusOpen)
	createDealWithStatus(t, constants.StatusPendingL2Approval)
	createDealWithStatus(t, constants.StatusPendingBooking)
	createDealWithStatus(t, constants.StatusCompleted)

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 100}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}
	// Dealer should see ALL deals
	if result.Total < 4 {
		t.Fatalf("dealer should see all deals, got total=%d", result.Total)
	}

	// Verify various statuses are present
	statusMap := make(map[string]bool)
	for _, d := range result.Data {
		statusMap[d.Status] = true
	}
	if !statusMap[constants.StatusOpen] {
		t.Fatal("dealer should see OPEN deals")
	}
	if !statusMap[constants.StatusPendingBooking] {
		t.Fatal("dealer should see PENDING_BOOKING deals")
	}
}

func TestListDeals_AccountantOnlyFromBooking(t *testing.T) {
	// Create deals in various statuses
	createDealWithStatus(t, constants.StatusOpen)
	createDealWithStatus(t, constants.StatusPendingL2Approval)
	pendingBooking := createDealWithStatus(t, constants.StatusPendingBooking)
	completed := createDealWithStatus(t, constants.StatusCompleted)

	acctUserID := createTestUser(t, testPool, constants.RoleAccountant)
	ctx := makeAuthContext(t, acctUserID, []string{constants.RoleAccountant})
	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 100}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	// Accountant should NOT see OPEN or PENDING_L2_APPROVAL
	for _, d := range result.Data {
		if d.Status == constants.StatusOpen || d.Status == constants.StatusPendingL2Approval {
			t.Fatalf("accountant should NOT see %s deals", d.Status)
		}
	}

	// Should see PENDING_BOOKING and COMPLETED
	foundBooking := false
	foundCompleted := false
	for _, d := range result.Data {
		if d.ID == pendingBooking.ID {
			foundBooking = true
		}
		if d.ID == completed.ID {
			foundCompleted = true
		}
	}
	if !foundBooking {
		t.Fatal("accountant should see PENDING_BOOKING deals")
	}
	if !foundCompleted {
		t.Fatal("accountant should see COMPLETED deals")
	}
}

func TestListDeals_SettlementOnlyPendingToday(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	// Create deals
	createDealWithStatus(t, constants.StatusCompleted)
	pendingToday := createDealWithStatusAndDate(t, constants.StatusPendingSettlement, today)
	_ = createDealWithStatusAndDate(t, constants.StatusPendingSettlement, yesterday) // yesterday - should be hidden

	settlementUserID := createTestUser(t, testPool, constants.RoleSettlementOfficer)
	ctx := makeAuthContext(t, settlementUserID, []string{constants.RoleSettlementOfficer})
	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 100}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	// Should NOT see COMPLETED
	for _, d := range result.Data {
		if d.Status == constants.StatusCompleted {
			t.Fatal("settlement officer should NOT see COMPLETED deals")
		}
		if d.Status != constants.StatusPendingSettlement {
			t.Fatalf("settlement officer should only see PENDING_SETTLEMENT, got %s", d.Status)
		}
	}

	// Should see PENDING_SETTLEMENT today
	found := false
	for _, d := range result.Data {
		if d.ID == pendingToday.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("settlement officer should see today's PENDING_SETTLEMENT deals")
	}
}

func TestListDeals_RiskOfficerSeesNothing(t *testing.T) {
	// Create deals in various statuses
	createDealWithStatus(t, constants.StatusOpen)
	createDealWithStatus(t, constants.StatusCompleted)

	riskUserID := createTestUser(t, testPool, constants.RoleRiskOfficer)
	ctx := makeAuthContext(t, riskUserID, []string{constants.RoleRiskOfficer})
	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 100}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	// Risk officer should see NOTHING in FX module
	if result.Total != 0 {
		t.Fatalf("risk officer should see 0 FX deals, got %d", result.Total)
	}
	if len(result.Data) != 0 {
		t.Fatalf("risk officer should see 0 FX deals, got %d items", len(result.Data))
	}
}

func TestListDeals_AdminSeesAll(t *testing.T) {
	// Create deals in various statuses
	createDealWithStatus(t, constants.StatusOpen)
	createDealWithStatus(t, constants.StatusPendingBooking)
	createDealWithStatus(t, constants.StatusCompleted)

	adminUserID := createTestUser(t, testPool, constants.RoleAdmin)
	ctx := makeAuthContext(t, adminUserID, []string{constants.RoleAdmin})
	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 100}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	// Admin should see ALL deals
	if result.Total < 3 {
		t.Fatalf("admin should see all deals, got total=%d", result.Total)
	}

	// Verify various statuses
	statusMap := make(map[string]bool)
	for _, d := range result.Data {
		statusMap[d.Status] = true
	}
	if !statusMap[constants.StatusOpen] {
		t.Fatal("admin should see OPEN deals")
	}
	if !statusMap[constants.StatusCompleted] {
		t.Fatal("admin should see COMPLETED deals")
	}
}

// ============================================================================
// Keyset Pagination Tests
// ============================================================================

func TestListDeals_KeysetPagination_FirstPage(t *testing.T) {
	// Create enough deals
	for i := 0; i < 5; i++ {
		createTestDeal(t, testPool)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}

	// First page with limit (no cursor yet) — uses PageSize as fallback
	pag := dto.PaginationRequest{Page: 1, PageSize: 3, SortBy: "created_at", SortDir: "desc"}
	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals cursor first page failed: %v", err)
	}

	if len(result.Data) != 3 {
		t.Fatalf("expected 3 items on first cursor page, got %d", len(result.Data))
	}

	// Verify we got a valid response with pagination info
	if result.Total < 5 {
		t.Fatalf("expected total >= 5, got %d", result.Total)
	}
}

func TestListDeals_KeysetPagination_NextPage(t *testing.T) {
	// Create deals with slight delay to ensure different timestamps
	var dealIDs []uuid.UUID
	for i := 0; i < 6; i++ {
		d := createTestDeal(t, testPool)
		dealIDs = append(dealIDs, d.ID)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}

	// Get first page via offset to establish cursors
	pag1 := dto.PaginationRequest{Page: 1, PageSize: 3, SortBy: "created_at", SortDir: "desc"}
	r1, err := testService.ListDeals(ctx, filter, pag1)
	if err != nil {
		t.Fatalf("first page failed: %v", err)
	}
	if len(r1.Data) < 3 {
		t.Fatalf("expected at least 3 items, got %d", len(r1.Data))
	}

	// Get cursor from last item of first page
	lastItem := r1.Data[len(r1.Data)-1]
	cursor := dto.EncodeCursor(lastItem.ID, lastItem.CreatedAt)

	// Get second page via cursor
	pag2 := dto.PaginationRequest{Cursor: cursor, Limit: 3}
	r2, err := testService.ListDeals(ctx, filter, pag2)
	if err != nil {
		t.Fatalf("cursor next page failed: %v", err)
	}

	if len(r2.Data) < 1 {
		t.Fatalf("expected at least 1 item on cursor next page, got %d", len(r2.Data))
	}

	// Verify no overlap between pages
	page1IDs := make(map[uuid.UUID]bool)
	for _, d := range r1.Data {
		page1IDs[d.ID] = true
	}
	for _, d := range r2.Data {
		if page1IDs[d.ID] {
			t.Fatalf("cursor page 2 returned item from page 1: %s", d.ID)
		}
	}
}

func TestListDeals_KeysetPagination_HasMore(t *testing.T) {
	// Create exactly 5 deals
	for i := 0; i < 5; i++ {
		createTestDeal(t, testPool)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}

	// Request with limit 2 — should have HasMore=true since there are more than 2
	pag := dto.PaginationRequest{Page: 1, PageSize: 2, SortBy: "created_at", SortDir: "desc"}
	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	if result.Total <= 2 {
		t.Skipf("not enough deals for HasMore test (total=%d)", result.Total)
	}
	if !result.HasMore {
		t.Fatalf("expected HasMore=true when total=%d and page_size=2", result.Total)
	}
}

func TestListDeals_OffsetPagination_BackwardCompat(t *testing.T) {
	// Create enough deals
	for i := 0; i < 5; i++ {
		createTestDeal(t, testPool)
	}

	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})
	filter := repository.FxDealFilter{}

	// Classic offset pagination should still work
	pag1 := dto.PaginationRequest{Page: 1, PageSize: 2, SortBy: "created_at", SortDir: "desc"}
	r1, err := testService.ListDeals(ctx, filter, pag1)
	if err != nil {
		t.Fatalf("page 1 failed: %v", err)
	}
	if len(r1.Data) != 2 {
		t.Fatalf("expected 2 items on page 1, got %d", len(r1.Data))
	}
	if r1.Page != 1 {
		t.Fatalf("expected page=1, got %d", r1.Page)
	}
	if r1.PageSize != 2 {
		t.Fatalf("expected page_size=2, got %d", r1.PageSize)
	}
	if r1.TotalPages < 1 {
		t.Fatalf("expected total_pages >= 1, got %d", r1.TotalPages)
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

// ============================================================================
// View Performance Tests
// ============================================================================

func TestGetDeal_UsesDetailView(t *testing.T) {
	// Create a deal and retrieve it — verify counterparty_name is populated
	deal := createTestDeal(t, testPool)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	resp, err := testService.GetDeal(ctx, deal.ID)
	if err != nil {
		t.Fatalf("GetDeal failed: %v", err)
	}

	// The detail view should populate counterparty info
	if resp.CounterpartyName == "" {
		t.Fatal("expected counterparty_name to be populated from detail view")
	}
	if resp.CounterpartyCode == "" {
		t.Fatal("expected counterparty_code to be populated from detail view")
	}
}

func TestListDeals_UsesListView(t *testing.T) {
	// Create a deal and list — verify counterparty info in list response
	createTestDeal(t, testPool)
	ctx := makeAuthContext(t, dealerUserID, []string{constants.RoleDealer})

	filter := repository.FxDealFilter{}
	pag := dto.PaginationRequest{Page: 1, PageSize: 10, SortBy: "created_at", SortDir: "desc"}

	result, err := testService.ListDeals(ctx, filter, pag)
	if err != nil {
		t.Fatalf("ListDeals failed: %v", err)
	}

	if len(result.Data) < 1 {
		t.Fatal("expected at least 1 deal")
	}

	// The list view should populate counterparty info
	for _, d := range result.Data {
		if d.CounterpartyName == "" {
			t.Fatalf("expected counterparty_name in list response for deal %s", d.ID)
		}
		if d.CounterpartyCode == "" {
			t.Fatalf("expected counterparty_code in list response for deal %s", d.ID)
		}
	}
}
