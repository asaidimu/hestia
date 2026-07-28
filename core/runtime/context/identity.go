package context

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
)

type contextKey string

const claimsKey contextKey = "auth.claims"

func ContextWithClaims(ctx context.Context, claims *abstract.Claims) context.Context {
	ctx = context.WithValue(ctx, claimsKey, claims)

	perms := claims.Scopes
	if perms == nil {
		perms = []string{}
	}

	props := map[string]any{
		"user_id":     claims.UserID,
		"email":       claims.Email,
		"permissions": perms,
		"token_type":  claims.TokenType,
		"tenant_id":   claims.TenantID,
	}
	if claims.Operations != nil {
		props["operations"] = claims.Operations
	}

	return iam.WithIdentity(ctx, iam.Identity{
		Permissions: perms,
		Properties:  props,
	})
}

func ClaimsFromContext(ctx context.Context) (*abstract.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*abstract.Claims)
	return claims, ok
}

var SystemScopePrefix = "system"

var systemIdentity = iam.Identity{
	Permissions: []string{SystemScopePrefix + ":http"},
	Properties:  map[string]any{"system": "http"},
}

func SystemContext(ctx context.Context) context.Context {
	return iam.WithIdentity(ctx, systemIdentity)
}
