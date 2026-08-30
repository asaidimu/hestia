package http

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

// authRateLimitBurst/authRateLimitPeriod parameterize the per-Interface
// authentication limiter (Interface.authRateLimiter, built in New): a token
// bucket of 10 attempts per minute per source IP, refilled continuously.
const (
	authRateLimitKey    = "auth:attempts:"
	authRateLimitBurst  = 10
	authRateLimitRefill = 10
	authRateLimitPeriod = time.Minute
)

// @note #sec-20260821-003 issue resolved priority=P1 tags=#security,#auth : No rate limiting on authentication attempts
//
// Resolved (S-3, security batch 1): authMiddleware throttles unauthenticated
// authentication attempts per source IP through authRateLimiter (a shared
// ratestore) via CheckAndConsume with authRateLimitBurst/authRateLimitPeriod
// before any credential validation runs; once the bucket is empty the
// request is rejected with 429. The gate covers the credential-carrying
// messages listed in authRateLimitedMessages (login, elevate, confirm, ...).
// The per-operation RateLimitPolicy on the dispatch path remains the second
// layer (see the ratelimit dispatcher link).
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
				currentVersion := user.GetTokenVersion()
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
			// S-4: carry the live session ID through the identity so handlers
			// (logout) can address the current session.
			if ident != nil {
				if props, ok := ident.Properties.(map[string]any); ok {
					props["token_id"] = info.SessionID
					props["expires_at"] = info.ExpiresAt
				}
			}

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

	// 3. Rate-limit password-based login attempts by source IP to prevent
	//    brute-force attacks. The actual credential check happens downstream
	//    in AuthService.CreateSession, but we gate entry here so repeated
	//    failures are blocked before reaching the database.
	//
	//    authLimitedOps is keyed by BOTH the route pattern ("POST /api/...")
	//    — which is what the HTTP transport puts in req.Operation — and the
	//    message name (system:auth:session:create). Matching on the message
	//    name alone never fired on the HTTP path: the two string formats are
	//    never equal. Gate covers session:create, token:elevate and
	//    password:confirm (see register.go authRateLimitedMessages).
	if _, limited := o.authLimitedOps[req.Operation]; limited {
		key := authRateLimitKey + req.ClientIP
		_, allowed, err := o.authRateLimiter.CheckAndConsume(ctx, key, authRateLimitBurst, authRateLimitRefill, authRateLimitPeriod)
		if err == nil && !allowed {
			_, _ = fmt.Fprintf(os.Stderr, "AUTH RATE LIMIT: IP=%s op=%s\n", req.ClientIP, req.Operation)
			return Response{Status: 429}, runtime.ErrRateLimited
		}
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

	perms := user.GetPermissions()

	return &iam.Identity{
		Permissions: perms,
		Properties: map[string]any{
			"user_id":     userID,
			"email":       user.GetEmail(),
			"tenant_id":   user.GetTenantID(),
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
			UserID:    getProp[string](props, "user_id"),
			Email:     getProp[string](props, "email"),
			TenantID:  getProp[string](props, "tenant_id"),
			Scopes:    ident.Permissions,
			TokenType: getProp[string](props, "token_type"),
			TokenID:   getProp[string](props, "token_id"),
			ExpiresAt: getProp[int64](props, "expires_at"),
		}
	} else {
		claims = &abstract.Claims{}
	}
	ctx = runtimecontext.ContextWithClaims(ctx, claims)
	ctx = addAuditContext(ctx, claims)
	return next(ctx, req)
}

func getProp[T any](props map[string]any, key string) T {
	if props == nil {
		var zero T
		return zero
	}
	v, _ := props[key].(T)
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
