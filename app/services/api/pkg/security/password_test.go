package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	password := "SecureP@ssw0rd!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if hash == password {
		t.Fatal("hash should differ from plaintext")
	}

	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword should return true for correct password")
	}
	if VerifyPassword("WrongPassword!", hash) {
		t.Error("VerifyPassword should return false for wrong password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "SamePassword123"
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)
	if hash1 == hash2 {
		t.Error("hashing the same password twice should produce different hashes (salt)")
	}
}
