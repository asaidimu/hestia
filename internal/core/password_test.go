package core

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	password := "super-secure-password-123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == password {
		t.Error("hash should not equal plaintext password")
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword should return false for incorrect password")
	}
}
