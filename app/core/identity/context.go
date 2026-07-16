package identity

import (
	"context"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
)

type Claims = core.Claims

type contextKey string

const claimsKey contextKey = "auth.claims"

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, claimsKey, claims)

	perms := claims.Scopes
	if perms == nil {
		perms = []string{}
	}

	props := map[string]any{
		"user_id":    claims.UserID,
		"email":      claims.Email,
		"scopes":     perms,
		"token_type": claims.TokenType,
	}

	return iam.WithIdentity(ctx, iam.Identity{
		Permissions: perms,
		Properties:  props,
	})
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}
