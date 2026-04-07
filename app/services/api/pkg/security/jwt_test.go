package security

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func testJWTManager() *JWTManager {
	return NewJWTManager(JWTConfig{
		Secret:     "test-secret-key-for-unit-tests",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	mgr := testJWTManager()
	userID := uuid.New()
	roles := []string{"DEALER", "DESK_HEAD"}
	branchID := "HCM001"

	token, err := mgr.GenerateAccessToken(userID, roles, branchID)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if len(claims.Roles) != 2 || claims.Roles[0] != "DEALER" {
		t.Errorf("expected roles [DEALER, DESK_HEAD], got %v", claims.Roles)
	}
	if claims.BranchID != "HCM001" {
		t.Errorf("expected branchID HCM001, got %s", claims.BranchID)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	mgr := testJWTManager()
	userID := uuid.New()

	token, err := mgr.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{
		Secret:     "test-secret",
		AccessTTL:  -1 * time.Hour, // already expired
		RefreshTTL: 7 * 24 * time.Hour,
	})
	userID := uuid.New()

	token, err := mgr.GenerateAccessToken(userID, []string{"DEALER"}, "HCM001")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = mgr.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	mgr1 := NewJWTManager(JWTConfig{
		Secret:     "secret-1",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	mgr2 := NewJWTManager(JWTConfig{
		Secret:     "secret-2",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	userID := uuid.New()

	token, err := mgr1.GenerateAccessToken(userID, []string{"DEALER"}, "HCM001")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	_, err = mgr2.ValidateToken(token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestExtractClaims_ExpiredToken(t *testing.T) {
	mgr := NewJWTManager(JWTConfig{
		Secret:     "test-secret",
		AccessTTL:  -1 * time.Hour,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	userID := uuid.New()

	token, err := mgr.GenerateAccessToken(userID, []string{"DEALER"}, "HCM001")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// ExtractClaims should work even for expired tokens
	claims, err := mgr.ExtractClaims(token)
	if err != nil {
		t.Fatalf("ExtractClaims should parse expired token: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	mgr := testJWTManager()
	_, err := mgr.ValidateToken("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid token format")
	}
}
