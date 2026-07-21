package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/feature/operations"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

const (
	adminSeedKey         = "admin_user_id"
	adminPasswordHashKey = "admin_password_hash"
)

type SeedAdminOptions struct {
	Email            string
	Password         string
	ForceBootstrapped bool
}

func SeedAdmin(ctx context.Context, userModel *users.UserModel, seedModel *operations.SeedModel, logger *zap.Logger, opts ...SeedAdminOptions) (adminID string, adminEmail string, bootstrapped bool, err error) {
	var opt SeedAdminOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	existingID, _ := seedModel.Get(ctx, adminSeedKey)
	if existingID != "" {
		adminID, bootstrapped, err = checkBootstrapped(ctx, userModel, seedModel, existingID)
		if err != nil {
			return
		}
		if opt.ForceBootstrapped {
			bootstrapped = true
		}
		var doc *data.Document
		doc, err = userModel.GetByID(ctx, existingID)
		if err != nil {
			err = nil
			adminEmail = ""
			return
		}
		adminEmail, _ = doc.GetString("email")
		return
	}

	email := opt.Email
	if email == "" {
		var e string
		e, err = randomHex(8)
		if err != nil {
			return "", "", false, fmt.Errorf("generate admin email: %w", err)
		}
		email = fmt.Sprintf("admin-%s@seed.local", e)
	}

	password := opt.Password
	if password == "" {
		password, err = randomHex(16)
		if err != nil {
			return "", "", false, fmt.Errorf("generate admin password: %w", err)
		}
	}

	adminEmail = email

	doc, err := userModel.Register(ctx, email, password, "System Administrator", "administrator")
	if err != nil {
		return "", "", false, fmt.Errorf("create admin user: %w", err)
	}

	adminUserID := doc.ID()

	if err := seedModel.Set(ctx, adminSeedKey, adminUserID); err != nil {
		return "", "", false, fmt.Errorf("save admin seed: %w", err)
	}

	if err := RecordInitialAdminHash(ctx, seedModel, userModel, adminUserID); err != nil {
		return "", "", false, fmt.Errorf("record initial admin hash: %w", err)
	}

	bootstrapped = opt.ForceBootstrapped

	return adminUserID, adminEmail, bootstrapped, nil
}

func checkBootstrapped(ctx context.Context, userModel *users.UserModel, seedModel *operations.SeedModel, adminUserID string) (string, bool, error) {
	seedHash, err := seedModel.Get(ctx, adminPasswordHashKey)
	if err != nil || seedHash == "" {
		return adminUserID, false, nil
	}

	currentHash, err := userModel.GetPasswordHash(ctx, adminUserID)
	if err != nil {
		return adminUserID, false, nil
	}

	return adminUserID, seedHash != currentHash, nil
}

func RecordInitialAdminHash(ctx context.Context, seedModel *operations.SeedModel, userModel *users.UserModel, adminUserID string) error {
	hash, err := userModel.GetPasswordHash(ctx, adminUserID)
	if err != nil {
		return err
	}
	return seedModel.Set(ctx, adminPasswordHashKey, hash)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
