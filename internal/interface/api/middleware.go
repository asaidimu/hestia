package api

import (
	"context"
	"strings"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/identity"
)

func (o *Orchestrator) authMiddleware(ctx context.Context, req Request, next handlerFunc) (Response, error) {
	// 1. Try Bearer token
	token := extractBearer(req)
	if token != "" {
		claims, err := o.validateBearer(ctx, req, token)
		if err == nil {
			ctx = identity.ContextWithClaims(ctx, claims)
			ctx = addAccessLogIdentity(ctx, claims)
			return next(ctx, req)
		}
		// Invalid token — fall through to anonymous.
		// SecureDispatcher will return 403 if access is denied.
	}

	// 2. Try access token cookie (for browser-based requests like <img>, <link>, etc.)
	if token == "" && o.cookieCfg.AccessName != "" {
		if at, ok := req.Cookies[o.cookieCfg.AccessName]; ok && at != "" {
			claims, err := o.validateBearer(ctx, req, at)
			if err == nil {
				ctx = identity.ContextWithClaims(ctx, claims)
				ctx = addAccessLogIdentity(ctx, claims)
				return next(ctx, req)
			}
		}
	}

	// 3. Try API key
	apiKey := req.Headers["X-Api-Key"]
	if len(apiKey) == 0 {
		apiKey = req.Headers["X-API-Key"]
	}
	if len(apiKey) > 0 && apiKey[0] != "" {
		claims, err := o.authenticateAPIKey(ctx, req, apiKey[0])
		if err == nil {
			ctx = identity.ContextWithClaims(ctx, claims)
			ctx = addAccessLogIdentity(ctx, claims)
			return next(ctx, req)
		}
		// fall through to anonymous
	}

	// Default to anonymous identity — SecureDispatcher will enforce rules.
	claims := &identity.Claims{}
	ctx = identity.ContextWithClaims(ctx, claims)
	ctx = addAccessLogIdentity(ctx, claims)
	return next(ctx, req)
}

func addAccessLogIdentity(ctx context.Context, claims *identity.Claims) context.Context {
	credential := ""
	if claims.TokenID != "" {
		credential = claims.TokenType + ":" + claims.TokenID
	}
	return core.ContextWithAccessLogIdentity(ctx, claims.UserID, credential)
}

func extractBearer(req Request) string {
	authHeader := req.Headers["Authorization"]
	if len(authHeader) == 0 {
		return ""
	}
	parts := strings.SplitN(authHeader[0], " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
