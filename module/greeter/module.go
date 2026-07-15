package greeter

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/module/greeter/feature/greetings"
	"github.com/asaidimu/hestia/internal/abstract"
)

type Module struct {
	store    *greetings.GreetingStore
	messages []abstract.MessageRegistration
}

func New() *Module {
	return &Module{
		store: greetings.NewGreetingStore(),
	}
}

func (m *Module) Name() string { return "greeter" }

func (m *Module) Setup(_ context.Context, _ base.Persistence) error {
	m.messages = greetings.Registrations(greetings.Dependencies{Store: m.store})
	return nil
}

func (m *Module) Capabilities() []abstract.Capability {
	return []abstract.Capability{
		{Name: "greeter", Messages: m.messages},
	}
}
