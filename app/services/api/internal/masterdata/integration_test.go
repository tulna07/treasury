package masterdata

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

	"github.com/kienlongbank/treasury-api/internal/model"
	"github.com/kienlongbank/treasury-api/internal/repository"
	"github.com/kienlongbank/treasury-api/pkg/apperror"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

var (
	testPool    *pgxpool.Pool
	testCPRepo  repository.CounterpartyRepository
	testMDRepo  repository.MasterDataRepository
	testLogger  *zap.Logger
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-md")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-md-test")).
		Port(15434).
		Database("treasury_md_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15434/treasury_md_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}
	testPool = pool

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

	testCPRepo = NewCounterpartyRepository(pool)
	testMDRepo = NewMasterDataRepository(pool)

	code := m.Run()

	pool.Close()
	pg.Stop()
	os.Exit(code)
}

// --- Counterparty Tests ---

func TestCreateCounterparty(t *testing.T) {
	ctx := context.Background()
	cp := &model.Counterparty{
		Code:     "TEST-" + uuid.New().String()[:8],
		FullName: "Test Counterparty",
		CIF:      "CIF-" + uuid.New().String()[:8],
	}

	err := testCPRepo.Create(ctx, cp)
	if err != nil {
		t.Fatalf("CreateCounterparty failed: %v", err)
	}
	if cp.ID == uuid.Nil {
		t.Fatal("expected non-nil ID")
	}
	if !cp.IsActive {
		t.Fatal("expected counterparty to be active")
	}
}

func TestCreateCounterparty_DuplicateCode(t *testing.T) {
	ctx := context.Background()
	code := "DUP-" + uuid.New().String()[:8]
	cp := &model.Counterparty{Code: code, FullName: "Dup Test", CIF: "CIF-DUP1"}
	_ = testCPRepo.Create(ctx, cp)

	cp2 := &model.Counterparty{Code: code, FullName: "Dup Test 2", CIF: "CIF-DUP2"}
	err := testCPRepo.Create(ctx, cp2)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !apperror.Is(err, apperror.ErrConflict) {
		t.Fatalf("expected CONFLICT, got %v", err)
	}
}

func TestGetCounterpartyByID(t *testing.T) {
	ctx := context.Background()
	cp := &model.Counterparty{
		Code: "GET-" + uuid.New().String()[:8], FullName: "Get Test",
		CIF: "CIF-GET-" + uuid.New().String()[:8],
	}
	_ = testCPRepo.Create(ctx, cp)

	found, err := testCPRepo.GetByID(ctx, cp.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.Code != cp.Code {
		t.Fatalf("expected code %s, got %s", cp.Code, found.Code)
	}
}

func TestGetCounterparty_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := testCPRepo.GetByID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestListCounterparties(t *testing.T) {
	ctx := context.Background()
	pag := dto.DefaultPagination()

	result, total, err := testCPRepo.List(ctx, dto.CounterpartyFilter{}, pag)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total < 1 {
		t.Fatalf("expected at least 1 counterparty, got %d", total)
	}
	if len(result) < 1 {
		t.Fatalf("expected at least 1 result, got %d", len(result))
	}
}

func TestListCounterparties_Search(t *testing.T) {
	ctx := context.Background()

	// Create a unique counterparty
	cp := &model.Counterparty{
		Code: "SRCH-" + uuid.New().String()[:6], FullName: "Searchable Bank XYZ",
		CIF: "CIF-SRCH-" + uuid.New().String()[:6],
	}
	_ = testCPRepo.Create(ctx, cp)

	search := "Searchable"
	filter := dto.CounterpartyFilter{Search: &search}
	result, _, err := testCPRepo.List(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("List with search failed: %v", err)
	}
	if len(result) < 1 {
		t.Fatal("expected at least 1 search result")
	}
}

func TestUpdateCounterparty(t *testing.T) {
	ctx := context.Background()
	cp := &model.Counterparty{
		Code: "UPD-" + uuid.New().String()[:8], FullName: "Before Update",
		CIF: "CIF-UPD-" + uuid.New().String()[:8],
	}
	_ = testCPRepo.Create(ctx, cp)

	cp.FullName = "After Update"
	err := testCPRepo.Update(ctx, cp)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	found, _ := testCPRepo.GetByID(ctx, cp.ID)
	if found.FullName != "After Update" {
		t.Fatalf("expected updated name, got %s", found.FullName)
	}
}

func TestSoftDeleteCounterparty(t *testing.T) {
	ctx := context.Background()
	cp := &model.Counterparty{
		Code: "DEL-" + uuid.New().String()[:8], FullName: "Delete Test",
		CIF: "CIF-DEL-" + uuid.New().String()[:8],
	}
	_ = testCPRepo.Create(ctx, cp)

	// Use a real user ID from seed data
	seedUserID := uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	err := testCPRepo.SoftDelete(ctx, cp.ID, seedUserID)
	if err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}

	_, err = testCPRepo.GetByID(ctx, cp.ID)
	if err == nil {
		t.Fatal("expected not found after soft delete")
	}
	if !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

// --- Master Data Tests ---

func TestListCurrencies(t *testing.T) {
	ctx := context.Background()
	currencies, err := testMDRepo.ListCurrencies(ctx)
	if err != nil {
		t.Fatalf("ListCurrencies failed: %v", err)
	}
	if len(currencies) < 1 {
		t.Fatal("expected at least 1 currency from seed data")
	}
	// Check VND exists
	found := false
	for _, c := range currencies {
		if c.Code == "VND" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected VND in currencies")
	}
}

func TestListCurrencyPairs(t *testing.T) {
	ctx := context.Background()
	pairs, err := testMDRepo.ListCurrencyPairs(ctx)
	if err != nil {
		t.Fatalf("ListCurrencyPairs failed: %v", err)
	}
	if len(pairs) < 1 {
		t.Fatal("expected at least 1 currency pair from seed data")
	}
}

func TestListBranches(t *testing.T) {
	ctx := context.Background()
	branches, err := testMDRepo.ListBranches(ctx)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) < 1 {
		t.Fatal("expected at least 1 branch from seed data")
	}
}

func TestListExchangeRates(t *testing.T) {
	ctx := context.Background()
	filter := dto.ExchangeRateFilter{}
	rates, _, err := testMDRepo.ListExchangeRates(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("ListExchangeRates failed: %v", err)
	}
	// May be 0 if no rates seeded — that's OK, just testing no errors
	_ = rates
}

func TestGetLatestRate(t *testing.T) {
	ctx := context.Background()
	_, err := testMDRepo.GetLatestRate(ctx, "USD")
	// May get NOT_FOUND if no rates seeded — just verify no crash
	if err != nil && !apperror.Is(err, apperror.ErrNotFound) {
		t.Fatalf("GetLatestRate failed unexpectedly: %v", err)
	}
}
