package audit

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

	"github.com/kienlongbank/treasury-api/internal/repository"
	auditpkg "github.com/kienlongbank/treasury-api/pkg/audit"
	"github.com/kienlongbank/treasury-api/pkg/dto"
)

var (
	testPool   *pgxpool.Pool
	testRepo   repository.AuditLogRepository
	testLogger *zap.Logger
	testAudit  *auditpkg.Logger
)

var (
	testUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-audit")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-audit-test")).
		Port(15435).
		Database("treasury_audit_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15435/treasury_audit_test?sslmode=disable"
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

	testRepo = NewRepository(pool)
	testAudit = auditpkg.NewLogger(pool, testLogger)

	code := m.Run()

	pool.Close()
	pg.Stop()
	os.Exit(code)
}

func TestWriteAndListAuditLog(t *testing.T) {
	ctx := context.Background()

	// Write some audit entries
	testAudit.Log(ctx, auditpkg.Entry{
		UserID:     testUserID,
		FullName:   "Test Dealer",
		Action:     "CREATE_DEAL",
		DealModule: "FX",
		NewValues:  map[string]interface{}{"amount": 100000},
		IPAddress:  "127.0.0.1",
	})

	testAudit.Log(ctx, auditpkg.Entry{
		UserID:       testUserID,
		FullName:     "Test Dealer",
		Action:       "APPROVE_DEAL",
		DealModule:   "FX",
		StatusBefore: "OPEN",
		StatusAfter:  "PENDING_L2_APPROVAL",
		IPAddress:    "127.0.0.1",
	})

	// List all
	filter := dto.AuditLogFilter{}
	logs, total, err := testRepo.List(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", total)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(logs))
	}
}

func TestListAuditLogs_FilterByAction(t *testing.T) {
	ctx := context.Background()

	action := "CREATE_DEAL"
	filter := dto.AuditLogFilter{Action: &action}
	logs, _, err := testRepo.List(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("List with action filter failed: %v", err)
	}
	for _, l := range logs {
		resp := AuditLogToResponse(&l)
		if resp.Action != "CREATE_DEAL" {
			t.Fatalf("expected CREATE_DEAL action, got %s", resp.Action)
		}
	}
}

func TestListAuditLogs_FilterByModule(t *testing.T) {
	ctx := context.Background()

	module := "FX"
	filter := dto.AuditLogFilter{DealModule: &module}
	logs, _, err := testRepo.List(ctx, filter, dto.DefaultPagination())
	if err != nil {
		t.Fatalf("List with module filter failed: %v", err)
	}
	for _, l := range logs {
		if l.DealModule != "FX" {
			t.Fatalf("expected FX module, got %s", l.DealModule)
		}
	}
}

func TestAuditLogStats(t *testing.T) {
	ctx := context.Background()

	stats, err := testRepo.Stats(ctx, "2020-01-01", "2030-12-31")
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if len(stats) < 1 {
		t.Fatal("expected at least 1 stat entry")
	}
	// Verify structure
	for _, s := range stats {
		if s.Action == "" {
			t.Fatal("expected non-empty action in stats")
		}
		if s.Count < 1 {
			t.Fatalf("expected count >= 1, got %d", s.Count)
		}
	}
}
