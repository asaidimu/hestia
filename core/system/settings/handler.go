package settings

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func NewListSettingsHandler(m *SettingsModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		tenantID := runtimecontext.GetTenantID(ctx)
		entries, err := m.All(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		docs := make([]*document.Document, 0, len(entries))
		for k, v := range entries {
			doc, err := document.New(&SettingDocumentView{Key: k, Value: v}).Document()
			if err != nil {
				return nil, err
			}
			docs = append(docs, doc)
		}
		return &abstract.Result{Documents: docs}, nil
	}
}

func NewGetSettingHandler(m *SettingsModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		tenantID := runtimecontext.GetTenantID(ctx)
		doc := msg.Input()
		key, _ := doc.GetString("arguments.key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		val, err := m.Get(ctx, tenantID, key)
		if err != nil {
			return nil, err
		}
		return dispatch.NewDocumentResultFrom(&SettingDocumentView{
			Key:   key,
			Value: val,
		})
	}
}

func NewSetSettingHandler(m *SettingsModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		tenantID := runtimecontext.GetTenantID(ctx)
		doc := msg.Input()
		key, _ := doc.GetString("arguments.key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		value, hasValue := body["value"]
		if !hasValue {
			return nil, fmt.Errorf("payload.value is required")
		}
		updatedBy := ""
		if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
			updatedBy = claims.UserID
		}
		if err := m.Set(ctx, tenantID, key, value, updatedBy); err != nil {
			return nil, err
		}
		return dispatch.NewDocumentResultFrom(&MessageOutput{
			Message: fmt.Sprintf("setting %q saved", key),
		})
	}
}

func NewDeleteSettingHandler(m *SettingsModel) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		tenantID := runtimecontext.GetTenantID(ctx)
		doc := msg.Input()
		key, _ := doc.GetString("arguments.key")
		if key == "" {
			return nil, fmt.Errorf("key is required")
		}
		if err := m.Unset(ctx, tenantID, key); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}
