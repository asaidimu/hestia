package apikeys

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/apikeys/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"
)

// APIKeysService is the service for the API keys domain. The model collection
// is initialized in the constructor by resolving persistence from the
// runtime.Runtime DI container; the struct is scaffolded once and then owned
// by the feature author.
type APIKeysService struct {
	model *model.SystemAPIKeys
}

func NewAPIKeysService(rt abstract.Container) (*APIKeysService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)

	m, err := model.InitSystemAPIKeysModel(persist, logger)
	if err != nil {
		return nil, err
	}
	return &APIKeysService{model: m}, nil
}

// userIDFrom resolves the acting user from the request claims. All apikeys
// registrations are authenticated (secure dispatcher), so claims are always
// present; there is no fallback to a bound argument.
func userIDFrom(ctx context.Context, msg abstract.Message) string {
	if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
		return claims.UserID
	}
	return ""
}

// keyDocWithSecret builds the response model for a key whose raw material is
// being revealed (create/rotate): the public projection plus the raw key.
func keyDocWithSecret(ctx context.Context, k *model.SystemAPIKey, rawKey string) (*model.APIKeyCreatedOutput, error) {
	var pub model.APIKeyPublic
	d, err := k.Document()
	if err != nil {
		return nil, err
	}
	if err := d.BindTo(&pub); err != nil {
		return nil, err
	}
	return document.New(&model.APIKeyCreatedOutput{APIKeyPublic: pub, Key: rawKey}), nil
}

// ListAPIKeys lists the API keys owned by the acting user.
//
// @hestia.register(
//   name="system:apikeys:key:list",
//   intent="read",
//   rule="administrator",
//   description="List own API keys",
// )
func (s *APIKeysService) ListAPIKeys(ctx context.Context, msg abstract.Message, input *model.APIKeyListInput) ([]*document.Document, error) {
	userID := userIDFrom(ctx, msg)

	docs, err := s.model.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]*document.Document, 0, len(docs))
	for _, k := range docs {
		doc, err := k.Document()
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

// GetAPIKey returns an API key owned by the acting user.
//
// @hestia.register(
//   name="system:apikeys:key:get",
//   intent="read",
//   rule="administrator",
//   description="Get own API key details",
//   resource_id="key_id",
// )
func (s *APIKeysService) GetAPIKey(ctx context.Context, msg abstract.Message, input *model.APIKeyGetInput) (*model.SystemAPIKey, error) {
	return s.model.Get(ctx, input.KeyID, userIDFrom(ctx, msg))
}

// CreateAPIKey generates and persists a new API key for the acting user,
// revealing the raw key material once.
//
// @hestia.register(
//   name="system:apikeys:key:create",
//   intent="create",
//   rule="administrator",
//   description="Create a new API key",
// )
func (s *APIKeysService) CreateAPIKey(ctx context.Context, msg abstract.Message, input *model.APIKeyCreate) (*model.APIKeyCreatedOutput, error) {
	userID := userIDFrom(ctx, msg)
	if userID == "" {
		return nil, fmt.Errorf("authentication context missing")
	}

	generated, err := s.model.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	created, err := s.model.CreateKey(ctx, generated, userID, input)
	if err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}

	return keyDocWithSecret(ctx, created, generated.FullKey)
}

// UpdateAPIKey updates the metadata of an API key owned by the acting user.
//
// @hestia.register(
//   name="system:apikeys:key:update",
//   intent="update",
//   rule="administrator",
//   description="Update API key metadata",
//   resource_id="key_id",
// )
func (s *APIKeysService) UpdateAPIKey(ctx context.Context, msg abstract.Message, input *model.APIKeyUpdate) (*model.SystemAPIKey, error) {
	return s.model.UpdateKey(ctx, input.ID, userIDFrom(ctx, msg), input)
}

// DeleteAPIKey deletes an API key owned by the acting user.
//
// @hestia.register(
//   name="system:apikeys:key:delete",
//   intent="delete",
//   rule="administrator",
//   description="Delete an API key",
//   resource_id="key_id",
// )
func (s *APIKeysService) DeleteAPIKey(ctx context.Context, msg abstract.Message, input *model.APIKeyDeleteInput) error {
	return s.model.Delete(ctx, input.KeyID, userIDFrom(ctx, msg))
}

// RotateAPIKey replaces the key material of an owned key, revealing the new
// raw key once.
//
// @hestia.register(
//   name="system:apikeys:key:rotate",
//   intent="create",
//   rule="administrator",
//   description="Rotate API key material",
//   resource_id="key_id",
// )
func (s *APIKeysService) RotateAPIKey(ctx context.Context, msg abstract.Message, input *model.APIKeyRotateInput) (*model.APIKeyCreatedOutput, error) {
	generated, k, err := s.model.Rotate(ctx, input.KeyID, userIDFrom(ctx, msg))
	if err != nil {
		return nil, err
	}

	return keyDocWithSecret(ctx, k, generated.FullKey)
}