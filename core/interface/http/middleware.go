package http

import (
	"context"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/identity"
)

func (o *Interface) authMiddleware(ctx context.Context, req Request, next handlerFunc) (Response, error) {
	if claims, ok := identity.ClaimsFromContext(ctx); ok && claims.UserID != "" {
		return next(ctx, req)
	}

	// 1. Try session cookie
	if o.credProv != nil {
		if cookie, ok := req.Cookies[o.cookieCfg.SessionName]; ok && cookie != "" {
			action, _ := ctx.Value(cookieActionKey).(*cookieAction)

			info, err := o.credProv.ValidateSession(cookie)
			if err != nil {
				if action != nil {
					action.Clear = true
				}
				return Response{Status: 401}, runtime.ErrUnauthorized.WithCause(err)
			}

			now := time.Now().Unix()

			// Absolute expiry check
			if now > info.ExpiresAt {
				if action != nil {
					action.Clear = true
				}
				return Response{Status: 401}, runtime.ErrUnauthorized
			}

			elapsed := now - info.IssuedAt

			// Idle timeout — session expired
			if elapsed > int64(o.idleTTL.Seconds()) {
				if action != nil {
					action.Clear = true
				}
				return Response{Status: 401}, runtime.ErrUnauthorized
			}

			// Session revocation check — ensure token_version matches the current user's
			if o.userModel != nil {
				user, err := o.userModel.GetActiveByID(ctx, info.UserID)
				if err != nil {
					if action != nil {
						action.Clear = true
					}
					return Response{Status: 401}, runtime.ErrUnauthorized
				}
				currentVersion, _ := user.GetInt("token_version")
				if info.TokenVersion != currentVersion {
					if action != nil {
						action.Clear = true
					}
					return Response{Status: 401}, runtime.ErrUnauthorized
				}
			}

			// Sliding window — refresh cookie
			if elapsed > int64(o.refreshTTL.Seconds()) {
				if _, skip := o.noRefreshOps[req.Operation]; !skip {
					newToken, err := o.credProv.RefreshSession(info)
					if err == nil && action != nil {
						action.SetToken = newToken
					}
				}
			}

			ident := o.resolveIdentity(ctx, info.UserID)
			return o.authenticated(ctx, ident, next, req)
		}
	}

	// 2. Try API key
	apiKey := req.Headers["X-Api-Key"]
	if len(apiKey) == 0 {
		apiKey = req.Headers["X-API-Key"]
	}
	if len(apiKey) > 0 && apiKey[0] != "" {
		ident, err := o.identityProv.Authenticate("api_key", apiKey[0])
		if err != nil {
			return Response{Status: 401}, runtime.ErrInvalidCredentials.WithCause(err)
		}
		return o.authenticated(ctx, ident, next, req)
	}

	// No auth provided — use anonymous identity; policy engine handles enforcement
	return o.authenticated(ctx, nil, next, req)
}

func (o *Interface) resolveIdentity(ctx context.Context, userID string) *iam.Identity {
	if userID == "" || o.userModel == nil {
		return nil
	}

	user, err := o.userModel.GetActiveByID(ctx, userID)
	if err != nil {
		return nil
	}

	userEmail, _ := user.GetString("email")
	tenantID, _ := user.GetString("tenant_id")
	perms := []string{}
	if rawPerms, err := user.GetStringArray("permissions"); err == nil {
		perms = rawPerms
	}

	return &iam.Identity{
		Permissions: perms,
		Properties: map[string]any{
			"user_id":     userID,
			"email":       userEmail,
			"tenant_id":   tenantID,
			"permissions": perms,
			"token_type":  "session",
		},
	}
}

func (o *Interface) authenticated(ctx context.Context, ident *iam.Identity, next handlerFunc, req Request) (Response, error) {
	var claims *identity.Claims
	if ident != nil {
		props, _ := ident.Properties.(map[string]any)
		claims = &identity.Claims{
			UserID:    getStringProp(props, "user_id"),
			Email:     getStringProp(props, "email"),
			TenantID:  getStringProp(props, "tenant_id"),
			Scopes:    ident.Permissions,
			TokenType: getStringProp(props, "token_type"),
		}
	} else {
		claims = &identity.Claims{}
	}
	ctx = identity.ContextWithClaims(ctx, claims)
	ctx = addAuditContext(ctx, claims)
	return next(ctx, req)
}

func getStringProp(props map[string]any, key string) string {
	if props == nil {
		return ""
	}
	v, _ := props[key].(string)
	return v
}

func addAuditContext(ctx context.Context, claims *identity.Claims) context.Context {
	actorID := claims.UserID
	if actorID == "" {
		actorID = "unknown"
	}

	actorType := runtime.ActorTypeUser
	authMethod := runtime.AuthMethodPassword

	switch claims.TokenType {
	case "api_key":
		actorType = runtime.ActorTypeService
		authMethod = runtime.AuthMethodAPIKey
	}

	return runtime.ContextWithAuditIdentity(ctx, actorID, actorType, authMethod)
}
