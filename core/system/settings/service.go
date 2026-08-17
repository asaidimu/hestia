package settings

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/system/settings/model"

	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
)

// SettingsService is the service for the key/value settings domain. It wraps
// the hand-rolled SettingsModel (raw persistence over the _settings_ collection)
// rather than the generated collection, preserving the existing tenant-scoped
// read/upsert/delete semantics.
type SettingsService struct {
	model *model.SettingsModel
}

func NewSettingsService(rt abstract.Container) (*SettingsService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)

	return &SettingsService{model: model.NewSettingsModel(persist)}, nil
}

// ListSettings lists all settings for the current tenant.
//
// @hestia.register(
//   name="system:settings:list",
//   intent="read",
//   rule="administrator",
//   description="List all settings",
// )
func (s *SettingsService) ListSettings(ctx context.Context, msg abstract.Message, input *model.SettingListInput) ([]*document.Document, error) {
	tenantID := runtimecontext.GetTenantID(ctx)
	entries, err := s.model.All(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	docs := make([]*document.Document, 0, len(entries))
	for k, v := range entries {
		doc, err := document.New(&model.SettingDocumentView{Key: k, Value: v}).Document()
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// GetSetting returns a single setting by key for the current tenant.
//
// @hestia.register(
//   name="system:settings:get",
//   intent="read",
//   rule="administrator",
//   description="Get a setting by key",
//   resource_id="key",
// )
func (s *SettingsService) GetSetting(ctx context.Context, msg abstract.Message, input *model.SettingKeyInput) (*model.SettingDocumentView, error) {
	tenantID := runtimecontext.GetTenantID(ctx)
	if input.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	val, err := s.model.Get(ctx, tenantID, input.Key)
	if err != nil {
		return nil, err
	}
	return document.New(&model.SettingDocumentView{Key: input.Key, Value: val}), nil
}

// SetSetting creates or updates a setting for the current tenant.
//
// @hestia.register(
//   name="system:settings:set",
//   intent="create",
//   rule="administrator",
//   description="Create or update a setting",
//   resource_id="key",
// )
func (s *SettingsService) SetSetting(ctx context.Context, msg abstract.Message, input *model.SetSettingInput) (*model.MessageOutput, error) {
	tenantID := runtimecontext.GetTenantID(ctx)
	if input.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if input.Value == nil {
		return nil, fmt.Errorf("payload.value is required")
	}
	updatedBy := ""
	if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
		updatedBy = claims.UserID
	}
	if err := s.model.Set(ctx, tenantID, input.Key, input.Value, updatedBy); err != nil {
		return nil, err
	}
	return document.New(&model.MessageOutput{Message: fmt.Sprintf("setting %q saved", input.Key)}), nil
}

// DeleteSetting deletes a setting by key for the current tenant.
//
// @hestia.register(
//   name="system:settings:delete",
//   intent="delete",
//   rule="administrator",
//   description="Delete a setting",
//   resource_id="key",
// )
func (s *SettingsService) DeleteSetting(ctx context.Context, msg abstract.Message, input *model.SettingKeyInput) error {
	tenantID := runtimecontext.GetTenantID(ctx)
	if input.Key == "" {
		return fmt.Errorf("key is required")
	}
	return s.model.Unset(ctx, tenantID, input.Key)
}