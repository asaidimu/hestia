package runtime

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/registration"
)

func NewSetCapabilityEnabledHandler(registry Registry) MessageHandler {
	return func(ctx context.Context, msg Message) (*registration.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		enabled, _ := doc.GetOr("payload.enabled", false).(bool)
		if name == "" {
			return nil, ErrValidation.WithOperation("system:core:capability:set")
		}
		if err := registry.SetHandlerEnabled(name, enabled); err != nil {
			return nil, err
		}
		return &registration.Result{}, nil
	}
}

func NewListCapabilitiesHandler(registry Registry) MessageHandler {
	return func(ctx context.Context, msg Message) (*registration.Result, error) {
		all := registry.ListHandlers()
		doc := data.MustNewDocument(map[string]any{
			"capabilities": all,
		}, ctx)
		return &registration.Result{Document: doc}, nil
	}
}
