package auth

import (
	"context"

	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/internal/core/identity"
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

func (a *APIKeyAuthenticator) loadUserScopes(ctx context.Context) []string {
	if a.adminUserID == "" {
		return nil
	}
	doc, err := a.userModel.GetByID(ctx, a.adminUserID)
	if err != nil {
		return nil
	}
	scopes, _ := doc.GetStringArray("scopes")
	return scopes
}

func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, key string) (*identity.Claims, error) {
	if a.ephemeralKey != "" && key == a.ephemeralKey {
		scopes := a.loadUserScopes(ctx)
		return &identity.Claims{
			UserID: a.adminUserID,
			Email:  a.adminEmail,
			Scopes: scopes,
		}, nil
	}
	return a.apiKeyModel.ValidateKey(ctx, key)
}
