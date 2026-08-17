package runtime

import (
	"reflect"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/di"
)

// Runtime is the application runtime: a DI-backed struct services resolve
// their dependencies from. It embeds the di.Container, so every Container
// method (Register, RegisterInstance, Resolve, MustResolve, Build) is promoted
// onto Runtime directly.
type Runtime struct {
	*di.Container
}

// NewRuntime creates an unbuilt Runtime backed by a fresh di.Container.
func NewRuntime() *Runtime {
	return &Runtime{Container: di.New()}
}

var _ abstract.Container = (*Runtime)(nil)

// RegisterType implements abstract.Container. The registered constructor
// receives this Runtime (which satisfies abstract.Container) so it can resolve
// its own dependencies.
func (r *Runtime) RegisterType(t reflect.Type, ctor func(abstract.Container) (any, error)) error {
	return r.Container.RegisterType(t, func(c *di.Container) (any, error) {
		return ctor(r)
	})
}

// RegisterInstanceType implements abstract.Container.
func (r *Runtime) RegisterInstanceType(t reflect.Type, value any) error {
	return r.Container.RegisterInstanceType(t, value)
}

// ResolveType implements abstract.Container.
func (r *Runtime) ResolveType(t reflect.Type) (any, error) {
	return r.Container.ResolveType(t)
}
