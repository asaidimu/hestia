package http

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

type hestiaIdentityProvider struct {
	credProv     abstract.CredentialsProvider
	internalDisp abstract.Dispatcher
}

func newIdentityProvider(credProv abstract.CredentialsProvider, internalDisp abstract.Dispatcher) iam.IdentityProvider {
	return &hestiaIdentityProvider{credProv: credProv, internalDisp: internalDisp}
}

func (p *hestiaIdentityProvider) Authenticate(args ...any) (*iam.Identity, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no credentials provided")
	}
	method, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid auth method type")
	}
	switch method {
	case "api_key":
		if len(args) < 2 {
			return nil, fmt.Errorf("missing API key")
		}
		key, ok := args[1].(string)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid API key")
		}
		return p.authenticateAPIKey(key)
	default:
		return nil, fmt.Errorf("unknown auth method: %s", method)
	}
}

func (p *hestiaIdentityProvider) Deauthenticate(props any) (bool, error) {
	return false, nil
}

func (p *hestiaIdentityProvider) authenticateAPIKey(key string) (*iam.Identity, error) {
	ctx := context.Background()

	apiKeyMsg := dispatch.NewMessage("system:auth:apikey:validate", ctx,
		data.MustNewDocument(map[string]any{"api_key": key}, ctx))
	result, err := p.internalDisp.Send(apiKeyMsg)
	if err != nil {
		return nil, common.NewSystemError("UNAUTHORIZED", err.Error())
	}
	defer result.Release()

	claims := extractClaims(result.Document)
	return claimsToIdentity(claims), nil
}

func claimsToIdentity(claims *abstract.Claims) *iam.Identity {
	perms := claims.Scopes
	if perms == nil {
		perms = []string{}
	}
	props := map[string]any{
		"user_id":     claims.UserID,
		"email":       claims.Email,
		"permissions": perms,
		"tenant_id":   claims.TenantID,
		"token_type":  claims.TokenType,
		"token_id":    claims.TokenID,
		"expires_at":  claims.ExpiresAt,
	}
	if claims.Operations != nil {
		props["operations"] = claims.Operations
	}
	return &iam.Identity{
		Permissions: perms,
		Properties:  props,
	}
}

func extractClaims(doc data.Documenter) *abstract.Claims {
	if doc == nil {
		return &abstract.Claims{}
	}
	userID, _ := doc.GetOr("user_id", "").(string)
	email, _ := doc.GetOr("email", "").(string)
	tenantID, _ := doc.GetOr("tenant_id", "").(string)
	perms, _ := doc.GetOr("permissions", []string{}).([]string)
	tokenType, _ := doc.GetOr("token_type", "").(string)
	tokenID, _ := doc.GetOr("token_id", "").(string)
	expiresAt, _ := doc.GetOr("expires_at", int64(0)).(int64)
	return &abstract.Claims{
		UserID:    userID,
		Email:     email,
		TenantID:  tenantID,
		Scopes:    perms,
		TokenType: tokenType,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}
}
