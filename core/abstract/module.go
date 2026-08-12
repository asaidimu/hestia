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
	HeaderFields    map[string]string
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
