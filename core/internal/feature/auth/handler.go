package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/identity"
	"github.com/asaidimu/hestia/core/registration"
	"github.com/asaidimu/hestia/core/internal/feature/users"
)

func NewCreateSessionHandler(users *users.UserModel, credProv abstract.CredentialsProvider, sessionTTL time.Duration) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

		if !runtime.CheckPassword(password, storedPassword) {
			return nil, fmt.Errorf("invalid email or password")
		}

		userID := user.ID()

		token, _, err := credProv.CreateSession(userID, sessionTTL)
		if err != nil {
			return nil, err
		}

		sctx := common.ContextWithCollectionName(ctx, "_user_")
		sane, err := user.Sanitize(sctx)
		if err != nil {
			return nil, err
		}

		respDoc := data.MustNewDocument(map[string]any{}, ctx)
		if sane != nil {
			respDoc.Set("user", sane)
		}

		return &registration.Result{Document: respDoc, SessionToken: token}, nil
	}
}

func NewRegisterHandler(users *users.UserModel) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

func NewDeleteSessionHandler() runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		return &registration.Result{}, nil
	}
}

func NewPasswordResetHandler(users *users.UserModel, credProv abstract.CredentialsProvider) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		email, _ := body["email"].(string)

		user, err := users.GetByEmail(ctx, email)
		if err != nil {
			return &registration.Result{}, nil
		}
		userID := user.ID()
		credProv.IssueResetToken(userID)
		return &registration.Result{}, nil
	}
}

func NewPasswordConfirmHandler(users *users.UserModel, credProv abstract.CredentialsProvider) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		token, _ := body["token"].(string)
		password, _ := body["password"].(string)

		userID, err := credProv.ValidateResetToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid or expired reset token")
		}

		if err := users.ChangePassword(ctx, userID, password); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewSetBootstrapPasswordHandler(users *users.UserModel, adminUserID string) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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

func NewValidateSessionHandler(credProv abstract.CredentialsProvider) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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
		return &registration.Result{Document: claimsDoc}, nil
	}
}

type keyAuth interface {
	Authenticate(ctx context.Context, key string) (*identity.Claims, error)
}

func NewValidateAPIKeyHandler(keyAuth keyAuth) runtime.MessageHandler {
	return func(ctx context.Context, msg runtime.Message) (*registration.Result, error) {
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
