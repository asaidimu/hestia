// @note #org-20260821-001 todo status=open priority=P2 tags=#organization,#refactor : Split module.go into focused files
//
// module.go currently contains 4 unrelated concerns in 175 lines:
// - Input, ArgumentDefinition (message input schema)
// - Module interface (lifecycle)
// - Capability, MessageRegistration (message registration)
//
// A human looking for Input would guess input.go, not module.go.
// This hurts discoverability and increases modification risk.
//
// Resolution:
// 1. Create input.go with Input, ArgumentDefinition, and their methods
// 2. Keep module.go with Module interface only
// 3. Create registration.go with Capability and MessageRegistration
//
// Files affected:
// - core/abstract/module.go (split into 3 files)
// - All files importing these types (no code changes needed, just file moves)
package abstract

import (
	"context"
	"sort"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
)

type ArgumentDefinition struct {
	Name string
	Type definition.FieldType
}

// @note #review-20260821-020 issue status=open priority=P1 tags=#review,#design : Abstraction leak in Input struct
// The Input struct contains a comment "THIS IS AN ABSTRACTION LEAK" for
// HeaderFields, indicating known technical debt. The struct also has multiple
// deprecated fields (Arguments, Modifiers, Payload) that should be removed
// once all callers have migrated to the schema-based approach.
//
// Consider creating a migration plan to remove the deprecated fields and
// properly encapsulate the abstraction leak.
type Input struct {
	// THIS IS AN ABSTRACTION LEAK
	HeaderFields map[string]string
	Schema       *definition.Schema
	// Deprecated
	Arguments []ArgumentDefinition
	// Deprecated
	Modifiers map[string]definition.FieldType
	// Deprecated
	Payload         definition.FieldType
	ResourceIDField string
}

// Args extracts argument definitions by looking up the "arguments" field
// in the primary schema, resolving its referenced nested schema, and returning
// its sub-fields sorted by FieldId (UUIDv7) to preserve order.
func (i Input) Args() []ArgumentDefinition {
	if i.Schema == nil {
		return nil
	}

	// 1. Locate the top-level "arguments" field
	_, field := i.Schema.FindField("arguments")
	if field == nil || field.Schema.IsZero() {
		return nil
	}

	// 2. Extract the schema reference payload
	ref, ok := field.Schema.Value().(definition.SchemaReference)
	if !ok {
		return nil
	}

	// 3. Resolve the nested schema from the primary schema registry
	nestedSchema, exists := i.Schema.Schemas[ref.ID]
	if !exists {
		return nil
	}

	// 4. Collect field IDs and sort them to preserve deterministic order
	ids := make([]definition.FieldId, 0, len(nestedSchema.Fields))
	for id := range nestedSchema.Fields {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(a, b int) bool {
		return ids[a] < ids[b]
	})

	// 5. Map the sorted fields into []ArgumentDefinition
	args := make([]ArgumentDefinition, 0, len(ids))
	for _, id := range ids {
		f := nestedSchema.Fields[id]
		args = append(args, ArgumentDefinition{
			Name: string(f.Name),
			Type: f.Type,
		})
	}

	return args
}

// Mods returns the field type definitions for modifiers if present.
func (i Input) Mods() map[string]definition.FieldType {
	if i.Schema == nil {
		return nil
	}

	_, field := i.Schema.FindField("modifiers")
	if field == nil || field.Schema.IsZero() {
		return nil
	}

	ref, ok := field.Schema.Value().(definition.SchemaReference)
	if !ok {
		return nil
	}

	nestedSchema, exists := i.Schema.Schemas[ref.ID]
	if !exists {
		return nil
	}

	mods := make(map[string]definition.FieldType, len(nestedSchema.Fields))
	for _, f := range nestedSchema.Fields {
		mods[string(f.Name)] = f.Type
	}

	return mods
}

// PayloadType retrieves the FieldType of the "payload" field in the schema.
func (i Input) PayloadType() definition.FieldType {
	if i.Schema == nil {
		return definition.FieldTypeUnknown
	}

	_, field := i.Schema.FindField("payload")
	if field == nil {
		return definition.FieldTypeUnknown
	}

	return field.Type
}



type Module interface {
	Name() string

	// Setup registers the module's service providers into the shared runtime
	// container rt. Boot pre-populates rt with base providers (persistence,
	// logger, dispatcher) before any module runs; modules resolve them via
	// abstract.MustResolve[T](rt). In the two-phase boot, Setup only registers
	// providers; the container is sealed with Rebuild after all modules run.
	Setup(ctx context.Context, rt Container) error

	// Capabilities is called after all modules have been set up and the shared
	// runtime container has been rebuilt. It resolves the module's services
	// from rt and returns the message registrations they expose.
	Capabilities(rt Container) ([]Capability, error)

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
	// FireAndForget tells transports to accept the message and respond
	// immediately without waiting for handler completion. HTTP answers 202
	// Accepted with the message ID; validation and synchronous chain
	// rejections (auth, rate limit) still fail the request normally.
	// Until durable execution lands, accepted work is best-effort: if the
	// process dies after the ack, the handler never runs and the caller is
	// not notified. HTTP-only today; other transports await regardless.
	FireAndForget bool               `json:"fire_and_forget,omitempty"`
	Input         Input              `json:"input,omitempty"`
	Output        *definition.Schema `json:"output,omitempty"`
}
