package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/system/apikeys/model"
	"github.com/asaidimu/hestia/core/system/users/model"
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

		ok, err := runtime.CheckPassword(password, user.Password)
		if err != nil {
			return nil, runtime.ErrInternal.WithCause(err)
		}
		if !ok {
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

		respDoc, err := document.New(&ElevateDocumentView{
			Key: key.FullKey,
		}).Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: respDoc}, nil
	}
}
