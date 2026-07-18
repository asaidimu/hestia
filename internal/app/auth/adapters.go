package auth

import (
	"context"

	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core/identity"
)

type APIKeyAuthenticator struct {
	apiKeyModel  *apikeys.APIKeyModel
	userModel    *users.UserModel
	ephemeralKey string
	adminUserID  string
	adminEmail   string
}

func NewAPIKeyAuthenticator(apiKeyModel *apikeys.APIKeyModel, userModel *users.UserModel, ephemeralKey, adminUserID, adminEmail string) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKeyModel:  apiKeyModel,
		userModel:    userModel,
		ephemeralKey: ephemeralKey,
		adminUserID:  adminUserID,
		adminEmail:   adminEmail,
	}
}

func (a *APIKeyAuthenticator) loadUserScopes(ctx context.Context, userID string) []string {
	if userID == "" {
		return nil
	}
	doc, err := a.userModel.GetByID(ctx, userID)
	if err != nil {
		return nil
	}
	perms, _ := doc.GetStringArray("permissions")
	return perms
}

func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, key string) (*identity.Claims, error) {
	if a.ephemeralKey != "" && key == a.ephemeralKey {
		scopes := a.loadUserScopes(ctx, a.adminUserID)
		return &identity.Claims{
			UserID: a.adminUserID,
			Email:  a.adminEmail,
			Scopes: scopes,
		}, nil
	}

	claims, err := a.apiKeyModel.ValidateKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// API keys don't store permission scopes — they inherit the owning user's
	// current permissions at authentication time. This ensures permission
	// changes are reflected immediately without key rotation.
	claims.Scopes = a.loadUserScopes(ctx, claims.UserID)
	return claims, nil
}
