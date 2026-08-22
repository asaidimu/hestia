// @note #cruft-20260821-003 issue status=open priority=P1 tags=#cruft,#dead-code : Old-style handler functions superseded by service methods
// @see #8uuufn
//
// This file contains NewCreateSessionHandler, NewDeleteSessionHandler,
// NewPasswordResetHandler, NewPasswordConfirmHandler, NewSetBootstrapPasswordHandler,
// NewValidateSessionHandler, and NewValidateAPIKeyHandler — all using the old
// pattern of manual input extraction via doc.GetOr("payload", nil).(map[string]any).
//
// The generated registrations in auth/registrations.go use the typed service
// methods from auth/service.go (CreateSession, DeleteSession, etc.) with
// dispatch.Handle[T] for automatic input binding.
//
// These handler functions are only used in auth_test.go. The tests should be
// migrated to use the service methods directly, then this file can be deleted.
package auth

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/users/model"
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

		ok, err := runtime.CheckPassword(password, user.Password)
		if err != nil {
			return nil, runtime.ErrInternal.WithCause(err)
		}
		if !ok {
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

		view := &LoginDocumentView{}
		if sane != nil {
			view.User = sane.ToMap()
		}

		resultDoc, err := document.New(view).Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: resultDoc, SessionToken: token}, nil
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
		claimsDoc, err := document.New(&ClaimsDocumentView{
			UserID:    info.UserID,
			SessionID: info.SessionID,
			IssuedAt:  info.IssuedAt,
			ExpiresAt: info.ExpiresAt,
			CreatedAt: info.CreatedAt,
		}).Document()
		if err != nil {
			return nil, err
		}
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
		claimsDoc, err := document.New(&APIKeyClaimsView{
			UserID:      claims.UserID,
			Email:       claims.Email,
			Permissions: claims.Scopes,
			TokenType:   claims.TokenType,
			TokenID:     claims.TokenID,
			ExpiresAt:   claims.ExpiresAt,
		}).Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: claimsDoc}, nil
	}
}
