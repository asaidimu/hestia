package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/feature/apikeys/model"
	"github.com/asaidimu/hestia/core/feature/users/model"
	"github.com/asaidimu/hestia/core/runtime"
)

func NewElevateTokenHandler(users *model.SystemUsers, apiKeys *apikeysmodel.SystemAPIKeys) abstract.MessageHandler {
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

		key, err := apiKeys.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate elevation key: %w", err)
		}

		expiry := time.Now().Add(5 * time.Minute).Format(time.RFC3339)
		if _, err := apiKeys.CreateKey(ctx, key, userID, &apikeysmodel.APIKeyCreate{
			Name:   "elevation",
			Expiry: &expiry,
		}); err != nil {
			return nil, fmt.Errorf("create elevation key: %w", err)
		}

		respDoc := data.MustNewDocument(map[string]any{
			"key": key.FullKey,
		}, ctx)
		return &abstract.Result{Document: respDoc}, nil
	}
}
