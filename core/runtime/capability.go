package runtime

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
)

// CapabilityItem is the wire shape of one registered handler capability.
type CapabilityItem struct {
	Name          string `anansi:"name"`
	IntentType    string `anansi:"intent_type"`
	Description   string `anansi:"description"`
	Enabled       bool   `anansi:"enabled"`
	BootstrapSafe bool   `anansi:"bootstrap_safe"`
}

// CapabilitiesDocument is the body of a capabilities list response.
type CapabilitiesDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Capabilities           []CapabilityItem `anansi:"capabilities"`
}

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
		items := make([]CapabilityItem, len(all))
		for i, h := range all {
			items[i] = CapabilityItem{
				Name:          h.Name,
				IntentType:    string(h.IntentType),
				Description:   h.Description,
				Enabled:       h.Enabled,
				BootstrapSafe: h.BootstrapSafe,
			}
		}
		doc, err := document.New(&CapabilitiesDocument{Capabilities: items}).Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}
