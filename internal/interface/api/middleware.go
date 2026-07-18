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
const setAccessCookieKey contextKey = "set_access_cookie"
const setRefreshCookieKey contextKey = "set_refresh_cookie"

func (o *Interface) authenticated(ctx context.Context, ident *iam.Identity, next handlerFunc, req Request) (Response, error) {
	claims := identityToClaims(ident)
	ctx = identity.ContextWithClaims(ctx, claims)
	ctx = addAuditContext(ctx, claims)
	return next(ctx, req)
}

func (o *Interface) authMiddleware(ctx context.Context, req Request, next handlerFunc) (Response, error) {
	// 1. Try Bearer token
	token := extractBearer(req)
	if token != "" {
		ident, err := o.identityProv.Authenticate("bearer", token)
		if err == nil {
			return o.authenticated(ctx, ident, next, req)
		}
	}

	// 2. Try access token cookie (for browser-based requests like <img>, <link>, etc.)
	if token == "" && o.cookieCfg.AccessName != "" {
		if at, ok := req.Cookies[o.cookieCfg.AccessName]; ok && at != "" {
			ident, err := o.identityProv.Authenticate("bearer", at)
			if err == nil {
				return o.authenticated(ctx, ident, next, req)
			}

			// Access token expired — try auto-refresh if we have a refresh cookie
			// and this isn't an auth endpoint (which handles its own auth).
			if !isAuthOperation(req.Operation) && o.cookieCfg.RefreshName != "" {
				if rt, ok := req.Cookies[o.cookieCfg.RefreshName]; ok && rt != "" {
					if newCtx, ok := o.autoRefresh(ctx, req, rt); ok {
						return next(newCtx, req)
					}
				}
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
		ident, err := o.identityProv.Authenticate("api_key", apiKey[0])
		if err == nil {
			return o.authenticated(ctx, ident, next, req)
		}
	}

	// Default to anonymous identity
	ctx = identity.ContextWithClaims(ctx, &identity.Claims{})
	ctx = addAuditContext(ctx, &identity.Claims{})
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

func isAuthOperation(op string) bool {
	return strings.Contains(op, "/system/auth/")
}

func (o *Interface) autoRefresh(ctx context.Context, req Request, refreshToken string) (context.Context, bool) {
	newAccess, newRefresh, err := o.credProv.Refresh(ctx, refreshToken)
	if err != nil {
		return ctx, false
	}

	ctx = context.WithValue(ctx, setAccessCookieKey, newAccess)
	ctx = context.WithValue(ctx, setRefreshCookieKey, newRefresh)

	ident, err := o.identityProv.Authenticate("bearer", newAccess)
	if err != nil {
		return ctx, false
	}

	newClaims := identityToClaims(ident)
	ctx = identity.ContextWithClaims(ctx, newClaims)
	ctx = addAuditContext(ctx, newClaims)
	return ctx, true
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
