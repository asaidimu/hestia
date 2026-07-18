package auth

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/registration"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core/identity"
)

func NewCreateSessionHandler(users *users.UserModel, credProv abstract.CredentialsProvider) core.MessageHandler {
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

		if !core.CheckPassword(password, storedPassword) {
			return nil, fmt.Errorf("invalid email or password")
		}

		userID := user.ID()
		userEmail, _ := user.GetString("email")
		perms := []string{}
		if rawPerms, err := user.GetStringArray("permissions"); err == nil {
			perms = rawPerms
		}

		accessToken, err := credProv.IssueAccess(userID, userEmail, perms)
		if err != nil {
			return nil, err
		}

		refreshToken, err := credProv.IssueRefresh(userID, userEmail)
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

func NewRefreshSessionHandler(credProv abstract.CredentialsProvider) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		token, _ := body["refresh_token"].(string)

		accessToken, newRefresh, err := credProv.Refresh(ctx, token)
		if err != nil {
			return nil, err
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

func NewDeleteSessionHandler(credProv abstract.CredentialsProvider) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		claims, ok := identity.ClaimsFromContext(ctx)
		if ok && claims != nil && claims.TokenID != "" {
			if err := credProv.Revoke(ctx, claims); err != nil {
				return nil, err
			}
		}

		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		if body != nil {
			if refreshToken, _ := body["refresh_token"].(string); refreshToken != "" {
				refreshClaims, err := credProv.Validate(refreshToken)
				if err == nil && refreshClaims.TokenID != "" {
					if err := credProv.Revoke(ctx, refreshClaims); err != nil {
						return nil, err
					}
				}
			}
		}

		return &registration.Result{}, nil
	}
}

func NewPasswordResetHandler(users *users.UserModel, credProv abstract.CredentialsProvider) core.MessageHandler {
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
		credProv.IssueReset(userID, userEmail)
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

func NewValidateTokenHandler(credProv abstract.CredentialsProvider) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		token, _ := doc.GetOr("token", "").(string)

		claims, err := credProv.Validate(token)
		if err != nil {
			return nil, err
		}
		claimsDoc := data.MustNewDocument(map[string]any{
			"user_id":     claims.UserID,
			"email":       claims.Email,
			"permissions": claims.Scopes,
			"token_type":  claims.TokenType,
			"token_id":    claims.TokenID,
			"expires_at":  claims.ExpiresAt,
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
			"user_id":     claims.UserID,
			"email":       claims.Email,
			"permissions": claims.Scopes,
			"token_type":  claims.TokenType,
			"token_id":    claims.TokenID,
			"expires_at":  claims.ExpiresAt,
		}, ctx)
		return &registration.Result{Document: claimsDoc}, nil
	}
}
