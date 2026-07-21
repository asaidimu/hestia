package api

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/identity"
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

	apiKeyMsg := abstract.NewMessage("system:auth:apikey:validate", ctx,
		data.MustNewDocument(map[string]any{"api_key": key}, ctx))
	result, err := p.internalDisp.Send(apiKeyMsg)
	if err != nil {
		return nil, common.NewSystemError("UNAUTHORIZED", err.Error())
	}

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

func identityToClaims(id *iam.Identity) *abstract.Claims {
	if id == nil {
		return &abstract.Claims{}
	}
	props, _ := id.Properties.(map[string]any)
	userID, _ := props["user_id"].(string)
	email, _ := props["email"].(string)
	tokenType, _ := props["token_type"].(string)
	tokenID, _ := props["token_id"].(string)
	expiresAt, _ := props["expires_at"].(int64)
	rawOps, _ := props["operations"].([]string)
	return &abstract.Claims{
		UserID:     userID,
		Email:      email,
		Scopes:     id.Permissions,
		Operations: rawOps,
		TokenType:  tokenType,
		TokenID:    tokenID,
		ExpiresAt:  expiresAt,
	}
}

func extractClaims(doc *data.Document) *identity.Claims {
	if doc == nil {
		return &identity.Claims{}
	}
	userID, _ := doc.GetOr("user_id", "").(string)
	email, _ := doc.GetOr("email", "").(string)
	perms, _ := doc.GetOr("permissions", []string{}).([]string)
	tokenType, _ := doc.GetOr("token_type", "").(string)
	tokenID, _ := doc.GetOr("token_id", "").(string)
	expiresAt, _ := doc.GetOr("expires_at", int64(0)).(int64)
	return &identity.Claims{
		UserID:    userID,
		Email:     email,
		Scopes:    perms,
		TokenType: tokenType,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	}
}
