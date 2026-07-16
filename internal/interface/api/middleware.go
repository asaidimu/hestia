package api

import (
	"context"
	"strings"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/identity"
)

type contextKey string

const clearAccessCookieKey contextKey = "clear_access_cookie"
const clearRefreshCookieKey contextKey = "clear_refresh_cookie"

func (o *Interface) authMiddleware(ctx context.Context, req Request, next handlerFunc) (Response, error) {
	// 1. Try Bearer token
	token := extractBearer(req)
	if token != "" {
		claims, err := o.validateBearer(ctx, req, token)
		if err == nil {
			ctx = identity.ContextWithClaims(ctx, claims)
			ctx = addAuditContext(ctx, claims)
			return next(ctx, req)
		}
	}

	// 2. Try access token cookie (for browser-based requests like <img>, <link>, etc.)
	if token == "" && o.cookieCfg.AccessName != "" {
		if at, ok := req.Cookies[o.cookieCfg.AccessName]; ok && at != "" {
			claims, err := o.validateBearer(ctx, req, at)
			if err == nil {
				ctx = identity.ContextWithClaims(ctx, claims)
				ctx = addAuditContext(ctx, claims)
				return next(ctx, req)
			}
			ctx = context.WithValue(ctx, clearAccessCookieKey, true)
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
			ctx = addAuditContext(ctx, claims)
			return next(ctx, req)
		}
	}

	// Default to anonymous identity
	claims := &identity.Claims{}
	ctx = identity.ContextWithClaims(ctx, claims)
	ctx = addAuditContext(ctx, claims)
	return next(ctx, req)
}

func addAuditContext(ctx context.Context, claims *identity.Claims) context.Context {
	actorID := claims.UserID
	if actorID == "" {
		actorID = "anonymous"
	}

	actorType := core.ActorTypeUser
	authMethod := core.AuthMethodPassword

	switch claims.TokenType {
	case "session":
		authMethod = core.AuthMethodOAuth
	case "api_key":
		actorType = core.ActorTypeService
		authMethod = core.AuthMethodAPIKey
	case "bearer":
		authMethod = core.AuthMethodOAuth
	case "":
		ident, ok := iam.GetIdentity(ctx)
		if ok {
			props, _ := ident.Properties.(map[string]any)
			if v, _ := props["system"].(string); v == "http" {
				actorType = core.ActorTypeSystem
				authMethod = core.AuthMethodServiceAccount
			} else if actorID == "anonymous" {
				actorType = core.ActorTypeAnonymous
				authMethod = core.AuthMethodNone
			}
		} else {
			actorType = core.ActorTypeAnonymous
			authMethod = core.AuthMethodNone
		}
	}

	return core.ContextWithAuditIdentity(ctx, actorID, actorType, authMethod)
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
