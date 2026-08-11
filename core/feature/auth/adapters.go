package auth

import (
	"context"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/feature/users"
)

type APIKeyAuthenticator struct {
	apiKeyModel  *apikeysmodel.SystemAPIKeys
	liveUsers    collection.LiveCollection[*users.UserClaims]
	ephemeralKey string
	adminUserID  string
	adminEmail   string
	logger       *zap.Logger
}

func NewAPIKeyAuthenticator(apiKeyModel *apikeysmodel.SystemAPIKeys, liveUsers collection.LiveCollection[*users.UserClaims], ephemeralKey, adminUserID, adminEmail string, logger *zap.Logger) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKeyModel:  apiKeyModel,
		liveUsers:    liveUsers,
		ephemeralKey: ephemeralKey,
		adminUserID:  adminUserID,
		adminEmail:   adminEmail,
		logger:       logger,
	}
}

func (a *APIKeyAuthenticator) loadUserClaims(ctx context.Context, userID string) *users.UserClaims {
	if userID == "" || a.liveUsers == nil {
		return nil
	}
	claims, ok := a.liveUsers.Get(userID)
	if !ok {
		return nil
	}
	return claims
}

func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, key string) (*abstract.Claims, error) {
	if a.ephemeralKey != "" && key == a.ephemeralKey {
		a.logger.Warn("ephemeral API key used for authentication",
			zap.String("admin_user_id", a.adminUserID),
			zap.String("admin_email", a.adminEmail),
		)
		uc := a.loadUserClaims(ctx, a.adminUserID)
		scopes := a.adminScopes(uc)
		tenantID := a.adminTenant(uc)
		return &abstract.Claims{
			UserID:   a.adminUserID,
			Email:    a.adminEmail,
			Scopes:   scopes,
			TenantID: tenantID,
		}, nil
	}

	claims, err := a.apiKeyModel.ValidateKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// API keys don't store permission scopes — they inherit the owning user's
	// current permissions at authentication time. This ensures permission
	// changes are reflected immediately without key rotation.
	uc := a.loadUserClaims(ctx, claims.UserID)
	if uc != nil {
		claims.Scopes = uc.Permissions
		claims.TenantID = uc.TenantID
	}
	return claims, nil
}

func (a *APIKeyAuthenticator) adminScopes(uc *users.UserClaims) []string {
	if uc != nil {
		return uc.Permissions
	}
	return nil
}

func (a *APIKeyAuthenticator) adminTenant(uc *users.UserClaims) string {
	if uc != nil {
		return uc.TenantID
	}
	return ""
}
