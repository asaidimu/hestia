package auth

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/internal/core/identity"
)

func NewCreateSessionHandler(users *users.UserModel, jwtSvc core.JWTService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)
		password, _ := body["password"].(string)

		user, err := users.GetByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("invalid email or password")
		}

		storedPassword, err := user.GetString("password")
		if err != nil {
			return nil, fmt.Errorf("invalid email or password")
		}

		if !	core.CheckPassword(password, storedPassword) {
			return nil, fmt.Errorf("invalid email or password")
		}

		userID := user.ID()
		userEmail, _ := user.GetString("email")
		scopes := []string{}
		if rawScopes, err := user.GetStringArray("scopes"); err == nil {
			scopes = rawScopes
		}

		accessToken, err := jwtSvc.GenerateAccessToken(userID, userEmail, scopes)
		if err != nil {
			return nil, err
		}

		refreshToken, err := jwtSvc.GenerateRefreshToken(userID, userEmail)
		if err != nil {
			return nil, err
		}

		sctx := common.ContextWithCollectionName(ctx, "_user_")
		sane, err := user.Sanitize(sctx)
		if err != nil {
			return nil, err
		}

		respDoc := data.MustNewDocument(map[string]any{
			"token": map[string]any{
				"access":   accessToken,
				"refresh":  refreshToken,
				"type":     "Bearer",
				"validity": 900,
			},
		}, ctx)
		if sane != nil {
			respDoc.Set("user", sane)
		}

		return &registration.Result{Document: respDoc}, nil
	}
}

func NewRegisterHandler(users *users.UserModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)
		password, _ := body["password"].(string)
		name, _ := body["name"].(string)

		user, err := users.Register(ctx, email, password, name)
		if err != nil {
			return nil, err
		}
		return &registration.Result{Document: user}, nil
	}
}

func validateRefreshToken(token string, jwtSvc core.JWTService, sessionSvc core.SessionService) (*identity.Claims, error) {
	claims, err := jwtSvc.ValidateToken(token)
	if err == nil && claims.TokenType == "refresh" {
		return claims, nil
	}
	return sessionSvc.Validate(token)
}

func NewRefreshSessionHandler(users *users.UserModel, sessionSvc core.SessionService, jwtSvc core.JWTService, blocklist *TokenBlocklistService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		token, _ := body["refresh_token"].(string)

		claims, err := validateRefreshToken(token, jwtSvc, sessionSvc)
		if err != nil {
			return nil, fmt.Errorf("invalid refresh token")
		}

		if blocklist != nil && claims.TokenID != "" {
			blocklisted, err := blocklist.IsBlocklisted(ctx, claims.TokenID)
			if err != nil {
				return nil, fmt.Errorf("check blocklist: %w", err)
			}
			if blocklisted {
				return nil, fmt.Errorf("refresh token has been revoked")
			}
		}

		scopes := claims.Scopes
		user, err := users.GetByID(ctx, claims.UserID)
		if err == nil {
			if rawScopes, err := user.GetStringArray("scopes"); err == nil && len(rawScopes) > 0 {
				scopes = rawScopes
			}
		}

		if blocklist != nil && claims.TokenID != "" && claims.ExpiresAt > 0 {
			if err := blocklist.Blocklist(ctx, claims.TokenID, claims.ExpiresAt, claims.UserID); err != nil {
				return nil, fmt.Errorf("rotate refresh token: %w", err)
			}
		}

		accessToken, err := jwtSvc.GenerateAccessToken(claims.UserID, claims.Email, scopes)
		if err != nil {
			return nil, fmt.Errorf("generate access token: %w", err)
		}

		newRefresh, err := jwtSvc.GenerateRefreshToken(claims.UserID, claims.Email)
		if err != nil {
			return nil, fmt.Errorf("generate refresh token: %w", err)
		}

		respDoc := data.MustNewDocument(map[string]any{
			"token": map[string]any{
				"access":   accessToken,
				"refresh":  newRefresh,
				"type":     "Bearer",
				"validity": 900,
			},
		}, ctx)
		return &registration.Result{Document: respDoc}, nil
	}
}

func NewDeleteSessionHandler(blocklist *TokenBlocklistService, jwtSvc core.JWTService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		claims, ok := identity.ClaimsFromContext(ctx)
		if ok && claims != nil && blocklist != nil && claims.TokenID != "" && claims.ExpiresAt > 0 {
			if err := blocklist.Blocklist(ctx, claims.TokenID, claims.ExpiresAt, claims.UserID); err != nil {
				return nil, err
			}
		} else {
		}

		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		if body != nil {
			if refreshToken, _ := body["refresh_token"].(string); refreshToken != "" {
				refreshClaims, err := jwtSvc.ValidateToken(refreshToken)
				if err == nil && refreshClaims != nil && refreshClaims.TokenID != "" && refreshClaims.ExpiresAt > 0 {
					if err := blocklist.Blocklist(ctx, refreshClaims.TokenID, refreshClaims.ExpiresAt, refreshClaims.UserID); err != nil {
						return nil, err
					}
				}
			} else {
			}
		} else {
		}

		return &registration.Result{}, nil
	}
}

func safeTokenID(claims *identity.Claims) string {
	if claims == nil { return "<nil>" }
	return claims.TokenID
}

func safeExpiresAt(claims *identity.Claims) int64 {
	if claims == nil { return 0 }
	return claims.ExpiresAt
}

func NewPasswordResetHandler(users *users.UserModel, jwtSvc core.JWTService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)

		user, err := users.GetByEmail(ctx, email)
		if err != nil {
			return &registration.Result{}, nil
		}
		userID := user.ID()
		userEmail, _ := user.GetString("email")
		jwtSvc.GenerateResetToken(userID, userEmail)
		return &registration.Result{}, nil
	}
}

func NewPasswordConfirmHandler(users *users.UserModel) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		password, _ := body["password"].(string)

		claims, ok := identity.ClaimsFromContext(ctx)
		if !ok || claims == nil {
			return nil, fmt.Errorf("authorization required")
		}
		if claims.TokenType != "password_reset" {
			return nil, fmt.Errorf("invalid token type for this operation")
		}
		if err := users.ChangePassword(ctx, claims.UserID, password); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewSetBootstrapPasswordHandler(users *users.UserModel, adminUserID string) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		password, _ := body["password"].(string)
		email, _ := body["email"].(string)
		callerID, _ := body["caller_id"].(string)

		if callerID != adminUserID {
			return nil, fmt.Errorf("only the seeded admin can change the bootstrap password")
		}
		if email == "" {
			return nil, fmt.Errorf("replacement admin email is required")
		}
		if err := users.ChangePassword(ctx, adminUserID, password); err != nil {
			return nil, err
		}
		if err := users.Update(ctx, adminUserID, map[string]any{"email": email}); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewValidateTokenHandler(jwtSvc core.JWTService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		token, _ := doc.GetOr("token", "").(string)

		claims, err := jwtSvc.ValidateToken(token)
		if err != nil {
			return nil, err
		}
		claimsDoc := data.MustNewDocument(map[string]any{
			"user_id":    claims.UserID,
			"email":      claims.Email,
			"scopes":     claims.Scopes,
			"token_type": claims.TokenType,
			"token_id":   claims.TokenID,
			"expires_at": claims.ExpiresAt,
		}, ctx)
		return &registration.Result{Document: claimsDoc}, nil
	}
}

func NewCheckBlocklistHandler(blocklist *TokenBlocklistService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		tokenID, _ := doc.GetOr("token_id", "").(string)

		if blocklist == nil || tokenID == "" {
			return &registration.Result{
				Document: data.MustNewDocument(map[string]any{"blocklisted": false}, ctx),
			}, nil
		}
		blocklisted, err := blocklist.IsBlocklisted(ctx, tokenID)
		if err != nil {
			return &registration.Result{
				Document: data.MustNewDocument(map[string]any{"blocklisted": false}, ctx),
			}, nil
		}
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"blocklisted": blocklisted}, ctx),
		}, nil
	}
}

func NewValidateSessionHandler(sessionSvc core.SessionService) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		token, _ := doc.GetOr("token", "").(string)

		claims, err := sessionSvc.Validate(token)
		if err != nil {
			return nil, err
		}
		claimsDoc := data.MustNewDocument(map[string]any{
			"user_id":    claims.UserID,
			"email":      claims.Email,
			"scopes":     claims.Scopes,
			"token_type": claims.TokenType,
			"token_id":   claims.TokenID,
			"expires_at": claims.ExpiresAt,
		}, ctx)
		return &registration.Result{Document: claimsDoc}, nil
	}
}

type keyAuth interface {
	Authenticate(ctx context.Context, key string) (*identity.Claims, error)
}

func NewValidateAPIKeyHandler(keyAuth keyAuth) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		key, _ := doc.GetOr("api_key", "").(string)

		claims, err := keyAuth.Authenticate(ctx, key)
		if err != nil {
			return nil, err
		}
		claimsDoc := data.MustNewDocument(map[string]any{
			"user_id":    claims.UserID,
			"email":      claims.Email,
			"scopes":     claims.Scopes,
			"token_type": claims.TokenType,
			"token_id":   claims.TokenID,
			"expires_at": claims.ExpiresAt,
		}, ctx)
		return &registration.Result{Document: claimsDoc}, nil
	}
}
