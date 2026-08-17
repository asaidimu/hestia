package abstract

import (
	"fmt"
	"reflect"
)

// Container is the abstract contract for the application runtime's
// dependency-injection container. Modules receive it in Setup so they can
// register service providers and resolve the base services (persistence,
// logger, dispatcher, ...) that boot pre-populated.
//
// The methods are intentionally non-generic: Go interfaces may not declare
// generic methods, and generic methods cannot satisfy interface methods
// (Go 1.27). Typed access is provided by the generic helper functions in this
// package (Register, RegisterInstance, Resolve, MustResolve), which adapt the
// reflection-based seam below.
type Container interface {
	// Build validates the full dependency graph and constructs singletons.
	Build() error

	// Rebuild allows registering additional providers after a previous
	// Build, then re-verifies the graph.
	Rebuild() error

	// RegisterType registers a constructor for the given type. The constructor
	// receives the container so it can resolve its own dependencies.
	RegisterType(t reflect.Type, ctor func(Container) (any, error)) error

	// RegisterInstanceType registers a pre-built value as a singleton.
	RegisterInstanceType(t reflect.Type, value any) error

	// ResolveType returns the value registered for t, constructing it if needed.
	ResolveType(t reflect.Type) (any, error)
}

// Register registers a constructor for T in the container.
func Register[T any](c Container, ctor func(Container) (T, error)) error {
	return c.RegisterType(reflect.TypeFor[T](), func(c Container) (any, error) {
		return ctor(c)
	})
}

// RegisterInstance registers a pre-built value of type T as a singleton.
func RegisterInstance[T any](c Container, value T) error {
	return c.RegisterInstanceType(reflect.TypeFor[T](), value)
}

// Resolve returns the value registered for T, constructing it if needed.
func Resolve[T any](c Container) (T, error) {
	var zero T
	v, err := c.ResolveType(reflect.TypeFor[T]())
	if err != nil {
		return zero, err
	}
	out, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("abstract: Resolve(%s): resolved value of type %T is not assignable to requested type", reflect.TypeFor[T](), v)
	}
	return out, nil
}

// MustResolve is like Resolve but panics on error.
func MustResolve[T any](c Container) T {
	v, err := Resolve[T](c)
	if err != nil {
		panic(err)
	}
	return v
}
