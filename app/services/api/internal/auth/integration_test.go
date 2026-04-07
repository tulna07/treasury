package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
	"github.com/kienlongbank/treasury-api/internal/ctxutil"
	"github.com/kienlongbank/treasury-api/internal/middleware"
	"github.com/kienlongbank/treasury-api/pkg/dto"
	"github.com/kienlongbank/treasury-api/pkg/security"
)

var (
	testPool        *pgxpool.Pool
	testService     *Service
	testHandler     *Handler
	testJWTMgr      *security.JWTManager
	testSecurityCfg *config.SecurityConfig
	testLogger      *zap.Logger
)

// Seed user UUIDs (matching 001_seed.sql)
var (
	dealerUserID   = uuid.MustParse("d0000000-0000-0000-0000-000000000001")
	deskHeadUserID = uuid.MustParse("d0000000-0000-0000-0000-000000000002")
	branchID       = uuid.MustParse("a0000000-0000-0000-0000-000000000001")
)

func TestMain(m *testing.M) {
	testLogger, _ = zap.NewDevelopment()
	defer testLogger.Sync()

	// Start embedded postgres on a different port than FX tests
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		CachePath(filepath.Join(os.TempDir(), "epg-cache-auth")).
		RuntimePath(filepath.Join(os.TempDir(), "treasury-auth-test")).
		Port(15433).
		Database("treasury_auth_test"))

	if err := pg.Start(); err != nil {
		fmt.Printf("Failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	// Connect
	ctx := context.Background()
	connStr := "postgres://postgres:postgres@localhost:15433/treasury_auth_test?sslmode=disable"
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

	// Create dependencies
	jwtCfg := security.JWTConfig{
		Secret:     "test-secret-key-for-auth-tests",
		AccessTTL:  3600_000_000_000,  // 1h in ns
		RefreshTTL: 720_000_000_000_000, // 30d in ns
	}
	testJWTMgr = security.NewJWTManager(jwtCfg)
	rbacChecker := security.NewRBACChecker()

	secCfg := config.SecurityConfig{
		AuthMode:           "standalone",
		AccessTokenTTL:     3600_000_000_000,
		RefreshTokenTTL:    720_000_000_000_000,
		CookieSecure:       false,
		CookieSameSite:     "Lax",
		CookieDomain:       "",
		CookiePath:         "/",
		SecurityLevel:      "development",
		LoginRateLimit:     100,
		LoginRateWindow:    60_000_000_000,
		MaxSessionsPerUser: 0, // unlimited for tests
		MinPasswordLength:  6,
	}
	testSecurityCfg = &secCfg

	repo := NewRepository(pool)
	testService = NewService(repo, testJWTMgr, rbacChecker, &secCfg, testLogger)
	testHandler = NewHandler(testService, &secCfg, testLogger)

	code := m.Run()

	pool.Close()
	pg.Stop()
	os.Exit(code)
}

// ─── Login Tests ───

func TestLogin_Success(t *testing.T) {
	result, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "dealer01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.User.Username != "dealer01" {
		t.Errorf("expected username dealer01, got %s", result.User.Username)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if result.CSRFToken == "" {
		t.Error("expected non-empty CSRF token")
	}
	if len(result.User.Roles) == 0 {
		t.Error("expected user to have roles")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	_, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "dealer01",
		Password: "wrongpassword",
	}, "127.0.0.1", "test-agent")

	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	_, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "nonexistent",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")

	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	// Deactivate a user temporarily
	ctx := context.Background()
	_, err := testPool.Exec(ctx, "UPDATE users SET is_active = false WHERE username = 'settlement01'")
	if err != nil {
		t.Fatalf("failed to deactivate user: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, "UPDATE users SET is_active = true WHERE username = 'settlement01'")
	}()

	_, err = testService.Login(ctx, dto.LoginRequest{
		Username: "settlement01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")

	if err == nil {
		t.Fatal("expected error for inactive user")
	}
}

func TestLogin_SetsCookies(t *testing.T) {
	body := `{"username":"dealer01","password":"Treasury@2026"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	testHandler.Login(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	var accessCookie, refreshCookie, csrfCookie *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case "treasury_access_token":
			accessCookie = c
		case "treasury_refresh_token":
			refreshCookie = c
		case "treasury_csrf_token":
			csrfCookie = c
		}
	}

	if accessCookie == nil {
		t.Fatal("missing treasury_access_token cookie")
	}
	if !accessCookie.HttpOnly {
		t.Error("access token cookie should be HttpOnly")
	}
	if accessCookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("access token cookie SameSite should be Lax, got %v", accessCookie.SameSite)
	}

	if refreshCookie == nil {
		t.Fatal("missing treasury_refresh_token cookie")
	}
	if !refreshCookie.HttpOnly {
		t.Error("refresh token cookie should be HttpOnly")
	}
	if refreshCookie.Path != "/api/v1/auth/refresh" {
		t.Errorf("refresh token cookie path should be /api/v1/auth/refresh, got %s", refreshCookie.Path)
	}

	if csrfCookie == nil {
		t.Fatal("missing treasury_csrf_token cookie")
	}
	if csrfCookie.HttpOnly {
		t.Error("CSRF token cookie should NOT be HttpOnly (JS needs to read it)")
	}

	// Verify response body does NOT contain tokens
	var apiResp dto.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Marshal data back to check it doesn't contain token fields
	dataBytes, _ := json.Marshal(apiResp.Data)
	dataStr := string(dataBytes)
	if strings.Contains(dataStr, "access_token") || strings.Contains(dataStr, "refresh_token") {
		t.Error("response body should NOT contain tokens — tokens must only be in cookies")
	}
}

// ─── Refresh Token Tests ───

func TestRefreshToken_Success(t *testing.T) {
	// Login first
	result, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "dealer01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Refresh
	refreshResult, err := testService.RefreshToken(context.Background(), result.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshResult.AccessToken == "" {
		t.Error("expected new access token")
	}
	if refreshResult.RefreshToken != result.RefreshToken {
		t.Error("refresh token should remain the same")
	}
}

func TestRefreshToken_ExpiredSession(t *testing.T) {
	// Login first
	result, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "dealer01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Expire the session manually
	tokenHash := hashToken(result.RefreshToken)
	_, err = testPool.Exec(context.Background(),
		"UPDATE user_sessions SET expires_at = NOW() - INTERVAL '1 hour' WHERE token_hash = $1", tokenHash)
	if err != nil {
		t.Fatalf("failed to expire session: %v", err)
	}

	// Refresh should fail
	_, err = testService.RefreshToken(context.Background(), result.RefreshToken)
	if err == nil {
		t.Fatal("expected error for expired session")
	}
}

func TestRefreshToken_RevokedSession(t *testing.T) {
	// Login first
	result, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "dealer01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Revoke session
	if err := testService.Logout(context.Background(), result.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Refresh should fail
	_, err = testService.RefreshToken(context.Background(), result.RefreshToken)
	if err == nil {
		t.Fatal("expected error for revoked session")
	}
}

// ─── Logout Tests ───

func TestLogout_RevokesSession(t *testing.T) {
	// Login
	result, err := testService.Login(context.Background(), dto.LoginRequest{
		Username: "deskhead01",
		Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Logout
	if err := testService.Logout(context.Background(), result.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Refresh should fail
	_, err = testService.RefreshToken(context.Background(), result.RefreshToken)
	if err == nil {
		t.Fatal("expected error after logout")
	}
}

func TestLogoutAll_RevokesAllSessions(t *testing.T) {
	ctx := context.Background()

	// Login multiple times
	result1, _ := testService.Login(ctx, dto.LoginRequest{
		Username: "deskhead01", Password: "Treasury@2026",
	}, "127.0.0.1", "agent-1")
	result2, _ := testService.Login(ctx, dto.LoginRequest{
		Username: "deskhead01", Password: "Treasury@2026",
	}, "127.0.0.2", "agent-2")

	// Logout all as deskhead01
	authCtx := ctxutil.WithUserID(ctx, deskHeadUserID)
	if err := testService.LogoutAll(authCtx); err != nil {
		t.Fatalf("logout all failed: %v", err)
	}

	// Both refresh tokens should fail
	if _, err := testService.RefreshToken(ctx, result1.RefreshToken); err == nil {
		t.Error("expected error for session 1 after logout-all")
	}
	if _, err := testService.RefreshToken(ctx, result2.RefreshToken); err == nil {
		t.Error("expected error for session 2 after logout-all")
	}
}

// ─── Current User Tests ───

func TestGetCurrentUser(t *testing.T) {
	ctx := context.Background()
	ctx = ctxutil.WithUserID(ctx, dealerUserID)

	profile, err := testService.GetCurrentUser(ctx)
	if err != nil {
		t.Fatalf("get current user failed: %v", err)
	}
	if profile.Username != "dealer01" {
		t.Errorf("expected dealer01, got %s", profile.Username)
	}
	if len(profile.Roles) == 0 {
		t.Error("expected user to have roles")
	}
}

// ─── Change Password Tests ───

func TestChangePassword_Success(t *testing.T) {
	ctx := context.Background()

	// Login first to create a session
	_, err := testService.Login(ctx, dto.LoginRequest{
		Username: "director01", Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Change password
	directorID := uuid.MustParse("d0000000-0000-0000-0000-000000000003")
	authCtx := ctxutil.WithUserID(ctx, directorID)
	err = testService.ChangePassword(authCtx, dto.ChangePasswordRequest{
		OldPassword: "Treasury@2026",
		NewPassword: "NewPassword@2026",
	})
	if err != nil {
		t.Fatalf("change password failed: %v", err)
	}

	// Old password should no longer work
	_, err = testService.Login(ctx, dto.LoginRequest{
		Username: "director01", Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Error("old password should not work after change")
	}

	// New password should work
	_, err = testService.Login(ctx, dto.LoginRequest{
		Username: "director01", Password: "NewPassword@2026",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("new password should work: %v", err)
	}

	// Restore original password for other tests
	hash, _ := security.HashPassword("Treasury@2026")
	testPool.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE id = $2", hash, directorID)
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	ctx := context.Background()
	ctx = ctxutil.WithUserID(ctx, dealerUserID)

	err := testService.ChangePassword(ctx, dto.ChangePasswordRequest{
		OldPassword: "wrongpassword",
		NewPassword: "NewPassword@2026",
	})
	if err == nil {
		t.Fatal("expected error for wrong old password")
	}
}

// ─── Max Sessions Tests ───

func TestMaxSessions_EvictsOldest(t *testing.T) {
	ctx := context.Background()

	// Create a service with max 2 sessions
	limitedCfg := *testSecurityCfg
	limitedCfg.MaxSessionsPerUser = 2

	repo := NewRepository(testPool)
	limitedService := NewService(repo, testJWTMgr, security.NewRBACChecker(), &limitedCfg, testLogger)

	// Clear existing sessions for accountant01
	accountantID := uuid.MustParse("d0000000-0000-0000-0000-000000000004")
	testPool.Exec(ctx, "UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1", accountantID)

	// Login 3 times — should evict oldest
	result1, err := limitedService.Login(ctx, dto.LoginRequest{
		Username: "accountant01", Password: "Treasury@2026",
	}, "10.0.0.1", "session-1")
	if err != nil {
		t.Fatalf("login 1 failed: %v", err)
	}

	_, err = limitedService.Login(ctx, dto.LoginRequest{
		Username: "accountant01", Password: "Treasury@2026",
	}, "10.0.0.2", "session-2")
	if err != nil {
		t.Fatalf("login 2 failed: %v", err)
	}

	_, err = limitedService.Login(ctx, dto.LoginRequest{
		Username: "accountant01", Password: "Treasury@2026",
	}, "10.0.0.3", "session-3")
	if err != nil {
		t.Fatalf("login 3 failed: %v", err)
	}

	// First session should be evicted
	_, err = limitedService.RefreshToken(ctx, result1.RefreshToken)
	if err == nil {
		t.Error("first session should have been evicted")
	}
}

// ─── Auth Middleware Tests ───

func TestAuthMiddleware_CookieFirst(t *testing.T) {
	// Generate a valid token
	token, err := testJWTMgr.GenerateAccessToken(dealerUserID, []string{"DEALER"}, branchID.String())
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Create a request with BOTH cookie and header (cookie should win)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "treasury_access_token", Value: token})
	req.Header.Set("Authorization", "Bearer invalid-token")

	w := httptest.NewRecorder()
	var capturedUserID string

	handler := middleware.Auth(testJWTMgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = ctxutil.GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedUserID != dealerUserID.String() {
		t.Errorf("expected user ID from cookie, got %s", capturedUserID)
	}
}

func TestAuthMiddleware_FallbackToHeader(t *testing.T) {
	token, err := testJWTMgr.GenerateAccessToken(dealerUserID, []string{"DEALER"}, branchID.String())
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Request with header only (no cookie) — for Swagger/API clients
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	var capturedUserID string

	handler := middleware.Auth(testJWTMgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = ctxutil.GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedUserID != dealerUserID.String() {
		t.Errorf("expected user ID from header, got %s", capturedUserID)
	}
}

func TestAuthMiddleware_NoCreds(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler := middleware.Auth(testJWTMgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ─── HTTP Handler Integration Tests ───

func TestLoginHandler_HTTP(t *testing.T) {
	body := `{"username":"dealer01","password":"Treasury@2026"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	testHandler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var apiResp dto.APIResponse
	json.NewDecoder(w.Result().Body).Decode(&apiResp)
	if !apiResp.Success {
		t.Error("expected success=true")
	}
}

func TestLogoutHandler_ClearsCookies(t *testing.T) {
	// Login first to get cookies
	loginBody := `{"username":"dealer01","password":"Treasury@2026"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testHandler.Login(loginW, loginReq)

	// Get refresh token cookie
	var refreshCookie *http.Cookie
	for _, c := range loginW.Result().Cookies() {
		if c.Name == "treasury_refresh_token" {
			refreshCookie = c
		}
	}

	// Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	if refreshCookie != nil {
		logoutReq.AddCookie(refreshCookie)
	}
	logoutW := httptest.NewRecorder()
	testHandler.Logout(logoutW, logoutReq)

	if logoutW.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", logoutW.Code)
	}

	// Check cookies are cleared
	for _, c := range logoutW.Result().Cookies() {
		if c.Name == "treasury_access_token" || c.Name == "treasury_refresh_token" || c.Name == "treasury_csrf_token" {
			if c.MaxAge != -1 {
				t.Errorf("cookie %s should have MaxAge=-1, got %d", c.Name, c.MaxAge)
			}
		}
	}
}

func TestMeHandler_Authenticated(t *testing.T) {
	// Generate token
	token, _ := testJWTMgr.GenerateAccessToken(dealerUserID, []string{"DEALER"}, branchID.String())

	// Build a router to apply middleware
	r := chi.NewRouter()
	r.Use(middleware.Auth(testJWTMgr))
	r.Get("/api/v1/auth/me", testHandler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "treasury_access_token", Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestRevokeSessionHandler(t *testing.T) {
	ctx := context.Background()

	// Login to create a session
	result, _ := testService.Login(ctx, dto.LoginRequest{
		Username: "dealer01", Password: "Treasury@2026",
	}, "127.0.0.1", "test-agent")

	// Find the session
	tokenHash := hashToken(result.RefreshToken)
	session, _ := testService.repo.GetSessionByTokenHash(ctx, tokenHash)
	if session == nil {
		t.Fatal("session not found")
	}

	// Build request to revoke it
	token, _ := testJWTMgr.GenerateAccessToken(dealerUserID, []string{"DEALER"}, branchID.String())
	r := chi.NewRouter()
	r.Use(middleware.Auth(testJWTMgr))
	r.Delete("/api/v1/auth/sessions/{id}", testHandler.RevokeSession)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+session.ID.String(), nil)
	req.AddCookie(&http.Cookie{Name: "treasury_access_token", Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify session is revoked
	_, err := testService.RefreshToken(ctx, result.RefreshToken)
	if err == nil {
		t.Error("session should be revoked")
	}
}
