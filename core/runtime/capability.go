package runtime

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
)

func NewSetCapabilityEnabledHandler(registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		doc := msg.Input()
		name, _ := doc.GetOr("arguments.name", "").(string)
		enabled, _ := doc.GetOr("payload.enabled", false).(bool)
		if name == "" {
			return nil, ErrValidation.WithOperation("system:core:capability:set")
		}
		if err := registry.SetHandlerEnabled(name, enabled); err != nil {
			return nil, err
		}
		return &abstract.Result{}, nil
	}
}

func NewListCapabilitiesHandler(registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		all := registry.ListHandlers()
		doc := data.MustNewDocument(map[string]any{
			"capabilities": all,
		}, ctx)
		return &abstract.Result{Document: doc}, nil
	}
}
