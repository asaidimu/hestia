package apikeys

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/feature/apikeys/model"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
)

func userIDFrom(ctx context.Context, msg abstract.Message) string {
	if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
		return claims.UserID
	}
	var input model.APIKeyListInput
	if err := msg.Input().BindToTag(&input, "input"); err != nil {
		return ""
	}
	return input.UserID
}

func NewListAPIKeysHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		userID := userIDFrom(ctx, msg)

		docs, err := keys.List(ctx, userID)
		if err != nil {
			return nil, err
		}

		out := make(data.DocumentSet, 0, len(docs))
		for _, k := range docs {
			doc, err := k.Document()
			if err != nil {
				return nil, err
			}
			out = append(out, doc)
		}
		return &abstract.Result{Documents: out}, nil
	}
}

func NewGetAPIKeyHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.APIKeyGetInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		k, err := keys.Get(ctx, input.KeyID, userIDFrom(ctx, msg))
		if err != nil {
			return nil, err
		}

		doc, err := k.Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}

// keyDocWithSecret builds the response document for a key whose raw material
// is being revealed (create/rotate): the public projection plus the raw key.
func keyDocWithSecret(ctx context.Context, k *model.SystemAPIKey, rawKey string) (*document.Document, error) {
	var pub model.APIKeyPublic
	d, err := k.Document()
	if err != nil {
		return nil, err
	}
	if err := d.BindTo(&pub); err != nil {
		return nil, err
	}
	out := document.New(&model.APIKeyCreatedOutput{APIKeyPublic: pub, Key: rawKey})
	return out.Document()
}

func NewCreateAPIKeyHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.APIKeyCreate
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		userID := userIDFrom(ctx, msg)
		if userID == "" {
			return nil, fmt.Errorf("authentication context missing")
		}

		generated, err := keys.Generate()
		if err != nil {
			return nil, fmt.Errorf("generate key: %w", err)
		}

		created, err := keys.CreateKey(ctx, generated, userID, &input)
		if err != nil {
			return nil, fmt.Errorf("create key: %w", err)
		}

		doc, err := keyDocWithSecret(ctx, created, generated.FullKey)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}

func NewUpdateAPIKeyHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.APIKeyUpdate
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		updated, err := keys.UpdateKey(ctx, input.ID, userIDFrom(ctx, msg), &input)
		if err != nil {
			return nil, err
		}

		doc, err := updated.Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}

func NewDeleteAPIKeyHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.APIKeyDeleteInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		if err := keys.Delete(ctx, input.KeyID, userIDFrom(ctx, msg)); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewRotateAPIKeyHandler(keys *model.SystemAPIKeys) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.APIKeyRotateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}

		generated, k, err := keys.Rotate(ctx, input.KeyID, userIDFrom(ctx, msg))
		if err != nil {
			return nil, err
		}

		doc, err := keyDocWithSecret(ctx, k, generated.FullKey)
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}
