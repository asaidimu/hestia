package abstract

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
)

type ArgDef struct {
	Name string
	Type definition.FieldType
}

type Input struct {
	Schema          *definition.Schema
	Arguments       []ArgDef
	Modifiers       map[string]definition.FieldType
	Payload         definition.FieldType
	ResourceIDField string
}

type Module interface {
	Name() string
	Setup(ctx context.Context, persist base.Persistence) error
	Capabilities() []Capability

	// Dependencies returns the names of modules that must be set up
	// before this one. Used for topological ordering of module init.
	Dependencies() []string

	// Start is called after all modules are set up, in dependency order.
	Start(ctx context.Context) error

	// Stop is called on graceful shutdown, in reverse dependency order.
	Stop(ctx context.Context) error

	// Health returns module-specific health data. Nil means healthy
	// with no additional detail. An error signals degraded health.
	Health(ctx context.Context) any
}

// BaseModule provides no-op defaults for Dependencies, Start, Stop, and
// Health so that simple modules only need to implement Name, Setup, and
// Capabilities.
type BaseModule struct{}

func (BaseModule) Dependencies() []string                    { return nil }
func (BaseModule) Start(_ context.Context) error              { return nil }
func (BaseModule) Stop(_ context.Context) error               { return nil }
func (BaseModule) Health(_ context.Context) any               { return nil }

type Capability struct {
	Name     string
	Messages []MessageRegistration
}

type MessageRegistration struct {
	Name          string             `json:"name"`
	Handler       MessageHandler     `json:"-"`
	Description   string             `json:"description"`
	Intent        Verb               `json:"intent"`
	Enabled       bool               `json:"enabled"`
	BootstrapSafe bool               `json:"bootstrap_safe"`
	Internal      bool               `json:"internal"`
	Input         Input              `json:"input,omitempty"`
	Output        *definition.Schema `json:"output,omitempty"`
}
