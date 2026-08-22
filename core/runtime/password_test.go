package runtime

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

	ok, err := CheckPassword(password, hash)
	if err != nil {
		t.Fatalf("CheckPassword returned error for correct password: %v", err)
	}
	if !ok {
		t.Error("CheckPassword should return true for correct password")
	}

	ok, err = CheckPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("CheckPassword returned error for wrong password: %v", err)
	}
	if ok {
		t.Error("CheckPassword should return false for incorrect password")
	}
}
