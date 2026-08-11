package auth

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/runtime"
)

func NewCreateSessionHandler(users *model.SystemUsers, credProv abstract.CredentialsProvider, sessionTTL time.Duration) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)
		password, _ := body["password"].(string)

		user, err := users.GetByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("invalid email or password")
		}

		if !runtime.CheckPassword(password, user.Password) {
			return nil, fmt.Errorf("invalid email or password")
		}

		userID := user.ID

		token, _, err := credProv.CreateSession(userID, sessionTTL)
		if err != nil {
			return nil, err
		}

		sctx := common.ContextWithCollectionName(ctx, "_user_")
		sane, err := user.MustDocument().Sanitize(sctx)
		if err != nil {
			return nil, err
		}

		respDoc := data.MustNewDocument(map[string]any{}, ctx)
		if sane != nil {
			respDoc.Set("user", sane)
		}

		return &abstract.Result{Document: respDoc, SessionToken: token}, nil
	}
}

func NewDeleteSessionHandler() abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		return &abstract.Result{}, nil
	}
}

func NewPasswordResetHandler(users *model.SystemUsers, credProv abstract.CredentialsProvider, notifier abstract.Notifier, appURL string) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)

		user, err := users.GetByEmail(ctx, email)
		if err != nil {
			return &abstract.Result{}, nil
		}
		userID := user.ID
		token, err := credProv.IssueResetToken(userID)
		if err != nil {
			return nil, err
		}
		if notifier != nil && appURL != "" {
			if err := notifier.Send(ctx, abstract.Notification{
				Recipient: abstract.Recipient{Email: email},
				Template:  "password_reset",
				Data:      map[string]any{"token": url.QueryEscape(token), "app_url": appURL},
				Channels:  []abstract.ChannelType{abstract.ChannelEmail},
			}); err != nil {
				return nil, err
			}
		}
		return &abstract.Result{}, nil
	}
}

func NewPasswordConfirmHandler(users *model.SystemUsers, credProv abstract.CredentialsProvider) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		token, _ := body["token"].(string)
		password, _ := body["password"].(string)

		userID, err := credProv.ValidateResetToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid or expired reset token")
		}

		hashed, err := runtime.HashPassword(password)
		if err != nil {
			return nil, err
		}
		if _, err := users.Update(ctx, userID, &model.SystemUser{Password: hashed}); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewSetBootstrapPasswordHandler(users *model.SystemUsers, adminUserID string) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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
		hashed, err := runtime.HashPassword(password)
		if err != nil {
			return nil, err
		}
		if _, err := users.Update(ctx, adminUserID, &model.SystemUser{Password: hashed}); err != nil {
			return nil, err
		}
		if err := users.UpdateEmail(ctx, adminUserID, email); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewValidateSessionHandler(credProv abstract.CredentialsProvider) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		token, _ := doc.GetOr("token", "").(string)

		info, err := credProv.ValidateSession(token)
		if err != nil {
			return nil, err
		}
		claimsDoc := data.MustNewDocument(map[string]any{
			"user_id":    info.UserID,
			"session_id": info.SessionID,
			"issued_at":  info.IssuedAt,
			"expires_at": info.ExpiresAt,
			"created_at": info.CreatedAt,
		}, ctx)
		return &abstract.Result{Document: claimsDoc}, nil
	}
}

type keyAuth interface {
	Authenticate(ctx context.Context, key string) (*abstract.Claims, error)
}

func NewValidateAPIKeyHandler(keyAuth keyAuth) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
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
		return &abstract.Result{Document: claimsDoc}, nil
	}
}
