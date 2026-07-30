package http

import (
	"context"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func (o *Interface) authMiddleware(ctx context.Context, req Request, next handlerFunc) (Response, error) {
	if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok && claims.UserID != "" {
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

			// Check for API key elevation — if present, overlay elevated
			// identity on the session, setting on_behalf_of_id for audit.
			// A failed elevation is NOT silently ignored — it flows through
			// as anonymous so the authorization dispatcher logs the denial.
			if apiKey := o.extractAPIKey(req); apiKey != "" {
				elevated, err := o.identityProv.Authenticate("api_key", apiKey)
				if err != nil || elevated == nil {
					ctx = context.WithValue(ctx, runtime.AuditOnBehalfOfIDKey, info.UserID)
					return o.authenticated(ctx, nil, next, req)
				}
				ctx = context.WithValue(ctx, runtime.AuditOnBehalfOfIDKey, info.UserID)
				props, _ := elevated.Properties.(map[string]any)
				props["token_type"] = "elevated"
				return o.authenticated(ctx, elevated, next, req)
			}

			return o.authenticated(ctx, ident, next, req)
		}
	}

	// 2. Try API key (standalone — no session)
	apiKey := o.extractAPIKey(req)
	if apiKey != "" {
		ident, err := o.identityProv.Authenticate("api_key", apiKey)
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
	var claims *abstract.Claims
	if ident != nil {
		props, _ := ident.Properties.(map[string]any)
		claims = &abstract.Claims{
			UserID:    getStringProp(props, "user_id"),
			Email:     getStringProp(props, "email"),
			TenantID:  getStringProp(props, "tenant_id"),
			Scopes:    ident.Permissions,
			TokenType: getStringProp(props, "token_type"),
		}
	} else {
		claims = &abstract.Claims{}
	}
	ctx = runtimecontext.ContextWithClaims(ctx, claims)
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

func (o *Interface) extractAPIKey(req Request) string {
	for _, header := range []string{"X-Api-Key", "X-API-Key"} {
		if vals := req.Headers[header]; len(vals) > 0 && vals[0] != "" {
			return vals[0]
		}
	}
	return ""
}

func addAuditContext(ctx context.Context, claims *abstract.Claims) context.Context {
	actorID := claims.UserID
	if actorID == "" {
		actorID = "unknown"
	}

	actorType := audit.ActorTypeUser
	authMethod := audit.AuthMethodPassword

	switch claims.TokenType {
	case "api_key":
		actorType = audit.ActorTypeService
		authMethod = audit.AuthMethodAPIKey
	case "elevated":
		authMethod = audit.AuthMethodAPIKey
	}

	return runtime.ContextWithAuditIdentity(ctx, actorID, actorType, authMethod)
}
