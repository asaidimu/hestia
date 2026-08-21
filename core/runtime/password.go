// @note #review-20260821-003 issue status=open priority=P2 tags=#review,#consistency : Duplicate BcryptCost constant
// BcryptCost is defined here as a constant (value 12), but DefaultBcryptCost
// is also defined in config.go with the same value. This duplication can lead
// to inconsistencies if one is updated without the other.
//
// Consider removing the constant here and using DefaultBcryptCost from config.go,
// or consolidating both into a single constants package.
package runtime

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const BcryptCost = 12

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// @note #review-20260821-004 issue status=open priority=P2 tags=#review,#security : CheckPassword swallows bcrypt errors
// CheckPassword returns a boolean but silently swallows all errors from
// bcrypt.CompareHashAndPassword. While this is a common pattern for
// password verification, it means malformed hashes or other internal errors
// are indistinguishable from wrong passwords.
//
// Consider returning (bool, error) for better observability, or at minimum
// logging the error for debugging purposes.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
