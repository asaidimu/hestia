package auth

import (
	"context"
	"crypto/subtle"
	"errors"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/system/apikeys/model"
	"github.com/asaidimu/hestia/core/system/users"
)

var errInvalidAPIKey = errors.New("invalid api key")

type APIKeyAuthenticator struct {
	apiKeyModel  *apikeysmodel.SystemAPIKeys
	liveUsers    collection.LiveCollection[*users.UserClaims]
	ephemeralKey string
	adminUserID  string
	adminEmail   string
	logger       *zap.Logger
	// bootstrapped reports whether the system has completed bootstrap. The
	// ephemeral key is only valid before that point.
	bootstrapped func() bool
}

func NewAPIKeyAuthenticator(apiKeyModel *apikeysmodel.SystemAPIKeys, liveUsers collection.LiveCollection[*users.UserClaims], ephemeralKey, adminUserID, adminEmail string, logger *zap.Logger, bootstrapped func() bool) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKeyModel:  apiKeyModel,
		liveUsers:    liveUsers,
		ephemeralKey: ephemeralKey,
		adminUserID:  adminUserID,
		adminEmail:   adminEmail,
		logger:       logger,
		bootstrapped: bootstrapped,
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
	// Constant-time comparison: the ephemeral key is a bearer secret.
	if a.ephemeralKey != "" && subtle.ConstantTimeCompare([]byte(key), []byte(a.ephemeralKey)) == 1 {
		if a.bootstrapped != nil && a.bootstrapped() {
			// The bootstrap key's only purpose is to bootstrap the system.
			// Reject after bootstrap so a key captured from boot logs cannot
			// keep permanent admin access (todo/first_run_api_key.md).
			a.logger.Warn("ephemeral API key rejected: system is already bootstrapped")
			return nil, errInvalidAPIKey
		}
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
